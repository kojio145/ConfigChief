package pack

import (
	"regexp"
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
	Unsupported []string
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
		Unsupported []string          `yaml:"unsupported"`
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
		out := &Model{ID: m.ID, Title: m.Title, Family: m.Family, Logical: m.Logical, Limits: m.Limits, Unsupported: m.Unsupported}
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
		l.p.Models = append(l.p.Models, out)
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
