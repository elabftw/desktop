/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas CARPi / Deltablot
 * @author Moustapha Camara / Deltablot
 * @copyright 2026 Nicolas CARPi
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 *
 * This file handles pulling existing remote entries into their local copies.
 */

package main

import (
	"database/sql"
	"encoding/json"
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
	ID               int64           `json:"id"`
	Page             string          `json:"page"`
	Title            string          `json:"title"`
	Body             string          `json:"body"`
	ContentType      int             `json:"content_type"`
	ModifiedAt       string          `json:"modified_at"`
	CategoryTitle    string          `json:"category_title"`
	CategoryColor    string          `json:"category_color"`
	StatusTitle      string          `json:"status_title"`
	StatusColor      string          `json:"status_color"`
	Tags             string          `json:"tags"`
	Metadata         json.RawMessage `json:"metadata"`
	ItemsLinks       json.RawMessage `json:"items_links"`
	ExperimentsLinks json.RawMessage `json:"experiments_links"`
	Uploads          []remoteUpload  `json:"uploads"`
}

type remoteUpload struct {
	ID       int64  `json:"id"`
	RealName string `json:"real_name"`
}

// Metadata staged in memory until all uploads are ready for the database transaction
type preparedPulledUpload struct {
	RemoteID int64
	RealName string
	Hash     string
	Filesize int64
}

// overwrite a linked local entry with the selected remote version
// Prepare encrypted files first, then update the entry and its uploads together
// with readonly for tags, links and custom fields
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

	pdir, err := profileDir(profileUUID)
	if err != nil {
		return nil, err
	}

	db, err := OpenProfileDB(pdir)
	if err != nil {
		return nil, fmt.Errorf("open profile db: %w", err)
	}
	defer db.Close()

	// One local entry can link to multiple instances, so use the selected mapping
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

	return a.pullRemoteEntry(profileUUID, db, entryID, instanceID, entityType, remoteID)
}

