package check

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"github.com/kojio145/configchief/internal/model"
)

// The checks of IPv6 (appendix A 3.23, appendix C 1.1 and 1.2): the IPv6
// versions of V-201, V-202 and V-204, what the IPv6 tables need of the other
// tables (V-264 to V-268), and the pairs of OSPFv3 and of BGP over IPv6
// (X-391 to X-399).

// ipv6ValueIDs and ipv6CrossIDs are the checks of this file.
var (
	ipv6ValueIDs = []string{"V-261", "V-262", "V-263", "V-264", "V-265", "V-266", "V-267", "V-268", "V-269"}
	ipv6CrossIDs = []string{"X-391", "X-392", "X-393", "X-394", "X-395", "X-396", "X-397", "X-398", "X-399", "X-404"}
)

// addrs6 gives the IPv6 addresses of the interfaces, in the order of the
// rows: the fixed addresses, and the prefixes an interface takes an EUI-64
// address from (the address itself is not known: host is false).
func (x *values) addrs6() []if6Addr {
	var list []if6Addr
	vrfOf := map[string]string{}
	for _, r := range x.rows("interfaces") {
		dev, name, vrf := r.Get("device"), r.Get("name"), r.Get("vrf")
		vrfOf[dev+"\x00"+name] = vrf
		cell := r.Cells["ipv6"]
		if cell == nil {
			continue
		}
		for _, v := range cell.Values {
			if p, err := netip.ParsePrefix(v); err == nil && p.Addr().Is6() {
				list = append(list, if6Addr{ifAddr{r, "ipv6", dev, name, vrf, p}, "interfaces", true})
			}
		}
	}
	for _, r := range x.rows("ipv6_interfaces") {
		dev, name := r.Get("device"), r.Get("if_name")
		cell := r.Cells["eui64"]
		if cell == nil {
			continue
		}
		for _, v := range cell.Values {
			if p, err := netip.ParsePrefix(v); err == nil && p.Addr().Is6() {
				list = append(list, if6Addr{ifAddr{r, "eui64", dev, name, vrfOf[dev+"\x00"+name], p}, "ipv6_interfaces", false})
			}
		}
	}
	return list
}

type if6Addr struct {
	ifAddr
	table string
	host  bool // the address is written out (not made by EUI-64)
}

// linkLocal tells an address of fe80::/10.
func linkLocal(a netip.Addr) bool { return a.Is6() && a.IsLinkLocalUnicast() }

