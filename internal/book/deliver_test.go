package book

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/kojio145/configchief/internal/model"
)

// sheetsData is two devices as the book spreads them out.
func sheetsData() *model.Dataset {
	d := model.NewDataset("book")
	put(d, "devices", kv("hostname", "rt-a01"), "vendor", "cisco_ios", "site", "sample-site-a", "enable_secret", "dummy-enable-a")
	put(d, "devices", kv("hostname", "rt-a02"), "vendor", "cisco_ios", "site", "sample-site-b")
	put(d, "interfaces", kv("device", "rt-a01", "name", "GigabitEthernet0/10"), "enabled", "true", "ipv4", "198.51.100.1/24", "mtu", "1500")
	put(d, "interfaces", kv("device", "rt-a01", "name", "GigabitEthernet0/2"), "enabled", "false", "ipv4_secondary", "192.0.2.1/26|192.0.2.65/26", "description", "0010")
	put(d, "interfaces", kv("device", "rt-a02", "name", "GigabitEthernet0/0"), "enabled", "true", "ipv4", "203.0.113.1/30")
	put(d, "static_routes", kv("device", "rt-a02", "vrf", "", "prefix", "0.0.0.0/0", "next_hop", "203.0.113.2", "out_if", ""))
	return d
}

// sheetsTemplate is a template with a list of every device and a sheet for
// each device.
func sheetsTemplate(t *testing.T, edit func(f *excelize.File)) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	f.SetSheetName("Sheet1", "一覧")
	set := func(sheet, cell, v string) {
		if err := f.SetCellStr(sheet, cell, v); err != nil {
			t.Fatal(err)
		}
	}
	set("一覧", "A1", "案件: {{ project.name }}")
	set("一覧", "A3", "機器")
	set("一覧", "B3", "IF")
	set("一覧", "A4", "{{#rows interfaces}}{{ row.device }}")
	set("一覧", "B4", "{{ row.name }}（{{ device.site }}）")
	set("一覧", "A5", "以上")
	f.NewSheet("{{ device.hostname }}")
	const dev = "{{ device.hostname }}"
	set(dev, "A1", "ホスト名")
	set(dev, "B1", "{{ device.hostname }}")
	set(dev, "A2", "特権パスワード")
	set(dev, "B2", "{{ device.enable_secret }}")
	set(dev, "A4", "{{#rows インタフェース}}{{ row.name }}")
	set(dev, "B4", "{{ row.ipv4 }}")
	set(dev, "C4", "{{ row.ipv4_secondary }}")
	set(dev, "D4", "{{ row.enabled }}")
	set(dev, "E4", "{{ row.mtu }}")
	set(dev, "F4", "{{ row.description }}")
	set(dev, "A5", "経路")
	set(dev, "A6", "{{ #rows static_routes }}{{ row.prefix }}")
	set(dev, "B6", "{{ row.next_hop }}")
	set(dev, "A7", "末尾")
	if edit != nil {
		edit(f)
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func openSheets(t *testing.T, data []byte) *excelize.File {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func cell(t *testing.T, f *excelize.File, sheet, at string) string {
	t.Helper()
	v, err := f.GetCellValue(sheet, at)
	if err != nil {
		t.Fatalf("%s!%s: %v", sheet, at, err)
	}
	return v
}

// One book for the case: a list of every device and a sheet for each one.
func TestDeliverBook(t *testing.T) {
	sc := defs(t)
	out, problems := Deliver("sheet.xlsx", sheetsTemplate(t, nil), sc, sheetsData(), DeliverOptions{Project: "sample-case", Lang: "ja"})
	if len(problems) != 0 || len(out) != 1 || out[0].Name != "sheet.xlsx" {
		t.Fatalf("out %d, problems %v", len(out), problems)
	}
	f := openSheets(t, out[0].Data)
	if got := strings.Join(f.GetSheetList(), ","); got != "一覧,rt-a01,rt-a02" {
		t.Fatalf("sheets: %s", got)
	}
	for at, want := range map[string]string{
		"A1": "案件: sample-case",
		// The devices in the order of the book, the rows in the natural order of the keys.
		"A4": "rt-a01", "B4": "GigabitEthernet0/2（sample-site-a）",
		"A5": "rt-a01", "B5": "GigabitEthernet0/10（sample-site-a）",
		"A6": "rt-a02", "B6": "GigabitEthernet0/0（sample-site-b）",
		"A7": "以上",
	} {
		if got := cell(t, f, "一覧", at); got != want {
			t.Errorf("一覧!%s = %q, want %q", at, got, want)
		}
	}
	for at, want := range map[string]string{
		"B1": "rt-a01", "B2": "dummy-enable-a",
		"A4": "GigabitEthernet0/2", "B4": "", "C4": "192.0.2.1/26, 192.0.2.65/26", "D4": "無効", "F4": "0010",
		"A5": "GigabitEthernet0/10", "B5": "198.51.100.1/24", "D5": "有効", "E5": "1500",
		// No route on this device: the row is gone, and what stood below moved up.
		"A6": "経路", "A7": "末尾",
	} {
		if got := cell(t, f, "rt-a01", at); got != want {
			t.Errorf("rt-a01!%s = %q, want %q", at, got, want)
		}
	}
	// A number is a number, a value with a leading zero stays a text.
	if k, _ := f.GetCellType("rt-a01", "E5"); k == excelize.CellTypeSharedString || k == excelize.CellTypeInlineString {
		t.Errorf("the MTU is a text (%v)", k)
	}
	if k, _ := f.GetCellType("rt-a01", "F4"); k != excelize.CellTypeSharedString && k != excelize.CellTypeInlineString {
		t.Errorf("0010 is no text (%v)", k)
	}
	for at, want := range map[string]string{"B1": "rt-a02", "B2": "", "A4": "GigabitEthernet0/0", "A6": "0.0.0.0/0", "B6": "203.0.113.2", "A7": "末尾"} {
		if got := cell(t, f, "rt-a02", at); got != want {
			t.Errorf("rt-a02!%s = %q, want %q", at, got, want)
		}
	}
}

// A file for each device, and the secret values hidden.
func TestDeliverFiles(t *testing.T) {
	sc := defs(t)
	out, problems := Deliver("{{ device.hostname }}_sheet.xlsx", sheetsTemplate(t, nil), sc, sheetsData(), DeliverOptions{Mask: true, Lang: "ja"})
	if len(problems) != 0 || len(out) != 2 || out[0].Name != "rt-a01_sheet.xlsx" || out[1].Name != "rt-a02_sheet.xlsx" {
		t.Fatalf("out %d, problems %v", len(out), problems)
	}
	f := openSheets(t, out[0].Data)
	if got := strings.Join(f.GetSheetList(), ","); got != "一覧,rt-a01" {
		t.Fatalf("sheets: %s", got)
	}
	// Every sheet of the file holds this device only.
	if a, b := cell(t, f, "一覧", "A5"), cell(t, f, "一覧", "A6"); a != "rt-a01" || b != "以上" {
		t.Errorf("the list of the file of rt-a01: %q, %q", a, b)
	}
	if got := cell(t, f, "rt-a01", "B2"); got != "***" {
		t.Errorf("the secret: %q", got)
	}
}

// What is wrong in a template is told with its place, and no file is made.
func TestDeliverProblems(t *testing.T) {
	sc := defs(t)
	tpl := sheetsTemplate(t, func(f *excelize.File) {
		f.SetCellStr("一覧", "C1", "{{ device.hostname }}")             // no device here
		f.SetCellStr("一覧", "C4", "{{ row.nothing }}")                 // no such column
		f.SetCellStr("一覧", "A9", "{{#rows nothing}}{{ row.name }}")   // no such table
		f.SetCellStr("一覧", "A10", "{{ row.name")                      // not closed
		f.SetCellStr("一覧", "A11", "{{#rows users}}{{#rows devices}}") // two in a row
	})
	out, problems := Deliver("sheet.xlsx", tpl, sc, sheetsData(), DeliverOptions{Lang: "ja"})
	if out != nil {
		t.Errorf("a file was made")
	}
	all := strings.Join(problems, "\n")
	for _, want := range []string{"「一覧」の C1: device は", "「一覧」の C4: 表 interfaces に列 nothing はありません", "{{#rows nothing}} の表がありません", "「一覧」の A", "{{#rows}} が 2 つ以上"} {
		if !strings.Contains(all, want) {
			t.Errorf("no %q in:\n%s", want, all)
		}
	}
	// A sheet name that Excel does not take.
	d := sheetsData()
	put(d, "devices", kv("hostname", "rt-with-a-very-long-name-for-a-sheet-a03"), "vendor", "cisco_ios")
	if _, problems = Deliver("sheet.xlsx", sheetsTemplate(t, nil), sc, d, DeliverOptions{Lang: "ja"}); len(problems) != 1 || !strings.Contains(problems[0], "シートの名前に使えません") {
		t.Errorf("the long sheet name: %v", problems)
	}
}

// Without a template: the sheets of the book, with the secrets hidden on demand.
func TestDeliverStandard(t *testing.T) {
	sc := defs(t)
	data, err := Standard(sc, sheetsData(), DeliverOptions{Mask: true, Lang: "ja"})
	if err != nil {
		t.Fatal(err)
	}
	f := openSheets(t, data)
	if got := strings.Join(f.GetSheetList(), ","); got != "機器,インタフェース,スタティック経路" {
		t.Fatalf("sheets: %s", got)
	}
	rows, _ := f.GetRows("機器")
	if len(rows) != 3 || !strings.Contains(strings.Join(rows[1], "|"), "***") || strings.Contains(strings.Join(rows[1], "|"), "dummy-enable-a") {
		t.Errorf("the devices: %v", rows)
	}
}
