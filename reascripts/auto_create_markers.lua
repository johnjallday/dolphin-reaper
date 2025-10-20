--[[
ReaScript Name: Marker Creation Tools
Description: Contains functions to add markers to a project.
             - create_marker_at_regions: Adds a marker at the start of each region.
             - create_marker_at_start_and_end: Adds 'start' and 'end' markers based on project content, if they don't already exist.
Author: Gemini
Version: 2.1
--]]

-- Helper function to find an existing marker by its exact name.
function findMarkerByName(name_to_find)
	local num_items = reaper.CountProjectMarkers(0)
	for i = 0, num_items - 1 do
		-- EnumProjectMarkers3 gets info including the name of the marker/region.
		local retval, _, _, _, name = reaper.EnumProjectMarkers3(0, i)
		if retval > 0 and name == name_to_find then
			return true -- Return true as soon as we find a match
		end
	end
	return false -- Return false if the loop completes without finding the marker
end

-- Function to create a marker at the beginning of each region
function create_marker_at_regions()
	reaper.Undo_BeginBlock()

	local markers_added_count = 0
	local regions_to_mark = {}

	-- Step 1: Find all regions and store their info
	local num_items = reaper.CountProjectMarkers(0)
	for i = 0, num_items - 1 do
		local retval, isrgn, pos, _, name = reaper.EnumProjectMarkers3(0, i)
		if retval > 0 and isrgn then
			table.insert(regions_to_mark, { position = pos, name = name })
		end
	end

	-- Step 2: Add markers based on the collected region info
	if #regions_to_mark > 0 then
		for _, region_info in ipairs(regions_to_mark) do
			local new_marker_name = "Start: " .. region_info.name
			reaper.AddProjectMarker2(0, false, region_info.position, 0, new_marker_name, -1, 0)
			markers_added_count = markers_added_count + 1
		end
	end

	if markers_added_count > 0 then
		reaper.ShowConsoleMsg("Added " .. markers_added_count .. " markers at the start of regions.\n")
	else
		reaper.ShowConsoleMsg("No regions found to add markers to.\n")
	end

	reaper.Undo_EndBlock("Add Marker at Start of Each Region", -1)
end

-- Function to create markers at the start and end of the project content (items and regions)
function create_marker_at_start_and_end()
	reaper.Undo_BeginBlock()

	local earliest_start, latest_end = nil, nil

	-- Loop 1: Find boundaries from Media Items
	for i = 0, reaper.CountTracks(0) - 1 do
		local track = reaper.GetTrack(0, i)
		for j = 0, reaper.CountTrackMediaItems(track) - 1 do
			local item = reaper.GetTrackMediaItem(track, j)
			local item_pos = reaper.GetMediaItemInfo_Value(item, "D_POSITION")
			local item_end = item_pos + reaper.GetMediaItemInfo_Value(item, "D_LENGTH")
			if earliest_start == nil or item_pos < earliest_start then
				earliest_start = item_pos
			end
			if latest_end == nil or item_end > latest_end then
				latest_end = item_end
			end
		end
	end

	-- Loop 2: Update boundaries from Regions
	for i = 0, reaper.CountProjectMarkers(0) - 1 do
		local retval, isrgn, pos, rgn_end = reaper.EnumProjectMarkers3(0, i)
		if retval > 0 and isrgn then
			if earliest_start == nil or pos < earliest_start then
				earliest_start = pos
			end
			if latest_end == nil or rgn_end > latest_end then
				latest_end = rgn_end
			end
		end
	end

	if earliest_start == nil then
		reaper.ShowConsoleMsg("No items or regions found. No markers added.\n")
		reaper.Undo_EndBlock("Add Start/End Markers (no content)", 0)
		return
	end

	local marker_start_exists = findMarkerByName("start")
	local marker_end_exists = findMarkerByName("end")
	local markers_were_added = false

	-- Only add "start" marker if it doesn't already exist
	if not marker_start_exists then
		reaper.AddProjectMarker2(0, false, earliest_start, 0, "start", -1, 0)
		reaper.ShowConsoleMsg("Added 'start' marker.\n")
		markers_were_added = true
	else
		reaper.ShowConsoleMsg("'start' marker already exists. Skipped.\n")
	end

	-- Only add "end" marker if it doesn't already exist
	if not marker_end_exists then
		reaper.AddProjectMarker2(0, false, latest_end, 0, "end", -1, 0)
		reaper.ShowConsoleMsg("Added 'end' marker.\n")
		markers_were_added = true
	else
		reaper.ShowConsoleMsg("'end' marker already exists. Skipped.\n")
	end

	if not markers_were_added and marker_start_exists and marker_end_exists then
		reaper.Undo_EndBlock("Add Start/End Markers (none added)", 0)
	else
		reaper.Undo_EndBlock("Add Start/End Markers", -1)
	end
end

-- Defer the execution to ensure Reaper's UI is ready.
-- To use, uncomment the function you wish to run.
reaper.defer(function()
	-- To add markers at the start of each region:
	-- create_marker_at_regions()

	-- To add 'start' and 'end' markers for the whole project:
	create_marker_at_start_and_end()

	reaper.UpdateArrange() -- Refresh the ruler to show the new markers.
end)
