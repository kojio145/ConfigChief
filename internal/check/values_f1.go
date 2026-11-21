package check

import (
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// The checks of the values that came with the rest of the access lists, NAT,
// line monitoring and DHCP (appendix A 3.25, appendix C 1.1): V-276 to V-282.

var f1IDs = []string{"V-276", "V-277", "V-278", "V-279", "V-280", "V-281", "V-282"}

var (
	reClock = regexp.MustCompile(`^([01]?\d|2[0-3]):[0-5]\d$`)
	reDay   = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])$`)
	reLease = regexp.MustCompile(`^[1-9]\d*[dhms]$`)
)

// list gives the values of a cell that holds several.
func list(r *model.Row, column string) []string {
	if c := r.Cells[column]; c != nil {
		return c.Values
	}
	return nil
}

// f1 runs the checks of this file.
func (x *values) f1() {
	for _, id := range f1IDs {
		x.tally.Look(id, "", 0)
	}
	x.aclWords()
	x.remoteLists()
	x.timeRanges()
	x.natShapes()
	x.trackLists()
	x.leases()
}

// aclWords looks at the entries of the access lists (V-276 and V-277): the
// TCP flags belong to TCP, the ICMP type and code to ICMP.
func (x *values) aclWords() {
	tcp := map[string]bool{"ack": true, "fin": true, "psh": true, "rst": true, "syn": true, "urg": true}
	for _, name := range []string{"acl_entries", "ipv6_acl_entries"} {
		t := x.in.Schema.Table(name)
		if t == nil {
			continue
		}
		for _, r := range x.rows(name) {
			dev, proto := device(t, r), r.Get("protocol")
			flags := 0
			var words []string
			for _, f := range list(r, "flags") {
				if tcp[f] {
					flags++
				}
				if tcp[f] || f == "established" {
					words = append(words, f)
				}
			}
			x.tally.Look("V-276", dev, 1)
			if len(words) > 0 && proto != "tcp" {
				x.add("V-276", t, r, "flags", fmt.Sprintf("付加条件の %s は、プロトコルが tcp の行にだけ書けます（この行は「%s」）", strings.Join(words, "・"), proto),
					model.KV{Name: "値", Value: strings.Join(words, " ")})
			} else if flags > 1 {
				x.add("V-276", t, r, "flags", fmt.Sprintf("TCP のフラグが %d つあります（1 行に 1 つです）", flags), model.KV{Name: "値", Value: strings.Join(words, " ")})
			}
			typ, code := r.Get("icmp_type"), r.Get("icmp_code")
			x.tally.Look("V-277", dev, 1)
			switch {
			case typ != "" && proto != "icmp":
				x.add("V-277", t, r, "icmp_type", fmt.Sprintf("ICMP の種類は、プロトコルが icmp の行にだけ書けます（この行は「%s」）", proto), model.KV{Name: "値", Value: typ})
			case code != "" && proto != "icmp":
				x.add("V-277", t, r, "icmp_code", fmt.Sprintf("ICMP のコードは、プロトコルが icmp の行にだけ書けます（この行は「%s」）", proto), model.KV{Name: "値", Value: code})
			case code != "":
				if _, err := strconv.Atoi(typ); err != nil {
					x.add("V-277", t, r, "icmp_code", "ICMP のコードは、ICMP の種類を番号で書いた行にだけ書けます", model.KV{Name: "ICMP の種類", Value: typ})
				}
			}
		}
	}
}

// remoteLists looks at the lists of the remote login (V-278): the list of
// the row and the list of one protocol do not stand together.
func (x *values) remoteLists() {
	t := x.in.Schema.Table("remote_access")
	if t == nil {
		return
	}
	for _, r := range x.rows("remote_access") {
		x.tally.Look("V-278", device(t, r), 1)
		if r.Get("acl") == "" {
			continue
		}
		for _, col := range []string{"acl_ssh", "acl_telnet"} {
			if v := r.Get(col); v != "" {
				x.add("V-278", t, r, col, "「接続元の制限」と、方式ごとの制限の両方が書かれています（どちらか片方に書きます）", model.KV{Name: "値", Value: v})
			}
		}
	}
}

// timeRanges looks at the time ranges (V-279): what each kind needs, and the
// form of the times and the dates.
func (x *values) timeRanges() {
	t := x.in.Schema.Table("time_ranges")
	if t == nil {
		return
	}
	absolute := map[string]bool{}
	for _, r := range x.rows("time_ranges") {
		dev := device(t, r)
		x.tally.Look("V-279", dev, 1)
		for _, col := range []string{"start_time", "end_time"} {
			if v := r.Get(col); v != "" && !reClock.MatchString(v) {
				x.add("V-279", t, r, col, fmt.Sprintf("時刻「%s」が読めません（8:00、18:30 の形で書きます）", v), model.KV{Name: "値", Value: v})
			}
		}
		for _, col := range []string{"start_date", "end_date"} {
			if v := r.Get(col); v != "" && !reDay.MatchString(v) {
				x.add("V-279", t, r, col, fmt.Sprintf("日「%s」が読めません（2026-01-31 の形で書きます）", v), model.KV{Name: "値", Value: v})
			}
		}
		switch r.Get("kind") {
		case "periodic":
			switch {
			case len(list(r, "days")) == 0:
				x.add("V-279", t, r, "days", "種類が periodic の行に、曜日がありません")
			case r.Get("start_time") == "" || r.Get("end_time") == "":
				x.add("V-279", t, r, "start_time", "種類が periodic の行には、開始の時刻と終了の時刻の両方が要ります")
			case r.Get("start_date") != "" || r.Get("end_date") != "":
				x.add("V-279", t, r, "start_date", "種類が periodic の行に、日は書けません（日で区切るなら、種類を absolute にします）")
			}
		case "absolute":
			key := dev + "\x00" + r.Get("name")
			switch {
			case absolute[key]:
				x.add("V-279", t, r, "kind", "種類が absolute の行が、同じ名前に 2 つあります（1 つの時間帯に 1 行までです）")
			case r.Get("start_date") == "" && r.Get("end_date") == "":
				x.add("V-279", t, r, "start_date", "種類が absolute の行に、開始の日も終了の日もありません")
			case (r.Get("start_date") == "") != (r.Get("start_time") == "") || (r.Get("end_date") == "") != (r.Get("end_time") == ""):
				x.add("V-279", t, r, "start_time", "種類が absolute の行では、日と時刻を組で書きます（開始の日と開始の時刻、終了の日と終了の時刻）")
			case len(list(r, "days")) > 0 || len(list(r, "end_days")) > 0:
				x.add("V-279", t, r, "days", "種類が absolute の行に、曜日は書けません")
			}
			absolute[key] = true
		}
	}
}

// natShapes looks at the rules of NAT (V-280): a pool runs from its first
// address to its last, a network has its length, and "extendable" is of the
// static kinds.
func (x *values) natShapes() {
	t := x.in.Schema.Table("nat_rules")
	if t == nil {
		return
	}
	for _, r := range x.rows("nat_rules") {
		x.tally.Look("V-280", device(t, r), 1)
		kind := r.Get("type")
		switch kind {
		case "pat_pool", "dynamic_pool":
			a, errA := netip.ParseAddr(r.Get("pool_start"))
			b, errB := netip.ParseAddr(r.Get("pool_end"))
			if errA == nil && errB == nil && b.Less(a) {
				x.add("V-280", t, r, "pool_end", fmt.Sprintf("変換後アドレスの末尾 %s が、先頭 %s より前です", b, a), model.KV{Name: "値", Value: b.String()})
			}
		case "static_network":
			n, err := strconv.Atoi(r.Get("pool_prefix"))
			if err != nil {
				x.add("V-280", t, r, "pool_prefix", "種類が static_network の行に、「範囲のプレフィックス長」（ネットワークの長さ）がありません")
				break
			}
			for _, col := range []string{"local_ip", "global_ip"} {
				a, err := netip.ParseAddr(r.Get(col))
				if err != nil {
					continue
				}
				if p, err := a.Prefix(n); err == nil && p.Addr() != a {
					x.add("V-280", t, r, col, fmt.Sprintf("%s は、長さ %d のネットワークのアドレスではありません（%s と書きます）", a, n, p.Addr()), model.KV{Name: "値", Value: a.String()})
				}
			}
		}
		if r.Get("extendable") == "true" && kind != "static" && kind != "static_port" && kind != "static_network" {
			x.add("V-280", t, r, "extendable", fmt.Sprintf("「複数の外側アドレスに対応させる」は、種類が static・static_port・static_network の行にだけ書けます（この行は %s）", kind))
		}
	}
}

// trackLists looks at the rows of line monitoring (V-281): a list holds
// other rows of its device, and the columns of one kind do not stand on
// another.
func (x *values) trackLists() {
	t := x.in.Schema.Table("tracks")
	if t == nil {
		return
	}
	for _, r := range x.rows("tracks") {
		dev, kind, id := device(t, r), r.Get("type"), r.Get("id")
		x.tally.Look("V-281", dev, 1)
		members := list(r, "members")
		if kind == "list" {
			if len(members) == 0 {
				x.add("V-281", t, r, "members", "監視の方法が list の行に、「まとめる監視」がありません")
			}
			for _, m := range members {
				switch w := x.find("tracks", dev, "id", m); {
				case m == id:
					x.add("V-281", t, r, "members", "「まとめる監視」に、この行自身が入っています", model.KV{Name: "値", Value: m})
				case w != nil && w.Get("type") == "list" && containsWord(list(w, "members"), id):
					x.add("V-281", t, r, "members", fmt.Sprintf("%s と %s が、互いをまとめています", id, m), model.KV{Name: "値", Value: m})
				}
			}
			if r.Get("list_rule") == "" {
				x.add("V-281", t, r, "list_rule", "監視の方法が list の行に、「まとめ方」がありません（all_up か any_up）")
			}
			continue
		}
		if len(members) > 0 || r.Get("list_rule") != "" {
			x.add("V-281", t, r, "members", fmt.Sprintf("「まとめる監視」「まとめ方」は、監視の方法が list の行にだけ書けます（この行は %s）", kind))
		}
		if r.Get("source_if") != "" && r.Get("source_ip") != "" {
			x.add("V-281", t, r, "source_ip", "「送信元 IF」と「送信元アドレス」の両方が書かれています（どちらか 1 つです）", model.KV{Name: "値", Value: r.Get("source_ip")})
		}
		x.trackFamily(t, r, dev, kind)
		if kind != "icmp" {
			for _, col := range []string{"source_ip", "threshold", "up_when"} {
				if v := r.Get(col); v != "" {
					x.add("V-281", t, r, col, fmt.Sprintf("「%s」は、監視の方法が icmp の行にだけ書けます（この行は %s）", t.Column(col).Title.JA, kind), model.KV{Name: "値", Value: v})
				}
			}
		}
	}
}

// trackFamily looks at the target of a watch by ping or of a route, and at
// the addresses that go with it (V-229): the target is an address or a
// network of either family, and the source and the next hop of the probe
// are of the family of the target (appendix A 3.23, 2026-11-21).
func (x *values) trackFamily(t *schema.Table, r *model.Row, dev, kind string) {
	if kind != "icmp" && kind != "route" {
		return
	}
	x.tally.Look("V-229", dev, 1)
	target := r.Get("target")
	if target == "" {
		return
	}
	var v6 bool
	if kind == "icmp" {
		a, err := netip.ParseAddr(target)
		if err != nil || a.Zone() != "" {
			x.add("V-229", t, r, "target", fmt.Sprintf("監視先「%s」は、IPv4 か IPv6 のアドレスではありません", target), model.KV{Name: "値", Value: target})
			return
		}
		v6 = a.Is6()
	} else {
		p, err := netip.ParsePrefix(target)
		if err != nil || p != p.Masked() {
			x.add("V-229", t, r, "target", fmt.Sprintf("経路の監視先「%s」は、ネットワークのアドレス（192.0.2.0/24・2001:db8::/32 の形）ではありません", target), model.KV{Name: "値", Value: target})
			return
		}
		v6 = p.Addr().Is6()
	}
	fam := map[bool]string{false: "IPv4", true: "IPv6"}
	for _, col := range []string{"source_ip", "via_next_hop"} {
		v := r.Get(col)
		if v == "" {
			continue
		}
		if a, err := netip.ParseAddr(v); err == nil && a.Is6() != v6 {
			x.add("V-229", t, r, col, fmt.Sprintf("「%s」%s は %s で、監視先 %s（%s）と種類が違います", t.Column(col).Title.JA, v, fam[a.Is6()], target, fam[v6]), model.KV{Name: "値", Value: v})
		}
	}
}

func containsWord(words []string, w string) bool {
	for _, v := range words {
		if v == w {
			return true
		}
	}
	return false
}

// leases looks at the DHCP pools (V-282): the form of the lease, and a
// second gateway without a first.
func (x *values) leases() {
	t := x.in.Schema.Table("dhcp_pools")
	if t == nil {
		return
	}
	for _, r := range x.rows("dhcp_pools") {
		x.tally.Look("V-282", device(t, r), 1)
		if v := r.Get("lease"); v != "" && v != "infinite" && !reLease.MatchString(v) {
			x.add("V-282", t, r, "lease", fmt.Sprintf("貸し出し時間「%s」が読めません（2d、12h、90m、100s の形か、infinite と書きます）", v), model.KV{Name: "値", Value: v})
		}
		if len(list(r, "default_routers_more")) > 0 && r.Get("default_router") == "" {
			x.add("V-282", t, r, "default_routers_more", "「デフォルトゲートウェイ（2 つ目以降）」があるのに、「デフォルトゲートウェイ」が空です")
		}
	}
}
