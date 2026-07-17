/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas CARPi <Deltablot>
 * @author Moustapha Camara <Deltablot>
 * @copyright 2026 Nicolas CARPi
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 *
 * Handle uploads locally
 */

package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type StoredUpload struct {
	ID            int64  `json:"id"`
	EntryID       int64  `json:"entryId"`
	RealName      string `json:"realName"`
	LongName      string `json:"longName"`
	Hash          string `json:"hash"`
	HashAlgorithm string `json:"hashAlgorithm"`
	Filesize      int64  `json:"filesize"`
	State         string `json:"state"`
	CreatedAt     string `json:"createdAt"`
	ModifiedAt    string `json:"modifiedAt"`
}

func fileSHA256(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func (a *App) ImportUpload(profileUUID string, entryID int64, sourcePath string) (*StoredUpload, error) {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return nil, err
	}

	if entryID <= 0 {
		return nil, fmt.Errorf("Invalid entry id")
	}

	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		return nil, fmt.Errorf("source path is empty")
	}

	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read source file: %w", err)
	}

	hash := fileSHA256(content)
	hashAlgorithm := "sha256"
	filesize := int64(len(content))
	realName := filepath.Base(sourcePath)
	longName := realName

	encryptedContent, err := encryptRawBytes(a.activeKey, content)
	if err != nil {
		return nil, fmt.Errorf("encrypt file: %w", err)
	}

	pdir, err := profileDir(profileUUID)
	if err != nil {
		return nil, err
	}

	db, err := OpenProfileDB(pdir)
	if err != nil {
		return nil, fmt.Errorf("open profile db: %w", err)
	}
	defer db.Close()

	res, err := db.Exec(`
		INSERT INTO uploads (
			entry_id,
			real_name,
			long_name,
			hash,
			hash_algorithm,
			filesize,
			state
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, entryID, realName, longName, hash, hashAlgorithm, filesize, "local")
	if err != nil {
		return nil, fmt.Errorf("insert file metadata: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get file id: %w", err)
	}

	destPath, err := encryptedProfileUploadPath(profileUUID, hash, id)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(destPath, encryptedContent, 0o600); err != nil {
		return nil, fmt.Errorf("write encrypted file: %w", err)
	}

	return &StoredUpload{
		ID:            id,
		EntryID:       entryID,
		RealName:      realName,
		LongName:      longName,
		Hash:          hash,
		HashAlgorithm: hashAlgorithm,
		Filesize:      filesize,
		State:         "local",
	}, nil
}

func listEntryUploadsFromDB(db *sql.DB, entryID int64) ([]StoredUpload, error) {
	rows, err := db.Query(`
		SELECT
			id,
			entry_id,
			real_name,
			long_name,
			hash,
			hash_algorithm,
			filesize,
			state,
			created_at,
			modified_at
		FROM uploads
		WHERE entry_id = ?
		ORDER BY modified_at DESC, id DESC
	`, entryID)
	if err != nil {
		return nil, fmt.Errorf("query entry uploads: %w", err)
	}
	defer rows.Close()

	out := []StoredUpload{}

	for rows.Next() {
		var upload StoredUpload

		if err := rows.Scan(
			&upload.ID,
			&upload.EntryID,
			&upload.RealName,
			&upload.LongName,
			&upload.Hash,
			&upload.HashAlgorithm,
			&upload.Filesize,
			&upload.State,
			&upload.CreatedAt,
			&upload.ModifiedAt,
		); err != nil {
			return nil, fmt.Errorf("scan entry upload: %w", err)
		}

		out = append(out, upload)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return out, nil
}

func (a *App) ListEntryUploads(profileUUID string, entryID int64) ([]StoredUpload, error) {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return nil, err
	}

	if entryID <= 0 {
		return nil, fmt.Errorf("Invalid entry id")
	}

	pdir, err := profileDir(profileUUID)
	if err != nil {
		return nil, err
	}

	db, err := OpenProfileDB(pdir)
	if err != nil {
		return nil, fmt.Errorf("open profile db: %w", err)
	}
	defer db.Close()

	return listEntryUploadsFromDB(db, entryID)
}

func (a *App) SelectFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select file",
	})
}
