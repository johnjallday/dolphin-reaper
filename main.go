package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/johnjallday/dolphin-agent/pluginapi"
	"github.com/openai/openai-go/v2"
	"github.com/shirou/gopsutil/v3/process"
)

// Settings represents the REAPER plugin configuration
type Settings struct {
	ScriptsDir  string `json:"scripts_dir"`
	Initialized bool   `json:"initialized"`
}

// SettingsManager manages plugin settings
type SettingsManager struct {
	settings *Settings
}

var globalSettings = &SettingsManager{}

// GetSettings returns the current settings as JSON
func (sm *SettingsManager) GetSettings() (string, error) {
	if sm.settings == nil {
		sm.settings = sm.getDefaultSettings()
	}
	data, err := json.MarshalIndent(sm.settings, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal settings: %w", err)
	}
	return string(data), nil
}

// SetSettings updates settings from JSON
func (sm *SettingsManager) SetSettings(settingsJSON string) error {
	var settings Settings
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		return fmt.Errorf("failed to unmarshal settings: %w", err)
	}
	sm.settings = &settings
	return nil
}

// UpdateSettings updates both in-memory settings and persists to agent_settings.json
func (sm *SettingsManager) UpdateSettings(scriptsDir string, initialized bool, agentContext *pluginapi.AgentContext) error {
	if sm.settings == nil {
		sm.settings = sm.getDefaultSettings()
	}

	// Update in-memory settings
	if scriptsDir != "" {
		sm.settings.ScriptsDir = scriptsDir
	}
	sm.settings.Initialized = initialized

	// If we have agent context, also persist to agent_settings.json
	if agentContext != nil {
		return sm.persistToAgentSettings(scriptsDir, agentContext)
	}

	return nil
}

// persistToAgentSettings saves the current settings to the agent's settings file
func (sm *SettingsManager) persistToAgentSettings(scriptsDir string, agentContext *pluginapi.AgentContext) error {
	settingsFilePath := agentContext.SettingsPath

	var agentSettings map[string]interface{}
	if settingsData, err := os.ReadFile(settingsFilePath); err == nil {
		if err := json.Unmarshal(settingsData, &agentSettings); err != nil {
			return fmt.Errorf("failed to parse agent settings at %s: %w", settingsFilePath, err)
		}
	} else {
		agentSettings = make(map[string]interface{})
	}

	if _, exists := agentSettings["reaper_script_launcher"]; !exists {
		agentSettings["reaper_script_launcher"] = make(map[string]interface{})
	}

	reaperSettings := agentSettings["reaper_script_launcher"].(map[string]interface{})

	if scriptsDir != "" {
		reaperSettings["scripts_dir"] = scriptsDir
	}

	reaperSettings["initialized"] = sm.settings.Initialized

	if err := os.MkdirAll(filepath.Dir(settingsFilePath), 0755); err != nil {
		return fmt.Errorf("failed to create agent directory: %w", err)
	}

	updatedData, err := json.MarshalIndent(agentSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated agent settings: %w", err)
	}

	if err := os.WriteFile(settingsFilePath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write to %s: %w", settingsFilePath, err)
	}

	return nil
}

// GetDefaultSettings returns default settings as JSON
func (sm *SettingsManager) GetDefaultSettings() (string, error) {
	defaults := sm.getDefaultSettings()
	data, err := json.MarshalIndent(defaults, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal default settings: %w", err)
	}
	return string(data), nil
}

// IsInitialized returns true if the plugin has been configured
func (sm *SettingsManager) IsInitialized() bool {
	if sm.settings == nil {
		sm.settings = sm.getDefaultSettings()
	}
	return sm.settings.Initialized
}

// getDefaultSettings creates default settings
func (sm *SettingsManager) getDefaultSettings() *Settings {
	return &Settings{
		ScriptsDir:  defaultScriptsDir(),
		Initialized: false, // Require explicit setup
	}
}

// getCurrentSettings returns current settings, initializing if needed
func (sm *SettingsManager) getCurrentSettings() *Settings {
	if sm.settings == nil {
		sm.settings = sm.getDefaultSettings()
	}
	return sm.settings
}

// getCurrentScriptsDir returns the current scripts directory from settings
func getCurrentScriptsDir() string {
	settings := globalSettings.getCurrentSettings()
	return settings.ScriptsDir
}

