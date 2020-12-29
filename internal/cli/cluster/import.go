package cluster

import (
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/cluster"
)

var importCmd = &cobra.Command{
	Use:   "import [flags]",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if Provider == "aws" {
			cluster.ImportCluster(Region, Name)
			fmt.Println("Import finished successfully!")
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", Provider)
		}
	},
}

func init() {
	importCmd.Flags().StringVarP(&Name, "name", "n", "", "EKS cluster name")
	importCmd.MarkFlagRequired("name")
	Cluster.AddCommand(importCmd)
}
