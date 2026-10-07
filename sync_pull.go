/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas CARPi <Deltablot>
 * @author Moustapha Camara <Deltablot>
 * @copyright 2026 Nicolas CARPi
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 *
 * This file handles pulling existing remote entries into their local copies.
 */

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type PullEntryResult struct {
	LocalID  int64    `json:"localId"`
	RemoteID int64    `json:"remoteId"`
	Type     string   `json:"type"`
	Uploads  int      `json:"uploads"`
	Warnings []string `json:"warnings,omitempty"`
}

type remoteEntry struct {
	ID          int64          `json:"id"`
	Page        string         `json:"page"`
	Title       string         `json:"title"`
	Body        string         `json:"body"`
	ContentType int            `json:"content_type"`
	ModifiedAt  string         `json:"modified_at"`
	Uploads     []remoteUpload `json:"uploads"`
}

type remoteUpload struct {
	ID       int64  `json:"id"`
	RealName string `json:"real_name"`
}

type preparedPulledUpload struct {
	RemoteID int64
	RealName string
	Hash     string
	Filesize int64
}

func (a *App) PullEntryFromElabftw(
	profileUUID string,
	entryID int64,
	instanceID int64,
	entityType string,
) (*PullEntryResult, error) {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return nil, err
	}
	if entryID <= 0 {
		return nil, fmt.Errorf("Invalid entry id")
	}

	basePath, err := elabftwEntityPath(entityType)
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

	var remoteID int64
	err = db.QueryRow(`
		SELECT remote_id
		FROM local2remote
		WHERE instance = ? AND local_id = ? AND type = ?
	`, instanceID, entryID, entityType).Scan(&remoteID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("entry has no remote %s mapping for this instance", entityType)
	}
	if err != nil {
		return nil, fmt.Errorf("query local2remote: %w", err)
	}

	resp, err := a.elabftwRequest(
		profileUUID,
		instanceID,
		http.MethodGet,
		fmt.Sprintf("%s/%d", basePath, remoteID),
		nil,
		false,
	)
	if err != nil {
		return nil, err
	}

	var remote remoteEntry
	if err := decodeElabftwJSONResponse(resp, &remote); err != nil {
		return nil, err
	}

	if err := validatePulledRemoteIdentity(remote, remoteID, entityType); err != nil {
		return nil, err
	}

	remoteModifiedAt, err := parseElabftwModifiedAt(remote.ModifiedAt)
	if err != nil {
		return nil, err
	}

	remote.Title = strings.TrimSpace(remote.Title)
	if remote.Title == "" {
		return nil, fmt.Errorf("remote entry title is empty")
	}

	if remote.ContentType != 1 && remote.ContentType != 2 {
		return nil, fmt.Errorf("unsupported remote content_type %d", remote.ContentType)
	}

	encryptedTitle, err := encryptString(a.activeKey, remote.Title)
	if err != nil {
		return nil, fmt.Errorf("encrypt remote title: %w", err)
	}
	encryptedBody, err := encryptString(a.activeKey, remote.Body)
	if err != nil {
		return nil, fmt.Errorf("encrypt remote body: %w", err)
	}

	preparedUploads, createdPaths, err := a.prepareRemoteUploads(
		profileUUID,
		instanceID,
		entityType,
		remoteID,
		remote.Uploads,
	)
	if err != nil {
		removeFiles(createdPaths)
		return nil, err
	}

	oldHashes, err := replaceLocalEntryFromRemote(
		db,
		instanceID,
		entryID,
		entityType,
		remoteID,
		encryptedTitle,
		encryptedBody,
		remote.ContentType,
		remoteModifiedAt.Format(time.RFC3339Nano),
		preparedUploads,
	)
	if err != nil {
		removeFiles(createdPaths)
		return nil, err
	}

	warnings := cleanupUnreferencedUploadFiles(profileUUID, db, oldHashes)

	return &PullEntryResult{
		LocalID:  entryID,
		RemoteID: remoteID,
		Type:     entityType,
		Uploads:  len(preparedUploads),
		Warnings: warnings,
	}, nil
}

