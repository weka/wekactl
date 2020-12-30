package cluster

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var Name string
var Provider string
var Region string
var Username string
var Password string
var Cluster = &cobra.Command{
	Use:   "cluster [command] [flags]",
	Short: "Cluster operations",
	Run: func(c *cobra.Command, _ []string) {
		if err := c.Help(); err != nil {
			log.Debug().Msgf("ignoring cobra error %q", err.Error())
		}
	},
	SilenceUsage: true,
	Aliases:      []string{"clusters"},
}

func init() {
	Cluster.PersistentFlags().StringVarP(&Provider, "provider", "c", "aws", "Cloud provider")
	Cluster.PersistentFlags().StringVarP(&Region, "region", "r", "", "Region")
}
