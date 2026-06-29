package store

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID                 uuid.UUID       `json:"id"`
	Channel            string          `json:"channel"`
	Name               string          `json:"name"`
	ExternalID         string          `json:"external_id"`
	ClientSecret       string          `json:"client_secret,omitempty"`
	AccessToken        string          `json:"access_token,omitempty"`
	WebhookURL         string          `json:"webhook_url"`
	WebhookVerifyToken string          `json:"webhook_verify_token"`
	WabaID             string          `json:"waba_id"`
	DisplayPhone       string          `json:"display_phone"`
	SentDelayMs        int             `json:"sent_delay_ms"`
	DeliveredDelayMs   int             `json:"delivered_delay_ms"`
	ReadDelayMs        int             `json:"read_delay_ms"`
	FailureRate        int             `json:"failure_rate"`
	AutoRead           bool            `json:"auto_read"`
	Capabilities       json.RawMessage `json:"capabilities"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type Conversation struct {
	ID                        uuid.UUID  `json:"id"`
	AccountID                 uuid.UUID  `json:"account_id"`
	Channel                   string     `json:"channel"`
	UserPhone                 string     `json:"user_phone"`
	LastMessageAt             *time.Time `json:"last_message_at"`
	UnreadCount               int        `json:"unread_count"`
	ConversationWindowExpires *time.Time `json:"conversation_window_expires_at"`
	CreatedAt                 time.Time  `json:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at"`
	AccountName               string     `json:"account_name,omitempty"`
	AccountExternalID         string     `json:"account_external_id,omitempty"`
	LastMessagePreview        string     `json:"last_message_preview,omitempty"`
}

type Message struct {
	ID              uuid.UUID       `json:"id"`
	ConversationID  uuid.UUID       `json:"conversation_id"`
	AccountID       uuid.UUID       `json:"account_id"`
	Direction       string          `json:"direction"`
	VendorMessageID string          `json:"vendor_message_id"`
	MessageType     string          `json:"message_type"`
	Status          string          `json:"status"`
	Body            json.RawMessage `json:"body"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type MediaFile struct {
	ID          uuid.UUID  `json:"id"`
	AccountID   *uuid.UUID `json:"account_id"`
	FileID      string     `json:"file_id"`
	MimeType    string     `json:"mime_type"`
	FileName    string     `json:"file_name"`
	SizeBytes   int64      `json:"size_bytes"`
	StoragePath string     `json:"storage_path"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Template struct {
	ID         uuid.UUID       `json:"id"`
	AccountID  uuid.UUID       `json:"account_id"`
	Name       string          `json:"name"`
	Language   string          `json:"language"`
	Category   string          `json:"category"`
	Components json.RawMessage `json:"components"`
	CreatedAt  time.Time       `json:"created_at"`
}

type WebhookDelivery struct {
	ID           uuid.UUID       `json:"id"`
	AccountID    *uuid.UUID      `json:"account_id"`
	Channel      string          `json:"channel"`
	EventType    string          `json:"event_type"`
	Payload      json.RawMessage `json:"payload"`
	HTTPStatus   *int            `json:"http_status"`
	ResponseBody string          `json:"response_body"`
	ErrorMessage string          `json:"error_message"`
	Attempts     int             `json:"attempts"`
	CreatedAt    time.Time       `json:"created_at"`
}

type DeliveryJob struct {
	ID          uuid.UUID       `json:"id"`
	AccountID   uuid.UUID       `json:"account_id"`
	MessageID   *uuid.UUID      `json:"message_id"`
	Channel     string          `json:"channel"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
	ScheduledAt time.Time       `json:"scheduled_at"`
	Status      string          `json:"status"`
	Attempts    int             `json:"attempts"`
	CreatedAt   time.Time       `json:"created_at"`
}
