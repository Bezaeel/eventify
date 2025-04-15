package service_test

import (
	"context"
	"eventify/internal/domain"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db        *gorm.DB
	dbName    = "eventify_test"
	container testcontainers.Container
	ctx       = context.Background()
)

func baseIntegrationTest(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Set up the PostgreSQL container
	container, err := setupPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start Postgres container: %s", err)
	}

	// Get the connection string
	connString, err := getConnectionString(ctx, container)
	if err != nil {
		t.Fatalf("Failed to get connection string: %s", err)
	}

	// Try to connect with retries
	for i := 0; i < 5; i++ {
		db, err = gorm.Open(postgres.Open(connString), &gorm.Config{})
		if err == nil {
			break
		}
		t.Logf("Failed to connect to database, retrying... (attempt %d/5)", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		t.Fatalf("Failed to connect to the database after retries: %s", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %s", err)
	}

	return db, func() {
		cleanup()
	}
}

func cleanup() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				return fmt.Errorf("error closing database connection: %w", err)
			}
		}
	}

	if container != nil {
		if err := container.Terminate(ctx); err != nil {
			return fmt.Errorf("error terminating container: %w", err)
		}
	}

	return nil
}

func setupPostgresContainer(ctx context.Context) (testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:latest",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_DB":       dbName,
		},
		WaitingFor: wait.Strategy(
			wait.ForAll(
				wait.ForLog("database system is ready to accept connections"),
				wait.ForListeningPort("5432/tcp"),
			).WithDeadline(30 * time.Second),
		),
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

func getConnectionString(ctx context.Context, container testcontainers.Container) (string, error) {
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return "", fmt.Errorf("failed to get container port: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get container host: %w", err)
	}

	return fmt.Sprintf("host=%s port=%s user=postgres password=postgres dbname=%s sslmode=disable",
		host, port.Port(), dbName), nil
}

func runMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.Event{},
		&domain.User{},
		&domain.Role{},
		&domain.Permission{},
		&domain.UserRole{},
		&domain.RolePermissions{},
	)
}

func GetTestDB() *gorm.DB {
	return db
}
