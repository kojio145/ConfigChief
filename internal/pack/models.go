package pack

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Model is one model of models.yaml (appendix B 6): the ports it has, how
// its logical interfaces are named, and numbers the templates read.
type Model struct {
	ID, Family  string
	Title       map[string]string
	Ports       []Port
	Logical     map[string]string
	Limits      map[string]int
	// OtherNames are more forms of the names of interfaces the model may
	// have (V-111), besides its ports and the forms of Logical.
	OtherNames []string
	// Unsupported are what the model does not have (G-607).
	Unsupported []Unsupported
	// names are the forms of Logical and OtherNames as expressions.
	names []*regexp.Regexp
}

// Unsupported is a function a model does not have: the rows of a table
// (those for which When is true; all if When is empty) use it.
type Unsupported struct {
	Name, Table, When string
}

// Port is a physical port of a model.
type Port struct {
	Name string
	Role string // wan, lan, switch
	// Sub is the name its subinterfaces start with (the name itself if the
	// file does not say).
	Sub string
	// VlanIf is the interface of a VLAN whose lowest access port this is
	// ("" where the vendor names such an interface by the VLAN).
	VlanIf string
}

type portY struct {
	Name   string `yaml:"name"`
	Range  string `yaml:"range"`
	Role   string `yaml:"role"`
	Sub    string `yaml:"sub"`
	VlanIf string `yaml:"vlan_if"`
}

type modelsY struct {
	Families []struct {
		ID    string `yaml:"id"`
		Title string `yaml:"title"`
	} `yaml:"families"`
	Models []struct {
		ID          string            `yaml:"id"`
		Title       map[string]string `yaml:"title"`
		Family      string            `yaml:"family"`
		Ports       []portY           `yaml:"ports"`
		Logical     map[string]string `yaml:"logical"`
		Limits      map[string]int    `yaml:"limits"`
		OtherNames  []string          `yaml:"other_names"`
		Unsupported []struct {
			Name  string `yaml:"name"`
			Table string `yaml:"table"`
			When  string `yaml:"when"`
		} `yaml:"unsupported"`
	} `yaml:"models"`
}

var (
	reRange   = regexp.MustCompile(`^(.*)\[(\d+)-(\d+)\]$`)
	roles     = map[string]bool{"": true, "wan": true, "lan": true, "switch": true}
	logicals  = map[string]bool{"loopback": true, "tunnel": true, "subif": true, "pppoe": true, "vlan": true, "lag": true}
	reLogMark = regexp.MustCompile(`\{[^}]*\}`)
	logMarks  = map[string]bool{"{n}": true, "{parent}": true, "{vlan}": true, "{i}": true, "{j}": true}
)

// models reads models.yaml, if the pack has one.
func (l *loader) models() {
	var y modelsY
	if !l.read("models.yaml", &y, false) {
		return
	}
	families := map[string]bool{}
	for _, f := range y.Families {
		families[f.ID] = true
	}
	seen := map[string]bool{}
	for _, m := range y.Models {
		where := "models.yaml " + m.ID
		id := strings.ToLower(m.ID)
		if id == "" || seen[id] {
			l.errf("models.yaml: 機種の id「%s」が空か、重なっています", m.ID)
			continue
		}
		seen[id] = true
		if m.Family != "" && !families[m.Family] {
			l.errf("%s: family「%s」が families にありません", where, m.Family)
		}
		out := &Model{ID: m.ID, Title: m.Title, Family: m.Family, Logical: m.Logical, Limits: m.Limits, OtherNames: m.OtherNames}
		for _, u := range m.Unsupported {
			if strings.TrimSpace(u.Name) == "" || strings.TrimSpace(u.Table) == "" {
				l.errf("%s: unsupported の行には name と table を書きます", where)
				continue
			}
			out.Unsupported = append(out.Unsupported, Unsupported{Name: u.Name, Table: u.Table, When: u.When})
		}
		names := map[string]bool{}
		for _, p := range m.Ports {
			if !roles[p.Role] {
				l.errf("%s: role「%s」は書けません（wan・lan・switch）", where, p.Role)
			}
			if (p.Name == "") == (p.Range == "") {
				l.errf("%s: ports の行には name か range のどちらか 1 つを書きます", where)
				continue
			}
			if p.Name != "" {
				out.Ports = append(out.Ports, Port{Name: p.Name, Role: p.Role, Sub: p.Sub, VlanIf: strings.ReplaceAll(p.VlanIf, "{p}", "")})
				continue
			}
			g := reRange.FindStringSubmatch(p.Range)
			if g == nil {
				l.errf("%s: range「%s」が読めません（終わりを [1-8] の形にします）", where, p.Range)
				continue
			}
			from, _ := strconv.Atoi(g[2])
			to, _ := strconv.Atoi(g[3])
			if to < from || to-from > 1000 {
				l.errf("%s: range「%s」の番号が逆か、多すぎます", where, p.Range)
				continue
			}
			for n := from; n <= to; n++ {
				num := strconv.Itoa(n)
				out.Ports = append(out.Ports, Port{Name: g[1] + num, Role: p.Role, Sub: strings.ReplaceAll(p.Sub, "{p}", num), VlanIf: strings.ReplaceAll(p.VlanIf, "{p}", num)})
			}
		}
		for i := range out.Ports {
			pt := &out.Ports[i]
			if names[pt.Name] {
				l.errf("%s: ポート「%s」が重なっています", where, pt.Name)
			}
			names[pt.Name] = true
			if pt.Sub == "" {
				pt.Sub = pt.Name
			}
		}
		if len(out.Ports) == 0 {
			l.errf("%s: ports がありません", where)
		}
		for k, v := range m.Logical {
			if !logicals[k] {
				l.errf("%s: logical「%s」は書けません（loopback・tunnel・subif・pppoe・vlan・lag）", where, k)
			}
			for _, mark := range reLogMark.FindAllString(v, -1) {
				if !logMarks[mark] {
					l.errf("%s: logical %s の印「%s」は使えません（{n}・{parent}・{vlan}・{i}・{j}）", where, k, mark)
				}
			}
		}
		for i, v := range m.OtherNames {
			for _, mark := range reLogMark.FindAllString(v, -1) {
				if mark != "{n}" && mark != "{parent}" {
					l.errf("%s: other_names %d 番目の印「%s」は使えません（{n}・{parent}）", where, i+1, mark)
				}
			}
		}
		out.names = nameForms(out)
		l.p.Models = append(l.p.Models, out)
	}
}