func validatePulledRemoteIdentity(remote remoteEntry, remoteID int64, entityType string) error {
	if remote.ID != remoteID {
		return fmt.Errorf("remote %s id mismatch: expected %d, got %d", entityType, remoteID, remote.ID)
	}

	expectedPage := "experiments"
	if entityType == "resource" {
		expectedPage = "database"
	}
	if remote.Page != expectedPage {
		return fmt.Errorf("remote entity type mismatch: expected %s, got %s", expectedPage, remote.Page)
	}

	return nil
}

func (a *App) prepareRemoteUploads(
	profileUUID string,
	instanceID int64,
	entityType string,
	remoteEntityID int64,
	remoteUploads []remoteUpload,
) ([]preparedPulledUpload, []string, error) {
	prepared := make([]preparedPulledUpload, 0, len(remoteUploads))
	createdPaths := make([]string, 0, len(remoteUploads))

	for _, upload := range remoteUploads {
		pulled, createdPath, err := a.pullRemoteUpload(
			profileUUID,
			instanceID,
			entityType,
			remoteEntityID,
			upload,
		)
		if err != nil {
			return nil, createdPaths, err
		}
		if createdPath != "" {
			createdPaths = append(createdPaths, createdPath)
		}
		prepared = append(prepared, *pulled)
	}

	return prepared, createdPaths, nil
}

func (a *App) pullRemoteUpload(
	profileUUID string,
	instanceID int64,
	entityType string,
	remoteEntityID int64,
	upload remoteUpload,
) (*preparedPulledUpload, string, error) {
	if upload.ID <= 0 {
		return nil, "", fmt.Errorf("remote upload has invalid id")
	}

	remoteEntityType, err := elabftwUploadEntityType(entityType)
	if err != nil {
		return nil, "", err
	}

	resp, err := a.elabftwRequest(
		profileUUID,
		instanceID,
		http.MethodGet,
		fmt.Sprintf("/%s/%d/uploads/%d?format=binary", remoteEntityType, remoteEntityID, upload.ID),
		nil,
		true,
		map[string]string{"Accept": "application/octet-stream"},
	)
	if err != nil {
		return nil, "", err
	}

	content, err := readElabftwBinaryResponse(resp)
	if err != nil {
		return nil, "", fmt.Errorf("download remote upload %d: %w", upload.ID, err)
	}
	defer zeroBytes(content)

	// Pull is an overwrite operation. The remote upload metadata is informational:
	// store exactly the bytes returned by eLabFTW and compute the local hash from
	// those bytes for Desktop's content-addressed encrypted storage.
	hash := fileSHA256(content)

	encryptedContent, err := encryptRawBytes(a.activeKey, content)
	if err != nil {
		return nil, "", fmt.Errorf("encrypt remote upload %q: %w", upload.RealName, err)
	}
	defer zeroBytes(encryptedContent)

	destination, err := encryptedProfileUploadPath(profileUUID, hash)
	if err != nil {
		return nil, "", err
	}

	createdPath := ""
	_, statErr := os.Stat(destination)
	switch {
	case statErr == nil:
		// Identical plaintext content already exists in local encrypted storage.
	case errors.Is(statErr, os.ErrNotExist):
		if err := os.WriteFile(destination, encryptedContent, 0o600); err != nil {
			return nil, "", fmt.Errorf("write pulled upload %q: %w", upload.RealName, err)
		}
		createdPath = destination
	default:
		return nil, "", fmt.Errorf("stat pulled upload %q: %w", upload.RealName, statErr)
	}

	realName := strings.TrimSpace(upload.RealName)
	if realName == "" {
		realName = fmt.Sprintf("upload-%d", upload.ID)
	}

	return &preparedPulledUpload{
		RemoteID: upload.ID,
		RealName: realName,
		Hash:     hash,
		Filesize: int64(len(content)),
	}, createdPath, nil
}

