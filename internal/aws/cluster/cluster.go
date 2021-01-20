package cluster

import (
	"strings"
	"wekactl/internal/aws/db"
	"wekactl/internal/cluster"
)

type InstanceRole string

const RoleBackend InstanceRole = "backend"
const RoleClient InstanceRole = "client"

type DynamoDBName string

type AWSCluster struct {
	Name          cluster.ClusterName
	DefaultParams db.DefaultClusterParams
	CFStack       Stack
	Hostgroups 	  []HostGroup
}

func (c *AWSCluster) GetDBName() DynamoDBName {
	return DynamoDBName(strings.Join([]string{string(c.Name)}, "-"))
}

type Stack struct {
	StackId   string
	StackName string
}

