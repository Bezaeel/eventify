// Command subscriber consumes eventify events and projects them into the
// analytics read model.
//
// One binary, one queue, many events. Handlers register per (event, version)
// and the router binds the queue to each handler's routing key. Adding a new
// event means adding one Handler to the registry below — not a new main.go,
// which is how the previous analytics module ended up with two near-identical
// entry points differing only by a queue-name string.
package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"eventify/events"
	platformamqp "eventify/platform/amqp"
	"eventify/platform/config"
	"eventify/platform/logger"
	"eventify/platform/postgres"
	"eventify/subscribers/internal/handler"

	"github.com/ThreeDotsLabs/watermill"
	wamqp "github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
)

const queueName = "eventify.analytics"

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

	// Register every event this binary consumes. Add new handlers here.
	registry, err := handler.NewRegistry(
		handler.NewEventCreatedV1(pool, log),
	)
	if err != nil {
		log.ErrorWithError("build handler registry", err)
		os.Exit(1)
	}

	// Topology comes from platform/amqp so the relay and this subscriber cannot
	// disagree about the exchange name, its type, or the routing keys.
	amqpCfg := platformamqp.SubscriberConfig(
		config.String("AMQP_URI", "amqp://guest:guest@localhost:5672/"),
		queueName,
	)
	subscriber, err := wamqp.NewSubscriber(amqpCfg, watermill.NewStdLogger(false, false))
	if err != nil {
		log.ErrorWithError("connect amqp", err)
		os.Exit(1)
	}
	defer func() { _ = subscriber.Close() }()

	for _, key := range registry.RoutingKeys() {
		msgs, err := subscriber.Subscribe(ctx, key)
		if err != nil {
			log.ErrorWithError("subscribe "+key, err)
			os.Exit(1)
		}
		go consume(ctx, msgs, registry, log)
		log.Info("subscribed to " + key)
	}

	log.Info("subscriber started")
	<-ctx.Done()
	log.Info("subscriber stopped")
}

// consume dispatches each message, acking only on success.
//
// The previous implementation acked unconditionally — it called msg.Ack() after
// logging the handler error, so a failed projection silently discarded the
// event. Nacking instead redelivers, and lets a dead-letter policy catch
// messages that can never succeed.
func consume(ctx context.Context, msgs <-chan *message.Message, registry *handler.Registry, log *logger.Logger) {
	for msg := range msgs {
		var env events.Envelope
		if err := json.Unmarshal(msg.Payload, &env); err != nil {
			log.ErrorWithError("unmarshal envelope", err)
			msg.Nack()
			continue
		}

		if err := registry.Dispatch(ctx, env); err != nil {
			log.ErrorWithError("dispatch "+env.Name+"."+env.Version, err)
			msg.Nack()
			continue
		}
		msg.Ack()
	}
}
