package produce

import (
	"context"
	"errors"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSender struct {
	event cloudevents.Event
	err   error
}

func (s *stubSender) Send(_ context.Context, event cloudevents.Event) error {
	s.event = event
	return s.err
}

func TestPublishSendsCloudEventWithDefaultsAndOptions(t *testing.T) {
	sender := &stubSender{}
	publisher, err := NewPublisher(Config{}, sender)
	require.NoError(t, err)

	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetID("evt-1")
	event.SetType("artifact.ready")
	event.SetSource("oneweave://producer")
	require.NoError(t, event.SetData("application/json", map[string]any{"id": "a-1"}))

	got, err := publisher.Publish(context.Background(), event, WithSubject("artifacts"), WithExtension("region", "us-east-1"))
	require.NoError(t, err)
	assert.Equal(t, "evt-1", got.ID())
	assert.Equal(t, "artifact.ready", got.Type())
	assert.Equal(t, "oneweave://producer", got.Source())
	assert.Equal(t, "artifacts", got.Subject())
	assert.Equal(t, "application/json", got.DataContentType())
	assert.Equal(t, "us-east-1", got.Extensions()["region"])
	assert.Equal(t, "evt-1", sender.event.ID())
}

func TestPublishWithSubjectSetsSubject(t *testing.T) {
	sender := &stubSender{}
	publisher, err := NewPublisher(Config{}, sender)
	require.NoError(t, err)
	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetID("evt-1")
	event.SetType("evt")
	event.SetSource("oneweave://producer")

	event, err = publisher.Publish(context.Background(), event, WithSubject("topic-A"))
	require.NoError(t, err)
	assert.Equal(t, "topic-A", event.Subject())
}

func TestNewPublisherValidation(t *testing.T) {
	t.Run("sender required", func(t *testing.T) {
		publisher, err := NewPublisher(Config{}, nil)
		require.Error(t, err)
		assert.Nil(t, publisher)
		assert.Contains(t, err.Error(), "sender is required")
	})

	t.Run("config can be empty", func(t *testing.T) {
		sender := &stubSender{}
		publisher, err := NewPublisher(Config{}, sender)
		require.NoError(t, err)
		assert.NotNil(t, publisher)
	})
}

func TestPublishValidationAndErrors(t *testing.T) {
	sender := &stubSender{}
	publisher, err := NewPublisher(Config{}, sender)
	require.NoError(t, err)

	t.Run("event source required", func(t *testing.T) {
		event := cloudevents.NewEvent(cloudevents.VersionV1)
		event.SetID("evt-1")
		event.SetType("evt")

		event, err := publisher.Publish(context.Background(), event)
		require.Error(t, err)
		assert.Equal(t, cloudevents.Event{}, event)
		assert.Contains(t, err.Error(), "invalid cloudevent")
	})

	t.Run("event type required", func(t *testing.T) {
		event := cloudevents.NewEvent(cloudevents.VersionV1)
		event.SetID("evt-1")
		event.SetSource("oneweave://producer")

		event, err := publisher.Publish(context.Background(), event)
		require.Error(t, err)
		assert.Equal(t, cloudevents.Event{}, event)
		assert.Contains(t, err.Error(), "invalid cloudevent")
	})

	t.Run("event id required", func(t *testing.T) {
		event := cloudevents.NewEvent(cloudevents.VersionV1)
		event.SetType("evt")
		event.SetSource("oneweave://producer")

		event, err := publisher.Publish(context.Background(), event)
		require.Error(t, err)
		assert.Equal(t, cloudevents.Event{}, event)
		assert.Contains(t, err.Error(), "invalid cloudevent")
	})

	t.Run("sender error wrapped", func(t *testing.T) {
		sender.err = errors.New("transport down")
		event := cloudevents.NewEvent(cloudevents.VersionV1)
		event.SetID("evt-1")
		event.SetType("evt")
		event.SetSource("oneweave://producer")
		require.NoError(t, event.SetData("application/json", map[string]string{"k": "v"}))

		event, err := publisher.Publish(context.Background(), event)
		require.Error(t, err)
		assert.Equal(t, cloudevents.Event{}, event)
		assert.Contains(t, err.Error(), "send cloudevent")
		assert.Contains(t, err.Error(), "transport down")
		sender.err = nil
	})
}

func TestPublishWithContentTypeOverride(t *testing.T) {
	sender := &stubSender{}
	publisher, err := NewPublisher(Config{}, sender)
	require.NoError(t, err)
	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetID("evt-1")
	event.SetType("evt")
	event.SetSource("oneweave://producer")
	require.NoError(t, event.SetData("application/json", map[string]string{"ok": "true"}))

	event, err = publisher.Publish(
		context.Background(),
		event,
		WithDataContentType("application/cloudevents+json"),
	)
	require.NoError(t, err)
	assert.Equal(t, "application/cloudevents+json", event.DataContentType())
}
