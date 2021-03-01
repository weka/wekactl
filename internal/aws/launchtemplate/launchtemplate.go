package launchtemplate

import (
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/lithammer/dedent"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/apigateway"
	cluster2 "wekactl/internal/aws/cluster"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
)

func generateBlockDeviceMappingRequest(name cluster2.HostGroupName, volumeInfo VolumeInfo) []*ec2.LaunchTemplateBlockDeviceMappingRequest {

	log.Debug().Msgf("%s launch template total root device volume size: %d", string(name), volumeInfo.Size)

	return []*ec2.LaunchTemplateBlockDeviceMappingRequest{
		{
			DeviceName: &volumeInfo.Name,
			Ebs: &ec2.LaunchTemplateEbsBlockDeviceRequest{
				VolumeType:          &volumeInfo.Type,
				VolumeSize:          &volumeInfo.Size,
				DeleteOnTermination: aws.Bool(true),
			},
		},
	}
}

func CreateLaunchTemplate(tags []*ec2.Tag, hostGroupName cluster2.HostGroupName, hostGroupParams cluster2.HostGroupParams, restApiGateway apigateway.RestApiGateway, launchTemplateName string) (err error) {
	svc := connectors.GetAWSSession().EC2
	userDataTemplate := `
	#!/usr/bin/env bash
	
	if ! curl --location --request GET '%s' --header 'x-api-key: %s' | sudo sh; then
		shutdown now
	fi
	`

	userData := fmt.Sprintf(dedent.Dedent(userDataTemplate), restApiGateway.Url, restApiGateway.ApiKey)
	input := &ec2.CreateLaunchTemplateInput{
		LaunchTemplateData: &ec2.RequestLaunchTemplateData{
			ImageId:               &hostGroupParams.ImageID,
			InstanceType:          &hostGroupParams.InstanceType,
			KeyName:               &hostGroupParams.KeyName,
			UserData:              aws.String(base64.StdEncoding.EncodeToString([]byte(userData))),
			DisableApiTermination: aws.Bool(true),
			IamInstanceProfile: &ec2.LaunchTemplateIamInstanceProfileSpecificationRequest{
				Arn: &hostGroupParams.IamArn,
			},
			BlockDeviceMappings: generateBlockDeviceMappingRequest(hostGroupName, VolumeInfo{
				Name: hostGroupParams.VolumeName,
				Type: hostGroupParams.VolumeType,
				Size: hostGroupParams.VolumeSize,
			}),
			TagSpecifications: []*ec2.LaunchTemplateTagSpecificationRequest{
				{
					ResourceType: aws.String("instance"),
					Tags:         tags,
				},
			},
			NetworkInterfaces: []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{
				{
					AssociatePublicIpAddress: aws.Bool(true),
					DeviceIndex:              aws.Int64(0),
					Ipv6AddressCount:         aws.Int64(0),
					SubnetId:                 &hostGroupParams.Subnet,
					Groups:                   hostGroupParams.SecurityGroupsIds,
				},
			},
		},
		VersionDescription: aws.String("v1"),
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
	svc := connectors.GetAWSSession().EC2
	launchTemplateOutput, err := svc.DescribeLaunchTemplates(&ec2.DescribeLaunchTemplatesInput{
		LaunchTemplateNames: []*string{&launchTemplateName},
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "InvalidLaunchTemplateName.NotFoundException" {
				return "", nil
			}
		}
		return
	}

	for _, lt := range launchTemplateOutput.LaunchTemplates {
		for _, tag := range lt.Tags {
			if *tag.Key == cluster.VersionTagKey {
				version = *tag.Value
				return
			}
		}
	}

	return
}
