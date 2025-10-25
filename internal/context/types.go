package context

import "time"

// REAPERContext represents the current state of REAPER
type REAPERContext struct {
	IsRunning   bool      `json:"is_running"`
	ProjectName string    `json:"project_name,omitempty"`
	ProjectPath string    `json:"project_path,omitempty"`
	LastChecked time.Time `json:"last_checked"`
}
