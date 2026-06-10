/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas CARPi <Deltablot>
 * @author Moustapha Camara <Deltablot>
 * @copyright 2026 Nicolas CARPi
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 *
 * Handle uploads
 */

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type StoredUpload struct {
	ID            int64  `json:"id"`
	RealName      string `json:"realName"`
	LongName      string `json:"longName"`
	StorageName   string `json:"storageName"`
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

func (a *App) ImportUpload(profileUUID string, sourcePath string) (*StoredUpload, error) {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return nil, err
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
	storageName := hash

	if _, err := ensureProfileUploadHashDir(profileUUID, hash); err != nil {
		return nil, err
	}

	destPath, err := encryptedProfileUploadPath(profileUUID, hash)
	if err != nil {
		return nil, err
	}

	encryptedContent, err := encryptRawBytes(a.activeKey, content)
	if err != nil {
		return nil, fmt.Errorf("encrypt file: %w", err)
	}

	if err := os.WriteFile(destPath, encryptedContent, 0o600); err != nil {
		return nil, fmt.Errorf("write encrypted file: %w", err)
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
			real_name,
			long_name,
			storage_name,
			hash,
			hash_algorithm,
			filesize,
			state
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, realName, longName, storageName, hash, hashAlgorithm, filesize, "local")
	if err != nil {
		return nil, fmt.Errorf("insert file metadata: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get file id: %w", err)
	}

	return &StoredUpload{
		ID:            id,
		RealName:      realName,
		LongName:      longName,
		StorageName:   storageName,
		Hash:          hash,
		HashAlgorithm: hashAlgorithm,
		Filesize:      filesize,
		State:         "local",
	}, nil
}

func (a *App) AttachUploadToEntry(profileUUID string, entryID int64, uploadID int64) error {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return err
	}
	if entryID <= 0 {
		return fmt.Errorf("Invalid entry id")
	}
	if uploadID <= 0 {
		return fmt.Errorf("Invalid upload id")
	}

	pdir, err := profileDir(profileUUID)
	if err != nil {
		return err
	}

	db, err := OpenProfileDB(pdir)
	if err != nil {
		return fmt.Errorf("open profile db: %w", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT OR IGNORE INTO entry_uploads (entry_id, upload_id)
		VALUES (?, ?)
	`, entryID, uploadID)
	if err != nil {
		return fmt.Errorf("attach upload to entry: %w", err)
	}

	return nil
}

func (a *App) ListUploads(profileUUID string) ([]StoredUpload, error) {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return nil, err
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

	rows, err := db.Query(`
		SELECT
			id,
			real_name,
			long_name,
			storage_name,
			hash,
			hash_algorithm,
			filesize,
			state,
			created_at,
			modified_at
		FROM uploads
		ORDER BY modified_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query uploads: %w", err)
	}
	defer rows.Close()

	out := []StoredUpload{}

	for rows.Next() {
		var file StoredUpload
		if err := rows.Scan(
			&file.ID,
			&file.RealName,
			&file.LongName,
			&file.StorageName,
			&file.Hash,
			&file.HashAlgorithm,
			&file.Filesize,
			&file.State,
			&file.CreatedAt,
			&file.ModifiedAt,
		); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}

		out = append(out, file)
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

	rows, err := db.Query(`
		SELECT
			u.id,
			u.real_name,
			u.long_name,
			u.storage_name,
			u.hash,
			u.hash_algorithm,
			u.filesize,
			u.state,
			u.created_at,
			u.modified_at
		FROM uploads u
		JOIN entry_uploads eu ON eu.upload_id = u.id
		WHERE eu.entry_id = ?
		ORDER BY u.modified_at DESC, u.id DESC
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
			&upload.RealName,
			&upload.LongName,
			&upload.StorageName,
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

func (a *App) DetachUploadFromEntry(profileUUID string, entryID int64, uploadID int64) error {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return err
	}

	pdir, err := profileDir(profileUUID)
	if err != nil {
		return err
	}

	db, err := OpenProfileDB(pdir)
	if err != nil {
		return fmt.Errorf("open profile db: %w", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		DELETE FROM entry_uploads
		WHERE entry_id = ? AND upload_id = ?
	`, entryID, uploadID)
	if err != nil {
		return fmt.Errorf("detach upload from entry: %w", err)
	}

	return nil
}
