package terminate

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/rs/zerolog/log"
	"os"
	"time"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/lambdas/protocol"
	"wekactl/internal/connectors"
	"wekactl/internal/lib/types"
)

type instanceIdsSet map[string]types.Nilt
type instancesMap map[string]*ec2.Instance

func getInstanceIdsSet(scaleResponse protocol.ScaleResponse) instanceIdsSet {
	instanceIdsSet := make(instanceIdsSet)
	for _, instance := range scaleResponse.Hosts {
		instanceIdsSet[instance.InstanceId] = types.Nilv
	}
	return instanceIdsSet
}

func instancesToMap(instances []*ec2.Instance) instancesMap {
	im := make(instancesMap)
	for _, instance := range instances {
		im[*instance.InstanceId] = instance
	}
	return im
}

func getDeltaInstancesIds(asgInstanceIds []*string, scaleResponse protocol.ScaleResponse) ([]*string, error) {
	instanceIdsSet := getInstanceIdsSet(scaleResponse)
	var deltaInstanceIDs []*string

	for _, instanceId := range asgInstanceIds {
		if _, ok := instanceIdsSet[*instanceId]; !ok {
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

func setForExplicitRemoval(instance *ec2.Instance, toRemove []protocol.HgInstance) bool {
	for _, i := range toRemove {
		if *instance.PrivateIpAddress == i.PrivateIp && *instance.InstanceId == i.Id {
			return true
		}
	}
	return false
}

func terminateUnneededInstances(asgName string, instances []*ec2.Instance, explicitRemoval []protocol.HgInstance) (terminated []*ec2.Instance, errs []error) {
	terminateInstanceIds := make([]*string, 0, 0)
	imap := instancesToMap(instances)

	for _, instance := range instances {
		if !setForExplicitRemoval(instance, explicitRemoval) {
			if time.Now().Sub(*instance.LaunchTime) < time.Minute*30 {
				continue
			}
		}
		instanceState := *instance.State.Name
		if instanceState != ec2.InstanceStateNameShuttingDown && instanceState != ec2.InstanceStateNameTerminated {
			terminateInstanceIds = append(terminateInstanceIds, instance.InstanceId)
		}
	}

	if len(terminateInstanceIds) == 0 {
		return
	}
	setToTerminate, errs := common.SetDisableInstancesApiTermination(
		terminateInstanceIds[:common.Min(len(terminateInstanceIds), 50)],
		false,
	)

	err := removeAutoScalingProtection(asgName, setToTerminate)
	if err != nil {
		// WARNING: This is debatable if error here is transient or not
		//	Specifically now we can return empty list of what we were able to terminate because this API call failed
		//   But in future with adding more lambdas into state machine this might become wrong decision
		log.Error().Err(err)
		setToTerminate = setToTerminate[:0]
		errs = append(errs, err)
	}

	for _, id := range setToTerminate {
		terminated = append(terminated, imap[*id])
	}
	return
}

func Handler(scaleResponse protocol.ScaleResponse) (response protocol.TerminatedInstancesResponse, err error) {
	asgName := os.Getenv("ASG_NAME")
	if asgName == "" {
		err = errors.New("ASG_NAME env var is mandatory")
		return
	}
	response.TransientErrors = scaleResponse.TransientErrors[0:len(scaleResponse.TransientErrors):len(scaleResponse.TransientErrors)]

	asgInstanceIds, err := common.GetAutoScalingGroupInstanceIds(asgName)
	if err != nil {
		return
	}

	deltaInstanceIds, err := getDeltaInstancesIds(asgInstanceIds, scaleResponse)
	if err != nil {
		return
	}

	if len(deltaInstanceIds) == 0 {
		return
	}
	candidatesToTerminate, err := common.GetInstances(deltaInstanceIds)
	if err != nil {
		return
	}

	terminatedInstances, errs := terminateUnneededInstances(asgName, candidatesToTerminate, scaleResponse.ToTerminate)
	response.AddTransientErrors(errs)

	for _, instance := range terminatedInstances {
		response.Instances = append(response.Instances, protocol.TerminatedInstance{
			InstanceId: *instance.InstanceId,
			Creation:   *instance.LaunchTime,
		})
	}
	// TODO: Add another step that handles transient errors
	return
}
