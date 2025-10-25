package platform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

// UserHome returns the user's home directory
func UserHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// DefaultScriptsDir returns the default REAPER scripts directory for the current platform
func DefaultScriptsDir() string {
	switch runtime.GOOS {
	case "darwin":
		// macOS: ~/Library/Application Support/REAPER/Scripts
		return filepath.Join(UserHome(), "Library", "Application Support", "REAPER", "Scripts")
	case "windows":
		// Windows: %APPDATA%\REAPER\Scripts
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "REAPER", "Scripts")
		}
		return filepath.Join(UserHome(), "AppData", "Roaming", "REAPER", "Scripts")
	default:
		// Linux: ~/.config/REAPER/Scripts
		return filepath.Join(UserHome(), ".config", "REAPER", "Scripts")
	}
}

// IsReaperRunning checks if REAPER is currently running
func IsReaperRunning() (bool, error) {
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

// LaunchScript launches a REAPER script using platform-specific methods
func LaunchScript(scriptsDir, base string) error {
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