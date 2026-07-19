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
	stopped bool
}

func (f *fakeGooglePubSubTopic) Publish(_ context.Context, msg *gcppubsub.Message) googlePubSubPublishResult {
	f.message = msg
	return fakeGooglePubSubPublishResult{err: f.err}
}

func (f *fakeGooglePubSubTopic) Stop() {
	f.stopped = true
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

type fakeGooglePubSubClient struct {
	topics map[string]*fakeGooglePubSubTopic
}

func (c *fakeGooglePubSubClient) Publisher(topicID string) googlePubSubTopic {
	if c.topics == nil {
		c.topics = make(map[string]*fakeGooglePubSubTopic)
	}
	if _, exists := c.topics[topicID]; !exists {
		c.topics[topicID] = &fakeGooglePubSubTopic{}
	}
	return c.topics[topicID]
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

func TestNewGooglePubSubClientSenderValidation(t *testing.T) {
	sender, err := NewGooglePubSubClientSender(nil)
	require.Error(t, err)
	assert.Nil(t, sender)
	assert.Contains(t, err.Error(), "pubsub client is required")
}

func TestGooglePubSubClientSenderSend(t *testing.T) {
	event := cloudevents.NewEvent()
	event.SetID("evt-1")
	event.SetSource("oneweave://producer")
	event.SetType("artifact.ready")
	require.NoError(t, event.SetData("application/json", map[string]any{"id": "a-1"}))

	t.Run("sender requires client", func(t *testing.T) {
		sender := &GooglePubSubClientSender{}
		err := sender.Send(context.Background(), event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pubsub client is required")
	})

	t.Run("publishes event json using default topic resolver", func(t *testing.T) {
		client := &fakeGooglePubSubClient{}
		sender := &GooglePubSubClientSender{
			client:     client,
			resolver:   DefaultTopicResolver,
			publishers: make(map[string]googlePubSubTopic),
		}

		err := sender.Send(context.Background(), event)
		require.NoError(t, err)

		topic, exists := client.topics["artifact.ready"]
		require.True(t, exists)
		require.NotNil(t, topic.message)

		encoded, err := event.MarshalJSON()
		require.NoError(t, err)
		assert.JSONEq(t, string(encoded), string(topic.message.Data))
	})

	t.Run("publishes event json using custom topic resolver", func(t *testing.T) {
		client := &fakeGooglePubSubClient{}
		resolver := func(e cloudevents.Event) string {
			return "custom-topic-" + e.Type()
		}
		sender := &GooglePubSubClientSender{
			client:     client,
			resolver:   resolver,
			publishers: make(map[string]googlePubSubTopic),
		}

		err := sender.Send(context.Background(), event)
		require.NoError(t, err)

		topic, exists := client.topics["custom-topic-artifact.ready"]
		require.True(t, exists)
		require.NotNil(t, topic.message)
	})

	t.Run("caches publisher instances", func(t *testing.T) {
		client := &fakeGooglePubSubClient{}
		sender := &GooglePubSubClientSender{
			client:     client,
			resolver:   DefaultTopicResolver,
			publishers: make(map[string]googlePubSubTopic),
		}

		err := sender.Send(context.Background(), event)
		require.NoError(t, err)
		assert.Len(t, sender.publishers, 1)

		pub1 := sender.publishers["artifact.ready"]

		err = sender.Send(context.Background(), event)
		require.NoError(t, err)
		assert.Len(t, sender.publishers, 1)

		pub2 := sender.publishers["artifact.ready"]
		assert.Same(t, pub1, pub2)
	})

	t.Run("fails when resolved topic is empty", func(t *testing.T) {
		client := &fakeGooglePubSubClient{}
		resolver := func(e cloudevents.Event) string {
			return ""
		}
		sender := &GooglePubSubClientSender{
			client:     client,
			resolver:   resolver,
			publishers: make(map[string]googlePubSubTopic),
		}

		err := sender.Send(context.Background(), event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resolved topic name is empty")
	})

	t.Run("fails if publish fails", func(t *testing.T) {
		client := &fakeGooglePubSubClient{}
		sender := &GooglePubSubClientSender{
			client:     client,
			resolver:   DefaultTopicResolver,
			publishers: make(map[string]googlePubSubTopic),
		}

		// Inject mock publish error
		topic := client.Publisher("artifact.ready").(*fakeGooglePubSubTopic)
		topic.err = errors.New("gcp pubsub err")

		err := sender.Send(context.Background(), event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "publish pubsub message to topic")
		assert.Contains(t, err.Error(), "gcp pubsub err")
	})

	t.Run("fails after closed", func(t *testing.T) {
		client := &fakeGooglePubSubClient{}
		sender := &GooglePubSubClientSender{
			client:     client,
			resolver:   DefaultTopicResolver,
			publishers: make(map[string]googlePubSubTopic),
		}

		err := sender.Close()
		require.NoError(t, err)

		err = sender.Send(context.Background(), event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sender is closed")
	})

	t.Run("closing stops all publishers", func(t *testing.T) {
		client := &fakeGooglePubSubClient{}
		sender := &GooglePubSubClientSender{
			client:     client,
			resolver:   DefaultTopicResolver,
			publishers: make(map[string]googlePubSubTopic),
		}

		err := sender.Send(context.Background(), event)
		require.NoError(t, err)

		topic := client.topics["artifact.ready"]
		require.False(t, topic.stopped)

		err = sender.Close()
		require.NoError(t, err)
		assert.True(t, topic.stopped)

		// Calling close again is a no-op
		err = sender.Close()
		require.NoError(t, err)
	})
}
