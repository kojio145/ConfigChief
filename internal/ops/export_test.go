package ops

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// exportCase is a case whose book holds the two sample site routers, under
// their old names and interfaces.
func exportCase(t *testing.T) *Env {
	t.Helper()
	e := newCase(t, map[string]string{
		"input/old/rt-s01.txt": siteFile(t, "site", "rt-s01.txt"),
		"input/old/rt-w01.txt": siteFile(t, "vpn", "rt-w01.txt"),
	})
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	for _, h := range []string{"rt-s01", "rt-w01"} {
		setCell(t, e, "機器対応", "旧ホスト名", h, "新ホスト名", h)
	}
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"GigabitEthernet0/0", "GigabitEthernet0/1", "Loopback0", "Dialer1", "Tunnel1"} {
		setCell(t, e, "IF 対応", "旧 IF 名", name, "新 IF 名", name)
	}
	// The second device has the two Ethernet ports as well: setCell writes the
	// first row it finds, so the rest is filled here.
	f, err := excelize.OpenFile(e.Proj.BookPath())
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := f.GetRows("IF 対応")
	oldCol, newCol := -1, -1
	for i, h := range rows[0] {
		switch h {
		case "旧 IF 名":
			oldCol = i
		case "新 IF 名":
			newCol = i
		}
	}
	for i, row := range rows[1:] {
		if oldCol < len(row) && (newCol >= len(row) || row[newCol] == "") {
			cell, _ := excelize.CoordinatesToCellName(newCol+1, i+2)
			f.SetCellStr("IF 対応", cell, row[oldCol])
		}
	}
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	b, err := e.MakeBook("old", false, at)
	if err != nil {
		t.Fatal(err)
	}
	if a := b.Applied; len(a.Undecided)+len(a.Dropped)+len(a.Clashes)+len(b.Problems) != 0 {
		t.Fatalf("the mapping: %+v, problems %v", a, b.Problems)
	}
	return e
}

// The table for the other side: the neighbours that are not of the case, a
// list of all and a sheet for each, the secrets hidden on demand.
func TestExportPeer(t *testing.T) {
	e := exportCase(t)
	o, err := e.ExportPeer(false, at)
	if err != nil {
		t.Fatal(err)
	}
	// BGP and OSPF of rt-s01, IPsec of rt-w01.
	if o.Peers != 3 || o.Items == 0 {
		t.Fatalf("peers %d, items %d", o.Peers, o.Items)
	}
	f, err := excelize.OpenFile(e.Proj.Path(o.Files.XLSX))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got := strings.Join(f.GetSheetList(), ","); got != "まとめ,一覧,相手 203.0.113.50,相手 203.0.113.48_30,相手 203.0.113.20" {
		t.Errorf("sheets: %s", got)
	}
	rows, _ := f.GetRows("一覧")
	all := ""
	for _, r := range rows {
		all += strings.Join(r, "|") + "\n"
	}
	for _, want := range []string{
		// The other side writes this side's address and AS number.
		"203.0.113.50|BGP|ネイバーアドレス|rt-s01|203.0.113.50|203.0.113.49|",
		"203.0.113.50|BGP|こちらの AS 番号|rt-s01|64500|64500|",
		"203.0.113.50|BGP|パスワード|rt-s01|dummy-bgp-key-s|dummy-bgp-key-s",
		"203.0.113.48/30|OSPF（GigabitEthernet0/0）|Hello 間隔（秒）|rt-s01|5|5",
		// The address of this side comes from the line: it is not known.
		"203.0.113.20|IPsec（Tunnel1）|相手アドレス|rt-w01|203.0.113.20||こちらのアドレスが、設計データからは決まりません",
		"203.0.113.20|IPsec（Tunnel1）|事前共有鍵|rt-w01|dummy-psk-site-w|dummy-psk-site-w",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("the list lacks %q:\n%s", want, all)
		}
	}
	html := read(t, e, o.Files.HTML)
	if !strings.Contains(html, "dummy-psk-site-w") {
		t.Errorf("the HTML form lacks the key")
	}

	// With the secrets hidden, no secret value is in either form.
	later := at.Add(1e9)
	if o, err = e.ExportPeer(true, later); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(e.Proj.Path(o.Files.XLSX))
	g, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	text := read(t, e, o.Files.HTML)
	for _, s := range g.GetSheetList() {
		rows, _ := g.GetRows(s)
		for _, r := range rows {
			text += strings.Join(r, "|") + "\n"
		}
	}
	for _, secret := range []string{"dummy-psk-site-w", "dummy-bgp-key-s"} {
		if strings.Contains(text, secret) {
			t.Errorf("%s is in the masked table", secret)
		}
	}
	if !strings.Contains(text, "事前共有鍵|rt-w01|***|***") {
		t.Errorf("the masked key:\n%s", text)
	}
}

// The delivery sheets: the standard form without a template; with templates,
// one output for each, and a template with a fault gives none.
func TestExportSheets(t *testing.T) {
	e := exportCase(t)
	o, err := e.ExportSheets(false, at)
	if err != nil {
		t.Fatal(err)
	}
	if !o.Standard || len(o.Files) != 1 || !strings.HasSuffix(o.Files[0], "/sheets/parameter_sheet.xlsx") || len(o.Problems) != 0 {
		t.Fatalf("the standard form: %+v", o)
	}

	tpl := func(name string, cells map[string]string) {
		f := excelize.NewFile()
		defer f.Close()
		for at, v := range cells {
			f.SetCellStr("Sheet1", at, v)
		}
		if err := os.MkdirAll(e.Proj.Path("templates"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := f.SaveAs(e.Proj.Path("templates/" + name)); err != nil {
			t.Fatal(err)
		}
	}
	tpl("{{ device.hostname }}_sheet.xlsx", map[string]string{"A1": "{{ device.hostname }}", "A2": "{{#rows pppoe}}{{ row.username }}", "B2": "{{ row.password }}"})
	tpl("list.xlsx", map[string]string{"A1": "{{#rows devices}}{{ row.hostname }}", "B1": "{{ row.vendor }}"})
	tpl("wrong.xlsx", map[string]string{"A1": "{{ row.hostname }}", "A2": "{{#rows interfaces}}{{ row.nothing }}"})
	os.WriteFile(e.Proj.Path("templates/~$list.xlsx"), []byte("x"), 0o644)
	os.WriteFile(e.Proj.Path("templates/memo.txt"), []byte("x"), 0o644)

	later := at.Add(1e9)
	if o, err = e.ExportSheets(true, later); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range o.Files {
		names = append(names, f[strings.LastIndex(f, "/")+1:])
	}
	if got := strings.Join(names, ","); o.Standard || got != "list.xlsx,rt-s01_sheet.xlsx,rt-w01_sheet.xlsx" {
		t.Fatalf("files: %s (standard %v)", got, o.Standard)
	}
	if all := strings.Join(o.Problems, "\n"); len(o.Problems) < 2 || !strings.Contains(all, "templates/wrong.xlsx: シート「Sheet1」の A1") || !strings.Contains(all, "列 nothing はありません") {
		t.Errorf("problems:\n%s", all)
	}
	f, err := excelize.OpenFile(e.Proj.Path(o.Dir + "/rt-w01_sheet.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	a, _ := f.GetCellValue("Sheet1", "A2")
	b, _ := f.GetCellValue("Sheet1", "B2")
	if a != "sample-w@example.com" || b != "***" {
		t.Errorf("the PPPoE row of rt-w01: %q, %q", a, b)
	}
}
