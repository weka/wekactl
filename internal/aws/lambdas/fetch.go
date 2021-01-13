package lambdas

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/lambdas/protocol"
	"wekactl/internal/connectors"
)

func getRoleFromASGOutput(asgOutput *autoscaling.DescribeAutoScalingGroupsOutput) string {
	if len(asgOutput.AutoScalingGroups) == 0 {
		return ""
	}

	for _, tag := range asgOutput.AutoScalingGroups[0].Tags {
		if *tag.Key == "wekactl.io/hostgroup_type" {
			return *tag.Value
		}
	}
	return ""
}

func GetFetchDataParams(asgName, tableName string) (fd protocol.HostGroupInfoResponse, err error) {
	svc := connectors.GetAWSSession().ASG
	input := &autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{&asgName}}
	asgOutput, err := svc.DescribeAutoScalingGroups(input)
	if err != nil {
		return
	}

	instanceIds := common.GetInstanceIdsFromAutoScalingGroupOutput(asgOutput)
	ips, err := common.GetAutoScalingGroupInstanceIps(instanceIds)
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
		PrivateIps:      ips,
		DesiredCapacity: getAutoScalingGroupDesiredCapacity(asgOutput),
		InstanceIds:     ids,
		Role:            getRoleFromASGOutput(asgOutput),
	}, nil
}
