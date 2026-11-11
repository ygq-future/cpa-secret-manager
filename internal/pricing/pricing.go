// Package pricing turns the models.dev catalog into compact per-model reference
// prices. It is a pure domain package: callers supply the catalog stream, and
// the package only parses, matches and ranks it. Network access and cache
// persistence belong to the application layer.
//
// The catalog is a provider-keyed object whose model entries carry descriptions
// and metadata far beyond pricing. Decode therefore walks the JSON token by
// token, skips every model that is not part of the caller's model list, and only
// retains the few pricing fields of the matches. The full document is never
// materialized in memory.
package pricing

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
)

const (
	// Source is the upstream catalog name shown in the management page.
	Source = "models.dev"
	// SourceURL is the catalog endpoint. It is fixed: no user input reaches it.
	SourceURL = "https://models.dev/api.json"
	// MaxModels bounds one suggestion request. The page only asks about models
	// it already knows, so a larger list signals a malformed caller.
	MaxModels = 500
	// maxModelName bounds one model identifier.
	maxModelName = 200
)

// Suggestion is the reference price for one requested model, resolved from the
// provider offers found in the catalog. Prices are USD per 1M tokens, the same
// unit the plugin's ModelPrice uses.
type Suggestion struct {
	// Model is the upstream model identifier the price came from.
	Model string `json:"model"`
	// Provider is the upstream provider identifier, for example "openai".
	Provider  string  `json:"provider"`
	Input     float64 `json:"input"`
	Output    float64 `json:"output"`
	Cache     float64 `json:"cache"`
	UpdatedAt string  `json:"updated_at,omitempty"`
	// Providers is how many provider offers matched this model.
	Providers int `json:"providers,omitempty"`
}

// NormalizeModel returns the case-insensitive lookup key of a model name.
func NormalizeModel(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// NormalizeModels trims, drops empty names, removes duplicates and enforces the
// request bounds. The returned names preserve the caller's spelling so responses
// can be keyed by the exact model the page asked about.
func NormalizeModels(models []string) ([]string, error) {
	seen := make(map[string]struct{}, len(models))
	out := make([]string, 0, len(models))
	for _, raw := range models {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if len(name) > maxModelName {
			return nil, fmt.Errorf("model name exceeds %d characters", maxModelName)
		}
		key := NormalizeModel(name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	if len(out) > MaxModels {
		return nil, fmt.Errorf("too many models: %d (max %d)", len(out), MaxModels)
	}
	return out, nil
}

// Decode reads a models.dev catalog and returns suggestions keyed by normalized
// model name. Only the wanted models are retained; every other model entry is
// skipped without allocating its value.
func Decode(r io.Reader, wanted []string) (map[string]Suggestion, error) {
	normalized, err := NormalizeModels(wanted)
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		return map[string]Suggestion{}, nil
	}

	index := newAliasIndex(normalized)
	candidates := make(map[string][]candidate, len(normalized))

	decoder := json.NewDecoder(r)
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("decode catalog: %w", err)
	}
	if !isDelim(token, '{') {
		return nil, errors.New("decode catalog: top-level value is not an object")
	}
	for decoder.More() {
		providerToken, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("decode catalog: %w", err)
		}
		provider, ok := providerToken.(string)
		if !ok {
			return nil, errors.New("decode catalog: provider key is not a string")
		}
		if err := decodeProvider(decoder, provider, index, candidates); err != nil {
			return nil, err
		}
	}
	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("decode catalog: %w", err)
	}
	return resolve(candidates), nil
}

type costEntry struct {
	Input     *float64 `json:"input"`
	Output    *float64 `json:"output"`
	CacheRead *float64 `json:"cache_read"`
}

type modelEntry struct {
	ID               string     `json:"id"`
	CanonicalModelID string     `json:"canonical_model_id"`
	LastUpdated      string     `json:"last_updated"`
	Cost             *costEntry `json:"cost"`
}

