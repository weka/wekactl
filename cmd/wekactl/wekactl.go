package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"wekactl/internal/cli/aws"
	"wekactl/internal/cli/cluster"
	"wekactl/internal/cli/debug"
	"wekactl/internal/cli/hostgroup"
	"wekactl/internal/cli/version"
	"wekactl/internal/env"
)

var rootCmd = &cobra.Command{
	Use:   "wekactl [group] [command] [flags]",
	Short: "The official CLI for managing weka cloud formation stacks",
	Run: func(c *cobra.Command, _ []string) {
		if err := c.Help(); err != nil {
			log.Debug().Msgf("ignoring cobra error %q", err.Error())
		}
	},
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err)
	}
}

func init() {
	rootCmd.AddCommand(cluster.Cluster)
	rootCmd.AddCommand(hostgroup.HostGroup)
	rootCmd.AddCommand(aws.AWS)
	rootCmd.AddCommand(debug.Debug)
	rootCmd.AddCommand(version.Version)

	rootCmd.PersistentFlags().StringVarP(&env.Config.Provider, "provider", "c", "aws", "Cloud provider")
	rootCmd.PersistentFlags().StringVarP(&env.Config.Region, "region", "r", "", "Region")
}

func configureLogging() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	} else {
		level, err := zerolog.ParseLevel(logLevel)
		if err != nil {
			log.Fatal().Err(err)
		}
		zerolog.SetGlobalLevel(level)
	}
}

func main() {
	configureLogging()
	Execute()
}
