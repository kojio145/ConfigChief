package importer

import (
	"os"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/pack"
)

// The typed pack tests what the table definitions do to an import.
func loadTyped(t *testing.T) *Result {
	t.Helper()
	p, err := pack.Load(os.DirFS("testdata/typed"))
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	text, err := os.ReadFile("testdata/typed/tests/import/basic/input.txt")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Import(p, defs(t), "basic.txt", string(text))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	return res
}

// Values go in in the one way of writing of their column, and a key is made
// whole in the order of the definition.
func TestTypedValues(t *testing.T) {
	res := loadTyped(t)
	gi1 := "device=rt-d01,name=GigabitEthernet0/1"
	want := []struct{ table, key, col, val string }{
		{"devices", "hostname=rt-d01", "vendor", "typed"},
		{"interfaces", gi1, "mtu", "1500"},
		{"interfaces", gi1, "speed", "auto"},
		{"interfaces", gi1, "ipv4", "192.0.2.1/30"},
		{"interfaces", gi1, "proxy_arp", "true"},
		{"interfaces", gi1, "dhcp_relay", "192.0.2.67,192.0.2.68"},
		{"interfaces", gi1, "kind", "physical"},
		{"static_routes", "device=rt-d01,vrf=,prefix=198.51.100.0/24,next_hop=192.0.2.2,out_if=", "distance", "10"},
		{"bgp_neighbors", "device=rt-d01,vrf=,neighbor=192.0.2.9", "remote_as", "64500"},
	}
	for _, w := range want {
		if got := cell(t, res, w.table, w.key, w.col); got != w.val {
			t.Errorf("%s[%s].%s = %q, want %q", w.table, w.key, w.col, got, w.val)
		}
	}
}

// A line with a value that does not fit stays unknown and says why; nothing
// of it is in the data.
func TestTypedUnknown(t *testing.T) {
	res := loadTyped(t)
	want := []struct{ text, reason string }{
		{"mtu big", "interfaces の mtu: 「big」は整数ではありません"},
		{"speed 2500", "「2500」は選択肢（auto, 10, 100, 1000, 10000）にありません"},
		{"address 192.0.2.300/24", "アドレス/長さ"},
		{"proxy maybe", "有効・無効として読めません"},
		{"relay 192.0.2.67 not-an-address", "「not-an-address」は IPv4 アドレスではありません"},
		{"colour red", "表 interfaces に列 colour はありません"},
		{"relay-one 192.0.2.67", "append で入れます"},
		{"mtu-more 1400", "set で入れます"},
		{"rename Gi0/9", "キーの列です"},
		{"route 198.51.100.1/24 192.0.2.2", "ネットワークのアドレスではありません（198.51.100.0/24 と書きます）"},
		{"route 203.0.113.0/24 192.0.2.2 300", "範囲（1〜255）の外"},
		{"route-by 203.0.113.0/24 5", "distance は表 static_routes のキーの列ではありません"},
		{"shelf top", "表 shelves は設計データにありません"},
		{"peer 192.0.2.10 as none", "まとめる規則 peer.row が行を作れませんでした: bgp_neighbors の remote_as: 「none」は整数ではありません"},
	}
	for _, w := range want {
		l := line(t, res, w.text)
		if l.State != Unknown || !strings.Contains(l.Reason, w.reason) {
			t.Errorf("%q: %s, reason %q; want unknown with %q", w.text, l.State, l.Reason, w.reason)
		}
		if !strings.Contains(l.Reason, "規則 ") {
			t.Errorf("%q: the reason %q does not name the rule", w.text, l.Reason)
		}
	}
	for _, text := range []string{"mtu 01500", "peer 192.0.2.9 as 64500", "interface Gi0/2"} {
		if l := line(t, res, text); l.State != Taken {
			t.Errorf("%q: %s (%s), want taken", text, l.State, l.Reason)
		}
	}

	// The second interface is there, with nothing from the lines that failed.
	gi2 := "device=rt-d01,name=GigabitEthernet0/2"
	for _, col := range []string{"mtu", "speed", "ipv4", "proxy_arp", "dhcp_relay", "colour"} {
		if got := cell(t, res, "interfaces", gi2, col); got != "" {
			t.Errorf("interfaces[%s].%s = %q, want empty", gi2, col, got)
		}
	}
	if n := len(res.Data.Tables["static_routes"].Rows); n != 1 {
		t.Errorf("static_routes: %d rows, want 1", n)
	}
	if n := len(res.Data.Tables["bgp_neighbors"].Rows); n != 1 {
		t.Errorf("bgp_neighbors: %d rows, want 1", n)
	}
	if res.Data.Tables["shelves"] != nil {
		t.Error("a table that is not in the definitions was made")
	}
}

// A text without a hostname line is still one device: named after the file,
// with the pack that read it.
func TestVendorWithoutHostname(t *testing.T) {
	p, err := pack.Load(os.DirFS("testdata/typed"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Import(p, defs(t), "rt-d09.txt", "shelf top\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := cell(t, res, "devices", "hostname=rt-d09", "vendor"); got != "typed" {
		t.Errorf("vendor = %q", got)
	}
}

// Without table definitions nothing is checked.
func TestNoDefinitions(t *testing.T) {
	p, err := pack.Load(os.DirFS("testdata/typed"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Import(p, nil, "rt-d09.txt", "shelf top\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := cell(t, res, "shelves", "device=rt-d09,name=top", "name"); got != "top" {
		t.Errorf("name = %q", got)
	}
}
