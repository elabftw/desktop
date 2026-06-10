/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas CARPi <Deltablot>
 * @author Moustapha Camara <Deltablot>
 * @copyright 2026 Nicolas CARPi
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 *
 * This file handles the PUSHing of entries to configured eLabFTW instance
 */

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type PushEntryResult struct {
	LocalID  int64  `json:"localId"`
	RemoteID int64  `json:"remoteId"`
	Action   string `json:"action"` // posted or patched
	Type     string `json:"type"`
}

func elabftwEntityPath(entityType string) (string, error) {
	switch entityType {
	case "experiment":
		return "/experiments", nil
	case "resource":
		return "/items", nil
	default:
		return "", fmt.Errorf("Invalid eLabFTW entity type")
	}
}

func (a *App) PushEntryToElabftw(profileUUID string, entryID int64, instanceID int64, entityType string, force bool) (*PushEntryResult, error) {
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

	var encryptedTitle string
	var encryptedBody string

	err = db.QueryRow(`
		SELECT title, body
		FROM entries
		WHERE id = ?
	`, entryID).Scan(&encryptedTitle, &encryptedBody)
	if err != nil {
		return nil, fmt.Errorf("query entry: %w", err)
	}

	title, err := decryptString(a.activeKey, encryptedTitle)
	if err != nil {
		return nil, fmt.Errorf("decrypt title: %w", err)
	}

	bodyText, err := decryptString(a.activeKey, encryptedBody)
	if err != nil {
		return nil, fmt.Errorf("decrypt body: %w", err)
	}

	payload := map[string]any{
		"title":        title,
		"body":         renderMarkdownToHTML(bodyText),
		"content_type": 1,
	}

	var remoteID int64
	var lastSyncModifiedAt string

	err = db.QueryRow(`
    	SELECT remote_id, modified_at
    	FROM local2remote
    	WHERE instance = ? AND local_id = ? AND type = ?
    `, instanceID, entryID, entityType).Scan(&remoteID, &lastSyncModifiedAt)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("query local2remote: %w", err)
	}

	if err == nil {
		// Existing local2remote row means this should PATCH.
		return a.patchExistingRemoteEntry(
			profileUUID,
			db,
			instanceID,
			entryID,
			entityType,
			basePath,
			remoteID,
			lastSyncModifiedAt,
			payload,
			force,
		)
	}

	return a.postNewRemoteEntry(
		profileUUID,
		db,
		instanceID,
		entryID,
		entityType,
		basePath,
		payload,
	)
}

func (a *App) patchExistingRemoteEntry(
	profileUUID string,
	db *sql.DB,
	instanceID int64,
	entryID int64,
	entityType string,
	basePath string,
	remoteID int64,
	lastSyncModifiedAt string,
	payload map[string]any,
	force bool,
) (*PushEntryResult, error) {
	// First GET remote to check if someone edited it after our last successful sync.
	resp, err := a.elabftwRequest(
		profileUUID,
		instanceID,
		http.MethodGet,
		fmt.Sprintf("%s/%d", basePath, remoteID),
		nil,
	)
	if err != nil {
		return nil, err
	}

	var remote map[string]any
	if err := decodeElabftwJSONResponse(resp, &remote); err != nil {
		return nil, err
	}

	remoteModifiedAt, err := parseElabftwModifiedAt(remote["modified_at"])
	if err != nil {
		return nil, err
	}

	lastSyncAt, err := parseLocalModifiedAt(lastSyncModifiedAt)
	if err != nil {
		return nil, err
	}

	if !force && remoteModifiedAt.After(lastSyncAt) {
		return nil, errors.New(remoteModifiedConflictMessage(entityType, remoteID))
	}

	reqBody, err := jsonBody(payload)
	if err != nil {
		return nil, err
	}

	resp, err = a.elabftwRequest(
		profileUUID,
		instanceID,
		http.MethodPatch,
		fmt.Sprintf("%s/%d", basePath, remoteID),
		reqBody,
	)
	if err != nil {
		return nil, err
	}

	if err := decodeElabftwJSONResponse(resp, nil); err != nil {
		return nil, err
	}

	patchedRemoteModifiedAt, err := a.fetchRemoteModifiedAt(profileUUID, instanceID, basePath, remoteID)
	if err != nil {
		return nil, err
	}

	if err := updateLocalRemoteModifiedAt(db, instanceID, entryID, entityType, patchedRemoteModifiedAt); err != nil {
		return nil, err
	}

	return &PushEntryResult{
		LocalID:  entryID,
		RemoteID: remoteID,
		Action:   "patched",
		Type:     entityType,
	}, nil
}

