package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

// Encrypt encrypts plaintext using AES-GCM with a random nonce.
// Returns nonce (12 bytes) prepended to the ciphertext.
func Encrypt(plaintext []byte, key [32]byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

// Decrypt decrypts ciphertext (with prepended nonce) using AES-GCM.
func Decrypt(ciphertext []byte, key [32]byte) ([]byte, error) {
	if len(ciphertext) < 12 {
		return nil, errors.New("ciphertext too short")
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := ciphertext[:12]
	ciphertext = ciphertext[12:]

	return gcm.Open(nil, nonce, ciphertext, nil)
}
