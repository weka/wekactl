package aws

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var AWS = &cobra.Command{
	Use:   "aws [command] [flags]",
	Short: "Aws operations",
	Run: func(c *cobra.Command, _ []string) {
		if err := c.Help(); err != nil {
			log.Debug().Msgf("ignoring cobra error %q", err.Error())
		}
	},
	SilenceUsage: true,
	Hidden:       true,
}
