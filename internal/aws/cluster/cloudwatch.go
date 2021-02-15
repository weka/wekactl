package cluster

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/cloudwatch"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/iam"
	"wekactl/internal/cluster"
)

type CloudWatch struct {
	HostGroupInfo   hostgroups.HostGroupInfo
	HostGroupParams hostgroups.HostGroupParams
	ScaleMachine    ScaleMachine
	Profile         IamProfile
}

func (c *CloudWatch) ResourceName() string {
	return common.GenerateResourceName(c.HostGroupInfo.ClusterName, c.HostGroupInfo.Name)
}

func (c *CloudWatch) Fetch() error {
	return nil
}

func (c *CloudWatch) DeployedVersion() string {
	return ""
}

func (c *CloudWatch) TargetVersion() string {
	return ""
}

func (c *CloudWatch) Delete() error {
	err := c.Profile.Delete()
	if err != nil {
		return err
	}

	err = c.ScaleMachine.Delete()
	if err != nil {
		return err
	}

	return cloudwatch.DeleteCloudWatchEventRule(c.ResourceName())
}

func (c *CloudWatch) Create() (err error) {
	err = cluster.EnsureResource(&c.Profile)
	if err != nil {
		return
	}

	err = cluster.EnsureResource(&c.ScaleMachine)
	if err != nil {
		return
	}
	return cloudwatch.CreateCloudWatchEventRule(c.HostGroupInfo, &c.ScaleMachine.Arn, c.Profile.Arn, c.ResourceName())
}

func (c *CloudWatch) Update() error {
	panic("implement me")
}

func (c *CloudWatch) Init() {
	log.Debug().Msgf("Initializing hostgroup %s cloudwatch ...", string(c.HostGroupInfo.Name))
	c.Profile.Name = "cw"
	c.Profile.PolicyName = fmt.Sprintf("wekactl-%s-cw-%s", string(c.HostGroupInfo.ClusterName), string(c.HostGroupInfo.Name))
	c.Profile.AssumeRolePolicy = iam.GetCloudWatchEventAssumeRolePolicy()
	c.Profile.HostGroupInfo = c.HostGroupInfo
	c.Profile.Policy = iam.GetCloudWatchEventRolePolicy()
	c.Profile.Init()

	c.ScaleMachine.HostGroupInfo = c.HostGroupInfo
	c.ScaleMachine.HostGroupParams = c.HostGroupParams
	c.ScaleMachine.Init()
}
