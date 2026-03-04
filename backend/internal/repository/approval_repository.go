package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ApprovalRepository interface {
	Create(ctx context.Context, approval *model.AgentPendingApproval) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.AgentPendingApproval, error)
	ListPending(ctx context.Context, userID uuid.UUID) ([]model.AgentPendingApproval, error)
	Resolve(ctx context.Context, id uuid.UUID, status model.ApprovalStatus, resolvedBy uuid.UUID) error
}

type pgApprovalRepository struct {
	db *pgxpool.Pool
}

func NewApprovalRepository(db *pgxpool.Pool) ApprovalRepository {
	return &pgApprovalRepository{db: db}
}

func (r *pgApprovalRepository) Create(ctx context.Context, approval *model.AgentPendingApproval) error {
	approval.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO agent_pending_approvals (id, user_id, conversation_id, message_id, tool_name, arguments, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		approval.ID, approval.UserID, approval.ConversationID, approval.MessageID,
		approval.ToolName, approval.Arguments, approval.Status,
	)
	if err != nil {
		return fmt.Errorf("create approval: %w", err)
	}
	return nil
}

func (r *pgApprovalRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.AgentPendingApproval, error) {
	var a model.AgentPendingApproval
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, conversation_id, message_id, tool_name, arguments, status, resolved_by, resolved_at, created_at
		 FROM agent_pending_approvals WHERE id = $1`, id,
	).Scan(&a.ID, &a.UserID, &a.ConversationID, &a.MessageID, &a.ToolName,
		&a.Arguments, &a.Status, &a.ResolvedBy, &a.ResolvedAt, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get approval by id: %w", err)
	}
	return &a, nil
}

func (r *pgApprovalRepository) ListPending(ctx context.Context, userID uuid.UUID) ([]model.AgentPendingApproval, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, conversation_id, message_id, tool_name, arguments, status, resolved_by, resolved_at, created_at
		 FROM agent_pending_approvals WHERE user_id = $1 AND status = 'pending'
		 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list pending approvals: %w", err)
	}
	defer rows.Close()

	var approvals []model.AgentPendingApproval
	for rows.Next() {
		var a model.AgentPendingApproval
		if err := rows.Scan(&a.ID, &a.UserID, &a.ConversationID, &a.MessageID, &a.ToolName,
			&a.Arguments, &a.Status, &a.ResolvedBy, &a.ResolvedAt, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan approval: %w", err)
		}
		approvals = append(approvals, a)
	}
	return approvals, rows.Err()
}

func (r *pgApprovalRepository) Resolve(ctx context.Context, id uuid.UUID, status model.ApprovalStatus, resolvedBy uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE agent_pending_approvals SET status=$1, resolved_by=$2, resolved_at=NOW() WHERE id=$3`,
		status, resolvedBy, id,
	)
	if err != nil {
		return fmt.Errorf("resolve approval: %w", err)
	}
	return nil
}
