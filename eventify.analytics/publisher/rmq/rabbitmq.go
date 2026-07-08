package rmq

import (
	"fmt"
	"log"
	"os"

	"github.com/streadway/amqp"
)

var Channel *amqp.Channel

// ConnectAmqp connects to RabbitMQ and returns the connection and channel
func ConnectAmqp() (*amqp.Connection, *amqp.Channel, error) {
	// Get RabbitMQ connection details from environment variables
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://guest:guest@localhost:5672/"
	}

	// Connect to RabbitMQ
	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Create a channel
	Channel, err = conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	log.Println("Connected to RabbitMQ")
	return conn, Channel, nil
}
