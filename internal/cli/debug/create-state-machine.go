package debug

import (
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var createStateMachineCmd = &cobra.Command{
	Use:   "create-state-machine",
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
			policy, err := cluster.GetJoinAndFetchLambdaPolicy()
			if err != nil {
				return err
			}
			assumeRolePolicy, err := cluster.GetJoinAndFetchAssumeRolePolicy()
			if err != nil {
				return err
			}
			lambdaConfiguration, err := cluster.CreateLambda(hostGroup, "fetch", "Backends", assumeRolePolicy, policy)
			if err != nil {
				return err
			}
			err = cluster.CreateStateMachine(hostGroup, *lambdaConfiguration.FunctionArn)
			if err != nil {
				return err
			}
			logging.UserSuccess("State machine creation completed successfully!")
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", env.Config.Provider)
		}
		return nil
	},
}

func init() {
	createStateMachineCmd.Flags().StringVarP(&StackName, "name", "n", "", "Cloudformation Stack name")

	_ = createStateMachineCmd.MarkFlagRequired("name")
	Debug.AddCommand(createStateMachineCmd)
}
