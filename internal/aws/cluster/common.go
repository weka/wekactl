package cluster

import (
	"math"
	"wekactl/internal/cluster"
)

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
	MaxSize           int64
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

func GetMaxSize(role InstanceRole, initialSize int) int64 {
	var maxSize int
	switch role {
	case "backend":
		maxSize = 7 * initialSize
	case "client":
		maxSize = int(math.Ceil(float64(initialSize)/float64(500))) * 500
	default:
		maxSize = 1000
	}
	return int64(maxSize)
}
