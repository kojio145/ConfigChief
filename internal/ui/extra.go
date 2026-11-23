package ui

import (
	"fmt"
	"strings"

	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/project"
)

// The extra configurations written on the screen (spec 14.11.3).

// ExtraPlaces lists the extra configurations and what a new one can go to.
func (s *Session) ExtraPlaces() (*ops.ExtraPlaces, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	return e.ExtraPlaces()
}

// ExtraName gives the file name for a target ("all", "group", "device").
func (s *Session) ExtraName(kind, target string) (string, error) {
	return ops.ExtraName(kind, target)
}

// ReadExtra gives the text of an extra configuration.
func (s *Session) ReadExtra(name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return "", err
	}
	return e.ReadExtra(name)
}

// ExtraFrom names the unhandled lines an extra configuration was drafted
// from: after the save, the lines of the device are judged "manual" if
// they were not judged or on hold.
type ExtraFrom struct {
	Source string `json:"source"`
	Group  string `json:"group"`
	Device string `json:"device"`
	By     string `json:"by"`
}

// ExtraSavedView is what saving an extra configuration did.
type ExtraSavedView struct {
	Message string   `json:"message"`
	Notes   []string `json:"notes"`
}

// SaveExtra writes an extra configuration.
func (s *Session) SaveExtra(name, text string, from *ExtraFrom) (*ExtraSavedView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	now := s.now()
	saved, err := e.SaveExtra(name, text, now)
	if err != nil {
		return nil, err
	}
	v := &ExtraSavedView{Message: "extra/" + name + " を保存しました", Notes: saved.Notes}
	if saved.Backup != "" {
		v.Message += "（前の中身の控え: " + saved.Backup + "）"
	}
	if from == nil || from.Group == "" || from.Device == "" {
		return v, nil
	}
	by := strings.TrimSpace(from.By)
	if by == "" {
		by = s.name
	}
	j, err := e.Review(from.Source)
	if err != nil {
		v.Notes = append(v.Notes, "読めなかった行に判定を付けられませんでした: "+err.Error())
		return v, nil
	}
	for _, g := range j.Groups {
		if g.ID != from.Group {
			continue
		}
		for _, d := range g.Devices {
			if d.Device != from.Device || (d.Verdict != "" && d.Verdict != project.VerdictHold) {
				continue
			}
			if by == "" {
				v.Notes = append(v.Notes, "判定した人の名前が無いので、読めなかった行に判定を付けていません（読めなかった行の画面で付けてください）")
				return v, nil
			}
			o, err := e.JudgeLines(ops.JudgeRequest{Source: from.Source, Group: from.Group, Devices: from.Device,
				Verdict: project.VerdictManual, Reason: "追加コンフィグ extra/" + name + " に書いた", By: by}, now)
			if err != nil {
				v.Notes = append(v.Notes, "読めなかった行に判定を付けられませんでした: "+err.Error())
				return v, nil
			}
			s.judged(from.Source, o.Judging)
			v.Message += fmt.Sprintf("。%s のかたまり %s に「追加コンフィグで書く」を付けました", from.Device, from.Group)
		}
	}
	return v, nil
}

// DeleteExtra moves an extra configuration to state/backup.
func (s *Session) DeleteExtra(name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.beginWrite()
	if err != nil {
		return "", err
	}
	to, err := e.DeleteExtra(name, s.now())
	if err != nil {
		return "", err
	}
	return "extra/" + name + " を消しました（" + to + " に移しました）", nil
}

// DraftExtra gives the extra configuration of a device with the lines of a
// group of unhandled lines added at its end. It writes nothing.
func (s *Session) DraftExtra(source, group, device string) (*ops.ExtraDraft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := checkSource(source); err != nil {
		return nil, err
	}
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	return e.DraftExtra(source, group, device)
}
