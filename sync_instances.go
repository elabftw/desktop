package main

import (
	"fmt"
	"strings"
)

type ElabftwInstance struct {
	ID        int64  `json:"id"`
	SiteURL   string `json:"siteUrl"`
	APIKey    string `json:"apiKey,omitempty"`
	VerifyTLS bool   `json:"verifyTls"`
}

func (a *App) ListElabftwInstances(profileUUID string) ([]ElabftwInstance, error) {
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
		SELECT id, site_url, verify_tls
		FROM elabftw_instances
		ORDER BY site_url ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query elabftw instances: %w", err)
	}
	defer rows.Close()

	out := []ElabftwInstance{}
	for rows.Next() {
		var inst ElabftwInstance
		if err := rows.Scan(&inst.ID, &inst.SiteURL, &inst.VerifyTLS); err != nil {
			return nil, fmt.Errorf("scan elabftw instance: %w", err)
		}
		out = append(out, inst)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return out, nil
}

func normalizeElabftwSiteURL(siteURL string) string {
	siteURL = strings.TrimSpace(siteURL)
	siteURL = strings.TrimRight(siteURL, "/")
	return siteURL
}

func elabftwAPIBaseURL(siteURL string) string {
	return normalizeElabftwSiteURL(siteURL) + "/api/v2"
}

func (a *App) AddElabftwInstance(profileUUID string, siteURL string, apiKey string, verifyTLS bool) (int64, error) {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return 0, err
	}

	siteURL = normalizeElabftwSiteURL(siteURL)
	apiKey = strings.TrimSpace(apiKey)

	if siteURL == "" {
		return 0, fmt.Errorf("Site URL is empty")
	}
	if apiKey == "" {
		return 0, fmt.Errorf("API key is empty")
	}

	encryptedAPIKey, err := encryptString(a.activeKey, apiKey)
	if err != nil {
		return 0, fmt.Errorf("encrypt api key: %w", err)
	}

	pdir, err := profileDir(profileUUID)
	if err != nil {
		return 0, err
	}

	db, err := OpenProfileDB(pdir)
	if err != nil {
		return 0, fmt.Errorf("open profile db: %w", err)
	}
	defer db.Close()

	res, err := db.Exec(`
		INSERT INTO elabftw_instances (site_url, api_key, verify_tls)
		VALUES (?, ?, ?)
	`, siteURL, encryptedAPIKey, verifyTLS)
	if err != nil {
		return 0, fmt.Errorf("insert elabftw instance: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get instance id: %w", err)
	}

	return id, nil
}

func (a *App) DeleteElabftwInstance(profileUUID string, id int64) error {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return err
	}
	if id <= 0 {
		return fmt.Errorf("Invalid instance id")
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

	res, err := db.Exec(`
		DELETE FROM elabftw_instances
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("delete elabftw instance: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted rows count: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("Instance not found")
	}

	return nil
}

// EDIT/UPDATE ELABFTW Instances

func (a *App) UpdateElabftwInstance(profileUUID string, id int64, siteURL string, apiKey string, verifyTLS bool) error {
	profileUUID, err := a.requireUnlockedProfile(profileUUID)
	if err != nil {
		return err
	}
	if id <= 0 {
		return fmt.Errorf("Invalid instance id")
	}

	siteURL = normalizeElabftwSiteURL(siteURL)
	apiKey = strings.TrimSpace(apiKey)

	if siteURL == "" {
		return fmt.Errorf("Site URL is empty")
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

	if apiKey == "" {
		res, err := db.Exec(`
			UPDATE elabftw_instances
			SET site_url = ?, verify_tls = ?, updated_at = (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
			WHERE id = ?
		`, siteURL, verifyTLS, id)
		if err != nil {
			return fmt.Errorf("update elabftw instance: %w", err)
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("get updated rows count: %w", err)
		}
		if rowsAffected == 0 {
			return fmt.Errorf("Instance not found")
		}

		return nil
	}

	encryptedAPIKey, err := encryptString(a.activeKey, apiKey)
	if err != nil {
		return fmt.Errorf("encrypt api key: %w", err)
	}

	res, err := db.Exec(`
		UPDATE elabftw_instances
		SET site_url = ?, api_key = ?, verify_tls = ?, updated_at = (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		WHERE id = ?
	`, siteURL, encryptedAPIKey, verifyTLS, id)
	if err != nil {
		return fmt.Errorf("update elabftw instance: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated rows count: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("Instance not found")
	}

	return nil
}
