package produce

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// Sender abstracts a message transport that can send a CloudEvent.
type Sender interface {
	Send(ctx context.Context, event cloudevents.Event) error
}
