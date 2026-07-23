/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas CARPi <Deltablot>
 * @author Moustapha Camara <Deltablot>
 * @copyright 2026 Nicolas CARPi
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const AppName = "elabftw-desktop"

type ProfileIndex struct {
	Version  int            `json:"version"`
	Profiles []ProfileEntry `json:"profiles"`
}

type ProfileEntry struct {
	UUID        string `json:"uuid"`
	DisplayName string `json:"display_name,omitempty"`
	CreatedAt   string `json:"created_at"`

	// Salt is a random per-profile salt used with the user's passphrase
	// to derive the symmetric encryption key. It is not secret.
	Salt string `json:"salt,omitempty"`

	// small encrypted value used to check that the passphrase-derived key is correct when unlocking a profile.
	EncryptedVerifier string `json:"encrypted_verifier,omitempty"`
}

// either XDG_DATA_HOME or default to .local/share
func appRootDir() (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("os.UserHomeDir: %w", err)
		}
		base = filepath.Join(home, ".local", "share")
	}

	return filepath.Join(base, AppName), nil
}

func profileUploadsDir(uuid string) (string, error) {
	dir, err := profileDir(uuid)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "files"), nil
}

func profilesDir() (string, error) {
	root, err := appRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "profiles"), nil
}

func profileDir(uuid string) (string, error) {
	if uuid == "" {
		return "", errors.New("uuid is empty")
	}
	pdir, err := profilesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(pdir, uuid), nil
}

func ensureAppDirs() (string, error) {
	root, err := appRootDir()
	if err != nil {
		return "", err
	}
	// 0700: user read/write/execute only
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", fmt.Errorf("mkdir app root: %w", err)
	}

	pdir := filepath.Join(root, "profiles")
	if err := os.MkdirAll(pdir, 0o700); err != nil {
		return "", fmt.Errorf("mkdir profiles: %w", err)
	}

	return root, nil
}

func indexPath() (string, error) {
	root, err := appRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "index.json"), nil
}

func loadProfileIndex() (*ProfileIndex, error) {
	if _, err := ensureAppDirs(); err != nil {
		return nil, err
	}

	path, err := indexPath()
	if err != nil {
		return nil, err
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &ProfileIndex{Version: 1, Profiles: []ProfileEntry{}}, nil
		}
		return nil, fmt.Errorf("read index: %w", err)
	}

	var idx ProfileIndex
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, fmt.Errorf("Parse index.json: %w", err)
	}
	if idx.Version == 0 {
		idx.Version = 1
	}
	if idx.Profiles == nil {
		idx.Profiles = []ProfileEntry{}
	}

	return &idx, nil
}

func saveProfileIndex(idx *ProfileIndex) error {
	if idx == nil {
		return errors.New("idx is nil")
	}
	if _, err := ensureAppDirs(); err != nil {
		return err
	}

	path, err := indexPath()
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}

	// 0600: user read/write only
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write tmp index: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename index: %w", err)
	}

	return nil
}

func ensureProfileDir(uuid string) (string, error) {
	dir, err := profileDir(uuid)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("mkdir profile dir: %w", err)
	}
	return dir, nil
}

// Example: creates profiles/<uuid>/meta.json with restrictive permissions.
func writeProfileMetaFile(uuid string, content []byte) (string, error) {
	dir, err := ensureProfileDir(uuid)
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, "meta.json")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return "", fmt.Errorf("write meta.json: %w", err)
	}
	return path, nil
}

func encryptedProfileUploadPath(uuid string, hash string) (string, error) {
	if len(hash) < 3 {
		return "", fmt.Errorf("hash is too short")
	}

	filesDir, err := profileUploadsDir(uuid)
	if err != nil {
		return "", err
	}

	dir := filepath.Join(filesDir, hash[:3])
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("mkdir upload hash dir: %w", err)
	}

	return filepath.Join(dir, hash), nil
}
