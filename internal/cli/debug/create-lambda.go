package debug

import (
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var Lambda string
var createLambdaCmd = &cobra.Command{
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
			policy, err := cluster.GetJoinAndFetchLambdaPolicy()
			if err != nil {
				return err
			}
			assumeRolePolicy, err := cluster.GetLambdaAssumeRolePolicy()
			if err != nil {
				return err
			}
			err = cluster.CreateLambdaEndPoint(hostGroup, Lambda, "Backends", assumeRolePolicy, policy)
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
