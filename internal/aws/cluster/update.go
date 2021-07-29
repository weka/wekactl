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

type hostGroupGenerationInfo struct {
	params common.HostGroupParams
	name   common.HostGroupName
	role   common.InstanceRole
}

func fetchHostGroupParams(asg *autoscaling.Group) (hostGroupParams common.HostGroupParams, err error) {
	svcEc2 := connectors.GetAWSSession().EC2

	launchTemplateVersionsOutput, err := svcEc2.DescribeLaunchTemplateVersions(&ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateName: asg.LaunchTemplate.LaunchTemplateName,
	})
	if err != nil {
		return
	}

	launchTemplateData := launchTemplateVersionsOutput.LaunchTemplateVersions[0].LaunchTemplateData




	hostGroupParams = common.HostGroupParams{
		SecurityGroupsIds: launchTemplateData.SecurityGroupIds,
		ImageID:           *launchTemplateData.ImageId,
		IamArn:            *launchTemplateData.IamInstanceProfile.Arn,
		InstanceType:      *launchTemplateData.InstanceType,
		Subnet:            *launchTemplateData.NetworkInterfaces[0].SubnetId,
		VolumeName:        *launchTemplateData.BlockDeviceMappings[0].DeviceName,
		VolumeType:        *launchTemplateData.BlockDeviceMappings[0].Ebs.VolumeType,
		VolumeSize:        *launchTemplateData.BlockDeviceMappings[0].Ebs.VolumeSize,
		MaxSize:           *asg.MaxSize,
	}

	if launchTemplateData.KeyName != nil{
		hostGroupParams.KeyName = *launchTemplateData.KeyName
	}
	return
}

func getHostGroupsGenerationInfo(clusterName cluster.ClusterName, fetchHotGroupParams bool) (hostGroupsGenerationInfo []hostGroupGenerationInfo, err error) {
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
			var role common.InstanceRole
			asgRole := autoscaling2.GetAsgTagValue(asg, RoleTagKey)
			asgClusterName := autoscaling2.GetAsgTagValue(asg, cluster.ClusterNameTagKey)
			hostGroupName := autoscaling2.GetAsgTagValue(asg, HostGroupNameTagKey)
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

			params := common.HostGroupParams{}
			if fetchHotGroupParams {
				params, err = fetchHostGroupParams(asg)
				if err != nil {
					return
				}
			}

			hostGroupsGenerationInfo = append(hostGroupsGenerationInfo,
				hostGroupGenerationInfo{
					params: params,
					name:   common.HostGroupName(hostGroupName),
					role:   role,
				},
			)
		}
		nextToken = asgOutput.NextToken
	}
	return
}

func getHostGroups(clusterName cluster.ClusterName, fetchHotGroupParams bool) (hostGroups []HostGroup, err error) {
	hostGroupsGenerationInfo, err := getHostGroupsGenerationInfo(clusterName, fetchHotGroupParams)
	if err != nil {
		return
	}

	for _, generationInfo := range hostGroupsGenerationInfo {
		hostGroups = append(hostGroups, GenerateHostGroup(
			clusterName, generationInfo.params, generationInfo.role, generationInfo.name))
	}
	return
}

func UpdateCluster(name cluster.ClusterName, dryRun bool) error {
	awsCluster, err := GetCluster(name, true)
	if err != nil {
		return err
	}

	return cluster.EnsureResource(&awsCluster, awsCluster.ClusterSettings, dryRun)
}
