package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/scrypt"
)

// tokenPrefix marks a passphrase-wrapped key token so it can be told apart
// from a plain base64-encoded key.
const tokenPrefix = "envy-key:"

type wrappedKey struct {
	Version    int    `json:"version"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// scrypt parameters (interactive-strength).
const (
	scryptN = 1 << 15
	scryptR = 8
	scryptP = 1
)

func deriveKey(passphrase, salt []byte) ([]byte, error) {
	return scrypt.Key(passphrase, salt, scryptN, scryptR, scryptP, 32)
}

// WrapKey encrypts a raw key with a passphrase and returns a transportable
// token string prefixed with tokenPrefix.
func WrapKey(key, passphrase []byte) (string, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}
	dk, err := deriveKey(passphrase, salt)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(dk)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, key, nil)

	payload := wrappedKey{
		Version:    1,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return tokenPrefix + base64.StdEncoding.EncodeToString(raw), nil
}

// IsWrapped reports whether a value looks like a passphrase-wrapped token.
func IsWrapped(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), tokenPrefix)
}

// UnwrapKey decrypts a passphrase-wrapped token back into the raw key.
func UnwrapKey(token string, passphrase []byte) ([]byte, error) {
	token = strings.TrimSpace(token)
	if !IsWrapped(token) {
		return nil, fmt.Errorf("not a wrapped key token")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(token, tokenPrefix))
	if err != nil {
		return nil, fmt.Errorf("malformed token: %w", err)
	}
	var payload wrappedKey
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("malformed token: %w", err)
	}
	salt, err := base64.StdEncoding.DecodeString(payload.Salt)
	if err != nil {
		return nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(payload.Nonce)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return nil, err
	}
	dk, err := deriveKey(passphrase, salt)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(dk)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	key, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("wrong passphrase or corrupted token")
	}
	return key, nil
}
