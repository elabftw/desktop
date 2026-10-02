/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas CARPi <Deltablot>
 * @author Moustapha Camara <Deltablot>
 * @copyright 2026 Deltablot
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPullEntryFromElabftwReplacesLocalDataAndKeepsPushBaseline(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	profileUUID := "pull-test-profile"
	app := &App{
		activeProfileUUID: profileUUID,
		activeKey:         bytes.Repeat([]byte{0x42}, 32),
	}

	entryID, err := app.SaveEntry(profileUUID, "Local title", "Local body")
	if err != nil {
		t.Fatalf("SaveEntry: %v", err)
	}

	oldContent := []byte("old local upload")
	oldSource := filepath.Join(t.TempDir(), "old.txt")
	if err := os.WriteFile(oldSource, oldContent, 0o600); err != nil {
		t.Fatalf("write old upload: %v", err)
	}
	oldUpload, err := app.ImportUpload(profileUUID, entryID, oldSource)
	if err != nil {
		t.Fatalf("ImportUpload: %v", err)
	}
	oldEncryptedPath, err := encryptedProfileUploadPath(profileUUID, oldUpload.Hash)
	if err != nil {
		t.Fatalf("old upload path: %v", err)
	}

	remoteContent := []byte("content downloaded from eLabFTW")
	remoteHash := fileSHA256(remoteContent)
	pullModifiedAt := "2026-10-01T12:00:00Z"
	pushModifiedAt := "2026-10-01T12:05:00Z"

	var patched bool
	var patchPayload map[string]any
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "test-api-key" {
			t.Errorf("missing API key header")
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/experiments/42":
			modifiedAt := pullModifiedAt
			if patched {
				modifiedAt = pushModifiedAt
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":           42,
				"page":         "experiments",
				"title":        "Remote title",
				"body":         "# Remote body\n\nPulled from eLabFTW.",
				"content_type": 2,
				"modified_at":  modifiedAt,
				"uploads": []map[string]any{
					{
						"id":        7,
						"real_name": "remote.txt",
						// Pull deliberately ignores remote hash/size metadata and
						// stores exactly the bytes returned by format=binary.
						"hash":           strings.Repeat("0", 64),
						"hash_algorithm": "sha256",
						"filesize":       len(remoteContent) + 123,
					},
				},
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/experiments/42/uploads/7":
			if r.URL.Query().Get("format") != "binary" {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 7, "real_name": "remote.txt"})
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(remoteContent)

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v2/experiments/42":
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&patchPayload); err != nil {
				t.Errorf("decode push payload: %v", err)
			}
			patched = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{}`)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	instanceID, err := app.AddElabftwInstance(profileUUID, server.URL, "test-api-key", false)
	if err != nil {
		t.Fatalf("AddElabftwInstance: %v", err)
	}

	pdir, err := profileDir(profileUUID)
	if err != nil {
		t.Fatalf("profileDir: %v", err)
	}
	db, err := OpenProfileDB(pdir)
	if err != nil {
		t.Fatalf("OpenProfileDB: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		INSERT INTO local2remote (instance, remote_id, local_id, type, modified_at)
		VALUES (?, ?, ?, ?, ?)
	`, instanceID, 42, entryID, "experiment", "2026-09-30T12:00:00Z"); err != nil {
		t.Fatalf("insert local2remote: %v", err)
	}

	result, err := app.PullEntryFromElabftw(profileUUID, entryID, instanceID, "experiment")
	if err != nil {
		t.Fatalf("PullEntryFromElabftw: %v", err)
	}
	if result.RemoteID != 42 || result.Uploads != 1 || result.Type != "experiment" {
		t.Fatalf("unexpected pull result: %+v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("unexpected pull warnings: %+v", result.Warnings)
	}

	entry, err := app.GetEntry(profileUUID, entryID)
	if err != nil {
		t.Fatalf("GetEntry after pull: %v", err)
	}
	if entry.Title != "Remote title" {
		t.Fatalf("title = %q, want remote title", entry.Title)
	}
	if entry.Body != "# Remote body\n\nPulled from eLabFTW." {
		t.Fatalf("body = %q", entry.Body)
	}

	uploads, err := app.ListEntryUploads(profileUUID, entryID)
	if err != nil {
		t.Fatalf("ListEntryUploads: %v", err)
	}
	if len(uploads) != 1 {
		t.Fatalf("uploads = %d, want 1", len(uploads))
	}
	if uploads[0].RealName != "remote.txt" || uploads[0].State != "synced" || uploads[0].Hash != remoteHash {
		t.Fatalf("unexpected upload: %+v", uploads[0])
	}

	remoteEncryptedPath, err := encryptedProfileUploadPath(profileUUID, remoteHash)
	if err != nil {
		t.Fatalf("remote upload path: %v", err)
	}
	encryptedRemoteContent, err := os.ReadFile(remoteEncryptedPath)
	if err != nil {
		t.Fatalf("read pulled encrypted upload: %v", err)
	}
	plaintext, err := decryptRawBytes(app.activeKey, encryptedRemoteContent)
	if err != nil {
		t.Fatalf("decrypt pulled upload: %v", err)
	}
	if !bytes.Equal(plaintext, remoteContent) {
		t.Fatalf("pulled upload content = %q", plaintext)
	}
	zeroBytes(plaintext)

	if _, err := os.Stat(oldEncryptedPath); !os.IsNotExist(err) {
		t.Fatalf("old unreferenced upload still exists, stat err = %v", err)
	}

	var syncedAt string
	if err := db.QueryRow(`
		SELECT modified_at FROM local2remote
		WHERE instance = ? AND local_id = ? AND type = ?
	`, instanceID, entryID, "experiment").Scan(&syncedAt); err != nil {
		t.Fatalf("query sync timestamp: %v", err)
	}
	parsedSyncedAt, err := parseLocalModifiedAt(syncedAt)
	if err != nil {
		t.Fatalf("parse sync timestamp: %v", err)
	}
	wantSyncedAt, _ := time.Parse(time.RFC3339, pullModifiedAt)
	if !parsedSyncedAt.Equal(wantSyncedAt) {
		t.Fatalf("sync timestamp = %s, want %s", parsedSyncedAt, wantSyncedAt)
	}

	var mappedRemoteUploadID int64
	if err := db.QueryRow(`
		SELECT remote_upload_id FROM upload2remote
		WHERE instance = ? AND local_entry_id = ? AND type = ?
	`, instanceID, entryID, "experiment").Scan(&mappedRemoteUploadID); err != nil {
		t.Fatalf("query upload mapping: %v", err)
	}
	if mappedRemoteUploadID != 7 {
		t.Fatalf("remote upload mapping = %d, want 7", mappedRemoteUploadID)
	}

	if err := app.UpdateEntry(profileUUID, entryID, "Edited locally", "Changed after pull"); err != nil {
		t.Fatalf("UpdateEntry after pull: %v", err)
	}
	if _, err := app.PushEntryToElabftw(profileUUID, entryID, instanceID, "experiment", false); err != nil {
		t.Fatalf("PushEntryToElabftw after pull: %v", err)
	}
	if !patched {
		t.Fatal("expected push to PATCH the existing remote entry")
	}
	if got := fmt.Sprint(patchPayload["title"]); got != "Edited locally" {
		t.Fatalf("pushed title = %q", got)
	}
}

