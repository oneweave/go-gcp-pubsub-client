package produce

import (
	"context"
	"fmt"
	"io"
	"sync"

	gcppubsub "cloud.google.com/go/pubsub/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type googlePubSubTopic interface {
	Publish(context.Context, *gcppubsub.Message) googlePubSubPublishResult
	Stop()
}

type googlePubSubPublishResult interface {
	Get(context.Context) (string, error)
}

type googlePubSubTopicAdapter struct {
	publisher *gcppubsub.Publisher
}

func (a googlePubSubTopicAdapter) Publish(ctx context.Context, msg *gcppubsub.Message) googlePubSubPublishResult {
	return googlePubSubPublishResultAdapter{result: a.publisher.Publish(ctx, msg)}
}

func (a googlePubSubTopicAdapter) Stop() {
	a.publisher.Stop()
}

type googlePubSubPublishResultAdapter struct {
	result *gcppubsub.PublishResult
}

func (a googlePubSubPublishResultAdapter) Get(ctx context.Context) (string, error) {
	return a.result.Get(ctx)
}

// GooglePubSubSender publishes CloudEvent JSON envelopes into Google Pub/Sub message data.
type GooglePubSubSender struct {
	topic googlePubSubTopic
}

// Compile-time assertion that GooglePubSubSender implements Sender.
var _ Sender = (*GooglePubSubSender)(nil)

// NewGooglePubSubSender creates a sender bound to a specific Pub/Sub publisher.
func NewGooglePubSubSender(publisher *gcppubsub.Publisher) (Sender, error) {
	if publisher == nil {
		return nil, fmt.Errorf("pubsub publisher is required")
	}

	return &GooglePubSubSender{topic: googlePubSubTopicAdapter{publisher: publisher}}, nil
}

// Send publishes the CloudEvent as JSON in Pub/Sub message data.
func (s *GooglePubSubSender) Send(ctx context.Context, event cloudevents.Event) error {
	if s == nil || s.topic == nil {
		return fmt.Errorf("pubsub publisher is required")
	}

	payload, err := event.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal cloudevent: %w", err)
	}

	result := s.topic.Publish(ctx, &gcppubsub.Message{Data: payload})
	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("publish pubsub message: %w", err)
	}

	return nil
}

type googlePubSubClient interface {
	Publisher(topicID string) googlePubSubTopic
}

type googlePubSubClientAdapter struct {
	client *gcppubsub.Client
}

func (a googlePubSubClientAdapter) Publisher(topicID string) googlePubSubTopic {
	return googlePubSubTopicAdapter{publisher: a.client.Publisher(topicID)}
}

// TopicResolver resolves the topic name for a given CloudEvent.
type TopicResolver func(event cloudevents.Event) string

// DefaultTopicResolver resolves the topic using the CloudEvent Type.
func DefaultTopicResolver(event cloudevents.Event) string {
	return event.Type()
}

// GooglePubSubClientSenderOption configures GooglePubSubClientSender.
type GooglePubSubClientSenderOption func(*GooglePubSubClientSender)

// WithTopicResolver overrides the default topic resolver.
func WithTopicResolver(resolver TopicResolver) GooglePubSubClientSenderOption {
	return func(s *GooglePubSubClientSender) {
		if resolver != nil {
			s.resolver = resolver
		}
	}
}

// GooglePubSubClientSender publishes CloudEvent JSON envelopes into Google Pub/Sub.
// It resolves the destination topic dynamically per event.
type GooglePubSubClientSender struct {
	client     googlePubSubClient
	resolver   TopicResolver
	mu         sync.RWMutex
	publishers map[string]googlePubSubTopic
	closed     bool
}

// Compile-time assertion that GooglePubSubClientSender implements Sender and io.Closer.
var _ Sender = (*GooglePubSubClientSender)(nil)
var _ io.Closer = (*GooglePubSubClientSender)(nil)

// NewGooglePubSubClientSender creates a sender that uses a Google Pub/Sub client to dynamically route messages.
func NewGooglePubSubClientSender(client *gcppubsub.Client, opts ...GooglePubSubClientSenderOption) (*GooglePubSubClientSender, error) {
	if client == nil {
		return nil, fmt.Errorf("pubsub client is required")
	}

	sender := &GooglePubSubClientSender{
		client:     googlePubSubClientAdapter{client: client},
		resolver:   DefaultTopicResolver,
		publishers: make(map[string]googlePubSubTopic),
	}

	for _, opt := range opts {
		opt(sender)
	}

	return sender, nil
}

// Send publishes the CloudEvent as JSON to the resolved topic in Google Pub/Sub.
func (s *GooglePubSubClientSender) Send(ctx context.Context, event cloudevents.Event) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("pubsub client is required")
	}

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("sender is closed")
	}
	s.mu.RUnlock()

	topicName := s.resolver(event)
	if topicName == "" {
		return fmt.Errorf("resolved topic name is empty")
	}

	pub, err := s.getOrCreatePublisher(topicName)
	if err != nil {
		return fmt.Errorf("get publisher for topic %q: %w", topicName, err)
	}

	payload, err := event.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal cloudevent: %w", err)
	}

	result := pub.Publish(ctx, &gcppubsub.Message{Data: payload})
	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("publish pubsub message to topic %q: %w", topicName, err)
	}

	return nil
}

func (s *GooglePubSubClientSender) getOrCreatePublisher(topicName string) (googlePubSubTopic, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("sender is closed")
	}
	pub, exists := s.publishers[topicName]
	s.mu.RUnlock()
	if exists {
		return pub, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	if s.closed {
		return nil, fmt.Errorf("sender is closed")
	}
	if pub, exists = s.publishers[topicName]; exists {
		return pub, nil
	}

	pub = s.client.Publisher(topicName)
	s.publishers[topicName] = pub
	return pub, nil
}

// Close stops all cached publishers.
func (s *GooglePubSubClientSender) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	for _, pub := range s.publishers {
		pub.Stop()
	}
	s.closed = true
	s.publishers = nil
	return nil
}
