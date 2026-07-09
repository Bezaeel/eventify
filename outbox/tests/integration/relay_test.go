package relay_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"eventify/events"
	"eventify/outbox"
	"eventify/outbox/relay"
	"eventify/platform/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// fakePublisher records what the relay hands it, and can be told to fail.
//
// The relay declares its own Publisher interface rather than importing
// watermill, precisely so this can exist.
type fakePublisher struct {
	mu   sync.Mutex
	sent []events.Envelope
	err  error
}

func (f *fakePublisher) Publish(_ context.Context, _ string, env events.Envelope) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, env)
	return nil
}

func (f *fakePublisher) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
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

	_, thisFile, _, _ := runtime.Caller(0)
	migrations := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations"))
	sql, err := os.ReadFile(filepath.Join(migrations, "000001_create_outbox_messages.up.sql"))
	require.NoError(t, err)

	conn, err := p.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()
	_, err = conn.Conn().PgConn().Exec(ctx, string(sql)).ReadAll()
	require.NoError(t, err)

	return p
}

func enqueue(t *testing.T, p *pgxpool.Pool, n int) {
	t.Helper()
	ctx := context.Background()
	for i := range n {
		tx, err := p.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, outbox.Enqueue(ctx, tx, "EventCreated", "v1",
			map[string]any{"id": uuid.NewString(), "seq": i}))
		require.NoError(t, tx.Commit(ctx))
	}
}

func unpublished(t *testing.T, p *pgxpool.Pool) int {
	t.Helper()
	var n int
	require.NoError(t, p.QueryRow(context.Background(),
		`SELECT count(*) FROM outbox_messages WHERE published_at IS NULL`).Scan(&n))
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

func TestIntegrationRelay_DrainsAndMarksPublished(t *testing.T) {
	skipUnlessDocker(t)
	p := pool(t)
	enqueue(t, p, 3)
	require.Equal(t, 3, unpublished(t, p))

	pub := &fakePublisher{}
	r := relay.New(p, pub, logger.New(false), 50*time.Millisecond, 100)

	runFor(t, r, func() bool { return pub.count() == 3 })

	require.Equal(t, 0, unpublished(t, p), "drained rows must be stamped published_at")
	require.Len(t, pub.sent, 3)

	for _, env := range pub.sent {
		require.Equal(t, "EventCreated", env.Name)
		require.Equal(t, "v1", env.Version)
		require.NotEmpty(t, env.MessageID, "subscribers deduplicate on this")
	}
}

// A publish failure must leave the rows unpublished so the next pass retries.
// Losing them would defeat the entire pattern.
func TestIntegrationRelay_PublishFailureLeavesRowsForRetry(t *testing.T) {
	skipUnlessDocker(t)
	p := pool(t)
	enqueue(t, p, 3)

	failing := &fakePublisher{err: errors.New("broker down")}
	r := relay.New(p, failing, logger.New(false), 20*time.Millisecond, 100)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = r.Run(ctx)

	require.Equal(t, 3, unpublished(t, p), "nothing may be marked published when the broker rejects")

	// Once the broker recovers, the same rows drain.
	recovered := &fakePublisher{}
	r2 := relay.New(p, recovered, logger.New(false), 20*time.Millisecond, 100)
	runFor(t, r2, func() bool { return recovered.count() == 3 })

	require.Equal(t, 0, unpublished(t, p))
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
	require.NoError(t, outbox.Enqueue(ctx, tx, "EventCreated", "v1", map[string]any{"id": uuid.NewString()}))
	require.NoError(t, tx.Rollback(ctx))

	require.Equal(t, 0, unpublished(t, p), "a rolled-back transaction must leave no outbox row")
}
