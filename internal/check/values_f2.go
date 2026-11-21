package check

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/kojio145/configchief/internal/model"
)

// The checks of appendix A 3.26 (the device as a whole, the secrets in the
// form a device saved, the rest of QoS and PPPoE, EtherIP): V-283 to V-289
// and X-407.

// basicsIDs are the checks basics runs.
var basicsIDs = []string{"V-283", "V-284", "V-285", "V-286", "V-287", "V-288", "V-289"}

// basicsCrossIDs are the checks etherIPPairs runs.
var basicsCrossIDs = []string{"X-407"}

// has tells whether a cell holds something (a list: one value or more).
func has(r *model.Row, col string) bool {
	if c := r.Cells[col]; c != nil && len(c.Values) > 0 {
		return true
	}
	return r.Get(col) != ""
}

// basics looks at the columns appendix A 3.26 added.
func (x *values) basics() {
	x.savedSecrets()
	x.reversibleUsers()
	x.qosActions()
	x.qosConditions()
	x.etherIP()
	x.pppoeOptions()
	x.deviceBasics()
}

// savedSecrets tells the secrets a device saved in its encrypted form (V-283):
// "<kind> <value>", as the device shows it. A plain secret has no space. The
// tables that have a column for the form (devices, users) are not looked at.
func (x *values) savedSecrets() {
	for _, t := range x.in.Schema.Tables {
		if t.Column("secret_form") != nil {
			continue
		}
		for _, c := range t.Columns {
			if !c.Secret {
				continue
			}
			for _, r := range x.rows(t.Name) {
				v := r.Get(c.Name)
				if v == "" {
					continue
				}
				x.tally.Look("V-283", device(t, r), 1)
				if strings.Contains(v, " ") {
					x.add("V-283", t, r, c.Name, fmt.Sprintf("「%s」は、機器が暗号化して保存した形（種別と値）です。同じメーカーの機器へはそのまま書けますが、別メーカーへは引き継げません", c.Title.JA))
				}
			}
		}
	}
}

// reversibleUsers: a user the device keeps the password of cannot have a
// hash, and the encrypted form is of such a user only (V-284).
func (x *values) reversibleUsers() {
	t := x.in.Schema.Table("users")
	if t == nil || t.Column("reversible") == nil {
		return
	}
	for _, r := range x.rows("users") {
		x.tally.Look("V-284", device(t, r), 1)
		rev, form := r.Get("reversible") == "true", r.Get("secret_form")
		switch {
		case rev && form == "hashed":
			x.add("V-284", t, r, "reversible", fmt.Sprintf("ユーザ %s は「%s」が有効ですが、パスワードがハッシュです（ハッシュは元に戻せません）", r.Get("name"), t.Column("reversible").Title.JA))
		case !rev && form == "encrypted":
			x.add("V-284", t, r, "secret_form", fmt.Sprintf("ユーザ %s のパスワードは暗号化した形（encrypted）ですが、「%s」が有効ではありません", r.Get("name"), t.Column("reversible").Title.JA))
		}
	}
}

// qosActions: the actions of a class that cannot stand together (V-285).
func (x *values) qosActions() {
	t := x.in.Schema.Table("qos_policies")
	if t == nil || t.Column("shape_percent") == nil {
		return
	}
	title := func(col string) string { return t.Column(col).Title.JA }
	for _, r := range x.rows("qos_policies") {
		x.tally.Look("V-285", device(t, r), 1)
		what := fmt.Sprintf("QoS のポリシー %s のクラス %s", r.Get("policy"), r.Get("class"))
		tell := func(col, msg string) { x.add("V-285", t, r, col, what+": "+msg) }
		shape := has(r, "shape_bps") || has(r, "shape_percent")
		police := has(r, "police_bps") || has(r, "police_percent")
		if has(r, "shape_bps") && has(r, "shape_percent") {
			tell("shape_percent", fmt.Sprintf("「%s」と「%s」の両方があります（どちらか 1 つ）", title("shape_bps"), title("shape_percent")))
		}
		if has(r, "shape_mode") && !shape {
			tell("shape_mode", fmt.Sprintf("「%s」がありますが、上限（bps か %%）がありません", title("shape_mode")))
		}
		if has(r, "police_bps") && has(r, "police_percent") {
			tell("police_percent", fmt.Sprintf("「%s」と「%s」の両方があります（どちらか 1 つ）", title("police_bps"), title("police_percent")))
		}
		for _, col := range []string{"police_pir_bps", "police_violate"} {
			if has(r, col) && !police {
				tell(col, fmt.Sprintf("「%s」がありますが、「%s」がありません", title(col), title("police_bps")))
			}
		}
		if v := r.Get("police_violate"); has(r, "police_violate_value") && v != "set_dscp" && v != "set_precedence" {
			tell("police_violate_value", fmt.Sprintf("「%s」は、「%s」が set_dscp か set_precedence のときだけ書けます", title("police_violate_value"), title("police_violate")))
		}
		if has(r, "random_detect") && has(r, "queue_limit") {
			tell("random_detect", fmt.Sprintf("「%s」と「%s」は一緒に書けません", title("random_detect"), title("queue_limit")))
		}
		if has(r, "queue_limit_unit") && !has(r, "queue_limit") {
			tell("queue_limit_unit", fmt.Sprintf("「%s」がありますが、「%s」がありません", title("queue_limit_unit"), title("queue_limit")))
		}
	}
}

