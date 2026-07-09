// Package publisher adapts watermill's AMQP publisher to relay.Publisher.
package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	"eventify/events"
	platformamqp "eventify/platform/amqp"

	"github.com/ThreeDotsLabs/watermill"
	wamqp "github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
)

// AMQP publishes envelopes to the shared durable topic exchange.
type AMQP struct {
	pub message.Publisher
}

// NewAMQP dials amqpURI using the shared topology.
func NewAMQP(amqpURI string) (*AMQP, error) {
	pub, err := wamqp.NewPublisher(
		platformamqp.PublisherConfig(amqpURI),
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return nil, fmt.Errorf("create amqp publisher: %w", err)
	}
	return &AMQP{pub: pub}, nil
}

// Publish sends env under routingKey.
func (a *AMQP) Publish(_ context.Context, routingKey string, env events.Envelope) error {
	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	// MessageID doubles as the watermill UUID so consumers can deduplicate
	// without unmarshalling the payload.
	msg := message.NewMessage(env.MessageID, body)
	if err := a.pub.Publish(routingKey, msg); err != nil {
		return fmt.Errorf("publish %s: %w", routingKey, err)
	}
	return nil
}

// Close releases the AMQP connection.
func (a *AMQP) Close() error { return a.pub.Close() }
