package context

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/johnjallday/dolphin-reaper-plugin/pkg/platform"
)

// GetREAPERContext retrieves the current REAPER context (project name, state, etc.)
func GetREAPERContext() (*REAPERContext, error) {
	ctx := &REAPERContext{
		LastChecked: time.Now(),
	}

	// Check if REAPER is running
	running, err := platform.IsReaperRunning()
	if err != nil {
		return nil, fmt.Errorf("failed to check if REAPER is running: %w", err)
	}
	ctx.IsRunning = running

	if !running {
		return ctx, nil
	}

	// Get project name and path by executing a temporary Lua script
	projectName, projectPath, err := getProjectInfo()
	if err != nil {
		// REAPER is running but we couldn't get project info
		// This is not a fatal error - return what we have
		return ctx, nil
	}

	ctx.ProjectName = projectName
	ctx.ProjectPath = projectPath

	return ctx, nil
}

// getProjectInfo executes a temporary Lua script in REAPER to get the current project name and path
func getProjectInfo() (string, string, error) {
	// Create a temporary Lua script that writes project info to a temp file
	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "dolphin_get_context.lua")
	outputPath := filepath.Join(tmpDir, "dolphin_context_output.txt")

	// Lua script that gets project info and writes to file
	// Using proper escaping for the path
	escapedOutputPath := strings.ReplaceAll(outputPath, "\\", "\\\\")

	luaScript := fmt.Sprintf(`-- Dolphin Context Reader
-- Get current project info and write to temp file

-- Use EnumProjects to get the current project path and name
-- -1 refers to the currently active project
local retval, project_full_path = reaper.EnumProjects(-1, "")

-- Extract just the filename from the full path
local project_name = "untitled"
local project_path = ""

if project_full_path and project_full_path ~= "" then
    -- Split the full path into directory and filename
    project_name = project_full_path:match("([^/\\]+)$") or "untitled"
    project_path = project_full_path:match("^(.+)[/\\]") or ""
end

-- Write to output file
local file = io.open("%s", "w")
if file then
    file:write(project_name .. "\n")
    file:write(project_path .. "\n")
    file:close()
end
`, escapedOutputPath)

	// Write the Lua script to temp file
	if err := os.WriteFile(scriptPath, []byte(luaScript), 0644); err != nil {
		return "", "", fmt.Errorf("failed to write temp script: %w", err)
	}
	defer os.Remove(scriptPath)

	// Remove old output file if it exists
	os.Remove(outputPath)

	// Execute the script in REAPER using the same method as LaunchScript
	if err := executeScriptInREAPER(scriptPath); err != nil {
		return "", "", fmt.Errorf("failed to execute script in REAPER: %w", err)
	}

	// Wait for REAPER to execute the script and write the file
	time.Sleep(1 * time.Second)

	// Read the output file
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read output file (REAPER may not have executed the script): %w", err)
	}
	// Don't delete output file yet for debugging
	// defer os.Remove(outputPath)

	// Parse the output
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 1 {
		return "", "", fmt.Errorf("unexpected output format: no data")
	}

	projectName := strings.TrimSpace(lines[0])
	projectPath := ""
	if len(lines) >= 2 {
		projectPath = strings.TrimSpace(lines[1])
	}

	// If project name is empty or untitled, indicate no project is open
	if projectName == "" || projectName == "untitled" {
		return "No project open", "", nil
	}

	return projectName, projectPath, nil
}

// executeScriptInREAPER executes a Lua script in REAPER using platform-specific methods
// Uses the same approach as platform.LaunchScript
func executeScriptInREAPER(scriptPath string) error {
	switch runtime.GOOS {
	case "darwin":
		// macOS: open -a Reaper <script>
		cmd := exec.Command("open", "-a", "Reaper", scriptPath)
		return cmd.Run()

	case "windows":
		// Windows: start with the registered app
		cmd := exec.Command("cmd", "/c", "start", "", scriptPath)
		return cmd.Run()

	case "linux":
		// Linux: launch reaper with script
		cmd := exec.Command("reaper", scriptPath)
		return cmd.Run()

	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
