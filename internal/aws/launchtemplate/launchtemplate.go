package launchtemplate

import (
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/lithammer/dedent"
	"github.com/rs/zerolog/log"
	"strconv"
	"wekactl/internal/aws/apigateway"
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
	"wekactl/internal/env"
)

const LaunchtemplateVersion = "v2"

func generateBlockDeviceMappingRequest(name common.HostGroupName, volumesInfo []common.VolumeInfo) (request []*ec2.LaunchTemplateBlockDeviceMappingRequest) {
	log.Debug().Msgf("generating %s launch template block device mapping", string(name))
	for i := 0; i < len(volumesInfo); i++ {
		volumeInfo := volumesInfo[i]
		request = append(request, &ec2.LaunchTemplateBlockDeviceMappingRequest{
			DeviceName: &volumeInfo.Name,
			Ebs: &ec2.LaunchTemplateEbsBlockDeviceRequest{
				VolumeType:          &volumeInfo.Type,
				VolumeSize:          &volumeInfo.Size,
				DeleteOnTermination: aws.Bool(true),
			},
		})
	}

	return
}

func getUserData(restApiGateway apigateway.RestApiGateway, subnetId, instanceType string, securityGroupsIds []*string) string {
	securityGroupsIdsStr := ""
	for _, securityGroupsId := range securityGroupsIds {
		securityGroupsIdsStr = securityGroupsIdsStr + *securityGroupsId + " "
	}

	additionalNicsNum := common.GetIoNodesNumber(instanceType)

	userDataTemplate := `
	#!/usr/bin/env bash
	set -ex
	
	region=%s
	subnet_id=%s
	groups=%s
	nics_num=%d
	join_url=%s
	api_key=%s

	token=$(curl -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")
	instance_id=$(curl -H "X-aws-ec2-metadata-token: $token" http://169.254.169.254/latest/meta-data/instance-id)

	for (( i=1; i<=$nics_num; i++ ))
	do
		eni_file="/tmp/eni-$i.json"
		aws ec2 create-network-interface --region $region --subnet-id $subnet_id  --groups $groups > $eni_file
		network_interface_id=$(cat $eni_file | python3 -c "import sys, json; print(json.load(sys.stdin)['NetworkInterface']['NetworkInterfaceId'])")
		aws ec2 attach-network-interface --region $region --device-index $i --instance-id $instance_id --network-interface-id $network_interface_id
	done

	if ! curl --location --request GET "$join_url" --header "x-api-key: $api_key" | sudo sh; then
		shutdown now
	fi
	`
	return fmt.Sprintf(
		dedent.Dedent(userDataTemplate),
		env.Config.Region,
		subnetId,
		securityGroupsIdsStr,
		additionalNicsNum,
		restApiGateway.Url(),
		restApiGateway.ApiKey,
	)
}

func getKeyName(keyName string) *string {
	if keyName == "" {
		return nil
	} else {
		return &keyName
	}
}

func CreateLaunchTemplate(tags []*ec2.Tag, hostGroupName common.HostGroupName, hostGroupParams common.HostGroupParams, restApiGateway apigateway.RestApiGateway, launchTemplateName string, associatePublicIpAddress bool) (err error) {
	svc := connectors.GetAWSSession().EC2
	userData := getUserData(restApiGateway, hostGroupParams.Subnet, hostGroupParams.InstanceType, hostGroupParams.SecurityGroupsIds)
	keyName := getKeyName(hostGroupParams.KeyName)

	input := &ec2.CreateLaunchTemplateInput{
		LaunchTemplateData: &ec2.RequestLaunchTemplateData{
			ImageId:               &hostGroupParams.ImageID,
			InstanceType:          &hostGroupParams.InstanceType,
			KeyName:               keyName,
			UserData:              aws.String(base64.StdEncoding.EncodeToString([]byte(userData))),
			DisableApiTermination: aws.Bool(true),
			IamInstanceProfile: &ec2.LaunchTemplateIamInstanceProfileSpecificationRequest{
				Arn: &hostGroupParams.IamArn,
			},
			BlockDeviceMappings: generateBlockDeviceMappingRequest(hostGroupName, hostGroupParams.VolumesInfo),
			TagSpecifications: []*ec2.LaunchTemplateTagSpecificationRequest{
				{
					ResourceType: aws.String("instance"),
					Tags:         tags,
				},
			},
			NetworkInterfaces: []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{
				{
					AssociatePublicIpAddress: aws.Bool(associatePublicIpAddress),
					DeviceIndex:              aws.Int64(0),
					Ipv6AddressCount:         aws.Int64(0),
					SubnetId:                 &hostGroupParams.Subnet,
					Groups:                   hostGroupParams.SecurityGroupsIds,
				},
			},
			MetadataOptions: &ec2.LaunchTemplateInstanceMetadataOptionsRequest{
				HttpTokens: &hostGroupParams.HttpTokens,
			},
		},
		VersionDescription: aws.String(LaunchtemplateVersion),
		LaunchTemplateName: aws.String(launchTemplateName),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("launch-template"),
				Tags:         tags,
			},
		},
	}

	_, err = svc.CreateLaunchTemplate(input)
	if err != nil {
		return
	}
	log.Debug().Msgf("LaunchTemplate: \"%s\" was created successfully!", launchTemplateName)
	return
}

