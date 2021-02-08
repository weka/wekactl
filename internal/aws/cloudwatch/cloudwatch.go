package cloudwatch

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/connectors"
)

func getCloudWatchEventTags(hostGroupInfo hostgroups.HostGroupInfo) []*cloudwatchevents.Tag {
	var cloudWatchEventTags []*cloudwatchevents.Tag
	for key, value := range common.GetHostGroupTags(hostGroupInfo) {
		cloudWatchEventTags = append(cloudWatchEventTags, &cloudwatchevents.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return cloudWatchEventTags
}
func CreateCloudWatchEventRule(hostGroupInfo hostgroups.HostGroupInfo, arn *string, roleArn, ruleName string) error {
	svc := connectors.GetAWSSession().CloudWatchEvents
	_, err := svc.PutRule(&cloudwatchevents.PutRuleInput{
		Name:               &ruleName,
		ScheduleExpression: aws.String("rate(1 minute)"),
		State:              aws.String("ENABLED"),
		Tags:               getCloudWatchEventTags(hostGroupInfo),
	})
	if err != nil {
		return err
	}
	log.Debug().Msgf("cloudwatch rule %s was created successfully!", ruleName)

	_, err = svc.PutTargets(&cloudwatchevents.PutTargetsInput{
		Rule: &ruleName,
		Targets: []*cloudwatchevents.Target{
			{
				Arn:     arn,
				Id:      aws.String(uuid.New().String()),
				RoleArn: &roleArn,
			},
		},
	})
	if err != nil {
		return err
	}
	log.Debug().Msgf("cloudwatch state machine target was set successfully!")

	return nil
}
