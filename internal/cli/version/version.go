package version

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"strings"
)

var BuildVersion string
var Commit string

var Version = &cobra.Command{
	Use:   "version",
	Short: "Version",
	RunE: func(c *cobra.Command, _ []string) error {
		if BuildVersion == "" {
			os.Setenv("WEKACTL_FORCE_DEV", "1")
			out, err := exec.Command("./scripts/get_build_params.sh").Output()
			if err != nil {
				return err
			}
			BuildVersion = strings.TrimSuffix(string(out), "\n")
		}
		fmt.Printf("%s\n%s\n", BuildVersion, Commit)
		return nil
	},
	SilenceUsage: true,
}