func GetLatestLaunchTemplateVersion(launchTemplateName string) (launchTemplateVersion *ec2.LaunchTemplateVersion, err error) {
	svc := connectors.GetAWSSession().EC2
	launchTemplateVersionsOutput, err := svc.DescribeLaunchTemplateVersions(&ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateName: &launchTemplateName,
		Versions:           []*string{aws.String("$Latest")},
	})
	if err != nil {
		return
	}

	launchTemplateVersion = launchTemplateVersionsOutput.LaunchTemplateVersions[0]
	return
}

func CreateNewLaunchTemplateVersion(tags []*ec2.Tag, hostGroupName common.HostGroupName, hostGroupParams common.HostGroupParams, restApiGateway apigateway.RestApiGateway, launchTemplateName string, associatePublicIpAddress bool) (newVersion string, err error) {
	svc := connectors.GetAWSSession().EC2
	launchTemplateVersion, err := GetLatestLaunchTemplateVersion(launchTemplateName)
	if err != nil {
		return
	}

	userData := getUserData(restApiGateway, hostGroupParams.Subnet, hostGroupParams.InstanceType, hostGroupParams.SecurityGroupsIds)
	keyName := getKeyName(hostGroupParams.KeyName)

	input := &ec2.CreateLaunchTemplateVersionInput{
		LaunchTemplateData: &ec2.RequestLaunchTemplateData{
			ImageId:               &hostGroupParams.ImageID,
			InstanceType:          &hostGroupParams.InstanceType,
			KeyName:               keyName,
			UserData:              aws.String(base64.StdEncoding.EncodeToString([]byte(userData))),
			DisableApiTermination: aws.Bool(true),
			IamInstanceProfile: &ec2.LaunchTemplateIamInstanceProfileSpecificationRequest{
				Arn: &hostGroupParams.IamArn,
			},
			BlockDeviceMappings: generateBlockDeviceMappingRequest(hostGroupName, hostGroupParams.VolumesInfo),
			TagSpecifications: []*ec2.LaunchTemplateTagSpecificationRequest{
				{
					ResourceType: aws.String("instance"),
					Tags:         tags,
				},
			},
			NetworkInterfaces: []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{
				{
					AssociatePublicIpAddress: aws.Bool(associatePublicIpAddress),
					DeviceIndex:              aws.Int64(0),
					Ipv6AddressCount:         aws.Int64(0),
					SubnetId:                 &hostGroupParams.Subnet,
					Groups:                   hostGroupParams.SecurityGroupsIds,
				},
			},
		},
		VersionDescription: aws.String(LaunchtemplateVersion),
		LaunchTemplateName: aws.String(launchTemplateName),
		SourceVersion:      aws.String(strconv.Itoa(int(*launchTemplateVersion.VersionNumber))),
	}

	launchTemplateVersionOutput, err := svc.CreateLaunchTemplateVersion(input)
	if err != nil {
		return
	}
	newVersion = strconv.Itoa(int(*launchTemplateVersionOutput.LaunchTemplateVersion.VersionNumber))
	log.Debug().Msgf(
		"New LaunchTemplateVersion: \"%s\" version \"%s\" was created successfully!",
		launchTemplateName, LaunchtemplateVersion)
	return
}

func ModifyLaunchTemplateDefaultVersion(launchTemplateName, newVersion string) (err error) {
	svc := connectors.GetAWSSession().EC2
	_, err = svc.ModifyLaunchTemplate(&ec2.ModifyLaunchTemplateInput{
		LaunchTemplateName: &launchTemplateName,
		DefaultVersion:     &newVersion,
	})

	return
}

func DeleteLaunchTemplate(launchTemplateName string) error {
	svc := connectors.GetAWSSession().EC2
	_, err := svc.DeleteLaunchTemplate(&ec2.DeleteLaunchTemplateInput{
		LaunchTemplateName: &launchTemplateName,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "InvalidLaunchTemplateName.NotFoundException" {
				return nil
			}
		}
		return err
	} else {
		log.Debug().Msgf("launch template %s was deleted successfully", launchTemplateName)
	}
	return nil
}

func GetLaunchTemplateVersion(launchTemplateName string) (version string, err error) {
	launchTemplateVersion, err := GetLatestLaunchTemplateVersion(launchTemplateName)
	if err != nil {
		return
	}

	for _, lt := range launchTemplateVersion.LaunchTemplateData.TagSpecifications {
		for _, tag := range lt.Tags {
			if *tag.Key == cluster.VersionTagKey {
				version = *tag.Value
				return
			}
		}
	}

	return
}

func GetClusterLaunchTemplates(clusterName cluster.ClusterName) (launchTemplates []*ec2.LaunchTemplate, err error) {
	svc := connectors.GetAWSSession().EC2
	launchTemplateOutput, err := svc.DescribeLaunchTemplates(&ec2.DescribeLaunchTemplatesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:wekactl.io/cluster_name"),
				Values: []*string{
					aws.String(string(clusterName)),
				},
			},
		},
	})
	if err != nil {
		return
	}
	launchTemplates = launchTemplateOutput.LaunchTemplates
	return
}

func DeleteLaunchTemplates(launchTemplates []*ec2.LaunchTemplate) error {
	for _, launchTemplate := range launchTemplates {
		err := DeleteLaunchTemplate(*launchTemplate.LaunchTemplateName)
		if err != nil {
			return err
		}
	}
	return nil
}
