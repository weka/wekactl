package alb

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
)

const ListenerTypeTagKey = "wekactl.io/listener_type"

func CreateApplicationLoadBalancer(tags []*elbv2.Tag, albName string, subnets []*string, securityGroupsIds []*string) (arn string, err error) {
	svc := connectors.GetAWSSession().ELBV2
	albOutput, err := svc.CreateLoadBalancer(&elbv2.CreateLoadBalancerInput{
		Name:           aws.String(albName),
		Scheme:         aws.String("internal"),
		Subnets:        subnets,
		Tags:           tags,
		SecurityGroups: securityGroupsIds,
		Type:           aws.String(elbv2.LoadBalancerTypeEnumApplication),
	})
	if err != nil {
		return
	}

	arn = *albOutput.LoadBalancers[0].LoadBalancerArn
	return

}

func GetApplicationLoadBalancerArn(albName string) (arn string, err error) {
	svc := connectors.GetAWSSession().ELBV2

	loadBalancerOutput, err := svc.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
		Names: []*string{
			&albName,
		},
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == elbv2.ErrCodeLoadBalancerNotFoundException {
				err = nil
			}
		}
		return
	}
	arn = *loadBalancerOutput.LoadBalancers[0].LoadBalancerArn
	return
}

func GetTargetGroupArn(clusterName cluster.ClusterName) (arn string, err error) {
	svc := connectors.GetAWSSession().ELBV2

	targetGroupOutput, err := svc.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{
		Names: []*string{
			aws.String(GetTargetGroupName(clusterName)),
		},
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == elbv2.ErrCodeTargetGroupNotFoundException {
				err = nil
			}
		}
		return
	}

	for _, tg := range targetGroupOutput.TargetGroups {
		arn = *tg.TargetGroupArn
		return
	}

	return
}

func DeleteApplicationLoadBalancer(albName string) (err error) {
	svc := connectors.GetAWSSession().ELBV2

	arn, err := GetApplicationLoadBalancerArn(albName)
	if err != nil {
		return
	}

	if arn == "" {
		return
	}

	_, err = svc.DeleteLoadBalancer(&elbv2.DeleteLoadBalancerInput{
		LoadBalancerArn: &arn,
	})

	return
}

func GetTargetGroupName(clusterName cluster.ClusterName) string {
	return fmt.Sprintf("%s-backends-api", common.GenerateResourceName(clusterName, ""))
}

func CreateTargetGroup(tags []*elbv2.Tag, targetName, vpcId string) (arn string, err error) {
	svc := connectors.GetAWSSession().ELBV2

	targetOutput, err := svc.CreateTargetGroup(&elbv2.CreateTargetGroupInput{
		Name:            aws.String(targetName),
		Port:            aws.Int64(14000),
		Protocol:        aws.String("HTTP"),
		VpcId:           aws.String(vpcId),
		Tags:            tags,
		HealthCheckPath: aws.String("/api/v2/healthcheck/"),
	})
	if err != nil {
		return
	}
	arn = *targetOutput.TargetGroups[0].TargetGroupArn
	return
}

func DeleteTargetGroup(clusterName cluster.ClusterName) (err error) {
	svc := connectors.GetAWSSession().ELBV2

	targetOutput, err := svc.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{
		Names: []*string{
			aws.String(GetTargetGroupName(clusterName)),
		},
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == elbv2.ErrCodeTargetGroupNotFoundException {
				err = nil
			}
		}
		return
	}

	for _, target := range targetOutput.TargetGroups {
		_, err = svc.DeleteTargetGroup(&elbv2.DeleteTargetGroupInput{
			TargetGroupArn: target.TargetGroupArn,
		})
	}

	return
}

func CreateListener(tags []*elbv2.Tag, albArn, targetArn string) error {
	svc := connectors.GetAWSSession().ELBV2
	_, err := svc.CreateListener(&elbv2.CreateListenerInput{
		DefaultActions: []*elbv2.Action{
			{
				TargetGroupArn: &targetArn,
				Type:           aws.String("forward"),
			},
		},
		LoadBalancerArn: &albArn,
		Port:            aws.Int64(14000),
		Protocol:        aws.String("HTTP"),
		Tags:            tags,
	})

	return err
}

func DeleteListener(albName string) error {
	svc := connectors.GetAWSSession().ELBV2

	albArn, err := GetApplicationLoadBalancerArn(albName)
	if err != nil {
		return err
	}

	if albArn == "" {
		return nil
	}

	listenersOutput, err := svc.DescribeListeners(&elbv2.DescribeListenersInput{
		LoadBalancerArn: &albArn,
	})
	if err != nil {
		return err
	}

	for _, listener := range listenersOutput.Listeners {
		_, err = svc.DeleteListener(&elbv2.DeleteListenerInput{
			ListenerArn: listener.ListenerArn,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func getTagValue(tags *elbv2.DescribeTagsOutput, tagKey string) (tagValue string) {
	for _, tagDesc := range tags.TagDescriptions {
		for _, tag := range tagDesc.Tags {
			if *tag.Key == tagKey {
				tagValue = *tag.Value
				break
			}
		}
		if tagValue != "" {
			break
		}
	}
	return
}

func getResourceTagValue(arn, tagKey string) (tagValue string, err error) {
	svc := connectors.GetAWSSession().ELBV2
	if arn == "" {
		return
	}
	tags, err := svc.DescribeTags(&elbv2.DescribeTagsInput{
		ResourceArns: []*string{
			&arn,
		},
	})
	if err != nil {
		return
	}

	tagValue = getTagValue(tags, tagKey)
	return
}

func GetApplicationLoadBalancerVersion(albName string) (version string, err error) {
	arn, err := GetApplicationLoadBalancerArn(albName)
	if err != nil {
		return
	}

	return getResourceTagValue(arn, cluster.VersionTagKey)
}

func GetTargetGroupVersion(clusterName cluster.ClusterName) (version string, err error) {
	arn, err := GetTargetGroupArn(clusterName)
	if err != nil {
		return
	}

	return getResourceTagValue(arn, cluster.VersionTagKey)
}

func GetListenerVersion(albName, requestedListenerType string) (version string, err error) {
	svc := connectors.GetAWSSession().ELBV2

	arn, err := GetApplicationLoadBalancerArn(albName)
	if err != nil {
		return
	}

	if arn == "" {
		return
	}

	listenersOutput, err := svc.DescribeListeners(&elbv2.DescribeListenersInput{
		LoadBalancerArn: &arn,
	})

	if err != nil {
		return
	}

	var tags *elbv2.DescribeTagsOutput
	for _, listener := range listenersOutput.Listeners {
		tags, err = svc.DescribeTags(&elbv2.DescribeTagsInput{
			ResourceArns: []*string{
				listener.ListenerArn,
			},
		})

		listenerType := getTagValue(tags, ListenerTypeTagKey)
		if err != nil {
			return
		}
		if listenerType == requestedListenerType {
			version = getTagValue(tags, cluster.VersionTagKey)
			return
		}
	}

	return
}
