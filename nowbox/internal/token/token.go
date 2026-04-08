package token

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

// 32-byte key = AES-256. Generate a new one with: openssl rand -hex 32
// This is compiled into the binary. It prevents casual extraction of keys
// from .now files but is not unbreakable — someone reversing the binary can
// find it. Good enough for demo keys and rate-limited API tokens.
//
// Set at build time with: go build -ldflags "-X github.com/nowbox/nowbox/internal/token.keyHex=..."
var keyHex = "0000000000000000000000000000000000000000000000000000000000000000"

func deriveKey() ([]byte, error) {
	if len(keyHex) != 64 {
		return nil, fmt.Errorf("invalid token key length")
	}
	key := make([]byte, 32)
	for i := 0; i < 32; i++ {
		_, err := fmt.Sscanf(keyHex[i*2:i*2+2], "%02x", &key[i])
		if err != nil {
			return nil, fmt.Errorf("invalid token key: %w", err)
		}
	}
	return key, nil
}

// Payload is the decrypted content of a .now token.
type Payload struct {
	Host       string            `json:"host"`
	Agent      string            `json:"agent"`
	Vars       map[string]string `json:"vars"`
	InstanceID string            `json:"instance_id,omitempty"`
}

// Seal encrypts a Payload into a base64 string using AES-256-GCM.
func Seal(p *Payload) (string, error) {
	plaintext, err := json.Marshal(p)
	if err != nil {
		return "", err
	}

	key, err := deriveKey()
	if err != nil {
		return "", err
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

// Open decrypts a base64 token string back into a Payload.
func Open(tokenStr string) (*Payload, error) {
	data, err := base64.StdEncoding.DecodeString(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("invalid token encoding: %w", err)
	}

	key, err := deriveKey()
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

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("token too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("token decryption failed — file may be corrupted or from a different version")
	}

	var p Payload
	if err := json.Unmarshal(plaintext, &p); err != nil {
		return nil, fmt.Errorf("invalid token payload: %w", err)
	}
	return &p, nil
}
