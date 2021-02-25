package db

import (
	"wekactl/internal/aws/hostgroups"
)

const ModelClusterCreds = "cluster-creds"

type ClusterCreds struct {
	Key      string
	Username string
	Password string
}

const ModelDefaultClusterParams = "default-cluster-params"

type DefaultClusterParams struct {
	Key      string
	Backends hostgroups.HostGroupParams
	Clients  hostgroups.HostGroupParams
	Subnet   string
}

type ResourceVersion struct {
	Key     string
	Version string
}