// qosConditions: one condition on a row (V-286). An access list may stand
// with a CoS, and an NGN tunnel with a QoS group.
func (x *values) qosConditions() {
	t := x.in.Schema.Table("qos_class_matches")
	if t == nil || t.Column("input_if") == nil {
		return
	}
	kinds := []string{"acl", "dscp", "precedence", "input_if", "cos", "qos_group", "protocol", "class_ref", "ipv6_acl", "local_generated", "rtp_port_start", "ngn_if"}
	for _, r := range x.rows("qos_class_matches") {
		x.tally.Look("V-286", device(t, r), 1)
		what := fmt.Sprintf("QoS のクラス %s の %s 番の条件", r.Get("class"), r.Get("seq"))
		var set []string
		for _, k := range kinds {
			if k == "local_generated" {
				if r.Get(k) == "true" {
					set = append(set, k)
				}
				continue
			}
			if has(r, k) {
				set = append(set, k)
			}
		}
		pair := func(a, b string) bool {
			return len(set) == 2 && ((set[0] == a && set[1] == b) || (set[0] == b && set[1] == a))
		}
		if len(set) > 1 && !pair("acl", "cos") && !pair("qos_group", "ngn_if") {
			var names []string
			for _, k := range set {
				names = append(names, t.Column(k).Title.JA)
			}
			x.add("V-286", t, r, set[1], fmt.Sprintf("%s: 1 行に条件が 2 つ以上あります（%s）。行を分けます", what, strings.Join(names, "・")))
		}
		if r.Get("negate") == "true" && len(set) == 0 {
			x.add("V-286", t, r, "negate", fmt.Sprintf("%s: 「%s」が有効ですが、逆にする条件がありません", what, t.Column("negate").Title.JA))
		}
		if ref := r.Get("class_ref"); ref != "" && ref == r.Get("class") {
			x.add("V-286", t, r, "class_ref", fmt.Sprintf("%s: 「%s」が自分自身です", what, t.Column("class_ref").Title.JA))
		}
		if has(r, "rtp_port_range") != has(r, "rtp_port_start") {
			x.add("V-286", t, r, "rtp_port_start", fmt.Sprintf("%s: RTP のポートは、先頭と幅を組で書きます", what))
		}
	}
}

// etherIP: a tunnel of EtherIP carries the frames of a bridge group, so its
// interface has to be in one, with an interface to extend (V-287).
func (x *values) etherIP() {
	t := x.in.Schema.Table("tunnels")
	it := x.in.Schema.Table("interfaces")
	if t == nil || it == nil || it.Column("bridge_group") == nil {
		return
	}
	for _, r := range x.rows("tunnels") {
		if r.Get("type") != "etherip" {
			continue
		}
		dev, name := device(t, r), r.Get("if_name")
		x.tally.Look("V-287", dev, 1)
		i := x.find("interfaces", dev, "name", name)
		if i == nil {
			continue // V-102 tells a tunnel without its interface.
		}
		grp := i.Get("bridge_group")
		if grp == "" {
			x.add("V-287", t, r, "type", fmt.Sprintf("トンネル %s は種類が etherip ですが、インタフェースの表の「%s」が空です（延伸する IF と同じ番号を書きます）", name, it.Column("bridge_group").Title.JA))
			continue
		}
		others := 0
		for _, o := range x.rows("interfaces") {
			if o.Get("device") == dev && o.Get("name") != name && o.Get("bridge_group") == grp && o.Get("kind") != "tunnel" {
				others++
			}
		}
		if others == 0 {
			x.add("V-287", t, r, "type", fmt.Sprintf("トンネル %s（EtherIP）のブリッジグループ %s に、延伸する IF がありません", name, grp))
		}
	}
}

