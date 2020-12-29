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

type InstancesMap map[string][]*ec2.Instance
type AutoScalingGroupMap map[string]string

var Roles = []string{
	"backends",
	"clients",
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

func getInstancesInfo(region, stackName string) InstancesMap {
	sess := common.NewSession(region)
	svc := ec2.New(sess)
	input := &ec2.DescribeInstancesInput{
		InstanceIds: getClusterInstances(region, stackName),
	}
	result, err := svc.DescribeInstances(input)
	if err != nil {
		log.Fatal().Err(err)
	}

	clusterInstances := make(InstancesMap)
	var backends []*ec2.Instance
	var clients []*ec2.Instance

	for _, reservation := range result.Reservations {
		instance := reservation.Instances[0]
		arn := *instance.IamInstanceProfile.Arn
		if strings.Contains(arn, "InstanceProfileBackend") {
			backends = append(backends, instance)
		} else if strings.Contains(arn, "InstanceProfileClient") {
			clients = append(clients, instance)
		}

	}

	clusterInstances["backends"] = backends
	clusterInstances["clients"] = clients
	return clusterInstances
}

func getInstanceSecurityGroupsId(instance *ec2.Instance) []*string {
	var securityGroupIds []*string
	for _, securityGroup := range instance.SecurityGroups {
		securityGroupIds = append(securityGroupIds, securityGroup.GroupId)
	}
	return securityGroupIds
}

func createLaunchTemplate(region, stackName, role string, instance *ec2.Instance) string {
	sess := common.NewSession(region)
	svc := ec2.New(sess)
	u := uuid.New().String()
	launchTemplateName := "weka-" + stackName + "-" + role + "-" + u
	input := &ec2.CreateLaunchTemplateInput{
		LaunchTemplateData: &ec2.RequestLaunchTemplateData{
			ImageId:      instance.ImageId,
			InstanceType: instance.InstanceType,
			KeyName:      instance.KeyName,
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

func createAutoScalingGroup(region, stackName string, clusterInstances InstancesMap) AutoScalingGroupMap {
	autoScalingGroupsNames := make(AutoScalingGroupMap)
	for _, role := range Roles {
		if len(clusterInstances[role]) > 0 {
			launchTemplateName := createLaunchTemplate(region, stackName, role, clusterInstances[role][0])
			instancesNumber := int64(len(clusterInstances[role]))
			sess := common.NewSession(region)
			svc := autoscaling.New(sess)
			u := uuid.New().String()
			name := "weka-" + stackName + "-" + role + "-" + u
			input := &autoscaling.CreateAutoScalingGroupInput{
				AutoScalingGroupName: aws.String(name),
				LaunchTemplate: &autoscaling.LaunchTemplateSpecification{
					LaunchTemplateName: aws.String(launchTemplateName),
					Version: aws.String("1"),
				},
				MinSize:         aws.Int64(0),
				MaxSize:         aws.Int64(instancesNumber),
			}
			_, err := svc.CreateAutoScalingGroup(input)
			if err != nil {
				log.Fatal().Err(err)
			}
			autoScalingGroupsNames[role] = name
			log.Debug().Msgf("AutoScalingGroup: \"%s\" was created sucessfully!", name)
		} else {
			fmt.Printf("No %s where found\n", strings.Title(role))
		}
	}

	return autoScalingGroupsNames
}

func getInstancesIdsFromEc2Instance(instances []*ec2.Instance) []*string {
	var instanceIds []*string
	for _, instance := range instances {
		instanceIds = append(instanceIds, instance.InstanceId)
	}
	return instanceIds
}

func attachInstancesToAutoScalingGroups(region string, clusterInstances InstancesMap, autoScalingGroupsNames AutoScalingGroupMap) {
	sess := common.NewSession(region)
	svc := autoscaling.New(sess)
	for _, role := range Roles {
		if name, ok := autoScalingGroupsNames[role]; ok {
			input := &autoscaling.AttachInstancesInput{
				AutoScalingGroupName: &name,
				InstanceIds:          getInstancesIdsFromEc2Instance(clusterInstances[role]),
			}
			_, err := svc.AttachInstances(input)
			if err != nil {
				log.Fatal().Err(err)
			}
			log.Debug().Msgf("Attaced %d instances to %s successfully!", len(clusterInstances[role]), name)
		}
	}
}

func ImportCluster(region, stackName string) {
	clusterInstances := getInstancesInfo(region, stackName)
	autoScalingGroupsNames := createAutoScalingGroup(region, stackName, clusterInstances)
	attachInstancesToAutoScalingGroups(region, clusterInstances, autoScalingGroupsNames)
}