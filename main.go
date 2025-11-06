package main

import (
	"context"
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
	"github.com/openai/openai-go/v2"
)

// Global settings manager
var globalSettingsManager = settings.NewManager()

// reaperTool implements pluginapi.Tool for launching ReaScripts (.lua) in REAPER.
type reaperTool struct {
	agentContext    *pluginapi.AgentContext
	settingsManager *settings.Manager
	webpageProvider *webpage.Provider
}

// Ensure compile-time conformance.
var _ pluginapi.Tool = reaperTool{}
var _ pluginapi.VersionedTool = reaperTool{}
var _ pluginapi.PluginCompatibility = reaperTool{}
var _ pluginapi.MetadataProvider = reaperTool{}
var _ pluginapi.InitializationProvider = (*reaperTool)(nil)
var _ pluginapi.WebPageProvider = (*reaperTool)(nil)

// Version information set at build time via -ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// Definition returns the OpenAI function definition for REAPER script management
func (t reaperTool) Definition() openai.FunctionDefinitionParam {
	// Build enum of scripts from current settings directory
	enum := []string(nil)
	scriptsDir := globalSettingsManager.GetCurrentScriptsDir()
	if scriptList, err := scripts.ListLuaScripts(scriptsDir); err == nil && len(scriptList) > 0 {
		enum = append(enum, scriptList...)
	}

	return openai.FunctionDefinitionParam{
		Name:        "ori_reaper",
		Description: openai.String("Manage REAPER ReaScripts: list available scripts, launch them, add new scripts, delete them, browse marketplace, configure Web Remote, or manage control surfaces"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"operation": map[string]any{
					"type":        "string",
					"description": "Operation to perform. Use 'download_script' to get the marketplace URL for browsing and downloading scripts visually.",
					"enum":        []string{"list", "run", "add", "delete", "list_available_scripts", "download_script", "register_script", "register_all_scripts", "clean_scripts", "get_context", "get_web_remote_port", "get_tracks"},
				},
				"script": map[string]any{
					"type":        "string",
					"description": "Base name of the ReaScript (without extension). Required for 'run', 'add', and 'delete' operations.",
					"enum":        enum, // may be nil/empty if directory unreadable; that's fine
				},
				"filename": map[string]any{
					"type":        "string",
					"description": "Full filename of the script (including extension). Not used by 'download_script' - that operation now redirects to the marketplace.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Script content. Required for 'add' operation.",
				},
				"script_type": map[string]any{
					"type":        "string",
					"description": "Script type/extension. Required for 'add' operation. Valid values: lua, eel, py",
					"enum":        []string{"lua", "eel", "py"},
				},
			},
			"required": []string{"operation"},
		},
	}
}

// Call handles the function call and dispatches to appropriate handlers
func (t reaperTool) Call(ctx context.Context, args string) (string, error) {
	var p struct {
		Operation  string `json:"operation"`
		Script     string `json:"script"`
		Content    string `json:"content"`
		ScriptType string `json:"script_type"`
		Filename   string `json:"filename"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", err
	}

	// Get current scripts directory and create a script manager
	scriptsDir := globalSettingsManager.GetCurrentScriptsDir()
	scriptManager := scripts.NewScriptManager(scriptsDir)

	switch p.Operation {
	case "list":
		return scriptManager.ListScripts()
	case "run":
		return scriptManager.RunScript(p.Script)
	case "add":
		return scriptManager.AddScript(p.Script, p.Content, p.ScriptType)
	case "delete":
		return scriptManager.DeleteScript(p.Script)
	case "list_available_scripts":
		downloader := scripts.NewScriptDownloader()
		return downloader.ListAvailableScripts()
	case "download_script":
		// Redirect to marketplace for visual browsing and downloading
		return "🎵 Browse and download scripts at the marketplace:\nhttp://localhost:8080/api/plugins/ori-reaper/pages/marketplace", nil
	case "register_script":
		if p.Script == "" {
			return "", fmt.Errorf("script name is required for 'register_script' operation")
		}
		return scriptManager.RegisterScript(p.Script)
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
		return "", fmt.Errorf("unknown operation: %s. Valid operations: list, run, add, delete, list_available_scripts, download_script, register_script, register_all_scripts, clean_scripts, get_context, get_web_remote_port, get_tracks", p.Operation)
	}
}

// Version returns the plugin version
func (t reaperTool) Version() string {
	return Version
}

// MinAgentVersion returns the minimum ori-agent version required
func (t reaperTool) MinAgentVersion() string {
	return "0.0.6" // Minimum version that supports plugin metadata
}

// MaxAgentVersion returns the maximum compatible ori-agent version
func (t reaperTool) MaxAgentVersion() string {
	return "" // No maximum limit
}

// APIVersion returns the plugin API version
func (t reaperTool) APIVersion() string {
	return "v1"
}

// GetMetadata returns plugin metadata (maintainers, license, repository)
func (t reaperTool) GetMetadata() (*pluginapi.PluginMetadata, error) {
	return &pluginapi.PluginMetadata{
		Maintainers: []*pluginapi.Maintainer{
			{
				Name:         "John J",
				Email:        "john@example.com",
				Organization: "Ori Project",
				Website:      "https://github.com/johnjallday",
				Role:         "author",
				Primary:      true,
			},
		},
		License:    "MIT",
		Repository: "https://github.com/johnjallday/ori-plugin-registry",
	}, nil
}

// GetDefaultSettings returns the default settings as JSON
func (t reaperTool) GetDefaultSettings() (string, error) {
	return globalSettingsManager.GetDefaultSettingsJSON()
}

// SetAgentContext provides the current agent information to the plugin
func (t *reaperTool) SetAgentContext(ctx pluginapi.AgentContext) {
	t.agentContext = &ctx
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
	if t.agentContext == nil {
		return fmt.Errorf("agent context not set")
	}

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
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: pluginapi.Handshake,
		Plugins: map[string]plugin.Plugin{
			"tool": &pluginapi.ToolRPCPlugin{Impl: &reaperTool{
				settingsManager: globalSettingsManager,
				webpageProvider: webpage.NewProvider(globalSettingsManager),
			}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
