package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version string
var commit string
var date string

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Displays the current version",
	Long:  `Displays the current version of the rds-auth-proxy binary`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("rds-auth-proxy: %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("date: %s\n", date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
