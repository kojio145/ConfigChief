package check

import (
	"fmt"
	"net/netip"

	"github.com/kojio145/configchief/internal/model"
)

// The checks of the pairs of L2TP (appendix C 1.2; appendix A 3.21): X-371
// to X-375. Two rows of L2TPv3 are a pair when each points at an address of
// the other device and both have the same circuit number. A LAC and an LNS
// are a pair when the LAC points at an address of the device of the LNS.

// l2tpCrossIDs are the checks l2tpv3Pairs and l2tpPairs run.
var l2tpCrossIDs = []string{"X-371", "X-372", "X-373", "X-374", "X-375"}

// l2tpv3Pairs runs the checks of the two ends of an L2TPv3 circuit.
func (c *cross) l2tpv3Pairs() {
	t := c.in.Schema.Table("l2tpv3")
	if t == nil {
		return
	}
	rows := c.rows("l2tpv3")
	title := func(col string) string { return t.Column(col).Title.JA }
	mate := map[*model.Row]*model.Row{}
	for _, r := range rows {
		dev, name := r.Get("device"), r.Get("if_name")
		addr, err := netip.ParseAddr(r.Get("peer_address"))
		if err != nil {
			continue
		}
		c.tally.Look("X-300", dev, 1)
		peer := c.anyOwner(addr, dev)
		if peer == nil {
			c.add("X-300", t, r, "peer_address", fmt.Sprintf("L2 延伸の相手 %s を持つ機器が、案件の中にありません（管理外の相手）", addr),
				model.KV{Name: "相手アドレス", Value: addr.String()})
			continue
		}
		c.tally.Look("X-371", dev, 1)
		if mate[r] != nil {
			continue // found from the other side
		}
		// The rows of the other device that point back at this one.
		var back []*model.Row
		for _, o := range rows {
			if o.Get("device") != peer.device {
				continue
			}
			if a, err := netip.ParseAddr(o.Get("peer_address")); err == nil && c.devHas(dev, a) {
				back = append(back, o)
			}
		}
		var rev *model.Row
		for _, o := range back {
			if mate[o] == nil && o.Get("vc_id") == r.Get("vc_id") {
				rev = o
				break
			}
		}
		switch {
		case rev != nil:
			mate[r], mate[rev] = rev, r
		case len(back) == 0:
			c.add("X-371", t, r, "peer_address", fmt.Sprintf("相手の %s（%s）に、この機器を相手にする L2 延伸の行がありません", peer.device, peer.name),
				model.KV{Name: "相手アドレス", Value: addr.String()}, model.KV{Name: "相手", Value: where(peer.device, peer.name)})
		default:
			c.tally.Look("X-372", dev, 1)
			ev := []model.KV{{Name: where(dev, name), Value: r.Get("vc_id")}}
			for _, o := range back {
				ev = append(ev, model.KV{Name: where(peer.device, o.Get("if_name")), Value: o.Get("vc_id")})
			}
			c.add("X-372", t, r, "vc_id", fmt.Sprintf("L2 延伸 %s の回線番号 %s に合う行が、相手の %s にありません（回線番号は両端で同じ値にします）", name, r.Get("vc_id"), peer.device), ev...)
		}
	}

	seen := map[*model.Row]bool{}
	for _, r := range rows {
		rev := mate[r]
		if rev == nil {
			continue
		}
		dev, other := r.Get("device"), rev.Get("device")
		// What has a direction: each row is looked at.
		c.tally.Look("X-374", dev, 1)
		if src, ok := c.srcAddr(dev, r.Get("local_if")); ok {
			if dst, err := netip.ParseAddr(rev.Get("peer_address")); err == nil && src != dst {
				c.add("X-374", t, r, "local_if", fmt.Sprintf("送信元 IF %s のアドレス %s が、相手の %s が指す相手アドレス %s と違います", r.Get("local_if"), src, other, dst),
					model.KV{Name: where(dev, "の送信元"), Value: src.String()}, model.KV{Name: where(other, "の相手アドレス"), Value: dst.String()})
			}
		}
		if r.Get("signaling") == "manual" && rev.Get("signaling") == "manual" {
			c.tally.Look("X-372", dev, 1)
			for _, p := range [][2]string{{"local_session_id", "remote_session_id"}, {"local_cookie", "remote_cookie"}} {
				if a, b := r.Get(p[0]), rev.Get(p[1]); a != b {
					c.add("X-372", t, r, p[0], fmt.Sprintf("「%s」が、相手の %s の「%s」と違います", title(p[0]), other, title(p[1])),
						model.KV{Name: where(dev, "の"+title(p[0])), Value: a}, model.KV{Name: where(other, "の"+title(p[1])), Value: b})
				}
			}
		}
		// What is the same on both ends: once for a pair.
		if seen[r] {
			continue
		}
		seen[r], seen[rev] = true, true
		c.tally.Look("X-372", dev, 1)
		if a, b := r.Get("signaling"), rev.Get("signaling"); both(a, b) {
			c.add("X-372", t, r, "signaling", "L2 延伸の張り方が両端で違います", model.KV{Name: dev, Value: a}, model.KV{Name: other, Value: b})
			continue
		}
		if r.Get("signaling") != "dynamic" {
			continue
		}
		c.tally.Look("X-373", dev, 1)
		if a, b := r.Get("password"), rev.Get("password"); a != b {
			msg := "L2 延伸のトンネルのパスワードが両端で違います"
			if a == "" || b == "" {
				msg = "L2 延伸のトンネルのパスワードが片側にだけあります"
			}
			c.add("X-373", t, r, "password", msg, model.KV{Name: dev, Value: Seal(a)}, model.KV{Name: other, Value: Seal(b)})
		}
	}
}

