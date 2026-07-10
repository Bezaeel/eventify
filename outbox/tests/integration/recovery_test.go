package relay_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"eventify/outbox"
	"eventify/outbox/relay"
	"eventify/platform/logger"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// A message leaves the queue for good in two ways — EXCEEDED and POISONED — and
// the outbox package doc tells an operator to reset both with a single UPDATE.
// If that query is wrong the advice is worse than none: it is the only path back
// for a backlog stalled by a broker outage.
//
// This test runs the query verbatim from that doc.
func TestIntegrationOutbox_DocumentedRecoveryQueryRequeuesStalledMessages(t *testing.T) {
	skipUnlessDocker(t)
	p := pool(t)
	ctx := context.Background()

	// One message nothing can publish, because the broker is down -> EXCEEDED.
	enqueueEvents(t, p, 1)
	failing := &fakePublisher{err: errors.New("broker down")}
	r := relay.New(p, eventCreatedProcessors(failing), logger.New(false), 10*time.Millisecond, 100)
	runFor(t, r, func() bool { return countByStatus(t, p, outbox.Exceeded) == 1 })

	// One message no processor claims -> POISONED.
	tx, err := p.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, outbox.Enqueue(ctx, tx, "NobodyHandlesThis", uuid.New(), map[string]any{}))
	require.NoError(t, tx.Commit(ctx))

	r2 := relay.New(p, eventCreatedProcessors(failing), logger.New(false), 10*time.Millisecond, 100)
	runFor(t, r2, func() bool { return countByStatus(t, p, outbox.Poisoned) == 1 })

	// Verbatim from the outbox package doc. Statuses 2 and 4 are POISONED and
	// EXCEEDED; if those constants are ever renumbered, this fails.
	_, err = p.Exec(ctx, `UPDATE outbox_messages SET status = 1, attempts = 0 WHERE status IN (2, 4)`)
	require.NoError(t, err)

	require.Equal(t, 2, countByStatus(t, p, outbox.Queued), "both stalled rows must return to the queue")
	require.Equal(t, 0, countByStatus(t, p, outbox.Exceeded))
	require.Equal(t, 0, countByStatus(t, p, outbox.Poisoned))

	// Once the cause is fixed, the reset message really publishes. The unknown
	// payload type poisons again, which is correct: resetting a row does not
	// teach the relay an event it was never built to handle.
	failing.recover()
	r3 := relay.New(p, eventCreatedProcessors(failing), logger.New(false), 20*time.Millisecond, 100)
	runFor(t, r3, func() bool { return countByStatus(t, p, outbox.Completed) == 1 })

	require.Equal(t, 1, failing.count(), "the reset message really did publish")
	require.Equal(t, 1, countByStatus(t, p, outbox.Poisoned), "the unknown event poisons again")
}
