package cluster

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var Region string
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
	Cluster.AddCommand(createCmd)
	Cluster.AddCommand(importCmd)
	Cluster.AddCommand(listCmd)
	_ = Cluster.MarkPersistentFlagRequired("region")
}
