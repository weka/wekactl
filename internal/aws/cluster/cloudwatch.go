package cluster

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/cloudwatch"
	"wekactl/internal/aws/common"
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
	ASGName         string
}

func (c *CloudWatch) Tags() interface{} {
	return cloudwatch.GetCloudWatchEventTags(c.HostGroupInfo, c.TargetVersion())
}

func (c *CloudWatch) SubResources() []cluster.Resource {
	return []cluster.Resource{&c.ScaleMachine, &c.Profile}
}

func (c *CloudWatch) ResourceName() string {
	return common.GenerateResourceName(c.HostGroupInfo.ClusterName, c.HostGroupInfo.Name)
}

func (c *CloudWatch) Fetch() error {
	version, err := cloudwatch.GetCloudWatchEventRuleVersion(c.ResourceName())
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
	return cloudwatch.DeleteCloudWatchEventRule(c.ResourceName())
}

func (c *CloudWatch) Create() (err error) {
	return cloudwatch.CreateCloudWatchEventRule(c.Tags().([]*cloudwatchevents.Tag), &c.ScaleMachine.Arn, c.Profile.Arn, c.ResourceName())
}

func (c *CloudWatch) Update() error {
	err := c.Delete()
	if err != nil {
		return err
	}
	return c.Create()
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
	c.ScaleMachine.ASGName = c.ASGName
	c.ScaleMachine.Init()
}
