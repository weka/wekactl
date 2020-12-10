package cluster

import "C"
import (
	"fmt"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cluster list called")
	},
}

func init() {
	Cluster.AddCommand(listCmd)
}
