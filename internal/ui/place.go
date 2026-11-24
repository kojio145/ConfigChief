package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/project"
)

// The place of the cases (spec 14.15): every case is a folder named after it
// under one place, projects next to the exe unless the user chose another.

// PlaceFolder is the name of the place next to the exe (spec 14.15.4).
const PlaceFolder = "projects"

// SampleFolder is the name of the folder of the sample case (spec 14.15.3).
const SampleFolder = "sample"

// DefaultPlace gives projects next to the exe ("" if the exe is not known).
func DefaultPlace() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return filepath.Join(filepath.Dir(exe), PlaceFolder)
}

// placeDir gives the place the cases are made in.
func (s *Session) placeDir() string {
	if s.place != "" {
		return s.place
	}
	return s.base
}

// PlaceView is what the page of the cases shows of the place.
type PlaceView struct {
	Dir     string   `json:"dir"`
	Default bool     `json:"default"` // the place next to the exe
	Shared  string   `json:"shared"`  // why it is a shared place ("" = it is not)
	Cases   []Recent `json:"cases"`   // the cases in it, by name
}

// Place tells the place of the cases and the cases in it.
func (s *Session) Place() PlaceView {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.placeView()
}

func (s *Session) placeView() PlaceView {
	dir := s.placeDir()
	v := PlaceView{Dir: dir, Default: s.place == "", Cases: []Recent{}}
	if dir == "" {
		return v
	}
	if why := project.SharedPlace(dir); why != "" {
		v.Shared = why
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return v
	}
	for _, en := range entries {
		if !en.IsDir() || strings.HasPrefix(en.Name(), ".") {
			continue
		}
		sub := filepath.Join(dir, en.Name())
		p, err := project.Open(sub)
		if err != nil {
			continue
		}
		name := p.Config.Name
		if name == "" {
			name = en.Name()
		}
		v.Cases = append(v.Cases, Recent{Dir: sub, Name: name})
	}
	sort.SliceStable(v.Cases, func(i, j int) bool {
		return strings.ToLower(filepath.Base(v.Cases[i].Dir)) < strings.ToLower(filepath.Base(v.Cases[j].Dir))
	})
	return v
}

// SetPlace makes dir the place of the cases ("" = the one next to the exe).
func (s *Session) SetPlace(dir string) (PlaceView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir = strings.TrimSpace(dir)
	if dir != "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return PlaceView{}, fmt.Errorf("%s を置き場所にできません: %v", dir, err)
		}
		if _, err := os.Stat(filepath.Join(abs, project.ConfigFile)); err == nil {
			return PlaceView{}, fmt.Errorf("%s は案件フォルダです。置き場所には、案件フォルダを並べる親のフォルダを選んでください", abs)
		}
		dir = abs
		if s.base != "" && sameDir(dir, s.base) {
			dir = ""
		}
	}
	s.place = dir
	s.save()
	return s.placeView(), nil
}

// makePlace makes the place if it is not there yet, and tells plainly when
// it cannot be written.
func (s *Session) makePlace() (string, error) {
	dir := s.placeDir()
	if dir == "" {
		return "", errors.New("案件の置き場所が分かりません。「置き場所を変える」で選んでください")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", placeError(dir, err)
	}
	return dir, nil
}

func placeError(dir string, err error) error {
	return fmt.Errorf("置き場所 %s に書けません（%v）。ConfigChief.exe を書ける場所（例: C:\\ConfigChief）に置くか、「置き場所を変える」で選んでください", dir, err)
}

// CaseDir gives the folder a new case named name is made in (spec 14.15.2):
// <place>\<name>. A folder of that name that has something in it is refused.
func (s *Session) CaseDir(name string) (dir, clean string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.caseDir(name)
}

func (s *Session) caseDir(name string) (dir, clean string, err error) {
	clean, err = project.CleanName(name)
	if err != nil {
		return "", "", err
	}
	place, err := s.makePlace()
	if err != nil {
		return "", "", err
	}
	dir = filepath.Join(place, clean)
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return "", "", fmt.Errorf("置き場所に、同じ名前のフォルダ（%s）があります。別の名前にしてください", dir)
	}
	return dir, clean, nil
}

