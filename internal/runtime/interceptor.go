package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cpa-secret-manager/internal/quota"
	"cpa-secret-manager/internal/remarks"
)

// RequestInterceptWire represents the incoming wire format from CPA for request.intercept_before.
type RequestInterceptWire struct {
	Headers      map[string][]string `json:"headers"`
	HeadersUpper map[string][]string `json:"Headers"`
	Body         json.RawMessage     `json:"body"`
	BodyUpper    json.RawMessage     `json:"Body"`
	Model        string              `json:"model"`
	RequestID    string              `json:"request_id"`
}

// RequestInterceptResponse is the envelope payload returned to CPA.
type RequestInterceptResponse struct {
	Terminate       bool                `json:"terminate"`
	StatusCode      int                 `json:"status_code,omitempty"`
	ResponseHeaders map[string][]string `json:"response_headers,omitempty"`
	ResponseBody    string              `json:"response_body,omitempty"`
	Headers         map[string][]string `json:"headers,omitempty"`
	Body            json.RawMessage     `json:"body,omitempty"`
}

func (r *Runtime) handleInterceptBefore(ctx context.Context, raw []byte) []byte {
	var wire RequestInterceptWire
	if err := json.Unmarshal(raw, &wire); err != nil {
		return failureEnvelope("invalid_request", fmt.Sprintf("decode intercept request: %v", err))
	}

	headers := wire.Headers
	if len(headers) == 0 && len(wire.HeadersUpper) > 0 {
		headers = wire.HeadersUpper
	}

	body := wire.Body
	if len(body) == 0 && len(wire.BodyUpper) > 0 {
		body = wire.BodyUpper
	}

	apiKey := extractAPIKey(headers)
	if apiKey != "" {
		hash := remarks.Index(apiKey)
		now := r.clock.Now().UTC()
		decision := r.quota.Check(ctx, hash, wire.Model, now)
		if !decision.Allowed {
			return okEnvelope(RequestInterceptResponse{
				Terminate:  true,
				StatusCode: http.StatusForbidden,
				ResponseHeaders: map[string][]string{
					"Content-Type": {"application/json"},
				},
				ResponseBody: buildQuotaExceededBody(decision),
			})
		}
	}

	return okEnvelope(RequestInterceptResponse{
		Terminate: false,
		Headers:   headers,
		Body:      body,
	})
}

func extractAPIKey(headers map[string][]string) string {
	for k, vals := range headers {
		lower := strings.ToLower(k)
		for _, v := range vals {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if lower == "authorization" {
				if strings.HasPrefix(strings.ToLower(v), "bearer ") {
					return strings.TrimSpace(v[7:])
				}
				return v
			}
			if lower == "x-api-key" || lower == "api-key" {
				return v
			}
		}
	}
	return ""
}

func buildQuotaExceededBody(d quota.Decision) string {
	msg := "Usage quota exceeded: The API key has reached its token limit for the current window."
	if d.Reason != "" {
		msg = fmt.Sprintf("Usage quota exceeded: %s.", d.Reason)
	}
	if !d.ResetAt.IsZero() {
		msg += fmt.Sprintf(" Quota will reset at %s.", d.ResetAt.UTC().Format(time.RFC3339))
	}

	payload, err := json.Marshal(map[string]any{
		"error": map[string]any{
			"message": msg,
			"type":    "insufficient_quota",
			"param":   nil,
			"code":    "quota_exceeded",
		},
	})
	if err != nil {
		return `{"error":{"message":"Usage quota exceeded","type":"insufficient_quota","code":"quota_exceeded"}}`
	}
	return string(payload)
}
