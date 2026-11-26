// Package secret seals the few values the base must give back, not only check (the invitation links), with the
// server's key CALLBACK_SECRET (AES-256-GCM): a stolen base or backup does not give them away.
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"github.com/cyril-fl/callback/config"
)

var ErrNoKey = errors.New(config.Env("SECRET") + " is not set: openssl rand -base64 32")

// key reads the environment each time: cheap next to a request, and tests change it.
func key() (cipher.AEAD, error) {
	value := os.Getenv(config.Env("SECRET"))
	if value == "" {
		return nil, ErrNoKey
	}
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(raw) != 32 {
		return nil, fmt.Errorf("%s must be 32 random bytes in base64: openssl rand -base64 32", config.Env("SECRET"))
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// Check tells a malformed key, so the server refuses to start on it; no key at all is ErrNoKey.
func Check() error {
	_, err := key()
	return err
}

// Seal encrypts plain, in base64.
func Seal(plain string) (string, error) {
	gcm, err := key()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil)), nil
}

// Open decrypts what Seal gave.
func Open(sealed string) (string, error) {
	gcm, err := key()
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(sealed)
	if err != nil || len(raw) < gcm.NonceSize() {
		return "", errors.New("not a sealed value")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	return string(plain), err
}
