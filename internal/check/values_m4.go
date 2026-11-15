package check

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/model"
)

// The checks of the values that came with the VPN tables (appendix C 1.1,
// stage M4): V-213, V-215, V-216 and V-217.

// ifAddrs lists the addresses written on the interfaces of a device, with
// the name of the interface of each.
func (x *values) ifAddrs(dev string) map[netip.Addr]string {
	out := map[netip.Addr]string{}
	t := x.in.Schema.Table("interfaces")
	if t == nil {
		return out
	}
	for _, r := range x.rows("interfaces") {
		if device(t, r) != dev {
			continue
		}
		for _, col := range []string{"ipv4", "ipv4_secondary"} {
			cell := r.Cells[col]
			if cell == nil {
				continue
			}
			for _, v := range cell.Values {
				if p, err := netip.ParsePrefix(v); err == nil {
					if _, seen := out[p.Addr()]; !seen {
						out[p.Addr()] = r.Get("name")
					}
				}
			}
		}
	}
	return out
}

// dhcp looks at the pools (V-213): what a pool hands out has to lie in its
// network, and must not be an address of the device itself.
func (x *values) dhcp() {
	t := x.in.Schema.Table("dhcp_pools")
	if t == nil {
		return
	}
	for _, r := range x.rows("dhcp_pools") {
		dev := device(t, r)
		net, err := netip.ParsePrefix(r.Get("network"))
		if err != nil {
			continue
		}
		x.tally.Look("V-213", dev, 1)
		at := map[string]netip.Addr{}
		for _, col := range []string{"range_start", "range_end", "default_router"} {
			a, err := netip.ParseAddr(r.Get(col))
			if err != nil {
				continue
			}
			at[col] = a
			if !net.Contains(a) {
				x.add("V-213", t, r, col, fmt.Sprintf("「%s」の %s が、ネットワーク %s の外にあります", t.Column(col).Title.JA, a, net),
					model.KV{Name: "値", Value: a.String()}, model.KV{Name: "ネットワーク", Value: net.String()})
			}
		}
		start, ok1 := at["range_start"]
		end, ok2 := at["range_end"]
		if !ok1 || !ok2 {
			continue
		}
		if end.Less(start) {
			x.add("V-213", t, r, "range_start", fmt.Sprintf("配る範囲の先頭 %s が、末尾 %s より後ろにあります", start, end),
				model.KV{Name: "先頭", Value: start.String()}, model.KV{Name: "末尾", Value: end.String()})
			continue
		}
		within := func(a, lo, hi netip.Addr) bool { return !a.Less(lo) && !hi.Less(a) }
		kept := func(a netip.Addr) bool {
			cell := r.Cells["excluded"]
			if cell == nil {
				return false
			}
			for _, v := range cell.Values {
				ends := strings.SplitN(v, "-", 2)
				lo, err := netip.ParseAddr(strings.TrimSpace(ends[0]))
				if err != nil {
					continue
				}
				hi := lo
				if len(ends) == 2 {
					if h, err := netip.ParseAddr(strings.TrimSpace(ends[1])); err == nil {
						hi = h
					}
				}
				if within(a, lo, hi) {
					return true
				}
			}
			return false
		}
		// In the order of the addresses: the same input, the same output.
		own := x.ifAddrs(dev)
		addrs := make([]netip.Addr, 0, len(own))
		for a := range own {
			addrs = append(addrs, a)
		}
		sort.Slice(addrs, func(i, j int) bool { return addrs[i].Less(addrs[j]) })
		for _, a := range addrs {
			name := own[a]
			if within(a, start, end) && !kept(a) {
				x.add("V-213", t, r, "range_start", fmt.Sprintf("配る範囲（%s〜%s）が、この機器の %s のアドレス %s を含みます", start, end, name, a),
					model.KV{Name: "IF", Value: name}, model.KV{Name: "アドレス", Value: a.String()})
			}
		}
	}
}

// weak are the values V-216 tells of, by column.
var weak = []struct {
	column string
	values []string
}{
	{"ike_encryption", []string{"des"}},
	{"ike_hash", []string{"md5"}},
	{"ike_dh", []string{"dh1", "dh2", "dh5"}},
	{"esp_encryption", []string{"des"}},
	{"esp_hash", []string{"md5"}},
	{"pfs", []string{"dh1", "dh2", "dh5"}},
}

// ipsec looks at the rows of IPsec: what the style of a row needs is there
// (V-215), and no weak algorithm is written (V-216).
func (x *values) ipsec() {
	t := x.in.Schema.Table("ipsec_tunnels")
	if t == nil {
		return
	}
	st := x.in.Schema.Table("ipsec_selectors")
	for _, r := range x.rows("ipsec_tunnels") {
		dev, name := device(t, r), r.Get("name")
		x.tally.Look("V-215", dev, 1)
		switch r.Get("style") {
		case "tunnel_if":
			tif := r.Get("tunnel_if")
			if tif == "" {
				x.add("V-215", t, r, "tunnel_if", fmt.Sprintf("IPsec %s は方式が tunnel_if ですが、トンネル IF が空です", name))
				break
			}
			tun := x.find("tunnels", dev, "if_name", tif)
			if tun == nil {
				x.add("V-215", t, r, "tunnel_if", fmt.Sprintf("IPsec %s のトンネル IF %s が、トンネルの表にありません", name, tif),
					model.KV{Name: "トンネル IF", Value: tif})
			} else if kind := tun.Get("type"); kind != "" && kind != "ipsec" {
				x.add("V-215", t, r, "tunnel_if", fmt.Sprintf("IPsec %s のトンネル IF %s は、トンネルの表で種類が %s です（ipsec ではありません）", name, tif, kind),
					model.KV{Name: "トンネル IF", Value: tif}, model.KV{Name: "種類", Value: kind})
			}
		case "policy":
			found := false
			if st != nil {
				for _, s := range x.rows("ipsec_selectors") {
					if device(st, s) == dev && s.Get("tunnel") == name {
						found = true
						break
					}
				}
			}
			if !found {
				x.add("V-215", t, r, "style", fmt.Sprintf("IPsec %s は方式が policy ですが、対象通信の行がありません", name))
			}
		}
		x.tally.Look("V-216", dev, 1)
		for _, w := range weak {
			v := r.Get(w.column)
			for _, bad := range w.values {
				if v == bad {
					x.add("V-216", t, r, w.column, fmt.Sprintf("IPsec %s の「%s」が、弱い方式（%s）です", name, t.Column(w.column).Title.JA, v),
						model.KV{Name: "値", Value: v})
				}
			}
		}
	}
}

// pppoe looks at the two interfaces of a PPPoE row (V-217): both have a row
// on the sheet of the interfaces.
func (x *values) pppoe() {
	t := x.in.Schema.Table("pppoe")
	if t == nil {
		return
	}
	for _, r := range x.rows("pppoe") {
		dev := device(t, r)
		x.tally.Look("V-217", dev, 1)
		for _, col := range []string{"physical_if", "logical_if"} {
			name := r.Get(col)
			if name == "" || x.find("interfaces", dev, "name", name) != nil {
				continue
			}
			x.add("V-217", t, r, col, fmt.Sprintf("PPPoE %s の「%s」%s が、インタフェースの表にありません", r.Get("name"), t.Column(col).Title.JA, name),
				model.KV{Name: "IF", Value: name})
		}
	}
}
