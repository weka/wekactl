package lambdas

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"time"
	"wekactl/internal/aws/common"
	"wekactl/internal/connectors"
)

type InstanceIdsSet map[string]bool

type ClusterHostInfo struct {
	Hosts []struct {
		InstanceId string `json:"instance_id"`
		Status     string `json:"status"`
		AddedTime  string `json:"added_time"`
		HostId     string `json:"host_id"`
	} `json:"hosts"`
}

type TerminatedInstance struct {
	InstanceId string    `json:"instance_id"`
	Creation   time.Time `json:"creation"`
}
type TerminatedInstances struct {
	Instances []TerminatedInstance `json:"instances"`
}

func getInstanceIdsSet(clusterHostInfo ClusterHostInfo) InstanceIdsSet {
	instanceIdsSet := make(InstanceIdsSet)
	for _, instance := range clusterHostInfo.Hosts {
		instanceIdsSet[instance.InstanceId] = true
	}
	return instanceIdsSet
}

func getDeltaInstancesIds(asgName string, clusterHostInfo ClusterHostInfo) ([]*string, error) {
	instanceIdsSet := getInstanceIdsSet(clusterHostInfo)
	var deltaInstanceIDs []*string

	asgInstanceIds, err := getAutoScalingGroupInstanceIds(asgName)
	if err != nil {
		return nil, err
	}

	for _, instanceId := range asgInstanceIds {
		if !instanceIdsSet[*instanceId] {
			deltaInstanceIDs = append(deltaInstanceIDs, instanceId)
		}
	}
	return deltaInstanceIDs, nil
}

func removeAutoScalingProtection(asgName string, instanceIds []*string) error {
	svc := connectors.GetAWSSession().ASG
	_, err := svc.SetInstanceProtection(&autoscaling.SetInstanceProtectionInput{
		AutoScalingGroupName: &asgName,
		InstanceIds:          instanceIds,
		ProtectedFromScaleIn: aws.Bool(false),
	})
	if err != nil {
		return err
	}
	return nil
}

func terminateDeltaInstances(asgName string, instanceIds []*string) (TerminatedInstances, error) {
	svc := connectors.GetAWSSession().EC2
	var terminatedInstances TerminatedInstances

	if len(instanceIds) == 0 {
		return terminatedInstances, nil
	}

	var terminateInstanceIds []*string
	result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	})
	if err != nil {
		return terminatedInstances, err
	}

	for _, reservation := range result.Reservations {
		instance := reservation.Instances[0]
		if time.Now().Sub(*instance.LaunchTime) > time.Minute*30 {
			terminatedInstances.Instances = append(terminatedInstances.Instances, TerminatedInstance{
				*instance.InstanceId,
				*instance.LaunchTime,
			})
			terminateInstanceIds = append(terminateInstanceIds, instance.InstanceId)
		}
	}
	err = common.DisableInstancesApiTermination(terminateInstanceIds, false)
	if err != nil {
		return terminatedInstances, err
	}
	err = removeAutoScalingProtection(asgName, terminateInstanceIds)
	if err != nil {
		return terminatedInstances, err
	}
	return terminatedInstances, nil
}

func TerminateInstances(asgName string, clusterHostInfo ClusterHostInfo) (terminatedInstances TerminatedInstances, err error) {
	deltaInstanceIds, err := getDeltaInstancesIds(asgName, clusterHostInfo)
	if err != nil {
		return
	}
	terminatedInstances, err = terminateDeltaInstances(asgName, deltaInstanceIds)
	return
}
