package db

import "wekactl/internal/aws/cluster"

const ModelClusterCreds = "cluster-creds"

type ClusterCreds struct {
	Key      string
	Username string
	Password string
}

const ModelDefaultClusterParams = "default-cluster-params"

type DefaultClusterParams struct {
	Key      string
	Backends cluster.HGParams
	Clients  cluster.HGParams
	VPC      string
	Subnet   string
}
