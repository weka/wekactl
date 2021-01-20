package cluster

import "wekactl/internal/cluster"

type HostGroupName string

type HGParams struct {
	SecurityGroupsIds []string
	ImageID           string
	KeyName           string
	IamArn            string
	InstanceType      string
	Subnet            string
}


type HostGroupInfo struct {
	ClusterName cluster.ClusterName
	Role  InstanceRole
	Name  HostGroupName
}

type HostGroup struct {
	HostGroupInfo
	JoinApi ApiGateway
	//ScaleMachine ScaleMachine
	//AutoscalingGroup AutoscalingGroup
}

func (h *HostGroup) Fetch() error {
	panic("implement me")
}

func (h *HostGroup) DeployedVersion() string {
	panic("implement me")
}

func (h *HostGroup) TargetVersion() string {
	return h.JoinApi.TargetVersion()
}

func (h *HostGroup) Delete() error {
	panic("implement me")
}

func (h *HostGroup) Create() error {
	panic("implement me")
}

func (h *HostGroup) Update() error {
	panic("implement me")
}

func (h *HostGroup) Init() {
	h.JoinApi.HgInfo = h.HostGroupInfo
	h.JoinApi.Init()
}



