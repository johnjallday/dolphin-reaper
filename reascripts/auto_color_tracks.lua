function AutoColorTracks(target_color, string_group)
	-- Get the number of tracks in the project
	local num_tracks = reaper.CountTracks(0)

	-- Lowercase the search string once to optimize
	local search_string = string.lower(string_group)

	-- Iterate through all tracks
	for i = 0, num_tracks - 1 do
		-- Get the current track
		local track = reaper.GetTrack(0, i)

		-- Get the track name
		local _, track_name = reaper.GetSetMediaTrackInfo_String(track, "P_NAME", "", false)

		-- Convert track name to lowercase for case-insensitive comparison
		local track_name_lower = string.lower(track_name)

		-- Check if the track name STARTS with the search string
		if track_name_lower:sub(1, #search_string) == search_string then
			-- Change the track color
			reaper.SetMediaTrackInfo_Value(track, "I_CUSTOMCOLOR", target_color | 0x1000000)
		end
	end
end

-- oneliner color/group calls
AutoColorTracks(reaper.ColorToNative(180, 20, 20), "DR ")
AutoColorTracks(reaper.ColorToNative(0, 115, 245), "BA ")
AutoColorTracks(reaper.ColorToNative(232, 128, 255), "M ")
AutoColorTracks(reaper.ColorToNative(55, 204, 5), "FX ")
AutoColorTracks(reaper.ColorToNative(255, 234, 0), "V ")
AutoColorTracks(reaper.ColorToNative(0, 250, 140), "V BG ") -- lighter yellow for background vocals
AutoColorTracks(reaper.ColorToNative(80, 40, 55), "INST")     -- NEW: Instrument tracks
AutoColorTracks(reaper.ColorToNative(180, 45, 25), "Group Drums")
AutoColorTracks(reaper.ColorToNative(0, 50, 205), "Group Bass")
AutoColorTracks(reaper.ColorToNative(200, 70, 255), "Group Melody")
AutoColorTracks(reaper.ColorToNative(80, 40, 55), "Group Inst")
AutoColorTracks(reaper.ColorToNative(230, 234, 143), "Group Vox")
AutoColorTracks(reaper.ColorToNative(200, 60, 150), "BUS MASTER")
AutoColorTracks(reaper.ColorToNative(200, 60, 150), "BUS INST")
AutoColorTracks(reaper.ColorToNative(200, 60, 150), "BUS VOX")
AutoColorTracks(reaper.ColorToNative(200, 60, 150), "BUS SEND")
AutoColorTracks(reaper.ColorToNative(200, 60, 150), "BUS SENDS") -- NEW
AutoColorTracks(reaper.ColorToNative(200, 60, 150), "SEND ")     -- NEW

-- also match ALL-UPPER-CASE track prefixes
AutoColorTracks(reaper.ColorToNative(180, 45, 25), "GROUP DRUMS")
AutoColorTracks(reaper.ColorToNative(0, 50, 205), "GROUP BASS")
AutoColorTracks(reaper.ColorToNative(200, 70, 255), "GROUP MELODY")
AutoColorTracks(reaper.ColorToNative(80, 40, 55), "GROUP INST")
AutoColorTracks(reaper.ColorToNative(230, 234, 143), "GROUP VOX")
AutoColorTracks(reaper.ColorToNative(200, 60, 150), "BUS MASTER")

