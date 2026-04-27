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
	if !json.Valid(decodedData) {
		return cloudevents.Event{}, fmt.Errorf("pubsub message data is not valid JSON")
	}

	if embeddedEvent, ok := parseEmbeddedCloudEvent(decodedData); ok {
		if envelope.DeliveryAttempt > 0 {
			embeddedEvent.SetExtension("deliveryattempt", envelope.DeliveryAttempt)
		}
		return embeddedEvent, nil
	}

	return cloudevents.Event{}, fmt.Errorf("pubsub message data is not a cloudevent")
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

func parseEmbeddedCloudEvent(data []byte) (cloudevents.Event, bool) {
	var embedded cloudevents.Event
	if err := json.Unmarshal(data, &embedded); err != nil {
		return cloudevents.Event{}, false
	}
	if embedded.SpecVersion() == "" || embedded.ID() == "" || embedded.Source() == "" || embedded.Type() == "" {
		return cloudevents.Event{}, false
	}
	return embedded, true
}