// Shared pipeline for overwriting an existing linked entry or importing a new
// remote entry. The latter passes entryID = 0; both paths download the exact
// same content, metadata, links and uploads before starting a DB transaction.
func (a *App) pullRemoteEntry(profileUUID string, db *sql.DB, entryID int64, instanceID int64, entityType string, remoteID int64) (*PullEntryResult, error) {
	basePath, err := elabftwEntityPath(entityType)
	if err != nil {
		return nil, err
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

	// guard: reject an unexpected remote ID or type before overwriting anything locally
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

	// The API returns plaintext but local titles and bodies stay encrypted in SQLite
	encryptedTitle, err := encryptString(a.activeKey, remote.Title)
	if err != nil {
		return nil, fmt.Errorf("encrypt remote title: %w", err)
	}
	encryptedBody, err := encryptString(a.activeKey, remote.Body)
	if err != nil {
		return nil, fmt.Errorf("encrypt remote body: %w", err)
	}

	// Save read-only information from the existing remote entity response.
	snapshotJSON, err := json.Marshal(remoteReadOnlySnapshot{
		CategoryTitle:    remote.CategoryTitle,
		CategoryColor:    remote.CategoryColor,
		StatusTitle:      remote.StatusTitle,
		StatusColor:      remote.StatusColor,
		Tags:             remote.Tags,
		Metadata:         remote.Metadata,
		ItemsLinks:       remote.ItemsLinks,
		ExperimentsLinks: remote.ExperimentsLinks,
	})
	if err != nil {
		return nil, fmt.Errorf("encode remote details: %w", err)
	}
	encryptedDetails, err := encryptString(a.activeKey, string(snapshotJSON))
	if err != nil {
		return nil, fmt.Errorf("encrypt remote details: %w", err)
	}

	// Entries can share an encrypted file when their content hashes match
	// Keep file writes, database changes and cleanup locked across concurrent calls
	uploadFilesMu.Lock()
	defer uploadFilesMu.Unlock()

	// Download and encrypt every upload before modifying any database rows
	// On failure, remove only the files created by this attempt
	preparedUploads := make([]preparedPulledUpload, 0, len(remote.Uploads))
	createdPaths := make([]string, 0, len(remote.Uploads))
	for _, upload := range remote.Uploads {
		pulled, createdPath, err := a.pullRemoteUpload(profileUUID, instanceID, entityType, remoteID, upload)
		if err != nil {
			removeFiles(createdPaths)
			return nil, err
		}
		if createdPath != "" {
			createdPaths = append(createdPaths, createdPath)
		}
		preparedUploads = append(preparedUploads, *pulled)
	}

	// replace the entry and upload mappings in one SQLite transaction
	oldHashes, err := replaceLocalEntryFromRemote(
		db,
		instanceID,
		&entryID,
		entityType,
		remoteID,
		encryptedTitle,
		encryptedBody,
		remote.ContentType,
		remoteModifiedAt.Format(time.RFC3339Nano),
		preparedUploads,
		encryptedDetails,
	)
	if err != nil {
		removeFiles(createdPaths)
		return nil, err
	}

	// Delete old files only when no remaining upload row references their hash
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

type remoteReadOnlySnapshot struct {
	CategoryTitle    string          `json:"categoryTitle"`
	CategoryColor    string          `json:"categoryColor"`
	StatusTitle      string          `json:"statusTitle"`
	StatusColor      string          `json:"statusColor"`
	Tags             string          `json:"tags"`
	Metadata         json.RawMessage `json:"metadata"`
	ItemsLinks       json.RawMessage `json:"itemsLinks"`
	ExperimentsLinks json.RawMessage `json:"experimentsLinks"`
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

	// Hash downloaded bytes, not the remote upload ID or filename
	// Identical content can reuse a single encrypted file inside this profile
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
			_ = os.Remove(destination) // WriteFile can leave a partial encrypted file
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
	entryID *int64,
	entityType string,
	remoteID int64,
	encryptedTitle string,
	encryptedBody string,
	contentType int,
	modifiedAt string,
	uploads []preparedPulledUpload,
	encryptedDetails string,
) ([]string, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin pull transaction: %w", err)
	}
	// Roll back an incomplete replacement; after Commit this becomes a no-op
	defer tx.Rollback()

	localID := *entryID
	if localID == 0 {
		// Create both the entry and its remote mapping in the SAME transaction
		// as uploads and read-only details. Any failure leaves no partial entry.
		result, err := tx.Exec(`
			INSERT INTO entries (title, body, content_type) VALUES (?, ?, ?)
		`, encryptedTitle, encryptedBody, contentType)
		if err != nil {
			return nil, fmt.Errorf("create pulled entry: %w", err)
		}
		localID, err = result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("get pulled entry id: %w", err)
		}
		if _, err := tx.Exec(`
			INSERT INTO local2remote (instance, remote_id, local_id, type, modified_at)
			VALUES (?, ?, ?, ?, ?)
		`, instanceID, remoteID, localID, entityType, modifiedAt); err != nil {
			return nil, fmt.Errorf("map imported entry (possibly already imported): %w", err)
		}
	}

	var oldHashes []string
	rows, err := tx.Query(`SELECT DISTINCT hash FROM uploads WHERE entry_id = ?`, localID)
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

	// The entry gets a fresh local timestamp while the link records the remote sync time
	result, err := tx.Exec(`
		UPDATE entries
		SET title = ?, body = ?, content_type = ?,
			modified_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE id = ?
	`, encryptedTitle, encryptedBody, contentType, localID)
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

	// Foreign keys also remove upload2remote mappings for these deleted uploads
	if _, err := tx.Exec(`DELETE FROM uploads WHERE entry_id = ?`, localID); err != nil {
		return nil, fmt.Errorf("replace local uploads: %w", err)
	}

	for _, upload := range uploads {
		result, err := tx.Exec(`
			INSERT INTO uploads (entry_id, real_name, hash, hash_algorithm, filesize, state)
			VALUES (?, ?, ?, 'sha256', ?, 'synced')
		`, localID, upload.RealName, upload.Hash, upload.Filesize)
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
		`, instanceID, localUploadID, localID, remoteID, upload.RemoteID, entityType)
		if err != nil {
			return nil, fmt.Errorf("map pulled upload %q: %w", upload.RealName, err)
		}
	}

	result, err = tx.Exec(`
		UPDATE local2remote
		SET modified_at = ?
		WHERE instance = ? AND local_id = ? AND type = ? AND remote_id = ?
	`, modifiedAt, instanceID, localID, entityType, remoteID)
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

	// Only update the read-only snapshot if the matching remote mapping still exists
	result, err = tx.Exec(`
        INSERT INTO remote_details (link_id, encrypted_data, pulled_at)
        SELECT id, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now')
        FROM local2remote
        WHERE instance = ? AND local_id = ? AND type = ? AND remote_id = ?
        ON CONFLICT(link_id) DO UPDATE SET
            encrypted_data = excluded.encrypted_data,
            pulled_at = excluded.pulled_at
    `, encryptedDetails, instanceID, localID, entityType, remoteID)
	if err != nil {
		return nil, fmt.Errorf("save read-only remote details: %w", err)
	}
	affected, err = result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read saved remote details count: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("remote mapping changed while saving details")
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit pull transaction: %w", err)
	}
	*entryID = localID
	return oldHashes, nil
}

// Cleanup failures are warnings because the new entry has already been committed
func cleanupUnreferencedUploadFiles(profileUUID string, db *sql.DB, hashes []string) []string {
	var warnings []string

	for _, hash := range hashes {
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
