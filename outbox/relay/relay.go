// Package relay drains outbox_messages onto the message bus.
package relay

import (
	"context"
	"errors"
	"fmt"
	"time"

	"eventify/outbox"
	"eventify/outbox/processors"
	"eventify/platform/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// DefaultPollInterval is how long the relay waits after an empty poll.
	DefaultPollInterval = time.Second
	// DefaultBatchSize is how many messages one pass claims and processes.
	DefaultBatchSize = 100
)

// Relay polls the outbox and hands each claimed message to the processor that
// declares its payload type.
//
// One instance, scaled vertically: raise batchSize before adding a replica. The
// claim query still takes FOR UPDATE SKIP LOCKED, so a second instance would be
// correct rather than fast — it costs one clause today and is the difference
// between a config change and a redesign the day a single relay stops keeping up.
type Relay struct {
	db         *pgxpool.Pool
	log        *logger.Logger
	processors []processors.IOutboxProcessor
	interval   time.Duration
	batchSize  int
}

// New builds a Relay. A non-positive interval or batchSize takes the default.
func New(db *pgxpool.Pool, procs []processors.IOutboxProcessor, log *logger.Logger,
	interval time.Duration, batchSize int) *Relay {

	if interval <= 0 {
		interval = DefaultPollInterval
	}
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	return &Relay{db: db, processors: procs, log: log, interval: interval, batchSize: batchSize}
}

// Run processes the outbox until ctx is cancelled.
func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// A backlog larger than one batch is drained one batch per tick,
			// rather than in a nested catch-up loop. At one instance the extra
			// latency is bounded by the poll interval, and the loop it replaces
			// was the hardest thing here to follow.
			if _, err := r.ProcessTopQueued(ctx); err != nil && !errors.Is(err, context.Canceled) {
				r.log.ErrorWithError("outbox processing failed", err)
			}
		}
	}
}

// processorFor returns the processor that claims m, or nil if none does.
func (r *Relay) processorFor(m *outbox.Message) processors.IOutboxProcessor {
	for _, p := range r.processors {
		if p.CanProcess(m) {
			return p
		}
	}
	return nil
}

// ProcessTopQueued claims the oldest batch of queued messages, processes each,
// and records the outcome — all inside one transaction, so a crash mid-batch
// releases the claim and the rows are retried. It returns how many messages it
// took out of the queue.
//
// Processing happens before the commit. If a publish succeeds and the commit
// then fails, the message goes out twice. That is the deliberate trade: the
// outbox guarantees at-least-once, never exactly-once. Subscribers deduplicate
// on the payload's MessageID.
func (r *Relay) ProcessTopQueued(ctx context.Context) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin outbox tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	msgs, err := outbox.FetchQueued(ctx, tx, r.batchSize)
	if err != nil {
		return 0, err
	}
	if len(msgs) == 0 {
		return 0, nil
	}

	handled := 0
	for i := range msgs {
		m := &msgs[i]

		proc := r.processorFor(m)
		if proc == nil {
			// No processor claims this payload type, and none will on the next
			// poll either. Retrying would burn attempts and hold up the queue
			// behind it, so take it out of circulation where it can be inspected.
			r.log.Error("no processor for " + m.PayloadType + ", poisoned message " + m.MessageID.String())
			if err := m.Poison(ctx, tx); err != nil {
				return 0, err
			}
			handled++
			continue
		}

		if perr := proc.ProcessAsync(ctx, m); perr != nil {
			if err := m.FailOrRequeue(ctx, tx); err != nil {
				return 0, err
			}
			r.log.ErrorWithError("process "+m.PayloadType+" ("+m.Status.String()+")", perr)
			// Stop the batch and commit what did go out. A failure here is most
			// often the broker being unreachable, in which case every remaining
			// message would fail too — and each would spend an attempt doing it.
			// The untouched rows keep their status and are retried next poll.
			break
		}

		if err := m.Complete(ctx, tx); err != nil {
			return 0, err
		}
		handled++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit outbox tx: %w", err)
	}
	return handled, nil
}
