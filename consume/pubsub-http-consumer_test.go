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

type pubSubPayload struct {
	OrderID string `json:"orderId"`
}

func TestNewPubSubHTTPConsumer(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		consumer, err := NewPubSubHTTPConsumer(PubSubHTTPConsumerConfig{})
		require.NoError(t, err)
		require.NotNil(t, consumer)
	})
}

func TestPubSubHTTPConsumerConsumeHTTPRequest(t *testing.T) {
	t.Run("request required", func(t *testing.T) {
		consumer, err := NewPubSubHTTPConsumer(PubSubHTTPConsumerConfig{})
		require.NoError(t, err)

		event, consumeErr := consumer.ConsumeHTTPRequest(nil)
		require.Error(t, consumeErr)
		assert.Equal(t, "request is required", consumeErr.Error())
		assert.Empty(t, event)
	})

	t.Run("nil consumer", func(t *testing.T) {
		var consumer *PubSubHTTPConsumer
		req, err := http.NewRequest(http.MethodPost, "http://example.com/push", strings.NewReader("{}"))
		require.NoError(t, err)

		event, consumeErr := consumer.ConsumeHTTPRequest(req)
		require.Error(t, consumeErr)
		assert.Equal(t, "consumer is required", consumeErr.Error())
		assert.Empty(t, event)
	})

	t.Run("method not allowed", func(t *testing.T) {
		consumer, err := NewPubSubHTTPConsumer(PubSubHTTPConsumerConfig{})
		require.NoError(t, err)

		req, reqErr := http.NewRequest(http.MethodGet, "http://example.com/push", nil)
		require.NoError(t, reqErr)
		event, consumeErr := consumer.ConsumeHTTPRequest(req)
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "method not allowed")
		assert.Empty(t, event)
	})

	t.Run("decode envelope error", func(t *testing.T) {
		consumer, err := NewPubSubHTTPConsumer(PubSubHTTPConsumerConfig{})
		require.NoError(t, err)

		req := pubSubRequest(t, http.MethodPost, "{")
		event, consumeErr := consumer.ConsumeHTTPRequest(req)
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "decode pubsub push envelope")
		assert.Empty(t, event)
	})

	t.Run("message id required", func(t *testing.T) {
		consumer, err := NewPubSubHTTPConsumer(PubSubHTTPConsumerConfig{})
		require.NoError(t, err)

		req := pubSubRequest(t, http.MethodPost, fmt.Sprintf(`{"message":{"data":%q}}`, base64.StdEncoding.EncodeToString([]byte(`{"ok":true}`))))
		event, consumeErr := consumer.ConsumeHTTPRequest(req)
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "pubsub message id is required")
		assert.Empty(t, event)
	})

	t.Run("message data required", func(t *testing.T) {
		consumer, err := NewPubSubHTTPConsumer(PubSubHTTPConsumerConfig{})
		require.NoError(t, err)

		req := pubSubRequest(t, http.MethodPost, `{"message":{"messageId":"m-1"}}`)
		event, consumeErr := consumer.ConsumeHTTPRequest(req)
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "pubsub message data is empty")
		assert.Empty(t, event)
	})

	t.Run("invalid base64", func(t *testing.T) {
		consumer, err := NewPubSubHTTPConsumer(PubSubHTTPConsumerConfig{})
		require.NoError(t, err)

		req := pubSubRequest(t, http.MethodPost, `{"message":{"messageId":"m-1","data":"@@@"}}`)
		event, consumeErr := consumer.ConsumeHTTPRequest(req)
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "decode pubsub message data")
		assert.Empty(t, event)
	})

	t.Run("non cloudevent json rejected", func(t *testing.T) {
		consumer, err := NewPubSubHTTPConsumer(PubSubHTTPConsumerConfig{})
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

		event, consumeErr := consumer.ConsumeHTTPRequest(req)
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "pubsub message data is not a cloudevent")
		assert.Empty(t, event)
	})

	t.Run("success with embedded cloudevent payload", func(t *testing.T) {
		consumer, err := NewPubSubHTTPConsumer(PubSubHTTPConsumerConfig{})
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

		event, consumeErr := consumer.ConsumeHTTPRequest(req)
		require.NoError(t, consumeErr)
		assert.Equal(t, "evt-inner-1", event.ID())
		assert.Equal(t, "order.created", event.Type())
		assert.Equal(t, "oneweave://orders", event.Source())
		assert.EqualValues(t, 3, event.Extensions()["deliveryattempt"])
	})
}

