package apigateway

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
	"wekactl/internal/env"
)

const policyTemplate = `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Deny",
            "Principal": "*",
            "Action": "execute-api:Invoke",
            "Resource": "*",
            "Condition": {
                "StringNotEquals": {
                    "aws:sourceVpc": "%s"
                }
            }
        },
        {
            "Effect": "Allow",
            "Principal": "*",
            "Action": "execute-api:Invoke",
            "Resource": "*"
        }
    ]
}`

func getAccountId() (string, error) {
	svc := connectors.GetAWSSession().STS
	result, err := svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return *result.Account, nil
}

func createRestApiGateway(tags cluster.TagsRefsValues, lambdaUri string, apiGatewayName string, vpcId string) (restApiGateway RestApiGateway, err error) {
	svc := connectors.GetAWSSession().ApiGateway

	var endpointType, policy string
	if vpcId != "" {
		endpointType = "PRIVATE"
		policy = fmt.Sprintf(policyTemplate, vpcId)
	} else {
		endpointType = "EDGE"
	}

	restApi, err := svc.CreateRestApi(&apigateway.CreateRestApiInput{
		Name:         aws.String(apiGatewayName),
		Tags:         tags,
		Description:  aws.String("Wekactl host join info API"),
		ApiKeySource: aws.String("HEADER"),
		EndpointConfiguration: &apigateway.EndpointConfiguration{
			Types: []*string{aws.String(endpointType)},
		},
		Policy: aws.String(policy),
	})
	if err != nil {
		return
	}
	restApiId := restApi.Id
	log.Debug().Msgf("rest api gateway id:%s for lambda:%s was created successfully!", *restApiId, apiGatewayName)

	resources, err := svc.GetResources(&apigateway.GetResourcesInput{
		RestApiId: restApiId,
	})
	if err != nil {
		return
	}

	rootResource := resources.Items[0]

	createResourceOutput, err := svc.CreateResource(&apigateway.CreateResourceInput{
		ParentId:  rootResource.Id,
		RestApiId: restApiId,
		PathPart:  aws.String(apiGatewayName),
	})
	if err != nil {
		return
	}
	log.Debug().Msgf("rest api gateway resource %s was created successfully!", apiGatewayName)

	httpMethod := "GET"

	_, err = svc.PutMethod(&apigateway.PutMethodInput{
		RestApiId:         restApiId,
		ResourceId:        createResourceOutput.Id,
		HttpMethod:        aws.String(httpMethod),
		AuthorizationType: aws.String("NONE"),
		ApiKeyRequired:    aws.Bool(true),
	})
	if err != nil {
		return
	}
	log.Debug().Msgf("rest api %s method was created successfully!", httpMethod)

	log.Debug().Msgf("creating rest api %s method integration with uri: %s", httpMethod, lambdaUri)
	_, err = svc.PutIntegration(&apigateway.PutIntegrationInput{
		RestApiId:             restApiId,
		ResourceId:            createResourceOutput.Id,
		HttpMethod:            aws.String(httpMethod),
		Type:                  aws.String("AWS_PROXY"),
		IntegrationHttpMethod: aws.String("POST"),
		Uri:                   aws.String(lambdaUri),
	})
	if err != nil {
		return
	}
	log.Debug().Msgf("rest api %s method integration created successfully!", httpMethod)

	stageName := "default"
	_, err = svc.CreateDeployment(&apigateway.CreateDeploymentInput{
		RestApiId: restApiId,
		StageName: aws.String(stageName),
	})
	if err != nil {
		return
	}
	log.Debug().Msgf("rest api gateway deployment for stage %s was created successfully!", stageName)

	resourceName := apiGatewayName
	usagePlanOutput, err := svc.CreateUsagePlan(&apigateway.CreateUsagePlanInput{
		Name: aws.String(resourceName),
		ApiStages: []*apigateway.ApiStage{
			{
				ApiId: restApiId,
				Stage: aws.String("default"),
			},
		},
	})
	if err != nil {
		return
	}
	log.Debug().Msgf("usage plan %s was created successfully!", *usagePlanOutput.Name)

	apiKeyOutput, err := svc.CreateApiKey(&apigateway.CreateApiKeyInput{
		Enabled: aws.Bool(true),
		Name:    aws.String(resourceName),
		Tags:    tags,
	})
	if err != nil {
		return
	}
	log.Debug().Msgf("api key %s was created successfully!", *apiKeyOutput.Name)

	_, err = svc.CreateUsagePlanKey(&apigateway.CreateUsagePlanKeyInput{
		UsagePlanId: usagePlanOutput.Id,
		KeyId:       apiKeyOutput.Id,
		KeyType:     aws.String("API_KEY"),
	})
	if err != nil {
		return
	}
	log.Debug().Msg("api key was associated to usage plan successfully!")

	restApiGateway = RestApiGateway{
		Id:     *restApiId,
		Name:   apiGatewayName,
		ApiKey: *apiKeyOutput.Value,
	}
	return
}

