package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "assh",
	Short: "aws command line utility",
	Long: `A fast and easy-to-use aws command line utility to interact
with aws resources created with love by jchenrev in Go.
Complete documentation is available at http://github.com/jchenrev/awstool`,
	Run: runSSH,
}

// Execute executes the root command
func Execute() {
	rootCmd.Execute()
}
