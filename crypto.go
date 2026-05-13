// This file contains local profile encryption helpers.
//
// The user's passphrase is never stored directly. When a profile is created,
// we derive a symmetric key from the passphrase using Argon2id and use that key
// to encrypt the generated Ed25519 private key.
//
// Unlocking a profile means deriving the same key again and successfully
// decrypting the encrypted private key. If decryption fails, the passphrase is
// wrong.
//
// Entry title/body encryption also uses the active passphrase-derived key.
// The SQLite database remains a normal SQLite file, but sensitive entry content
// is encrypted before being written.

package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"           // produce a 32-byte key xchacha can use
	"golang.org/x/crypto/chacha20poly1305" // encrypt and authenticate
)

const (
	profileSaltSize = 16

	// Argon2id parameters for deriving a profile encryption key from the user's passphrase.
	//
	// argonMemory is in KiB, so 64 * 1024 means 64 MiB.
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeySize = chacha20poly1305.KeySize
)

var base64Encoding = base64.RawStdEncoding

type profileCryptoParams struct {
	PublicKey           string
	Salt                string
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

func profilePrivateKeyAAD(profileUUID string) []byte {
	return []byte("elabftw-desktop profile private key v1:" + profileUUID)
}

func encryptBytes(key []byte, plaintext []byte) (string, error) {
	return encryptBytesWithAAD(key, plaintext, nil)
}

func encryptBytesWithAAD(key []byte, plaintext []byte, additionalData []byte) (string, error) {
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

	ciphertext := aead.Seal(nil, nonce, plaintext, additionalData)

	payload := make([]byte, 0, len(nonce)+len(ciphertext))
	payload = append(payload, nonce...)
	payload = append(payload, ciphertext...)

	return base64Encoding.EncodeToString(payload), nil
}

func decryptBytes(key []byte, encoded string) ([]byte, error) {
	return decryptBytesWithAAD(key, encoded, nil)
}

func decryptBytesWithAAD(key []byte, encoded string, additionalData []byte) ([]byte, error) {
	// The encrypted payload is stored as base64(nonce || ciphertext).
	// Split the nonce back out before opening the ciphertext.
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

	plaintext, err := aead.Open(nil, nonce, ciphertext, additionalData)
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

func createProfileCryptoParams(profileUUID string, passphrase string) (*profileCryptoParams, error) {
	if profileUUID == "" {
		return nil, fmt.Errorf("profile uuid is empty")
	}

	// Each profile gets its own Ed25519 identity keypair.
	// The public key is stored as metadata; the private key is encrypted
	// with a key derived from the user's passphrase.
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 keypair: %w", err)
	}
	defer zeroBytes(privateKey)

	salt, err := randomBytes(profileSaltSize)
	if err != nil {
		return nil, err
	}

	key := deriveProfileKey(passphrase, salt)
	defer zeroBytes(key)

	encryptedPrivateKey, err := encryptBytesWithAAD(
		key,
		privateKey,
		profilePrivateKeyAAD(profileUUID),
	)
	if err != nil {
		return nil, fmt.Errorf("encrypt private key: %w", err)
	}

	return &profileCryptoParams{
		PublicKey:           base64Encoding.EncodeToString(publicKey),
		Salt:                base64Encoding.EncodeToString(salt),
		EncryptedPrivateKey: encryptedPrivateKey,
	}, nil
}

func unlockProfileCryptoParams(profile *ProfileEntry, passphrase string) ([]byte, ed25519.PrivateKey, error) {
	// Re-derive profile key from the passphrase and stored salt.
	// If the passphrase is wrong, decryptBytesWithAAD will fail authentication.
	if profile == nil {
		return nil, nil, fmt.Errorf("profile is nil")
	}
	if profile.UUID == "" {
		return nil, nil, fmt.Errorf("profile uuid is empty")
	}
	if profile.PublicKey == "" || profile.Salt == "" || profile.EncryptedPrivateKey == "" {
		return nil, nil, fmt.Errorf("profile has no encrypted key metadata")
	}

	publicKey, err := base64Encoding.DecodeString(profile.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("decode public key: %w", err)
	}

	salt, err := base64Encoding.DecodeString(profile.Salt)
	if err != nil {
		return nil, nil, fmt.Errorf("decode salt: %w", err)
	}

	key := deriveProfileKey(passphrase, salt)

	privateKeyBytes, err := decryptBytesWithAAD(
		key,
		profile.EncryptedPrivateKey,
		profilePrivateKeyAAD(profile.UUID),
	)
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

	// Sanity-check that the decrypted private key matches the stored public key.
	// This detects corrupted metadata or accidental key mismatch.
	derivedPublicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok || !bytes.Equal(derivedPublicKey, publicKey) {
		zeroBytes(key)
		zeroBytes(privateKey)
		return nil, nil, fmt.Errorf("invalid profile key metadata")
	}

	return key, privateKey, nil
}
