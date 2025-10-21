package main

import (
	"fmt"
	"github.com/johnjallday/dolphin-reaper-plugin/pkg/scripts"
)

func main() {
	downloader := scripts.NewScriptDownloader()
	result, err := downloader.ListAvailableScripts()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Success! Result length: %d\n", len(result))
	fmt.Println(result[:500]) // First 500 chars
}
