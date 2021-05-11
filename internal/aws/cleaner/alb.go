package cleaner

import (
	"github.com/aws/aws-sdk-go/service/elbv2"
	"wekactl/internal/aws/alb"
	"wekactl/internal/cluster"
	"wekactl/internal/logging"
)

type ApplicationLoadBalancer struct {
	ApplicationLoadBalancer *elbv2.LoadBalancer
	Listener                *elbv2.Listener
	TargetGroup             *elbv2.TargetGroup
	ClusterName             cluster.ClusterName
}

func (a *ApplicationLoadBalancer) Fetch() error {
	applicationLoadBalancer, err := alb.GetClusterApplicationLoadBalancer(a.ClusterName)
	if err != nil {
		return err
	}
	a.ApplicationLoadBalancer = applicationLoadBalancer

	if applicationLoadBalancer != nil {
		listener, err := alb.GetClusterListener(a.ClusterName, *applicationLoadBalancer.LoadBalancerArn)
		if err != nil {
			return err
		}
		a.Listener = listener
	}

	targetGroup, err := alb.GetClusterTargetGroup(a.ClusterName)
	if err != nil {
		return err
	}
	a.TargetGroup = targetGroup
	return nil
}

func (a *ApplicationLoadBalancer) Delete() error {
	return alb.DeleteAlb(a.ApplicationLoadBalancer, a.Listener, a.TargetGroup, a.ClusterName)
}

func (a *ApplicationLoadBalancer) Print() {
	logging.UserInfo("ApplicationLoadBalancer:")
	if a.ApplicationLoadBalancer != nil {
		logging.UserInfo("\t- %s", *a.ApplicationLoadBalancer.LoadBalancerName)
	}

	logging.UserInfo("Listener:")
	if a.Listener != nil {
		logging.UserInfo("\t- %s", *a.Listener.ListenerArn)
	}

	logging.UserInfo("TargetGroup:")
	if a.ApplicationLoadBalancer != nil {
		logging.UserInfo("\t- %s", *a.TargetGroup.TargetGroupName)
	}
}
