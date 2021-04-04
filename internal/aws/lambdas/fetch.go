package lambdas

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/lambdas/protocol"
	"wekactl/internal/connectors"
)

func GetFetchDataParams(clusterName, asgName, tableName, role string) (fd protocol.HostGroupInfoResponse, err error) {
	svc := connectors.GetAWSSession().ASG
	input := &autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{&asgName}}
	asgOutput, err := svc.DescribeAutoScalingGroups(input)
	if err != nil {
		return
	}

	instanceIds := common.UnpackASGInstanceIds(asgOutput.AutoScalingGroups[0].Instances)
	instances, err := common.GetInstances(instanceIds)
	if err != nil {
		return
	}

	backendIps, err := common.GetBackendsPrivateIps(clusterName)
	if err != nil {
		return
	}

	creds, err := getUsernameAndPassword(tableName)
	if err != nil {
		return
	}

	return protocol.HostGroupInfoResponse{
		Username:        creds.Username,
		Password:        creds.Password,
		DesiredCapacity: getAutoScalingGroupDesiredCapacity(asgOutput),
		Instances:       getHostGroupInfoInstances(instances),
		BackendIps:      backendIps,
		Role:            role,
	}, nil
}

func getHostGroupInfoInstances(instances []*ec2.Instance) (ret []protocol.HgInstance) {
	for _, i := range instances {
		if i.InstanceId != nil && i.PrivateIpAddress != nil {
			ret = append(ret, protocol.HgInstance{
				Id:        *i.InstanceId,
				PrivateIp: *i.PrivateIpAddress,
			})
		}
	}
	return
}
