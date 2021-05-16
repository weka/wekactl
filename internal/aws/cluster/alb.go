package cluster

import (
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/alb"
	route53internal "wekactl/internal/aws/route53"
	"wekactl/internal/cluster"
	"wekactl/internal/lib/strings"
)

const albVersion = "v1"

type ApplicationLoadBalancer struct {
	ClusterName        cluster.ClusterName
	Version            string
	TargetGroupVersion string
	ListenerVersion    string
	VpcSubnets         []string
	VpcId              string
	SecurityGroupsIds  []*string
	DnsAlias           string
	DnsZoneId          string
	RecordSet          *route53.ResourceRecordSet
}

func (a *ApplicationLoadBalancer) Tags() cluster.Tags {
	return cluster.GetCommonResourceTags(a.ClusterName, a.TargetVersion())
}

func (a *ApplicationLoadBalancer) SubResources() []cluster.Resource {
	return []cluster.Resource{}
}

func (a *ApplicationLoadBalancer) ResourceName() string {
	return alb.GetApplicationLoadBalancerName(a.ClusterName)
}

func (a *ApplicationLoadBalancer) Fetch() error {
	version, err := alb.GetApplicationLoadBalancerVersion(a.ResourceName())
	if err != nil {
		return err
	}
	a.Version = version

	targetGroupVersion, err := alb.GetTargetGroupVersion(a.ClusterName)
	if err != nil {
		return err
	}
	a.TargetGroupVersion = targetGroupVersion

	listenerVersion, err := alb.GetListenerVersion(a.ResourceName(), "api")
	if err != nil {
		return err
	}
	a.ListenerVersion = listenerVersion

	if a.DnsAlias != "" && a.DnsZoneId != "" {
		loadBalancer, err := alb.GetClusterApplicationLoadBalancer(a.ClusterName)
		if err != nil {
			return err
		}
		if loadBalancer != nil {
			recordSet, err := route53internal.GetRoute53Record(a.DnsAlias, a.DnsZoneId)
			if err != nil {
				return err
			}
			a.RecordSet = recordSet
		}
	}

	return nil
}

func (a *ApplicationLoadBalancer) Init() {
	log.Debug().Msgf("Initializing cluster %s ALB ...", string(a.ClusterName))
	return
}

func (a *ApplicationLoadBalancer) DeployedVersion() string {
	differentVersion := a.TargetVersion() + "#" // just to make it different from TargetVersion so we will enter Update flow
	if a.DnsAlias != "" && a.DnsZoneId != "" && a.RecordSet == nil {
		return differentVersion
	}
	if a.Version == a.TargetGroupVersion && a.Version == a.ListenerVersion {
		return a.Version
	}
	return differentVersion
}

func (a *ApplicationLoadBalancer) TargetVersion() string {
	return albVersion
}

func (a *ApplicationLoadBalancer) Create(tags cluster.Tags) (err error) {
	//TODO: consider separating into 3 different resources

	loadBalancer, err := alb.CreateApplicationLoadBalancer(tags.AsAlb(), a.ResourceName(), strings.ListToRefList(a.VpcSubnets), a.SecurityGroupsIds)
	if err != nil {
		return
	}
	targetArn, err := alb.CreateTargetGroup(tags.AsAlb(), alb.GetTargetGroupName(a.ClusterName), a.VpcId)
	if err != nil {
		return
	}

	err = alb.CreateListener(tags.Update(cluster.Tags{alb.ListenerTypeTagKey: "api"}).AsAlb(), *loadBalancer.LoadBalancerArn, targetArn)
	if err != nil {
		return err
	}

	if a.DnsAlias != "" && a.DnsZoneId != "" {
		err = route53internal.CreateApplicationLoadBalancerAliasRecord(loadBalancer, a.DnsAlias, a.DnsZoneId)
	}
	return
}

func (a *ApplicationLoadBalancer) Update() (err error) {
	// currently we will enter here only if Create failed at some point (during import).
	// the only case we need to support is when for some reason alb/targetGroup/listener where not created
	var targetArn string

	if a.TargetGroupVersion == "" {
		targetArn, err = alb.CreateTargetGroup(a.Tags().AsAlb(), alb.GetTargetGroupName(a.ClusterName), a.VpcId)
		if err != nil {
			return
		}
	} else {
		targetArn, err = alb.GetTargetGroupArn(a.ClusterName)
		if err != nil {
			return
		}
	}

	var loadBalancer *elbv2.LoadBalancer
	if a.Version == "" {
		loadBalancer, err = alb.CreateApplicationLoadBalancer(a.Tags().AsAlb(), a.ResourceName(), strings.ListToRefList(a.VpcSubnets), a.SecurityGroupsIds)
		if err != nil {
			return
		}
	} else {
		loadBalancer, err = alb.GetClusterApplicationLoadBalancer(a.ClusterName)
		if err != nil {
			return
		}
	}

	err = alb.CreateListener(a.Tags().Update(cluster.Tags{alb.ListenerTypeTagKey: "api"}).AsAlb(), *loadBalancer.LoadBalancerArn, targetArn)
	if err != nil {
		return err
	}

	if a.DnsAlias != "" && a.DnsZoneId != "" && a.RecordSet == nil {
		err = route53internal.CreateApplicationLoadBalancerAliasRecord(loadBalancer, a.DnsAlias, a.DnsZoneId)
	}
	return

}
