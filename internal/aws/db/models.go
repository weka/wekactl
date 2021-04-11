package db

import (
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
)

const ModelClusterCreds = "cluster-creds"

type ClusterCreds struct {
	Key      string
	Username string
	Password string
}

const ModelDefaultClusterParams = "default-cluster-params"

type ClusterParams struct {
	Key                  string
	DefaultBackendParams common.HostGroupParams
	DefaultClientParams  common.HostGroupParams
	Subnet               string
	Tags                 cluster.Tags
	PrivateSubnet        bool
}

type ResourceVersion struct {
	Key     string
	Version string
}
