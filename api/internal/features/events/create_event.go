package events

import (
	"context"
	"time"

	eventcreatedv1 "eventify/events/eventcreated/v1"
	"eventify/outbox"
	"eventify/platform/apperrors"
	"eventify/platform/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateEventCommand is the input to CreateEventHandler. Fields are parsed Go
// types; parsing is the transport's job.
type CreateEventCommand struct {
	Date        time.Time
	Name        string
	Description string
	Location    string
	Organizer   string
	Category    string
	Tags        []string
	CreatedBy   uuid.UUID
	Capacity    int
}

// CreateEventResult is what the caller gets back.
type CreateEventResult struct {
	CreatedAt time.Time
	EventID   uuid.UUID
}

// CreateEventHandler inserts an event and emits EventCreated.v1.
//
// It holds a *pgxpool.Pool rather than a postgres.Querier because it opens its
// own transaction: the insert and the outbox row must commit together.
type CreateEventHandler struct {
	pool *pgxpool.Pool
}

func NewCreateEventHandler(pool *pgxpool.Pool) CreateEventHandler {
	return CreateEventHandler{pool: pool}
}

// Handle writes the event and its outbox row in one transaction.
//
// Publishing to RabbitMQ directly here would give two commit points and no way
// to make them atomic: a crash between them loses the event forever. Instead
// the intent-to-publish is a row in the same transaction, and the relay drains
// it. See package outbox.
func (h CreateEventHandler) Handle(ctx context.Context, cmd CreateEventCommand) (CreateEventResult, error) {
	var res CreateEventResult

	if cmd.Capacity < 1 {
		return res, apperrors.New(apperrors.Invalid, "capacity must be at least 1")
	}

	tags, err := encodeTags(cmd.Tags)
	if err != nil {
		return res, apperrors.Wrap(apperrors.Invalid, "encode tags", err)
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return res, apperrors.Wrap(apperrors.Internal, "begin transaction", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	id := uuid.New()
	err = tx.QueryRow(ctx,
		`INSERT INTO events
		     (id, name, description, location, date, organizer, category, tags, capacity, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
		 RETURNING id, created_at`,
		id, cmd.Name, cmd.Description, cmd.Location, cmd.Date,
		cmd.Organizer, cmd.Category, tags, cmd.Capacity, cmd.CreatedBy,
	).Scan(&res.EventID, &res.CreatedAt)
	if err != nil {
		return res, apperrors.Wrap(apperrors.Internal, "insert event", err)
	}

	evt := eventcreatedv1.EventCreated{
		ID:         res.EventID.String(),
		Name:       cmd.Name,
		Type:       cmd.Category,
		CreatedBy:  cmd.CreatedBy.String(),
		OccurredAt: res.CreatedAt.UTC(),
	}
	if err := outbox.Enqueue(ctx, tx, eventcreatedv1.Name, eventcreatedv1.Version, evt); err != nil {
		return res, apperrors.Wrap(apperrors.Internal, "enqueue EventCreated", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return res, apperrors.Wrap(apperrors.Internal, "commit transaction", err)
	}
	return res, nil
}

// compile-time assertion that a pool satisfies the Querier a plain handler uses
var _ postgres.Querier = (*pgxpool.Pool)(nil)
