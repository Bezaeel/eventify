package events

import (
	"context"
	"errors"

	"eventify/api/internal/domain"
	"eventify/platform/apperrors"
	"eventify/platform/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetEventsQuery lists events, newest first.
//
// Limit and Offset are mandatory in practice: the old GetAllEvents did an
// unbounded `SELECT * FROM events` and returned every row to every caller,
// including the GraphQL resolver, which then discarded all but one page.
type GetEventsQuery struct {
	Limit  int
	Offset int
}

// GetEventsResult carries the page plus the total, so a caller can paginate.
type GetEventsResult struct {
	Events []domain.Event
	Total  int
}

// GetEventsHandler lists events.
type GetEventsHandler struct {
	db postgres.Querier
}

func NewGetEventsHandler(db postgres.Querier) GetEventsHandler {
	return GetEventsHandler{db: db}
}

const (
	defaultLimit = 50
	maxLimit     = 200
)

// Handle returns one page of events and the total row count.
func (h GetEventsHandler) Handle(ctx context.Context, q GetEventsQuery) (GetEventsResult, error) {
	var res GetEventsResult

	limit := q.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	offset := max(q.Offset, 0)

	rows, err := h.db.Query(ctx,
		`SELECT `+columns+` FROM events ORDER BY date DESC LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return res, apperrors.Wrap(apperrors.Internal, "list events", err)
	}
	defer rows.Close()

	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return res, apperrors.Wrap(apperrors.Internal, "scan event", err)
		}
		res.Events = append(res.Events, e)
	}
	if err := rows.Err(); err != nil {
		return res, apperrors.Wrap(apperrors.Internal, "iterate events", err)
	}

	// Second round trip, so it lives in an unexported helper in this file
	// rather than in a repository type.
	res.Total, err = countEvents(ctx, h.db)
	if err != nil {
		return res, err
	}
	return res, nil
}

func countEvents(ctx context.Context, db postgres.Querier) (int, error) {
	var n int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM events`).Scan(&n); err != nil {
		return 0, apperrors.Wrap(apperrors.Internal, "count events", err)
	}
	return n, nil
}

// GetEventQuery fetches one event by id.
type GetEventQuery struct {
	EventID uuid.UUID
}

// GetEventHandler fetches one event.
type GetEventHandler struct {
	db postgres.Querier
}

func NewGetEventHandler(db postgres.Querier) GetEventHandler {
	return GetEventHandler{db: db}
}

// Handle returns the event, or NotFound.
//
// The old GetEventById called gorm's First(event, id) with a uuid.UUID, which
// gorm interprets as an inline primary-key condition only for integer keys; and
// it swallowed every error into a nil return, so "not found" and "database is
// down" were indistinguishable to the caller.
func (h GetEventHandler) Handle(ctx context.Context, q GetEventQuery) (domain.Event, error) {
	row := h.db.QueryRow(ctx, `SELECT `+columns+` FROM events WHERE id = $1`, q.EventID)

	e, err := scanEvent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Event{}, apperrors.New(apperrors.NotFound, "event not found")
	}
	if err != nil {
		return domain.Event{}, apperrors.Wrap(apperrors.Internal, "get event", err)
	}
	return e, nil
}
