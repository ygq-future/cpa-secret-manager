package management

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubBackend struct {
	settings Settings
	updated  *bool
	remark   *RemarkUpdate
	key      string
	forgot   []string
}

func (s *stubBackend) Settings(context.Context) (Settings, error) {
	return s.settings, nil
}

func (s *stubBackend) UpdateSettings(_ context.Context, usageEnabled bool) error {
	s.updated = &usageEnabled
	return nil
}

func (s *stubBackend) Resolve(_ context.Context, keys []string) (ResolveResult, error) {
	items := make([]KeyEntry, 0, len(keys))
	for _, key := range keys {
		items = append(items, KeyEntry{Hash: "hash:" + key, Remark: key})
	}
	return ResolveResult{Items: items}, nil
}

func (s *stubBackend) Forget(_ context.Context, hashes []string) (int, error) {
	s.forgot = append([]string(nil), hashes...)
	return len(hashes), nil
}

func (s *stubBackend) SetRemark(_ context.Context, key string, remark string) error {
	s.remark = &RemarkUpdate{Key: key, Remark: remark}
	return nil
}

func (s *stubBackend) GenerateKey(context.Context) (string, error) {
	return s.key, nil
}

func TestNormalizePath_ReducesHostPrefixes(t *testing.T) {
	cases := map[string]string{
		"/v0/management/plugins/" + PluginID + "/settings": PathSettings,
		"/v0/resource/plugins/" + PluginID + "/settings":   PathSettings,
		"/plugins/" + PluginID + "/settings":               PathSettings,
		"settings":                                         PathSettings,
		PathSettings:                                       PathSettings,
	}
	for input, want := range cases {
		if got := normalizePath(input); got != want {
			t.Fatalf("normalizePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestHandler_SettingsRoundTrip(t *testing.T) {
	backend := &stubBackend{settings: Settings{UsageEnabled: true, StatePath: "data/x.json"}}
	handler := NewHandler(backend)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v0/management/plugins/"+PluginID+"/settings", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", recorder.Code)
	}
	var got Settings
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if got.StatePath != "data/x.json" || !got.UsageEnabled {
		t.Fatalf("settings = %+v, want the backend payload", got)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/v0/management/plugins/"+PluginID+"/settings",
		strings.NewReader(`{"usage_enabled":false}`)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200", recorder.Code)
	}
	if backend.updated == nil || *backend.updated {
		t.Fatalf("backend update = %v, want false", backend.updated)
	}
}

func TestHandler_RejectsInvalidSettingsPayload(t *testing.T) {
	handler := NewHandler(&stubBackend{})
	for name, body := range map[string]string{
		"missing field": `{}`,
		"not an object": `[]`,
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, PathSettings, strings.NewReader(body)))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400", name, recorder.Code)
		}
	}
}

func TestHandler_UnknownRouteAndMethod(t *testing.T) {
	handler := NewHandler(&stubBackend{})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v0/management/plugins/"+PluginID+"/nope", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unknown route status = %d, want 404", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, PathSettings, nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unsupported method status = %d, want 405", recorder.Code)
	}
}

func TestHandler_ResolveKeepsKeyOrderAndMetadata(t *testing.T) {
	handler := NewHandler(&stubBackend{})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, RouteResolve, strings.NewReader(`{"keys":["sk-a","sk-b"]}`)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var result ResolveResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode resolve result: %v", err)
	}
	if len(result.Items) != 2 || result.Items[1].Hash != "hash:sk-b" {
		t.Fatalf("items = %+v, want one entry per submitted key in order", result.Items)
	}
}

func TestHandler_ForgetDropsSubmittedKeys(t *testing.T) {
	backend := &stubBackend{}
	handler := NewHandler(backend)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, RouteForget, strings.NewReader(`{"hashes":["aa","bb"]}`)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if len(backend.forgot) != 2 || backend.forgot[0] != "aa" || backend.forgot[1] != "bb" {
		t.Fatalf("forgot = %v, want the submitted hashes", backend.forgot)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, RouteForget, strings.NewReader(`{"hashes":[]}`)))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("empty hashes status = %d, want 400", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, RouteForget, nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want 405", recorder.Code)
	}
}

func TestHandler_RemarksRejectsWrongMethod(t *testing.T) {
	handler := NewHandler(&stubBackend{})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, RouteRemarks, strings.NewReader(`{}`)))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", recorder.Code)
	}
	if allow := recorder.Header().Get("Allow"); allow != http.MethodPut {
		t.Fatalf("Allow = %q, want PUT", allow)
	}
}

func TestHandler_GenerateReturnsBackendKey(t *testing.T) {
	handler := NewHandler(&stubBackend{key: "sk-generated"})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, RouteGenerate, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var result GenerateResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode generate result: %v", err)
	}
	if result.Key != "sk-generated" {
		t.Fatalf("key = %q, want sk-generated", result.Key)
	}
}

func TestHandler_MapsBackendValidationToBadRequest(t *testing.T) {
	handler := NewHandler(&rejectingBackend{})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, RouteRemarks, strings.NewReader(`{"key":"sk-a","remark":"x"}`)))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a rejected remark", recorder.Code)
	}
}

type rejectingBackend struct {
	stubBackend
}

func (b *rejectingBackend) SetRemark(context.Context, string, string) error {
	return NewInvalidRequest("remark too long")
}

func TestRecorder_DefaultsToOKAndKeepsHeaders(t *testing.T) {
	recorder := NewRecorder()
	recorder.Header().Set("Content-Type", "application/json")
	if _, err := recorder.Write([]byte("body")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	status, headers, body := recorder.Result()
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if headers.Get("Content-Type") != "application/json" {
		t.Fatalf("content type = %q, want application/json", headers.Get("Content-Type"))
	}
	if string(body) != "body" {
		t.Fatalf("body = %q, want body", string(body))
	}
}
