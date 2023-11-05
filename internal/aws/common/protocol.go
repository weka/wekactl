package common

import (
	"wekactl/internal/cluster"
)

type InstanceRole string

const RoleBackend InstanceRole = "backend"
const RoleClient InstanceRole = "client"

type HostGroupName string

type VolumeInfo struct {
	Name string
	Type string
	Size int64
}

type HostGroupParams struct {
	SecurityGroupsIds []*string
	ImageID           string
	KeyName           string
	IamArn            string
	InstanceType      string
	Subnet            string
	VolumesInfo       []VolumeInfo
	MaxSize           int64
	HttpTokens        string
}

type HostGroupInfo struct {
	ClusterName cluster.ClusterName
	Role        InstanceRole
	Name        HostGroupName
}
