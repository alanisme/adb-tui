package main

import (
	"os"

	"github.com/spf13/cobra"
)

func completionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for adb-tui.

To load completions:

Bash:
  $ source <(adb-tui completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ adb-tui completion bash > /etc/bash_completion.d/adb-tui
  # macOS:
  $ adb-tui completion bash > $(brew --prefix)/etc/bash_completion.d/adb-tui

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ adb-tui completion zsh > "${fpath[1]}/_adb-tui"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ adb-tui completion fish | source

  # To load completions for each session, execute once:
  $ adb-tui completion fish > ~/.config/fish/completions/adb-tui.fish

PowerShell:
  PS> adb-tui completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> adb-tui completion powershell > adb-tui.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return nil
			}
		},
	}

	return cmd
}
