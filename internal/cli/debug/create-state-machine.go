package debug

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
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
			fetchAndJoinPolicy, err := cluster.GetJoinAndFetchLambdaPolicy()
			if err != nil {
				return err
			}
			assumeRolePolicy, err := cluster.GetLambdaAssumeRolePolicy()
			if err != nil {
				return err
			}
			fetchLambda, err := cluster.CreateLambda(hostGroup, "fetch", "Backends", assumeRolePolicy, fetchAndJoinPolicy, lambda.VpcConfig{})
			if err != nil {
				return err
			}
			scaleInPolicy, err := cluster.GetScaleInLambdaPolicy()
			if err != nil {
				return err
			}
			terminatePolicy, err := cluster.GetTerminateLambdaPolicy()
			if err != nil {
				return err
			}
			stackInstances, err := cluster.GetInstancesInfo(StackName)
			if err != nil {
				return err
			}
			lambdaVpcConfig := cluster.GetLambdaVpcConfig(stackInstances.Backends[0])
			scaleInLambda, err := cluster.CreateLambda(hostGroup, "scale-in", "Backends", assumeRolePolicy, scaleInPolicy, lambdaVpcConfig)
			if err != nil {
				return err
			}
			terminateLambda, err := cluster.CreateLambda(hostGroup, "terminate", "Backends", assumeRolePolicy, terminatePolicy, lambdaVpcConfig)
			if err != nil {
				return err
			}
			lambdas := cluster.StateMachineLambdas{
				Fetch:     *fetchLambda.FunctionArn,
				ScaleIn:   *scaleInLambda.FunctionArn,
				Terminate: *terminateLambda.FunctionArn,
			}
			err = cluster.CreateStateMachine(hostGroup, lambdas)
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
