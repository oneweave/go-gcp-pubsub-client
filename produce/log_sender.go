package produce

import (
	"context"
	"fmt"
	"log"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// LogSender logs CloudEvents to standard logger output.
type LogSender struct {
	Prefix string
}

// Compile-time assertion that LogSender implements Sender.
var _ Sender = (*LogSender)(nil)

// NewLogSender constructs a sender that logs event JSON.
func NewLogSender(prefix string) Sender {
	return &LogSender{Prefix: prefix}
}

// Send logs the CloudEvent as JSON.
func (s *LogSender) Send(_ context.Context, event cloudevents.Event) error {
	if s == nil {
		return fmt.Errorf("log sender is required")
	}

	payload, err := event.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal cloudevent: %w", err)
	}

	if s.Prefix != "" {
		log.Printf("%s %s", s.Prefix, string(payload))
		return nil
	}

	log.Printf("%s", string(payload))
	return nil
}
