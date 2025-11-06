package scripts

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// GetReaperIniPath returns the platform-specific path to reaper.ini
func GetReaperIniPath() (string, error) {
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

	iniPath := filepath.Join(basePath, "reaper.ini")

	// Check if the file exists
	if _, err := os.Stat(iniPath); os.IsNotExist(err) {
		return "", fmt.Errorf("reaper.ini not found at %s (is REAPER installed?)", iniPath)
	}

	return iniPath, nil
}

// WebRemoteConfig represents the web remote control surface configuration
type WebRemoteConfig struct {
	Port      int    `json:"port"`
	Enabled   bool   `json:"enabled"`
	CSurfID   int    `json:"csurf_id"`   // The csurf_N index
	RawConfig string `json:"raw_config"` // The full csurf line
}

// GetWebRemotePort reads reaper.ini and extracts the web remote port from csurf entries
// Returns the port number, or an error if not found
func GetWebRemotePort() (int, error) {
	config, err := GetWebRemoteConfig()
	if err != nil {
		return 0, err
	}
	return config.Port, nil
}

// GetWebRemoteConfig reads reaper.ini and extracts the full web remote configuration
// REAPER's web remote is configured as a control surface entry like:
// csurf_0=HTTP 0 2307 ” 'index.html' 0 ”
// or older format:
// csurf_0=WEBR 0 0 0 0 0 0 - - - - - 8080
func GetWebRemoteConfig() (*WebRemoteConfig, error) {
	iniPath, err := GetReaperIniPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(iniPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open reaper.ini: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Look for csurf entries: csurf_0=HTTP 0 2307 '' 'index.html' 0 ''
		if strings.HasPrefix(trimmed, "csurf_") {
			// Parse the line
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) != 2 {
				continue
			}

			csurfKey := parts[0]   // e.g., "csurf_0"
			csurfValue := parts[1] // e.g., "HTTP 0 2307 '' 'index.html' 0 ''"

			// Check if this is a web remote entry (starts with "HTTP" or "WEBR")
			if strings.HasPrefix(csurfValue, "HTTP ") || strings.HasPrefix(csurfValue, "WEBR ") {
				// Extract the csurf ID number
				csurfIDStr := strings.TrimPrefix(csurfKey, "csurf_")
				csurfID, err := strconv.Atoi(csurfIDStr)
				if err != nil {
					continue
				}

				// Parse the web remote configuration
				fields := strings.Fields(csurfValue)
				if len(fields) < 3 {
					continue
				}

				var port int
				var enabled bool

				if strings.HasPrefix(csurfValue, "HTTP ") {
					// Format: HTTP <enabled> <port> '' 'index.html' 0 ''
					// Field 0: HTTP
					// Field 1: enabled (0 or 1)
					// Field 2: port number
					if len(fields) >= 3 {
						enabledVal := fields[1]
						enabled = (enabledVal == "1" || enabledVal == "true")

						portStr := fields[2]
						port, err = strconv.Atoi(portStr)
						if err != nil {
							continue
						}
					}
				} else {
					// Format: WEBR <enabled> <flags...> <port>
					// The port is typically the last field
					portStr := fields[len(fields)-1]
					port, err = strconv.Atoi(portStr)
					if err != nil {
						continue
					}

					if len(fields) >= 2 {
						enabledVal := fields[1]
						enabled = (enabledVal == "1" || enabledVal == "true")
					}
				}

				return &WebRemoteConfig{
					Port:      port,
					Enabled:   enabled,
					CSurfID:   csurfID,
					RawConfig: csurfValue,
				}, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading reaper.ini: %w", err)
	}

	return nil, errors.New("web remote (HTTP/WEBR) control surface not found in reaper.ini - make sure Web Remote is enabled in REAPER preferences")
}

// GetAllCSurfEntries reads all control surface entries from reaper.ini
// Returns a map of csurf_N -> configuration string
func GetAllCSurfEntries() (map[string]string, error) {
	iniPath, err := GetReaperIniPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(iniPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open reaper.ini: %w", err)
	}
	defer file.Close()

	csurfEntries := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Look for csurf entries
		if strings.HasPrefix(trimmed, "csurf_") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				csurfKey := parts[0]
				csurfValue := parts[1]
				csurfEntries[csurfKey] = csurfValue
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading reaper.ini: %w", err)
	}

	return csurfEntries, nil
}

// ParseCSurfEntry parses a csurf value and returns the control surface type and configuration
type CSurfEntry struct {
	Type   string   `json:"type"`   // e.g., "WEBR", "OSCII", "MCU"
	Fields []string `json:"fields"` // All fields after the type
	ID     int      `json:"id"`     // The csurf_N number
}

