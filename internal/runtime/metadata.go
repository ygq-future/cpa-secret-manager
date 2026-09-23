package runtime

// PluginVersion is the plugin release version. It must stay in sync with
// registry.json, the embedded page version badge and the release notes file.
const PluginVersion = "1.0.0"

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
		Version:          PluginVersion,
		Author:           pluginAuthor,
		GitHubRepository: pluginRepository,
		Description:      pluginSummary,
	}
}
