package webpage

import (
	"encoding/json"
	"fmt"

	"github.com/johnjallday/ori-reaper-plugin/internal/scripts"
	"github.com/johnjallday/ori-reaper-plugin/internal/settings"
)

// Provider handles web page serving for the ori-reaper plugin
type Provider struct {
	settingsManager *settings.Manager
}

// NewProvider creates a new webpage provider
func NewProvider(settingsManager *settings.Manager) *Provider {
	return &Provider{
		settingsManager: settingsManager,
	}
}

// GetPages returns the list of available web pages
func (p *Provider) GetPages() []string {
	return []string{"marketplace"}
}

// ServePage serves the requested web page
func (p *Provider) ServePage(path string, query map[string]string) (string, string, error) {
	switch path {
	case "marketplace":
		return p.serveMarketplace()
	default:
		return "", "", fmt.Errorf("page not found: %s", path)
	}
}

// serveMarketplace generates the script marketplace HTML page
func (p *Provider) serveMarketplace() (string, string, error) {
	// Get available scripts from repository
	downloader := scripts.NewScriptDownloader()
	scriptsJSON, err := downloader.ListAvailableScripts()
	if err != nil {
		return "", "", fmt.Errorf("failed to list available scripts: %w", err)
	}

	// Parse the modal result structure
	var modalResult struct {
		Type  string                   `json:"type"`
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal([]byte(scriptsJSON), &modalResult); err != nil {
		return "", "", fmt.Errorf("failed to parse scripts list: %w", err)
	}

	scriptsList := modalResult.Items

	// Get currently installed scripts
	scriptsDir := p.settingsManager.GetCurrentScriptsDir()
	installedScripts, _ := scripts.ListLuaScripts(scriptsDir)
	installedMap := make(map[string]bool)
	for _, name := range installedScripts {
		installedMap[name] = true
	}

	// Generate HTML using template
	html := generateMarketplaceHTML(scriptsList, installedMap)
	return html, "text/html; charset=utf-8", nil
}

// generateMarketplaceHTML creates the marketplace HTML from script data
func generateMarketplaceHTML(scriptsList []map[string]interface{}, installedMap map[string]bool) string {
	html := getMarketplaceTemplate()

	// Add script cards
	for _, script := range scriptsList {
		name, _ := script["name"].(string)
		description, _ := script["description"].(string)
		filename, _ := script["filename"].(string)
		scriptType, _ := script["type"].(string)

		installed := installedMap[name]

		html += fmt.Sprintf(`
            <div class="script-card" data-name="%s" data-description="%s">
                <div class="script-name">%s</div>
                <div class="script-description">%s</div>
                <div class="script-meta">
                    <span class="meta-badge">📄 %s</span>
                    <span class="meta-badge">🏷️ %s</span>
                </div>`,
			name, description, name, description, filename, scriptType)

		if installed {
			html += `<div class="installed-badge">✓ Installed</div>`
		} else {
			html += fmt.Sprintf(`<button class="install-btn" onclick="installScript('%s')">Install Script</button>`, filename)
		}

		html += `</div>`
	}

	// Close HTML
	html += getMarketplaceFooter()

	return html
}
