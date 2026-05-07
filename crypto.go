package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	profileKeySaltSize = 16

	// Argon2id parameters.
	// Memory is in KiB, so 64 * 1024 = 64 MiB.
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeySize = chacha20poly1305.KeySize
)

var base64Encoding = base64.RawStdEncoding

type profileSecrets struct {
	PublicKey           string
	KeySalt             string
	EncryptedPrivateKey string
}

func randomBytes(size int) ([]byte, error) {
	out := make([]byte, size)
	if _, err := rand.Read(out); err != nil {
		return nil, fmt.Errorf("random bytes: %w", err)
	}
	return out, nil
}

func deriveProfileKey(passphrase string, salt []byte) []byte {
	return argon2.IDKey(
		[]byte(passphrase),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeySize,
	)
}

func encryptBytes(key []byte, plaintext []byte) (string, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	nonce, err := randomBytes(aead.NonceSize())
	if err != nil {
		return "", err
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	payload := make([]byte, 0, len(nonce)+len(ciphertext))
	payload = append(payload, nonce...)
	payload = append(payload, ciphertext...)

	return base64Encoding.EncodeToString(payload), nil
}

func decryptBytes(key []byte, encoded string) ([]byte, error) {
	payload, err := base64Encoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	nonceSize := aead.NonceSize()
	if len(payload) < nonceSize {
		return nil, fmt.Errorf("ciphertext is too short")
	}

	nonce := payload[:nonceSize]
	ciphertext := payload[nonceSize:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

func encryptString(key []byte, plaintext string) (string, error) {
	return encryptBytes(key, []byte(plaintext))
}

func decryptString(key []byte, encoded string) (string, error) {
	plaintext, err := decryptBytes(key, encoded)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func createProfileSecrets(passphrase string) (*profileSecrets, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 keypair: %w", err)
	}

	keySalt, err := randomBytes(profileKeySaltSize)
	if err != nil {
		return nil, err
	}

	key := deriveProfileKey(passphrase, keySalt)
	defer zeroBytes(key)

	encryptedPrivateKey, err := encryptBytes(key, privateKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt private key: %w", err)
	}

	return &profileSecrets{
		PublicKey:           base64Encoding.EncodeToString(publicKey),
		KeySalt:             base64Encoding.EncodeToString(keySalt),
		EncryptedPrivateKey: encryptedPrivateKey,
	}, nil
}

func unlockProfileSecrets(profile *ProfileEntry, passphrase string) ([]byte, ed25519.PrivateKey, error) {
	if profile == nil {
		return nil, nil, fmt.Errorf("profile is nil")
	}
	if profile.PublicKey == "" || profile.KeySalt == "" || profile.EncryptedPrivateKey == "" {
		return nil, nil, fmt.Errorf("profile has no encrypted key metadata")
	}

	publicKey, err := base64Encoding.DecodeString(profile.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("decode public key: %w", err)
	}

	keySalt, err := base64Encoding.DecodeString(profile.KeySalt)
	if err != nil {
		return nil, nil, fmt.Errorf("decode key salt: %w", err)
	}

	key := deriveProfileKey(passphrase, keySalt)

	privateKeyBytes, err := decryptBytes(key, profile.EncryptedPrivateKey)
	if err != nil {
		zeroBytes(key)
		return nil, nil, fmt.Errorf("invalid passphrase")
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)
	if len(privateKey) != ed25519.PrivateKeySize {
		zeroBytes(key)
		zeroBytes(privateKey)
		return nil, nil, fmt.Errorf("invalid private key size")
	}

	derivedPublicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok || !bytes.Equal(derivedPublicKey, publicKey) {
		zeroBytes(key)
		zeroBytes(privateKey)
		return nil, nil, fmt.Errorf("invalid profile key metadata")
	}

	return key, privateKey, nil
}
