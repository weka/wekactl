package cluster

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/cloudwatch"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/iam"
	"wekactl/internal/cluster"
)

const cloudwatchVersion = "v1"

type CloudWatch struct {
	HostGroupInfo   hostgroups.HostGroupInfo
	HostGroupParams hostgroups.HostGroupParams
	ScaleMachine    ScaleMachine
	Profile         IamProfile
	TableName       string
	Version         string
}

func (c *CloudWatch) ResourceName() string {
	return common.GenerateResourceName(c.HostGroupInfo.ClusterName, c.HostGroupInfo.Name)
}

func (c *CloudWatch) Fetch() error {
	version, err := db.GetResourceVersion(c.TableName, "cloudwatch", "", c.HostGroupInfo.Name)
	if err != nil {
		return err
	}
	c.Version = version
	return nil
}

func (c *CloudWatch) DeployedVersion() string {
	return c.Version
}

func (c *CloudWatch) TargetVersion() string {
	return cloudwatchVersion
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

	err = cloudwatch.CreateCloudWatchEventRule(c.HostGroupInfo, &c.ScaleMachine.Arn, c.Profile.Arn, c.ResourceName())
	if err != nil {
		return err
	}

	return db.SaveResourceVersion(c.TableName, "cloudwatch", "", c.HostGroupInfo.Name, c.TargetVersion())
}

func (c *CloudWatch) Update() error {
	panic("implement me")
}

func (c *CloudWatch) Init() {
	log.Debug().Msgf("Initializing hostgroup %s cloudwatch ...", string(c.HostGroupInfo.Name))
	c.Profile.Name = "cw"
	c.Profile.PolicyName = fmt.Sprintf("wekactl-%s-cw-%s", string(c.HostGroupInfo.ClusterName), string(c.HostGroupInfo.Name))
	c.Profile.TableName = c.TableName
	c.Profile.AssumeRolePolicy = iam.GetCloudWatchEventAssumeRolePolicy()
	c.Profile.HostGroupInfo = c.HostGroupInfo
	c.Profile.Policy = iam.GetCloudWatchEventRolePolicy()
	c.Profile.Init()

	c.ScaleMachine.TableName = c.TableName
	c.ScaleMachine.HostGroupInfo = c.HostGroupInfo
	c.ScaleMachine.HostGroupParams = c.HostGroupParams
	c.ScaleMachine.Init()
}
