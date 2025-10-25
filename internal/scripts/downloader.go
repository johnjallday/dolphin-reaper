package scripts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/johnjallday/ori-agent/pluginapi"
)

const (
	// GitHub API endpoint for the dev branch reascripts directory
	GitHubAPIURL = "https://api.github.com/repos/johnjallday/ori-reaper/contents/reascripts?ref=dev"
)

// GitHubFile represents a file from GitHub API response
type GitHubFile struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int    `json:"size"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	GitURL      string `json:"git_url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}

// DownloadableScript represents a script available for download
type DownloadableScript struct {
	Name        string `json:"name"`
	Filename    string `json:"filename"`
	Description string `json:"description"`
	Size        string `json:"size"`
	DownloadURL string `json:"downloadUrl"`
}

// ScriptDownloader handles fetching scripts from GitHub
type ScriptDownloader struct {
	apiURL string
}

// NewScriptDownloader creates a new script downloader
func NewScriptDownloader() *ScriptDownloader {
	return &ScriptDownloader{
		apiURL: GitHubAPIURL,
	}
}

// ListAvailableScripts fetches and returns a list of downloadable scripts from GitHub
func (sd *ScriptDownloader) ListAvailableScripts() (string, error) {
	// Fetch files from GitHub API
	files, err := sd.fetchGitHubFiles()
	if err != nil {
		return "", fmt.Errorf("failed to fetch scripts from GitHub: %w", err)
	}

	// Filter to only script files (.lua, .eel, .py)
	var scripts []DownloadableScript
	for _, file := range files {
		if file.Type != "file" {
			continue
		}

		// Check if it's a script file
		name := file.Name
		if !isScriptFile(name) {
			continue
		}

		// Extract display name (remove extension)
		displayName := strings.TrimSuffix(name, ".lua")
		displayName = strings.TrimSuffix(displayName, ".eel")
		displayName = strings.TrimSuffix(displayName, ".py")
		displayName = ToTitleCase(strings.ReplaceAll(displayName, "_", " "))

		// Format file size
		sizeStr := formatFileSize(file.Size)

		// Determine script type/description
		description := getScriptDescription(name)

		scripts = append(scripts, DownloadableScript{
			Name:        displayName,
			Filename:    name,
			Description: description,
			Size:        sizeStr,
			DownloadURL: file.DownloadURL,
		})
	}

	if len(scripts) == 0 {
		return "No scripts found in the repository", nil
	}

	// Convert to modal items format
	modalItems := make([]map[string]interface{}, len(scripts))
	for i, script := range scripts {
		modalItems[i] = map[string]interface{}{
			"name":        script.Name,
			"title":       script.Name,
			"filename":    script.Filename,
			"description": script.Description,
			"size":        script.Size,
			"downloadUrl": script.DownloadURL,
			"index":       i,
		}
	}

	// Create structured modal result for interactive selection
	result := pluginapi.NewModalResult(
		"Available ReaScripts for Download",
		fmt.Sprintf("Found %d scripts in the repository. Click on a script to select it, then click Download.", len(scripts)),
		modalItems,
	)

	// Add metadata for download functionality
	result.Metadata["action"] = "download_script"
	result.Metadata["source"] = "https://github.com/johnjallday/ori-reaper/tree/dev/reascripts"
	result.Metadata["buttonLabel"] = "Download"
	result.Metadata["operation"] = "ori_reaper"

	return result.ToJSON()
}

// fetchGitHubFiles fetches the file list from GitHub API
func (sd *ScriptDownloader) fetchGitHubFiles() ([]GitHubFile, error) {
	resp, err := http.Get(sd.apiURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var files []GitHubFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	return files, nil
}

// isScriptFile checks if a filename is a script file
func isScriptFile(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".lua") ||
		strings.HasSuffix(lower, ".eel") ||
		strings.HasSuffix(lower, ".py")
}

// formatFileSize formats a file size in bytes to a human-readable string
func formatFileSize(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// getScriptDescription returns a description based on the script filename
func getScriptDescription(filename string) string {
	lower := strings.ToLower(filename)

	switch {
	case strings.Contains(lower, "normalize"):
		return "Audio normalization script"
	case strings.Contains(lower, "midi"):
		return "MIDI processing script"
	case strings.Contains(lower, "tempo"):
		return "Tempo manipulation script"
	case strings.Contains(lower, "marker"):
		return "Marker management script"
	case strings.Contains(lower, "render"):
		return "Render/export script"
	case strings.Contains(lower, "track"):
		return "Track management script"
	case strings.Contains(lower, "fx") || strings.Contains(lower, "effect"):
		return "FX/effects script"
	case strings.Contains(lower, "item"):
		return "Item manipulation script"
	case strings.Contains(lower, "region"):
		return "Region management script"
	case strings.HasSuffix(lower, ".lua"):
		return "Lua script"
	case strings.HasSuffix(lower, ".eel"):
		return "EEL script"
	case strings.HasSuffix(lower, ".py"):
		return "Python script"
	default:
		return "ReaScript"
	}
}

// DownloadScript downloads a specific script from GitHub and saves it to the scripts directory
func (sd *ScriptDownloader) DownloadScript(filename, targetDir string) (string, error) {
	// Fetch all files to get the download URL
	files, err := sd.fetchGitHubFiles()
	if err != nil {
		return "", fmt.Errorf("failed to fetch scripts from GitHub: %w", err)
	}

	// Find the requested file
	var downloadURL string
	for _, file := range files {
		if file.Name == filename {
			downloadURL = file.DownloadURL
			break
		}
	}

	if downloadURL == "" {
		return "", fmt.Errorf("script not found: %s", filename)
	}

	// Download the file content
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download script: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read script content: %w", err)
	}

	// Use ScriptManager to add the script
	sm := NewScriptManager(targetDir)

	// Determine script type from extension
	var scriptType string
	switch {
	case strings.HasSuffix(strings.ToLower(filename), ".lua"):
		scriptType = "lua"
	case strings.HasSuffix(strings.ToLower(filename), ".eel"):
		scriptType = "eel"
	case strings.HasSuffix(strings.ToLower(filename), ".py"):
		scriptType = "py"
	default:
		return "", fmt.Errorf("unsupported file type: %s", filename)
	}

	// Remove extension from filename for AddScript
	scriptName := strings.TrimSuffix(filename, ".lua")
	scriptName = strings.TrimSuffix(scriptName, ".eel")
	scriptName = strings.TrimSuffix(scriptName, ".py")

	return sm.AddScript(scriptName, string(content), scriptType)
}
