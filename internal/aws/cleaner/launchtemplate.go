package cleaner

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"wekactl/internal/aws/launchtemplate"
	"wekactl/internal/cluster"
	"wekactl/internal/logging"
)

type LaunchTemplate struct {
	LaunchTemplates []*ec2.LaunchTemplate
	ClusterName     cluster.ClusterName
}

func (l *LaunchTemplate) Fetch() error {
	launchTemplates, err := launchtemplate.GetClusterLaunchTemplates(l.ClusterName)
	if err != nil {
		return err
	}
	l.LaunchTemplates = launchTemplates
	return nil
}

func (l *LaunchTemplate) Delete() error {
	return launchtemplate.DeleteLaunchTemplates(l.LaunchTemplates)
}

func (l *LaunchTemplate) Print() {
	logging.UserInfo("LaunchTemplates:")
	for _, launchTemplate := range l.LaunchTemplates {
		logging.UserInfo("\t- %s", *launchTemplate.LaunchTemplateName)
	}
}
