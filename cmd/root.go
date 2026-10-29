package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// version is the current envy release. Bump this when cutting a new tag.
const version = "v0.1.0"

var rootCmd = &cobra.Command{
	Use:     "envy",
	Short:   "Secure environment variable manager backed by Cloudflare R2",
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate("envy {{.Version}}\n")
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(versionCmd)
}
