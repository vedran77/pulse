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

type DMRepo struct {
	pool *pgxpool.Pool
}

func NewDMRepo(pool *pgxpool.Pool) *DMRepo {
	return &DMRepo{pool: pool}
}

func (r *DMRepo) CreateConversation(ctx context.Context, conv *domain.DMConversation) error {
	query := `
		INSERT INTO dm_conversations (id, user1_id, user2_id, created_at)
		VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, query, conv.ID, conv.User1ID, conv.User2ID, conv.CreatedAt)
	return err
}

func (r *DMRepo) GetConversationByUsers(ctx context.Context, user1ID, user2ID uuid.UUID) (*domain.DMConversation, error) {
	query := `
		SELECT id, user1_id, user2_id, created_at
		FROM dm_conversations
		WHERE user1_id = $1 AND user2_id = $2`
	var conv domain.DMConversation
	err := r.pool.QueryRow(ctx, query, user1ID, user2ID).Scan(
		&conv.ID, &conv.User1ID, &conv.User2ID, &conv.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &conv, err
}

func (r *DMRepo) GetConversationByID(ctx context.Context, id uuid.UUID) (*domain.DMConversation, error) {
	query := `
		SELECT id, user1_id, user2_id, created_at
		FROM dm_conversations
		WHERE id = $1`
	var conv domain.DMConversation
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&conv.ID, &conv.User1ID, &conv.User2ID, &conv.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &conv, err
}

func (r *DMRepo) ListConversations(ctx context.Context, userID uuid.UUID) ([]domain.DMConversation, error) {
	query := `
		SELECT c.id, c.user1_id, c.user2_id, c.created_at,
			CASE WHEN c.user1_id = $1 THEN c.user2_id ELSE c.user1_id END AS other_user_id,
			CASE WHEN c.user1_id = $1 THEN u2.username ELSE u1.username END AS other_username,
			CASE WHEN c.user1_id = $1 THEN u2.display_name ELSE u1.display_name END AS other_display_name
		FROM dm_conversations c
		JOIN users u1 ON c.user1_id = u1.id
		JOIN users u2 ON c.user2_id = u2.id
		WHERE c.user1_id = $1 OR c.user2_id = $1
		ORDER BY c.created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []domain.DMConversation
	for rows.Next() {
		var conv domain.DMConversation
		if err := rows.Scan(
			&conv.ID, &conv.User1ID, &conv.User2ID, &conv.CreatedAt,
			&conv.OtherUserID, &conv.OtherUserUsername, &conv.OtherUserDisplayName,
		); err != nil {
			return nil, err
		}
		convs = append(convs, conv)
	}
	return convs, rows.Err()
}

func (r *DMRepo) CreateMessage(ctx context.Context, msg *domain.DMMessage) error {
	query := `
		INSERT INTO dm_messages (id, conversation_id, sender_id, content, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.pool.Exec(ctx, query,
		msg.ID, msg.ConversationID, msg.SenderID, msg.Content, msg.CreatedAt,
	)
	return err
}

func (r *DMRepo) GetMessageByID(ctx context.Context, id uuid.UUID) (*domain.DMMessage, error) {
	query := `
		SELECT m.id, m.conversation_id, m.sender_id, m.content,
			m.edited_at, m.deleted_at, m.created_at, u.username, u.display_name
		FROM dm_messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.id = $1`
	var msg domain.DMMessage
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content,
		&msg.EditedAt, &msg.DeletedAt, &msg.CreatedAt,
		&msg.SenderUsername, &msg.SenderDisplayName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &msg, err
}

func (r *DMRepo) ListMessages(ctx context.Context, conversationID uuid.UUID, before *uuid.UUID, limit int) ([]domain.DMMessage, error) {
	var query string
	var args []any

	if before != nil {
		query = fmt.Sprintf(`
			SELECT m.id, m.conversation_id, m.sender_id, m.content,
				m.edited_at, m.deleted_at, m.created_at, u.username, u.display_name
			FROM dm_messages m
			JOIN users u ON m.sender_id = u.id
			WHERE m.conversation_id = $1 AND m.deleted_at IS NULL
				AND m.created_at < (SELECT created_at FROM dm_messages WHERE id = $2)
			ORDER BY m.created_at DESC
			LIMIT %d`, limit)
		args = []any{conversationID, *before}
	} else {
		query = fmt.Sprintf(`
			SELECT m.id, m.conversation_id, m.sender_id, m.content,
				m.edited_at, m.deleted_at, m.created_at, u.username, u.display_name
			FROM dm_messages m
			JOIN users u ON m.sender_id = u.id
			WHERE m.conversation_id = $1 AND m.deleted_at IS NULL
			ORDER BY m.created_at DESC
			LIMIT %d`, limit)
		args = []any{conversationID}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.DMMessage
	for rows.Next() {
		var msg domain.DMMessage
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content,
			&msg.EditedAt, &msg.DeletedAt, &msg.CreatedAt,
			&msg.SenderUsername, &msg.SenderDisplayName,
		); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// Reverse to chronological order (query returns DESC)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, rows.Err()
}

func (r *DMRepo) UpdateMessage(ctx context.Context, msg *domain.DMMessage) error {
	query := `UPDATE dm_messages SET content = $1, edited_at = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, msg.Content, time.Now(), msg.ID)
	return err
}

func (r *DMRepo) SoftDeleteMessage(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE dm_messages SET deleted_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}
