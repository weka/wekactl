package hostgroup

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	uuid "github.com/nu7hatch/gouuid"
	"log"
	"strings"
	"wekactl/internal/aws/common"
)

func getTagValue(tags []*ec2.Tag, key string) string{
	for _, tag := range tags{
		if *tag.Key == key{
			return *tag.Value
		}
	}
	return ""
}

func getInstanceRole(region, instanceId string) string{
	sess := common.NewSession(region)
	svc := ec2.New(sess)
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceId),
		},
	}
	result, err := svc.DescribeInstances(input)
	if err != nil {
		log.Fatal(err)
	} else {
		tagValue:= getTagValue(result.Reservations[0].Instances[0].Tags, "WekaInstallationId")
		if strings.Contains(tagValue, "Backend"){
			return "backend"
		} else if strings.Contains(tagValue, "Client"){
			return "client"
		} else {
			log.Fatalf("InstanceId: %s is not a weka cluster instance role\n", instanceId)
		}
	}
	return ""
}


func CreateAutoScalingGroup(region, instanceId string, minSize,maxSize int64) string{
	sess := common.NewSession(region)
	svc := autoscaling.New(sess)
	role:= getInstanceRole(region, instanceId)
	u, err := uuid.NewV4()
	if err != nil {
		log.Fatal(err)
	}
	name:= "weka-" + role + "-" + u.String()
	input := &autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName:    aws.String(name),
		InstanceId: aws.String(instanceId),
		MinSize:                 aws.Int64(minSize),
		MaxSize:                 aws.Int64(maxSize),
	}
	_, err = svc.CreateAutoScalingGroup(input)
	if err != nil {
		log.Fatal(err)
	}
	return name
}
