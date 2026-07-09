// Package testsupport spins up a real Postgres for integration tests.
//
// Feature handlers hold raw SQL. A mock cannot tell you that a query parses,
// that its columns exist, that ON CONFLICT hits a real constraint, or that a
// FOR UPDATE SKIP LOCKED does what you think. Only a database can.
package testsupport

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// repoRoot returns the workspace root, derived from this file's own path so it
// does not depend on the working directory a test happens to run in.
func repoRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	// .../api/tests/integration/testsupport/postgres.go -> up 4 -> api -> up 1 -> root
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", ".."))
}

// Pool starts a Postgres container, applies every migration the api depends on,
// and returns a pool pointed at it. The container is terminated when the test
// finishes.
//
// Migrations are applied in module order: api owns users/roles/permissions/
// events, and outbox owns outbox_messages. CreateEventHandler writes to both in
// one transaction, so both must exist.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("eventify_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("container dsn: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}
	t.Cleanup(pool.Close)

	root := repoRoot()
	applyMigrations(t, pool, filepath.Join(root, "api", "internal", "migrations"))
	applyMigrations(t, pool, filepath.Join(root, "outbox", "migrations"))

	return pool
}

// applyMigrations executes every *.up.sql in dir, in filename order.
//
// A migration file holds several statements. pgx's Exec uses the extended
// protocol, which permits exactly one statement per call, so the raw pgconn
// (simple protocol) is used instead.
func applyMigrations(t *testing.T, pool *pgxpool.Pool, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations %s: %v", dir, err)
	}

	var ups []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			ups = append(ups, e.Name())
		}
	}
	sort.Strings(ups)

	ctx := context.Background()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire conn: %v", err)
	}
	defer conn.Release()

	for _, name := range ups {
		sql, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := conn.Conn().PgConn().Exec(ctx, string(sql)).ReadAll(); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
}

// SkipUnlessDocker skips a test when it cannot start containers, so that
// `make test-unit` (which passes -short) stays Docker-free.
func SkipUnlessDocker(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: requires docker (-short given)")
	}
}
