package cluster

import (
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/cluster"
)

type AWSCluster struct {
	Name            cluster.ClusterName
	ClusterSettings db.ClusterSettings
	HostGroups      []HostGroup
	TableName       string
	ALB             ApplicationLoadBalancer
}

func (c *AWSCluster) Tags() cluster.Tags {
	return cluster.Tags{}
}

func (c *AWSCluster) SubResources() []cluster.Resource {
	resources := []cluster.Resource{
		&c.ALB,
	}
	for i := range c.HostGroups {
		resources = append(resources, &c.HostGroups[i])
	}
	return resources
}

func (c *AWSCluster) ResourceName() string {
	return common.GenerateResourceName(c.Name, "")
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
		c.HostGroups[i].ClusterSettings = c.ClusterSettings
		c.HostGroups[i].Init()
	}

	c.ALB.ClusterName = c.Name
	c.ALB.VpcSubnets = []string{c.ClusterSettings.Subnet, c.ClusterSettings.AdditionalSubnet}
	c.ALB.VpcId = c.ClusterSettings.VpcId
	c.ALB.SecurityGroupsIds = c.ClusterSettings.Backends.SecurityGroupsIds
	c.ALB.DnsAlias = c.ClusterSettings.DnsAlias
	c.ALB.DnsZoneId = c.ClusterSettings.DnsZoneId
	return
}

func (c *AWSCluster) DeployedVersion() string {
	return ""
}

func (c *AWSCluster) TargetVersion() string {
	return ""
}

func (c *AWSCluster) Create(tags cluster.Tags) (err error) {
	return nil
}

func (c *AWSCluster) Update() error {
	panic("implement me")
}
