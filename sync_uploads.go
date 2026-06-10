/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas CARPi <Deltablot>
 * @author Moustapha Camara <Deltablot>
 * @copyright 2026 Nicolas CARPi
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 *
 * This file handles pushing uploads to an eLabFTW entry
 */

package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

func (a *App) pushUploadToRemoteEntity(
	profileUUID string,
	db *sql.DB,
	instanceID int64,
	entityType string,
	remoteEntityID int64,
	upload StoredUpload,
) error {
	remoteEntityType, err := elabftwUploadEntityType(entityType)
	if err != nil {
		return err
	}

	encryptedPath, err := encryptedProfileUploadPath(profileUUID, upload.Hash)
	if err != nil {
		return err
	}

	encryptedContent, err := os.ReadFile(encryptedPath)
	if err != nil {
		return fmt.Errorf("read encrypted upload: %w", err)
	}

	plaintext, err := decryptRawBytes(a.activeKey, encryptedContent)
	if err != nil {
		return fmt.Errorf("decrypt upload: %w", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	fileWriter, err := writer.CreateFormFile("file", upload.RealName)
	if err != nil {
		return fmt.Errorf("create upload form file: %w", err)
	}

	if _, err := io.Copy(fileWriter, bytes.NewReader(plaintext)); err != nil {
		return fmt.Errorf("write upload form file: %w", err)
	}

	if upload.State != "" {
		if err := writer.WriteField("state", upload.State); err != nil {
			return fmt.Errorf("write upload state: %w", err)
		}
	}

	if upload.LongName != "" && upload.LongName != upload.RealName {
		if err := writer.WriteField("comment", upload.LongName); err != nil {
			return fmt.Errorf("write upload comment: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	path := fmt.Sprintf("/%s/%d/uploads", remoteEntityType, remoteEntityID)

	resp, err := a.elabftwRequest(
    	profileUUID,
    	instanceID,
    	http.MethodPost,
    	path,
    	&body,
    	map[string]string{
    		"Content-Type": writer.FormDataContentType(),
    	},
    )
	if err != nil {
		return err
	}

	if err := decodeElabftwJSONResponse(resp, nil); err != nil {
		return err
	}

	return nil
}

func (a *App) pushEntryUploadsToRemoteEntity(
	profileUUID string,
	db *sql.DB,
	instanceID int64,
	entryID int64,
	entityType string,
	remoteEntityID int64,
) error {
	uploads, err := listEntryUploadsFromDB(db, entryID)
	if err != nil {
		return err
	}

	for _, upload := range uploads {
		if err := a.pushUploadToRemoteEntity(profileUUID, db, instanceID, entityType, remoteEntityID, upload); err != nil {
			return fmt.Errorf("push upload %d: %w", upload.ID, err)
		}
	}

	return nil
}
