package check

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/tmpl"
)

// PackModels is what a vendor pack knows of its models (appendix B 6.1),
// for the checks of the model (appendix C 1.1: V-111, V-114, V-116, G-607).
type PackModels struct {
	// IDs are the IDs of the models; empty: the pack has no models.yaml.
	IDs []string
	// Fits tells whether a device of the model may have an interface of
	// that name: a port of the model, or a logical interface the pack knows.
	Fits func(model, ifname string) bool
	// Unsupported gives what the model does not have.
	Unsupported func(model string) []Unsupported
}

// Unsupported is a function a model does not have: the rows of a table
// for which When is true (all of them if When is empty) use it.
type Unsupported struct {
	Name, Table, When string
}

// The reasons the checks of the model give for a device they did not look
// at (appendix C 1.1).
const (
	whyNoModels   = "メーカーパックの機種の定義を読んでいません"
	whyNoPack     = "メーカーパックが無いため（V-109）"
	whyNoVendor   = "機器の「メーカー」が空のため"
	whyModelUnset = "機種が空か、メーカーパックの機種の定義に無いため（V-114・V-116 を見てください）"
)

// MaxModelTypo is the most single letters a model written may be away from
// an ID of the pack to count as a mistyped ID (V-116).
const MaxModelTypo = 2

// modelOf gives the device's model as the pack writes its ID ("" if the
// pack does not know it), the pack's models, and why the checks of the
// model do not look at the device ("" if they do, or if only the model is
// not known).
func (in Input) modelOf(dev *model.Row) (id string, pm *PackModels, why string) {
	vendor := dev.Get("vendor")
	switch {
	case in.Models == nil:
		return "", nil, whyNoModels
	case vendor == "":
		return "", nil, whyNoVendor
	case in.HasPack != nil && !in.HasPack(vendor):
		return "", nil, whyNoPack
	}
	pm = in.Models(vendor)
	if pm == nil {
		return "", nil, whyNoPack
	}
	written := strings.TrimSpace(dev.Get("model"))
	for _, m := range pm.IDs {
		if written != "" && strings.EqualFold(m, written) {
			return m, pm, ""
		}
	}
	return "", pm, ""
}

// models runs V-111, V-114 and V-116 (appendix C 1.1).
func (x *values) models() {
	if x.in.Models == nil {
		for _, id := range []string{"V-111", "V-114", "V-116"} {
			x.tally.Skip(id, "", whyNoModels)
		}
		return
	}
	t := x.in.Schema.Table("devices")
	tab := x.in.Data.Tables["devices"]
	ifT := x.in.Schema.Table("interfaces")
	if t == nil || tab == nil {
		return
	}
	for _, id := range []string{"V-111", "V-114", "V-116"} {
		x.tally.Look(id, "", 0)
	}
	for _, r := range tab.Rows {
		h := r.Get("hostname")
		if h == "" {
			continue
		}
		id, pm, why := x.in.modelOf(r)
		if why != "" {
			for _, rule := range []string{"V-111", "V-114", "V-116"} {
				x.tally.Skip(rule, h, why)
			}
			continue
		}
		x.tally.Look("V-114", h, 1)
		x.tally.Look("V-116", h, 1)
		if id == "" {
			x.tally.Skip("V-111", h, whyModelUnset)
			x.unknownModel(r, pm)
			continue
		}
		if ifT == nil || pm.Fits == nil {
			continue
		}
		for _, i := range x.of("interfaces", h) {
			name := i.Get("name")
			if name == "" {
				continue
			}
			x.tally.Look("V-111", h, 1)
			if !pm.Fits(id, name) {
				x.add("V-111", ifT, i, "name", fmt.Sprintf("IF「%s」は、機種 %s のポートにも、メーカーパックが知っている論理 IF の名前の形にも当たりません", name, id),
					model.KV{Name: "値", Value: name}, model.KV{Name: "機種", Value: id})
			}
		}
	}
}

