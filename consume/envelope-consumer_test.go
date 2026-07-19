package consume

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnvelopeConsumer(t *testing.T) {
	consumer, err := NewEnvelopeConsumer(EnvelopeConsumerConfig{})
	require.NoError(t, err)
	require.NotNil(t, consumer)

	var eventConsumer Consumer = consumer
	require.NotNil(t, eventConsumer)
}

func TestEnvelopeConsumerConsume(t *testing.T) {
	t.Run("request required", func(t *testing.T) {
		consumer, err := NewEnvelopeConsumer(EnvelopeConsumerConfig{})
		require.NoError(t, err)

		event, consumeErr := consumer.Consume(NewHTTPEnvelopeAdapter(nil))
		require.Error(t, consumeErr)
		assert.Equal(t, "request is required", consumeErr.Error())
		assert.Empty(t, event)
	})

	t.Run("nil consumer", func(t *testing.T) {
		var consumer *EnvelopeConsumer
		req, err := http.NewRequest(http.MethodPost, "http://example.com/push", strings.NewReader("{}"))
		require.NoError(t, err)

		event, consumeErr := consumer.Consume(NewHTTPEnvelopeAdapter(req))
		require.Error(t, consumeErr)
		assert.Equal(t, "consumer is required", consumeErr.Error())
		assert.Empty(t, event)
	})

	t.Run("adapter required", func(t *testing.T) {
		consumer, err := NewEnvelopeConsumer(EnvelopeConsumerConfig{})
		require.NoError(t, err)

		event, consumeErr := consumer.Consume(nil)
		require.Error(t, consumeErr)
		assert.Equal(t, "adapter is required", consumeErr.Error())
		assert.Empty(t, event)
	})

	t.Run("method not allowed", func(t *testing.T) {
		consumer, err := NewEnvelopeConsumer(EnvelopeConsumerConfig{})
		require.NoError(t, err)

		req, reqErr := http.NewRequest(http.MethodGet, "http://example.com/push", nil)
		require.NoError(t, reqErr)
		event, consumeErr := consumer.Consume(NewHTTPEnvelopeAdapter(req))
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "method not allowed")
		assert.Empty(t, event)
	})

	t.Run("decode envelope error", func(t *testing.T) {
		consumer, err := NewEnvelopeConsumer(EnvelopeConsumerConfig{})
		require.NoError(t, err)

		req := pubSubRequest(t, http.MethodPost, "{")
		event, consumeErr := consumer.Consume(NewHTTPEnvelopeAdapter(req))
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "decode pubsub push envelope")
		assert.Empty(t, event)
	})

	t.Run("message id required", func(t *testing.T) {
		consumer, err := NewEnvelopeConsumer(EnvelopeConsumerConfig{})
		require.NoError(t, err)

		req := pubSubRequest(t, http.MethodPost, fmt.Sprintf(`{"message":{"data":%q}}`, base64.StdEncoding.EncodeToString([]byte(`{"ok":true}`))))
		event, consumeErr := consumer.Consume(NewHTTPEnvelopeAdapter(req))
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "pubsub message id is required")
		assert.Empty(t, event)
	})

	t.Run("message data required", func(t *testing.T) {
		consumer, err := NewEnvelopeConsumer(EnvelopeConsumerConfig{})
		require.NoError(t, err)

		req := pubSubRequest(t, http.MethodPost, `{"message":{"messageId":"m-1"}}`)
		event, consumeErr := consumer.Consume(NewHTTPEnvelopeAdapter(req))
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "pubsub message data is empty")
		assert.Empty(t, event)
	})

	t.Run("invalid base64", func(t *testing.T) {
		consumer, err := NewEnvelopeConsumer(EnvelopeConsumerConfig{})
		require.NoError(t, err)

		req := pubSubRequest(t, http.MethodPost, `{"message":{"messageId":"m-1","data":"@@@"}}`)
		event, consumeErr := consumer.Consume(NewHTTPEnvelopeAdapter(req))
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "decode pubsub message data")
		assert.Empty(t, event)
	})

	t.Run("invalid decoded json rejected", func(t *testing.T) {
		consumer, err := NewEnvelopeConsumer(EnvelopeConsumerConfig{})
		require.NoError(t, err)

		req := pubSubRequest(t, http.MethodPost, fmt.Sprintf(`{
			"message":{
				"messageId":"m-1",
				"data":%q
			}
		}`,
			base64.StdEncoding.EncodeToString([]byte(`{"specversion":`)),
		))
		event, consumeErr := consumer.Consume(NewHTTPEnvelopeAdapter(req))
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "pubsub message data is not valid JSON")
		assert.Empty(t, event)
	})

	t.Run("non cloudevent json rejected", func(t *testing.T) {
		consumer, err := NewEnvelopeConsumer(EnvelopeConsumerConfig{})
		require.NoError(t, err)

		req := pubSubRequest(t, http.MethodPost, fmt.Sprintf(`{
			"message":{
				"messageId":"m-2",
				"data":%q,
				"publishTime":"2026-02-17T10:11:12.999999999Z"
			}
		}`,
			base64.StdEncoding.EncodeToString([]byte(`{"orderId":"o-1"}`)),
		))

		event, consumeErr := consumer.Consume(NewHTTPEnvelopeAdapter(req))
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "pubsub message data is not a cloudevent")
		assert.Empty(t, event)
	})

	t.Run("success with embedded cloudevent payload", func(t *testing.T) {
		consumer, err := NewEnvelopeConsumer(EnvelopeConsumerConfig{})
		require.NoError(t, err)

		embedded := `{
			"specversion":"1.0",
			"id":"evt-inner-1",
			"source":"oneweave://orders",
			"type":"order.created",
			"datacontenttype":"application/json",
			"data":{"orderId":"o-inner"}
		}`
		req := pubSubRequest(t, http.MethodPost, fmt.Sprintf(`{
			"deliveryAttempt":3,
			"message":{
				"messageId":"outer-message-id",
				"data":%q
			}
		}`,
			base64.StdEncoding.EncodeToString([]byte(embedded)),
		))

		event, consumeErr := consumer.Consume(NewHTTPEnvelopeAdapter(req))
		require.NoError(t, consumeErr)
		assert.Equal(t, "evt-inner-1", event.ID())
		assert.Equal(t, "order.created", event.Type())
		assert.Equal(t, "oneweave://orders", event.Source())
		assert.EqualValues(t, 3, event.Extensions()["deliveryattempt"])
	})
}

func pubSubRequest(t *testing.T, method, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, "http://example.com/push", strings.NewReader(body))
	require.NoError(t, err)
	return req
}
