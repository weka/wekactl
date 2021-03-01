package cluster

import (
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/launchtemplate"
	"wekactl/internal/cluster"
)

const launchtemplateVersion = "v1"

type LaunchTemplate struct {
	HostGroupInfo   common.HostGroupInfo
	HostGroupParams common.HostGroupParams
	JoinApi         ApiGateway
	TableName       string
	Version         string
	ASGName         string
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
		return err
	}
	l.Version = version
	return nil
}

func (l *LaunchTemplate) DeployedVersion() string {
	return l.Version
}

func (l *LaunchTemplate) TargetVersion() string {
	return launchtemplateVersion
}

func (l *LaunchTemplate) Delete() error {
	return launchtemplate.DeleteLaunchTemplate(l.ResourceName())
}

func (l *LaunchTemplate) Create() error {
	return launchtemplate.CreateLaunchTemplate(l.Tags().AsEc2(), l.HostGroupInfo.Name, l.HostGroupParams, l.JoinApi.RestApiGateway, l.ResourceName())
}

func (l *LaunchTemplate) Update() error {
	err := l.Delete()
	if err != nil {
		return err
	}
	return l.Create()
}

func (l *LaunchTemplate) Init() {
	log.Debug().Msgf("Initializing hostgroup %s autoscaling group ...", string(l.HostGroupInfo.Name))
	l.JoinApi.HostGroupInfo = l.HostGroupInfo
	l.JoinApi.TableName = l.TableName
	l.JoinApi.ASGName = l.ASGName
	l.JoinApi.Init()
}
