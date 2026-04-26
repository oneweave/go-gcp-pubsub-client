package oneweavepubsub

import "github.com/oneweave/oneweave-pubsub/produce"

// NewPublisher creates a high-level publisher configured for CloudEvent output.
func NewPublisher(config produce.Config, sender produce.Sender) (*produce.Publisher, error) {
	return produce.NewPublisher(config, sender)
}
