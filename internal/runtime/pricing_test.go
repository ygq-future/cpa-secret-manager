package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"cpa-secret-manager/internal/management"
	"cpa-secret-manager/internal/state"
)

const pricingCatalog = `{
  "openai": {
    "models": {
      "gpt-4o": {"id": "gpt-4o", "cost": {"input": 2.5, "output": 10, "cache_read": 1.25}},
      "gpt-4o-2024-08-06": {"id": "gpt-4o-2024-08-06", "cost": {"input": 2.5, "output": 10, "cache_read": 1.25}},
      "gpt-4.1": {"id": "gpt-4.1", "cost": {"input": 2, "output": 8}}
    }
  },
  "reseller": {
    "models": {
      "openai/gpt-4o": {"id": "openai/gpt-4o", "cost": {"input": 2.4, "output": 9.6}}
    }
  },
  "anthropic": {
    "models": {
      "claude-sonnet-4-5": {"id": "claude-sonnet-4-5", "cost": {"input": 3, "output": 15, "cache_read": 0.3}}
    }
  }
}`

type pricingTestServer struct {
	*httptest.Server
	fail      atomic.Bool
	malformed atomic.Bool
	etag      string
	requests  atomic.Int32
	notMod    atomic.Int32
}

func newPricingTestServer(t *testing.T) *pricingTestServer {
	t.Helper()
	server := &pricingTestServer{etag: `"catalog-1"`}
	server.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.requests.Add(1)
		if server.fail.Load() {
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
			return
		}
		if server.malformed.Load() {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"openai":`))
			return
		}
		if r.Header.Get("If-None-Match") == server.etag {
			server.notMod.Add(1)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", server.etag)
		_, _ = w.Write([]byte(pricingCatalog))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestRefreshPricing_StreamsCachesAndReusesETag(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	server := newPricingTestServer(t)
	rt := registerRuntimeWith(t, Options{
		StatePath:  statePath,
		PricingURL: server.URL,
		HTTPClient: server.Client(),
	})

	body := []byte(`{"models":["gpt-4o","claude-sonnet-4-5","missing"]}`)
	status, raw := managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, body)
	if status != http.StatusOK {
		t.Fatalf("refresh status = %d: %s", status, raw)
	}
	var result management.PricingResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode refresh result: %v", err)
	}
	if result.Source != "models.dev" || result.FetchedAt == "" || result.Stale {
		t.Fatalf("refresh metadata = %+v, want a fresh models.dev result", result)
	}
	if result.Requested != 3 || result.Matched != 2 || len(result.Models) != 2 {
		t.Fatalf("refresh result = %+v, want 2 matches out of 3 requested", result)
	}
	if got := result.Models["gpt-4o"]; got.Provider != "openai" || got.Input != 2.5 || got.Output != 10 || got.Cache != 1.25 {
		t.Fatalf("gpt-4o suggestion = %+v, want the OpenAI offer", got)
	}
	if got := result.Models["claude-sonnet-4-5"]; got.Provider != "anthropic" || got.Input != 3 {
		t.Fatalf("claude suggestion = %+v, want the Anthropic offer", got)
	}

	cachePath := PricesPath(statePath)
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("price cache was not written: %v", err)
	}
	cache, err := state.LoadPrices(cachePath)
	if err != nil {
		t.Fatalf("load price cache: %v", err)
	}
	if _, fetchedAt, etag := cache.Meta(); fetchedAt == "" || etag != `"catalog-1"` {
		t.Fatalf("cache meta fetchedAt=%q etag=%q", fetchedAt, etag)
	}
	if got := cache.Suggestions(); len(got) != 2 {
		t.Fatalf("cached suggestions = %+v, want only the matched models", got)
	}

	// The read route must serve the cache without touching the network.
	server.Close()
	status, raw = managementCall(t, rt, http.MethodPost, management.RoutePricing, body)
	if status != http.StatusOK {
		t.Fatalf("cached pricing status = %d: %s", status, raw)
	}
	var cached management.PricingResult
	if err := json.Unmarshal(raw, &cached); err != nil {
		t.Fatalf("decode cached result: %v", err)
	}
	if cached.Matched != 2 || cached.Models["gpt-4o"].Provider != "openai" {
		t.Fatalf("cached result = %+v", cached)
	}
	if server.requests.Load() != 1 {
		t.Fatalf("catalog requests = %d, want 1 (the cached read must not fetch)", server.requests.Load())
	}
}

func TestRefreshPricing_ConditionalGETKeepsCache(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	server := newPricingTestServer(t)
	rt := registerRuntimeWith(t, Options{
		StatePath:  statePath,
		PricingURL: server.URL,
		HTTPClient: server.Client(),
	})

	body := []byte(`{"models":["gpt-4o"]}`)
	if status, raw := managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, body); status != http.StatusOK {
		t.Fatalf("first refresh status = %d: %s", status, raw)
	}
	status, raw := managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, body)
	if status != http.StatusOK {
		t.Fatalf("second refresh status = %d: %s", status, raw)
	}
	var result management.PricingResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode second refresh: %v", err)
	}
	if !result.NotModified || result.Matched != 1 {
		t.Fatalf("second refresh = %+v, want not_modified with the retained suggestion", result)
	}
	if server.notMod.Load() != 1 {
		t.Fatalf("not-modified responses = %d, want 1", server.notMod.Load())
	}
}

func TestRefreshPricing_MergePreservesOtherModels(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	server := newPricingTestServer(t)
	rt := registerRuntimeWith(t, Options{
		StatePath:  statePath,
		PricingURL: server.URL,
		HTTPClient: server.Client(),
	})

	if status, raw := managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, []byte(`{"models":["gpt-4o"]}`)); status != http.StatusOK {
		t.Fatalf("first refresh status = %d: %s", status, raw)
	}
	status, raw := managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, []byte(`{"models":["claude-sonnet-4-5"],"merge":true}`))
	if status != http.StatusOK {
		t.Fatalf("merge refresh status = %d: %s", status, raw)
	}
	var result management.PricingResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode merge refresh: %v", err)
	}
	if result.Matched != 1 || len(result.Models) != 1 {
		t.Fatalf("merge refresh result = %+v, want only the requested model", result)
	}

	cache, err := state.LoadPrices(PricesPath(statePath))
	if err != nil {
		t.Fatalf("load price cache: %v", err)
	}
	suggestions := cache.Suggestions()
	if len(suggestions) != 2 {
		t.Fatalf("cached suggestions = %+v, want the preserved and refreshed models", suggestions)
	}
	if _, ok := suggestions["gpt-4o"]; !ok {
		t.Fatal("merge refresh discarded a model outside the request")
	}
	if got := suggestions["claude-sonnet-4-5"]; got.Provider != "anthropic" || got.Input != 3 {
		t.Fatalf("refreshed suggestion = %+v, want the Anthropic offer", got)
	}
	known := cache.Known()
	seen := map[string]bool{}
	for _, name := range known {
		seen[name] = true
	}
	if !seen["gpt-4o"] || !seen["claude-sonnet-4-5"] {
		t.Fatalf("known models = %#v, want the union of both refreshes", known)
	}
}

func TestRefreshPricing_FailureKeepsPreviousCache(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	server := newPricingTestServer(t)
	rt := registerRuntimeWith(t, Options{
		StatePath:  statePath,
		PricingURL: server.URL,
		HTTPClient: server.Client(),
	})

	body := []byte(`{"models":["gpt-4o"]}`)
	if status, raw := managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, body); status != http.StatusOK {
		t.Fatalf("first refresh status = %d: %s", status, raw)
	}

	server.fail.Store(true)
	status, raw := managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, body)
	if status != http.StatusInternalServerError {
		t.Fatalf("failed refresh status = %d, want 500: %s", status, raw)
	}

	status, raw = managementCall(t, rt, http.MethodPost, management.RoutePricing, body)
	if status != http.StatusOK {
		t.Fatalf("cached pricing status = %d: %s", status, raw)
	}
	var cached management.PricingResult
	if err := json.Unmarshal(raw, &cached); err != nil {
		t.Fatalf("decode cached result: %v", err)
	}
	if cached.Matched != 1 || cached.Models["gpt-4o"].Input != 2.5 {
		t.Fatalf("cache was lost after a failed refresh: %+v", cached)
	}
}

func TestRefreshPricing_MalformedCatalogKeepsPreviousCache(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	server := newPricingTestServer(t)
	rt := registerRuntimeWith(t, Options{
		StatePath:  statePath,
		PricingURL: server.URL,
		HTTPClient: server.Client(),
	})

	body := []byte(`{"models":["gpt-4o"]}`)
	if status, raw := managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, body); status != http.StatusOK {
		t.Fatalf("first refresh status = %d: %s", status, raw)
	}
	server.malformed.Store(true)
	if status, _ := managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, body); status != http.StatusInternalServerError {
		t.Fatalf("malformed refresh status = %d, want 500", status)
	}
	status, raw := managementCall(t, rt, http.MethodPost, management.RoutePricing, body)
	if status != http.StatusOK {
		t.Fatalf("cached pricing status = %d: %s", status, raw)
	}
	var cached management.PricingResult
	if err := json.Unmarshal(raw, &cached); err != nil {
		t.Fatalf("decode cached result: %v", err)
	}
	if cached.Matched != 1 {
		t.Fatalf("cache was lost after a malformed catalog: %+v", cached)
	}
}

func TestRefreshPricing_NewModelBypassesETag(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	server := newPricingTestServer(t)
	rt := registerRuntimeWith(t, Options{
		StatePath:  statePath,
		PricingURL: server.URL,
		HTTPClient: server.Client(),
	})

	if status, raw := managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, []byte(`{"models":["gpt-4o"]}`)); status != http.StatusOK {
		t.Fatalf("first refresh status = %d: %s", status, raw)
	}
	status, raw := managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, []byte(`{"models":["gpt-4o","claude-sonnet-4-5"]}`))
	if status != http.StatusOK {
		t.Fatalf("second refresh status = %d: %s", status, raw)
	}
	var result management.PricingResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode second refresh: %v", err)
	}
	if result.NotModified {
		t.Fatal("a newly requested model must bypass the cached ETag")
	}
	if result.Matched != 2 || result.Models["claude-sonnet-4-5"].Provider != "anthropic" {
		t.Fatalf("second refresh = %+v, want both models resolved", result)
	}
	if server.notMod.Load() != 0 {
		t.Fatalf("not-modified responses = %d, want 0 when the request covers a new model", server.notMod.Load())
	}

	status, raw = managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, []byte(`{"models":["gpt-4o","claude-sonnet-4-5"]}`))
	if status != http.StatusOK {
		t.Fatalf("third refresh status = %d: %s", status, raw)
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode third refresh: %v", err)
	}
	if !result.NotModified || result.Matched != 2 {
		t.Fatalf("third refresh = %+v, want a conditional 304 with both suggestions", result)
	}
}

func TestRefreshPricing_CaseChangeStillUsesETag(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	server := newPricingTestServer(t)
	rt := registerRuntimeWith(t, Options{
		StatePath:  statePath,
		PricingURL: server.URL,
		HTTPClient: server.Client(),
	})

	if status, raw := managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, []byte(`{"models":["GPT-4o"]}`)); status != http.StatusOK {
		t.Fatalf("first refresh status = %d: %s", status, raw)
	}
	status, raw := managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, []byte(`{"models":["gpt-4o"]}`))
	if status != http.StatusOK {
		t.Fatalf("second refresh status = %d: %s", status, raw)
	}
	var result management.PricingResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode second refresh: %v", err)
	}
	if !result.NotModified || result.Matched != 1 {
		t.Fatalf("case-only change = %+v, want a conditional 304 with the cached suggestion", result)
	}
	if server.notMod.Load() != 1 {
		t.Fatalf("not-modified responses = %d, want 1", server.notMod.Load())
	}
}

func TestRefreshPricing_RequiresModels(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	rt := registerRuntimeWith(t, Options{StatePath: statePath})
	status, raw := managementCall(t, rt, http.MethodPost, management.RoutePricingRefresh, []byte(`{"models":[]}`))
	if status != http.StatusBadRequest {
		t.Fatalf("empty refresh status = %d, want 400: %s", status, raw)
	}
}

func TestPricing_CacheSurvivesRestart(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	server := newPricingTestServer(t)
	first := registerRuntimeWith(t, Options{
		StatePath:  statePath,
		PricingURL: server.URL,
		HTTPClient: server.Client(),
	})
	body := []byte(`{"models":["gpt-4o"]}`)
	if status, raw := managementCall(t, first, http.MethodPost, management.RoutePricingRefresh, body); status != http.StatusOK {
		t.Fatalf("refresh status = %d: %s", status, raw)
	}
	if err := first.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown first runtime: %v", err)
	}

	// A fresh runtime must read the compact cache without a network call.
	second := registerRuntimeWith(t, Options{
		StatePath:  statePath,
		PricingURL: "http://127.0.0.1:1/unreachable",
		HTTPClient: &http.Client{Timeout: time.Second},
	})
	status, raw := managementCall(t, second, http.MethodPost, management.RoutePricing, body)
	if status != http.StatusOK {
		t.Fatalf("cached pricing status = %d: %s", status, raw)
	}
	var result management.PricingResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode cached result: %v", err)
	}
	if result.Matched != 1 || result.Models["gpt-4o"].Provider != "openai" {
		t.Fatalf("restart did not restore the cache: %+v", result)
	}
}

func TestPricing_StaleStatus(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	if priceStale(now.Add(-pricingStaleAfter-time.Hour).Format(time.RFC3339), now) != true {
		t.Fatal("an old sync was reported as fresh")
	}
	if priceStale(now.Add(-pricingStaleAfter+time.Hour).Format(time.RFC3339), now) != false {
		t.Fatal("a recent sync was reported as stale")
	}
	if !priceStale("", now) {
		t.Fatal("an absent sync was reported as fresh")
	}
	if !priceStale("not-a-time", now) {
		t.Fatal("a malformed sync was reported as fresh")
	}
}

func TestManagementRoutes_IncludePricing(t *testing.T) {
	routes := managementRoutes()
	wanted := map[string]bool{
		management.RoutePricing:        false,
		management.RoutePricingRefresh: false,
	}
	for _, route := range routes {
		if _, ok := wanted[route.Path]; ok && route.Method == http.MethodPost {
			wanted[route.Path] = true
		}
	}
	for path, found := range wanted {
		if !found {
			t.Fatalf("route %s is not registered for POST", path)
		}
	}
}
