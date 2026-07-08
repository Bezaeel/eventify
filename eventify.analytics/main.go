package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
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

	url := "http://localhost:9999/druid/indexer/v1/task"
	// use req.json file as request body
	jsonData, err := os.ReadFile("req.json")
	if err != nil {
		logger.Fatalf("Failed to read request file: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Print(err)
	}
	defer resp.Body.Close()

	// Read response body (whether 200 or 400)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Request failed with status: %s\n", resp.Status)
		fmt.Printf("Response body: %s\n", string(body))
		return
	}

	// Create a new AMQP subscriber
	amqpURI := "amqp://guest:guest@localhost:5672/"
	subscriber, err := amqp.NewSubscriber(
		amqp.NewDurablePubSubConfig(
			amqpURI,
			func(topic string) string {
				return "eventify.events.EventCreated1"
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

	messages, _ := subscriber.Subscribe(context.Background(), "eventify.events.EventCreated")
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
