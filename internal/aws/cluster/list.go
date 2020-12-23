package cluster

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/olekukonko/tablewriter"
	"os"
	"reflect"
)

type Cluster struct {
	stackId      string
	stackName    string
	creationTime string
}

func newSession(region string) *session.Session {
	config := aws.NewConfig()
	config = config.WithRegion(region)
	config = config.WithCredentialsChainVerboseErrors(true)

	// Create the options for the session
	opts := session.Options{
		Config:                  *config,
		SharedConfigState:       session.SharedConfigEnable,
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
	}

	return session.Must(session.NewSessionWithOptions(opts))
}

func getStacks(region string) ([]Cluster, error) {
	sess := newSession(region)
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

func createTable(fields []string) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(fields)
	table.SetRowLine(true)
	return table
}

func ClustersListAWS(region string) {

	fields := []string{"stackName", "creationTime"}

	clusters, err := getStacks(region)
	if err != nil {
		println(err.Error())
	} else {
		table := createTable(fields)
		for _, stack := range clusters {
			s := reflect.ValueOf(&stack)
			var values []string
			for _, field := range fields {
				values = append(values, reflect.Indirect(s).FieldByName(field).String())
			}
			table.Append(values)
		}
		table.Render()
	}
}
