package management

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

// This file owns the Go side of the page's shared component layer: the glyph
// table, the markup helpers every feature builds controls with, and the
// placeholder resolver that expands them while the page is assembled. The
// styles live in components_styles.go and the runtime in components_script.go.
//
// Components never hardcode a colour: they consume the tokens declared in
// ThemePaletteCSS. Features consume components instead of re-declaring them, so
// a feature only ships markup and styles that exist nowhere else.

// pageIcons is the page's single glyph table: every inline SVG, in static markup
// and in runtime-built nodes, resolves through this map. Each entry is the inner
// content of a 24x24 stroke icon.
var pageIcons = map[string]string{
	"key":      `<circle cx="7.5" cy="15.5" r="4.5"></circle><path d="M10.7 12.3 20 3"></path><path d="M16.5 6.5 20 10"></path>`,
	"refresh":  `<polyline points="23 4 23 10 17 10"></polyline><polyline points="1 20 1 14 7 14"></polyline><path d="M3.5 9a9 9 0 0 1 14.9-3.4L23 10M1 14l4.6 4.4A9 9 0 0 0 20.5 15"></path>`,
	"sliders":  `<path d="M4 21v-7M4 10V3M12 21v-9M12 8V3M20 21v-5M20 12V3"></path><path d="M1 14h6M9 8h6M17 16h6"></path>`,
	"help":     `<circle cx="12" cy="12" r="10"></circle><path d="M9.1 9a3 3 0 0 1 5.8 1c0 2-2.9 3-2.9 3"></path><path d="M12 17h.01"></path>`,
	"search":   `<circle cx="11" cy="11" r="7"></circle><path d="M20 20l-3.6-3.6"></path>`,
	"calendar": `<rect width="18" height="18" x="3" y="4" rx="2" ry="2"></rect><line x1="16" x2="16" y1="2" y2="6"></line><line x1="8" x2="8" y1="2" y2="6"></line><line x1="3" x2="21" y1="10" y2="10"></line>`,
	"unit":     `<path d="M4 9h16M4 15h16M10 3 8 21M16 3l-2 18"></path>`,
	"plus":     `<path d="M12 5v14M5 12h14"></path>`,
	"close":    `<path d="M18 6 6 18M6 6l12 12"></path>`,
	"copy":     `<rect x="9" y="9" width="12" height="12" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>`,
	"check":    `<polyline points="20 6 9 17 4 12"></polyline>`,
	"edit":     `<path d="M12 20h9"></path><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"></path>`,
	"trash":    `<polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>`,
	"chevron":  `<polyline points="6 9 12 15 18 9"></polyline>`,
	"prev":     `<polyline points="15 18 9 12 15 6"></polyline>`,
	"next":     `<polyline points="9 18 15 12 9 6"></polyline>`,
	"quota":    `<path d="M4 15.5a8 8 0 0 1 16 0"></path><path d="M12 15.5l3.4-5.2"></path><circle cx="12" cy="15.5" r="1.1"></circle>`,
	"clock":    `<circle cx="12" cy="12" r="8.5"></circle><path d="M12 7.5v4.8l3.2 2"></path>`,
	"warning":  `<path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z"></path><path d="M12 9v4"></path><path d="M12 17h.01"></path>`,
	"github":   `<path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"></path>`,
}

// icon renders one glyph for the static markup. An unknown name panics while
// the page is assembled, so a typo can never ship an invisible icon.
func icon(name string) string {
	glyph, ok := pageIcons[name]
	if !ok {
		panic("management: unknown icon " + name)
	}
	return `<svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">` + glyph + `</svg>`
}

// toggleSwitchMarkup renders the shared ON/OFF switch: the page's only checkbox
// wrapper, so every switch shares one DOM shape, one style and one focus ring.
func toggleSwitchMarkup(id string, attrs ...string) string {
	var extra strings.Builder
	for _, attr := range attrs {
		if attr == "" {
			continue
		}
		extra.WriteString(" ")
		extra.WriteString(attr)
	}
	return `<label class="toggle-switch"><input id="` + id + `" type="checkbox" role="switch"` + extra.String() +
		`><span class="toggle-slider" aria-hidden="true"></span></label>`
}

// markupSnippets holds the component instances that static markup references by
// placeholder, so one component can never be spelled two different ways.
var markupSnippets = map[string]string{
	"{{TOGGLE_USAGE}}": toggleSwitchMarkup("settings-usage-toggle",
		`data-i18n-title="settings.usage_toggle"`, `data-i18n-aria-label="settings.usage_toggle"`),
	"{{TOGGLE_QUOTA}}": toggleSwitchMarkup("key-form-quota-enabled"),
}

var iconPlaceholder = regexp.MustCompile(`\{\{ICON:([a-z-]+)\}\}`)

// resolveMarkup expands the icon and component placeholders of one markup chunk.
// An unknown placeholder panics: the page is assembled at build time, so a typo
// must never reach a browser.
func resolveMarkup(markup string) string {
	if markup == "" {
		return markup
	}
	resolved := iconPlaceholder.ReplaceAllStringFunc(markup, func(match string) string {
		return icon(iconPlaceholder.FindStringSubmatch(match)[1])
	})
	for placeholder, snippet := range markupSnippets {
		resolved = strings.ReplaceAll(resolved, placeholder, snippet)
	}
	if index := strings.Index(resolved, "{{"); index >= 0 {
		end := index + 24
		if end > len(resolved) {
			end = len(resolved)
		}
		panic("management: unresolved markup placeholder " + resolved[index:end])
	}
	return resolved
}

// iconTableJSON serializes the glyph table for the page runtime. The keys are
// sorted so the generated page stays byte-stable.
func iconTableJSON() string {
	names := make([]string, 0, len(pageIcons))
	for name := range pageIcons {
		names = append(names, name)
	}
	sort.Strings(names)

	var table strings.Builder
	table.WriteString("{\n")
	for index, name := range names {
		encoded, err := json.Marshal(pageIcons[name])
		if err != nil {
			continue
		}
		table.WriteString("  ")
		table.WriteString(jsString(name))
		table.WriteString(": ")
		table.Write(encoded)
		if index < len(names)-1 {
			table.WriteString(",")
		}
		table.WriteString("\n")
	}
	table.WriteString("}")
	return table.String()
}

// jsString renders a glyph name as a JavaScript object key.
func jsString(name string) string {
	encoded, _ := json.Marshal(name)
	return string(encoded)
}

// templateStyleBase normalizes the browser defaults the page never wants to
// show: native steppers, native search decorations, autofill paints and native
// scrollbars. Every interactive widget on the page is drawn by the component
// layer instead.

// templateStyleComponents declares every reusable component: buttons, pills,
// badges, cards, fields, the toggle switch, the listbox, the chip grid, tables,
// overlays and status chrome.

// templateScriptComponentsTemplate declares the runtime of the shared
// components. The glyph table is injected from pageIcons, so markup and runtime
// icons can never drift apart.

// templateScriptComponents is the component runtime with the glyph table
// resolved.
var templateScriptComponents = strings.Replace(templateScriptComponentsTemplate, "{{ICONS}}", iconTableJSON(), 1)
