package cluster

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/apigateway"
	"wekactl/internal/aws/autoscaling"
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
)

const autoscalingVersion = "v1"

type AutoscalingGroup struct {
	HostGroupInfo          common.HostGroupInfo
	HostGroupParams        common.HostGroupParams
	RestApiGateway         apigateway.RestApiGateway
	LaunchTemplate         LaunchTemplate
	ScaleMachineCloudWatch CloudWatch
	TableName              string
	Version                string
}

func (a *AutoscalingGroup) Tags() cluster.Tags {
	return GetHostGroupResourceTags(a.HostGroupInfo, a.TargetVersion()).Update(cluster.Tags{
		"Name": fmt.Sprintf("%s-%s", a.HostGroupInfo.ClusterName, a.HostGroupInfo.Name)})
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
	return autoscaling.CreateAutoScalingGroup(
		a.Tags().AsAsg(), a.LaunchTemplate.ResourceName(), a.HostGroupParams.MaxSize, a.ResourceName())
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
