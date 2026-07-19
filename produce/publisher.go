package produce

import (
	"context"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// Config controls publisher defaults.
type Config struct {
	// for future use, currently unused
}

// Publisher wraps payloads in CloudEvents and sends them through a transport sender.
type Publisher struct {
	sender Sender
}

// NewPublisher constructs a publisher with safe defaults.
func NewPublisher(config Config, sender Sender) (*Publisher, error) {
	if sender == nil {
		return nil, fmt.Errorf("sender is required")
	}

	return &Publisher{
		sender: sender,
	}, nil
}

// Publish validates and sends a CloudEvent through the configured sender.
func (p *Publisher) Publish(ctx context.Context, event cloudevents.Event, opts ...PublishOption) (cloudevents.Event, error) {
	options := publishOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	if options.subject != "" {
		event.SetSubject(options.subject)
	}

	for k, v := range options.extensions {
		event.SetExtension(k, v)
	}

	if options.dataContentType != "" {
		event.SetDataContentType(options.dataContentType)
	}

	if err := event.Validate(); err != nil {
		return cloudevents.Event{}, fmt.Errorf("invalid cloudevent: %w", err)
	}

	if err := p.sender.Send(ctx, event); err != nil {
		return cloudevents.Event{}, fmt.Errorf("send cloudevent: %w", err)
	}

	return event, nil
}
