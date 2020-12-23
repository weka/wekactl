package cluster

import (
	"github.com/spf13/cobra"
	"log"
)

var Name string
var Provider string
var Region string
var Cluster = &cobra.Command{
	Use:   "cluster [command] [flags]",
	Short: "Cluster operations",
	Run: func(c *cobra.Command, _ []string) {
		if err := c.Help(); err != nil {
			log.Printf("ignoring cobra error %q", err.Error())
		}
	},
	SilenceUsage: true,
	Aliases: []string{"clusters"},
}