// ipv6 runs the checks of the values of the IPv6 tables.
func (x *values) ipv6() {
	for _, id := range ipv6ValueIDs {
		x.tally.Look(id, "", 0)
	}
	list := x.addrs6()

	// V-261 and V-262: the subnets and the addresses.
	seen := map[string]if6Addr{}
	for i, a := range list {
		t := x.in.Schema.Table(a.table)
		if t == nil {
			continue
		}
		x.tally.Look("V-261", a.device, 1)
		for _, b := range list[:i] {
			if b.device == a.device && b.vrf == a.vrf && b.name != a.name && a.prefix.Masked().Overlaps(b.prefix.Masked()) {
				x.add("V-261", t, a.row, a.column, fmt.Sprintf("%s の IPv6 のサブネットと重なっています", b.name),
					model.KV{Name: a.name, Value: a.prefix.String()}, model.KV{Name: b.name, Value: b.prefix.String()})
				break
			}
		}
		if !a.host {
			continue
		}
		x.tally.Look("V-262", a.device, 1)
		key := a.vrf + "\x00" + a.prefix.Addr().String()
		if b, ok := seen[key]; ok && b.row != a.row {
			x.add("V-262", t, a.row, a.column, fmt.Sprintf("同じ IPv6 アドレスが %s にも付いています", b.where()),
				model.KV{Name: a.where(), Value: a.prefix.String()}, model.KV{Name: b.where(), Value: b.prefix.String()})
		} else if !ok {
			seen[key] = a
		}
	}

	// V-263 and V-264: the next hops of the static routes. A device whose
	// address comes from outside (autoconfig, DHCPv6, a delegated prefix) is
	// not looked at for V-263: the tool cannot know its subnets.
	nets := map[string][]netip.Prefix{}
	for _, a := range list {
		key := a.device + "\x00" + a.vrf
		nets[key] = append(nets[key], a.prefix.Masked())
	}
	dynamic := map[string]bool{}
	for _, r := range x.rows("ipv6_interfaces") {
		if r.Get("autoconfig") == "true" || r.Get("dhcp_address") == "true" || r.Get("pd_from") != "" || r.Get("nd_proxy") != "" {
			dynamic[r.Get("device")] = true
		}
	}
	if t := x.in.Schema.Table("ipv6_routes"); t != nil {
		for _, r := range x.rows("ipv6_routes") {
			dev, vrf := r.Get("device"), r.Get("vrf")
			hop, err := netip.ParseAddr(r.Get("next_hop"))
			if err != nil {
				continue
			}
			x.tally.Look("V-264", dev, 1)
			if linkLocal(hop) {
				if r.Get("out_if") == "" {
					x.add("V-264", t, r, "next_hop", fmt.Sprintf("ネクストホップ %s はリンクローカルアドレスですが、出力 IF がありません", hop), model.KV{Name: "値", Value: hop.String()})
				}
				continue
			}
			if r.Get("out_if") != "" || dynamic[dev] {
				continue
			}
			x.tally.Look("V-263", dev, 1)
			reached := false
			for _, p := range nets[dev+"\x00"+vrf] {
				if p.Contains(hop) {
					reached = true
					break
				}
			}
			for _, o := range x.of("ipv6_routes", dev) {
				if reached {
					break
				}
				if o == r || o.Get("vrf") != vrf {
					continue
				}
				if p, err := netip.ParsePrefix(o.Get("prefix")); err == nil && p.Bits() > 0 && p.Contains(hop) {
					reached = true
				}
			}
			if !reached {
				x.add("V-263", t, r, "next_hop", fmt.Sprintf("ネクストホップ %s は、この機器のどの IF の IPv6 のサブネットにも入りません", hop), model.KV{Name: "値", Value: hop.String()})
			}
		}
	}

	// V-265: a row of an IPv6 table names an interface that has a row.
	for _, name := range []string{"ipv6_interfaces", "ipv6_ra_prefixes", "ospfv3_interfaces"} {
		t := x.in.Schema.Table(name)
		if t == nil {
			continue
		}
		for _, r := range x.rows(name) {
			dev, ifn := r.Get("device"), r.Get("if_name")
			if ifn == "" {
				continue
			}
			x.tally.Look("V-265", dev, 1)
			if x.find("interfaces", dev, "name", ifn) == nil {
				x.add("V-265", t, r, "if_name", fmt.Sprintf("「%s」の IF %s が、インタフェースの表にありません", t.Title.JA, ifn), model.KV{Name: "値", Value: ifn})
			}
		}
	}

	// V-266: an address made from a delegated prefix needs an interface of
	// the device that asks for the prefix by the same name.
	if t := x.in.Schema.Table("ipv6_interfaces"); t != nil {
		for _, r := range x.rows("ipv6_interfaces") {
			dev, from := r.Get("device"), r.Get("pd_from")
			if from == "" {
				continue
			}
			x.tally.Look("V-266", dev, 1)
			if x.find("ipv6_interfaces", dev, "pd_request", from) == nil {
				x.add("V-266", t, r, "pd_from", fmt.Sprintf("プレフィックスの名前 %s を、「プレフィックスをもらう（DHCPv6-PD）」に書いた IF が、この機器にありません", from), model.KV{Name: "値", Value: from})
			}
		}
	}

	// V-269: whether an interface sends router advertisements when nothing
	// is said differs by vendor. Told for an Ethernet interface that has a
	// subnet of its own (a fixed address, an EUI-64 prefix or a delegated
	// prefix) and says nothing.
	if t := x.in.Schema.Table("interfaces"); t != nil {
		own := map[string]bool{}
		for _, a := range list {
			if a.prefix.Bits() < 127 {
				own[a.device+"\x00"+a.name] = true
			}
		}
		said := map[string]bool{}
		for _, r := range x.rows("ipv6_interfaces") {
			k := r.Get("device") + "\x00" + r.Get("if_name")
			if r.Get("pd_from") != "" {
				own[k] = true
			}
			if r.Get("ra") != "" {
				said[k] = true
			}
		}
		for _, r := range x.rows("interfaces") {
			dev, name := r.Get("device"), r.Get("name")
			k := dev + "\x00" + name
			switch r.Get("kind") {
			case "", "physical", "subif", "vlan", "lag":
			default:
				continue
			}
			if !own[k] {
				continue
			}
			x.tally.Look("V-269", dev, 1)
			if !said[k] {
				x.add("V-269", t, r, "name", fmt.Sprintf("%s は IPv6 のサブネットを持ちますが、「RA を送る」（シート「IPv6 インタフェース」）が空です（何も書かないとき、Cisco は送り、NEC IX は送りません）", name))
			}
		}
	}

	// V-267: an interface or an area of OSPFv3 names a process that has a row.
	for _, name := range []string{"ospfv3_interfaces", "ospfv3_areas"} {
		t := x.in.Schema.Table(name)
		if t == nil {
			continue
		}
		for _, r := range x.rows(name) {
			dev, pid := r.Get("device"), r.Get("process_id")
			if pid == "" {
				continue
			}
			x.tally.Look("V-267", dev, 1)
			if x.find("ospfv3_processes", dev, "process_id", pid) == nil {
				x.add("V-267", t, r, "process_id", fmt.Sprintf("プロセス番号 %s の行が、OSPFv3 の表にありません", pid), model.KV{Name: "値", Value: pid})
			}
		}
	}

	// V-268: where a tunnel leads has to fit its type.
	if t := x.in.Schema.Table("tunnels"); t != nil {
		for _, r := range x.rows("tunnels") {
			dev, typ := r.Get("device"), r.Get("type")
			v6, fqdn := r.Get("destination_v6"), r.Get("destination_fqdn")
			switch typ {
			case "ipv4_over_ipv6":
				x.tally.Look("V-268", dev, 1)
				if v6 == "" && fqdn == "" {
					x.add("V-268", t, r, "destination_v6", "種類が ipv4_over_ipv6 のトンネルに、「宛先（IPv6）」も「宛先（名前）」もありません")
				} else if v6 != "" && fqdn != "" {
					x.add("V-268", t, r, "destination_fqdn", "「宛先（IPv6）」と「宛先（名前）」の両方が書かれています（どちらか片方です）",
						model.KV{Name: "宛先（IPv6）", Value: v6}, model.KV{Name: "宛先（名前）", Value: fqdn})
				}
			case "map_e", "auto_ds_lite", "auto_ipip":
				x.tally.Look("V-268", dev, 1)
				for _, col := range []string{"destination", "destination_v6", "destination_fqdn"} {
					if v := r.Get(col); v != "" {
						x.add("V-268", t, r, col, fmt.Sprintf("種類が %s のトンネルに宛先は書けません（宛先は回線の事業者からもらいます）", typ), model.KV{Name: "値", Value: v})
					}
				}
			default:
				// A tunnel of IPsec may name its peer (appendix A 3.20), and
				// EtherIP may run over IPv6 (appendix A 3.26).
				if typ == "ipsec" {
					fqdn = ""
				}
				if typ == "etherip" {
					v6 = ""
				}
				if v6 != "" || fqdn != "" {
					x.tally.Look("V-268", dev, 1)
					col := "destination_v6"
					if v6 == "" {
						col = "destination_fqdn"
					}
					x.add("V-268", t, r, col, fmt.Sprintf("「宛先（IPv6）」「宛先（名前）」は、種類が ipv4_over_ipv6 のトンネルに書けます（ほかに、ipsec の「宛先（名前）」と etherip の「宛先（IPv6）」。この行の種類は %s）", typ))
				}
			}
		}
	}
}

