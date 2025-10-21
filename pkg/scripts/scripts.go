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

	"github.com/johnjallday/dolphin-reaper-plugin/pkg/platform"
	"github.com/johnjallday/dolphin-reaper-plugin/pkg/types"
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