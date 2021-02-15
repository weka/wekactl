package cluster

import (
	"wekactl/internal/aws/apigateway"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/launchtemplate"
)

type LaunchTemplate struct {
	HostGroupInfo   hostgroups.HostGroupInfo
	HostGroupParams hostgroups.HostGroupParams
	RestApiGateway  apigateway.RestApiGateway
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
	return ""
}

func (l *LaunchTemplate) Delete() error {
	return launchtemplate.DeleteLaunchTemplate(l.ResourceName())
}

func (l *LaunchTemplate) Create() error {
	err := launchtemplate.CreateLaunchTemplate(l.HostGroupInfo, l.HostGroupParams, l.RestApiGateway, l.ResourceName())
	if err != nil {
		return err
	}
	return nil
}

func (l *LaunchTemplate) Update() error {
	panic("implement me")
}

func (l *LaunchTemplate) Init() {
}
