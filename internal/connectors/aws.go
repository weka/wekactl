package connectors

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/kms"
	"sync"
	"wekactl/internal/env"
)

type SAwsSession struct {
	sync.RWMutex
	Session  *session.Session
	CF       *cloudformation.CloudFormation
	EC2      *ec2.EC2
	ASG      *autoscaling.AutoScaling
	KMS      *kms.KMS
	DynamoDB *dynamodb.DynamoDB
}

var awsSession SAwsSession

func GetAWSSession() *SAwsSession {
	if awsSession.Session != nil {
		return &awsSession
	}
	awsSession.Lock()
	defer awsSession.Unlock()
	if awsSession.Session == nil {
		awsSession.Session = newSession(env.Config.Region)
		awsSession.CF = cloudformation.New(awsSession.Session)
		awsSession.EC2 = ec2.New(awsSession.Session)
		awsSession.ASG = autoscaling.New(awsSession.Session)
		awsSession.KMS = kms.New(awsSession.Session)
		awsSession.DynamoDB = dynamodb.New(awsSession.Session)
	}
	return &awsSession
}

func newSession(region string) *session.Session {
	// TODO: Double-check if works profile
	config := aws.NewConfig()
	config = config.WithRegion(region)
	config = config.WithCredentialsChainVerboseErrors(true)

	// Create the options for the Session
	opts := session.Options{
		Config:                  *config,
		SharedConfigState:       session.SharedConfigEnable,
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
	}

	return session.Must(session.NewSessionWithOptions(opts))
}