// ---- the pairs ----

// x6If is an interface with its IPv6 subnets.
type x6If struct {
	*xIf
	nets  []netip.Prefix
	hosts []netip.Addr
}

func shared6(a, b *x6If) bool {
	if a.device == b.device || a.vrf != b.vrf {
		return false
	}
	for _, pa := range a.nets {
		for _, pb := range b.nets {
			if pa == pb {
				return true
			}
		}
	}
	return false
}

// ipv6 runs the checks of the pairs of OSPFv3 and of BGP over IPv6.
func (c *cross) ipv6() {
	for _, id := range ipv6CrossIDs {
		c.tally.Look(id, "", 0)
	}
	by := map[string]*x6If{}
	var all []*x6If
	get := func(dev, name string) *x6If {
		k := dev + "\x00" + name
		if i := by[k]; i != nil {
			return i
		}
		base := c.byDev[dev][name]
		if base == nil {
			return nil
		}
		i := &x6If{xIf: base}
		by[k] = i
		all = append(all, i)
		return i
	}
	for _, a := range c.addrs6() {
		i := get(a.device, a.name)
		if i == nil {
			continue
		}
		i.nets = append(i.nets, a.prefix.Masked())
		if a.host {
			i.hosts = append(i.hosts, a.prefix.Addr())
		}
	}
	c.ospfv3(get, all)
	c.bgp6(all)
}

