package amqp

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	wamqp "github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Publisher sends event payloads to the shared durable topic exchange.
//
// It lives beside the topology it publishes into, rather than in the outbox
// module that happens to be its only caller today. Dialing a broker is
// infrastructure, not a property of the outbox pattern: a second publisher —
// say, one that republishes from a dead-letter queue — would otherwise have to
// import outbox to reach it.
type Publisher struct {
	pub message.Publisher
}

// NewPublisher dials amqpURI using the shared topology.
func NewPublisher(amqpURI string) (*Publisher, error) {
	pub, err := wamqp.NewPublisher(
		PublisherConfig(amqpURI),
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return nil, fmt.Errorf("create amqp publisher: %w", err)
	}
	return &Publisher{pub: pub}, nil
}

// Publish sends body under routingKey.
//
// messageID doubles as the watermill UUID, so a consumer can deduplicate from
// broker metadata without unmarshalling the payload. The payload carries the
// same value, and that is the copy a consumer should trust: it survives a
// bridge, a replay from a dump, or any hop that does not preserve metadata.
func (p *Publisher) Publish(_ context.Context, routingKey, messageID string, body []byte) error {
	msg := message.NewMessage(messageID, body)
	if err := p.pub.Publish(routingKey, msg); err != nil {
		return fmt.Errorf("publish %s: %w", routingKey, err)
	}
	return nil
}

// Close releases the AMQP connection.
func (p *Publisher) Close() error { return p.pub.Close() }
