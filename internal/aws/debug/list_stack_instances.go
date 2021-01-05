package debug

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"wekactl/internal/aws/common"
	"wekactl/internal/connectors"
)

type Instance struct {
	instanceId string
	status     string
	role       string
}

func getInstancesStatus(instanceId string) string {
	svc := connectors.GetAWSSession().EC2
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
func getStackInstances(stackName string) ([]Instance, error) {
	svc := connectors.GetAWSSession().CF
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
					status:     getInstancesStatus(*resource.PhysicalResourceId),
					role:       *resource.LogicalResourceId,
				})
			}
		}
	}
	return instances, nil
}

func RenderInstancesTable(stackName string) {
	fields := []string{
		"instanceId",
		"status",
		"role",
	}
	instances, err := getStackInstances(stackName)
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
