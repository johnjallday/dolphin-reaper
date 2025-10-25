package scripts

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/johnjallday/ori-reaper-plugin/internal/platform"
	"github.com/johnjallday/ori-reaper-plugin/internal/types"
)

// ListLuaScripts lists all .lua script files in the given directory
func ListLuaScripts(dir string) ([]string, error) {
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

// ToTitleCase converts a string to title case
func ToTitleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// ScriptManager handles script operations
type ScriptManager struct {
	scriptsDir string
}

// NewScriptManager creates a new script manager with the given scripts directory
func NewScriptManager(scriptsDir string) *ScriptManager {
	return &ScriptManager{scriptsDir: scriptsDir}
}

// ListScripts returns a structured list of available scripts
func (sm *ScriptManager) ListScripts() (string, error) {
	// Get fresh list of scripts from the directory
	scripts, err := ListLuaScripts(sm.scriptsDir)
	if err != nil {
		return "", fmt.Errorf("failed to list scripts in %s: %w", sm.scriptsDir, err)
	}

	if len(scripts) == 0 {
		return fmt.Sprintf("No ReaScripts (.lua files) found in: %s", sm.scriptsDir), nil
	}

	var scriptItems []types.ScriptItem
	for i, script := range scripts {
		displayName := strings.ReplaceAll(script, "_", " ")
		displayName = ToTitleCase(displayName)

		scriptItems = append(scriptItems, types.ScriptItem{
			Index:       i + 1,
			Name:        script,
			DisplayName: displayName,
			Action:      script,
		})
	}

	result := types.ScriptList{
		Type:        "reaper_script_list",
		Title:       "🎵 Available REAPER Scripts",
		Count:       len(scripts),
		Location:    sm.scriptsDir,
		Scripts:     scriptItems,
		Instruction: "To run a script, say: \"Run the [script_name] script\"",
	}

	// Return as JSON string with special prefix to indicate structured data
	jsonData, err := json.Marshal(result)
	if err != nil {
		// Fallback to markdown format if JSON marshaling fails
		return sm.listScriptsMarkdown(scripts)
	}

	return "STRUCTURED_DATA:" + string(jsonData), nil
}

// listScriptsMarkdown returns a markdown-formatted list of scripts
func (sm *ScriptManager) listScriptsMarkdown(scripts []string) (string, error) {
	// Fallback markdown format
	result := fmt.Sprintf("## 🎵 Available REAPER Scripts (%d found)\n\n", len(scripts))
	result += "| # | Script Name | Action |\n"
	result += "|---|-------------|--------|\n"

	for i, script := range scripts {
		displayName := strings.ReplaceAll(script, "_", " ")
		displayName = ToTitleCase(displayName)
		result += fmt.Sprintf("| %d | **%s** | `%s` |\n", i+1, displayName, script)
	}

	result += fmt.Sprintf("\n📂 **Location:** `%s`\n", sm.scriptsDir)
	result += "\n💡 **To run a script, say:** *\"Run the [script_name] script\"*"

	return result, nil
}

// RunScript launches a script in REAPER
func (sm *ScriptManager) RunScript(script string) (string, error) {
	if strings.TrimSpace(script) == "" {
		return "", errors.New("script name is required for 'run' operation")
	}

	running, err := platform.IsReaperRunning()
	if err != nil {
		return "", fmt.Errorf("could not check for REAPER process: %w", err)
	}
	if !running {
		// Not an error for the model; return a friendly message.
		return "REAPER is not running. Please start REAPER first, then try running the script again.", nil
	}

	if err := platform.LaunchScript(sm.scriptsDir, script); err != nil {
		return "", err
	}
	return fmt.Sprintf("Successfully launched REAPER script: %s", script), nil
}

// DeleteScript deletes a script file from the scripts directory
func (sm *ScriptManager) DeleteScript(script string) (string, error) {
	if strings.TrimSpace(script) == "" {
		return "", errors.New("script name is required for 'delete' operation")
	}

	// Add .lua extension if not present
	scriptFile := script
	if !strings.HasSuffix(strings.ToLower(scriptFile), ".lua") {
		scriptFile = script + ".lua"
	}

	// Construct full path
	scriptPath := fmt.Sprintf("%s/%s", sm.scriptsDir, scriptFile)

	// Check if file exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("script not found: %s", script)
	}

	// Delete the file
	if err := os.Remove(scriptPath); err != nil {
		return "", fmt.Errorf("failed to delete script %s: %w", script, err)
	}

	return fmt.Sprintf("Successfully deleted REAPER script: %s", script), nil
}

