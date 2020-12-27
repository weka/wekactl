package debug

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"wekactl/internal/aws/common"
)

type Instance struct {
	instanceId string
	status     string
	role       string
}

func getInstancesStatus(region string, instanceId string) string {
	sess := common.NewSession(region)
	svc := ec2.New(sess)
	input := &ec2.DescribeInstanceStatusInput{
		InstanceIds: []*string{
			aws.String(instanceId),
		},
	}
	result, err := svc.DescribeInstanceStatus(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				return aerr.Code()
			}
		} else {
			return ""
		}
	} else {
		return *result.InstanceStatuses[0].InstanceState.Name
	}
}
func getStackInstances(region, stackName string) ([]Instance, error) {
	sess := common.NewSession(region)
	svc := cloudformation.New(sess)
	input := &cloudformation.DescribeStackResourcesInput{StackName: &stackName}
	result, err := svc.DescribeStackResources(input)
	var instances []Instance
	if err != nil {
		return nil, err
	} else {
		for _, resource := range result.StackResources {
			if *resource.ResourceType == "AWS::EC2::Instance" {
				instances = append(instances, Instance{
					instanceId: *resource.PhysicalResourceId,
					status:     getInstancesStatus(region, *resource.PhysicalResourceId),
					role:       *resource.LogicalResourceId,
				})
			}
		}
	}
	return instances, nil
}

func RenderInstancesTable(region, stackName string) {
	fields := []string{
		"instanceId",
		"status",
		"role",
	}
	instances, err := getStackInstances(region, stackName)
	if err != nil {
		println(err.Error())
	} else {
		var data [][]string
		for _, instance := range instances {
			data = append(data, []string{
				instance.instanceId,
				instance.status,
				instance.role,
			})
		}
		common.RenderTable(fields, data)
	}
}
