package ops

import (
	"os"
	"strings"
	"testing"
)

// migCase is a case with one old device and a new configuration that differs
// in one description, imported once (the book holds the mapping).
func migCase(t *testing.T) *Env {
	t.Helper()
	text := sample(t, "cisco_ios")
	e := newCase(t, map[string]string{
		"input/old/rt-a01.txt": text,
		"input/new/rt-a01.txt": strings.Replace(text, "description sample loopback", "description sample loopback 2", 1),
	})
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	return e
}

func TestMigrationApproveFinding(t *testing.T) {
	e := migCase(t)
	c, err := e.Compare("old", "new", at)
	if err != nil {
		t.Fatal(err)
	}
	if c.Mig == nil || len(c.Mig.Items) != 1 {
		t.Fatalf("findings = %+v", c.Mig)
	}
	f := c.Mig.Items[0]
	if f.RuleID != "M-405" || f.Device != "rt-a01" || f.Ref.Table != "interfaces" || f.Ref.Column != "description" || f.Approval != nil {
		t.Fatalf("finding = %+v", f)
	}
	if n := c.Mig.Count(); n.Warnings != 1 || n.Errors != 0 || c.Mig.Open() != 1 {
		t.Errorf("count = %+v", n)
	}
	h := read(t, e, c.Report.HTML)
	for _, want := range []string{"未承認（警告）", f.Key, "未承認の警告"} {
		if !strings.Contains(h, want) {
			t.Errorf("the report lacks %q", want)
		}
	}
	before := c.Findings()

	// The fingerprint, by its beginning.
	a, err := e.Approve(ApproveRequest{Finding: f.Key[:8], Reason: "説明を新しい書き方にそろえた", By: "試験"}, at)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Migration || a.Hits != 1 || a.Approval.Rule != "M-405" || a.Approval.Finding != f.Key || a.Approval.Device != "rt-a01" {
		t.Errorf("approved = %+v", a)
	}
	// It counts whichever side comes first.
	for _, pair := range [][2]string{{"old", "new"}, {"new", "old"}} {
		c, err = e.Compare(pair[0], pair[1], at)
		if err != nil {
			t.Fatal(err)
		}
		if n := c.Mig.Count(); n.Approved != 1 || c.Mig.Open() != 0 || c.Mig.Hits[a.Approval.ID] != 1 {
			t.Errorf("%v after the approval: %+v, hits %v", pair, n, c.Mig.Hits)
		}
		if c.Findings() != before-1 {
			t.Errorf("%v: findings = %d, want %d", pair, c.Findings(), before-1)
		}
		h = read(t, e, c.Report.HTML)
		for _, want := range []string{"承認済み", a.Approval.ID + " 試験: 説明を新しい書き方にそろえた", "当たった件数"} {
			if !strings.Contains(h, want) {
				t.Errorf("%v: the report lacks %q", pair, want)
			}
		}
	}
	if _, err := e.Approve(ApproveRequest{Finding: f.Key, Reason: "もう一度", By: "試験"}, at); err == nil || !strings.Contains(err.Error(), "すでに承認されています") {
		t.Errorf("approving twice: %v", err)
	}
	if _, err := e.Approve(ApproveRequest{Finding: "0123456789abcdef", Reason: "無い指摘", By: "試験"}, at); err == nil || !strings.Contains(err.Error(), "今の結果にありません") {
		t.Errorf("approving what is not there: %v", err)
	}

	if err := e.Unapprove(a.Approval.ID, at); err != nil {
		t.Fatal(err)
	}
	if c, err = e.Compare("old", "new", at); err != nil || c.Mig.Open() != 1 {
		t.Errorf("after taking the approval back: %v, %+v", err, c.Mig.Count())
	}
}

