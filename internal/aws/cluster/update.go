package cluster

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/rs/zerolog/log"
	autoscaling2 "wekactl/internal/aws/autoscaling"
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
)

type HostGroupsParamsMap map[common.InstanceRole][]common.HostGroupParams

func getHostGroupsParams(clusterName cluster.ClusterName) (hostGroupsParamsMap HostGroupsParamsMap, err error) {
	svcAsg := connectors.GetAWSSession().ASG
	svcEc2 := connectors.GetAWSSession().EC2
	var nextToken *string
	var asgOutput *autoscaling.DescribeAutoScalingGroupsOutput
	var launchTemplateVersionsOutput *ec2.DescribeLaunchTemplateVersionsOutput

	hostGroupsParamsMap = make(HostGroupsParamsMap)

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
			var role common.InstanceRole
			asgRole := autoscaling2.GetAsgTagValue(asg, RoleTagKey)
			asgClusterName := autoscaling2.GetAsgTagValue(asg, cluster.ClusterNameTagKey)
			if asgClusterName != string(clusterName) || asgRole == "" {
				continue
			}

			switch asgRole {
			case string(common.RoleBackend):
				role = common.RoleBackend
			case string(common.RoleClient):
				role = common.RoleClient
			default:
				err = errors.New(fmt.Sprintf("Non recognized role %s found", asgRole))
			}

			log.Debug().Msgf("Generating host group params using '%s' ASG ...", *asg.AutoScalingGroupName)
			launchTemplateVersionsOutput, err = svcEc2.DescribeLaunchTemplateVersions(&ec2.DescribeLaunchTemplateVersionsInput{
				LaunchTemplateName: asg.LaunchTemplate.LaunchTemplateName,
			})
			if err != nil {
				return
			}
			launchTemplateData := launchTemplateVersionsOutput.LaunchTemplateVersions[0].LaunchTemplateData
			hostGroupsParamsMap[role] = append(hostGroupsParamsMap[role],
				common.HostGroupParams{
					SecurityGroupsIds: launchTemplateData.SecurityGroupIds,
					ImageID:           *launchTemplateData.ImageId,
					KeyName:           *launchTemplateData.KeyName,
					IamArn:            *launchTemplateData.IamInstanceProfile.Arn,
					InstanceType:      *launchTemplateData.InstanceType,
					Subnet:            *launchTemplateData.NetworkInterfaces[0].SubnetId,
					VolumeName:        *launchTemplateData.BlockDeviceMappings[0].DeviceName,
					VolumeType:        *launchTemplateData.BlockDeviceMappings[0].Ebs.VolumeType,
					VolumeSize:        *launchTemplateData.BlockDeviceMappings[0].Ebs.VolumeSize,
					MaxSize:           *asg.MaxSize,
				})
		}
		nextToken = asgOutput.NextToken
	}
	return
}

func roleToName(role common.InstanceRole) (name common.HostGroupName) {
	switch role {
	case "backend":
		name = "Backends"
	case "client":
		name = "Clients"
	}
	return
}

func getHostGroups(clusterName cluster.ClusterName) (hostGroups []HostGroup, err error) {
	hostGroupsParamsMap, err := getHostGroupsParams(clusterName)
	if err != nil {
		return
	}

	for role, hostGroupsParams := range hostGroupsParamsMap {
		for _, hostGroupParams := range hostGroupsParams {
			hostGroups = append(hostGroups, GenerateHostGroup(clusterName, hostGroupParams, role, roleToName(role)))
		}
	}
	return
}

func UpdateCluster(name cluster.ClusterName) error {
	awsCluster, err := GetCluster(name)
	if err != nil {
		return err
	}

	return cluster.EnsureResource(&awsCluster, awsCluster.ClusterSettings)
}
