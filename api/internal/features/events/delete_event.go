package events

import (
	"context"

	"eventify/platform/apperrors"
	"eventify/platform/postgres"

	"github.com/google/uuid"
)

// DeleteEventCommand removes an event.
type DeleteEventCommand struct {
	EventID uuid.UUID
}

// DeleteEventHandler removes an event.
type DeleteEventHandler struct {
	db postgres.Querier
}

func NewDeleteEventHandler(db postgres.Querier) DeleteEventHandler {
	return DeleteEventHandler{db: db}
}

// Handle deletes the event, reporting NotFound when nothing matched.
//
// The old DeleteEvent returned nil whether or not a row was removed, so a
// caller deleting a nonexistent id received 200 OK.
func (h DeleteEventHandler) Handle(ctx context.Context, cmd DeleteEventCommand) error {
	tag, err := h.db.Exec(ctx, `DELETE FROM events WHERE id = $1`, cmd.EventID)
	if err != nil {
		return apperrors.Wrap(apperrors.Internal, "delete event", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.New(apperrors.NotFound, "event not found")
	}
	return nil
}
