package cluster

import "C"
import (
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/cluster"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if Provider == "aws" {
			cluster.ClustersListAWS(Region)
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", Provider)
		}
	},
}

func init() {
	listCmd.Flags().StringVarP(&Provider, "provider", "p", "aws", "Cloud provider")
	listCmd.Flags().StringVarP(&Region, "region", "r", "", "Region")
	Cluster.AddCommand(listCmd)
}
