package hostgroup

import (
	"github.com/spf13/cobra"
	"log"
)

var HostGroup = &cobra.Command{
	Use:   "hostgroup [command] [flags]",
	Short: "HostGroup operations",
	Run: func(c *cobra.Command, _ []string) {
		if err := c.Help(); err != nil {
			log.Printf("ignoring cobra error %q", err.Error())
		}
	},
	SilenceUsage: true,
}

