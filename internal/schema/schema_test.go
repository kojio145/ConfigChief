package schema

import (
	"strings"
	"testing"
	"testing/fstest"
)

func load(t *testing.T, files map[string]string) (*Set, error) {
	t.Helper()
	fsys := fstest.MapFS{}
	for name, text := range files {
		fsys[name] = &fstest.MapFile{Data: []byte(text)}
	}
	return Load(fsys)
}

const devices = `table: devices
title: {ja: 機器, en: Devices}
key: [hostname]
columns:
  - {name: hostname, title: {ja: ホスト名, en: Hostname}, type: string, required: true}
`

const routes = `table: routes
title: {ja: 経路, en: Routes}
key: [device, prefix]
columns:
  - {name: device, title: {ja: 機器, en: Device}, type: ref, ref: devices.hostname, required: true}
  - {name: prefix, title: {ja: 宛先, en: Prefix}, type: net4, required: true}
  - {name: next_hop, title: {ja: ネクストホップ, en: Next hop}, type: ip4, also: [dhcp, "if:*"]}
  - {name: distance, title: {ja: 優先度, en: Distance}, type: int, min: 1, max: 255}
  - {name: tags, title: {ja: 目印, en: Tags}, type: "list<enum>", values: [red, blue, "null"]}
  - {name: enabled, title: {ja: 有効, en: Enabled}, type: bool}
  - {name: addr, title: {ja: アドレス, en: Address}, type: cidr4}
  - {name: v6, title: {ja: IPv6, en: IPv6}, type: cidr6}
  - {name: key, title: {ja: 鍵, en: Key}, type: secret}
  - {name: peer, title: {ja: 相手, en: Peer}, type: ip}
  - {name: summary, title: {ja: 集約, en: Summary}, type: net}
`

func TestLoad(t *testing.T) {
	s, err := load(t, map[string]string{"02_routes.yaml": routes, "01_devices.yaml": devices})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Tables) != 2 || s.Tables[0].Name != "devices" || s.Tables[1].Name != "routes" {
		t.Fatalf("tables = %v", s.Tables)
	}
	r := s.Table("routes")
	if r.DeviceColumn() != "device" || s.Table("devices").DeviceColumn() != "hostname" {
		t.Error("device column")
	}
	if c := r.Column("tags"); !c.List || c.Type != "enum" {
		t.Errorf("tags = %+v", c)
	}
	if !r.Column("key").Secret || !r.IsKey("prefix") || r.IsKey("enabled") {
		t.Error("secret or key")
	}
	if s.Table("none") != nil || r.Column("none") != nil {
		t.Error("a name that is not there")
	}
	if got := r.Title.In("en"); got != "Routes" {
		t.Errorf("title = %s", got)
	}
}

func TestNormalize(t *testing.T) {
	s, err := load(t, map[string]string{"02_routes.yaml": routes, "01_devices.yaml": devices})
	if err != nil {
		t.Fatal(err)
	}
	r := s.Table("routes")
	cases := []struct{ col, in, want, bad string }{
		{"distance", " 010 ", "10", ""},
		{"distance", "", "", ""},
		{"distance", "ten", "", "整数"},
		{"distance", "0", "", "範囲（1〜255）"},
		{"distance", "256", "", "範囲"},
		{"enabled", "Yes", "true", ""},
		{"enabled", "無効", "false", ""},
		{"enabled", "○", "true", ""},
		{"enabled", "maybe", "", "有効・無効"},
		{"tags", "RED", "red", ""},
		{"tags", "null", "null", ""},
		{"tags", "green", "", "選択肢（red, blue, null）"},
		{"prefix", "192.0.2.0/24", "192.0.2.0/24", ""},
		{"prefix", "192.0.2.1/24", "", "192.0.2.0/24 と書きます"},
		{"prefix", "192.0.2.0", "", "アドレス/長さ"},
		{"prefix", "2001:db8::/32", "", "IPv4"},
		{"addr", "192.0.2.1/24", "192.0.2.1/24", ""},
		{"v6", "2001:DB8::1/64", "2001:db8::1/64", ""},
		{"v6", "192.0.2.1/24", "", "IPv6"},
		{"next_hop", "192.0.2.1", "192.0.2.1", ""},
		{"next_hop", "192.0.2.300", "", "IPv4 アドレス"},
		{"next_hop", "2001:db8::1", "", "IPv4 アドレス"},
		{"next_hop", "DHCP", "dhcp", ""},
		{"next_hop", "IF:Tunnel1", "if:Tunnel1", ""},
		{"next_hop", "if:", "", "IPv4 アドレス"},
		{"key", " example-key ", " example-key ", ""},
		// Either family: an IPv4 value comes out as an ip4 / net4 column
		// writes it, so widening a key column changes no key.
		{"peer", "192.0.2.1", "192.0.2.1", ""},
		{"peer", "2001:DB8::1", "2001:db8::1", ""},
		{"peer", "192.0.2.300", "", "IPv4 アドレスでも IPv6 アドレスでも"},
		{"peer", "fe80::1%eth0", "", "IPv4 アドレスでも IPv6 アドレスでも"},
		{"peer", "192.0.2.0/24", "", "IPv4 アドレスでも IPv6 アドレスでも"},
		{"summary", "192.0.2.0/24", "192.0.2.0/24", ""},
		{"summary", "2001:DB8:0:10::/64", "2001:db8:0:10::/64", ""},
		{"summary", "192.0.2.1/24", "", "192.0.2.0/24 と書きます"},
		{"summary", "2001:db8::1/64", "", "2001:db8::/64 と書きます"},
		{"summary", "192.0.2.0", "", "アドレス/長さ"},
	}
	// The same value normalizes the same way in an ip4 and in an ip column.
	for _, v := range []string{"192.0.2.1", "192.000.2.1", " 198.51.100.7 ", "0.0.0.0"} {
		a, errA := r.Column("next_hop").Normalize(v)
		b, errB := r.Column("peer").Normalize(v)
		if a != b || (errA == nil) != (errB == nil) {
			t.Errorf("%q: ip4 gives %q (%v), ip gives %q (%v)", v, a, errA, b, errB)
		}
	}
	for _, c := range cases {
		got, err := r.Column(c.col).Normalize(c.in)
		switch {
		case c.bad == "" && err != nil:
			t.Errorf("%s %q: %v", c.col, c.in, err)
		case c.bad == "" && got != c.want:
			t.Errorf("%s %q = %q, want %q", c.col, c.in, got, c.want)
		case c.bad != "" && (err == nil || !strings.Contains(err.Error(), c.bad)):
			t.Errorf("%s %q: error %v, want one with %q", c.col, c.in, err, c.bad)
		}
	}
}

