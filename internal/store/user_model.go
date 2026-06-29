package store

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Email        string          `json:"email"`
	PasswordHash string          `json:"-"`
	IsAdmin      bool            `json:"is_admin"`
	IsActive     bool            `json:"is_active"`
	Permissions  json.RawMessage `json:"permissions"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func (u User) Public() User {
	u.PasswordHash = ""
	return u
}
