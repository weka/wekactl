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

func getUserData(restApiGateway apigateway.RestApiGateway) string {
	userDataTemplate := `
	#!/usr/bin/env bash
	
	if ! curl --location --request GET '%s' --header 'x-api-key: %s' | sudo sh; then
		shutdown now
	fi
	`
	return fmt.Sprintf(dedent.Dedent(userDataTemplate), restApiGateway.Url(), restApiGateway.ApiKey)
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
	userData := getUserData(restApiGateway)
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

	userData := getUserData(restApiGateway)
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
