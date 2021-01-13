package lambdas

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
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

	instanceIds := getInstanceIdsFromAutoScalingGroupOutput(asgOutput)
	ips, err := getAutoScalingGroupInstanceIps(instanceIds)
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

	return protocol.HostGroupInfoResponse{
		Username:        username,
		Password:        password,
		PrivateIps:      ips,
		DesiredCapacity: getAutoScalingGroupDesiredCapacity(asgOutput),
		InstanceIds:     ids,
		Role:            getRoleFromASGOutput(asgOutput),
	}, nil
}
