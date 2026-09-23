package runtime

import (
	"encoding/json"
	"fmt"
	"time"

	"cpa-secret-manager/internal/remarks"
	"cpa-secret-manager/internal/usage"
)

func (r *Runtime) handleUsage(raw []byte) []byte {
	record, err := decodeUsageRecord(raw)
	if err != nil {
		return failureEnvelope("invalid_request", err.Error())
	}
	r.ObserveUsage(record)
	return okEnvelope(map[string]string{"status": "ok"})
}

// decodeUsageRecord translates the host UsageRecord payload. Field names are
// looked up tolerantly because the host serializes Go struct fields while older
// payloads used snake_case.
func decodeUsageRecord(raw []byte) (usage.Record, error) {
	if len(raw) == 0 {
		return usage.Record{}, fmt.Errorf("%w: usage record is required", ErrInvalidRequest)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return usage.Record{}, fmt.Errorf("%w: decode usage record: %v", ErrInvalidRequest, err)
	}

	model := pickString(fields, "Model", "model")
	if model == "" {
		model = pickString(fields, "Alias", "alias")
	}
	record := usage.Record{
		Model:  model,
		Failed: pickBool(fields, "Failed", "failed"),
		Detail: decodeUsageDetail(pickField(fields, "Detail", "detail")),
	}
	if at := pickTime(fields, "RequestedAt", "requested_at"); !at.IsZero() {
		record.At = at.UTC()
	}
	if apiKey := pickString(fields, "APIKey", "api_key"); apiKey != "" {
		record.Hash = remarks.Index(apiKey)
	}
	return record, nil
}

func decodeUsageDetail(raw json.RawMessage) usage.Detail {
	var fields map[string]json.RawMessage
	if len(raw) == 0 {
		return usage.Detail{}
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return usage.Detail{}
	}
	return usage.Detail{
		Input:         pickInt64(fields, "InputTokens", "input_tokens"),
		Output:        pickInt64(fields, "OutputTokens", "output_tokens"),
		Reasoning:     pickInt64(fields, "ReasoningTokens", "reasoning_tokens"),
		Cached:        pickInt64(fields, "CachedTokens", "cached_tokens"),
		CacheRead:     pickInt64(fields, "CacheReadTokens", "cache_read_tokens"),
		CacheCreation: pickInt64(fields, "CacheCreationTokens", "cache_creation_tokens"),
		Total:         pickInt64(fields, "TotalTokens", "total_tokens"),
	}
}

func pickField(fields map[string]json.RawMessage, names ...string) json.RawMessage {
	for _, name := range names {
		if raw, ok := fields[name]; ok {
			return raw
		}
	}
	return nil
}

func pickString(fields map[string]json.RawMessage, names ...string) string {
	var value string
	if raw := pickField(fields, names...); raw != nil {
		_ = json.Unmarshal(raw, &value)
	}
	return value
}

func pickBool(fields map[string]json.RawMessage, names ...string) bool {
	var value bool
	if raw := pickField(fields, names...); raw != nil {
		_ = json.Unmarshal(raw, &value)
	}
	return value
}

func pickInt64(fields map[string]json.RawMessage, names ...string) int64 {
	var value int64
	if raw := pickField(fields, names...); raw != nil {
		_ = json.Unmarshal(raw, &value)
	}
	return value
}

func pickTime(fields map[string]json.RawMessage, names ...string) time.Time {
	var value time.Time
	if raw := pickField(fields, names...); raw != nil {
		_ = json.Unmarshal(raw, &value)
	}
	return value
}
