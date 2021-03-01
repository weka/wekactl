package cluster

import (
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
)

const hostGroupVersion = "v1"

type HostGroup struct {
	HostGroupInfo    common.HostGroupInfo
	HostGroupParams  common.HostGroupParams
	AutoscalingGroup AutoscalingGroup
	TableName        string
}

func (h *HostGroup) Tags() cluster.Tags {
	return nil
}

func (h *HostGroup) SubResources() []cluster.Resource {
	return []cluster.Resource{&h.AutoscalingGroup}
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
	return hostGroupVersion
}

func (h *HostGroup) Delete() error {
	return nil
}

func (h *HostGroup) Create() error {
	return nil
}

func (h *HostGroup) Update() error {
	panic("implement me")
}

func (h *HostGroup) Init() {
	log.Debug().Msgf("Initializing hostgroup %s ...", string(h.HostGroupInfo.Name))
	h.AutoscalingGroup.HostGroupInfo = h.HostGroupInfo
	h.AutoscalingGroup.HostGroupParams = h.HostGroupParams
	h.AutoscalingGroup.TableName = h.TableName
	h.AutoscalingGroup.Init()
}

func GetHostGroupResourceTags(hostGroup common.HostGroupInfo, version string) cluster.Tags {
	tags := cluster.GetCommonResourceTags(hostGroup.ClusterName, version)
	return tags.Update(cluster.Tags{
		"wekactl.io/hostgroup_name": string(hostGroup.Name),
		"wekactl.io/hostgroup_type": string(hostGroup.Role),
	})
}
