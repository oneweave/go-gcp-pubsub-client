package consume

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// Compile-time assertion that EnvelopeConsumer implements Consumer.
var _ Consumer = (*EnvelopeConsumer)(nil)

// NewEnvelopeConsumer constructs an envelope consumer.
func NewEnvelopeConsumer(config EnvelopeConsumerConfig) (*EnvelopeConsumer, error) {
	_ = config // no config options currently
	return &EnvelopeConsumer{}, nil
}

// Consume validates and parses envelope payload into a CloudEvent.
func (c *EnvelopeConsumer) Consume(adapter EnvelopeAdapter) (cloudevents.Event, error) {
	if c == nil {
		return cloudevents.Event{}, fmt.Errorf("consumer is required")
	}
	if adapter == nil {
		return cloudevents.Event{}, fmt.Errorf("adapter is required")
	}

	envelope, err := adapter.Envelope()
	if err != nil {
		return cloudevents.Event{}, err
	}

	if envelope.MessageID == "" {
		return cloudevents.Event{}, fmt.Errorf("pubsub message id is required")
	}
	if len(envelope.EventData) == 0 {
		return cloudevents.Event{}, fmt.Errorf("pubsub message data is empty")
	}

	embeddedEvent, err := parseEmbeddedCloudEvent(envelope.EventData)
	if err != nil {
		return cloudevents.Event{}, err
	}

	if envelope.DeliveryAttempt > 0 {
		embeddedEvent.SetExtension("deliveryattempt", envelope.DeliveryAttempt)
	}
	return embeddedEvent, nil
}

func parseEmbeddedCloudEvent(data []byte) (cloudevents.Event, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return cloudevents.Event{}, fmt.Errorf("pubsub message data is not valid JSON")
	}

	if _, ok := raw["specversion"]; !ok {
		return cloudevents.Event{}, fmt.Errorf("pubsub message data is not a cloudevent")
	}
	if _, ok := raw["id"]; !ok {
		return cloudevents.Event{}, fmt.Errorf("pubsub message data is not a cloudevent")
	}
	if _, ok := raw["source"]; !ok {
		return cloudevents.Event{}, fmt.Errorf("pubsub message data is not a cloudevent")
	}
	if _, ok := raw["type"]; !ok {
		return cloudevents.Event{}, fmt.Errorf("pubsub message data is not a cloudevent")
	}

	var embedded cloudevents.Event
	if err := json.Unmarshal(data, &embedded); err != nil {
		return cloudevents.Event{}, fmt.Errorf("pubsub message data is not a cloudevent")
	}
	if embedded.SpecVersion() == "" || embedded.ID() == "" || embedded.Source() == "" || embedded.Type() == "" {
		return cloudevents.Event{}, fmt.Errorf("pubsub message data is not a cloudevent")
	}
	return embedded, nil
}
