//go:build ignore

package main

import (
	"fmt"
	"github.com/johnjallday/ori-agent/internal/pluginloader"
	"github.com/johnjallday/ori-agent/pluginapi"
)

func main() {
	// Load the plugin
	tool, err := pluginloader.LoadPluginUnified("./ori-reaper")
	if err != nil {
		fmt.Printf("Error loading plugin: %v\n", err)
		return
	}

	// Check if it implements InitializationProvider
	if initProvider, ok := tool.(pluginapi.InitializationProvider); ok {
		fmt.Println("✓ Plugin implements InitializationProvider")

		// Get required config
		configVars := initProvider.GetRequiredConfig()
		fmt.Printf("✓ Required config fields: %d\n", len(configVars))
		for _, cv := range configVars {
			fmt.Printf("  - %s (%s): %s\n", cv.Key, cv.Type, cv.Description)
		}
	} else {
		fmt.Println("✗ Plugin DOES NOT implement InitializationProvider")
	}
}
