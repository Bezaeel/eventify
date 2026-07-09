// Package outbox implements the transactional outbox pattern.
//
// The problem it solves: a handler that writes a row and then publishes to
// RabbitMQ has two commit points. If the publish fails after the DB commit, the
// event is lost forever; if the process dies between them, likewise. No amount
// of retry logic closes that window, because the two systems cannot commit
// together.
//
// The outbox collapses them into one. Enqueue writes the event into an
// outbox_messages row inside the caller's transaction, so the business write
// and the intent-to-publish commit atomically. A separate relay process then
// reads unpublished rows and publishes them, marking each as published.
//
// This yields at-least-once delivery: the relay may publish a row and die
// before marking it, re-publishing on restart. Subscribers must therefore be
// idempotent, keyed on Envelope.MessageID.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"eventify/platform/postgres"

	"github.com/google/uuid"
)

// Message is one row of outbox_messages awaiting publication.
type Message struct {
	OccurredAt time.Time
	ID         uuid.UUID
	MessageID  uuid.UUID
	Name       string
	Version    string
	Payload    []byte
}

// Enqueue records an event for publication inside the caller's transaction.
//
// q must be the same pgx.Tx that performed the business write. Passing a pool
// here instead of a transaction defeats the entire pattern: the row would
// commit independently of the write it is supposed to accompany.
func Enqueue(ctx context.Context, q postgres.Querier, name, version string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s.%s payload: %w", name, version, err)
	}

	_, err = q.Exec(ctx,
		`INSERT INTO outbox_messages (id, message_id, name, version, payload, occurred_at)
		 VALUES ($1, $2, $3, $4, $5, now())`,
		uuid.New(), uuid.New(), name, version, body,
	)
	if err != nil {
		return fmt.Errorf("enqueue %s.%s: %w", name, version, err)
	}
	return nil
}

// FetchUnpublished claims up to limit unpublished rows for this relay instance.
//
// FOR UPDATE SKIP LOCKED lets several relay replicas poll the same table
// concurrently without handing the same row to two of them, and without
// blocking each other. Rows stay claimed until the surrounding transaction
// ends, so a crashed relay releases its claim automatically.
func FetchUnpublished(ctx context.Context, q postgres.Querier, limit int) ([]Message, error) {
	rows, err := q.Query(ctx,
		`SELECT id, message_id, name, version, payload, occurred_at
		   FROM outbox_messages
		  WHERE published_at IS NULL
		  ORDER BY occurred_at
		  LIMIT $1
		    FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch unpublished: %w", err)
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.MessageID, &m.Name, &m.Version, &m.Payload, &m.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan outbox row: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// MarkPublished stamps rows as published. Called after a successful publish,
// in the same transaction that claimed them.
func MarkPublished(ctx context.Context, q postgres.Querier, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := q.Exec(ctx,
		`UPDATE outbox_messages SET published_at = now() WHERE id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("mark published: %w", err)
	}
	return nil
}
