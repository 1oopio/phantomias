package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stratumfarm/phantomias/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\n", version.Version)
		fmt.Printf("Commit: %s\n", version.Commit)
		fmt.Printf("Date: %s\n", version.Date)
		fmt.Printf("Build by: %s\n", version.BuiltBy)
	},
}
