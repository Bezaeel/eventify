module eventify/subscribers

go 1.24.2

require (
	eventify/events v0.0.0
	eventify/platform v0.0.0
	github.com/ThreeDotsLabs/watermill v1.4.7
	github.com/ThreeDotsLabs/watermill-amqp/v2 v2.1.3
)

require (
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.5.5 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

// Bare module paths are not fetchable. These replaces let each module build
// standalone (Docker, CI) outside the go.work context.
replace eventify/events => ../events

replace eventify/platform => ../platform
