package render

import (
	"fmt"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/compare"
	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/model"
)

// RoundTrip reads a generated configuration back with the same pack and
// compares what it says with the design data it was made from (spec 9.6,
// appendix B 5.6). A difference is a fault of the pack, or a value the pack
// cannot write: G-604. A line the pack wrote and cannot read is G-605.
func (r *Renderer) RoundTrip(out *Output) []check.Finding {
	device := out.Device
	if out.Failed {
		return nil // nothing to learn from a text that is not whole (spec 9.6)
	}
	res, err := importer.Import(r.p, r.sc, device+".txt", out.Text)
	if err != nil {
		return []check.Finding{check.New("G-604", check.Error, device, model.Ref{}, fmt.Sprintf("生成した文面を取り込み直せません: %v", err))}
	}
	if res.Device != device {
		return []check.Finding{check.New("G-604", check.Error, device, model.Ref{},
			"生成した文面から読み取ったホスト名が違います", model.KV{Name: "読み取った名前", Value: res.Device})}
	}
	var findings []check.Finding
	extraLine := func(n int) bool { return n >= 1 && n <= len(out.Lines) && out.Lines[n-1].Extra }
	// fromExtra: every line the value came from was written by a person.
	fromExtra := func(os []model.Origin) bool {
		seen := false
		for _, o := range os {
			if o.Kind != "config" && o.Line == 0 {
				continue
			}
			if !extraLine(o.Line) {
				return false
			}
			seen = true
		}
		return seen
	}
	for _, l := range res.Lines {
		if l.State == importer.Unknown && !extraLine(l.Line) {
			f := check.New("G-605", check.Warning, device, model.Ref{},
				fmt.Sprintf("パック %s が書いた行を、取り込み直しで読めません", r.p.ID), model.KV{Name: "行", Value: out.Shown(l.Line)})
			f.Line = l.Line
			findings = append(findings, f)
		}
	}

	design := model.NewDataset("book")
	for _, name := range out.Data.TableNames() {
		for _, row := range out.Data.Tables[name].Rows {
			design.Adopt(cloneRow(row))
		}
	}
	importer.FillDefaults(r.p, r.sc, design, device, false)
	diff := compare.Datasets(r.sc, design, res.Data)

	for _, d := range diff.OnlyA {
		t := r.sc.Table(d.Table)
		// A row no value of which was written is told by G-602 already. A
		// default put in above is no value of the design data.
		// A row that holds values the pack could not write (G-602 names
		// them) and, besides them, nothing but its key: the row is not in the
		// text because of those values, which is told already.
		written := false
		lost := len(out.unused[rowKey(d.Row)]) > 0
		for _, c := range t.Columns {
			cell := d.Row.Cells[c.Name]
			if c.Generate && c.Name != t.DeviceColumn() && cell != nil && len(cell.Values) > 0 && !isDefault(cell) && !out.unused[rowKey(d.Row)][c.Name] && !(lost && t.IsKey(c.Name)) {
				written = true
			}
		}
		if !written {
			continue
		}
		findings = append(findings, check.New("G-604", check.Error, device, check.RefOf(t, d.Row, ""),
			"設計データの行が、生成した文面から読み取れません"))
	}
	for _, d := range diff.OnlyB {
		if fromExtra(d.Row.Origins) {
			continue
		}
		f := check.New("G-604", check.Error, device, check.RefOf(r.sc.Table(d.Table), d.Row, ""),
			"設計データにない行が、生成した文面から読み取られました")
		if len(d.Row.Origins) > 0 {
			f.Line = d.Row.Origins[0].Line
		}
		findings = append(findings, f)
	}
	for _, d := range diff.Cells {
		t := r.sc.Table(d.Table)
		c := t.Column(d.Column)
		if c == nil || !c.Generate || out.unused[rowKey(d.ARow)][d.Column] {
			continue
		}
		if d.B != nil && fromExtra(d.B.Origins) {
			continue
		}
		f := check.New("G-604", check.Error, device, check.RefOf(t, d.ARow, d.Column),
			fmt.Sprintf("「%s」が、生成した文面を取り込み直した値と違います", c.Title.JA),
			model.KV{Name: "設計データ", Value: check.Shown(c, d.A)}, model.KV{Name: "取り込み直し", Value: check.Shown(c, d.B)})
		if d.B != nil && len(d.B.Origins) > 0 {
			f.Line = d.B.Origins[0].Line
		}
		findings = append(findings, f)
	}
	return findings
}

// isDefault tells that every value of a cell is a default of the pack.
func isDefault(c *model.Cell) bool {
	if len(c.Origins) == 0 {
		return false
	}
	for _, o := range c.Origins {
		if o.Kind != "default" {
			return false
		}
	}
	return true
}
