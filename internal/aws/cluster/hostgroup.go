package cluster

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/alb"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
)

const hostGroupVersion = "v1"
const RoleTagKey = "wekactl.io/hostgroup_type"
const HostGroupNameTagKey = "wekactl.io/hostgroup_name"

type HostGroup struct {
	HostGroupInfo    common.HostGroupInfo
	HostGroupParams  common.HostGroupParams
	AutoscalingGroup AutoscalingGroup
	TableName        string
	ClusterSettings  db.ClusterSettings
}

func (h *HostGroup) Tags() cluster.Tags {
	return cluster.Tags{}
}

func (h *HostGroup) SubResources() []cluster.Resource {
	return []cluster.Resource{&h.AutoscalingGroup}
}

func (h *HostGroup) ResourceName() string {
	return common.GenerateResourceName(h.HostGroupInfo.ClusterName, h.HostGroupInfo.Name)
}

func (h *HostGroup) Fetch() error {
	return nil
}

func (h *HostGroup) DeployedVersion() string {
	return ""
}

func (h *HostGroup) TargetVersion() string {
	return hostGroupVersion
}

func (h *HostGroup) Create(tags cluster.Tags) (err error) {
	if h.HostGroupInfo.Role != common.RoleBackend {
		return
	}

	arn, err := alb.GetTargetGroupArn(h.HostGroupInfo.ClusterName)
	if err != nil {
		return err
	}
	svc := connectors.GetAWSSession().ASG
	_, err = svc.AttachLoadBalancerTargetGroups(&autoscaling.AttachLoadBalancerTargetGroupsInput{
		TargetGroupARNs: []*string{
			aws.String(arn),
		},
		AutoScalingGroupName: aws.String(h.AutoscalingGroup.ResourceName()),
	})

	return
}

func (h *HostGroup) Update() error {
	panic("implement me")
}

func (h *HostGroup) Init() {
	log.Debug().Msgf("Initializing hostgroup %s ...", string(h.HostGroupInfo.Name))
	h.AutoscalingGroup.HostGroupInfo = h.HostGroupInfo
	h.AutoscalingGroup.HostGroupParams = h.HostGroupParams
	h.AutoscalingGroup.TableName = h.TableName
	h.AutoscalingGroup.ClusterSettings = h.ClusterSettings
	h.AutoscalingGroup.Init()
}

func GetHostGroupResourceTags(hostGroup common.HostGroupInfo, version string) cluster.Tags {
	tags := cluster.GetCommonResourceTags(hostGroup.ClusterName, version)
	return tags.Update(cluster.Tags{
		HostGroupNameTagKey: string(hostGroup.Name),
		RoleTagKey:          string(hostGroup.Role),
	})
}
