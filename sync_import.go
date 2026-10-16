/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas CARPi / Deltablot
 * @author Moustapha Camara / Deltablot
 * @copyright 2026 Nicolas CARPi
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 *
 * Browse a lightweight remote entry list and import one entry at a time.
 */
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const remoteEntryPageSize = 50

type RemoteEntryOption struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

// fastq selects the server's readAllSimple projection (id/title and a few labels)
// rather than readAll. A % query matches all titles, and paging keeps the picker
// responsive on servers with many entries. Scope 3 means all readable entries;
// scope 1 would show only entries owned by the API key's user.
func (a *App) ListRemoteEntries(profileUUID string, instanceID int64, entityType string, search string, offset int) ([]RemoteEntryOption, error) {
	if _, err := a.requireUnlockedProfile(profileUUID); err != nil {
		return nil, err
	}
	if instanceID <= 0 || offset < 0 {
		return nil, fmt.Errorf("invalid instance id or offset")
	}
	basePath, err := elabftwEntityPath(entityType)
	if err != nil {
		return nil, err
	}
	search = strings.TrimSpace(search)
	if search == "" {
		search = "%"
	}
	params := url.Values{}
	params.Set("fastq", search)
	params.Set("scope", "3")
	params.Set("state", "1,2") // normal and archived (never deleted)
	params.Set("limit", strconv.Itoa(remoteEntryPageSize))
	params.Set("offset", strconv.Itoa(offset))

	resp, err := a.elabftwRequest(profileUUID, instanceID, http.MethodGet, basePath+"?"+params.Encode(), nil, false)
	if err != nil {
		return nil, err
	}
	entries := []RemoteEntryOption{}
	if err := decodeElabftwJSONResponse(resp, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// Import one remote experiment/resource as a new local entry. Never silently
// overwrite a previously imported entry; the editor's Pull button handles that.
func (a *App) ImportEntryFromElabftw(profileUUID string, instanceID int64, entityType string, remoteID int64) (*PullEntryResult, error) {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return nil, err
	}
	if instanceID <= 0 || remoteID <= 0 {
		return nil, fmt.Errorf("invalid instance or remote entry id")
	}
	if _, err := elabftwEntityPath(entityType); err != nil {
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

	var existingID int64
	err = db.QueryRow(`
        SELECT local_id FROM local2remote
        WHERE instance = ? AND remote_id = ? AND type = ?
    `, instanceID, remoteID, entityType).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("remote entry already exists locally as entry #%d; use Pull on that entry", existingID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check existing remote mapping: %w", err)
	}

	return a.pullRemoteEntry(profileUUID, db, 0, instanceID, entityType, remoteID)
}
