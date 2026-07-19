package produce

import (
	"bytes"
	"context"
	"log"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogSender(t *testing.T) {
	sender := NewLogSender("events")
	require.NotNil(t, sender)
}

func TestLogSenderSend(t *testing.T) {
	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetID("evt-1")
	event.SetSource("oneweave://producer")
	event.SetType("artifact.ready")
	require.NoError(t, event.SetData("application/json", map[string]any{"id": "a-1"}))

	t.Run("nil receiver", func(t *testing.T) {
		var sender *LogSender
		err := sender.Send(context.Background(), event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "log sender is required")
	})

	t.Run("logs event json", func(t *testing.T) {
		buf := &bytes.Buffer{}
		originalWriter := log.Writer()
		originalFlags := log.Flags()
		log.SetOutput(buf)
		log.SetFlags(0)
		t.Cleanup(func() {
			log.SetOutput(originalWriter)
			log.SetFlags(originalFlags)
		})

		sender := &LogSender{Prefix: "events"}
		err := sender.Send(context.Background(), event)
		require.NoError(t, err)

		logged := buf.String()
		assert.Contains(t, logged, "events")
		assert.Contains(t, logged, "evt-1")
		assert.Contains(t, logged, "artifact.ready")
	})
}
