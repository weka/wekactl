package common

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
	MaxSize           int64
}

type HostGroupInfo struct {
	ClusterName cluster.ClusterName
	Role        InstanceRole
	Name        HostGroupName
}
