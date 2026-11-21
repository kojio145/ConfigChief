package ops

import (
	"fmt"
	"net/netip"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/report"
	"github.com/kojio145/configchief/internal/schema"
)

// The delivery sheets (spec 13.2) and the table of values for the other
// side (spec 13.5). Both read the book and write nothing but their files.

// SheetsOutcome is what one run of the delivery sheets made.
type SheetsOutcome struct {
	Dir      string   // out/<time>/sheets
	Files    []string // the files written, with their folder
	Standard bool     // no template: the standard form
	// Problems lists what was wrong in a template, the template first. A
	// template with a problem gave no file.
	Problems []string
	Notes    []string
	Masked   bool
}

// templateFiles lists the templates of the case: the xlsx files of
// templates/, by name.
func (e *Env) templateFiles() ([]string, error) {
	entries, err := os.ReadDir(e.Proj.Path("templates"))
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("templates を読めません: %v", err)
	}
	var names []string
	for _, ent := range entries {
		n := ent.Name()
		if ent.IsDir() || strings.HasPrefix(n, ".") || strings.HasPrefix(n, "~$") || !strings.EqualFold(n[max(0, len(n)-5):], ".xlsx") {
			continue
		}
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

// ExportSheets writes the delivery sheets: one output for each template of
// templates/, or the standard form when there is none.
func (e *Env) ExportSheets(mask bool, now time.Time) (*SheetsOutcome, error) {
	_, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	if bk == nil {
		return nil, fmt.Errorf("%s がありません。先にパラメータブックを作ってください", e.Proj.Config.Book)
	}
	ds, problems := bk.Expand(e.Schema)
	names, err := e.templateFiles()
	if err != nil {
		return nil, err
	}
	out := &SheetsOutcome{Dir: "out/" + project.Stamp(now) + "/sheets", Masked: mask}
	if len(problems) > 0 {
		out.Notes = append(out.Notes, fmt.Sprintf("パラメータブックに、読めなかった所が %d 件あります（check で確かめてください）。読めた値だけを書き出しています", len(problems)))
	}
	opt := book.DeliverOptions{Project: e.Proj.Config.Name, Mask: mask, Lang: e.lang()}
	var files []book.Delivered
	if len(names) == 0 {
		data, err := book.Standard(e.Schema, ds, opt)
		if err != nil {
			return nil, err
		}
		out.Standard = true
		files = append(files, book.Delivered{Name: "parameter_sheet.xlsx", Data: data})
	}
	taken := map[string]string{}
	for _, name := range names {
		tpl, err := os.ReadFile(e.Proj.Path("templates/" + name))
		if err != nil {
			out.Problems = append(out.Problems, fmt.Sprintf("templates/%s: 読めません: %v", name, err))
			continue
		}
		made, bad := book.Deliver(name, tpl, e.Schema, ds, opt)
		for _, p := range bad {
			out.Problems = append(out.Problems, "templates/"+name+": "+p)
		}
		for _, m := range made {
			if other := taken[strings.ToLower(m.Name)]; other != "" {
				out.Problems = append(out.Problems, fmt.Sprintf("templates/%s: 出力のファイル名 %s が、templates/%s の出力と重なります", name, m.Name, other))
				continue
			}
			taken[strings.ToLower(m.Name)] = name
			files = append(files, m)
		}
	}
	if len(files) > 0 {
		if err := os.MkdirAll(e.Proj.Path(out.Dir), 0o755); err != nil {
			return nil, fmt.Errorf("%s を作れません: %v", out.Dir, err)
		}
	}
	for _, f := range files {
		path := out.Dir + "/" + f.Name
		if err := os.WriteFile(e.Proj.Path(path), f.Data, 0o644); err != nil {
			return nil, fmt.Errorf("%s を書けません: %v", path, err)
		}
		out.Files = append(out.Files, path)
	}
	e.Proj.Log(now, "export sheet: ひな形 %d 個、書いたファイル %d 個、ひな形の誤り %d 件、秘密の値は%s", len(names), len(out.Files), len(out.Problems), secretWord(mask))
	return out, nil
}

func secretWord(mask bool) string {
	if mask {
		return "伏せた"
	}
	return "書いた"
}

// PeerOutcome is what one run of the table for the other side made.
type PeerOutcome struct {
	Files  Files
	Peers  int // the other sides that are not of the case
	Items  int
	Masked bool
	Notes  []string
}

// peerItem is one value the other side has to match.
type peerItem struct {
	peer, feature, item, device, ours, theirs, note string
}

// peerTable builds the items of the devices that are not of the case.
type peerTable struct {
	e     *Env
	ds    *model.Dataset // the book with the defaults of each device's pack
	mask  bool
	items []peerItem
}

func (p *peerTable) rows(table, dev string) []*model.Row {
	var out []*model.Row
	if t := p.ds.Tables[table]; t != nil {
		for _, r := range t.Rows {
			if r.Get("device") == dev {
				out = append(out, r)
			}
		}
	}
	return out
}

func (p *peerTable) find(table, dev, col, val string) *model.Row {
	for _, r := range p.rows(table, dev) {
		if r.Get(col) == val {
			return r
		}
	}
	return nil
}

// addrOf gives the first address of an interface, or the address itself.
func (p *peerTable) addrOf(dev, src string) string {
	if _, err := netip.ParseAddr(src); err == nil {
		return src
	}
	if i := p.find("interfaces", dev, "name", src); i != nil {
		if pf, err := netip.ParsePrefix(i.Get("ipv4")); err == nil {
			return pf.Addr().String()
		}
	}
	return ""
}

// value shows a cell as the book does, a secret hidden on demand.
func (p *peerTable) value(t *schema.Table, r *model.Row, col string) string {
	c := t.Column(col)
	if c == nil || r == nil {
		return ""
	}
	v := book.Show(c, r.Cells[col], p.e.lang())
	if v != "" && c.Secret && p.mask {
		return masked
	}
	return v
}

func (p *peerTable) add(peer, feature, dev, item, ours, theirs, note string) {
	if ours == "" && theirs == "" {
		return
	}
	p.items = append(p.items, peerItem{peer: peer, feature: feature, item: item, device: dev, ours: ours, theirs: theirs, note: note})
}

// same adds the columns both sides write alike.
func (p *peerTable) same(peer, feature string, t *schema.Table, r *model.Row, note string, cols ...string) {
	for _, col := range cols {
		if t.Column(col) == nil {
			continue
		}
		v := p.value(t, r, col)
		p.add(peer, feature, r.Get("device"), t.Column(col).Title.In(p.e.lang()), v, v, note)
	}
}

const noLocal = "こちらのアドレスが、設計データからは決まりません（回線から取るアドレスなど）。決まったら伝えてください"

func (p *peerTable) ipsec(t *schema.Table, r *model.Row) {
	dev, peer := r.Get("device"), r.Get("peer_address")
	feature := "IPsec（" + r.Get("name") + "）"
	local := ""
	if tif := r.Get("tunnel_if"); tif != "" {
		if tun := p.find("tunnels", dev, "if_name", tif); tun != nil {
			local = p.addrOf(dev, tun.Get("source"))
		}
	}
	if local == "" {
		local = r.Get("local_address")
	}
	if local == "" && r.Get("local_if") != "" {
		local = p.addrOf(dev, r.Get("local_if"))
	}
	note := "相手の機器では、こちらのアドレスを相手アドレスに書く"
	if local == "" {
		note = noLocal
	}
	p.add(peer, feature, dev, "相手アドレス", peer, local, note)
	p.same(peer, feature, t, r, "", "ike_version", "ike_mode")
	// The keys: with IKE version 2 each side may prove itself with a key
	// of its own, and the two keys change places on the other device.
	// A key in the form the device saved it ("<kind> <value>", appendix A
	// 3.26) is of no use to the other side: the key itself has to be told.
	saved := ""
	if strings.Contains(r.Get("psk"), " ") || strings.Contains(r.Get("psk_remote"), " ") {
		saved = "機器が暗号化して保存した形（種別と値）です。相手には、元の鍵を別に伝えてください"
	}
	if theirs := p.value(t, r, "psk_remote"); theirs != "" {
		p.add(peer, feature, dev, title(t, "psk", p), p.value(t, r, "psk"), "", strings.TrimSuffix("相手の機器では「"+title(t, "psk_remote", p)+"」に書く。"+saved, "。"))
		p.add(peer, feature, dev, title(t, "psk_remote", p), theirs, "", strings.TrimSuffix("相手の機器では「"+title(t, "psk", p)+"」に書く。"+saved, "。"))
	} else {
		p.same(peer, feature, t, r, saved, "psk")
	}
	p.same(peer, feature, t, r, "hex は 16 進で書いた鍵。両端で同じ書き方にする", "psk_form")
	// A value with further candidates: the list, in the order of liking.
	listed := func(col, more, note string) {
		if t.Column(col) == nil {
			return
		}
		v := p.value(t, r, col)
		if c := r.Cells[more]; v != "" && c != nil && len(c.Values) > 0 {
			v += "、" + strings.Join(c.Values, "、")
			if note == "" {
				note = "並べた値のどれでもよい（先に書いた値を優先）。相手の機器にも、折り合う値が 1 つは要る"
			}
		}
		p.add(peer, feature, dev, title(t, col, p), v, v, note)
	}
	listed("ike_encryption", "ike_encryption_more", "")
	listed("ike_hash", "ike_hash_more", "")
	listed("ike_dh", "ike_dh_more", "")
	listed("ike_prf", "ike_prf_more", "")
	p.same(peer, feature, t, r, "", "ike_lifetime")
	listed("esp_encryption", "esp_encryption_more", "")
	listed("esp_hash", "esp_hash_more", "")
	p.same(peer, feature, t, r, "", "ah_hash")
	p.same(peer, feature, t, r, "空は tunnel", "ipsec_mode")
	// What the tunnel carries, where it is not plain IP (EtherIP that this
	// row protects in transport mode; appendix A 3.20).
	if tun := p.find("tunnels", dev, "if_name", r.Get("tunnel_if")); tun != nil && r.Get("tunnel_if") != "" && tun.Get("type") == "etherip" {
		p.add(peer, feature, dev, "守るトンネルの種類", "etherip", "etherip", "EtherIP（L2 の延伸）を IPsec のトランスポートモードで守る。相手の機器も同じ形にする")
	}
	listed("pfs", "pfs_more", "")
	p.same(peer, feature, t, r, "", "ipsec_lifetime", "ipsec_lifetime_kb")
	// The further candidates (one combination each), in their order.
	if pt := p.e.Schema.Table("ipsec_proposals"); pt != nil {
		var more []*model.Row
		for _, x := range p.rows("ipsec_proposals", dev) {
			if x.Get("tunnel") == r.Get("name") {
				more = append(more, x)
			}
		}
		sort.SliceStable(more, func(i, j int) bool {
			if a, b := more[i].Get("phase"), more[j].Get("phase"); a != b {
				return a < b
			}
			a, _ := strconv.Atoi(more[i].Get("seq"))
			b, _ := strconv.Atoi(more[j].Get("seq"))
			return a < b
		})
		n := map[string]int{}
		for _, x := range more {
			ph := x.Get("phase")
			n[ph]++
			var parts []string
			for _, col := range []string{"encryption", "hash", "dh", "prf", "ah_hash", "lifetime", "lifetime_kb"} {
				if v := p.value(pt, x, col); v != "" {
					parts = append(parts, title(pt, col, p)+" "+v)
				}
			}
			v := strings.Join(parts, "、")
			if v == "" {
				v = "（機器の既定の提案）"
			}
			what := "IKE"
			if ph == "ipsec" {
				what = "IPsec"
			}
			p.add(peer, feature, dev, fmt.Sprintf("%s の追加の候補 %d", what, n[ph]+1), v, v, "1 番目の候補（上の値）でまとまらないときに出す候補。相手の機器にも、折り合う候補が 1 つは要る")
		}
	}
	if v := p.value(t, r, "local_id"); v != "" {
		p.add(peer, feature, dev, "自分の ID", v, v, "相手の機器では「相手の ID」に書く")
	}
	if v := p.value(t, r, "remote_id"); v != "" {
		p.add(peer, feature, dev, "相手の ID", v, v, "相手の機器では「自分の ID」に書く")
	}
	for _, s := range p.rows("ipsec_selectors", dev) {
		if s.Get("tunnel") != r.Get("name") {
			continue
		}
		proto := s.Get("protocol")
		if proto == "" {
			proto = "ip"
		}
		end := func(net, port string) string {
			if port != "" {
				return net + " ポート " + port
			}
			return net
		}
		mine, theirs := end(s.Get("local_net"), s.Get("local_port")), end(s.Get("remote_net"), s.Get("remote_port"))
		p.add(peer, feature, dev, "対象通信 "+s.Get("seq")+"（自分側 → 相手側）",
			mine+" → "+theirs+"（"+proto+"）", theirs+" → "+mine+"（"+proto+"）", "相手の機器では、自分側と相手側が入れ替わる")
	}
	p.same(peer, feature, t, r, "両端で同じ設定にするのが普通（違っていてもつながる）", "dpd_interval", "dpd_retry", "dpd_retry_interval", "dpd_mode", "nat_traversal", "nat_keepalive", "nat_force")
}

// title gives the heading of a column in the language of the case.
func title(t *schema.Table, col string, p *peerTable) string {
	if c := t.Column(col); c != nil {
		return c.Title.In(p.e.lang())
	}
	return col
}

func (p *peerTable) tunnel(t *schema.Table, r *model.Row) {
	dev, peer := r.Get("device"), r.Get("destination")
	// A tunnel over IPv6, or one whose other end is written by its name.
	for _, col := range []string{"destination_v6", "destination_fqdn"} {
		if peer == "" && t.Column(col) != nil {
			peer = r.Get(col)
		}
	}
	feature := "トンネル（" + r.Get("if_name") + "）"
	p.same(peer, feature, t, r, "", "type")
	src := r.Get("source")
	local := p.addrOf(dev, src)
	ours := src
	if local != "" && local != src {
		ours = src + "（" + local + "）"
	}
	note := "相手の機器では「宛先」に書く"
	if local == "" {
		note = noLocal
	}
	p.add(peer, feature, dev, "送信元", ours, local, note)
	p.add(peer, feature, dev, "宛先", peer, peer, "相手の機器では「送信元」に書く")
	p.same(peer, feature, t, r, "", "key", "keepalive_interval", "keepalive_retry")
	if it := p.e.Schema.Table("interfaces"); it != nil {
		if i := p.find("interfaces", dev, "name", r.Get("if_name")); i != nil {
			if v := i.Get("ipv4"); v != "" {
				p.add(peer, feature, dev, "トンネル IF のアドレス", v, "", "相手の機器には、同じサブネットの別のアドレスを書く")
			}
			p.same(peer, feature, it, i, "", "ip_mtu", "tcp_mss")
		}
	}
}

func (p *peerTable) bgp(t *schema.Table, r *model.Row) {
	dev, peer := r.Get("device"), r.Get("neighbor")
	feature := "BGP"
	if vrf := r.Get("vrf"); vrf != "" {
		feature = "BGP（VRF " + vrf + "）"
	}
	p.add(peer, feature, dev, "相手 AS 番号", r.Get("remote_as"), r.Get("remote_as"), "相手の機器自身の AS 番号")
	if pr := p.find("bgp_processes", dev, "vrf", r.Get("vrf")); pr != nil {
		p.add(peer, feature, dev, "こちらの AS 番号", pr.Get("asn"), pr.Get("asn"), "相手の機器では「相手 AS 番号」に書く")
	}
	// The address the other side points at: the one this side sends from,
	// or the one on the link to the neighbour.
	local := ""
	if src := r.Get("update_source"); src != "" {
		local = p.addrOf(dev, src)
	} else if na, err := netip.ParseAddr(peer); err == nil {
		for _, i := range p.rows("interfaces", dev) {
			if pf, err := netip.ParsePrefix(i.Get("ipv4")); err == nil && pf.Contains(na) {
				local = pf.Addr().String()
				break
			}
		}
	}
	note := "相手の機器では、こちらのアドレスをネイバーに書く"
	if local == "" {
		note = noLocal
	}
	p.add(peer, feature, dev, "ネイバーアドレス", peer, local, note)
	p.same(peer, feature, t, r, "", "password", "ebgp_multihop", "keepalive", "holdtime")
}

func (p *peerTable) ospf(t *schema.Table, r *model.Row) {
	dev := r.Get("device")
	i := p.find("interfaces", dev, "name", r.Get("if_name"))
	if i == nil {
		return
	}
	pf, err := netip.ParsePrefix(i.Get("ipv4"))
	if err != nil {
		return
	}
	peer := pf.Masked().String()
	feature := "OSPF（" + r.Get("if_name") + "）"
	p.add(peer, feature, dev, "こちらのアドレス", pf.String(), "", "相手の機器には、同じサブネットの別のアドレスを書く")
	p.same(peer, feature, t, r, "", "area")
	if at := p.e.Schema.Table("ospf_areas"); at != nil {
		for _, a := range p.rows("ospf_areas", dev) {
			if a.Get("process_id") == r.Get("process_id") && a.Get("area") == r.Get("area") {
				p.same(peer, feature, at, a, "", "type")
			}
		}
	}
	p.same(peer, feature, t, r, "", "network_type", "hello", "dead", "auth_type", "auth_key_id", "auth_key")
	if it := p.e.Schema.Table("interfaces"); it != nil {
		p.same(peer, feature, it, i, "", "mtu", "ip_mtu")
	}
}

// ExportPeer writes the table of the values the devices outside the case
// have to match (spec 13.5): a list of everything, and a sheet for each.
func (e *Env) ExportPeer(mask bool, now time.Time) (*PeerOutcome, error) {
	_, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	if bk == nil {
		return nil, fmt.Errorf("%s がありません。先にパラメータブックを作ってください", e.Proj.Config.Book)
	}
	filled, _, notes := e.bookData(bk)
	out := &PeerOutcome{Masked: mask, Notes: notes}
	p := &peerTable{e: e, ds: filled, mask: mask}
	in := check.Input{Schema: e.Schema, Data: filled, Filled: filled, HasPack: func(id string) bool { return e.Packs[id] != nil }, Reads: e.packReads}
	for _, f := range check.Cross(in, check.NewTally()) {
		if f.RuleID != "X-300" {
			continue
		}
		t := e.Schema.Table(f.Ref.Table)
		if t == nil {
			continue
		}
		var row *model.Row
		for _, r := range p.rows(t.Name, f.Device) {
			if check.RefOf(t, r, "").Key == f.Ref.Key {
				row = r
				break
			}
		}
		if row == nil {
			continue
		}
		switch t.Name {
		case "ipsec_tunnels":
			p.ipsec(t, row)
		case "tunnels":
			p.tunnel(t, row)
		case "bgp_neighbors":
			p.bgp(t, row)
		case "ospf_interfaces":
			p.ospf(t, row)
		}
	}
	if len(p.items) == 0 {
		return out, nil
	}

	header := []string{"相手", "機能", "項目", "こちらの機器", "こちらの設定", "相手に設定してもらう値", "備考"}
	line := func(it peerItem) []string {
		return []string{it.peer, it.feature, it.item, it.device, it.ours, it.theirs, it.note}
	}
	all := report.Sheet{Name: "一覧", Lead: "管理外の相手すべての値です。相手ごとのシートにも、同じ値があります。", Header: header}
	var peers []string
	by := map[string][]peerItem{}
	for _, it := range p.items {
		all.Rows = append(all.Rows, line(it))
		if _, seen := by[it.peer]; !seen {
			peers = append(peers, it.peer)
		}
		by[it.peer] = append(by[it.peer], it)
	}
	d := &report.Doc{Title: "相手に渡すパラメータ表", Facts: e.facts(now)}
	d.Counts = []report.Pair{{Label: "管理外の相手", Value: num(len(peers))}, {Label: "項目", Value: num(len(p.items))}}
	d.Notes = append(d.Notes,
		"案件の中に相手の機器がない設定（対向のチェックの X-300）について、相手の機器と合わせる値を並べています。IPsec・トンネル・BGP・OSPF が対象です。",
		"「相手に設定してもらう値」は、相手の機器から見た形に直してあります（アドレスや対象通信の向きを入れ替えています）。空の欄は、備考を見てください。",
		"値は、パラメータブックの値を、機器のメーカーの既定値で補ったものです。")
	if mask {
		d.Notes = append(d.Notes, "秘密の値は "+masked+" に伏せています。")
	} else {
		d.Notes = append(d.Notes, "秘密の値（事前共有鍵・パスワード・認証の鍵）を、そのまま書いています。渡す相手と渡し方に気を付けてください。")
	}
	d.Sheets = append(d.Sheets, all)
	used := map[string]bool{"一覧": true, "まとめ": true}
	for _, peer := range peers {
		name := "相手 " + strings.NewReplacer("/", "_", ":", "_", "\\", "_", "?", "_", "*", "_", "[", "_", "]", "_").Replace(peer)
		if r := []rune(name); len(r) > 31 {
			name = string(r[:31])
		}
		for n := 2; used[name]; n++ {
			base := []rune(name)
			if len(base) > 28 {
				base = base[:28]
			}
			name = fmt.Sprintf("%s_%d", string(base), n)
		}
		used[name] = true
		s := report.Sheet{Name: name, Lead: "相手 " + peer + " と合わせる値です。", Header: header}
		for _, it := range by[peer] {
			s.Rows = append(s.Rows, line(it))
		}
		d.Sheets = append(d.Sheets, s)
	}
	files, err := e.writeReport(now, "peer_parameters", d)
	if err != nil {
		return nil, err
	}
	out.Files, out.Peers, out.Items = files, len(peers), len(p.items)
	e.Proj.Log(now, "export peer: 管理外の相手 %d、項目 %d、秘密の値は%s、レポート %s", out.Peers, out.Items, secretWord(mask), files.XLSX)
	return out, nil
}