func TestMigrationApproveScope(t *testing.T) {
	e := migCase(t)
	// By the names the book shows.
	a, err := e.Approve(ApproveRequest{Rule: "M-405", Devices: "*", Table: "interfaces", Column: "description", Reason: "説明は全台で書き直した", By: "試験"}, at)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Migration || a.Hits != 1 || a.Approval.Scope == nil {
		t.Errorf("approved = %+v", a)
	}
	c, err := e.Compare("old", "new", at)
	if err != nil {
		t.Fatal(err)
	}
	if c.Mig.Open() != 0 || c.Mig.Count().Approved != 1 {
		t.Errorf("count = %+v", c.Mig.Count())
	}
	// Another column is not covered.
	if _, err := e.Approve(ApproveRequest{Rule: "M-405", Devices: "*", Table: "interfaces", Column: "mtu", Reason: "x", By: "試験"}, at); err == nil || !strings.Contains(err.Error(), "その範囲に当たる指摘") {
		t.Errorf("a scope with nothing in it: %v", err)
	}

	// The book holds no design data yet: every row of the old side is only
	// there (M-403, errors). They can be approved by a scope.
	c, err = e.Compare("old", "book", at)
	if err != nil {
		t.Fatal(err)
	}
	n := c.Mig.Count()
	if n.Errors == 0 || n.Errors != len(c.Result.OnlyA) {
		t.Fatalf("old and book: %+v, only A %d", n, len(c.Result.OnlyA))
	}
	if h := read(t, e, c.Report.HTML); !strings.Contains(h, "未承認（エラー）") || !strings.Contains(h, "M-403") {
		t.Error("the report does not show the errors")
	}
	b, err := e.Approve(ApproveRequest{Rule: "M-403", Devices: "rt-a01", Reason: "この機器は作り直す", By: "試験"}, at)
	if err != nil {
		t.Fatal(err)
	}
	if b.Hits != n.Errors {
		t.Errorf("hits = %d, want %d", b.Hits, n.Errors)
	}
	if c, err = e.Compare("old", "book", at); err != nil || c.Mig.Open() != 0 {
		t.Errorf("after the scope: %v, %+v", err, c.Mig.Count())
	}

	// A check that is switched off makes no finding; the difference stays in
	// the report.
	e.Proj.Config.Checks.Disable = []string{"M-403"}
	if c, err = e.Compare("old", "book", at); err != nil || len(c.Mig.Items) != 0 || len(c.Result.OnlyA) == 0 {
		t.Errorf("with M-403 off: %v, %d findings", err, len(c.Mig.Items))
	}
	e.Proj.Config.Checks.Disable = nil
	e.Proj.Config.Checks.Severity = map[string]string{"M-403": "info"}
	if c, err = e.Compare("old", "book", at); err != nil || c.Mig.Open() != 0 || c.Mig.Count().Info == 0 || c.Mig.Count().Approved != 0 {
		t.Errorf("with M-403 as information: %v, %+v", err, c.Mig.Count())
	}
}

func TestMigrationMapping(t *testing.T) {
	e := migCase(t)
	// An empty new name is filled again where there is a proposal (spec
	// 10.2): the new configuration has the interface.
	setCell(t, e, "IF 対応", "旧 IF 名", "Loopback0", "新 IF 名", "")
	o, err := e.RunImport("old", "", at)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Applied.Undecided) != 0 || o.Book.IfFilled != 1 {
		t.Fatalf("with a proposal: undecided %d, filled %d", len(o.Applied.Undecided), o.Book.IfFilled)
	}
	// Without the new configuration there is none: the name stays empty.
	if err := os.Remove(e.Proj.Path("input/new/rt-a01.txt")); err != nil {
		t.Fatal(err)
	}
	setCell(t, e, "IF 対応", "旧 IF 名", "Loopback0", "新 IF 名", "")
	if o, err = e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	if len(o.Applied.Undecided) != 1 || len(o.Mig.Items) != 1 || o.Mig.Open() != 1 {
		t.Fatalf("undecided %d, findings %+v", len(o.Applied.Undecided), o.Mig.Items)
	}
	f := o.Mig.Items[0]
	if f.RuleID != "M-406" || f.Device != "rt-a01" || f.Severity != "error" {
		t.Errorf("finding = %+v", f)
	}
	before := o.Findings()
	if h := read(t, e, o.Report.HTML); !strings.Contains(h, f.Key) || !strings.Contains(h, "未承認（エラー）") {
		t.Error("the report of the import lacks the finding")
	}
	// An error of the mapping is approved one by one.
	if _, err := e.Approve(ApproveRequest{Rule: "M-406", Devices: "*", Reason: "x", By: "試験"}, at); err == nil || !strings.Contains(err.Error(), "1 件ずつ") {
		t.Errorf("a scope for M-406: %v", err)
	}
	a, err := e.Approve(ApproveRequest{Finding: f.Key, Reason: "新機器でも同じ名前", By: "試験"}, at)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Migration || a.Hits != 1 {
		t.Errorf("approved = %+v", a)
	}
	if o, err = e.RunImport("old", "", at); err != nil || o.Mig.Open() != 0 || o.Mig.Count().Approved != 1 || o.Findings() != before-1 {
		t.Errorf("import after the approval: %v, %+v, findings %d (before %d)", err, o.Mig.Count(), o.Findings(), before)
	}
	b, err := e.MakeBook("old", false, at)
	if err != nil {
		t.Fatal(err)
	}
	if b.Mig.Open() != 0 || b.Mig.Count().Approved != 1 {
		t.Errorf("book: %+v", b.Mig.Count())
	}
	m, err := e.Mapping()
	if err != nil {
		t.Fatal(err)
	}
	s := e.MappingSheet(m.Applied, m.Mig, false)
	if len(s.Rows) != 1 || s.Rows[0][0] != "M-406" || s.Rows[0][7] != "承認済み" || s.Rows[0][9] != f.Key {
		t.Errorf("the sheet of the mapping = %v", s.Rows)
	}

	// The results of the generation tell that the approval is of a migration.
	g, err := e.Generate(GenOptions{Dry: true}, at)
	if err != nil {
		t.Fatal(err)
	}
	if g.Hits[a.Approval.ID] != 0 || !MigrationApproval(g.Approvals[0]) {
		t.Errorf("the generation counts the approval: %v", g.Hits)
	}
}

