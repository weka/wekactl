package lambdas

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"wekactl/internal/connectors"
)

type FetchData struct {
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	PrivateIps      []string `json:"private_ips"`
	DesiredCapacity int      `json:"desired_capacity"`
	InstanceIds     []string   `json:"instance_ids"`
}

func GetFetchDataParams(asgName, tableName string) (string, error) {
	svc := connectors.GetAWSSession().ASG
	input := &autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{&asgName}}
	asgOutput, err := svc.DescribeAutoScalingGroups(input)
	if err != nil {
		return "", err
	}

	instanceIds := getAutoScalingGroupInstanceIds(asgOutput)
	ips, err := getAutoScalingGroupInstanceIps(instanceIds)
	if err != nil {
		return "", err
	}

	var ids []string
	for _, instanceId := range instanceIds {
		ids = append(ids, *instanceId)
	}

	username, password, err := getUsernameAndPassword(tableName)
	if err != nil {
		return "", err
	}

	joinInfo := FetchData{
		Username:        username,
		Password:        password,
		PrivateIps:      ips,
		DesiredCapacity: getAutoScalingGroupDesiredCapacity(asgOutput),
		InstanceIds: ids,
	}
	js, err := json.Marshal(joinInfo)
	if err != nil {
		return "", err
	}

	return string(js), nil
}
