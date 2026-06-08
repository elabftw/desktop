/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas <Deltablot>
 * @author Moustapha <Deltablot>
 * @copyright 2026 Deltablot
 * @see https://www.elabftw.net
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

// This file contains local profile encryption helpers.
//
// The user's passphrase is never stored directly. When a profile is created,
// we derive a symmetric master key from the passphrase using Argon2id and a
// random per-profile salt.
//
// Unlocking a profile means deriving the same master key again from the
// passphrase and stored salt. Entry decryption is authenticated (aead), so using the
// wrong passphrase-derived key will fail when decrypting existing data.
//
// Entry title/body encryption uses the active passphrase-derived key.
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

	// Argon2id parameters for deriving a profile encryption key from the user's passphrase.
	//
	// argonMemory is in KiB, so 64 * 1024 means 64 MiB.
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeySize = chacha20poly1305.KeySize
)

const profileVerifierPlaintext = "elabftw-desktop profile verifier v1"

var base64Encoding = base64.RawStdEncoding

type profileCryptoParams struct {
	Salt              string
	EncryptedVerifier string
}

func randomBytes(size int) ([]byte, error) {
	out := make([]byte, size)
	if _, err := rand.Read(out); err != nil {
		return nil, fmt.Errorf("Random bytes: %w", err)
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
		return "", fmt.Errorf("Create cipher: %w", err)
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
	// The encrypted payload is stored as base64(nonce || ciphertext).
	// Split the nonce back out before opening the ciphertext.
	payload, err := base64Encoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("Decode ciphertext: %w", err)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("Create cipher: %w", err)
	}

	nonceSize := aead.NonceSize()
	if len(payload) < nonceSize {
		return nil, fmt.Errorf("Ciphertext is too short")
	}

	nonce := payload[:nonceSize]
	ciphertext := payload[nonceSize:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("Decrypt: %w", err)
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

func createProfileCryptoParams(passphrase string) (*profileCryptoParams, error) {
	salt, err := randomBytes(profileSaltSize)
	if err != nil {
		return nil, err
	}

	key := deriveProfileKey(passphrase, salt)
	defer zeroBytes(key)

	encryptedVerifier, err := encryptString(key, profileVerifierPlaintext)
	if err != nil {
		return nil, fmt.Errorf("Encrypt verifier: %w", err)
	}

	return &profileCryptoParams{
		Salt:              base64Encoding.EncodeToString(salt),
		EncryptedVerifier: encryptedVerifier,
	}, nil
}

func unlockProfileCryptoParams(profile *ProfileEntry, passphrase string) ([]byte, error) {
	if profile == nil {
		return nil, fmt.Errorf("Profile is nil")
	}
	if profile.Salt == "" || profile.EncryptedVerifier == "" {
		return nil, fmt.Errorf("Profile has no encrypted key metadata")
	}

	salt, err := base64Encoding.DecodeString(profile.Salt)
	if err != nil {
		return nil, fmt.Errorf("Decode salt: %w", err)
	}

	key := deriveProfileKey(passphrase, salt)

	verifier, err := decryptString(key, profile.EncryptedVerifier)
	if err != nil {
		zeroBytes(key)
		return nil, fmt.Errorf("Invalid passphrase")
	}

	if verifier != profileVerifierPlaintext {
		zeroBytes(key)
		return nil, fmt.Errorf("Invalid passphrase")
	}

	return key, nil
}
