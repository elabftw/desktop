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
	"errors"
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
			hash,
			hash_algorithm,
			filesize,
			state
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`, entryID, realName, hash, hashAlgorithm, filesize, "local")
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

// getUploadFromDB fetches one upload and ensures that it belongs to the expected entry
func getUploadFromDB(
	db *sql.DB,
	entryID int64,
	uploadID int64,
) (*StoredUpload, error) {
	var upload StoredUpload

	err := db.QueryRow(`
		SELECT
			id,
			entry_id,
			real_name,
			hash,
			hash_algorithm,
			filesize,
			state,
			created_at,
			modified_at
		FROM uploads
		WHERE id = ?
		  AND entry_id = ?
	`, uploadID, entryID).Scan(
		&upload.ID,
		&upload.EntryID,
		&upload.RealName,
		&upload.Hash,
		&upload.HashAlgorithm,
		&upload.Filesize,
		&upload.State,
		&upload.CreatedAt,
		&upload.ModifiedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("upload not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query upload: %w", err)
	}

	return &upload, nil
}

// DownloadUpload decrypts an upload and exports it to a location selected by
// the user. An empty returned path means that the user cancelled the dialog!
func (a *App) DownloadUpload(
	profileUUID string,
	entryID int64,
	uploadID int64,
) (string, error) {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return "", err
	}

	if entryID <= 0 || uploadID <= 0 {
		return "", fmt.Errorf("invalid entry or upload id")
	}

	pdir, err := profileDir(profileUUID)
	if err != nil {
		return "", err
	}

	db, err := OpenProfileDB(pdir)
	if err != nil {
		return "", fmt.Errorf("open profile db: %w", err)
	}
	defer db.Close()

	upload, err := getUploadFromDB(db, entryID, uploadID)
	if err != nil {
		return "", err
	}

	encryptedPath, err := encryptedProfileUploadPath(
		profileUUID,
		upload.Hash,
		upload.ID,
	)
	if err != nil {
		return "", err
	}

	encryptedContent, err := os.ReadFile(encryptedPath)
	if err != nil {
		return "", fmt.Errorf("read encrypted upload: %w", err)
	}

	plaintext, err := decryptRawBytes(a.activeKey, encryptedContent)
	if err != nil {
		return "", fmt.Errorf("decrypt upload: %w", err)
	}
	defer zeroBytes(plaintext)

	destinationPath, err := runtime.SaveFileDialog(
		a.ctx,
		runtime.SaveDialogOptions{
			Title:           "Save upload",
			DefaultFilename: upload.RealName,
		},
	)
	if err != nil {
		return "", fmt.Errorf("select destination: %w", err)
	}

	// When user cancelled the dialog:
	if destinationPath == "" {
		return "", nil
	}

	if err := os.WriteFile(destinationPath, plaintext, 0o600); err != nil {
		return "", fmt.Errorf("write downloaded upload: %w", err)
	}

	return destinationPath, nil
}

// DeleteUpload removes a locally stored upload.
// This deletes the local encrypted file and local database record. It does not
// delete an upload that has already been pushed to an eLabFTW instance.
func (a *App) DeleteUpload(
	profileUUID string,
	entryID int64,
	uploadID int64,
) error {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return err
	}

	if entryID <= 0 || uploadID <= 0 {
		return fmt.Errorf("invalid entry or upload id")
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

	upload, err := getUploadFromDB(db, entryID, uploadID)
	if err != nil {
		return err
	}

	encryptedPath, err := encryptedProfileUploadPath(
		profileUUID,
		upload.Hash,
		upload.ID,
	)
	if err != nil {
		return err
	}

	// Remove the physical file first. Missing files are tolerated so users can
	// still clean up stale database records.
	if err := os.Remove(encryptedPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete encrypted upload: %w", err)
	}

	result, err := db.Exec(`
		DELETE FROM uploads
		WHERE id = ?
		  AND entry_id = ?
	`, uploadID, entryID)
	if err != nil {
		return fmt.Errorf("delete upload metadata: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted row count: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("upload not found")
	}

	return nil
}
