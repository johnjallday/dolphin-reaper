package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-plugin"
	"github.com/johnjallday/ori-agent/pluginapi"
	reapercontext "github.com/johnjallday/ori-reaper-plugin/internal/context"
	"github.com/johnjallday/ori-reaper-plugin/internal/scripts"
	"github.com/johnjallday/ori-reaper-plugin/internal/settings"
	"github.com/openai/openai-go/v2"
)

// Global settings manager
var globalSettingsManager = settings.NewManager()

// reaperTool implements pluginapi.Tool for launching ReaScripts (.lua) in REAPER.
type reaperTool struct {
	agentContext    *pluginapi.AgentContext
	settingsManager *settings.Manager
}

// Ensure compile-time conformance.
var _ pluginapi.Tool = reaperTool{}
var _ pluginapi.VersionedTool = reaperTool{}
var _ pluginapi.InitializationProvider = (*reaperTool)(nil)

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
		Description: openai.String("Manage REAPER ReaScripts: list available scripts, launch them, add new scripts, delete them, download scripts from repository, or configure setup"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"operation": map[string]any{
					"type":        "string",
					"description": "Operation to perform",
					"enum":        []string{"list", "run", "add", "delete", "list_available_scripts", "download_script", "register_script", "register_all_scripts", "clean_scripts", "get_context"},
				},
				"script": map[string]any{
					"type":        "string",
					"description": "Base name of the ReaScript (without extension). Required for 'run', 'add', and 'delete' operations.",
					"enum":        enum, // may be nil/empty if directory unreadable; that's fine
				},
				"filename": map[string]any{
					"type":        "string",
					"description": "Full filename of the script to download (including extension). Required for 'download_script' operation.",
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
		if p.Filename == "" {
			return "", fmt.Errorf("filename is required for 'download_script' operation")
		}
		downloader := scripts.NewScriptDownloader()
		return downloader.DownloadScript(p.Filename, scriptsDir)
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
	default:
		return "", fmt.Errorf("unknown operation: %s. Valid operations: list, run, add, delete, list_available_scripts, download_script, register_script, register_all_scripts, clean_scripts, get_context", p.Operation)
	}
}

// Version returns the plugin version
func (t reaperTool) Version() string {
	return Version
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
	return []pluginapi.ConfigVariable{
		{
			Key:          "scripts_dir",
			Name:         "Scripts Directory",
			Description:  "Directory where REAPER scripts (.lua, .eel, .py) are stored",
			Type:         pluginapi.ConfigTypeDirPath,
			Required:     true,
			DefaultValue: "/Users/YOUR_USERNAME/Library/Application Support/REAPER/Scripts",
			Placeholder:  "/path/to/REAPER/Scripts",
		},
	}
}

func (t *reaperTool) ValidateConfig(config map[string]interface{}) error {
	scriptsDir, ok := config["scripts_dir"].(string)
	if !ok || scriptsDir == "" {
		return fmt.Errorf("scripts_dir is required")
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

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: pluginapi.Handshake,
		Plugins: map[string]plugin.Plugin{
			"tool": &pluginapi.ToolRPCPlugin{Impl: &reaperTool{
				settingsManager: globalSettingsManager,
			}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

