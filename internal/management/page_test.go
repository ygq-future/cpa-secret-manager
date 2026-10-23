package management

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"cpa-secret-manager/internal/version"
)

var (
	styleBlockPattern       = regexp.MustCompile(`(?s)<style>(.*?)</style>`)
	scriptBlockPattern      = regexp.MustCompile(`(?s)<script>(.*?)</script>`)
	bodyMarkupPattern       = regexp.MustCompile(`(?s)<body>(.*?)<script>`)
	inlineStyleAssign       = regexp.MustCompile(`\.style\.[A-Za-z-]+\s*=`)
	colorLiteralPattern     = regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b|rgba?\([^)]*\)`)
	tokenDeclarationPattern = regexp.MustCompile(`^\s*--[a-z0-9-]+\s*:`)
	refreshSpinnerPattern   = regexp.MustCompile(`(?s)<button id="refresh-button".*?<span class="spinner"`)
	retentionSavePattern    = regexp.MustCompile(`(?s)<button id="settings-retention-save"[^>]*disabled[^>]*>.*?<span class="spinner"`)
	sharedTagMetricsPattern = regexp.MustCompile(`(?s)\.badge-remark,\s*\.badge-quota,\s*\.badge-reset,\s*\.badge-models \{(.*?)\}`)
	rootBlockPattern        = regexp.MustCompile(`(?s):root\s*\{(.*?)\}`)
	whiteBlockPattern       = regexp.MustCompile(`(?s)\[data-theme='white'\]\s*\{(.*?)\}`)
	darkBlockPattern        = regexp.MustCompile(`(?s)\[data-theme='dark'\]\s*\{(.*?)\}`)
	tokenPattern            = regexp.MustCompile(`(--[a-z0-9-]+)\s*:`)
	referencePattern        = regexp.MustCompile(`var\(\s*(--[a-z0-9-]+)`)
	hostVarPattern          = regexp.MustCompile(`'(--[a-z0-9-]+)'`)
	translationPattern      = regexp.MustCompile(`data-i18n[a-z-]*="([^"]+)"`)
	callPattern             = regexp.MustCompile(`\btf?\('([a-z0-9_.]+)'`)
	domIdPattern            = regexp.MustCompile(`byId\('([^']+)'\)`)
	bindingPattern          = regexp.MustCompile(`bind(?:Click|Submit|Backdrop)\('([^']+)'`)
	markupIDPattern         = regexp.MustCompile(`id="([^"]+)"`)
	inlineHandler           = regexp.MustCompile(`\son[a-z]+\s*=\s*"`)
	remoteURLPattern        = regexp.MustCompile(`https?://`)
)

func extractPageStyle(t *testing.T) string {
	t.Helper()
	match := styleBlockPattern.FindStringSubmatch(PageHTML)
	if match == nil {
		t.Fatal("page has no <style> block")
	}
	return match[1]
}

func extractPageScript(t *testing.T) string {
	t.Helper()
	match := scriptBlockPattern.FindStringSubmatch(PageHTML)
	if match == nil {
		t.Fatal("page has no <script> block")
	}
	return match[1]
}

// extractBodyMarkup returns the markup between <body> and the single <script>
// block: the DOM the browser actually parses, without CSS or JS comments.
func extractBodyMarkup(t *testing.T) string {
	t.Helper()
	match := bodyMarkupPattern.FindStringSubmatch(PageHTML)
	if match == nil {
		t.Fatal("page has no body markup block")
	}
	return match[1]
}

func extractTokens(t *testing.T, pattern *regexp.Regexp, source string, label string) []string {
	t.Helper()
	match := pattern.FindStringSubmatch(source)
	if match == nil {
		t.Fatalf("page has no %s block", label)
	}
	seen := map[string]bool{}
	var names []string
	for _, token := range tokenPattern.FindAllStringSubmatch(match[1], -1) {
		if seen[token[1]] {
			continue
		}
		seen[token[1]] = true
		names = append(names, token[1])
	}
	sort.Strings(names)
	return names
}

func TestPage_ServesEmbeddedMarkup(t *testing.T) {
	handler := NewHandler(&stubBackend{})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v0/resource/plugins/"+PluginID+PathPage, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("content type = %q, want text/html", contentType)
	}
	body := recorder.Body.String()
	if !strings.HasPrefix(body, "<!DOCTYPE html>") {
		t.Fatal("page does not start with a doctype")
	}
	if strings.Count(body, "<script>") != 1 || strings.Count(body, "</script>") != 1 {
		t.Fatalf("page has %d script blocks, want exactly 1", strings.Count(body, "<script>"))
	}
	if !strings.Contains(body, "v"+version.PluginVersion) {
		t.Fatalf("page is missing the version badge for %s", version.PluginVersion)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, PathPage, nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST page status = %d, want 405", recorder.Code)
	}
}

// TestPage_HasNoRemoteResources keeps the page offline-capable and avoids
// leaking navigation to third parties.
func TestPage_HasNoRemoteResources(t *testing.T) {
	stripped := strings.ReplaceAll(PageHTML, "https://github.com/ygq-future/cpa-secret-manager", "")
	// The SVG namespace is an XML identifier handed to createElementNS, never a
	// fetch target; the page ships its own <style> and <script>.
	stripped = strings.ReplaceAll(stripped, "http://www.w3.org/2000/svg", "")
	stripped = strings.ReplaceAll(stripped, "http://www.w3.org/1999/xlink", "")
	if matches := remoteURLPattern.FindAllString(stripped, -1); len(matches) > 0 {
		t.Fatalf("page references remote resources: %v", matches)
	}
}

func TestPage_HasNoInlineHandlers(t *testing.T) {
	if matches := inlineHandler.FindAllString(PageHTML, -1); len(matches) > 0 {
		t.Fatalf("page uses inline event handlers %v; bind events from the single script scope instead", matches)
	}
}

// TestPageThemeTokensAreComplete keeps the three theme blocks in lockstep so a
// host theme switch never leaves an undefined variable behind.
func TestPageThemeTokensAreComplete(t *testing.T) {
	style := extractPageStyle(t)
	rootTokens := extractTokens(t, rootBlockPattern, style, ":root")
	if len(rootTokens) < 20 {
		t.Fatalf("only %d tokens declared in :root; the CPA theme surface looks truncated", len(rootTokens))
	}

	for label, pattern := range map[string]*regexp.Regexp{
		"white": whiteBlockPattern,
		"dark":  darkBlockPattern,
	} {
		blockTokens := extractTokens(t, pattern, style, label)
		declared := map[string]bool{}
		for _, name := range blockTokens {
			declared[name] = true
		}
		for _, name := range rootTokens {
			if !declared[name] {
				t.Fatalf("token %s is declared in :root but missing from the %s theme block", name, label)
			}
		}
	}
}

