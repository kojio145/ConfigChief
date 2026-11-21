package book

import (
	"regexp"
	"strings"
)

// Searching and replacing in every sheet of the book (spec 14.6). The table
// of a screen searches the sheet it shows by itself; these work on the
// sheets it does not show, with the draft put on.

// FindOptions tells how a search matches.
type FindOptions struct {
	Case    bool `json:"case"`    // upper and lower case differ
	Whole   bool `json:"whole"`   // the whole cell is the text
	Secrets bool `json:"secrets"` // the secret columns are searched too (the screen shows them)
	// Hidden: sheet → the columns (by name) the screen does not show. They
	// are not searched.
	Hidden map[string][]string `json:"hidden"`
}

// FindCount is what a search found in one sheet.
type FindCount struct {
	Sheet   string `json:"sheet"`   // the internal name
	Name    string `json:"name"`    // the name of the sheet
	Count   int    `json:"count"`   // cells that match
	Skipped int    `json:"skipped"` // of a replacement: cells that match and were left alone
}

type finder struct {
	query string
	opt   FindOptions
	re    *regexp.Regexp
}

func newFinder(query string, opt FindOptions) *finder {
	fd := &finder{query: query, opt: opt}
	if !opt.Case {
		fd.query = strings.ToLower(query)
		fd.re = regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))
	}
	return fd
}

func (fd *finder) hit(v string) bool {
	if !fd.opt.Case {
		v = strings.ToLower(v)
	}
	if fd.opt.Whole {
		return v == fd.query
	}
	return strings.Contains(v, fd.query)
}

func (fd *finder) replace(v, with string) string {
	switch {
	case fd.opt.Whole:
		return with
	case fd.opt.Case:
		return strings.ReplaceAll(v, fd.query, with)
	}
	return fd.re.ReplaceAllLiteralString(v, with)
}

// searched tells which columns of a sheet a search looks into.
func (fd *finder) searched(es *EditSheet) []bool {
	hidden := map[string]bool{}
	for _, k := range fd.opt.Hidden[es.ID] {
		hidden[k] = true
	}
	out := make([]bool, len(es.Columns))
	for i, c := range es.Columns {
		out[i] = !c.Dropped && !hidden[c.Key] && (!c.Secret || fd.opt.Secrets)
	}
	return out
}

// Find counts the cells that hold the text, sheet by sheet, with the draft
// put on. Only the sheets with such a cell are given.
func Find(sheets []*EditSheet, d *Draft, query string, opt FindOptions) []FindCount {
	out := []FindCount{}
	if query == "" {
		return out
	}
	fd := newFinder(query, opt)
	for _, es := range sheets {
		v := es.View(d.Of(es.ID))
		in := fd.searched(v.EditSheet)
		n := 0
		for _, r := range v.Rows {
			for ci, val := range r.Cells {
				if in[ci] && fd.hit(val) {
					n++
				}
			}
		}
		if n > 0 {
			out = append(out, FindCount{Sheet: es.ID, Name: es.Name, Count: n})
		}
	}
	return out
}

// Replace puts another text in place of the text in every sheet but the one
// named skip (the one a screen shows: it replaces there by itself). It gives
// the part of the draft of each sheet it changed, and what it did sheet by
// sheet. A row to delete, a cell with a formula and a cell of a column with
// choices that would get a value that is none of them are left alone.
func Replace(sheets []*EditSheet, d *Draft, query, with string, opt FindOptions, skip string) ([]DraftSheet, []FindCount) {
	var parts []DraftSheet
	counts := []FindCount{}
	if query == "" {
		return parts, counts
	}
	fd := newFinder(query, opt)
	for _, es := range sheets {
		if es.ID == skip {
			continue
		}
		v := es.View(d.Of(es.ID))
		in := fd.searched(v.EditSheet)
		c := FindCount{Sheet: es.ID, Name: es.Name}
		for i := range v.Rows {
			r := &v.Rows[i]
			formula := map[int]bool{}
			for _, ci := range r.Formula {
				formula[ci] = true
			}
			copied := false
			for ci, val := range r.Cells {
				if !in[ci] || !fd.hit(val) {
					continue
				}
				next := fd.replace(val, with)
				if next == val {
					continue
				}
				if r.Deleted || formula[ci] || !allowed(&v.Columns[ci], next) {
					c.Skipped++
					continue
				}
				if !copied {
					// The cells of a row the draft does not touch are the ones of the sheet.
					r.Cells = append([]string{}, r.Cells...)
					copied = true
				}
				r.Cells[ci] = next
				c.Count++
			}
		}
		if c.Count > 0 {
			parts = append(parts, v.Part())
		}
		if c.Count+c.Skipped > 0 {
			counts = append(counts, c)
		}
	}
	return parts, counts
}

// allowed tells whether a replacement may put the value into the column: a
// column with choices takes one of them, or nothing.
func allowed(c *EditColumn, v string) bool {
	if len(c.Choices) == 0 || v == "" {
		return true
	}
	for _, o := range c.Choices {
		if o == v {
			return true
		}
	}
	return false
}
