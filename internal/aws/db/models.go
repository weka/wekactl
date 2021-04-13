package db

import (
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
)

const ModelClusterCreds = "cluster-creds"
const ModelClusterSettings = "cluster-settings"

type ClusterCreds struct {
	Key      string
	Username string
	Password string
}

const ModelDefaultClusterParams = "default-cluster-params"

type ClusterSettings struct {
	Key           string
	Backends      common.HostGroupParams
	Clients       common.HostGroupParams
	Subnet        string
	TagsMap       cluster.Tags
	PrivateSubnet bool
}

func (c ClusterSettings) Tags() cluster.Tags {
	return c.TagsMap
}

type ResourceVersion struct {
	Key     string
	Version string
}
