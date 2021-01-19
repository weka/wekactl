package lambdas

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/lambdas/protocol"
	"wekactl/internal/connectors"
)

func GetFetchDataParams(asgName, tableName, role string) (fd protocol.HostGroupInfoResponse, err error) {
	svc := connectors.GetAWSSession().ASG
	input := &autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{&asgName}}
	asgOutput, err := svc.DescribeAutoScalingGroups(input)
	if err != nil {
		return
	}

	instanceIds := common.GetInstanceIdsFromAutoScalingGroupOutput(asgOutput)
	instances, err := common.GetInstances(instanceIds)
	if err != nil {
		return
	}

	var ids []string
	for _, instanceId := range instanceIds {
		ids = append(ids, *instanceId)
	}

	username, password, err := getUsernameAndPassword(tableName)
	if err != nil {
		return
	}

	// TODO: Replace with call to GetInstance and return structs of InstanceId+PrivateIp,
	// 	so we can select Inactive hosts that belong to HostGroip
	return protocol.HostGroupInfoResponse{
		Username:        username,
		Password:        password,
		DesiredCapacity: getAutoScalingGroupDesiredCapacity(asgOutput),
		Instances:       getHostGroupInfoInstances(instances),
		Role:            role,
	}, nil
}

func getHostGroupInfoInstances(instances []*ec2.Instance) (ret []protocol.HgInstance) {
	for _, i := range instances {
		ret = append(ret, protocol.HgInstance{
			Id:        *i.InstanceId,
			PrivateIp: *i.PrivateIpAddress,
		})
	}
	return
}
