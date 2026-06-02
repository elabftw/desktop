/*
 * This file will handle the PUSHing of entries to configured eLabFTW instance
 *
 */

package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
	err = db.QueryRow(`
		SELECT remote_id
		FROM local2remote
		WHERE instance = ? AND local_id = ? AND type = ?
	`, instanceID, entryID, entityType).Scan(&remoteID)

	if err == nil {
		reqBody, err := jsonBody(payload)
		if err != nil {
			return nil, err
		}

		resp, err := a.elabftwRequest(
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

		_, err = db.Exec(`
			UPDATE local2remote
			SET updated_at = (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
			WHERE instance = ? AND local_id = ? AND type = ?
		`, instanceID, entryID, entityType)
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

	remoteID, err = remoteIDFromLocation(resp.Header.Get("Location"))
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		INSERT INTO local2remote (instance, remote_id, local_id, type)
		VALUES (?, ?, ?, ?)
	`, instanceID, remoteID, entryID, entityType)
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
