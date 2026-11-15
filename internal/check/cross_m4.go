package check

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/model"
)

// The checks of the pairs for IPsec and the tunnels (appendix C 1.2, stage
// M4): X-301 to X-309 and X-311 to X-314. A row and the row of the other
// side are a pair when each points at an address of the other device
// (spec 11.5).

// devHas tells whether an address is written on an interface of a device.
func (c *cross) devHas(dev string, addr netip.Addr) bool {
	for _, i := range c.byDev[dev] {
		if i.has(addr) {
			return true
		}
	}
	return false
}

// anyOwner gives an interface of another device that has an address.
func (c *cross) anyOwner(addr netip.Addr, not string) *xIf {
	for _, i := range c.ifs {
		if i.device != not && i.has(addr) {
			return i
		}
	}
	return nil
}

// srcAddr turns the source of a tunnel into an address: the address itself,
// or the first one of the interface it names.
func (c *cross) srcAddr(dev, src string) (netip.Addr, bool) {
	if a, err := netip.ParseAddr(src); err == nil {
		return a, true
	}
	if i := c.byDev[dev][src]; i != nil && len(i.addrs) > 0 {
		return i.addrs[0].Addr(), true
	}
	return netip.Addr{}, false
}

// secLocal gives the address a row of IPsec sends from, where it is known:
// the source of its tunnel, its own address or that of its interface.
func (c *cross) secLocal(r *model.Row) (netip.Addr, bool) {
	dev := r.Get("device")
	if tif := r.Get("tunnel_if"); tif != "" {
		if tun := c.find("tunnels", dev, "if_name", tif); tun != nil && tun.Get("source") != "" {
			return c.srcAddr(dev, tun.Get("source"))
		}
	}
	if a, err := netip.ParseAddr(r.Get("local_address")); err == nil {
		return a, true
	}
	if lif := r.Get("local_if"); lif != "" {
		return c.srcAddr(dev, lif)
	}
	return netip.Addr{}, false
}

// ipsec runs the checks of the IPsec pairs and gives the pairs back, each
// once, the row that comes first in the book first.
func (c *cross) ipsec() [][2]*model.Row {
	t := c.in.Schema.Table("ipsec_tunnels")
	if t == nil {
		return nil
	}
	rows := c.rows("ipsec_tunnels")
	mate := map[*model.Row]*model.Row{}
	var pairs [][2]*model.Row
	for _, r := range rows {
		dev := r.Get("device")
		c.tally.Look("X-300", dev, 1)
		addr, err := netip.ParseAddr(r.Get("peer_address"))
		if err != nil {
			c.add("X-300", t, r, "peer_address", fmt.Sprintf("IPsec %s の相手が、アドレスで決まっていません（管理外の相手として扱います）", r.Get("name")),
				model.KV{Name: "相手アドレス", Value: r.Get("peer_address")})
			continue
		}
		peer := c.anyOwner(addr, dev)
		if peer == nil {
			c.add("X-300", t, r, "peer_address", fmt.Sprintf("IPsec の相手 %s を持つ機器が、案件の中にありません（管理外の相手）", addr),
				model.KV{Name: "相手アドレス", Value: addr.String()})
			continue
		}
		c.tally.Look("X-301", dev, 1)
		if mate[r] != nil {
			continue // found from the other side
		}
		// The rows of the other device that point back at this one. Where
		// the address this row sends from is known, the row that points at
		// it comes first.
		local, known := c.secLocal(r)
		var rev *model.Row
		for _, o := range rows {
			if o.Get("device") != peer.device || mate[o] != nil {
				continue
			}
			back, err := netip.ParseAddr(o.Get("peer_address"))
			if err != nil || !c.devHas(dev, back) {
				continue
			}
			if rev == nil || (known && back == local) {
				rev = o
				if known && back == local {
					break
				}
			}
		}
		if rev == nil {
			c.add("X-301", t, r, "peer_address", fmt.Sprintf("相手の %s（%s）に、この機器を相手にする IPsec の行がありません", peer.device, peer.name),
				model.KV{Name: "相手アドレス", Value: addr.String()}, model.KV{Name: "相手", Value: where(peer.device, peer.name)})
			continue
		}
		mate[r], mate[rev] = rev, r
		pairs = append(pairs, [2]*model.Row{r, rev})
	}

	title := func(col string) string { return t.Column(col).Title.JA }
	for _, r := range rows {
		rev := mate[r]
		if rev == nil {
			continue
		}
		// The identity has a direction: each row is looked at.
		c.tally.Look("X-306", r.Get("device"), 1)
		if a, b := r.Get("local_id"), rev.Get("remote_id"); both(a, b) {
			c.add("X-306", t, r, "local_id", fmt.Sprintf("自分の ID が、相手の %s が待っている相手の ID と違います", rev.Get("device")),
				model.KV{Name: where(r.Get("device"), "の自分の ID"), Value: a}, model.KV{Name: where(rev.Get("device"), "の相手の ID"), Value: b})
		}
	}
	for _, p := range pairs {
		r, rev := p[0], p[1]
		dev, other := r.Get("device"), rev.Get("device")
		differ := func(rule string, cols ...string) {
			c.tally.Look(rule, dev, 1)
			for _, col := range cols {
				if a, b := r.Get(col), rev.Get(col); both(a, b) {
					c.add(rule, t, r, col, fmt.Sprintf("「%s」が両端で違います", title(col)),
						model.KV{Name: dev, Value: a}, model.KV{Name: other, Value: b})
				}
			}
		}
		differ("X-302", "ike_version", "ike_mode")
		differ("X-303", "ike_encryption", "ike_hash", "ike_dh")
		differ("X-304", "esp_encryption", "esp_hash", "pfs")
		differ("X-308", "ike_lifetime", "ipsec_lifetime")
		c.tally.Look("X-305", dev, 1)
		if a, b := r.Get("psk"), rev.Get("psk"); both(a, b) {
			c.add("X-305", t, r, "psk", "事前共有鍵が両端で違います（不一致）",
				model.KV{Name: "こちら", Value: where(dev, r.Get("name"))}, model.KV{Name: "相手", Value: where(other, rev.Get("name"))})
		}
		c.tally.Look("X-309", dev, 1)
		for _, col := range []string{"dpd_interval", "nat_traversal"} {
			a, b := r.Get(col), rev.Get(col)
			if col == "nat_traversal" {
				// "false" says the same as nothing written.
				a, b = strings.TrimPrefix(a, "false"), strings.TrimPrefix(b, "false")
			}
			if (a == "") != (b == "") {
				c.add("X-309", t, r, col, fmt.Sprintf("「%s」が片側にだけあります", title(col)),
					model.KV{Name: dev, Value: a}, model.KV{Name: other, Value: b})
			}
		}
		if r.Get("style") == "policy" && rev.Get("style") == "policy" {
			c.tally.Look("X-307", dev, 1)
			mine, theirs := c.selectors(r, false), c.selectors(rev, true)
			if strings.Join(mine, "\n") != strings.Join(theirs, "\n") {
				c.add("X-307", t, r, "name", fmt.Sprintf("対象通信が、相手の %s（%s）と鏡写しになっていません", other, rev.Get("name")),
					model.KV{Name: where(dev, "の対象通信（自分側 → 相手側）"), Value: strings.Join(mine, "、")},
					model.KV{Name: where(other, "の対象通信（相手側 → 自分側）"), Value: strings.Join(theirs, "、")})
			}
		}
	}
	return pairs
}

