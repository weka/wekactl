package debug

import (
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/debug"
	"wekactl/internal/env"
)

var listInstancesCmd = &cobra.Command{
	Use:   "list-stack-instances",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if env.Config.Provider == "aws" {
			debug.RenderInstancesTable(StackName)
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", env.Config.Provider)
		}
	},
}

func init() {
	listInstancesCmd.Flags().StringVarP(&StackName, "name", "n", "", "Cloudformation Stack name")
	listInstancesCmd.MarkFlagRequired("name")
	Debug.AddCommand(listInstancesCmd)
}
