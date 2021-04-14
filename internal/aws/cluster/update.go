package cluster

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/rs/zerolog/log"
	autoscaling2 "wekactl/internal/aws/autoscaling"
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
)

func getHostGroupsParams(clusterName cluster.ClusterName, role common.InstanceRole) (hostGroupsParams []common.HostGroupParams, err error) {
	svcAsg := connectors.GetAWSSession().ASG
	svcEc2 := connectors.GetAWSSession().EC2
	var nextToken *string
	var asgOutput *autoscaling.DescribeAutoScalingGroupsOutput
	var launchTemplateVersionsOutput *ec2.DescribeLaunchTemplateVersionsOutput

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
			asgRole := autoscaling2.GetAsgTagValue(asg, RoleTagKey)
			asgClusterName := autoscaling2.GetAsgTagValue(asg, cluster.ClusterNameTagKey)
			if asgRole != string(role) || asgClusterName != string(clusterName) {
				continue
			}
			log.Debug().Msgf("Generating %s host group params ...", *asg.AutoScalingGroupName)
			launchTemplateVersionsOutput, err = svcEc2.DescribeLaunchTemplateVersions(&ec2.DescribeLaunchTemplateVersionsInput{
				LaunchTemplateName: asg.LaunchTemplate.LaunchTemplateName,
			})
			if err != nil {
				return
			}
			launchTemplateData := launchTemplateVersionsOutput.LaunchTemplateVersions[0].LaunchTemplateData
			hostGroupsParams = append(hostGroupsParams,
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

func getHostGroups(clusterName cluster.ClusterName, role common.InstanceRole, name common.HostGroupName) (hostGroups []HostGroup, err error) {
	hostGroupsParams, err := getHostGroupsParams(clusterName, role)
	if err != nil {
		return
	}
	for _, hostGroupParams := range hostGroupsParams {
		hostGroups = append(hostGroups, GenerateHostGroup(clusterName, hostGroupParams, role, name))
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
