package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vedran77/pulse/internal/domain"
)

type WorkspaceRepo struct {
	pool *pgxpool.Pool
}

func NewWorkspaceRepo(pool *pgxpool.Pool) *WorkspaceRepo {
	return &WorkspaceRepo{pool: pool}
}

func (r *WorkspaceRepo) Create(ctx context.Context, ws *domain.Workspace) error {
	query := `
		INSERT INTO workspaces (id, name, slug, description, owner_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.pool.Exec(ctx, query,
		ws.ID, ws.Name, ws.Slug, ws.Description, ws.OwnerID, ws.CreatedAt,
	)
	return err
}

func (r *WorkspaceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Workspace, error) {
	query := `SELECT id, name, slug, description, owner_id, created_at FROM workspaces WHERE id = $1`
	return r.scanWorkspace(ctx, query, id)
}

func (r *WorkspaceRepo) GetBySlug(ctx context.Context, slug string) (*domain.Workspace, error) {
	query := `SELECT id, name, slug, description, owner_id, created_at FROM workspaces WHERE slug = $1`
	return r.scanWorkspace(ctx, query, slug)
}

func (r *WorkspaceRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	query := `
		SELECT w.id, w.name, w.slug, w.description, w.owner_id, w.created_at
		FROM workspaces w
		INNER JOIN workspace_members wm ON w.id = wm.workspace_id
		WHERE wm.user_id = $1
		ORDER BY w.created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []domain.Workspace
	for rows.Next() {
		var ws domain.Workspace
		if err := rows.Scan(&ws.ID, &ws.Name, &ws.Slug, &ws.Description, &ws.OwnerID, &ws.CreatedAt); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, ws)
	}
	return workspaces, rows.Err()
}

func (r *WorkspaceRepo) Update(ctx context.Context, ws *domain.Workspace) error {
	query := `UPDATE workspaces SET name = $1, slug = $2, description = $3 WHERE id = $4`
	_, err := r.pool.Exec(ctx, query, ws.Name, ws.Slug, ws.Description, ws.ID)
	return err
}

func (r *WorkspaceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM workspaces WHERE id = $1`, id)
	return err
}

func (r *WorkspaceRepo) AddMember(ctx context.Context, m *domain.WorkspaceMember) error {
	query := `
		INSERT INTO workspace_members (workspace_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, query, m.WorkspaceID, m.UserID, m.Role, m.JoinedAt)
	return err
}

func (r *WorkspaceRepo) RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`, workspaceID, userID)
	return err
}

func (r *WorkspaceRepo) GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*domain.WorkspaceMember, error) {
	query := `SELECT workspace_id, user_id, role, joined_at FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`
	var m domain.WorkspaceMember
	err := r.pool.QueryRow(ctx, query, workspaceID, userID).Scan(&m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &m, err
}

func (r *WorkspaceRepo) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]domain.WorkspaceMember, error) {
	query := `
		SELECT wm.workspace_id, wm.user_id, wm.role, wm.joined_at, u.username, u.display_name
		FROM workspace_members wm
		JOIN users u ON wm.user_id = u.id
		WHERE wm.workspace_id = $1
		ORDER BY wm.joined_at`

	rows, err := r.pool.Query(ctx, query, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []domain.WorkspaceMember
	for rows.Next() {
		var m domain.WorkspaceMember
		if err := rows.Scan(&m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt, &m.Username, &m.DisplayName); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *WorkspaceRepo) scanWorkspace(ctx context.Context, query string, arg any) (*domain.Workspace, error) {
	var ws domain.Workspace
	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&ws.ID, &ws.Name, &ws.Slug, &ws.Description, &ws.OwnerID, &ws.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &ws, err
}
