package cluster

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/apigateway"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/aws/launchtemplate"
	"wekactl/internal/cluster"
)

type LaunchTemplate struct {
	HostGroupInfo   common.HostGroupInfo
	HostGroupParams common.HostGroupParams
	JoinApi         ApiGateway
	TableName       string
	Version         string
	ASGName         string
	ClusterSettings db.ClusterSettings
}

func (l *LaunchTemplate) Tags() cluster.Tags {
	return GetHostGroupResourceTags(l.HostGroupInfo, l.TargetVersion())
}

func (l *LaunchTemplate) SubResources() []cluster.Resource {
	return []cluster.Resource{&l.JoinApi}
}

func (l *LaunchTemplate) ResourceName() string {
	return common.GenerateResourceName(l.HostGroupInfo.ClusterName, l.HostGroupInfo.Name)
}

func (l *LaunchTemplate) Fetch() error {
	version, err := launchtemplate.GetLaunchTemplateVersion(l.ResourceName())
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != "InvalidLaunchTemplateName.NotFoundException" {
				return err
			}
		} else {
			return err
		}
	}
	l.Version = version

	if l.JoinApi.RestApiGateway.Id == "" {
		restApiGateway, err := apigateway.GetRestApiGateway(l.JoinApi.ResourceName())
		if err != nil {
			return err
		}
		l.JoinApi.RestApiGateway = restApiGateway
	}

	return nil
}

func (l *LaunchTemplate) DeployedVersion() string {
	return l.Version
}

func (l *LaunchTemplate) TargetVersion() string {
	return launchtemplate.LaunchtemplateVersion
}

func (l *LaunchTemplate) Create(tags cluster.Tags) error {
	return launchtemplate.CreateLaunchTemplate(tags.AsEc2(), l.HostGroupInfo.Name, l.HostGroupParams, l.JoinApi.RestApiGateway, l.ResourceName(), !l.ClusterSettings.PrivateSubnet)
}

func (l *LaunchTemplate) Update(tags cluster.Tags) error {
	newVersion, err := launchtemplate.CreateNewLaunchTemplateVersion(
		tags.AsEc2(), l.HostGroupInfo.Name, l.HostGroupParams, l.JoinApi.RestApiGateway, l.ResourceName(), !l.ClusterSettings.PrivateSubnet)
	if err != nil {
		return err
	}
	return launchtemplate.ModifyLaunchTemplateDefaultVersion(l.ResourceName(), newVersion)
}

func (l *LaunchTemplate) Init() {
	log.Debug().Msgf("Initializing hostgroup %s launch template ...", string(l.HostGroupInfo.Name))
	l.JoinApi.HostGroupInfo = l.HostGroupInfo
	l.JoinApi.TableName = l.TableName
	l.JoinApi.ASGName = l.ASGName
	l.JoinApi.Subnet = l.HostGroupParams.Subnet
	l.JoinApi.ClusterSettings = l.ClusterSettings
	l.JoinApi.Init()
}
