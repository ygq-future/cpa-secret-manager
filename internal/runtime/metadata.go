package runtime

import "cpa-secret-manager/internal/version"

const (
	pluginName       = "CPA Secret Manager"
	pluginAuthor     = "ygq-future"
	pluginRepository = "https://github.com/ygq-future/cpa-secret-manager"
	pluginSummary    = "API key manager for CLIProxyAPI with per-key remarks and per-key token usage grouped by model."
)

// buildMetadata returns the plugin metadata sent to the host on registration.
func buildMetadata() Metadata {
	return Metadata{
		Name:             pluginName,
		Version:          version.PluginVersion,
		Author:           pluginAuthor,
		GitHubRepository: pluginRepository,
		Description:      pluginSummary,
	}
}
