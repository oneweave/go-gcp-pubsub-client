package oneweavepubsub

import (
	"net/http"

	"github.com/oneweave/go-gcp-pubsub-client/v2/consume"
)

type Consumer = consume.Consumer
type EnvelopeAdapter = consume.EnvelopeAdapter
type EnvelopeMessage = consume.EnvelopeMessage

type EventConsumer = consume.EventConsumer
type EnvelopeConsumer = consume.EnvelopeConsumer
type EnvelopeConsumerConfig = consume.EnvelopeConsumerConfig
type HTTPEnvelopeAdapter = consume.HTTPEnvelopeAdapter

// NewEnvelopeConsumer creates an envelope consumer.
func NewEnvelopeConsumer(config consume.EnvelopeConsumerConfig) (*consume.EnvelopeConsumer, error) {
	return consume.NewEnvelopeConsumer(config)
}

// NewHTTPEnvelopeAdapter creates an adapter for HTTP envelope requests.
func NewHTTPEnvelopeAdapter(request *http.Request) *consume.HTTPEnvelopeAdapter {
	return consume.NewHTTPEnvelopeAdapter(request)
}