// TestPage_ReferencedVariablesAreDefined keeps the stylesheet free of undefined
// token references.
func TestPage_ReferencedVariablesAreDefined(t *testing.T) {
	style := extractPageStyle(t)
	defined := map[string]bool{}
	for _, name := range extractTokens(t, rootBlockPattern, style, ":root") {
		defined[name] = true
	}
	for _, reference := range referencePattern.FindAllStringSubmatch(style, -1) {
		if !defined[reference[1]] {
			t.Fatalf("style references undefined variable %s", reference[1])
		}
	}
}

// TestPage_RulesUseTokensOnly enforces the palette contract from ADR-0003: a
// colour literal may only appear in a custom-property declaration, never inside
// a rule, so a host theme can repaint every surface and a component cannot fork
// the palette by accident.
func TestPage_RulesUseTokensOnly(t *testing.T) {
	for index, line := range strings.Split(extractPageStyle(t), "\n") {
		if !colorLiteralPattern.MatchString(line) || tokenDeclarationPattern.MatchString(line) {
			continue
		}
		t.Fatalf("style line %d hardcodes a colour (%s); declare a token instead", index+1, strings.TrimSpace(line))
	}
}

func TestPage_HostThemeVariablesMatchTokens(t *testing.T) {
	script := extractPageScript(t)
	match := regexp.MustCompile(`(?s)HOST_THEME_VARIABLES = \[(.*?)\];`).FindStringSubmatch(script)
	if match == nil {
		t.Fatal("page script does not declare HOST_THEME_VARIABLES")
	}

	declared := map[string]bool{}
	for _, name := range extractTokens(t, rootBlockPattern, extractPageStyle(t), ":root") {
		declared[name] = true
	}
	mirrored := hostVarPattern.FindAllStringSubmatch(match[1], -1)
	if len(mirrored) == 0 {
		t.Fatal("HOST_THEME_VARIABLES is empty")
	}
	for _, item := range mirrored {
		if !declared[item[1]] {
			t.Fatalf("host theme variable %s is copied but never declared as a token", item[1])
		}
	}
}

func TestPage_TranslationsAreComplete(t *testing.T) {
	script := extractPageScript(t)
	dictionaries := extractDictionaries(t, script)
	if len(dictionaries) != 2 {
		t.Fatalf("found %d dictionaries, want 2", len(dictionaries))
	}

	languages := make([]string, 0, len(dictionaries))
	for language := range dictionaries {
		languages = append(languages, language)
	}
	sort.Strings(languages)

	used := map[string]bool{}
	for _, key := range translationPattern.FindAllStringSubmatch(PageHTML, -1) {
		used[key[1]] = true
	}
	for _, key := range callPattern.FindAllStringSubmatch(script, -1) {
		used[key[1]] = true
	}
	if len(used) < 30 {
		t.Fatalf("only %d translation keys detected; the page translation scan looks broken", len(used))
	}

	for _, language := range languages {
		if len(dictionaries[language]) < 40 {
			t.Fatalf("dictionary %s has only %d keys; the dictionary scan looks broken", language, len(dictionaries[language]))
		}
		for key := range used {
			if !dictionaries[language][key] {
				t.Fatalf("translation key %q is missing from %s", key, language)
			}
		}
	}

	reference := dictionaries[languages[0]]
	for _, language := range languages[1:] {
		for key := range dictionaries[language] {
			if !reference[key] {
				t.Fatalf("translation key %q exists in %s but not in %s", key, language, languages[0])
			}
		}
		for key := range reference {
			if !dictionaries[language][key] {
				t.Fatalf("translation key %q exists in %s but not in %s", key, languages[0], language)
			}
		}
	}
}

func TestPage_DomReferencesResolve(t *testing.T) {
	present := map[string]bool{}
	for _, id := range markupIDPattern.FindAllStringSubmatch(PageHTML, -1) {
		present[id[1]] = true
	}

	script := extractPageScript(t)
	references := domIdPattern.FindAllStringSubmatch(script, -1)
	bindings := bindingPattern.FindAllStringSubmatch(script, -1)
	if len(references)+len(bindings) < 15 {
		t.Fatalf("only %d DOM lookups detected; the reference scan looks broken", len(references)+len(bindings))
	}
	for _, reference := range references {
		if !present[reference[1]] {
			t.Fatalf("script reads #%s but the page declares no such element", reference[1])
		}
	}
	for _, reference := range bindings {
		if !present[reference[1]] {
			t.Fatalf("script binds #%s but the page declares no such element", reference[1])
		}
	}
}

// TestPage_UsesHostManagementAPI pins the wiring to the host contract: keys are
// owned by the host management API, metadata by the plugin routes.
func TestPage_UsesHostManagementAPI(t *testing.T) {
	for _, fragment := range []string{
		"'/v0/management'",
		"KEYS_PATH",
		"RESOLVE_PATH",
		"REMARKS_PATH",
		"GENERATE_PATH",
		"Authorization",
		"X-Management-Key",
		"cli-proxy-auth",
		"enc::v1::",
		"cli-proxy-theme",
	} {
		if !strings.Contains(PageHTML, fragment) {
			t.Fatalf("page is missing %q", fragment)
		}
	}
}

func TestPage_JavaScriptSyntax(t *testing.T) {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is unavailable; skipping the JavaScript syntax gate")
	}

	scriptPath := filepath.Join(t.TempDir(), "page.js")
	if err := os.WriteFile(scriptPath, []byte(extractPageScript(t)), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}
	if output, err := exec.Command(nodePath, "--check", scriptPath).CombinedOutput(); err != nil {
		t.Fatalf("node --check failed: %v\n%s", err, output)
	}
}

