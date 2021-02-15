package cluster

import (
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/cluster"
)

func GenerateHostGroup(clusterName cluster.ClusterName, hostGroupParams hostgroups.HostGroupParams, role hostgroups.InstanceRole) HostGroup {
	var name string
	if role == hostgroups.RoleBackend {
		name = "Backends"
	} else {
		name = "Clients"
	}

	hostGroupInfo := hostgroups.HostGroupInfo{
		Name:        hostgroups.HostGroupName(name),
		Role:        role,
		ClusterName: clusterName,
	}

	hostGroup := HostGroup{
		HostGroupInfo:   hostGroupInfo,
		HostGroupParams: hostGroupParams,
	}

	return hostGroup
}