// selectors lists the traffic of a row as "local > remote protocol", sorted;
// with flip, as the other side has to write it.
func (c *cross) selectors(r *model.Row, flip bool) []string {
	var out []string
	for _, s := range c.rows("ipsec_selectors") {
		if s.Get("device") != r.Get("device") || s.Get("tunnel") != r.Get("name") {
			continue
		}
		a, b := s.Get("local_net"), s.Get("remote_net")
		if flip {
			a, b = b, a
		}
		proto := s.Get("protocol")
		if proto == "" {
			proto = "ip"
		}
		out = append(out, a+" > "+b+" "+proto)
	}
	sort.Strings(out)
	return out
}

// tunnels runs the checks of the tunnel pairs. A tunnel of IPsec is paired
// by the pair of its IPsec rows; any other by its destination.
func (c *cross) tunnels(sec [][2]*model.Row) {
	t := c.in.Schema.Table("tunnels")
	if t == nil {
		return
	}
	rows := c.rows("tunnels")
	mate := map[*model.Row]*model.Row{}
	var pairs [][2]*model.Row
	link := func(a, b *model.Row) {
		if a == nil || b == nil || mate[a] != nil || mate[b] != nil {
			return
		}
		mate[a], mate[b] = b, a
		pairs = append(pairs, [2]*model.Row{a, b})
	}
	for _, p := range sec {
		if p[0].Get("tunnel_if") == "" || p[1].Get("tunnel_if") == "" {
			continue
		}
		link(c.find("tunnels", p[0].Get("device"), "if_name", p[0].Get("tunnel_if")),
			c.find("tunnels", p[1].Get("device"), "if_name", p[1].Get("tunnel_if")))
	}
	for _, r := range rows {
		dev := r.Get("device")
		if r.Get("type") == "ipsec" || mate[r] != nil {
			continue
		}
		dst, err := netip.ParseAddr(r.Get("destination"))
		if err != nil {
			continue
		}
		c.tally.Look("X-300", dev, 1)
		peer := c.anyOwner(dst, dev)
		if peer == nil {
			c.add("X-300", t, r, "destination", fmt.Sprintf("トンネルの宛先 %s を持つ機器が、案件の中にありません（管理外の相手）", dst),
				model.KV{Name: "宛先", Value: dst.String()})
			continue
		}
		c.tally.Look("X-301", dev, 1)
		local, known := c.srcAddr(dev, r.Get("source"))
		var rev *model.Row
		for _, o := range rows {
			if o.Get("device") != peer.device || o.Get("type") == "ipsec" || mate[o] != nil {
				continue
			}
			back, err := netip.ParseAddr(o.Get("destination"))
			if err != nil || !c.devHas(dev, back) {
				continue
			}
			if rev == nil || (known && back == local) {
				rev = o
				if known && back == local {
					break
				}
			}
		}
		if rev == nil {
			c.add("X-301", t, r, "destination", fmt.Sprintf("相手の %s（%s）に、この機器を宛先にするトンネルの行がありません", peer.device, peer.name),
				model.KV{Name: "宛先", Value: dst.String()}, model.KV{Name: "相手", Value: where(peer.device, peer.name)})
			continue
		}
		link(r, rev)
	}

	title := func(col string) string { return t.Column(col).Title.JA }
	for _, p := range pairs {
		r, rev := p[0], p[1]
		dev, other := r.Get("device"), rev.Get("device")
		pair := []model.KV{{Name: "こちら", Value: where(dev, r.Get("if_name"))}, {Name: "相手", Value: where(other, rev.Get("if_name"))}}

		// X-311: each side sends from where the other sends to.
		if r.Get("source") != "" && rev.Get("source") != "" && r.Get("destination") != "" && rev.Get("destination") != "" {
			c.tally.Look("X-311", dev, 1)
			srcA, okA := c.srcAddr(dev, r.Get("source"))
			srcB, okB := c.srcAddr(other, rev.Get("source"))
			dstA, _ := netip.ParseAddr(r.Get("destination"))
			dstB, _ := netip.ParseAddr(rev.Get("destination"))
			if okA && dstB.IsValid() && srcA != dstB {
				c.add("X-311", t, r, "source", fmt.Sprintf("送信元 %s が、相手の %s のトンネルの宛先 %s と違います", srcA, other, dstB),
					model.KV{Name: where(dev, "の送信元"), Value: srcA.String()}, model.KV{Name: where(other, "の宛先"), Value: dstB.String()})
			}
			if okB && dstA.IsValid() && dstA != srcB {
				c.add("X-311", t, r, "destination", fmt.Sprintf("宛先 %s が、相手の %s のトンネルの送信元 %s と違います", dstA, other, srcB),
					model.KV{Name: where(dev, "の宛先"), Value: dstA.String()}, model.KV{Name: where(other, "の送信元"), Value: srcB.String()})
			}
		}

		c.tally.Look("X-312", dev, 1)
		if a, b := r.Get("type"), rev.Get("type"); both(a, b) {
			c.add("X-312", t, r, "type", "トンネルの種類が両端で違います", model.KV{Name: dev, Value: a}, model.KV{Name: other, Value: b})
		} else if a == "gre" && r.Get("key") != rev.Get("key") {
			msg := "トンネルの鍵が両端で違います"
			if r.Get("key") == "" || rev.Get("key") == "" {
				msg = "トンネルの鍵が片側にだけあります"
			}
			c.add("X-312", t, r, "key", msg, model.KV{Name: dev, Value: r.Get("key")}, model.KV{Name: other, Value: rev.Get("key")})
		}

		ia, ib := c.byDev[dev][r.Get("if_name")], c.byDev[other][rev.Get("if_name")]
		if ia != nil && ib != nil {
			if len(ia.addrs) > 0 && len(ib.addrs) > 0 {
				c.tally.Look("X-313", dev, 1)
				same := false
				for _, pa := range ia.addrs {
					for _, pb := range ib.addrs {
						if pa.Masked() == pb.Masked() {
							same = true
						}
					}
				}
				if !same {
					c.add("X-313", t, r, "if_name", "両端のトンネル IF のアドレスが、同じサブネットにありません",
						model.KV{Name: where(dev, r.Get("if_name")), Value: ia.addrs[0].String()}, model.KV{Name: where(other, rev.Get("if_name")), Value: ib.addrs[0].String()})
				}
			}
			c.tally.Look("X-314", dev, 1)
			if both(ia.mtu, ib.mtu) {
				c.add("X-314", t, r, "if_name", "トンネル IF の MTU が両端で違います", model.KV{Name: where(dev, r.Get("if_name")), Value: ia.mtu}, model.KV{Name: where(other, rev.Get("if_name")), Value: ib.mtu})
			}
			if a, b := ia.row.Get("tcp_mss"), ib.row.Get("tcp_mss"); both(a, b) {
				c.add("X-314", t, r, "if_name", "トンネル IF の TCP MSS 調整が両端で違います", model.KV{Name: where(dev, r.Get("if_name")), Value: a}, model.KV{Name: where(other, rev.Get("if_name")), Value: b})
			}
		}
		for _, col := range []string{"keepalive_interval", "keepalive_retry"} {
			if a, b := r.Get(col), rev.Get(col); both(a, b) {
				c.add("X-314", t, r, col, fmt.Sprintf("「%s」が両端で違います", title(col)), append([]model.KV{{Name: dev, Value: a}, {Name: other, Value: b}}, pair...)...)
			}
		}
	}
}
