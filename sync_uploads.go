/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas CARPi <Deltablot>
 * @author Moustapha Camara <Deltablot>
 * @copyright 2026 Nicolas CARPi
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 *
 * Handle upload synchronization with eLabFTW entries.
 */

package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

// pushUploadToRemoteEntity creates one upload on an eLabFTW experiment or item
// and returns the ID assigned to the new remote upload.
func (a *App) pushUploadToRemoteEntity(
	profileUUID string,
	instanceID int64,
	entityType string,
	remoteEntityID int64,
	upload StoredUpload,
) (int64, error) {
	remoteEntityType, err := elabftwUploadEntityType(entityType)
	if err != nil {
		return 0, err
	}

	encryptedPath, err := encryptedProfileUploadPath(
		profileUUID,
		upload.Hash,
	)
	if err != nil {
		return 0, err
	}

	encryptedContent, err := os.ReadFile(encryptedPath)
	if err != nil {
		return 0, fmt.Errorf("read encrypted upload: %w", err)
	}

	plaintext, err := decryptRawBytes(a.activeKey, encryptedContent)
	if err != nil {
		return 0, fmt.Errorf("decrypt upload: %w", err)
	}

	// The encrypted content is no longer needed after decryption.
	encryptedContent = nil
	defer zeroBytes(plaintext)

	// Stream the multipart request body directly to the HTTP request instead of
	// buffering the complete payload in memory
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)
	contentType := writer.FormDataContentType()

	// Report any error that occurs while producing the multipart body.
	writeErr := make(chan error, 1)

	// Build the multipart request body concurrently while the HTTP client reads
	// from the pipe. This avoids creating a second full copy of the upload.
	go func() {
		fileWriter, err := writer.CreateFormFile("file", upload.RealName)
		if err != nil {
			_ = pipeWriter.CloseWithError(
				fmt.Errorf("create upload form file: %w", err),
			)
			writeErr <- err
			return
		}

		if _, err := io.Copy(fileWriter, bytes.NewReader(plaintext)); err != nil {
			_ = pipeWriter.CloseWithError(
				fmt.Errorf("write upload form file: %w", err),
			)
			writeErr <- err
			return
		}

		if err := writer.Close(); err != nil {
			_ = pipeWriter.CloseWithError(
				fmt.Errorf("close multipart writer: %w", err),
			)
			writeErr <- err
			return
		}

		_ = pipeWriter.Close()
		writeErr <- nil
	}()

	path := fmt.Sprintf(
		"/%s/%d/uploads",
		remoteEntityType,
		remoteEntityID,
	)

	resp, err := a.elabftwRequest(
		profileUUID,
		instanceID,
		http.MethodPost,
		path,
		pipeReader,
		map[string]string{
			"Content-Type": contentType,
		},
	)
	if err != nil {
		_ = pipeReader.Close()
		<-writeErr
		return 0, err
	}

	if err := <-writeErr; err != nil {
		_ = resp.Body.Close()
		return 0, err
	}

	// The new upload ID is returned through the Location header
	location := resp.Header.Get("Location")

	if err := decodeElabftwJSONResponse(resp, nil); err != nil {
		return 0, err
	}

	remoteUploadID, err := remoteIDFromLocation(location)
	if err != nil {
		return 0, fmt.Errorf("parse remote upload id: %w", err)
	}

	return remoteUploadID, nil
}

// pushEntryUploadsToRemoteEntity pushes uploads attached to an entry that do
// not already have a remote mapping. Failures are returned as warnings because
// the parent entry may already have been successfully posted or patched
func (a *App) pushEntryUploadsToRemoteEntity(
	profileUUID string,
	db *sql.DB,
	instanceID int64,
	entryID int64,
	entityType string,
	remoteEntityID int64,
) []string {
	uploads, err := listEntryUploadsFromDB(db, entryID)
	if err != nil {
		return []string{err.Error()}
	}

	warnings := make([]string, 0)

	for _, upload := range uploads {
		_, alreadyPushed, err := findRemoteUploadID(
			db,
			instanceID,
			upload.ID,
			remoteEntityID,
			entityType,
		)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}

		// This local upload is already attached to this remote entity.
		if alreadyPushed {
			continue
		}

		remoteUploadID, err := a.pushUploadToRemoteEntity(
			profileUUID,
			instanceID,
			entityType,
			remoteEntityID,
			upload,
		)
		if err != nil {
			warnings = append(
				warnings,
				fmt.Sprintf(
					"upload %q failed: %s",
					upload.RealName,
					err,
				),
			)
			continue
		}

		if err := rememberRemoteUpload(
			db,
			instanceID,
			entryID,
			upload.ID,
			remoteEntityID,
			remoteUploadID,
			entityType,
		); err != nil {
			warnings = append(
				warnings,
				fmt.Sprintf(
					"remember remote upload %q: %s",
					upload.RealName,
					err,
				),
			)
		}
	}

	return warnings
}

// findRemoteUploadID checks whether a local upload has already been pushed to
// the specified remote entity
func findRemoteUploadID(
	db *sql.DB,
	instanceID int64,
	localUploadID int64,
	remoteEntityID int64,
	entityType string,
) (int64, bool, error) {
	var remoteUploadID int64

	err := db.QueryRow(`
		SELECT remote_upload_id
		FROM upload2remote
		WHERE instance = ?
		  AND local_upload_id = ?
		  AND remote_entity_id = ?
		  AND type = ?
	`,
		instanceID,
		localUploadID,
		remoteEntityID,
		entityType,
	).Scan(&remoteUploadID)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("query upload2remote: %w", err)
	}

	return remoteUploadID, true, nil
}

// rememberRemoteUpload records the correspondence between a local upload and
// the remote upload created on an eLabFTW experiment or item
func rememberRemoteUpload(
	db *sql.DB,
	instanceID int64,
	entryID int64,
	localUploadID int64,
	remoteEntityID int64,
	remoteUploadID int64,
	entityType string,
) error {
	_, err := db.Exec(`
		INSERT INTO upload2remote (
			instance,
			local_upload_id,
			local_entry_id,
			remote_entity_id,
			remote_upload_id,
			type
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		instanceID,
		localUploadID,
		entryID,
		remoteEntityID,
		remoteUploadID,
		entityType,
	)
	if err != nil {
		return fmt.Errorf("insert upload2remote: %w", err)
	}

	return nil
}