// AddScript adds a new script file to the scripts directory
// Supports .lua, .eel, and .py extensions
func (sm *ScriptManager) AddScript(scriptName, content, scriptType string) (string, error) {
	if strings.TrimSpace(scriptName) == "" {
		return "", errors.New("script name is required for 'add' operation")
	}

	if strings.TrimSpace(content) == "" {
		return "", errors.New("script content is required for 'add' operation")
	}

	// Validate and normalize script type
	var extension string
	switch strings.ToLower(scriptType) {
	case "lua", ".lua":
		extension = ".lua"
	case "eel", ".eel":
		extension = ".eel"
	case "py", "python", ".py":
		extension = ".py"
	default:
		return "", fmt.Errorf("unsupported script type: %s. Supported types: lua, eel, py", scriptType)
	}

	// Remove extension from script name if already present
	scriptName = strings.TrimSuffix(scriptName, ".lua")
	scriptName = strings.TrimSuffix(scriptName, ".eel")
	scriptName = strings.TrimSuffix(scriptName, ".py")

	// Construct filename with proper extension
	scriptFile := scriptName + extension

	// Construct full path
	scriptPath := fmt.Sprintf("%s/%s", sm.scriptsDir, scriptFile)

	// Check if file already exists
	if _, err := os.Stat(scriptPath); err == nil {
		return "", fmt.Errorf("script already exists: %s", scriptFile)
	}

	// Write the file
	if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write script %s: %w", scriptFile, err)
	}

	return fmt.Sprintf("Successfully added REAPER script: %s", scriptFile), nil
}

// GetReaperKBIniPath returns the platform-specific path to reaper-kb.ini
func GetReaperKBIniPath() (string, error) {
	var basePath string

	switch runtime.GOOS {
	case "darwin": // macOS
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		basePath = filepath.Join(homeDir, "Library", "Application Support", "REAPER")

	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", errors.New("APPDATA environment variable not set")
		}
		basePath = filepath.Join(appData, "REAPER")

	case "linux":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		// Try common Linux paths
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig != "" {
			basePath = filepath.Join(xdgConfig, "REAPER")
		} else {
			basePath = filepath.Join(homeDir, ".config", "REAPER")
		}

	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	kbIniPath := filepath.Join(basePath, "reaper-kb.ini")

	// Check if the file exists
	if _, err := os.Stat(kbIniPath); os.IsNotExist(err) {
		return "", fmt.Errorf("reaper-kb.ini not found at %s (is REAPER installed?)", kbIniPath)
	}

	return kbIniPath, nil
}

