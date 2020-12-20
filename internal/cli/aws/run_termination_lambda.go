package aws

import (
	"fmt"
	"github.com/spf13/cobra"
)

var InstanceID string
var RunTerminationLambdaCmd = &cobra.Command{
	Use:   "run-termination-lambda [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Running aws termination lambda InstanceID:%s...\n", InstanceID)
		return nil
	},
}

func init() {
	RunTerminationLambdaCmd.Flags().StringVarP(
		&InstanceID, "instance-id", "i", "", "Instance ID")
	RunTerminationLambdaCmd.MarkFlagRequired("instance-id")
	AWS.AddCommand(RunTerminationLambdaCmd)
}
