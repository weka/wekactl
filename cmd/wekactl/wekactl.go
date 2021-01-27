package main

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"strings"
	"unicode"
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

func Find(array []string, val string) bool {
	for _, item := range array {
		if item == val {
			return true
		}
	}
	return false
}

func Usage(cmd *cobra.Command) error {
	if cmd == nil {
		return fmt.Errorf("nil command")
	}

	usage := []string{fmt.Sprintf("Usage: %s", cmd.UseLine())}
	cmdPath := cmd.CommandPath()
	groups := []string{"cluster", "hostgroup"}

	if cmdPath == "wekactl" {
		usage = append(usage, "\nGroups:")
		for _, subCommand := range cmd.Commands() {
			if Find(groups, subCommand.Name()) {
				usage = append(usage, fmt.Sprintf("  %s %-30s  %s", cmd.CommandPath(), subCommand.Name(), subCommand.Short))
			}
		}
	}

	usage = append(usage, "\nCommands:")
	for _, subCommand := range cmd.Commands() {
		if !subCommand.Hidden && !Find(groups, subCommand.Name()) {
			usage = append(usage, fmt.Sprintf("  %s %-30s  %s", cmd.CommandPath(), subCommand.Name(), subCommand.Short))
		}
	}

	if len(cmd.Aliases) > 0 {
		usage = append(usage, "\nAliases: "+cmd.NameAndAliases())
	}

	usage = append(usage, "\nCommon flags:")
	if len(cmd.PersistentFlags().FlagUsages()) != 0 {
		usage = append(usage, strings.TrimRightFunc(cmd.PersistentFlags().FlagUsages(), unicode.IsSpace))
	}
	if len(cmd.InheritedFlags().FlagUsages()) != 0 {
		usage = append(usage, strings.TrimRightFunc(cmd.InheritedFlags().FlagUsages(), unicode.IsSpace))
	}

	if cmdPath == "wekactl" {
		cmdPath += " [group]"
	} else {
		cmdPath += " [command]"
	}
	usage = append(usage, fmt.Sprintf("\nUse '%s --help' for more information about a command.\n", cmdPath))

	cmd.Println(strings.Join(usage, "\n"))

	return nil
}

func init() {
	rootCmd.AddCommand(cluster.Cluster)
	rootCmd.AddCommand(hostgroup.HostGroup)
	rootCmd.AddCommand(aws.AWS)
	rootCmd.AddCommand(debug.Debug)
	rootCmd.AddCommand(version.Version)

	rootCmd.PersistentFlags().BoolP("help", "h", false, "help for this command")
	rootCmd.PersistentFlags().StringVarP(&env.Config.Provider, "provider", "c", "aws", "Cloud provider")
	rootCmd.PersistentFlags().StringVarP(&env.Config.Region, "region", "r", "", "Region")
	rootCmd.SetUsageFunc(Usage)
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
