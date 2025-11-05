# Web Remote Port Auto-Detection

## Overview

The ori-reaper plugin now automatically detects existing REAPER Web Remote configuration and uses it as the default port value during plugin setup. This provides a smarter, more user-friendly configuration experience.

## How It Works

### Auto-Detection Process

When the plugin configuration UI loads, it:

1. **Scans reaper.ini** for existing Web Remote entries
2. **Detects port and status** (enabled/disabled)
3. **Sets intelligent defaults** based on findings
4. **Provides contextual messages** to guide the user

### Detection Logic

```go
func GetRequiredConfig() []pluginapi.ConfigVariable {
    // Try to detect existing web remote port from reaper.ini
    if config, err := scripts.GetWebRemoteConfig(); err == nil {
        // Found existing configuration - use it
        webRemotePort = fmt.Sprintf("%d", config.Port)

        if config.Enabled {
            // Web Remote is enabled
            portDescription = "Auto-detected from REAPER (enabled on port X)"
        } else {
            // Web Remote exists but disabled
            portDescription = "Auto-detected from REAPER (disabled, port X)"
        }
    } else {
        // No Web Remote found - use default
        webRemotePort = "2307"
        portDescription = "No Web Remote found. Using default port 2307."
    }
}
```

## User Experience Scenarios

### Scenario 1: Web Remote Already Configured and Enabled

**REAPER Configuration:**
```ini
csurf_0=HTTP 1 8080 '' 'index.html' 0 ''
```

**Plugin Configuration UI:**
```
┌─────────────────────────────────────────────────┐
│ Configure ori-reaper Plugin                     │
├─────────────────────────────────────────────────┤
│                                                 │
│ Web Remote Port *                               │
│ ┌─────────────────────────────────────────────┐ │
│ │ 8080                                        │ │
│ └─────────────────────────────────────────────┘ │
│ Auto-detected from REAPER configuration         │
│ (currently enabled on port 8080).               │
│ No need to change unless you modify REAPER's    │
│ Web Remote settings.                            │
│                                                 │
│          [Cancel]  [Save Configuration]         │
└─────────────────────────────────────────────────┘
```

**User Action:** Simply click "Save Configuration" - no changes needed!

### Scenario 2: Web Remote Configured but Disabled

**REAPER Configuration:**
```ini
csurf_0=HTTP 0 2307 '' 'index.html' 0 ''
```

**Plugin Configuration UI:**
```
┌─────────────────────────────────────────────────┐
│ Configure ori-reaper Plugin                     │
├─────────────────────────────────────────────────┤
│                                                 │
│ Web Remote Port *                               │
│ ┌─────────────────────────────────────────────┐ │
│ │ 2307                                        │ │
│ └─────────────────────────────────────────────┘ │
│ Auto-detected from REAPER configuration         │
│ (currently disabled, port 2307).                │
│ This port will be used when you enable          │
│ Web Remote in REAPER.                           │
│                                                 │
│          [Cancel]  [Save Configuration]         │
└─────────────────────────────────────────────────┘
```

**User Action:**
1. Click "Save Configuration"
2. Enable Web Remote in REAPER if needed

### Scenario 3: No Web Remote Configuration

**REAPER Configuration:**
```ini
# No csurf entries with HTTP or WEBR
```

**Plugin Configuration UI:**
```
┌─────────────────────────────────────────────────┐
│ Configure ori-reaper Plugin                     │
├─────────────────────────────────────────────────┤
│                                                 │
│ Web Remote Port *                               │
│ ┌─────────────────────────────────────────────┐ │
│ │ 2307                                        │ │
│ └─────────────────────────────────────────────┘ │
│ No Web Remote found in REAPER configuration.    │
│ Using default port 2307.                        │
│ Configure Web Remote in REAPER                  │
│ (Preferences → Control/OSC/web) to match this   │
│ port.                                           │
│                                                 │
│          [Cancel]  [Save Configuration]         │
└─────────────────────────────────────────────────┘
```

