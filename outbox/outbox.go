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
// claims queued rows, hands each to the processor that declares its payload
// type, and records the outcome.
//
// This yields at-least-once delivery: the relay may publish a row and die
// before marking it complete, re-publishing on restart. Subscribers must
// therefore be idempotent, keyed on the payload's MessageID.
//
// # Recovering a stalled outbox
//
// A message leaves the queue for good in two ways, and both are meant to be
// noticed rather than absorbed:
//
//   - EXCEEDED: it failed MaxAttempts times. Attempts are spent one per poll,
//     so a broker outage lasting MaxAttempts poll intervals will exceed the
//     backlog rather than wait it out. This is deliberate — the outbox stopping
//     is a thing to alert on, not to silently retry forever.
//   - POISONED: no processor claims its payload type. The relay binary does not
//     know the event, usually because a producer shipped ahead of the relay.
//
// Both are cleared the same way, once the cause is fixed. Alert on either count
// being non-zero, then:
//
//	UPDATE outbox_messages
//	   SET status = 1, attempts = 0
//	 WHERE status IN (2, 4);   -- POISONED, EXCEEDED
//
// Messages resume in occurred_at order, and consumers deduplicate on MessageID,
// so replaying one that did in fact publish is safe.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"eventify/platform/postgres"

	"github.com/google/uuid"
)

// MaxAttempts is how many times the relay will retry a message before giving up
// on it. An exceeded message stays in the table for inspection; it is never
// claimed again until an operator resets it — see the package doc.
//
// Attempts are consumed once per poll, with no delay between them, so a failure
// that persists for MaxAttempts poll intervals exceeds the message. Size the
// relay's poll interval against how long you are willing to let a broker outage
// run before it needs a hand.
const MaxAttempts = 10

// Status is where a message sits in its lifecycle.
//
// The numeric values are persisted in outbox_messages.status. Append new
// statuses; never renumber an existing one.
type Status int8

const (
	// Queued is awaiting its first attempt, or a retry after a failure.
	Queued Status = iota + 1
	// Poisoned means no processor claimed the message. Retrying cannot help:
	// the relay binary does not know how to handle this payload type.
	Poisoned
	// Completed means a processor handled it successfully.
	Completed
	// Exceeded means it failed MaxAttempts times.
	Exceeded
)

// String renders the status for logs.
func (s Status) String() string {
	switch s {
	case Queued:
		return "QUEUED"
	case Poisoned:
		return "POISONED"
	case Completed:
		return "COMPLETED"
	case Exceeded:
		return "EXCEEDED"
	default:
		return "UNKNOWN(" + strconv.Itoa(int(s)) + ")"
	}
}

// Message is one row of outbox_messages.
//
// PayloadType is what the relay dispatches on. It is the event's declared name
// (see events.EventCreatedName), not a Go type name obtained by reflection: the
// row is written by one binary and read by another, possibly after a refactor
// renamed the struct or moved its package. A reflected name would stop matching
// and the backlog would be poisoned.
type Message struct {
	OccurredAt  time.Time
	CompletedAt *time.Time
	Payload     []byte
	PayloadType string
	ID          uuid.UUID
	MessageID   uuid.UUID
	Attempts    int32
	Status      Status
}

// Each transition writes itself through q, which must be the transaction that
// claimed the message. A transition that only mutated the struct would leave the
// caller to remember a separate Save — and a forgotten Save means a message that
// was published, was never marked, and is published again on the next poll.

// Complete marks the message handled. It will not be claimed again.
func (m *Message) Complete(ctx context.Context, q postgres.Querier) error {
	now := time.Now().UTC()
	m.Status = Completed
	m.CompletedAt = &now
	return m.save(ctx, q)
}

// Poison marks the message unroutable: no registered processor claims its
// payload type.
//
// This is distinct from failure. A failure is worth retrying because the cause
// may be transient — the broker was down, the network blipped. An unclaimed
// message will be unclaimed on every subsequent poll too, so retrying it only
// burns attempts and delays the messages behind it.
func (m *Message) Poison(ctx context.Context, q postgres.Querier) error {
	m.Status = Poisoned
	return m.save(ctx, q)
}

// FailOrRequeue records a failed attempt, re-queueing the message unless it has
// run out of retries.
func (m *Message) FailOrRequeue(ctx context.Context, q postgres.Querier) error {
	m.Attempts++
	if m.Attempts >= MaxAttempts {
		m.Status = Exceeded
	} else {
		m.Status = Queued
	}
	return m.save(ctx, q)
}

// save persists the current status. Called in the same transaction that claimed
// the message, so the claim and the outcome commit together.
func (m *Message) save(ctx context.Context, q postgres.Querier) error {
	_, err := q.Exec(ctx,
		`UPDATE outbox_messages
		    SET status = $1, attempts = $2, completed_at = $3
		  WHERE id = $4`,
		m.Status, m.Attempts, m.CompletedAt, m.ID)
	if err != nil {
		return fmt.Errorf("save outbox message %s: %w", m.ID, err)
	}
	return nil
}

// Enqueue records an event for publication inside the caller's transaction.
//
// q must be the same pgx.Tx that performed the business write. Passing a pool
// here instead of a transaction defeats the entire pattern: the row would
// commit independently of the write it is supposed to accompany.
//
// messageID is supplied by the caller rather than minted here, because the
// caller has already stamped it into the payload as the consumer's
// deduplication key. Minting a second one would leave the row and its payload
// disagreeing about the identity of the same message.
func Enqueue(ctx context.Context, q postgres.Querier, payloadType string, messageID uuid.UUID, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", payloadType, err)
	}

	_, err = q.Exec(ctx,
		`INSERT INTO outbox_messages (id, message_id, payload_type, payload, occurred_at, status)
		 VALUES ($1, $2, $3, $4, now(), $5)`,
		uuid.New(), messageID, payloadType, body, Queued,
	)
	if err != nil {
		return fmt.Errorf("enqueue %s: %w", payloadType, err)
	}
	return nil
}

// FetchQueued claims up to limit queued rows for this relay instance.
//
// FOR UPDATE SKIP LOCKED lets several relay replicas poll the same table
// concurrently without handing the same row to two of them, and without
// blocking each other. Rows stay claimed until the surrounding transaction
// ends, so a crashed relay releases its claim automatically.
func FetchQueued(ctx context.Context, q postgres.Querier, limit int) ([]Message, error) {
	rows, err := q.Query(ctx,
		`SELECT id, message_id, payload_type, payload, occurred_at, attempts, status
		   FROM outbox_messages
		  WHERE status = $1
		  ORDER BY occurred_at
		  LIMIT $2
		    FOR UPDATE SKIP LOCKED`, Queued, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch queued: %w", err)
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.MessageID, &m.PayloadType, &m.Payload,
			&m.OccurredAt, &m.Attempts, &m.Status); err != nil {
			return nil, fmt.Errorf("scan outbox row: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
