package cmd

import (
	"fmt"
	"strings"

	"github.com/pradeepbgs/envy/internal/config"
	"github.com/pradeepbgs/envy/internal/storage"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all envs stored in your bucket",
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	// load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config:  %w", err)
	}

	// load r2
	r2 := storage.New(cfg)
	keys, err := r2.List(cmd.Context())
	if err != nil {
		return fmt.Errorf("Error listing envs %w", err)
	}

	if len(keys) == 0 {
		fmt.Println("No env sets found.")
		return nil
	}

	fmt.Println("Available env sets:")
	for _, k := range keys {
		fmt.Printf("  %s\n", strings.TrimSuffix(k, ".enc"))
	}

	return nil
}
