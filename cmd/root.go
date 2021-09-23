package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "rds-auth-proxy",
	Short: "rds-auth-proxy launches an SSL-capable postgres proxy",
}

// Execute kicks off the CLI
func Execute() error {
	return rootCmd.Execute()
}
