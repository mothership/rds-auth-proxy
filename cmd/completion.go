package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generates CLI completion scripts",
	Long: `To load completion in bash run

	echo '. <(rds-auth-proxy completion bash)' >> ~/.bash_profile
	source ~/.bash_profile

Run 'rds-auth-proxy completion bash --help' for more information.


To load completion in zsh, run the following command to generate a completion file:

	rds-auth-proxy completion zsh > _rds-auth-proxy


Then move this file somewhere along your $fpath and source your ~/.zshrc again. Run 'rds-auth-proxy completion zsh --help' for more information.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := rootCmd.GenBashCompletion(os.Stdout)
		if err != nil {
			return err
		}
		return rootCmd.GenZshCompletion(os.Stdout)
	},
}

var bashCompletionCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generates bash completion scripts",
	Long: `To load completion in bash run

	echo '. <(rds-auth-proxy completion bash)' >> ~/.bash_profile
	source ~/.bash_profile

If this gives you trouble, ensure you're not using the version of bash bundled 
with OSX (OSX bundles 3.2, Homebrew bundles version 5).
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenBashCompletion(os.Stdout)
	},
}

var zshCompletionCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generates zsh completion scripts",
	Long: `To load completion in ZSH, run the following command to generate a completion file:

	rds-auth-proxy completion zsh > _rds-auth-proxy

Then move this file somewhere along your $fpath and source your ~/.zshrc again. If you're 
still not getting completion behavior, ensure you have the following in your ~/.zshrc

	autoload -U compaudit && compinit

If you do and it's still not working, try removing the completion cache:

	rm ~/.zcompdump* && source ~/.zshrc
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(os.Stdout)
	},
}

func init() {
	completionCmd.AddCommand(bashCompletionCmd)
	completionCmd.AddCommand(zshCompletionCmd)
	rootCmd.AddCommand(completionCmd)
}
