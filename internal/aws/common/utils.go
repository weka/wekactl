package common

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/semaphore"
	"os"
	"sync"
	"sync/atomic"
	"wekactl/internal/connectors"
)

func RenderTable(fields []string, data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(fields)
	table.SetRowLine(true)
	table.AppendBulk(data)
	table.Render()
}

func disableInstanceApiTermination(instanceId string, value bool) (*ec2.ModifyInstanceAttributeOutput, error) {
	svc := connectors.GetAWSSession().EC2
	input := &ec2.ModifyInstanceAttributeInput{
		DisableApiTermination: &ec2.AttributeBooleanValue{
			Value: aws.Bool(value),
		},
		InstanceId: aws.String(instanceId),
	}
	return svc.ModifyInstanceAttribute(input)
}

var terminationSemaphore *semaphore.Weighted

func init() {
	terminationSemaphore = semaphore.NewWeighted(20)
}

func DisableInstancesApiTermination(instanceIds []*string, value bool) error {
	var wg sync.WaitGroup
	var failedInstances int64

	wg.Add(len(instanceIds))
	for i := range instanceIds {
		go func(i int) {
			_ = terminationSemaphore.Acquire(context.Background(), 1)
			defer terminationSemaphore.Release(1)
			defer wg.Done()

			_, err := disableInstanceApiTermination(*instanceIds[i], value)
			if err != nil {
				atomic.AddInt64(&failedInstances, 1)
				log.Error().Msgf("failed to set DisableApiTermination on %s", *instanceIds[i])
			}
		}(i)
	}
	wg.Wait()
	if failedInstances != 0 {
		return errors.New(fmt.Sprintf("failed to set DisableApiTermination on %d instances", failedInstances))
	}
	return nil

}
