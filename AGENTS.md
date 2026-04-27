# Oneweave PubSub Agent Guide

Treat [README.md](README.md) as the primary usage and package overview.

## Scope

- This repository is a Go library, not an application service.
- Keep APIs small and transport-agnostic.
- Preserve backwards compatibility for exported symbols in `produce` and `consume`.

## Package Boundaries

- `produce/`: create and publish CloudEvents from arbitrary payloads.
- `consume/`: parse CloudEvents from HTTP requests, including Pub/Sub push wrappers.
- `pubsub.go`: thin convenience re-exports only.

## Consume Conventions

- `consume.HTTPConsumer` parses direct CloudEvent HTTP requests.
- `consume.PubSubHTTPConsumer` expects Pub/Sub `message.data` to contain an embedded CloudEvent JSON.
- Treat non-CloudEvent Pub/Sub payload JSON as invalid input for `PubSubHTTPConsumer`.
- Use `ConsumeHTTPRequestDataAs` when callers need typed decoding of embedded CloudEvent `data`.

## Root API Notes

- Keep root package re-exports in `pubsub.go` aligned with subpackage APIs (`produce` and `consume`).
- Root constructors should remain thin wrappers over subpackage constructors.
- Preserve the module path `github.com/oneweave/go-gcp-pubsub-client` in examples and imports.

## Publish Conventions

- Wrap publish payloads as CloudEvents v1.0.
- Default `datacontenttype` is `application/json` unless explicitly overridden.
- Use CloudEvent `subject` for topic/routing helpers (for example via `PublishToTopic`).
- Keep sender integration behind the `produce.Sender` interface.

## Go Conventions

- Propagate caller context; avoid introducing `context.Background()` in library paths.
- Handle and return errors explicitly.
- Wrap propagated errors with `fmt.Errorf(... %w ...)`.

## Validation

Run before handoff:

```bash
./test.sh   
```

```
./lint.sh
```