package rmq

import (
	"encoding/json"
	"fmt"

	"eventify.analytics/publisher/extensions"
	"github.com/streadway/amqp"
)

type EventPublisher struct {
	channel *amqp.Channel
}

func NewEventPublisher(ch *amqp.Channel) *EventPublisher {
	// Declare the topic exchange
	err := ch.ExchangeDeclare(
		"eventify", // exchange name
		"topic",    // exchange type
		true,       // durable
		false,      // auto-deleted
		false,      // internal
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		return nil
	}

	return &EventPublisher{
		channel: ch,
	}
}

func (p *EventPublisher) Publish(event interface{}) error {
	// Get the type name and use it as routing key
	// Convert type name to routing key format
	// e.g., "CourseCreated" -> "eventify.CourseCreated"
	fmt.Printf("eventify.%s \n", extensions.GetType(event))
	routingKey := fmt.Sprintf("eventify.%s", extensions.GetType(event))

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.channel.Publish(
		"eventify", // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
