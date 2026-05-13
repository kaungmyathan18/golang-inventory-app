package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"database/sql"
	"strings"
	"github.com/kaungmyathan18/golang-inventory-app/internal/database"
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
		u.ID, u.Email, u.Name, u.CreatedAt,
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
	var u User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, name, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
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
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	return out, total, rows.Err()
}
