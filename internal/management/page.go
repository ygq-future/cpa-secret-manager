package management

import (
	"strings"
)

// pageFeature is one self-contained section of the embedded page.
type pageFeature struct {
	// markup is appended to the body.
	markup string
	// styles are appended inside the single <style> element.
	styles string
	// scripts are appended inside the single IIFE.
	scripts string
}

var pageFeatures = []pageFeature{
	keysFeature,
	settingsFeature,
	helpFeature,
}

// PageHTML is the assembled browser resource page. It runs entirely offline
// with no external resources, see ADR-0003.
var PageHTML = assemblePage()

func assemblePage() string {
	var page strings.Builder
	page.Grow(64 << 10)

	page.WriteString(pageDocumentStart)
	page.WriteString(templateStyleTokens)
	page.WriteString(templateStyleShell)
	for _, feature := range pageFeatures {
		page.WriteString(feature.styles)
	}
	page.WriteString(templateStyleResponsive)
	page.WriteString(pageBodyStart)

	page.WriteString(shellMarkup)
	for _, feature := range pageFeatures {
		page.WriteString(feature.markup)
	}
	page.WriteString(shellModals)
	page.WriteString(pageScriptStart)

	page.WriteString(templateScriptI18N)
	page.WriteString(templateScriptShellCore)
	for _, feature := range pageFeatures {
		page.WriteString(feature.scripts)
	}
	page.WriteString(templateScriptBoot)
	page.WriteString(pageDocumentEnd)
	return page.String()
}

const pageDocumentStart = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light dark">
<title>API Key Manager</title>
<style>
`

const pageBodyStart = `</style>
</head>
<body>
`

const pageScriptStart = `<script>
(function () {
'use strict';
`

const pageDocumentEnd = `})();
</script>
</body>
</html>
`

const templateScriptBoot = `
initializeApp();
`
