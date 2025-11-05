package types

// Settings represents the REAPER plugin configuration
type Settings struct {
	ScriptsDir     string `json:"scripts_dir"`
	WebRemotePort  int    `json:"web_remote_port"`
}

// AgentsConfig represents the agents.json file structure
type AgentsConfig struct {
	CurrentAgent string `json:"current"`
}

// ScriptItem represents a single script in the list
type ScriptItem struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Action      string `json:"action"`
}

// ScriptList represents a structured list of scripts
type ScriptList struct {
	Type        string       `json:"type"`
	Title       string       `json:"title"`
	Count       int          `json:"count"`
	Location    string       `json:"location"`
	Scripts     []ScriptItem `json:"scripts"`
	Instruction string       `json:"instruction"`
}