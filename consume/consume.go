package consume

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// DecodeJSON decodes a CloudEvent JSON payload into the provided generic type.
func DecodeJSON[T any](event cloudevents.Event) (T, error) {
	var out T
	if len(event.Data()) == 0 {
		return out, fmt.Errorf("event data is empty")
	}

	if err := json.Unmarshal(event.Data(), &out); err != nil {
		return out, fmt.Errorf("decode event data: %w", err)
	}

	return out, nil
}
