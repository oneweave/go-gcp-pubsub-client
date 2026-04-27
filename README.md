# go-gcp-pubsub-client

A lightweight Go library for producing and consuming CloudEvents in pub/sub workflows.

## Packages

- `produce`: High-level publisher helpers that wrap payloads as CloudEvents.
- `consume`: HTTP consumer helpers for parsing CloudEvents from requests.
- `shared`: Reusable CloudEvent JSON helpers shared by produce and consume.

## HTTP Consumer

```go
package main

import (
    "log"
    "net/http"

    oneweavepubsub "github.com/oneweave/go-gcp-pubsub-client"
)

func main() {
    httpConsumer := oneweavepubsub.NewHTTPConsumer()

    http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
        event, err := httpConsumer.ConsumeHTTPRequest(r)
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

```go
package main

import (
    "log"
    "net/http"

    oneweavepubsub "github.com/oneweave/go-gcp-pubsub-client"
)

func main() {
    consumer, err := oneweavepubsub.NewPubSubHTTPConsumer(oneweavepubsub.PubSubHTTPConsumerConfig{})
    if err != nil {
        log.Fatal(err)
    }

    http.HandleFunc("/pubsub/push", func(w http.ResponseWriter, r *http.Request) {
        event, err := consumer.ConsumeHTTPRequest(r)
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
    consumer, _ := oneweavepubsub.NewPubSubHTTPConsumer(oneweavepubsub.PubSubHTTPConsumerConfig{})

    var payload OrderEvent
    event, err := consumer.ConsumeHTTPRequestDataAs(r, &payload)
    if err != nil {
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
    "github.com/oneweave/go-gcp-pubsub-client/produce"
)

type sender struct{}

func (s sender) Send(ctx context.Context, event cloudevents.Event) error {
    // Push event to your broker transport here.
    return nil
}

func main() {
    publisher, err := produce.NewPublisher(produce.Config{
        Source:           "oneweave://artifact-builder",
        DefaultEventType: "artifact.created",
    }, sender{})
    if err != nil {
        log.Fatal(err)
    }

    _, err = publisher.Publish(context.Background(), "", map[string]any{
        "artifactID": "a-123",
        "status":     "ready",
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## Development

```bash
go test ./...
```
