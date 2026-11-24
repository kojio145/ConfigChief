package main

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/ui"
)

// The calls of the page for the third look at what saves a person work
// (spec 14.12, 13.6, 13.7).

// Steps tells the steps of the case and the next thing to do (spec
// 14.12.1). quick: without counting the unhandled lines anew.
func (a *App) Steps(quick bool) (r0 *ops.Progress, err error) {
	defer a.guard("Steps", &err)
	return a.s.Steps(quick)
}

// DeliveryPlan tells whether the whole set of the deliverables can be made.
func (a *App) DeliveryPlan() (r0 *ui.DeliveryPlanView, err error) {
	defer a.guard("DeliveryPlan", &err)
	return a.s.DeliveryPlan()
}

// MakeDelivery makes the whole set of the deliverables (spec 13.6, 13.7).
func (a *App) MakeDelivery(mask, allowUnverified bool, version, summary, by string) (r0 *ui.DeliveryView, err error) {
	defer a.guard("MakeDelivery", &err)
	return a.s.MakeDelivery(mask, allowUnverified, version, summary, by)
}

// Outputs lists the folders of out/ (spec 14.12.6).
func (a *App) Outputs() (r0 []ops.OutputDir, err error) {
	defer a.guard("Outputs", &err)
	return a.s.Outputs()
}

// DropOutputs takes folders of out/ away.
func (a *App) DropOutputs(dirs []string) (r0 []ops.OutputDir, err error) {
	defer a.guard("DropOutputs", &err)
	return a.s.DropOutputs(dirs)
}

// CaseParts lists what can be taken from a case before (spec 14.12.8).
func (a *App) CaseParts() []ui.CasePart { defer a.guard("CaseParts", nil); return a.s.CaseParts() }

// Made is a case that was just made, and what the page tells about it.
type Made struct {
	Home *ui.Home `json:"home"`
	Said string   `json:"said"`
}

// CreateFrom makes a case that takes the settings of one before.
func (a *App) CreateFrom(name, kind, from string, parts []string) (r0 *Made, err error) {
	defer a.guard("CreateFrom", &err)
	h, said, err := a.s.NewCaseFrom(name, kind, from, parts)
	if err != nil {
		return nil, err
	}
	return &Made{h, said}, nil
}

// SampleCase makes the sample case (spec 14.12.7).
func (a *App) SampleCase() (r0 *Made, err error) {
	defer a.guard("SampleCase", &err)
	h, said, err := a.s.NewSample()
	if err != nil {
		return nil, err
	}
	return &Made{h, said}, nil
}

// DraftCheck checks the book with the draft of the table put in (spec
// 14.12.3).
func (a *App) DraftCheck() (r0 *ui.ResultView, err error) {
	defer a.guard("DraftCheck", &err)
	return a.s.DraftCheck()
}

// BookDevices lists the host names of the sheet 機器 (spec 14.12.5).
func (a *App) BookDevices() (r0 []string, err error) {
	defer a.guard("BookDevices", &err)
	return a.s.BookDevices()
}

// SetTitle puts the name of the open case into the title of the window
// (spec 14.12.5; "" = none).
func (a *App) SetTitle(name string) {
	defer a.guard("SetTitle", nil)
	if a.ctx == nil {
		return
	}
	title := "ConfigChief"
	if name != "" {
		title += " - " + name
	}
	runtime.WindowSetTitle(a.ctx, title)
}
