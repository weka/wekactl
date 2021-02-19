package autoscaling

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/rs/zerolog/log"
	"math"
	"time"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
	"wekactl/internal/logging"
)

func GetAutoScalingTags(hostGroupInfo hostgroups.HostGroupInfo, version string) []*autoscaling.Tag {
	var autoscalingTags []*autoscaling.Tag
	for key, value := range common.GetHostGroupTags(hostGroupInfo, version) {
		autoscalingTags = append(autoscalingTags, &autoscaling.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	autoscalingTags = append(autoscalingTags, &autoscaling.Tag{
		Key:   aws.String("Name"),
		Value: aws.String(fmt.Sprintf("%s-%s", hostGroupInfo.ClusterName, hostGroupInfo.Name)),
	})
	return autoscalingTags
}

func GetMaxSize(role hostgroups.InstanceRole, initialSize int) int {
	var maxSize int
	switch role {
	case "backend":
		maxSize = 7 * initialSize
	case "client":
		maxSize = int(math.Ceil(float64(initialSize)/float64(500))) * 500
	default:
		maxSize = 1000
	}
	return maxSize
}

func CreateAutoScalingGroup(tags []*autoscaling.Tag, launchTemplateName string, maxSize int64, autoScalingGroupName string) (err error) {
	svc := connectors.GetAWSSession().ASG
	input := &autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName:             &autoScalingGroupName,
		NewInstancesProtectedFromScaleIn: aws.Bool(true),
		LaunchTemplate: &autoscaling.LaunchTemplateSpecification{
			LaunchTemplateName: aws.String(launchTemplateName),
			Version:            aws.String("1"),
		},
		MinSize: aws.Int64(0),
		MaxSize: aws.Int64(maxSize),
		Tags:    tags,
	}
	_, err = svc.CreateAutoScalingGroup(input)
	if err != nil {
		return
	}
	log.Debug().Msgf("AutoScalingGroup: \"%s\" was created successfully!", autoScalingGroupName)
	return
}

func AttachInstancesToAutoScalingGroups(instancesIds []*string, autoScalingGroupsName string) error {
	asgInstanceIds, err := common.GetAutoScalingGroupInstanceIds(autoScalingGroupsName)
	if err != nil {
		return err
	}
	instancesIds = common.GetDeltaInstancesIds(asgInstanceIds, instancesIds)
	svc := connectors.GetAWSSession().ASG
	limit := 20
	for i := 0; i < len(instancesIds); i += limit {
		batch := instancesIds[i:common.Min(i+limit, len(instancesIds))]
		_, err := svc.AttachInstances(&autoscaling.AttachInstancesInput{
			AutoScalingGroupName: &autoScalingGroupsName,
			InstanceIds:          batch,
		})
		if err != nil {
			return err
		}
		log.Debug().Msgf("Attached %d instances to %s successfully!", len(batch), autoScalingGroupsName)
	}
	return nil
}

func getStackLoadBalancer(stackName string) (loadBalancerName *string, err error) {
	svc := connectors.GetAWSSession().CF
	result, err := svc.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: &stackName,
	})
	if err != nil {
		return
	}

	for _, resource := range result.StackResources {
		if *resource.ResourceType == "AWS::ElasticLoadBalancing::LoadBalancer" {
			return resource.PhysicalResourceId, nil
		}
	}
	return
}

func AttachLoadBalancer(clusterName cluster.ClusterName, AutoScalingGroupName string) (err error) {
	loadBalancerName, err := getStackLoadBalancer(string(clusterName))
	if loadBalancerName == nil || err != nil {
		return
	}
	svc := connectors.GetAWSSession().ASG
	asgOutput, err := svc.DescribeAutoScalingGroups(
		&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{&AutoScalingGroupName},
		},
	)
	if err != nil {
		return
	}
	for _,asgGroup := range asgOutput.AutoScalingGroups{
		for _, asgLoadBalancerName := range asgGroup.LoadBalancerNames{
			if *asgLoadBalancerName == *loadBalancerName{
				log.Debug().Msgf("load balancer %s is already attached to %s", *loadBalancerName, AutoScalingGroupName)
				return
			}
		}
	}

	_, err = svc.AttachLoadBalancers(&autoscaling.AttachLoadBalancersInput{
		LoadBalancerNames:    []*string{loadBalancerName},
		AutoScalingGroupName: aws.String(AutoScalingGroupName),
	})
	if err != nil {
		return
	}
	log.Debug().Msgf("load balancer %s was attached to %s successfully!", *loadBalancerName, AutoScalingGroupName)

	svcElb := connectors.GetAWSSession().ELB
	_, err = svcElb.ConfigureHealthCheck(&elb.ConfigureHealthCheckInput{
		HealthCheck: &elb.HealthCheck{
			HealthyThreshold:   aws.Int64(3),
			Interval:           aws.Int64(30),
			Target:             aws.String("HTTP:14000/ui"),
			Timeout:            aws.Int64(5),
			UnhealthyThreshold: aws.Int64(5),
		},
		LoadBalancerName: loadBalancerName,
	})
	if err != nil {
		return
	}
	log.Debug().Msgf("load balancer %s health check configured successfully!", *loadBalancerName)

	return
}

func DeleteAutoScalingGroup(autoScalingGroupName string) error {
	svc := connectors.GetAWSSession().ASG

	asgOutput, err := svc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{&autoScalingGroupName},
	})
	if err != nil {
		return err
	}

	if len(asgOutput.AutoScalingGroups) == 0 {
		return nil
	}

	_, err = svc.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: &autoScalingGroupName,
		MinSize:              aws.Int64(0),
	})
	if err != nil {
		return err
	}
	log.Debug().Msgf("auto scaling group %s min updated value to 0", autoScalingGroupName)

	var instanceIds []*string
	for _, asg := range asgOutput.AutoScalingGroups {
		for _, instance := range asg.Instances {
			instanceIds = append(instanceIds, instance.InstanceId)
		}
	}

	if len(instanceIds) > 0 {
		_, err = svc.DetachInstances(&autoscaling.DetachInstancesInput{
			AutoScalingGroupName:           &autoScalingGroupName,
			ShouldDecrementDesiredCapacity: aws.Bool(true),
			InstanceIds:                    instanceIds,
		})
		if err != nil {
			return err
		}
		log.Debug().Msgf("auto scaling group %s instances detached", autoScalingGroupName)
	}

	retry := true
	for i := 0; i < 6 && retry; i++ {
		activitiesOutput, err := svc.DescribeScalingActivities(&autoscaling.DescribeScalingActivitiesInput{
			AutoScalingGroupName: &autoScalingGroupName})
		if err != nil {
			return err
		}
		retry = false
		for _, activity := range activitiesOutput.Activities {
			if *activity.StatusCode != autoscaling.ScalingActivityStatusCodeSuccessful {
				logging.UserProgress("waiting 10 sec for auto scaling group %s instances detaching to finish", autoScalingGroupName)
				time.Sleep(10 * time.Second)
				retry = true
				break
			}
		}
	}

	_, err = svc.DeleteAutoScalingGroup(&autoscaling.DeleteAutoScalingGroupInput{
		AutoScalingGroupName: &autoScalingGroupName,
	})
	if err != nil {
		return err
	}
	log.Debug().Msgf("scaling group %s deleted", autoScalingGroupName)

	return nil
}