func TestPage_HasKeysTableLoadingAndDefaultExpandedContracts(t *testing.T) {
	if !strings.Contains(PageHTML, `id="keys-loading"`) {
		t.Fatal("PageHTML is missing #keys-loading element")
	}
	if !strings.Contains(PageHTML, `id="keys-table-wrap"`) {
		t.Fatal("PageHTML is missing #keys-table-wrap element")
	}
	if !strings.Contains(PageHTML, `class="spinner"`) {
		t.Fatal("PageHTML is missing the shared spinner component")
	}
	if !strings.Contains(PageHTML, `class="table-loading"`) {
		t.Fatal("PageHTML is missing the shared table mask component")
	}

	script := extractPageScript(t)
	for _, fn := range []string{"isModelsExpanded", "setKeysLoading"} {
		if !strings.Contains(script, "function "+fn) {
			t.Fatalf("page script missing function %s", fn)
		}
	}
	if !strings.Contains(PageHTML, ".badge-remark") {
		t.Fatal("PageHTML is missing .badge-remark style definition")
	}
	if !strings.Contains(PageHTML, "key-row.is-expanded") {
		t.Fatal("PageHTML is missing key-row.is-expanded style definition")
	}

	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed; skipping runtime JS evaluation test")
	}
	fnMatch := regexp.MustCompile(`(?s)function isModelsExpanded\(.*?\n\}`).FindString(script)
	if fnMatch == "" {
		t.Fatal("cannot find function isModelsExpanded in script")
	}
	testRunner := `
var state = { expanded: {}, keys: ['k1', 'k2', 'k3'], entries: [
  { usage: { models: [{ model: 'm1' }] } },
  { usage: { models: [{ model: 'm1' }, { model: 'm2' }] } },
  { usage: { models: [] } }
] };
` + fnMatch + `
if (isModelsExpanded('k1', [{ model: 'm1' }]) !== false) {
  console.error('k1 single model should not be expanded by default');
  process.exit(1);
}
if (isModelsExpanded('k2', [{ model: 'm1' }, { model: 'm2' }]) !== true) {
  console.error('k2 multi models should be expanded by default');
  process.exit(1);
}
if (isModelsExpanded('k3', []) !== false) {
  console.error('k3 empty models should not be expanded by default');
  process.exit(1);
}
if (isModelsExpanded('k1') !== false) {
  console.error('k1 fallback lookup should not be expanded');
  process.exit(1);
}
if (isModelsExpanded('k2') !== true) {
  console.error('k2 fallback lookup should be expanded');
  process.exit(1);
}
state.expanded['k1'] = true;
if (isModelsExpanded('k1', [{ model: 'm1' }]) !== true) {
  console.error('k1 should respect state.expanded override to true');
  process.exit(1);
}
state.expanded['k2'] = false;
if (isModelsExpanded('k2', [{ model: 'm1' }, { model: 'm2' }]) !== false) {
  console.error('k2 should respect state.expanded override to false');
  process.exit(1);
}
`
	testScriptPath := filepath.Join(t.TempDir(), "test_expanded.js")
	if err := os.WriteFile(testScriptPath, []byte(testRunner), 0o600); err != nil {
		t.Fatalf("write test runner: %v", err)
	}
	cmd := exec.Command(nodePath, testScriptPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("node test failed: %v\n%s", err, string(out))
	}
}

func TestPage_KeyMutationsTriggerKeysLoading(t *testing.T) {
	script := extractPageScript(t)
	saveKeyFormMatch := regexp.MustCompile(`(?s)function saveKeyForm\(\)\s*\{.*?\n\}`).FindString(script)
	if saveKeyFormMatch == "" {
		t.Fatal("saveKeyForm function not found")
	}
	if count := strings.Count(saveKeyFormMatch, "setKeysLoading(true)"); count < 2 {
		t.Fatalf("saveKeyForm should call setKeysLoading(true) in both edit and add branches, found %d", count)
	}
	confirmDeleteMatch := regexp.MustCompile(`(?s)function confirmDeleteKey\(.*?\)\s*\{.*?\n\}`).FindString(script)
	if confirmDeleteMatch == "" {
		t.Fatal("confirmDeleteKey function not found")
	}
	if !strings.Contains(confirmDeleteMatch, "setKeysLoading(true)") {
		t.Fatal("confirmDeleteKey should call setKeysLoading(true)")
	}
}

// TestPage_ModernQuotaComponentsContract pins the quota form to the shared
// component layer: a toggle switch, a listbox-backed unit input, preset pills
// and the chip grid all come from ComponentsCSS/ComponentsScript.
func TestPage_ModernQuotaComponentsContract(t *testing.T) {
	if !strings.Contains(PageHTML, `id="key-form-quota-enabled"`) {
		t.Fatal("key-form-quota-enabled should be rendered by the shared toggle switch")
	}

	if !strings.Contains(PageHTML, `class="input-unit-group"`) || !strings.Contains(PageHTML, `id="key-form-quota-limit-unit"`) {
		t.Fatal("quota limit should use .input-unit-group with the shared unit listbox")
	}
	script := extractPageScript(t)
	for _, unit := range []string{"'1000'", "'1000000'", "'1000000000'"} {
		if !strings.Contains(script, "value: "+unit) {
			t.Fatalf("quota unit listbox missing option %s", unit)
		}
	}

	if !strings.Contains(PageHTML, `id="key-form-quota-limits-row"`) {
		t.Fatal("quota token limit and cost limit should sit on the same line")
	}
	// The limit field is manual input: shortcuts and the resolved calculation
	// hint were dropped by design per user preference.
	for _, dropped := range []string{"data-duration=", "quota-duration-calc-hint", "quota-calc-inline", "data-val="} {
		if strings.Contains(PageHTML, dropped) {
			t.Fatalf("quota form should no longer carry %s", dropped)
		}
	}
	if !strings.Contains(PageHTML, `id="key-form-models-container"`) || !strings.Contains(templateScriptComponents, "function mountMultiSelect") {
		t.Fatal("allowed models should be rendered by the shared searchable multi-select component")
	}

	for _, dropped := range []string{"key-form-quota-status", "key-form-chip-field"} {
		if strings.Contains(PageHTML, dropped) {
			t.Fatalf("key form should no longer carry %s", dropped)
		}
	}

	style := extractPageStyle(t)
	if !strings.Contains(style, ".chip-check") || !strings.Contains(style, "max-width: 0") || !strings.Contains(style, ".chip.is-selected .chip-check") {
		t.Fatal("the chip check icon should collapse to zero width when unselected to keep text centered")
	}

	if !strings.Contains(PageHTML, `id="key-form-quota-duration-unit"`) || !strings.Contains(PageHTML, `id="key-form-quota-duration-sec"`) {
		t.Fatal("window duration should use the shared listbox with a resolved-seconds field")
	}
	for _, unit := range []string{"'60'", "'3600'", "'86400'"} {
		if !strings.Contains(script, "value: "+unit) {
			t.Fatalf("duration unit listbox missing option %s", unit)
		}
	}

	if !strings.Contains(templateStyleBase, `-webkit-outer-spin-button`) {
		t.Fatal("the base layer should hide the native number spin arrows")
	}
}

