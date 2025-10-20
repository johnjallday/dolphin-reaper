function auto_name_track(track)
	local fxCount = reaper.TrackFX_GetCount(track)

	local new_name = "No FX"
	if fxCount > 0 then
		-- Get the name of the first FX plugin
		local retval, fx_name = reaper.TrackFX_GetFXName(track, 0, "")
		if retval then
			new_name = fx_name
		end
	end

	-- Set the track's name
	reaper.GetSetMediaTrackInfo_String(track, "P_NAME", new_name, true)
end

-- Main function to iterate over all tracks
function main()
	local trackCount = reaper.CountTracks(0)
	for i = 0, trackCount - 1 do
		local track = reaper.GetTrack(0, i)
		auto_name_track(track)
	end
	reaper.UpdateArrange()
end

main()
