package book

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

const editFile = "parameters.xlsx"

// editBook is a book with the sample rows, a user with a password, and what
// a person adds in Excel: a free column, a cell with a colour, a sheet of
// notes.
func editBook(t *testing.T) []byte {
	t.Helper()
	d := sampleData()
	put(d, "users", kv("device", "rt-a01", "name", "sample-admin"), "secret", "sample-pass-1")
	data, _, err := Update(nil, editFile, defs(t), "ja", Change{Data: d})
	if err != nil {
		t.Fatal(err)
	}
	return inExcel(t, data, func(f *excelize.File) {
		must(t, f.SetCellStr("インタフェース", "AZ1", "回線メモ"))
		must(t, f.SetCellStr("インタフェース", "AZ2", "sample note"))
		style, err := f.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFFF00"}}})
		must(t, err)
		must(t, f.SetCellStyle("インタフェース", "B2", "B2", style))
		_, err = f.NewSheet("メモ")
		must(t, err)
		must(t, f.SetCellStr("メモ", "A1", "sample memo"))
	})
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// inExcel changes a book the way a person does in Excel.
func inExcel(t *testing.T, data []byte, change func(f *excelize.File)) []byte {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	must(t, err)
	defer f.Close()
	change(f)
	buf, err := f.WriteToBuffer()
	must(t, err)
	return buf.Bytes()
}

func editSheet(t *testing.T, data []byte, id string) *EditSheet {
	t.Helper()
	sheets, err := ReadSheets(data, editFile, defs(t), "ja")
	must(t, err)
	for _, es := range sheets {
		if es.ID == id {
			return es
		}
	}
	t.Fatalf("no sheet %s", id)
	return nil
}

// cellOf gives the row that has the value in one of its cells, and its cell
// of the column named.
func cellOf(t *testing.T, es *EditSheet, find, column string) (EditRow, string) {
	t.Helper()
	ci := es.index(column)
	for _, r := range es.Rows {
		for _, v := range r.Cells {
			if v == find {
				return r, r.Cells[ci]
			}
		}
	}
	t.Fatalf("no row with %s", find)
	return EditRow{}, ""
}

// all gives every cell of every sheet as text, for telling what a save changed.
func all(t *testing.T, data []byte) map[string]string {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	must(t, err)
	defer f.Close()
	out := map[string]string{}
	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		must(t, err)
		for r, row := range rows {
			for c, v := range row {
				if v != "" {
					out[name+"!"+addr(c, r)] = v
				}
			}
		}
	}
	return out
}

func changed(a, b map[string]string) []string {
	var out []string
	for k, v := range a {
		if b[k] != v {
			out = append(out, k)
		}
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			out = append(out, k)
		}
	}
	return out
}

func draftOf(parts ...DraftSheet) *Draft {
	d := &Draft{}
	for _, p := range parts {
		d.Put(p)
	}
	return d
}

// inspect tells what of a book a save must not touch: the colour a person
// gave a cell, the sheet of notes, the free column, the order of the sheets,
// the comments and the lists to choose from.
func inspect(t *testing.T, data []byte) string {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	must(t, err)
	defer f.Close()
	style, err := f.GetCellStyle("インタフェース", "B2")
	must(t, err)
	st, err := f.GetStyle(style)
	must(t, err)
	memo, _ := f.GetCellValue("メモ", "A1")
	free, _ := f.GetCellValue("インタフェース", "AZ2")
	comments, err := f.GetComments("インタフェース")
	must(t, err)
	dvs, err := f.GetDataValidations("インタフェース")
	must(t, err)
	return fmt.Sprintf("%v | %s | %s | %v | comments %d | lists %d", st.Fill.Color, memo, free, f.GetSheetList(), len(comments), len(dvs))
}

