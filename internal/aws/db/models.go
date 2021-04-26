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

type ClusterSettings struct {
	Key              string
	Backends         common.HostGroupParams
	Clients          common.HostGroupParams
	Subnet           string
	AdditionalSubnet string
	VpcId            string
	TagsMap          cluster.Tags
	PrivateSubnet    bool
	StackId          *string // != nil in case it is created from CF stack
}

func (c ClusterSettings) Tags() cluster.Tags {
	return c.TagsMap
}

func (c ClusterSettings) UsePrivateSubnet() bool {
	return c.PrivateSubnet
}

type ResourceVersion struct {
	Key     string
	Version string
}
