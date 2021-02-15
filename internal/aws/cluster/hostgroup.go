package cluster

import (
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/cluster"
)

type HostGroup struct {
	HostGroupInfo          hostgroups.HostGroupInfo
	HostGroupParams        hostgroups.HostGroupParams
	JoinApi                ApiGateway
	AutoscalingGroup       AutoscalingGroup
	ScaleMachineCloudWatch CloudWatch
}

func (h *HostGroup) ResourceName() string {
	return common.GenerateResourceName(h.HostGroupInfo.ClusterName, h.HostGroupInfo.Name)
}

func (h *HostGroup) Fetch() error {
	return nil
}

func (h *HostGroup) DeployedVersion() string {
	return ""
}

func (h *HostGroup) TargetVersion() string {
	return h.JoinApi.TargetVersion()
}

func (h *HostGroup) Delete() error {
	err := h.JoinApi.Delete()
	if err != nil {
		return err
	}

	err = h.AutoscalingGroup.Delete()
	if err != nil {
		return err
	}

	return h.ScaleMachineCloudWatch.Delete()
}

func (h *HostGroup) Create() (err error) {
	err = cluster.EnsureResource(&h.JoinApi)
	if err != nil {
		return
	}

	h.AutoscalingGroup.RestApiGateway = h.JoinApi.RestApiGateway
	err = cluster.EnsureResource(&h.AutoscalingGroup)
	if err != nil {
		return
	}

	err = cluster.EnsureResource(&h.ScaleMachineCloudWatch)
	if err != nil {
		return
	}
	return
}

func (h *HostGroup) Update() error {
	panic("implement me")
}

func (h *HostGroup) Init() {
	log.Debug().Msgf("Initializing hostgroup %s ...", string(h.HostGroupInfo.Name))
	h.JoinApi.HostGroupInfo = h.HostGroupInfo
	h.JoinApi.Init()
	h.AutoscalingGroup.HostGroupInfo = h.HostGroupInfo
	h.AutoscalingGroup.HostGroupParams = h.HostGroupParams
	h.AutoscalingGroup.Init()
	h.ScaleMachineCloudWatch.HostGroupInfo = h.HostGroupInfo
	h.ScaleMachineCloudWatch.HostGroupParams = h.HostGroupParams
	h.ScaleMachineCloudWatch.Init()
}