// l2tpPairs runs the checks of a LAC and the LNS it points at.
func (c *cross) l2tpPairs() {
	t := c.in.Schema.Table("l2tp_groups")
	if t == nil {
		return
	}
	rows := c.rows("l2tp_groups")
	for _, r := range rows {
		if r.Get("role") != "lac" {
			continue
		}
		dev, name := r.Get("device"), r.Get("name")
		addr, err := netip.ParseAddr(r.Get("peer_address"))
		if err != nil {
			continue // a name, or nothing: V-241
		}
		c.tally.Look("X-300", dev, 1)
		peer := c.anyOwner(addr, dev)
		if peer == nil {
			c.add("X-300", t, r, "peer_address", fmt.Sprintf("L2TP の相手（LNS）%s を持つ機器が、案件の中にありません（管理外の相手）", addr),
				model.KV{Name: "相手アドレス", Value: addr.String()})
			continue
		}
		// The LNS that waits for this name comes first, then one that takes
		// any name.
		c.tally.Look("X-375", dev, 1)
		var lns, any, first *model.Row
		for _, o := range rows {
			if o.Get("device") != peer.device || o.Get("role") != "lns" {
				continue
			}
			if first == nil {
				first = o
			}
			switch pn := o.Get("peer_name"); {
			case pn == "" && any == nil:
				any = o
			case pn != "" && pn == r.Get("local_name") && lns == nil:
				lns = o
			}
		}
		if lns == nil {
			lns = any
		}
		if lns == nil {
			if first == nil {
				c.add("X-375", t, r, "peer_address", fmt.Sprintf("相手の %s（%s）に、役割が lns の L2TP の行がありません", peer.device, peer.name),
					model.KV{Name: "相手アドレス", Value: addr.String()}, model.KV{Name: "相手", Value: where(peer.device, peer.name)})
				continue
			}
			c.add("X-375", t, r, "local_name", fmt.Sprintf("L2TP %s の「自分の名前」が、相手の %s の LNS が受け付ける「相手の名前」と合いません", name, peer.device),
				model.KV{Name: where(dev, "の自分の名前"), Value: r.Get("local_name")}, model.KV{Name: where(peer.device, "の相手の名前"), Value: first.Get("peer_name")})
			lns = first
		}
		other := lns.Get("device")
		c.tally.Look("X-373", dev, 1)
		if a, b := r.Get("tunnel_auth"), lns.Get("tunnel_auth"); both(a, b) {
			c.add("X-373", t, r, "tunnel_auth", "L2TP のトンネルの認証が、LAC と LNS で違います（片側だけが有効です）",
				model.KV{Name: where(dev, name), Value: a}, model.KV{Name: where(other, lns.Get("name")), Value: b})
		} else if a, b := r.Get("password"), lns.Get("password"); both(a, b) {
			c.add("X-373", t, r, "password", "L2TP のトンネルのパスワードが、LAC と LNS で違います",
				model.KV{Name: where(dev, name), Value: Seal(a)}, model.KV{Name: where(other, lns.Get("name")), Value: Seal(b)})
		}
	}
}
