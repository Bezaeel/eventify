// Package domain holds the plain data structures the api module reads and
// writes.
//
// No ORM tags, no framework types, no service interfaces. These structs are
// scanned from SQL by hand in internal/features and encoded into DTOs by the
// transport adapters. If a struct here needs a `gorm:` tag or a fiber import,
// it has been put in the wrong layer.
//
// The `I*Service` interfaces that used to live beside these structs are gone
// along with the service layer they described.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Event is a scheduled event. Tags is persisted as JSONB.
type Event struct {
	Date      time.Time
	CreatedAt time.Time
	UpdatedAt *time.Time
	// Creator backs the GraphQL `creator: User` field. It is nil unless a query
	// explicitly joins and populates it, which none currently do — under GORM it
	// was equally always nil, because no service ever called Preload. The
	// GraphQL schema has therefore always returned `creator: null`.
	Creator     *User
	Name        string
	Description string
	Location    string
	Organizer   string
	Category    string
	Tags        []string
	ID          uuid.UUID
	CreatedBy   uuid.UUID
	Capacity    int
}

// User is an account. Password holds the bcrypt hash, never plaintext, and is
// never serialised — transports build their own response DTOs.
type User struct {
	CreatedAt time.Time
	UpdatedAt *time.Time
	Email     string
	Password  string
	FirstName string
	LastName  string
	ID        uuid.UUID
}

// Role groups permissions.
type Role struct {
	CreatedAt   time.Time
	UpdatedAt   *time.Time
	Name        string
	Description string
	ID          uuid.UUID
}

// Permission is a single named capability, e.g. "events.update".
type Permission struct {
	CreatedAt   time.Time
	UpdatedAt   *time.Time
	Name        string
	Description string
	ID          uuid.UUID
}
