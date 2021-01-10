package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/semaphore"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"wekactl/internal/aws/dist"
	"wekactl/internal/connectors"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

type StackInstances struct {
	backends []*ec2.Instance
	clients  []*ec2.Instance
}

func (s *StackInstances) All() []*ec2.Instance {
	return append(s.clients[0:len(s.clients):len(s.clients)], s.backends...)
}

type Tag struct {
	Key   *string
	Value *string
}

type JoinParamsDb struct {
	Key      string
	Username string
	Password string
}

type Stack struct {
	StackId       string
	StackName     string
	DynamodbTable string
}

type HostGroup struct {
	Role  string
	Name  string
	Stack Stack
}

type StatementEntry struct {
	Effect   string
	Action   []string
	Resource string
}

type PolicyDocument struct {
	Version   string
	Statement []StatementEntry
}

func GetStackId(stackName string) (string, error) {
	svc := connectors.GetAWSSession().CF
	result, err := svc.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})
	if err != nil {
		log.Error().Err(err)
		return "", err
	}
	return *result.Stacks[0].StackId, nil
}

func getClusterInstances(stackName string) ([]*string, error) {
	svc := connectors.GetAWSSession().CF

	result, err := svc.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: &stackName,
	})
	var instancesIds []*string
	if err != nil {
		log.Fatal().Err(err)
	} else {
		for _, resource := range result.StackResources {
			if *resource.ResourceType == "AWS::EC2::Instance" {
				instancesIds = append(instancesIds, resource.PhysicalResourceId)
			}
		}
	}
	return instancesIds, nil
}

func getInstancesInfo(stackName string) (stackInstances StackInstances, err error) {
	svc := connectors.GetAWSSession().EC2
	instances, err := getClusterInstances(stackName)
	if err != nil {
		return
	}
	result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: instances,
	})
	if err != nil {
		return
	}

	for _, reservation := range result.Reservations {
		instance := reservation.Instances[0]
		arn := *instance.IamInstanceProfile.Arn
		if strings.Contains(arn, "InstanceProfileBackend") {
			stackInstances.backends = append(stackInstances.backends, instance)
		} else if strings.Contains(arn, "InstanceProfileClient") {
			stackInstances.clients = append(stackInstances.clients, instance)
		}

	}
	return stackInstances, nil
}

func getInstancesIdsFromEc2Instance(instances []*ec2.Instance) []*string {
	var instanceIds []*string
	for _, instance := range instances {
		instanceIds = append(instanceIds, instance.InstanceId)
	}
	return instanceIds
}

func disableInstanceApiTermination(instanceId string) (*ec2.ModifyInstanceAttributeOutput, error) {
	svc := connectors.GetAWSSession().EC2
	input := &ec2.ModifyInstanceAttributeInput{
		DisableApiTermination: &ec2.AttributeBooleanValue{
			Value: aws.Bool(true),
		},
		InstanceId: aws.String(instanceId),
	}
	return svc.ModifyInstanceAttribute(input)
}

var terminationSemaphore *semaphore.Weighted

func init() {
	terminationSemaphore = semaphore.NewWeighted(20)
}

func disableInstancesApiTermination(instances []*ec2.Instance) error {
	instanceIds := getInstancesIdsFromEc2Instance(instances)

	var wg sync.WaitGroup
	var failedInstances int64

	wg.Add(len(instanceIds))
	for i := range instanceIds {
		go func(i int) {
			_ = terminationSemaphore.Acquire(context.Background(), 1)
			defer terminationSemaphore.Release(1)
			defer wg.Done()

			_, err := disableInstanceApiTermination(*instanceIds[i])
			if err != nil {
				atomic.AddInt64(&failedInstances, 1)
				log.Error().Msgf("failed to set DisableApiTermination on %s", *instanceIds[i])
			}
		}(i)
	}
	wg.Wait()
	if failedInstances != 0 {
		return errors.New(fmt.Sprintf("failed to set DisableApiTermination on %d instances", failedInstances))
	}
	return nil

}