// pppoeOptions: values of PPPoE that need another one (V-288).
func (x *values) pppoeOptions() {
	t := x.in.Schema.Table("pppoe")
	if t == nil || t.Column("auth_both") == nil {
		return
	}
	title := func(col string) string { return t.Column(col).Title.JA }
	for _, r := range x.rows("pppoe") {
		x.tally.Look("V-288", device(t, r), 1)
		if r.Get("auth_both") == "true" && r.Get("auth") == "" {
			x.add("V-288", t, r, "auth_both", fmt.Sprintf("PPPoE %s は「%s」が有効ですが、「%s」が空です", r.Get("name"), title("auth_both"), title("auth")))
		}
		if has(r, "idle_timeout") && r.Get("on_demand") != "true" {
			x.add("V-288", t, r, "idle_timeout", fmt.Sprintf("PPPoE %s に「%s」がありますが、「%s」が有効ではありません（いつもつなぐ回線は、無通信で切りません）", r.Get("name"), title("idle_timeout"), title("on_demand")))
		}
	}
}

// deviceBasics: the values of a device that need another one (V-289).
func (x *values) deviceBasics() {
	t := x.in.Schema.Table("devices")
	if t == nil || t.Column("log_timestamp") == nil {
		return
	}
	title := func(col string) string { return t.Column(col).Title.JA }
	for _, r := range x.rows("devices") {
		dev := device(t, r)
		x.tally.Look("V-289", dev, 1)
		tell := func(col, msg string) { x.add("V-289", t, r, col, msg) }
		for _, p := range [][2]string{{"log_timestamp", "log_timestamp_options"}, {"debug_timestamp", "debug_timestamp_options"}} {
			if has(r, p[1]) && r.Get(p[0]) != "datetime" {
				tell(p[1], fmt.Sprintf("「%s」は、「%s」が datetime のときだけ書けます", title(p[1]), title(p[0])))
			}
		}
		for _, col := range []string{"log_buffer_size", "log_buffer_mode"} {
			if has(r, col) && r.Get("log_buffer") != "true" {
				tell(col, fmt.Sprintf("「%s」がありますが、「%s」が有効ではありません", title(col), title("log_buffer")))
			}
		}
		if has(r, "console_timeout_sec") && !has(r, "console_timeout") {
			tell("console_timeout_sec", fmt.Sprintf("「%s」がありますが、「%s」が空です（分が無ければ 0 と書きます）", title("console_timeout_sec"), title("console_timeout")))
		}
		if has(r, "name_servers_vrf") && !has(r, "name_servers") {
			tell("name_servers_vrf", fmt.Sprintf("「%s」がありますが、「%s」が空です", title("name_servers_vrf"), title("name_servers")))
		}
	}
	if rt := x.in.Schema.Table("remote_access"); rt != nil && rt.Column("idle_timeout_sec") != nil {
		for _, r := range x.rows("remote_access") {
			if has(r, "idle_timeout_sec") && !has(r, "idle_timeout") {
				x.add("V-289", rt, r, "idle_timeout_sec", fmt.Sprintf("「%s」がありますが、「%s」が空です（分が無ければ 0 と書きます）", rt.Column("idle_timeout_sec").Title.JA, rt.Column("idle_timeout").Title.JA))
			}
		}
	}
}

// etherIPPairs: the two ends of a tunnel are the same kind of tunnel. A
// tunnel of EtherIP whose other end, in the case, is a tunnel of another
// kind does not come up (X-407). The end that is missing is told by X-301.
func (c *cross) etherIPPairs() {
	t := c.in.Schema.Table("tunnels")
	if t == nil {
		return
	}
	rows := c.rows("tunnels")
	for _, r := range rows {
		if r.Get("type") != "etherip" {
			continue
		}
		dev := r.Get("device")
		dst, err := netip.ParseAddr(r.Get("destination"))
		if err != nil {
			continue
		}
		peer := c.anyOwner(dst, dev)
		if peer == nil {
			continue
		}
		c.tally.Look("X-407", dev, 1)
		for _, o := range rows {
			if o.Get("device") != peer.device || o.Get("type") == "etherip" {
				continue
			}
			back, err := netip.ParseAddr(o.Get("destination"))
			if err != nil || !c.devHas(dev, back) {
				continue
			}
			c.add("X-407", t, r, "type", fmt.Sprintf("トンネル %s は種類が etherip ですが、相手の機器 %s のトンネル %s は種類が %s です（両端を同じ種類にします）", r.Get("if_name"), peer.device, o.Get("if_name"), o.Get("type")),
				model.KV{Name: "相手の機器", Value: peer.device}, model.KV{Name: "相手のトンネル", Value: o.Get("if_name")})
			break
		}
	}
}
