# Web Remote Port Configuration

## Overview

The REAPER Web Remote port is now a **required configuration parameter** for the ori-reaper plugin. Instead of auto-detecting or dynamically setting the port, users must configure the port when setting up the plugin.

## Configuration Approach

### Required Settings

The plugin requires the following configuration:

1. **scripts_dir** - Directory where REAPER scripts are stored
2. **web_remote_port** - Port number for REAPER Web Remote interface

### Configuration Location

Settings are stored per-agent in:
```
agents/<agent-name>/ori-reaper_settings.json
```

Example configuration:
```json
{
  "scripts_dir": "/Users/username/Library/Application Support/REAPER/Scripts",
  "web_remote_port": 8080
}
```

## Setup Process

### 1. First Time Setup

When the plugin is first loaded, ori-agent will prompt for configuration:

**Plugin Configuration UI:**
```
┌─────────────────────────────────────────────────┐
│ Configure ori-reaper Plugin                     │
├─────────────────────────────────────────────────┤
│                                                 │
│ Scripts Directory *                             │
│ ┌─────────────────────────────────────────────┐ │
│ │ /Users/...REAPER/Scripts                    │ │
│ └─────────────────────────────────────────────┘ │
│ Directory where REAPER scripts are stored       │
│                                                 │
│ Web Remote Port *                               │
│ ┌─────────────────────────────────────────────┐ │
│ │ 8080                                        │ │
│ └─────────────────────────────────────────────┘ │
│ Port number for REAPER Web Remote interface     │
│ (must match your REAPER configuration)          │
│                                                 │
│          [Cancel]  [Save Configuration]         │
└─────────────────────────────────────────────────┘
```

### 2. REAPER Configuration

**Ensure REAPER's Web Remote matches the configured port:**

1. Open REAPER
2. Go to **Preferences** (Cmd+, or Ctrl+P)
3. Navigate to **Control/OSC/web**
4. Add or edit **Web browser interface**
5. Set port to **match plugin configuration** (e.g., 8080)
6. Click **OK**
7. Restart REAPER

### 3. Verification

Test the configuration:

```
User: "What's my REAPER web remote port?"

Response:
REAPER Web Remote:
  Configured Port: 8080
  URL: http://localhost:8080
  Note: This port is set in plugin configuration. Ensure REAPER's Web Remote matches this port.
```

Visit `http://localhost:8080` in browser to verify REAPER's Web Remote is accessible.

## Port Validation

### Validation Rules

- **Required**: Port must be specified
- **Type**: Must be an integer
- **Range**: 1024-65535
  - Minimum 1024 (avoid privileged ports)
  - Maximum 65535 (maximum TCP port)

### Validation Errors

**"web_remote_port is required"**
- Port not provided in configuration
- Solution: Set port in plugin configuration

**"web_remote_port must be a number"**
- Invalid port format
- Solution: Provide integer value

**"web_remote_port must be between 1024 and 65535"**
- Port out of valid range
- Solution: Use port between 1024-65535

## Usage

### Operations Using Configured Port

All Web Remote operations use the configured port:

#### Get Tracks
```
User: "Show me all tracks in REAPER"

Plugin: Uses configured port (8080) to connect to Web Remote
Response: Lists all tracks from REAPER
```

#### Get Web Remote Port
```
User: "What's my web remote port?"

Response:
REAPER Web Remote:
  Configured Port: 8080
  URL: http://localhost:8080
  Note: This port is set in plugin configuration. Ensure REAPER's Web Remote matches this port.
```

### Updating Configuration

To change the port:

1. **Via UI Settings:**
   - Open plugin settings in ori-agent
   - Update "Web Remote Port" field
   - Save configuration

2. **Via JSON File:**
   - Edit `agents/<agent-name>/ori-reaper_settings.json`
   - Update `web_remote_port` value
   - Restart ori-agent or reload plugin

## Benefits of Configuration Approach

### 1. Explicit Configuration
✅ **Clear expectations** - User knows exactly which port to use
✅ **No auto-detection failures** - Port is always known
✅ **Easier troubleshooting** - Configuration is visible and explicit

### 2. Consistency
✅ **Single source of truth** - Port stored in one location
✅ **Works across sessions** - Configuration persists
✅ **Per-agent configuration** - Different agents can use different ports

### 3. Simplified Operations
✅ **No port detection logic** - Operations simply use configured port
✅ **Faster execution** - No need to read reaper.ini for every operation
✅ **Reduced errors** - Fewer failure points

### 4. Better User Experience
✅ **Configuration UI** - Easy setup through ori-agent interface
✅ **Validation** - Port validated before saving
✅ **Default values** - Sensible defaults provided (8080)

## Comparison with Previous Approach

### Previous: Auto-Detection + Dynamic Setting

**Operations:**
- `get_web_remote_port` - Read port from reaper.ini
- `set_web_remote_port` - Create new csurf entry
- `get_tracks` - Auto-detect port, then fetch tracks

