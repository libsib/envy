package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pradeepbgs/envy/internal/config"
	"github.com/pradeepbgs/envy/internal/crypto"
	"github.com/pradeepbgs/envy/internal/storage"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <name> <envfile path>",
	Short: "Encrypt and upload a .env file to R2",
	Args:  cobra.ExactArgs(2),
	RunE:  runPush,
}

func runPush(cmd *cobra.Command, args []string) error {
	name := args[0]
	env_file, err := filepath.Abs(args[1])
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// load the config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config not found — run 'envy init' first: %w", err)
	}
	// create a key first?
	decrypted_key, err := config.Decodekey(cfg.Encryptionkey)
	if err != nil {
		return fmt.Errorf("invalid encryption key in config: %w", err)
	}

	// read the file
	data, err := os.ReadFile(env_file)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", env_file, err)
	}

	// encrypt the file
	encrypted, err := crypto.Encrypt(data, decrypted_key)
	if err != nil {
		return fmt.Errorf("failed to encrypt %s: %w", env_file, err)
	}

	// upload to r2
	r2 := storage.New(cfg)
	if err := r2.Upload(cmd.Context(), name+".enc", encrypted); err != nil {
		return fmt.Errorf("failed to upload to R2: %w", err)
	}

	fmt.Printf("Successfully uploaded %s to R2 as %s\n", env_file, name+".enc")
	return nil
}
