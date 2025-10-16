package scripts

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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

// Manager handles script operations
type Manager struct {
	scriptsDir string
}

// NewManager creates a new script manager with the given scripts directory
func NewManager(scriptsDir string) *Manager {
	return &Manager{scriptsDir: scriptsDir}
}

// ListScripts returns a structured list of available scripts
func (m *Manager) ListScripts() (string, error) {
	// Get fresh list of scripts from the directory
	scripts, err := ListLuaScripts(m.scriptsDir)
	if err != nil {
		return "", fmt.Errorf("failed to list scripts in %s: %w", m.scriptsDir, err)
	}

	if len(scripts) == 0 {
		return fmt.Sprintf("No ReaScripts (.lua files) found in: %s", m.scriptsDir), nil
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
		Location:    m.scriptsDir,
		Scripts:     scriptItems,
		Instruction: "To run a script, say: \"Run the [script_name] script\"",
	}

	// Return as JSON string with special prefix to indicate structured data
	jsonData, err := json.Marshal(result)
	if err != nil {
		// Fallback to markdown format if JSON marshaling fails
		return m.listScriptsMarkdown(scripts)
	}

	return "STRUCTURED_DATA:" + string(jsonData), nil
}

// listScriptsMarkdown returns a markdown-formatted list of scripts
func (m *Manager) listScriptsMarkdown(scripts []string) (string, error) {
	// Fallback markdown format
	result := fmt.Sprintf("## 🎵 Available REAPER Scripts (%d found)\n\n", len(scripts))
	result += "| # | Script Name | Action |\n"
	result += "|---|-------------|--------|\n"

	for i, script := range scripts {
		displayName := strings.ReplaceAll(script, "_", " ")
		displayName = ToTitleCase(displayName)
		result += fmt.Sprintf("| %d | **%s** | `%s` |\n", i+1, displayName, script)
	}

	result += fmt.Sprintf("\n📂 **Location:** `%s`\n", m.scriptsDir)
	result += "\n💡 **To run a script, say:** *\"Run the [script_name] script\"*"

	return result, nil
}

// RunScript launches a script in REAPER
func (m *Manager) RunScript(script string) (string, error) {
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

	if err := platform.LaunchScript(m.scriptsDir, script); err != nil {
		return "", err
	}
	return fmt.Sprintf("Successfully launched REAPER script: %s", script), nil
}

// DeleteScript deletes a script file from the scripts directory
func (m *Manager) DeleteScript(script string) (string, error) {
	if strings.TrimSpace(script) == "" {
		return "", errors.New("script name is required for 'delete' operation")
	}

	// Add .lua extension if not present
	scriptFile := script
	if !strings.HasSuffix(strings.ToLower(scriptFile), ".lua") {
		scriptFile = script + ".lua"
	}

	// Construct full path
	scriptPath := fmt.Sprintf("%s/%s", m.scriptsDir, scriptFile)

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