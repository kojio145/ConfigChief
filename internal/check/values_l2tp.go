package check

import (
	"fmt"
	"net/netip"
	"regexp"

	"github.com/kojio145/configchief/internal/model"
)

// The checks of the values of the L2TP tables (appendix C 1.1; appendix A
// 3.21): V-241 to V-245, and V-235 (the row of IPsec an LNS names).

// l2tpIDs are the checks l2tp runs.
var l2tpIDs = []string{"V-235", "V-241", "V-242", "V-243", "V-244", "V-245"}

// cookieForm is how a cookie of L2TPv3 is written: the size, then the value
// in one or two parts.
var cookieForm = regexp.MustCompile(`^(4 \d+|8 \d+( \d+)?)$`)

// l2tp looks at the rows of L2TP: what a role needs (V-241), a password
// without the authentication (V-242), the pool and the callers (V-243), and
// the row of IPsec that protects the callers of an LNS (V-235).
func (x *values) l2tp() {
	t := x.in.Schema.Table("l2tp_groups")
	if t == nil {
		return
	}
	title := func(col string) string { return t.Column(col).Title.JA }
	for _, r := range x.rows("l2tp_groups") {
		dev, name := device(t, r), r.Get("name")
		x.tally.Look("V-241", dev, 1)
		switch r.Get("role") {
		case "lac":
			if r.Get("peer_address") == "" {
				x.add("V-241", t, r, "peer_address", fmt.Sprintf("L2TP %s は役割が lac ですが、「%s」（LNS のアドレス）が空です", name, title("peer_address")))
			}
		case "lns":
			if c := r.Cells["logical_ifs"]; c == nil || len(c.Values) == 0 {
				x.add("V-241", t, r, "logical_ifs", fmt.Sprintf("L2TP %s は役割が lns ですが、「%s」が空です", name, title("logical_ifs")))
			}
		}
		// The row of IPsec an LNS names takes callers whose address is not
		// known, and wraps L2TP as it is (transport mode); a row that is not
		// there is V-106.
		x.tally.Look("V-235", dev, 1)
		if p := r.Get("ipsec_policy"); p != "" && r.Get("role") != "lac" {
			if sec := x.find("ipsec_tunnels", dev, "name", p); sec != nil {
				st := x.in.Schema.Table("ipsec_tunnels")
				for _, w := range []struct{ col, want string }{{"style", "policy"}, {"peer_address", "any"}, {"ipsec_mode", "transport"}} {
					if got := sec.Get(w.col); got != w.want {
						shown := got
						if shown == "" {
							shown = "（空）"
						}
						x.add("V-235", t, r, "ipsec_policy", fmt.Sprintf("L2TP %s の「%s」が指す IPsec %s は、「%s」が %s です（L2TP の受け口を守る行は %s です）", name, title("ipsec_policy"), p, st.Column(w.col).Title.JA, shown, w.want),
							model.KV{Name: st.Column(w.col).Title.JA, Value: got})
					}
				}
			}
		}
		x.tally.Look("V-242", dev, 1)
		if r.Get("tunnel_auth") == "false" && r.Get("password") != "" {
			x.add("V-242", t, r, "password", fmt.Sprintf("L2TP %s は「%s」が無効ですが、「%s」があります", name, title("tunnel_auth"), title("password")))
		}
		x.tally.Look("V-243", dev, 1)
		lo, hi := r.Get("pool_start"), r.Get("pool_end")
		switch {
		case lo == "" && hi == "":
		case lo == "" || hi == "":
			col := "pool_start"
			if hi == "" {
				col = "pool_end"
			}
			x.add("V-243", t, r, col, fmt.Sprintf("L2TP %s の払い出すアドレスは、先頭と末尾の片方しかありません", name))
		default:
			a, errA := netip.ParseAddr(lo)
			b, errB := netip.ParseAddr(hi)
			if errA == nil && errB == nil && b.Less(a) {
				x.add("V-243", t, r, "pool_start", fmt.Sprintf("L2TP %s の払い出すアドレスの先頭 %s が、末尾 %s より後ろにあります", name, a, b),
					model.KV{Name: "先頭", Value: lo}, model.KV{Name: "末尾", Value: hi})
			}
		}
	}
	if ut := x.in.Schema.Table("l2tp_users"); ut != nil {
		for _, u := range x.rows("l2tp_users") {
			dev := device(ut, u)
			g := x.find("l2tp_groups", dev, "name", u.Get("group"))
			if g == nil {
				continue // V-106
			}
			x.tally.Look("V-243", dev, 1)
			if role := g.Get("role"); role != "" && role != "lns" {
				x.add("V-243", ut, u, "group", fmt.Sprintf("L2TP 接続ユーザ %s の L2TP「%s」は、役割が %s です（利用者を確かめるのは lns です）", u.Get("name"), u.Get("group"), role),
					model.KV{Name: "役割", Value: role})
			}
		}
	}
}

