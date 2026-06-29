package store

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type MediaRepo struct {
	db *DB
}

func NewMediaRepo(db *DB) *MediaRepo {
	return &MediaRepo{db: db}
}

func (r *MediaRepo) Create(ctx context.Context, m *MediaFile) error {
	return r.db.Pool.QueryRow(ctx, `
		INSERT INTO media_files (account_id, file_id, mime_type, file_name, size_bytes, storage_path)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at`,
		m.AccountID, m.FileID, m.MimeType, m.FileName, m.SizeBytes, m.StoragePath,
	).Scan(&m.ID, &m.CreatedAt)
}

func (r *MediaRepo) GetByFileID(ctx context.Context, fileID string) (MediaFile, error) {
	var m MediaFile
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, account_id, file_id, mime_type, file_name, size_bytes, storage_path, created_at
		FROM media_files WHERE file_id=$1`, fileID,
	).Scan(&m.ID, &m.AccountID, &m.FileID, &m.MimeType, &m.FileName, &m.SizeBytes, &m.StoragePath, &m.CreatedAt)
	return m, err
}

func (r *MediaRepo) ListPaths(ctx context.Context) ([]string, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT storage_path FROM media_files`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, rows.Err()
}

type WebhookRepo struct {
	db *DB
}

func NewWebhookRepo(db *DB) *WebhookRepo {
	return &WebhookRepo{db: db}
}

func (r *WebhookRepo) Create(ctx context.Context, w *WebhookDelivery) error {
	return r.db.Pool.QueryRow(ctx, `
		INSERT INTO webhook_deliveries (account_id, channel, event_type, payload, http_status, response_body, error_message, attempts)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, created_at`,
		w.AccountID, w.Channel, w.EventType, w.Payload, w.HTTPStatus, w.ResponseBody, w.ErrorMessage, w.Attempts,
	).Scan(&w.ID, &w.CreatedAt)
}

func (r *WebhookRepo) List(ctx context.Context, limit int) ([]WebhookDelivery, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, account_id, channel, event_type, payload, http_status, response_body, error_message, attempts, created_at
		FROM webhook_deliveries ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WebhookDelivery
	for rows.Next() {
		var w WebhookDelivery
		if err := rows.Scan(&w.ID, &w.AccountID, &w.Channel, &w.EventType, &w.Payload,
			&w.HTTPStatus, &w.ResponseBody, &w.ErrorMessage, &w.Attempts, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

type DeliveryJobRepo struct {
	db *DB
}

func NewDeliveryJobRepo(db *DB) *DeliveryJobRepo {
	return &DeliveryJobRepo{db: db}
}

func (r *DeliveryJobRepo) Create(ctx context.Context, j *DeliveryJob) error {
	return r.db.Pool.QueryRow(ctx, `
		INSERT INTO delivery_jobs (account_id, message_id, channel, event_type, payload, scheduled_at, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at`,
		j.AccountID, j.MessageID, j.Channel, j.EventType, j.Payload, j.ScheduledAt, j.Status,
	).Scan(&j.ID, &j.CreatedAt)
}

func (r *DeliveryJobRepo) ClaimPending(ctx context.Context, limit int) ([]DeliveryJob, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id, account_id, message_id, channel, event_type, payload, scheduled_at, status, attempts, created_at
		FROM delivery_jobs
		WHERE status='pending' AND scheduled_at <= NOW()
		ORDER BY scheduled_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []DeliveryJob
	var ids []uuid.UUID
	for rows.Next() {
		var j DeliveryJob
		if err := rows.Scan(&j.ID, &j.AccountID, &j.MessageID, &j.Channel, &j.EventType,
			&j.Payload, &j.ScheduledAt, &j.Status, &j.Attempts, &j.CreatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
		ids = append(ids, j.ID)
	}
	if len(ids) == 0 {
		return nil, tx.Commit(ctx)
	}
	for _, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE delivery_jobs SET status='processing' WHERE id=$1`, id); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *DeliveryJobRepo) MarkDone(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE delivery_jobs SET status='done' WHERE id=$1`, id)
	return err
}

func (r *DeliveryJobRepo) MarkFailed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE delivery_jobs SET status='failed', attempts=attempts+1 WHERE id=$1`, id)
	return err
}

func (r *DeliveryJobRepo) CancelAll(ctx context.Context) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE delivery_jobs SET status='cancelled' WHERE status IN ('pending','processing')`)
	return err
}

type TemplateRepo struct {
	db *DB
}

func NewTemplateRepo(db *DB) *TemplateRepo {
	return &TemplateRepo{db: db}
}

func (r *TemplateRepo) Create(ctx context.Context, t *Template) error {
	if t.Components == nil {
		t.Components = json.RawMessage(`[]`)
	}
	return r.db.Pool.QueryRow(ctx, `
		INSERT INTO templates (account_id, name, language, category, components)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at`,
		t.AccountID, t.Name, t.Language, t.Category, t.Components,
	).Scan(&t.ID, &t.CreatedAt)
}

func (r *TemplateRepo) GetByName(ctx context.Context, accountID uuid.UUID, name, language string) (Template, error) {
	var t Template
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, account_id, name, language, category, components, created_at
		FROM templates WHERE account_id=$1 AND name=$2 AND language=$3`,
		accountID, name, language,
	).Scan(&t.ID, &t.AccountID, &t.Name, &t.Language, &t.Category, &t.Components, &t.CreatedAt)
	return t, err
}

func (r *TemplateRepo) ListByAccount(ctx context.Context, accountID uuid.UUID) ([]Template, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, account_id, name, language, category, components, created_at
		FROM templates WHERE account_id=$1`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.AccountID, &t.Name, &t.Language, &t.Category, &t.Components, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

type DataRepo struct {
	db *DB
}

func NewDataRepo(db *DB) *DataRepo {
	return &DataRepo{db: db}
}

func (r *DataRepo) PurgeMessagesOnly(ctx context.Context) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	stmts := []string{
		`UPDATE delivery_jobs SET status='cancelled' WHERE status IN ('pending','processing')`,
		`DELETE FROM delivery_jobs`,
		`DELETE FROM webhook_deliveries`,
		`DELETE FROM messages`,
		`DELETE FROM conversations`,
		`DELETE FROM media_files`,
		`DELETE FROM templates`,
	}
	for _, s := range stmts {
		if _, err := tx.Exec(ctx, s); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *DataRepo) PurgeAll(ctx context.Context) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	stmts := []string{
		`DELETE FROM delivery_jobs`,
		`DELETE FROM webhook_deliveries`,
		`DELETE FROM messages`,
		`DELETE FROM conversations`,
		`DELETE FROM media_files`,
		`DELETE FROM templates`,
		`DELETE FROM accounts`,
	}
	for _, s := range stmts {
		if _, err := tx.Exec(ctx, s); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
