# go-gcp-pubsub-cloudevent

A lightweight Go library for producing and consuming CloudEvents in pub/sub workflows.

## Packages

- `produce`: High-level publisher helpers that wrap payloads as CloudEvents.
- `consume`: High-level consumer helpers for decoding CloudEvent data.

## Quick Start

```go
package main

import (
    "context"
    "log"

    cloudevents "github.com/cloudevents/sdk-go/v2"
    "github.com/oneweave/oneweave-pubsub/produce"
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
