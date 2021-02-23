package cluster

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/launchtemplate"
	"wekactl/internal/cluster"
)

const launchtemplateVersion = "v1"

type LaunchTemplate struct {
	HostGroupInfo   hostgroups.HostGroupInfo
	HostGroupParams hostgroups.HostGroupParams
	JoinApi         ApiGateway
	TableName       string
	Version         string
	ASGName         string
}

func (l *LaunchTemplate) Tags() interface{} {
	return launchtemplate.GetEc2Tags(l.HostGroupInfo, l.TargetVersion())
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
	err := l.JoinApi.Delete()
	if err != nil {
		return err
	}
	return launchtemplate.DeleteLaunchTemplate(l.ResourceName())
}

func (l *LaunchTemplate) Create() error {
	return launchtemplate.CreateLaunchTemplate(l.Tags().([]*ec2.Tag), l.HostGroupInfo.Name, l.HostGroupParams, l.JoinApi.RestApiGateway, l.ResourceName())
}

func (l *LaunchTemplate) Update() error {
	panic("implement me")
}

func (l *LaunchTemplate) Init() {
	log.Debug().Msgf("Initializing hostgroup %s autoscaling group ...", string(l.HostGroupInfo.Name))
	l.JoinApi.HostGroupInfo = l.HostGroupInfo
	l.JoinApi.TableName = l.TableName
	l.JoinApi.ASGName = l.ASGName
	l.JoinApi.Init()
}
