package consume

import (
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// Consumer consumes a CloudEvent from an envelope adapter.
type Consumer interface {
	Consume(adapter EnvelopeAdapter) (cloudevents.Event, error)
}

// EnvelopeMessage is the transport-neutral envelope payload passed to EnvelopeConsumer.
type EnvelopeMessage struct {
	MessageID       string
	EventData       []byte
	DeliveryAttempt int
}

// EnvelopeAdapter adapts a transport-specific envelope into an EnvelopeMessage.
type EnvelopeAdapter interface {
	Envelope() (EnvelopeMessage, error)
}

// EventConsumer parses direct CloudEvents from requests.
type EventConsumer struct{}

// EnvelopeConsumerConfig defines configuration options for EnvelopeConsumer.
type EnvelopeConsumerConfig struct{}

// EnvelopeConsumer parses push-style envelopes into embedded CloudEvents.
//
// The current implementation expects the Pub/Sub push envelope shape where
// message.data contains base64-encoded CloudEvent JSON.
type EnvelopeConsumer struct{}
