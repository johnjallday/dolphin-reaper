# Conditional Configuration: Smart Port Detection

## Overview

The ori-reaper plugin uses **conditional configuration** - if REAPER already has a Web Remote configured, the port does NOT need to be configured in the plugin. It automatically detects and uses the existing port from REAPER's configuration.

## How It Works

### Configuration Detection Flow

```
Plugin Initialization
    ↓
Scan reaper.ini for Web Remote
    ↓
┌─────────────────────────────────┐
│ Web Remote Found?               │
└─────────────────────────────────┘
        ↓              ↓
       YES            NO
        ↓              ↓
   ┌────────┐    ┌─────────┐
   │ Skip   │    │ Require │
   │ Port   │    │ Port    │
   │ Config │    │ Config  │
   └────────┘    └─────────┘
        ↓              ↓
   Auto-use      User sets
   detected      port 2307
   port          (default)
        ↓              ↓
   ┌──────────────────────┐
   │ Plugin Ready         │
   └──────────────────────┘
```

## User Experience

### Scenario 1: Web Remote Already Configured (Your Case)

**REAPER has:**
```ini
csurf_0=HTTP 0 2307 '' 'index.html' 0 ''
```

**Plugin Configuration UI:**
```
┌─────────────────────────────────────┐
│ Configure ori-reaper Plugin         │
├─────────────────────────────────────┤
│                                     │
│ Scripts Directory * [Required]      │
│ ┌─────────────────────────────────┐ │
│ │ /Users/.../REAPER/Scripts       │ │
│ └─────────────────────────────────┘ │
│                                     │
│ [Save Configuration]                │
└─────────────────────────────────────┘
```

**Notice:** NO web remote port field! ✨

**Result:**
- Plugin automatically uses port **2307** from REAPER
- No manual configuration needed
- Zero-config experience

### Scenario 2: No Web Remote in REAPER

**REAPER has:**
```ini
# No HTTP or WEBR csurf entries
```

**Plugin Configuration UI:**
```
┌─────────────────────────────────────┐
│ Configure ori-reaper Plugin         │
├─────────────────────────────────────┤
│                                     │
│ Scripts Directory * [Required]      │
│ ┌─────────────────────────────────┐ │
│ │ /Users/.../REAPER/Scripts       │ │
│ └─────────────────────────────────┘ │
│                                     │
│ Web Remote Port * [Required]        │
│ ┌─────────────────────────────────┐ │
│ │ 2307                            │ │
│ └─────────────────────────────────┘ │
│ No Web Remote found in REAPER       │
│ configuration. Please specify the   │
│ port you want to use.               │
│                                     │
│ [Save Configuration]                │
└─────────────────────────────────────┘
```

**Notice:** Port field IS shown because no Web Remote detected

**Result:**
- User configures port (default: 2307)
- User then sets up REAPER to match this port
- Port stored in plugin settings

## API Behavior

### GET `/api/plugins/ori-reaper/config`

**With Web Remote Detected:**
```json
{
  "required_config": [
    {
      "key": "scripts_dir",
      "name": "Scripts Directory",
      "type": "dirpath",
      "required": true,
      "default_value": "/Users/jj/Library/Application Support/REAPER/Scripts"
    }
  ],
  "is_initialized": false,
  "supports_initialization": true
}
```

**Notice:** `web_remote_port` is NOT in the required_config array!

**Without Web Remote Detected:**
```json
{
  "required_config": [
    {
      "key": "scripts_dir",
      "name": "Scripts Directory",
      "type": "dirpath",
      "required": true,
      "default_value": "/Users/jj/Library/Application Support/REAPER/Scripts"
    },
    {
      "key": "web_remote_port",
      "name": "Web Remote Port",
      "type": "int",
      "required": true,
      "default_value": "2307",
      "description": "No Web Remote found in REAPER configuration..."
    }
  ],
  "is_initialized": false,
  "supports_initialization": true
}
```

**Notice:** `web_remote_port` IS in the required_config array

## Port Resolution Logic

### GetWebRemotePort() Method

