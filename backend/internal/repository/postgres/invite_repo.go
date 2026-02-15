package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vedran77/pulse/internal/domain"
)

type InviteRepo struct {
	pool *pgxpool.Pool
}

func NewInviteRepo(pool *pgxpool.Pool) *InviteRepo {
	return &InviteRepo{pool: pool}
}

func (r *InviteRepo) Create(ctx context.Context, inv *domain.WorkspaceInvite) error {
	query := `
		INSERT INTO workspace_invites (id, workspace_id, email, token, invited_by, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query,
		inv.ID, inv.WorkspaceID, inv.Email, inv.Token, inv.InvitedBy, inv.CreatedAt, inv.ExpiresAt,
	)
	return err
}

func (r *InviteRepo) GetByToken(ctx context.Context, token string) (*domain.WorkspaceInvite, error) {
	query := `
		SELECT wi.id, wi.workspace_id, wi.email, wi.token, wi.invited_by,
		       wi.created_at, wi.expires_at, wi.accepted_at, wi.accepted_by,
		       w.name
		FROM workspace_invites wi
		JOIN workspaces w ON w.id = wi.workspace_id
		WHERE wi.token = $1`

	var inv domain.WorkspaceInvite
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Token, &inv.InvitedBy,
		&inv.CreatedAt, &inv.ExpiresAt, &inv.AcceptedAt, &inv.AcceptedBy,
		&inv.WorkspaceName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &inv, err
}

func (r *InviteRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]domain.WorkspaceInvite, error) {
	query := `
		SELECT id, workspace_id, email, token, invited_by, created_at, expires_at, accepted_at, accepted_by
		FROM workspace_invites
		WHERE workspace_id = $1
		  AND accepted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []domain.WorkspaceInvite
	for rows.Next() {
		var inv domain.WorkspaceInvite
		if err := rows.Scan(
			&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Token, &inv.InvitedBy,
			&inv.CreatedAt, &inv.ExpiresAt, &inv.AcceptedAt, &inv.AcceptedBy,
		); err != nil {
			return nil, err
		}
		invites = append(invites, inv)
	}
	return invites, rows.Err()
}

func (r *InviteRepo) MarkAccepted(ctx context.Context, id, userID uuid.UUID) error {
	query := `UPDATE workspace_invites SET accepted_at = NOW(), accepted_by = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, userID, id)
	return err
}

func (r *InviteRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM workspace_invites WHERE id = $1`, id)
	return err
}
