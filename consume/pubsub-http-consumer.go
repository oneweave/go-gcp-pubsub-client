package consume

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/oneweave/go-gcp-pubsub-client/shared"
)

// PubSubHTTPConsumerConfig defines configuration options for PubSubHTTPConsumer.
type PubSubHTTPConsumerConfig struct{}

// PubSubHTTPConsumer parses Google Pub/Sub push HTTP requests into CloudEvents.
type PubSubHTTPConsumer struct{}

// NewPubSubHTTPConsumer constructs a Pub/Sub push HTTP consumer.
func NewPubSubHTTPConsumer(config PubSubHTTPConsumerConfig) (*PubSubHTTPConsumer, error) {
	_ = config // no config options currently
	return &PubSubHTTPConsumer{}, nil
}

// ConsumeHTTPRequest validates and parses a Pub/Sub push request into a CloudEvent.
func (c *PubSubHTTPConsumer) ConsumeHTTPRequest(request *http.Request) (cloudevents.Event, error) {
	if request == nil {
		return cloudevents.Event{}, fmt.Errorf("request is required")
	}
	if c == nil {
		return cloudevents.Event{}, fmt.Errorf("consumer is required")
	}
	if request.Method != http.MethodPost {
		return cloudevents.Event{}, fmt.Errorf("method not allowed: %s", request.Method)
	}

	var envelope shared.PubSubPushEnvelope
	if err := json.NewDecoder(request.Body).Decode(&envelope); err != nil {
		return cloudevents.Event{}, fmt.Errorf("decode pubsub push envelope: %w", err)
	}

	if strings.TrimSpace(envelope.Message.MessageID) == "" {
		return cloudevents.Event{}, fmt.Errorf("pubsub message id is required")
	}
	if strings.TrimSpace(envelope.Message.Data) == "" {
		return cloudevents.Event{}, fmt.Errorf("pubsub message data is empty")
	}

	decodedData, err := base64.StdEncoding.DecodeString(envelope.Message.Data)
	if err != nil {
		return cloudevents.Event{}, fmt.Errorf("decode pubsub message data: %w", err)
	}

	fmt.Printf("decoded Data %s\n", decodedData)
	embeddedEvent, err := parseEmbeddedCloudEvent(decodedData)
	if err != nil {
		return cloudevents.Event{}, err
	}

	if envelope.DeliveryAttempt > 0 {
		embeddedEvent.SetExtension("deliveryattempt", envelope.DeliveryAttempt)
	}
	return embeddedEvent, nil
}

// ConsumeHTTPRequestDataAs parses a Pub/Sub push request and decodes CloudEvent data into out.
func (c *PubSubHTTPConsumer) ConsumeHTTPRequestDataAs(request *http.Request, out any) (cloudevents.Event, error) {
	if out == nil {
		return cloudevents.Event{}, fmt.Errorf("out is required")
	}

	event, err := c.ConsumeHTTPRequest(request)
	if err != nil {
		return cloudevents.Event{}, err
	}

	if err := event.DataAs(out); err != nil {
		return cloudevents.Event{}, fmt.Errorf("decode cloudevent data: %w", err)
	}

	return event, nil
}

func parseEmbeddedCloudEvent(data []byte) (cloudevents.Event, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		fmt.Printf("error unmarshalling data: %v\n", err)
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
