package debug

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/google/uuid"
	"log"
	"strings"
	"wekactl/internal/connectors"
)

func getTagValue(tags []*ec2.Tag, key string) string {
	for _, tag := range tags {
		if *tag.Key == key {
			return *tag.Value
		}
	}
	return ""
}

func getInstanceRole(instanceId string) string {
	svc := connectors.GetAWSSession().EC2
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceId),
		},
	}
	result, err := svc.DescribeInstances(input)
	if err != nil {
		log.Fatal(err)
	} else {
		tagValue := getTagValue(result.Reservations[0].Instances[0].Tags, "WekaInstallationId")
		if strings.Contains(tagValue, "Backend") {
			return "backend"
		} else if strings.Contains(tagValue, "Client") {
			return "client"
		} else {
			log.Fatalf("InstanceId: %s is not a weka cluster instance role\n", instanceId)
		}
	}
	return ""
}

func CreateAutoScalingGroup(instanceId string, minSize, maxSize int64) string {
	svc := connectors.GetAWSSession().ASG
	role := getInstanceRole(instanceId)
	u := uuid.New().String()
	name := "weka-" + role + "-" + u
	input := &autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(name),
		InstanceId:           aws.String(instanceId),
		MinSize:              aws.Int64(minSize),
		MaxSize:              aws.Int64(maxSize),
	}
	_, err := svc.CreateAutoScalingGroup(input)
	if err != nil {
		log.Fatal(err)
	}
	return name
}
