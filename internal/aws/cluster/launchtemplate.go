package cluster

import (
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
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
}

func (l *LaunchTemplate) SubResources() []cluster.Resource {
	return []cluster.Resource{&l.JoinApi}
}

func (l *LaunchTemplate) ResourceName() string {
	return common.GenerateResourceName(l.HostGroupInfo.ClusterName, l.HostGroupInfo.Name)
}

func (l *LaunchTemplate) Fetch() error {
	version, err := db.GetResourceVersion(l.TableName, "launchtemplate", "", l.HostGroupInfo.Name)
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
	err := launchtemplate.CreateLaunchTemplate(l.HostGroupInfo, l.HostGroupParams, l.JoinApi.RestApiGateway, l.ResourceName())
	if err != nil {
		return err
	}
	return db.SaveResourceVersion(l.TableName, "launchtemplate", "", l.HostGroupInfo.Name, l.TargetVersion())
}

func (l *LaunchTemplate) Update() error {
	panic("implement me")
}

func (l *LaunchTemplate) Init() {
	log.Debug().Msgf("Initializing hostgroup %s autoscaling group ...", string(l.HostGroupInfo.Name))
	l.JoinApi.HostGroupInfo = l.HostGroupInfo
	l.JoinApi.TableName = l.TableName
	l.JoinApi.Init()
}