type candidate struct {
	provider       string
	modelID        string
	updated        string
	input          float64
	output         float64
	cache          float64
	hasPrice       bool
	hasCache       bool
	exact          bool
	canonicalMatch bool
}

func decodeProvider(decoder *json.Decoder, provider string, index aliasIndex, out map[string][]candidate) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode provider %q: %w", provider, err)
	}
	if !isDelim(token, '{') {
		return fmt.Errorf("decode provider %q: value is not an object", provider)
	}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("decode provider %q: %w", provider, err)
		}
		key, ok := keyToken.(string)
		if !ok {
			return fmt.Errorf("decode provider %q: field name is not a string", provider)
		}
		if key != "models" {
			if err := skipValue(decoder); err != nil {
				return fmt.Errorf("decode provider %q: %w", provider, err)
			}
			continue
		}
		if err := decodeModels(decoder, provider, index, out); err != nil {
			return err
		}
	}
	if _, err := decoder.Token(); err != nil {
		return fmt.Errorf("decode provider %q: %w", provider, err)
	}
	return nil
}

func decodeModels(decoder *json.Decoder, provider string, index aliasIndex, out map[string][]candidate) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode models of %q: %w", provider, err)
	}
	if !isDelim(token, '{') {
		return fmt.Errorf("decode models of %q: value is not an object", provider)
	}
	for decoder.More() {
		modelToken, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("decode models of %q: %w", provider, err)
		}
		modelID, ok := modelToken.(string)
		if !ok {
			return fmt.Errorf("decode models of %q: model key is not a string", provider)
		}
		matched := index.match(modelID)
		if len(matched) == 0 {
			if err := skipValue(decoder); err != nil {
				return fmt.Errorf("decode model %q: %w", modelID, err)
			}
			continue
		}

		var entry modelEntry
		if err := decoder.Decode(&entry); err != nil {
			return fmt.Errorf("decode model %q: %w", modelID, err)
		}
		if entry.ID == "" {
			entry.ID = modelID
		}
		price, ok := priceOf(entry.Cost)
		if !ok {
			continue
		}
		for _, wanted := range matched {
			out[wanted] = append(out[wanted], candidate{
				provider:       provider,
				modelID:        entry.ID,
				updated:        entry.LastUpdated,
				input:          price.input,
				output:         price.output,
				cache:          price.cache,
				hasPrice:       price.hasPrice,
				hasCache:       price.hasCache,
				exact:          NormalizeModel(entry.ID) == NormalizeModel(wanted),
				canonicalMatch: index.matchesName(wanted, entry.CanonicalModelID),
			})
		}
	}
	if _, err := decoder.Token(); err != nil {
		return fmt.Errorf("decode models of %q: %w", provider, err)
	}
	return nil
}

type price struct {
	input    float64
	output   float64
	cache    float64
	hasPrice bool
	hasCache bool
}

func priceOf(entry *costEntry) (price, bool) {
	if entry == nil || (entry.Input == nil && entry.Output == nil) {
		return price{}, false
	}
	out := price{hasPrice: true}
	if entry.Input != nil {
		out.input = *entry.Input
	}
	if entry.Output != nil {
		out.output = *entry.Output
	}
	if entry.CacheRead != nil {
		out.cache = *entry.CacheRead
		out.hasCache = true
	}
	return out, true
}

func resolve(candidates map[string][]candidate) map[string]Suggestion {
	out := make(map[string]Suggestion, len(candidates))
	for wanted, offers := range candidates {
		if len(offers) == 0 {
			continue
		}
		best := offers[0]
		for _, offer := range offers[1:] {
			if better(offer, best, wanted) {
				best = offer
			}
		}
		out[NormalizeModel(wanted)] = Suggestion{
			Model:     best.modelID,
			Provider:  best.provider,
			Input:     roundPrice(best.input),
			Output:    roundPrice(best.output),
			Cache:     roundPrice(best.cache),
			UpdatedAt: best.updated,
			Providers: len(offers),
		}
	}
	return out
}

