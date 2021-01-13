package lambdas

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"wekactl/internal/aws/cluster"
	"wekactl/internal/connectors"
)

func getAutoScalingGroupDesiredCapacity(asgOutput *autoscaling.DescribeAutoScalingGroupsOutput) int {
	if len(asgOutput.AutoScalingGroups) == 0 {
		return -1
	}

	return int(*asgOutput.AutoScalingGroups[0].DesiredCapacity)
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
