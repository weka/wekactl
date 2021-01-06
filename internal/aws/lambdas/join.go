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
	Username string
	Password string
	Ips      []string
}

func getAutoScalingGroupInstanceIps(asgName string) ([]string, error) {
	asgsvc := connectors.GetAWSSession().ASG
	input := &autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{&asgName}}
	result, err := asgsvc.DescribeAutoScalingGroups(input)
	if err != nil {
		return nil, err
	} else {
		var instanceIds []*string
		if len(result.AutoScalingGroups) == 0{
			return []string{}, nil
		}
		for _, instance := range result.AutoScalingGroups[0].Instances {
			instanceIds = append(instanceIds, instance.InstanceId)
		}
		ec2svc := connectors.GetAWSSession().EC2
		input := &ec2.DescribeInstancesInput{InstanceIds: instanceIds}
		result, err := ec2svc.DescribeInstances(input)
		if err != nil {
			return nil, err
		} else {
			var instanceIps []string
			for _, reservation := range result.Reservations {
				if len(reservation.Instances) > 0{
					instanceIps = append(instanceIps, *reservation.Instances[0].PublicIpAddress)
				}
			}
			return instanceIps, nil
		}
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
	ips, err := getAutoScalingGroupInstanceIps(asgName)
	if err != nil {
		return "", err
	} else {
		username, password, err := getUsernameAndPassword(tableName)
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
