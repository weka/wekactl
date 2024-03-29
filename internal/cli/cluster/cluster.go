package cluster

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var Region string
var StackName string
var DryRun bool

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
	//Cluster.AddCommand(createCmd)
	Cluster.AddCommand(importCmd)
	Cluster.AddCommand(listCmd)
	Cluster.AddCommand(destroyCmd)
	Cluster.AddCommand(updateCmd)
	Cluster.AddCommand(changeCredentialsCmd)
	Cluster.AddCommand(joinParamsCmd)
	_ = Cluster.MarkPersistentFlagRequired("region")
}