// TestPage_RefreshShowsLoadingState pins the refresh feedback: the shell button
// carries the shared spinner, the runtime toggles .is-loading on it, and the
// table mask covers the first paint and every explicit refresh.
func TestPage_RefreshShowsLoadingState(t *testing.T) {
	if !refreshSpinnerPattern.MatchString(PageHTML) {
		t.Fatal("the refresh button should carry the shared spinner")
	}
	if !strings.Contains(extractPageStyle(t), ".btn.is-loading .spinner") {
		t.Fatal("the component layer should reveal the button spinner while loading")
	}

	script := extractPageScript(t)
	if !strings.Contains(script, "function setRefreshLoading") {
		t.Fatal("page script is missing setRefreshLoading")
	}
	refreshAll := regexp.MustCompile(`(?s)function refreshAll\(options\)\s*\{.*?\n\}`).FindString(script)
	if refreshAll == "" {
		t.Fatal("refreshAll function not found")
	}
	for _, call := range []string{"setRefreshLoading(true)", "setRefreshLoading(false)", "setKeysLoading(true)"} {
		if !strings.Contains(refreshAll, call) {
			t.Fatalf("refreshAll should call %s so a refresh is always visible", call)
		}
	}
}

// TestPage_EventDrivenAndUsageIntervalContract ensures that silent auto-refresh
// only polls resolve (refreshUsage), while key mutations use event-driven refreshKeys
// and do not poll settings.
func TestPage_EventDrivenAndUsageIntervalContract(t *testing.T) {
	script := extractPageScript(t)

	autoRefresh := regexp.MustCompile(`(?s)function startAutoRefresh\(\)\s*\{.*?\n\}`).FindString(script)
	if autoRefresh == "" {
		t.Fatal("startAutoRefresh function not found")
	}
	if !strings.Contains(autoRefresh, "refreshUsage({ silent: true })") {
		t.Fatalf("startAutoRefresh should only invoke refreshUsage, got:\n%s", autoRefresh)
	}
	if strings.Contains(autoRefresh, "refreshAll") {
		t.Fatalf("startAutoRefresh should not invoke refreshAll directly:\n%s", autoRefresh)
	}

	refreshUsage := regexp.MustCompile(`(?s)function refreshUsage\(options\)\s*\{.*?\n\}`).FindString(script)
	if refreshUsage == "" {
		t.Fatal("refreshUsage function not found")
	}
	if !strings.Contains(refreshUsage, "loadEntries(false)") {
		t.Fatal("refreshUsage must query loadEntries to resolve usage metrics")
	}
	if strings.Contains(refreshUsage, "loadSettings") || strings.Contains(refreshUsage, "loadKeys") {
		t.Fatal("refreshUsage must never poll settings or host keys")
	}

	refreshKeys := regexp.MustCompile(`(?s)function refreshKeys\(options\)\s*\{.*?\n\}`).FindString(script)
	if refreshKeys == "" {
		t.Fatal("refreshKeys function not found")
	}
	if !strings.Contains(refreshKeys, "loadKeys()") {
		t.Fatal("refreshKeys must call loadKeys to update keys list and usage")
	}
	if strings.Contains(refreshKeys, "loadSettings") {
		t.Fatal("refreshKeys must not poll settings")
	}
}

// TestPage_RetentionSaveFollowsTheField pins the retention action contract: it
// starts disabled, enables only when the field diverges from the persisted
// value, and owns the shared spinner while the PUT is in flight.
func TestPage_RetentionSaveFollowsTheField(t *testing.T) {
	if !retentionSavePattern.MatchString(PageHTML) {
		t.Fatal("the retention save button should start disabled and carry the shared spinner")
	}

	script := extractPageScript(t)
	for _, fragment := range []string{
		"function syncRetentionSaveState",
		"function setRetentionSaving",
		"button.disabled = entered === null || saved === null || entered === saved;",
		"button.classList.toggle('is-loading', saving);",
		"retInput.addEventListener('input', syncRetentionSaveState);",
	} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("retention save wiring is missing %q", fragment)
		}
	}

	saveFn := regexp.MustCompile(`(?s)function saveRetentionDays\(\)\s*\{.*?\n\}`).FindString(script)
	if saveFn == "" {
		t.Fatal("saveRetentionDays function not found")
	}
	for _, call := range []string{"setRetentionSaving(true)", "setRetentionSaving(false)"} {
		if !strings.Contains(saveFn, call) {
			t.Fatalf("saveRetentionDays should call %s", call)
		}
	}
}

// TestPage_SearchClearCentresItsGlyph guards the reported regression: the
// leading glyph rule was a descendant selector, so it also absolutely
// positioned the clear button's cross and pushed it out of the button.
func TestPage_SearchClearCentresItsGlyph(t *testing.T) {
	style := extractPageStyle(t)
	if !strings.Contains(style, ".search > .icon {") {
		t.Fatal("only the leading search glyph should be absolutely positioned")
	}
	if strings.Contains(style, ".search .icon {") {
		t.Fatal("the descendant selector would reposition the clear button glyph")
	}
	if !strings.Contains(style, ".search-clear .icon {") || !regexp.MustCompile(`(?s)\.search-clear \.icon \{.*?position: static;`).MatchString(style) {
		t.Fatal("the clear button glyph should stay in its flex box")
	}
}

// TestPage_ResistsPasswordManagerAutofill pins the reported fix: the key
// dialog is not advertised as a savable credential, and the filter starts
// read-only so the password manager has nothing to populate on load. The guard
// is released on first interaction instead of clearing the field on a timer.
func TestPage_ResistsPasswordManagerAutofill(t *testing.T) {
	filterTag := regexp.MustCompile(`<input id="keys-filter"[^>]*>`).FindString(PageHTML)
	if filterTag == "" {
		t.Fatal("keys filter input not found")
	}
	for _, attr := range []string{`name="keys-filter-query"`, `autocomplete="off"`, `readonly`, `data-lpignore="true"`, `data-1p-ignore`, `data-bwignore`, `spellcheck="false"`} {
		if !strings.Contains(filterTag, attr) {
			t.Fatalf("keys filter should carry %s to resist autofill", attr)
		}
	}

	keyTag := regexp.MustCompile(`<input id="key-modal-input"[^>]*>`).FindString(PageHTML)
	for _, attr := range []string{`autocomplete="new-password"`, `data-lpignore="true"`, `data-1p-ignore`, `data-bwignore`} {
		if !strings.Contains(keyTag, attr) {
			t.Fatalf("management key input should carry %s so the page is not treated as a login form", attr)
		}
	}

	formTag := regexp.MustCompile(`<form id="key-form"[^>]*>`).FindString(PageHTML)
	for _, attr := range []string{`autocomplete="off"`, `data-bwignore="true"`, `data-lpignore="true"`, `data-1p-ignore`} {
		if !strings.Contains(formTag, attr) {
			t.Fatalf("key-form should carry %s to prevent password managers from latching on", attr)
		}
	}

	remarkTag := regexp.MustCompile(`<textarea id="key-form-remark"[^>]*>`).FindString(PageHTML)
	for _, attr := range []string{`data-bwignore="true"`, `data-lpignore="true"`, `data-1p-ignore`} {
		if !strings.Contains(remarkTag, attr) {
			t.Fatalf("key-form remark textarea should carry %s to prevent password managers from latching on", attr)
		}
	}

	script := extractPageScript(t)
	for _, fragment := range []string{"var unlockFilter = function", "filter.addEventListener('focus', unlockFilter)", "filter.addEventListener('pointerdown', unlockFilter)"} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("filter unlock wiring is missing %q", fragment)
		}
	}
	if strings.Contains(script, "clearAutofilledFilter") {
		t.Fatal("the filter must not be cleared on a timer after load")
	}

	style := extractPageStyle(t)
	if !regexp.MustCompile(`(?s)\.search input\[readonly\] \{.*?background: var\(--bg-card\)`).MatchString(style) {
		t.Fatal("the read-only filter must keep its normal background")
	}
}

