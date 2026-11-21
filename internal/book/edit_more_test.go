package book

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// frameOf tells what the tool gives a sheet it makes: the headings, the
// comments on them and the lists to choose from.
func frameOf(t *testing.T, data []byte, name string) string {
	t.Helper()
	f, err := excelize.OpenReader(strings.NewReader(string(data)))
	must(t, err)
	defer f.Close()
	rows, err := f.GetRows(name)
	must(t, err)
	if len(rows) == 0 {
		t.Fatalf("the sheet %s has no heading", name)
	}
	comments, err := f.GetComments(name)
	must(t, err)
	var notes []string
	for _, c := range comments {
		notes = append(notes, c.Cell+"="+c.Text)
	}
	dvs, err := f.GetDataValidations(name)
	must(t, err)
	var lists []string
	for _, dv := range dvs {
		lists = append(lists, dv.Sqref)
	}
	style, err := f.GetCellStyle(name, "A1")
	must(t, err)
	st, err := f.GetStyle(style)
	must(t, err)
	body, err := f.GetCellStyle(name, "A2")
	must(t, err)
	bst, err := f.GetStyle(body)
	must(t, err)
	panes, err := f.GetPanes(name)
	must(t, err)
	return fmt.Sprintf("%v | %v | %v | bold %v fmt %d | body fmt %d | frozen %v", rows[0], notes, lists, st.Font != nil && st.Font.Bold, st.NumFmt, bst.NumFmt, panes.Freeze)
}

