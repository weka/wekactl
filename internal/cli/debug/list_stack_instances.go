package debug

import (
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/debug"
)

var listInstancesCmd = &cobra.Command{
	Use:   "list-stack-instances",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if Provider == "aws" {
			debug.RenderInstancesTable(Region, Name)
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", Provider)
		}
	},
}

func init() {
	listInstancesCmd.Flags().StringVarP(&Name, "name", "n", "", "Cloudformation Stack name")
	listInstancesCmd.MarkFlagRequired("name")
	Debug.AddCommand(listInstancesCmd)
}
