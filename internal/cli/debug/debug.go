package debug

import (
"github.com/spf13/cobra"
"log"
)

var Name string
var Provider string
var Region string
var Debug = &cobra.Command{
	Use:   "debug [command] [flags]",
	Short: "Debug operations",
	Run: func(c *cobra.Command, _ []string) {
		if err := c.Help(); err != nil {
			log.Printf("ignoring cobra error %q", err.Error())
		}
	},
	SilenceUsage: true,
}

