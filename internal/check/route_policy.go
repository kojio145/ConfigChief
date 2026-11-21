package check

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/model"
)

// The checks that came with the route policies (appendix C 1.1 and 1.2,
// appendix A 3.19): V-221 to V-226, X-361 and X-362.

// policyUses are the columns that name a route policy, and what the policy
// is used for there.
var policyUses = []struct {
	table, column string
	packets       bool // policy routing: the policy looks at packets, not at routes
}{
	{"bgp_neighbors", "filter_in", false},
	{"bgp_neighbors", "filter_out", false},
	{"bgp_neighbors", "default_originate_filter", false},
	{"bgp_ipv6_neighbors", "filter_in", false},
	{"bgp_ipv6_neighbors", "filter_out", false},
	{"ospf_processes", "default_filter", false},
	{"ospf_processes", "filter_in_policy", false},
	{"ospf_redistribute", "filter", false},
	{"bgp_redistribute", "filter", false},
	{"ospfv3_redistribute", "filter", false},
	{"bgp_ipv6_redistribute", "filter", false},
	{"interfaces", "policy_route", true},
	{"devices", "local_policy_route", true},
	{"ipv6_interfaces", "policy_route", true},
	{"devices", "local_policy_route_v6", true},
}

// Columns of a route policy that say something about routes only, and about
// where packets go only. A row that is used for the other purpose does
// nothing with them (V-225).
var (
	routeOnly = []string{"match_prefix_list", "match_next_hop_prefix_list", "match_as_path", "match_community", "match_community_list",
		"match_interface", "match_metric", "match_tag", "match_ipv6_prefix_list", "match_ipv6_next_hop_prefix_list", "match_next_hop_acl", "match_route_type", "continue_to",
		"set_community_delete", "set_extcommunity_rt", "set_extcommunity_soo", "set_local_preference", "set_metric", "set_metric_type", "set_as_path_prepend",
		"set_community", "set_weight", "set_origin", "set_tag"}
	packetOnly = []string{"set_default_next_hop", "set_interface", "set_default_interface", "set_ipv6_default_next_hop"}
)

// filled tells whether a cell holds a value.
func filled(r *model.Row, column string) bool {
	c := r.Cells[column]
	if c == nil {
		return false
	}
	for _, v := range c.Values {
		if v != "" {
			return true
		}
	}
	return false
}

