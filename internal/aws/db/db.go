package db

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/rs/zerolog/log"
	"strings"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
	"wekactl/internal/logging"
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

func GetItem(tableName string, key string, item interface{}) error {
	svc := connectors.GetAWSSession().DynamoDB
	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"Key": {
				S: aws.String(key),
			},
		},
	})
	if err != nil {
		return err
	}

	err = dynamodbattribute.UnmarshalMap(result.Item, &item)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func CreateDb(tableName, kmsKey string, tags cluster.Tags) error {
	svc := connectors.GetAWSSession().DynamoDB

	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("Key"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("Key"),
				KeyType:       aws.String("HASH"),
			},
		},
		BillingMode: aws.String(dynamodb.BillingModePayPerRequest),
		TableName:   aws.String(tableName),
		Tags:        tags.ToDynamoDb(),
		SSESpecification: &dynamodb.SSESpecification{
			Enabled:        aws.Bool(true),
			KMSMasterKeyId: &kmsKey,
			SSEType:        aws.String("KMS"),
		},
	}

	_, err := svc.CreateTable(input)
	if err != nil {
		log.Debug().Msg("Failed creating table")
		return err
	}

	logging.UserProgress("Waiting for table \"%s\" to be created ...", tableName)
	err = svc.WaitUntilTableExists(&dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})

	if err != nil {
		return err
	}

	logging.UserProgress("Table %s was created successfully!", tableName)
	return nil
}

func SaveCredentials(tableName string, username, password string) error {
	err := PutItem(tableName, ClusterCreds{
		Key:      ModelClusterCreds,
		Username: username,
		Password: password,
	})
	if err != nil {
		log.Debug().Msgf("error saving credentials to DB %v", err)
		return err
	}
	log.Debug().Msgf("Username:%s and Password:%s were added to DB successfully!", username, strings.Repeat("*", len(password)))
	return nil
}

func saveClusterParams(tableName string, params DefaultClusterParams) error {
	if params.Key == "" {
		params.Key = ModelDefaultClusterParams
	}
	err := PutItem(tableName, params)
	if err != nil {
		log.Debug().Msgf("error saving cluster params to DB %v", err)
		return err
	}
	return nil
}

func DeleteDB(tableName string) error {
	svc := connectors.GetAWSSession().DynamoDB
	_, err := svc.DeleteTable(&dynamodb.DeleteTableInput{
		TableName: &tableName,
	})
	if err != nil {
		if _, ok := err.(*dynamodb.ResourceNotFoundException); !ok {
			return err
		}
	} else {
		log.Debug().Msgf("DB %s was deleted successfully", tableName)
	}

	return nil
}

func GetDbVersion(tableName string) (version string, err error) {
	svc := connectors.GetAWSSession().DynamoDB
	dbOutput, err := svc.DescribeTable(&dynamodb.DescribeTableInput{TableName: &tableName})
	if err != nil {
		if _, ok := err.(*dynamodb.ResourceNotFoundException); ok {
			return "", nil
		}
		return
	}

	tagsOutput, err := svc.ListTagsOfResource(&dynamodb.ListTagsOfResourceInput{
		ResourceArn: dbOutput.Table.TableArn,
	})
	if err != nil {
		return
	}

	for _, tag := range tagsOutput.Tags {
		if *tag.Key == cluster.VersionTagKey {
			version = *tag.Value
			return
		}
	}

	return
}
