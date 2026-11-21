package check

import (
	"fmt"
	"net/netip"
	"strconv"

	"github.com/kojio145/configchief/internal/model"
)

// The checks of the third round of IPv6 and routing (appendix A 3.23 and
// 3.27, appendix C 1.1 and 1.2): what a DHCPv6 server hands out (V-227), the
// rows of RIP (V-228) and its pairs (X-363, X-364).

// l3ValueIDs and l3CrossIDs are the checks of this file.
var (
	l3ValueIDs = []string{"V-227", "V-228"}
	l3CrossIDs = []string{"X-363", "X-364"}
)

// l3 runs the checks of the values.
func (x *values) l3() {
	for _, id := range l3ValueIDs {
		x.tally.Look(id, "", 0)
	}
	x.dhcp6Delegation()
	x.rip()
}

// l3 runs the checks between devices.
func (c *cross) l3() {
	for _, id := range l3CrossIDs {
		c.tally.Look(id, "", 0)
	}
	c.ripPairs()
}

// dhcp6Delegation is V-227: the columns of what a DHCPv6 server hands out
// have to fit each other.
func (x *values) dhcp6Delegation() {
	t := x.in.Schema.Table("ipv6_interfaces")
	if t == nil {
		return
	}
	asked := map[string]bool{} // device, name of a prefix that is asked for
	for _, r := range x.rows("ipv6_interfaces") {
		if n := r.Get("pd_request"); n != "" {
			asked[r.Get("device")+"\x00"+n] = true
		}
	}
	has := func(r *model.Row, col string) bool {
		c := r.Cells[col]
		return c != nil && len(c.Values) > 0 && c.Values[0] != ""
	}
	for _, r := range x.rows("ipv6_interfaces") {
		dev := r.Get("device")
		var set []string
		for _, col := range []string{"dhcp_pd_pool", "dhcp_pd_length", "dhcp_pd_from", "dhcp_pd_valid", "dhcp_pd_preferred", "dhcp_address_prefix", "dhcp_ntp"} {
			if has(r, col) {
				set = append(set, col)
			}
		}
		if len(set) == 0 {
			continue
		}
		x.tally.Look("V-227", dev, 1)
		if r.Get("dhcp_server") == "" {
			x.add("V-227", t, r, set[0], fmt.Sprintf("「%s」は、「DHCPv6 サーバ（名前）」のある行にだけ書けます", t.Column(set[0]).Title.JA))
			continue
		}
		pool, length := r.Get("dhcp_pd_pool"), r.Get("dhcp_pd_length")
		switch {
		case (pool == "") != (length == ""):
			col := "dhcp_pd_pool"
			if pool != "" {
				col = "dhcp_pd_length"
			}
			x.add("V-227", t, r, col, "「下流へ配るプレフィックスの元」と「下流へ配る長さ」は、組で書きます",
				model.KV{Name: "元", Value: pool}, model.KV{Name: "長さ", Value: length})
		case pool != "":
			p, err := netip.ParsePrefix(pool)
			n, err2 := strconv.Atoi(length)
			if err == nil && err2 == nil && n < p.Bits() {
				x.add("V-227", t, r, "dhcp_pd_length", fmt.Sprintf("「下流へ配る長さ」%d が、元のプレフィックス %s の長さより短くなっています", n, pool))
			}
		}
		if from := r.Get("dhcp_pd_from"); from != "" {
			if pool == "" && length == "" {
				x.add("V-227", t, r, "dhcp_pd_from", "「上流からもらったプレフィックスから配る」には、「下流へ配るプレフィックスの元」（もらったプレフィックスの下の部分）と「下流へ配る長さ」が要ります")
			}
			if !asked[dev+"\x00"+from] {
				x.add("V-227", t, r, "dhcp_pd_from", fmt.Sprintf("プレフィックスの名前 %s が、この機器のどの IF の「プレフィックスをもらう（DHCPv6-PD）」にもありません", from))
			}
		}
		valid, pref := r.Get("dhcp_pd_valid"), r.Get("dhcp_pd_preferred")
		switch {
		case (valid == "") != (pref == ""):
			col := "dhcp_pd_valid"
			if valid != "" {
				col = "dhcp_pd_preferred"
			}
			x.add("V-227", t, r, col, "「配るプレフィックスの有効時間」と「推奨時間」は、組で書きます")
		case valid != "" && pool == "":
			x.add("V-227", t, r, "dhcp_pd_valid", "有効時間・推奨時間は、下流へプレフィックスを配る行にだけ書けます")
		case valid != "" && valid != "infinite":
			v, err := strconv.Atoi(valid)
			p, err2 := strconv.Atoi(pref)
			if err == nil && (pref == "infinite" || (err2 == nil && p > v)) {
				x.add("V-227", t, r, "dhcp_pd_preferred", "推奨時間が、有効時間より長くなっています",
					model.KV{Name: "有効時間", Value: valid}, model.KV{Name: "推奨時間", Value: pref})
			}
		}
	}
}

