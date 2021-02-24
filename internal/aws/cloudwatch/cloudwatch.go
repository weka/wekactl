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

func GetCloudWatchEventTags(hostGroupInfo hostgroups.HostGroupInfo, version string) []*cloudwatchevents.Tag {
	var cloudWatchEventTags []*cloudwatchevents.Tag
	for key, value := range common.GetHostGroupResourceTags(hostGroupInfo, version) {
		cloudWatchEventTags = append(cloudWatchEventTags, &cloudwatchevents.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return cloudWatchEventTags
}
func CreateCloudWatchEventRule(tags []*cloudwatchevents.Tag, arn *string, roleArn, ruleName string) error {
	svc := connectors.GetAWSSession().CloudWatchEvents
	_, err := svc.PutRule(&cloudwatchevents.PutRuleInput{
		Name:               &ruleName,
		ScheduleExpression: aws.String("rate(1 minute)"),
		State:              aws.String("ENABLED"),
		Tags:               tags,
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

func DeleteCloudWatchEventRule(ruleName string) error {
	svc := connectors.GetAWSSession().CloudWatchEvents

	targetsOutput, err := svc.ListTargetsByRule(&cloudwatchevents.ListTargetsByRuleInput{Rule: &ruleName})
	if err != nil {
		if _, ok := err.(*cloudwatchevents.ResourceNotFoundException); ok {
			return nil
		}
		return err
	}

	var targetIds []*string
	for _, target := range targetsOutput.Targets {
		targetIds = append(targetIds, target.Id)
	}
	_, err = svc.RemoveTargets(&cloudwatchevents.RemoveTargetsInput{Rule: &ruleName, Ids: targetIds})
	if err != nil {
		return err
	}

	_, err = svc.DeleteRule(&cloudwatchevents.DeleteRuleInput{
		Name: &ruleName,
	})
	if err != nil {
		return err
	}
	log.Debug().Msgf("cloud watch event rule %s deleted", ruleName)

	return nil
}

func GetCloudWatchEventRuleVersion(ruleName string) (version string, err error) {
	svc := connectors.GetAWSSession().CloudWatchEvents

	ruleOutput, err := svc.DescribeRule(&cloudwatchevents.DescribeRuleInput{
		Name: &ruleName,
	})
	if err != nil {
		if _, ok := err.(*cloudwatchevents.ResourceNotFoundException); ok {
			return "", nil
		}
		return
	}

	tagsOutput, err := svc.ListTagsForResource(&cloudwatchevents.ListTagsForResourceInput{
		ResourceARN: ruleOutput.Arn,
	})
	if err != nil {
		return
	}
	for _, tag := range tagsOutput.Tags {
		if *tag.Key == common.VersionTagKey {
			version = *tag.Value
			return
		}
	}

	return
}