func TestPubSubHTTPConsumerConsumeHTTPRequestDataAs(t *testing.T) {
	t.Run("out required", func(t *testing.T) {
		consumer, err := NewPubSubHTTPConsumer(PubSubHTTPConsumerConfig{})
		require.NoError(t, err)

		event, consumeErr := consumer.ConsumeHTTPRequestDataAs(nil, nil)
		require.Error(t, consumeErr)
		assert.Equal(t, "out is required", consumeErr.Error())
		assert.Empty(t, event)
	})

	t.Run("non cloudevent payload decode fails", func(t *testing.T) {
		consumer, err := NewPubSubHTTPConsumer(PubSubHTTPConsumerConfig{})
		require.NoError(t, err)

		req := pubSubRequest(t, http.MethodPost, fmt.Sprintf(`{
			"message":{
				"messageId":"m-9",
				"data":%q
			}
		}`,
			base64.StdEncoding.EncodeToString([]byte(`{"orderId":"o-99"}`)),
		))

		var payload pubSubPayload
		event, consumeErr := consumer.ConsumeHTTPRequestDataAs(req, &payload)
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "pubsub message data is not a cloudevent")
		assert.Empty(t, event)
		assert.Equal(t, pubSubPayload{}, payload)
	})

	t.Run("decode success with embedded cloudevent payload", func(t *testing.T) {
		consumer, err := NewPubSubHTTPConsumer(PubSubHTTPConsumerConfig{})
		require.NoError(t, err)

		embedded := `{
			"specversion":"1.0",
			"id":"evt-inner-2",
			"source":"oneweave://orders",
			"type":"order.updated",
			"datacontenttype":"application/json",
			"data":{"orderId":"o-101"}
		}`
		req := pubSubRequest(t, http.MethodPost, fmt.Sprintf(`{
			"message":{
				"messageId":"outer-message-id",
				"data":%q
			}
		}`,
			base64.StdEncoding.EncodeToString([]byte(embedded)),
		))

		var payload pubSubPayload
		event, consumeErr := consumer.ConsumeHTTPRequestDataAs(req, &payload)
		require.NoError(t, consumeErr)
		assert.Equal(t, "evt-inner-2", event.ID())
		assert.Equal(t, "o-101", payload.OrderID)
	})

	t.Run("decode failure", func(t *testing.T) {
		consumer, err := NewPubSubHTTPConsumer(PubSubHTTPConsumerConfig{})
		require.NoError(t, err)

		embedded := `{
			"specversion":"1.0",
			"id":"evt-inner-3",
			"source":"oneweave://orders",
			"type":"order.updated",
			"datacontenttype":"application/json",
			"data":{"orderId":"o-100"}
		}`

		req := pubSubRequest(t, http.MethodPost, fmt.Sprintf(`{
			"message":{
				"messageId":"outer-message-id",
				"data":%q
			}
		}`,
			base64.StdEncoding.EncodeToString([]byte(embedded)),
		))

		var payload string
		event, consumeErr := consumer.ConsumeHTTPRequestDataAs(req, &payload)
		require.Error(t, consumeErr)
		assert.Contains(t, consumeErr.Error(), "decode cloudevent data")
		assert.Empty(t, event)
	})
}

func pubSubRequest(t *testing.T, method, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, "http://example.com/push", strings.NewReader(body))
	require.NoError(t, err)
	return req
}
