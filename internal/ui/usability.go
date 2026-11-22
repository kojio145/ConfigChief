package ui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kojio145/configchief/internal/ops"
)

// The things of spec 14.10 that save a person work.

// AddView is what adding files did (spec 14.10.1).
type AddView struct {
	Message string `json:"message"`
	Copied  int    `json:"copied"`
	// Clash are the names that are there with another content: nothing was
	// written for them. The page asks, and sends them again to write over.
	Clash []string `json:"clash"`
	Files int      `json:"files"` // the files of the folder now
}

// AddFiles copies files a person dropped or chose into input/<source>.
func (s *Session) AddFiles(source string, paths []string, overwrite bool) (*AddView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	a, err := e.AddInputs(source, paths, overwrite, s.now())
	if err != nil {
		return nil, err
	}
	v := &AddView{Message: a.Said(), Copied: len(a.Copied), Clash: a.Clash}
	if len(a.Clash) > 0 && len(a.Copied)+len(a.Same)+len(a.Skipped) == 0 {
		v.Message = ""
	}
	if list, err := e.Proj.Inputs(source); err == nil {
		v.Files = len(list)
	}
	return v, nil
}

// Next tells the one thing to do next (spec 14.10.4); nil when it cannot
// be told (the book cannot be read, say).
func (s *Session) Next() *ops.Next {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil
	}
	n, err := e.NextStep()
	if err != nil {
		return nil
	}
	return n
}

// LastResult gives the result of the newest output of "generate" or
// "check" after the case was opened again (spec 14.10.6): the checks run
// once more and write nothing, and the files are those of that output. nil
// when there is no such output.
func (s *Session) LastResult() (*ResultView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	dir := e.LastOutput()
	if dir == "" {
		return nil, nil
	}
	now := s.now()
	o, err := e.Generate(ops.GenOptions{CheckOnly: true, Dry: true}, now)
	if err != nil {
		return nil, err
	}
	configs, unverified := o.Attach(e, dir)
	ch := ChangesView{Counts: []Pair{}, Notes: []string{}, Tables: []Table{}}
	if _, err := os.Stat(e.Proj.Path(dir + "/reports/changes.xlsx")); err == nil {
		ch.Report = Files{dir + "/reports/changes.xlsx", dir + "/reports/changes.html"}
	}
	s.shown = o
	s.wrote = written{dir: dir, configs: configs, unverified: unverified,
		report: Files{o.Report.XLSX, o.Report.HTML}, changes: ch}
	v := s.result(e, o, now)
	when := ops.Stamped(dir)
	v.Title = "前回の結果（" + when + " の出力）"
	v.Message = "前回の出力 " + dir + " を開き、指摘は今のパラメータブックと承認で数え直しました（ファイルは書いていません）。"
	if changed := newerThan(e, dir); changed != "" {
		v.Notice = changed + "出力フォルダのファイルは、前のブックから作ったものです。もう一度「生成とチェック」を押すと新しくなります。"
	}
	return v, nil
}

// newerThan tells what changed after the output folder was written ("" =
// nothing): the parameter book or the approvals.
func newerThan(e *ops.Env, dir string) string {
	out, err := os.Stat(e.Proj.Path(dir + "/reports/check_results.xlsx"))
	if err != nil {
		return ""
	}
	var what []string
	if info, err := os.Stat(e.Proj.BookPath()); err == nil && info.ModTime().After(out.ModTime()) {
		what = append(what, "パラメータブック")
	}
	if info, err := os.Stat(e.Proj.Path("state/approvals.yaml")); err == nil && info.ModTime().After(out.ModTime()) {
		what = append(what, "承認")
	}
	if len(what) == 0 {
		return ""
	}
	return strings.Join(what, "と") + "が、この出力の後に変わりました。"
}

// ApproveMany approves the findings picked in the list, one approval each,
// with one reason (appendix C 3.2, spec 14.10.8). A finding that cannot be
// approved is left out and told. The checks then run again; no file is
// written by them.
func (s *Session) ApproveMany(keys []string, reason, by string) (*ResultView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil || s.shown == nil {
		return nil, errors.New("先に「生成とチェック」か「チェックだけ」を行ってください")
	}
	if strings.TrimSpace(reason) == "" || strings.TrimSpace(by) == "" {
		return nil, errors.New("理由と承認者を書いてください")
	}
	if len(keys) == 0 {
		return nil, errors.New("承認する指摘を選んでください")
	}
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	now := s.now()
	done, left, err := e.ApproveFindings(s.shown.Items, keys, reason, by, now)
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(by); name != s.name {
		s.name = name
		s.save()
	}
	msg := fmt.Sprintf("%d 件を承認しました（1 件ずつの承認として残しました）", len(done))
	if len(left) > 0 {
		msg += fmt.Sprintf("。%d 件は承認できませんでした: %s", len(left), firstKinds(left))
	}
	return s.recheck(e, now, msg)
}

// firstKinds gives the reasons of a list, each once.
func firstKinds(list []string) string {
	seen := map[string]bool{}
	var out []string
	for _, x := range list {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	if len(out) > 3 {
		out = append(out[:3], "ほか")
	}
	return strings.Join(out, "／")
}

// BookWritable tells whether the parameter book can be written now: the
// page waits for Excel to let it go (spec 14.10.7).
func (s *Session) BookWritable() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil {
		return true
	}
	return s.env.Proj.BookWritable() == nil
}
