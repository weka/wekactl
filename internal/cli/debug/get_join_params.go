package debug

import (
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/lambdas"
)

var AsgName string
var TableName string
var StackId string
var GetInstanceJoinParamsCmd = &cobra.Command{
	Use:   "get-join-params",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if Provider == "aws" {
			res, err := lambdas.GetJoinParams(AsgName, TableName)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(res)
			}
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", Provider)
		}
	},
}

func init() {
	GetInstanceJoinParamsCmd.Flags().StringVarP(&AsgName, "asg-name", "g", "", "Auto scaling group name")
	GetInstanceJoinParamsCmd.Flags().StringVarP(&TableName, "table-name", "t", "", "Dynamo DB table name")

	_ = GetInstanceJoinParamsCmd.MarkFlagRequired("asg-name")
	_ = GetInstanceJoinParamsCmd.MarkFlagRequired("table-name")
	Debug.AddCommand(GetInstanceJoinParamsCmd)
}
