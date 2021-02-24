package cluster

import (
	asg "github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/apigateway"
	"wekactl/internal/aws/autoscaling"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/cluster"
)

const autoscalingVersion = "v1"

type AutoscalingGroup struct {
	HostGroupInfo          hostgroups.HostGroupInfo
	HostGroupParams        hostgroups.HostGroupParams
	RestApiGateway         apigateway.RestApiGateway
	LaunchTemplate         LaunchTemplate
	ScaleMachineCloudWatch CloudWatch
	TableName              string
	Version                string
}

func (a *AutoscalingGroup) Tags() interface{} {
	return autoscaling.GetAutoScalingTags(a.HostGroupInfo, a.TargetVersion())
}

func (a *AutoscalingGroup) SubResources() []cluster.Resource {
	return []cluster.Resource{&a.LaunchTemplate, &a.ScaleMachineCloudWatch}
}

func (a *AutoscalingGroup) ResourceName() string {
	return common.GenerateResourceName(a.HostGroupInfo.ClusterName, a.HostGroupInfo.Name)
}

func (a *AutoscalingGroup) Fetch() error {
	version, err := autoscaling.GetAutoScalingGroupVersion(a.ResourceName())
	if err != nil {
		return err
	}
	a.Version = version
	return nil
}

func (a *AutoscalingGroup) DeployedVersion() string {
	return a.Version
}

func (a *AutoscalingGroup) TargetVersion() string {
	return autoscalingVersion
}

func (a *AutoscalingGroup) Delete() error {
	return autoscaling.DeleteAutoScalingGroup(a.ResourceName())
}

func (a *AutoscalingGroup) Create() error {
	maxSize := int64(autoscaling.GetMaxSize(a.HostGroupInfo.Role, len(a.HostGroupParams.InstanceIds)))
	return autoscaling.CreateAutoScalingGroup(a.Tags().([]*asg.Tag), a.LaunchTemplate.ResourceName(), maxSize, a.ResourceName())
}

func (a *AutoscalingGroup) Update() error {
	err := a.Delete()
	if err != nil {
		return err
	}
	return a.Create()
}

func (a *AutoscalingGroup) Init() {
	log.Debug().Msgf("Initializing hostgroup %s autoscaling group ...", string(a.HostGroupInfo.Name))
	a.LaunchTemplate.HostGroupInfo = a.HostGroupInfo
	a.LaunchTemplate.HostGroupParams = a.HostGroupParams
	a.LaunchTemplate.TableName = a.TableName
	a.LaunchTemplate.ASGName = a.ResourceName()
	a.LaunchTemplate.Init()
	a.ScaleMachineCloudWatch.HostGroupInfo = a.HostGroupInfo
	a.ScaleMachineCloudWatch.HostGroupParams = a.HostGroupParams
	a.ScaleMachineCloudWatch.TableName = a.TableName
	a.ScaleMachineCloudWatch.ASGName = a.ResourceName()
	a.ScaleMachineCloudWatch.Init()
}
