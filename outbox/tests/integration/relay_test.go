package relay_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"eventify/events"
	"eventify/outbox"
	"eventify/outbox/processors"
	"eventify/outbox/relay"
	"eventify/platform/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// published is one call to fakePublisher.Publish.
type published struct {
	routingKey string
	messageID  string
	body       []byte
}

// fakePublisher records what a processor hands it, and can be told to fail.
//
// processors declares its own Publisher interface rather than importing
// watermill, precisely so this can exist.
type fakePublisher struct {
	sent []published
	err  error
	mu   sync.Mutex
}

func (f *fakePublisher) Publish(_ context.Context, routingKey, messageID string, body []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, published{routingKey, messageID, body})
	return nil
}

func (f *fakePublisher) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
}

func (f *fakePublisher) recover() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = nil
}

// eventCreatedProcessors is the production processor set: one generic
// passthrough for EventCreated.
func eventCreatedProcessors(pub processors.Publisher) []processors.IOutboxProcessor {
	return []processors.IOutboxProcessor{
		processors.NewGeneric(pub, events.EventCreatedName),
	}
}

func skipUnlessDocker(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: requires docker (-short given)")
	}
}

func pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("eventify_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	p, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(p.Close)

	migrate(t, p)
	return p
}

// migrate applies every up migration in order, so the test schema is the schema
// the relay will actually meet in production rather than a hand-maintained copy.
func migrate(t *testing.T, p *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations"))
	ups, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, ups, "no migrations found in %s", dir)
	sort.Strings(ups)

	conn, err := p.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()

	for _, up := range ups {
		sql, err := os.ReadFile(up)
		require.NoError(t, err)
		_, err = conn.Conn().PgConn().Exec(ctx, string(sql)).ReadAll()
		require.NoError(t, err, "applying %s", filepath.Base(up))
	}
}

// enqueueEvents writes n valid EventCreated rows, each in its own transaction.
func enqueueEvents(t *testing.T, p *pgxpool.Pool, n int) {
	t.Helper()
	ctx := context.Background()
	for range n {
		messageID := uuid.New()
		evt := events.EventCreated{
			MessageID:  messageID,
			ID:         uuid.New(),
			Name:       "Summer Gala",
			Type:       "conference",
			DoneBy:     uuid.NewString(),
			OccurredAt: time.Now().UTC(),
		}
		tx, err := p.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, outbox.Enqueue(ctx, tx, events.EventCreatedName, messageID, evt))
		require.NoError(t, tx.Commit(ctx))
	}
}

// countByStatus reports how many rows sit in the given status.
func countByStatus(t *testing.T, p *pgxpool.Pool, s outbox.Status) int {
	t.Helper()
	var n int
	require.NoError(t, p.QueryRow(context.Background(),
		`SELECT count(*) FROM outbox_messages WHERE status = $1`, s).Scan(&n))
	return n
}

// runFor starts the relay and stops it once cond holds, or the deadline passes.
func runFor(t *testing.T, r *relay.Relay, cond func() bool) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); _ = r.Run(ctx) }()

	deadline := time.After(15 * time.Second)
	for !cond() {
		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatal("relay did not reach the expected state in time")
		case <-time.After(20 * time.Millisecond):
		}
	}
	cancel()
	<-done
}

func TestIntegrationRelay_DrainsAndCompletes(t *testing.T) {
	skipUnlessDocker(t)
	p := pool(t)
	enqueueEvents(t, p, 3)
	require.Equal(t, 3, countByStatus(t, p, outbox.Queued))

	pub := &fakePublisher{}
	r := relay.New(p, eventCreatedProcessors(pub), logger.New(false), 50*time.Millisecond, 100)

	runFor(t, r, func() bool { return pub.count() == 3 })

	require.Equal(t, 0, countByStatus(t, p, outbox.Queued))
	require.Equal(t, 3, countByStatus(t, p, outbox.Completed), "drained rows must be marked COMPLETED")
	require.Len(t, pub.sent, 3)

	for _, got := range pub.sent {
		require.Equal(t, events.RoutingKey(events.EventCreatedName), got.routingKey)
		require.NotEmpty(t, got.messageID, "subscribers deduplicate on this")
		require.Contains(t, string(got.body), got.messageID,
			"the payload must carry the same message id the broker sees")
	}
}