func TestLoadErrors(t *testing.T) {
	head := "table: t\ntitle: {ja: 表, en: T}\nkey: [device]\ncolumns:\n  - {name: device, title: {ja: 機器, en: Device}, type: ref, ref: devices.hostname}\n"
	cases := []struct{ name, text, want string }{
		{"type", head + "  - {name: a, title: {ja: あ, en: A}, type: number}\n", "type「number」はありません"},
		{"enum", head + "  - {name: a, title: {ja: あ, en: A}, type: enum}\n", "enum には values"},
		{"values", head + "  - {name: a, title: {ja: あ, en: A}, type: string, values: [x]}\n", "values は enum に"},
		{"null", head + "  - {name: a, title: {ja: あ, en: A}, type: enum, values: [x, null]}\n", "空の語"},
		{"min", head + "  - {name: a, title: {ja: あ, en: A}, type: string, min: 1}\n", "min・max は int"},
		{"ref", head + "  - {name: a, title: {ja: あ, en: A}, type: ref}\n", "type: ref には ref"},
		{"ref target", head + "  - {name: a, title: {ja: あ, en: A}, type: ref, ref: devices.nothing}\n", "ref「devices.nothing」の先がありません"},
		{"title", head + "  - {name: a, title: {ja: あ}, type: string}\n", "title の ja と en"},
		{"twice", head + "  - {name: device, title: {ja: い, en: I}, type: string}\n", "列 device が 2 つ"},
		{"heading", head + "  - {name: a, title: {ja: 機器, en: A}, type: string}\n", "見出し「機器」が列 device と a の両方"},
		{"key", strings.Replace(head, "key: [device]", "key: [device, b]", 1), "key の列 b がありません"},
		{"unknown field", head + "  - {name: a, title: {ja: あ, en: A}, type: string, requierd: true}\n", "requierd"},
		{"default", head + "  - {name: a, title: {ja: あ, en: A}, type: int, default: x}\n", "default: 「x」は整数ではありません"},
		{"work", strings.Replace(head, "table: t", "table: x_t", 1), "作業用の表"},
		{"sheet", strings.Replace(head, "{ja: 表, en: T}", "{ja: 機器, en: T}", 1), "シート名「機器」が devices と重なっています"},
		{"reserved", strings.Replace(head, "{ja: 表, en: T}", "{ja: 共通値, en: T}", 1), "シート名「共通値」が 管理用のシート と重なっています"},
		{"sheet name", strings.Replace(head, "{ja: 表, en: T}", "{ja: 表/裏, en: T}", 1), "Excel で使えません"},
	}
	for _, c := range cases {
		_, err := load(t, map[string]string{"01_devices.yaml": devices, "02_t.yaml": c.text})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %v, want one with %q", c.name, err, c.want)
		}
	}
}
