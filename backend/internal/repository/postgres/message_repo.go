package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vedran77/pulse/internal/domain"
)

type MessageRepo struct {
	pool *pgxpool.Pool
}

func NewMessageRepo(pool *pgxpool.Pool) *MessageRepo {
	return &MessageRepo{pool: pool}
}

func (r *MessageRepo) Create(ctx context.Context, msg *domain.Message) error {
	query := `
		INSERT INTO messages (id, channel_id, sender_id, content, type, parent_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, query,
		msg.ID, msg.ChannelID, msg.SenderID, msg.Content, msg.Type, msg.ParentID, msg.CreatedAt,
	)
	return err
}

func (r *MessageRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	query := `
		SELECT m.id, m.channel_id, m.sender_id, m.content, m.type, m.parent_id,
			m.edited_at, m.deleted_at, m.created_at, u.username, u.display_name
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.id = $1`
	var msg domain.Message
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&msg.ID, &msg.ChannelID, &msg.SenderID, &msg.Content, &msg.Type,
		&msg.ParentID, &msg.EditedAt, &msg.DeletedAt, &msg.CreatedAt,
		&msg.SenderUsername, &msg.SenderDisplayName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &msg, err
}

func (r *MessageRepo) ListByChannel(ctx context.Context, channelID uuid.UUID, before *uuid.UUID, limit int) ([]domain.Message, error) {
	var query string
	var args []any

	if before != nil {
		// Dohvati created_at od "before" poruke za cursor paginaciju
		query = fmt.Sprintf(`
			SELECT m.id, m.channel_id, m.sender_id, m.content, m.type, m.parent_id,
				m.edited_at, m.deleted_at, m.created_at, u.username, u.display_name
			FROM messages m
			JOIN users u ON m.sender_id = u.id
			WHERE m.channel_id = $1 AND m.deleted_at IS NULL
				AND m.created_at < (SELECT created_at FROM messages WHERE id = $2)
			ORDER BY m.created_at DESC
			LIMIT %d`, limit)
		args = []any{channelID, *before}
	} else {
		query = fmt.Sprintf(`
			SELECT m.id, m.channel_id, m.sender_id, m.content, m.type, m.parent_id,
				m.edited_at, m.deleted_at, m.created_at, u.username, u.display_name
			FROM messages m
			JOIN users u ON m.sender_id = u.id
			WHERE m.channel_id = $1 AND m.deleted_at IS NULL
			ORDER BY m.created_at DESC
			LIMIT %d`, limit)
		args = []any{channelID}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var msg domain.Message
		if err := rows.Scan(
			&msg.ID, &msg.ChannelID, &msg.SenderID, &msg.Content, &msg.Type,
			&msg.ParentID, &msg.EditedAt, &msg.DeletedAt, &msg.CreatedAt,
			&msg.SenderUsername, &msg.SenderDisplayName,
		); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// Reverse da budu chronological (query ih daje DESC)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, rows.Err()
}

func (r *MessageRepo) Update(ctx context.Context, msg *domain.Message) error {
	query := `UPDATE messages SET content = $1, edited_at = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, msg.Content, time.Now(), msg.ID)
	return err
}

func (r *MessageRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE messages SET deleted_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}
