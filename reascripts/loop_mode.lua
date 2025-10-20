--reaper.ShowConsoleMsg("Loop Creation Mode")

--get Region
--if there's only one region
--
--
--
--toggle repeat on

function add4BarRegion()
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
--select Region
function SetLoopPoint()
	_, n_markers, n_regions = reaper.CountProjectMarkers(0)

	if n_regions == 1 then
		reaper.GoToRegion(0, 1, true)
		reaper.Main_OnCommand(43102, 0) --Set loop points to current region
	end
end

reaper.GetSetRepeat(1)
add4BarRegion()
SetLoopPoint()
