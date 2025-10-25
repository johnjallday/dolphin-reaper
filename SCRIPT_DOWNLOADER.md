# Script Downloader Feature

The ori-reaper plugin now includes a **Script Downloader** feature that allows you to browse and download ReaScripts directly from the GitHub repository.

## New Operations

### `list_available_scripts`

Fetches and displays a table of all available scripts from the GitHub repository.

**Usage:**
```
"Show me available scripts to download"
"List downloadable scripts"
"What scripts are available in the repository?"
```

**Returns:**
- A structured table with:
  - **Name**: Display name of the script
  - **Filename**: Actual filename (including extension)
  - **Description**: Auto-generated description based on the script name
  - **Size**: File size in human-readable format

**Example Output:**
```
Available ReaScripts for Download

Found 5 scripts in the repository. Use 'download_script' to install.

┌─────────────────────────┬──────────────────────────┬─────────────────────────────┬────────┐
│ Name                    │ Filename                 │ Description                 │ Size   │
├─────────────────────────┼──────────────────────────┼─────────────────────────────┼────────┤
│ Audio Normalizer        │ audio_normalizer.lua     │ Audio normalization script  │ 2.3 KB │
│ Midi Velocity Random    │ midi_velocity_random.lua │ MIDI processing script      │ 1.8 KB │
│ Tempo Calculator        │ tempo_calculator.lua     │ Tempo manipulation script   │ 1.2 KB │
└─────────────────────────┴──────────────────────────┴─────────────────────────────┴────────┘
```

---

### `download_script`

Downloads and installs a specific script from the repository.

**Usage:**
```json
{
  "operation": "download_script",
  "filename": "audio_normalizer.lua"
}
```

**Via Chat:**
```
"Download the audio_normalizer.lua script"
"Install midi_velocity_random.lua"
```

**Returns:**
- Success message: `"Successfully added REAPER script: audio_normalizer.lua"`
- Error if script already exists or download fails

---

## Implementation Details

### GitHub API Integration

- **Repository**: `johnjallday/ori-reaper`
- **Branch**: `dev`
- **Directory**: `/reascripts`
- **API Endpoint**: `https://api.github.com/repos/johnjallday/ori-reaper/contents/reascripts?ref=dev`

### Supported Script Types

- `.lua` - Lua scripts
- `.eel` - EEL scripts
- `.py` - Python scripts

### Auto-Description

The system automatically generates descriptions based on script names:

| Keyword      | Description              |
|--------------|--------------------------|
| normalize    | Audio normalization      |
| midi         | MIDI processing          |
| tempo        | Tempo manipulation       |
| marker       | Marker management        |
| render       | Render/export            |
| track        | Track management         |
| fx/effect    | FX/effects               |
| item         | Item manipulation        |
| region       | Region management        |

### File Size Formatting

Automatically formats file sizes in human-readable format:
- Bytes (B)
- Kilobytes (KB)
- Megabytes (MB)

---

## Code Structure

### New Files

**`pkg/scripts/downloader.go`**
- `ScriptDownloader` struct
- `ListAvailableScripts()` - Fetches and displays script list
- `DownloadScript()` - Downloads and installs a script
- GitHub API integration
- File size formatting
- Auto-description generation

### Updated Files

**`main.go`**
- Added `list_available_scripts` to operation enum
- Added `download_script` to operation enum
- Added `filename` parameter for downloads
- Added operation handlers in `Call()` function

---

## Usage Examples

### 1. Browse Available Scripts

**User:** "What scripts can I download?"

**Plugin Response:**
```
Available ReaScripts for Download

Found 5 scripts in the repository. Use 'download_script' to install.

[Table of scripts with Name, Filename, Description, Size]
```

### 2. Download a Specific Script

**User:** "Download the audio_normalizer.lua script"

**Plugin Response:**
```
Successfully added REAPER script: audio_normalizer.lua
```

### 3. List Installed Scripts

**User:** "List my scripts"

**Plugin Response:**
```
🎵 Available REAPER Scripts

[Table of installed scripts]
```

---

## Technical Details

### Structured Result Format

The `list_available_scripts` operation returns a structured table result using the `pluginapi.StructuredResult` format:

```go
result := pluginapi.NewTableResult(
    "Available ReaScripts for Download",
    []string{"Name", "Filename", "Description", "Size"},
    scripts,
)
result.Description = fmt.Sprintf("Found %d scripts in the repository. Use 'download_script' to install.", len(scripts))
result.Metadata["action"] = "download_script"
result.Metadata["source"] = "https://github.com/johnjallday/ori-reaper/tree/dev/reascripts"
```

This ensures the frontend displays the data as a nicely formatted table.

---

## Error Handling

### Common Errors

1. **Script not found**: Returns error if the requested script doesn't exist in the repository
2. **Download failed**: Returns HTTP error if GitHub API is unreachable
3. **Script already exists**: Returns error if trying to download a script that's already installed
4. **Invalid file type**: Returns error if trying to download a non-script file

### Error Messages

- `"failed to fetch scripts from GitHub: [error]"`
- `"script not found: [filename]"`
- `"script already exists: [filename]"`
- `"download failed with status: [code]"`
- `"filename is required for 'download_script' operation"`

---

## Future Enhancements

### Planned Features

1. **Script Preview**: View script content before downloading
2. **Auto-Update**: Check for updates to installed scripts
3. **Categories**: Filter scripts by category (Audio, MIDI, FX, etc.)
4. **Search**: Search scripts by name or description
5. **Batch Download**: Download multiple scripts at once
6. **Script Metadata**: Display author, version, dependencies
7. **Custom Repositories**: Support downloading from other repositories

### Potential Improvements

- Cache GitHub API responses
- Add script ratings/popularity
- Support downloading entire script packs
- Integration with ReaPack
- Script dependency management
- Automatic script updates on startup

---

## Testing

To test the new feature:

1. **Restart ori-agent** to load the updated plugin
2. Ask: **"Show me available scripts to download"**
3. Verify the table displays correctly with Name, Filename, Description, Size
4. Ask: **"Download the [filename] script"**
5. Verify the script is downloaded and saved to your scripts directory
6. Ask: **"List my scripts"** to confirm it was installed

---

## Version

- **Plugin Version**: 0.0.5
- **Feature Added**: October 21, 2025
- **GitHub API**: v3 (REST API)

---

## Support

For issues or feature requests related to the Script Downloader:
1. Check the GitHub repository: https://github.com/johnjallday/ori-reaper
2. Report issues in the repository's issue tracker
3. Contribute scripts to the `/reascripts` directory in the `dev` branch
