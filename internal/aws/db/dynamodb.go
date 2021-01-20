package db

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/rs/zerolog/log"
	"wekactl/internal/connectors"
)

func PutItem(tableName string, item interface{}) error {
	svc := connectors.GetAWSSession().DynamoDB
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		log.Debug().Msg("Got error marshalling user name and password")
		return err
	}
	_, err = svc.PutItem(&dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tableName),
	})
	return err
}
