// Package ui makes what the screens show: it calls the steps of package ops
// and turns their outcomes into plain values the page can draw. It knows
// nothing of the window; the entry point of the tool binds a Session to it.
//
// A screen masks the secrets whatever the case says (spec 14).
//
// Spec: spec 14, 14.1.
package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/project"
)

// maxRecent is how many cases the list of recent cases keeps (spec 14.1).
const maxRecent = 10

// Session is what one window works on: the open case and the last outcomes.
type Session struct {
	// Progress, if set, is told before each file of an import and before each
	// device of a generation how many are done.
	Progress func(done, total int, file string)

	version  string
	settings string // the file of the settings ("" = none is kept)
	now      func() time.Time

	// base is the place of the cases by default (spec 14.15.1: projects
	// next to the exe); place is the one the user chose instead ("" = base).
	base  string
	place string

	mu      sync.Mutex
	recent  []Recent
	name    string              // the name the last approval was given with
	grid    map[string]GridPref // how the table of each sheet is seen (spec 14.6)
	edit    ops.EditCache       // the sheets of the book the table read last
	env     *ops.Env
	imports map[string]*ops.ImportOutcome // the last import of "old" and of "new"
	shown   *ops.GenerateOutcome          // the result the screen shows (nil = none)
	mig     *ops.Verdicts                 // the findings of a migration the screen showed last (nil = none)
	tree    *ops.TreeOutcome              // the comparison of every device the screen showed last (nil = none)
	wrote   written                       // what the last generation or check wrote
	// draft tells that shown is a check over the draft of the table (spec
	// 14.12.3): nothing is approved on it. kept is what wrote was before.
	draft bool
	kept  written
	stop    atomic.Bool
	zoom    int // the zoom of the page in per cent (0 = 100; spec 14.8.6)
	opens   openCache // the count of the unhandled lines (spec 14.12.1)

	// held is the mark of the open case (spec 14.8.2; nil = none could be
	// made); newer is the version of a newer ConfigChief that used the case
	// ("" = this one may write it; spec 14.8.3).
	held  *project.Held
	newer string
}

// SettingsFile gives the place of the settings of the user: the recent cases
// and the name for the approvals ("" if the system names no such place).
func SettingsFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "ConfigChief", "settings.json")
}

// New starts a session. settings is the file that keeps the recent cases
// ("" = keep none).
func New(version, settings string) *Session {
	s := &Session{version: version, settings: settings, now: time.Now, imports: map[string]*ops.ImportOutcome{}, base: DefaultPlace()}
	s.load()
	return s
}

// Recent is a case that was open before.
type Recent struct {
	Dir     string `json:"dir"`
	Name    string `json:"name"`
	Missing bool   `json:"missing"` // the folder is not a case folder any more
}

type settingsFile struct {
	Recent []Recent `json:"recent"`
	Name   string   `json:"name,omitempty"` // who approves (spec 11.4, 14.2)
	// Grid: sheet → the hidden columns and the widths of the table over the
	// book (spec 14.6).
	Grid map[string]GridPref `json:"grid,omitempty"`
	// Zoom: the zoom of the page in per cent (spec 14.8.6; 0 = 100).
	Zoom int `json:"zoom,omitempty"`
	// Place: where the cases are made, if not next to the exe (spec
	// 14.15.1).
	Place string `json:"place,omitempty"`
}

func (s *Session) load() {
	if s.settings == "" {
		return
	}
	data, err := os.ReadFile(s.settings)
	if err != nil {
		return
	}
	var f settingsFile
	if json.Unmarshal(data, &f) != nil {
		return // a broken file is as good as none: it is written anew
	}
	s.name = strings.TrimSpace(f.Name)
	s.zoom = cleanZoom(f.Zoom)
	s.place = strings.TrimSpace(f.Place)
	s.grid = map[string]GridPref{}
	for k, p := range f.Grid {
		if p = p.clean(); k != "" && len(p.Hidden)+len(p.Widths) > 0 {
			s.grid[k] = p
		}
	}
	for _, r := range f.Recent {
		if r.Dir != "" && len(s.recent) < maxRecent {
			s.recent = append(s.recent, Recent{Dir: r.Dir, Name: r.Name})
		}
	}
}