// A publish failure must leave the rows queued so a later pass retries. Losing
// them would defeat the entire pattern.
func TestIntegrationRelay_PublishFailureRequeuesForRetry(t *testing.T) {
	skipUnlessDocker(t)
	p := pool(t)
	enqueueEvents(t, p, 3)

	// Poll slowly enough that the brief outage below cannot spend all
	// MaxAttempts — attempts are consumed one per poll, with no delay between.
	failing := &fakePublisher{err: errors.New("broker down")}
	r := relay.New(p, eventCreatedProcessors(failing), logger.New(false), 50*time.Millisecond, 100)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()
	_ = r.Run(ctx)

	require.Equal(t, 0, countByStatus(t, p, outbox.Completed),
		"nothing may be marked completed when the broker rejects")
	require.Equal(t, 3, countByStatus(t, p, outbox.Queued))
	require.Equal(t, 0, countByStatus(t, p, outbox.Exceeded))

	// Once the broker recovers, the same rows drain.
	failing.recover()
	r2 := relay.New(p, eventCreatedProcessors(failing), logger.New(false), 20*time.Millisecond, 100)
	runFor(t, r2, func() bool { return failing.count() == 3 })

	require.Equal(t, 3, countByStatus(t, p, outbox.Completed))
}

// A message that keeps failing eventually stops being retried, rather than
// looping forever behind a broker that will never accept it.
//
// Note the coupling this asserts: attempts are spent one per poll, with no
// delay. A relay polling every 10ms exceeds a message in 100ms of downtime.
func TestIntegrationRelay_ExhaustedRetriesStopTheMessage(t *testing.T) {
	skipUnlessDocker(t)
	p := pool(t)
	enqueueEvents(t, p, 1)

	failing := &fakePublisher{err: errors.New("broker down")}
	r := relay.New(p, eventCreatedProcessors(failing), logger.New(false), 10*time.Millisecond, 100)

	runFor(t, r, func() bool { return countByStatus(t, p, outbox.Exceeded) == 1 })

	require.Equal(t, 0, countByStatus(t, p, outbox.Queued), "an exceeded message is never claimed again")
	require.Equal(t, 0, failing.count())

	var attempts int32
	require.NoError(t, p.QueryRow(context.Background(),
		`SELECT attempts FROM outbox_messages`).Scan(&attempts))
	require.Equal(t, int32(outbox.MaxAttempts), attempts)
}

// An event no processor claims can never succeed. Retrying it burns attempts
// and holds up the queue, so it is taken out of circulation immediately.
func TestIntegrationRelay_UnclaimedEventIsPoisoned(t *testing.T) {
	skipUnlessDocker(t)
	p := pool(t)
	ctx := context.Background()

	tx, err := p.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, outbox.Enqueue(ctx, tx, "NobodyHandlesThis", uuid.New(),
		map[string]any{"id": uuid.NewString()}))
	require.NoError(t, tx.Commit(ctx))

	pub := &fakePublisher{}
	r := relay.New(p, eventCreatedProcessors(pub), logger.New(false), 20*time.Millisecond, 100)

	runFor(t, r, func() bool { return countByStatus(t, p, outbox.Poisoned) == 1 })

	require.Equal(t, 0, pub.count(), "an unclaimed message must never be published")
	require.Equal(t, 0, countByStatus(t, p, outbox.Queued))

	var attempts int32
	require.NoError(t, p.QueryRow(ctx, `SELECT attempts FROM outbox_messages`).Scan(&attempts))
	require.Equal(t, int32(0), attempts, "poisoning is not a failed attempt; retrying cannot help")
}

// Enqueue must join the caller's transaction. If it did not, a rolled-back
// business write would still leave an outbox row, and the relay would announce
// an event that never happened.
func TestIntegrationEnqueue_RollsBackWithTheCallersTransaction(t *testing.T) {
	skipUnlessDocker(t)
	p := pool(t)
	ctx := context.Background()

	tx, err := p.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, outbox.Enqueue(ctx, tx, events.EventCreatedName, uuid.New(),
		events.EventCreated{MessageID: uuid.New()}))
	require.NoError(t, tx.Rollback(ctx))

	require.Equal(t, 0, countByStatus(t, p, outbox.Queued),
		"a rolled-back transaction must leave no outbox row")
}
