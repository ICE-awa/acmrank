package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
)

const (
	nonceSize       = 12
	ciphertextV1    = 1
	ciphertextMinV1 = 1 + nonceSize + 16
)

type AEAD struct {
	aead cipher.AEAD
}

func NewAEAD(secret string) (*AEAD, error) {
	if secret == "" {
		return nil, errors.New("secret key is required")
	}

	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("new aes cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm cipher: %w", err)
	}

	return &AEAD{aead: aead}, nil
}

func (c *AEAD) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	sealed := c.aead.Seal(nil, nonce, plaintext, nil)
	result := make([]byte, 1+len(nonce)+len(sealed))
	result[0] = ciphertextV1
	copy(result[1:], nonce)
	copy(result[1+len(nonce):], sealed)
	return result, nil
}

func (c *AEAD) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < ciphertextMinV1 {
		return nil, errors.New("ciphertext is too short")
	}
	if ciphertext[0] != ciphertextV1 {
		return nil, errors.New("unsupported ciphertext version")
	}

	nonce := ciphertext[1 : 1+nonceSize]
	payload := ciphertext[1+nonceSize:]
	plaintext, err := c.aead.Open(nil, nonce, payload, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt payload: %w", err)
	}

	return plaintext, nil
}
