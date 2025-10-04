module github.com/johnjallday/dolphin-reaper-plugin

go 1.24.0

require (
	github.com/johnjallday/dolphin-agent v0.0.0-20250814050009-07ed70c7c8b8
	github.com/openai/openai-go/v2 v2.0.2
	github.com/shirou/gopsutil/v3 v3.24.5
)

replace github.com/johnjallday/dolphin-agent => ../dolphin-agent

require (
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/sys v0.30.0 // indirect
)
