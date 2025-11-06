package scripts

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Track represents a REAPER track with its properties
type Track struct {
	Index     int     `json:"index"`                // Track index (1-based)
	Name      string  `json:"name"`                 // Track name
	Volume    float64 `json:"volume,omitempty"`     // Volume (dB)
	Pan       float64 `json:"pan,omitempty"`        // Pan (-1.0 to 1.0)
	Mute      bool    `json:"mute,omitempty"`       // Mute state
	Solo      bool    `json:"solo,omitempty"`       // Solo state
	RecArm    bool    `json:"rec_arm,omitempty"`    // Record arm state
	Selected  bool    `json:"selected,omitempty"`   // Selection state
	FXEnabled bool    `json:"fx_enabled,omitempty"` // FX enabled state
}

// WebRemoteClient handles communication with REAPER's Web Remote interface
type WebRemoteClient struct {
	baseURL string
	client  *http.Client
}

// NewWebRemoteClient creates a new Web Remote client
// If port is 0, it will auto-detect from reaper.ini
func NewWebRemoteClient(port int) (*WebRemoteClient, error) {
	if port == 0 {
		// Auto-detect port from reaper.ini
		detectedPort, err := GetWebRemotePort()
		if err != nil {
			return nil, fmt.Errorf("failed to detect web remote port: %w", err)
		}
		port = detectedPort
	}

	return &WebRemoteClient{
		baseURL: fmt.Sprintf("http://localhost:%d", port),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}, nil
}

// GetTracks retrieves all tracks from REAPER via Web Remote API
func (wrc *WebRemoteClient) GetTracks() ([]Track, error) {
	url := wrc.baseURL + "/_/TRACK"

	resp, err := wrc.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to REAPER Web Remote at %s: %w (is REAPER running?)", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("REAPER Web Remote returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	tracks, err := parseTrackData(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse track data: %w", err)
	}

	return tracks, nil
}

// GetTrackNames retrieves just the track names (simplified)
func (wrc *WebRemoteClient) GetTrackNames() ([]string, error) {
	tracks, err := wrc.GetTracks()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(tracks))
	for i, track := range tracks {
		names[i] = track.Name
	}

	return names, nil
}

// parseTrackData parses the REAPER Web Remote TRACK response
// Actual format from REAPER Web Remote API (tab-delimited):
// TRACK\t{index}\t{name}\t{color}\t{volume_mult}\t{pan}\t{?}\t{?}\t{?}\t{?}\t{mute}\t{solo}\t{recarm}\t{?}
// Example: TRACK	1	Tame Impala - Breathe Deeper	8	1.000000	0.000000	-1500	-1500	1.000000	3	0	0	0	0
func parseTrackData(data string) ([]Track, error) {
	lines := strings.Split(strings.TrimSpace(data), "\n")
	tracks := make([]Track, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split by tab
		fields := strings.Split(line, "\t")
		if len(fields) < 13 {
			// Need at least 13 fields for full track data
			continue
		}

		// Field 0: "TRACK" literal (skip)
		if fields[0] != "TRACK" {
			continue
		}

		track := Track{}

		// Field 1: Track index
		if idx, err := strconv.Atoi(fields[1]); err == nil {
			track.Index = idx
		}

		// Field 2: Track name
		track.Name = fields[2]

		// Field 3: Unknown (color?)
		// Skip

		// Field 4: Volume multiplier (convert to dB)
		if volMult, err := strconv.ParseFloat(fields[4], 64); err == nil {
			if volMult > 0 {
				track.Volume = 20 * math.Log10(volMult)
			} else {
				track.Volume = -150.0 // -inf dB for 0 volume
			}
		}

		// Field 5: Pan (-1.0 to 1.0, 0.0 = center)
		if pan, err := strconv.ParseFloat(fields[5], 64); err == nil {
			track.Pan = pan
		}

		// Fields 6-9: Unknown (skip)

		// Field 10: Mute (0 or 1)
		track.Mute = (fields[10] == "1")

		// Field 11: Solo (0, 1, or 2)
		track.Solo = (fields[11] == "1" || fields[11] == "2")

		// Field 12: Record arm (0 or 1)
		track.RecArm = (fields[12] == "1")

		// Always add the track (even if name is empty)
		tracks = append(tracks, track)
	}

	return tracks, nil
}

// GetTracksFromREAPER is a convenience function that auto-detects the port and retrieves tracks
func GetTracksFromREAPER() ([]Track, error) {
	client, err := NewWebRemoteClient(0) // 0 = auto-detect
	if err != nil {
		return nil, err
	}

	return client.GetTracks()
}

// GetTrackNamesFromREAPER is a convenience function that auto-detects the port and retrieves track names
func GetTrackNamesFromREAPER() ([]string, error) {
	client, err := NewWebRemoteClient(0) // 0 = auto-detect
	if err != nil {
		return nil, err
	}

	return client.GetTrackNames()
}

// FormatTracksTable formats tracks as a readable table
func FormatTracksTable(tracks []Track) string {
	if len(tracks) == 0 {
		return "No tracks found in REAPER project"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d tracks:\n\n", len(tracks)))
	result.WriteString("Index | Name                    | Volume  | Pan    | M | S | R\n")
	result.WriteString("------|-------------------------|---------|--------|---|---|---\n")

	for _, track := range tracks {
		// Format flags
		muteFlag := " "
		if track.Mute {
			muteFlag = "M"
		}
		soloFlag := " "
		if track.Solo {
			soloFlag = "S"
		}
		recFlag := " "
		if track.RecArm {
			recFlag = "R"
		}

		// Format pan
		panStr := "Center"
		if track.Pan < -0.01 {
			panStr = fmt.Sprintf("L%.0f%%", -track.Pan*100)
		} else if track.Pan > 0.01 {
			panStr = fmt.Sprintf("R%.0f%%", track.Pan*100)
		}

		result.WriteString(fmt.Sprintf("%-5d | %-23s | %6.1fdB | %-6s | %s | %s | %s\n",
			track.Index,
			truncateString(track.Name, 23),
			track.Volume,
			panStr,
			muteFlag,
			soloFlag,
			recFlag,
		))
	}

	result.WriteString("\nLegend: M=Muted, S=Solo, R=Record Armed")
	return result.String()
}

// FormatTrackNames formats track names as a simple list
func FormatTrackNames(names []string) string {
	if len(names) == 0 {
		return "No tracks found in REAPER project"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d tracks:\n\n", len(names)))

	for i, name := range names {
		result.WriteString(fmt.Sprintf("%d. %s\n", i+1, name))
	}

	return result.String()
}

// truncateString truncates a string to maxLen and adds "..." if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// GetProjectInfo retrieves general project information from REAPER
func (wrc *WebRemoteClient) GetProjectInfo() (map[string]string, error) {
	url := wrc.baseURL + "/_"

	resp, err := wrc.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to REAPER Web Remote: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse project info (basic implementation)
	info := make(map[string]string)
	lines := strings.Split(string(body), "\n")

	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			info[key] = value
		}
	}

	return info, nil
}

// IsWebRemoteRunning checks if REAPER Web Remote is accessible
func IsWebRemoteRunning() bool {
	client, err := NewWebRemoteClient(0)
	if err != nil {
		return false
	}

	url := client.baseURL + "/_"
	resp, err := client.client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
