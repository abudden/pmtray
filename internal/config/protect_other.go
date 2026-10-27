//go:build !windows

package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
)

// key returns a random key stored in a file readable only by the
// current user. This only protects against casual inspection of the
// settings file.
func key() ([]byte, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "key")
	k, err := os.ReadFile(path)
	if err == nil && len(k) == 32 {
		return k, nil
	}
	k = make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	return k, os.WriteFile(path, k, 0o600)
}

func gcm() (cipher.AEAD, error) {
	k, err := key()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func protect(data []byte) ([]byte, error) {
	g, err := gcm()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, g.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return g.Seal(nonce, nonce, data, nil), nil
}

func unprotect(data []byte) ([]byte, error) {
	g, err := gcm()
	if err != nil {
		return nil, err
	}
	if len(data) < g.NonceSize() {
		return nil, errors.New("stored password too short")
	}
	return g.Open(nil, data[:g.NonceSize()], data[g.NonceSize():], nil)
}
