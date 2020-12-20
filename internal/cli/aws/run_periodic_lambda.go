package aws

import (
	"fmt"
	"github.com/spf13/cobra"
)

var RunPeriodicLambdaCmd = &cobra.Command{
	Use:   "run-periodic-lambda [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Running aws periodic lambda...")
		return nil
	},
}

func init() {
	AWS.AddCommand(RunPeriodicLambdaCmd)
}
