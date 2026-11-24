package ops

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/project"
)

// CheckSetting is one check with what the case says of it (spec 14.11.5).
type CheckSetting struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Default string `json:"default"` // error | warning | info
	// Now is what the case says: "" (the default), error, warning, info or
	// "off" (the check is stopped).
	Now string `json:"now"`
	// Stays tells that the check cannot be stopped: a finding of it keeps a
	// device from being generated (spec 4.1).
	Stays bool `json:"stays"`
}

// CheckSettings is every check of the tool and what the case says of it, and
// the settings of IDs this version does not know (kept as they are).
type CheckSettings struct {
	Checks  []CheckSetting `json:"checks"`
	Unknown []string       `json:"unknown"`
}

// CheckSettings lists the checks with the settings of project.yaml.
func (e *Env) CheckSettings() *CheckSettings {
	c := e.Proj.Config.Checks
	off := map[string]bool{}
	for _, id := range c.Disable {
		off[id] = true
	}
	out := &CheckSettings{Checks: []CheckSetting{}, Unknown: []string{}}
	known := map[string]bool{}
	for _, r := range check.AllRules() {
		if known[r.ID] {
			continue
		}
		known[r.ID] = true
		s := CheckSetting{ID: r.ID, Title: r.Title, Default: r.Severity, Now: c.Severity[r.ID], Stays: check.Blocks(r.ID)}
		if off[r.ID] && !s.Stays {
			s.Now = "off"
		}
		if s.Now == r.Severity {
			s.Now = ""
		}
		out.Checks = append(out.Checks, s)
	}
	for id := range c.Severity {
		if !known[id] {
			out.Unknown = append(out.Unknown, id)
		}
	}
	for _, id := range c.Disable {
		if !known[id] && c.Severity[id] == "" {
			out.Unknown = append(out.Unknown, id)
		}
	}
	sort.Strings(out.Unknown)
	return out
}

// SaveCheckSettings writes what the case says of the checks into
// project.yaml: checks.severity and checks.disable, nothing else (spec 4.2,
// 14.11.5). set is ID → "" (the default) | error | warning | info | off, for
// the checks the tool has; the settings of IDs it does not know stay. A copy
// of project.yaml goes to state/backup first. It gives the copy ("" when
// nothing changed).
func (e *Env) SaveCheckSettings(set map[string]string, now time.Time) (string, error) {
	rules := map[string]check.Rule{}
	for _, r := range check.AllRules() {
		if _, dup := rules[r.ID]; !dup {
			rules[r.ID] = r
		}
	}
	c := e.Proj.Config.Checks
	sev := map[string]string{}
	var off []string
	for id, s := range c.Severity {
		if _, ok := rules[id]; !ok {
			sev[id] = s
		}
	}
	for _, id := range c.Disable {
		if _, ok := rules[id]; !ok {
			off = append(off, id)
		}
	}
	ids := make([]string, 0, len(rules))
	for id := range rules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	was := e.CheckSettings()
	before := map[string]string{}
	for _, s := range was.Checks {
		before[s.ID] = s.Now
	}
	for _, id := range ids {
		r := rules[id]
		v, ok := set[id]
		if !ok {
			v = before[id]
		}
		switch v {
		case "", r.Severity:
		case check.Error, check.Warning, check.Info:
			sev[id] = v
		case "off":
			if check.Blocks(id) {
				return "", fmt.Errorf("%s（%s）は止められません。値の誤りがある機器を生成しないためのチェックです", id, r.Title)
			}
			off = append(off, id)
		default:
			return "", fmt.Errorf("%s に「%s」は使えません（エラー・警告・情報・止める、または既定のまま）", id, v)
		}
	}
	sort.Strings(off)
	if sameChecks(c.Severity, c.Disable, sev, off) {
		return "", nil
	}
	path := e.Proj.Path(project.ConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s を読めません: %v", project.ConfigFile, err)
	}
	next, err := putChecks(data, sev, off)
	if err != nil {
		return "", fmt.Errorf("%s を書き換えられません: %v", project.ConfigFile, err)
	}
	copyTo, err := e.copyToBackup(project.ConfigFile, "project", data, now)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, next, 0o644); err != nil {
		return "", fmt.Errorf("%s を書けません: %v", project.ConfigFile, err)
	}
	e.Proj.Config.Checks.Severity, e.Proj.Config.Checks.Disable = sev, off
	e.Proj.Log(now, "チェックの設定を %s に書きました（重さを変えた %d 個・止めた %d 個。控え %s）", project.ConfigFile, len(sev), len(off), copyTo)
	return copyTo, nil
}

