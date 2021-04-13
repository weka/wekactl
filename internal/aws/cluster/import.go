package cluster

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/rs/zerolog/log"
	"strings"
	"wekactl/internal/aws/autoscaling"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/aws/launchtemplate"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
)

type StackInstances struct {
	Backends []*ec2.Instance
	Clients  []*ec2.Instance
}

func (s *StackInstances) All() []*ec2.Instance {
	return append(s.Clients[0:len(s.Clients):len(s.Clients)], s.Backends...)
}

type Tag struct {
	Key   string
	Value string
}

func GetStackId(stackName string) (string, error) {
	log.Debug().Msgf("Retrieving %s stack id ...", stackName)
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

func getStackInstances(stackName string) ([]*string, error) {
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

func GetStackInstancesInfo(stackName string) (stackInstances StackInstances, err error) {
	log.Debug().Msgf("Retrieving %s instances info ...", stackName)
	svc := connectors.GetAWSSession().EC2
	instances, err := getStackInstances(stackName)
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
		for _, instance := range reservation.Instances {
			if *instance.State.Name == "terminated" {
				continue
			}
			arn := *instance.IamInstanceProfile.Arn
			if strings.Contains(arn, "InstanceProfileBackend") {
				stackInstances.Backends = append(stackInstances.Backends, instance)
			} else if strings.Contains(arn, "InstanceProfileClient") {
				stackInstances.Clients = append(stackInstances.Clients, instance)
			}
		}
	}
	return stackInstances, nil
}

func GetInstanceSecurityGroupsId(instance *ec2.Instance) []*string {
	var securityGroupIds []*string
	for _, securityGroup := range instance.SecurityGroups {
		securityGroupIds = append(securityGroupIds, securityGroup.GroupId)
	}
	return securityGroupIds
}

func getVolumeInfo(instance *ec2.Instance, role common.InstanceRole) (volumeInfo launchtemplate.VolumeInfo, err error) {
	log.Debug().Msgf("Retrieving %s instance volume info ...", string(role))
	var volumeIds []*string
	for _, blockDeviceMapping := range instance.BlockDeviceMappings {
		volumeIds = append(volumeIds, blockDeviceMapping.Ebs.VolumeId)
	}

	svc := connectors.GetAWSSession().EC2
	volumesOutput, err := svc.DescribeVolumes(&ec2.DescribeVolumesInput{
		VolumeIds: volumeIds,
	})
	if err != nil {
		return
	}
	if len(volumesOutput.Volumes) == 0 {
		err = errors.New(fmt.Sprintf("Instance has %s no volumes", *instance.InstanceId))
		return
	}

	totalSize := int64(0)
	for _, volume := range volumesOutput.Volumes {
		totalSize += *volume.Size
	}

	volumeInfo = launchtemplate.VolumeInfo{
		Name: *instance.RootDeviceName,
		Type: *volumesOutput.Volumes[0].VolumeType,
		Size: totalSize,
	}
	return
}

func generateAWSCluster(stackId, stackName, tableName string, defaultParams db.ClusterSettings) AWSCluster {
	clusterName := cluster.ClusterName(stackName)

	backendsHostGroup := GenerateHostGroup(
		clusterName,
		defaultParams.Backends,
		common.RoleBackend,
		"Backends",
	)

	clientsHostGroup := GenerateHostGroup(
		clusterName,
		defaultParams.Clients,
		common.RoleClient,
		"Clients",
	)

	return AWSCluster{
		Name:            clusterName,
		ClusterSettings: defaultParams,
		HostGroups: []HostGroup{
			backendsHostGroup,
			clientsHostGroup,
		},
		TableName: tableName,
	}
}

func ImportCluster(params cluster.ImportParams) error {
	stackId, err := GetStackId(params.Name)
	if err != nil {
		return err
	}

	stackInstances, err := GetStackInstancesInfo(params.Name)
	if err != nil {
		return err
	}

	clusterSettings, err := importClusterParamsFromCF(stackInstances)
	if err != nil {
		return err
	}
	clusterSettings.PrivateSubnet = params.PrivateSubnet
	clusterSettings.TagsMap = params.TagsMap()

	dynamoDb := DynamoDb{
		ClusterName: cluster.ClusterName(params.Name),
		StackId:     stackId,
	}
	dynamoDb.Init()
	err = cluster.EnsureResource(&dynamoDb, clusterSettings)
	if err != nil {
		return err
	}

	err = db.SaveCredentials(dynamoDb.ResourceName(), params.Username, params.Password)
	if err != nil {
		return err
	}
	if err = db.SaveClusterSettings(dynamoDb.ResourceName(), clusterSettings); err != nil {
		return err
	}

	instanceIds := common.GetInstancesIds(stackInstances.All())
	_, errs := common.SetDisableInstancesApiTermination(instanceIds, true)
	if len(errs) != 0 {
		return errs[0]
	}

	awsCluster := generateAWSCluster(stackId, params.Name, dynamoDb.ResourceName(), clusterSettings)
	awsCluster.Init()
	err = cluster.EnsureResource(&awsCluster, clusterSettings)
	if err != nil {
		return err
	}

	roleInstanceIdsRefs := make(map[common.InstanceRole][]*string)
	roleInstanceIdsRefs[common.RoleBackend] = common.GetInstancesIdsRefs(stackInstances.Backends)
	roleInstanceIdsRefs[common.RoleClient] = common.GetInstancesIdsRefs(stackInstances.Clients)
	for _, hostgroup := range awsCluster.HostGroups {
		autoscalingGroupName := hostgroup.AutoscalingGroup.ResourceName()
		err = autoscaling.AttachInstancesToASG(roleInstanceIdsRefs[hostgroup.HostGroupInfo.Role], autoscalingGroupName)
		if err != nil {
			return err
		}
		if hostgroup.HostGroupInfo.Role == common.RoleBackend {
			err = autoscaling.AttachLoadBalancer(hostgroup.HostGroupInfo.ClusterName, autoscalingGroupName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func importClusterParamsFromCF(instances StackInstances) (defaultParams db.ClusterSettings, err error) {
	if len(instances.Backends) == 0 {
		return defaultParams, errors.New("backend instances not found, can't proceed with import")
	}

	err = importRoleParams(&defaultParams.Backends, instances.Backends, common.RoleBackend)
	if err != nil {
		return
	}

	defaultParams.Subnet = defaultParams.Backends.Subnet

	if len(instances.Clients) == 0 {
		defaultParams.Clients = defaultParams.Backends
		return
	}
	err = importRoleParams(&defaultParams.Clients, instances.Clients, common.RoleClient)
	if err != nil {
		return
	}

	return
}

func importRoleParams(hostGroupParams *common.HostGroupParams, instances []*ec2.Instance, role common.InstanceRole) error {
	instance := instances[0]

	volumeInfo, err := getVolumeInfo(instance, role)
	if err != nil {
		return err
	}
	hostGroupParams.SecurityGroupsIds = GetInstanceSecurityGroupsId(instance)
	hostGroupParams.ImageID = *instance.ImageId
	hostGroupParams.KeyName = *instance.KeyName
	hostGroupParams.IamArn = *instance.IamInstanceProfile.Arn
	hostGroupParams.InstanceType = *instance.InstanceType
	hostGroupParams.Subnet = *instance.SubnetId
	hostGroupParams.VolumeName = volumeInfo.Name
	hostGroupParams.VolumeType = volumeInfo.Type
	hostGroupParams.VolumeSize = volumeInfo.Size
	hostGroupParams.MaxSize = common.GetMaxSize(role, len(instances))
	return nil
}
