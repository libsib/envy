package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pradeepbgs/envy/internal/config"
	"github.com/pradeepbgs/envy/internal/crypto"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	keyExportPassphrase bool
	keyImportForce      bool
)

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Export or import the encryption key (for multi-machine setup)",
	Long: "Move the envy encryption key between machines without copying your R2 credentials.\n\n" +
		"On machine A:  envy key export > mykey.txt\n" +
		"On machine B:  envy key import \"$(cat mykey.txt)\"\n\n" +
		"Use --passphrase to wrap the key with a passphrase before it leaves the machine.",
}

var keyExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Print the encryption key so it can be moved to another machine",
	Args:  cobra.NoArgs,
	RunE:  runKeyExport,
}

var keyImportCmd = &cobra.Command{
	Use:   "import [key]",
	Short: "Set the encryption key on this machine from an exported value",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runKeyImport,
}

func init() {
	keyExportCmd.Flags().BoolVar(&keyExportPassphrase, "passphrase", false, "Wrap the key with a passphrase before printing")
	keyImportCmd.Flags().BoolVarP(&keyImportForce, "force", "f", false, "Overwrite an existing encryption key without prompting")
	keyCmd.AddCommand(keyExportCmd)
	keyCmd.AddCommand(keyImportCmd)
}

func runKeyExport(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config not found — run 'envy init' first: %w", err)
	}
	if cfg.Encryptionkey == "" {
		return fmt.Errorf("no encryption key found in config")
	}

	if !keyExportPassphrase {
		// Raw base64 key. Warn on stderr so stdout stays clean for piping.
		fmt.Fprintln(os.Stderr, "WARNING: exporting the key in plaintext. Keep it secret; anyone with it can decrypt your envs.")
		fmt.Fprintln(os.Stderr, "Tip: use --passphrase to wrap it before transport.")
		fmt.Println(cfg.Encryptionkey)
		return nil
	}

	rawKey, err := config.Decodekey(cfg.Encryptionkey)
	if err != nil {
		return fmt.Errorf("invalid encryption key in config: %w", err)
	}
	pass, err := readPassphrase("Passphrase to wrap the key: ", true)
	if err != nil {
		return err
	}
	token, err := crypto.WrapKey(rawKey, pass)
	if err != nil {
		return fmt.Errorf("failed to wrap key: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Wrapped key (import it on the other machine with 'envy key import'):")
	fmt.Println(token)
	return nil
}

func runKeyImport(cmd *cobra.Command, args []string) error {
	var value string
	if len(args) == 1 {
		value = strings.TrimSpace(args[0])
	} else {
		v, err := readLine("Paste exported key: ")
		if err != nil {
			return err
		}
		value = strings.TrimSpace(v)
	}
	if value == "" {
		return fmt.Errorf("no key value provided")
	}

	var encoded string
	if crypto.IsWrapped(value) {
		pass, err := readPassphrase("Passphrase to unwrap the key: ", false)
		if err != nil {
			return err
		}
		rawKey, err := crypto.UnwrapKey(value, pass)
		if err != nil {
			return err
		}
		encoded, err = config.EncodeKey(rawKey)
		if err != nil {
			return err
		}
	} else {
		// Treat as a plain base64 key; validate it decodes to 32 bytes.
		if _, err := config.Decodekey(value); err != nil {
			return fmt.Errorf("invalid key: %w", err)
		}
		encoded = value
	}

	cfg, err := config.LoadOrNew()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Encryptionkey != "" && cfg.Encryptionkey != encoded && !keyImportForce {
		fmt.Print("An encryption key already exists and differs. Overwrite? [y/N]: ")
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	cfg.Encryptionkey = encoded
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Println("Encryption key imported to ~/.envy/config.json")
	if cfg.R2Endpoint == "" {
		fmt.Println("Note: R2 credentials are not set yet — run 'envy init' to configure them.")
	}
	return nil
}

// readPassphrase reads a passphrase without echoing. When confirm is true it
// prompts twice and checks they match.
func readPassphrase(prompt string, confirm bool) ([]byte, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Non-interactive: read a line from stdin.
		line, err := readLine(prompt)
		if err != nil {
			return nil, err
		}
		return []byte(strings.TrimRight(line, "\r\n")), nil
	}
	fmt.Fprint(os.Stderr, prompt)
	pass, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, err
	}
	if len(pass) == 0 {
		return nil, fmt.Errorf("passphrase must not be empty")
	}
	if confirm {
		fmt.Fprint(os.Stderr, "Confirm passphrase: ")
		again, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return nil, err
		}
		if string(pass) != string(again) {
			return nil, fmt.Errorf("passphrases do not match")
		}
	}
	return pass, nil
}

func readLine(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	return bufio.NewReader(os.Stdin).ReadString('\n')
}
