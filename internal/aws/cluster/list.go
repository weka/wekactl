package cluster

import (
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"wekactl/internal/aws/common"
)

type Cluster struct {
	stackId      string
	stackName    string
	creationTime string
}

func getStacks(region string) ([]Cluster, error) {
	sess := common.NewSession(region)
	svc := cloudformation.New(sess)
	input := &cloudformation.ListStacksInput{}

	result, err := svc.ListStacks(input)
	if err != nil {
		return nil, err
	} else {
		var stacks []Cluster
		for _, stack := range result.StackSummaries {
			if *stack.TemplateDescription ==
				"[WekaIO] To learn more about this template visit https://docs.weka.io/install/aws/cloudformation" &&
				*stack.StackStatus == "CREATE_COMPLETE" {
				stacks = append(stacks, Cluster{
					stackId:      *stack.StackId,
					stackName:    *stack.StackName,
					creationTime: (*stack.CreationTime).Format("2006-01-02 15:04:05.000"),
				})
			}
		}
		return stacks, nil
	}
}

func RenderStacksTable(region string) {

	fields := []string{
		"stackName",
		"creationTime",
	}

	clusters, err := getStacks(region)
	if err != nil {
		println(err.Error())
	} else {
		var data [][]string
		for _, stack := range clusters {
			data = append(data, []string{
				stack.stackName,
				stack.creationTime,
			})
		}
		common.RenderTable(fields, data)
	}
}
