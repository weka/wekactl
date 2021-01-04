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
	"wekactl/internal/aws/common"
)

type JoinInfo struct {
	Username string
	Password string
	Ips      []string
}

func getAutoScalingGroupInstanceIps(region, asgName string) ([]string, error) {
	sess := common.NewSession(region)
	svc := autoscaling.New(sess)
	input := &autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{&asgName}}
	result, err := svc.DescribeAutoScalingGroups(input)
	if err != nil {
		return nil, err
	} else {
		var instanceIds []*string
		for _, instance := range result.AutoScalingGroups[0].Instances {
			instanceIds = append(instanceIds, instance.InstanceId)
		}
		svc := ec2.New(sess)
		input := &ec2.DescribeInstancesInput{InstanceIds: instanceIds}
		result, err := svc.DescribeInstances(input)
		if err != nil {
			return nil, err
		} else {
			var instanceIps []string
			for _, reservation := range result.Reservations {
				instanceIps = append(instanceIps, *reservation.Instances[0].PublicIpAddress)
			}
			return instanceIps, nil
		}
	}
}

func getUsernameAndPassword(region, tableName string) (string, string, error) {
	sess := common.NewSession(region)
	svc := dynamodb.New(sess)
	input := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"Key": {
				S: aws.String("join-params"),
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

func GetJoinParams(region, asgName, tableName string) (string, error) {
	ips, err := getAutoScalingGroupInstanceIps(region, asgName)
	if err != nil {
		return "", err
	} else {
		username, password, err := getUsernameAndPassword(region, tableName)
		if err != nil {
			return "", err
		} else {
			joinInfo := JoinInfo{
				Username: username,
				Password: password,
				Ips:      ips,
			}
			js, err := json.Marshal(joinInfo)
			if err != nil {
				return "", err
			} else {
				return string(js), nil
			}
		}
	}
}