The settings manager uses smart fallback logic:

```go
func (sm *Manager) GetWebRemotePort() int {
    settings := sm.GetCurrentSettings()

    // 1. Check if port is in settings
    if settings.WebRemotePort != 0 {
        return settings.WebRemotePort  // Use configured port
    }

    // 2. Auto-detect from reaper.ini
    if config, err := scripts.GetWebRemoteConfig(); err == nil {
        return config.Port  // Use detected port
    }

    // 3. Fallback to default
    return 2307
}
```

### Resolution Priority

1. **Configured Port** (in settings.json)
   - User explicitly set the port
   - Takes highest priority

2. **Detected Port** (from reaper.ini)
   - Automatically detected from REAPER
   - Used if not configured in settings

3. **Default Port** (2307)
   - Fallback if detection fails
   - Safe default value

## Benefits

### 1. Zero Configuration for Existing Users
✅ **No setup needed** - Plugin just works
✅ **No port confusion** - Uses actual REAPER port
✅ **Instant readiness** - Works immediately after install

### 2. Flexible for New Users
✅ **Guided setup** - Port field appears when needed
✅ **Clear defaults** - Port 2307 suggested
✅ **Easy configuration** - Simple one-field setup

### 3. Always Works
✅ **Smart fallback** - Multiple detection methods
✅ **No breaking changes** - Handles all scenarios
✅ **Robust** - Works even if detection fails

## Technical Implementation

### Files Modified

**1. main.go:198-232** - `GetRequiredConfig()`
```go
// Conditional logic
if _, err := scripts.GetWebRemoteConfig(); err == nil {
    // Found - don't require port config
} else {
    // Not found - add port to required config
    configVars = append(configVars, ...)
}
```

**2. main.go:235-260** - `ValidateConfig()`
```go
// Port validation is now optional
if portValue, ok := config["web_remote_port"]; ok {
    // Validate only if present
}
```

**3. settings.go:61-92** - `GetWebRemotePort()`
```go
// Multi-level fallback
if settings.WebRemotePort != 0 {
    return settings.WebRemotePort
}
return sm.getAutoDetectedPort()  // Auto-detect
```

### Configuration States

| REAPER Config | Plugin Config | Result |
|---------------|---------------|--------|
| Has Web Remote | Not Set | Uses REAPER port |
| Has Web Remote | Set | Uses plugin port |
| No Web Remote | Not Set | Uses 2307 |
| No Web Remote | Set | Uses plugin port |

## Testing

### Test Case 1: Existing Web Remote (Most Common)

**Setup:**
```bash
# Ensure REAPER has Web Remote
grep "HTTP" ~/Library/Application\ Support/REAPER/reaper.ini
# Output: csurf_0=HTTP 0 2307 '' 'index.html' 0 ''
```

**Test:**
1. Install plugin
2. Open configuration UI
3. Verify: NO port field shown
4. Save configuration
5. Use plugin: Should connect to port 2307

**Expected:**
✅ Port field not in UI
✅ Plugin uses port 2307
✅ Operations work immediately

### Test Case 2: No Web Remote

**Setup:**
```bash
# Remove Web Remote from REAPER
sed -i '' '/HTTP.*csurf/d' ~/Library/Application\ Support/REAPER/reaper.ini
```

**Test:**
1. Install plugin
2. Open configuration UI
3. Verify: Port field IS shown
4. Accept default 2307 or change
5. Save configuration
6. Configure REAPER to match

**Expected:**
✅ Port field shown in UI
✅ Default is 2307
✅ Port saved in settings

### Test Case 3: Mixed Configuration

**Setup:**
```bash
# REAPER has port 8080
echo "csurf_0=HTTP 1 8080 '' 'index.html' 0 ''" >> reaper.ini
```

**Manually set plugin config to 9000:**
```json
{
  "scripts_dir": "...",
  "web_remote_port": 9000
}
```

**Test:**
Use plugin operations

**Expected:**
✅ Plugin uses configured port 9000 (not detected 8080)
✅ Configuration takes precedence

