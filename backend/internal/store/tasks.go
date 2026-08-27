package store

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"img-app/backend/internal/history"
)

func (d *appDatabase) createTask(mode string, input ImageTaskInput) (string, error) {
	id := uuid.NewString()
	if _, err := d.db.Exec(`INSERT INTO image_tasks (id, mode, prompt, model, size, quality, status, created_at) VALUES (?, ?, ?, ?, ?, ?, 'pending', ?)`, id, mode, input.Prompt, input.Model, input.Size, input.Quality, nowString()); err != nil {
		return "", err
	}
	if _, err := d.db.Exec(`DELETE FROM image_tasks WHERE id IN (SELECT id FROM image_tasks ORDER BY created_at DESC, id DESC LIMIT -1 OFFSET ?)`, maxStoredTasks); err != nil {
		return "", err
	}
	return id, nil
}

func (d *appDatabase) completeTask(id, image string) error {
	imageURL, ok := history.NormalizeImageURL(image)
	if !ok {
		imageURL = ""
	}
	_, err := d.db.Exec(`UPDATE image_tasks SET status = 'succeeded', image_url = ?, error_message = '', completed_at = ? WHERE id = ?`, imageURL, nowString(), id)
	return err
}

func (d *appDatabase) historyImageData(id string) (string, error) {
	var image string
	err := d.db.QueryRow(`SELECT image_url FROM image_tasks WHERE id = ? AND status = 'succeeded'`, id).Scan(&image)
	if err == sql.ErrNoRows {
		return "", errNotFound
	}
	if err != nil {
		return "", err
	}
	if !history.IsBase64ImageDataURL(image) {
		return "", errNotFound
	}
	return image, nil
}

func (d *appDatabase) listTasks(limit int, cursor string) (historyPage, error) {
	if limit < 1 {
		limit = 5
	}
	if limit > 5 {
		limit = 5
	}
	args := []any{}
	where := "WHERE status = 'succeeded' AND image_url <> ''"
	if cursor != "" {
		position, err := decodeHistoryCursor(cursor)
		if err != nil {
			return historyPage{}, err
		}
		where += " AND (created_at < ? OR (created_at = ? AND id < ?))"
		args = append(args, position.CreatedAt, position.CreatedAt, position.ID)
	}
	args = append(args, limit+1)
	rows, err := d.db.Query(`SELECT id, mode, prompt, model, size, quality, status, image_url, error_message, created_at, completed_at FROM image_tasks `+where+` ORDER BY created_at DESC, id DESC LIMIT ?`, args...)
	if err != nil {
		return historyPage{}, err
	}
	defer rows.Close()
	page := historyPage{Tasks: make([]imageTask, 0, limit)}
	for rows.Next() {
		var task imageTask
		if err := rows.Scan(&task.ID, &task.Mode, &task.Prompt, &task.Model, &task.Size, &task.Quality, &task.Status, &task.Image, &task.Error, &task.CreatedAt, &task.CompletedAt); err != nil {
			return historyPage{}, err
		}
		task.Image = history.ImageReference(task.ID, task.Image)
		page.Tasks = append(page.Tasks, task)
	}
	if err := rows.Err(); err != nil {
		return historyPage{}, err
	}
	page.HasMore = len(page.Tasks) > limit
	if page.HasMore {
		page.Tasks = page.Tasks[:limit]
		last := page.Tasks[len(page.Tasks)-1]
		page.NextCursor = encodeHistoryCursor(historyCursor{CreatedAt: last.CreatedAt, ID: last.ID})
	}
	return page, nil
}

func (d *appDatabase) deleteTask(id string) error {
	result, err := d.db.Exec(`DELETE FROM image_tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errNotFound
	}
	return nil
}

func (d *appDatabase) listImageSources() ([]string, error) {
	rows, err := d.db.Query(`SELECT image_url FROM image_tasks WHERE status = 'succeeded' AND image_url <> '' ORDER BY created_at DESC, id DESC LIMIT ?`, maxStoredTasks)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sources []string
	for rows.Next() {
		var source string
		if err := rows.Scan(&source); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, rows.Err()
}

func encodeHistoryCursor(cursor historyCursor) string {
	value, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(value)
}

func decodeHistoryCursor(value string) (historyCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return historyCursor{}, fmt.Errorf("invalid history cursor")
	}
	var cursor historyCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil || cursor.ID == "" || cursor.CreatedAt == "" {
		return historyCursor{}, fmt.Errorf("invalid history cursor")
	}
	return cursor, nil
}

func nowString() string { return time.Now().UTC().Format(time.RFC3339Nano) }