// TestPage_SwitchTokensAreDeclaredEverywhere guards the reported regression: the
// off-state track was --bg-subtle, which vanished on the --bg-subtle quota
// header, leaving only the knob visible.
func TestPage_SwitchTokensAreDeclaredEverywhere(t *testing.T) {
	style := extractPageStyle(t)
	for _, token := range []string{"--switch-knob:", "--switch-track:", "--switch-track-border:"} {
		if count := strings.Count(style, token); count != 4 {
			t.Fatalf("token %s is declared %d times; every theme block needs it", token, count)
		}
	}

	track := regexp.MustCompile(`(?s)\.toggle-slider \{(.*?)\}`).FindStringSubmatch(style)
	if track == nil {
		t.Fatal("the component layer declares no .toggle-slider rule")
	}
	if !strings.Contains(track[1], "var(--switch-track)") {
		t.Fatal(".toggle-slider must paint the dedicated switch track, not a page surface")
	}
}

// TestPage_UsesNoNativeFormControls enforces the design rule behind the
// component layer: the page never paints a browser widget. Selects, option
// lists, radios, date pickers and file inputs are replaced by components; the
// only native element left is the checkbox inside a toggle switch, which keeps
// the switch keyboard- and screen-reader-accessible while .toggle-slider paints
// it.
func TestPage_UsesNoNativeFormControls(t *testing.T) {
	markup := extractBodyMarkup(t)
	for _, control := range []string{"<select", "<option", "<datalist", "<progress", "<meter", "<details", "<summary", "<output", "<fieldset", "<legend"} {
		if strings.Contains(markup, control) {
			t.Fatalf("page renders the native control %s; use the component layer instead", control)
		}
	}
	for _, inputType := range []string{"date", "datetime-local", "time", "month", "week", "range", "color", "file", "radio"} {
		if strings.Contains(markup, `type="`+inputType+`"`) {
			t.Fatalf("page renders a native input[type=%s]; use the component layer instead", inputType)
		}
	}

	checkboxCount := strings.Count(markup, `type="checkbox"`)
	switchCount := strings.Count(markup, `class="toggle-switch"`)
	if checkboxCount == 0 || checkboxCount != switchCount {
		t.Fatalf("found %d checkboxes but %d toggle switches; every checkbox must be painted by the shared switch", checkboxCount, switchCount)
	}

	numberCount := strings.Count(markup, `type="number"`)
	unitCount := strings.Count(markup, `class="input-unit-group"`)
	if numberCount == 0 || numberCount != unitCount {
		t.Fatalf("found %d number inputs but %d unit groups; every numeric field needs the shared number+unit input", numberCount, unitCount)
	}
}

// TestPage_HasNoInlineStyles keeps every declaration in the stylesheet: an
// inline style is how a component silently forks from the design system.
func TestPage_HasNoInlineStyles(t *testing.T) {
	markup := extractBodyMarkup(t)
	for _, pattern := range []string{` style="`, ` style='`} {
		if strings.Contains(markup, pattern) {
			t.Fatalf("markup carries the inline style %q; move the rule into the component layer", pattern)
		}
	}
	if matches := inlineStyleAssign.FindAllString(extractPageScript(t), -1); len(matches) > 0 {
		t.Fatalf("script assigns inline styles %v; use a component class instead", matches)
	}
}

// TestPage_SharesOneGlyphTable keeps every inline SVG resolved from pageIcons,
// so markup and runtime icons cannot drift apart.
func TestPage_SharesOneGlyphTable(t *testing.T) {
	for _, glyph := range []string{"refresh", "search", "calendar", "unit", "plus", "close", "copy", "check", "edit", "trash", "chevron", "prev", "next", "quota", "clock", "warning", "github", "key", "sliders", "help"} {
		if _, ok := pageIcons[glyph]; !ok {
			t.Fatalf("glyph %s is missing from the shared icon table", glyph)
		}
	}
	if !strings.Contains(PageHTML, "var PAGE_ICONS = {") {
		t.Fatal("the page runtime does not receive the shared glyph table")
	}
	if matches := iconPlaceholder.FindAllString(PageHTML, -1); len(matches) > 0 {
		t.Fatalf("page still contains unresolved icon placeholders: %v", matches)
	}
}

// TestPage_LastSeenColumnUsesLifetimeEntry pins the column contract: the header
// exists, the cell reads the entry-level lifetime value instead of the
// window-scoped usage block, and the shared timestamp formatter is reused.
func TestPage_LastSeenColumnUsesLifetimeEntry(t *testing.T) {
	if !strings.Contains(PageHTML, `data-i18n="keys.column.last_seen"`) {
		t.Fatal("the keys table should declare a last seen column")
	}
	script := extractPageScript(t)
	for _, fragment := range []string{
		"item.last_seen ? formatTimestamp(item.last_seen) : '—'",
		"keys.column.last_seen",
	} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("last seen column wiring is missing %q", fragment)
		}
	}
	if strings.Contains(script, "function formatLastSeen") {
		t.Fatal("last seen should reuse formatTimestamp instead of a second formatter")
	}
	if !strings.Contains(script, "table.tHead.rows[0].cells.length") {
		t.Fatal("column count must read the direct header; nested model tables would inflate it")
	}
	if strings.Contains(script, "colSpan = 9") {
		t.Fatal("full-width rows must derive their span from the header, not a fixed column count")
	}
	if !strings.Contains(script, "function keyTableColumnCount") || !strings.Contains(script, "cell.colSpan = keyTableColumnCount();") {
		t.Fatal("detail and empty rows should span every table column")
	}
}