**User Action:**
1. Click "Save Configuration"
2. Set up Web Remote in REAPER on port 2307

## Benefits

### 1. Zero Configuration for Existing Users
✅ **No manual input needed** - Port automatically matches REAPER
✅ **One-click setup** - Just save the detected configuration
✅ **No confusion** - Clear what port to use

### 2. Smart Defaults for New Users
✅ **Sensible default** - Port 2307 if no Web Remote exists
✅ **Clear guidance** - Instructions on setting up REAPER
✅ **Reduced errors** - Less chance of port mismatch

### 3. Context-Aware Messaging
✅ **Different messages** - Based on detection results
✅ **Helpful hints** - Tells user if Web Remote is enabled/disabled
✅ **Clear next steps** - Guidance on what to do

### 4. Intelligent Detection
✅ **Reads reaper.ini** - Actual REAPER configuration
✅ **Detects status** - Knows if enabled or disabled
✅ **Supports both formats** - HTTP and WEBR control surfaces

## Technical Implementation

### Detection Function

Uses existing `scripts.GetWebRemoteConfig()` which:
- Reads platform-specific reaper.ini location
- Parses csurf entries
- Identifies HTTP/WEBR control surfaces
- Returns port and enabled status

### Configuration Variable

```go
{
    Key:          "web_remote_port",
    Name:         "Web Remote Port",
    Description:  portDescription,  // Dynamic based on detection
    Type:         pluginapi.ConfigTypeInt,
    Required:     true,
    DefaultValue: webRemotePort,    // Auto-detected or 2307
    Placeholder:  webRemotePort,
}
```

### Dynamic Descriptions

**If Web Remote found and enabled:**
```
"Auto-detected from REAPER configuration (currently enabled on port {PORT}).
No need to change unless you modify REAPER's Web Remote settings."
```

**If Web Remote found but disabled:**
```
"Auto-detected from REAPER configuration (currently disabled, port {PORT}).
This port will be used when you enable Web Remote in REAPER."
```

**If no Web Remote found:**
```
"No Web Remote found in REAPER configuration. Using default port 2307.
Configure Web Remote in REAPER (Preferences → Control/OSC/web) to match this port."
```

## Testing

### Test Case 1: Existing Enabled Web Remote

**Setup:**
```bash
# Set up REAPER with enabled Web Remote on port 8080
echo "csurf_0=HTTP 1 8080 '' 'index.html' 0 ''" >> reaper.ini
```

**Expected:**
- Default value: `8080`
- Description: "Auto-detected...currently enabled on port 8080"

### Test Case 2: Existing Disabled Web Remote

**Setup:**
```bash
# Set up REAPER with disabled Web Remote on port 2307
echo "csurf_0=HTTP 0 2307 '' 'index.html' 0 ''" >> reaper.ini
```

**Expected:**
- Default value: `2307`
- Description: "Auto-detected...currently disabled, port 2307"

### Test Case 3: No Web Remote

**Setup:**
```bash
# Remove all Web Remote entries from reaper.ini
sed -i '' '/csurf_.*HTTP/d' reaper.ini
```

**Expected:**
- Default value: `2307`
- Description: "No Web Remote found...Using default port 2307"

### Test Case 4: Multiple Web Remote Entries

**Setup:**
```bash
# Add multiple Web Remote entries
echo "csurf_0=HTTP 0 2307 '' 'index.html' 0 ''" >> reaper.ini
echo "csurf_1=HTTP 1 8080 '' 'fancier.html' 0 ''" >> reaper.ini
```

**Expected:**
- Default value: `2307` (first entry)
- Description: "Auto-detected...currently disabled, port 2307"

## Why 2307 as Default?

When no Web Remote is found, the plugin uses **2307** as the default port:

### Reasons:
1. **Common REAPER default** - Many REAPER installations use 2307
2. **Avoids conflicts** - Less likely to conflict with other services than 8080
3. **User-friendly range** - In the 2000-3000 range commonly used for web services
4. **Not well-known** - Avoids well-known service ports

