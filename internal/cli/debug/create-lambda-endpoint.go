package debug

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var createLambdaEndPointCmd = &cobra.Command{
	Use:   "create-lambda-endpoint",
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
			if Lambda != "join" {
				logging.UserFailure("Supported only with join lambda")
				return nil
			}
			policy, err := cluster.GetJoinAndFetchLambdaPolicy()
			if err != nil {
				return err
			}
			assumeRolePolicy, err := cluster.GetLambdaAssumeRolePolicy()
			if err != nil {
				return err
			}
			err = cluster.CreateLambdaEndPoint(hostGroup, Lambda, "Backends", assumeRolePolicy, policy, lambda.VpcConfig{})
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
	createLambdaEndPointCmd.Flags().StringVarP(&StackName, "name", "n", "", "Cloudformation Stack name")
	createLambdaEndPointCmd.Flags().StringVarP(&Lambda, "lambda", "t", "", "Lambda type to create")

	_ = createLambdaEndPointCmd.MarkFlagRequired("name")
	_ = createLambdaEndPointCmd.MarkFlagRequired("lambda")
	Debug.AddCommand(createLambdaEndPointCmd)
}
