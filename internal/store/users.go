package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepo struct {
	db *DB
}

func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

const userCols = `id, name, email, password_hash, is_admin, is_active, permissions, created_at, updated_at`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.IsAdmin, &u.IsActive,
		&u.Permissions, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (r *UserRepo) Create(ctx context.Context, u *User) error {
	if u.Permissions == nil {
		u.Permissions = json.RawMessage(`[]`)
	}
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	return r.db.Pool.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash, is_admin, is_active, permissions)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING `+userCols,
		u.Name, u.Email, u.PasswordHash, u.IsAdmin, u.IsActive, u.Permissions,
	).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.IsAdmin, &u.IsActive,
		&u.Permissions, &u.CreatedAt, &u.UpdatedAt)
}

func (r *UserRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	row := r.db.Pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE email=$1`, email)
	u, err := scanUser(row)
	if err == pgx.ErrNoRows {
		return User{}, fmt.Errorf("user not found")
	}
	return u, err
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	row := r.db.Pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id=$1`, id)
	u, err := scanUser(row)
	if err == pgx.ErrNoRows {
		return User{}, fmt.Errorf("user not found")
	}
	return u, err
}

func (r *UserRepo) List(ctx context.Context) ([]User, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT `+userCols+` FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u.Public())
	}
	return out, rows.Err()
}

func (r *UserRepo) Update(ctx context.Context, u User) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE users SET name=$2, is_active=$3, permissions=$4, updated_at=NOW()
		WHERE id=$1`,
		u.ID, u.Name, u.IsActive, u.Permissions,
	)
	return err
}

func (r *UserRepo) SetPassword(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE users SET password_hash=$2, updated_at=NOW() WHERE id=$1`, id, hash)
	return err
}
