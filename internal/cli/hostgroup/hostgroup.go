package hostgroup

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var HostGroup = &cobra.Command{
	Use:   "hostgroup [command] [flags]",
	Short: "HostGroup operations",
	Run: func(c *cobra.Command, _ []string) {
		if err := c.Help(); err != nil {
			log.Debug().Msgf("ignoring cobra error %q", err.Error())
		}
	},
	SilenceUsage: true,
}