// reaperTool implements pluginapi.Tool for launching ReaScripts (.lua) in REAPER.
type reaperTool struct {
	agentContext *pluginapi.AgentContext
	scriptsDir   string   // absolute path to the REAPER Scripts folder (deprecated, use settings)
	scripts      []string // script base names (without .lua extension)
}

// Ensure compile-time conformance.
var _ pluginapi.Tool = reaperTool{}
var _ pluginapi.VersionedTool = reaperTool{}
var _ pluginapi.ConfigurableTool = reaperTool{}

// -------- Helper functions --------

func userHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func defaultScriptsDir() string {
	switch runtime.GOOS {
	case "darwin":
		// macOS: ~/Library/Application Support/REAPER/Scripts
		return filepath.Join(userHome(), "Library", "Application Support", "REAPER", "Scripts")
	case "windows":
		// Windows: %APPDATA%\REAPER\Scripts
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "REAPER", "Scripts")
		}
		return filepath.Join(userHome(), "AppData", "Roaming", "REAPER", "Scripts")
	default:
		// Linux: ~/.config/REAPER/Scripts
		return filepath.Join(userHome(), ".config", "REAPER", "Scripts")
	}
}

func listLuaScripts(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".lua") {
			names = append(names, strings.TrimSuffix(name, ".lua"))
		}
	}
	return names, nil
}

func toTitleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func isReaperRunning() (bool, error) {
	procs, err := process.Processes()
	if err != nil {
		return false, err
	}
	for _, p := range procs {
		n, err := p.Name()
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(n), "reaper") {
			return true, nil
		}
	}
	return false, nil
}

func launchScript(scriptsDir, base string) error {
	scriptPath := filepath.Join(scriptsDir, base+".lua")

	// Verify the script exists
	if _, err := os.Stat(scriptPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("script not found: %s", scriptPath)
		}
		return err
	}

	switch runtime.GOOS {
	case "darwin":
		// macOS: open -a Reaper <script>
		cmd := exec.Command("open", "-a", "Reaper", scriptPath)
		return cmd.Run()

	case "windows":
		// Best effort: try to open with the registered app (REAPER) using "start".
		// Note: requires proper association; otherwise, customize to call the REAPER exe with args.
		cmd := exec.Command("cmd", "/c", "start", "", scriptPath)
		return cmd.Run()

	default: // linux
		// If REAPER is in PATH and supports opening scripts directly
		// you may need to adjust this depending on your REAPER install.
		cmd := exec.Command("reaper", scriptPath)
		return cmd.Run()
	}
}

// -------- pluginapi.Tool implementation --------

func (t reaperTool) Definition() openai.FunctionDefinitionParam {
	// Build enum of scripts from current settings directory
	enum := []string(nil)
	scriptsDir := getCurrentScriptsDir()
	if scripts, err := listLuaScripts(scriptsDir); err == nil && len(scripts) > 0 {
		enum = append(enum, scripts...)
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
					"enum":        []string{"list", "run", "get_settings", "init_setup", "complete_setup"},
				},
				"script": map[string]any{
					"type":        "string",
					"description": "Base name of the ReaScript (without .lua). Required only for 'run' operation.",
					"enum":        enum, // may be nil/empty if directory unreadable; that's fine
				},
				"scripts_dir": map[string]any{
					"type":        "string",
					"description": "Path to REAPER Scripts directory (required for complete_setup)",
				},
			},
			"required": []string{"operation"},
		},
	}
}

