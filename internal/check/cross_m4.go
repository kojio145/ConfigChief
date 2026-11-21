package check

import (
	"fmt"
	"net/netip"
	"sort"
	"strconv"
	"strings"

	"github.com/kojio145/configchief/internal/model"
)

// The checks of the pairs for IPsec and the tunnels (appendix C 1.2, stage
// M4): X-301 to X-309, X-365 and X-311 to X-314. A row and the row of the other
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
	return c.srcAddrOf(dev, src, false)
}

// srcAddrOf is srcAddr for one family: the first IPv6 address of the
// interface for a tunnel over IPv6.
func (c *cross) srcAddrOf(dev, src string, v6 bool) (netip.Addr, bool) {
	if a, err := netip.ParseAddr(src); err == nil {
		return a, true
	}
	i := c.byDev[dev][src]
	switch {
	case i == nil:
	case !v6 && len(i.addrs) > 0:
		return i.addrs[0].Addr(), true
	case v6 && len(i.addrs6) > 0:
		return i.addrs6[0].Addr(), true
	}
	return netip.Addr{}, false
}

// tunnelDst gives the far end of a tunnel: its IPv4 destination, or the
// IPv6 one (a tunnel over IPv6, appendix A 3.23; 2026-11-21).
func tunnelDst(r *model.Row) (netip.Addr, string, error) {
	if v := r.Get("destination"); v != "" {
		a, err := netip.ParseAddr(v)
		return a, "destination", err
	}
	a, err := netip.ParseAddr(r.Get("destination_v6"))
	return a, "destination_v6", err
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

// byName gives the device a peer written by its name is: the host name with
// the domain name of a device of the case, or its host name alone.
func (c *cross) byName(name, not string) string {
	name = strings.ToLower(strings.TrimSuffix(name, "."))
	if name == "" || name == "any" {
		return ""
	}
	for _, d := range c.rows("devices") {
		host := d.Get("hostname")
		if host == "" || host == not {
			continue
		}
		if dom := d.Get("domain_name"); dom != "" && strings.ToLower(host+"."+dom) == name {
			return host
		}
		if strings.ToLower(host) == name {
			return host
		}
	}
	return ""
}

// version2 tells that a row of IPsec uses IKE version 2.
func version2(r *model.Row) bool { return r.Get("ike_version") == "2" }

// ipsec runs the checks of the IPsec pairs and gives the pairs back, each
// once, the row that comes first in the book first. A row and the row of the
// other device are a pair when each points at the other: by an address, by
// the name of the device, or, on one side, by "any" (a peer whose address is
// not known: such a row is the other side of every row that points at its
// device and finds no row that points back).
func (c *cross) ipsec() [][2]*model.Row {
	t := c.in.Schema.Table("ipsec_tunnels")
	if t == nil {
		return nil
	}
	rows := c.rows("ipsec_tunnels")
	mate := map[*model.Row]*model.Row{}
	waits := map[*model.Row]bool{} // the rows with the peer "any" that found a pair
	var pairs [][2]*model.Row
	for _, r := range rows {
		dev := r.Get("device")
		if r.Get("peer_address") == "any" {
			continue // looked at below, once the pairs are known
		}
		c.tally.Look("X-300", dev, 1)
		var peerDev, peerName string
		addr, err := netip.ParseAddr(r.Get("peer_address"))
		if err != nil {
			peerDev = c.byName(r.Get("peer_address"), dev)
			if peerDev == "" {
				c.add("X-300", t, r, "peer_address", fmt.Sprintf("IPsec %s の相手が、アドレスで決まっていません（管理外の相手として扱います）", r.Get("name")),
					model.KV{Name: "相手アドレス", Value: r.Get("peer_address")})
				continue
			}
			peerName = r.Get("peer_address")
		} else {
			peer := c.anyOwner(addr, dev)
			if peer == nil {
				c.add("X-300", t, r, "peer_address", fmt.Sprintf("IPsec の相手 %s を持つ機器が、案件の中にありません（管理外の相手）", addr),
					model.KV{Name: "相手アドレス", Value: addr.String()})
				continue
			}
			peerDev, peerName = peer.device, peer.name
		}
		c.tally.Look("X-301", dev, 1)
		if mate[r] != nil {
			continue // found from the other side
		}
		// The rows of the other device that point back at this one. Where
		// the address this row sends from is known, the row that points at
		// it comes first; a row that waits for any peer comes last.
		local, known := c.secLocal(r)
		var rev, any *model.Row
		for _, o := range rows {
			if o.Get("device") != peerDev || mate[o] != nil {
				continue
			}
			if o.Get("peer_address") == "any" {
				if any == nil && !both(o.Get("remote_id"), r.Get("local_id")) && o.Get("ike_version") == r.Get("ike_version") {
					any = o
				}
				continue
			}
			back, err := netip.ParseAddr(o.Get("peer_address"))
			if err != nil {
				if rev == nil && c.byName(o.Get("peer_address"), peerDev) == dev {
					rev = o
				}
				continue
			}
			if !c.devHas(dev, back) {
				continue
			}
			if rev == nil || (known && back == local) {
				rev = o
				if known && back == local {
					break
				}
			}
		}
		if rev == nil && any != nil {
			waits[any] = true
			pairs = append(pairs, [2]*model.Row{r, any})
			continue
		}
		if rev == nil {
			c.add("X-301", t, r, "peer_address", fmt.Sprintf("相手の %s（%s）に、この機器を相手にする IPsec の行がありません", peerDev, peerName),
				model.KV{Name: "相手アドレス", Value: r.Get("peer_address")}, model.KV{Name: "相手", Value: where(peerDev, peerName)})
			continue
		}
		mate[r], mate[rev] = rev, r
		pairs = append(pairs, [2]*model.Row{r, rev})
	}
	for _, r := range rows {
		if r.Get("peer_address") != "any" {
			continue
		}
		c.tally.Look("X-300", r.Get("device"), 1)
		if !waits[r] {
			c.add("X-300", t, r, "peer_address", fmt.Sprintf("IPsec %s の相手が、アドレスで決まっていません（管理外の相手として扱います）", r.Get("name")),
				model.KV{Name: "相手アドレス", Value: r.Get("peer_address")})
		}
	}

	title := func(col string) string { return t.Column(col).Title.JA }
	// The identity has a direction: both rows of a pair are looked at. With
	// IKE version 2 an empty cell is the address (appendix A 3.20): the own
	// one is the address the other side points at, and the one waited for is
	// the address of the peer.
	ident := func(r, rev *model.Row) {
		c.tally.Look("X-306", r.Get("device"), 1)
		a, b := r.Get("local_id"), rev.Get("remote_id")
		if version2(r) && version2(rev) {
			if _, err := netip.ParseAddr(rev.Get("peer_address")); err == nil {
				if a == "" {
					a = "address:" + rev.Get("peer_address")
				}
				if b == "" {
					b = "address:" + rev.Get("peer_address")
				}
			}
		}
		if both(a, b) {
			c.add("X-306", t, r, "local_id", fmt.Sprintf("自分の ID が、相手の %s が待っている相手の ID と違います", rev.Get("device")),
				model.KV{Name: where(r.Get("device"), "の自分の ID"), Value: a}, model.KV{Name: where(rev.Get("device"), "の相手の ID"), Value: b})
		}
	}
	for _, p := range pairs {
		ident(p[0], p[1])
		ident(p[1], p[0])
	}
	// orElse gives a value, or what an empty cell stands for.
	orElse := func(v, empty string) string {
		if v == "" {
			return empty
		}
		return v
	}
	for _, p := range pairs {
		r, rev := p[0], p[1]
		dev, other := r.Get("device"), rev.Get("device")
		differ := func(rule string, cols ...string) {
			c.tally.Look(rule, dev, 1)
			for _, col := range cols {
				if t.Column(col) == nil {
					continue
				}
				if a, b := r.Get(col), rev.Get(col); both(a, b) {
					c.add(rule, t, r, col, fmt.Sprintf("「%s」が両端で違います", title(col)),
						model.KV{Name: dev, Value: a}, model.KV{Name: other, Value: b})
				}
			}
		}
		differ("X-302", "ike_version", "ike_mode")
		// Where neither side has more than one candidate, the values are
		// compared one by one. Where a side has more (the "(more)" columns,
		// the rows of "IPsec more proposals"), the two ends need one
		// candidate in common (X-303, X-304), and lists that are not the
		// same are told as a warning (X-366).
		c.tally.Look("X-366", dev, 1)
		for _, ph := range []struct{ rule, phase, col, what string }{
			{"X-303", "ike", "ike_encryption", "IKE の提案（暗号・ハッシュ・DH グループ）"},
			{"X-304", "ipsec", "esp_encryption", "IPsec の提案（ESP 暗号・ESP ハッシュ）"},
		} {
			mine, theirs := c.candidates(r, ph.phase), c.candidates(rev, ph.phase)
			if len(mine) == 1 && len(theirs) == 1 {
				if ph.phase == "ike" {
					differ("X-303", "ike_encryption", "ike_hash", "ike_dh", "ike_prf")
				} else {
					differ("X-304", "esp_encryption", "esp_hash")
				}
				continue
			}
			c.tally.Look(ph.rule, dev, 1)
			shown := func(l [][]string) string {
				var out []string
				for _, x := range l {
					out = append(out, strings.Join(x, "/"))
				}
				return strings.Join(out, "、")
			}
			if !agree(mine, theirs) {
				c.add(ph.rule, t, r, ph.col, fmt.Sprintf("%sに、両端で折り合う候補がありません", ph.what),
					model.KV{Name: dev, Value: shown(mine)}, model.KV{Name: other, Value: shown(theirs)})
			} else if !sameList(mine, theirs) {
				c.add("X-366", t, r, ph.col, fmt.Sprintf("%sの候補が、両端で違います（折り合う候補はあります）", ph.what),
					model.KV{Name: dev, Value: shown(mine)}, model.KV{Name: other, Value: shown(theirs)})
			}
		}
		if pa, pb := listOf(r, "pfs", "pfs_more"), listOf(rev, "pfs", "pfs_more"); len(pa) <= 1 && len(pb) <= 1 {
			differ("X-304", "pfs")
		} else {
			c.tally.Look("X-304", dev, 1)
			if !agree(tuples(pa), tuples(pb)) {
				c.add("X-304", t, r, "pfs", "PFS に、両端で折り合うグループがありません",
					model.KV{Name: dev, Value: strings.Join(pa, "、")}, model.KV{Name: other, Value: strings.Join(pb, "、")})
			} else if strings.Join(pa, " ") != strings.Join(pb, " ") {
				c.add("X-366", t, r, "pfs", "PFS の候補が、両端で違います（折り合う候補はあります）",
					model.KV{Name: dev, Value: strings.Join(pa, "、")}, model.KV{Name: other, Value: strings.Join(pb, "、")})
			}
		}
		// AH on one side only, and another mode, are differences too: an empty
		// cell is "no AH" and the tunnel mode.
		for _, e := range [][2]string{{"ah_hash", "（なし）"}, {"ipsec_mode", "tunnel"}} {
			col := e[0]
			if t.Column(col) == nil {
				continue
			}
			if a, b := orElse(r.Get(col), e[1]), orElse(rev.Get(col), e[1]); a != b {
				c.add("X-304", t, r, col, fmt.Sprintf("「%s」が両端で違います", title(col)),
					model.KV{Name: dev, Value: a}, model.KV{Name: other, Value: b})
			}
		}
		differ("X-308", "ike_lifetime", "ipsec_lifetime", "ipsec_lifetime_kb")
		// The key each side proves itself with is the key the other side
		// checks it with ("psk_remote", or the one key of the row). A key in
		// hexadecimal and one written as text cannot be compared.
		c.tally.Look("X-365", dev, 1)
		c.tally.Look("X-305", dev, 1)
		if fa, fb := orElse(r.Get("psk_form"), "text"), orElse(rev.Get("psk_form"), "text"); fa != fb {
			c.add("X-365", t, r, "psk_form", "「鍵の書き方」が両端で違うので、鍵が同じかどうかを比べられません",
				model.KV{Name: dev, Value: fa}, model.KV{Name: other, Value: fb})
		} else if saved := func(v string) bool { return strings.Contains(v, " ") }; saved(r.Get("psk")) || saved(rev.Get("psk")) || saved(r.Get("psk_remote")) || saved(rev.Get("psk_remote")) {
			// A key in the form a device saved it ("<kind> <value>",
			// appendix A 3.26) cannot be compared (V-283 tells the cell).
		} else {
			checks := func(x *model.Row) string { return orElse(x.Get("psk_remote"), x.Get("psk")) }
			col := ""
			if both(r.Get("psk"), checks(rev)) {
				col = "psk"
			} else if both(rev.Get("psk"), checks(r)) {
				col = "psk"
				if r.Get("psk_remote") != "" {
					col = "psk_remote"
				}
			}
			if col != "" {
				c.add("X-305", t, r, col, "事前共有鍵が両端で違います（不一致）",
					model.KV{Name: "こちら", Value: where(dev, r.Get("name"))}, model.KV{Name: "相手", Value: where(other, rev.Get("name"))})
			}
		}
		c.tally.Look("X-309", dev, 1)
		for _, col := range []string{"dpd_interval", "nat_traversal"} {
			a, b := r.Get(col), rev.Get(col)
			if col == "nat_traversal" {
				// "false" says the same as nothing written, and so does a
				// value the pack filled in because its device does so by
				// itself (a device that wraps IPsec in UDP where it finds
				// NAT, without a line that says so).
				a, b = strings.TrimPrefix(a, "false"), strings.TrimPrefix(b, "false")
				if byDefault(r.Cells[col]) {
					a = ""
				}
				if byDefault(rev.Cells[col]) {
					b = ""
				}
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

// byDefault tells that every value of a cell was put in by a pack as the
// default of its device.
func byDefault(c *model.Cell) bool {
	if c == nil || len(c.Origins) == 0 {
		return false
	}
	for _, o := range c.Origins {
		if o.Kind != "default" {
			return false
		}
	}
	return true
}

// listOf gives the values of a column and of its "(more)" column, the first
// value first; nothing for an empty first value.
func listOf(r *model.Row, col, more string) []string {
	v := r.Get(col)
	if v == "" {
		return nil
	}
	out := []string{v}
	if c := r.Cells[more]; c != nil {
		out = append(out, c.Values...)
	}
	return out
}

// tuples makes candidates of one value each.
func tuples(vals []string) [][]string {
	out := make([][]string, len(vals))
	for i, v := range vals {
		out[i] = []string{v}
	}
	return out
}

// candidates lists what a row of IPsec offers for a phase, in the order it
// offers it: the values of the row itself (with IKE version 2, every
// combination of its lists), then the rows of "IPsec more proposals". A
// candidate of IKE is {encryption, hash, DH group}, one of IPsec is
// {encryption, hash, AH hash}; an empty value is left to the device.
func (c *cross) candidates(r *model.Row, phase string) [][]string {
	var out [][]string
	if phase == "ike" {
		encs, hashes, dhs := listOf(r, "ike_encryption", "ike_encryption_more"), listOf(r, "ike_hash", "ike_hash_more"), listOf(r, "ike_dh", "ike_dh_more")
		for _, e := range orEmpty(encs) {
			for _, h := range orEmpty(hashes) {
				for _, d := range orEmpty(dhs) {
					out = append(out, []string{e, h, d})
				}
			}
		}
	} else {
		encs, hashes := listOf(r, "esp_encryption", "esp_encryption_more"), listOf(r, "esp_hash", "esp_hash_more")
		for _, e := range orEmpty(encs) {
			for _, h := range orEmpty(hashes) {
				out = append(out, []string{e, h, r.Get("ah_hash")})
			}
		}
	}
	var more []*model.Row
	for _, x := range c.rows("ipsec_proposals") {
		if x.Get("device") == r.Get("device") && x.Get("tunnel") == r.Get("name") && x.Get("phase") == phase {
			more = append(more, x)
		}
	}
	sort.SliceStable(more, func(i, j int) bool {
		a, _ := strconv.Atoi(more[i].Get("seq"))
		b, _ := strconv.Atoi(more[j].Get("seq"))
		return a < b
	})
	for _, x := range more {
		if phase == "ike" {
			out = append(out, []string{x.Get("encryption"), x.Get("hash"), x.Get("dh")})
		} else {
			out = append(out, []string{x.Get("encryption"), x.Get("hash"), x.Get("ah_hash")})
		}
	}
	return out
}

func orEmpty(l []string) []string {
	if len(l) == 0 {
		return []string{""}
	}
	return l
}

// agree tells that two lists of candidates have one in common: every value
// the same, an empty value going with any.
func agree(a, b [][]string) bool {
	for _, x := range a {
		for _, y := range b {
			ok := true
			for i := range x {
				if x[i] != "" && y[i] != "" && x[i] != y[i] {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
	}
	return false
}

// sameList tells that two lists of candidates hold the same candidates (the
// order is the liking of each side, and may differ).
func sameList(a, b [][]string) bool {
	key := func(l [][]string) string {
		var out []string
		for _, x := range l {
			out = append(out, strings.Join(x, "/"))
		}
		sort.Strings(out)
		return strings.Join(out, ";")
	}
	return key(a) == key(b)
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
		pa, pb := s.Get("local_port"), s.Get("remote_port")
		if flip {
			pa, pb = pb, pa
		}
		if pa != "" {
			a += ":" + pa
		}
		if pb != "" {
			b += ":" + pb
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
		dst, dcol, err := tunnelDst(r)
		if err != nil {
			continue
		}
		c.tally.Look("X-300", dev, 1)
		peer := c.anyOwner(dst, dev)
		if peer == nil {
			c.add("X-300", t, r, dcol, fmt.Sprintf("トンネルの宛先 %s を持つ機器が、案件の中にありません（管理外の相手）", dst),
				model.KV{Name: "宛先", Value: dst.String()})
			continue
		}
		c.tally.Look("X-301", dev, 1)
		local, known := c.srcAddrOf(dev, r.Get("source"), dst.Is6())
		var rev *model.Row
		for _, o := range rows {
			if o.Get("device") != peer.device || o.Get("type") == "ipsec" || mate[o] != nil {
				continue
			}
			back, _, err := tunnelDst(o)
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
			c.add("X-301", t, r, dcol, fmt.Sprintf("相手の %s（%s）に、この機器を宛先にするトンネルの行がありません", peer.device, peer.name),
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
		dstA, colA, errA := tunnelDst(r)
		dstB, _, errB := tunnelDst(rev)
		if r.Get("source") != "" && rev.Get("source") != "" && errA == nil && errB == nil {
			c.tally.Look("X-311", dev, 1)
			srcA, okA := c.srcAddrOf(dev, r.Get("source"), dstB.Is6())
			srcB, okB := c.srcAddrOf(other, rev.Get("source"), dstA.Is6())
			if okA && dstB.IsValid() && srcA != dstB {
				c.add("X-311", t, r, "source", fmt.Sprintf("送信元 %s が、相手の %s のトンネルの宛先 %s と違います", srcA, other, dstB),
					model.KV{Name: where(dev, "の送信元"), Value: srcA.String()}, model.KV{Name: where(other, "の宛先"), Value: dstB.String()})
			}
			if okB && dstA.IsValid() && dstA != srcB {
				c.add("X-311", t, r, colA, fmt.Sprintf("宛先 %s が、相手の %s のトンネルの送信元 %s と違います", dstA, other, srcB),
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
