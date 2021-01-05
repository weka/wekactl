package cluster

import (
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/cluster"
	"wekactl/internal/env"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if env.Config.Provider == "aws" {
			cluster.RenderStacksTable(Region)
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", env.Config.Provider)
		}
	},
}
