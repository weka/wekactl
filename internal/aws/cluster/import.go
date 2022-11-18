package cluster

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	errors2 "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"strings"
	"wekactl/internal/aws/autoscaling"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
	"wekactl/internal/env"
	strings2 "wekactl/internal/lib/strings"
)

type ClusterInstances struct {
	Backends []*ec2.Instance
	Clients  []*ec2.Instance
}

func (s *ClusterInstances) All() []*ec2.Instance {
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

func GetStackInstancesInfo(stackName string) (clusterInstances ClusterInstances, err error) {
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
				clusterInstances.Backends = append(clusterInstances.Backends, instance)
			} else if strings.Contains(arn, "InstanceProfileClient") {
				// ASG for clients is deprecated until need
				clusterInstances.Clients = append(clusterInstances.Clients, instance)
			}
		}
	}
	return clusterInstances, nil
}

func GetInstanceSecurityGroupsId(instance *ec2.Instance) []*string {
	var securityGroupIds []*string
	for _, securityGroup := range instance.SecurityGroups {
		securityGroupIds = append(securityGroupIds, securityGroup.GroupId)
	}
	return securityGroupIds
}

func GetVolumesInfo(instance *ec2.Instance, role common.InstanceRole) (volumesInfo []common.VolumeInfo, err error) {
	log.Debug().Msgf("Retrieving %s instance volume info ...", string(role))
	var volumeIds []*string
	volumeIdToDeviceName := make(map[string]string)
	for _, blockDeviceMapping := range instance.BlockDeviceMappings {
		volumeId := blockDeviceMapping.Ebs.VolumeId
		volumeIds = append(volumeIds, volumeId)
		volumeIdToDeviceName[*volumeId] = *blockDeviceMapping.DeviceName
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

	for _, volume := range volumesOutput.Volumes {
		volumeName := volumeIdToDeviceName[*volume.VolumeId]
		size := *volume.Size
		if volumeName == *instance.RootDeviceName && size < common.RootFsMinimalSize {
			size = common.RootFsMinimalSize
		}
		volumesInfo = append(volumesInfo, common.VolumeInfo{
			Name: volumeName,
			Type: *volume.VolumeType,
			Size: size,
		})
	}
	return
}

func generateAWSCluster(stackName, tableName string, defaultParams db.ClusterSettings, clientsExist bool) AWSCluster {
	clusterName := cluster.ClusterName(stackName)

	hostGroups := []HostGroup{
		GenerateHostGroup(
			clusterName,
			defaultParams.Backends,
			common.RoleBackend,
			"Backends",
		),
	}

	// ASG for clients is deprecated

	//if clientsExist {
	//	hostGroups = append(hostGroups, GenerateHostGroup(
	//		clusterName,
	//		defaultParams.Clients,
	//		common.RoleClient,
	//		"Clients",
	//	))
	//}

	return AWSCluster{
		Name:            clusterName,
		ClusterSettings: defaultParams,
		HostGroups:      hostGroups,
		TableName:       tableName,
	}
}

func instanceIdsToClusterInstances(instanceIds []string) (clusterInstances ClusterInstances, err error) {
	log.Debug().Msgf("Retrieving instances info from instance ids...")
	svc := connectors.GetAWSSession().EC2
	result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: strings2.ListToRefList(instanceIds),
	})
	if err != nil {
		return
	}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			clusterInstances.Backends = append(clusterInstances.Backends, instance)
		}
	}
	return
}

