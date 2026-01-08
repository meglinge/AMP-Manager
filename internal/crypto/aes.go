package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var ErrEncryptionKeyNotSet = errors.New("encryption key not configured")

func Encrypt(plaintext []byte, key []byte) (string, error) {
	if len(key) == 0 {
		return "", ErrEncryptionKeyNotSet
	}

	block, err := aes.NewCipher(key)
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

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(ciphertext string, key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, ErrEncryptionKeyNotSet
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(data) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertextBytes, nil)
}

func IsEncrypted(value string) bool {
	if value == "" {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return false
	}
	return len(value) > 50
}
