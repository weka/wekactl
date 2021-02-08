package lambdas

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/rs/zerolog/log"
	"time"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/dist"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/connectors"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

func GetLambdaVpcConfig(subnetId string, securityGroupIds []*string) lambda.VpcConfig {
	return lambda.VpcConfig{
		SubnetIds:        []*string{&subnetId},
		SecurityGroupIds: securityGroupIds,
	}
}

func CreateLambda(hostGroupInfo hostgroups.HostGroupInfo, lambdaType LambdaType, resourceName, roleArn string, vpcConfig lambda.VpcConfig) (*lambda.FunctionConfiguration, error) {
	svc := connectors.GetAWSSession().Lambda

	bucket, err := dist.GetLambdaBucket()
	if err != nil {
		return nil, err
	}

	lambdaPackage := string(dist.WekaCtl)
	lambdaHandler := "lambdas-bin"
	runtime := "go1.x"

	s3Key := fmt.Sprintf("%s/%s", dist.LambdasID, lambdaPackage)

	asgName := common.GenerateResourceName(hostGroupInfo.ClusterName, hostGroupInfo.Name)
	tableName := common.GenerateResourceName(hostGroupInfo.ClusterName, "")
	lambdaName := resourceName

	input := &lambda.CreateFunctionInput{
		Code: &lambda.FunctionCode{
			S3Bucket: aws.String(bucket),
			S3Key:    aws.String(s3Key),
		},
		Description: aws.String(fmt.Sprintf("Wekactl %s", string(lambdaType))),
		Environment: &lambda.Environment{
			Variables: map[string]*string{
				"LAMBDA":     aws.String(string(lambdaType)),
				"REGION":     aws.String(env.Config.Region),
				"ASG_NAME":   aws.String(asgName),
				"TABLE_NAME": aws.String(tableName),
				"ROLE":       aws.String(string(hostGroupInfo.Role)),
			},
		},
		Handler:      aws.String(lambdaHandler),
		FunctionName: aws.String(lambdaName),
		MemorySize:   aws.Int64(256),
		Publish:      aws.Bool(true),
		Role:         &roleArn,
		Runtime:      aws.String(runtime),
		Tags:         common.GetMapCommonTags(hostGroupInfo),
		Timeout:      aws.Int64(15),
		TracingConfig: &lambda.TracingConfig{
			Mode: aws.String("Active"),
		},
		VpcConfig: &vpcConfig,
	}

	var lambdaCreateOutput *lambda.FunctionConfiguration

	// it takes some time for the trust entity to be updated
	retry := true
	for i := 0; i < 3 && retry; i++ {
		retry = false
		log.Debug().Msgf("try %d: creating lambda %s using: %s", i+1, lambdaName, s3Key)
		lambdaCreateOutput, err = svc.CreateFunction(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == lambda.ErrCodeInvalidParameterValueException {
					logging.UserProgress("%s \"%s\" lambda creation failed, waiting for 10 sec for IAM role trust entity to finish update", string(hostGroupInfo.Name), string(lambdaType))
					time.Sleep(10 * time.Second)
					retry = true
				}
			}
		}
	}
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("lambda %s was created successfully!", lambdaName)

	return lambdaCreateOutput, nil
}
