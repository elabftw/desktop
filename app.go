package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"crypto/ed25519"
	"github.com/google/uuid"
)

// App struct
type App struct {
	ctx               context.Context
	index             *ProfileIndex
	activeProfileUUID string
	activeKey         []byte
	activePrivateKey  ed25519.PrivateKey
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

// require unlocked profile to perform any authentified actions
func (a *App) requireUnlockedProfile(profileUUID string) error {
	profileUUID = strings.TrimSpace(profileUUID)
	if profileUUID == "" {
		return fmt.Errorf("profile uuid is empty")
	}
	if a.activeProfileUUID != profileUUID || len(a.activeKey) == 0 {
		return fmt.Errorf("profile is locked")
	}
	return nil
}

// Unlock a profile (login)
func (a *App) UnlockProfile(profileUUID string, passphrase string) error {
	profileUUID = strings.TrimSpace(profileUUID)
	if profileUUID == "" {
		return fmt.Errorf("profile uuid is empty")
	}
	passphrase = strings.TrimSpace(passphrase)
	if passphrase == "" {
		return fmt.Errorf("passphrase is empty")
	}

	if a.index == nil {
		return fmt.Errorf("profile index is not loaded")
	}

	var selected *ProfileEntry
	for i := range a.index.Profiles {
		if a.index.Profiles[i].UUID == profileUUID {
			selected = &a.index.Profiles[i]
			break
		}
	}

	if selected == nil {
		return fmt.Errorf("unknown profile uuid")
	}

	key, privateKey, err := unlockProfileSecrets(selected, passphrase)
	if err != nil {
		return err
	}

	a.LockProfile()

	a.activeProfileUUID = profileUUID
	a.activeKey = key
	a.activePrivateKey = privateKey

	return nil
}

// Lock a profile (logout)
func (a *App) LockProfile() {
	a.activeProfileUUID = ""

	if a.activeKey != nil {
		zeroBytes(a.activeKey)
	}
	a.activeKey = nil

	if a.activePrivateKey != nil {
		zeroBytes(a.activePrivateKey)
	}
	a.activePrivateKey = nil
}

// DEV: Delete a profile (requires passphrase)
func (a *App) DeleteProfile(profileUUID string, passphrase string) (*ProfileIndex, error) {
	profileUUID = strings.TrimSpace(profileUUID)
	if profileUUID == "" {
		return nil, fmt.Errorf("profile uuid is empty")
	}

	passphrase = strings.TrimSpace(passphrase)
	if passphrase == "" {
		return nil, fmt.Errorf("passphrase is empty")
	}

	idx, err := loadProfileIndex()
	if err != nil {
		return nil, err
	}

	found := false
	filtered := make([]ProfileEntry, 0, len(idx.Profiles))

	// check passphrase is correct
	for _, profile := range idx.Profiles {
		if profile.UUID == profileUUID {
			found = true

			key, privateKey, err := unlockProfileSecrets(&profile, passphrase)
			if err != nil {
				return nil, err
			}
			zeroBytes(key)
			zeroBytes(privateKey)

			continue
		}

		filtered = append(filtered, profile)
	}

	if !found {
		return nil, fmt.Errorf("unknown profile uuid")
	}

	dir, err := profileDir(profileUUID)
	if err != nil {
		return nil, err
	}

	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("delete profile dir: %w", err)
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
	UUID        string    `json:"uuid"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
}

// create a profile
func (a *App) AddProfile(displayName string, passphrase string) (*ProfileIndex, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, fmt.Errorf("display name is empty")
	}

	passphrase = strings.TrimSpace(passphrase)
	if passphrase == "" {
		return nil, fmt.Errorf("passphrase is empty")
	}

	secrets, err := createProfileSecrets(passphrase)
	if err != nil {
		return nil, err
	}

	idx, err := loadProfileIndex()
	if err != nil {
		return nil, err
	}

	newUUID := uuid.NewString()
	now := time.Now()

	idx.Profiles = append(idx.Profiles, ProfileEntry{
		UUID:                newUUID,
		DisplayName:         displayName,
		CreatedAt:           now,
		LastUsedAt:          time.Time{},
		PublicKey:           secrets.PublicKey,
		KeySalt:             secrets.KeySalt,
		EncryptedPrivateKey: secrets.EncryptedPrivateKey,
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
	profileUUID = strings.TrimSpace(profileUUID)
	if err := a.requireUnlockedProfile(profileUUID); err != nil {
		return 0, err
	}

	pdir, err := profileDir(profileUUID)
	if err != nil {
		return 0, err
	}

	db, err := OpenProfileDB(pdir)
	if err != nil {
		return 0, fmt.Errorf("open profile db: %w", err)
	}
	defer func() { _ = db.Close() }()

	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		return 0, fmt.Errorf("title is empty")
	}

	encryptedTitle, err := encryptString(a.activeKey, title)
	if err != nil {
		return 0, fmt.Errorf("encrypt title: %w", err)
	}

	encryptedBody, err := encryptString(a.activeKey, body)
	if err != nil {
		return 0, fmt.Errorf("encrypt body: %w", err)
	}

	// 	res, err := db.Exec(`INSERT INTO entries (title, body) VALUES (?, ?)`, title, body)
	res, err := db.Exec(
		`INSERT INTO entries (title, body) VALUES (?, ?)`,
		encryptedTitle,
		encryptedBody,
	)
	if err != nil {
		return 0, fmt.Errorf("insert entry: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}

	return id, nil
}

type EntrySummary struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func (a *App) ListEntries(profileUUID string) ([]EntrySummary, error) {
	profileUUID = strings.TrimSpace(profileUUID)
	if err := a.requireUnlockedProfile(profileUUID); err != nil {
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
		SELECT id, title, created_at, updated_at
		FROM entries
		ORDER BY updated_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]EntrySummary, 0)
	for rows.Next() {
		var e EntrySummary
		if err := rows.Scan(&e.ID, &e.Title, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		title, err := decryptString(a.activeKey, e.Title)
		if err != nil {
			return nil, fmt.Errorf("decrypt entry title: %w", err)
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
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func (a *App) GetEntry(profileUUID string, id int64) (*Entry, error) {
	profileUUID = strings.TrimSpace(profileUUID)
	if err := a.requireUnlockedProfile(profileUUID); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, fmt.Errorf("invalid id")
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
		SELECT id, title, body, created_at, updated_at
		FROM entries
		WHERE id = ?
	`, id).Scan(&e.ID, &e.Title, &e.Body, &e.CreatedAt, &e.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("entry not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query entry: %w", err)
	}

	title, err := decryptString(a.activeKey, e.Title)
	if err != nil {
		return nil, fmt.Errorf("decrypt entry title: %w", err)
	}

	body, err := decryptString(a.activeKey, e.Body)
	if err != nil {
		return nil, fmt.Errorf("decrypt entry body: %w", err)
	}

	e.Title = title
	e.Body = body

	return &e, nil
}
