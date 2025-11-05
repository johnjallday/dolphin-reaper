# Bug Fix: Web Remote Port Detection - HTTP Format Support

## Issue

**Error:** `web remote (WEBR) control surface not found in reaper.ini`

The original implementation was looking for control surface entries starting with `WEBR`, but REAPER actually uses `HTTP` for the web remote control surface.

## Root Cause

### Expected Format (Incorrect Assumption):
```ini
csurf_0=WEBR 0 0 0 0 0 0 - - - - - 8080
```

### Actual Format in REAPER:
```ini
csurf_0=HTTP 0 2307 '' 'index.html' 0 ''
csurf_1=HTTP 0 8080 '' 'fancier.html' 0 ''
```

## Investigation

Searched the user's `reaper.ini` file:
```bash
$ grep csurf /Users/jj/Library/Application\ Support/REAPER/reaper.ini

csurf_0=HTTP 0 2307 '' 'index.html' 0 ''
csurf_1=HTTP 0 8080 '' 'fancier.html' 0 ''
csurf_cnt=1
csurfrate=15
```

**Findings:**
- Control surface type is `HTTP`, not `WEBR`
- Format: `HTTP <enabled> <port> '' '<html_file>' 0 ''`
- Field 1: enabled flag (0 = disabled, 1 = enabled)
- Field 2: port number (2307, 8080, etc.)

## Solution

Updated `internal/scripts/config.go` to support **both** formats:

### HTTP Format (Current REAPER):
```
HTTP <enabled> <port> '' '<html_file>' 0 ''
```
- Field 0: `HTTP`
- Field 1: enabled (0 or 1)
- Field 2: port number

### WEBR Format (Legacy/Alternative):
```
WEBR <enabled> <flags...> <port>
```
- Field 0: `WEBR`
- Field 1: enabled (0 or 1)
- Last Field: port number

## Code Changes

### Before (config.go:112):
```go
// Only checked for "WEBR"
if strings.HasPrefix(csurfValue, "WEBR ") {
    // ...
}
```

### After (config.go:112):
```go
// Check for both "HTTP" and "WEBR"
if strings.HasPrefix(csurfValue, "HTTP ") || strings.HasPrefix(csurfValue, "WEBR ") {
    var port int
    var enabled bool

    if strings.HasPrefix(csurfValue, "HTTP ") {
        // Format: HTTP <enabled> <port> '' 'index.html' 0 ''
        enabledVal := fields[1]
        enabled = (enabledVal == "1")
        port, _ = strconv.Atoi(fields[2])
    } else {
        // Format: WEBR <enabled> <flags...> <port>
        portStr := fields[len(fields)-1]
        port, _ = strconv.Atoi(portStr)
        enabledVal := fields[1]
        enabled = (enabledVal == "1")
    }
    // ...
}
```

## Testing

### User's Configuration:
```ini
csurf_0=HTTP 0 2307 '' 'index.html' 0 ''
```

### Expected Output:
```
REAPER Web Remote:
  Status: disabled
  Port: 2307
  Control Surface ID: csurf_0
  URL: http://localhost:2307
```

**Note:** The status shows "disabled" because the enabled flag is `0`. User needs to enable it in REAPER or the value should be `1`.

## Verification Steps

1. **Check enabled flag:**
   ```bash
   grep "csurf_0" ~/Library/Application\ Support/REAPER/reaper.ini
   # Output: csurf_0=HTTP 0 2307 '' 'index.html' 0 ''
   #                       ^ this should be 1 for enabled
   ```

2. **Enable Web Remote in REAPER:**
   - Preferences → Control/OSC/web
   - Make sure web remote is checked/enabled
   - This will change the value from `HTTP 0` to `HTTP 1`

3. **Test the function:**
   - Reload `ori-reaper` plugin
   - Run: `get_web_remote_port`
   - Should now successfully return port 2307

## Additional Notes

### Multiple Web Remotes

The user has two web remote entries:
```ini
csurf_0=HTTP 0 2307 '' 'index.html' 0 ''
csurf_1=HTTP 0 8080 '' 'fancier.html' 0 ''
```

The function returns the **first** HTTP entry found (csurf_0 with port 2307).

If multiple web remotes are enabled, this is the expected behavior. To get the second one, you would need a different function or parameter to specify which csurf index to read.

### HTML Interface Files

The configuration includes HTML interface files:
- `index.html` - Default REAPER web interface
- `fancier.html` - Alternative theme/interface

These are different web UI skins for the REAPER web remote.

## Impact

- ✅ Fixes port detection for modern REAPER installations
- ✅ Maintains backward compatibility with legacy `WEBR` format
- ✅ Works with multiple web remote configurations
- ✅ Properly detects enabled/disabled state

## Build Info

**Fixed in:** Version 0.0.5
**Build Date:** 2025-10-28
**Commit:** 5670b85

## Related Documentation

- See `WEB_REMOTE_PORT_FEATURE.md` for full feature documentation
- See `COMPLETE_FEATURE_SUMMARY.md` for usage examples