// The book is compared as an imported configuration is (spec 12.3): its rows
// for "*" are given to the devices, and its empty columns hold the defaults
// of the pack of the device.
func TestCompareBookFilled(t *testing.T) {
	e := newCase(t, map[string]string{"input/old/rt-a01.txt": sample(t, "cisco_ios")})
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	setCell(t, e, "機器対応", "旧ホスト名", "rt-a01", "新ホスト名", "rt-a01")
	setCell(t, e, "機器対応", "旧ホスト名", "rt-a01", "新メーカー", "cisco_ios")
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	_, bk, err := e.readBook()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range bk.Ifs {
		setCell(t, e, "IF 対応", "旧 IF 名", m.OldIf, "新 IF 名", m.OldIf)
	}
	if _, err := e.MakeBook("old", false, at); err != nil {
		t.Fatal(err)
	}
	c, err := e.Compare("old", "book", at)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Result.Equal() {
		t.Fatalf("the book made from the old side differs: only A %d, only B %d, cells %+v", len(c.Result.OnlyA), len(c.Result.OnlyB), c.Result.Cells)
	}

	// A cell that is left empty stands for the default of the pack: the
	// interface works unless it is shut down.
	setCell(t, e, "インタフェース", "IF 名", "Loopback0", "有効", "")
	if c, err = e.Compare("old", "book", at); err != nil || !c.Result.Equal() {
		t.Fatalf("an empty cell with the default of the pack: %v, cells %+v", err, c.Result.Cells)
	}
	// A row for every device is a row of each device.
	addRow(t, e, "NTP サーバ", map[string]string{"機器": "*", "サーバ": "192.0.2.125"})
	if c, err = e.Compare("old", "book", at); err != nil {
		t.Fatal(err)
	}
	r := c.Result
	if len(r.OnlyA) != 0 || len(r.Cells) != 0 || len(r.OnlyB) != 1 || r.OnlyB[0].Device != "rt-a01" || r.OnlyB[0].Row.Get("server") != "192.0.2.125" {
		t.Fatalf("a row for every device: only A %d, only B %+v, cells %d", len(r.OnlyA), r.OnlyB, len(r.Cells))
	}
	if f := c.Mig.Items[len(c.Mig.Items)-1]; f.RuleID != "M-404" || f.Device != "rt-a01" {
		t.Errorf("finding = %+v", f)
	}
	h := read(t, e, c.Report.HTML)
	for _, want := range []string{"機器ごとの行に広げ", "192.0.2.125"} {
		if !strings.Contains(h, want) {
			t.Errorf("the report lacks %q", want)
		}
	}

	// A value that differs from the default is told with where each side has
	// it from.
	setCell(t, e, "インタフェース", "IF 名", "GigabitEthernet0/4", "有効", "")
	if c, err = e.Compare("old", "book", at); err != nil {
		t.Fatal(err)
	}
	if len(c.Result.Cells) != 1 || c.Result.Cells[0].Column != "enabled" || c.Result.Cells[0].A.Text() != "false" || c.Result.Cells[0].B.Text() != "true" {
		t.Fatalf("an interface that was shut down: %+v", c.Result.Cells)
	}
	if h := read(t, e, c.Report.HTML); !strings.Contains(h, "B は既定値（書かれていない値）") {
		t.Error("the report does not tell that the value is a default")
	}
}