// TestPage_DetailRowIsRecessed pins the hierarchy fix: the expanded detail area
// uses the page-owned inset surface instead of the same token as the table
// header, and hover never repaints it.
func TestPage_DetailRowIsRecessed(t *testing.T) {
	style := extractPageStyle(t)
	detail := regexp.MustCompile(`(?s)\.detail-row > td \{.*?\}`).FindString(style)
	if detail == "" || !strings.Contains(detail, "background: var(--bg-inset);") {
		t.Fatal("the detail row should sit on the inset surface")
	}
	if strings.Contains(style, ".detail-row:hover") {
		t.Fatal("hover must not repaint the detail well")
	}
	if !strings.Contains(style, ".data-table tbody tr:not(.detail-row):hover td") {
		t.Fatal("the generic row hover should exclude the detail row")
	}
	if count := strings.Count(style, "--bg-inset:"); count != 4 {
		t.Fatalf("--bg-inset declared %d times, want once per theme block", count)
	}
}

// TestPage_QuotaBadgeExplainsItself pins the remark-column quota tags: both tags
// are icon led (gauge for usage/limit, clock for the reset), the visible quota
// tag carries no label word, and the reset tag counts down while its tooltip
// keeps the exact local instant.
func TestPage_QuotaBadgeExplainsItself(t *testing.T) {
	script := extractPageScript(t)
	for _, fragment := range []string{
		"function buildQuotaBadge",
		"iconMarkup('quota'",
		"formatTokenValue(quota.used_value) + ' / '",
		"function buildQuotaResetBadge",
		"iconMarkup('clock'",
		"'keys.quota_reset_title'",
		"formatTimestamp(quota.reset_at)",
	} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("page script is missing %s", fragment)
		}
	}
	if strings.Contains(script, "'keys.quota_badge'") || strings.Contains(script, "'限额:") {
		t.Fatal("the quota tag should be icon led, without a label word")
	}
	if strings.Contains(script, "reset_at.substring") {
		t.Fatal("the reset time must be parsed as an instant, not truncated out of the UTC string")
	}

	if !strings.Contains(PageHTML, `class="col-remark"`) {
		t.Fatal("the remark column should be marked as the flexible column")
	}
	if !strings.Contains(extractPageStyle(t), ".data-table th.col-remark { width: 100%; }") {
		t.Fatal("the remark column should absorb the table's leftover width")
	}
}

// TestPage_RemarkTagsShareOneShape pins the reported misalignment: the remark
// column renders one tag row, and every tag in it shares a single metric block,
// so a row reads as one set instead of four drifting look-alikes.
func TestPage_RemarkTagsShareOneShape(t *testing.T) {
	script := extractPageScript(t)
	if !strings.Contains(script, "className = 'tag-row'") {
		t.Fatal("the remark column must render its tags inside one .tag-row")
	}

	style := extractPageStyle(t)
	metrics := sharedTagMetricsPattern.FindStringSubmatch(style)
	if metrics == nil {
		t.Fatal("the remark tags should share one metric block")
	}
	for _, declaration := range []string{
		"display: inline-flex",
		"min-height: 26px",
		"padding: 0 11px",
		"border-radius: var(--radius-pill)",
		"font-size: 13px",
		"font-weight: 650",
	} {
		if !strings.Contains(metrics[1], declaration) {
			t.Fatalf("shared tag metrics are missing %q", declaration)
		}
	}
	if !strings.Contains(style, ".tag-row {") {
		t.Fatal("the component layer should declare the .tag-row container")
	}

	// UI fonts leave descent space below the baseline, so the glyphs need a small
	// upward nudge to look centred in the pill.
	optical := regexp.MustCompile(`(?s)\.badge-remark > span,\s*\.badge-quota > span,\s*\.badge-reset > span,\s*\.badge-models > span \{(.*?)\}`).FindStringSubmatch(style)
	if optical == nil || !strings.Contains(optical[1], "translateY(") {
		t.Fatal("the tag text should be optically centred inside its pill")
	}
	if !strings.Contains(script, "remarkText.textContent") {
		t.Fatal("the remark text should live in its own span so the nudge applies")
	}
}

// TestPage_ResetCountdown covers the compact countdown of the reset tag.
func TestPage_ResetCountdown(t *testing.T) {
	script := extractPageScript(t)
	countdownFn := regexp.MustCompile(`(?s)function formatRemaining\(.*?\n\}`).FindString(script)
	if countdownFn == "" {
		t.Fatal("cannot extract formatRemaining from the page script")
	}

	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed; skipping the countdown check")
	}

	runner := countdownFn + `
function at(offsetSeconds) {
  return new Date(Date.now() + offsetSeconds * 1000).toISOString();
}
// Offsets carry a half-second of slack: the implementation floors against the
// real clock, so an exact boundary would flake by one second.
function check(offset, want) {
  var got = formatRemaining(at(offset));
  if (got !== want) {
    console.error('formatRemaining(+' + offset + 's) = ' + got + ', want ' + want);
    process.exit(1);
  }
}
check(30.5, '30s');
check(59.5, '59s');
check(0, '0s');
check(-60, '0s');
check(65.5, '1m');
check(3599.5, '59m');
check(3601.5, '1h');
check(86399.5, '23h');
check(90061.5, '1d');
check(200000.5, '2d');
if (formatRemaining('') !== '—' || formatRemaining('nonsense') !== '—') {
  console.error('an empty or unparsable instant must render as a dash');
  process.exit(1);
}
`
	testScriptPath := filepath.Join(t.TempDir(), "test_countdown.js")
	if err := os.WriteFile(testScriptPath, []byte(runner), 0o600); err != nil {
		t.Fatalf("write test runner: %v", err)
	}
	if out, err := exec.Command(nodePath, testScriptPath).CombinedOutput(); err != nil {
		t.Fatalf("node test failed: %v\n%s", err, string(out))
	}
}

// TestPage_TokenUnitsFollowTheValue pins the unit rule behind the badge: a value
// below the selected unit keeps the next smaller unit, so 1K against a 1M limit
// reads "1K" and never "0M"/"0B".
func TestPage_TokenUnitsFollowTheValue(t *testing.T) {
	script := extractPageScript(t)
	numberFn := regexp.MustCompile(`(?s)function formatNumber\(.*?\n\}`).FindString(script)
	unitFn := regexp.MustCompile(`(?s)function formatTokenValue\(.*?\n\}`).FindString(script)
	if numberFn == "" || unitFn == "" {
		t.Fatal("cannot extract formatNumber/formatTokenValue from the page script")
	}

	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed; skipping the token unit check")
	}

	runner := `
var state = { tokenUnit: 2 };
var TOKEN_UNITS = ['', 'K', 'M', 'B'];
` + numberFn + "\n" + unitFn + `
function check(value, want) {
  var got = formatTokenValue(value);
  if (got !== want) {
    console.error('formatTokenValue(' + value + ') = ' + got + ', want ' + want);
    process.exit(1);
  }
}
check(1000, '1K');
check(1000000, '1M');
check(1500, '1.5K');
check(999, '999');
check(0, '0');
check(1500000000, '1500M');
state.tokenUnit = 3;
check(1000, '1K');
check(0, '0');
`
	testScriptPath := filepath.Join(t.TempDir(), "test_token_units.js")
	if err := os.WriteFile(testScriptPath, []byte(runner), 0o600); err != nil {
		t.Fatalf("write test runner: %v", err)
	}
	if out, err := exec.Command(nodePath, testScriptPath).CombinedOutput(); err != nil {
		t.Fatalf("node test failed: %v\n%s", err, string(out))
	}
}