func better(candidate, current candidate, wanted string) bool {
	candidateScore := score(candidate, wanted)
	currentScore := score(current, wanted)
	if candidateScore != currentScore {
		return candidateScore > currentScore
	}
	if candidate.updated != current.updated {
		return candidate.updated > current.updated
	}
	if candidate.provider != current.provider {
		return candidate.provider < current.provider
	}
	return candidate.modelID < current.modelID
}

func score(candidate candidate, wanted string) int {
	total := providerRank(candidate.provider, candidate.modelID)
	if candidate.exact {
		total += 60
	}
	if candidate.canonicalMatch {
		total += 40
	}
	if candidate.hasPrice && (candidate.input > 0 || candidate.output > 0) {
		total += 20
	}
	if candidate.hasCache {
		total += 5
	}
	if candidate.updated != "" {
		total += 2
	}
	return total
}

func roundPrice(value float64) float64 {
	if value <= 0 {
		return 0
	}
	return math.Round(value*1000) / 1000
}

// firstPartyFamilies maps a model family prefix to the vendor that publishes it.
// A vendor's own listing is the best reference price; resellers and gateways are
// only used when no first-party offer exists.
var firstPartyFamilies = []struct {
	prefix   string
	provider string
}{
	{"claude", "anthropic"},
	{"gpt", "openai"},
	{"chatgpt", "openai"},
	{"o1", "openai"},
	{"o3", "openai"},
	{"o4", "openai"},
	{"dall-e", "openai"},
	{"sora", "openai"},
	{"whisper", "openai"},
	{"gemini", "google"},
	{"imagen", "google"},
	{"veo", "google"},
	{"lyria", "google"},
	{"deepseek", "deepseek"},
	{"grok", "xai"},
	{"kimi", "moonshotai"},
	{"qwen", "alibaba"},
	{"qvq", "alibaba"},
	{"glm", "zhipuai"},
	{"minimax", "minimax"},
	{"mistral", "mistral"},
	{"codestral", "mistral"},
	{"devstral", "mistral"},
	{"pixtral", "mistral"},
	{"llama", "meta"},
	{"command", "cohere"},
	{"nemotron", "nvidia"},
	{"seed", "bytedance-seed"},
	{"hunyuan", "tencent"},
	{"mimo", "xiaomi"},
	{"step", "stepfun"},
	{"solar", "upstage"},
	{"granite", "ibm"},
	{"ling", "inclusionai"},
}

var vendorProviders = map[string]struct{}{
	"anthropic": {}, "openai": {}, "google": {}, "deepseek": {}, "xai": {},
	"moonshotai": {}, "alibaba": {}, "zhipuai": {}, "minimax": {}, "mistral": {},
	"meta": {}, "cohere": {}, "nvidia": {}, "bytedance-seed": {}, "tencent": {},
	"xiaomi": {}, "stepfun": {}, "upstage": {}, "ibm": {}, "inclusionai": {},
}

func providerRank(provider, modelID string) int {
	base := baseModelName(modelID)
	for _, family := range firstPartyFamilies {
		if strings.HasPrefix(base, family.prefix) && provider == family.provider {
			return 120
		}
	}
	if lead := leadingSegment(modelID); lead != "" && provider == lead {
		return 80
	}
	if _, ok := vendorProviders[provider]; ok {
		return 40
	}
	return 0
}

func leadingSegment(modelID string) string {
	normalized := NormalizeModel(modelID)
	if index := strings.IndexByte(normalized, '/'); index > 0 {
		return normalized[:index]
	}
	return ""
}

var vendorDotPrefixes = []string{
	"us.", "eu.", "apac.", "global.", "us-gov.",
	"anthropic.", "amazon.", "meta.", "mistral.", "cohere.", "deepseek.",
	"qwen.", "moonshot.", "zai.", "google.", "vertex.", "ai21.", "stability.",
	"openai.",
}

