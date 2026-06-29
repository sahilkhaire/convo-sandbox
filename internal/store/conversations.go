package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ConversationRepo struct {
	db *DB
}

func NewConversationRepo(db *DB) *ConversationRepo {
	return &ConversationRepo{db: db}
}

func (r *ConversationRepo) GetOrCreate(ctx context.Context, accountID uuid.UUID, channel, userPhone string) (Conversation, error) {
	var c Conversation
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, account_id, channel, user_phone, last_message_at, unread_count,
			conversation_window_expires_at, created_at, updated_at
		FROM conversations WHERE account_id=$1 AND user_phone=$2`,
		accountID, userPhone,
	).Scan(&c.ID, &c.AccountID, &c.Channel, &c.UserPhone, &c.LastMessageAt, &c.UnreadCount,
		&c.ConversationWindowExpires, &c.CreatedAt, &c.UpdatedAt)
	if err == nil {
		return c, nil
	}
	if err != pgx.ErrNoRows {
		return Conversation{}, err
	}
	err = r.db.Pool.QueryRow(ctx, `
		INSERT INTO conversations (account_id, channel, user_phone)
		VALUES ($1,$2,$3)
		RETURNING id, account_id, channel, user_phone, last_message_at, unread_count,
			conversation_window_expires_at, created_at, updated_at`,
		accountID, channel, userPhone,
	).Scan(&c.ID, &c.AccountID, &c.Channel, &c.UserPhone, &c.LastMessageAt, &c.UnreadCount,
		&c.ConversationWindowExpires, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *ConversationRepo) List(ctx context.Context) ([]Conversation, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT c.id, c.account_id, c.channel, c.user_phone, c.last_message_at, c.unread_count,
			c.conversation_window_expires_at, c.created_at, c.updated_at,
			a.name, a.external_id,
			COALESCE((
				SELECT CASE
					WHEN m.body->>'plainText' IS NOT NULL THEN m.body->>'plainText'
					WHEN m.body->'text'->>'body' IS NOT NULL THEN m.body->'text'->>'body'
					WHEN m.body->'content'->>'plainText' IS NOT NULL THEN m.body->'content'->>'plainText'
					ELSE m.message_type
				END
				FROM messages m WHERE m.conversation_id = c.id ORDER BY m.created_at DESC LIMIT 1
			), '') as preview
		FROM conversations c
		JOIN accounts a ON a.id = c.account_id
		ORDER BY c.last_message_at DESC NULLS LAST, c.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.AccountID, &c.Channel, &c.UserPhone, &c.LastMessageAt,
			&c.UnreadCount, &c.ConversationWindowExpires, &c.CreatedAt, &c.UpdatedAt,
			&c.AccountName, &c.AccountExternalID, &c.LastMessagePreview); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *ConversationRepo) GetByID(ctx context.Context, id uuid.UUID) (Conversation, error) {
	var c Conversation
	err := r.db.Pool.QueryRow(ctx, `
		SELECT c.id, c.account_id, c.channel, c.user_phone, c.last_message_at, c.unread_count,
			c.conversation_window_expires_at, c.created_at, c.updated_at, a.name, a.external_id, ''
		FROM conversations c JOIN accounts a ON a.id = c.account_id WHERE c.id=$1`, id,
	).Scan(&c.ID, &c.AccountID, &c.Channel, &c.UserPhone, &c.LastMessageAt, &c.UnreadCount,
		&c.ConversationWindowExpires, &c.CreatedAt, &c.UpdatedAt, &c.AccountName, &c.AccountExternalID, &c.LastMessagePreview)
	if err == pgx.ErrNoRows {
		return Conversation{}, fmt.Errorf("conversation not found")
	}
	return c, err
}

func (r *ConversationRepo) Touch(ctx context.Context, id uuid.UUID, inbound bool) error {
	if inbound {
		_, err := r.db.Pool.Exec(ctx, `
			UPDATE conversations SET last_message_at=NOW(), unread_count=unread_count+1,
				conversation_window_expires_at=NOW() + INTERVAL '24 hours', updated_at=NOW()
			WHERE id=$1`, id)
		return err
	}
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE conversations SET last_message_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *ConversationRepo) MarkRead(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE conversations SET unread_count=0, updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *ConversationRepo) ExtendWindow(ctx context.Context, id uuid.UUID, until time.Time) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE conversations SET conversation_window_expires_at=$2, updated_at=NOW() WHERE id=$1`, id, until)
	return err
}