type ospf3If struct {
	row  *model.Row
	i    *x6If
	proc *model.Row
}

func (o ospf3If) passive() bool {
	if v := o.row.Get("passive"); v != "" {
		return v == "true"
	}
	return o.proc != nil && o.proc.Get("passive_default") == "true"
}

func (c *cross) ospfv3(get func(dev, name string) *x6If, all []*x6If) {
	t := c.in.Schema.Table("ospfv3_interfaces")
	if t == nil {
		return
	}
	var list []ospf3If
	member := map[*x6If]bool{}
	for _, r := range c.rows("ospfv3_interfaces") {
		i := get(r.Get("device"), r.Get("if_name"))
		if i == nil || len(i.nets) == 0 {
			continue // an interface with a link-local address only cannot be paired
		}
		list = append(list, ospf3If{r, i, c.find("ospfv3_processes", i.device, "process_id", r.Get("process_id"))})
		member[i] = true
	}
	areaType := func(o ospf3If) string {
		for _, a := range c.rows("ospfv3_areas") {
			if a.Get("device") == o.i.device && a.Get("process_id") == o.row.Get("process_id") && area(a.Get("area")) == area(o.row.Get("area")) && a.Get("type") != "" {
				return a.Get("type")
			}
		}
		return "normal"
	}
	// The authentication an interface uses (2026-11-21): its own, or that of
	// its area unless it says none.
	auth := func(o ospf3If) string {
		pick := func(r *model.Row) string {
			return r.Get("auth_hash") + " " + r.Get("auth_spi") + " " + r.Get("auth_key")
		}
		switch o.row.Get("auth_hash") {
		case "none":
			return ""
		case "":
		default:
			return pick(o.row)
		}
		for _, a := range c.rows("ospfv3_areas") {
			if a.Get("device") == o.i.device && a.Get("process_id") == o.row.Get("process_id") && area(a.Get("area")) == area(o.row.Get("area")) && a.Get("auth_hash") != "" {
				return pick(a)
			}
		}
		return ""
	}
	for n, a := range list {
		dev := a.i.device
		c.tally.Look("X-396", dev, 1)
		for _, o := range all {
			if shared6(a.i, o) && !member[o] {
				c.add("X-396", t, a.row, "if_name", fmt.Sprintf("同じ IPv6 のサブネットの %s の %s は、OSPFv3 に入っていません", o.device, o.name),
					model.KV{Name: where(dev, a.i.name), Value: "OSPFv3 " + a.row.Get("process_id")}, model.KV{Name: where(o.device, o.name), Value: "OSPFv3 なし"})
			}
		}
		for _, b := range list[n+1:] {
			if !shared6(a.i, b.i) {
				continue
			}
			for _, id := range []string{"X-391", "X-392", "X-393", "X-394", "X-395"} {
				c.tally.Look(id, dev, 1)
			}
			wa, wb := where(dev, a.i.name), where(b.i.device, b.i.name)
			diff := func(rule, col, msg, va, vb string) {
				c.add(rule, t, a.row, col, msg, model.KV{Name: wa, Value: va}, model.KV{Name: wb, Value: vb})
			}
			if x, y := area(a.row.Get("area")), area(b.row.Get("area")); both(x, y) {
				diff("X-391", "area", fmt.Sprintf("同じ IPv6 のサブネットの %s と OSPFv3 のエリアが違います", wb), a.row.Get("area"), b.row.Get("area"))
			} else if x == y {
				if x, y := areaType(a), areaType(b); x != y {
					diff("X-394", "area", fmt.Sprintf("OSPFv3 のエリア %s の種類が、%s と違います", a.row.Get("area"), b.i.device), x, y)
				}
			}
			for _, col := range []string{"hello", "dead"} {
				if x, y := a.row.Get(col), b.row.Get(col); both(x, y) {
					diff("X-392", col, fmt.Sprintf("OSPFv3 の「%s」が、同じ IPv6 のサブネットの %s と違います", t.Column(col).Title.JA, wb), x, y)
				}
			}
			if x, y := a.row.Get("network_type"), b.row.Get("network_type"); both(x, y) {
				diff("X-393", "network_type", fmt.Sprintf("OSPFv3 の網の種類が、同じ IPv6 のサブネットの %s と違います", wb), x, y)
			}
			c.tally.Look("X-404", dev, 1)
			if x, y := auth(a), auth(b); x != y {
				show := func(v string) string {
					if v == "" {
						return "認証なし"
					}
					f := strings.Fields(v)
					return f[0] + " SPI " + f[1]
				}
				msg := fmt.Sprintf("OSPFv3 の認証（アルゴリズム・SPI・鍵）が、同じ IPv6 のサブネットの %s と違います", wb)
				if x == "" || y == "" {
					msg = fmt.Sprintf("同じ IPv6 のサブネットの %s と、片側だけ OSPFv3 の認証があります", wb)
				}
				diff("X-404", "auth_hash", msg, show(x), show(y))
			}
			if a.passive() != b.passive() {
				diff("X-395", "passive", fmt.Sprintf("同じ IPv6 のサブネットの %s と、片側だけ OSPFv3 が受動になっています", wb), strconv.FormatBool(a.passive()), strconv.FormatBool(b.passive()))
			}
		}
	}
}

