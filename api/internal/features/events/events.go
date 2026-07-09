// Package events holds the event use cases.
//
// One file per use case. Each owns its command or query struct, its result
// struct, and its SQL. There is no service layer and no repository layer: the
// SQL is here, inline, and this is the only place it appears.
//
// Handlers take a postgres.Querier, satisfied by both *pgxpool.Pool and pgx.Tx,
// so a caller can enrol a handler in a transaction alongside outbox.Enqueue.
package events

import (
	"encoding/json"
	"time"

	"eventify/api/internal/domain"

	"github.com/jackc/pgx/v5"
)

// columns is the projection every read of `events` uses. Kept in one place so a
// schema change touches one line rather than five Scan calls.
const columns = `id, name, description, location, date, organizer, category, tags, capacity, created_by, created_at, updated_at`

// scanEvent reads one row in `columns` order.
//
// tags is JSONB in Postgres, so it arrives as bytes and is decoded here rather
// than by a driver-level type. A NULL tags column yields a nil slice.
func scanEvent(row pgx.Row) (domain.Event, error) {
	var (
		e       domain.Event
		rawTags []byte
	)
	err := row.Scan(
		&e.ID, &e.Name, &e.Description, &e.Location, &e.Date,
		&e.Organizer, &e.Category, &rawTags, &e.Capacity,
		&e.CreatedBy, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return domain.Event{}, err
	}
	if len(rawTags) > 0 {
		if err := json.Unmarshal(rawTags, &e.Tags); err != nil {
			return domain.Event{}, err
		}
	}
	return e, nil
}

// encodeTags renders tags for the JSONB column. A nil slice becomes a JSON
// array, not SQL NULL, so a round-trip is stable.
func encodeTags(tags []string) ([]byte, error) {
	if tags == nil {
		tags = []string{}
	}
	return json.Marshal(tags)
}

// nowPtr is a helper for the UpdatedAt pointer.
func nowPtr() *time.Time {
	t := time.Now().UTC()
	return &t
}
