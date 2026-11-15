package book

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/schemas"
)

func defs(t *testing.T) *schema.Set {
	t.Helper()
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

var imported = model.Origin{Kind: "config", File: "input/old/rt-a01.txt", Line: 1}

// put adds a row; cells are "column", "value" pairs, and a value for a list
// column is written with "|" between the items.
func put(d *model.Dataset, table string, key []model.KV, cells ...string) *model.Row {
	r := d.Open(table, key, imported)
	for i := 0; i < len(cells); i += 2 {
		if strings.Contains(cells[i+1], "|") {
			for _, v := range strings.Split(cells[i+1], "|") {
				r.Append(cells[i], v, imported)
			}
		} else {
			r.Set(cells[i], cells[i+1], imported)
		}
	}
	return r
}

func kv(pairs ...string) []model.KV {
	var out []model.KV
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, model.KV{Name: pairs[i], Value: pairs[i+1]})
	}
	return out
}

func sampleData() *model.Dataset {
	d := model.NewDataset("old")
	put(d, "devices", kv("hostname", "rt-a01"), "vendor", "cisco_ios", "name_servers", "192.0.2.53|198.51.100.53", "banner", "sample line 1\nsample line 2")
	put(d, "interfaces", kv("device", "rt-a01", "name", "GigabitEthernet0/0"), "enabled", "true", "ipv4", "192.0.2.1/30", "description", "to sample-wan", "mtu", "1500")
	put(d, "interfaces", kv("device", "rt-a01", "name", "GigabitEthernet0/1"), "enabled", "false")
	put(d, "ntp_servers", kv("device", "rt-a01", "server", "192.0.2.123"), "prefer", "true", "source_if", "GigabitEthernet0/0")
	return d
}

// flat gives a dataset as table → key → column → text, for comparing.
func flat(d *model.Dataset) map[string]map[string]map[string]string {
	out := map[string]map[string]map[string]string{}
	for name, t := range d.Tables {
		if len(t.Rows) == 0 {
			continue
		}
		out[name] = map[string]map[string]string{}
		for _, r := range t.Rows {
			cells := map[string]string{}
			for col, c := range r.Cells {
				cells[col] = strings.Join(c.Values, "|")
			}
			out[name][r.KeyText()] = cells
		}
	}
	return out
}

func mustUpdate(t *testing.T, old []byte, sc *schema.Set, lang string, ch Change) ([]byte, *Summary) {
	t.Helper()
	data, sum, err := Update(old, "parameters.xlsx", sc, lang, ch)
	if err != nil {
		t.Fatal(err)
	}
	return data, sum
}

