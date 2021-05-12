package cluster

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/autoscaling"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/cluster"
)

const autoscalingVersion = "v1"

type AutoscalingGroup struct {
	HostGroupInfo          common.HostGroupInfo
	HostGroupParams        common.HostGroupParams
	LaunchTemplate         LaunchTemplate
	ScaleMachineCloudWatch CloudWatch
	TableName              string
	Version                string
	ClusterSettings        db.ClusterSettings
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

func (a *AutoscalingGroup) Create(tags cluster.Tags) error {
	return autoscaling.CreateAutoScalingGroup(
		tags.AsAsg(), a.LaunchTemplate.ResourceName(), a.HostGroupParams.MaxSize, a.ResourceName())
}

func (a *AutoscalingGroup) Update() error {
	panic("update not supported")
}

func (a *AutoscalingGroup) Init() {
	log.Debug().Msgf("Initializing hostgroup %s autoscaling group ...", string(a.HostGroupInfo.Name))
	a.LaunchTemplate.HostGroupInfo = a.HostGroupInfo
	a.LaunchTemplate.HostGroupParams = a.HostGroupParams
	a.LaunchTemplate.TableName = a.TableName
	a.LaunchTemplate.ASGName = a.ResourceName()
	a.LaunchTemplate.ClusterSettings = a.ClusterSettings
	a.LaunchTemplate.Init()
	a.ScaleMachineCloudWatch.HostGroupInfo = a.HostGroupInfo
	a.ScaleMachineCloudWatch.HostGroupParams = a.HostGroupParams
	a.ScaleMachineCloudWatch.TableName = a.TableName
	a.ScaleMachineCloudWatch.ASGName = a.ResourceName()
	a.ScaleMachineCloudWatch.Init()
}