func stripVendorDots(value string) string {
	for {
		changed := false
		for _, prefix := range vendorDotPrefixes {
			if strings.HasPrefix(value, prefix) {
				value = value[len(prefix):]
				changed = true
				break
			}
		}
		if !changed {
			return value
		}
	}
}

func baseModelName(raw string) string {
	value := NormalizeModel(raw)
	if index := strings.LastIndexByte(value, '/'); index >= 0 {
		value = value[index+1:]
	}
	if index := strings.LastIndexByte(value, ':'); index >= 0 {
		value = value[:index]
	}
	value = stripVendorDots(value)
	value = stripVersionSuffix(value)
	value = stripDateSuffix(value)
	return value
}

func stripDateSuffix(value string) string {
	if len(value) > 9 {
		tail := value[len(value)-8:]
		if isDigits(tail) && strings.HasPrefix(tail, "20") && value[len(value)-9] == '-' {
			return value[:len(value)-9]
		}
	}
	if len(value) > 11 {
		tail := value[len(value)-10:]
		if tail[4] == '-' && tail[7] == '-' &&
			isDigits(tail[:4]) && isDigits(tail[5:7]) && isDigits(tail[8:]) &&
			value[len(value)-11] == '-' {
			return value[:len(value)-11]
		}
	}
	return value
}

func stripVersionSuffix(value string) string {
	index := strings.LastIndexByte(value, '-')
	if index <= 0 || index+2 > len(value) || value[index+1] != 'v' {
		return value
	}
	if !isDigits(value[index+2:]) {
		return value
	}
	return value[:index]
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

// aliasIndex maps every lookup spelling of the requested models back to the
// caller's original names. A model can match through its full id, its last path
// segment, a vendor-dotted variant, or a date/version-stripped variant.
type aliasIndex struct {
	byAlias map[string][]string
}

func newAliasIndex(wanted []string) aliasIndex {
	index := aliasIndex{byAlias: make(map[string][]string, len(wanted)*4)}
	for _, name := range wanted {
		for _, alias := range aliases(name) {
			index.byAlias[alias] = appendUnique(index.byAlias[alias], name)
		}
	}
	return index
}

func (index aliasIndex) match(modelID string) []string {
	var matches []string
	for _, alias := range aliases(modelID) {
		matches = append(matches, index.byAlias[alias]...)
	}
	return dedupeStrings(matches)
}

func (index aliasIndex) matchesName(name string, modelID string) bool {
	if modelID == "" {
		return false
	}
	for _, match := range index.match(modelID) {
		if match == name {
			return true
		}
	}
	return false
}

func aliases(raw string) []string {
	base := NormalizeModel(raw)
	if base == "" {
		return nil
	}
	out := []string{base}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	if index := strings.LastIndexByte(base, '/'); index >= 0 {
		add(base[index+1:])
	}
	if index := strings.LastIndexByte(base, '@'); index > 0 {
		add(base[:index])
	}
	if index := strings.LastIndexByte(base, ':'); index > 0 {
		add(base[:index])
	}
	add(stripVendorDots(base))

	initial := len(out)
	for index := 0; index < initial; index++ {
		value := out[index]
		if stripped := stripDateSuffix(value); stripped != value {
			add(stripped)
		}
		if stripped := stripVersionSuffix(value); stripped != value {
			add(stripped)
		}
		if stripped := stripDateSuffix(stripVersionSuffix(value)); stripped != value {
			add(stripped)
		}
	}
	add(baseModelName(base))
	return dedupeStrings(out)
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func dedupeStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	out := values[:0]
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func isDelim(token json.Token, want json.Delim) bool {
	delim, ok := token.(json.Delim)
	return ok && delim == want
}

// skipValue consumes one JSON value without retaining it. The value's opening
// token has not been read yet, so scalars complete after one token and objects
// or arrays are followed until their matching close.
func skipValue(decoder *json.Decoder) error {
	depth := 0
	for {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			if depth == 0 {
				return nil
			}
			continue
		}
		switch delim {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return nil
			}
		}
	}
}
