// Package amqp defines the eventify AMQP topology in exactly one place.
//
// Both the outbox relay (publisher) and the subscribers (consumer) build their
// watermill config from here. They must agree on the exchange name, the
// exchange type, and how a topic maps to a routing key — and nothing in the
// type system forces them to. Sharing the constructors is what forces it.
//
// The topology is a single durable topic exchange:
//
//	exchange "eventify" (topic, durable)
//	  ├── routing key eventify.events.EventCreated.v1   ─┐
//	  └── routing key eventify.events.EventCancelled.v1 ─┴─► queue "eventify.analytics"
//
// One exchange, one queue per consumer group, N bindings. Adding an event adds
// a binding, not a queue and not a binary.
//
// Note that watermill's NewDurablePubSubConfig defaults to a *fanout* exchange
// named after the topic, with empty routing keys. That default gives one
// exchange per event type and makes routing keys inert, so every constructor
// below overrides it.
package amqp

import (
	wamqp "github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
)

// ExchangeName is the single durable topic exchange all eventify events flow
// through.
const ExchangeName = "eventify"

// identity is used wherever watermill hands us a topic that is already the
// fully-qualified routing key (see events.RoutingKey).
func identity(topic string) string { return topic }

func constant(name string) func(string) string {
	return func(string) string { return name }
}

// PublisherConfig builds a config that publishes to the topic exchange, using
// the topic argument as the routing key verbatim.
func PublisherConfig(amqpURI string) wamqp.Config {
	cfg := wamqp.NewDurablePubSubConfig(amqpURI, nil)
	cfg.Exchange.GenerateName = constant(ExchangeName)
	cfg.Exchange.Type = "topic"
	cfg.Exchange.Durable = true
	cfg.Publish.GenerateRoutingKey = identity
	return cfg
}

// SubscriberConfig builds a config that binds one durable queue to the topic
// exchange, once per routing key subscribed to.
//
// queueName identifies the consumer group. Two replicas of the same subscriber
// share a queue and compete for messages; two different subscribers use two
// queue names and each receive every message.
func SubscriberConfig(amqpURI, queueName string) wamqp.Config {
	cfg := wamqp.NewDurablePubSubConfig(amqpURI, constant(queueName))
	cfg.Exchange.GenerateName = constant(ExchangeName)
	cfg.Exchange.Type = "topic"
	cfg.Exchange.Durable = true
	cfg.Queue.Durable = true
	// The topic passed to Subscribe IS the routing key to bind with.
	cfg.QueueBind.GenerateRoutingKey = identity
	return cfg
}
