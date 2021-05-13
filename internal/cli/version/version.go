package version

import (
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/env"
)

var Version = &cobra.Command{
	Use:   "version",
	Short: "Version",
	RunE: func(c *cobra.Command, _ []string) error {
		versionInfo, err := env.GetBuildVersion()
		if err != nil {
			return err
		}
		fmt.Printf("%s\n%s\n", versionInfo.BuildVersion, versionInfo.Commit)
		return nil
	},
	SilenceUsage: true,
}
