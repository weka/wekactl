package cluster

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/rs/zerolog/log"
	"strings"
	"sync"
	"wekactl/internal/aws/common"
	"wekactl/internal/logging"
)

type StackInstances struct {
	backends []*ec2.Instance
	clients  []*ec2.Instance
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

func getStackId(region, stackName string) string {
	sess := common.NewSession(region)
	svc := cloudformation.New(sess)
	input := &cloudformation.DescribeStacksInput{StackName: &stackName}
	result, err := svc.DescribeStacks(input)
	if err != nil {
		log.Fatal().Err(err)
	}
	return *result.Stacks[0].StackId
}

func getClusterInstances(region, stackName string) []*string {
	sess := common.NewSession(region)
	svc := cloudformation.New(sess)
	input := &cloudformation.DescribeStackResourcesInput{StackName: &stackName}
	result, err := svc.DescribeStackResources(input)
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
	return instancesIds
}

func getInstancesInfo(region, stackName string) StackInstances {
	sess := common.NewSession(region)
	svc := ec2.New(sess)
	input := &ec2.DescribeInstancesInput{
		InstanceIds: getClusterInstances(region, stackName),
	}
	result, err := svc.DescribeInstances(input)
	if err != nil {
		log.Fatal().Err(err)
	}

	stackInstances := StackInstances{}

	for _, reservation := range result.Reservations {
		instance := reservation.Instances[0]
		arn := *instance.IamInstanceProfile.Arn
		if strings.Contains(arn, "InstanceProfileBackend") {
			stackInstances.backends = append(stackInstances.backends, instance)
		} else if strings.Contains(arn, "InstanceProfileClient") {
			stackInstances.clients = append(stackInstances.clients, instance)
		}

	}
	return stackInstances
}

func getInstancesIdsFromEc2Instance(instances []*ec2.Instance) []*string {
	var instanceIds []*string
	for _, instance := range instances {
		instanceIds = append(instanceIds, instance.InstanceId)
	}
	return instanceIds
}

func disableInstanceApiTermination(instanceId string, svc *ec2.EC2) (*ec2.ModifyInstanceAttributeOutput, error) {
	input := &ec2.ModifyInstanceAttributeInput{
		DisableApiTermination: &ec2.AttributeBooleanValue{
			Value: aws.Bool(true),
		},
		InstanceId: aws.String(instanceId),
	}
	return svc.ModifyInstanceAttribute(input)
}

func disableInstancesApiTermination(region string, instances []*ec2.Instance) {
	sess := common.NewSession(region)
	svc := ec2.New(sess)

	instanceIds := getInstancesIdsFromEc2Instance(instances)
	parallelization := len(instanceIds)
	c := make(chan string)

	var wg sync.WaitGroup
	wg.Add(parallelization)
	for ii := 0; ii < parallelization; ii++ {
		go func(c chan string) {
			for {
				v, more := <-c
				if more == false {
					wg.Done()
					return
				}
				_, err := disableInstanceApiTermination(v, svc)
				if err != nil {
					log.Debug().Msgf("Failed to se DisableApiTermination on %s", v)
				}
			}
		}(c)
	}
	for _, instanceId := range instanceIds {
		c <- *instanceId
	}
	close(c)
	wg.Wait()
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

func getEc2Tags(role, stackId string) []*ec2.Tag {
	var ec2Tags []*ec2.Tag
	for _, tag := range getHostGroupTags(HostGroup{
		name:  role,
		role:  strings.TrimSuffix(role, "s"),
		stack: Stack{stackId: stackId},
	}) {
		ec2Tags = append(ec2Tags, &ec2.Tag{
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	return ec2Tags
}

func generateResourceName(stackId, stackName, role string) string {
	name := "weka-" + stackName + "-"
	if role != "" {
		name += role + "-"
	}
	return name + getUuidFromStackId(stackId)
}

func createLaunchTemplate(region, stackId, stackName, role string, instance *ec2.Instance) string {
	sess := common.NewSession(region)
	svc := ec2.New(sess)
	launchTemplateName := generateResourceName(stackId, stackName, role)
	input := &ec2.CreateLaunchTemplateInput{
		LaunchTemplateData: &ec2.RequestLaunchTemplateData{
			ImageId:      instance.ImageId,
			InstanceType: instance.InstanceType,
			KeyName:      instance.KeyName,
			UserData:     aws.String(""), // TODO: add necessary init script here
			TagSpecifications: []*ec2.LaunchTemplateTagSpecificationRequest{
				{
					ResourceType: aws.String("instance"),
					Tags:         getEc2Tags(role, stackId),
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
	log.Debug().Msgf("LaunchTemplate: \"%s\" was created sucessfully!", launchTemplateName)
	return launchTemplateName
}

func getAutoScalingTags(role, stackId string) []*autoscaling.Tag {
	var autoscalingTags []*autoscaling.Tag
	for _, tag := range getHostGroupTags(HostGroup{
		name:  role,
		role:  strings.TrimSuffix(role, "s"),
		stack: Stack{stackId: stackId},
	}) {
		autoscalingTags = append(autoscalingTags, &autoscaling.Tag{
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	return autoscalingTags
}

func createAutoScalingGroup(region, stackId, stackName, role string, roleInstances []*ec2.Instance) (string, error) {
	if len(roleInstances) > 0 {
		launchTemplateName := createLaunchTemplate(region, stackId, stackName, role, roleInstances[0])
		instancesNumber := int64(len(roleInstances))
		sess := common.NewSession(region)
		svc := autoscaling.New(sess)
		name := generateResourceName(stackId, stackName, role)
		input := &autoscaling.CreateAutoScalingGroupInput{
			AutoScalingGroupName:             aws.String(name),
			NewInstancesProtectedFromScaleIn: aws.Bool(true),
			LaunchTemplate: &autoscaling.LaunchTemplateSpecification{
				LaunchTemplateName: aws.String(launchTemplateName),
				Version:            aws.String("1"),
			},
			MinSize: aws.Int64(0),
			MaxSize: aws.Int64(instancesNumber),
			Tags:    getAutoScalingTags(role, stackId),
		}
		_, err := svc.CreateAutoScalingGroup(input)
		if err != nil {
			return "", err
		}
		log.Debug().Msgf("AutoScalingGroup: \"%s\" was created sucessfully!", name)
		return name, nil
	} else {
		logging.UserProgress("No %s where found", strings.Title(role))
		return "", nil
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func attachInstancesToAutoScalingGroups(region string, roleInstances []*ec2.Instance, autoScalingGroupsName string) {
	sess := common.NewSession(region)
	svc := autoscaling.New(sess)
	limit := 20
	instancesIds := getInstancesIdsFromEc2Instance(roleInstances)
	for i := 0; i < len(instancesIds); i += limit {
		batch := instancesIds[i:min(i+limit, len(instancesIds))]
		input := &autoscaling.AttachInstancesInput{
			AutoScalingGroupName: &autoScalingGroupsName,
			InstanceIds:          batch,
		}
		_, err := svc.AttachInstances(input)
		if err != nil {
			log.Debug().Msgf(err.Error())
		} else {
			log.Debug().Msgf("Attached %d instances to %s successfully!", len(batch), autoScalingGroupsName)
		}
	}
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

func createKMSKey(region, stackId, stackName string) (*string, error) {
	sess := common.NewSession(region)
	svc := kms.New(sess)
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

func createAndUpdateDB(region, stackName, stackId, username, password string) error {
	kmsKey, err := createKMSKey(region, stackId, stackName)
	if err != nil {
		log.Debug().Msg("Failed creating KMS key, DB was not created")
		return err
	}

	sess := common.NewSession(region)
	svc := dynamodb.New(sess)

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
	} else {
		logging.UserProgress("Waiting for table \"%s\" to be created...", tableName)
		input := &dynamodb.DescribeTableInput{TableName: aws.String(tableName)}
		err := svc.WaitUntilTableExists(input)
		if err != nil {
			return err
		} else {
			log.Debug().Msgf("Table %s was created successfully!", tableName)
			item := JoinParamsDb{
				Key:      "join-params",
				Username: username,
				Password: password,
			}
			av, err := dynamodbattribute.MarshalMap(item)
			if err != nil {
				log.Debug().Msg("Got error marshalling user name and password")
				return err
			} else {
				input := &dynamodb.PutItemInput{
					Item:      av,
					TableName: aws.String(tableName),
				}
				_, err = svc.PutItem(input)
				if err != nil {
					log.Debug().Msg("Got error inserting username and password to DB")
					return err
				} else {
					log.Debug().Msgf("Username:%s and Password:%s were added to DB successfully!", username, strings.Repeat("*", len(password)))
					return nil
				}
			}
		}
	}
}

func importClusterRole(region, stackId, stackName, role string, roleInstances []*ec2.Instance) error{
	autoScalingGroupName, err := createAutoScalingGroup(region, stackId, stackName, role, roleInstances)
	if err != nil {
		return err
	}
	attachInstancesToAutoScalingGroups(region, roleInstances, autoScalingGroupName)
	return nil
}

func ImportCluster(region, stackName, username, password string) error {
	stackId := getStackId(region, stackName)
	err := createAndUpdateDB(region, stackName, stackId, username, password)
	if err != nil {
		return err
	}
	stackInstances := getInstancesInfo(region, stackName)
	disableInstancesApiTermination(region, append(stackInstances.clients, stackInstances.backends...))
	err = importClusterRole(region, stackId, stackName, "clients", stackInstances.clients)
	if err != nil{
		return err
	}
	err = importClusterRole(region, stackId, stackName, "backends", stackInstances.backends)
	if err != nil{
		return err
	}
	return nil
}
