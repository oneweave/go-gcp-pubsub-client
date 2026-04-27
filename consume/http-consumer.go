package consume

import (
	"fmt"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/oneweave/go-gcp-pubsub-client/shared"
)

// HTTPConsumer parses CloudEvents from HTTP requests.
type HTTPConsumer struct{}

// NewHTTPConsumer constructs an HTTP CloudEvent consumer.
func NewHTTPConsumer() *HTTPConsumer {
	return &HTTPConsumer{}
}

// ConsumeHTTPRequest parses and returns a CloudEvent from the request.
func (h *HTTPConsumer) ConsumeHTTPRequest(request *http.Request) (cloudevents.Event, error) {
	if request == nil {
		return cloudevents.Event{}, fmt.Errorf("request is required")
	}

	event, err := cehttp.NewEventFromHTTPRequest(request)
	if err != nil {
		return cloudevents.Event{}, fmt.Errorf("decode cloudevent request: %w", err)
	}

	return *event, nil
}

// DecodeHTTPCloudEventJSON decodes a CloudEvent HTTP request and returns
// both a CloudEvent with decoded JSON data and the typed decoded payload.
func DecodeHTTPCloudEventJSON[T any](request *http.Request) (cloudevents.Event, T, error) {
	var payload T
	if request == nil {
		return cloudevents.Event{}, payload, fmt.Errorf("request is required")
	}

	event, err := cehttp.NewEventFromHTTPRequest(request)
	if err != nil {
		return cloudevents.Event{}, payload, fmt.Errorf("decode cloudevent request: %w", err)
	}

	decodedPayload, decodeErr := shared.DecodeEventJSON[T](*event)
	if decodeErr != nil {
		return cloudevents.Event{}, payload, decodeErr
	}

	return *event, decodedPayload, nil
}
