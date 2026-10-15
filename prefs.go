/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas CARPi / Deltablot
 * @author Moustapha Camara / Deltablot
 * @copyright 2026 Deltablot
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 *
 * preferences table. Will be filled with more options later
 */
package main

import (
	"database/sql"
	"errors"
	"fmt"
)

const preferredEditorKey = "preferred_editor"

// read the preferred editor from the unlocked profile's own SQLite database
func (a *App) GetPreferredEditor(profileUUID string) (string, error) {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return "", err
	}
	pdir, err := profileDir(profileUUID)
	if err != nil {
		return "", err
	}
	db, err := OpenProfileDB(pdir)
	if err != nil {
		return "", fmt.Errorf("Open profile db: %w", err)
	}
	defer db.Close()

	var editor string
	err = db.QueryRow(`SELECT value FROM preferences WHERE key = ?`, preferredEditorKey).Scan(&editor)
	// A new profile defaults to TinyMCE until its preference is saved
	if errors.Is(err, sql.ErrNoRows) {
		return "tinymce", nil
	}
	if err != nil {
		return "", fmt.Errorf("Read preferred editor: %w", err)
	}
	if editor != "tinymce" && editor != "markdown" {
		return "", fmt.Errorf("Invalid preferred editor preference")
	}
	return editor, nil
}

// save the preferred editor setting only for the unlocked profile
func (a *App) SetPreferredEditor(profileUUID string, editor string) error {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return err
	}
	if editor != "tinymce" && editor != "markdown" {
		return fmt.Errorf("Invalid preferred editor")
	}
	pdir, err := profileDir(profileUUID)
	if err != nil {
		return err
	}
	db, err := OpenProfileDB(pdir)
	if err != nil {
		return fmt.Errorf("Open profile db: %w", err)
	}
	defer db.Close()

	// UPSERT updates one key instead of creating duplicate preference rows
	_, err = db.Exec(`
INSERT INTO preferences (key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value
`, preferredEditorKey, editor)
	if err != nil {
		return fmt.Errorf("Save preferred editor: %w", err)
	}
	return nil
}