func ImportCluster(params cluster.ImportParams) (err error) {
	var stackId string
	var clusterSettings db.ClusterSettings
	var clusterInstances ClusterInstances
	stackImport := len(params.InstanceIds) == 0

	if stackImport {
		stackId, err = GetStackId(params.Name)
		if err != nil {
			return err
		}

		clusterInstances, err = GetStackInstancesInfo(params.Name)
		if err != nil {
			return err
		}
	} else {
		clusterInstances, err = instanceIdsToClusterInstances(params.InstanceIds)
		if err != nil {
			return err
		}

	}

	clusterSettings, err = importClusterParamsFromClusterInstances(clusterInstances)
	if err != nil {
		return err
	}

	vpcId, err := common.VpcBySubnet(clusterSettings.Subnet)
	if err != nil {
		return err
	}
	clusterSettings.VpcId = vpcId

	clusterSettings.AdditionalSubnet = params.AdditionalAlbSubnet
	if clusterSettings.AdditionalSubnet == "" {
		additionalSubnet, err := common.GetAdditionalVpcSubnet(vpcId, clusterSettings.Subnet)
		if err != nil {
			if err == common.NoAdditionalSubnet {
				return errors2.Wrap(err, "supply additional ALB subnet via --additional-alb-subnet")
			}
			return err
		}
		clusterSettings.AdditionalSubnet = additionalSubnet
	}

	clusterSettings.PrivateSubnet = params.PrivateSubnet
	clusterSettings.TagsMap = params.TagsMap()
	versionInfo, err := env.GetBuildVersion()
	if err != nil {
		return err
	}
	clusterSettings.BuildVersion = versionInfo.BuildVersion
	clusterSettings.DnsAlias = params.DnsAlias
	clusterSettings.DnsZoneId = params.DnsZoneId

	dynamoDb := DynamoDb{
		ClusterName: cluster.ClusterName(params.Name),
		StackId:     stackId,
	}
	dynamoDb.Init()
	err = cluster.EnsureResource(&dynamoDb, clusterSettings, false)
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

	instanceIds := common.GetInstancesIds(clusterInstances.All())
	_, errs := common.SetDisableInstancesApiTermination(instanceIds, true)
	if len(errs) != 0 {
		return errs[0]
	}

	clientsExist := len(clusterInstances.Clients) > 0
	awsCluster := generateAWSCluster(params.Name, dynamoDb.ResourceName(), clusterSettings, clientsExist)
	awsCluster.Init()
	err = cluster.EnsureResource(&awsCluster, clusterSettings, false)
	if err != nil {
		return err
	}

	roleInstanceIdsRefs := make(map[common.InstanceRole][]*string)
	roleInstanceIdsRefs[common.RoleBackend] = common.GetInstancesIdsRefs(clusterInstances.Backends)
	roleInstanceIdsRefs[common.RoleClient] = common.GetInstancesIdsRefs(clusterInstances.Clients)
	for _, hostgroup := range awsCluster.HostGroups {
		autoscalingGroupName := hostgroup.AutoscalingGroup.ResourceName()
		err = autoscaling.AttachInstancesToASG(roleInstanceIdsRefs[hostgroup.HostGroupInfo.Role], autoscalingGroupName)
		if err != nil {
			return err
		}
		if stackImport && hostgroup.HostGroupInfo.Role == common.RoleBackend {
			err = autoscaling.AttachLoadBalancer(hostgroup.HostGroupInfo.ClusterName, autoscalingGroupName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func importClusterParamsFromClusterInstances(instances ClusterInstances) (defaultParams db.ClusterSettings, err error) {
	minBackendsNumber := 5
	if len(instances.Backends) < minBackendsNumber {
		return defaultParams, errors.New(fmt.Sprintf(
			"%d backend instances found, minimum is: %d, can't proceed with import",
			len(instances.Backends),
			minBackendsNumber))
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

	volumeInfo, err := GetVolumesInfo(instance, role)
	if err != nil {
		return err
	}
	hostGroupParams.SecurityGroupsIds = GetInstanceSecurityGroupsId(instance)
	hostGroupParams.ImageID = *instance.ImageId
	if instance.KeyName != nil {
		hostGroupParams.KeyName = *instance.KeyName
	}
	hostGroupParams.IamArn = *instance.IamInstanceProfile.Arn
	hostGroupParams.InstanceType = *instance.InstanceType
	hostGroupParams.Subnet = *instance.SubnetId
	hostGroupParams.VolumesInfo = volumeInfo
	hostGroupParams.MaxSize = common.GetMaxSize(role, len(instances))
	return nil
}
