package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cpa-secret-manager/internal/management"
	"cpa-secret-manager/internal/pricing"
	"cpa-secret-manager/internal/version"
)

const (
	// pricingStaleAfter is how long a cached catalog stays "fresh" in the page's
	// status line. It does not trigger a fetch: syncing is always explicit.
	pricingStaleAfter = 7 * 24 * time.Hour
	// pricingTimeout bounds one catalog download.
	pricingTimeout = 20 * time.Second
	// maxCatalogBytes caps the response body. The current catalog is under 6 MiB;
	// the generous cap leaves room for growth while protecting memory.
	maxCatalogBytes = 32 << 20
)

// Pricing implements management.Backend. It only reads the compact cache and
// therefore never performs network I/O.
func (r *Runtime) Pricing(_ context.Context, models []string) (management.PricingResult, error) {
	normalized, err := pricing.NormalizeModels(models)
	if err != nil {
		return management.PricingResult{}, management.NewInvalidRequest(err.Error())
	}
	r.mu.RLock()
	closed := r.closed
	r.mu.RUnlock()
	if closed {
		return management.PricingResult{}, ErrShutdown
	}
	return r.pricingResult(normalized), nil
}

// RefreshPricing implements management.Backend. It downloads the upstream
// catalog, keeps only the requested models and atomically replaces the compact
// cache. The page calls it from an explicit sync action.
func (r *Runtime) RefreshPricing(ctx context.Context, models []string) (management.PricingResult, error) {
	normalized, err := pricing.NormalizeModels(models)
	if err != nil {
		return management.PricingResult{}, management.NewInvalidRequest(err.Error())
	}
	if len(normalized) == 0 {
		return management.PricingResult{}, management.NewInvalidRequest("at least one model is required")
	}

	// One refresh at a time: a second click reuses the cache instead of
	// downloading the catalog twice.
	r.pricingMu.Lock()
	defer r.pricingMu.Unlock()

	r.mu.RLock()
	store := r.priceStore
	closed := r.closed
	catalogURL := r.pricingURL
	client := r.httpClient
	r.mu.RUnlock()
	if closed || store == nil {
		return management.PricingResult{}, ErrShutdown
	}
	if client == nil {
		client = newPricingClient()
	}

	_, _, etag := store.Meta()
	known := store.Known()
	knownSet := make(map[string]struct{}, len(known))
	for _, name := range known {
		knownSet[name] = struct{}{}
	}
	requestETag := etag
	for _, name := range normalized {
		if _, ok := knownSet[pricing.NormalizeModel(name)]; !ok {
			// The catalog may already contain a newly requested model while its
			// ETag is unchanged, so a conditional request would wrongly return
			// 304 and leave the new model without a suggestion.
			requestETag = ""
			break
		}
	}

	suggestions, newETag, notModified, err := fetchCatalog(ctx, client, catalogURL, requestETag, normalized)
	if err != nil {
		return management.PricingResult{}, err
	}

	now := r.clock.Now().UTC()
	if notModified {
		store.Replace(pricing.Source, formatTimestamp(now), etag, store.Suggestions(), known)
	} else {
		store.Replace(pricing.Source, formatTimestamp(now), newETag, suggestions, normalized)
	}
	if err := store.SaveAtomic(); err != nil {
		return management.PricingResult{}, err
	}

	result := r.pricingResult(normalized)
	result.NotModified = notModified
	return result, nil
}

// pricingResult projects the cache onto the requested model list. The response
// is keyed by the caller's spelling; the cache is keyed by the normalized name.
func (r *Runtime) pricingResult(models []string) management.PricingResult {
	result := management.PricingResult{
		Source:    pricing.Source,
		Stale:     true,
		Models:    map[string]pricing.Suggestion{},
		Requested: len(models),
	}

	r.mu.RLock()
	store := r.priceStore
	r.mu.RUnlock()
	if store == nil {
		return result
	}

	source, fetchedAt, _ := store.Meta()
	if source != "" {
		result.Source = source
	}
	result.FetchedAt = fetchedAt
	result.Stale = priceStale(fetchedAt, r.clock.Now().UTC())

	suggestions := store.Suggestions()
	for _, name := range models {
		if suggestion, ok := suggestions[pricing.NormalizeModel(name)]; ok {
			result.Models[name] = suggestion
		}
	}
	result.Matched = len(result.Models)
	return result
}

func priceStale(fetchedAt string, now time.Time) bool {
	trimmed := strings.TrimSpace(fetchedAt)
	if trimmed == "" {
		return true
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return true
	}
	return now.Sub(parsed) > pricingStaleAfter
}

func fetchCatalog(ctx context.Context, client *http.Client, catalogURL string, etag string, models []string) (map[string]pricing.Suggestion, string, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, catalogURL, nil)
	if err != nil {
		return nil, "", false, fmt.Errorf("pricing: build request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "cpa-secret-manager/"+version.PluginVersion)
	if strings.TrimSpace(etag) != "" {
		request.Header.Set("If-None-Match", etag)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, "", false, fmt.Errorf("pricing: fetch %s: %w", pricing.Source, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	switch response.StatusCode {
	case http.StatusNotModified:
		return nil, etag, true, nil
	case http.StatusOK:
	default:
		return nil, "", false, fmt.Errorf("pricing: %s returned HTTP %d", pricing.Source, response.StatusCode)
	}

	if response.ContentLength > maxCatalogBytes {
		return nil, "", false, fmt.Errorf("pricing: %s catalog is too large (%d bytes)", pricing.Source, response.ContentLength)
	}

	counter := &countingReader{reader: response.Body}
	suggestions, err := pricing.Decode(io.LimitReader(counter, maxCatalogBytes+1), models)
	if counter.read > maxCatalogBytes {
		return nil, "", false, fmt.Errorf("pricing: %s catalog exceeds %d bytes", pricing.Source, maxCatalogBytes)
	}
	if err != nil {
		return nil, "", false, fmt.Errorf("pricing: decode %s: %w", pricing.Source, err)
	}
	return suggestions, strings.TrimSpace(response.Header.Get("ETag")), false, nil
}

type countingReader struct {
	reader io.Reader
	read   int64
}

func (r *countingReader) Read(buffer []byte) (int, error) {
	n, err := r.reader.Read(buffer)
	r.read += int64(n)
	return n, err
}

func newPricingClient() *http.Client {
	return &http.Client{
		Timeout: pricingTimeout,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("pricing: too many redirects")
			}
			if request.URL.Scheme != "https" {
				return errors.New("pricing: redirect target is not HTTPS")
			}
			return nil
		},
	}
}
