# go-gcp-pubsub-client

A lightweight Go library for producing and consuming CloudEvents in pub/sub workflows.

## Packages

- `produce`: High-level publisher helpers that wrap payloads as CloudEvents.
- `consume`: consumer helpers for parsing CloudEvents from wrapped requests.
- `shared`: Reusable CloudEvent JSON helpers shared by produce and consume.

## Wrapped Event Consumer

The HTTP adapter expects Pub/Sub-style wrapped payloads where `message.data` is base64-encoded CloudEvent JSON.

```go
package main

import (
    "log"
    "net/http"

    oneweavepubsub "github.com/oneweave/go-gcp-pubsub-client/v2"
)

func main() {
    consumer := oneweavepubsub.NewEventConsumer()

    http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
        adapter := oneweavepubsub.NewHTTPEnvelopeAdapter(r)
        event, err := consumer.Consume(adapter)
        if err != nil {
            http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
            return
        }
        // Use the parsed event directly.
        _ = event
        w.WriteHeader(http.StatusNoContent)
    })

    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Pub/Sub HTTP Consumer (Cloud Run)

The consumer API now accepts transport adapters.

```go
package main

import (
    "log"
    "net/http"

    oneweavepubsub "github.com/oneweave/go-gcp-pubsub-client/v2"
)

func main() {
    consumer, err := oneweavepubsub.NewEnvelopeConsumer(oneweavepubsub.EnvelopeConsumerConfig{})
    if err != nil {
        log.Fatal(err)
    }

    http.HandleFunc("/pubsub/push", func(w http.ResponseWriter, r *http.Request) {
        adapter := oneweavepubsub.NewHTTPEnvelopeAdapter(r)
        event, err := consumer.Consume(adapter)
        if err != nil {
            http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
            return
        }

        // event is the embedded CloudEvent v1 from Pub/Sub message.data.
        // If message.data is not a valid CloudEvent JSON, parsing fails.
        _ = event
        w.WriteHeader(http.StatusOK)
    })

    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Decode Typed Payload From Embedded CloudEvent

```go
type OrderEvent struct {
    OrderID string `json:"orderId"`
}

func handlePush(w http.ResponseWriter, r *http.Request) {
    consumer, _ := oneweavepubsub.NewEnvelopeConsumer(oneweavepubsub.EnvelopeConsumerConfig{})
    adapter := oneweavepubsub.NewHTTPEnvelopeAdapter(r)

    var payload OrderEvent
    event, err := consumer.Consume(adapter)
    if err != nil {
        http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
        return
    }

    if err := event.DataAs(&payload); err != nil {
        http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
        return
    }

    // event is the embedded CloudEvent; payload is event.data decoded into OrderEvent.
    _ = event
    _ = payload
    w.WriteHeader(http.StatusOK)
}
```

## Quick Start

```go
package main

import (
    "context"
    "log"

    cloudevents "github.com/cloudevents/sdk-go/v2"
    "github.com/oneweave/go-gcp-pubsub-client/v2/produce"
)

type sender struct{}

func (s sender) Send(ctx context.Context, event cloudevents.Event) error {
    // Push event to your broker transport here.
    return nil
}

func main() {
    publisher, err := produce.NewPublisher(produce.Config{}, sender{})
    if err != nil {
        log.Fatal(err)
    }

    event := cloudevents.NewEvent(cloudevents.VersionV1)
    event.SetID("evt-1")
    event.SetType("artifact.created")
    event.SetSource("oneweave://artifact-builder")
    if err := event.SetData("application/json", map[string]any{
        "artifactID": "a-123",
        "status":     "ready",
    }); err != nil {
        log.Fatal(err)
    }

    _, err = publisher.Publish(context.Background(), event)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Development

```bash
go test ./...
```

## Migration: v1 to v2

v2 is a focused API cleanup. Consumer behavior remains the same, and producer APIs are now centered on publishing a pre-built CloudEvent.

### 1) Update module imports

Change imports from v1 to v2:

```go
// v1
import oneweavepubsub "github.com/oneweave/go-gcp-pubsub-client"

// v2
import oneweavepubsub "github.com/oneweave/go-gcp-pubsub-client/v2"
```

For direct package imports:

```go
// v1
import "github.com/oneweave/go-gcp-pubsub-client/produce"

// v2
import "github.com/oneweave/go-gcp-pubsub-client/v2/produce"
```

### 2) Build CloudEvent before publish

Changed in v2:

- `publisher.Publish` now accepts a `cloudevents.Event` instance.
- Publisher no longer injects `source`, `type`, or default extensions from `Config`.

```go
// v1
_, err := publisher.Publish(ctx, "artifact.ready", payload)

// v2
event := cloudevents.NewEvent(cloudevents.VersionV1)
event.SetID("evt-1")
event.SetType("artifact.ready")
event.SetSource("oneweave://artifact-builder")
_ = event.SetData("application/json", payload)
_, err := publisher.Publish(ctx, event)
```

### 3) Replace removed Publisher helper

Removed in v2:

- `(*produce.Publisher).PublishToTopic(...)`

Use `Publish` with `WithSubject` instead:

```go
// v1
event, err := publisher.PublishToTopic(ctx, "topic-A", "artifact.ready", payload)

// v2
event := cloudevents.NewEvent(cloudevents.VersionV1)
event.SetID("evt-1")
event.SetType("artifact.ready")
event.SetSource("oneweave://artifact-builder")
_ = event.SetData("application/json", payload)
event, err := publisher.Publish(ctx, event, produce.WithSubject("topic-A"))
```

### 4) Replace removed Pub/Sub constructor helper

Removed in v2:

- `produce.NewGooglePubSubSenderFromClient(client, topicID)`

If you directly use the Google Pub/Sub client in your code, also migrate imports from `cloud.google.com/go/pubsub` to `cloud.google.com/go/pubsub/v2`.

Use explicit publisher lookup plus `NewGooglePubSubSender`:

```go
// v1
sender, err := produce.NewGooglePubSubSenderFromClient(client, topicID)

// v2
sender, err := produce.NewGooglePubSubSender(client.Publisher(topicID))
```

### 5) Replace removed root publisher wrapper

Removed in v2:

- `oneweavepubsub.NewPublisher(...)`

Use the `produce` package directly:

```go
// v1
publisher, err := oneweavepubsub.NewPublisher(cfg, sender)

// v2
publisher, err := produce.NewPublisher(cfg, sender)
```

### 6) Consumer APIs now use adapters

Consumer methods now accept transport adapters instead of transport-native request objects.

Use these root APIs in v2:

- `NewEventConsumer`
- `NewEnvelopeConsumer`
- `NewHTTPEnvelopeAdapter`
