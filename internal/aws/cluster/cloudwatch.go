package cluster

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/cloudwatch"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/iam"
	"wekactl/internal/aws/scalemachine"
	"wekactl/internal/cluster"
)

const cloudwatchVersion = "v1"

type CloudWatch struct {
	HostGroupInfo   common.HostGroupInfo
	HostGroupParams common.HostGroupParams
	ScaleMachine    ScaleMachine
	Profile         IamProfile
	TableName       string
	Version         string
	ASGName         string
}

func (c *CloudWatch) Tags() cluster.Tags {
	return GetHostGroupResourceTags(c.HostGroupInfo, c.TargetVersion())
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

	if c.ScaleMachine.Arn == "" {
		scaleMachineArn, err := scalemachine.GetStateMachineArn(c.ScaleMachine.ResourceName())
		if err != nil {
			return err
		}
		c.ScaleMachine.Arn = scaleMachineArn
	}

	if c.Profile.Arn == "" {
		profileArn, err := iam.GetIamRoleArn(c.Profile.resourceNameBase())
		if err != nil {
			return err
		}
		c.Profile.Arn = profileArn
	}

	return nil
}

func (c *CloudWatch) DeployedVersion() string {
	return c.Version
}

func (c *CloudWatch) TargetVersion() string {
	return cloudwatchVersion
}

func (c *CloudWatch) Create(tags cluster.Tags) (err error) {
	return cloudwatch.CreateCloudWatchEventRule(tags.AsCloudWatch(), &c.ScaleMachine.Arn, c.Profile.Arn, c.ResourceName())
}

func (c *CloudWatch) Update() error {
	panic("update not supported")
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
