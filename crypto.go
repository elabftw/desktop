// This file contains local profile encryption helpers.
//
// The user's passphrase is never stored directly. When a profile is created,
// we derive a symmetric key from the passphrase using Argon2id and a random
// per-profile salt.
//
// Unlocking a profile means deriving the same key again and successfully
// decrypting a small encrypted verifier stored in the profile index. If
// decryption fails, the passphrase is wrong.
//
// Entry title/body encryption also uses the active passphrase-derived key.
// The SQLite database remains a normal SQLite file, but sensitive entry content
// is encrypted before being written.

package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"           // produce a 32-byte key xchacha can use
	"golang.org/x/crypto/chacha20poly1305" // encrypt and authenticate
)

const (
	profileSaltSize = 16

	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeySize = chacha20poly1305.KeySize

	profileVerifierPlaintext = "elabftw-desktop profile verifier v1"
)

var base64Encoding = base64.RawStdEncoding

type profileSecrets struct {
	Salt              string
	EncryptedVerifier string
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
	// XChaCha20-Poly1305 requires a unique nonce for each encryption.
	// The nonce is not secret, so we store it next to the ciphertext.
	// Stored format:
	//   base64(nonce || ciphertext)
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
	// The encrypted payload is stored as base64(nonce || ciphertext)
	// Split the nonce back out before opening the ciphertext
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
	salt, err := randomBytes(profileSaltSize)
	if err != nil {
		return nil, err
	}

	key := deriveProfileKey(passphrase, salt)
	defer zeroBytes(key)

	encryptedVerifier, err := encryptBytes(key, []byte(profileVerifierPlaintext))
	if err != nil {
		return nil, fmt.Errorf("encrypt verifier: %w", err)
	}

	return &profileSecrets{
		Salt:              base64Encoding.EncodeToString(salt),
		EncryptedVerifier: encryptedVerifier,
	}, nil
}

func unlockProfileSecrets(profile *ProfileEntry, passphrase string) ([]byte, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile is nil")
	}
	if profile.Salt == "" || profile.EncryptedVerifier == "" {
		return nil, fmt.Errorf("profile has no encrypted key metadata")
	}

	salt, err := base64Encoding.DecodeString(profile.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}

	key := deriveProfileKey(passphrase, salt)

	verifier, err := decryptBytes(key, profile.EncryptedVerifier)
	if err != nil {
		zeroBytes(key)
		return nil, fmt.Errorf("invalid passphrase")
	}

	if string(verifier) != profileVerifierPlaintext {
		zeroBytes(key)
		return nil, fmt.Errorf("invalid passphrase")
	}

	return key, nil
}
