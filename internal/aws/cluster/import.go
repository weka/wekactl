package cluster

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"strings"
	"wekactl/internal/aws/common"
)

type StackInstances struct {
	backends []*ec2.Instance
	clients  []*ec2.Instance
}

var Roles = []string{
	"backends",
	"clients",
}

func getStackId(region, stackName string) string {
	sess := common.NewSession(region)
	svc := cloudformation.New(sess)
	input := &cloudformation.DescribeStacksInput{StackName: &stackName}
	result, err := svc.DescribeStacks(input)
	if err != nil {
		log.Fatal().Err(err)
	}
	return *result.Stacks[0].StackId
}

func getClusterInstances(region, stackName string) []*string {
	sess := common.NewSession(region)
	svc := cloudformation.New(sess)
	input := &cloudformation.DescribeStackResourcesInput{StackName: &stackName}
	result, err := svc.DescribeStackResources(input)
	var instancesIds []*string
	if err != nil {
		log.Fatal().Err(err)
	} else {
		for _, resource := range result.StackResources {
			if *resource.ResourceType == "AWS::EC2::Instance" {
				instancesIds = append(instancesIds, resource.PhysicalResourceId)
			}
		}
	}
	return instancesIds
}

func getInstancesInfo(region, stackName string) StackInstances {
	sess := common.NewSession(region)
	svc := ec2.New(sess)
	input := &ec2.DescribeInstancesInput{
		InstanceIds: getClusterInstances(region, stackName),
	}
	result, err := svc.DescribeInstances(input)
	if err != nil {
		log.Fatal().Err(err)
	}

	stackInstances := StackInstances{}

	for _, reservation := range result.Reservations {
		instance := reservation.Instances[0]
		arn := *instance.IamInstanceProfile.Arn
		if strings.Contains(arn, "InstanceProfileBackend") {
			stackInstances.backends = append(stackInstances.backends, instance)
		} else if strings.Contains(arn, "InstanceProfileClient") {
			stackInstances.clients = append(stackInstances.clients, instance)
		}

	}
	return stackInstances
}

func getInstanceSecurityGroupsId(instance *ec2.Instance) []*string {
	var securityGroupIds []*string
	for _, securityGroup := range instance.SecurityGroups {
		securityGroupIds = append(securityGroupIds, securityGroup.GroupId)
	}
	return securityGroupIds
}

func createLaunchTemplate(region, stackId, stackName, role string, instance *ec2.Instance) string {
	sess := common.NewSession(region)
	svc := ec2.New(sess)
	u := uuid.New().String()
	launchTemplateName := "weka-" + stackName + "-" + role + "-" + u
	input := &ec2.CreateLaunchTemplateInput{
		LaunchTemplateData: &ec2.RequestLaunchTemplateData{
			ImageId:      instance.ImageId,
			InstanceType: instance.InstanceType,
			KeyName:      instance.KeyName,
			UserData:     aws.String(""), // TODO: add necessary init script here
			TagSpecifications: []*ec2.LaunchTemplateTagSpecificationRequest{
				{
					ResourceType: aws.String("instance"),
					Tags: []*ec2.Tag{
						{
							Key:   aws.String("wekactl.io/managed"),
							Value: aws.String("true"),
						},
						{
							Key:   aws.String("wekactl.io/api_version"),
							Value: aws.String("v1"),
						},
						{
							Key:   aws.String("wekactl.io/hostgroup_type"),
							Value: aws.String(strings.TrimSuffix(role, "s")),
						},
						{
							Key:   aws.String("wekactl.io/name"),
							Value: aws.String(role),
						},
						{
							Key:   aws.String("wekactl.io/stack_id"),
							Value: aws.String(stackId),
						},
					},
				},
			},
			NetworkInterfaces: []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{
				{
					AssociatePublicIpAddress: aws.Bool(true),
					DeviceIndex:              aws.Int64(0),
					Ipv6AddressCount:         aws.Int64(0),
					SubnetId:                 instance.SubnetId,
					Groups:                   getInstanceSecurityGroupsId(instance),
				},
			},
		},
		VersionDescription: aws.String("v1"),
		LaunchTemplateName: aws.String(launchTemplateName),
	}

	_, err := svc.CreateLaunchTemplate(input)
	if err != nil {
		log.Fatal().Err(err)
	}
	log.Debug().Msgf("LaunchTemplate: \"%s\" was created sucessfully!", launchTemplateName)
	return launchTemplateName
}