// rip is V-228: the rows of RIP have to fit each other.
func (x *values) rip() {
	procs := map[string]*model.Row{} // device, family
	for _, r := range x.rows("rip_processes") {
		procs[r.Get("device")+"\x00"+r.Get("family")] = r
	}
	v4only := func(table string, r *model.Row, cols ...string) {
		t := x.in.Schema.Table(table)
		if t == nil || r.Get("family") != "ipv6" {
			return
		}
		for _, col := range cols {
			if filled(r, col) {
				x.add("V-228", t, r, col, fmt.Sprintf("「%s」は RIP（ipv4）だけの設定です。RIPng（ipv6）の行には書けません", t.Column(col).Title.JA))
			}
		}
	}
	for _, r := range x.rows("rip_processes") {
		x.tally.Look("V-228", r.Get("device"), 1)
		v4only("rip_processes", r, "version", "auto_summary")
	}
	for _, table := range []string{"rip_interfaces", "rip_redistribute"} {
		t := x.in.Schema.Table(table)
		if t == nil {
			continue
		}
		for _, r := range x.rows(table) {
			dev, fam := r.Get("device"), r.Get("family")
			x.tally.Look("V-228", dev, 1)
			if procs[dev+"\x00"+fam] == nil {
				x.add("V-228", t, r, "family", fmt.Sprintf("シート「RIP」に、この機器の種類 %s の行がありません", fam))
			}
			if table != "rip_interfaces" {
				if r.Get("process_id") != "" && r.Get("source") != "ospf" {
					x.add("V-228", t, r, "process_id", "「OSPF のプロセス番号」は、再配布する経路が ospf の行にだけ書けます")
				}
				continue
			}
			v4only(table, r, "send_version", "receive_version", "v2_broadcast", "auth_mode", "auth_key", "auth_key_id")
			if fam == "ipv6" {
				continue
			}
			mode, key := r.Get("auth_mode"), r.Get("auth_key")
			switch {
			case mode != "" && key == "":
				x.add("V-228", t, r, "auth_key", "「認証」があるのに、「認証キー」がありません")
			case mode == "" && key != "":
				x.add("V-228", t, r, "auth_mode", "「認証キー」があるのに、「認証」（text・md5）がありません")
			}
			if r.Get("auth_key_id") != "" && mode != "md5" {
				x.add("V-228", t, r, "auth_key_id", "「認証キーの番号」は、認証が md5 の行にだけ書けます")
			}
			if r.Get("v2_broadcast") == "true" {
				if vs := ripVersions(r, procs[dev+"\x00"+fam], "send_version"); len(vs) > 0 && !(len(vs) == 1 && vs[0] == "2") {
					x.add("V-228", t, r, "v2_broadcast", "「版 2 をブロードキャストで送る」は、送る版が 2 の IF にだけ書けます")
				}
			}
		}
	}
}

// ripVersions gives the versions an interface sends or receives: what the
// row says, or else the version of the process. Empty = not known (the
// default of the vendor).
func ripVersions(r, proc *model.Row, col string) []string {
	if c := r.Cells[col]; c != nil && len(c.Values) > 0 && c.Values[0] != "" {
		return c.Values
	}
	if proc != nil && proc.Get("version") != "" {
		return []string{proc.Get("version")}
	}
	return nil
}

// ripPairs runs X-363 and X-364: two routers that run RIP on one subnet
// have to send a version the other takes, and to agree on the key.
func (c *cross) ripPairs() {
	t := c.in.Schema.Table("rip_interfaces")
	if t == nil {
		return
	}
	procs := map[string]*model.Row{}
	for _, r := range c.rows("rip_processes") {
		if r.Get("family") == "ipv4" {
			procs[r.Get("device")] = r
		}
	}
	rows := map[*xIf]*model.Row{}
	var list []*xIf
	for _, r := range c.rows("rip_interfaces") {
		if r.Get("family") != "ipv4" {
			continue
		}
		if i := c.byDev[r.Get("device")][r.Get("if_name")]; i != nil {
			rows[i] = r
			list = append(list, i)
		}
	}
	for _, a := range list {
		ra := rows[a]
		for _, b := range c.others(a) {
			rb := rows[b]
			if rb == nil {
				continue
			}
			c.tally.Look("X-363", a.device, 1)
			c.tally.Look("X-364", a.device, 1)
			// What a sends, b has to take (the other way is told on the
			// row of b).
			send, recv := ripVersions(ra, procs[a.device], "send_version"), ripVersions(rb, procs[b.device], "receive_version")
			if ra.Get("send") != "false" && rb.Get("receive") != "false" && len(send) > 0 && len(recv) > 0 {
				common := false
				for _, s := range send {
					for _, v := range recv {
						common = common || s == v
					}
				}
				if !common {
					c.add("X-363", t, ra, "send_version", fmt.Sprintf("%s が送る RIP の版を、相手の %s（%s）が受けません", a.name, b.device, b.name),
						model.KV{Name: where(a.device, "送る版"), Value: join(send)}, model.KV{Name: where(b.device, "受ける版"), Value: join(recv)})
				}
			}
			ma, mb := ra.Get("auth_mode"), rb.Get("auth_mode")
			switch {
			case ma != mb:
				c.add("X-364", t, ra, "auth_mode", fmt.Sprintf("RIP の認証が、相手の %s（%s）と違います", b.device, b.name),
					model.KV{Name: a.device, Value: none(ma)}, model.KV{Name: b.device, Value: none(mb)})
			case ma != "" && both(ra.Get("auth_key"), rb.Get("auth_key")):
				c.add("X-364", t, ra, "auth_key", fmt.Sprintf("RIP の認証キーが、相手の %s（%s）と違います（不一致）", b.device, b.name))
			case ma == "md5" && ripKeyID(ra) != ripKeyID(rb):
				c.add("X-364", t, ra, "auth_key_id", fmt.Sprintf("RIP の認証キーの番号が、相手の %s（%s）と違います", b.device, b.name),
					model.KV{Name: a.device, Value: ripKeyID(ra)}, model.KV{Name: b.device, Value: ripKeyID(rb)})
			}
		}
	}
}

func ripKeyID(r *model.Row) string {
	if v := r.Get("auth_key_id"); v != "" {
		return v
	}
	return "1"
}

// none shows an empty value as "なし".
func none(v string) string {
	if v == "" {
		return "なし"
	}
	return v
}