// unknownModel tells about a device whose model the pack does not know:
// a mistyped ID (V-116) or a model without a definition (V-114).
func (x *values) unknownModel(r *model.Row, pm *PackModels) {
	devT := x.in.Schema.Table("devices")
	written := strings.TrimSpace(r.Get("model"))
	vendor := r.Get("vendor")
	if written != "" {
		best, dist := "", MaxModelTypo+1
		for _, id := range pm.IDs {
			if d := editDistance(squash(written), squash(id)); d < dist {
				best, dist = id, d
			}
		}
		if best != "" {
			x.add("V-116", devT, r, "model", fmt.Sprintf("機種「%s」は、メーカーパック %s の機種の定義にありません。「%s」の打ち間違いではありませんか", written, vendor, best),
				model.KV{Name: "値", Value: written}, model.KV{Name: "似ている機種", Value: best})
			return
		}
	}
	var msg string
	switch {
	case written == "" && len(pm.IDs) == 0:
		msg = fmt.Sprintf("機種が空です（メーカーパック %s には機種の定義がありません）。IF 名の確かめ（V-111）と、機種の機能の確かめ（G-607）は行いません", vendor)
	case written == "":
		msg = fmt.Sprintf("機種が空です。IF 名の確かめ（V-111）と、機種の機能の確かめ（G-607）は行いません。メーカーパック %s の機種: %s", vendor, strings.Join(pm.IDs, "・"))
	case len(pm.IDs) == 0:
		msg = fmt.Sprintf("メーカーパック %s には機種の定義がありません。IF 名の確かめ（V-111）と、機種の機能の確かめ（G-607）は行いません", vendor)
	default:
		msg = fmt.Sprintf("機種「%s」は、メーカーパック %s の機種の定義にありません。IF 名の確かめ（V-111）と、機種の機能の確かめ（G-607）は行いません。定義のある機種: %s", written, vendor, strings.Join(pm.IDs, "・"))
	}
	ev := []model.KV{}
	if written != "" {
		ev = append(ev, model.KV{Name: "値", Value: written})
	}
	x.add("V-114", devT, r, "model", msg, ev...)
}

// squash keeps the letters and digits of a name, in small letters.
func squash(s string) []rune {
	var out []rune
	for _, c := range strings.ToLower(s) {
		if unicode.IsLetter(c) || unicode.IsDigit(c) {
			out = append(out, c)
		}
	}
	return out
}

// editDistance counts the single letters to add, take away or change to
// make a into b.
func editDistance(a, b []rune) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// ModelFunctions runs G-607 for one device that was generated (appendix C
// 1.1): the rows that use a function its model does not have.
func ModelFunctions(in Input, host string, tally *Tally) []Finding {
	var dev *model.Row
	if tab := in.Data.Tables["devices"]; tab != nil {
		for _, r := range tab.Rows {
			if r.Get("hostname") == host {
				dev = r
				break
			}
		}
	}
	if dev == nil {
		return nil
	}
	id, pm, why := in.modelOf(dev)
	switch {
	case why != "":
		tally.Skip("G-607", host, why)
		return nil
	case id == "":
		tally.Skip("G-607", host, whyModelUnset)
		return nil
	}
	var list []Unsupported
	if pm.Unsupported != nil {
		list = pm.Unsupported(id)
	}
	if len(list) == 0 {
		tally.Skip("G-607", host, "機種の定義（"+id+"）に、対応していない機能を書いていません")
		return nil
	}
	tally.Look("G-607", host, 1)
	eng := tmpl.New(tmpl.Options{})
	var out []Finding
	for _, u := range list {
		t := in.Schema.Table(u.Table)
		tab := in.Data.Tables[u.Table]
		if t == nil || tab == nil {
			continue
		}
		var hits []*model.Row
		var bad error
		for _, r := range tab.Rows {
			if device(t, r) != host {
				continue
			}
			if u.When != "" {
				v, err := eng.Eval(u.When, map[string]any{"row": rowRec{t, r}})
				if err != nil {
					bad = err
					break
				}
				if !tmpl.Truth(v) {
					continue
				}
			}
			hits = append(hits, r)
		}
		if bad != nil {
			out = append(out, New("G-607", Error, host, model.Ref{}, fmt.Sprintf("機種 %s の「%s」の式を調べられません: %v", id, u.Name, bad)))
			continue
		}
		if len(hits) == 0 {
			continue
		}
		first := hits[0]
		f := New("G-607", Error, host, RefOf(t, first, ""),
			fmt.Sprintf("機種 %s には「%s」がありません。表「%s」の %d 行が使っています", id, u.Name, t.Title.JA, len(hits)),
			model.KV{Name: "機種", Value: id}, model.KV{Name: "機能", Value: u.Name}, model.KV{Name: "行の数", Value: fmt.Sprint(len(hits))})
		f.At = CellAt(first, "")
		out = append(out, f)
	}
	return out
}
