package main

// The pure arithmetic of the window's place (spec 14.8.5; winplace_windows.go
// does the rest). From PalaTerm (mainplace.go, term_bcast.go, v1.10.8).

// The window's default size (logical pixels) and the floor it may shrink to.
const (
	winWidth     = 1280
	winHeight    = 820
	winMinWidth  = 960
	winMinHeight = 600
)

// clampInto keeps r inside work: shrunk to it when larger, then moved in.
func clampInto(r, work rect) rect {
	w, h := r.R-r.L, r.B-r.T
	if w > work.R-work.L {
		w = work.R - work.L
	}
	if h > work.B-work.T {
		h = work.B - work.T
	}
	x, y := r.L, r.T
	if x+w > work.R {
		x = work.R - w
	}
	if y+h > work.B {
		y = work.B - h
	}
	if x < work.L {
		x = work.L
	}
	if y < work.T {
		y = work.T
	}
	return rect{x, y, x + w, y + h}
}

// firstPlace is the window of a first start: w×h cut to the work area,
// centred in it.
func firstPlace(work rect, w, h int32) rect {
	if w > work.R-work.L {
		w = work.R - work.L
	}
	if h > work.B-work.T {
		h = work.B - work.T
	}
	x := work.L + (work.R-work.L-w)/2
	y := work.T + (work.B-work.T-h)/2
	return rect{x, y, x + w, y + h}
}

// workspaceToScreen converts a workspace rectangle of the monitor mi to
// screen coordinates.
func workspaceToScreen(r rect, mi monitorInfo) rect {
	dx, dy := mi.Work.L-mi.Monitor.L, mi.Work.T-mi.Monitor.T
	return rect{r.L + dx, r.T + dy, r.R + dx, r.B + dy}
}

// expandBy is work grown by the window's invisible frame border (win is the
// window, ext its visible part): a window flush with a screen edge has its
// border outside the work area.
func expandBy(work, win, ext rect) rect {
	if ext.R <= ext.L || ext.B <= ext.T {
		return work
	}
	return rect{work.L - (ext.L - win.L), work.T - (ext.T - win.T), work.R + (win.R - ext.R), work.B + (win.B - ext.B)}
}

// frameFit is the window rectangle whose visible part is cell.
func frameFit(cell, win, ext rect) rect {
	if ext.R <= ext.L || ext.B <= ext.T {
		return cell
	}
	return rect{cell.L - (ext.L - win.L), cell.T - (ext.T - win.T), cell.R + (win.R - ext.R), cell.B + (win.B - ext.B)}
}