func extractDictionaries(t *testing.T, script string) map[string]map[string]bool {
	t.Helper()
	match := regexp.MustCompile(`(?s)var I18N = \{(.*?)\n\};`).FindStringSubmatch(script)
	if match == nil {
		t.Fatal("page script does not declare the I18N dictionary")
	}

	out := map[string]map[string]bool{}
	for _, language := range []string{"zh-CN", "en-US"} {
		start := strings.Index(match[1], "'"+language+"'")
		if start < 0 {
			t.Fatalf("dictionary %s is missing", language)
		}
		body := match[1][start:]
		if next := strings.Index(body[1:], "'en-US'"); language == "zh-CN" && next >= 0 {
			body = body[:next+1]
		}
		keys := map[string]bool{}
		for _, key := range regexp.MustCompile(`'([a-zA-Z0-9_.]+)'\s*:`).FindAllStringSubmatch(body, -1) {
			keys[key[1]] = true
		}
		delete(keys, language)
		out[language] = keys
	}
	return out
}

func TestPage_RenderKeysCleansBodyBeforeAppend(t *testing.T) {
	script := extractPageScript(t)
	if !strings.Contains(script, "body.textContent = '';") {
		t.Fatal("renderKeys must clear body.textContent before appending rows to prevent duplicate items on refresh")
	}
}

func TestPage_HostModelsDiscoveryContract(t *testing.T) {
	script := extractPageScript(t)
	for _, token := range []string{
		"var V1_MODELS_PATH = '/v1/models';",
		"function extractModelIds",
		"function fetchHostModels",
		"function loadAvailableModels",
	} {
		if !strings.Contains(script, token) {
			t.Fatalf("page script missing %s", token)
		}
	}

	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed; skipping runtime JS evaluation test")
	}

	extractFn := regexp.MustCompile(`(?s)function extractModelIds\(.*?\n\}`).FindString(script)
	if extractFn == "" {
		t.Fatal("cannot find function extractModelIds in script")
	}

	testRunner := extractFn + `
var test1 = extractModelIds({ object: 'list', data: [{ id: 'gpt-5.6' }, { id: 'claude-sonnet-4-6' }] });
if (test1.length !== 2 || test1[0] !== 'gpt-5.6' || test1[1] !== 'claude-sonnet-4-6') {
  console.error('failed test1:', test1);
  process.exit(1);
}

var test2 = extractModelIds({ models: ['gemini-2.5-pro', 'gemini-4-argon'] });
if (test2.length !== 2 || test2[0] !== 'gemini-2.5-pro' || test2[1] !== 'gemini-4-argon') {
  console.error('failed test2:', test2);
  process.exit(1);
}

var test3 = extractModelIds([{ name: 'named-model' }, 'raw-model', '', null, { id: 'id-model' }]);
if (test3.length !== 3 || test3[0] !== 'named-model' || test3[1] !== 'raw-model' || test3[2] !== 'id-model') {
  console.error('failed test3:', test3);
  process.exit(1);
}

var testEmpty = extractModelIds(null);
if (!Array.isArray(testEmpty) || testEmpty.length !== 0) {
  console.error('failed testEmpty:', testEmpty);
  process.exit(1);
}
`
	cmd := exec.Command(nodePath, "-e", testRunner)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node test runner failed: %v, output: %s", err, string(output))
	}
}

func TestPage_DefaultsAndMaskedKeyContract(t *testing.T) {
	if !strings.Contains(PageHTML, `id="key-form-raw-value" type="hidden"`) {
		t.Fatal("key-form should have hidden input key-form-raw-value for raw key storage")
	}

	if strings.Contains(PageHTML, `class="switch-row quota-switch-header"`) {
		t.Fatal("quota card header should not carry switch-row to avoid unwanted margins/borders")
	}

	script := extractPageScript(t)
	for _, fragment := range []string{
		"tokenUnit: 3",
		"rawInput.value = value",
		"input.value = maskKey(value)",
		"var bounds = dateRangeBounds()",
	} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("script is missing expected contract fragment %q", fragment)
		}
	}
}

func TestPage_SettingsCacheFilesAndUsageHintContract(t *testing.T) {
	if strings.Contains(PageHTML, `data-i18n="settings.usage_hint"`) {
		t.Fatal("settings.usage_hint should be removed from page markup")
	}
	if !strings.Contains(PageHTML, `id="settings-cache-files"`) {
		t.Fatal("settings page markup should include settings-cache-files container")
	}
	if !strings.Contains(PageHTML, `data-i18n="settings.cache_files_title"`) {
		t.Fatal("settings page markup should include cache files title")
	}

	script := extractPageScript(t)
	for _, fragment := range []string{
		"renderCacheFiles()",
		"function formatBytes(bytes)",
		"loadSettings().catch",
	} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("script is missing expected cache files contract fragment %q", fragment)
		}
	}
}

