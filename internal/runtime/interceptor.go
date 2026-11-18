package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cpa-secret-manager/internal/expiry"
	"cpa-secret-manager/internal/quota"
	"cpa-secret-manager/internal/remarks"
)

// RequestInterceptWire represents the incoming wire format from CPA for
// request.intercept_before. Only the client API key and the requested model
// drive the gate, so the request body is not decoded.
type RequestInterceptWire struct {
	Headers map[string][]string `json:"Headers"`
	Model   string              `json:"Model"`
}

// RequestInterceptResponse is the envelope payload returned to CPA. It mirrors
// the host's sdk/pluginapi.RequestInterceptResponse field names: byte slices
// travel as base64 JSON strings, and omitted fields leave the request
// untouched.
type RequestInterceptResponse struct {
	Terminate       bool                `json:"Terminate,omitempty"`
	StatusCode      int                 `json:"StatusCode,omitempty"`
	ResponseHeaders map[string][]string `json:"ResponseHeaders,omitempty"`
	ResponseBody    []byte              `json:"ResponseBody,omitempty"`
}

// handleInterceptAfter is a pass-through. The host invokes both stages for
// every plugin that declares request_interceptor, and this plugin's gate runs
// before credential selection, where the client key and requested model are
// already known.
func (r *Runtime) handleInterceptAfter() []byte {
	return okEnvelope(RequestInterceptResponse{})
}

func (r *Runtime) handleInterceptBefore(ctx context.Context, raw []byte) []byte {
	var wire RequestInterceptWire
	if err := json.Unmarshal(raw, &wire); err != nil {
		return failureEnvelope("invalid_request", fmt.Sprintf("decode intercept request: %v", err))
	}

	apiKey := extractAPIKey(wire.Headers)
	if apiKey != "" {
		hash := remarks.Index(apiKey)
		now := r.clock.Now().UTC()
		if at, ok := r.expiryInstant(hash); ok && expiry.Expired(at, now) {
			return okEnvelope(RequestInterceptResponse{
				Terminate:  true,
				StatusCode: http.StatusUnauthorized,
				ResponseHeaders: map[string][]string{
					"Content-Type": {"application/json"},
				},
				ResponseBody: []byte(buildKeyExpiredBody(at)),
			})
		}
		decision := r.quota.Check(ctx, hash, wire.Model, now)
		if !decision.Allowed {
			return okEnvelope(RequestInterceptResponse{
				Terminate:  true,
				StatusCode: http.StatusForbidden,
				ResponseHeaders: map[string][]string{
					"Content-Type": {"application/json"},
				},
				ResponseBody: []byte(buildQuotaExceededBody(decision)),
			})
		}
	}

	return okEnvelope(RequestInterceptResponse{})
}

// expiryInstant returns the stored expiration instant of one hash index. A
// runtime without a loaded state never expires anything.
func (r *Runtime) expiryInstant(hash string) (time.Time, bool) {
	r.mu.RLock()
	store := r.store
	r.mu.RUnlock()
	if store == nil {
		return time.Time{}, false
	}
	return store.Expiry(hash)
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

func buildKeyExpiredBody(at time.Time) string {
	payload, err := json.Marshal(map[string]any{
		"error": map[string]any{
			"message": fmt.Sprintf("API key expired at %s.", at.UTC().Format(time.RFC3339)),
			"type":    "invalid_request_error",
			"param":   nil,
			"code":    "key_expired",
		},
	})
	if err != nil {
		return `{"error":{"message":"API key expired","type":"invalid_request_error","code":"key_expired"}}`
	}
	return string(payload)
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