// bgp6 pairs a neighbor row with the device that has its address, and with
// the row of that device that points back.
func (c *cross) bgp6(all []*x6If) {
	t := c.in.Schema.Table("bgp_ipv6_neighbors")
	if t == nil {
		return
	}
	owner := func(addr netip.Addr, not string) *x6If {
		for _, i := range all {
			if i.device == not {
				continue
			}
			for _, h := range i.hosts {
				if h == addr {
					return i
				}
			}
		}
		return nil
	}
	has := func(dev string, addr netip.Addr) bool {
		for _, i := range all {
			if i.device != dev {
				continue
			}
			for _, h := range i.hosts {
				if h == addr {
					return true
				}
			}
		}
		return false
	}
	asn := func(dev string) string {
		if p := c.find("bgp_processes", dev, "vrf", ""); p != nil {
			return p.Get("asn")
		}
		return ""
	}
	done := map[*model.Row]bool{}
	for _, r := range c.rows("bgp_ipv6_neighbors") {
		dev := r.Get("device")
		addr, err := netip.ParseAddr(r.Get("neighbor"))
		if err != nil {
			continue
		}
		peer := owner(addr, dev)
		if peer == nil {
			continue // not a device of the case
		}
		c.tally.Look("X-397", dev, 1)
		c.tally.Look("X-398", dev, 1)
		if pa, ras := asn(peer.device), r.Get("remote_as"); pa != "" && ras != "" && pa != ras {
			c.add("X-397", t, r, "remote_as", fmt.Sprintf("相手 AS 番号 %s が、相手の %s の AS 番号 %s と違います", ras, peer.device, pa),
				model.KV{Name: where(dev, "相手 AS 番号"), Value: ras}, model.KV{Name: where(peer.device, "AS 番号"), Value: pa})
		}
		var rev *model.Row
		for _, o := range c.rows("bgp_ipv6_neighbors") {
			if o.Get("device") != peer.device {
				continue
			}
			if back, err := netip.ParseAddr(o.Get("neighbor")); err == nil && has(dev, back) {
				rev = o
				break
			}
		}
		if rev == nil {
			c.add("X-398", t, r, "neighbor", fmt.Sprintf("相手の %s に、この機器を指す BGP IPv6 ネイバーの行がありません", peer.device),
				model.KV{Name: where(dev, "ネイバーアドレス"), Value: addr.String()})
			continue
		}
		if done[r] {
			continue
		}
		done[rev] = true
		c.tally.Look("X-399", dev, 1)
		if r.Get("password") != rev.Get("password") {
			c.add("X-399", t, r, "password", fmt.Sprintf("パスワードが、相手の %s と違います（不一致）", peer.device))
		}
	}
}
