package version

import (
	"fmt"
	"github.com/spf13/cobra"
	"os/exec"
	"strings"
)

var BuildVersion string

var Version = &cobra.Command{
	Use:   "version",
	Short: "Version",
	RunE: func(c *cobra.Command, _ []string) error {
		if BuildVersion == "" {
			out, err := exec.Command("./scripts/get_version.sh").Output()
			if err != nil {
				return err
			}
			BuildVersion = strings.TrimSuffix(string(out), "\n")
		}
		fmt.Println(BuildVersion)
		return nil
	},
	SilenceUsage: true,
}
