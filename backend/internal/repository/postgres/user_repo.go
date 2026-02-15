package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vedran77/pulse/internal/domain"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, username, display_name, password_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.Username, user.DisplayName,
		user.PasswordHash, user.Status, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return r.scanUser(ctx, "SELECT id, email, username, display_name, password_hash, public_key, avatar_url, status, created_at, updated_at FROM users WHERE id = $1", id)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.scanUser(ctx, "SELECT id, email, username, display_name, password_hash, public_key, avatar_url, status, created_at, updated_at FROM users WHERE email = $1", email)
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return r.scanUser(ctx, "SELECT id, email, username, display_name, password_hash, public_key, avatar_url, status, created_at, updated_at FROM users WHERE username = $1", username)
}

func (r *UserRepo) scanUser(ctx context.Context, query string, arg any) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&u.ID, &u.Email, &u.Username, &u.DisplayName,
		&u.PasswordHash, &u.PublicKey, &u.AvatarURL,
		&u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}
