package ops

import (
	"os"
	"sort"
	"strings"
	"time"
)

// Next is the one thing the home screen tells a person to do next (spec
// 14.10.4).
type Next struct {
	Text string `json:"text"`
	Page string `json:"page"` // the screen to go to
	// Act names the control on that screen to point at ("" = none).
	Act string `json:"act"`
}

// NextStep tells what to do next, from what the case folder holds: the
// first step of the case left to do (spec 14.10.4, 14.12.1). It counts the
// unhandled lines, which reads the configurations again; the screen keeps
// that count while nothing it depends on changes (Progress). It writes
// nothing.
func (e *Env) NextStep() (*Next, error) {
	p, err := e.Progress(e.CountOpen())
	if err != nil {
		return nil, err
	}
	return p.Next, nil
}

// lastSnapshotTime is when the newest record of a generation was written
// (the time of the file: the name tells the time of the generation, the file
// is written after the book was read).
func (e *Env) lastSnapshotTime() time.Time {
	entries, err := os.ReadDir(e.Proj.Path(snapshotDir))
	if err != nil {
		return time.Time{}
	}
	var last time.Time
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		if info, err := ent.Info(); err == nil && info.ModTime().After(last) {
			last = info.ModTime()
		}
	}
	return last
}

// LastOutput finds the newest output of "generate" or "check": the folder
// under out/ that holds reports/check_results.xlsx ("" = none).
func (e *Env) LastOutput() string {
	entries, err := os.ReadDir(e.Proj.Path("out"))
	if err != nil {
		return ""
	}
	var dirs []string
	for _, ent := range entries {
		if ent.IsDir() {
			dirs = append(dirs, ent.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	for _, d := range dirs {
		if _, err := os.Stat(e.Proj.Path("out/" + d + "/reports/check_results.xlsx")); err == nil {
			return "out/" + d
		}
	}
	return ""
}

// Attach points the devices of an outcome that wrote nothing (Dry) at the
// files of an output folder of before (spec 14.10.6): the configurations
// that are there. It tells whether the folder has configurations, and the
// mark UNVERIFIED.txt ("" = none).
func (o *GenerateOutcome) Attach(e *Env, dir string) (configs bool, unverified string) {
	o.Dir = dir
	if info, err := os.Stat(e.Proj.Path(dir + "/configs")); err == nil && info.IsDir() {
		configs = true
	}
	for i := range o.Devices {
		d := &o.Devices[i]
		d.File = ""
		// Only where the check of now also gives a text: a device that is
		// blocked now does not point at a file of before.
		var try []string
		switch {
		case (d.State == GenWritten || d.State == GenChecked) && d.name != "":
			try = append(try, dir+"/configs/"+d.name)
		case d.State == GenFailed:
			try = append(try, dir+"/configs/_failed/"+safeName(d.Host)+".txt")
		}
		for _, rel := range try {
			if _, err := os.Stat(e.Proj.Path(rel)); err == nil {
				d.File = rel
				break
			}
		}
	}
	if _, err := os.Stat(e.Proj.Path(dir + "/UNVERIFIED.txt")); err == nil {
		unverified = dir + "/UNVERIFIED.txt"
	}
	o.Report = Files{XLSX: dir + "/reports/check_results.xlsx", HTML: dir + "/reports/check_results.html"}
	if _, err := os.Stat(e.Proj.Path(dir + "/reports/rewrite.xlsx")); err == nil {
		o.RewriteReport = Files{XLSX: dir + "/reports/rewrite.xlsx", HTML: dir + "/reports/rewrite.html"}
	}
	return configs, unverified
}

// Stamped gives the time of an output folder out/<time> in words.
func Stamped(dir string) string {
	stamp := strings.TrimPrefix(dir, "out/")
	if len(stamp) > 17 {
		stamp = stamp[:17]
	}
	if t, err := time.Parse("2006-01-02_150405", stamp); err == nil {
		return t.Format("2006-01-02 15:04:05")
	}
	return stamp
}
