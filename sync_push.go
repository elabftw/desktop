/*
 * This file will handle the PUSHing of entries to configured eLabFTW instance
 *
 */

package main

import (
	"database/sql"
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

func (a *App) PushEntryToElabftw(profileUUID string, entryID int64, instanceID int64, entityType string) (*PushEntryResult, error) {
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
		"title": title,
		"body":  bodyText,
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
		// First GET the remote entity and check whether it changed since last sync.
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

		if remoteModifiedAt.After(lastSyncAt) {
			return nil, fmt.Errorf(remoteModifiedConflictMessage(entityType, remoteID))
		}

		reqBody, err := jsonBody(payload)
		if err != nil {
			return nil, err
		}

		resp, err = a.elabftwRequest(
			profileUUID,
			instanceID,
			http.MethodGet,
			fmt.Sprintf("%s/%d", basePath, remoteID),
			nil,
		)
		if err != nil {
			return nil, err
		}

		var patchedRemote map[string]any
		if err := decodeElabftwJSONResponse(resp, &patchedRemote); err != nil {
			return nil, err
		}

		patchedRemoteModifiedAt, err := parseElabftwModifiedAt(patchedRemote["modified_at"])
		if err != nil {
			return nil, err
		}

		_, err = db.Exec(`
        	UPDATE local2remote
        	SET modified_at = ?
        	WHERE instance = ? AND local_id = ? AND type = ?
        `, patchedRemoteModifiedAt.Format(time.RFC3339Nano), instanceID, entryID, entityType)
		if err != nil {
			return nil, fmt.Errorf("update local2remote: %w", err)
		}

		return &PushEntryResult{
			LocalID:  entryID,
			RemoteID: remoteID,
			Action:   "patched",
			Type:     entityType,
		}, nil
	}

	reqBody, err := jsonBody(payload)
	if err != nil {
		return nil, err
	}
	resp, err = a.elabftwRequest(
		profileUUID,
		instanceID,
		http.MethodGet,
		fmt.Sprintf("%s/%d", basePath, remoteID),
		nil,
	)
	if err != nil {
		return nil, err
	}

	var createdRemote map[string]any
	if err := decodeElabftwJSONResponse(resp, &createdRemote); err != nil {
		return nil, err
	}

	createdRemoteModifiedAt, err := parseElabftwModifiedAt(createdRemote["modified_at"])
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

func (a *App) PushAllEntriesToElabftw(profileUUID string, instanceID int64, entityType string) ([]PushEntryResult, error) {
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

		result, err := a.PushEntryToElabftw(profileUUID, id, instanceID, entityType)
		if err != nil {
			return nil, fmt.Errorf("push entry %d: %w", id, err)
		}

		results = append(results, *result)
	}

	return results, rows.Err()
}

// the posted experiment's id is in the Location header so we need to parse it
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

	s = strings.TrimSpace(s)

	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000000Z",
	}

	for _, layout := range formats {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse remote modified_at %q", s)
}

func parseLocalModifiedAt(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("local modified_at is empty")
	}

	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000000Z",
	}

	for _, layout := range formats {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse local modified_at %q", value)
}

func remoteModifiedConflictMessage(entityType string, remoteID int64) string {
	return fmt.Sprintf(
		"Remote %s #%d was modified after your last sync. Pull or review the online version before pushing.",
		entityType,
		remoteID,
	)
}
