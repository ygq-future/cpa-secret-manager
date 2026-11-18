package management

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cpa-secret-manager/internal/errorslog"
	"cpa-secret-manager/internal/pricing"
)

type stubBackend struct {
	settings Settings
	updated  *SettingsUpdate
	remark   *RemarkUpdate
	expiry   *ExpiryUpdate
	key      string
	forgot   []string
	failures []FailureItem
	clearReq *ClearFailuresRequest
	cleared  int
	resolved *ResolveRequest

	pricingResult PricingResult
	pricingErr    error
	pricingModels []string
	refreshModels []string
	refreshMerge  bool
}

func (s *stubBackend) Settings(context.Context) (Settings, error) {
	return s.settings, nil
}

func (s *stubBackend) UpdateSettings(_ context.Context, update SettingsUpdate) error {
	s.updated = &update
	return nil
}

func (s *stubBackend) Resolve(_ context.Context, req ResolveRequest) (ResolveResult, error) {
	s.resolved = &req
	items := make([]KeyEntry, 0, len(req.Keys))
	for _, key := range req.Keys {
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

func (s *stubBackend) SetExpiry(_ context.Context, key string, expiresAt string) error {
	s.expiry = &ExpiryUpdate{Key: key, ExpiresAt: expiresAt}
	return nil
}

func (s *stubBackend) GenerateKey(context.Context) (string, error) {
	return s.key, nil
}

func (s *stubBackend) Failures(context.Context, FailuresRequest) (FailuresResult, error) {
	return FailuresResult{Items: s.failures}, nil
}

func (s *stubBackend) ClearFailures(_ context.Context, req ClearFailuresRequest) (ClearFailuresResult, error) {
	copyReq := req
	s.clearReq = &copyReq
	return ClearFailuresResult{Status: "ok", Removed: s.cleared}, nil
}
func (s *stubBackend) Quota(context.Context, string) (QuotaSummary, bool, error) {
	return QuotaSummary{Enabled: true, LimitValue: 1000}, true, nil
}

func (s *stubBackend) SetQuota(context.Context, QuotaUpdate) error {
	return nil
}

func (s *stubBackend) DeleteQuota(context.Context, string) error {
	return nil
}
func (s *stubBackend) Models(context.Context) ([]string, error) {
	return []string{"gpt-5.6", "claude-3-7-sonnet"}, nil
}

func (s *stubBackend) Pricing(_ context.Context, models []string) (PricingResult, error) {
	s.pricingModels = append([]string(nil), models...)
	return s.pricingResult, s.pricingErr
}

func (s *stubBackend) RefreshPricing(_ context.Context, models []string, merge bool) (PricingResult, error) {
	s.refreshModels = append([]string(nil), models...)
	s.refreshMerge = merge
	return s.pricingResult, s.pricingErr
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
	backend := &stubBackend{settings: Settings{
		UsageEnabled: true,
		StatePath:    "data/x.json",
		CacheFiles: []CacheFile{
			{Key: "state", Name: "State Cache", Path: "data/x.json", SizeBytes: 123, Exists: true},
		},
	}}
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
	if got.StatePath != "data/x.json" || !got.UsageEnabled || len(got.CacheFiles) != 1 || got.CacheFiles[0].SizeBytes != 123 {
		t.Fatalf("settings = %+v, want the backend payload with cache files", got)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/v0/management/plugins/"+PluginID+"/settings",
		strings.NewReader(`{"usage_enabled":false}`)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200", recorder.Code)
	}
	if backend.updated == nil || backend.updated.UsageEnabled == nil || *backend.updated.UsageEnabled {
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

func TestHandler_ResolveForwardsTheUsageWindow(t *testing.T) {
	backend := &stubBackend{}
	handler := NewHandler(backend)

	recorder := httptest.NewRecorder()
	body := `{"keys":["sk-a"],"start":"2026-09-19T00:00:00Z","end":"2026-10-19T09:00:00Z"}`
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, RouteResolve, strings.NewReader(body)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if backend.resolved == nil {
		t.Fatal("backend never received the resolve request")
	}
	if backend.resolved.Start != "2026-09-19T00:00:00Z" || backend.resolved.End != "2026-10-19T09:00:00Z" {
		t.Fatalf("window = %q..%q, want the submitted bounds", backend.resolved.Start, backend.resolved.End)
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

func TestHandler_ExpiryForwardsTheInstant(t *testing.T) {
	backend := &stubBackend{}
	handler := NewHandler(backend)

	recorder := httptest.NewRecorder()
	body := `{"key":"sk-a","expires_at":"2026-12-01T12:30:00Z"}`
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, RouteExpiry, strings.NewReader(body)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if backend.expiry == nil || backend.expiry.Key != "sk-a" || backend.expiry.ExpiresAt != "2026-12-01T12:30:00Z" {
		t.Fatalf("expiry = %+v, want the submitted key and instant", backend.expiry)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, RouteExpiry, nil))
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

func TestHandler_FailuresQueriesErrorLogs(t *testing.T) {
	backend := &stubBackend{
		failures: []FailureItem{
			{Hash: "h1", ErrorEntry: errorslog.ErrorEntry{Model: "gpt-5.6", StatusCode: 429, Message: "rate limit"}},
		},
	}
	handler := NewHandler(backend)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, RouteFailures, strings.NewReader(`{"hash":"h1"}`))
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var res FailuresResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal failures result: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].StatusCode != 429 {
		t.Fatalf("unexpected failures result: %+v", res)
	}
}
func TestHandler_QuotaRoutes(t *testing.T) {
	backend := &stubBackend{}
	handler := NewHandler(backend)

	// GET without hash -> 400
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, RouteQuota, nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("GET without hash status = %d, want 400", rec.Code)
	}

	// GET with hash -> 200
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, RouteQuota+"?hash=h1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET with hash status = %d, want 200", rec.Code)
	}

	// PUT with valid body -> 200
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, RouteQuota, strings.NewReader(`{"hash":"h1","enabled":true,"limit_value":5000}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT quota status = %d, want 200", rec.Code)
	}

	// DELETE with hash -> 200
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, RouteQuota+"?hash=h1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE with hash status = %d, want 200", rec.Code)
	}
}

func TestHandler_ModelsRoute(t *testing.T) {
	backend := &stubBackend{}
	handler := NewHandler(backend)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, RouteModels, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET models status = %d, want 200", rec.Code)
	}
}

func TestHandler_PricingRoutes(t *testing.T) {
	backend := &stubBackend{pricingResult: PricingResult{
		Source:    "models.dev",
		FetchedAt: "2026-11-10T12:00:00Z",
		Models: map[string]pricing.Suggestion{
			"gpt-4o": {Model: "gpt-4o", Provider: "openai", Input: 2.5, Output: 10, Cache: 1.25},
		},
		Matched: 1, Requested: 2,
	}}
	handler := NewHandler(backend)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, RoutePricing, strings.NewReader(`{"models":["gpt-4o","missing"]}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST pricing status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if len(backend.pricingModels) != 2 || backend.pricingModels[0] != "gpt-4o" {
		t.Fatalf("pricing models = %#v, want the submitted list", backend.pricingModels)
	}
	var result PricingResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode pricing result: %v", err)
	}
	if result.Matched != 1 || result.Models["gpt-4o"].Provider != "openai" {
		t.Fatalf("pricing result = %+v", result)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, RoutePricingRefresh, strings.NewReader(`{"models":["gpt-4o"]}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST pricing refresh status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if len(backend.refreshModels) != 1 || backend.refreshModels[0] != "gpt-4o" {
		t.Fatalf("refresh models = %#v, want the submitted list", backend.refreshModels)
	}
	if backend.refreshMerge {
		t.Fatal("refresh merge = true, want the default full refresh")
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, RoutePricingRefresh, strings.NewReader(`{"models":["gpt-4o"],"merge":true}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST pricing merge refresh status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !backend.refreshMerge {
		t.Fatal("refresh merge = false, want the submitted merge flag")
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, RoutePricing, nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET pricing status = %d, want 405", rec.Code)
	}
}

func TestHandler_PricingRejectsMalformedBody(t *testing.T) {
	handler := NewHandler(&stubBackend{})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, RoutePricingRefresh, strings.NewReader(`{`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
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

func TestHandler_FailuresClear(t *testing.T) {
	backend := &stubBackend{cleared: 3}
	handler := NewHandler(backend)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, PathFailuresClear, nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want 405", rec.Code)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, PathFailuresClear, strings.NewReader(`{"hash":"abc"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing after status = %d, want 400", rec.Code)
	}

	rec = httptest.NewRecorder()
	body := `{"before":"2026-11-17T10:00:00Z","hash":"abc","auth_id":"auth-1","model":"gpt-5.6","status_code":0}`
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v0/management/plugins/"+PluginID+PathFailuresClear, strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	got := backend.clearReq
	if got == nil {
		t.Fatal("clear request was not forwarded to the backend")
	}
	if got.Before != "2026-11-17T10:00:00Z" || got.Hash != "abc" || got.AuthID != "auth-1" || got.Model != "gpt-5.6" {
		t.Fatalf("clear request = %+v, want before/hash/auth/model preserved", *got)
	}
	if got.StatusCode == nil || *got.StatusCode != 0 {
		t.Fatalf("status_code = %v, want a pointer to 0", got.StatusCode)
	}

	var res ClearFailuresResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if res.Status != "ok" || res.Removed != 3 {
		t.Fatalf("result = %+v, want status=ok removed=3", res)
	}
}