// save writes the settings. Settings that cannot be written do not stop the
// work.
func (s *Session) save() {
	if s.settings == "" {
		return
	}
	f := settingsFile{Recent: []Recent{}, Name: s.name, Grid: s.grid, Zoom: s.zoom, Place: s.place}
	for _, r := range s.recent {
		f.Recent = append(f.Recent, Recent{Dir: r.Dir, Name: r.Name})
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(s.settings), 0o755) != nil {
		return
	}
	os.WriteFile(s.settings, append(data, '\n'), 0o644)
}

// remember puts a case at the top of the recent ones.
func (s *Session) remember(dir, name string) {
	list := []Recent{{Dir: dir, Name: name}}
	for _, r := range s.recent {
		if !sameDir(r.Dir, dir) && len(list) < maxRecent {
			list = append(list, r)
		}
	}
	s.recent = list
	s.save()
}

func sameDir(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// Case names the open case.
type Case struct {
	Dir  string `json:"dir"`
	Name string `json:"name"`
	Kind string `json:"kind"` // "更改" | "新規"
	Book string `json:"book"` // the file name of the parameter book
}

// State is what the page needs to draw its frame.
type State struct {
	Version string   `json:"version"`
	Case    *Case    `json:"case"` // nil = no case is open
	Recent  []Recent `json:"recent"`
	// ReadOnly says why the open case is only read ("" = it is not; spec
	// 14.8.3).
	ReadOnly string `json:"readOnly"`
	Zoom     int    `json:"zoom"` // per cent
}

// State tells which case is open and which were open before.
func (s *Session) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state()
}

func (s *Session) state() State {
	st := State{Version: s.version, Recent: []Recent{}, Zoom: s.zoom}
	if st.Zoom == 0 {
		st.Zoom = 100
	}
	if s.env != nil {
		st.Case = caseOf(s.env)
		if s.newer != "" {
			st.ReadOnly = (&project.ReadOnlyError{Newer: s.newer, Mine: s.version}).Error()
		}
	}
	place := s.placeDir()
	for _, r := range s.recent {
		if place != "" && sameDir(filepath.Dir(r.Dir), place) {
			continue // the list of the place has it (spec 14.15.1)
		}
		if _, err := os.Stat(filepath.Join(r.Dir, project.ConfigFile)); err != nil {
			r.Missing = true
		}
		st.Recent = append(st.Recent, r)
	}
	return st
}

func caseOf(e *ops.Env) *Case {
	c := e.Proj.Config
	kind := "更改"
	if c.Kind == "new" {
		kind = "新規"
	}
	return &Case{Dir: e.Proj.Dir, Name: c.Name, Kind: kind, Book: c.Book}
}

// Create makes a case folder in dir and opens it. kind is "migration" or
// "new"; an empty name takes the name of the folder.
func (s *Session) Create(dir, name, kind string) (*Home, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, err := cleanDir(dir)
	if err != nil {
		return nil, err
	}
	if _, err := project.Init(dir, strings.TrimSpace(name), kind); err != nil {
		return nil, err
	}
	return s.open(dir)
}

// Open opens a case folder.
func (s *Session) Open(dir string) (*Home, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, err := cleanDir(dir)
	if err != nil {
		return nil, err
	}
	return s.open(dir)
}

func cleanDir(dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return "", errors.New("フォルダを選んでください")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("%s を開けません: %v", dir, err)
	}
	return abs, nil
}

func (s *Session) open(dir string) (*Home, error) {
	// The mark of the case first (spec 14.8.2): a case open elsewhere is not
	// read at all. The case this window has open keeps its mark.
	var held *project.Held
	same := s.env != nil && sameDir(s.env.Proj.Dir, dir)
	if !same && lockCases {
		h, err := project.Lock(dir)
		if err != nil {
			return nil, err
		}
		held = h
	}
	e, err := ops.Open(dir)
	if err != nil {
		held.Release()
		return nil, err
	}
	if !same {
		s.held.Release()
		s.held = held
	}
	s.newer = project.NoteTool(dir, s.version)
	s.use(e)
	s.forget()
	s.remember(dir, e.Proj.Config.Name)
	return s.home(e)
}

