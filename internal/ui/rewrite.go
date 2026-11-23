package ui

import (
	"errors"
	"fmt"

	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/report"
)

// RewriteKind is a kind of the rewriting report with its number of lines
// (spec 10.4).
type RewriteKind struct {
	ID    string `json:"id"` // ops.RwAuto and the like
	Word  string `json:"word"`
	Count int    `json:"count"`
}

// RewriteDevice is an old device of the rewriting report.
type RewriteDevice struct {
	Device    string         `json:"device"` // the old host name ("" for a file that was not read)
	File      string         `json:"file"`
	NewDevice string         `json:"newDevice"`
	State     string         `json:"state"` // what the generation did for the new device, in words
	Total     int            `json:"total"`
	Counts    map[string]int `json:"counts"` // kind ID → lines
	Open      int            `json:"open"`   // the lines a person still has to look at
	Mark      string         `json:"mark"`
}

// RewriteView is the tab 書き換え of the result (spec 13.4): the kinds, the
// devices and the lines, narrowed to a device and a kind.
type RewriteView struct {
	// Has is false when the case is not one of renewal (no file in
	// input/old, or no new host name in 新旧対照表（機器）): the tab is not shown.
	Has    bool   `json:"has"`
	Report Files  `json:"report"` // the files in the output folder (empty = none written)
	Device string `json:"device"` // the filter: the old host name ("" = every device)
	Kind   string `json:"kind"`   // the filter: a kind ID ("" = every kind)
	Total  int    `json:"total"`  // the lines of every old configuration
	// Matched counts the lines the filter takes; Table holds at most
	// RewriteScreenMost of them (Cut tells that it holds fewer).
	Matched  int             `json:"matched"`
	Cut      bool            `json:"cut"`
	Kinds    []RewriteKind   `json:"kinds"`
	Counts   []Pair          `json:"counts"` // the numbers of the summary of the report
	Devices  []RewriteDevice `json:"devices"`
	Notes    []string        `json:"notes"`
	Table    Table           `json:"table"`   // the lines (the columns of the sheet 行の一覧)
	NewOnly  Table           `json:"newOnly"` // the new lines no old line led to (narrowed to the device's new host)
	Unjudged int             `json:"unjudged"`
	// Unapproved counts the lines with a finding that is not approved.
	Unapproved int `json:"unapproved"`
}

// RewriteScreenMost is the most lines the screen gets at once: a case of
// hundreds of devices has a hundred thousand lines.
const RewriteScreenMost = 5000

// Rewrite gives the rewriting report of the result that is shown, narrowed
// to an old device ("" = all) and a kind ("" = all; ops.RwAuto and the
// like). Nothing is written. The secrets are masked whatever the case says.
func (s *Session) Rewrite(device, kind string) (*RewriteView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil || s.shown == nil {
		return nil, errors.New("先に「生成とチェック」か「チェックだけ」を行ってください")
	}
	if kind != "" && ops.RewriteWord(kind) == "" {
		return nil, fmt.Errorf("区分「%s」はありません", kind)
	}
	v := &RewriteView{Device: device, Kind: kind, Report: s.wrote.rewrite,
		Kinds: []RewriteKind{}, Counts: []Pair{}, Devices: []RewriteDevice{}, Notes: []string{},
		Table:   table(report.Sheet{Name: "行の一覧", Header: ops.RewriteHeader, Code: []int{3, 9}}),
		NewOnly: table(report.Sheet{Name: "新にだけある行", Header: []string{"新機器", "行番号", "文面", "由来", "値"}, Code: []int{2}})}
	rw := s.env.Rewrite(s.shown)
	if rw == nil {
		return v, nil
	}
	v.Has, v.Total, v.Unjudged, v.Unapproved = true, rw.Total, rw.Unjudged, rw.Unapproved
	v.Notes = strs(rw.Notes)
	for _, k := range ops.RewriteKinds {
		v.Kinds = append(v.Kinds, RewriteKind{ID: k, Word: ops.RewriteWord(k), Count: rw.Counts[k]})
	}
	v.Counts = append(v.Counts, Pair{"旧コンフィグの全行", fmt.Sprint(rw.Total)})
	for _, k := range v.Kinds {
		v.Counts = append(v.Counts, Pair{k.Word, fmt.Sprint(k.Count)})
	}
	v.Counts = append(v.Counts, Pair{"うち未判定・保留の読めなかった行", fmt.Sprint(rw.Unjudged)},
		Pair{"うち未承認の指摘がある行", fmt.Sprint(rw.Unapproved)})
	known, newHost := device == "", ""
	for _, d := range rw.Devices {
		rd := RewriteDevice{Device: d.Device, File: d.File, NewDevice: d.NewDevice, State: d.State, Total: d.Total, Counts: map[string]int{}, Open: d.Open}
		for _, k := range ops.RewriteKinds {
			rd.Counts[k] = d.Counts[k]
		}
		if d.Open > 0 {
			rd.Mark = report.MarkWarn
		}
		v.Devices = append(v.Devices, rd)
		if device != "" && d.Device == device {
			known, newHost = true, d.NewDevice
		}
	}
	if !known {
		return nil, fmt.Errorf("旧機器「%s」は、書き換えレポートにありません", device)
	}
	for _, l := range rw.Lines {
		if (device != "" && l.Device != device) || (kind != "" && l.Kind != kind) {
			continue
		}
		v.Matched++
		if len(v.Table.Rows) == RewriteScreenMost {
			v.Cut = true
			continue
		}
		v.Table.Rows = append(v.Table.Rows, l.Cells())
		v.Table.Marks = append(v.Table.Marks, l.Mark())
	}
	for _, l := range rw.NewOnly {
		if device != "" && l.Device != newHost {
			continue
		}
		v.NewOnly.Rows = append(v.NewOnly.Rows, []string{l.Device, fmt.Sprint(l.Line), l.Text, l.From, l.Values})
		v.NewOnly.Marks = append(v.NewOnly.Marks, "")
	}
	return v, nil
}
