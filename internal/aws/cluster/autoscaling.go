package cluster

import (
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/apigateway"
	"wekactl/internal/aws/autoscaling"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/cluster"
)

type AutoscalingGroup struct {
	HostGroupInfo   hostgroups.HostGroupInfo
	HostGroupParams hostgroups.HostGroupParams
	RestApiGateway  apigateway.RestApiGateway
	LaunchTemplate  LaunchTemplate
}

func (a *AutoscalingGroup) ResourceName() string {
	return common.GenerateResourceName(a.HostGroupInfo.ClusterName, a.HostGroupInfo.Name)
}

func (a *AutoscalingGroup) Fetch() error {
	return nil
}

func (a *AutoscalingGroup) DeployedVersion() string {
	return ""
}

func (a *AutoscalingGroup) TargetVersion() string {
	return ""
}

func (a *AutoscalingGroup) Delete() error {
	err := a.LaunchTemplate.Delete()
	if err != nil {
		return err
	}
	return autoscaling.DeleteAutoScalingGroup(a.ResourceName())
}

func (a *AutoscalingGroup) Create() error {
	a.LaunchTemplate.RestApiGateway = a.RestApiGateway
	err := cluster.EnsureResource(&a.LaunchTemplate)
	if err != nil {
		return err
	}
	err = autoscaling.CreateAutoScalingGroup(a.HostGroupInfo, a.LaunchTemplate.ResourceName(), a.HostGroupParams, a.ResourceName())
	if err != nil {
		return err
	}
	return nil
}

func (a *AutoscalingGroup) Update() error {
	panic("implement me")
}

func (a *AutoscalingGroup) Init() {
	log.Debug().Msgf("Initializing hostgroup %s autoscaling group ...", string(a.HostGroupInfo.Name))
	a.LaunchTemplate.HostGroupInfo = a.HostGroupInfo
	a.LaunchTemplate.HostGroupParams = a.HostGroupParams
}
