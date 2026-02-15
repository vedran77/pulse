package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vedran77/pulse/internal/domain"
)

type PulsemateRepo struct {
	pool *pgxpool.Pool
}

func NewPulsemateRepo(pool *pgxpool.Pool) *PulsemateRepo {
	return &PulsemateRepo{pool: pool}
}

func (r *PulsemateRepo) CreateRequest(ctx context.Context, req *domain.PulsemateRequest) error {
	query := `
		INSERT INTO pulsemate_requests (id, sender_id, receiver_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.pool.Exec(ctx, query, req.ID, req.SenderID, req.ReceiverID, req.Status, req.CreatedAt)
	return err
}

func (r *PulsemateRepo) GetRequestByID(ctx context.Context, id uuid.UUID) (*domain.PulsemateRequest, error) {
	query := `
		SELECT id, sender_id, receiver_id, status, created_at
		FROM pulsemate_requests
		WHERE id = $1`
	var req domain.PulsemateRequest
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&req.ID, &req.SenderID, &req.ReceiverID, &req.Status, &req.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &req, err
}

func (r *PulsemateRepo) GetRequestByUsers(ctx context.Context, senderID, receiverID uuid.UUID) (*domain.PulsemateRequest, error) {
	query := `
		SELECT id, sender_id, receiver_id, status, created_at
		FROM pulsemate_requests
		WHERE sender_id = $1 AND receiver_id = $2`
	var req domain.PulsemateRequest
	err := r.pool.QueryRow(ctx, query, senderID, receiverID).Scan(
		&req.ID, &req.SenderID, &req.ReceiverID, &req.Status, &req.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &req, err
}

func (r *PulsemateRepo) ListIncomingRequests(ctx context.Context, userID uuid.UUID) ([]domain.PulsemateRequest, error) {
	query := `
		SELECT r.id, r.sender_id, r.receiver_id, r.status, r.created_at,
			u.username, u.display_name
		FROM pulsemate_requests r
		JOIN users u ON r.sender_id = u.id
		WHERE r.receiver_id = $1 AND r.status = 'pending'
		ORDER BY r.created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reqs []domain.PulsemateRequest
	for rows.Next() {
		var req domain.PulsemateRequest
		if err := rows.Scan(
			&req.ID, &req.SenderID, &req.ReceiverID, &req.Status, &req.CreatedAt,
			&req.SenderUsername, &req.SenderDisplayName,
		); err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	return reqs, rows.Err()
}

func (r *PulsemateRepo) ListOutgoingRequests(ctx context.Context, userID uuid.UUID) ([]domain.PulsemateRequest, error) {
	query := `
		SELECT r.id, r.sender_id, r.receiver_id, r.status, r.created_at,
			u.username, u.display_name
		FROM pulsemate_requests r
		JOIN users u ON r.receiver_id = u.id
		WHERE r.sender_id = $1 AND r.status = 'pending'
		ORDER BY r.created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reqs []domain.PulsemateRequest
	for rows.Next() {
		var req domain.PulsemateRequest
		if err := rows.Scan(
			&req.ID, &req.SenderID, &req.ReceiverID, &req.Status, &req.CreatedAt,
			&req.ReceiverUsername, &req.ReceiverDisplayName,
		); err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	return reqs, rows.Err()
}

func (r *PulsemateRepo) DeleteRequest(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM pulsemate_requests WHERE id = $1`, id)
	return err
}

func (r *PulsemateRepo) CreatePulsemate(ctx context.Context, pm *domain.Pulsemate) error {
	query := `
		INSERT INTO pulsemates (id, user1_id, user2_id, created_at)
		VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, query, pm.ID, pm.User1ID, pm.User2ID, pm.CreatedAt)
	return err
}

func (r *PulsemateRepo) GetPulsemateByUsers(ctx context.Context, user1ID, user2ID uuid.UUID) (*domain.Pulsemate, error) {
	query := `
		SELECT id, user1_id, user2_id, created_at
		FROM pulsemates
		WHERE user1_id = $1 AND user2_id = $2`
	var pm domain.Pulsemate
	err := r.pool.QueryRow(ctx, query, user1ID, user2ID).Scan(
		&pm.ID, &pm.User1ID, &pm.User2ID, &pm.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &pm, err
}

func (r *PulsemateRepo) DeletePulsemate(ctx context.Context, user1ID, user2ID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM pulsemates WHERE user1_id = $1 AND user2_id = $2`, user1ID, user2ID)
	return err
}

func (r *PulsemateRepo) ListPulsemates(ctx context.Context, userID uuid.UUID) ([]domain.Pulsemate, error) {
	query := `
		SELECT p.id, p.user1_id, p.user2_id, p.created_at,
			CASE WHEN p.user1_id = $1 THEN p.user2_id ELSE p.user1_id END AS other_user_id,
			CASE WHEN p.user1_id = $1 THEN u2.username ELSE u1.username END AS other_username,
			CASE WHEN p.user1_id = $1 THEN u2.display_name ELSE u1.display_name END AS other_display_name,
			CASE WHEN p.user1_id = $1 THEN u2.status ELSE u1.status END AS other_status
		FROM pulsemates p
		JOIN users u1 ON p.user1_id = u1.id
		JOIN users u2 ON p.user2_id = u2.id
		WHERE p.user1_id = $1 OR p.user2_id = $1
		ORDER BY other_display_name ASC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pms []domain.Pulsemate
	for rows.Next() {
		var pm domain.Pulsemate
		if err := rows.Scan(
			&pm.ID, &pm.User1ID, &pm.User2ID, &pm.CreatedAt,
			&pm.OtherUserID, &pm.OtherUsername, &pm.OtherDisplayName, &pm.OtherStatus,
		); err != nil {
			return nil, err
		}
		pms = append(pms, pm)
	}
	return pms, rows.Err()
}

func (r *PulsemateRepo) ArePulsemates(ctx context.Context, userA, userB uuid.UUID) (bool, error) {
	u1, u2 := userA, userB
	if u1.String() > u2.String() {
		u1, u2 = u2, u1
	}
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pulsemates WHERE user1_id = $1 AND user2_id = $2)`,
		u1, u2,
	).Scan(&exists)
	return exists, err
}
