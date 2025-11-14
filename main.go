package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os/user"
	"path/filepath"

	"github.com/hashicorp/go-plugin"
	"github.com/johnjallday/ori-agent/pluginapi"
	reapercontext "github.com/johnjallday/ori-reaper-plugin/internal/context"
	"github.com/johnjallday/ori-reaper-plugin/internal/scripts"
	"github.com/johnjallday/ori-reaper-plugin/internal/settings"
	"github.com/johnjallday/ori-reaper-plugin/internal/webpage"
)

//go:embed plugin.yaml
var configYAML string

// Global settings manager
var globalSettingsManager = settings.NewManager()

// reaperTool implements the PluginTool interface.
type reaperTool struct {
	pluginapi.BasePlugin
	settingsManager *settings.Manager
	webpageProvider *webpage.Provider
}

// Ensure compile-time conformance
var _ pluginapi.PluginTool = (*reaperTool)(nil)
var _ pluginapi.VersionedTool = (*reaperTool)(nil)
var _ pluginapi.PluginCompatibility = (*reaperTool)(nil)
var _ pluginapi.MetadataProvider = (*reaperTool)(nil)
var _ pluginapi.AgentAwareTool = (*reaperTool)(nil)
var _ pluginapi.InitializationProvider = (*reaperTool)(nil)
var _ pluginapi.WebPageProvider = (*reaperTool)(nil)

// Definition returns the tool definition for ori-reaper
func (t *reaperTool) Definition() pluginapi.Tool {
	return pluginapi.Tool{
		Name:        "ori-reaper",
		Description: "Manage REAPER ReaScripts: list available scripts, launch them, add new scripts, delete them, browse marketplace, configure Web Remote, or manage control surfaces",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"description": "Operation to perform. Use 'download_script' to get the marketplace URL for browsing and downloading scripts visually.",
					"enum":        []string{"list", "run", "add", "delete", "list_available_scripts", "download_script", "register_script", "register_all_scripts", "clean_scripts", "get_context", "get_web_remote_port", "get_tracks"},
				},
				"script": map[string]interface{}{
					"type":        "string",
					"description": "Base name of the ReaScript (without extension). Required for 'run', 'add', and 'delete' operations.",
				},
				"filename": map[string]interface{}{
					"type":        "string",
					"description": "Full filename of the script (including extension). Not used by 'download_script' - that operation now redirects to the marketplace.",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Script content. Required for 'add' operation.",
				},
				"script_type": map[string]interface{}{
					"type":        "string",
					"description": "Script type/extension. Required for 'add' operation. Valid values: lua, eel, py",
					"enum":        []string{"lua", "eel", "py"},
				},
			},
			"required": []string{"operation"},
		},
	}
}

// Call implements the PluginTool interface
func (t *reaperTool) Call(ctx context.Context, args string) (string, error) {
	// Parse parameters
	var params struct {
		Operation  string `json:"operation"`
		Script     string `json:"script"`
		Filename   string `json:"filename"`
		Content    string `json:"content"`
		ScriptType string `json:"script_type"`
	}

	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("failed to parse parameters: %w", err)
	}
	// Get current scripts directory and create a script manager
	scriptsDir := globalSettingsManager.GetCurrentScriptsDir()
	scriptManager := scripts.NewScriptManager(scriptsDir)

	switch params.Operation {
	case "list":
		return scriptManager.ListScripts()
	case "run":
		return scriptManager.RunScript(params.Script)
	case "add":
		return scriptManager.AddScript(params.Script, params.Content, params.ScriptType)
	case "delete":
		return scriptManager.DeleteScript(params.Script)
	case "list_available_scripts":
		downloader := scripts.NewScriptDownloader()
		return downloader.ListAvailableScripts()
	case "download_script":
		// Redirect to marketplace for visual browsing and downloading
		return "🎵 Browse and download scripts at the marketplace:\nhttp://localhost:8080/api/plugins/ori-reaper/pages/marketplace", nil
	case "register_script":
		if params.Script == "" {
			return "", fmt.Errorf("script name is required for 'register_script' operation")
		}
		return scriptManager.RegisterScript(params.Script)
	case "register_all_scripts":
		return scriptManager.RegisterAllScripts()
	case "clean_scripts":
		return scriptManager.CleanScripts()
	case "get_context":
		ctx, err := reapercontext.GetREAPERContext()
		if err != nil {
			return "", fmt.Errorf("failed to get REAPER context: %w", err)
		}
		contextJSON, err := json.Marshal(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to marshal context: %w", err)
		}
		return string(contextJSON), nil
	case "get_web_remote_port":
		// Get port from configuration
		configuredPort := globalSettingsManager.GetWebRemotePort()
		result := fmt.Sprintf("REAPER Web Remote:\n"+
			"  Configured Port: %d\n"+
			"  URL: http://localhost:%d\n"+
			"  Note: This port is set in plugin configuration. Ensure REAPER's Web Remote matches this port.\n",
			configuredPort, configuredPort)
		return result, nil
	case "get_tracks":
		// Get port from configuration
		configuredPort := globalSettingsManager.GetWebRemotePort()

		// Create Web Remote client with configured port
		client, err := scripts.NewWebRemoteClient(configuredPort)
		if err != nil {
			return "", fmt.Errorf("failed to create web remote client: %w", err)
		}

		tracks, err := client.GetTracks()
		if err != nil {
			return "", fmt.Errorf("failed to get tracks from REAPER: %w", err)
		}
		return scripts.FormatTracksTable(tracks), nil
	default:
		return "", fmt.Errorf("unknown operation: %s. Valid operations: list, run, add, delete, list_available_scripts, download_script, register_script, register_all_scripts, clean_scripts, get_context, get_web_remote_port, get_tracks", params.Operation)
	}
}