### Alternative Ports:
- **8080** - Very common, but may conflict with other services
- **3000** - Common for dev servers
- **5000** - Common for Flask, etc.
- **2307** - Good balance of availability and convention

## Comparison: Before vs After

### Before (Static Default)

**Configuration UI:**
```
Web Remote Port: [8080]
Description: "Port number for REAPER Web Remote interface
             (must match your REAPER configuration)"
```

**Issues:**
- ❌ User has to manually check REAPER configuration
- ❌ Risk of port mismatch
- ❌ No guidance if REAPER not configured
- ❌ Generic message, not helpful

### After (Auto-Detection)

**Configuration UI (with detection):**
```
Web Remote Port: [2307]
Description: "Auto-detected from REAPER configuration
             (currently disabled, port 2307).
             This port will be used when you enable Web Remote in REAPER."
```

**Benefits:**
- ✅ Automatically matches REAPER
- ✅ No port mismatch possible
- ✅ Clear guidance on next steps
- ✅ Context-aware messaging

## Error Handling

### If reaper.ini Cannot Be Read

**Fallback behavior:**
- Use default port: `2307`
- Show default message: "No Web Remote found..."
- User can still configure manually

**Logging:**
```
[ori-reaper] Warning: Could not read reaper.ini, using default port 2307
```

### If Invalid Port Detected

**Validation:**
- Port must be 1024-65535
- If invalid, fallback to 2307
- Log warning

**Example:**
```go
if config.Port < 1024 || config.Port > 65535 {
    log.Printf("[ori-reaper] Invalid port %d detected, using default 2307", config.Port)
    webRemotePort = "2307"
}
```

## User Workflow

### Happy Path (Web Remote Already Configured)

1. **Install plugin** → ori-agent prompts for configuration
2. **See auto-detected port** → "8080 (enabled)"
3. **Click Save** → Configuration complete
4. **Use plugin** → Immediately works with REAPER

**Time to setup:** ~5 seconds

### Setup Path (No Web Remote)

1. **Install plugin** → ori-agent prompts for configuration
2. **See default port** → "2307 (no Web Remote found)"
3. **Click Save** → Configuration saved
4. **Configure REAPER** → Set Web Remote to port 2307
5. **Use plugin** → Works after REAPER configuration

**Time to setup:** ~2 minutes (including REAPER setup)

## Documentation Updates

### Files Modified

- `main.go:198-240` - Added auto-detection logic to `GetRequiredConfig()`
- `PORT_AUTO_DETECTION.md` - This documentation (new)

### Related Documentation

- `WEB_REMOTE_PORT_CONFIGURATION.md` - Configuration overview
- `COMPLETE_FEATURE_SUMMARY.md` - Feature summary
- `TRACK_RETRIEVAL_FEATURE.md` - Track retrieval using Web Remote

## Changelog

**Version 0.0.8** (2025-10-28)
- ✅ Added auto-detection of existing Web Remote configuration
- ✅ Dynamic default values based on detection
- ✅ Context-aware configuration messages
- ✅ Intelligent fallback to port 2307 if no Web Remote found
- ✅ Detects both enabled and disabled Web Remote states
- ✅ Provides clear guidance based on detection results

## Future Enhancements

### Potential Improvements

1. **Multiple Web Remote Support**
   - Detect all Web Remote entries
   - Let user choose which one to use
   - Support multiple simultaneous connections

2. **Real-Time Validation**
   - Test connection during configuration
   - Verify Web Remote is accessible
   - Show live status indicator

3. **Auto-Enable Web Remote**
   - Offer to enable Web Remote in REAPER
   - Modify reaper.ini if disabled
   - Restart REAPER notification

4. **Configuration Sync**
   - Monitor reaper.ini for changes
   - Update plugin config automatically
   - Notify user of port changes

---

**Status:** ✅ Complete and Production Ready
**Version:** 0.0.8
**Last Updated:** 2025-10-28
