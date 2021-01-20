package debug

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var createLambdaCmd = &cobra.Command{
	Use:   "create-lambda",
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

			stackInstances, err := cluster.GetStackInstancesInfo(StackName)
			var lambdaVpcConfig lambda.VpcConfig
			var policy string
			switch Lambda {
			case "join":
				policy, err = cluster.GetJoinAndFetchLambdaPolicy()
			case "fetch":
				policy, err = cluster.GetJoinAndFetchLambdaPolicy()
			case "scale":
				policy, err = cluster.GetScaleLambdaPolicy()
				lambdaVpcConfig = cluster.GetLambdaVpcConfig(stackInstances.Backends[0])

			case "terminate":
				policy, err = cluster.GetTerminateLambdaPolicy()
			default:
				err = errors.New("invalid lambda type")
			}

			if err != nil {
				return err
			}
			assumeRolePolicy, err := cluster.GetLambdaAssumeRolePolicy()
			if err != nil {
				return err
			}

			if err != nil {
				return err
			}
			_, err = cluster.CreateLambda(hostGroup, Lambda, "Backends", assumeRolePolicy, policy, lambdaVpcConfig)
			if err != nil {
				return err
			}
			logging.UserSuccess("Lambda endpoint creation completed successfully!")
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", env.Config.Provider)
		}
		return nil
	},
}

func init() {
	createLambdaCmd.Flags().StringVarP(&StackName, "name", "n", "", "Cloudformation Stack name")
	createLambdaCmd.Flags().StringVarP(&Lambda, "lambda", "t", "", "Lambda type to create")

	_ = createLambdaCmd.MarkFlagRequired("name")
	_ = createLambdaCmd.MarkFlagRequired("lambda")
	Debug.AddCommand(createLambdaCmd)
}
