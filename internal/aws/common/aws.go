package common

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/semaphore"
	"os"
	"sync"
	"wekactl/internal/aws/hostgroups"
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

func GetAutoScalingGroupInstanceIds(asgName string) ([]*string, error) {
	svc := connectors.GetAWSSession().ASG
	asgOutput, err := svc.DescribeAutoScalingGroups(
		&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{&asgName},
		},
	)
	if err != nil {
		return []*string{}, err
	}
	return GetInstanceIdsFromAutoScalingGroupOutput(asgOutput), nil
}

func GetInstanceIdsFromAutoScalingGroupOutput(asgOutput *autoscaling.DescribeAutoScalingGroupsOutput) []*string {
	var instanceIds []*string
	if len(asgOutput.AutoScalingGroups) == 0 {
		return []*string{}
	}
	for _, instance := range asgOutput.AutoScalingGroups[0].Instances {
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

func GetInstancesIps(instances []*ec2.Instance) []string {
	var instanceIps []string
	for _, instance := range instances {
		instanceIps = append(instanceIps, *instance.PrivateIpAddress)
	}
	return instanceIps
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

	for _, reservation := range describeResponse.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, instance)
		}
	}
	return
}

func GenerateResourceName(clusterName cluster.ClusterName, hostGroupName hostgroups.HostGroupName) string {
	resourceName := "wekactl-" + string(clusterName)
	name := string(hostGroupName)
	if name != "" {
		resourceName += "-" + name
	}
	return resourceName
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