package check

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/model"
)

// The checks of VRRP version 3 (appendix A 3.23, "IPv6 の VRRP"; appendix C
// 1.1 and 1.2): the virtual addresses of IPv6 VRRP (V-290), "vrrpv2" where it
// does nothing (V-291), the version two IPv4 peers run (X-405, in fhrp of
// cross.go) and the pairs of IPv6 VRRP (X-406; X-300, X-343 and X-346 as for
// IPv4).

var (
	vrrp3ValueIDs = []string{"V-290", "V-291"}
	vrrp3CrossIDs = []string{"X-405", "X-406"}
)


// vrrpVersion gives the version the IPv4 groups of a device run: "3" or
// "2" (an empty cell is 2, the default of every vendor).
func (x *values) vrrpVersion(dev string) string {
	if d := x.find("devices", dev, "hostname", dev); d != nil && d.Get("vrrp_version") == "3" {
		return "3"
	}
	return "2"
}

// fixed6 gives the fixed IPv6 addresses of an interface.
func (x *values) fixed6(dev, name string) []netip.Prefix {
	var out []netip.Prefix
	if i := x.find("interfaces", dev, "name", name); i != nil {
		if cell := i.Cells["ipv6"]; cell != nil {
			for _, v := range cell.Values {
				if p, err := netip.ParsePrefix(v); err == nil && p.Addr().Is6() {
					out = append(out, p)
				}
			}
		}
	}
	return out
}

func (x *values) vrrp3() {
	for _, id := range vrrp3ValueIDs {
		x.tally.Look(id, "", 0)
	}
	if t := x.in.Schema.Table("ipv6_fhrp"); t != nil {
		for _, r := range x.rows("ipv6_fhrp") {
			dev, ifn := r.Get("device"), r.Get("if_name")
			x.tally.Look("V-290", dev, 1)
			if ll, err := netip.ParseAddr(r.Get("link_local")); err == nil && !linkLocal(ll) {
				x.add("V-290", t, r, "link_local", fmt.Sprintf("仮想リンクローカルアドレス %s が、fe80::/10 に入りません", ll),
					model.KV{Name: "値", Value: ll.String()})
			}
			nets := x.fixed6(dev, ifn)
			if len(nets) == 0 {
				continue
			}
			var shown []string
			for _, p := range nets {
				shown = append(shown, p.Masked().String())
			}
			if cell := r.Cells["addresses"]; cell != nil {
				for _, v := range cell.Values {
					a, err := netip.ParseAddr(v)
					if err != nil {
						continue
					}
					inside := false
					for _, p := range nets {
						inside = inside || p.Masked().Contains(a)
					}
					if !inside {
						x.add("V-290", t, r, "addresses", fmt.Sprintf("仮想グローバルアドレス %s は、%s の IPv6 のサブネットに入りません", a, ifn),
							model.KV{Name: "仮想アドレス", Value: a.String()}, model.KV{Name: ifn, Value: join(shown)})
					}
				}
			}
		}
	}
	if t := x.in.Schema.Table("fhrp"); t != nil {
		for _, r := range x.rows("fhrp") {
			if r.Get("vrrpv2") != "true" {
				continue
			}
			dev := r.Get("device")
			x.tally.Look("V-291", dev, 1)
			switch {
			case r.Get("protocol") != "vrrp":
				x.add("V-291", t, r, "vrrpv2", fmt.Sprintf("「版 2 の相手とも話す」は VRRP の設定です（方式が %s の行では効きません）", r.Get("protocol")))
			case x.vrrpVersion(dev) != "3":
				x.add("V-291", t, r, "vrrpv2", fmt.Sprintf("「版 2 の相手とも話す」は、VRRP の版が 3 の機器の設定です（%s の「VRRP の版（IPv4）」は 3 ではないので、もとから版 2 で動きます）", dev))
			}
		}
	}
}

// versionClash tells whether two IPv4 VRRP groups of a pair run versions
// that do not meet (X-405): one on version 3 only, the other on version 2.
// It gives the row on version 3 first.
func (c *cross) versionClash(a, b *model.Row) (v3, v2 *model.Row, ok bool) {
	if a.Get("protocol") != "vrrp" || b.Get("protocol") != "vrrp" {
		return nil, nil, false
	}
	only3 := func(r *model.Row) bool { return c.vrrpVersion(r.Get("device")) == "3" && r.Get("vrrpv2") != "true" }
	switch {
	case only3(a) && c.vrrpVersion(b.Get("device")) == "2":
		return a, b, true
	case only3(b) && c.vrrpVersion(a.Get("device")) == "2":
		return b, a, true
	}
	return nil, nil, false
}

