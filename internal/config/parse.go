package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

const pluginConfigKey = "cpa-secret-manager"

// extractPluginConfigYAML returns the YAML section that belongs to this plugin.
// It accepts the bare plugin section and full CPA config.yaml documents.
func extractPluginConfigYAML(data string) string {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return ""
	}
	if looksLikePluginSection(trimmed) {
		return trimmed
	}

	lines := strings.Split(data, "\n")
	pluginsIndex := -1
	for index, line := range lines {
		if leadingSpaces(line) == 0 && strings.TrimSpace(line) == "plugins:" {
			pluginsIndex = index
			break
		}
	}
	if pluginsIndex < 0 {
		return ""
	}

	pluginsBody := indentedBlock(lines, pluginsIndex+1, 0)
	configsIndex := indexOfKey(pluginsBody, "configs:")
	if configsIndex < 0 {
		return ""
	}
	configsIndent := leadingSpaces(pluginsBody[configsIndex])
	configsBody := indentedBlock(pluginsBody, configsIndex+1, configsIndent)
	pluginIndex := indexOfKey(configsBody, pluginConfigKey+":")
	if pluginIndex < 0 {
		return ""
	}
	pluginIndent := leadingSpaces(configsBody[pluginIndex])
	return dedent(indentedBlock(configsBody, pluginIndex+1, pluginIndent))
}

// looksLikePluginSection reports whether the document already is the plugin
// section rather than a full host configuration.
func looksLikePluginSection(data string) bool {
	if strings.HasPrefix(data, "{") {
		return true
	}
	if strings.Contains(data, "\nplugins:") || strings.HasPrefix(data, "plugins:") {
		return false
	}
	for _, line := range strings.Split(data, "\n") {
		if leadingSpaces(line) != 0 {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "state_path:") || trimmed == "state_path:" {
			return true
		}
	}
	return false
}

// indentedBlock collects the lines belonging to a nested block that starts at
// start and whose content is indented deeper than parentIndent.
func indentedBlock(lines []string, start int, parentIndent int) []string {
	for index := start; index < len(lines); index++ {
		line := lines[index]
		if strings.TrimSpace(line) == "" {
			continue
		}
		if leadingSpaces(line) <= parentIndent {
			return lines[start:index]
		}
	}
	return lines[start:]
}

// indexOfKey returns the index of the first non-empty line whose trimmed form
// is key.
func indexOfKey(lines []string, key string) int {
	for index, line := range lines {
		if strings.TrimSpace(line) == key {
			return index
		}
	}
	return -1
}

func dedent(lines []string) string {
	indent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		current := leadingSpaces(line)
		if indent < 0 || current < indent {
			indent = current
		}
	}
	if indent <= 0 {
		return strings.Join(lines, "\n")
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(line) >= indent {
			out = append(out, line[indent:])
			continue
		}
		out = append(out, strings.TrimLeft(line, " "))
	}
	return strings.Join(out, "\n")
}

func leadingSpaces(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

// parseConfigMap parses top-level scalar keys of a plugin configuration
// section. Nested blocks are ignored: this plugin owns no nested settings.
func parseConfigMap(data string) (map[string]any, error) {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return map[string]any{}, nil
	}
	if strings.HasPrefix(trimmed, "{") {
		var raw map[string]any
		if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
			return nil, fmt.Errorf("config: decode JSON section: %w", err)
		}
		return raw, nil
	}

	values := make(map[string]any)
	for _, line := range strings.Split(data, "\n") {
		if strings.TrimSpace(line) == "" || leadingSpaces(line) != 0 {
			continue
		}
		body := line
		if index := strings.Index(body, " #"); index >= 0 {
			body = body[:index]
		}
		body = strings.TrimSpace(body)
		if body == "" || strings.HasPrefix(body, "#") {
			continue
		}
		key, rawValue, found := strings.Cut(body, ":")
		if !found {
			return nil, fmt.Errorf("config: line %q is not a key/value pair", body)
		}
		key = strings.TrimSpace(key)
		rawValue = strings.TrimSpace(rawValue)
		if key == "" || strings.HasPrefix(rawValue, "|") || strings.HasPrefix(rawValue, ">") {
			continue
		}
		if rawValue == "" {
			// Nested block; not part of this plugin's configuration surface.
			continue
		}
		values[key] = yamlScalar(rawValue)
	}
	return values, nil
}

// yamlScalar converts a YAML scalar token into its Go value.
func yamlScalar(raw string) any {
	value := strings.TrimSpace(raw)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	switch strings.ToLower(value) {
	case "true", "yes", "on":
		return true
	case "false", "no", "off":
		return false
	case "null", "~":
		return nil
	}
	return value
}
