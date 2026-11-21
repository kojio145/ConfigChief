package ops

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/fill"
	"github.com/kojio145/configchief/internal/listimport"
	"github.com/kojio145/configchief/internal/project"
)

// What the screen of the templates needs (spec 14.7): saving and removing
// templates and variable rules, a reading of a rule that writes nothing, and
// a first template from one device.

var fileNameRe = regexp.MustCompile(`^[\p{L}_][\p{L}\p{N}_\-]*$`)

// uniqueRel gives dir/name.ext, with _2… after the name if that file is there.
func (e *Env) uniqueRel(dir, name, ext string) string {
	rel := dir + "/" + name + ext
	for n := 2; ; n++ {
		if _, err := os.Stat(e.Proj.Path(rel)); err != nil {
			return rel
		}
		rel = fmt.Sprintf("%s/%s_%d%s", dir, name, n, ext)
	}
}

// SaveTemplate writes a template: over the file it was read from (rel), or
// as fill/<name>.txt for a new one. A template whose form has a fault is
// written all the same (a person may stop halfway); the fault is given.
func (e *Env) SaveTemplate(rel, name, text string) (string, string, error) {
	if rel != "" {
		ts, err := e.FillTemplates()
		if err != nil {
			return "", "", err
		}
		ok := false
		for _, t := range ts {
			ok = ok || t.Path == rel
		}
		if !ok {
			return "", "", fmt.Errorf("%s は、この案件のひな形ではありません", rel)
		}
	} else {
		name = strings.TrimSpace(strings.TrimSuffix(name, ".txt"))
		if !fileNameRe.MatchString(name) {
			return "", "", fmt.Errorf("ひな形の名前「%s」: 文字か _ で始め、文字・数字・_・- で付けます", name)
		}
		rel = e.uniqueRel(fillDir, name, ".txt")
	}
	if err := os.MkdirAll(e.Proj.Path(fillDir), 0o755); err != nil {
		return "", "", fmt.Errorf("%s を作れません: %v", fillDir, err)
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if !strings.HasSuffix(text, "\n") && text != "" {
		text += "\n"
	}
	if err := os.WriteFile(e.Proj.Path(rel), []byte(text), 0o644); err != nil {
		return "", "", fmt.Errorf("%s を書けません: %v", rel, err)
	}
	fault := ""
	if _, err := fill.Parse(rel, text); err != nil {
		fault = err.Error()
	}
	return rel, fault, nil
}

// backup moves a file of the case to state/backup (prefix_<name>_<time>).
func (e *Env) backup(rel, prefix string, now time.Time) (string, error) {
	ext := path.Ext(rel)
	base := "state/backup/" + prefix + "_" + strings.TrimSuffix(path.Base(rel), ext) + "_" + project.Stamp(now)
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
	if err := os.Rename(e.Proj.Path(rel), e.Proj.Path(to)); err != nil {
		return "", fmt.Errorf("%s を消せません: %v", rel, err)
	}
	return to, nil
}

// DeleteTemplate moves a template to state/backup.
func (e *Env) DeleteTemplate(rel string, now time.Time) (string, error) {
	ts, err := e.FillTemplates()
	if err != nil {
		return "", err
	}
	for _, t := range ts {
		if t.Path == rel {
			return e.backup(rel, "fill", now)
		}
	}
	return "", fmt.Errorf("%s は、この案件のひな形ではありません", rel)
}

// SaveVarRule writes a variable rule: over its file, or as vars/<name>.yaml.
// A rule with a fault is not written.
func (e *Env) SaveVarRule(rel, name string, r *listimport.VarRule) (string, error) {
	rule := *r
	if faults := rule.Validate(); len(faults) > 0 {
		return "", fmt.Errorf("この変数ルールは保存できません:\n  %s", strings.Join(faults, "\n  "))
	}
	if rel != "" {
		files, err := e.VarFiles()
		if err != nil {
			return "", err
		}
		ok := false
		for _, f := range files {
			ok = ok || (f.Path == rel && strings.HasSuffix(strings.ToLower(rel), ".yaml"))
		}
		if !ok {
			return "", fmt.Errorf("%s は、この案件の変数ルールではありません", rel)
		}
	} else {
		name = strings.TrimSpace(strings.TrimSuffix(name, ".yaml"))
		if !fileNameRe.MatchString(name) {
			return "", fmt.Errorf("変数ルールの名前「%s」: 文字か _ で始め、文字・数字・_・- で付けます（ひな形から {{#each 名前}} で呼ぶため）", name)
		}
		files, err := e.VarFiles()
		if err != nil {
			return "", err
		}
		for _, f := range files {
			if strings.EqualFold(strings.TrimSuffix(path.Base(f.Path), ".yaml"), name) {
				return "", fmt.Errorf("変数ルール「%s」はもうあります（%s）", name, f.Path)
			}
		}
		rel = varsDir + "/" + name + ".yaml"
	}
	data, err := listimport.MarshalVars(&rule)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(e.Proj.Path(varsDir), 0o755); err != nil {
		return "", fmt.Errorf("%s を作れません: %v", varsDir, err)
	}
	head := "# 変数ルール（仕様書 9.7.2）。一覧のどの列をどの変数にするか。画面の「ひな形から作る」で保存したものです。\n"
	if err := os.WriteFile(e.Proj.Path(rel), append([]byte(head), data...), 0o644); err != nil {
		return "", fmt.Errorf("%s を書けません: %v", rel, err)
	}
	return rel, nil
}

// DeleteVarRule moves a variable rule to state/backup.
func (e *Env) DeleteVarRule(rel string, now time.Time) (string, error) {
	files, err := e.VarFiles()
	if err != nil {
		return "", err
	}
	for _, f := range files {
		if f.Path == rel {
			return e.backup(rel, "vars", now)
		}
	}
	return "", fmt.Errorf("%s は、この案件の変数ルールではありません", rel)
}

// VarTry is a reading of a variable rule that writes nothing (PalaTerm's
// FillRuleTest and FillListRead).
type VarTry struct {
	Faults   []string
	File     string
	Lines    int
	Excluded int
	Devices  int
	Missing  []string
	Warnings []string
	// Sample are the first lines of the list as the rule reads them.
	Sample []listimport.VarLine
	Header []string   // 機器, 一覧の行, the variables
	Rows   [][]string // each device's rows, at most tryRows
	Total  int
	// Unknown are the devices of the list the book does not have (the
	// first few); Known how many it has. NoBook: the case has no book.
	Unknown []string
	Known   int
	NoBook  bool
}

// TryVarRule reads a rule's list and shows each device's values.
func (e *Env) TryVarRule(r *listimport.VarRule) (*VarTry, error) {
	rule := *r
	out := &VarTry{}
	if out.Faults = rule.Validate(); len(out.Faults) > 0 {
		return out, nil
	}
	bk, _, err := e.fillBook()
	if err != nil {
		return nil, err
	}
	var known []string
	if bk != nil {
		known = bk.hosts
	}
	_, file, res, err := e.fillList(&rule, known)
	if err != nil {
		return nil, err
	}
	out.File, out.Lines, out.Excluded, out.Devices = file, res.Lines, res.Excluded, len(res.Devices)
	out.Missing, out.Warnings, out.Sample = res.Missing, res.Warnings, res.Sample
	out.NoBook = bk == nil
	if bk != nil {
		for _, d := range res.Devices {
			if bk.Has(listimport.DeviceKey(d)) {
				out.Known++
			} else if len(out.Unknown) < 20 {
				out.Unknown = append(out.Unknown, d)
			}
		}
	}
	for i := range out.Sample {
		for _, v := range rule.Vars {
			if secretName.MatchString(v.Name) {
				for _, m := range []map[string]string{out.Sample[i].Values, out.Sample[i].Raw} {
					if m[v.Name] != "" {
						m[v.Name] = masked
					}
				}
			}
		}
	}
	out.Header = []string{"機器", "一覧の行"}
	for _, v := range rule.Vars {
		out.Header = append(out.Header, v.Name)
	}
	for _, d := range res.Devices {
		for _, row := range res.Rows[listimport.DeviceKey(d)] {
			if out.Total++; out.Total > tryRows {
				continue
			}
			cells := []string{d, fmt.Sprintf("%d", row.Line)}
			for _, v := range rule.Vars {
				val := row.Values[v.Name]
				switch {
				case row.Empty[v.Name]:
					val = "（空: この機器は作りません）"
				case row.Lines[v.Name]:
					val = "（空: 行を出しません）"
				case secretName.MatchString(v.Name) && val != "":
					val = masked
				}
				cells = append(cells, val)
			}
			out.Rows = append(out.Rows, cells)
		}
	}
	return out, nil
}

// FillDraftOutcome is a first template made from one device.
type FillDraftOutcome struct {
	File   string // the configuration it was made from
	Device string
	fill.Draft
}

// findConfig finds a configuration file: a path of the case, or a name looked
// for in input/old, new and logs in turn.
func (e *Env) findConfig(name string) (string, error) {
	name = strings.TrimSpace(strings.ReplaceAll(name, `\`, "/"))
	for _, side := range FillSides {
		ins, err := e.Proj.Inputs(side)
		if err != nil {
			return "", err
		}
		for _, p := range ins {
			if p == name || strings.EqualFold(path.Base(p), name) {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("「%s」というファイルが input/old・new・logs にありません", name)
}

// FillDraft makes a first template from a device's configuration and its
// values (spec 9.7.5). Nothing is written.
func (e *Env) FillDraft(file, device string) (*FillDraftOutcome, error) {
	p, err := e.findConfig(file)
	if err != nil {
		return nil, err
	}
	text, err := project.ReadText(e.Proj.Path(p))
	if err != nil {
		return nil, err
	}
	l := e.Packset().listing(p, text, "")
	var lines []string
	for i, s := range l.Source.Lines {
		if l.Source.Outside == nil || !l.Source.Outside[i] {
			lines = append(lines, s)
		}
	}
	text = strings.Join(lines, "\n") + "\n"
	src, failed, _, err := e.fillSources()
	if err != nil {
		return nil, err
	}
	key := listimport.DeviceKey(device)
	var values []fill.DraftValue
	var eaches []fill.DraftEach
	found := false
	for _, li := range src.Lists {
		if failed[li.Name] != nil {
			continue
		}
		rows := li.Rows[key]
		switch {
		case len(rows) == 1:
			found = true
			for _, v := range li.Vars {
				values = append(values, fill.DraftValue{Name: v, Value: rows[0].Values[v]})
			}
		case len(rows) > 1:
			found = true
			e := fill.DraftEach{Source: li.Name, Names: li.Vars}
			for _, r := range rows {
				e.Rows = append(e.Rows, r.Values)
			}
			eaches = append(eaches, e)
		}
	}
	if src.Book != nil && src.Book.Has(key) {
		found = true
		if rows := src.Book.Rows(key, "devices"); len(rows) > 0 {
			for _, c := range e.Schema.Table("devices").Columns {
				if v := rows[0][c.Name]; v != "" && !c.Secret {
					values = append(values, fill.DraftValue{Name: "devices." + c.Name, Value: v})
				}
			}
		}
	}
	if !found {
		return nil, fmt.Errorf("機器「%s」の値が、変数ルールの一覧にもパラメータブックにもありません", device)
	}
	d := fill.MakeDraft(text, values, eaches)
	return &FillDraftOutcome{File: p, Device: device, Draft: *d}, nil
}