**Issues:**
- ❌ Auto-detection could fail if reaper.ini format changed
- ❌ Required parsing reaper.ini for every operation
- ❌ `set_web_remote_port` created entries but REAPER needed restart
- ❌ Confusion about which port was "active"
- ❌ Multiple Web Remote entries could exist with different ports

### Current: Required Configuration

**Operations:**
- `get_web_remote_port` - Show configured port
- `get_tracks` - Use configured port directly

**Benefits:**
- ✅ Simple, explicit configuration
- ✅ No reaper.ini parsing needed
- ✅ Clear which port to use
- ✅ User controls REAPER configuration directly
- ✅ Consistent behavior

## Troubleshooting

### Issue: "Failed to connect to REAPER Web Remote"

**Causes:**
1. REAPER Web Remote disabled
2. REAPER not running
3. Port mismatch between plugin config and REAPER

**Solutions:**
1. Enable Web Remote in REAPER preferences
2. Start REAPER
3. Verify port matches:
   ```bash
   # Check REAPER configuration
   grep "csurf_" ~/Library/Application\ Support/REAPER/reaper.ini

   # Check plugin configuration
   cat agents/<agent-name>/ori-reaper_settings.json
   ```

### Issue: "Connection refused"

**Causes:**
- Wrong port number
- REAPER Web Remote not enabled
- Firewall blocking connection

**Solutions:**
1. Verify port in browser: `http://localhost:8080`
2. Check REAPER preferences → Control/OSC/web
3. Check firewall settings

### Issue: "Cannot update configuration"

**Causes:**
- Plugin already initialized
- Configuration validation failed

**Solutions:**
1. Update via ori-agent settings UI
2. Manually edit settings JSON file
3. Restart ori-agent after changes

## Implementation Details

### Files Modified

1. **internal/types/types.go**
   - Added `WebRemotePort int` field to `Settings` struct

2. **internal/settings/settings.go**
   - Updated `GetDefaultSettings()` to include default port (8080)
   - Added `GetWebRemotePort()` method

3. **main.go**
   - Added `web_remote_port` to `GetRequiredConfig()`
   - Added port validation in `ValidateConfig()`
   - Updated `get_web_remote_port` to return configured port
   - Updated `get_tracks` to use configured port
   - **Removed** `set_web_remote_port` operation

### Configuration Schema

```go
type Settings struct {
    ScriptsDir     string `json:"scripts_dir"`
    WebRemotePort  int    `json:"web_remote_port"`
}
```

### Required Config Variable

```go
{
    Key:          "web_remote_port",
    Name:         "Web Remote Port",
    Description:  "Port number for REAPER Web Remote interface (must match your REAPER configuration)",
    Type:         pluginapi.ConfigTypeInt,
    Required:     true,
    DefaultValue: "8080",
    Placeholder:  "8080",
}
```

## Migration Guide

### For Existing Users

If you were using the plugin before this change:

1. **Reload the plugin** in ori-agent
2. **Configure the port** when prompted
   - Use the port shown in REAPER preferences
   - Default: 8080
3. **Verify REAPER matches** the configured port
4. **Test** with "show me all tracks in REAPER"

### For New Users

1. **Install plugin** via ori-agent
2. **Configure during setup:**
   - Scripts directory
   - Web Remote port (default 8080)
3. **Configure REAPER** to match:
   - Preferences → Control/OSC/web
   - Add Web browser interface
   - Set port to match plugin config
4. **Test connection**

## Best Practices

### 1. Use Standard Ports

**Recommended ports:**
- 8080 (most common, default)
- 3000-9999 (user ports)

**Avoid:**
- 1-1023 (privileged ports)
- Well-known ports (3306 MySQL, 5432 PostgreSQL)

### 2. Document Your Port

Keep track of which port you configured:
- Write it down
- Add to documentation
- Include in project README

### 3. Consistent Configuration

If using multiple agents:
- Use different ports for each agent
- Or use same port if agents don't run simultaneously

### 4. Test After Changes

After updating configuration:
1. Save changes
2. Test connection: `http://localhost:<port>`
3. Verify plugin operations work

## Related Documentation

- `WEB_REMOTE_PORT_FEATURE.md` - Previous port detection feature (deprecated)
- `TRACK_RETRIEVAL_FEATURE.md` - Track retrieval using Web Remote
- `COMPLETE_FEATURE_SUMMARY.md` - Complete feature overview

## Changelog

**Version 0.0.7** (2025-10-28)
- ✅ Made `web_remote_port` a required configuration parameter
- ✅ Removed auto-detection from reaper.ini
- ✅ Removed `set_web_remote_port` operation (no longer needed)
- ✅ Simplified `get_web_remote_port` to show configured port
- ✅ Updated `get_tracks` to use configured port directly
- ✅ Added port validation (1024-65535)
- ✅ Added configuration UI via `GetRequiredConfig()`

---

**Status:** ✅ Complete and Production Ready
**Version:** 0.0.7
**Last Updated:** 2025-10-28
