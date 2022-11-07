package autoscaling

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/rs/zerolog/log"
	"time"
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
	"wekactl/internal/lib/strings"
	"wekactl/internal/logging"
)

var KeepInstances bool

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

	log.Debug().Msgf("AutoScalingGroup: \"%s\" suspending ReplaceUnhealthy...", autoScalingGroupName)
	_, err = svc.SuspendProcesses(&autoscaling.ScalingProcessQuery{
		AutoScalingGroupName: &autoScalingGroupName,
		ScalingProcesses: []*string{
			aws.String("ReplaceUnhealthy"),
		},
	})

	return
}

func UpdateAutoScalingGroup(launchTemplateName, autoScalingGroupName string, tags []*autoscaling.Tag) (err error) {
	svc := connectors.GetAWSSession().ASG

	_, err = svc.UpdateAutoScalingGroup(&autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: &autoScalingGroupName,
		LaunchTemplate: &autoscaling.LaunchTemplateSpecification{
			LaunchTemplateName: aws.String(launchTemplateName),
			Version:            aws.String("$Latest"),
		},
	})
	if err != nil {
		return
	}

	for _, tag := range tags {
		tag.ResourceId = &autoScalingGroupName
		tag.ResourceType = aws.String("auto-scaling-group")
		tag.PropagateAtLaunch = aws.Bool(false)
	}
	_, err = svc.CreateOrUpdateTags(&autoscaling.CreateOrUpdateTagsInput{
		Tags: tags,
	})
	if err != nil {
		return
	}

	log.Debug().Msgf("AutoScalingGroup: \"%s\" was updated successfully!", autoScalingGroupName)

	return
}

func AttachInstancesToASG(instancesIds []*string, autoScalingGroupsName string) error {
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

func DetachInstancesFromASG(instancesIds []string, autoScalingGroupsName string) error {
	svc := connectors.GetAWSSession().ASG
	limit := 20
	for i := 0; i < len(instancesIds); i += limit {
		batch := strings.ListToRefList(instancesIds[i:common.Min(i+limit, len(instancesIds))])
		_, err := svc.DetachInstances(&autoscaling.DetachInstancesInput{
			AutoScalingGroupName:           &autoScalingGroupsName,
			InstanceIds:                    batch,
			ShouldDecrementDesiredCapacity: aws.Bool(false),
		})
		if err != nil {
			return err
		}
		log.Info().Msgf("Detached %d instances from %s successfully!", len(batch), autoScalingGroupsName)
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
	for _, asgGroup := range asgOutput.AutoScalingGroups {
		for _, asgLoadBalancerName := range asgGroup.LoadBalancerNames {
			if *asgLoadBalancerName == *loadBalancerName {
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

	if KeepInstances && len(instanceIds) > 0 {
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
			if *activity.StatusCode == autoscaling.ScalingActivityStatusCodeInProgress {
				logging.UserProgress("Waiting 10 sec for auto scaling group %s instances detaching to finish", autoScalingGroupName)
				time.Sleep(10 * time.Second)
				retry = true
				break
			}
		}
	}

	if retry {
		return errors.New(fmt.Sprintf(
			"some asg scaling activities are still in progress, can't delete asg - %s", autoScalingGroupName))
	}

	_, err = svc.DeleteAutoScalingGroup(&autoscaling.DeleteAutoScalingGroupInput{
		AutoScalingGroupName: &autoScalingGroupName,
		ForceDelete:          aws.Bool(true),
	})
	if err != nil {
		return err
	}
	log.Debug().Msgf("scaling group %s deleted", autoScalingGroupName)

	return nil
}

func GetAsgTagValue(asg *autoscaling.Group, key string) (value string) {
	for _, tag := range asg.Tags {
		if *tag.Key == key {
			value = *tag.Value
			return
		}
	}
	return
}

func GetAutoScalingGroupVersion(autoScalingGroupName string) (version string, err error) {
	svc := connectors.GetAWSSession().ASG

	asgOutput, err := svc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{&autoScalingGroupName},
	})
	if err != nil {
		return
	}

	for _, asg := range asgOutput.AutoScalingGroups {
		version = GetAsgTagValue(asg, cluster.VersionTagKey)
	}
	return
}

func GetClusterAutoScalingGroups(clusterName cluster.ClusterName) (autoScalingGroups []*autoscaling.Group, err error) {
	svcAsg := connectors.GetAWSSession().ASG
	var nextToken *string
	var asgOutput *autoscaling.DescribeAutoScalingGroupsOutput

	for asgOutput == nil || nextToken != nil {
		asgOutput, err = svcAsg.DescribeAutoScalingGroups(
			&autoscaling.DescribeAutoScalingGroupsInput{
				NextToken: nextToken,
			},
		)
		if err != nil {
			return
		}

		for _, asg := range asgOutput.AutoScalingGroups {
			for _, tag := range asg.Tags {
				if *tag.Key == cluster.ClusterNameTagKey && *tag.Value == string(clusterName) {
					autoScalingGroups = append(autoScalingGroups, asg)
					break
				}
			}

		}
		nextToken = asgOutput.NextToken
	}
	return
}

func DeleteAutoScalingGroups(autoScalingGroups []*autoscaling.Group) error {
	for _, autoScalingGroup := range autoScalingGroups {
		err := DeleteAutoScalingGroup(*autoScalingGroup.AutoScalingGroupName)
		if err != nil {
			return err
		}
	}
	return nil
}