## Migration from Previous Version

### For Users Upgrading

**If you had manually configured the port before:**
- Your configuration is preserved
- Plugin continues using your configured port
- No changes needed

**If you never configured the port (shouldn't happen with old versions):**
- Plugin will auto-detect from REAPER
- Or use default 2307
- Seamless upgrade

### Configuration File

**Old format (still works):**
```json
{
  "scripts_dir": "/path/to/scripts",
  "web_remote_port": 8080
}
```

**New format (optional port):**
```json
{
  "scripts_dir": "/path/to/scripts"
}
```

**Both formats are valid!**

## Troubleshooting

### Issue: Plugin can't connect to REAPER

**Diagnosis:**
```bash
# Check what port plugin is using
curl http://localhost:8080/api/plugins/ori-reaper/config

# Check REAPER config
grep "csurf.*HTTP" ~/Library/Application\ Support/REAPER/reaper.ini
```

**Solutions:**

**If plugin doesn't have port configured:**
```json
# Add to settings file
{
  "scripts_dir": "...",
  "web_remote_port": 2307
}
```

**If REAPER doesn't have Web Remote:**
1. Open REAPER → Preferences
2. Control/OSC/web → Add
3. Web browser interface
4. Set port to match plugin (2307)

**If ports mismatch:**
- Either change REAPER to match plugin
- Or reconfigure plugin to match REAPER

### Issue: Port field appears when it shouldn't

**Cause:** Plugin can't read reaper.ini

**Check:**
```bash
ls -la ~/Library/Application\ Support/REAPER/reaper.ini
```

**Solution:** Ensure file exists and is readable

### Issue: Port field doesn't appear when it should

**Cause:** reaper.ini has Web Remote entry

**Check:**
```bash
grep "HTTP" ~/Library/Application\ Support/REAPER/reaper.ini
```

**Solution:** If you want to configure manually, delete the Web Remote entry in REAPER first

## Why Conditional Configuration?

### Design Philosophy

**Problem:** Traditional approach requires BOTH:
1. Configure port in plugin
2. Configure port in REAPER
3. Ensure they match

**Solution:** Conditional configuration:
1. Plugin auto-detects REAPER config
2. OR asks user to configure
3. Always works correctly

### User-Centered Design

**For experienced users:**
- Already have REAPER configured
- Don't want redundant configuration
- Expect plugin to "just work"

**For new users:**
- Don't have REAPER configured yet
- Need guidance on what port to use
- Want clear setup process

**Conditional config satisfies both!**

## Comparison

### Old Approach: Always Required

**Configuration:**
```json
{
  "scripts_dir": "...",
  "web_remote_port": 8080  // ALWAYS required
}
```

**Issues:**
- ❌ Redundant if REAPER already configured
- ❌ Risk of mismatch between plugin and REAPER
- ❌ Extra step for users

### New Approach: Conditional

**Configuration (with REAPER setup):**
```json
{
  "scripts_dir": "..."
  // Port auto-detected - not needed!
}
```

**Configuration (without REAPER setup):**
```json
{
  "scripts_dir": "...",
  "web_remote_port": 2307  // Required only if not detected
}
```

**Benefits:**
- ✅ No redundancy
- ✅ No mismatch possible
- ✅ Minimal configuration

## Related Documentation

- `PORT_AUTO_DETECTION.md` - Auto-detection details
- `WEB_REMOTE_PORT_CONFIGURATION.md` - Configuration overview
- `COMPLETE_FEATURE_SUMMARY.md` - Feature summary

## Changelog

**Version 0.0.9** (2025-10-28)
- ✅ Made web_remote_port conditionally required
- ✅ Auto-detects existing REAPER Web Remote configuration
- ✅ Skips port config if Web Remote already exists
- ✅ Requires port config only if no Web Remote found
- ✅ Smart fallback: configured → detected → default (2307)
- ✅ Zero-config experience for users with existing REAPER setup

---

**Status:** ✅ Complete and Production Ready
**Version:** 0.0.9
**Last Updated:** 2025-10-28
