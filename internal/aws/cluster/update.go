package cluster

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
)

func fetchClusterLaunchTemplateParams(clusterName cluster.ClusterName, name hostgroups.HostGroupName) (hostGroupParams hostgroups.HostGroupParams, err error) {
	launchTemplateName := common.GenerateResourceName(clusterName, name)
	svc := connectors.GetAWSSession().EC2

	launchTemplateVersionsOutput, err := svc.DescribeLaunchTemplateVersions(&ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateName: &launchTemplateName,
	})

	if err != nil {
		return
	}

	launchTemplateData := launchTemplateVersionsOutput.LaunchTemplateVersions[0].LaunchTemplateData

	hostGroupParams = hostgroups.HostGroupParams{
		SecurityGroupsIds: launchTemplateData.SecurityGroupIds,
		ImageID:           *launchTemplateData.ImageId,
		KeyName:           *launchTemplateData.KeyName,
		IamArn:            *launchTemplateData.IamInstanceProfile.Arn,
		InstanceType:      *launchTemplateData.InstanceType,
		Subnet:            *launchTemplateData.NetworkInterfaces[0].SubnetId,
		VolumeName:        *launchTemplateData.BlockDeviceMappings[0].DeviceName,
		VolumeType:        *launchTemplateData.BlockDeviceMappings[0].Ebs.VolumeType,
		VolumeSize:        *launchTemplateData.BlockDeviceMappings[0].Ebs.VolumeSize,
	}

	return
}

func GenerateHostGroupFromLaunchTemplate(clusterName cluster.ClusterName, role hostgroups.InstanceRole, name hostgroups.HostGroupName) (hostGroup HostGroup, err error) {
	hostGroupParams, err := fetchClusterLaunchTemplateParams(clusterName, name)
	if err != nil {
		return
	}
	hostGroup = GenerateHostGroup(clusterName, hostGroupParams, role, name)
	return
}
