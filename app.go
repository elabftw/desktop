package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// App struct
type App struct {
	ctx            context.Context
	index          *ProfileIndex
	passphraseHash string
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
func (a *App) SaveEntry(profileUUID string, title string, body string) (int64, error) {
	fmt.Println("SaveEntry called")
	profileUUID = strings.TrimSpace(profileUUID)
	if profileUUID == "" {
		return 0, fmt.Errorf("profile uuid is empty")
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
	if profileUUID == "" {
		return nil, fmt.Errorf("profile uuid is empty")
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
	if profileUUID == "" {
		return nil, fmt.Errorf("profile uuid is empty")
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
