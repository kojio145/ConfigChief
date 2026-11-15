package pack

import (
	"io/fs"
	"path"
	"regexp"
	"sort"

	"github.com/kojio145/configchief/internal/tmpl"
)

// Output is how a generated configuration is written to a file.
type Output struct {
	Encoding string // "" = utf-8
	Newline  string // "lf" | "crlf"; "" = lf
	Indent   string
}

// Section is one part of a generated configuration (appendix B 5.1): a
// template, or the place where the extra configurations go.
type Section struct {
	ID       string
	Template string // file name in templates/; "" for the slot
	Slot     string // "extra" or ""
}

// ExtraSlot is the id of the section the extra configurations go to when
// they name no place; "" if the pack has none.
func (p *Pack) ExtraSlot() string {
	for _, s := range p.Sections {
		if s.Slot == "extra" {
			return s.ID
		}
	}
	return ""
}

// Engine makes a template engine that knows the words of the pack.
func (p *Pack) Engine() *tmpl.Engine {
	return tmpl.New(tmpl.Options{IfName: p.IfName, IfShort: p.IfShort, Lex: p.Lex, Unlex: p.Unlex})
}

var reDefaultFilter = regexp.MustCompile(`\|\s*default\b`)

// render reads render.yaml and the templates. A pack without render.yaml
// reads configurations and writes none.
func (l *loader) render() {
	var ry struct {
		Sections []struct {
			ID       string `yaml:"id"`
			Template string `yaml:"template"`
			Slot     string `yaml:"slot"`
		} `yaml:"sections"`
	}
	if !l.read("render.yaml", &ry, false) {
		return
	}
	p := l.p
	p.Templates = map[string]string{}
	files, _ := fs.Glob(l.fsys, "templates/*.j2")
	sort.Strings(files)
	eng := p.Engine()
	for _, f := range files {
		data, err := fs.ReadFile(l.fsys, f)
		if err != nil {
			l.errf("%s: 読めません（%v）", f, err)
			continue
		}
		name := path.Base(f)
		p.Templates[name] = string(data)
		if err := eng.Check(name, string(data)); err != nil {
			l.errf("%s: %v", f, err)
		}
		if reDefaultFilter.Match(data) {
			l.warnf("%s: default フィルタを使っています（空の値を黙って別の値にします。条件で書き分けます）", f)
		}
	}
	if len(ry.Sections) == 0 {
		l.errf("render.yaml: sections がありません")
	}
	seen := map[string]bool{}
	used := map[string]bool{}
	slots := 0
	for i, s := range ry.Sections {
		switch {
		case s.ID == "":
			l.errf("render.yaml %d 件目: id がありません", i+1)
			continue
		case seen[s.ID]:
			l.errf("render.yaml: id「%s」が 2 回あります", s.ID)
			continue
		}
		seen[s.ID] = true
		switch {
		case (s.Template == "") == (s.Slot == ""):
			l.errf("render.yaml %s: template か slot のどちらか 1 つを書きます", s.ID)
			continue
		case s.Slot != "" && s.Slot != "extra":
			l.errf("render.yaml %s: slot は extra だけが使えます", s.ID)
			continue
		case s.Slot != "":
			slots++
		default:
			if _, ok := p.Templates[s.Template]; !ok {
				l.errf("render.yaml %s: templates/%s がありません", s.ID, s.Template)
				continue
			}
			used[s.Template] = true
		}
		p.Sections = append(p.Sections, Section{s.ID, s.Template, s.Slot})
	}
	if slots != 1 {
		l.errf("render.yaml: 追加コンフィグの位置（slot: extra）を 1 つ書きます")
	}
	// A template no section names is a part for {% include %}; one that is
	// named nowhere is most likely a typo.
	var names []string
	for name := range p.Templates {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if used[name] {
			continue
		}
		inc := regexp.MustCompile(`\{%-?\s*include\s+['"]` + regexp.QuoteMeta(name) + `['"]`)
		found := false
		for _, src := range p.Templates {
			if inc.MatchString(src) {
				found = true
				break
			}
		}
		if !found {
			l.warnf("templates/%s は、render.yaml にも include にも出てきません", name)
		}
	}
}