func TestReadSheets(t *testing.T) {
	data := editBook(t)
	sheets, err := ReadSheets(data, editFile, defs(t), "ja")
	must(t, err)
	ids := map[string]bool{}
	for _, es := range sheets {
		ids[es.ID] = true
		if es.Name == "メモ" {
			t.Error("a sheet the tool does not know is in the list")
		}
	}
	for _, id := range []string{"devices", "interfaces", "users", "@devices", "@ifs", "@common"} {
		if !ids[id] {
			t.Errorf("no sheet %s", id)
		}
	}
	es := editSheet(t, data, "interfaces")
	free := es.index(freeCol("回線メモ"))
	if free < 0 || !es.Columns[free].Free || es.Columns[free].Title != "回線メモ" {
		t.Fatalf("the free column: %+v", es.Columns)
	}
	if es.Device < 0 || es.Columns[es.Device].Key != "device" || !es.Columns[es.Device].IsKey {
		t.Errorf("the device column: %d", es.Device)
	}
	if c := es.Columns[es.index("enabled")]; len(c.Choices) != 2 || c.Type != "bool" {
		t.Errorf("enabled: %+v", c)
	}
	if len(es.Rows) != 2 || es.Rows[0].Row != 2 || es.Rows[0].Cells[free] != "sample note" {
		t.Errorf("rows: %+v", es.Rows)
	}
	// The secret column is marked: the screen masks it.
	us := editSheet(t, data, "users")
	if c := us.Columns[us.index("secret")]; !c.Secret {
		t.Errorf("users.secret is not secret: %+v", c)
	}
	// What is wrong with a value is what reading the book says.
	if msg := es.Columns[es.index("mtu")].Check("many"); !strings.Contains(msg, "整数") {
		t.Errorf("mtu: %q", msg)
	}
	if msg := es.Columns[es.index("mtu")].Check("1400"); msg != "" {
		t.Errorf("mtu 1400: %q", msg)
	}
	if msg := CheckValue(defs(t), "interfaces", "ipv4", "192.0.2.300/24"); msg == "" {
		t.Error("no problem for an address that is none")
	}
	if msg := CheckValue(defs(t), "@devices", "new_host", "anything"); msg != "" {
		t.Errorf("a sheet without a definition: %q", msg)
	}
}

