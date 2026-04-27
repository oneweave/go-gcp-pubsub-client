package oneweavepubsub

import (
	"github.com/oneweave/go-gcp-pubsub-client/consume"
	"github.com/oneweave/go-gcp-pubsub-client/produce"
)

type HTTPConsumer = consume.HTTPConsumer
type PubSubHTTPConsumer = consume.PubSubHTTPConsumer
type PubSubHTTPConsumerConfig = consume.PubSubHTTPConsumerConfig

// NewPublisher creates a high-level publisher configured for CloudEvent output.
func NewPublisher(config produce.Config, sender produce.Sender) (*produce.Publisher, error) {
	return produce.NewPublisher(config, sender)
}

// NewHTTPConsumer creates a CloudEvent HTTP consumer.
func NewHTTPConsumer() *consume.HTTPConsumer {
	return consume.NewHTTPConsumer()
}

// NewPubSubHTTPConsumer creates a Pub/Sub push HTTP consumer.
func NewPubSubHTTPConsumer(config consume.PubSubHTTPConsumerConfig) (*consume.PubSubHTTPConsumer, error) {
	return consume.NewPubSubHTTPConsumer(config)
}
