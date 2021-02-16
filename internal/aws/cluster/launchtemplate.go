package cluster

import (
	"wekactl/internal/aws/apigateway"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/launchtemplate"
)

const launchtemplateVersion = "v1"

type LaunchTemplate struct {
	HostGroupInfo   hostgroups.HostGroupInfo
	HostGroupParams hostgroups.HostGroupParams
	RestApiGateway  apigateway.RestApiGateway
	TableName       string
}

func (l *LaunchTemplate) ResourceName() string {
	return common.GenerateResourceName(l.HostGroupInfo.ClusterName, l.HostGroupInfo.Name)
}

func (l *LaunchTemplate) Fetch() error {
	return nil
}

func (l *LaunchTemplate) DeployedVersion() string {
	return ""
}

func (l *LaunchTemplate) TargetVersion() string {
	return launchtemplateVersion
}

func (l *LaunchTemplate) Delete() error {
	return launchtemplate.DeleteLaunchTemplate(l.ResourceName())
}

func (l *LaunchTemplate) Create() error {
	err := launchtemplate.CreateLaunchTemplate(l.HostGroupInfo, l.HostGroupParams, l.RestApiGateway, l.ResourceName())
	if err != nil {
		return err
	}
	return db.SaveResourceVersion(l.TableName, "launchtemplate", "", l.HostGroupInfo.Name, l.TargetVersion())
}

func (l *LaunchTemplate) Update() error {
	panic("implement me")
}

func (l *LaunchTemplate) Init() {
}