// forget drops the outcomes of the case that was open before.
func (s *Session) forget() {
	s.imports = map[string]*ops.ImportOutcome{}
	s.mig, s.tree = nil, nil
	s.shown, s.wrote, s.draft, s.kept = nil, written{}, false, written{}
}

func (s *Session) use(e *ops.Env) {
	e.Version = s.version
	e.Tick = s.tick
	e.Edit = &s.edit
	s.env = e
}

// Close closes the case; the page shows the list of cases again.
func (s *Session) Close() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.env = nil
	s.held.Release()
	s.held, s.newer = nil, ""
	s.forget()
	return s.state()
}

// Forget takes a case off the list of recent ones. The folder stays.
func (s *Session) Forget(dir string) State {
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []Recent
	for _, r := range s.recent {
		if !sameDir(r.Dir, dir) {
			list = append(list, r)
		}
	}
	s.recent = list
	s.save()
	return s.state()
}

// errStopped is what a step gives when the user stopped it.
var errStopped = errors.New("止めました。途中までの結果は書いていません")

// Cancel stops the step that is running, before its next file or device. The
// step then writes nothing (spec 14).
func (s *Session) Cancel() { s.stop.Store(true) }

func (s *Session) tick(done, total int, file string) error {
	if s.stop.Load() {
		return errStopped
	}
	if s.Progress != nil {
		s.Progress(done, total, file)
	}
	return nil
}

// begin starts a step on the open case. The case is read anew each time:
// what the user changed in project.yaml or under packs/ since counts, as it
// does on the command line.
func (s *Session) begin() (*ops.Env, error) {
	if s.env == nil {
		return nil, errors.New("案件を開いていません")
	}
	e, err := ops.Open(s.env.Proj.Dir)
	if err != nil {
		return nil, err
	}
	s.use(e)
	s.stop.Store(false)
	return e, nil
}

// lockCases: a case is held while it is open (spec 14.8.2). The tests of the
// package open one case from several sessions and turn it off.
var lockCases = true

// beginWrite is begin for a step that writes the case: a case used by a
// newer ConfigChief is only read (spec 14.8.3).
func (s *Session) beginWrite() (*ops.Env, error) {
	if s.env != nil && s.newer != "" {
		return nil, &project.ReadOnlyError{Newer: s.newer, Mine: s.version}
	}
	return s.begin()
}

// ReadOnly tells whether the open case is only read.
func (s *Session) readOnly() bool { return s.env != nil && s.newer != "" }

// cleanZoom keeps the zoom between 70 and 150 per cent, in steps of 10 (0 =
// 100, nothing chosen).
func cleanZoom(z int) int {
	if z == 0 {
		return 0
	}
	z = (z + 5) / 10 * 10
	if z < 70 {
		z = 70
	}
	if z > 150 {
		z = 150
	}
	if z == 100 {
		return 0
	}
	return z
}

// SetZoom keeps the zoom of the page (spec 14.8.6) and gives it as kept.
func (s *Session) SetZoom(z int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.zoom = cleanZoom(z)
	s.save()
	if s.zoom == 0 {
		return 100
	}
	return s.zoom
}

