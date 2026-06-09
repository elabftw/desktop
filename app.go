/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas CARPi <Deltablot>
 * @author Moustapha Camara <Deltablot>
 * @copyright 2026 Nicolas CARPi
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 */
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

type App struct {
	ctx   context.Context
	index *ProfileIndex

	activeProfileUUID string

	// activeKey is the passphrase-derived symmetric key kept only in memory
	// while a profile is unlocked. It is cleared on LockProfile.
	activeKey []byte
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	idx, err := loadProfileIndex()
	if err != nil {
		panic(err)
	}

	a.index = idx
}

// requireUnlockedProfile gates authentified actions.
// Frontend can call exported Wails methods directly, so UI checks are not
// enough. Every method that reads or writes profile data must verify that
// the requested profile is currently unlocked.
func (a *App) requireUnlockedProfile(profileUUID string) (string, error) {
	profileUUID = strings.TrimSpace(profileUUID)
	if profileUUID == "" {
		return "", fmt.Errorf("Profile uuid is empty")
	}
	if a.activeProfileUUID != profileUUID || len(a.activeKey) == 0 {
		return "", fmt.Errorf("Profile is locked")
	}
	return profileUUID, nil
}

// UnlockProfile verifies the passphrase by decrypting the profile's encrypted
// verifier. On success, the derived key is kept in memory and used to
// encrypt/decrypt entry contents for this profile only.
func (a *App) UnlockProfile(profileUUID string, passphrase string) error {
	profileUUID = strings.TrimSpace(profileUUID)
	if profileUUID == "" {
		return fmt.Errorf("Profile uuid is empty")
	}
	passphrase = strings.TrimSpace(passphrase)
	if passphrase == "" {
		return fmt.Errorf("Passphrase is empty")
	}

	if a.index == nil {
		return fmt.Errorf("Profile index is not loaded")
	}

	var selected *ProfileEntry
	for i := range a.index.Profiles {
		if a.index.Profiles[i].UUID == profileUUID {
			selected = &a.index.Profiles[i]
			break
		}
	}

	if selected == nil {
		return fmt.Errorf("Unknown profile uuid")
	}

	key, err := unlockProfileCryptoParams(selected, passphrase)
	if err != nil {
		return err
	}

	a.LockProfile()

	a.activeProfileUUID = profileUUID
	a.activeKey = key

	return nil
}

// LockProfile clears the in-memory session state.
// After this, profile-scoped actions reject reads/writes until UnlockProfile
// succeeds again.
func (a *App) LockProfile() {
	a.activeProfileUUID = ""

	if a.activeKey != nil {
		zeroBytes(a.activeKey)
	}
	a.activeKey = nil
}

// DeleteProfile deletes a profile after verifying its passphrase.
// It does not require the profile to already be unlocked.
func (a *App) DeleteProfile(profileUUID string, passphrase string) (*ProfileIndex, error) {
	profileUUID = strings.TrimSpace(profileUUID)
	if profileUUID == "" {
		return nil, fmt.Errorf("Profile uuid is empty")
	}

	passphrase = strings.TrimSpace(passphrase)
	if passphrase == "" {
		return nil, fmt.Errorf("Passphrase is empty")
	}

	idx, err := loadProfileIndex()
	if err != nil {
		return nil, err
	}

	found := false
	filtered := make([]ProfileEntry, 0, len(idx.Profiles))

	for _, profile := range idx.Profiles {
		if profile.UUID == profileUUID {
			found = true

			key, err := unlockProfileCryptoParams(&profile, passphrase)
			if err != nil {
				return nil, err
			}
			zeroBytes(key)

			continue
		}

		filtered = append(filtered, profile)
	}

	if !found {
		return nil, fmt.Errorf("Unknown profile uuid")
	}

	dir, err := profileDir(profileUUID)
	if err != nil {
		return nil, err
	}

	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("Delete profile dir: %w", err)
	}

	idx.Profiles = filtered

	if err := saveProfileIndex(idx); err != nil {
		return nil, err
	}

	if a.activeProfileUUID == profileUUID {
		a.LockProfile()
	}

	a.index = idx
	return a.index, nil
}

func (a *App) GetProfileIndex() *ProfileIndex {
	return a.index
}

type diskProfileIndex struct {
	Version  int           `json:"version"`
	Profiles []diskProfile `json:"profiles"`
}

type diskProfile struct {
	UUID        string `json:"uuid"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"createdAt"`
}

// AddProfile creates a new profile with a passphrase-derived encryption key.
func (a *App) AddProfile(displayName string, passphrase string) (*ProfileIndex, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, fmt.Errorf("Display name is empty")
	}

	passphrase = strings.TrimSpace(passphrase)
	if passphrase == "" {
		return nil, fmt.Errorf("Passphrase is empty")
	}

	newUUID := uuid.NewString()

	cryptoParams, err := createProfileCryptoParams(passphrase)
	if err != nil {
		return nil, err
	}

	idx, err := loadProfileIndex()
	if err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339Nano)

	idx.Profiles = append(idx.Profiles, ProfileEntry{
		UUID:              newUUID,
		DisplayName:       displayName,
		CreatedAt:         now,
		Salt:              cryptoParams.Salt,
		EncryptedVerifier: cryptoParams.EncryptedVerifier,
	})

	if _, err := ensureProfileDir(newUUID); err != nil {
		return nil, err
	}

	if err := saveProfileIndex(idx); err != nil {
		dir, derr := profileDir(newUUID)
		if derr == nil {
			_ = os.RemoveAll(dir)
		}
		return nil, err
	}

	a.index = idx
	return a.index, nil
}