func getUuidFromStackId(stackId string) string {
	s := strings.Split(stackId, "/")
	return s[len(s)-1]
}

func getInstanceSecurityGroupsId(instance *ec2.Instance) []*string {
	var securityGroupIds []*string
	for _, securityGroup := range instance.SecurityGroups {
		securityGroupIds = append(securityGroupIds, securityGroup.GroupId)
	}
	return securityGroupIds
}

func getCommonTags(stack Stack) []Tag {
	tags := []Tag{
		{
			Key:   aws.String("wekactl.io/managed"),
			Value: aws.String("true"),
		},
		{
			Key:   aws.String("wekactl.io/api_version"),
			Value: aws.String("v1"),
		},
		{
			Key:   aws.String("wekactl.io/stack_id"),
			Value: aws.String(stack.StackId),
		},
	}
	return tags
}

func getHostGroupTags(hostGroup HostGroup) []Tag {
	tags := getCommonTags(hostGroup.Stack)
	tags = append(
		tags, Tag{
			Key:   aws.String("wekactl.io/name"),
			Value: aws.String(hostGroup.Name),
		}, Tag{
			Key:   aws.String("wekactl.io/hostgroup_type"),
			Value: aws.String(hostGroup.Role),
		},
	)
	return tags
}

func getEc2Tags(name, role, stackId string) []*ec2.Tag {
	var ec2Tags []*ec2.Tag
	for _, tag := range getHostGroupTags(HostGroup{
		Name:  name,
		Role:  role,
		Stack: Stack{StackId: stackId},
	}) {
		ec2Tags = append(ec2Tags, &ec2.Tag{
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	return ec2Tags
}

func generateResourceName(stackId, stackName, resourceName string) string {
	name := "weka-" + stackName + "-"
	if resourceName != "" {
		name += resourceName + "-"
	}
	return name + getUuidFromStackId(stackId)
}

func createLaunchTemplate(stackId, stackName, name string, role string, instance *ec2.Instance) string {
	svc := connectors.GetAWSSession().EC2
	launchTemplateName := generateResourceName(stackId, stackName, name)
	input := &ec2.CreateLaunchTemplateInput{
		LaunchTemplateData: &ec2.RequestLaunchTemplateData{
			ImageId:      instance.ImageId,
			InstanceType: instance.InstanceType,
			KeyName:      instance.KeyName,
			UserData:     aws.String(""), // TODO: add necessary init script here
			TagSpecifications: []*ec2.LaunchTemplateTagSpecificationRequest{
				{
					ResourceType: aws.String("instance"),
					Tags:         getEc2Tags(name, role, stackId),
				},
			},
			NetworkInterfaces: []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{
				{
					AssociatePublicIpAddress: aws.Bool(true),
					DeviceIndex:              aws.Int64(0),
					Ipv6AddressCount:         aws.Int64(0),
					SubnetId:                 instance.SubnetId,
					Groups:                   getInstanceSecurityGroupsId(instance),
				},
			},
		},
		VersionDescription: aws.String("v1"),
		LaunchTemplateName: aws.String(launchTemplateName),
	}

	_, err := svc.CreateLaunchTemplate(input)
	if err != nil {
		log.Fatal().Err(err)
	}
	log.Debug().Msgf("LaunchTemplate: \"%s\" was created successfully!", launchTemplateName)
	return launchTemplateName
}

func getAutoScalingTags(name, role, stackId string) []*autoscaling.Tag {
	var autoscalingTags []*autoscaling.Tag
	for _, tag := range getHostGroupTags(HostGroup{
		Name:  name,
		Role:  role,
		Stack: Stack{StackId: stackId},
	}) {
		autoscalingTags = append(autoscalingTags, &autoscaling.Tag{
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	return autoscalingTags
}

func createAutoScalingGroup(stackId, stackName, name string, role string, maxSize int, launchTemplateName string) (string, error) {
	hostGroup := HostGroup{
		Name: name,
		Role: role,
		Stack: Stack{
			StackId:   stackId,
			StackName: stackName,
		},
	}
	err := CreateLambdaEndPoint(hostGroup, "join", "Backends")
	if err != nil {
		return "", err
	}
	_, err = createLambda(hostGroup, "fetch", "Backends")
	if err != nil {
		return "", err
	}

	svc := connectors.GetAWSSession().ASG
	resourceName := generateResourceName(stackId, stackName, name)
	input := &autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName:             aws.String(resourceName),
		NewInstancesProtectedFromScaleIn: aws.Bool(true),
		LaunchTemplate: &autoscaling.LaunchTemplateSpecification{
			LaunchTemplateName: aws.String(launchTemplateName),
			Version:            aws.String("1"),
		},
		MinSize: aws.Int64(0),
		MaxSize: aws.Int64(int64(maxSize)),
		Tags:    getAutoScalingTags(name, role, stackId),
	}
	_, err = svc.CreateAutoScalingGroup(input)
	if err != nil {
		return "", err
	}
	log.Debug().Msgf("AutoScalingGroup: \"%s\" was created successfully!", resourceName)
	return resourceName, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func attachInstancesToAutoScalingGroups(roleInstances []*ec2.Instance, autoScalingGroupsName string) error {
	svc := connectors.GetAWSSession().ASG
	limit := 20
	instancesIds := getInstancesIdsFromEc2Instance(roleInstances)
	for i := 0; i < len(instancesIds); i += limit {
		batch := instancesIds[i:min(i+limit, len(instancesIds))]
		_, err := svc.AttachInstances(&autoscaling.AttachInstancesInput{
			AutoScalingGroupName: &autoScalingGroupsName,
			InstanceIds:          batch,
		})
		if err != nil {
			return err
		}
		log.Debug().Msgf("Attached %d instances to %s successfully!", len(batch), autoScalingGroupsName)
	}
	return nil
}

func getKMSTags(stackId string) []*kms.Tag {
	var kmsTags []*kms.Tag
	for _, tag := range getCommonTags(Stack{StackId: stackId}) {
		kmsTags = append(kmsTags, &kms.Tag{
			TagKey:   tag.Key,
			TagValue: tag.Value,
		})
	}
	return kmsTags
}

func createKMSKey(stackId, stackName string) (*string, error) {
	svc := connectors.GetAWSSession().KMS

	input := &kms.CreateKeyInput{
		Tags: getKMSTags(stackId),
	}
	result, err := svc.CreateKey(input)
	if err != nil {
		log.Debug().Msgf(err.Error())
		return nil, err
	} else {
		log.Debug().Msgf("KMS key %s was created successfully!", *result.KeyMetadata.KeyId)
		alias := generateResourceName(stackId, stackName, "")
		input := &kms.CreateAliasInput{
			AliasName:   aws.String("alias/" + alias),
			TargetKeyId: result.KeyMetadata.KeyId,
		}
		_, err := svc.CreateAlias(input)
		if err != nil {
			log.Debug().Msgf(err.Error())
		}
		return result.KeyMetadata.KeyId, nil
	}
}

func getDynamodbTags(stackId string) []*dynamodb.Tag {
	var dynamodbTags []*dynamodb.Tag
	for _, tag := range getCommonTags(Stack{StackId: stackId}) {
		dynamodbTags = append(dynamodbTags, &dynamodb.Tag{
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	return dynamodbTags
}

func createAndUpdateDB(stackName, stackId, username, password string) error {
	kmsKey, err := createKMSKey(stackId, stackName)
	if err != nil {
		log.Debug().Msg("Failed creating KMS key, DB was not created")
		return err
	}

	svc := connectors.GetAWSSession().DynamoDB

	tableName := generateResourceName(stackId, stackName, "")

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
		Tags:        getDynamodbTags(stackId),
		SSESpecification: &dynamodb.SSESpecification{
			Enabled:        aws.Bool(true),
			KMSMasterKeyId: kmsKey,
			SSEType:        aws.String("KMS"),
		},
	}

	_, err = svc.CreateTable(input)
	if err != nil {
		log.Debug().Msg("Failed creating table")
		return err
	}

	logging.UserProgress("Waiting for table \"%s\" to be created...", tableName)
	err = svc.WaitUntilTableExists(&dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})

	if err != nil {
		return err
	}

	logging.UserProgress("Table %s was created successfully!", tableName)
	item := JoinParamsDb{
		Key:      "cluster-creds",
		Username: username,
		Password: password,
	}
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		log.Debug().Msg("Got error marshalling user name and password")
		return err
	}
	_, err = svc.PutItem(&dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tableName),
	})
	if err != nil {
		log.Debug().Msg("Got error inserting username and password to DB")
		return err
	}
	log.Debug().Msgf("Username:%s and Password:%s were added to DB successfully!", username, strings.Repeat("*", len(password)))
	return nil
}

func getIAMTags(hostGroup HostGroup) []*iam.Tag {
	var iamTags []*iam.Tag
	for _, tag := range getHostGroupTags(hostGroup) {
		iamTags = append(iamTags, &iam.Tag{
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	return iamTags
}

func createIamPolicy(hostGroup HostGroup, lambdaType string) (*iam.Policy, error) {
	svc := connectors.GetAWSSession().IAM
	policy := PolicyDocument{
		Version: "2012-10-17",
		Statement: []StatementEntry{
			{
				Effect: "Allow",
				Action: []string{
					"logs:CreateLogStream",
					"logs:PutLogEvents",
					"logs:CreateLogGroup",
					"dynamodb:GetItem",
					"autoscaling:Describe*",
					"ec2:Describe*",
					"kms:Decrypt",
				},
				Resource: "*",
			},
		},
	}
	b, err := json.Marshal(&policy)
	if err != nil {
		log.Debug().Msg("Error marshaling policy")
		return nil, err
	}
	policyName := fmt.Sprintf("wekactl-%s-%s-%s", hostGroup.Name, lambdaType, getUuidFromStackId(hostGroup.Stack.StackId))
	result, err := svc.CreatePolicy(&iam.CreatePolicyInput{
		PolicyDocument: aws.String(string(b)),
		PolicyName:     aws.String(policyName),
	})

	if err != nil {
		fmt.Println("Error", err)
		return nil, err
	}
	log.Debug().Msgf("policy %s was create successfully!", policyName)
	return result.Policy, nil
}

func createIamRole(hostGroup HostGroup, lambdaType string) (*string, error) {
	svc := connectors.GetAWSSession().IAM
	doc := "{\"Version\": \"2012-10-17\", \"Statement\": [{\"Effect\": \"Allow\", \"Principal\": {\"Service\": \"lambda.amazonaws.com\"}, \"Action\": \"sts:AssumeRole\"}]}"
	//creating and deleting the same role name and use it for lambda caused problems, so we use unique uuid
	roleName := fmt.Sprintf("wekactl-%s-%s-%s", hostGroup.Name, lambdaType, uuid.New().String())
	input := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(doc),
		Path:                     aws.String("/"),
		//max roleName length must be 64 characters
		RoleName: aws.String(roleName),
		Tags:     getIAMTags(hostGroup),
	}

	result, err := svc.CreateRole(input)
	if err != nil {
		return nil, err
	}

	err = svc.WaitUntilRoleExists(&iam.GetRoleInput{RoleName: aws.String(roleName)})
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("role %s was created successfully!", roleName)
	logging.UserProgress("Waiting for lambda role trust entity to finish update")
	time.Sleep(10 * time.Second) // it takes some time for the trust entity to be updated

	policy, err := createIamPolicy(hostGroup, lambdaType)
	if err != nil {
		return nil, err
	}

	_, err = svc.AttachRolePolicy(&iam.AttachRolePolicyInput{PolicyArn: policy.Arn, RoleName: &roleName})
	if err != nil {
		return nil, err
	}
	return result.Role.Arn, nil
}

func getMapCommonTags(hostGroup HostGroup) map[string]*string {
	return map[string]*string{
		"wekactl.io/managed":        aws.String("true"),
		"wekactl.io/api_version":    aws.String("v1"),
		"wekactl.io/stack_id":       aws.String(hostGroup.Stack.StackId),
		"wekactl.io/name":           aws.String(hostGroup.Name),
		"wekactl.io/hostgroup_type": aws.String(hostGroup.Role),
	}
}

func createLambda(hostGroup HostGroup, lambdaType, name string) (*lambda.FunctionConfiguration, error) {
	svc := connectors.GetAWSSession().Lambda

	bucket, err := dist.GetLambdaBucket()
	if err != nil {
		return nil, err
	}
	s3Key := fmt.Sprintf("%s/%s", dist.LambdasID, string(dist.WekaCtl))

	roleArn, err := createIamRole(hostGroup, lambdaType)
	if err != nil {
		return nil, err
	}

	asgName := generateResourceName(hostGroup.Stack.StackId, hostGroup.Stack.StackName, name)
	tableName := generateResourceName(hostGroup.Stack.StackId, hostGroup.Stack.StackName, "")
	lambdaName := fmt.Sprintf("wekactl-%s-%s", hostGroup.Name, lambdaType)

	input := &lambda.CreateFunctionInput{
		Code: &lambda.FunctionCode{
			S3Bucket: aws.String(bucket),
			S3Key:    aws.String(s3Key),
		},
		Description: aws.String(fmt.Sprintf("Wekactl %s", lambdaType)),
		Environment: &lambda.Environment{
			Variables: map[string]*string{
				"LAMBDA":     aws.String(lambdaType),
				"REGION":     aws.String(env.Config.Region),
				"ASG_NAME":   aws.String(asgName),
				"TABLE_NAME": aws.String(tableName),
			},
		},
		Handler:      aws.String("lambdas-bin"),
		FunctionName: aws.String(lambdaName),
		MemorySize:   aws.Int64(256),
		Publish:      aws.Bool(true),
		Role:         roleArn,
		Runtime:      aws.String("go1.x"),
		Tags:         getMapCommonTags(hostGroup),
		Timeout:      aws.Int64(15),
		TracingConfig: &lambda.TracingConfig{
			Mode: aws.String("Active"),
		},
	}

	lambdaCreateOutput, err := svc.CreateFunction(input)
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("lambda %s was created successfully!", lambdaName)

	return lambdaCreateOutput, nil
}

func createRestApiGateway(hostGroup HostGroup, lambdaType, lambdaUri string) (string, error) {
	svc := connectors.GetAWSSession().ApiGateway
	apiGatewayName := fmt.Sprintf("wekactl-%s-%s", hostGroup.Name, lambdaType)

	createApiOutput, err := svc.CreateRestApi(&apigateway.CreateRestApiInput{
		Name:         aws.String(apiGatewayName),
		Tags:         getMapCommonTags(hostGroup),
		Description:  aws.String("Wekactl " + lambdaType + " lambda"),
		ApiKeySource: aws.String("HEADER"),
	})
	if err != nil {
		return "", err
	}
	restApiId := createApiOutput.Id
	log.Debug().Msgf("rest api gateway id:%s for lambda:%s was created successfully!", *restApiId, apiGatewayName)

	resources, err := svc.GetResources(&apigateway.GetResourcesInput{
		RestApiId: restApiId,
	})
	if err != nil {
		return "", err
	}

	rootResource := resources.Items[0]
	createResourceOutput, err := svc.CreateResource(&apigateway.CreateResourceInput{
		ParentId:  rootResource.Id,
		RestApiId: restApiId,
		PathPart:  aws.String(apiGatewayName),
	})
	if err != nil {
		return "", err
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
		return "", err
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
		return "", err
	}
	log.Debug().Msgf("rest api %s method integration created successfully!", httpMethod)

	stageName := "default"
	_, err = svc.CreateDeployment(&apigateway.CreateDeploymentInput{
		RestApiId: restApiId,
		StageName: aws.String(stageName),
	})
	log.Debug().Msgf("rest api gateway deployment for stage %s was created successfully!", stageName)

	usagePlanOutput, err := svc.CreateUsagePlan(&apigateway.CreateUsagePlanInput{
		Name: aws.String(apiGatewayName + "-usage-plan"),
		ApiStages: []*apigateway.ApiStage{
			{
				ApiId: restApiId,
				Stage: aws.String("default"),
			},
		},
	})
	if err != nil {
		return "", err
	}
	log.Debug().Msgf("usage plan %s was created successfully!", *usagePlanOutput.Name)

	apiKeyOutput, err := svc.CreateApiKey(&apigateway.CreateApiKeyInput{
		Enabled: aws.Bool(true),
		Name:    aws.String(apiGatewayName + "-api-key"),
		Tags:    getMapCommonTags(hostGroup),
	})
	if err != nil {
		return "", err
	}
	log.Debug().Msgf("api key %s was created successfully!", *apiKeyOutput.Name)

	_, err = svc.CreateUsagePlanKey(&apigateway.CreateUsagePlanKeyInput{
		UsagePlanId: usagePlanOutput.Id,
		KeyId:       apiKeyOutput.Id,
		KeyType:     aws.String("API_KEY"),
	})
	if err != nil {
		return "", err
	}
	log.Debug().Msg("api key was associated to usage plan successfully!")

	return *restApiId, nil
}

func getAccountId() (string, error) {
	svc := connectors.GetAWSSession().STS
	result, err := svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return *result.Account, nil
}

func addLambdaInvokePermissions(lambdaName, restApiId string) error {
	svc := connectors.GetAWSSession().Lambda
	account, err := getAccountId()
	if err != nil {
		return err
	}
	sourceArn := fmt.Sprintf("arn:aws:execute-api:eu-central-1:%s:%s/*/GET/%s", account, restApiId, lambdaName)
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

func CreateLambdaEndPoint(hostGroup HostGroup, lambdaType, name string) error {
	functionConfiguration, err := createLambda(hostGroup, lambdaType, name)
	if err != nil {
		return err
	}

	lambdaUri := fmt.Sprintf(
		"arn:aws:apigateway:%s:lambda:path/2015-03-31/functions/%s/invocations",
		env.Config.Region, *functionConfiguration.FunctionArn)

	restApiId, err := createRestApiGateway(hostGroup, lambdaType, lambdaUri)
	if err != nil {
		return err
	}

	err = addLambdaInvokePermissions(*functionConfiguration.FunctionName, restApiId)
	if err != nil {
		return err
	}

	return nil
}

func importClusterRole(stackId, stackName, role string, roleInstances []*ec2.Instance) error {
	if len(roleInstances) == 0 {
		logging.UserProgress("instances with role '%s' not found", role)
		return nil
	}

	var name string
	switch role {
	case "backend":
		name = "Backends"
	case "client":
		name = "Clients"
	default:
		return errors.New(fmt.Sprintf("import of role %s is unsupported", role))
	}

	launchTemplateName := createLaunchTemplate(stackId, stackName, name, role, roleInstances[0])
	autoScalingGroupName, err := createAutoScalingGroup(stackId, stackName, name, role, len(roleInstances), launchTemplateName)
	if err != nil {
		return err
	}
	return attachInstancesToAutoScalingGroups(roleInstances, autoScalingGroupName)
}

func ImportCluster(stackName, username, password string) error {
	stackId, err := GetStackId(stackName)
	if err != nil {
		return err
	}
	err = createAndUpdateDB(stackName, stackId, username, password)
	if err != nil {
		return err
	}
	stackInstances, err := getInstancesInfo(stackName)
	if err != nil {
		return err
	}

	err = disableInstancesApiTermination(stackInstances.All())
	if err != nil {
		return err
	}

	err = importClusterRole(stackId, stackName, "client", stackInstances.clients)
	if err != nil {
		return err
	}
	err = importClusterRole(stackId, stackName, "backend", stackInstances.backends)
	if err != nil {
		return err
	}
	return nil
}
