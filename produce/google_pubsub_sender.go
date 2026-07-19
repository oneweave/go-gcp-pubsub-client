package produce

import (
	"context"
	"fmt"

	gcppubsub "cloud.google.com/go/pubsub/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type googlePubSubTopic interface {
	Publish(context.Context, *gcppubsub.Message) googlePubSubPublishResult
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
