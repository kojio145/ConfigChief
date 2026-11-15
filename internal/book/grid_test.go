package book

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/sheet"
	"github.com/xuri/excelize/v2"
)

// The values a parameter book must keep as typed. All fictional.
func sampleGrids() []Grid {
	return []Grid{
		{
			Name:   "インタフェース",
			Header: []string{"機器", "IF 名", "IPv4 アドレス", "有効", "説明", "VLAN", "メモ"},
			Rows: [][]string{
				{"rt-a01", "GigabitEthernet0/0/0", "192.0.2.1/30", "有効", "to rt-b01 上流", "0010", ""},
				{"rt-a01", "GigabitEthernet0/0/1", "198.51.100.1/24", "無効", "", "1/2", "先頭が = の文字"},
				{"rt-a01", "Loopback0", "203.0.113.1/32", "有効", "=SUM(A1:A2)", "1e5", "  前後に空白  "},
				{"rt-b01", "Te1/0/1", "", "", "2 行の\nメモ", "00:11", "TRUE"},
				{},
				{"rt-b01", "Vlan10", "192.0.2.129/25", "有効", `"引用符" と , と <タグ> & 記号`, "-5", "１２３（全角）"},
				{"rt-c01", "Port-channel1", "2001:db8::1/64", "有効", strings.Repeat("長い説明 ", 200), "3.10", "0x1F"},
			},
			Notes:   map[int]string{2: "アドレス/プレフィックス長 で書く（例 192.0.2.1/24）", 3: "有効 か 無効"},
			Choices: map[int][]string{3: {"有効", "無効"}},
		},
		{
			Name:   "機器",
			Header: []string{"ホスト名", "メーカー", "OS の版"},
			Rows: [][]string{
				{"rt-a01", "cisco_ios", "17.9"},
				{"rt-b01", "nec_ix", "10.10"},
			},
			Choices: map[int][]string{1: {"cisco_ios", "nec_ix"}},
		},
		{Name: "空の表", Header: []string{"列 A", "列 B"}},
	}
}

// want is what sheet.Read gives back: header + rows, each row cut after its
// last filled cell, trailing empty rows dropped.
func want(g Grid) [][]string {
	rows := append([][]string{g.Header}, g.Rows...)
	var out [][]string
	for _, r := range rows {
		n := len(r)
		for n > 0 && r[n-1] == "" {
			n--
		}
		out = append(out, append([]string{}, r[:n]...))
	}
	for len(out) > 0 && len(out[len(out)-1]) == 0 {
		out = out[:len(out)-1]
	}
	return out
}

func norm(rows [][]string) [][]string {
	var out [][]string
	for _, r := range rows {
		n := len(r)
		for n > 0 && r[n-1] == "" {
			n--
		}
		out = append(out, append([]string{}, r[:n]...))
	}
	for len(out) > 0 && len(out[len(out)-1]) == 0 {
		out = out[:len(out)-1]
	}
	return out
}

func TestWriteThenRead(t *testing.T) {
	grids := sampleGrids()
	var buf bytes.Buffer
	if err := WriteGrids(&buf, grids); err != nil {
		t.Fatal(err)
	}
	b, err := sheet.Read(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if b.Format != "xlsx" {
		t.Errorf("format = %q", b.Format)
	}
	if len(b.Warnings) > 0 {
		t.Errorf("warnings: %v", b.Warnings)
	}
	if len(b.Sheets) != len(grids) {
		t.Fatalf("sheets = %d, want %d", len(b.Sheets), len(grids))
	}
	for i, g := range grids {
		s := b.Sheets[i]
		if s.Name != g.Name {
			t.Errorf("sheet %d name = %q, want %q", i, s.Name, g.Name)
		}
		got, exp := norm(s.Rows), want(g)
		if len(got) != len(exp) {
			t.Errorf("%s: rows = %d, want %d", g.Name, len(got), len(exp))
			continue
		}
		for r := range exp {
			if !reflect.DeepEqual(got[r], exp[r]) {
				t.Errorf("%s row %d:\n got  %q\n want %q", g.Name, r+1, got[r], exp[r])
			}
		}
	}
}

// The same grids give the same bytes, whenever they are written.
func TestWriteIsDeterministic(t *testing.T) {
	var a, b bytes.Buffer
	if err := WriteGrids(&a, sampleGrids()); err != nil {
		t.Fatal(err)
	}
	if err := WriteGrids(&b, sampleGrids()); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Errorf("two writes differ (%d and %d bytes)", a.Len(), b.Len())
	}
}

// A book the user edited in Excel comes back as numbers and shared strings,
// not as the text cells this tool writes. Simulate that with excelize's typed
// setters and see what the reader makes of it.
func TestReadTypedCells(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	set := func(cell string, v any) {
		if err := f.SetCellValue("Sheet1", cell, v); err != nil {
			t.Fatal(err)
		}
	}
	set("A1", "VLAN")
	set("B1", "有効")
	set("C1", "版")
	set("A2", 10)
	set("B2", true)
	set("C2", 17.9)
	set("A3", 4094)
	set("B3", false)
	set("C3", 15.0)
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	b, err := sheet.Read(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	got := norm(b.Sheets[0].Rows)
	exp := [][]string{{"VLAN", "有効", "版"}, {"10", "TRUE", "17.9"}, {"4094", "FALSE", "15"}}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("got  %q\nwant %q", got, exp)
	}
}

func TestChoicesTooLong(t *testing.T) {
	g := Grid{Name: "x", Header: []string{"a"}, Choices: map[int][]string{0: {strings.Repeat("あ", 300)}}}
	if err := WriteGrids(&bytes.Buffer{}, []Grid{g}); err == nil {
		t.Error("no error for a too long list")
	}
}