// l2tpv3 looks at the rows of L2TPv3: the way a session is set up and the
// values that go with it (V-244), a profile with two settings and a circuit
// written twice (V-245).
func (x *values) l2tpv3() {
	t := x.in.Schema.Table("l2tpv3")
	if t == nil {
		return
	}
	title := func(col string) string { return t.Column(col).Title.JA }
	same := []string{"local_if", "signaling", "password", "local_name", "hello", "sequencing", "pmtu", "df_bit"}
	rows := x.rows("l2tpv3")
	for n, r := range rows {
		dev, name := device(t, r), r.Get("if_name")
		x.tally.Look("V-244", dev, 1)
		switch r.Get("signaling") {
		case "manual":
			for _, col := range []string{"local_session_id", "remote_session_id"} {
				if r.Get(col) == "" {
					x.add("V-244", t, r, col, fmt.Sprintf("L2 延伸 %s は張り方が manual ですが、「%s」が空です", name, title(col)))
				}
			}
			for _, col := range []string{"password", "local_name"} {
				if r.Get(col) != "" {
					x.add("V-244", t, r, col, fmt.Sprintf("L2 延伸 %s は張り方が manual ですが、「%s」があります（制御の通信をしないので、使われません）", name, title(col)))
				}
			}
		case "dynamic":
			for _, col := range []string{"local_session_id", "remote_session_id", "local_cookie", "remote_cookie"} {
				if r.Get(col) != "" {
					x.add("V-244", t, r, col, fmt.Sprintf("L2 延伸 %s は張り方が dynamic ですが、「%s」があります（両端が自分で決めるので、書けません）", name, title(col)))
				}
			}
		}
		for _, col := range []string{"local_cookie", "remote_cookie"} {
			if v := r.Get(col); v != "" && !cookieForm.MatchString(v) {
				x.add("V-244", t, r, col, fmt.Sprintf("L2 延伸 %s の「%s」の形が違います（大きさ 4 と値 1 つ、または大きさ 8 と値 1〜2 つを、空白で区切ります。例 4 54321）", name, title(col)),
					model.KV{Name: "値", Value: v})
			}
		}

		x.tally.Look("V-245", dev, 1)
		for _, o := range rows[:n] {
			if device(t, o) != dev {
				continue
			}
			if o.Get("peer_address") == r.Get("peer_address") && o.Get("vc_id") == r.Get("vc_id") && r.Get("vc_id") != "" {
				x.add("V-245", t, r, "vc_id", fmt.Sprintf("L2 延伸 %s は、%s と相手アドレスも回線番号も同じです（1 つの回線番号は、1 つの IF だけがつなげます）", name, o.Get("if_name")),
					model.KV{Name: "相手アドレス", Value: r.Get("peer_address")}, model.KV{Name: "回線番号", Value: r.Get("vc_id")})
			}
			if p := r.Get("profile"); p != "" && o.Get("profile") == p {
				for _, col := range same {
					if o.Get(col) != r.Get(col) {
						x.add("V-245", t, r, col, fmt.Sprintf("L2 延伸 %s は、%s と設定名が同じ「%s」ですが、「%s」が違います", name, o.Get("if_name"), p, title(col)),
							model.KV{Name: name, Value: shown(t.Column(col), r.Get(col))}, model.KV{Name: o.Get("if_name"), Value: shown(t.Column(col), o.Get(col))})
					}
				}
			}
		}
	}
}
