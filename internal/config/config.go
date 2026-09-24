package config

import "strings"

// DefaultStatePath is the fallback location of the plugin state document,
// resolved relative to the CLIProxyAPI process working directory.
const DefaultStatePath = "data/cpa-secret-manager-cache.json"

// Config is the complete configuration owned by the CPA host config.yaml.
// Business parameters deliberately live in the plugin state document instead,
// see ADR-0001.
type Config struct {
	// StatePath is the location of the plugin state document.
	StatePath string
}

// Default returns the canonical fallback configuration.
func Default() Config {
	return Config{StatePath: DefaultStatePath}
}

// LoadBytes parses the plugin configuration section received from the host.
// The host passes plugins.configs.cpa-secret-manager, either as the bare
// section or as a full CPA config.yaml document.
func LoadBytes(data []byte) (Config, []string, error) {
	cfg := Default()

	section := extractPluginConfigYAML(string(data))
	if strings.TrimSpace(section) == "" {
		return cfg, nil, nil
	}

	values, err := parseConfigMap(section)
	if err != nil {
		return cfg, nil, err
	}

	var warnings []string
	if raw, ok := values["state_path"]; ok {
		text, okText := raw.(string)
		if !okText || strings.TrimSpace(text) == "" {
			warnings = append(warnings, "state_path must be a non-empty string; using default state path")
		} else {
			cfg.StatePath = strings.TrimSpace(text)
		}
	}
	return cfg, warnings, nil
}
