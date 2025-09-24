package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/johnjallday/dolphin-agent/pluginapi"
	"github.com/johnjallday/dolphin-reaper-plugin/pkg/scripts"
	"github.com/johnjallday/dolphin-reaper-plugin/pkg/settings"
	"github.com/openai/openai-go/v2"
)

// Global settings manager
var globalSettingsManager = settings.NewManager()

// reaperTool implements pluginapi.Tool for launching ReaScripts (.lua) in REAPER.
type reaperTool struct {
	agentContext    *pluginapi.AgentContext
	settingsManager *settings.Manager
	scriptsManager  *scripts.Manager
}

// Ensure compile-time conformance.
var _ pluginapi.Tool = reaperTool{}
var _ pluginapi.VersionedTool = reaperTool{}

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
		Name:        "reaper_manager",
		Description: openai.String("Manage REAPER ReaScripts: list available scripts, launch them, or configure setup"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"operation": map[string]any{
					"type":        "string",
					"description": "Operation to perform",
					"enum":        []string{"list", "run", "get_settings"},
				},
				"script": map[string]any{
					"type":        "string",
					"description": "Base name of the ReaScript (without .lua). Required only for 'run' operation.",
					"enum":        enum, // may be nil/empty if directory unreadable; that's fine
				},
			},
			"required": []string{"operation"},
		},
	}
}

// Call handles the function call and dispatches to appropriate handlers
func (t reaperTool) Call(ctx context.Context, args string) (string, error) {
	var p struct {
		Operation string `json:"operation"`
		Script    string `json:"script"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", err
	}

	// Get current scripts directory and create a scripts manager
	scriptsDir := globalSettingsManager.GetCurrentScriptsDir()
	scriptsManager := scripts.NewManager(scriptsDir)

	switch p.Operation {
	case "list":
		return scriptsManager.ListScripts()
	case "run":
		return scriptsManager.RunScript(p.Script)
	case "get_settings":
		return globalSettingsManager.GetSettingsStruct()
	default:
		return "", fmt.Errorf("unknown operation: %s. Valid operations: list, run, get_settings", p.Operation)
	}
}

// Version returns the plugin version
func (t reaperTool) Version() string {
	return Version
}

// Settings interface implementation
func (t reaperTool) GetSettings() (string, error) {
	return globalSettingsManager.GetSettings()
}

func (t reaperTool) SetSettings(settingsJSON string) error {
	return globalSettingsManager.SetSettings(settingsJSON)
}

// GetDefaultSettings returns the default settings as JSON
func (t reaperTool) GetDefaultSettings() (string, error) {
	return globalSettingsManager.GetDefaultSettingsJSON()
}

// SetAgentContext provides the current agent information to the plugin
func (t *reaperTool) SetAgentContext(ctx pluginapi.AgentContext) {
	t.agentContext = &ctx
}

// Tool is the exported symbol the host looks up via plugin.Open().Lookup("Tool").
var Tool = reaperTool{
	settingsManager: globalSettingsManager,
}

