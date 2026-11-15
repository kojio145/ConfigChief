package book

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

// maxCellChars is Excel's limit for the text of a cell, in UTF-16 units.
const maxCellChars = 32767

// fillRows writes the xlsx file of the grids: frame is the file with
// everything but the rows (putFrame), and the rows of each grid are put
// into its sheet as text. A report of some hundred devices has some
// hundred thousand rows; the library keeps a row as a tree of values and
// turns the tree into text at the end, which takes seconds for that many.
//
// A cell holds its text itself (an "inline string", which Excel and every
// reader of the format take). The parts of the file come out in the order
// and with the settings of the frame, so the same grids give the same
// bytes.
func fillRows(w io.Writer, frame []byte, grids []Grid, text int) error {
	zr, err := zip.NewReader(bytes.NewReader(frame), int64(len(frame)))
	if err != nil {
		return err
	}
	// A new file has its sheets in the order they were made: sheet1.xml is
	// the first grid.
	sheets := map[string]Grid{}
	for i, g := range grids {
		if len(g.Header) > 0 && len(g.Rows) > 0 {
			sheets[fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1)] = g
		}
	}
	zw := zip.NewWriter(w)
	done := 0
	for _, part := range zr.File {
		g, ok := sheets[part.Name]
		if !ok {
			if err := zw.Copy(part); err != nil {
				return err
			}
			continue
		}
		rc, err := part.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		const end = "</sheetData>"
		at := bytes.Index(data, []byte(end))
		if at < 0 || bytes.Count(data, []byte(end)) != 1 {
			return fmt.Errorf("シート「%s」に行を書き込めませんでした（行の入る場所が見つかりません）", g.Name)
		}
		out, err := zw.CreateHeader(&zip.FileHeader{Name: part.Name, Method: part.Method, Modified: part.Modified})
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		buf.Write(data[:at])
		for r, row := range g.Rows {
			writeRow(&buf, r+2, row, len(g.Header), text)
			if buf.Len() > 1<<20 {
				if _, err := out.Write(buf.Bytes()); err != nil {
					return err
				}
				buf.Reset()
			}
		}
		buf.Write(data[at:])
		if _, err := out.Write(buf.Bytes()); err != nil {
			return err
		}
		done++
	}
	if done != len(sheets) {
		return fmt.Errorf("行を書き込めなかったシートがあります（%d 枚のうち %d 枚）", len(sheets), done)
	}
	return zw.Close()
}

// writeRow writes row n (from 1) of a sheet. An empty cell is left out. The
// cells under a heading have the text format, like their columns.
func writeRow(buf *bytes.Buffer, n int, row []string, width, text int) {
	num := strconv.Itoa(n)
	buf.WriteString(`<row r="`)
	buf.WriteString(num)
	buf.WriteString(`">`)
	for c, v := range row {
		if v == "" {
			continue
		}
		col, _ := excelize.ColumnNumberToName(c + 1)
		buf.WriteString(`<c r="`)
		buf.WriteString(col)
		buf.WriteString(num)
		if c < width {
			buf.WriteString(`" s="`)
			buf.WriteString(strconv.Itoa(text))
		}
		buf.WriteString(`" t="inlineStr"><is><t`)
		if first, last := v[0], v[len(v)-1]; edge(first) || edge(last) {
			buf.WriteString(` xml:space="preserve"`)
		}
		buf.WriteString(`>`)
		_ = xml.EscapeText(buf, []byte(guard(clip(v))))
		buf.WriteString(`</t></is></c>`)
	}
	buf.WriteString(`</row>`)
}

func edge(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }

// clip cuts a text at Excel's limit for a cell, as the library does.
func clip(s string) string {
	if len(s) <= maxCellChars {
		return s // a byte is at least one unit
	}
	units := 0
	for i, r := range s {
		u := 1
		if r > 0xFFFF {
			u = 2
		}
		if units+u > maxCellChars {
			return s[:i]
		}
		units += u
	}
	return s
}

// guard keeps a text that looks like the format's way to write a character
// by its number ("_x000D_") from being read as that character: the format
// has the reader take "_x005F_" before such a piece as a plain "_".
func guard(s string) string {
	if !strings.Contains(s, "_x") {
		return s
	}
	var sb strings.Builder
	for i := 0; i < len(s); {
		if coded(s[i:]) {
			// The "_" at its end may begin the next such piece.
			sb.WriteString("_x005F")
			sb.WriteString(s[i : i+6])
			i += 6
			continue
		}
		_, n := utf8.DecodeRuneInString(s[i:])
		sb.WriteString(s[i : i+n])
		i += n
	}
	return sb.String()
}

// coded tells whether s begins with "_xHHHH_", H a hexadecimal digit.
func coded(s string) bool {
	if len(s) < 7 || s[0] != '_' || s[1] != 'x' || s[6] != '_' {
		return false
	}
	for _, c := range []byte(s[2:6]) {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}
