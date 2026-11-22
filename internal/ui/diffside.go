package ui

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strconv"
	"unicode/utf16"
)

// The comparison of two texts side by side (spec 14.8.9; the model is
// PalaTerm's diff window): A on the left, B on the right, the lines removed
// and added put at the same height, the pairs marked where their letters
// differ. The page and the HTML report draw the same rows.

// DiffRow is a row of the comparison side by side.
type DiffRow struct {
	Op string `json:"op"` // "same" | "change" | "del" | "add" | "gap"
	A  int    `json:"a"`  // the line number in A (0 = none on this row)
	B  int    `json:"b"`
	TA string `json:"ta"` // for a gap: how many lines are the same
	TB string `json:"tb"`
	// SA, SB: the letters that differ in a pair, as [from, to) in UTF-16
	// units.
	SA [][2]int `json:"sa"`
	SB [][2]int `json:"sb"`
}

// side lays the lines of a comparison out side by side: a run of removed
// lines and the run of added lines after it are paired line by line, the
// rest of the longer run stands alone.
func side(lines []DiffLine) []DiffRow {
	rows := []DiffRow{}
	for i := 0; i < len(lines); {
		l := lines[i]
		switch l.Op {
		case "same":
			rows = append(rows, DiffRow{Op: "same", A: l.A, B: l.B, TA: l.Text, TB: l.Text})
			i++
		case "gap":
			rows = append(rows, DiffRow{Op: "gap", TA: l.Text})
			i++
		default:
			var dels, adds []DiffLine
			for i < len(lines) && lines[i].Op == "del" {
				dels = append(dels, lines[i])
				i++
			}
			for i < len(lines) && lines[i].Op == "add" {
				adds = append(adds, lines[i])
				i++
			}
			for k := 0; k < max(len(dels), len(adds)); k++ {
				switch {
				case k < len(dels) && k < len(adds):
					sa, sb := changed(dels[k].Text, adds[k].Text)
					rows = append(rows, DiffRow{Op: "change", A: dels[k].A, B: adds[k].B, TA: dels[k].Text, TB: adds[k].Text, SA: sa, SB: sb})
				case k < len(dels):
					rows = append(rows, DiffRow{Op: "del", A: dels[k].A, TA: dels[k].Text})
				default:
					rows = append(rows, DiffRow{Op: "add", B: adds[k].B, TB: adds[k].Text})
				}
			}
		}
	}
	return rows
}

// changed marks what differs between a and b: what is left when the letters
// they start and end with in common are taken off.
func changed(a, b string) (sa, sb [][2]int) {
	x, y := utf16.Encode([]rune(a)), utf16.Encode([]rune(b))
	p := 0
	for p < len(x) && p < len(y) && x[p] == y[p] {
		p++
	}
	s := 0
	for s < len(x)-p && s < len(y)-p && x[len(x)-1-s] == y[len(y)-1-s] {
		s++
	}
	if p < len(x)-s {
		sa = [][2]int{{p, len(x) - s}}
	}
	if p < len(y)-s {
		sb = [][2]int{{p, len(y) - s}}
	}
	return sa, sb
}

// cut is a text split into the parts of a row: marked or not.
type cut struct {
	Text string
	Mark bool
}

func cuts(text string, spans [][2]int) []cut {
	u := utf16.Encode([]rune(text))
	var out []cut
	at := 0
	for _, sp := range spans {
		if sp[0] > at {
			out = append(out, cut{string(utf16.Decode(u[at:sp[0]])), false})
		}
		out = append(out, cut{string(utf16.Decode(u[sp[0]:sp[1]])), true})
		at = sp[1]
	}
	if at < len(u) {
		out = append(out, cut{string(utf16.Decode(u[at:])), false})
	}
	return out
}

var diffPage = template.Must(template.New("diff").Funcs(template.FuncMap{"cuts": cuts, "num": func(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}}).Parse(`<!doctype html>
<html lang="ja"><head><meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'">
<title>文面の比較 {{.NameA}} ↔ {{.NameB}}</title>
<style>
body{font-family:"Yu Gothic UI","Meiryo",sans-serif;margin:16px;color:#1f2328;background:#fff}
h1{font-size:18px;margin:0 0 4px}p{margin:2px 0;color:#59636e;font-size:13px}
table{border-collapse:collapse;width:100%;table-layout:fixed;margin-top:12px;font-family:Consolas,"BIZ UDGothic",monospace;font-size:12px}
td{padding:1px 6px;vertical-align:top;white-space:pre-wrap;word-break:break-all}
td.n{width:48px;color:#8c959f;text-align:right;user-select:none}
tr.del td.a,tr.change td.a{background:#ffebe9}tr.add td.b,tr.change td.b{background:#dafbe1}
tr.del td.b,tr.add td.a{background:#f6f8fa}tr.gap td{background:#ddf4ff;color:#59636e;text-align:center}
td.a mark{background:#ffb8b0}td.b mark{background:#9be9a8}mark{color:inherit}
th{text-align:left;font-size:12px;padding:4px 6px;border-bottom:1px solid #d1d9e0;word-break:break-all}
</style></head><body>
<h1>文面の比較</h1>
<p>A: {{.A}}</p><p>B: {{.B}}</p>
<p>A にだけある行 {{.Removed}} 行、B にだけある行 {{.Added}} 行。{{.When}} に ConfigChief {{.Version}} で作りました。中身はファイルのままです（秘密の値も伏せていません）。</p>
<table><thead><tr><th class="n"></th><th>A</th><th class="n"></th><th>B</th></tr></thead><tbody>
{{range .Rows}}{{if eq .Op "gap"}}<tr class="gap"><td colspan="4">… {{.TA}} …</td></tr>
{{else}}<tr class="{{.Op}}"><td class="n">{{num .A}}</td><td class="a">{{if or .A (eq .Op "same")}}{{range cuts .TA .SA}}{{if .Mark}}<mark>{{.Text}}</mark>{{else}}{{.Text}}{{end}}{{end}}{{end}}</td><td class="n">{{num .B}}</td><td class="b">{{if or .B (eq .Op "same")}}{{range cuts .TB .SB}}{{if .Mark}}<mark>{{.Text}}</mark>{{else}}{{.Text}}{{end}}{{end}}{{end}}</td></tr>
{{end}}{{end}}</tbody></table>
</body></html>
`))

// DiffHTML is the comparison side by side as one HTML file: no script, no
// address outside it (spec 14.8.9).
func (s *Session) DiffHTML(v *DiffView) ([]byte, error) {
	var buf bytes.Buffer
	err := diffPage.Execute(&buf, map[string]any{
		"A": v.A, "B": v.B, "NameA": filepath.Base(v.A), "NameB": filepath.Base(v.B),
		"Added": v.Added, "Removed": v.Removed, "Rows": v.Side,
		"When": s.now().Format("2006-01-02 15:04"), "Version": s.version,
	})
	return buf.Bytes(), err
}

