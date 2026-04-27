package produce

import (
	"context"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/oneweave/go-gcp-pubsub-client/shared"
)

// Config controls publisher defaults.
type Config struct {
	Source            string
	DefaultEventType  string
	DefaultExtensions map[string]any
}

// Publisher wraps payloads in CloudEvents and sends them through a transport sender.
type Publisher struct {
	sender            Sender
	source            string
	defaultEventType  string
	defaultExtensions map[string]any
	now               func() time.Time
	newID             func() string
}

// NewPublisher constructs a publisher with safe defaults.
func NewPublisher(config Config, sender Sender) (*Publisher, error) {
	if sender == nil {
		return nil, fmt.Errorf("sender is required")
	}
	if config.Source == "" {
		return nil, fmt.Errorf("source is required")
	}

	return &Publisher{
		sender:            sender,
		source:            config.Source,
		defaultEventType:  config.DefaultEventType,
		defaultExtensions: copyMap(config.DefaultExtensions),
		now:               time.Now,
		newID:             func() string { return uuid.NewString() },
	}, nil
}

// Publish wraps payload into a CloudEvent and sends it through the configured sender.
func (p *Publisher) Publish(ctx context.Context, eventType string, payload any, opts ...PublishOption) (cloudevents.Event, error) {
	if payload == nil {
		return cloudevents.Event{}, fmt.Errorf("payload is required")
	}

	resolvedType := eventType
	if resolvedType == "" {
		resolvedType = p.defaultEventType
	}
	if resolvedType == "" {
		return cloudevents.Event{}, fmt.Errorf("event type is required")
	}

	options := publishOptions{dataContentType: shared.JSONContentType}
	for _, opt := range opts {
		opt(&options)
	}

	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetID(p.newID())
	event.SetSource(p.source)
	event.SetType(resolvedType)
	event.SetTime(p.now().UTC())
	if options.subject != "" {
		event.SetSubject(options.subject)
	}

	for k, v := range p.defaultExtensions {
		event.SetExtension(k, v)
	}
	for k, v := range options.extensions {
		event.SetExtension(k, v)
	}

	contentType := options.dataContentType
	if contentType == "" {
		contentType = shared.JSONContentType
	}
	if err := shared.SetEventData(&event, contentType, payload); err != nil {
		return cloudevents.Event{}, err
	}
	if err := p.sender.Send(ctx, event); err != nil {
		return cloudevents.Event{}, fmt.Errorf("send cloudevent: %w", err)
	}

	return event, nil
}

// PublishToTopic is a convenience helper that sets the topic in CloudEvent subject.
func (p *Publisher) PublishToTopic(ctx context.Context, topic, eventType string, payload any, opts ...PublishOption) (cloudevents.Event, error) {
	if topic != "" {
		opts = append(opts, WithSubject(topic))
	}
	return p.Publish(ctx, eventType, payload, opts...)
}

func copyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
