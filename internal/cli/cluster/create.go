package cluster

import (
	"fmt"
	"github.com/spf13/cobra"
)

var createParams struct {
	name string
}

var createCmd = &cobra.Command{
	Use:   "create [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Creating cluster %s...\n", createParams.name)
		return nil
	},
}

func init() {
	createCmd.Flags().StringVarP(&createParams.name, "name", "n", "", "EKS cluster name")
	_ = createCmd.MarkFlagRequired("name")
}
