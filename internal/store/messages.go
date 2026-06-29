package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type MessageRepo struct {
	db *DB
}

func NewMessageRepo(db *DB) *MessageRepo {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) Create(ctx context.Context, m *Message) error {
	if m.Body == nil {
		m.Body = json.RawMessage(`{}`)
	}
	return r.db.Pool.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, account_id, direction, vendor_message_id, message_type, status, body)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at, updated_at`,
		m.ConversationID, m.AccountID, m.Direction, m.VendorMessageID, m.MessageType, m.Status, m.Body,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

func (r *MessageRepo) ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]Message, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, conversation_id, account_id, direction, vendor_message_id, message_type, status, body, created_at, updated_at
		FROM messages WHERE conversation_id=$1 ORDER BY created_at ASC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.AccountID, &m.Direction, &m.VendorMessageID,
			&m.MessageType, &m.Status, &m.Body, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *MessageRepo) GetByID(ctx context.Context, id uuid.UUID) (Message, error) {
	var m Message
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, conversation_id, account_id, direction, vendor_message_id, message_type, status, body, created_at, updated_at
		FROM messages WHERE id=$1`, id,
	).Scan(&m.ID, &m.ConversationID, &m.AccountID, &m.Direction, &m.VendorMessageID,
		&m.MessageType, &m.Status, &m.Body, &m.CreatedAt, &m.UpdatedAt)
	if err == pgx.ErrNoRows {
		return Message{}, fmt.Errorf("message not found")
	}
	return m, err
}

func (r *MessageRepo) GetByVendorID(ctx context.Context, vendorID string) (Message, error) {
	var m Message
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, conversation_id, account_id, direction, vendor_message_id, message_type, status, body, created_at, updated_at
		FROM messages WHERE vendor_message_id=$1`, vendorID,
	).Scan(&m.ID, &m.ConversationID, &m.AccountID, &m.Direction, &m.VendorMessageID,
		&m.MessageType, &m.Status, &m.Body, &m.CreatedAt, &m.UpdatedAt)
	if err == pgx.ErrNoRows {
		return Message{}, fmt.Errorf("message not found")
	}
	return m, err
}

func (r *MessageRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE messages SET status=$2, updated_at=NOW() WHERE id=$1`, id, status)
	return err
}