func (a *App) SaveEntry(profileUUID string, title string, body string) (int64, error) {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return 0, err
	}

	pdir, err := profileDir(profileUUID)
	if err != nil {
		return 0, err
	}

	db, err := OpenProfileDB(pdir)
	if err != nil {
		return 0, fmt.Errorf("Open profile db: %w", err)
	}
	defer func() { _ = db.Close() }()

	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		return 0, fmt.Errorf("Title is empty")
	}

	// Encrypt sensitive entry content before writing to SQLite.
	// The DB remains readable as SQLite, but title/body are ciphertext.
	encryptedTitle, err := encryptString(a.activeKey, title)
	if err != nil {
		return 0, fmt.Errorf("Encrypt title: %w", err)
	}

	encryptedBody, err := encryptString(a.activeKey, body)
	if err != nil {
		return 0, fmt.Errorf("Encrypt body: %w", err)
	}

	res, err := db.Exec(
		`INSERT INTO entries (title, body) VALUES (?, ?)`,
		encryptedTitle,
		encryptedBody,
	)
	if err != nil {
		return 0, fmt.Errorf("Insert entry: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}

	return id, nil
}

func (a *App) UpdateEntry(profileUUID string, id int64, title string, body string) error {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return err
	}
	if id <= 0 {
		return fmt.Errorf("Invalid id")
	}

	pdir, err := profileDir(profileUUID)
	if err != nil {
		return err
	}

	db, err := OpenProfileDB(pdir)
	if err != nil {
		return fmt.Errorf("Open profile db: %w", err)
	}
	defer func() { _ = db.Close() }()

	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		return fmt.Errorf("Title is empty")
	}

	encryptedTitle, err := encryptString(a.activeKey, title)
	if err != nil {
		return fmt.Errorf("Encrypt title: %w", err)
	}

	encryptedBody, err := encryptString(a.activeKey, body)
	if err != nil {
		return fmt.Errorf("Encrypt body: %w", err)
	}

	res, err := db.Exec(`
		UPDATE entries
		SET title = ?,
			body = ?,
			modified_at = (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		WHERE id = ?
	`, encryptedTitle, encryptedBody, id)
	if err != nil {
		return fmt.Errorf("Update entry: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated rows count: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("Entry not found")
	}

	return nil
}

func (a *App) DeleteEntry(profileUUID string, id int64) error {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return err
	}
	if id <= 0 {
		return fmt.Errorf("Invalid id")
	}

	pdir, err := profileDir(profileUUID)
	if err != nil {
		return err
	}

	db, err := OpenProfileDB(pdir)
	if err != nil {
		return fmt.Errorf("Open profile db: %w", err)
	}
	defer func() { _ = db.Close() }()

	res, err := db.Exec(`
		DELETE FROM entries
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("delete entry: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted rows count: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("Entry not found")
	}

	return nil
}

type EntrySummary struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	CreatedAt  string `json:"createdAt"`
	ModifiedAt string `json:"modifiedAt"`
}

// Titles are stored encrypted, so decrypt them before returning the summaries to frontend.
func (a *App) ListEntries(profileUUID string) ([]EntrySummary, error) {
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
	defer func() { _ = db.Close() }()

	rows, err := db.Query(`
		SELECT id, title, created_at, modified_at
		FROM entries
		ORDER BY modified_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]EntrySummary, 0)
	for rows.Next() {
		var e EntrySummary
		if err := rows.Scan(&e.ID, &e.Title, &e.CreatedAt, &e.ModifiedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		// The row stores encrypted title/body values.
		// Decrypt the title before returning the entry to frontend.
		title, err := decryptString(a.activeKey, e.Title)
		if err != nil {
			return nil, fmt.Errorf("Decrypt entry title: %w", err)
		}
		e.Title = title
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return out, nil
}

type Entry struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	CreatedAt  string `json:"createdAt"`
	ModifiedAt string `json:"modifiedAt"`
}

func (a *App) GetEntry(profileUUID string, id int64) (*Entry, error) {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, fmt.Errorf("Invalid id")
	}

	pdir, err := profileDir(profileUUID)
	if err != nil {
		return nil, err
	}

	db, err := OpenProfileDB(pdir)
	if err != nil {
		return nil, fmt.Errorf("open profile db: %w", err)
	}
	defer func() { _ = db.Close() }()

	var e Entry
	err = db.QueryRow(`
		SELECT id, title, body, created_at, modified_at
		FROM entries
		WHERE id = ?
	`, id).Scan(&e.ID, &e.Title, &e.Body, &e.CreatedAt, &e.ModifiedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Entry not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query entry: %w", err)
	}

	title, err := decryptString(a.activeKey, e.Title)
	if err != nil {
		return nil, fmt.Errorf("Decrypt entry title: %w", err)
	}

	body, err := decryptString(a.activeKey, e.Body)
	if err != nil {
		return nil, fmt.Errorf("Decrypt entry body: %w", err)
	}

	e.Title = title
	e.Body = body

	return &e, nil
}
