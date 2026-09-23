package runtime

import (
	"encoding/json"
	"fmt"
)

// Envelope is the JSON envelope format used by the CPA C ABI.
type Envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *EnvelopeError  `json:"error,omitempty"`
}

// EnvelopeError represents error details inside a failure envelope.
type EnvelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func decodeConfigYAML(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var asBytes struct {
		ConfigYAML []byte `json:"config_yaml"`
	}
	if err := json.Unmarshal(raw, &asBytes); err == nil && len(asBytes.ConfigYAML) > 0 {
		return string(asBytes.ConfigYAML), nil
	}
	var asString struct {
		ConfigYAML string `json:"config_yaml"`
	}
	if err := json.Unmarshal(raw, &asString); err != nil {
		return "", fmt.Errorf("%w: decode config_yaml: %v", ErrInvalidRequest, err)
	}
	return asString.ConfigYAML, nil
}

func okEnvelope(result any) []byte {
	raw, err := json.Marshal(result)
	if err != nil {
		return failureEnvelope("encode_failed", err.Error())
	}
	return mustMarshal(Envelope{OK: true, Result: raw})
}

func failureEnvelope(code string, message string) []byte {
	return mustMarshal(Envelope{OK: false, Error: &EnvelopeError{Code: code, Message: message}})
}

func mustMarshal(envelope Envelope) []byte {
	raw, err := json.Marshal(envelope)
	if err != nil {
		return []byte(`{"ok":false,"error":{"code":"encode_failed","message":"envelope encode failed"}}`)
	}
	return raw
}