func sameChecks(sevA map[string]string, offA []string, sevB map[string]string, offB []string) bool {
	if len(sevA) != len(sevB) || len(offA) != len(offB) {
		return false
	}
	for k, v := range sevA {
		if sevB[k] != v {
			return false
		}
	}
	a := append([]string{}, offA...)
	sort.Strings(a)
	for i := range a {
		if a[i] != offB[i] {
			return false
		}
	}
	return true
}

// copyToBackup keeps the bytes of a file of the case in state/backup
// (prefix_<name>_<time>), the file staying where it is.
func (e *Env) copyToBackup(rel, prefix string, data []byte, now time.Time) (string, error) {
	ext := ""
	if i := strings.LastIndex(rel, "."); i > strings.LastIndex(rel, "/") {
		ext = rel[i:]
	}
	name := strings.TrimSuffix(rel[strings.LastIndex(rel, "/")+1:], ext)
	base := "state/backup/" + prefix + "_" + name + "_" + project.Stamp(now)
	to := base + ext
	for n := 2; ; n++ {
		if _, err := os.Stat(e.Proj.Path(to)); err != nil {
			break
		}
		to = fmt.Sprintf("%s_%d%s", base, n, ext)
	}
	if err := os.MkdirAll(e.Proj.Path("state/backup"), 0o755); err != nil {
		return "", fmt.Errorf("state/backup を作れません: %v", err)
	}
	if err := os.WriteFile(e.Proj.Path(to), data, 0o644); err != nil {
		return "", fmt.Errorf("%s を書けません: %v", to, err)
	}
	return to, nil
}

// putChecks gives project.yaml with checks.severity and checks.disable put
// in; the other lines and their comments stay.
func putChecks(data []byte, sev map[string]string, off []string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("先頭が設定の並びではありません")
	}
	top := doc.Content[0]
	checks := mapValue(top, "checks")
	if checks == nil {
		checks = &yaml.Node{Kind: yaml.MappingNode}
		top.Content = append(top.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "checks"}, checks)
	}
	if checks.Kind != yaml.MappingNode {
		checks.Kind, checks.Tag, checks.Value, checks.Content = yaml.MappingNode, "", "", nil
	}
	keys := make([]string, 0, len(sev))
	for k := range sev {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// One line each, as the file the tool makes has them: the comment on
	// the line stays where it is.
	sn := &yaml.Node{Kind: yaml.MappingNode, Style: yaml.FlowStyle}
	for _, k := range keys {
		sn.Content = append(sn.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: k}, &yaml.Node{Kind: yaml.ScalarNode, Value: sev[k]})
	}
	dn := &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle}
	for _, id := range off {
		dn.Content = append(dn.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: id})
	}
	setMap(checks, "disable", dn)
	setMap(checks, "severity", sn)
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func mapValue(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// setMap puts a value under a key of a mapping. The node that was there
// takes the new content, so the comments on it stay.
func setMap(m *yaml.Node, key string, v *yaml.Node) {
	if old := mapValue(m, key); old != nil {
		old.Kind, old.Style, old.Tag, old.Value, old.Content = v.Kind, v.Style, "", v.Value, v.Content
		return
	}
	m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, v)
}

// MakeSample writes a sample template of the delivery sheets into
// templates/ (spec 4.2, 14.11.2): a new file only, "_2" and on when the name
// is taken. It gives the path of the file.
func (e *Env) MakeSample(kind string, now time.Time) (string, error) {
	data, err := book.Sample(kind, e.Schema, e.lang())
	if err != nil {
		return "", err
	}
	name := book.SampleName(kind)
	ext := ".xlsx"
	base := "templates/" + strings.TrimSuffix(name, ext)
	rel := base + ext
	for n := 2; ; n++ {
		if _, err := os.Stat(e.Proj.Path(rel)); err != nil {
			break
		}
		rel = fmt.Sprintf("%s_%d%s", base, n, ext)
	}
	if err := os.MkdirAll(e.Proj.Path("templates"), 0o755); err != nil {
		return "", fmt.Errorf("templates を作れません: %v", err)
	}
	if err := os.WriteFile(e.Proj.Path(rel), data, 0o644); err != nil {
		return "", fmt.Errorf("%s を書けません: %v", rel, err)
	}
	e.Proj.Log(now, "納品用シートのひな形の見本を作りました: %s", rel)
	return rel, nil
}
