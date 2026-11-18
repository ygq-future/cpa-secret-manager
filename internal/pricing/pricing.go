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
	"sort"
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
	// maxOffers bounds the per-model offer list kept in the cache and returned
	// to the page. Prefix matching can reach a broad family, so the list is
	// ranked first and truncated last.
	maxOffers = 50
)

// Offer is one catalog listing that matched a requested model. Prices are USD
// per 1M tokens, the same unit the plugin's ModelPrice uses.
type Offer struct {
	// Model is the upstream model identifier the price came from.
	Model string `json:"model"`
	// Provider is the upstream provider identifier, for example "openai".
	Provider  string  `json:"provider"`
	Input     float64 `json:"input"`
	Output    float64 `json:"output"`
	Cache     float64 `json:"cache"`
	UpdatedAt string  `json:"updated_at,omitempty"`
}

// Suggestion is the reference price for one requested model, resolved from the
// provider offers found in the catalog.
type Suggestion struct {
	// Model is the upstream model identifier of the best offer.
	Model string `json:"model"`
	// Provider is the upstream provider identifier of the best offer.
	Provider  string  `json:"provider"`
	Input     float64 `json:"input"`
	Output    float64 `json:"output"`
	Cache     float64 `json:"cache"`
	UpdatedAt string  `json:"updated_at,omitempty"`
	// Providers is how many distinct offers are listed for this model.
	Providers int `json:"providers,omitempty"`
	// Offers lists every matched listing, best first. The first entry is the
	// price the page selects by default; the leading fields mirror it.
	Offers []Offer `json:"offers,omitempty"`
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

	index := newWantedIndex(normalized)
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
	prefixOfWanted bool
	depth          int
	// wantedSegments is the segment count of the indexed wanted name that
	// produced this offer; depth == wantedSegments means the whole suffix
	// matched rather than a leading family token.
	wantedSegments int
	extra          int
}

