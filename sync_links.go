/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas CARPi / Deltablot
 * @author Moustapha Camara / Deltablot
 * @copyright 2026 Nicolas CARPi
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 *
 * This file's purpose is to have links related to current entry. "See in eLabFTW" will redirect to the exp/item
 */

package main

import "fmt"

type EntryRemoteLink struct {
	LocalID    int64  `json:"localId"`
	InstanceID int64  `json:"instanceId"`
	SiteURL    string `json:"siteUrl"`
	RemoteID   int64  `json:"remoteId"`
	Type       string `json:"type"`
	URL        string `json:"url"`
}

func elabftwOnlineURL(siteURL string, entityType string, remoteID int64) (string, error) {
	siteURL = normalizeElabftwSiteURL(siteURL)

	switch entityType {
	case "experiment":
		return fmt.Sprintf("%s/experiments.php?mode=view&id=%d", siteURL, remoteID), nil
	case "resource":
		return fmt.Sprintf("%s/database.php?mode=view&id=%d", siteURL, remoteID), nil
	default:
		return "", fmt.Errorf("invalid eLabFTW entity type %q", entityType)
	}
}

func (a *App) ListEntryRemoteLinks(profileUUID string, entryID int64) ([]EntryRemoteLink, error) {
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
		SELECT l.local_id, l.instance, i.site_url, l.remote_id, l.type
		FROM local2remote l
		JOIN elabftw_instances i ON i.id = l.instance
		WHERE l.local_id = ?
		ORDER BY l.id ASC
	`, entryID)
	if err != nil {
		return nil, fmt.Errorf("query remote links: %w", err)
	}
	defer rows.Close()

	out := []EntryRemoteLink{}

	for rows.Next() {
		var link EntryRemoteLink

		if err := rows.Scan(&link.LocalID, &link.InstanceID, &link.SiteURL, &link.RemoteID, &link.Type); err != nil {
			return nil, fmt.Errorf("scan remote link: %w", err)
		}

		link.URL, err = elabftwOnlineURL(link.SiteURL, link.Type, link.RemoteID)
		if err != nil {
			return nil, err
		}

		out = append(out, link)
	}

	return out, rows.Err()
}

// UnlinkEntryFromElabftw disconnects a local entry from one remote entry.
// The local and remote entries (including their uploaded files) are kept.
func (a *App) UnlinkEntryFromElabftw(profileUUID string, entryID, instanceID, remoteID int64, entityType string) error {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return err
	}
	if entryID <= 0 || instanceID <= 0 || remoteID <= 0 || (entityType != "experiment" && entityType != "resource") {
		return fmt.Errorf("invalid remote entry link")
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

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin unlink transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		DELETE FROM local2remote
		WHERE local_id = ? AND instance = ? AND remote_id = ? AND type = ?
	`, entryID, instanceID, remoteID, entityType)
	if err != nil {
		return fmt.Errorf("unlink remote entry: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check removed link: %w", err)
	}
	if deleted == 0 {
		return fmt.Errorf("remote entry link not found")
	}

	// upload2remote is not linked to local2remote by a foreign key.
	_, err = tx.Exec(`
		DELETE FROM upload2remote
		WHERE local_entry_id = ? AND instance = ? AND remote_entity_id = ? AND type = ?
	`, entryID, instanceID, remoteID, entityType)
	if err != nil {
		return fmt.Errorf("remove remote upload mappings: %w", err)
	}

	// remote_details is removed by ON DELETE CASCADE from local2remote.
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit unlink transaction: %w", err)
	}
	return nil
}