func TestRemoteEntryBodyToMarkdown(t *testing.T) {
	input := `<h2>Heading</h2><p>Hello <strong>world</strong>.<br><a href="https://example.com">Link</a></p><ul><li>one</li><li>two</li></ul>`

	got, err := remoteEntryBodyToMarkdown(input, 1)
	if err != nil {
		t.Fatalf("remoteEntryBodyToMarkdown: %v", err)
	}

	for _, want := range []string{"## Heading", "Hello **world**.", "[Link](https://example.com)", "- one", "- two"} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown %q does not contain %q", got, want)
		}
	}
}

func TestValidatePulledRemoteIdentity(t *testing.T) {
	tests := []struct {
		name       string
		remote     remoteEntry
		remoteID   int64
		entityType string
		wantError  bool
	}{
		{name: "experiment", remote: remoteEntry{ID: 42, Page: "experiments"}, remoteID: 42, entityType: "experiment"},
		{name: "resource", remote: remoteEntry{ID: 7, Page: "database"}, remoteID: 7, entityType: "resource"},
		{name: "wrong id", remote: remoteEntry{ID: 43, Page: "experiments"}, remoteID: 42, entityType: "experiment", wantError: true},
		{name: "wrong type", remote: remoteEntry{ID: 42, Page: "database"}, remoteID: 42, entityType: "experiment", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePulledRemoteIdentity(tt.remote, tt.remoteID, tt.entityType)
			if (err != nil) != tt.wantError {
				t.Fatalf("validatePulledRemoteIdentity() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestReplaceLocalEntryFromRemoteRequiresExistingMapping(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	profileUUID := "mapping-test-profile"
	pdir, err := profileDir(profileUUID)
	if err != nil {
		t.Fatalf("profileDir: %v", err)
	}
	db, err := OpenProfileDB(pdir)
	if err != nil {
		t.Fatalf("OpenProfileDB: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`INSERT INTO entries (id, title, body) VALUES (1, 'old-title', 'old-body')`); err != nil {
		t.Fatalf("insert local entry: %v", err)
	}

	_, err = replaceLocalEntryFromRemote(
		db,
		1,
		1,
		"experiment",
		42,
		"new-title",
		"new-body",
		time.Now().UTC().Format(time.RFC3339Nano),
		nil,
	)
	if err == nil {
		t.Fatal("expected an error without an existing remote mapping")
	}
	if err == sql.ErrNoRows {
		t.Fatalf("unexpected raw sql.ErrNoRows: %v", err)
	}

	var title string
	if err := db.QueryRow(`SELECT title FROM entries WHERE id = 1`).Scan(&title); err != nil {
		t.Fatalf("query rolled back entry: %v", err)
	}
	if title != "old-title" {
		t.Fatalf("entry update was not rolled back: title = %q", title)
	}
}
