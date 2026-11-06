package webhook

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrEventNotFound = errors.New("webhook event not found")
)

type IRepository interface {
	RecordEvent(ctx context.Context, provider string, eventID string, eventType string, payload string) (bool, error)
	GetEventByProviderAndID(ctx context.Context, provider string, eventID string) (*WebhookEvent, error)
	ListEvents(ctx context.Context, provider string, page int, limit int) ([]WebhookEvent, int, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) RecordEvent(ctx context.Context, provider string, eventID string, eventType string, payload string) (bool, error) {
	query := `
		INSERT INTO webhook_events (id, provider, event_id, event_type, payload, status, processed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, 'PROCESSED', $6, $6)
		ON CONFLICT (provider, event_id) DO NOTHING
	`
	id := uuid.New().String()
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, query, id, provider, eventID, eventType, payload, now)
	if err != nil {
		return false, fmt.Errorf("record webhook event: %w", err)
	}
	rows, _ := res.RowsAffected()
	return rows > 0, nil
}

func (r *repository) GetEventByProviderAndID(ctx context.Context, provider string, eventID string) (*WebhookEvent, error) {
	var ev WebhookEvent
	query := `SELECT * FROM webhook_events WHERE provider = $1 AND event_id = $2`
	err := r.db.GetContext(ctx, &ev, query, provider, eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEventNotFound
		}
		return nil, fmt.Errorf("get webhook event: %w", err)
	}
	return &ev, nil
}

func (r *repository) ListEvents(ctx context.Context, provider string, page int, limit int) ([]WebhookEvent, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int
	countQuery := `SELECT COUNT(*) FROM webhook_events WHERE 1=1`
	args := []any{}
	argIdx := 1
	if provider != "" {
		countQuery += fmt.Sprintf(" AND provider = $%d", argIdx)
		args = append(args, provider)
		argIdx++
	}

	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("count webhook events: %w", err)
	}

	selectQuery := `SELECT * FROM webhook_events WHERE 1=1`
	selectArgs := []any{}
	sArgIdx := 1
	if provider != "" {
		selectQuery += fmt.Sprintf(" AND provider = $%d", sArgIdx)
		selectArgs = append(selectArgs, provider)
		sArgIdx++
	}

	selectQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", sArgIdx, sArgIdx+1)
	selectArgs = append(selectArgs, limit, offset)

	var events []WebhookEvent
	err = r.db.SelectContext(ctx, &events, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list webhook events: %w", err)
	}

	return events, total, nil
}
