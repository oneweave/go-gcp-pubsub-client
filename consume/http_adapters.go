package consume

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/oneweave/go-gcp-pubsub-client/v2/shared"
)

// HTTPEnvelopeAdapter adapts HTTP requests into EnvelopeMessage.
type HTTPEnvelopeAdapter struct {
	Request *http.Request
}

// NewHTTPEnvelopeAdapter constructs an EnvelopeAdapter for HTTP envelope requests.
func NewHTTPEnvelopeAdapter(request *http.Request) *HTTPEnvelopeAdapter {
	return &HTTPEnvelopeAdapter{Request: request}
}

// Envelope parses and returns a transport-neutral envelope payload.
func (a *HTTPEnvelopeAdapter) Envelope() (EnvelopeMessage, error) {
	if a == nil || a.Request == nil {
		return EnvelopeMessage{}, fmt.Errorf("request is required")
	}

	if a.Request.Method != http.MethodPost {
		return EnvelopeMessage{}, fmt.Errorf("method not allowed: %s", a.Request.Method)
	}

	var envelope shared.PubSubPushEnvelope
	if err := json.NewDecoder(a.Request.Body).Decode(&envelope); err != nil {
		return EnvelopeMessage{}, fmt.Errorf("decode pubsub push envelope: %w", err)
	}

	messageID := strings.TrimSpace(envelope.Message.MessageID)
	if messageID == "" {
		return EnvelopeMessage{}, fmt.Errorf("pubsub message id is required")
	}
	if strings.TrimSpace(envelope.Message.Data) == "" {
		return EnvelopeMessage{}, fmt.Errorf("pubsub message data is empty")
	}

	decodedData, err := base64.StdEncoding.DecodeString(envelope.Message.Data)
	if err != nil {
		return EnvelopeMessage{}, fmt.Errorf("decode pubsub message data: %w", err)
	}

	return EnvelopeMessage{
		MessageID:       messageID,
		EventData:       decodedData,
		DeliveryAttempt: envelope.DeliveryAttempt,
	}, nil
}
