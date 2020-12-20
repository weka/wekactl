package aws

import (
	"fmt"
	"github.com/spf13/cobra"
)

var RunSystemScaleLambdaCmd = &cobra.Command{
	Use:   "run-system-scale-lambda [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Running aws system scale lambda...")
		return nil
	},
}

func init() {
	AWS.AddCommand(RunSystemScaleLambdaCmd)
}
