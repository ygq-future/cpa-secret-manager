package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cpa-secret-manager/internal/management"
	"cpa-secret-manager/internal/remarks"
)

type fixedClock struct {
	at time.Time
}

func (c *fixedClock) Now() time.Time { return c.at }

func setExpiryCall(t *testing.T, rt *Runtime, key string, expiresAt string) (int, []byte) {
	t.Helper()
	body, err := json.Marshal(management.ExpiryUpdate{Key: key, ExpiresAt: expiresAt})
	if err != nil {
		t.Fatalf("encode expiry update: %v", err)
	}
	return managementCall(t, rt, http.MethodPut, management.RouteExpiry, body)
}

func interceptCall(t *testing.T, rt *Runtime, apiKey string) RequestInterceptResponse {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"headers": map[string][]string{
			"Authorization": {"Bearer " + apiKey},
		},
		"model": "gpt-5.6",
	})
	if err != nil {
		t.Fatalf("encode intercept request: %v", err)
	}
	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodRequestInterceptBefore, raw))
	if !env.OK {
		t.Fatalf("intercept_before failed: %+v", env.Error)
	}
	var response RequestInterceptResponse
	if err := json.Unmarshal(env.Result, &response); err != nil {
		t.Fatalf("decode intercept response: %v", err)
	}
	return response
}

func TestExpiry_EnforcedAtInterceptAndReportedByResolve(t *testing.T) {
	now := time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)
	clock := &fixedClock{at: now}
	rt := registerRuntimeWith(t, Options{StatePath: filepath.Join(t.TempDir(), "state.json"), Clock: clock})

	expiring := "sk-expiring"
	permanent := "sk-permanent"
	expiresAt := now.Add(time.Hour)

	if status, body := setExpiryCall(t, rt, expiring, expiresAt.Format(time.RFC3339)); status != http.StatusOK {
		t.Fatalf("PUT expiry status = %d, body = %s", status, body)
	}

	result := resolveResult(t, rt, []string{expiring, permanent})
	if result.Items[0].ExpiresAt != expiresAt.Format(time.RFC3339) || result.Items[0].Expired {
		t.Fatalf("first item expiry = %q expired=%v, want %q and false", result.Items[0].ExpiresAt, result.Items[0].Expired, expiresAt.Format(time.RFC3339))
	}
	if result.Items[1].ExpiresAt != "" || result.Items[1].Expired {
		t.Fatalf("second item = %+v, want no expiry", result.Items[1])
	}

	if response := interceptCall(t, rt, expiring); response.Terminate {
		t.Fatalf("key was rejected before its expiry: %+v", response)
	}

	clock.at = expiresAt
	result = resolveResult(t, rt, []string{expiring, permanent})
	if !result.Items[0].Expired {
		t.Fatalf("item not marked expired at the boundary: %+v", result.Items[0])
	}

	response := interceptCall(t, rt, expiring)
	if !response.Terminate || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expired key response = %+v, want 401 termination", response)
	}
	if !json.Valid([]byte(response.ResponseBody)) || !strings.Contains(response.ResponseBody, "key_expired") {
		t.Fatalf("expired key body = %s, want a key_expired error", response.ResponseBody)
	}
	if response := interceptCall(t, rt, permanent); response.Terminate {
		t.Fatalf("key without expiry was rejected: %+v", response)
	}
}

func TestExpiry_ClearRejectsPastAndSurvivesForget(t *testing.T) {
	now := time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)
	clock := &fixedClock{at: now}
	rt := registerRuntimeWith(t, Options{StatePath: filepath.Join(t.TempDir(), "state.json"), Clock: clock})
	apiKey := "sk-expiry-lifecycle"
	hash := remarks.Index(apiKey)

	if status, _ := setExpiryCall(t, rt, apiKey, now.Add(-time.Minute).Format(time.RFC3339)); status != http.StatusBadRequest {
		t.Fatalf("past expiry status = %d, want 400", status)
	}
	if status, _ := setExpiryCall(t, rt, apiKey, "2026-11-14 08:00"); status != http.StatusBadRequest {
		t.Fatalf("malformed expiry status = %d, want 400", status)
	}

	if status, _ := setExpiryCall(t, rt, apiKey, now.Add(time.Hour).Format(time.RFC3339)); status != http.StatusOK {
		t.Fatalf("PUT expiry status = %d, want 200", status)
	}
	if _, ok := rt.Store().Expiry(hash); !ok {
		t.Fatal("expiry was not persisted")
	}

	if status, _ := setExpiryCall(t, rt, apiKey, ""); status != http.StatusOK {
		t.Fatalf("clear expiry status = %d, want 200", status)
	}
	if _, ok := rt.Store().Expiry(hash); ok {
		t.Fatal("empty expires_at did not clear the expiry")
	}

	if status, _ := setExpiryCall(t, rt, apiKey, now.Add(time.Hour).Format(time.RFC3339)); status != http.StatusOK {
		t.Fatalf("PUT expiry status = %d, want 200", status)
	}
	if _, err := rt.Forget(context.Background(), []string{hash}); err != nil {
		t.Fatalf("Forget() error = %v", err)
	}
	if _, ok := rt.Store().Expiry(hash); ok {
		t.Fatal("forget left the expiry behind")
	}
}

func TestManagementRegistration_DeclaresExpiryRoute(t *testing.T) {
	rt := registerRuntime(t, filepath.Join(t.TempDir(), "state.json"))

	env := decodeEnvelope(t, rt.Handle(context.Background(), MethodManagementRegister, nil))
	var registration managementRegistration
	if err := json.Unmarshal(env.Result, &registration); err != nil {
		t.Fatalf("decode management registration: %v", err)
	}
	for _, route := range registration.Routes {
		if route.Method == http.MethodPut && route.Path == management.RouteExpiry {
			return
		}
	}
	t.Fatalf("routes = %+v, want PUT %s", registration.Routes, management.RouteExpiry)
}
