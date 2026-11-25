package convert

import (
	"strconv"
	"strings"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// Refs fills the columns of 新旧対照表（ポート） the tool writes for people to
// read (spec 14.19.5): the old device of each interface and its settings
// now, from the old configurations. devs are the rows of 新旧対照表（機器）
// (the book's and the proposed ones). ifs are changed in place.
func Refs(sc *schema.Set, old *model.Dataset, devs []book.DeviceMap, ifs []book.IfMap) {
	if old == nil {
		return
	}
	olds := map[string][]string{} // new host → its old hosts, in order
	for _, d := range devs {
		if d.NewHost == "" {
			continue
		}
		dup := false
		for _, h := range olds[d.NewHost] {
			dup = dup || h == d.OldHost
		}
		if !dup {
			olds[d.NewHost] = append(olds[d.NewHost], d.OldHost)
		}
	}
	for i := range ifs {
		m := &ifs[i]
		for _, h := range olds[m.NewHost] {
			r := old.Tables["interfaces"].Find([]model.KV{{Name: "device", Value: h}, {Name: "name", Value: m.OldIf}})
			if r == nil {
				continue
			}
			m.OldHost, m.Now = h, IfNow(sc, old, h, r)
			break
		}
	}
}

// IfNow says, in one cell, what an old interface holds (spec 14.19.5): its
// own values worth knowing and the tables that name it.
func IfNow(sc *schema.Set, old *model.Dataset, host string, r *model.Row) string {
	var parts []string
	add := func(label, v string) {
		if v != "" {
			parts = append(parts, label+v)
		}
	}
	add("説明: ", r.Get("description"))
	if v := r.Get("ipv4"); v != "" {
		if n := len(cellValues(r.Cells["ipv4_secondary"])); n > 0 {
			v += "（セカンダリ " + strconv.Itoa(n) + " 個）"
		}
		parts = append(parts, v)
	}
	add("", r.Get("ipv6"))
	add("VRF: ", r.Get("vrf"))
	add("VLAN ", r.Get("encap_vlan"))
	add("VLAN ", r.Get("vlan_id"))
	add("ブリッジグループ ", r.Get("bridge_group"))
	// The switch port of the interface (spec 14.22.5): an access port told
	// only by its description could not be told apart.
	if sp := old.Tables["switchports"].Find([]model.KV{{Name: "device", Value: host}, {Name: "if_name", Value: r.Get("name")}}); sp != nil {
		switch mode := sp.Get("mode"); mode {
		case "trunk":
			t := "トランク"
			var in []string
			if v := sp.Get("allowed_vlans"); v != "" {
				in = append(in, "許可 VLAN "+v)
			}
			if v := sp.Get("native_vlan"); v != "" {
				in = append(in, "ネイティブ "+v)
			}
			if len(in) > 0 {
				t += "（" + strings.Join(in, "・") + "）"
			}
			parts = append(parts, t)
		case "routed":
		default:
			if v := sp.Get("access_vlan"); v != "" {
				parts = append(parts, "アクセス VLAN "+v)
			} else if mode == "access" {
				parts = append(parts, "アクセス VLAN 1（既定）")
			} else if mode != "" {
				parts = append(parts, "スイッチポート "+mode)
			}
		}
		add("音声 VLAN ", sp.Get("voice_vlan"))
	}
	if r.Get("enabled") == "false" {
		parts = append(parts, "無効（shutdown）")
	}
	add("ACL 入: ", r.Get("acl_in"))
	add("ACL 出: ", r.Get("acl_out"))
	add("NAT: ", r.Get("nat_role"))
	if pd, pi := r.Get("peer_device"), r.Get("peer_if"); pd != "" || pi != "" {
		parts = append(parts, "対向: "+strings.TrimSpace(pd+" "+pi))
	}
	if used := usedBy(sc, old, host, r.Get("name")); len(used) > 0 {
		parts = append(parts, "使っている所: "+strings.Join(used, "・"))
	}
	if len(parts) == 0 {
		return "（設定なし）"
	}
	return strings.Join(parts, " / ")
}

// usedBy names the tables, other than the interfaces, that have a row of
// host naming the interface in a column of interface names.
func usedBy(sc *schema.Set, old *model.Dataset, host, name string) []string {
	if sc == nil || name == "" {
		return nil
	}
	var out []string
	for _, t := range sc.Tables {
		// The switch port is told on its own (spec 14.22.5).
		if t.Name == "interfaces" || t.Name == "switchports" || old.Tables[t.Name] == nil {
			continue
		}
		var cols []string
		for _, c := range t.Columns {
			if c.Type == "ifname" || c.OrIfname || c.Ref == "interfaces.name" {
				cols = append(cols, c.Name)
			}
		}
		if len(cols) == 0 {
			continue
		}
		dc := t.DeviceColumn()
	rows:
		for _, row := range old.Tables[t.Name].Rows {
			if dc != "" && row.Get(dc) != host {
				continue
			}
			for _, c := range cols {
				for _, v := range cellValues(row.Cells[c]) {
					if v == name {
						out = append(out, t.Title.In("ja"))
						break rows
					}
				}
			}
		}
	}
	return out
}

func cellValues(c *model.Cell) []string {
	if c == nil {
		return nil
	}
	return c.Values
}
