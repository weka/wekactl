package cluster

import (
	"fmt"
	"github.com/spf13/cobra"
)

var CreateCmd = &cobra.Command{
	Use:   "create [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Creating cluster %s...\n", Name)
		return nil
	},
}

func init() {
	CreateCmd.Flags().StringVarP(&Name, "name", "n", "", "EKS cluster name")
	CreateCmd.MarkFlagRequired("name")
	Cluster.AddCommand(CreateCmd)
}
