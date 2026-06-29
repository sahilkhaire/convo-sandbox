package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AccountRepo struct {
	db *DB
}

func NewAccountRepo(db *DB) *AccountRepo {
	return &AccountRepo{db: db}
}

func scanAccount(row pgx.Row) (Account, error) {
	var a Account
	err := row.Scan(
		&a.ID, &a.Channel, &a.Name, &a.ExternalID, &a.ClientSecret, &a.AccessToken,
		&a.WebhookURL, &a.WebhookVerifyToken, &a.WabaID, &a.DisplayPhone,
		&a.SentDelayMs, &a.DeliveredDelayMs, &a.ReadDelayMs, &a.FailureRate, &a.AutoRead,
		&a.Capabilities, &a.CreatedAt, &a.UpdatedAt,
	)
	return a, err
}

const accountCols = `id, channel, name, external_id, client_secret, access_token, webhook_url,
	webhook_verify_token, waba_id, display_phone, sent_delay_ms, delivered_delay_ms, read_delay_ms,
	failure_rate, auto_read, capabilities, created_at, updated_at`

func (r *AccountRepo) Create(ctx context.Context, a *Account) error {
	if a.Capabilities == nil {
		a.Capabilities = json.RawMessage(`[]`)
	}
	return r.db.Pool.QueryRow(ctx, `
		INSERT INTO accounts (channel, name, external_id, client_secret, access_token, webhook_url,
			webhook_verify_token, waba_id, display_phone, sent_delay_ms, delivered_delay_ms, read_delay_ms,
			failure_rate, auto_read, capabilities)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING `+accountCols,
		a.Channel, a.Name, a.ExternalID, a.ClientSecret, a.AccessToken, a.WebhookURL,
		a.WebhookVerifyToken, a.WabaID, a.DisplayPhone, a.SentDelayMs, a.DeliveredDelayMs,
		a.ReadDelayMs, a.FailureRate, a.AutoRead, a.Capabilities,
	).Scan(
		&a.ID, &a.Channel, &a.Name, &a.ExternalID, &a.ClientSecret, &a.AccessToken,
		&a.WebhookURL, &a.WebhookVerifyToken, &a.WabaID, &a.DisplayPhone,
		&a.SentDelayMs, &a.DeliveredDelayMs, &a.ReadDelayMs, &a.FailureRate, &a.AutoRead,
		&a.Capabilities, &a.CreatedAt, &a.UpdatedAt,
	)
}

func (r *AccountRepo) List(ctx context.Context) ([]Account, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT `+accountCols+` FROM accounts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AccountRepo) GetByID(ctx context.Context, id uuid.UUID) (Account, error) {
	row := r.db.Pool.QueryRow(ctx, `SELECT `+accountCols+` FROM accounts WHERE id=$1`, id)
	a, err := scanAccount(row)
	if err == pgx.ErrNoRows {
		return Account{}, fmt.Errorf("account not found")
	}
	return a, err
}

func (r *AccountRepo) GetByExternalID(ctx context.Context, channel, externalID string) (Account, error) {
	row := r.db.Pool.QueryRow(ctx, `SELECT `+accountCols+` FROM accounts WHERE channel=$1 AND external_id=$2`, channel, externalID)
	a, err := scanAccount(row)
	if err == pgx.ErrNoRows {
		return Account{}, fmt.Errorf("account not found")
	}
	return a, err
}

func (r *AccountRepo) Update(ctx context.Context, a Account) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE accounts SET name=$2, webhook_url=$3, webhook_verify_token=$4, client_secret=$5,
			access_token=$6, waba_id=$7, display_phone=$8, sent_delay_ms=$9, delivered_delay_ms=$10,
			read_delay_ms=$11, failure_rate=$12, auto_read=$13, capabilities=$14, updated_at=NOW()
		WHERE id=$1`,
		a.ID, a.Name, a.WebhookURL, a.WebhookVerifyToken, a.ClientSecret, a.AccessToken,
		a.WabaID, a.DisplayPhone, a.SentDelayMs, a.DeliveredDelayMs, a.ReadDelayMs,
		a.FailureRate, a.AutoRead, a.Capabilities,
	)
	return err
}

func (r *AccountRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM accounts WHERE id=$1`, id)
	return err
}