// ParseCSurfEntries parses all csurf entries and returns structured data
func ParseCSurfEntries() ([]CSurfEntry, error) {
	allEntries, err := GetAllCSurfEntries()
	if err != nil {
		return nil, err
	}

	var parsed []CSurfEntry
	for key, value := range allEntries {
		// Extract ID from csurf_N
		idStr := strings.TrimPrefix(key, "csurf_")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}

		// Parse the value
		fields := strings.Fields(value)
		if len(fields) < 1 {
			continue
		}

		csurfType := fields[0]
		csurfFields := fields[1:]

		parsed = append(parsed, CSurfEntry{
			Type:   csurfType,
			Fields: csurfFields,
			ID:     id,
		})
	}

	return parsed, nil
}

// SetWebRemotePort creates a new web remote control surface entry with the specified port
// Instead of modifying existing entries, this creates a new csurf_N entry
func SetWebRemotePort(newPort int) error {
	iniPath, err := GetReaperIniPath()
	if err != nil {
		return err
	}

	// Read the entire file
	file, err := os.Open(iniPath)
	if err != nil {
		return fmt.Errorf("failed to open reaper.ini: %w", err)
	}
	defer file.Close()

	var lines []string
	var maxCSurfID int = -1
	var csurfCntLineIndex int = -1
	var insertIndex int = -1
	scanner := bufio.NewScanner(file)
	lineIndex := 0

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Track the highest csurf_N number
		if strings.HasPrefix(trimmed, "csurf_") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				csurfKey := parts[0]
				// Extract ID from csurf_N
				idStr := strings.TrimPrefix(csurfKey, "csurf_")
				if id, err := strconv.Atoi(idStr); err == nil {
					if id > maxCSurfID {
						maxCSurfID = id
					}
					// Remember where to insert (after the last csurf_N entry)
					insertIndex = lineIndex + 1
				}
			}
		}

		// Track csurf_cnt line for updating
		if strings.HasPrefix(trimmed, "csurf_cnt=") {
			csurfCntLineIndex = lineIndex
		}

		lines = append(lines, line)
		lineIndex++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading reaper.ini: %w", err)
	}

	// Create new csurf entry
	newCSurfID := maxCSurfID + 1
	newCSurfLine := fmt.Sprintf("csurf_%d=HTTP 1 %d '' 'index.html' 0 ''", newCSurfID, newPort)

	// Insert the new line
	if insertIndex == -1 {
		// No existing csurf entries, append at end
		lines = append(lines, newCSurfLine)
	} else {
		// Insert after last csurf entry
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:insertIndex]...)
		newLines = append(newLines, newCSurfLine)
		newLines = append(newLines, lines[insertIndex:]...)
		lines = newLines
		// Adjust csurfCntLineIndex if needed
		if csurfCntLineIndex >= insertIndex {
			csurfCntLineIndex++
		}
	}

	// Update csurf_cnt if it exists
	if csurfCntLineIndex != -1 {
		// csurf_cnt appears to be the highest index, not the total count
		// So set it to the new highest ID
		lines[csurfCntLineIndex] = fmt.Sprintf("csurf_cnt=%d", newCSurfID)
	} else {
		// Add csurf_cnt if it doesn't exist
		lines = append(lines, fmt.Sprintf("csurf_cnt=%d", newCSurfID))
	}

	// Write the file back
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(iniPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write reaper.ini: %w", err)
	}

	return nil
}

// SetWebRemoteEnabled enables or disables the web remote in reaper.ini
func SetWebRemoteEnabled(enabled bool) error {
	iniPath, err := GetReaperIniPath()
	if err != nil {
		return err
	}

	// Read the entire file
	file, err := os.Open(iniPath)
	if err != nil {
		return fmt.Errorf("failed to open reaper.ini: %w", err)
	}
	defer file.Close()

	var lines []string
	var modified bool
	scanner := bufio.NewScanner(file)

	enabledVal := "0"
	if enabled {
		enabledVal = "1"
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Look for csurf entries
		if !modified && strings.HasPrefix(trimmed, "csurf_") {
			// Parse the line
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				csurfKey := parts[0]
				csurfValue := parts[1]

				// Check if this is a web remote entry
				if strings.HasPrefix(csurfValue, "HTTP ") || strings.HasPrefix(csurfValue, "WEBR ") {
					fields := strings.Fields(csurfValue)

					if len(fields) >= 2 {
						// Field 1 is always the enabled flag
						fields[1] = enabledVal

						// Reconstruct the line
						newValue := strings.Join(fields, " ")
						line = csurfKey + "=" + newValue
						modified = true
					}
				}
			}
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading reaper.ini: %w", err)
	}

	if !modified {
		return errors.New("web remote (HTTP/WEBR) control surface not found in reaper.ini")
	}

	// Write the file back
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(iniPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write reaper.ini: %w", err)
	}

	return nil
}
