package db

import (
	"wekactl/internal/aws/common"
)

const ModelClusterCreds = "cluster-creds"
const ModelClusterSettings = "cluster-settings"

type ClusterCreds struct {
	Key      string
	Username string
	Password string
}

const ModelDefaultClusterParams = "default-cluster-params"

type DefaultClusterParams struct {
	Key      string
	Backends common.HostGroupParams
	Clients  common.HostGroupParams
	Subnet   string
}

type ResourceVersion struct {
	Key     string
	Version string
}