// GetDefaultSettings returns the default settings as JSON
func (t *reaperTool) GetDefaultSettings() (string, error) {
	return globalSettingsManager.GetDefaultSettingsJSON()
}

// InitializationProvider implementation for frontend settings
func (t *reaperTool) GetRequiredConfig() []pluginapi.ConfigVariable {
	usr, _ := user.Current()
	defaultReascriptDir := filepath.Join(usr.HomeDir, "Library", "Application Support", "REAPER", "Scripts")

	configVars := []pluginapi.ConfigVariable{
		{
			Key:          "scripts_dir",
			Name:         "Scripts Directory",
			Description:  "Directory where REAPER scripts (.lua, .eel, .py) are stored",
			Type:         pluginapi.ConfigTypeDirPath,
			Required:     true,
			DefaultValue: defaultReascriptDir,
			Placeholder:  defaultReascriptDir,
		},
	}

	// Try to detect existing web remote port from reaper.ini
	if _, err := scripts.GetWebRemoteConfig(); err == nil {
		// Found existing web remote configuration - no need to require it
		// The plugin will automatically use the detected port
		// Port is not added to required config
	} else {
		// No existing web remote found - require user to configure it
		configVars = append(configVars, pluginapi.ConfigVariable{
			Key:          "web_remote_port",
			Name:         "Web Remote Port",
			Description:  "No Web Remote found in REAPER configuration. Please specify the port you want to use. Configure Web Remote in REAPER (Preferences → Control/OSC/web) to match this port.",
			Type:         pluginapi.ConfigTypeInt,
			Required:     true,
			DefaultValue: "2307",
			Placeholder:  "2307",
		})
	}

	return configVars
}

func (t *reaperTool) ValidateConfig(config map[string]interface{}) error {
	scriptsDir, ok := config["scripts_dir"].(string)
	if !ok || scriptsDir == "" {
		return fmt.Errorf("scripts_dir is required")
	}

	// Validate web_remote_port if provided (it's optional if auto-detected)
	if portValue, ok := config["web_remote_port"]; ok {
		// Handle both float64 (from JSON unmarshal) and int
		var port int
		switch v := portValue.(type) {
		case float64:
			port = int(v)
		case int:
			port = v
		default:
			return fmt.Errorf("web_remote_port must be a number")
		}

		if port < 1024 || port > 65535 {
			return fmt.Errorf("web_remote_port must be between 1024 and 65535")
		}
	}

	return nil
}

func (t *reaperTool) InitializeWithConfig(config map[string]interface{}) error {
	// Save settings to agent directory
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return globalSettingsManager.SetSettings(string(data))
}

// GetWebPages returns list of available web pages
func (t *reaperTool) GetWebPages() []string {
	return t.webpageProvider.GetPages()
}

// ServeWebPage serves a custom web page
func (t *reaperTool) ServeWebPage(path string, query map[string]string) (string, string, error) {
	return t.webpageProvider.ServePage(path, query)
}

func main() {
	// Parse plugin config from embedded YAML
	config := pluginapi.ReadPluginConfig(configYAML)

	// Create REAPER tool with base plugin
	tool := &reaperTool{
		BasePlugin: pluginapi.NewBasePlugin(
			"ori-reaper",                      // Plugin name
			config.Version,                    // Version from config
			config.Requirements.MinOriVersion, // Min agent version
			"",                                // Max agent version (no limit)
			"v1",                              // API version
		),
		settingsManager: globalSettingsManager,
		webpageProvider: webpage.NewProvider(globalSettingsManager),
	}

	// Set metadata from config
	if metadata, err := config.ToMetadata(); err == nil {
		tool.SetMetadata(metadata)
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: pluginapi.Handshake,
		Plugins: map[string]plugin.Plugin{
			"tool": &pluginapi.ToolRPCPlugin{Impl: tool},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