func addLambdaInvokePermissions(lambdaName, restApiId, apiGatewayName string) error {
	svc := connectors.GetAWSSession().Lambda
	account, err := getAccountId()
	if err != nil {
		return err
	}
	sourceArn := fmt.Sprintf("arn:aws:execute-api:%s:%s:%s/*/GET/%s", env.Config.Region, account, restApiId, apiGatewayName)
	_, err = svc.AddPermission(&lambda.AddPermissionInput{
		FunctionName: aws.String(lambdaName),
		StatementId:  aws.String(lambdaName + "-" + uuid.New().String()),
		Action:       aws.String("lambda:InvokeFunction"),
		Principal:    aws.String("apigateway.amazonaws.com"),
		SourceArn:    aws.String(sourceArn),
	})
	if err != nil {
		return err
	}
	return nil
}

func CreateJoinApi(tags cluster.TagsRefsValues, lambdaArn, lambdaName, apiGatewayName, vpcId string) (restApiGateway RestApiGateway, err error) {

	lambdaUri := fmt.Sprintf(
		"arn:aws:apigateway:%s:lambda:path/2015-03-31/functions/%s/invocations",
		env.Config.Region, lambdaArn)

	restApiGateway, err = createRestApiGateway(tags, lambdaUri, apiGatewayName, vpcId)

	if err != nil {
		return
	}

	err = addLambdaInvokePermissions(lambdaName, restApiGateway.Id, restApiGateway.Name)
	if err != nil {
		return
	}

	return
}

func DeleteRestApiGateway(resourceName string) error {
	svc := connectors.GetAWSSession().ApiGateway

	restApisOutput, err := svc.GetRestApis(&apigateway.GetRestApisInput{})
	if err != nil {
		return err
	}
	for _, restApi := range restApisOutput.Items {
		if *restApi.Name != resourceName {
			continue
		}
		_, err = svc.DeleteRestApi(&apigateway.DeleteRestApiInput{
			RestApiId: restApi.Id,
		})
		if err != nil {
			return err
		}
		log.Debug().Msgf("rest api gateway %s %s was deleted successfully", resourceName, *restApi.Id)
	}

	usagePlansOutput, err := svc.GetUsagePlans(&apigateway.GetUsagePlansInput{})
	if err != nil {
		return err
	}
	for _, usagePlan := range usagePlansOutput.Items {
		if *usagePlan.Name != resourceName {
			continue
		}
		_, err := svc.DeleteUsagePlan(&apigateway.DeleteUsagePlanInput{
			UsagePlanId: usagePlan.Id,
		})
		if err != nil {
			return err
		}
		log.Debug().Msgf("rest api gateway %s %s usage plan was deleted successfully", resourceName, *usagePlan.Id)
	}

	apiKeysOutput, err := svc.GetApiKeys(&apigateway.GetApiKeysInput{})
	if err != nil {
		return err
	}
	for _, apiKey := range apiKeysOutput.Items {
		if *apiKey.Name != resourceName {
			continue
		}

		_, err = svc.DeleteApiKey(&apigateway.DeleteApiKeyInput{
			ApiKey: apiKey.Id,
		})
		if err != nil {
			return err
		}
		log.Debug().Msgf("rest api gateway %s %s api key was deleted successfully", resourceName, *apiKey.Id)
	}

	return nil
}

func GetRestApiGatewayVersion(resourceName string) (version string, err error) {
	svc := connectors.GetAWSSession().ApiGateway

	restApisOutput, err := svc.GetRestApis(&apigateway.GetRestApisInput{})
	if err != nil {
		return
	}
	for _, restApi := range restApisOutput.Items {
		if *restApi.Name != resourceName {
			continue
		}
		for key, value := range restApi.Tags {
			if key == cluster.VersionTagKey {
				version = *value
				return
			}
		}
	}
	return
}

func GetRestApi(resourceName string) (restApiResource *apigateway.RestApi, err error) {
	svc := connectors.GetAWSSession().ApiGateway

	restApisOutput, err := svc.GetRestApis(&apigateway.GetRestApisInput{})
	if err != nil {
		return
	}
	for _, restApi := range restApisOutput.Items {
		if *restApi.Name != resourceName {
			continue
		}
		restApiResource = restApi
		break
	}
	return
}

func GetRestApiGateway(resourceName string) (restApiGateway RestApiGateway, err error) {
	svc := connectors.GetAWSSession().ApiGateway

	restApi, err := GetRestApi(resourceName)
	if err != nil {
		return
	}
	if restApi == nil {
		err = errors.New("api gateway wasn't found")
		return
	}
	restApiGateway.Id = *restApi.Id

	apiKeysOutput, err := svc.GetApiKeys(&apigateway.GetApiKeysInput{IncludeValues: aws.Bool(true)})
	if err != nil {
		return
	}
	for _, apiKey := range apiKeysOutput.Items {
		if *apiKey.Name != resourceName {
			continue
		}
		restApiGateway.ApiKey = *apiKey.Value
		break
	}
	if restApiGateway.ApiKey == "" {
		err = errors.New("api key wasn't found")
		return
	}

	restApiGateway.Name = resourceName
	return
}

func GetRestApiGatewayTags(resourceName string) (tags cluster.Tags, err error) {
	restApi, err := GetRestApi(resourceName)
	if err != nil {
		return
	}
	tags = cluster.StringRefsMapToStrings(restApi.Tags)
	return
}
