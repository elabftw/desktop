package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// App struct
type App struct {
	ctx               context.Context
	index             *ProfileIndex
	activeProfileUUID string
	passphraseHash    string
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
	if a.activeProfileUUID != profileUUID || a.passphraseHash == "" {
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
	if strings.TrimSpace(passphrase) == "" {
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

	if selected.PassphraseHash == "" {
		return fmt.Errorf("profile has no passphrase set")
	}

	if !verifyPassphrase(passphrase, selected.PassphraseHash) {
		return fmt.Errorf("invalid passphrase")
	}

	a.activeProfileUUID = profileUUID
	a.passphraseHash = selected.PassphraseHash

	return nil
}

// Lock a profile (logout)
func (a *App) LockProfile() {
	a.activeProfileUUID = ""
	a.passphraseHash = ""
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

			if profile.PassphraseHash == "" {
				return nil, fmt.Errorf("profile has no passphrase set")
			}

			if !verifyPassphrase(passphrase, profile.PassphraseHash) {
				return nil, fmt.Errorf("invalid passphrase")
			}

			continue
		}

		filtered = append(filtered, profile)
	}

	if !found {
		return nil, fmt.Errorf("unknown profile uuid")
	}

	idx.Profiles = filtered

	if err := saveProfileIndex(idx); err != nil {
		return nil, err
	}

	dir, err := profileDir(profileUUID)
	if err != nil {
		return nil, err
	}

	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("delete profile dir: %w", err)
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

func (a *App) GetHash() string {
	return a.passphraseHash
}

func (a *App) Login(passphrase string) error {
	hash, err := hashPassphrase(passphrase)
	if err != nil {
		return err
	}

	a.passphraseHash = hash
	return nil
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

func indexJSONPath() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get config dir: %w", err)
	}
	return filepath.Join(cfgDir, "elabftw-desktop", "index.json"), nil
}

func loadDiskIndex(path string) (*diskProfileIndex, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &diskProfileIndex{
				Version:  1,
				Profiles: []diskProfile{},
			}, nil
		}
		return nil, fmt.Errorf("read index.json: %w", err)
	}

	var idx diskProfileIndex
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, fmt.Errorf("parse index.json: %w", err)
	}
	if idx.Version == 0 {
		idx.Version = 1
	}
	if idx.Profiles == nil {
		idx.Profiles = []diskProfile{}
	}
	return &idx, nil
}

func writeDiskIndex(path string, idx *diskProfileIndex) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index.json: %w", err)
	}
	b = append(b, '\n')

	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write index.json: %w", err)
	}
	return nil
}

// create a profile with a passphrase
func (a *App) AddProfile(displayName string, passphrase string) (*ProfileIndex, error) {
	// username
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, fmt.Errorf("display name is empty")
	}

	// passphrase
	passphrase = strings.TrimSpace(passphrase)
	if passphrase == "" {
		return nil, fmt.Errorf("passphrase is empty")
	}

	hash, err := hashPassphrase(passphrase)
	if err != nil {
		return nil, fmt.Errorf("hash passphrase: %w", err)
	}

	idx, err := loadProfileIndex()
	if err != nil {
		return nil, err
	}

	newUUID := uuid.NewString()
	now := time.Now()

	idx.Profiles = append(idx.Profiles, ProfileEntry{
		UUID:           newUUID,
		DisplayName:    displayName,
		CreatedAt:      now,
		LastUsedAt:     time.Time{},
		PassphraseHash: hash,
	})

	if err := saveProfileIndex(idx); err != nil {
		return nil, err
	}

	if _, err := ensureProfileDir(newUUID); err != nil {
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

	// Optional: validate the uuid exists in the loaded index (without depending on field names)
	if a.index != nil {
		found := false
		rv := reflect.ValueOf(a.index).Elem()
		profiles := rv.FieldByName("Profiles")
		if profiles.IsValid() && profiles.Kind() == reflect.Slice {
			for i := 0; i < profiles.Len(); i++ {
				pv := profiles.Index(i)
				if pv.Kind() == reflect.Pointer {
					pv = pv.Elem()
				}
				if pv.IsValid() && pv.Kind() == reflect.Struct {
					f := pv.FieldByName("UUID")
					if !f.IsValid() {
						f = pv.FieldByName("Uuid")
					}
					if f.IsValid() && f.Kind() == reflect.String && strings.TrimSpace(f.String()) == profileUUID {
						found = true
						break
					}
				}
			}
		}
		if !found {
			return 0, fmt.Errorf("unknown profile uuid")
		}
	}

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return 0, fmt.Errorf("get config dir: %w", err)
	}
	profileDir := filepath.Join(cfgDir, "elabftw-desktop", "profiles", profileUUID)

	db, err := OpenProfileDB(profileDir)
	if err != nil {
		return 0, fmt.Errorf("open profile db: %w", err)
	}
	defer func() { _ = db.Close() }()

	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		return 0, fmt.Errorf("title is empty")
	}

	res, err := db.Exec(`INSERT INTO entries (title, body) VALUES (?, ?)`, title, body)
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

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("get config dir: %w", err)
	}
	profileDir := filepath.Join(cfgDir, "elabftw-desktop", "profiles", profileUUID)

	db, err := OpenProfileDB(profileDir)
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

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("get config dir: %w", err)
	}
	profileDir := filepath.Join(cfgDir, "elabftw-desktop", "profiles", profileUUID)

	db, err := OpenProfileDB(profileDir)
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

	return &e, nil
}
func hashPassphrase(passphrase string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword(
		[]byte(passphrase),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

func verifyPassphrase(passphrase, storedHash string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(storedHash),
		[]byte(passphrase),
	)
	return err == nil
}
