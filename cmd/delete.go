package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pradeepbgs/envy/internal/config"
	"github.com/pradeepbgs/envy/internal/storage"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a key with .env file within it.",
	Args: cobra.ExactArgs(1),
	RunE:  runDelete,
}

func runDelete(cmd *cobra.Command, args []string) error {
	name := args[0]
	// Load the config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Error loading config %w", err)
	}

	// ask user for y or n
	fmt.Printf("Delete '%s' from R2? [y/N]: ", name)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(answer)) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	// r2
	r2 := storage.New(cfg)
	if err := r2.Delete(cmd.Context(),name+".enc"); err != nil {
		return fmt.Errorf("Error during deletion %w",err)
	}

	fmt.Printf("Deleted '%s'\n", name+".enc")
	return nil
}
