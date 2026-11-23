package report

import (
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/kojio145/configchief/internal/book"
)

// Pair is a label with its value.
type Pair struct {
	Label, Value string
}

// The marks of a row: they give the row its colour in the HTML form (spec
// 8.2: grey for an ignored line, red for an unhandled one).
const (
	MarkIgnored = "ignored"
	MarkUnknown = "unknown"
	MarkWarn    = "warn"
)

// Sheet is one list of a report: a sheet of the xlsx form, a section of the
// HTML form.
type Sheet struct {
	Name   string // at most 31 characters, none of : \ / ? * [ ]
	Lead   string // a sentence above the list (HTML only)
	Header []string
	Rows   [][]string
	Marks  []string // one per row ("" = none); may be shorter than Rows
	Code   []int    // the columns that hold configuration text
	// HTMLRows, if not nil, are the rows the HTML form shows (a list too
	// long for a browser); HTMLNote tells the reader so. The xlsx form has
	// every row.
	HTMLRows []int
	HTMLNote string
}

// Doc is a report: a title, the numbers at the top and the lists.
type Doc struct {
	Title string
	// Cover names the first sheet of the xlsx form ("" = まとめ).
	Cover  string
	Facts  []Pair // the case, the time, what was compared ...
	Counts []Pair // the numbers at the top
	Notes  []string
	Sheets []Sheet
}

// The first sheet of the xlsx form.
const summarySheet = "まとめ"

// WriteXLSX writes the report as one xlsx file: a summary sheet, then the
// lists. The same report gives the same bytes.
func (d *Doc) WriteXLSX(w io.Writer) error {
	sum := book.Grid{Name: summarySheet, Header: []string{"項目", "内容"}}
	if d.Cover != "" {
		sum.Name = d.Cover
	}
	sum.Rows = append(sum.Rows, []string{"レポート", d.Title})
	for _, p := range d.Facts {
		sum.Rows = append(sum.Rows, []string{p.Label, p.Value})
	}
	for _, p := range d.Counts {
		sum.Rows = append(sum.Rows, []string{p.Label, p.Value})
	}
	for _, n := range d.Notes {
		sum.Rows = append(sum.Rows, []string{"注意", n})
	}
	grids := []book.Grid{sum}
	for _, s := range d.Sheets {
		grids = append(grids, book.Grid{Name: s.Name, Header: s.Header, Rows: s.Rows})
	}
	return book.WriteGrids(w, grids)
}

const style = `
body { font-family: "Yu Gothic UI", "Meiryo", "Segoe UI", sans-serif; margin: 24px; color: #1f2328; background: #fff; }
h1 { font-size: 22px; margin: 0 0 12px; }
h2 { font-size: 17px; margin: 32px 0 6px; padding-top: 8px; border-top: 2px solid #d0d7de; }
p { margin: 4px 0 8px; }
p.note { color: #7d4e00; }
nav { margin: 12px 0; }
nav a { margin-right: 14px; white-space: nowrap; }
table { border-collapse: collapse; font-size: 13px; }
th, td { border: 1px solid #d0d7de; padding: 3px 8px; text-align: left; vertical-align: top; }
th { background: #d9e1f2; position: sticky; top: 0; }
table.facts th { background: #f6f8fa; position: static; font-weight: normal; }
table.facts td.num { text-align: right; font-weight: bold; }
td.code { font-family: Consolas, "MS Gothic", monospace; white-space: pre-wrap; word-break: break-all; }
tr.ignored td { color: #6e7781; background: #f6f8fa; }
tr.unknown td { background: #ffebe9; }
tr.warn td { background: #fff8c5; }
p.none { color: #6e7781; }
@media print { th { position: static; } nav { display: none; } }
`

// HTML gives the report as one HTML file that needs nothing from outside:
// no script, no picture, no style sheet to load (spec 13.3).
func (d *Doc) HTML() string {
	var sb strings.Builder
	esc := html.EscapeString
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"ja\">\n<head>\n<meta charset=\"utf-8\">\n")
	sb.WriteString("<title>" + esc(d.Title) + "</title>\n<style>" + style + "</style>\n</head>\n<body>\n")
	sb.WriteString("<h1>" + esc(d.Title) + "</h1>\n")
	if len(d.Facts) > 0 {
		sb.WriteString("<table class=\"facts\">\n")
		for _, p := range d.Facts {
			sb.WriteString("<tr><th>" + esc(p.Label) + "</th><td>" + esc(p.Value) + "</td></tr>\n")
		}
		sb.WriteString("</table>\n")
	}
	if len(d.Counts) > 0 {
		sb.WriteString("<h2>件数</h2>\n<table class=\"facts\">\n")
		for _, p := range d.Counts {
			sb.WriteString("<tr><th>" + esc(p.Label) + "</th><td class=\"num\">" + esc(p.Value) + "</td></tr>\n")
		}
		sb.WriteString("</table>\n")
	}
	for _, n := range d.Notes {
		sb.WriteString("<p class=\"note\">" + esc(n) + "</p>\n")
	}
	if len(d.Sheets) > 1 {
		sb.WriteString("<nav>")
		for i, s := range d.Sheets {
			fmt.Fprintf(&sb, "<a href=\"#s%d\">%s（%d）</a>", i+1, esc(s.Name), len(s.Rows))
		}
		sb.WriteString("</nav>\n")
	}
	for i, s := range d.Sheets {
		fmt.Fprintf(&sb, "<h2 id=\"s%d\">%s（%d 件）</h2>\n", i+1, esc(s.Name), len(s.Rows))
		if s.Lead != "" {
			sb.WriteString("<p>" + esc(s.Lead) + "</p>\n")
		}
		if s.HTMLRows != nil && s.HTMLNote != "" {
			sb.WriteString("<p class=\"note\">" + esc(s.HTMLNote) + "</p>\n")
		}
		if len(s.Rows) == 0 {
			sb.WriteString("<p class=\"none\">ありません</p>\n")
			continue
		}
		code := map[int]bool{}
		for _, c := range s.Code {
			code[c] = true
		}
		sb.WriteString("<table>\n<tr>")
		for _, h := range s.Header {
			sb.WriteString("<th>" + esc(h) + "</th>")
		}
		sb.WriteString("</tr>\n")
		order := s.HTMLRows
		if order == nil {
			order = make([]int, len(s.Rows))
			for ri := range order {
				order[ri] = ri
			}
		}
		for _, ri := range order {
			row := s.Rows[ri]
			if ri < len(s.Marks) && s.Marks[ri] != "" {
				sb.WriteString("<tr class=\"" + esc(s.Marks[ri]) + "\">")
			} else {
				sb.WriteString("<tr>")
			}
			for ci := range s.Header {
				v := ""
				if ci < len(row) {
					v = row[ci]
				}
				if code[ci] {
					sb.WriteString("<td class=\"code\">" + esc(v) + "</td>")
				} else {
					sb.WriteString("<td>" + esc(v) + "</td>")
				}
			}
			sb.WriteString("</tr>\n")
		}
		sb.WriteString("</table>\n")
	}
	sb.WriteString("</body>\n</html>\n")
	return sb.String()
}