// freeDir gives <place>\<name>, or <name>_2, _3 … when that is taken.
func freeDir(place, name string) string {
	dir := filepath.Join(place, name)
	for i := 2; ; i++ {
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			return dir
		}
		dir = filepath.Join(place, fmt.Sprintf("%s_%d", name, i))
	}
}

// NewCase makes a case named name in the place and opens it (spec 14.15.2).
func (s *Session) NewCase(name, kind string) (*Home, error) {
	dir, clean, err := s.CaseDir(name)
	if err != nil {
		return nil, err
	}
	return s.Create(dir, clean, kind)
}

// NewCaseFrom makes a case named name in the place with the settings of the
// case in from (spec 14.12.8, 14.15.2).
func (s *Session) NewCaseFrom(name, kind, from string, parts []string) (*Home, string, error) {
	dir, clean, err := s.CaseDir(name)
	if err != nil {
		return nil, "", err
	}
	return s.CreateFrom(dir, clean, kind, from, parts)
}

// NewSample makes the sample case in <place>\sample (spec 14.15.3).
func (s *Session) NewSample() (*Home, string, error) {
	s.mu.Lock()
	place, err := s.makePlace()
	s.mu.Unlock()
	if err != nil {
		return nil, "", err
	}
	return s.SampleCase(freeDir(place, SampleFolder))
}

// ImportHere spreads a case file in the place (spec 14.9.2, 14.15.3).
func (s *Session) ImportHere(path, password string) (*CaseImportView, error) {
	s.mu.Lock()
	place, err := s.makePlace()
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return s.CaseImport(path, password, place)
}

// SetDeliveryNames chooses the language of the names of the delivery set of
// the open case (spec 14.15.4) and gives the home again.
func (s *Session) SetDeliveryNames(lang string) (*Home, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	if _, err := e.SetDeliveryNames(lang, s.now()); err != nil {
		return nil, err
	}
	return s.home(e)
}

// MadePlace gives the place of the cases, made if it is not there yet (for
// opening it in the explorer).
func (s *Session) MadePlace() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.makePlace()
}

// RenameCase gives the open case a new name (spec 14.16.3): the folder is
// renamed in the folder it is in, and project.yaml takes the name. When the
// folder cannot be renamed, the case is opened again as it was and nothing
// changes.
func (s *Session) RenameCase(name string) (*Home, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil {
		return nil, errors.New("案件を開いていません")
	}
	if _, err := s.beginWrite(); err != nil {
		return nil, err
	}
	clean, err := project.CleanName(name)
	if err != nil {
		return nil, err
	}
	oldDir := s.env.Proj.Dir
	newDir := filepath.Join(filepath.Dir(oldDir), clean)
	if newDir != oldDir {
		if !sameDir(newDir, oldDir) {
			if _, err := os.Stat(newDir); err == nil {
				return nil, fmt.Errorf("同じ名前のフォルダ（%s）があります。別の名前にしてください", newDir)
			}
		}
		// The mark of the case is let go first: the folder cannot be
		// renamed while a file in it is held.
		s.env = nil
		s.held.Release()
		s.held = nil
		if err := os.Rename(oldDir, newDir); err != nil {
			if _, again := s.open(oldDir); again != nil {
				return nil, fmt.Errorf("フォルダの名前を変えられませんでした（%v）。開き直すこともできませんでした: %v", err, again)
			}
			return nil, fmt.Errorf("フォルダの名前を変えられません。案件のファイルを Excel やエクスプローラで開いていれば、閉じてからもう一度押してください（%v）", err)
		}
		var list []Recent
		for _, r := range s.recent {
			if !sameDir(r.Dir, oldDir) {
				list = append(list, r)
			}
		}
		s.recent = list
		s.edit = ops.EditCache{}
		if _, err := s.open(newDir); err != nil {
			return nil, fmt.Errorf("フォルダの名前は %s に変えましたが、開けませんでした: %v", newDir, err)
		}
	}
	if err := s.env.SetCaseName(clean, s.now()); err != nil {
		return nil, err
	}
	s.remember(newDir, clean)
	return s.home(s.env)
}