func (a *App) postNewRemoteEntry(
	profileUUID string,
	db *sql.DB,
	instanceID int64,
	entryID int64,
	entityType string,
	basePath string,
	payload map[string]any,
) (*PushEntryResult, error) {
	reqBody, err := jsonBody(payload)
	if err != nil {
		return nil, err
	}

	resp, err := a.elabftwRequest(
		profileUUID,
		instanceID,
		http.MethodPost,
		basePath,
		reqBody,
	)
	if err != nil {
		return nil, err
	}

	if err := decodeElabftwJSONResponse(resp, nil); err != nil {
		return nil, err
	}

	remoteID, err := remoteIDFromLocation(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}

	createdRemoteModifiedAt, err := a.fetchRemoteModifiedAt(profileUUID, instanceID, basePath, remoteID)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		INSERT INTO local2remote (instance, remote_id, local_id, type, modified_at)
		VALUES (?, ?, ?, ?, ?)
	`, instanceID, remoteID, entryID, entityType, createdRemoteModifiedAt.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("insert local2remote: %w", err)
	}

	return &PushEntryResult{
		LocalID:  entryID,
		RemoteID: remoteID,
		Action:   "posted",
		Type:     entityType,
	}, nil
}

// after pushing, we re-GET the item's modified_at or else it would always be slightly
// different (& superior) to local modified_at.
func (a *App) fetchRemoteModifiedAt(profileUUID string, instanceID int64, basePath string, remoteID int64) (time.Time, error) {
	resp, err := a.elabftwRequest(
		profileUUID,
		instanceID,
		http.MethodGet,
		fmt.Sprintf("%s/%d", basePath, remoteID),
		nil,
	)
	if err != nil {
		return time.Time{}, err
	}

	var remote map[string]any
	if err := decodeElabftwJSONResponse(resp, &remote); err != nil {
		return time.Time{}, err
	}

	return parseElabftwModifiedAt(remote["modified_at"])
}

func updateLocalRemoteModifiedAt(db *sql.DB, instanceID int64, entryID int64, entityType string, modifiedAt time.Time) error {
	_, err := db.Exec(`
		UPDATE local2remote
		SET modified_at = ?
		WHERE instance = ? AND local_id = ? AND type = ?
	`, modifiedAt.Format(time.RFC3339Nano), instanceID, entryID, entityType)
	if err != nil {
		return fmt.Errorf("update local2remote: %w", err)
	}

	return nil
}

func (a *App) PushAllEntriesToElabftw(profileUUID string, instanceID int64, entityType string, force bool) ([]PushEntryResult, error) {
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

	rows, err := db.Query(`SELECT id FROM entries ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	defer rows.Close()

	results := []PushEntryResult{}

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan entry id: %w", err)
		}

		result, err := a.PushEntryToElabftw(profileUUID, id, instanceID, entityType, force)
		if err != nil {
			return nil, fmt.Errorf("push entry %d: %w", id, err)
		}

		results = append(results, *result)
	}

	return results, rows.Err()
}

// the posted entry's id is in the Location header so we need to parse it
func remoteIDFromLocation(location string) (int64, error) {
	location = strings.TrimSpace(location)
	if location == "" {
		return 0, fmt.Errorf("missing Location header")
	}

	location = strings.TrimRight(location, "/")
	parts := strings.Split(location, "/")
	last := parts[len(parts)-1]

	id, err := strconv.ParseInt(last, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse remote id from Location header %q: %w", location, err)
	}

	return id, nil
}

/*
 * The behaviour we want, ON PATCH, is:
 * GET remote first. Read remote's modified_at.
 * If remote's modified_at is more recent then local modified_at, stop and return a warning
 * else patch. We want to keep remote as source of truth and warn
 */

func parseElabftwModifiedAt(value any) (time.Time, error) {
	s, ok := value.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return time.Time{}, fmt.Errorf("remote response does not contain a valid modified_at")
	}

	return parseSyncTime(s, "remote modified_at")
}

func parseLocalModifiedAt(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, fmt.Errorf("local modified_at is empty")
	}

	return parseSyncTime(value, "local modified_at")
}

func parseSyncTime(value string, label string) (time.Time, error) {
	value = strings.TrimSpace(value)

	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999Z",
		"2006-01-02T15:04:05.999Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
	}

	for _, layout := range formats {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse %s %q", label, value)
}

func remoteModifiedConflictMessage(entityType string, remoteID int64) string {
	return fmt.Sprintf(
		"Remote %s #%d was modified after your last sync. Pull or review the online version before pushing.",
		entityType,
		remoteID,
	)
}

// handle uploads
func elabftwUploadEntityType(entityType string) (string, error) {
	switch entityType {
	case "experiment":
		return "experiments", nil
	case "resource":
		return "items", nil
	default:
		return "", fmt.Errorf("Invalid eLabFTW entity type")
	}
}
