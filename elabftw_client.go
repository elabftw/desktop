/*
 * This file is part of eLabFTW Desktop.
 *
 * @author Nicolas <Deltablot>
 * @author Moustapha <Deltablot>
 * @copyright 2026 Deltablot
 * @see https://www.elabftw.net
 * SPDX-License-Identifier: GPL-3.0-or-later
 */
package main

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type elabftwClientConfig struct {
	InstanceID int64
	SiteURL    string
	APIKey     string
	VerifyTLS  bool
}

type ElabftwInfo struct {
	Raw map[string]any `json:"raw"`
}

func (a *App) loadElabftwClientConfig(profileUUID string, instanceID int64) (*elabftwClientConfig, error) {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return nil, err
	}
	if instanceID <= 0 {
		return nil, fmt.Errorf("Invalid instance id")
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

	var cfg elabftwClientConfig
	var encryptedAPIKey string

	err = db.QueryRow(`
		SELECT id, site_url, api_key, verify_tls
		FROM elabftw_instances
		WHERE id = ?
	`, instanceID).Scan(
		&cfg.InstanceID,
		&cfg.SiteURL,
		&encryptedAPIKey,
		&cfg.VerifyTLS,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("eLabFTW instance not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query elabftw instance: %w", err)
	}

	apiKey, err := decryptString(a.activeKey, encryptedAPIKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt api key: %w", err)
	}

	cfg.SiteURL = normalizeElabftwSiteURL(cfg.SiteURL)
	cfg.APIKey = strings.TrimSpace(apiKey)

	if cfg.SiteURL == "" {
		return nil, fmt.Errorf("eLabFTW site URL is empty")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("eLabFTW API key is empty")
	}

	return &cfg, nil
}

func elabftwHTTPClient(verifyTLS bool) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			// Only false when the user explicitly disables TLS verification.
			InsecureSkipVerify: !verifyTLS,
		},
	}

	// timeout prevents the desktop app from hanging forever if the server is unreachable
	// Transport carries our TLS configuration, including whether to verify certificates
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
}

func (a *App) elabftwRequest(
	profileUUID string,
	instanceID int64,
	method string,
	apiPath string,
	body io.Reader,
) (*http.Response, error) {
	cfg, err := a.loadElabftwClientConfig(profileUUID, instanceID)
	if err != nil {
		return nil, err
	}

	apiPath = "/" + strings.TrimLeft(apiPath, "/")
	url := elabftwAPIBaseURL(cfg.SiteURL) + apiPath

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create elabftw request: %w", err)
	}

	req.Header.Set("Authorization", cfg.APIKey)
	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := elabftwHTTPClient(cfg.VerifyTLS).Do(req)
	if err != nil {
		return nil, fmt.Errorf("call elabftw %s %s: %w", method, apiPath, err)
	}

	return resp, nil
}

// http.Response is an open stream, so we need to close it when done reading
func closeResponseBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

// this function is responsible for reading the response body,
// so it also owns closing it before returning
func decodeElabftwJSONResponse(resp *http.Response, target any) error {
	defer closeResponseBody(resp)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			return fmt.Errorf("elabftw returned HTTP %d", resp.StatusCode)
		}
		return fmt.Errorf("elabftw returned HTTP %d: %s", resp.StatusCode, msg)
	}

	if target == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode elabftw response: %w", err)
	}

	return nil
}

func jsonBody(v any) (*bytes.Reader, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal json body: %w", err)
	}
	return bytes.NewReader(b), nil
}

/* ---------- INFO ENDPOINT ---------- */
func (a *App) FetchElabftwInfo(profileUUID string, instanceID int64) (*ElabftwInfo, error) {
	resp, err := a.elabftwRequest(profileUUID, instanceID, http.MethodGet, "/info", nil)
	if err != nil {
		return nil, err
	}

	var out ElabftwInfo
	out.Raw = map[string]any{}

	if err := decodeElabftwJSONResponse(resp, &out.Raw); err != nil {
		return nil, err
	}

	return &out, nil
}
