package debug

import (
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var StateMachineArn string
var createCloudWatchEventCmd = &cobra.Command{
	Use:   "create-cloud-watch-event-rule",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {
			stackId, err := cluster.GetStackId(StackName)
			if err != nil {
				return err
			}
			hostGroup := cluster.HostGroup{
				Name: "Backends",
				Role: "backend",
				Stack: cluster.Stack{
					StackId:   stackId,
					StackName: StackName,
				},
			}
			err = cluster.CreateCloudWatchEventRule(hostGroup, &StateMachineArn)
			if err != nil {
				return err
			}
			logging.UserSuccess("CloudWatchEvent rule creation completed successfully!")
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", env.Config.Provider)
		}
		return nil
	},
}

func init() {
	createCloudWatchEventCmd.Flags().StringVarP(&StackName, "name", "n", "", "StateMachineArn")
	createCloudWatchEventCmd.Flags().StringVarP(&StateMachineArn, "arn", "m", "", "StateMachineArn")

	_ = createCloudWatchEventCmd.MarkFlagRequired("name")
	Debug.AddCommand(createCloudWatchEventCmd)
}
