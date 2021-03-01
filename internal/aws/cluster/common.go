package cluster

import (
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
)

func GenerateHostGroup(clusterName cluster.ClusterName, hostGroupParams common.HostGroupParams, role common.InstanceRole, name common.HostGroupName) HostGroup {
	hostGroupInfo := common.HostGroupInfo{
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
