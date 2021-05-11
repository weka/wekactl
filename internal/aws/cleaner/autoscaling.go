package cleaner

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	autoscaling2 "wekactl/internal/aws/autoscaling"
	"wekactl/internal/cluster"
	"wekactl/internal/logging"
)

type AutoscalingGroup struct {
	AutoScalingGroups []*autoscaling.Group
	ClusterName       cluster.ClusterName
}

func (a *AutoscalingGroup) Fetch() error {
	autoScalingGroups, err := autoscaling2.GetClusterAutoScalingGroups(a.ClusterName)
	if err != nil {
		return err
	}
	a.AutoScalingGroups = autoScalingGroups
	return nil
}

func (a *AutoscalingGroup) Delete() error {
	return autoscaling2.DeleteAutoScalingGroups(a.AutoScalingGroups)
}

func (a *AutoscalingGroup) Print() {
	logging.UserInfo("AutoscalingGroups:")
	for _, asg := range a.AutoScalingGroups {
		logging.UserInfo("\t- %s", *asg.AutoScalingGroupName)
	}
}
