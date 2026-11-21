package check

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/kojio145/configchief/internal/model"
)

// The checks of the values that came with IKE version 2 and the rest of
// IPsec (appendix A 3.20, appendix C 1.1): V-231 to V-234.

// idKinds are the kinds of an identity of IKE ("kind:value").
var idKinds = map[string]bool{"address": true, "fqdn": true, "email": true, "keyid": true}

// idOK tells whether an identity is written as "kind:value".
func idOK(v string) bool {
	kind, val, found := strings.Cut(v, ":")
	if !found || !idKinds[kind] || val == "" || strings.ContainsAny(val, " \t") {
		return false
	}
	if kind == "address" {
		_, err := netip.ParseAddr(val)
		return err == nil
	}
	return true
}

// peerOK tells whether the peer of a row is an address, "any" or a name.
func peerOK(v string) bool {
	if v == "any" {
		return true
	}
	if _, err := netip.ParseAddr(v); err == nil {
		return true
	}
	if v == "" || strings.ContainsAny(v, " \t/") {
		return false
	}
	// A name has a letter; what is all digits and dots is a broken address.
	return strings.ContainsAny(strings.ToLower(v), "abcdefghijklmnopqrstuvwxyz")
}

// selectorPorts looks at the selectors of IPsec: a port without the
// protocol that has ports (V-237).
func (x *values) selectorPorts() {
	t := x.in.Schema.Table("ipsec_selectors")
	if t == nil || t.Column("local_port") == nil {
		return
	}
	for _, s := range x.rows("ipsec_selectors") {
		dev := device(t, s)
		x.tally.Look("V-237", dev, 1)
		if p := s.Get("protocol"); p != "udp" && p != "tcp" {
			for _, col := range []string{"local_port", "remote_port"} {
				if s.Get(col) != "" {
					x.add("V-237", t, s, col, fmt.Sprintf("IPsec %s の対象通信 %s 番に「%s」がありますが、プロトコルが udp・tcp ではありません", s.Get("tunnel"), s.Get("seq"), t.Column(col).Title.JA),
						model.KV{Name: "プロトコル", Value: p})
				}
			}
		}
	}
}

// ipsecMore looks at the rows of IPsec: the identities (V-231), the values
// that the version of IKE of the row does not use (V-232), a watch of the
// peer without its interval (V-233) and the peer (V-234).
func (x *values) ipsecMore() {
	t := x.in.Schema.Table("ipsec_tunnels")
	if t == nil {
		return
	}
	title := func(col string) string { return t.Column(col).Title.JA }
	for _, r := range x.rows("ipsec_tunnels") {
		dev, name := device(t, r), r.Get("name")
		x.tally.Look("V-231", dev, 1)
		for _, col := range []string{"local_id", "remote_id"} {
			if v := r.Get(col); v != "" && !idOK(v) {
				x.add("V-231", t, r, col, fmt.Sprintf("IPsec %s の「%s」が「種類:値」の形ではありません（種類は address・fqdn・email・keyid）", name, title(col)),
					model.KV{Name: "値", Value: v})
			}
		}
		x.tally.Look("V-232", dev, 1)
		var cols []string
		switch r.Get("ike_version") {
		case "1":
			cols = []string{"ike_prf", "psk_remote"}
		case "2":
			cols = []string{"ike_mode"}
		}
		for _, col := range cols {
			if t.Column(col) != nil && r.Get(col) != "" {
				x.add("V-232", t, r, col, fmt.Sprintf("IPsec %s の「%s」は、IKE の版 %s では使いません", name, title(col), r.Get("ike_version")),
					model.KV{Name: "IKE の版", Value: r.Get("ike_version")})
			}
		}
		x.tally.Look("V-233", dev, 1)
		if r.Get("dpd_interval") == "" {
			for _, col := range []string{"dpd_retry", "dpd_retry_interval", "dpd_mode"} {
				if t.Column(col) != nil && r.Get(col) != "" {
					x.add("V-233", t, r, col, fmt.Sprintf("IPsec %s の「%s」がありますが、「%s」が空です", name, title(col), title("dpd_interval")))
				}
			}
		}
		// A list of values on one proposal is a way of writing of IKE
		// version 2, and a list has its first value on the row.
		x.tally.Look("V-236", dev, 1)
		for _, pair := range [][2]string{{"ike_encryption", "ike_encryption_more"}, {"ike_hash", "ike_hash_more"}, {"ike_dh", "ike_dh_more"}, {"ike_prf", "ike_prf_more"},
			{"esp_encryption", "esp_encryption_more"}, {"esp_hash", "esp_hash_more"}, {"pfs", "pfs_more"}} {
			c := r.Cells[pair[1]]
			if t.Column(pair[1]) == nil || c == nil || len(c.Values) == 0 {
				continue
			}
			if r.Get("ike_version") != "2" {
				x.add("V-236", t, r, pair[1], fmt.Sprintf("IPsec %s の「%s」は、IKE の版 2 の行だけに書けます", name, title(pair[1])),
					model.KV{Name: "IKE の版", Value: r.Get("ike_version")})
			} else if v := r.Get(pair[0]); v == "" || v == "none" {
				x.add("V-236", t, r, pair[1], fmt.Sprintf("IPsec %s の「%s」に値がありますが、「%s」が空です", name, title(pair[1]), title(pair[0])))
			}
		}
		x.tally.Look("V-234", dev, 1)
		if v := r.Get("peer_address"); v != "" && !peerOK(v) {
			x.add("V-234", t, r, "peer_address", fmt.Sprintf("IPsec %s の「%s」が、アドレス・any・名前のどれでもありません", name, title("peer_address")),
				model.KV{Name: "値", Value: v})
		}
		// The two VRFs (appendix A 3.20). The inner one of a tunnel
		// interface is the VRF of that interface, not a cell of this row;
		// the outer one of a row on an interface is the VRF of that
		// interface, where IKE is taken.
		if t.Column("inner_vrf") != nil && t.Column("outer_vrf") != nil {
			x.tally.Look("V-238", dev, 1)
			if v := r.Get("inner_vrf"); v != "" && r.Get("style") == "tunnel_if" {
				x.add("V-238", t, r, "inner_vrf", fmt.Sprintf("IPsec %s は方式が tunnel_if ですが、「%s」があります（内側は、トンネル IF の VRF です）", name, title("inner_vrf")),
					model.KV{Name: "値", Value: v})
			}
			x.tally.Look("V-239", dev, 1)
			if lif := r.Get("local_if"); lif != "" && r.Get("style") == "policy" {
				it := x.in.Schema.Table("interfaces")
				for _, i := range x.rows("interfaces") {
					if it == nil || device(it, i) != dev || i.Get("name") != lif {
						continue
					}
					if i.Get("vrf") != r.Get("outer_vrf") {
						x.add("V-239", t, r, "outer_vrf", fmt.Sprintf("IPsec %s の「%s」が、自分側 IF %s の VRF と違います", name, title("outer_vrf"), lif),
							model.KV{Name: "外側の VRF", Value: r.Get("outer_vrf")}, model.KV{Name: "自分側 IF の VRF", Value: i.Get("vrf")})
					}
				}
			}
		}
	}
}
