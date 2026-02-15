package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vedran77/pulse/internal/domain"
)

type ChannelRepo struct {
	pool *pgxpool.Pool
}

func NewChannelRepo(pool *pgxpool.Pool) *ChannelRepo {
	return &ChannelRepo{pool: pool}
}

func (r *ChannelRepo) Create(ctx context.Context, ch *domain.Channel) error {
	query := `
		INSERT INTO channels (id, workspace_id, name, description, type, is_encrypted, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query,
		ch.ID, ch.WorkspaceID, ch.Name, ch.Description, ch.Type, ch.IsEncrypted, ch.CreatedBy, ch.CreatedAt,
	)
	return err
}

func (r *ChannelRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Channel, error) {
	query := `SELECT id, workspace_id, name, description, type, is_encrypted, created_by, created_at, archived_at
		FROM channels WHERE id = $1`
	var ch domain.Channel
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ch.ID, &ch.WorkspaceID, &ch.Name, &ch.Description, &ch.Type,
		&ch.IsEncrypted, &ch.CreatedBy, &ch.CreatedAt, &ch.ArchivedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &ch, err
}

func (r *ChannelRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.Channel, error) {
	query := `SELECT id, workspace_id, name, description, type, is_encrypted, created_by, created_at, archived_at
		FROM channels WHERE workspace_id = $1 AND archived_at IS NULL ORDER BY created_at`

	rows, err := r.pool.Query(ctx, query, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []domain.Channel
	for rows.Next() {
		var ch domain.Channel
		if err := rows.Scan(&ch.ID, &ch.WorkspaceID, &ch.Name, &ch.Description, &ch.Type,
			&ch.IsEncrypted, &ch.CreatedBy, &ch.CreatedAt, &ch.ArchivedAt); err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

func (r *ChannelRepo) Update(ctx context.Context, ch *domain.Channel) error {
	query := `UPDATE channels SET name = $1, description = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, ch.Name, ch.Description, ch.ID)
	return err
}

func (r *ChannelRepo) Archive(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE channels SET archived_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}

func (r *ChannelRepo) AddMember(ctx context.Context, m *domain.ChannelMember) error {
	query := `INSERT INTO channel_members (channel_id, user_id, role, joined_at) VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, query, m.ChannelID, m.UserID, m.Role, m.JoinedAt)
	return err
}

func (r *ChannelRepo) RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM channel_members WHERE channel_id = $1 AND user_id = $2`, channelID, userID)
	return err
}

func (r *ChannelRepo) GetMember(ctx context.Context, channelID, userID uuid.UUID) (*domain.ChannelMember, error) {
	query := `SELECT channel_id, user_id, role, encrypted_key, last_read_msg_id, joined_at
		FROM channel_members WHERE channel_id = $1 AND user_id = $2`
	var m domain.ChannelMember
	err := r.pool.QueryRow(ctx, query, channelID, userID).Scan(
		&m.ChannelID, &m.UserID, &m.Role, &m.EncryptedKey, &m.LastReadMsgID, &m.JoinedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &m, err
}

func (r *ChannelRepo) ListMembers(ctx context.Context, channelID uuid.UUID) ([]domain.ChannelMember, error) {
	query := `SELECT channel_id, user_id, role, encrypted_key, last_read_msg_id, joined_at
		FROM channel_members WHERE channel_id = $1 ORDER BY joined_at`

	rows, err := r.pool.Query(ctx, query, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []domain.ChannelMember
	for rows.Next() {
		var m domain.ChannelMember
		if err := rows.Scan(&m.ChannelID, &m.UserID, &m.Role, &m.EncryptedKey, &m.LastReadMsgID, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}