// PackInfo names a vendor pack that can be used.
type PackInfo struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// Home is the first screen of a case (spec 14.1).
type Home struct {
	Case     Case       `json:"case"`
	Old      int        `json:"old"`      // files in input/old
	New      int        `json:"new"`      // files in input/new
	HasBook  bool       `json:"hasBook"`  // the parameter book is there
	BookTime string     `json:"bookTime"` // when it was written last
	LastOut  string     `json:"lastOut"`  // the newest folder under out/ ("" = none)
	LastGen  string     `json:"lastGen"`  // when the configurations were generated last ("" = never)
	Packs    []PackInfo `json:"packs"`
	Problems []string   `json:"problems"` // packs of the case folder that are not used
	// Draft is the number of changes made in the tables of the screen that
	// are not in the parameter book yet (spec 14.6).
	Draft int `json:"draft"`
	// Shared says the case is on a shared place ("" = it is not; spec
	// 14.8.13).
	Shared string `json:"shared"`
	// DeliveryNames is the language of the names of the delivery set: "ja"
	// | "en" (spec 14.15.4).
	DeliveryNames string `json:"deliveryNames"`
}

// Home tells where the case stands.
func (s *Session) Home() (*Home, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	return s.home(e)
}

func (s *Session) home(e *ops.Env) (*Home, error) {
	h := &Home{Case: *caseOf(e), Packs: []PackInfo{}, Problems: strs(e.PackProblems), LastGen: e.LastGeneration(), Draft: e.DraftCount()}
	old, err := e.Proj.Inputs("old")
	if err != nil {
		return nil, err
	}
	nw, err := e.Proj.Inputs("new")
	if err != nil {
		return nil, err
	}
	h.Old, h.New = len(old), len(nw)
	h.DeliveryNames = "ja"
	if e.Proj.Config.Export.DeliveryNames == "en" {
		h.DeliveryNames = "en"
	}
	if why := project.SharedPlace(e.Proj.Dir); why != "" {
		h.Shared = fmt.Sprintf(project.SharedNote, why)
	}
	if info, err := os.Stat(e.Proj.BookPath()); err == nil {
		h.HasBook = true
		h.BookTime = info.ModTime().Format("2006-01-02 15:04")
	}
	if entries, err := os.ReadDir(e.Proj.Path("out")); err == nil {
		var dirs []string
		for _, ent := range entries {
			if ent.IsDir() {
				dirs = append(dirs, ent.Name())
			}
		}
		sort.Strings(dirs)
		if len(dirs) > 0 {
			h.LastOut = "out/" + dirs[len(dirs)-1]
		}
	}
	for _, id := range e.PackIDs {
		h.Packs = append(h.Packs, PackInfo{ID: id, Version: e.Packs[id].Version})
	}
	return h, nil
}

// Path gives the place of a file or a folder of the case folder, for opening
// it with the program of the system. rel is written with "/" ("" = the case
// folder itself). Nothing outside the case folder is given, and of the files
// only what the tool writes for a person to read: a book or a report.
func (s *Session) Path(rel string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil {
		return "", errors.New("案件を開いていません")
	}
	if strings.ContainsAny(rel, `\:`) || strings.HasPrefix(rel, "/") {
		return "", fmt.Errorf("%s は開けません（案件フォルダの中の場所ではありません）", rel)
	}
	for _, part := range strings.Split(rel, "/") {
		if part == ".." {
			return "", fmt.Errorf("%s は開けません（案件フォルダの中の場所ではありません）", rel)
		}
	}
	path := s.env.Proj.Path(rel)
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) && caseFolder(rel) {
		// A folder of the case is made when it is first used (spec
		// 14.15.5): opening it is a use.
		if err := os.MkdirAll(path, 0o755); err != nil {
			return "", fmt.Errorf("%s を作れません: %v", rel, err)
		}
		info, err = os.Stat(path)
	}
	if err != nil {
		return "", fmt.Errorf("%s は、まだありません", rel)
	}
	if ext := strings.ToLower(filepath.Ext(path)); !info.IsDir() && ext != ".xlsx" && ext != ".html" {
		return "", fmt.Errorf("%s は開けません（開けるのはフォルダと .xlsx、.html だけです）", rel)
	}
	return path, nil
}

// strs gives a list that is never nil: the page gets [] and not null.
func strs(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

// caseFolder tells whether rel is one of the folders a case may have.
func caseFolder(rel string) bool {
	for _, f := range project.Folders {
		if rel == f {
			return true
		}
	}
	return false
}
