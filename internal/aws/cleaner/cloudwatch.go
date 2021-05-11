package cleaner

import (
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"wekactl/internal/aws/cloudwatch"
	"wekactl/internal/cluster"
	"wekactl/internal/logging"
)

type CloudWatch struct {
	CloudWatchEventRules []*cloudwatchevents.Rule
	ClusterName          cluster.ClusterName
}

func (c *CloudWatch) Fetch() error {
	cloudWatchEventRules, err := cloudwatch.GetCloudWatchEventRules(c.ClusterName)
	if err != nil {
		return err
	}
	c.CloudWatchEventRules = cloudWatchEventRules
	return nil
}

func (c *CloudWatch) Delete() error {
	return cloudwatch.DeleteCloudWatchEventRules(c.CloudWatchEventRules)
}

func (c *CloudWatch) Print() {
	logging.UserInfo("CloudWatch Event Rules:")
	for _, rule := range c.CloudWatchEventRules {
		logging.UserInfo("\t- %s", *rule.Name)
	}
}
