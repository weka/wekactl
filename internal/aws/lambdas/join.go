package lambdas

import (
	"encoding/json"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/ec2"
	"wekactl/internal/aws/cluster"
	"wekactl/internal/connectors"
)

type JoinInfo struct {
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	PrivateIps      []string `json:"private_ips"`
	DesiredCapacity int      `json:"desired_capacity"`
}

func getAutoScalingGroupDesiredCapacity(asgOutput *autoscaling.DescribeAutoScalingGroupsOutput) int {
	if len(asgOutput.AutoScalingGroups) == 0 {
		return -1
	}

	return int(*asgOutput.AutoScalingGroups[0].DesiredCapacity)
}

func getAutoScalingGroupInstanceIds(asgOutput *autoscaling.DescribeAutoScalingGroupsOutput) []*string {
	var instanceIds []*string
	if len(asgOutput.AutoScalingGroups) == 0 {
		return []*string{}
	}
	for _, instance := range asgOutput.AutoScalingGroups[0].Instances {
		instanceIds = append(instanceIds, instance.InstanceId)
	}
	return instanceIds
}

func getAutoScalingGroupInstanceIps(instanceIds []*string) ([]string, error) {

	ec2svc := connectors.GetAWSSession().EC2
	input := &ec2.DescribeInstancesInput{InstanceIds: instanceIds}
	result, err := ec2svc.DescribeInstances(input)
	if err != nil {
		return nil, err
	} else {
		var instanceIps []string
		for _, reservation := range result.Reservations {
			if len(reservation.Instances) > 0 {
				instanceIps = append(instanceIps, *reservation.Instances[0].PrivateIpAddress)
			}
		}
		return instanceIps, nil
	}
}

func getUsernameAndPassword(tableName string) (string, string, error) {
	svc := connectors.GetAWSSession().DynamoDB
	input := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"Key": {
				S: aws.String("cluster-creds"),
			},
		},
	}
	result, err := svc.GetItem(input)
	if err != nil {
		return "", "", err
	} else if result.Item == nil {
		return "", "", errors.New("couldn't find stackId")
	} else {
		item := cluster.JoinParamsDb{}
		err = dynamodbattribute.UnmarshalMap(result.Item, &item)
		if err != nil {
			return "", "", err
		} else {
			return item.Username, item.Password, err
		}
	}
}

func GetJoinParams(asgName, tableName string) (string, error) {
	svc := connectors.GetAWSSession().ASG
	input := &autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{&asgName}}
	asgOutput, err := svc.DescribeAutoScalingGroups(input)
	if err != nil {
		return "", err
	}

	instanceIds := getAutoScalingGroupInstanceIds(asgOutput)
	ips, err := getAutoScalingGroupInstanceIps(instanceIds)
	if err != nil {
		return "", err
	}

	var ids []string
	for _, instanceId := range instanceIds {
		ids = append(ids, *instanceId)
	}

	username, password, err := getUsernameAndPassword(tableName)
	if err != nil {
		return "", err
	}

	joinInfo := JoinInfo{
		Username:        username,
		Password:        password,
		PrivateIps:      ips,
		DesiredCapacity: getAutoScalingGroupDesiredCapacity(asgOutput),
	}
	js, err := json.Marshal(joinInfo)
	if err != nil {
		return "", err
	}

	return string(js), nil
}
