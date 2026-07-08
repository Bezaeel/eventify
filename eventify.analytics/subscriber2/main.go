package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"eventify.analytics/handlers"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	// Create a new AMQP subscriber
	amqpURI := "amqp://guest:guest@localhost:5672/"
	subscriber, err := amqp.NewSubscriber(
		amqp.NewDurablePubSubConfig(
			amqpURI,
			func(topic string) string {
				return "eventify.events.EventCreated2"
			},
		),
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		log.Fatalf("Failed to create subscriber: %v", err)
	}

	// Create a new message router
	router, err := message.NewRouter(message.RouterConfig{}, watermill.NewStdLogger(false, false))
	if err != nil {
		log.Fatalf("Failed to create router: %v", err)
	}

	// Create handlers
	eventCreatedHandler := handlers.NewEventCreatedHandler(logger)

	messages, _ := subscriber.Subscribe(context.Background(),"eventify.events.EventCreated")
	// while loop to process messages
	go func() {
		for msg := range messages {
			if err := eventCreatedHandler.Handle(msg); err != nil {
				logger.WithError(err).Error("Failed to handle message")
			}
			msg.Ack() // Acknowledge the message after processing
		}
	}()


	// Create context that listens for the interrupt signal from the OS
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Run the router
	if err := router.Run(ctx); err != nil {
		log.Fatalf("Failed to run router: %v", err)
	}
}