// RegisterScript registers a script in REAPER's keyboard shortcuts file (reaper-kb.ini)
func (sm *ScriptManager) RegisterScript(scriptName string) (string, error) {
	if strings.TrimSpace(scriptName) == "" {
		return "", errors.New("script name is required for 'register_script' operation")
	}

	// Add .lua extension if not present
	scriptFile := scriptName
	if !strings.HasSuffix(strings.ToLower(scriptFile), ".lua") {
		scriptFile = scriptName + ".lua"
	}

	// Construct full path to the script
	scriptPath := filepath.Join(sm.scriptsDir, scriptFile)

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("script not found: %s", scriptName)
	}

	// Get reaper-kb.ini path
	kbIniPath, err := GetReaperKBIniPath()
	if err != nil {
		return "", err
	}

	// Read existing reaper-kb.ini file
	file, err := os.Open(kbIniPath)
	if err != nil {
		return "", fmt.Errorf("failed to open reaper-kb.ini: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	scriptAlreadyRegistered := false

	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)

		// Check if script is already registered
		if strings.Contains(line, scriptPath) {
			scriptAlreadyRegistered = true
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read reaper-kb.ini: %w", err)
	}

	// If already registered, return early
	if scriptAlreadyRegistered {
		return fmt.Sprintf("Script '%s' is already registered in REAPER", scriptName), nil
	}

	// Find the [Main] section and add the script
	// REAPER format: SCR 4 0 "Script: scriptname" "path/to/script.lua"
	scriptEntry := fmt.Sprintf(`SCR 4 0 "Script: %s" "%s"`, scriptName, scriptPath)

	// Find where to insert (after [Main] section header)
	inserted := false
	for i, line := range lines {
		if strings.HasPrefix(line, "[Main]") {
			// Insert after [Main] line
			lines = append(lines[:i+1], append([]string{scriptEntry}, lines[i+1:]...)...)
			inserted = true
			break
		}
	}

	// If [Main] section not found, append to end
	if !inserted {
		lines = append(lines, "", "[Main]", scriptEntry)
	}

	// Write back to file
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(kbIniPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write reaper-kb.ini: %w", err)
	}

	return fmt.Sprintf("Successfully registered script '%s' in REAPER keyboard shortcuts", scriptName), nil
}

// RegisterAllScripts registers all scripts in the scripts directory to reaper-kb.ini
func (sm *ScriptManager) RegisterAllScripts() (string, error) {
	scripts, err := ListLuaScripts(sm.scriptsDir)
	if err != nil {
		return "", fmt.Errorf("failed to list scripts: %w", err)
	}

	if len(scripts) == 0 {
		return "No scripts found to register", nil
	}

	registered := 0
	alreadyRegistered := 0
	failed := 0

	for _, script := range scripts {
		result, err := sm.RegisterScript(script)
		if err != nil {
			failed++
			continue
		}

		if strings.Contains(result, "already registered") {
			alreadyRegistered++
		} else {
			registered++
		}
	}

	summary := fmt.Sprintf("Registration complete: %d newly registered, %d already registered", registered, alreadyRegistered)
	if failed > 0 {
		summary += fmt.Sprintf(", %d failed", failed)
	}

	return summary, nil
}

// CleanScripts removes script entries from reaper-kb.ini where the script files no longer exist
func (sm *ScriptManager) CleanScripts() (string, error) {
	// Get reaper-kb.ini path
	kbIniPath, err := GetReaperKBIniPath()
	if err != nil {
		return "", err
	}

	// Read existing reaper-kb.ini file
	file, err := os.Open(kbIniPath)
	if err != nil {
		return "", fmt.Errorf("failed to open reaper-kb.ini: %w", err)
	}
	defer file.Close()

	var lines []string
	var removedCount int
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this is a script entry line
		if strings.HasPrefix(strings.TrimSpace(line), "SCR ") {
			// Extract the script path from the line
			// Format: SCR 4 0 "Script: name" "path/to/script.lua"
			parts := strings.Split(line, "\"")
			if len(parts) >= 4 {
				scriptPath := parts[3] // The path is in the 4th quoted section

				// Check if the script file exists
				if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
					// Script file doesn't exist, skip this line (don't add to lines)
					removedCount++
					continue
				}
			}
		}

		// Keep this line
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read reaper-kb.ini: %w", err)
	}

	// If no changes, return early
	if removedCount == 0 {
		return "No missing scripts found in reaper-kb.ini. All script paths are valid.", nil
	}

	// Write back to file
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(kbIniPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write reaper-kb.ini: %w", err)
	}

	return fmt.Sprintf("Cleaned %d missing script(s) from reaper-kb.ini", removedCount), nil
}

// GetContext retrieves the current REAPER context
func (sm *ScriptManager) GetContext() (string, error) {
	// Import context package functionality inline to avoid circular imports
	// We'll call the context reader directly from main.go instead
	return "", fmt.Errorf("GetContext should be called directly from main.go using context package")
}
