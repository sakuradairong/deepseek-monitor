package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

var encryptionKey []byte
var encryptionEnabled bool

func init() {
	key := os.Getenv("DB_ENCRYPTION_KEY")
	if key == "" {
		return // No encryption configured — keys stored in plaintext
	}

	// Derive 32-byte key from any-length secret using SHA-256
	hash := sha256.Sum256([]byte(key))
	encryptionKey = hash[:]
	encryptionEnabled = true
}

// Encrypt encrypts plaintext with AES-256-GCM. Returns base64 of (nonce + ciphertext).
// If no encryption key is configured, returns the plaintext as-is.
func Encrypt(plaintext string) (string, error) {
	if !encryptionEnabled || plaintext == "" {
		return plaintext, nil
	}

	block, err := aes.NewCipher(encryptionKey)
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

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

// Decrypt decrypts AES-256-GCM ciphertext (base64 of nonce+ciphertext).
// If no encryption key is configured, returns the ciphertext as-is.
func Decrypt(cipherB64 string) (string, error) {
	if !encryptionEnabled || cipherB64 == "" {
		return cipherB64, nil
	}

	data, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// HasEncryption returns true if DB encryption is configured
func HasEncryption() bool {
	return encryptionEnabled
}
