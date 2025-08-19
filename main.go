package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	//	"plugin"
	"runtime"
	"strings"

	"github.com/johnjallday/dolphin-agent/pluginapi"
	"github.com/openai/openai-go/v2"
	"github.com/shirou/gopsutil/v3/process"
)

// reaperTool implements pluginapi.Tool for launching ReaScripts (.lua) in REAPER.
type reaperTool struct {
	scriptsDir string   // absolute path to the REAPER Scripts folder
	scripts    []string // script base names (without .lua extension)
}

// Ensure compile-time conformance.
var _ pluginapi.Tool = reaperTool{}

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
	// Build enum of scripts (if we can); if not, leave it empty.
	enum := []string(nil)
	if len(t.scripts) > 0 {
		enum = append(enum, t.scripts...)
	}

	return openai.FunctionDefinitionParam{
		Name:        "reaper_manager",
		Description: openai.String("Manage REAPER ReaScripts: list available scripts or launch them"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"operation": map[string]any{
					"type":        "string",
					"description": "Operation to perform: 'list' to see available scripts, 'run' to launch a script",
					"enum":        []string{"list", "run"},
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

func (t reaperTool) Call(ctx context.Context, args string) (string, error) {
	var p struct {
		Operation string `json:"operation"`
		Script    string `json:"script"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", err
	}

	switch p.Operation {
	case "list":
		return t.handleListScripts()
		//return listLuaScripts(t.scriptsDir)
	case "run":
		return t.handleRunScript(p.Script)
	default:
		return "", fmt.Errorf("unknown operation: %s. Use 'list' or 'run'", p.Operation)
	}
}

func (t reaperTool) handleListScripts() (string, error) {
	// Get fresh list of scripts from the directory
	scripts, err := listLuaScripts(t.scriptsDir)
	if err != nil {
		return "", fmt.Errorf("failed to list scripts in %s: %w", t.scriptsDir, err)
	}

	if len(scripts) == 0 {
		return fmt.Sprintf("No ReaScripts (.lua files) found in: %s", t.scriptsDir), nil
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
		Location:    t.scriptsDir,
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

	result += fmt.Sprintf("\n📂 **Location:** `%s`\n", t.scriptsDir)
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

	if err := launchScript(t.scriptsDir, script); err != nil {
		return "", err
	}
	return fmt.Sprintf("Successfully launched REAPER script: %s", script), nil
}

// Tool is the exported symbol the host looks up via plugin.Open().Lookup("Tool").
var Tool = func() reaperTool {
	dir := defaultScriptsDir()
	scripts, _ := listLuaScripts(dir) // best effort; keep working even if empty
	return reaperTool{
		scriptsDir: dir,
		scripts:    scripts,
	}
}()