func createAutoScalingGroup(region, stackId, stackName, role string, roleInstances []*ec2.Instance) (string, error) {
	if len(roleInstances) > 0 {
		launchTemplateName := createLaunchTemplate(region, stackId, stackName, role, roleInstances[0])
		instancesNumber := int64(len(roleInstances))
		sess := common.NewSession(region)
		svc := autoscaling.New(sess)
		u := uuid.New().String()
		name := "weka-" + stackName + "-" + role + "-" + u
		input := &autoscaling.CreateAutoScalingGroupInput{
			AutoScalingGroupName: aws.String(name),
			LaunchTemplate: &autoscaling.LaunchTemplateSpecification{
				LaunchTemplateName: aws.String(launchTemplateName),
				Version:            aws.String("1"),
			},
			MinSize: aws.Int64(0),
			MaxSize: aws.Int64(instancesNumber),
			Tags: []*autoscaling.Tag{
				{
					Key:   aws.String("wekactl.io/managed"),
					Value: aws.String("true"),
				},
				{
					Key:   aws.String("wekactl.io/api_version"),
					Value: aws.String("v1"),
				},
				{
					Key:   aws.String("wekactl.io/hostgroup_type"),
					Value: aws.String(strings.TrimSuffix(role, "s")),
				},
				{
					Key:   aws.String("wekactl.io/name"),
					Value: aws.String(role),
				},
				{
					Key:   aws.String("wekactl.io/stack_id"),
					Value: aws.String(stackId),
				},
			},
		}
		_, err := svc.CreateAutoScalingGroup(input)
		if err != nil {
			return "", err
		}
		log.Debug().Msgf("AutoScalingGroup: \"%s\" was created sucessfully!", name)
		return name, nil
	} else {
		fmt.Printf("No %s where found\n", strings.Title(role))
		return "", nil
	}
}

func getInstancesIdsFromEc2Instance(instances []*ec2.Instance) []*string {
	var instanceIds []*string
	for _, instance := range instances {
		instanceIds = append(instanceIds, instance.InstanceId)
	}
	return instanceIds
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func attachInstancesToAutoScalingGroups(region string, roleInstances []*ec2.Instance, autoScalingGroupsName string) {
	sess := common.NewSession(region)
	svc := autoscaling.New(sess)
	limit := 20
	instancesIds := getInstancesIdsFromEc2Instance(roleInstances)
	for i := 0; i < len(instancesIds); i += limit {
		batch := instancesIds[i:min(i+limit, len(instancesIds))]
		input := &autoscaling.AttachInstancesInput{
			AutoScalingGroupName: &autoScalingGroupsName,
			InstanceIds:          batch,
		}
		_, err := svc.AttachInstances(input)
		if err != nil {
			fmt.Println(err)
		} else {
			log.Debug().Msgf("Attaced %d instances to %s successfully!", len(batch), autoScalingGroupsName)
		}
	}
}

func importClusterRole(region, stackId, stackName, role string, roleInstances []*ec2.Instance) {
	autoScalingGroupName, err := createAutoScalingGroup(region, stackId, stackName, role, roleInstances)
	if err != nil {
		fmt.Println(err)
		return
	}
	attachInstancesToAutoScalingGroups(region, roleInstances, autoScalingGroupName)
}

func ImportCluster(region, stackName string) {
	stackInstances := getInstancesInfo(region, stackName)
	stackId := getStackId(region, stackName)
	importClusterRole(region, stackId, stackName, "clients", stackInstances.clients)
	importClusterRole(region, stackId, stackName, "backends", stackInstances.backends)
}
