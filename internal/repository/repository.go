package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kaungmyathan18/golang-inventory-app/internal/database"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrDuplicateEmail = errors.New("duplicate email")
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type UserRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewUserRepository(db *database.DB, log *zap.Logger) *UserRepository {
	return &UserRepository{db: db.SQL, log: log}
}

func (r *UserRepository) Create(ctx context.Context, email, name string) (*User, error) {
	u := User{
		ID:        uuid.NewString(),
		Email:     email,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, name, created_at) VALUES (?,?,?,?)`,
		u.ID, u.Email, u.Name, u.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, ErrDuplicateEmail
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Get(ctx context.Context, id string) (*User, error) {
	var createdAt string
	u := User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, name, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u.CreatedAt, err = parseCreatedAt(createdAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) ListPaged(ctx context.Context, offset, limit int) ([]User, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, name, created_at FROM users ORDER BY created_at ASC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		var createdAt string
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &createdAt); err != nil {
			return nil, 0, err
		}
		u.CreatedAt, err = parseCreatedAt(createdAt)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	return out, total, rows.Err()
}

func parseCreatedAt(value string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05Z07:00",
	} {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse created_at %q", value)
}