func TestApplyWritesOnlyTheChanges(t *testing.T) {
	data := editBook(t)
	es := editSheet(t, data, "interfaces")
	row, old := cellOf(t, es, "GigabitEthernet0/0", "description")
	d := draftOf(DraftSheet{Sheet: "interfaces", Changed: []DraftRow{{Row: row.Row, Key: es.keyOf(row.Cells),
		Old: map[string]string{"description": old}, New: map[string]string{"description": "to sample-wan-2"}}}})
	if d.Count() != 1 {
		t.Fatalf("count = %d", d.Count())
	}
	next, sum, conflicts, err := Apply(data, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if len(conflicts) != 0 || sum.Cells != 1 || sum.Added+sum.Deleted+sum.Cleared+sum.Skipped != 0 {
		t.Fatalf("sum = %+v, conflicts = %+v", sum, conflicts)
	}
	diff := changed(all(t, data), all(t, next))
	if len(diff) != 1 || !strings.HasPrefix(diff[0], "インタフェース!") {
		t.Fatalf("cells that differ: %v", diff)
	}
	_, now := cellOf(t, editSheet(t, next, "interfaces"), "GigabitEthernet0/0", "description")
	if now != "to sample-wan-2" {
		t.Errorf("description = %q", now)
	}
	// The colour, the free column, the notes and the comments stay.
	if before, after := inspect(t, data), inspect(t, next); before != after || !strings.Contains(after, "FFFF00") {
		t.Errorf("what a person made in Excel changed:\n%s\n%s", before, after)
	}
}

func TestApplyConflicts(t *testing.T) {
	data := editBook(t)
	es := editSheet(t, data, "interfaces")
	row, old := cellOf(t, es, "GigabitEthernet0/0", "description")
	_, mtu := cellOf(t, es, "GigabitEthernet0/0", "mtu")
	d := draftOf(DraftSheet{Sheet: "interfaces", Changed: []DraftRow{{Row: row.Row, Key: es.keyOf(row.Cells),
		Old: map[string]string{"description": old, "mtu": mtu}, New: map[string]string{"description": "mine", "mtu": "1400"}}}})
	dcol := es.Columns[es.index("description")].Col
	// Meanwhile the same cell is changed in Excel.
	theirs := inExcel(t, data, func(f *excelize.File) {
		must(t, f.SetCellStr("インタフェース", addr(dcol, row.Row-1), "theirs"))
	})

	next, _, conflicts, err := Apply(theirs, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if next != nil || len(conflicts) != 1 {
		t.Fatalf("written without a choice: %v, conflicts %+v", next != nil, conflicts)
	}
	c := conflicts[0]
	if c.Kind != "cell" || c.Old != old || c.Mine != "mine" || c.Theirs != "theirs" || c.Row != row.Row || !strings.Contains(c.Label, "GigabitEthernet0/0") {
		t.Errorf("conflict = %+v", c)
	}
	// The table of the screen shows the clash before the save.
	v := editSheet(t, theirs, "interfaces").View(d.Of("interfaces"))
	vr := v.Rows[0]
	di := es.index("description")
	if vr.Cells[di] != "mine" || vr.Base[di] != old || len(vr.Theirs) != 1 || v.Draft != 2 {
		t.Errorf("view row = %+v", vr)
	}

	// The value of the book stays: the other cell is still written.
	next, sum, _, err := Apply(theirs, editFile, defs(t), "ja", d, map[string]string{c.ID: "theirs"})
	must(t, err)
	if _, now := cellOf(t, editSheet(t, next, "interfaces"), "GigabitEthernet0/0", "description"); now != "theirs" || sum.Cells != 1 {
		t.Errorf("theirs: description = %q, sum %+v", now, sum)
	}
	// The value of the draft wins.
	next, sum, _, err = Apply(theirs, editFile, defs(t), "ja", d, map[string]string{c.ID: "mine"})
	must(t, err)
	if _, now := cellOf(t, editSheet(t, next, "interfaces"), "GigabitEthernet0/0", "description"); now != "mine" || sum.Cells != 2 {
		t.Errorf("mine: description = %q, sum %+v", now, sum)
	}
	// The same value written in Excel already is no conflict.
	same := inExcel(t, data, func(f *excelize.File) {
		must(t, f.SetCellStr("インタフェース", addr(dcol, row.Row-1), "mine"))
	})
	_, sum, conflicts, err = Apply(same, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if len(conflicts) != 0 || sum.Cells != 1 {
		t.Errorf("same value: sum %+v, conflicts %+v", sum, conflicts)
	}
}

func TestApplyRows(t *testing.T) {
	data := editBook(t)
	es := editSheet(t, data, "interfaces")
	gone, _ := cellOf(t, es, "GigabitEthernet0/1", "description")
	d := draftOf(DraftSheet{Sheet: "interfaces",
		Added:   []DraftAdd{{Cells: map[string]string{"device": "rt-a02", "name": "GigabitEthernet0/0", "mtu": "1400"}}, {Cells: map[string]string{"name": " "}}},
		Deleted: []DraftDel{{Row: gone.Row, Key: es.keyOf(gone.Cells), Sum: gone.Sum}}})
	if d.Count() != 2 {
		t.Fatalf("count = %d (a row without a value is no change)", d.Count())
	}
	next, sum, conflicts, err := Apply(data, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if len(conflicts) != 0 || sum.Added != 1 || sum.Deleted != 1 || sum.Cleared != 0 {
		t.Fatalf("sum = %+v, conflicts = %+v", sum, conflicts)
	}
	after := editSheet(t, next, "interfaces")
	if len(after.Rows) != 2 || after.Rows[1].Row != 3 || after.Rows[1].Cells[after.index("device")] != "rt-a02" || after.Rows[1].Cells[after.index("mtu")] != "1400" {
		t.Fatalf("rows = %+v", after.Rows)
	}
	if before, now := inspect(t, data), inspect(t, next); before != now {
		t.Errorf("what a person made in Excel changed:\n%s\n%s", before, now)
	}

	// A row that was changed in Excel is not deleted without a word.
	theirs := inExcel(t, data, func(f *excelize.File) {
		must(t, f.SetCellStr("インタフェース", addr(es.Columns[es.index("description")].Col, gone.Row-1), "still needed"))
	})
	next, _, conflicts, err = Apply(theirs, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if next != nil || len(conflicts) != 1 || conflicts[0].Kind != "deleted" {
		t.Fatalf("conflicts = %+v", conflicts)
	}
	if v := editSheet(t, theirs, "interfaces").View(d.Of("interfaces")); !v.Rows[1].Deleted || !v.Rows[1].Clash || v.Rows[1].Sum != gone.Sum {
		t.Errorf("view row = %+v", v.Rows[1])
	}
	next, sum, _, err = Apply(theirs, editFile, defs(t), "ja", d, map[string]string{conflicts[0].ID: "theirs"})
	must(t, err)
	if sum.Deleted != 0 || sum.Added != 1 || len(editSheet(t, next, "interfaces").Rows) != 3 {
		t.Errorf("theirs: sum = %+v", sum)
	}
	// A row deleted in Excel as well leaves nothing to do.
	both := inExcel(t, data, func(f *excelize.File) { must(t, f.RemoveRow("インタフェース", gone.Row)) })
	_, sum, conflicts, err = Apply(both, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if len(conflicts) != 0 || sum.Deleted != 0 || sum.Added != 1 {
		t.Errorf("deleted on both sides: sum = %+v, conflicts = %+v", sum, conflicts)
	}
}

func TestApplyRowMovedOrGone(t *testing.T) {
	data := editBook(t)
	es := editSheet(t, data, "interfaces")
	row, old := cellOf(t, es, "GigabitEthernet0/1", "description")
	d := draftOf(DraftSheet{Sheet: "interfaces", Changed: []DraftRow{{Row: row.Row, Key: es.keyOf(row.Cells),
		Old: map[string]string{"description": old}, New: map[string]string{"description": "to sample-lan"}}}})

	// A row put in above it in Excel: the change follows its row.
	moved := inExcel(t, data, func(f *excelize.File) {
		must(t, f.InsertRows("インタフェース", 2, 1))
		must(t, f.SetCellStr("インタフェース", addr(es.Columns[es.index("device")].Col, 1), "rt-a00"))
		must(t, f.SetCellStr("インタフェース", addr(es.Columns[es.index("name")].Col, 1), "Loopback0"))
	})
	next, sum, conflicts, err := Apply(moved, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if len(conflicts) != 0 || sum.Cells != 1 {
		t.Fatalf("sum = %+v, conflicts = %+v", sum, conflicts)
	}
	after := editSheet(t, next, "interfaces")
	if r, now := cellOf(t, after, "GigabitEthernet0/1", "description"); now != "to sample-lan" || r.Row != row.Row+1 {
		t.Errorf("row %d description %q", r.Row, now)
	}
	if _, v := cellOf(t, after, "Loopback0", "description"); v != "" {
		t.Errorf("the row put in got the value: %q", v)
	}

	// The row deleted in Excel: a person chooses.
	gone := inExcel(t, data, func(f *excelize.File) { must(t, f.RemoveRow("インタフェース", row.Row)) })
	next, _, conflicts, err = Apply(gone, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if next != nil || len(conflicts) != 1 || conflicts[0].Kind != "gone" || !strings.Contains(conflicts[0].Label, "GigabitEthernet0/1") {
		t.Fatalf("conflicts = %+v", conflicts)
	}
	if v := editSheet(t, gone, "interfaces").View(d.Of("interfaces")); len(v.Rows) != 2 || !v.Rows[1].Gone {
		t.Errorf("view rows = %+v", v.Rows)
	}
	next, sum, _, err = Apply(gone, editFile, defs(t), "ja", d, map[string]string{conflicts[0].ID: "mine"})
	must(t, err)
	if r, now := cellOf(t, editSheet(t, next, "interfaces"), "GigabitEthernet0/1", "description"); sum.Added != 1 || now != "to sample-lan" || r.Cells[es.index("device")] != "rt-a01" {
		t.Errorf("mine: sum = %+v, row = %+v", sum, r)
	}
	next, sum, _, err = Apply(gone, editFile, defs(t), "ja", d, map[string]string{conflicts[0].ID: "theirs"})
	must(t, err)
	if sum.Changed() || !bytes.Equal(next, gone) {
		t.Errorf("theirs: sum = %+v", sum)
	}

	// A sheet that is not in the book any more can only be given up.
	lost := draftOf(DraftSheet{Sheet: "no_such_table", Added: []DraftAdd{{Cells: map[string]string{"device": "rt-a01"}}}})
	next, _, conflicts, err = Apply(data, editFile, defs(t), "ja", lost, map[string]string{"sheet|no_such_table": "mine"})
	must(t, err)
	if len(conflicts) != 1 || conflicts[0].Kind != "sheet" || !bytes.Equal(next, data) {
		t.Errorf("conflicts = %+v", conflicts)
	}
}

func TestApplyFormulaAndComments(t *testing.T) {
	data := editBook(t)
	es := editSheet(t, data, "interfaces")
	row, _ := cellOf(t, es, "GigabitEthernet0/0", "description")
	dcol := es.Columns[es.index("description")].Col
	cell := addr(dcol, row.Row-1)
	withFormula := inExcel(t, data, func(f *excelize.File) {
		must(t, f.SetCellFormula("インタフェース", cell, "CONCATENATE(A2,B2)"))
	})
	fs := editSheet(t, withFormula, "interfaces")
	if r := fs.Rows[0]; len(r.Formula) != 1 || r.Formula[0] != fs.index("description") {
		t.Fatalf("formula cells = %v", r.Formula)
	}
	// A cell with a formula is never written over.
	d := draftOf(DraftSheet{Sheet: "interfaces", Changed: []DraftRow{{Row: row.Row, Key: es.keyOf(row.Cells),
		Old: map[string]string{"description": fs.Rows[0].Cells[fs.index("description")]}, New: map[string]string{"description": "typed"}}}})
	next, sum, _, err := Apply(withFormula, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if sum.Skipped != 1 || sum.Cells != 0 || !bytes.Equal(next, withFormula) {
		t.Errorf("sum = %+v", sum)
	}

	// A sheet with a comment on a cell of a row keeps its rows: the row to
	// delete is emptied.
	withComment := inExcel(t, data, func(f *excelize.File) {
		must(t, f.AddComment("インタフェース", excelize.Comment{Cell: "C3", Author: "sample", Text: "sample comment"}))
	})
	cs := editSheet(t, withComment, "interfaces")
	first := cs.Rows[0]
	del := draftOf(DraftSheet{Sheet: "interfaces", Deleted: []DraftDel{{Row: first.Row, Key: cs.keyOf(first.Cells), Sum: first.Sum}}})
	next, sum, _, err = Apply(withComment, editFile, defs(t), "ja", del, nil)
	must(t, err)
	after := editSheet(t, next, "interfaces")
	if sum.Cleared != 1 || sum.Deleted != 0 || len(after.Rows) != 1 || after.Rows[0].Row != 3 {
		t.Errorf("sum = %+v, rows = %+v", sum, after.Rows)
	}
}

func TestApplyNewColumn(t *testing.T) {
	// A sheet without a column of the definition gets the heading at its
	// right end when a value is saved for it.
	data := inExcel(t, editBook(t), func(f *excelize.File) {
		must(t, f.SetCellStr("新旧対照表（機器）", "A2", "rt-old01"))
		must(t, f.SetCellStr("新旧対照表（機器）", "F1", ""))
	})
	es := editSheet(t, data, "@devices")
	note := es.index("note")
	if es.Columns[note].Col != -1 || note != len(es.Columns)-1 {
		t.Fatalf("columns = %+v", es.Columns)
	}
	row := es.Rows[0]
	d := draftOf(DraftSheet{Sheet: "@devices", Changed: []DraftRow{{Row: row.Row, Key: es.keyOf(row.Cells),
		Old: map[string]string{"new_host": "", "note": ""}, New: map[string]string{"new_host": "rt-new01", "note": "sample reason"}}}})
	next, sum, _, err := Apply(data, editFile, defs(t), "ja", d, nil)
	must(t, err)
	bk, err := Read(next, editFile, defs(t))
	must(t, err)
	if sum.Cells != 2 || len(bk.Devices) != 1 || bk.Devices[0].NewHost != "rt-new01" || bk.Devices[0].Note != "sample reason" {
		t.Errorf("sum = %+v, devices = %+v", sum, bk.Devices)
	}
}
