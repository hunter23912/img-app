package store

import (
	"database/sql"
	"fmt"
	"strings"
)

func (d *appDatabase) getImageSettings() (imageSettings, error) {
	var settings imageSettings
	err := d.db.QueryRow(`SELECT endpoint, api_key, updated_at FROM image_settings WHERE id = 1`).Scan(
		&settings.Endpoint,
		&settings.APIKey,
		&settings.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return imageSettings{}, nil
	}
	if err != nil {
		return imageSettings{}, fmt.Errorf("read image settings: %w", err)
	}
	return settings, nil
}

func (d *appDatabase) saveImageSettings(endpoint, apiKey string) (imageSettings, error) {
	settings := imageSettings{
		Endpoint:  strings.TrimSpace(endpoint),
		APIKey:    strings.TrimSpace(apiKey),
		UpdatedAt: nowString(),
	}
	_, err := d.db.Exec(`
		INSERT INTO image_settings (id, endpoint, api_key, updated_at)
		VALUES (1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			endpoint = excluded.endpoint,
			api_key = excluded.api_key,
			updated_at = excluded.updated_at
	`, settings.Endpoint, settings.APIKey, settings.UpdatedAt)
	if err != nil {
		return imageSettings{}, fmt.Errorf("save image settings: %w", err)
	}
	return settings, nil
}
