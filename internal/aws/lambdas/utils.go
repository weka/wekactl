package lambdas

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"wekactl/internal/aws/db"
)

func getAutoScalingGroupDesiredCapacity(asgOutput *autoscaling.DescribeAutoScalingGroupsOutput) int {
	if len(asgOutput.AutoScalingGroups) == 0 {
		return -1
	}

	return int(*asgOutput.AutoScalingGroups[0].DesiredCapacity)
}

func GetUsernameAndPassword(tableName string) (creds db.ClusterCreds, err error) {
	err = db.GetItem(tableName, db.ModelClusterCreds, &creds)
	if err != nil {
		return
	}
	return
}
