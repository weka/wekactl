package common

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/semaphore"
	"math"
	"os"
	"sync"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
	strings2 "wekactl/internal/lib/strings"
	"wekactl/internal/lib/types"
)

type InstanceIdsSet map[string]types.Nilt

func RenderTable(fields []string, data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(fields)
	table.SetRowLine(true)
	table.AppendBulk(data)
	table.Render()
}

func setDisableInstanceApiTermination(instanceId string, value bool) (*ec2.ModifyInstanceAttributeOutput, error) {
	svc := connectors.GetAWSSession().EC2
	input := &ec2.ModifyInstanceAttributeInput{
		DisableApiTermination: &ec2.AttributeBooleanValue{
			Value: aws.Bool(value),
		},
		InstanceId: aws.String(instanceId),
	}
	return svc.ModifyInstanceAttribute(input)
}

var terminationSemaphore *semaphore.Weighted

func init() {
	terminationSemaphore = semaphore.NewWeighted(20)
}

func SetDisableInstancesApiTermination(instanceIds []string, value bool) (updated []string, errs []error) {
	var wg sync.WaitGroup
	var responseLock sync.Mutex

	log.Debug().Msgf("Setting instances DisableApiTermination to: %t ...", value)
	wg.Add(len(instanceIds))
	for i := range instanceIds {
		go func(i int) {
			_ = terminationSemaphore.Acquire(context.Background(), 1)
			defer terminationSemaphore.Release(1)
			defer wg.Done()

			responseLock.Lock()
			defer responseLock.Unlock()
			_, err := setDisableInstanceApiTermination(instanceIds[i], value)
			if err != nil {
				errs = append(errs, err)
				log.Error().Err(err)
				log.Error().Msgf("failed to set DisableApiTermination on %s", instanceIds[i])
			}
			updated = append(updated, instanceIds[i])
		}(i)
	}
	wg.Wait()
	return
}

func GetASGInstances(asgName string) ([]*autoscaling.Instance, error) {
	svc := connectors.GetAWSSession().ASG
	asgOutput, err := svc.DescribeAutoScalingGroups(
		&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{&asgName},
		},
	)
	if err != nil {
		return []*autoscaling.Instance{}, err
	}
	return asgOutput.AutoScalingGroups[0].Instances, nil
}

func GetAutoScalingGroupInstanceIds(asgName string) ([]*string, error) {
	instances, err := GetASGInstances(asgName)
	if err != nil {
		return []*string{}, err
	}
	return UnpackASGInstanceIds(instances), nil
}

func UnpackASGInstanceIds(instances []*autoscaling.Instance) []*string {
	instanceIds := []*string{}
	if len(instances) == 0 {
		return instanceIds
	}
	for _, instance := range instances {
		instanceIds = append(instanceIds, instance.InstanceId)
	}
	return instanceIds
}

func GetInstanceTypeFromAutoScalingGroupOutput(asgOutput *autoscaling.DescribeAutoScalingGroupsOutput) string {
	if len(asgOutput.AutoScalingGroups) == 0 {
		return ""
	}
	if len(asgOutput.AutoScalingGroups[0].Instances) == 0 {
		return ""
	}
	return *asgOutput.AutoScalingGroups[0].Instances[0].InstanceType
}

func GetInstancesIds(instances []*ec2.Instance) []string {
	var instanceIds []string
	for _, instance := range instances {
		instanceIds = append(instanceIds, *instance.InstanceId)
	}
	return instanceIds
}

func GetInstancesIdsRefs(instances []*ec2.Instance) []*string {
	return strings2.ListToRefList(GetInstancesIds(instances))
}

func getEc2InstancesFromDescribeOutput(describeResponse *ec2.DescribeInstancesOutput) (instances []*ec2.Instance) {
	for _, reservation := range describeResponse.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, instance)
		}
	}
	return
}

func GetInstances(instanceIds []*string) (instances []*ec2.Instance, err error) {
	if len(instanceIds) == 0 {
		err = errors.New("instanceIds list must not be empty")
		return
	}
	svc := connectors.GetAWSSession().EC2
	describeResponse, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	})
	if err != nil {
		return
	}

	instances = getEc2InstancesFromDescribeOutput(describeResponse)
	return
}

func getInstanceIdsSet(instanceIds []*string) InstanceIdsSet {
	instanceIdsSet := make(InstanceIdsSet)
	for _, instanceId := range instanceIds {
		instanceIdsSet[*instanceId] = types.Nilv
	}
	return instanceIdsSet
}

func GetDeltaInstancesIds(instanceIds1 []*string, instanceIds2 []*string) (deltaInstanceIds []*string) {
	instanceIdsSet := getInstanceIdsSet(instanceIds1)

	for _, instanceId := range instanceIds2 {
		if _, ok := instanceIdsSet[*instanceId]; !ok {
			deltaInstanceIds = append(deltaInstanceIds, instanceId)
		}
	}
	return
}

