package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/project"
)

// The things of spec 14.12 (the third look at what saves a person work).

// openCache keeps the count of the unhandled lines while nothing it depends
// on changes (spec 14.12.1).
type openCache struct {
	dir, mark string
	count     ops.OpenCount
}

// Steps tells the steps of the case and the next thing to do (spec
// 14.12.1). quick: do not count the unhandled lines now if the count kept
// is not of the case as it is (the page asks again with quick false).
func (s *Session) Steps(quick bool) (*ops.Progress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	mark := e.ProgressMark()
	count := ops.OpenCount{}
	if s.opens.dir == e.Proj.Dir && s.opens.mark == mark {
		count = s.opens.count
	} else if !quick {
		count = e.CountOpen()
		s.opens = openCache{dir: e.Proj.Dir, mark: mark, count: count}
	}
	return e.Progress(count)
}

// DeliveryPlanView is what the screen shows before the whole set is made.
type DeliveryPlanView struct {
	*ops.DeliveryPlan
	By string `json:"by"` // the name the last approval was given with
}

// DeliveryPlan tells whether the whole set of the deliverables can be made
// (spec 13.6).
func (s *Session) DeliveryPlan() (*DeliveryPlanView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	p, err := e.PlanDelivery()
	if err != nil {
		return nil, err
	}
	return &DeliveryPlanView{DeliveryPlan: p, By: s.name}, nil
}

// DeliveryView is what making the whole set did.
type DeliveryView struct {
	Summary string   `json:"summary"`
	Dir     string   `json:"dir"`
	Zip     string   `json:"zip"`
	Files   []string `json:"files"`
	Notes   []string `json:"notes"`
}

// MakeDelivery makes the whole set of the deliverables (spec 13.6, 13.7).
func (s *Session) MakeDelivery(mask, allowUnverified bool, version, summary, by string) (*DeliveryView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	o, err := e.MakeDelivery(ops.DeliveryOptions{Mask: mask, AllowUnverified: allowUnverified, Version: version, Summary: summary, By: by}, s.now())
	if err != nil {
		return nil, err
	}
	if by = strings.TrimSpace(by); by != "" && by != s.name {
		s.name = by
		s.save()
	}
	v := &DeliveryView{Dir: o.Dir, Zip: o.Zip, Files: o.Files, Notes: o.Notes}
	v.Summary = fmt.Sprintf("納品一式を %s に作りました（コンフィグ %d 台・ファイル %d 個）。同じ中身の zip: %s", o.Dir, o.Configs, len(o.Files), o.Zip)
	if o.Revision != nil {
		v.Summary += fmt.Sprintf("。改版履歴に版 %s を残しました", o.Revision.Version)
	}
	if o.Unverified {
		v.Notes = append(v.Notes, fmt.Sprintf("未承認の指摘が残ったまま作りました（エラー %d・警告 %d）。UNVERIFIED.txt を入れています", o.Errors, o.Warnings))
	}
	if v.Notes == nil {
		v.Notes = []string{}
	}
	return v, nil
}

// Outputs lists the folders of out/ (spec 14.12.6).
func (s *Session) Outputs() ([]ops.OutputDir, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	return e.Outputs(), nil
}

// DropOutputs takes folders of out/ away and lists the rest.
func (s *Session) DropOutputs(dirs []string) ([]ops.OutputDir, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	if _, err := e.DropOutputs(dirs, s.now()); err != nil {
		return nil, err
	}
	return e.Outputs(), nil
}

// CasePart is a part of a case that can be taken into a new one.
type CasePart struct {
	ID      string `json:"id"`
	Word    string `json:"word"`
	Default bool   `json:"default"`
}

// CaseParts lists what can be taken from a case before (spec 14.12.8).
func (s *Session) CaseParts() []CasePart {
	out := []CasePart{}
	for _, p := range ops.CaseParts {
		out = append(out, CasePart{p.ID, p.Word, p.Default})
	}
	return out
}

// CreateFrom makes a case folder, takes the settings of the case in from
// into it (parts: the IDs of CaseParts), and opens it. It gives what was
// taken in Home.Problems' place: the page tells it.
func (s *Session) CreateFrom(dir, name, kind, from string, parts []string) (*Home, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, err := cleanDir(dir)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(from) == "" {
		return nil, "", fmt.Errorf("設定を写す元の案件を選んでください")
	}
	if _, err := project.Open(from); err != nil {
		return nil, "", fmt.Errorf("設定を写す元の案件を開けません: %v", err)
	}
	if _, err := project.Init(dir, strings.TrimSpace(name), kind); err != nil {
		return nil, "", err
	}
	c, err := ops.CopyCaseSettings(from, dir, parts)
	said := ""
	switch {
	case err != nil:
		said = "案件は作りましたが、設定を写す途中で止まりました: " + err.Error()
	case c.Files == 0 && len(c.Parts) == 0:
		said = "元の案件に、写す設定はありませんでした。"
	default:
		said = fmt.Sprintf("元の案件 %s から写しました（ファイル %d 個）: %s", from, c.Files, strings.Join(c.Parts, "、"))
	}
	h, err := s.open(dir)
	if err != nil {
		return nil, "", err
	}
	return h, said, nil
}

// SampleCase makes the sample case in dir and opens it (spec 14.12.7).
func (s *Session) SampleCase(dir string) (*Home, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, err := cleanDir(dir)
	if err != nil {
		return nil, "", err
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return nil, "", fmt.Errorf("%s は空のフォルダではありません。見本の案件は、空のフォルダか、まだ無いフォルダに作ります", dir)
	}
	n, err := ops.MakeSampleCase(dir)
	if err != nil {
		return nil, "", err
	}
	h, err := s.open(dir)
	if err != nil {
		return nil, "", err
	}
	said := fmt.Sprintf("見本の案件を作りました。架空の旧コンフィグ（Cisco の拠点ルータ %d 台）を input/old に置いてあります。上の「次にやること」に沿って進めてください。新旧対照表（機器）では、新メーカーに nec_ix、新機種に ix-2ge-8sw を書くと、新旧対照表（ポート）の案が入ります。", n)
	return h, said, nil
}

// DraftCheck checks the book with the draft of the table put in, without
// writing it (spec 14.12.3).
func (s *Session) DraftCheck() (*ResultView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	now := s.now()
	d, err := e.DraftCheck(now)
	if err != nil {
		return nil, err
	}
	if !s.draft {
		s.kept = s.wrote
	}
	s.shown, s.draft = d.Outcome, true
	s.wrote = written{changes: ChangesView{Counts: []Pair{}, Notes: []string{}, Tables: []Table{}}}
	v := s.result(e, d.Outcome, now)
	v.Title = "下書きでチェックした結果（まだ保存していません）"
	v.Draft = true
	v.Moved = d.Moved
	v.Message = fmt.Sprintf("表の編集の下書き（%d 件）を当てたパラメータブックでチェックしました。パラメータブックは書いていません。承認は、保存してからチェックした結果で行います。", d.Draft)
	if d.Moved {
		v.Notice = "下書きで行を足した・動かした・消したため、指摘の行番号は保存した後のものです。指摘の場所は表と行のキーで見てください。"
	}
	return v, nil
}

// BookDevices lists the host names of the sheet 機器 of the book, in its
// order (for the copy of a device; spec 14.12.5).
func (s *Session) BookDevices() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	return e.BookHosts()
}