func (t reaperTool) Call(ctx context.Context, args string) (string, error) {
	var p struct {
		Operation  string `json:"operation"`
		Script     string `json:"script"`
		ScriptsDir string `json:"scripts_dir"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", err
	}

	// Allow certain operations even when not initialized
	allowedWhenUninitialized := []string{"get_settings", "init_setup", "complete_setup"}
	operationAllowed := false
	for _, allowed := range allowedWhenUninitialized {
		if p.Operation == allowed {
			operationAllowed = true
			break
		}
	}

	// Auto-initialize prompt if not initialized and operation requires it
	if !globalSettings.IsInitialized() && !operationAllowed {
		return t.handleInitSetup()
	}

	switch p.Operation {
	case "list":
		return t.handleListScripts()
	case "run":
		return t.handleRunScript(p.Script)
	case "get_settings":
		return t.handleGetSettings()
	case "init_setup":
		return t.handleInitSetup()
	case "complete_setup":
		return t.handleCompleteSetup(p.ScriptsDir)
	default:
		return "", fmt.Errorf("unknown operation: %s. Valid operations: list, run, get_settings, init_setup, complete_setup", p.Operation)
	}
}

func (t reaperTool) handleListScripts() (string, error) {
	// Get current scripts directory from settings
	scriptsDir := getCurrentScriptsDir()

	// Get fresh list of scripts from the directory
	scripts, err := listLuaScripts(scriptsDir)
	if err != nil {
		return "", fmt.Errorf("failed to list scripts in %s: %w", scriptsDir, err)
	}

	if len(scripts) == 0 {
		return fmt.Sprintf("No ReaScripts (.lua files) found in: %s", scriptsDir), nil
	}

	// Create structured data
	type ScriptItem struct {
		Index       int    `json:"index"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		Action      string `json:"action"`
	}

	type ScriptList struct {
		Type        string       `json:"type"`
		Title       string       `json:"title"`
		Count       int          `json:"count"`
		Location    string       `json:"location"`
		Scripts     []ScriptItem `json:"scripts"`
		Instruction string       `json:"instruction"`
	}

	var scriptItems []ScriptItem
	for i, script := range scripts {
		displayName := strings.ReplaceAll(script, "_", " ")
		displayName = toTitleCase(displayName)

		scriptItems = append(scriptItems, ScriptItem{
			Index:       i + 1,
			Name:        script,
			DisplayName: displayName,
			Action:      script,
		})
	}

	result := ScriptList{
		Type:        "reaper_script_list",
		Title:       "🎵 Available REAPER Scripts",
		Count:       len(scripts),
		Location:    scriptsDir,
		Scripts:     scriptItems,
		Instruction: "To run a script, say: \"Run the [script_name] script\"",
	}

	// Return as JSON string with special prefix to indicate structured data
	jsonData, err := json.Marshal(result)
	if err != nil {
		// Fallback to markdown format if JSON marshaling fails
		return t.handleListScriptsMarkdown(scripts)
	}

	return "STRUCTURED_DATA:" + string(jsonData), nil
}

func (t reaperTool) handleListScriptsMarkdown(scripts []string) (string, error) {
	// Fallback markdown format
	result := fmt.Sprintf("## 🎵 Available REAPER Scripts (%d found)\n\n", len(scripts))
	result += "| # | Script Name | Action |\n"
	result += "|---|-------------|--------|\n"

	for i, script := range scripts {
		displayName := strings.ReplaceAll(script, "_", " ")
		displayName = toTitleCase(displayName)
		result += fmt.Sprintf("| %d | **%s** | `%s` |\n", i+1, displayName, script)
	}

	result += fmt.Sprintf("\n📂 **Location:** `%s`\n", getCurrentScriptsDir())
	result += "\n💡 **To run a script, say:** *\"Run the [script_name] script\"*"

	return result, nil
}

func (t reaperTool) handleRunScript(script string) (string, error) {
	if strings.TrimSpace(script) == "" {
		return "", errors.New("script name is required for 'run' operation")
	}

	running, err := isReaperRunning()
	if err != nil {
		return "", fmt.Errorf("could not check for REAPER process: %w", err)
	}
	if !running {
		// Not an error for the model; return a friendly message.
		return "REAPER is not running. Please start REAPER first, then try running the script again.", nil
	}

	if err := launchScript(getCurrentScriptsDir(), script); err != nil {
		return "", err
	}
	return fmt.Sprintf("Successfully launched REAPER script: %s", script), nil
}

func (t reaperTool) handleGetSettings() (string, error) {
	settings := globalSettings.getCurrentSettings()
	return fmt.Sprintf("🎵 REAPER Script Manager Settings:\n\n"+
		"- Scripts Directory: %s\n"+
		"- Initialized: %t\n\n"+
		"Available scripts: %d", 
		settings.ScriptsDir, 
		settings.Initialized,
		func() int {
			if scripts, err := listLuaScripts(settings.ScriptsDir); err == nil {
				return len(scripts)
			}
			return 0
		}()), nil
}