func TestPage_KeySortingAndModelFailuresContract(t *testing.T) {
	// 1. Verify markup contract for sortable columns
	for _, col := range []string{`data-sort="last_seen"`, `data-sort="requests"`, `data-sort="failed"`, `data-sort="total"`} {
		if !strings.Contains(PageHTML, col) {
			t.Fatalf("PageHTML is missing sortable column marker %s", col)
		}
	}
	if !strings.Contains(PageHTML, "th-sortable") {
		t.Fatal("PageHTML is missing th-sortable CSS class")
	}

	// 2. Verify script fragments
	script := extractPageScript(t)
	for _, fragment := range []string{
		"KEYS_SORT_STORAGE_KEY",
		"function readStoredSort()",
		"function writeStoredSort(sort)",
		"function toggleKeySort(field)",
		"function updateSortHeaders()",
		"openFailuresModal(keyIndex, mName)",
		"function openFailuresModal(index, initialModel)",
	} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("script is missing expected contract fragment %q", fragment)
		}
	}

	// 3. Run node eval for sort logic and storage
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed; skipping runtime JS evaluation test")
	}

	testRunner := `
var mockStorage = {};
function readStorage(win, type, key) {
  return mockStorage[key] || '';
}
function writeStorage(win, type, key, val) {
  mockStorage[key] = String(val);
  return true;
}
var KEYS_SORT_STORAGE_KEY = 'cpa-secret-manager-keys-sort';

var DEFAULT_KEYS_SORT = { field: 'last_seen', order: 'desc' };

function readStoredSort() {
  var stored = readStorage(null, 'localStorage', KEYS_SORT_STORAGE_KEY);
  if (!stored) {
    return DEFAULT_KEYS_SORT;
  }
  try {
    var parsed = JSON.parse(stored);
    if (parsed && typeof parsed.field === 'string' && (parsed.order === 'asc' || parsed.order === 'desc')) {
      return { field: parsed.field, order: parsed.order };
    }
  } catch (error) {}
  return DEFAULT_KEYS_SORT;
}

function writeStoredSort(sort) {
  if (sort && sort.field && (sort.order === 'asc' || sort.order === 'desc')) {
    writeStorage(null, 'localStorage', KEYS_SORT_STORAGE_KEY, JSON.stringify(sort));
  } else {
    writeStorage(null, 'localStorage', KEYS_SORT_STORAGE_KEY, '');
  }
}

var state = { sort: readStoredSort() };
if (!state.sort || state.sort.field !== 'last_seen' || state.sort.order !== 'desc') {
  console.error('initial sort should default to last_seen desc', state.sort);
  process.exit(1);
}

function toggleKeySort(field) {
  if (!state.sort || state.sort.field !== field) {
    state.sort = { field: field, order: 'desc' };
  } else if (state.sort.order === 'desc') {
    state.sort = { field: field, order: 'asc' };
  } else {
    state.sort = { field: 'last_seen', order: 'desc' };
  }
  writeStoredSort(state.sort);
}

// Test toggle cycle: desc -> asc -> default(last_seen desc)
toggleKeySort('requests');
if (!state.sort || state.sort.field !== 'requests' || state.sort.order !== 'desc') {
  console.error('first toggle should be requests desc', state.sort);
  process.exit(1);
}
if (readStoredSort().order !== 'desc') {
  console.error('storage should reflect requests desc');
  process.exit(1);
}
toggleKeySort('requests');
if (!state.sort || state.sort.field !== 'requests' || state.sort.order !== 'asc') {
  console.error('second toggle should be requests asc', state.sort);
  process.exit(1);
}
toggleKeySort('requests');
if (!state.sort || state.sort.field !== 'last_seen' || state.sort.order !== 'desc') {
  console.error('third toggle should reset to default last_seen desc', state.sort);
  process.exit(1);
}

// Test sorting function with sample entries
state.entries = [
  { last_seen: '2026-03-01T10:00:00Z', usage: { requests: 10, failed: 2, total: 500 } },
  { last_seen: '2026-03-03T12:00:00Z', usage: { requests: 50, failed: 0, total: 100 } },
  { last_seen: '', usage: { requests: 30, failed: 5, total: 1000 } }
];

function sortIndices(field, order) {
  var indices = [0, 1, 2];
  var isAsc = order === 'asc';
  indices.sort(function (a, b) {
    var itemA = state.entries[a] || {};
    var itemB = state.entries[b] || {};
    var valA = 0;
    var valB = 0;
    if (field === 'last_seen') {
      valA = itemA.last_seen ? new Date(itemA.last_seen).getTime() : 0;
      valB = itemB.last_seen ? new Date(itemB.last_seen).getTime() : 0;
    } else if (field === 'requests') {
      valA = (itemA.usage && itemA.usage.requests) || 0;
      valB = (itemB.usage && itemB.usage.requests) || 0;
    } else if (field === 'failed') {
      valA = (itemA.usage && itemA.usage.failed) || 0;
      valB = (itemB.usage && itemB.usage.failed) || 0;
    } else if (field === 'total') {
      valA = (itemA.usage && itemA.usage.total) || 0;
      valB = (itemB.usage && itemB.usage.total) || 0;
    }
    if (valA !== valB) {
      return isAsc ? (valA > valB ? 1 : -1) : (valA < valB ? 1 : -1);
    }
    return a - b;
  });
  return indices;
}

// Requests desc: idx 1 (50), idx 2 (30), idx 0 (10)
var rDesc = sortIndices('requests', 'desc');
if (rDesc.join(',') !== '1,2,0') {
  console.error('requests desc mismatch:', rDesc);
  process.exit(1);
}

// Failed desc: idx 2 (5), idx 0 (2), idx 1 (0)
var fDesc = sortIndices('failed', 'desc');
if (fDesc.join(',') !== '2,0,1') {
  console.error('failed desc mismatch:', fDesc);
  process.exit(1);
}

// Total desc: idx 2 (1000), idx 0 (500), idx 1 (100)
var tDesc = sortIndices('total', 'desc');
if (tDesc.join(',') !== '2,0,1') {
  console.error('total desc mismatch:', tDesc);
  process.exit(1);
}

// Last seen desc: idx 1 (2026-03-03), idx 0 (2026-03-01), idx 2 (empty -> 0)
var lsDesc = sortIndices('last_seen', 'desc');
if (lsDesc.join(',') !== '1,0,2') {
  console.error('last_seen desc mismatch:', lsDesc);
  process.exit(1);
}
`
	testScriptPath := filepath.Join(t.TempDir(), "test_sorting.js")
	if err := os.WriteFile(testScriptPath, []byte(testRunner), 0o600); err != nil {
		t.Fatalf("write test runner: %v", err)
	}
	cmd := exec.Command(nodePath, testScriptPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("node test failed: %v\n%s", err, string(out))
	}
}

func TestPage_SettingsLayoutStructure(t *testing.T) {
	markup := settingsFeature.markup
	openDivs := strings.Count(markup, "<div")
	closeDivs := strings.Count(markup, "</div>")
	if openDivs != closeDivs {
		t.Fatalf("settings markup div tag mismatch: %d open vs %d close", openDivs, closeDivs)
	}
	openSections := strings.Count(markup, "<section")
	closeSections := strings.Count(markup, "</section>")
	if openSections != closeSections {
		t.Fatalf("settings markup section tag mismatch: %d open vs %d close", openSections, closeSections)
	}
	for _, token := range []string{
		"settings-layout",
		"settings-col-main",
		"settings-col-side",
		"settings-pricing-model-select",
		"settings-pricing-add",
		"settings-retention-input",
		"settings-cache-files",
	} {
		if !strings.Contains(markup, token) {
			t.Fatalf("settings markup missing %s", token)
		}
	}
}