// routePolicies looks at the route policies: the names that point at them
// (V-221), the ones nothing points at (V-222), the lists nothing uses
// (V-223), a policy that lets nothing through (V-224), a row that does
// nothing where it is used (V-225), and values that do not go together
// (V-226).
func (x *values) routePolicies() {
	t := x.in.Schema.Table("route_policies")
	if t == nil {
		return
	}
	type use struct{ routes, packets bool }
	uses := map[string]*use{} // device + NUL + policy
	note := func(dev, name string, packets bool) {
		k := dev + "\x00" + name
		if uses[k] == nil {
			uses[k] = &use{}
		}
		if packets {
			uses[k].packets = true
		} else {
			uses[k].routes = true
		}
	}
	for _, u := range policyUses {
		ut := x.in.Schema.Table(u.table)
		if ut == nil || ut.Column(u.column) == nil {
			continue
		}
		for _, r := range x.rows(u.table) {
			name := r.Get(u.column)
			if name == "" {
				continue
			}
			dev := device(ut, r)
			note(dev, name, u.packets)
			x.tally.Look("V-221", dev, 1)
			if x.find("route_policies", dev, "name", name) == nil {
				x.add("V-221", ut, r, u.column, fmt.Sprintf("「%s」の「%s」は、経路制御の方針の表にありません（方針を追加コンフィグに書いているなら、承認します）", ut.Column(u.column).Title.JA, name),
					model.KV{Name: "値", Value: name})
			}
		}
	}

	type policy struct {
		first   *model.Row
		permits int
	}
	seen := map[string]*policy{}
	var order []string
	for _, r := range x.rows("route_policies") {
		dev, name := device(t, r), r.Get("name")
		if name == "" {
			continue
		}
		k := dev + "\x00" + name
		p := seen[k]
		if p == nil {
			p = &policy{first: r}
			seen[k] = p
			order = append(order, k)
		}
		if r.Get("action") == "permit" {
			p.permits++
		}
		what := fmt.Sprintf("方針 %s の %s 番", name, r.Get("seq"))

		// V-226: values of one row that do not go together.
		x.tally.Look("V-226", dev, 1)
		var sets []string
		for _, c := range t.Columns {
			if strings.HasPrefix(c.Name, "set_") && filled(r, c.Name) {
				sets = append(sets, c.Title.JA)
			}
		}
		if r.Get("action") == "deny" && len(sets) > 0 {
			x.add("V-226", t, r, "action", fmt.Sprintf("%s: 動作が deny の行の「付ける値」は使われません（%s）", what, strings.Join(sets, "、")),
				model.KV{Name: "付ける値", Value: strings.Join(sets, "、")})
		}
		if filled(r, "match_community") && filled(r, "match_community_list") {
			x.add("V-226", t, r, "match_community_list", what+": 「条件: コミュニティ」と「条件: コミュニティのリスト」は、どちらか 1 つだけ書きます",
				model.KV{Name: "リスト", Value: r.Get("match_community_list")})
		}
		if r.Get("match_community_exact") == "true" && !filled(r, "match_community") && !filled(r, "match_community_list") {
			x.add("V-226", t, r, "match_community_exact", what+": 「条件: コミュニティが完全に同じ」は、コミュニティの条件がある行にだけ書けます")
		}
		if c := r.Cells["set_community"]; c != nil && filled(r, "set_community") {
			none := false
			for _, v := range c.Values {
				none = none || v == "none"
			}
			if none && (len(c.Values) > 1 || r.Get("set_community_additive") == "true") {
				x.add("V-226", t, r, "set_community", what+": コミュニティの none（すべて外す）は、ほかの値や「コミュニティを足す」と一緒には書けません",
					model.KV{Name: "値", Value: strings.Join(c.Values, " ")})
			}
		} else if r.Get("set_community_additive") == "true" {
			x.add("V-226", t, r, "set_community_additive", what+": 「コミュニティを足す」は、付けるコミュニティがある行にだけ書けます")
		}

		// V-225: what the row holds against what the policy is used for.
		u := uses[k]
		if u == nil {
			continue
		}
		x.tally.Look("V-225", dev, 1)
		titles := func(cols []string) []string {
			var out []string
			for _, c := range cols {
				if filled(r, c) {
					out = append(out, t.Column(c).Title.JA)
				}
			}
			return out
		}
		if u.packets && !u.routes {
			if cols := titles(routeOnly); len(cols) > 0 {
				x.add("V-225", t, r, "name", fmt.Sprintf("%s: ポリシールーティングに使う方針ですが、経路にしか効かない列に値があります（%s）。通信は、ACL の条件と、ネクストホップ・出力 IF で扱います", what, strings.Join(cols, "、")),
					model.KV{Name: "列", Value: strings.Join(cols, "、")})
			}
		}
		if u.routes && !u.packets {
			if cols := titles(packetOnly); len(cols) > 0 {
				x.add("V-225", t, r, "name", fmt.Sprintf("%s: 経路（BGP・再配布）に使う方針ですが、ポリシールーティングにしか効かない列に値があります（%s）", what, strings.Join(cols, "、")),
					model.KV{Name: "列", Value: strings.Join(cols, "、")})
			}
		}
	}
	for _, k := range order {
		p := seen[k]
		dev, name := device(t, p.first), p.first.Get("name")
		x.tally.Look("V-222", dev, 1)
		if uses[k] == nil {
			x.add("V-222", t, p.first, "name", fmt.Sprintf("経路制御の方針「%s」は、どの表からも使われていません（追加コンフィグなど、表の外で使っているなら承認します）", name),
				model.KV{Name: "値", Value: name})
		}
		x.tally.Look("V-224", dev, 1)
		if p.permits == 0 {
			x.add("V-224", t, p.first, "name", fmt.Sprintf("経路制御の方針「%s」に permit の行が 1 つもありません（当てた所の経路をすべて落とします。ポリシールーティングでは何もしません）", name),
				model.KV{Name: "値", Value: name})
		}
	}

	// V-223: a list no policy names.
	for _, l := range []struct{ table, column string }{{"as_path_lists", "match_as_path"}, {"community_lists", "match_community_list"}} {
		lt := x.in.Schema.Table(l.table)
		if lt == nil {
			continue
		}
		used := map[string]bool{}
		for _, r := range x.rows("route_policies") {
			if v := r.Get(l.column); v != "" {
				used[device(t, r)+"\x00"+v] = true
			}
		}
		// An AS path list may stand right on a BGP neighbor.
		if l.table == "as_path_lists" {
			if nt := x.in.Schema.Table("bgp_neighbors"); nt != nil {
				for _, r := range x.rows("bgp_neighbors") {
					for _, c := range []string{"as_path_filter_in", "as_path_filter_out"} {
						if v := r.Get(c); v != "" {
							used[device(nt, r)+"\x00"+v] = true
						}
					}
				}
			}
		}
		told := map[string]bool{}
		for _, r := range x.rows(l.table) {
			dev, name := device(lt, r), r.Get("name")
			k := dev + "\x00" + name
			if name == "" || told[k] {
				continue
			}
			told[k] = true
			x.tally.Look("V-223", dev, 1)
			if !used[k] {
				x.add("V-223", lt, r, "name", fmt.Sprintf("%s「%s」は、どの経路制御の方針からも（AS パスのリストは、BGP ネイバーからも）使われていません（追加コンフィグなど、表の外で使っているなら承認します）", lt.Title.JA, name),
					model.KV{Name: "値", Value: name})
			}
		}
	}
}

// ---- the pairs ----

// policyRows gives the rows of a policy of a device, in the order of the
// table.
func (c *cross) policyRows(dev, name string) []*model.Row {
	var out []*model.Row
	for _, r := range c.of("route_policies", dev) {
		if r.Get("name") == name {
			out = append(out, r)
		}
	}
	return out
}

