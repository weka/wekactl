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
	"wekactl/internal/cluster"
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

func CreateLambda(tags cluster.TagsRefsValues, lambdaType LambdaType, resourceName, roleArn, asgName, tableName string, role common.InstanceRole, vpcConfig lambda.VpcConfig) (*lambda.FunctionConfiguration, error) {
	svc := connectors.GetAWSSession().Lambda

	bucket, err := dist.GetLambdaBucket()
	if err != nil {
		return nil, err
	}

	lambdaPackage := string(dist.WekaCtl)
	lambdaHandler := "lambdas-bin"
	runtime := "go1.x"

	s3Key := fmt.Sprintf("%s/%s", dist.LambdasID, lambdaPackage)

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
				"ROLE":       aws.String(string(role)),
			},
		},
		Handler:      aws.String(lambdaHandler),
		FunctionName: aws.String(lambdaName),
		MemorySize:   aws.Int64(256),
		Publish:      aws.Bool(true),
		Role:         &roleArn,
		Runtime:      aws.String(runtime),
		Tags:         tags,
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
					logging.UserProgress("waiting 10 sec for IAM role trust entity to finish update on \"%s\" lambda ...", lambdaName)
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

func DeleteLambda(lambdaName string) error {
	svc := connectors.GetAWSSession().Lambda
	_, err := svc.DeleteFunction(&lambda.DeleteFunctionInput{
		FunctionName: &lambdaName,
	})
	if err != nil {
		if _, ok := err.(*lambda.ResourceNotFoundException); !ok {
			return err
		}
	} else {
		log.Debug().Msgf("lambda %s was deleted successfully", lambdaName)
	}
	return nil
}

func GetLambdaVersion(lambdaName string) (version string, err error) {
	svc := connectors.GetAWSSession().Lambda
	lambdaOutput, err := svc.GetFunction(&lambda.GetFunctionInput{
		FunctionName: &lambdaName,
	})

	if err != nil {
		if _, ok := err.(*lambda.ResourceNotFoundException); ok {
			return "", nil
		}
		return
	}

	for key, value := range lambdaOutput.Tags {
		if key == cluster.VersionTagKey {
			version = *value
			return
		}
	}
	return
}

func GetLambdaArn(lambdaName string) (arn string, err error) {
	svc := connectors.GetAWSSession().Lambda
	lambdaOutput, err := svc.GetFunction(&lambda.GetFunctionInput{
		FunctionName: &lambdaName,
	})
	if err != nil {
		return
	}
	arn = *lambdaOutput.Configuration.FunctionArn
	return
}

func InvokePolicyExists(lambdaName string) bool {
	svc := connectors.GetAWSSession().Lambda
	_, err := svc.GetPolicy(&lambda.GetPolicyInput{
		FunctionName: &lambdaName,
	})
	return err == nil
}

func UpdateLambdaHandler(lambdaName string, versionTag cluster.TagsRefsValues) error {
	svc := connectors.GetAWSSession().Lambda
	bucket, err := dist.GetLambdaBucket()
	if err != nil {
		return err
	}

	lambdaPackage := string(dist.WekaCtl)
	s3Key := fmt.Sprintf("%s/%s", dist.LambdasID, lambdaPackage)

	_, err = svc.UpdateFunctionCode(&lambda.UpdateFunctionCodeInput{
		FunctionName: &lambdaName,
		S3Bucket:     aws.String(bucket),
		S3Key:        aws.String(s3Key),
	})

	if err != nil {
		return err
	}

	functionInfo, err := svc.GetFunction(&lambda.GetFunctionInput{
		FunctionName: &lambdaName,
	})

	if err != nil {
		return err
	}

	_, err = svc.TagResource(&lambda.TagResourceInput{
		Resource: functionInfo.Configuration.FunctionArn,
		Tags:     versionTag,
	})

	return err
}
