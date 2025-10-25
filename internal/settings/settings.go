package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/johnjallday/ori-reaper-plugin/internal/platform"
	"github.com/johnjallday/ori-reaper-plugin/internal/types"
)

// Manager manages plugin settings
type Manager struct {
	settings *types.Settings
}

// NewManager creates a new settings manager
func NewManager() *Manager {
	return &Manager{}
}

// SetSettings updates settings from JSON
func (sm *Manager) SetSettings(settingsJSON string) error {
	var settings types.Settings
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		return fmt.Errorf("failed to unmarshal settings: %w", err)
	}
	sm.settings = &settings
	return nil
}

// GetDefaultSettings creates default settings
func (sm *Manager) GetDefaultSettings() *types.Settings {
	return &types.Settings{
		ScriptsDir: platform.DefaultScriptsDir(),
	}
}

// GetCurrentSettings returns current settings, initializing if needed
func (sm *Manager) GetCurrentSettings() *types.Settings {
	if sm.settings == nil {
		sm.settings = sm.GetDefaultSettings()
	}
	return sm.settings
}

// GetCurrentScriptsDir returns the current scripts directory from settings
func (sm *Manager) GetCurrentScriptsDir() string {
	// Try to load settings from file if not already loaded
	if sm.settings == nil {
		if loadedSettings, err := sm.loadSettingsFromAPI(); err == nil {
			sm.settings = loadedSettings
		}
	}
	settings := sm.GetCurrentSettings()
	return settings.ScriptsDir
}

// GetDefaultSettingsJSON returns the default settings as JSON string
func (sm *Manager) GetDefaultSettingsJSON() (string, error) {
	defaultSettings := sm.GetDefaultSettings()
	data, err := json.MarshalIndent(defaultSettings, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal default settings: %w", err)
	}
	return string(data), nil
}

// loadSettingsFromAPI loads settings from agent-specific settings file
func (sm *Manager) loadSettingsFromAPI() (*types.Settings, error) {
	// Get current agent from agents.json file
	currentAgent, err := sm.getCurrentAgentFromFile()
	if err != nil {
		// Fall back to default settings if no agent file or error reading it
		return sm.GetDefaultSettings(), nil
	}

	// Try to load settings from the agent-specific file
	settingsPath := filepath.Join(".", "agents", currentAgent, "ori-reaper_settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var settings types.Settings
		if err := json.Unmarshal(data, &settings); err == nil {
			return &settings, nil
		}
	}

	// Fall back to default settings if file doesn't exist or is invalid
	return sm.GetDefaultSettings(), nil
}

// getCurrentAgentFromFile reads the current agent from agents.json
func (sm *Manager) getCurrentAgentFromFile() (string, error) {
	agentsFilePath := filepath.Join(".", "agents.json")
	data, err := os.ReadFile(agentsFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read agents.json: %w", err)
	}

	var agentsConfig types.AgentsConfig
	if err := json.Unmarshal(data, &agentsConfig); err != nil {
		return "", fmt.Errorf("failed to parse agents.json: %w", err)
	}

	if agentsConfig.CurrentAgent == "" {
		return "", fmt.Errorf("no current agent set in agents.json")
	}

	return agentsConfig.CurrentAgent, nil
}