func mustRead(t *testing.T, data []byte, sc *schema.Set) *Book {
	t.Helper()
	b, err := Read(data, "parameters.xlsx", sc)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func sheetNames(t *testing.T, data []byte) []string {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	return f.GetSheetList()
}

// edit changes a workbook the way a person would in Excel.
func edit(t *testing.T, data []byte, fn func(f *excelize.File)) []byte {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fn(f)
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestNewBook(t *testing.T) {
	sc := defs(t)
	for _, lang := range []string{"ja", "en"} {
		ch := Change{
			Data:    sampleData(),
			Devices: []DeviceMap{{OldHost: "rt-a01", OldVendor: "cisco_ios", NewHost: "rt-a01", NewVendor: "nec_ix"}, {OldHost: "rt-a01"}},
			Ifs:     []IfMap{{NewHost: "rt-a01", OldIf: "GigabitEthernet0/0", NewIf: "GigaEthernet0.0", Basis: "sample basis"}, {NewHost: "rt-a01", OldIf: "GigabitEthernet0/1"}},
		}
		data, sum := mustUpdate(t, nil, sc, lang, ch)
		if !sum.Created || sum.Rows() != 4 || sum.DeviceRows != 1 || sum.IfRows != 2 {
			t.Errorf("%s: summary = %+v", lang, sum)
		}
		names := sheetNames(t, data)
		if len(names) != len(sc.Tables)+3 {
			t.Errorf("%s: %d sheets, want %d", lang, len(names), len(sc.Tables)+3)
		}
		if lang == "ja" && (names[0] != "機器" || names[1] != "インタフェース" || names[len(names)-3] != "機器対応" || names[len(names)-1] != "共通値") {
			t.Errorf("sheets = %v", names)
		}
		if lang == "en" && (names[0] != "Devices" || names[len(names)-2] != "Interface map") {
			t.Errorf("sheets = %v", names)
		}
		b := mustRead(t, data, sc)
		if len(b.Problems) != 0 {
			t.Errorf("%s: problems = %v", lang, b.Problems)
		}
		if got, want := flat(b.Data), flat(sampleData()); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: read back\n got  %v\n want %v", lang, got, want)
		}
		if len(b.Devices) != 1 || b.Devices[0].NewVendor != "nec_ix" || b.Devices[0].Row != 2 {
			t.Errorf("%s: devices = %+v", lang, b.Devices)
		}
		if len(b.Ifs) != 2 || b.Ifs[0].NewIf != "GigaEthernet0.0" || b.Ifs[0].Basis != "sample basis" || b.Ifs[1].NewIf != "" || b.Ifs[1].Row != 3 {
			t.Errorf("%s: ifs = %+v", lang, b.Ifs)
		}
		if len(b.Memo) != 0 {
			t.Errorf("%s: memo = %v", lang, b.Memo)
		}
		// The origin of a value is its cell.
		r := b.Data.Tables["interfaces"].Rows[0]
		ipv4 := sc.Table("interfaces").Column("ipv4")
		o := r.Cells["ipv4"].Origins[0]
		if o.Kind != "book" || o.File != "parameters.xlsx" || o.Sheet != sc.Table("interfaces").Title.In(lang) || !strings.HasSuffix(o.Addr, "2") {
			t.Errorf("%s: origin of %s = %+v", lang, ipv4.Name, o)
		}
	}
}

func TestNewBookIsDeterministic(t *testing.T) {
	sc := defs(t)
	a, _ := mustUpdate(t, nil, sc, "ja", Change{Data: sampleData()})
	b, _ := mustUpdate(t, nil, sc, "ja", Change{Data: sampleData()})
	if !bytes.Equal(a, b) {
		t.Error("the same data gave different bytes")
	}
}

func TestShow(t *testing.T) {
	sc := defs(t)
	enabled := sc.Table("interfaces").Column("enabled")
	servers := sc.Table("devices").Column("name_servers")
	cases := []struct {
		col  *schema.Column
		cell *model.Cell
		lang string
		want string
	}{
		{enabled, &model.Cell{Values: []string{"true"}}, "ja", "有効"},
		{enabled, &model.Cell{Values: []string{"false"}}, "ja", "無効"},
		{enabled, &model.Cell{Values: []string{"true"}}, "en", "yes"},
		{servers, &model.Cell{Values: []string{"192.0.2.53", "198.51.100.53"}, List: true}, "ja", "192.0.2.53, 198.51.100.53"},
		{servers, nil, "ja", ""},
	}
	for _, c := range cases {
		if got := Show(c.col, c.cell, c.lang); got != c.want {
			t.Errorf("Show(%s, %v, %s) = %q, want %q", c.col.Name, c.cell, c.lang, got, c.want)
		}
	}
}

func TestUpdateKeepsWhatAPersonWrote(t *testing.T) {
	sc := defs(t)
	first, _ := mustUpdate(t, nil, sc, "ja", Change{
		Data:    sampleData(),
		Devices: []DeviceMap{{OldHost: "rt-a01", OldVendor: "cisco_ios"}},
	})
	// A person fills the mapping, changes a value, adds a column of their
	// own and a sheet of notes, and deletes the MTU column.
	edited := edit(t, first, func(f *excelize.File) {
		f.SetCellStr("機器対応", "C2", "rt-b01")
		f.SetCellStr("インタフェース", "D2", "to sample-core")
		mtu, _ := excelize.ColumnNumberToName(indexOf(t, sc.Table("interfaces"), "mtu") + 1)
		f.RemoveCol("インタフェース", mtu)
		f.SetCellStr("NTP サーバ", "H1", "担当")
		f.SetCellStr("NTP サーバ", "H2", "sample-team")
		f.NewSheet("メモ")
		f.SetCellStr("メモ", "A1", "sample memo")
	})

	more := sampleData()
	put(more, "interfaces", kv("device", "rt-a01", "name", "Loopback0"), "enabled", "true", "ipv4", "192.0.2.255/32", "mtu", "1400")
	put(more, "interfaces", kv("device", "rt-a01", "name", "GigabitEthernet0/0"), "description", "changed by the tool")
	second, sum := mustUpdate(t, edited, sc, "ja", Change{
		Data:    more,
		Devices: []DeviceMap{{OldHost: "rt-a01", NewHost: "rt-z99"}, {OldHost: "rt-a02", OldVendor: "nec_ix"}},
		Ifs:     []IfMap{{NewHost: "rt-b01", OldIf: "GigabitEthernet0/0"}},
	})
	if sum.Created || sum.Added["interfaces"] != 1 || sum.Kept["interfaces"] != 2 || sum.Kept["devices"] != 1 || sum.DeviceRows != 1 || sum.IfRows != 1 {
		t.Errorf("summary = %+v", sum)
	}
	b := mustRead(t, second, sc)
	if len(b.Problems) != 0 {
		t.Errorf("problems = %v", b.Problems)
	}
	ifs := b.Data.Tables["interfaces"]
	if len(ifs.Rows) != 3 {
		t.Fatalf("interfaces rows = %d", len(ifs.Rows))
	}
	if got := ifs.Rows[0].Get("description"); got != "to sample-core" {
		t.Errorf("the value a person wrote became %q", got)
	}
	// The MTU column was gone: it comes back at the right end for the new row.
	if got := ifs.Rows[2].Get("mtu"); got != "1400" {
		t.Errorf("mtu of the new row = %q", got)
	}
	if got := ifs.Rows[0].Get("mtu"); got != "" {
		t.Errorf("mtu of the old row = %q", got)
	}
	if got := ifs.Rows[2].Get("ipv4"); got != "192.0.2.255/32" {
		t.Errorf("ipv4 of the new row = %q", got)
	}
	if len(b.Devices) != 2 || b.Devices[0].NewHost != "rt-b01" || b.Devices[1].OldHost != "rt-a02" {
		t.Errorf("devices = %+v", b.Devices)
	}
	if len(b.Ifs) != 1 || b.Ifs[0].OldIf != "GigabitEthernet0/0" {
		t.Errorf("ifs = %+v", b.Ifs)
	}
	if !reflect.DeepEqual(b.Memo, []string{"メモ"}) {
		t.Errorf("memo = %v", b.Memo)
	}
	ntp := b.Data.Tables["ntp_servers"].Rows[0]
	if len(ntp.Extra) != 1 || ntp.Extra[0] != (model.KV{Name: "担当", Value: "sample-team"}) {
		t.Errorf("free column = %+v", ntp.Extra)
	}
	// Putting the same rows in again adds nothing.
	_, sum = mustUpdate(t, second, sc, "ja", Change{Data: more, Devices: []DeviceMap{{OldHost: "rt-a02"}}})
	if sum.Rows() != 0 || sum.DeviceRows != 0 {
		t.Errorf("second time: %+v", sum)
	}
}

func indexOf(t *testing.T, tb *schema.Table, col string) int {
	t.Helper()
	for i, c := range tb.Columns {
		if c.Name == col {
			return i
		}
	}
	t.Fatalf("no column %s", col)
	return -1
}

func TestReplace(t *testing.T) {
	sc := defs(t)
	first, _ := mustUpdate(t, nil, sc, "ja", Change{
		Data:    sampleData(),
		Devices: []DeviceMap{{OldHost: "rt-a01", NewHost: "rt-b01"}},
	})
	first = edit(t, first, func(f *excelize.File) {
		f.NewSheet("メモ")
		f.SetCellStr("メモ", "A1", "sample memo")
	})
	before := sheetNames(t, first)

	d := model.NewDataset("old")
	put(d, "devices", kv("hostname", "rt-b01"), "vendor", "nec_ix")
	put(d, "interfaces", kv("device", "rt-b01", "name", "GigaEthernet0.0"), "enabled", "true")
	second, sum := mustUpdate(t, first, sc, "ja", Change{Data: d, Replace: true})
	if sum.Added["interfaces"] != 1 || sum.Added["devices"] != 1 || len(sum.Kept) != 0 {
		t.Errorf("summary = %+v", sum)
	}
	if after := sheetNames(t, second); !reflect.DeepEqual(after, before) {
		t.Errorf("sheets\n after  %v\n before %v", after, before)
	}
	b := mustRead(t, second, sc)
	if got, want := flat(b.Data), flat(d); !reflect.DeepEqual(got, want) {
		t.Errorf("data\n got  %v\n want %v", got, want)
	}
	if len(b.Devices) != 1 || b.Devices[0].NewHost != "rt-b01" {
		t.Errorf("the mapping was not kept: %+v", b.Devices)
	}
	if !reflect.DeepEqual(b.Memo, []string{"メモ"}) {
		t.Errorf("memo = %v", b.Memo)
	}
}

func TestReadWhatAPersonTyped(t *testing.T) {
	sc := defs(t)
	grids := []Grid{
		{
			// English and internal names in the header, in an order of
			// their own, with a column the definition does not have.
			Name:   "Interfaces",
			Header: []string{"name", "Device", "有効", "IPv4アドレス", "MTU", "IPv4 セカンダリ", "担当", "MTU"},
			Rows: [][]string{
				{"GigabitEthernet0/0", "rt-a01", "○", " 192.0.2.1/30 ", "01500", "192.0.2.9/30\n192.0.2.13/30", "sample-team"},
				{"GigabitEthernet0/1", "rt-a01", "×", "", "many"},
				{},
				{"GigabitEthernet0/0", "rt-a01", "yes"},
				{"GigabitEthernet0/2", "*", "maybe", "192.0.2.300/30"},
			},
		},
		{Name: "作業メモ", Header: []string{"sample"}, Rows: [][]string{{"x"}}},
		{Name: "IF対応", Header: []string{"旧IF名", "新IF名", "新ホスト名"}, Rows: [][]string{{"GigabitEthernet0/0", "-", "rt-b01"}, {"GigabitEthernet0/1", "", ""}}},
		{Name: "インタフェース", Header: []string{"name"}, Rows: [][]string{{"x"}}},
	}
	var buf bytes.Buffer
	if err := WriteGrids(&buf, grids); err != nil {
		t.Fatal(err)
	}
	b := mustRead(t, buf.Bytes(), sc)

	rows := b.Data.Tables["interfaces"].Rows
	if len(rows) != 3 {
		t.Fatalf("rows = %d: %v, problems %v", len(rows), flat(b.Data)["interfaces"], b.Problems)
	}
	r := rows[0]
	if r.Get("enabled") != "true" || r.Get("ipv4") != "192.0.2.1/30" || r.Get("mtu") != "1500" || r.KeyText() != "device=rt-a01,name=GigabitEthernet0/0" {
		t.Errorf("row 1 = %v %v", r.KeyText(), flat(b.Data)["interfaces"][r.KeyText()])
	}
	if c := r.Cells["ipv4_secondary"]; c == nil || !c.List || !reflect.DeepEqual(c.Values, []string{"192.0.2.9/30", "192.0.2.13/30"}) {
		t.Errorf("list = %+v", c)
	}
	if len(r.Extra) != 1 || r.Extra[0].Name != "担当" {
		t.Errorf("free = %+v", r.Extra)
	}
	if rows[1].Get("enabled") != "false" || rows[1].Get("mtu") != "many" {
		t.Errorf("row 2 = %v", flat(b.Data)["interfaces"][rows[1].KeyText()])
	}
	if rows[2].Get("device") != "*" || rows[2].Get("enabled") != "maybe" {
		t.Errorf("row 5 = %v", flat(b.Data)["interfaces"][rows[2].KeyText()])
	}

	var got []string
	for _, p := range b.Problems {
		got = append(got, p.String())
	}
	want := []string{
		"シート「Interfaces」の H1: 見出し「MTU」が 2 つあります。E1 の列を使います",
		"シート「Interfaces」の E3: 「many」は整数ではありません",
		"シート「Interfaces」の A5: A2 の行とキーが同じです（rt-a01 / GigabitEthernet0/0）。この行は読みません",
		"シート「Interfaces」の C6: 「maybe」は有効・無効として読めません",
		"シート「Interfaces」の D6: 「192.0.2.300/30」は IPv4 の「アドレス/長さ」の形ではありません",
		"シート「IF対応」の A3: 新ホスト名か旧 IF 名が空です。この行は使いません",
		"シート「インタフェース」: シート「Interfaces」と同じ表のシートです。このシートは読みません",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("problems:\n got  %s\n want %s", strings.Join(got, "\n      "), strings.Join(want, "\n      "))
	}
	if len(b.Ifs) != 1 || b.Ifs[0] != (IfMap{NewHost: "rt-b01", OldIf: "GigabitEthernet0/0", NewIf: "-", Row: 2}) {
		t.Errorf("ifs = %+v", b.Ifs)
	}
	if !reflect.DeepEqual(b.Memo, []string{"作業メモ"}) {
		t.Errorf("memo = %v", b.Memo)
	}
	if info := b.Sheets["interfaces"]; info.Name != "Interfaces" || info.Last != 6 || info.Width != 8 || info.Cols["device"] != 1 {
		t.Errorf("info = %+v", info)
	}
}
