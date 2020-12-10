package cluster

import (
	"fmt"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Importing cluster %s...\n", Name)
		return nil
	},
}

func init() {
	importCmd.Flags().StringVarP(&Name, "name", "n", "", "EKS cluster name")
	importCmd.MarkFlagRequired("name")
	Cluster.AddCommand(importCmd)
}
