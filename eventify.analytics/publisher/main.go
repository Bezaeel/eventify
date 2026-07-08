package main

import (
	"encoding/json"
	"fmt"
	"time"

	"eventify.analytics/events"
	// "eventify.analytics/publisher/rmq"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

func main() {
	// publisher logic here
	// rmq.ConnectAmqp()

	// publisher := rmq.NewEventPublisher(rmq.Channel)
	amqpURI := "amqp://guest:guest@localhost:5672/"
	publisher, err := amqp.NewPublisher(
		amqp.NewDurablePubSubConfig(
			amqpURI,
			func(topic string) string {
				return "eventify"
			},
		),
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		fmt.Printf("Failed to create publisher: %v\n", err)
		return
	}

	countryCodes := []string{"NG", "UK", "GH", "BR"}

	count := 5
	offset := 0
	for i := 0; i < count; i++ {
		pointer := (i + offset) % len(countryCodes)

		fmt.Printf("Publishing event %d with country code %s\n", i+1, countryCodes[pointer])
		var messagey  = events.EventCreated{
			MessageId:   uuid.New(),
			Name:        "Event " + uuid.New().String(),
			Type:        "Type" + uuid.New().String(),
			CountryCode: countryCodes[pointer],
			OccurredAt:  time.Now().AddDate(0, 0, i).Format(time.RFC3339),
			CreatedBy:   "system",
		}

		msgStr, err := json.Marshal(messagey)
		if err != nil {
			fmt.Printf("Failed to marshal message: %v\n", err)
			continue
		}

		publisher.Publish("eventify.events.EventCreated", message.NewMessage(
			uuid.New().String(),
			msgStr,
		))
	}
}