func readElabftwBinaryResponse(resp *http.Response) ([]byte, error) {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, decodeElabftwJSONResponse(resp, nil)
	}
	defer closeResponseBody(resp)

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read eLabFTW response: %w", err)
	}
	return content, nil
}

func replaceLocalEntryFromRemote(
	db *sql.DB,
	instanceID int64,
	entryID int64,
	entityType string,
	remoteID int64,
	encryptedTitle string,
	encryptedBody string,
	contentType int,
	modifiedAt string,
	uploads []preparedPulledUpload,
) ([]string, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin pull transaction: %w", err)
	}
	defer tx.Rollback()

	oldHashes := make([]string, 0)
	rows, err := tx.Query(`SELECT DISTINCT hash FROM uploads WHERE entry_id = ?`, entryID)
	if err != nil {
		return nil, fmt.Errorf("query old upload hashes: %w", err)
	}
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan old upload hash: %w", err)
		}
		oldHashes = append(oldHashes, hash)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close old upload hashes: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate old upload hashes: %w", err)
	}

	result, err := tx.Exec(`
		UPDATE entries
		SET title = ?, body = ?, content_type = ?, modified_at = ?
		WHERE id = ?
	`, encryptedTitle, encryptedBody, contentType, modifiedAt, entryID)
	if err != nil {
		return nil, fmt.Errorf("replace local entry: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read replaced entry count: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("Entry not found")
	}

	if _, err := tx.Exec(`DELETE FROM uploads WHERE entry_id = ?`, entryID); err != nil {
		return nil, fmt.Errorf("replace local uploads: %w", err)
	}

	for _, upload := range uploads {
		result, err := tx.Exec(`
			INSERT INTO uploads (entry_id, real_name, hash, hash_algorithm, filesize, state)
			VALUES (?, ?, ?, 'sha256', ?, 'synced')
		`, entryID, upload.RealName, upload.Hash, upload.Filesize)
		if err != nil {
			return nil, fmt.Errorf("insert pulled upload %q: %w", upload.RealName, err)
		}

		localUploadID, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("get pulled upload id: %w", err)
		}

		_, err = tx.Exec(`
			INSERT INTO upload2remote (
				instance,
				local_upload_id,
				local_entry_id,
				remote_entity_id,
				remote_upload_id,
				type
			)
			VALUES (?, ?, ?, ?, ?, ?)
		`, instanceID, localUploadID, entryID, remoteID, upload.RemoteID, entityType)
		if err != nil {
			return nil, fmt.Errorf("map pulled upload %q: %w", upload.RealName, err)
		}
	}

	result, err = tx.Exec(`
		UPDATE local2remote
		SET modified_at = ?
		WHERE instance = ? AND local_id = ? AND type = ? AND remote_id = ?
	`, modifiedAt, instanceID, entryID, entityType, remoteID)
	if err != nil {
		return nil, fmt.Errorf("update pull sync timestamp: %w", err)
	}
	affected, err = result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read updated mapping count: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("remote mapping changed during pull")
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit pull transaction: %w", err)
	}
	return oldHashes, nil
}

func cleanupUnreferencedUploadFiles(profileUUID string, db *sql.DB, hashes []string) []string {
	warnings := make([]string, 0)
	seen := make(map[string]struct{}, len(hashes))

	for _, hash := range hashes {
		if _, ok := seen[hash]; ok {
			continue
		}
		seen[hash] = struct{}{}

		var references int
		if err := db.QueryRow(`SELECT COUNT(*) FROM uploads WHERE hash = ?`, hash).Scan(&references); err != nil {
			warnings = append(warnings, fmt.Sprintf("cleanup upload %s: %s", hash, err))
			continue
		}
		if references != 0 {
			continue
		}

		path, err := encryptedProfileUploadPath(profileUUID, hash)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("cleanup upload %s: %s", hash, err))
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			warnings = append(warnings, fmt.Sprintf("cleanup upload %s: %s", hash, err))
		}
	}

	return warnings
}

func removeFiles(paths []string) {
	for _, path := range paths {
		_ = os.Remove(path)
	}
}
