package produce

import (
	"context"
	"errors"
	"testing"

	gcppubsub "cloud.google.com/go/pubsub/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeGooglePubSubTopic struct {
	message *gcppubsub.Message
	err     error
}

func (f *fakeGooglePubSubTopic) Publish(_ context.Context, msg *gcppubsub.Message) googlePubSubPublishResult {
	f.message = msg
	return fakeGooglePubSubPublishResult{err: f.err}
}

type fakeGooglePubSubPublishResult struct {
	err error
}

func (f fakeGooglePubSubPublishResult) Get(context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "msg-1", nil
}

func TestNewGooglePubSubSenderValidation(t *testing.T) {
	sender, err := NewGooglePubSubSender(nil)
	require.Error(t, err)
	assert.Nil(t, sender)
	assert.Contains(t, err.Error(), "pubsub publisher is required")
}

func TestGooglePubSubSenderSend(t *testing.T) {
	event := cloudevents.NewEvent()
	event.SetID("evt-1")
	event.SetSource("oneweave://producer")
	event.SetType("artifact.ready")
	require.NoError(t, event.SetData("application/json", map[string]any{"id": "a-1"}))

	t.Run("sender requires topic", func(t *testing.T) {
		sender := &GooglePubSubSender{}
		err := sender.Send(context.Background(), event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pubsub publisher is required")
	})

	t.Run("publishes event json to message data", func(t *testing.T) {
		topic := &fakeGooglePubSubTopic{}
		sender := &GooglePubSubSender{topic: topic}
		encoded, err := event.MarshalJSON()
		require.NoError(t, err)

		err = sender.Send(context.Background(), event)
		require.NoError(t, err)
		require.NotNil(t, topic.message)
		assert.JSONEq(t, string(encoded), string(topic.message.Data))
		assert.Contains(t, string(topic.message.Data), "evt-1")
		assert.Contains(t, string(topic.message.Data), "artifact.ready")
	})

	t.Run("wraps publish errors", func(t *testing.T) {
		topic := &fakeGooglePubSubTopic{err: errors.New("topic unavailable")}
		sender := &GooglePubSubSender{topic: topic}

		err := sender.Send(context.Background(), event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "publish pubsub message")
		assert.Contains(t, err.Error(), "topic unavailable")
	})
}