func (t reaperTool) handleInitSetup() (string, error) {
	if globalSettings.IsInitialized() {
		settings := globalSettings.getCurrentSettings()
		scriptCount := 0
		if scripts, err := listLuaScripts(settings.ScriptsDir); err == nil {
			scriptCount = len(scripts)
		}
		
		return fmt.Sprintf("🎵 REAPER Script Manager is already set up and ready to use.\n\n"+
			"Current settings:\n"+
			"- Scripts Directory: %s\n"+
			"- Available Scripts: %d\n\n"+
			"Use operation 'get_settings' to view detailed configuration or 'list' to see available scripts.", 
			settings.ScriptsDir, scriptCount), nil
	}

	defaultDir := defaultScriptsDir()
	return fmt.Sprintf("🎵 Welcome to REAPER Script Manager!\n\n"+
		"This is your first time using the plugin. Please complete the setup by providing:\n\n"+
		"**Scripts Directory** - Path to your REAPER Scripts folder where .lua files are stored\n"+
		"   Default suggestion: %s\n\n"+
		"Please use operation 'complete_setup' with scripts_dir parameter to finish the setup.\n\n"+
		"Example: reaper_manager(operation=\"complete_setup\", scripts_dir=\"%s\")\n\n"+
		"You can also specify a custom path if your REAPER Scripts are in a different location.",
		defaultDir, defaultDir), nil
}

func (t reaperTool) handleCompleteSetup(scriptsDir string) (string, error) {
	if scriptsDir == "" {
		return "", fmt.Errorf("scripts_dir is required for complete_setup operation")
	}

	// Expand tilde if present
	expandedDir, err := expandTilde(scriptsDir)
	if err != nil {
		return "", fmt.Errorf("failed to expand home directory in scripts path %q: %w", scriptsDir, err)
	}

	// Validate the directory exists
	if _, err := os.Stat(expandedDir); os.IsNotExist(err) {
		return "", fmt.Errorf("scripts directory does not exist: %s", expandedDir)
	}

	// Update settings with new directory and mark as initialized
	if err := globalSettings.UpdateSettings(expandedDir, true, t.agentContext); err != nil {
		return "", fmt.Errorf("failed to update settings: %w", err)
	}

	// Count available scripts
	scripts, err := listLuaScripts(expandedDir)
	scriptCount := 0
	if err == nil {
		scriptCount = len(scripts)
	}

	return fmt.Sprintf("✅ REAPER Script Manager setup completed successfully!\n\n"+
		"Configuration:\n"+
		"- Scripts Directory: %s\n"+
		"- Available Scripts: %d\n\n"+
		"The plugin is now ready to use. You can:\n"+
		"- Use 'list' to see available scripts\n"+
		"- Use 'run' to launch a specific script\n"+
		"- Use 'get_settings' to view current configuration", 
		expandedDir, scriptCount), nil
}

// expandTilde expands ~ to the user's home directory in a path
func expandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	
	if path == "~" {
		return homeDir, nil
	}
	
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:]), nil
	}
	
	return path, nil
}

// Version returns the plugin version.
// Version information set at build time via -ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func (t reaperTool) Version() string {
	return Version
}

// Settings interface implementation
func (t reaperTool) GetSettings() (string, error) {
	return globalSettings.GetSettings()
}

func (t reaperTool) SetSettings(settings string) error {
	return globalSettings.SetSettings(settings)
}

func (t reaperTool) GetDefaultSettings() (string, error) {
	return globalSettings.GetDefaultSettings()
}

func (t reaperTool) IsInitialized() bool {
	return globalSettings.IsInitialized()
}

// SetAgentContext provides the current agent information to the plugin
func (t *reaperTool) SetAgentContext(ctx pluginapi.AgentContext) {
	t.agentContext = &ctx

	// Only persist settings if already initialized, don't auto-initialize
	if globalSettings.IsInitialized() {
		scriptsDir := getCurrentScriptsDir()
		if updateErr := globalSettings.UpdateSettings(scriptsDir, true, t.agentContext); updateErr != nil {
			// Log the error but don't fail the operation
			fmt.Printf("Warning: Failed to update REAPER settings: %v\n", updateErr)
		}
	}
}

// Tool is the exported symbol the host looks up via plugin.Open().Lookup("Tool").
var Tool = reaperTool{
	// Legacy fields kept for backward compatibility but not used
	scriptsDir: "",
	scripts:    nil,
}