func GetMaxSize(role InstanceRole, initialSize int) int64 {
	var maxSize int
	switch role {
	case "backend":
		maxSize = 7 * initialSize
	case "client":
		maxSize = int(math.Ceil(float64(initialSize)/float64(500))) * 500
	default:
		maxSize = 1000
	}
	return int64(maxSize)
}

func GenerateResourceName(clusterName cluster.ClusterName, hostGroupName HostGroupName) string {
	resourceName := "wekactl-" + string(clusterName)
	name := string(hostGroupName)
	if name != "" {
		resourceName += "-" + name
	}
	return resourceName
}

func GetBackendsPrivateIps(clusterName string) (ips []string, err error) {
	svc := connectors.GetAWSSession().EC2
	log.Debug().Msgf("Fetching backends ips...")
	describeResponse, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
				},
			},
			{
				Name: aws.String("tag:wekactl.io/cluster_name"),
				Values: []*string{
					&clusterName,
				},
			},
			{
				Name: aws.String("tag:wekactl.io/hostgroup_type"),
				Values: []*string{
					aws.String("backend"),
				},
			},
		},
	})

	if err != nil {
		return
	}

	for _, reservation := range describeResponse.Reservations {
		for _, instance := range reservation.Instances {
			if instance.PrivateIpAddress == nil {
				log.Warn().Msgf("Found backend instance %s without private ip!", *instance.InstanceId)
				continue
			}
			ips = append(ips, *instance.PrivateIpAddress)
		}
	}
	log.Debug().Msgf("found %d backends private ips: %s", len(ips), ips)
	return
}

var NoAdditionalSubnet = errors.New("no subnet with same route table in different availability zone was found")

func filterOutSameAvailabilityZoneAdditionalSubnets(subnetId string, subnets []*ec2.Subnet) (filteredSubnets []*ec2.Subnet) {
	var availabilityZone string

	// adding the requested subnetId subnet to the filtered list
	for _, subnet := range subnets {
		if *subnet.SubnetId == subnetId {
			availabilityZone = *subnet.AvailabilityZone
			filteredSubnets = append(filteredSubnets, subnet)
			break
		}
	}

	// adding all different availability zone subnets
	for _, subnet := range subnets {
		if *subnet.AvailabilityZone != availabilityZone {
			filteredSubnets = append(filteredSubnets, subnet)
		}
	}

	return
}

func getSubnetsRouteMap(vpcId, subnetId string) (routeMap map[string]string, err error) {
	routeMap = map[string]string{}

	subnets, err := GetVpcSubnets(vpcId)
	if err != nil {
		return
	}

	subnets = filterOutSameAvailabilityZoneAdditionalSubnets(subnetId, subnets)

	tables, err := GetRouteTables(vpcId)
	if err != nil {
		return
	}

	for _, s := range subnets {
		var main string
		subnetId := *s.SubnetId
	TABLES:
		for _, r := range tables {
			for _, a := range r.Associations {
				if *a.Main {
					main = *r.RouteTableId
				}
				if a.SubnetId == nil {
					continue
				}
				if *a.SubnetId == *s.SubnetId {
					routeMap[subnetId] = *r.RouteTableId
					break TABLES
				}
			}
		}
		routeMap[subnetId] = main
	}
	return
}

func GetAdditionalVpcSubnet(vpcId, subnetId string) (additionalVpcSubnet string, err error) {
	routeMap, err := getSubnetsRouteMap(vpcId, subnetId)
	if err != nil {
		return
	}

	for subnet, route := range routeMap {
		if route == routeMap[subnetId] && subnet != subnetId {
			return subnet, nil
		}
	}
	return "", NoAdditionalSubnet
}

func GetClusterInstances(clusterName cluster.ClusterName) (ids []string, err error) {
	svc := connectors.GetAWSSession().EC2
	describeResponse, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
				},
			},
			{
				Name: aws.String("tag:wekactl.io/cluster_name"),
				Values: []*string{
					aws.String(string(clusterName)),
				},
			},
		},
	})

	if err != nil {
		return
	}

	for _, reservation := range describeResponse.Reservations {
		for _, instance := range reservation.Instances {
			if instance.PrivateIpAddress == nil {
				log.Warn().Msgf("Found backend instance %s without private ip!", *instance.InstanceId)
				continue
			}
			ids = append(ids, *instance.InstanceId)
		}
	}
	return
}

func DeleteInstances(ids []string) (err error) {
	svc := connectors.GetAWSSession().EC2
	if len(ids) > 0 {
		loops := int(math.Ceil(float64(len(ids)) / 50))
		i := 0
		for i < loops {
			setToTerminate, errs := SetDisableInstancesApiTermination(
				ids[i:Min(len(ids), i+50)],
				false,
			)

			if len(errs) > 0 {
				for _, err := range errs {
					log.Error().Err(err)
				}
			}

			_, err = svc.TerminateInstances(&ec2.TerminateInstancesInput{
				InstanceIds: strings2.ListToRefList(setToTerminate),
			})

			i += 50

		}

	}

	return
}

func GetAccountId() (string, error) {
	svc := connectors.GetAWSSession().STS
	result, err := svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return *result.Account, nil
}
