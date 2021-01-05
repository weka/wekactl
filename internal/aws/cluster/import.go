package cluster

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/semaphore"
	"strings"
	"sync"
	"sync/atomic"
	"wekactl/internal/connectors"
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
	stackId       string
	stackName     string
	dynamodbTable string
}

type HostGroup struct {
	role  string
	name  string
	stack Stack
}

func getStackId(stackName string) (string, error) {
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
			Value: aws.String(stack.stackId),
		},
	}
	return tags
}

func getHostGroupTags(hostGroup HostGroup) []Tag {
	tags := getCommonTags(hostGroup.stack)
	tags = append(
		tags, Tag{
			Key:   aws.String("wekactl.io/name"),
			Value: aws.String(hostGroup.name),
		}, Tag{
			Key:   aws.String("wekactl.io/hostgroup_type"),
			Value: aws.String(hostGroup.role),
		},
	)
	return tags
}

func getEc2Tags(name, role, stackId string) []*ec2.Tag {
	var ec2Tags []*ec2.Tag
	for _, tag := range getHostGroupTags(HostGroup{
		name:  name,
		role:  role,
		stack: Stack{stackId: stackId},
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
		name:  name,
		role:  role,
		stack: Stack{stackId: stackId},
	}) {
		autoscalingTags = append(autoscalingTags, &autoscaling.Tag{
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	return autoscalingTags
}

func createAutoScalingGroup(stackId, stackName, name string, role string, maxSize int, launchTemplateName string) (string, error) {
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
	_, err := svc.CreateAutoScalingGroup(input)
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
	for _, tag := range getCommonTags(Stack{stackId: stackId}) {
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
	for _, tag := range getCommonTags(Stack{stackId: stackId}) {
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
	stackId, err := getStackId(stackName)
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
