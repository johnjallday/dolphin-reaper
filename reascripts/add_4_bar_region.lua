-- description
-- add 4 bar region and set a loop point

function main()
	local proj = 0
	-- Get current edit cursor position in seconds
	--local startTime = reaper.GetCursorPosition()
	--reaper.TimeMap_QNToTime_abs(0, number qn)
	local startTime = reaper.TimeMap2_beatsToTime(0, 8)
	local bar = reaper.TimeMap2_beatsToTime(0, 4)
	local endTime = startTime + 4 * bar
	--snap = item_pos + item_snap
	reaper.AddProjectMarker2(0, true, startTime, endTime, "4bar", -1, 0)
	--reaper.AddProjectMarker(proj, true, qnStart, qnEnd, "test", 1)
	--forlocal endTime = reaper.TimeMap2_QNToTime(proj, qnEnd)
	reaper.Undo_BeginBlock()
end

reaper.defer(main)