// nameForms turns the forms of Logical and OtherNames into expressions:
// {parent} is the name a port's subinterfaces start with, the other marks
// stand for a number. Upper and lower case do not matter.
func nameForms(m *Model) []*regexp.Regexp {
	var parents []string
	seen := map[string]bool{}
	for _, p := range m.Ports {
		for _, n := range []string{p.Name, p.Sub} {
			if n != "" && !seen[n] {
				seen[n] = true
				parents = append(parents, regexp.QuoteMeta(n))
			}
		}
	}
	parent := "(?:" + strings.Join(parents, "|") + ")"
	keys := make([]string, 0, len(m.Logical))
	for k := range m.Logical {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	forms := make([]string, 0, len(keys)+len(m.OtherNames))
	for _, k := range keys {
		forms = append(forms, m.Logical[k])
	}
	forms = append(forms, m.OtherNames...)
	var out []*regexp.Regexp
	for _, f := range forms {
		if f == "" {
			continue
		}
		var b strings.Builder
		rest := f
		for rest != "" {
			loc := reLogMark.FindStringIndex(rest)
			if loc == nil {
				b.WriteString(regexp.QuoteMeta(rest))
				break
			}
			b.WriteString(regexp.QuoteMeta(rest[:loc[0]]))
			if rest[loc[0]:loc[1]] == "{parent}" {
				if len(parents) == 0 {
					b.WriteString(`[^\s\S]`) // no port: nothing fits
				} else {
					b.WriteString(parent)
				}
			} else {
				b.WriteString(`\d+`)
			}
			rest = rest[loc[1]:]
		}
		if re, err := regexp.Compile("(?i)^" + b.String() + "$"); err == nil {
			out = append(out, re)
		}
	}
	return out
}

// FitsModel tells whether a device of the model may have an interface of
// that name (V-111, appendix C 1.1): a port of the model or the interface of
// a VLAN on it, a name of a form of Logical or OtherNames, an interface the
// pack knows as not physical (pack.yaml, interfaces.types), or a
// subinterface of one of these. Upper and lower case do not matter.
func (p *Pack) FitsModel(m *Model, name string) bool {
	return p.fits(m, strings.TrimSpace(name), 0)
}

func (p *Pack) fits(m *Model, name string, depth int) bool {
	if m == nil || name == "" || depth > 3 {
		return false
	}
	for _, pt := range m.Ports {
		if strings.EqualFold(pt.Name, name) || (pt.VlanIf != "" && strings.EqualFold(pt.VlanIf, name)) {
			return true
		}
	}
	for _, re := range m.names {
		if re.MatchString(name) {
			return true
		}
	}
	switch kind := p.IfKind(name); kind {
	case "subif":
		if g := p.Subif.FindStringSubmatch(name); g != nil {
			if i := p.Subif.SubexpIndex("parent"); i > 0 {
				return p.fits(m, g[i], depth+1)
			}
		}
		return false
	case "", "physical":
		return false
	default:
		return true
	}
}

// Model gives the model with that ID (upper and lower case do not matter),
// or nil.
func (p *Pack) Model(id string) *Model {
	id = strings.TrimSpace(id)
	if p == nil || id == "" {
		return nil
	}
	for _, m := range p.Models {
		if strings.EqualFold(m.ID, id) {
			return m
		}
	}
	return nil
}
