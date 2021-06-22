package alb

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/lithammer/dedent"
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
	strings2 "wekactl/internal/lib/strings"
)

const ListenerTypeTagKey = "wekactl.io/listener_type"

func GetApplicationLoadBalancerName(clusterName cluster.ClusterName) string {
	return strings2.ElfHashSuffixed(common.GenerateResourceName(clusterName, ""), 32)
}

func CreateApplicationLoadBalancer(tags []*elbv2.Tag, albName string, subnets []*string, securityGroupsIds []*string) (loadBalancer *elbv2.LoadBalancer, err error) {
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

	loadBalancer = albOutput.LoadBalancers[0]
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
	return strings2.ElfHashSuffixed(
		fmt.Sprintf("%s-api", common.GenerateResourceName(clusterName, "")), 32)
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

func GetApplicationLoadBalancerDns(clusterName cluster.ClusterName) (dns string, err error) {
	svc := connectors.GetAWSSession().ELBV2

	loadBalancerOutput, err := svc.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
		Names: []*string{
			aws.String(GetApplicationLoadBalancerName(clusterName)),
		},
	})
	if err != nil {
		return
	}
	dns = *loadBalancerOutput.LoadBalancers[0].DNSName
	return
}

func PrintStatelessClientsJoinScript(clusterName cluster.ClusterName, dnsAlias string) (err error) {
	dns := dnsAlias
	if dns == "" {
		dns, err = GetApplicationLoadBalancerDns(clusterName)
		if err != nil {
			return
		}
	}

	bashScriptTemplate := `
	"""
	#!/bin/bash

	curl %s:14000/dist/v1/install | sh

	FILESYSTEM_NAME=default # replace with a different filesystem at need
	MOUNT_POINT=/mnt/weka # replace with a different mount point at need

	mkdir -p $MOUNT_POINT
	mount -t wekafs %s/"$FILESYSTEM_NAME" $MOUNT_POINT
	"""`

	fmt.Println(fmt.Sprintf(
		"Script for mounting stateless clients:\n%s",
		dedent.Dedent(fmt.Sprintf(bashScriptTemplate, dns, dns)[1:])))
	return
}

func GetClusterApplicationLoadBalancer(clusterName cluster.ClusterName) (applicationLoadBalancer *elbv2.LoadBalancer, err error) {
	svc := connectors.GetAWSSession().ELBV2

	loadBalancersOutput, err := svc.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{})
	if err != nil {
		return
	}

	var loadBalancerName string
	for _, loadBalancer := range loadBalancersOutput.LoadBalancers {
		loadBalancerName, err = getResourceTagValue(*loadBalancer.LoadBalancerArn, cluster.ClusterNameTagKey)
		if err != nil {
			return
		}
		if loadBalancerName == string(clusterName) {
			applicationLoadBalancer = loadBalancer
			return
		}
	}
	return
}

func GetClusterTargetGroup(clusterName cluster.ClusterName) (albTargetGroup *elbv2.TargetGroup, err error) {
	svc := connectors.GetAWSSession().ELBV2

	targetGroupsOutput, err := svc.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{})
	if err != nil {
		return
	}

	var targetGroupName string
	for _, targetGroup := range targetGroupsOutput.TargetGroups {
		targetGroupName, err = getResourceTagValue(*targetGroup.TargetGroupArn, cluster.ClusterNameTagKey)
		if err != nil {
			return
		}
		if targetGroupName == string(clusterName) {
			albTargetGroup = targetGroup
			return
		}
	}
	return
}

func GetClusterListener(clusterName cluster.ClusterName, loadBalancerArn string) (albListener *elbv2.Listener, err error) {
	svc := connectors.GetAWSSession().ELBV2

	listenersOutput, err := svc.DescribeListeners(&elbv2.DescribeListenersInput{
		LoadBalancerArn: &loadBalancerArn,
	})
	if err != nil {
		return
	}

	var listenerName string
	for _, listener := range listenersOutput.Listeners {
		listenerName, err = getResourceTagValue(*listener.ListenerArn, cluster.ClusterNameTagKey)
		if err != nil {
			return
		}
		if listenerName == string(clusterName) {
			albListener = listener
			return
		}
	}
	return
}

func DeleteAlb(applicationLoadBalancer *elbv2.LoadBalancer, listener *elbv2.Listener, targetGroup *elbv2.TargetGroup, clusterName cluster.ClusterName) (err error) {
	if listener != nil {
		err = DeleteListener(*applicationLoadBalancer.LoadBalancerName)
		if err != nil {
			return
		}
	}

	if applicationLoadBalancer != nil {
		err = DeleteApplicationLoadBalancer(*applicationLoadBalancer.LoadBalancerName)
		if err != nil {
			return
		}
	}

	if targetGroup != nil {
		err = DeleteTargetGroup(clusterName)
		if err != nil {
			return
		}
	}

	return
}
