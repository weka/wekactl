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
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/connectors"
)

func getEc2Tags(hostGroupInfo hostgroups.HostGroupInfo) []*ec2.Tag {
	var ec2Tags []*ec2.Tag
	for key, value := range common.GetHostGroupTags(hostGroupInfo) {
		ec2Tags = append(ec2Tags, &ec2.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return ec2Tags
}

func generateBlockDeviceMappingRequest(name hostgroups.HostGroupName, volumeInfo VolumeInfo) []*ec2.LaunchTemplateBlockDeviceMappingRequest {

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

func CreateLaunchTemplate(hostGroupInfo hostgroups.HostGroupInfo, hostGroupParams hostgroups.HostGroupParams, restApiGateway apigateway.RestApiGateway, launchTemplateName string) (err error) {
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
			BlockDeviceMappings: generateBlockDeviceMappingRequest(hostGroupInfo.Name, VolumeInfo{
				Name: hostGroupParams.VolumeName,
				Type: hostGroupParams.VolumeType,
				Size: hostGroupParams.VolumeSize,
			}),
			TagSpecifications: []*ec2.LaunchTemplateTagSpecificationRequest{
				{
					ResourceType: aws.String("instance"),
					Tags:         getEc2Tags(hostGroupInfo),
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
			if aerr.Code() == "InvalidLaunchTemplateName.NotFoundException"{
				return nil
			}
		}
		return err
	} else {
		log.Debug().Msgf("launch template %s was deleted successfully", launchTemplateName)
	}
	return nil
}