// bgpCommunities sets the policy a row of a BGP neighbor takes routes in by
// against the policy the other side sends them out by (X-362). When every
// row of the sender's policy that lets a route through puts its own
// communities in place of the ones the route had, the receiver sees those
// values and no others: a row of the receiver that waits for another value
// never applies.
func (c *cross) bgpCommunities(r, rev *model.Row) {
	t := c.in.Schema.Table("route_policies")
	nt := c.in.Schema.Table("bgp_neighbors")
	if t == nil || nt == nil || r.Get("filter_in") == "" || rev.Get("filter_out") == "" {
		return
	}
	dev, peer := r.Get("device"), rev.Get("device")
	in, out := c.policyRows(dev, r.Get("filter_in")), c.policyRows(peer, rev.Get("filter_out"))
	if len(in) == 0 || len(out) == 0 {
		return
	}
	sent := map[string]bool{}
	permits := 0
	for _, o := range out {
		if o.Get("action") != "permit" {
			continue
		}
		permits++
		cell := o.Cells["set_community"]
		if cell == nil || !filled(o, "set_community") || o.Get("set_community_additive") == "true" {
			return // a route may keep the communities it came with
		}
		for _, v := range cell.Values {
			sent[v] = true
		}
	}
	if permits == 0 {
		return
	}
	var all []string
	for v := range sent {
		all = append(all, v)
	}
	sort.Strings(all)
	for _, i := range in {
		cell := i.Cells["match_community"]
		if cell == nil || !filled(i, "match_community") {
			continue
		}
		c.tally.Look("X-362", dev, 1)
		var missing []string
		for _, v := range cell.Values {
			if !sent[v] {
				missing = append(missing, v)
			}
		}
		if len(missing) > 0 {
			c.add("X-362", t, i, "match_community", fmt.Sprintf("方針 %s の %s 番が待つコミュニティ %s を、相手の %s は送りません（相手の送信の方針 %s は、どの行もコミュニティを %s に置き換えます）",
				i.Get("name"), i.Get("seq"), strings.Join(missing, " "), peer, rev.Get("filter_out"), strings.Join(all, " ")),
				model.KV{Name: where(dev, "が待つ値"), Value: strings.Join(cell.Values, " ")},
				model.KV{Name: where(peer, "が付ける値"), Value: strings.Join(all, " ")})
		}
	}
}

// redistLoops looks for routes that can go round between OSPF and BGP
// (X-361): two or more devices of the case hand the routes of OSPF to BGP
// and those of BGP to OSPF, and one of them does so without a policy that
// could tell the routes it handed over itself.
func (c *cross) redistLoops() {
	ot, bt := c.in.Schema.Table("ospf_redistribute"), c.in.Schema.Table("bgp_redistribute")
	if ot == nil || bt == nil {
		return
	}
	type both struct {
		toOSPF, toBGP *model.Row
	}
	devs := map[string]*both{}
	var order []string
	get := func(dev string) *both {
		if devs[dev] == nil {
			devs[dev] = &both{}
			order = append(order, dev)
		}
		return devs[dev]
	}
	for _, r := range c.rows("ospf_redistribute") {
		if r.Get("source") == "bgp" {
			if b := get(r.Get("device")); b.toOSPF == nil || (b.toOSPF.Get("filter") != "" && r.Get("filter") == "") {
				b.toOSPF = r
			}
		}
	}
	for _, r := range c.rows("bgp_redistribute") {
		if r.Get("source") == "ospf" {
			if b := get(r.Get("device")); b.toBGP == nil || (b.toBGP.Get("filter") != "" && r.Get("filter") == "") {
				b.toBGP = r
			}
		}
	}
	var mutual []string
	for _, dev := range order {
		if b := devs[dev]; b.toOSPF != nil && b.toBGP != nil {
			mutual = append(mutual, dev)
		}
	}
	if len(mutual) < 2 {
		return
	}
	for _, dev := range mutual {
		b := devs[dev]
		c.tally.Look("X-361", dev, 1)
		var others []string
		for _, o := range mutual {
			if o != dev {
				others = append(others, o)
			}
		}
		ev := model.KV{Name: "同じことをしている機器", Value: strings.Join(others, "、")}
		switch {
		case b.toOSPF.Get("filter") == "":
			c.add("X-361", ot, b.toOSPF, "filter", fmt.Sprintf("%s は BGP と OSPF の間で互いに再配布していて、BGP → OSPF に経路制御の条件がありません。同じことをする機器（%s）がほかにもあるので、経路が回るおそれがあります", dev, strings.Join(others, "、")), ev)
		case b.toBGP.Get("filter") == "":
			c.add("X-361", bt, b.toBGP, "filter", fmt.Sprintf("%s は BGP と OSPF の間で互いに再配布していて、OSPF → BGP に経路制御の条件がありません。同じことをする機器（%s）がほかにもあるので、経路が回るおそれがあります", dev, strings.Join(others, "、")), ev)
		}
	}
}
