package debug

import (
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/common"
	"wekactl/internal/env"
)

var GetBackendsPrivateIpsCmd = &cobra.Command{
	Use:   "get-backends-private-ips",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {
			ips, err := common.GetBackendsPrivateIps(StackName)
			if err != nil {
				return err
			}
			fmt.Printf("Found %d backends private ips: %s\n", len(ips), ips)
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", env.Config.Provider)
		}
		return nil
	},
}

func init() {
	GetBackendsPrivateIpsCmd.Flags().StringVarP(&StackName, "name", "n", "", "Cloudformation Stack name")
	GetBackendsPrivateIpsCmd.MarkFlagRequired("name")
	Debug.AddCommand(GetBackendsPrivateIpsCmd)
}
