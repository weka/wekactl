package debug

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var Name string
var Provider string
var Region string
var Debug = &cobra.Command{
	Use:   "debug [command] [flags]",
	Short: "Debug operations",
	Run: func(c *cobra.Command, _ []string) {
		if err := c.Help(); err != nil {
			log.Debug().Msgf("ignoring cobra error %q", err.Error())
		}
	},
	SilenceUsage: true,
	Hidden:       true,
}

func init() {
	Debug.PersistentFlags().StringVarP(&Provider, "provider", "p", "aws", "Cloud provider")
	Debug.PersistentFlags().StringVarP(&Region, "region", "r", "", "Region")
}
