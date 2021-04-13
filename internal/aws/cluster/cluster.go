package cluster

import (
	"github.com/rs/zerolog/log"
	"strings"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/cluster"
)

type DynamoDBName string

type AWSCluster struct {
	Name          cluster.ClusterName
	DefaultParams db.ClusterSettings
	CFStack       Stack
	HostGroups    []HostGroup
	TableName     string
}

func (c *AWSCluster) Tags() cluster.Tags {
	return cluster.Tags{}
}

func (c *AWSCluster) SubResources() []cluster.Resource {
	var resources []cluster.Resource
	for i := range c.HostGroups {
		resources = append(resources, &c.HostGroups[i])
	}
	return resources
}

func (c *AWSCluster) ResourceName() string {
	return common.GenerateResourceName(c.Name, "")
}

func (c *AWSCluster) GetDBName() DynamoDBName {
	return DynamoDBName(strings.Join([]string{string(c.Name)}, "-"))
}

type Stack struct {
	StackId   string
	StackName string
}

func (c *AWSCluster) Fetch() error {
	return nil
}

func (c *AWSCluster) Init() {
	log.Debug().Msgf("Initializing cluster %s ...", string(c.Name))

	for i := range c.HostGroups {
		c.HostGroups[i].TableName = c.TableName
		c.HostGroups[i].Init()
	}
	return
}

func (c *AWSCluster) DeployedVersion() string {
	return ""
}

func (c *AWSCluster) TargetVersion() string {
	return ""
}

func (c *AWSCluster) Delete() error {
	return nil
}

func (c *AWSCluster) Create(tags cluster.Tags, PrivateSubnet bool) (err error) {
	return nil
}

func (c *AWSCluster) Update() error {
	panic("implement me")
}
