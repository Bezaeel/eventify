// Package relay drains outbox_messages onto the message bus.
package relay

import (
	"context"
	"errors"
	"fmt"
	"time"

	"eventify/events"
	"eventify/outbox"
	"eventify/platform/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Publisher is the bus the relay writes to. Declared here rather than imported
// from watermill so the relay can be tested against an in-memory fake.
type Publisher interface {
	Publish(ctx context.Context, routingKey string, env events.Envelope) error
}

// Relay polls the outbox and republishes anything unpublished.
type Relay struct {
	db        *pgxpool.Pool
	pub       Publisher
	log       *logger.Logger
	interval  time.Duration
	batchSize int
}

// New builds a Relay. interval is how long to wait after an empty poll;
// batchSize caps how many rows one pass claims.
func New(db *pgxpool.Pool, pub Publisher, log *logger.Logger, interval time.Duration, batchSize int) *Relay {
	if interval <= 0 {
		interval = time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return &Relay{db: db, pub: pub, log: log, interval: interval, batchSize: batchSize}
}

// Run drains the outbox until ctx is cancelled.
func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			n, err := r.drainOnce(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				r.log.ErrorWithError("outbox drain failed", err)
				continue
			}
			// A full batch means there is probably more waiting; poll again
			// immediately rather than sleeping through the backlog.
			for err == nil && n == r.batchSize {
				n, err = r.drainOnce(ctx)
			}
		}
	}
}

// drainOnce claims a batch, publishes it, and marks it published — all inside
// one transaction, so a crash mid-batch releases the claim and the rows are
// retried.
//
// Publishing happens before the commit. If the publish succeeds and the commit
// then fails, the messages go out twice. That is the deliberate trade: the
// outbox guarantees at-least-once, never exactly-once. Subscribers deduplicate
// on Envelope.MessageID.
func (r *Relay) drainOnce(ctx context.Context) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin outbox tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	msgs, err := outbox.FetchUnpublished(ctx, tx, r.batchSize)
	if err != nil {
		return 0, err
	}
	if len(msgs) == 0 {
		return 0, nil
	}

	published := make([]uuid.UUID, 0, len(msgs))
	for _, m := range msgs {
		env := events.Envelope{
			Name:       m.Name,
			Version:    m.Version,
			MessageID:  m.MessageID.String(),
			OccurredAt: m.OccurredAt.Format(time.RFC3339),
			Payload:    m.Payload,
		}
		key := events.RoutingKey(m.Name, m.Version)
		if err := r.pub.Publish(ctx, key, env); err != nil {
			// Stop at the first failure and commit what did go out. The rest
			// keep published_at NULL and are retried on the next pass.
			r.log.ErrorWithError("publish "+key+" failed", err)
			break
		}
		published = append(published, m.ID)
	}

	if err := outbox.MarkPublished(ctx, tx, published); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit outbox tx: %w", err)
	}

	return len(published), nil
}
