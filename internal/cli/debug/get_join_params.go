package debug

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/env"
)

var AsgName string
var TableName string
var StackId string
var GetInstanceJoinParamsCmd = &cobra.Command{
	Use:   "get-join-params",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if env.Config.Provider == "aws" {
			var ctx context.Context
			res, err := lambdas.GetJoinParams(ctx, StackName, AsgName, TableName, "Backends")
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(res)
			}
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", env.Config.Provider)
		}
	},
}

func init() {
	GetInstanceJoinParamsCmd.Flags().StringVarP(&StackName, "name", "n", "", "StackName")
	GetInstanceJoinParamsCmd.Flags().StringVarP(&AsgName, "asg-name", "g", "", "Auto scaling group name")
	GetInstanceJoinParamsCmd.Flags().StringVarP(&TableName, "table-name", "t", "", "Dynamo DB table name")

	_ = GetInstanceJoinParamsCmd.MarkFlagRequired("name")
	_ = GetInstanceJoinParamsCmd.MarkFlagRequired("asg-name")
	_ = GetInstanceJoinParamsCmd.MarkFlagRequired("table-name")
	Debug.AddCommand(GetInstanceJoinParamsCmd)
}
