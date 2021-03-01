package cluster

import "wekactl/internal/cluster"

type InstanceRole string

const RoleBackend InstanceRole = "backend"
const RoleClient InstanceRole = "client"

type HostGroupName string

type HostGroupParams struct {
	SecurityGroupsIds []*string
	ImageID           string
	KeyName           string
	IamArn            string
	InstanceType      string
	Subnet            string
	VolumeName        string
	VolumeType        string
	VolumeSize        int64
	InstanceIds       []*string // Is not part of Params, it is related only to importing
	//TODO: Replace instanceIds with max size
}

type HostGroupInfo struct {
	ClusterName cluster.ClusterName
	Role        InstanceRole
	Name        HostGroupName
}

func GenerateHostGroup(clusterName cluster.ClusterName, hostGroupParams HostGroupParams, role InstanceRole, name HostGroupName) HostGroup {
	hostGroupInfo := HostGroupInfo{
		Name:        name,
		Role:        role,
		ClusterName: clusterName,
	}

	hostGroup := HostGroup{
		HostGroupInfo:   hostGroupInfo,
		HostGroupParams: hostGroupParams,
	}

	return hostGroup
}
