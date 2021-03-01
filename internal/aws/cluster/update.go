package cluster

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
)

func fetchClusterLaunchTemplateParams(clusterName cluster.ClusterName, name common.HostGroupName) (hostGroupParams common.HostGroupParams, err error) {
	resourceName := common.GenerateResourceName(clusterName, name)
	svc := connectors.GetAWSSession().EC2

	launchTemplateVersionsOutput, err := svc.DescribeLaunchTemplateVersions(&ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateName: &resourceName,
	})

	if err != nil {
		return
	}

	launchTemplateData := launchTemplateVersionsOutput.LaunchTemplateVersions[0].LaunchTemplateData

	maxSize := int64(1000)
	svcAsg := connectors.GetAWSSession().ASG
	asgOutput, err := svcAsg.DescribeAutoScalingGroups(
		&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{&resourceName},
		},
	)
	if err == nil && len(asgOutput.AutoScalingGroups) > 0 {
		maxSize = *asgOutput.AutoScalingGroups[0].MaxSize
	}

	hostGroupParams = common.HostGroupParams{
		SecurityGroupsIds: launchTemplateData.SecurityGroupIds,
		ImageID:           *launchTemplateData.ImageId,
		KeyName:           *launchTemplateData.KeyName,
		IamArn:            *launchTemplateData.IamInstanceProfile.Arn,
		InstanceType:      *launchTemplateData.InstanceType,
		Subnet:            *launchTemplateData.NetworkInterfaces[0].SubnetId,
		VolumeName:        *launchTemplateData.BlockDeviceMappings[0].DeviceName,
		VolumeType:        *launchTemplateData.BlockDeviceMappings[0].Ebs.VolumeType,
		VolumeSize:        *launchTemplateData.BlockDeviceMappings[0].Ebs.VolumeSize,
		MaxSize:           maxSize,
	}

	return
}

func GenerateHostGroupFromLaunchTemplate(clusterName cluster.ClusterName, role common.InstanceRole, name common.HostGroupName) (hostGroup HostGroup, err error) {
	hostGroupParams, err := fetchClusterLaunchTemplateParams(clusterName, name)
	if err != nil {
		return
	}
	hostGroup = GenerateHostGroup(clusterName, hostGroupParams, role, name)
	return
}