// values6 gives the virtual global addresses of a row of IPv6 VRRP.
func values6(r *model.Row) []string {
	if cell := r.Cells["addresses"]; cell != nil {
		return cell.Values
	}
	return nil
}

// group6 is a row of IPv6 VRRP with the subnet it stands in.
type group6 struct {
	row    *model.Row
	i      *xIf
	subnet netip.Prefix
}

// fhrp6 sets the rows of IPv6 VRRP side by side (X-406, and X-300, X-343,
// X-346 as for IPv4). The subnet is that of the fixed IPv6 address that
// holds a virtual global address, or else the first one of the interface.
func (c *cross) fhrp6() {
	for _, id := range vrrp3CrossIDs {
		c.tally.Look(id, "", 0)
	}
	t := c.in.Schema.Table("ipv6_fhrp")
	if t == nil {
		return
	}
	var list []group6
	for _, r := range c.rows("ipv6_fhrp") {
		i := c.byDev[r.Get("device")][r.Get("if_name")]
		if i == nil || len(i.addrs6) == 0 {
			continue
		}
		g := group6{r, i, i.addrs6[0].Masked()}
	find:
		for _, v := range values6(r) {
			if a, err := netip.ParseAddr(v); err == nil {
				for _, p := range i.addrs6 {
					if p.Masked().Contains(a) {
						g.subnet = p.Masked()
						break find
					}
				}
			}
		}
		list = append(list, g)
	}
	twin := func(a, b group6) bool {
		return a.i.device != b.i.device && a.i.vrf == b.i.vrf && a.subnet == b.subnet && a.row.Get("group") == b.row.Get("group")
	}
	set := func(r *model.Row) string {
		v := append([]string{}, values6(r)...)
		sort.Strings(v)
		return strings.Join(v, ", ")
	}
	for n, a := range list {
		dev, no := a.i.device, a.row.Get("group")
		c.tally.Look("X-300", dev, 1)
		c.tally.Look("X-346", dev, 1)
		alone := true
		for _, b := range list {
			alone = alone && !twin(a, b)
		}
		if alone {
			var near []*xIf
			for _, o := range c.ifs {
				if o.device == dev || o.vrf != a.i.vrf {
					continue
				}
				for _, p := range o.addrs6 {
					if p.Masked() == a.subnet {
						near = append(near, o)
						break
					}
				}
			}
			if len(near) == 0 {
				c.add("X-300", t, a.row, "group", fmt.Sprintf("%s の IPv6 の VRRP %s の相手が、案件の中にありません（同じ IPv6 のサブネットに、ほかの機器の IF がありません。管理外の相手）", a.i.name, no),
					model.KV{Name: a.i.name, Value: a.subnet.String()})
			} else {
				c.add("X-346", t, a.row, "group", fmt.Sprintf("IPv6 の VRRP %s に機器が 1 台しかありません（同じ IPv6 のサブネットの %s の %s に、同じグループ番号の行がありません）", no, near[0].device, near[0].name),
					model.KV{Name: where(dev, a.i.name), Value: "グループ " + no}, model.KV{Name: where(near[0].device, near[0].name), Value: "行なし"})
			}
			continue
		}
		for _, b := range list[n+1:] {
			if !twin(a, b) {
				continue
			}
			for _, id := range []string{"X-406", "X-343"} {
				c.tally.Look(id, dev, 1)
			}
			wa, wb := where(dev, a.i.name), where(b.i.device, b.i.name)
			diff := func(rule, col, msg string) {
				c.add(rule, t, a.row, col, msg, model.KV{Name: wa, Value: a.row.Get(col)}, model.KV{Name: wb, Value: b.row.Get(col)})
			}
			if x, y := a.row.Get("link_local"), b.row.Get("link_local"); both(x, y) {
				diff("X-406", "link_local", fmt.Sprintf("IPv6 の VRRP %s の仮想リンクローカルアドレスが、%s と違います", no, wb))
			}
			if x, y := set(a.row), set(b.row); x != y {
				c.add("X-406", t, a.row, "addresses", fmt.Sprintf("IPv6 の VRRP %s の仮想グローバルアドレスが、%s と違います", no, wb),
					model.KV{Name: wa, Value: x}, model.KV{Name: wb, Value: y})
			}
			if x, y := a.row.Get("advert_interval"), b.row.Get("advert_interval"); both(x, y) {
				diff("X-406", "advert_interval", fmt.Sprintf("IPv6 の VRRP %s の広告間隔が、%s と違います（同じグループでは同じ値にします）", no, wb))
			}
			if x, y := a.row.Get("priority"), b.row.Get("priority"); x != "" && x == y {
				diff("X-343", "priority", fmt.Sprintf("IPv6 の VRRP %s の優先度が、%s と同じです（どちらが主になるかが決まりません）", no, wb))
			}
		}
	}
}
