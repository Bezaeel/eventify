package events

import (
	"context"
	"errors"
	"time"

	"eventify/platform/apperrors"
	"eventify/platform/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// UpdateEventCommand updates every mutable field of an event.
type UpdateEventCommand struct {
	Date        time.Time
	Name        string
	Description string
	Location    string
	Organizer   string
	Category    string
	Tags        []string
	EventID     uuid.UUID
	Capacity    int
}

// UpdateEventResult reports what was written.
type UpdateEventResult struct {
	UpdatedAt time.Time
	EventID   uuid.UUID
}

// UpdateEventHandler is shared by HTTP v1, HTTP v2, gRPC and GraphQL.
type UpdateEventHandler struct {
	db postgres.Querier
}

func NewUpdateEventHandler(db postgres.Querier) UpdateEventHandler {
	return UpdateEventHandler{db: db}
}

// Handle updates the event and returns its new state.
//
// Two bugs the old code had, worth not reintroducing:
//
//  1. The v1 mapper built a domain.Event without setting Id, then called
//     gorm's Save(). With a zero primary key Save() INSERTs, so every "update"
//     created a new row with a nil UUID and returned that nil UUID to the
//     client. Here the id is a WHERE predicate, and a miss is NotFound.
//
//  2. The same mapper silently dropped Description, Organizer, Category, Tags
//     and Capacity — the request accepted them and the write discarded them.
//     Every column named in the command is written.
func (h UpdateEventHandler) Handle(ctx context.Context, cmd UpdateEventCommand) (UpdateEventResult, error) {
	var res UpdateEventResult

	if cmd.Capacity < 1 {
		return res, apperrors.New(apperrors.Invalid, "capacity must be at least 1")
	}

	tags, err := encodeTags(cmd.Tags)
	if err != nil {
		return res, apperrors.Wrap(apperrors.Invalid, "encode tags", err)
	}

	err = h.db.QueryRow(ctx,
		`UPDATE events
		    SET name = $2, description = $3, location = $4, date = $5,
		        organizer = $6, category = $7, tags = $8, capacity = $9,
		        updated_at = now()
		  WHERE id = $1
		  RETURNING id, updated_at`,
		cmd.EventID, cmd.Name, cmd.Description, cmd.Location, cmd.Date,
		cmd.Organizer, cmd.Category, tags, cmd.Capacity,
	).Scan(&res.EventID, &res.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return res, apperrors.New(apperrors.NotFound, "event not found")
	}
	if err != nil {
		return res, apperrors.Wrap(apperrors.Internal, "update event", err)
	}
	return res, nil
}