func decodeProvider(decoder *json.Decoder, provider string, index wantedIndex, out map[string][]candidate) error {
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

func decodeModels(decoder *json.Decoder, provider string, index wantedIndex, out map[string][]candidate) error {
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
		base := modelBase(modelID)
		segments := splitSegments(base)
		for _, match := range matched {
			out[match.name] = append(out[match.name], candidate{
				provider:       provider,
				modelID:        entry.ID,
				updated:        entry.LastUpdated,
				input:          price.input,
				output:         price.output,
				cache:          price.cache,
				hasPrice:       price.hasPrice,
				hasCache:       price.hasCache,
				exact:          base == match.base,
				canonicalMatch: index.matchesName(match.name, entry.CanonicalModelID),
				prefixOfWanted: isStrictPrefix(match.base, base),
				depth:          match.depth,
				wantedSegments: len(splitSegments(match.base)),
				extra:          len(segments) - match.depth,
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
		// A lone family token is too weak: it would map "claude-3-5-haiku" onto
		// every Claude listing. An offer is confident when it matched at least
		// two segments or the whole indexed suffix — the latter keeps names with
		// a skipped vendor prefix (e.g. "wb-hy4flash") working, because their
		// remaining suffix is matched in full.
		accepted := make([]candidate, 0, len(offers))
		maxDepth := 0
		for _, offer := range offers {
			if offer.depth < 2 && offer.depth < offer.wantedSegments {
				continue
			}
			accepted = append(accepted, offer)
			if offer.depth > maxDepth {
				maxDepth = offer.depth
			}
		}
		if len(accepted) == 0 {
			continue
		}
		// Keep only the deepest prefix stage that produced matches. A model name
		// such as "gemini-3.5-flash-high" falls back to "gemini-3.5-flash" when
		// no catalog listing carries the trailing "high" marker.
		matched := make([]candidate, 0, len(accepted))
		for _, offer := range accepted {
			if offer.depth == maxDepth {
				matched = append(matched, offer)
			}
		}
		sort.SliceStable(matched, func(left, right int) bool {
			return better(matched[left], matched[right])
		})

		best := matched[0]
		list := make([]Offer, 0, len(matched))
		seen := make(map[string]struct{}, len(matched))
		for _, offer := range matched {
			key := offer.provider + "\x00" + offer.modelID
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			list = append(list, Offer{
				Model:     offer.modelID,
				Provider:  offer.provider,
				Input:     roundPrice(offer.input),
				Output:    roundPrice(offer.output),
				Cache:     roundPrice(offer.cache),
				UpdatedAt: offer.updated,
			})
			if len(list) >= maxOffers {
				break
			}
		}

		out[NormalizeModel(wanted)] = Suggestion{
			Model:     best.modelID,
			Provider:  best.provider,
			Input:     roundPrice(best.input),
			Output:    roundPrice(best.output),
			Cache:     roundPrice(best.cache),
			UpdatedAt: best.updated,
			Providers: len(list),
			Offers:    list,
		}
	}
	return out
}

func better(candidate, current candidate) bool {
	candidateScore := score(candidate)
	currentScore := score(current)
	if candidateScore != currentScore {
		return candidateScore > currentScore
	}
	if candidate.extra != current.extra {
		return candidate.extra < current.extra
	}
	if candidate.updated != current.updated {
		return candidate.updated > current.updated
	}
	if candidate.provider != current.provider {
		return candidate.provider < current.provider
	}
	return candidate.modelID < current.modelID
}

func score(candidate candidate) int {
	total := providerRank(candidate.provider, candidate.modelID)
	if candidate.exact {
		total += 100
	}
	if candidate.prefixOfWanted {
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

// modelBase reduces a model identifier to the spelling used for prefix
// matching: the last path segment, without a provider tag, vendor dots or a
// date marker. Underscores are treated as separators so both naming styles
// compare the same way.
func modelBase(raw string) string {
	value := NormalizeModel(raw)
	if index := strings.LastIndexByte(value, '/'); index >= 0 {
		value = value[index+1:]
	}
	if index := strings.IndexByte(value, '@'); index > 0 {
		value = value[:index]
	}
	if index := strings.LastIndexByte(value, ':'); index > 0 {
		value = value[:index]
	}
	value = stripVendorDots(value)
	return strings.ReplaceAll(value, "_", "-")
}

func splitSegments(value string) []string {
	if value == "" {
		return nil
	}
	return strings.FieldsFunc(value, func(char rune) bool {
		return char == '-' || char == '_' || char == ' '
	})
}

// hasBoundaryPrefix reports whether value starts with prefix and the prefix
// ends on a segment boundary, so "gemini-3.5-flash" does not match
// "gemini-3.5-flashlight".
func hasBoundaryPrefix(value, prefix string) bool {
	if prefix == "" || !strings.HasPrefix(value, prefix) {
		return false
	}
	if len(value) == len(prefix) {
		return true
	}
	return isSegmentBoundary(value[len(prefix)])
}

func isSegmentBoundary(b byte) bool {
	switch b {
	case '-', '_', ' ', '@', ':':
		return true
	}
	return false
}

func isSegmentDelimiter(b byte) bool {
	return b == '-' || b == '_' || b == ' '
}

// matchCompactPrefix reports whether candidate starts with wantedCompact when
// segment delimiters ('-', '_', ' ') are ignored, while still requiring the
// match to end on a segment boundary in candidate.
func matchCompactPrefix(candidate, wantedCompact string) bool {
	if wantedCompact == "" {
		return false
	}
	ci := 0
	wi := 0
	for ci < len(candidate) && wi < len(wantedCompact) {
		if isSegmentDelimiter(candidate[ci]) {
			ci++
			continue
		}
		if candidate[ci] != wantedCompact[wi] {
			return false
		}
		ci++
		wi++
	}
	if wi < len(wantedCompact) {
		return false
	}
	return ci == len(candidate) || isSegmentBoundary(candidate[ci])
}

// isStrictPrefix reports whether candidate is a shorter form of wanted, for
// example "gemini-3.5-flash" for "gemini-3.5-flash-high".
func isStrictPrefix(wanted, candidate string) bool {
	return candidate != "" && candidate != wanted && hasBoundaryPrefix(wanted, candidate)
}

// prefixDepth returns how many leading segments of wanted match the candidate
// at a segment boundary, or zero when even the first segment differs.
func prefixDepth(candidate string, wantedSegments []string) int {
	for depth := len(wantedSegments); depth >= 1; depth-- {
		if hasBoundaryPrefix(candidate, strings.Join(wantedSegments[:depth], "-")) ||
			matchCompactPrefix(candidate, strings.Join(wantedSegments[:depth], "")) {
			return depth
		}
	}
	return 0
}

func compactHead(value string) string {
	var head [2]byte
	n := 0
	for i := 0; i < len(value) && n < 2; i++ {
		if isSegmentDelimiter(value[i]) {
			continue
		}
		head[n] = value[i]
		n++
	}
	return string(head[:n])
}

// wantedIndex maps the requested models by their first segment and compact
// two-byte head. Indexing every suffix allows skipping vendor prefixes (e.g.
// "wb-gemini-flash" -> "gemini-flash") from the first matching prefix onward.
type wantedIndex struct {
	byHead map[string][]wantedEntry
}

type wantedEntry struct {
	name     string
	base     string
	segments []string
}

type wantedMatch struct {
	name  string
	base  string
	depth int
}

func newWantedIndex(wanted []string) wantedIndex {
	index := wantedIndex{byHead: make(map[string][]wantedEntry, len(wanted)*2)}
	for _, name := range wanted {
		base := modelBase(name)
		segments := splitSegments(base)
		for skip := range len(segments) {
			subSegments := segments[skip:]
			subBase := strings.Join(subSegments, "-")
			head := compactHead(subBase)
			if head == "" {
				continue
			}
			index.byHead[head] = append(index.byHead[head], wantedEntry{
				name:     name,
				base:     subBase,
				segments: subSegments,
			})
		}
	}
	return index
}

func (index wantedIndex) match(modelID string) []wantedMatch {
	base := modelBase(modelID)
	head := compactHead(base)
	if head == "" {
		return nil
	}
	var matches []wantedMatch
	seen := make(map[string]struct{})
	for _, entry := range index.byHead[head] {
		depth := prefixDepth(base, entry.segments)
		if depth == 0 {
			continue
		}
		if _, exists := seen[entry.name]; exists {
			// Prefer the earliest matching prefix (smallest skip).
			continue
		}
		seen[entry.name] = struct{}{}
		matches = append(matches, wantedMatch{name: entry.name, base: entry.base, depth: depth})
	}
	return matches
}

func (index wantedIndex) matchesName(name string, modelID string) bool {
	if modelID == "" {
		return false
	}
	base := modelBase(modelID)
	head := compactHead(base)
	if head == "" {
		return false
	}
	for _, entry := range index.byHead[head] {
		if entry.name != name {
			continue
		}
		if prefixDepth(base, entry.segments) > 0 {
			return true
		}
	}
	return false
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
