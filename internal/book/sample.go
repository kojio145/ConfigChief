package book

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/kojio145/configchief/internal/schema"
)

// SampleKinds are the ways a sample template groups the devices (spec 13.2,
// 14.11.2): one book with a sheet for each device, a file for each device,
// one list of every device.
var SampleKinds = []string{"sheets", "files", "list"}

// SampleName is the file name of the sample template of a kind.
func SampleName(kind string) string {
	switch kind {
	case "sheets":
		return "sample_per_device.xlsx"
	case "files":
		return "{{ device.hostname }}_sample.xlsx"
	case "list":
		return "sample_all_devices.xlsx"
	}
	return ""
}

// samplePart is a block of a sample: a table and the columns of it.
type samplePart struct {
	table   string
	columns []string
}

// The tables and columns a sample shows: the basics of a device, its
// interfaces and its static routes.
var (
	sampleDevice = []string{"hostname", "vendor", "model"}
	sampleParts  = []samplePart{
		{"interfaces", []string{"name", "description", "ipv4", "enabled"}},
		{"static_routes", []string{"prefix", "next_hop", "out_if", "description"}},
	}
)

// Sample makes a sample template of the delivery sheets (spec 14.11.2): the
// headings, borders and colours are there, and the expressions name the
// tables and columns of the definitions, so that the sample can be poured
// as it is.
func Sample(kind string, sc *schema.Set, lang string) ([]byte, error) {
	if SampleName(kind) == "" {
		return nil, fmt.Errorf("見本の形「%s」はありません（sheets・files・list）", kind)
	}
	f := excelize.NewFile()
	defer f.Close()
	st, err := sampleStyles(f)
	if err != nil {
		return nil, err
	}
	title := func(table, col string) (string, error) {
		t := sc.Table(table)
		if t == nil {
			return "", fmt.Errorf("表 %s がありません", table)
		}
		c := t.Column(col)
		if c == nil {
			return "", fmt.Errorf("表 %s に列 %s がありません", table, col)
		}
		return heading(c, lang), nil
	}
	tableTitle := func(table string) string {
		if t := sc.Table(table); t != nil {
			return t.Title.In(lang)
		}
		return table
	}
	w := &sampleWriter{f: f, st: st}
	if kind == "list" {
		const devs = "機器"
		if err := f.SetSheetName("Sheet1", devs); err != nil {
			return nil, err
		}
		w.sheet = devs
		w.put(1, 1, "{{ project.name }} 機器の一覧", st.title)
		cells := []string{}
		for _, c := range sampleDevice {
			h, err := title("devices", c)
			if err != nil {
				return nil, err
			}
			cells = append(cells, h)
		}
		w.row(3, cells, st.head)
		w.row(4, rowsCells("devices", sampleDevice), st.cell)
		w.widths(len(sampleDevice))
		for _, p := range sampleParts {
			name := tableTitle(p.table)
			if _, err := f.NewSheet(name); err != nil {
				return nil, err
			}
			w.sheet = name
			w.put(1, 1, "{{ project.name }} "+name, st.title)
			heads, _ := title("devices", "hostname")
			cells := []string{heads}
			for _, c := range p.columns {
				h, err := title(p.table, c)
				if err != nil {
					return nil, err
				}
				cells = append(cells, h)
			}
			w.row(3, cells, st.head)
			w.row(4, rowsCells(p.table, append([]string{"device"}, p.columns...)), st.cell)
			w.widths(len(cells))
		}
	} else {
		name := "{{ device.hostname }}"
		if kind == "files" {
			name = "パラメータ"
		}
		if err := f.SetSheetName("Sheet1", name); err != nil {
			return nil, err
		}
		w.sheet = name
		w.put(1, 1, "{{ project.name }} パラメータシート", st.title)
		r := 3
		for _, c := range sampleDevice {
			h, err := title("devices", c)
			if err != nil {
				return nil, err
			}
			w.put(r, 1, h, st.head)
			w.put(r, 2, "{{ device."+c+" }}", st.cell)
			r++
		}
		for _, p := range sampleParts {
			r++
			w.put(r, 1, tableTitle(p.table), st.title)
			r++
			cells := []string{}
			for _, c := range p.columns {
				h, err := title(p.table, c)
				if err != nil {
					return nil, err
				}
				cells = append(cells, h)
			}
			w.row(r, cells, st.head)
			r++
			w.row(r, rowsCells(p.table, p.columns), st.cell)
			r++
		}
		w.widths(4)
	}
	if w.err != nil {
		return nil, w.err
	}
	f.SetActiveSheet(0)
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// rowsCells are the cells of a row that repeats for the rows of a table.
func rowsCells(table string, cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = "{{ row." + c + " }}"
	}
	out[0] = "{{#rows " + table + "}}" + out[0]
	return out
}

type sampleStyle struct{ title, head, cell int }

func sampleStyles(f *excelize.File) (sampleStyle, error) {
	border := []excelize.Border{
		{Type: "left", Color: "808080", Style: 1}, {Type: "right", Color: "808080", Style: 1},
		{Type: "top", Color: "808080", Style: 1}, {Type: "bottom", Color: "808080", Style: 1},
	}
	var s sampleStyle
	var err error
	if s.title, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12}}); err != nil {
		return s, err
	}
	if s.head, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Border: border,
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"DDEBF7"}}}); err != nil {
		return s, err
	}
	s.cell, err = f.NewStyle(&excelize.Style{Border: border, Alignment: &excelize.Alignment{Vertical: "top", WrapText: true}})
	return s, err
}

// sampleWriter puts cells into one sheet, keeping the first error.
type sampleWriter struct {
	f     *excelize.File
	st    sampleStyle
	sheet string
	err   error
}

func (w *sampleWriter) put(row, col int, v string, style int) {
	if w.err != nil {
		return
	}
	cell, err := excelize.CoordinatesToCellName(col, row)
	if err == nil {
		err = w.f.SetCellStr(w.sheet, cell, v)
	}
	if err == nil {
		err = w.f.SetCellStyle(w.sheet, cell, cell, style)
	}
	w.err = err
}

func (w *sampleWriter) row(row int, cells []string, style int) {
	for i, v := range cells {
		w.put(row, i+1, v, style)
	}
}

func (w *sampleWriter) widths(n int) {
	if w.err != nil {
		return
	}
	last, _ := excelize.ColumnNumberToName(n)
	w.err = w.f.SetColWidth(w.sheet, "A", last, 24)
}
