package hostgroup

import (
	"fmt"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("hostgroup list called")
		return nil
	},
}

func init() {
	HostGroup.AddCommand(listCmd)
}
