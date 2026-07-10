// Command relay drains the transactional outbox onto RabbitMQ.
//
// Run one or more replicas. FOR UPDATE SKIP LOCKED in the claim query means
// replicas never contend over the same row.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"eventify/events"
	"eventify/outbox/processors"
	"eventify/outbox/relay"
	platformamqp "eventify/platform/amqp"
	"eventify/platform/config"
	"eventify/platform/logger"
	"eventify/platform/postgres"
)

func main() {
	log := logger.New(true)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbPassword, err := config.MustString("DB_PASSWORD")
	if err != nil {
		log.ErrorWithError("config", err)
		os.Exit(1)
	}

	dsn := postgres.DSN(
		config.String("DB_HOST", "localhost"),
		config.String("DB_PORT", "5432"),
		config.String("DB_USER", "postgres"),
		dbPassword,
		config.String("DB_NAME", "eventify"),
		config.String("DB_SSLMODE", "disable"),
	)

	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		log.ErrorWithError("connect postgres", err)
		os.Exit(1)
	}
	defer pool.Close()

	pub, err := platformamqp.NewPublisher(config.String("AMQP_URI", "amqp://guest:guest@localhost:5672/"))
	if err != nil {
		log.ErrorWithError("connect amqp", err)
		os.Exit(1)
	}
	defer func() { _ = pub.Close() }()

	// Every event this relay knows how to publish. An event with no processor
	// here is poisoned on its first poll rather than published, so adding an
	// event means adding a line to this slice.
	//
	// Most events publish unchanged and use NewGeneric. One that must do work
	// before it is safe to publish gets its own type embedding processors.Base.
	procs := []processors.IOutboxProcessor{
		processors.NewGeneric(pub, events.EventCreatedName),
	}

	// One relay instance, scaled vertically: raise OUTBOX_BATCH_SIZE before
	// running a second replica.
	r := relay.New(pool, procs, log,
		config.Duration("OUTBOX_POLL_INTERVAL", relay.DefaultPollInterval),
		config.Int("OUTBOX_BATCH_SIZE", relay.DefaultBatchSize),
	)

	log.Info("outbox relay started")
	if err := r.Run(ctx); err != nil && ctx.Err() == nil {
		log.ErrorWithError("relay stopped", err)
		os.Exit(1)
	}
	log.Info("outbox relay stopped")
}
