package main

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

$ source <(wekactl completion bash)

# To load completions for each session, execute once:
Linux:
  $ wekactl completion bash > /etc/bash_completion.d/wekactl
MacOS:
  $ wekactl completion bash > /usr/local/etc/bash_completion.d/wekactl

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ wekactl completion zsh > "${fpath[1]}/_wekactl"

# You will need to start a new shell for this setup to take effect.

Fish:

$ wekactl completion fish | source

# To load completions for each session, execute once:
$ wekactl completion fish > ~/.config/fish/completions/wekactl.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletion(os.Stdout)
		default:
			return errors.New(fmt.Sprintf("autocompletion for %s not supported", args[0]))
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
