package aws

import (
	"github.com/spf13/cobra"
	"log"
)

var AWS = &cobra.Command{
	Use:   "aws [command] [flags]",
	Short: "Aws operations",
	Run: func(c *cobra.Command, _ []string) {
		if err := c.Help(); err != nil {
			log.Printf("ignoring cobra error %q", err.Error())
		}
	},
	SilenceUsage: true,
	Hidden: true,
}