func TestApplyMakesMissingSheet(t *testing.T) {
	data := editBook(t)
	ntp := editSheet(t, data, "ntp_servers")
	common := editSheet(t, data, "@common")
	want, wantCommon := frameOf(t, data, ntp.Name), frameOf(t, data, common.Name)
	// A book from before the table was there, or one a person took the sheet out of.
	old := inExcel(t, data, func(f *excelize.File) {
		must(t, f.DeleteSheet(ntp.Name))
		must(t, f.DeleteSheet(common.Name))
	})
	es := editSheet(t, old, "ntp_servers")
	if !es.Missing || len(es.Rows) != 0 || len(es.Columns) != len(ntp.Columns) || es.Name != ntp.Name {
		t.Fatalf("missing sheet = %+v", es)
	}
	for i, c := range es.Columns {
		if c.Col != i || c.Title == "" {
			t.Errorf("column %d = %+v", i, c)
		}
	}
	if !es.Columns[es.index("device")].IsKey || !es.Columns[es.index("server")].IsKey {
		t.Errorf("the key columns are not marked: %+v", es.Columns)
	}
	if editSheet(t, old, "interfaces").Missing {
		t.Error("a sheet the book has is told to be missing")
	}

	d := draftOf(
		DraftSheet{Sheet: "ntp_servers", Added: []DraftAdd{{Cells: map[string]string{"device": "rt-a01", "server": "192.0.2.123", "prefer": "true"}}}},
		DraftSheet{Sheet: "@common", Added: []DraftAdd{{Cells: map[string]string{"scope": "*", "table": "ntp_servers", "column": "prefer", "value": "true"}}}})
	if v := es.View(d.Of("ntp_servers")); len(v.Rows) != 1 || v.Rows[0].Row != 0 || !v.Missing {
		t.Fatalf("view = %+v", v.Rows)
	}
	next, sum, conflicts, err := Apply(old, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if len(conflicts) != 0 || sum.Sheets != 2 || sum.Added != 2 {
		t.Fatalf("sum = %+v, conflicts = %+v", sum, conflicts)
	}
	// The sheets are as the tool makes them anywhere else.
	if got := frameOf(t, next, ntp.Name); got != want {
		t.Errorf("the sheet that was made:\n got %s\nwant %s", got, want)
	}
	if got := frameOf(t, next, common.Name); got != wantCommon {
		t.Errorf("the sheet of the common values that was made:\n got %s\nwant %s", got, wantCommon)
	}
	bk, err := Read(next, editFile, defs(t))
	must(t, err)
	row := bk.Data.Tables["ntp_servers"].Find(kv("device", "rt-a01", "server", "192.0.2.123"))
	if row == nil || row.Get("prefer") != "true" || len(bk.Common) != 1 || bk.Common[0].Value != "true" {
		t.Errorf("the rows did not arrive: %+v, %+v", row, bk.Common)
	}
	after := editSheet(t, next, "ntp_servers")
	if after.Missing || len(after.Rows) != 1 || after.Rows[0].Row != 2 {
		t.Errorf("after = %+v", after.Rows)
	}
	if before, now := inspect(t, old), inspect(t, next); strings.Replace(now, " "+common.Name+" "+ntp.Name+"]", "]", 1) != before {
		t.Errorf("what a person made in Excel changed:\n%s\n%s", before, now)
	}
	// The same draft gives the same book.
	again, _, _, err := Apply(old, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if !reflect.DeepEqual(all(t, again), all(t, next)) {
		t.Error("the same draft gave another book")
	}

	// Rows that were changed in a sheet that is gone have no place.
	lost := draftOf(DraftSheet{Sheet: "ntp_servers", Changed: []DraftRow{{Row: 2, Key: map[string]string{"device": "rt-a01", "server": "192.0.2.123"},
		Old: map[string]string{"prefer": "true"}, New: map[string]string{"prefer": "false"}}}})
	out, _, conflicts, err := Apply(old, editFile, defs(t), "ja", lost, nil)
	must(t, err)
	if out != nil || len(conflicts) != 1 || conflicts[0].Kind != "sheet" {
		t.Errorf("conflicts = %+v", conflicts)
	}
	// A free column alone makes the sheet too; nothing to write does not.
	only := draftOf(DraftSheet{Sheet: "ntp_servers", Columns: []DraftCol{{Op: "add", Title: "設置メモ"}}})
	next, sum, _, err = Apply(old, editFile, defs(t), "ja", only, nil)
	must(t, err)
	if es := editSheet(t, next, "ntp_servers"); sum.Sheets != 1 || es.Missing || es.index("free:設置メモ") < 0 {
		t.Errorf("sum = %+v", sum)
	}
}

func TestApplyInsertsRows(t *testing.T) {
	data := editBook(t)
	es := editSheet(t, data, "interfaces")
	first, second := es.Rows[0], es.Rows[1] // rows 2 and 3 of the sheet
	name := es.index("name")
	in := func(n string, before *EditRow) DraftAdd {
		a := DraftAdd{Cells: map[string]string{"device": "rt-a01", "name": n}}
		if before != nil {
			a.Before, a.BeforeKey = before.Row, es.keyOf(before.Cells)
		}
		return a
	}
	part := DraftSheet{Sheet: "interfaces", Added: []DraftAdd{in("Loopback1", &second), in("Loopback2", &second), in("Loopback9", nil)}}
	d := draftOf(part)

	// The table shows the rows where they will be, and gives the same draft back.
	v := es.View(d.Of("interfaces"))
	var shown []string
	for _, r := range v.Rows {
		shown = append(shown, fmt.Sprintf("%d:%s", r.Row, r.Cells[name]))
	}
	if got := strings.Join(shown, " "); got != "2:GigabitEthernet0/0 0:Loopback1 0:Loopback2 3:GigabitEthernet0/1 0:Loopback9" {
		t.Fatalf("view = %s", got)
	}
	if back := v.Part(); !reflect.DeepEqual(back, *d.Of("interfaces")) {
		t.Errorf("the draft of the view:\n got %+v\nwant %+v", back, *d.Of("interfaces"))
	}

	next, sum, conflicts, err := Apply(data, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if len(conflicts) != 0 || sum.Added != 3 || sum.Moved != 0 {
		t.Fatalf("sum = %+v, conflicts = %+v", sum, conflicts)
	}
	order := func(data []byte) string {
		var out []string
		for _, r := range editSheet(t, data, "interfaces").Rows {
			out = append(out, fmt.Sprintf("%d:%s", r.Row, r.Cells[name]))
		}
		return strings.Join(out, " ")
	}
	if got := order(next); got != "2:GigabitEthernet0/0 3:Loopback1 4:Loopback2 5:GigabitEthernet0/1 6:Loopback9" {
		t.Errorf("rows = %s", got)
	}
	if before, now := inspect(t, data), inspect(t, next); before != now {
		t.Errorf("what a person made in Excel changed:\n%s\n%s", before, now)
	}
	// The rows that moved down kept their cells, and the lists to choose
	// from still end at the last row of a sheet.
	after := editSheet(t, next, "interfaces")
	if !reflect.DeepEqual(after.Rows[3].Cells, second.Cells) {
		t.Errorf("the row below changed: %v", after.Rows[3].Cells)
	}
	f, err := excelize.OpenReader(strings.NewReader(string(next)))
	must(t, err)
	dvs, err := f.GetDataValidations("インタフェース")
	must(t, err)
	for _, dv := range dvs {
		if !strings.HasSuffix(dv.Sqref, "1048576") || !strings.Contains(dv.Sqref, "2:") {
			t.Errorf("a list to choose from covers %s", dv.Sqref)
		}
	}
	f.Close()

	// A row put in above a row that is deleted with the same save, and one
	// above the first row.
	both := draftOf(DraftSheet{Sheet: "interfaces", Added: []DraftAdd{in("Loopback0", &first), in("Loopback1", &second)},
		Deleted: []DraftDel{{Row: second.Row, Key: es.keyOf(second.Cells), Sum: second.Sum}}})
	next, sum, _, err = Apply(data, editFile, defs(t), "ja", both, nil)
	must(t, err)
	if got := order(next); sum.Added != 2 || sum.Deleted != 1 || got != "2:Loopback0 3:GigabitEthernet0/0 4:Loopback1" {
		t.Errorf("sum = %+v, rows = %s", sum, got)
	}

	// The row it was put in above moved in Excel: the row follows it. The
	// row is gone: the new row goes to the end, and the save tells so.
	moved := inExcel(t, data, func(f *excelize.File) { must(t, f.InsertRows("インタフェース", 2, 1)) })
	one := draftOf(DraftSheet{Sheet: "interfaces", Added: []DraftAdd{in("Loopback1", &second)}})
	next, sum, _, err = Apply(moved, editFile, defs(t), "ja", one, nil)
	must(t, err)
	if got := order(next); sum.Moved != 0 || got != "3:GigabitEthernet0/0 4:Loopback1 5:GigabitEthernet0/1" {
		t.Errorf("moved: sum = %+v, rows = %s", sum, got)
	}
	gone := inExcel(t, data, func(f *excelize.File) { must(t, f.RemoveRow("インタフェース", second.Row)) })
	next, sum, _, err = Apply(gone, editFile, defs(t), "ja", one, nil)
	must(t, err)
	if got := order(next); sum.Moved != 1 || sum.Added != 1 || got != "2:GigabitEthernet0/0 3:Loopback1" {
		t.Errorf("gone: sum = %+v, rows = %s", sum, got)
	}
	if v := editSheet(t, gone, "interfaces").View(one.Of("interfaces")); len(v.Rows) != 2 || v.Rows[1].Row != 0 {
		t.Errorf("the view of a row whose place is gone: %+v", v.Rows)
	}

	// A sheet with a comment on a cell of a row keeps its rows where they
	// are: the new row goes to the end.
	withComment := inExcel(t, data, func(f *excelize.File) {
		must(t, f.AddComment("インタフェース", excelize.Comment{Cell: "C3", Author: "sample", Text: "sample comment"}))
	})
	next, sum, _, err = Apply(withComment, editFile, defs(t), "ja", one, nil)
	must(t, err)
	if got := order(next); sum.Moved != 1 || sum.Added != 1 || got != "2:GigabitEthernet0/0 3:GigabitEthernet0/1 4:Loopback1" {
		t.Errorf("comments: sum = %+v, rows = %s", sum, got)
	}
}

func TestApplyFreeColumns(t *testing.T) {
	data := editBook(t) // the sheet of the interfaces has the free column 回線メモ in AZ
	es := editSheet(t, data, "interfaces")
	row := es.Rows[0]
	part := DraftSheet{Sheet: "interfaces",
		Columns: []DraftCol{{Op: "rename", Title: "回線メモ", To: "回線の覚え"}, {Op: "add", Title: "工事メモ"}, {Op: "add", Title: "空の列"}, {Op: "add", Title: " "}},
		Changed: []DraftRow{{Row: row.Row, Key: es.keyOf(row.Cells), Old: map[string]string{"free:工事メモ": "", "free:回線メモ": "sample note"},
			New: map[string]string{"free:工事メモ": "sample work", "free:回線メモ": "sample note 2"}}}}
	d := draftOf(part)
	if d.Count() != 5 {
		t.Fatalf("count = %d", d.Count())
	}
	v := es.View(d.Of("interfaces"))
	work, line := v.index("free:工事メモ"), v.index("free:回線メモ")
	if work < 0 || !v.Columns[work].Added || v.Columns[work].Col != -1 || v.Columns[line].Title != "回線の覚え" || v.Columns[line].Was != "回線メモ" {
		t.Fatalf("columns = %+v", v.Columns)
	}
	if v.Rows[0].Cells[work] != "sample work" || v.Rows[0].Cells[line] != "sample note 2" || len(v.Rows[1].Cells) != len(v.Columns) {
		t.Fatalf("rows = %+v", v.Rows)
	}
	if len(es.Columns) == len(v.Columns) {
		t.Error("the view changed the sheet it was made from")
	}
	if back := v.Part(); !reflect.DeepEqual(back, *d.Of("interfaces")) {
		t.Errorf("the draft of the view:\n got %+v\nwant %+v", back, *d.Of("interfaces"))
	}
	next, sum, conflicts, err := Apply(data, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if len(conflicts) != 0 || sum.Cells != 2 || sum.Columns != 3 {
		t.Fatalf("sum = %+v, conflicts = %+v", sum, conflicts)
	}
	after := editSheet(t, next, "interfaces")
	for _, k := range []string{"free:工事メモ", "free:空の列", "free:回線の覚え"} {
		if ci := after.index(k); ci < 0 || !after.Columns[ci].Free || after.Columns[ci].Col < 0 {
			t.Errorf("no column %s in %+v", k, after.Columns)
		}
	}
	if after.index("free:回線メモ") >= 0 || after.Rows[0].Cells[after.index("free:工事メモ")] != "sample work" || after.Rows[0].Cells[after.index("free:回線の覚え")] != "sample note 2" {
		t.Errorf("rows = %+v", after.Rows[0])
	}
	if got, want := frameOf(t, next, "インタフェース"), frameOf(t, data, "インタフェース"); strings.Split(got, " | ")[1] != strings.Split(want, " | ")[1] || strings.Split(got, " | ")[2] != strings.Split(want, " | ")[2] {
		t.Errorf("the comments or the lists of the headings changed:\n%s\n%s", got, want)
	}

	// A column taken out: the sheet loses it, the columns right of it move left.
	two := inExcel(t, data, func(f *excelize.File) {
		must(t, f.SetCellStr("インタフェース", "BA1", "予備"))
		must(t, f.SetCellStr("インタフェース", "BA2", "sample spare"))
	})
	drop := draftOf(DraftSheet{Sheet: "interfaces", Columns: []DraftCol{{Op: "drop", Title: "回線メモ"}}})
	if v := editSheet(t, two, "interfaces").View(drop.Of("interfaces")); !v.Columns[v.index("free:回線メモ")].Dropped || !reflect.DeepEqual(v.Part(), *drop.Of("interfaces")) {
		t.Errorf("the view of a column to take out: %+v", v.Columns)
	}
	next, sum, _, err = Apply(two, editFile, defs(t), "ja", drop, nil)
	must(t, err)
	after = editSheet(t, next, "interfaces")
	if spare := after.index("free:予備"); sum.Columns != 1 || after.index("free:回線メモ") >= 0 || spare < 0 || after.Columns[spare].Col != 51 || after.Rows[0].Cells[spare] != "sample spare" {
		t.Errorf("sum = %+v, columns = %+v", sum, after.Columns)
	}
	if before, now := frameOf(t, two, "インタフェース"), frameOf(t, next, "インタフェース"); strings.Split(before, " | ")[1] != strings.Split(now, " | ")[1] || strings.Split(before, " | ")[2] != strings.Split(now, " | ")[2] {
		t.Errorf("the comments or the lists of the headings changed:\n%s\n%s", before, now)
	}
	// With a comment right of it the column stays, emptied.
	noted := inExcel(t, two, func(f *excelize.File) {
		must(t, f.AddComment("インタフェース", excelize.Comment{Cell: "BA1", Author: "sample", Text: "sample comment"}))
	})
	next, sum, _, err = Apply(noted, editFile, defs(t), "ja", drop, nil)
	must(t, err)
	after = editSheet(t, next, "interfaces")
	if spare := after.index("free:予備"); sum.Columns != 1 || after.index("free:回線メモ") >= 0 || spare < 0 || after.Columns[spare].Col != 52 || after.Rows[0].Cells[spare] != "sample spare" {
		t.Errorf("with a comment: sum = %+v, columns = %+v", sum, after.Columns)
	}
	if cells := all(t, next); cells["インタフェース!AZ1"] != "" || cells["インタフェース!AZ2"] != "" {
		t.Error("the column that stays was not emptied")
	}

	// The headings a free column cannot have.
	sc := defs(t)
	if key, err := FreeColumn(sc, "interfaces", " 工事 メモ ", []string{"機器", "回線メモ"}); err != nil || key != "free:工事メモ" {
		t.Errorf("key = %q, %v", key, err)
	}
	for _, bad := range []string{"", "  ", "MTU", "mtu", "回線 メモ", "a\nb"} {
		if _, err := FreeColumn(sc, "interfaces", bad, []string{"機器", "回線メモ"}); err == nil {
			t.Errorf("the heading %q was taken", bad)
		}
	}
	if _, err := FreeColumn(sc, "@devices", "旧 ホスト名", nil); err == nil {
		t.Error("a heading of the device mapping was taken")
	}
}

func TestFindAndReplace(t *testing.T) {
	data := editBook(t)
	sheets, err := ReadSheets(data, editFile, defs(t), "ja")
	must(t, err)
	es := editSheet(t, data, "interfaces")
	row := es.Rows[0]
	// The draft is searched, not the book alone.
	d := draftOf(DraftSheet{Sheet: "interfaces", Changed: []DraftRow{{Row: row.Row, Key: es.keyOf(row.Cells),
		Old: map[string]string{"mtu": "1500"}, New: map[string]string{"mtu": "sample-mtu"}}}})
	count := func(list []FindCount) map[string]int {
		out := map[string]int{}
		for _, c := range list {
			out[c.Sheet] = c.Count
		}
		return out
	}
	// "sample": the description, the free column and the draft of the
	// interfaces, the banner of the device, the name of the user; the
	// password only while the secrets are shown.
	got := count(Find(sheets, d, "SAMPLE", FindOptions{}))
	if got["interfaces"] != 3 || got["devices"] != 1 || got["users"] != 1 || len(got) != 3 {
		t.Errorf("found %v", got)
	}
	if got := count(Find(sheets, d, "SAMPLE", FindOptions{Case: true})); len(got) != 0 {
		t.Errorf("found %v with the case", got)
	}
	if got := count(Find(sheets, d, "sample", FindOptions{Secrets: true})); got["users"] != 2 {
		t.Errorf("found %v with the secrets", got)
	}
	if got := count(Find(sheets, d, "sample-admin", FindOptions{Whole: true})); got["users"] != 1 || len(got) != 1 {
		t.Errorf("found %v as a whole cell", got)
	}
	if got := count(Find(sheets, d, "sample", FindOptions{Hidden: map[string][]string{"interfaces": {"description", "free:回線メモ"}}})); got["interfaces"] != 1 {
		t.Errorf("found %v with hidden columns", got)
	}
	if len(Find(sheets, d, "", FindOptions{})) != 0 {
		t.Error("an empty text was found")
	}

	parts, counts := Replace(sheets, d, "Sample", "example", FindOptions{}, "devices")
	n := count(counts)
	if len(parts) != 2 || n["interfaces"] != 3 || n["users"] != 1 || n["devices"] != 0 {
		t.Fatalf("replaced %v, parts = %+v", n, parts)
	}
	for _, p := range parts {
		d.Put(p)
	}
	iv := es.View(d.Of("interfaces"))
	if iv.Rows[0].Cells[iv.index("description")] != "to example-wan" || iv.Rows[0].Cells[iv.index("mtu")] != "example-mtu" || iv.Rows[0].Cells[iv.index("free:回線メモ")] != "example note" {
		t.Errorf("row = %v", iv.Rows[0].Cells)
	}
	// The cell the draft had changed keeps the value of the book as its old one.
	ch := d.Of("interfaces").Changed
	if len(ch) != 1 || ch[0].Old["mtu"] != "1500" || ch[0].Old["description"] != "to sample-wan" {
		t.Errorf("draft = %+v", ch)
	}
	us := editSheet(t, data, "users")
	uv := us.View(d.Of("users"))
	if uv.Rows[0].Cells[uv.index("name")] != "example-admin" || uv.Rows[0].Cells[uv.index("secret")] != "sample-pass-1" {
		t.Errorf("user = %v", uv.Rows[0].Cells)
	}
	next, sum, conflicts, err := Apply(data, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if len(conflicts) != 0 || sum.Cells != 4 {
		t.Errorf("sum = %+v, conflicts = %+v", sum, conflicts)
	}
	if cells := all(t, next); cells["メモ!A1"] != "sample memo" {
		t.Error("a sheet the tool does not know was changed")
	}

	// A column with choices takes no value that is none of them; a row to
	// delete and a cell with a formula are left alone.
	enabled := es.index("enabled")
	yes := row.Cells[enabled]
	_, counts = Replace(sheets, &Draft{}, yes, "sample", FindOptions{Whole: true}, "")
	for _, c := range counts {
		if c.Sheet == "interfaces" && (c.Count != 0 || c.Skipped == 0) {
			t.Errorf("a value that is no choice was put in: %+v", c)
		}
	}
	del := draftOf(DraftSheet{Sheet: "interfaces", Deleted: []DraftDel{{Row: row.Row, Key: es.keyOf(row.Cells), Sum: row.Sum}}})
	parts, counts = Replace(sheets, del, "sample-wan", "example-wan", FindOptions{}, "")
	if len(parts) != 0 || len(counts) != 1 || counts[0].Skipped != 1 {
		t.Errorf("a row to delete was changed: %+v, %+v", parts, counts)
	}
}

func TestViewPartRoundTrip(t *testing.T) {
	data := editBook(t)
	es := editSheet(t, data, "interfaces")
	a, b := es.Rows[0], es.Rows[1]
	part := DraftSheet{Sheet: "interfaces",
		Changed: []DraftRow{
			{Row: a.Row, Key: es.keyOf(a.Cells), Old: map[string]string{"mtu": "1500"}, New: map[string]string{"mtu": "1400"}},
			// A row that is not in the book any more stays in the draft as it is.
			{Row: 9, Key: map[string]string{"device": "rt-a09", "name": "GigabitEthernet0/9"}, Old: map[string]string{"mtu": ""}, New: map[string]string{"mtu": "1300"}},
		},
		Added:   []DraftAdd{{Cells: map[string]string{"device": "rt-a01", "name": "Loopback0"}, Before: b.Row, BeforeKey: es.keyOf(b.Cells)}, {Cells: map[string]string{"device": "rt-a02", "name": "Loopback0"}}},
		Deleted: []DraftDel{{Row: b.Row, Key: es.keyOf(b.Cells), Sum: b.Sum}},
		Columns: []DraftCol{{Op: "add", Title: "工事メモ"}},
	}
	want := *draftOf(part).Of("interfaces")
	if got := es.View(&want).Part(); !reflect.DeepEqual(got, want) {
		t.Errorf("the draft of the view:\n got %+v\nwant %+v", got, want)
	}
}

func TestApplyMovesRows(t *testing.T) {
	// Four interfaces in rows 2 to 5; the second one has a colour.
	d0 := sampleData()
	put(d0, "interfaces", kv("device", "rt-a01", "name", "GigabitEthernet0/2"), "enabled", "true")
	put(d0, "interfaces", kv("device", "rt-a01", "name", "GigabitEthernet0/3"), "enabled", "true")
	made, _, err := Update(nil, editFile, defs(t), "ja", Change{Data: d0})
	must(t, err)
	data := inExcel(t, made, func(f *excelize.File) {
		style, err := f.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFFF00"}}})
		must(t, err)
		must(t, f.SetCellStyle("インタフェース", "B3", "B3", style))
	})
	es := editSheet(t, data, "interfaces")
	name := es.index("name")
	order := func(data []byte) string {
		var out []string
		for _, r := range editSheet(t, data, "interfaces").Rows {
			out = append(out, fmt.Sprintf("%d:%s", r.Row, strings.TrimPrefix(r.Cells[name], "GigabitEthernet")))
		}
		return strings.Join(out, " ")
	}
	if got := order(data); got != "2:0/0 3:0/1 4:0/2 5:0/3" {
		t.Fatalf("rows = %s", got)
	}
	row := func(i int) EditRow { return es.Rows[i] }
	move := func(from EditRow, before *EditRow) DraftAdd {
		a := DraftAdd{From: from.Row, FromKey: es.keyOf(from.Cells)}
		if before != nil {
			a.Before, a.BeforeKey = before.Row, es.keyOf(before.Cells)
		}
		return a
	}
	first, second, third, last := row(0), row(1), row(2), row(3)

	// The last row goes above the first one, the second one to the end; a
	// new row goes in between, and a cell of the row that moves is changed.
	part := DraftSheet{Sheet: "interfaces",
		Added:   []DraftAdd{move(last, &first), {Cells: map[string]string{"device": "rt-a01", "name": "GigabitEthernet9/9"}, Before: third.Row, BeforeKey: es.keyOf(third.Cells)}, move(second, nil)},
		Changed: []DraftRow{{Row: second.Row, Key: es.keyOf(second.Cells), Old: map[string]string{"description": ""}, New: map[string]string{"description": "sample moved"}}}}
	d := draftOf(part)
	if d.Count() != 4 {
		t.Fatalf("count = %d", d.Count())
	}
	v := es.View(d.Of("interfaces"))
	var shown []string
	for _, r := range v.Rows {
		mark := ""
		if r.Moved {
			mark = "m"
		}
		shown = append(shown, fmt.Sprintf("%d%s:%s", r.Row, mark, strings.TrimPrefix(r.Cells[name], "GigabitEthernet")))
	}
	if got := strings.Join(shown, " "); got != "5m:0/3 2:0/0 0:9/9 4:0/2 3m:0/1" {
		t.Fatalf("view = %s", got)
	}
	if back := v.Part(); !reflect.DeepEqual(back, *d.Of("interfaces")) {
		t.Errorf("the draft of the view:\n got %+v\nwant %+v", back, *d.Of("interfaces"))
	}
	next, sum, conflicts, err := Apply(data, editFile, defs(t), "ja", d, nil)
	must(t, err)
	if len(conflicts) != 0 || sum.Shifted != 2 || sum.Added != 1 || sum.Cells != 1 || sum.Deleted != 0 {
		t.Fatalf("sum = %+v, conflicts = %+v", sum, conflicts)
	}
	if got := order(next); got != "2:0/3 3:0/0 4:9/9 5:0/2 6:0/1" {
		t.Errorf("rows = %s", got)
	}
	// The row took its cells along: the value that was changed, the colour.
	after := editSheet(t, next, "interfaces")
	if after.Rows[4].Cells[after.index("description")] != "sample moved" || after.Rows[1].Cells[after.index("description")] != "to sample-wan" {
		t.Errorf("rows = %+v", after.Rows)
	}
	f, err := excelize.OpenReader(strings.NewReader(string(next)))
	must(t, err)
	style, err := f.GetCellStyle("インタフェース", "B6")
	must(t, err)
	st, err := f.GetStyle(style)
	must(t, err)
	plain, _ := f.GetCellStyle("インタフェース", "B3")
	pst, _ := f.GetStyle(plain)
	if len(st.Fill.Color) != 1 || st.Fill.Color[0] != "FFFF00" || len(pst.Fill.Color) != 0 {
		t.Errorf("the colour did not move with the row: %+v, %+v", st.Fill, pst.Fill)
	}
	f.Close()
	if a, b := frameOf(t, data, "インタフェース"), frameOf(t, next, "インタフェース"); a != b {
		t.Errorf("the headings changed:\n%s\n%s", a, b)
	}

	// A row that is deleted, or gone from the book, is not moved.
	both := draftOf(DraftSheet{Sheet: "interfaces", Added: []DraftAdd{move(last, &first)},
		Deleted: []DraftDel{{Row: last.Row, Key: es.keyOf(last.Cells), Sum: last.Sum}}})
	next, sum, _, err = Apply(data, editFile, defs(t), "ja", both, nil)
	must(t, err)
	if got := order(next); sum.Shifted != 0 || sum.Deleted != 1 || got != "2:0/0 3:0/1 4:0/2" {
		t.Errorf("deleted and moved: sum = %+v, rows = %s", sum, got)
	}
	gone := inExcel(t, data, func(f *excelize.File) { must(t, f.RemoveRow("インタフェース", last.Row)) })
	one := draftOf(DraftSheet{Sheet: "interfaces", Added: []DraftAdd{move(last, &first)}})
	next, sum, _, err = Apply(gone, editFile, defs(t), "ja", one, nil)
	must(t, err)
	if got := order(next); sum.Shifted != 0 || got != "2:0/0 3:0/1 4:0/2" {
		t.Errorf("gone: sum = %+v, rows = %s", sum, got)
	}
	// A sheet with a comment on a cell of a row keeps its rows where they are.
	noted := inExcel(t, data, func(f *excelize.File) {
		must(t, f.AddComment("インタフェース", excelize.Comment{Cell: "C4", Author: "sample", Text: "sample comment"}))
	})
	next, sum, _, err = Apply(noted, editFile, defs(t), "ja", one, nil)
	must(t, err)
	if got := order(next); sum.Shifted != 0 || sum.Stayed != 1 || got != "2:0/0 3:0/1 4:0/2 5:0/3" {
		t.Errorf("comments: sum = %+v, rows = %s", sum, got)
	}
}
