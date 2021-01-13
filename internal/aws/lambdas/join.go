package lambdas

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"wekactl/internal/aws/common"
	"wekactl/internal/connectors"
)

type JoinInfo struct {
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	PrivateIps      []string `json:"private_ips"`
	DesiredCapacity int      `json:"desired_capacity"`
}

func GetJoinParams(asgName, tableName string) (string, error) {
	svc := connectors.GetAWSSession().ASG
	input := &autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{&asgName}}
	asgOutput, err := svc.DescribeAutoScalingGroups(input)
	if err != nil {
		return "", err
	}

	instanceIds := common.GetInstanceIdsFromAutoScalingGroupOutput(asgOutput)
	ips, err := common.GetAutoScalingGroupInstanceIps(instanceIds)
	if err != nil {
		return "", err
	}

	username, password, err := getUsernameAndPassword(tableName)
	if err != nil {
		return "", err
	}

	joinInfo := JoinInfo{
		Username:        username,
		Password:        password,
		PrivateIps:      ips,
		DesiredCapacity: getAutoScalingGroupDesiredCapacity(asgOutput),
	}
	js, err := json.Marshal(joinInfo)
	if err != nil {
		return "", err
	}

	return string(js), nil
}
