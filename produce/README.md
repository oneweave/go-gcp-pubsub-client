# produce package: adding new senders

This package sends CloudEvents and delegates transport to a `Sender`.

## Design overview

- `Publisher` is responsible for CloudEvent validation, option overlays, and dispatch.
- `Sender` is responsible for transport-specific delivery.
- New backends should be implemented as new `Sender` implementations.

The transport contract is intentionally small:

```go
type Sender interface {
    Send(ctx context.Context, event cloudevents.Event) error
}
```

## When to create a new sender

Create a new sender when you need to publish CloudEvents to a new broker or platform, for example:

- AWS EventBridge
- AWS SNS
- AWS SQS
- Kafka
- NATS

## Implementation checklist

1. Create a new file named `<transport>_sender.go`.
2. Define a sender struct that holds transport clients/handles.
3. Add a constructor that validates required dependencies.
4. Implement `Send(ctx, event)`.
5. Marshal the CloudEvent using `event.MarshalJSON()` when your transport expects bytes/JSON.
6. Publish through the transport client.
7. Wrap errors with context using `fmt.Errorf(... %w ...)`.
8. Add compile-time interface assertion:

```go
var _ Sender = (*YourSender)(nil)
```

## Example skeleton

```go
package produce

import (
    "context"
    "fmt"

    cloudevents "github.com/cloudevents/sdk-go/v2"
)

type YourSender struct {
    client yourTransportClient
}

var _ Sender = (*YourSender)(nil)

func NewYourSender(client yourTransportClient) (Sender, error) {
    if client == nil {
        return nil, fmt.Errorf("transport client is required")
    }
    return &YourSender{client: client}, nil
}

func (s *YourSender) Send(ctx context.Context, event cloudevents.Event) error {
    if s == nil || s.client == nil {
        return fmt.Errorf("transport client is required")
    }

    payload, err := event.MarshalJSON()
    if err != nil {
        return fmt.Errorf("marshal cloudevent: %w", err)
    }

    if err := s.client.Publish(ctx, payload); err != nil {
        return fmt.Errorf("publish message: %w", err)
    }

    return nil
}
```

## Testing checklist

Create `<transport>_sender_test.go` and cover at least:

1. Constructor validation errors.
2. Nil receiver / missing dependency errors in `Send`.
3. Successful publish path (assert expected payload content).
4. Transport errors are wrapped with helpful context.

Use small internal interfaces around transport primitives so tests can use fakes without real cloud dependencies.

## Usage with Publisher

All senders plug into the same publisher API:

```go
publisher, err := NewPublisher(Config{
    // currently unused
}, sender)
if err != nil {
    return err
}

event := cloudevents.NewEvent(cloudevents.VersionV1)
event.SetID("evt-1")
event.SetType("artifact.ready")
event.SetSource("oneweave://producer")
if err := event.SetData("application/json", map[string]any{"id": "a-1"}); err != nil {
    return err
}

_, err = publisher.Publish(ctx, event)
if err != nil {
    return err
}
```

## Current reference implementation

See `google_pubsub_sender.go` and `google_pubsub_sender_test.go` for a complete example sender implementation and test shape.
