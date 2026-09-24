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
	styleBlockPattern  = regexp.MustCompile(`(?s)<style>(.*?)</style>`)
	scriptBlockPattern = regexp.MustCompile(`(?s)<script>(.*?)</script>`)
	rootBlockPattern   = regexp.MustCompile(`(?s):root\s*\{(.*?)\}`)
	whiteBlockPattern  = regexp.MustCompile(`(?s)\[data-theme='white'\]\s*\{(.*?)\}`)
	darkBlockPattern   = regexp.MustCompile(`(?s)\[data-theme='dark'\]\s*\{(.*?)\}`)
	tokenPattern       = regexp.MustCompile(`(--[a-z0-9-]+)\s*:`)
	referencePattern   = regexp.MustCompile(`var\(\s*(--[a-z0-9-]+)`)
	hostVarPattern     = regexp.MustCompile(`'(--[a-z0-9-]+)'`)
	translationPattern = regexp.MustCompile(`data-i18n[a-z-]*="([^"]+)"`)
	callPattern        = regexp.MustCompile(`\btf?\('([a-z0-9_.]+)'`)
	domIdPattern       = regexp.MustCompile(`byId\('([^']+)'\)`)
	bindingPattern     = regexp.MustCompile(`bind(?:Click|Submit|Backdrop)\('([^']+)'`)
	markupIDPattern    = regexp.MustCompile(`id="([^"]+)"`)
	inlineHandler      = regexp.MustCompile(`\son[a-z]+\s*=\s*"`)
	remoteURLPattern   = regexp.MustCompile(`https?://`)
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
