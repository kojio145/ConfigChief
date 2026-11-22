package book

import (
	"strings"

	"github.com/kojio145/configchief/internal/schema"
)

// SecretMask is what a secret value of a book written out without its
// secrets reads (spec 14.9.1): four of one character, so the check V-218
// (a placeholder left in a secret) finds it in the case that reads it.
const SecretMask = "****"

// MaskSecrets gives the book with every value of a secret column put as
// SecretMask: the secret columns of the tables, and the values of the sheet
// 共通値 whose column is secret. It also gives the values it took out
// (without repeats), for the texts of the case to be masked the same way.
// The rest of the book (formats, free columns, other sheets) stays as it is.
func MaskSecrets(data []byte, file string, sc *schema.Set, lang string) ([]byte, []string, error) {
	sheets, err := ReadSheets(data, file, sc, lang)
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]bool{}
	var taken []string
	d := &Draft{Version: 1}
	for _, es := range sheets {
		if es.Missing {
			continue
		}
		part := DraftSheet{Sheet: es.ID}
		for _, r := range es.Rows {
			var cols []int
			if es.ID == "@common" {
				if ci := es.index("value"); ci >= 0 && commonSecret(sc, maskCell(r.Cells, es.index("table")), maskCell(r.Cells, es.index("column"))) {
					cols = []int{ci}
				}
			} else {
				for ci, c := range es.Columns {
					if c.Secret {
						cols = append(cols, ci)
					}
				}
			}
			ch := DraftRow{Row: r.Row, Key: es.keyOf(r.Cells), Old: map[string]string{}, New: map[string]string{}}
			for _, ci := range cols {
				v := maskCell(r.Cells, ci)
				if strings.TrimSpace(v) == "" || v == SecretMask {
					continue
				}
				ch.Old[es.Columns[ci].Key] = v
				ch.New[es.Columns[ci].Key] = SecretMask
				for _, part := range strings.Split(v, ",") {
					if p := strings.TrimSpace(part); p != "" && !seen[p] {
						seen[p] = true
						taken = append(taken, p)
					}
				}
				if !seen[v] {
					seen[v] = true
					taken = append(taken, v)
				}
			}
			if len(ch.New) > 0 {
				part.Changed = append(part.Changed, ch)
			}
		}
		if len(part.Changed) > 0 {
			d.Sheets = append(d.Sheets, part)
		}
	}
	if len(d.Sheets) == 0 {
		return data, taken, nil
	}
	out, _, _, err := Apply(data, file, sc, lang, d, nil)
	if err != nil {
		return nil, nil, err
	}
	return out, taken, nil
}

func maskCell(cells []string, i int) string {
	if i < 0 || i >= len(cells) {
		return ""
	}
	return cells[i]
}

// commonSecret tells whether the column a row of 共通値 names is secret (the
// table and the column by their internal names or headings, as the book
// reads them).
func commonSecret(sc *schema.Set, table, column string) bool {
	for _, t := range sc.Tables {
		if foldName(table) != foldName(t.Name) && foldName(table) != foldName(t.Title.JA) && foldName(table) != foldName(t.Title.EN) {
			continue
		}
		for _, c := range t.Columns {
			if foldName(column) == foldName(c.Name) || foldName(column) == foldName(c.Title.JA) || foldName(column) == foldName(c.Title.EN) {
				return c.Secret
			}
		}
	}
	return false
}
