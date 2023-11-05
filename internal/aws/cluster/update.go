package cluster

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/rs/zerolog/log"
	autoscaling2 "wekactl/internal/aws/autoscaling"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/launchtemplate"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
	"wekactl/internal/env"
)

const defaultVolumeSize = 48
const tracesPerIonode = 10
const defaultDeviceName = "/dev/sdp"

type hostGroupGenerationInfo struct {
	params common.HostGroupParams
	name   common.HostGroupName
	role   common.InstanceRole
}

func generateVolumesInfo(clusterName cluster.ClusterName, asg *autoscaling.Group, launchTemplateBlockDeviceMapping []*ec2.LaunchTemplateBlockDeviceMapping) (volumesInfo []common.VolumeInfo, err error) {
	log.Debug().Msgf("Retrieving %s volumes info ...", clusterName)

	// if we enter here it means we're already dealing with new volumes partition launch template
	if len(launchTemplateBlockDeviceMapping) > 1 {
		for _, blockDevice := range launchTemplateBlockDeviceMapping {
			volumesInfo = append(volumesInfo, common.VolumeInfo{
				Name: *blockDevice.DeviceName,
				Type: *blockDevice.Ebs.VolumeType,
				Size: *blockDevice.Ebs.VolumeSize,
			})
		}
		return
	}

	if len(asg.Instances) == 0 {
		err = errors.New(fmt.Sprintf("no instances in cluster"))
		return
	}
	asgInstance := asg.Instances[0]
	instanceType := *asgInstance.InstanceType

	wekaVolumeSize := defaultVolumeSize + common.GetIoNodesNumber(instanceType)*tracesPerIonode

	rootDeviceDeviceName := *launchTemplateBlockDeviceMapping[0].DeviceName
	rootDeviceSize := *launchTemplateBlockDeviceMapping[0].Ebs.VolumeSize - int64(wekaVolumeSize)
	if rootDeviceSize < common.RootFsMinimalSize {
		rootDeviceSize = common.RootFsMinimalSize
	}

	volumeType := common.GetVolumeType(env.Config.Region)

	volumesInfo = []common.VolumeInfo{
		{
			Name: rootDeviceDeviceName,
			Type: volumeType,
			Size: rootDeviceSize,
		},
		{
			Name: defaultDeviceName,
			Type: volumeType,
			Size: int64(wekaVolumeSize),
		},
	}
	return
}

func fetchHostGroupParams(clusterName cluster.ClusterName, asg *autoscaling.Group) (hostGroupParams common.HostGroupParams, err error) {
	launchTemplateVersion, err := launchtemplate.GetLatestLaunchTemplateVersion(*asg.LaunchTemplate.LaunchTemplateName)
	if err != nil {
		return
	}
	launchTemplateData := launchTemplateVersion.LaunchTemplateData

	volumesInfo, err := generateVolumesInfo(clusterName, asg, launchTemplateData.BlockDeviceMappings)
	if err != nil {
		return
	}

	hostGroupParams = common.HostGroupParams{
		SecurityGroupsIds: launchTemplateData.NetworkInterfaces[0].Groups,
		ImageID:           *launchTemplateData.ImageId,
		IamArn:            *launchTemplateData.IamInstanceProfile.Arn,
		InstanceType:      *launchTemplateData.InstanceType,
		Subnet:            *launchTemplateData.NetworkInterfaces[0].SubnetId,
		VolumesInfo:       volumesInfo,
		MaxSize:           *asg.MaxSize,
		HttpTokens:        *launchTemplateData.MetadataOptions.HttpTokens,
	}

	if launchTemplateData.KeyName != nil {
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
				params, err = fetchHostGroupParams(clusterName, asg)
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
