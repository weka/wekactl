package lambdas

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

func getAutoScalingGroupDesiredCapacity(asgOutput *autoscaling.DescribeAutoScalingGroupsOutput) int {
	if len(asgOutput.AutoScalingGroups) == 0 {
		return -1
	}

	return int(*asgOutput.AutoScalingGroups[0].DesiredCapacity)
}
