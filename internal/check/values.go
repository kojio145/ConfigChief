package check

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"unicode"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/tmpl"
)

// Input is what the checks of the values look at.
type Input struct {
	Schema *schema.Set
	// Data is the design data of each device: the book with its rows for "*"
	// and "@group" given to the devices (book.Expand).
	Data *model.Dataset
	// Book is the rows as the book has them (nil = not looked at): a row
	// without a device is in no device's data.
	Book *model.Dataset
	// HasPack tells whether there is a vendor pack of that ID.
	HasPack func(id string) bool
	// Reads tells whether the vendor pack of that ID reads a table from a
	// configuration; nil = every pack reads every table. A name that points
	// into a table the pack does not read yet is a warning (V-115), not an
	// error: what it names may be in the lines the pack left to a person.
	Reads func(pack, table string) bool
	// Filled is Data with the defaults of the vendor packs put in, for the
	// checks that set two devices side by side (Cross); nil = Data.
	Filled *model.Dataset
	// Placeholders are the words that stand for "not decided yet" in a
	// secret; nil = DefaultPlaceholders.
	Placeholders []string
}

// DefaultPlaceholders are the words V-218 looks for.
var DefaultPlaceholders = []string{"changeme", "password", "xxxx", "todo", "tbd", "仮"}

// MaxHostname is the longest host name V-206 lets through.
const MaxHostname = 63

type values struct {
	in    Input
	tally *Tally
	eng   *tmpl.Engine
	out   []Finding
	// byDev is the rows of the tables by device, made when first asked for:
	// a check that goes through a whole table for every row of another takes
	// seconds with some hundred devices (spec 16).
	byDev map[string]map[string][]*model.Row
}

// of gives the rows of a table that are of a device, in the order of the
// table. A table that is not per device has all its rows under "".
func (x *values) of(table, dev string) []*model.Row {
	by, ok := x.byDev[table]
	if !ok {
		by = map[string][]*model.Row{}
		if t, tab := x.in.Schema.Table(table), x.in.Data.Tables[table]; t != nil && tab != nil {
			for _, r := range tab.Rows {
				d := device(t, r)
				by[d] = append(by[d], r)
			}
		}
		if x.byDev == nil {
			x.byDev = map[string]map[string][]*model.Row{}
		}
		x.byDev[table] = by
	}
	return by[dev]
}

// rowRec hands a row to the expression of a check.
type rowRec struct {
	t *schema.Table
	r *model.Row
}

func (o rowRec) Field(name string) (any, bool) {
	c := o.t.Column(name)
	if c == nil {
		return nil, false
	}
	if cell := o.r.Cells[name]; cell != nil {
		return c.Value(cell.Values), true
	}
	return c.Value(nil), true
}

func (o rowRec) Fields() []string {
	out := make([]string, len(o.t.Columns))
	for i, c := range o.t.Columns {
		out[i] = c.Name
	}
	return out
}

// Values runs the checks of the values (appendix C 1.1) that need nothing
// but the design data. What they looked at goes to the tally.
func Values(in Input, tally *Tally) []Finding {
	x := &values{in: in, tally: tally, eng: tmpl.New(tmpl.Options{})}
	if in.Placeholders == nil {
		x.in.Placeholders = DefaultPlaceholders
	}
	// Each of these ran, also when the data holds nothing for it to look at.
	for _, id := range []string{"V-101", "V-102", "V-103", "V-104", "V-106", "V-109", "V-110", "V-115",
		"V-201", "V-202", "V-203", "V-204", "V-205", "V-206", "V-207", "V-208", "V-210", "V-211", "V-212", "V-213", "V-214", "V-215", "V-216", "V-217", "V-218",
		"V-221", "V-222", "V-223", "V-224", "V-225", "V-226",
		"V-231", "V-232", "V-233", "V-234",
		"V-251", "V-252", "V-253", "V-254", "V-255", "V-256", "V-257", "V-258", "V-259"} {
		tally.Look(id, "", 0)
	}
	x.noDevice()
	for _, t := range in.Schema.Tables {
		tab := in.Data.Tables[t.Name]
		if tab == nil {
			continue
		}
		for _, r := range tab.Rows {
			x.cells(t, r)
			x.refs(t, r)
		}
	}
	x.devices()
	x.interfaces()
	x.routes()
	x.redundancy()
	x.vrrp3()
	x.routerIDs()
	x.unused()
	x.emptyLists()
	x.trackSources()
	x.dhcp()
	x.ipsec()
	x.ipsecMore()
	x.selectorPorts()
	x.pppoe()
	for _, id := range l2tpIDs {
		tally.Look(id, "", 0)
	}
	x.l2tp()
	x.l2tpv3()
	x.routePolicies()
	x.vlans()
	x.ipv6()
	x.l3()
	x.f1()
	for _, id := range basicsIDs {
		tally.Look(id, "", 0)
	}
	x.basics()

	const noModels = "どのメーカーパックにも、機種の定義（models.yaml）がまだありません"
	tally.Skip("V-111", "", noModels)
	tally.Skip("V-114", "", noModels)
	tally.Skip("V-112", "", "列と語の対応表の結び付きが、パックの定義にまだありません。書けない語は、生成のときに G-603 で確かめます")
	tally.Skip("V-209", "", "ハッシュがどのメーカーのものかを、パラメータブックからは決められません。書けないハッシュは、生成のときにパックが誤りにします（G-601）")
	return x.out
}

func device(t *schema.Table, r *model.Row) string {
	if col := t.DeviceColumn(); col != "" {
		return r.Get(col)
	}
	return ""
}

func (x *values) add(rule string, t *schema.Table, r *model.Row, column, msg string, ev ...model.KV) {
	rl, _ := Lookup(rule)
	f := New(rule, rl.Severity, device(t, r), RefOf(t, r, column), msg, ev...)
	f.At = CellAt(r, column)
	x.out = append(x.out, f)
}

// shown is a value as a finding may tell it.
func shown(c *schema.Column, v string) string {
	if c.Secret {
		return Seal(v)
	}
	return v
}

// noDevice tells about the rows of the book that name no device: they are in
// no device's data (V-101).
func (x *values) noDevice() {
	if x.in.Book == nil {
		return
	}
	for _, t := range x.in.Schema.Tables {
		tab := x.in.Book.Tables[t.Name]
		if tab == nil || t.DeviceColumn() != "device" {
			continue
		}
		for _, r := range tab.Rows {
			if r.Get("device") != "" {
				continue
			}
			x.tally.Look("V-101", "", 1)
			f := New("V-101", Error, "", RefOf(t, r, "device"), "「機器」が空です。この行は、どの機器にも使われません")
			f.At = CellAt(r, "device")
			x.out = append(x.out, f)
		}
	}
}

// cells looks at each cell of a row by itself: V-101 to V-104, V-110, V-218
// and the checks the table definition writes as expressions.
func (x *values) cells(t *schema.Table, r *model.Row) {
	dev := device(t, r)
	for _, c := range t.Columns {
		cell := r.Cells[c.Name]
		empty := cell == nil || len(cell.Values) == 0
		// An empty "explicit" column gets the default of the target pack at
		// generation (V-113), or G-606 if the pack has none.
		if c.Required && !c.Explicit {
			x.tally.Look("V-101", dev, 1)
			if empty {
				x.add("V-101", t, r, c.Name, fmt.Sprintf("「%s」は必須です", c.Title.JA))
			}
		}
		if empty {
			continue
		}
		ranged := c.Type == "int" && (c.Min != nil || c.Max != nil)
		for _, v := range cell.Values {
			x.tally.Look("V-102", dev, 1)
			if ranged {
				x.tally.Look("V-103", dev, 1)
			}
			if c.Type == "enum" {
				x.tally.Look("V-104", dev, 1)
			}
			if _, err := c.Normalize(v); err != nil {
				rule := "V-102"
				switch {
				case c.Type == "enum":
					rule = "V-104"
				case ranged:
					if _, e := strconv.Atoi(strings.TrimSpace(v)); e == nil {
						rule = "V-103"
					}
				}
				x.add(rule, t, r, c.Name, fmt.Sprintf("「%s」: %v", c.Title.JA, err), model.KV{Name: "値", Value: v})
			}
		}
		x.tally.Look("V-110", dev, 1)
		if what := odd(c, cell.Values); what != "" {
			x.add("V-110", t, r, c.Name, fmt.Sprintf("「%s」の値に%sがあります", c.Title.JA, what), model.KV{Name: "値", Value: Shown(c, cell)})
		}
		if c.Secret {
			x.tally.Look("V-218", dev, 1)
			for _, v := range cell.Values {
				if x.placeholder(v) {
					x.add("V-218", t, r, c.Name, fmt.Sprintf("「%s」が仮の値のままです", c.Title.JA), model.KV{Name: "値", Value: Seal(v)})
					break
				}
			}
		}
	}
	for _, ck := range t.Checks {
		x.tally.Look(ck.ID, dev, 1)
		v, err := x.eng.Eval(ck.When, map[string]any{"row": rowRec{t, r}})
		switch {
		case err != nil:
			x.add(ck.ID, t, r, "", fmt.Sprintf("表「%s」のチェックの式を調べられません: %v", t.Title.JA, err))
		case tmpl.Truth(v):
			x.add(ck.ID, t, r, "", ck.Message.JA)
		}
	}
}

// odd tells what of V-110 the values of a cell have ("" = nothing).
func odd(c *schema.Column, vals []string) string {
	var wide, space, control bool
	for _, v := range vals {
		if v != strings.TrimSpace(v) {
			space = true
		}
		for _, r := range v {
			switch {
			case (r >= '０' && r <= '９') || (r >= 'Ａ' && r <= 'Ｚ') || (r >= 'ａ' && r <= 'ｚ'):
				wide = true
			case r == '\n' && c.Type == "text":
				// a text of several lines
			case unicode.IsControl(r):
				control = true
			}
		}
	}
	var parts []string
	if wide {
		parts = append(parts, "全角の英数字")
	}
	if space {
		parts = append(parts, "前後の空白")
	}
	if control {
		parts = append(parts, "制御文字")
	}
	return strings.Join(parts, "・")
}

func (x *values) placeholder(v string) bool {
	low := strings.ToLower(strings.TrimSpace(v))
	for _, w := range x.in.Placeholders {
		if low == strings.ToLower(w) {
			return true
		}
	}
	// One character over and over: "xxxxxx", "0000", "****".
	if rs := []rune(low); len(rs) >= 4 {
		same := true
		for _, r := range rs {
			if r != rs[0] {
				same = false
				break
			}
		}
		return same
	}
	return false
}

// has tells whether a table has, for the device, a row whose column holds
// the value.
func (x *values) has(table, column, dev, val string) bool {
	for _, r := range x.of(table, dev) {
		if r.Get(column) == val {
			return true
		}
	}
	return false
}

// refs looks at the cells that name a row of another table (V-106), and at
// the vendor of a device (V-109).
func (x *values) refs(t *schema.Table, r *model.Row) {
	dev := device(t, r)
	devCol := t.DeviceColumn()
	if t.Name == "devices" {
		if v := r.Get("vendor"); v != "" && x.in.HasPack != nil {
			x.tally.Look("V-109", dev, 1)
			if !x.in.HasPack(v) {
				x.add("V-109", t, r, "vendor", fmt.Sprintf("メーカー「%s」に当たるメーカーパックがありません", v), model.KV{Name: "値", Value: v})
			}
		}
	}
	for _, c := range t.Columns {
		cell := r.Cells[c.Name]
		if cell == nil || c.Name == devCol {
			continue
		}
		var table, column, what string
		switch {
		case c.Type == "ref" && c.Ref != "":
			table, column, _ = strings.Cut(c.Ref, ".")
			what = "表「" + x.title(table) + "」"
		case c.Type == "ifname" && !(t.Name == "interfaces" && c.Name == "name"):
			table, column, what = "interfaces", "name", "インタフェースの表"
		default:
			continue
		}
		target := x.in.Schema.Table(table)
		if target == nil || (devCol == "" && target.DeviceColumn() == "device") {
			continue // the row is of no device: the checks of the pairs look at it (M3)
		}
		for _, v := range cell.Values {
			if c.IsAlso(v) {
				continue
			}
			x.tally.Look("V-106", dev, 1)
			lookIn := dev
			if table == "devices" {
				lookIn = v
			}
			if x.has(table, column, lookIn, v) {
				continue
			}
			if vendor := x.vendor(dev); x.in.Reads != nil && vendor != "" && !x.in.Reads(vendor, table) {
				x.tally.Look("V-115", dev, 1)
				x.add("V-115", t, r, c.Name, fmt.Sprintf("「%s」の「%s」は、%sにありません。メーカーパック %s は、この表をまだ読みません。定義が追加コンフィグにあるか確かめてください",
					c.Title.JA, v, what, vendor), model.KV{Name: "値", Value: v})
				continue
			}
			x.add("V-106", t, r, c.Name, fmt.Sprintf("「%s」の「%s」は、%sにありません", c.Title.JA, v, what), model.KV{Name: "値", Value: v})
		}
	}
}

// vendor gives the vendor a device has in the table of devices.
func (x *values) vendor(host string) string {
	if tab := x.in.Data.Tables["devices"]; tab != nil {
		for _, r := range tab.Rows {
			if r.Get("hostname") == host {
				return r.Get("vendor")
			}
		}
	}
	return ""
}

func (x *values) title(table string) string {
	if t := x.in.Schema.Table(table); t != nil {
		return t.Title.JA
	}
	return table
}

// devices looks at the host names (V-206).
func (x *values) devices() {
	t := x.in.Schema.Table("devices")
	tab := x.in.Data.Tables["devices"]
	if t == nil || tab == nil {
		return
	}
	for _, r := range tab.Rows {
		h := r.Get("hostname")
		if h == "" {
			continue
		}
		x.tally.Look("V-206", h, 1)
		var bad []string
		for _, c := range h {
			ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.'
			if !ok && !strings.ContainsRune(strings.Join(bad, ""), c) {
				bad = append(bad, string(c))
			}
		}
		switch {
		case len(bad) > 0:
			x.add("V-206", t, r, "hostname", fmt.Sprintf("ホスト名に使えない文字があります（%s）。使えるのは英数字と - _ . です", strings.Join(bad, " ")), model.KV{Name: "値", Value: h})
		case len(h) > MaxHostname:
			x.add("V-206", t, r, "hostname", fmt.Sprintf("ホスト名が長すぎます（%d 文字。%d 文字まで）", len(h), MaxHostname), model.KV{Name: "値", Value: h})
		}
	}
}

// ifAddr is one IPv4 address of an interface.
type ifAddr struct {
	row    *model.Row
	column string
	device string
	name   string
	vrf    string
	prefix netip.Prefix // the address with its length, as written
}

func (a ifAddr) where() string { return a.device + " " + a.name }

// addrs gives the IPv4 addresses of the interfaces, in the order of the
// rows. dynamic tells the devices and VRFs that have an interface whose
// address the tool cannot know (dhcp, negotiated, unnumbered).
func (x *values) addrs() (list []ifAddr, dynamic map[string]bool) {
	dynamic = map[string]bool{}
	tab := x.in.Data.Tables["interfaces"]
	if tab == nil {
		return nil, dynamic
	}
	for _, r := range tab.Rows {
		dev, name, vrf := r.Get("device"), r.Get("name"), r.Get("vrf")
		for _, col := range []string{"ipv4", "ipv4_secondary"} {
			cell := r.Cells[col]
			if cell == nil {
				continue
			}
			for _, v := range cell.Values {
				p, err := netip.ParsePrefix(v)
				if err != nil || !p.Addr().Is4() {
					if col == "ipv4" {
						dynamic[dev+"\x00"+vrf] = true
					}
					continue
				}
				list = append(list, ifAddr{r, col, dev, name, vrf, p})
			}
		}
	}
	return list, dynamic
}

// broadcast gives the last address of an IPv4 network.
func broadcast(p netip.Prefix) netip.Addr {
	a := p.Masked().Addr().As4()
	bits := p.Bits()
	for i := 0; i < 4; i++ {
		for b := 0; b < 8; b++ {
			if i*8+b >= bits {
				a[i] |= 1 << (7 - b)
			}
		}
	}
	return netip.AddrFrom4(a)
}

// interfaces looks at the addresses and the settings of the interfaces:
// V-201, V-202, V-203, V-208, V-212.
func (x *values) interfaces() {
	t := x.in.Schema.Table("interfaces")
	tab := x.in.Data.Tables["interfaces"]
	if t == nil || tab == nil {
		return
	}
	list, _ := x.addrs()
	seen := map[string]ifAddr{} // vrf + address → the first interface that has it
	for i, a := range list {
		x.tally.Look("V-201", a.device, 1)
		x.tally.Look("V-202", a.device, 1)
		x.tally.Look("V-203", a.device, 1)
		for _, b := range list[:i] {
			if b.device == a.device && b.vrf == a.vrf && b.row != a.row && a.prefix.Masked().Overlaps(b.prefix.Masked()) {
				x.add("V-201", t, a.row, a.column, fmt.Sprintf("%s のサブネットと重なっています", b.name),
					model.KV{Name: a.name, Value: a.prefix.String()}, model.KV{Name: b.name, Value: b.prefix.String()})
				break
			}
		}
		key := a.vrf + "\x00" + a.prefix.Addr().String()
		if b, ok := seen[key]; ok && b.row != a.row {
			x.add("V-202", t, a.row, a.column, fmt.Sprintf("同じアドレスが %s にも付いています", b.where()),
				model.KV{Name: a.where(), Value: a.prefix.String()}, model.KV{Name: b.where(), Value: b.prefix.String()})
		} else if !ok {
			seen[key] = a
		}
		if a.prefix.Bits() < 31 {
			switch a.prefix.Addr() {
			case a.prefix.Masked().Addr():
				x.add("V-203", t, a.row, a.column, "ネットワークアドレスです（ホストのアドレスを書きます）", model.KV{Name: "値", Value: a.prefix.String()})
			case broadcast(a.prefix):
				x.add("V-203", t, a.row, a.column, "ブロードキャストアドレスです（ホストのアドレスを書きます）", model.KV{Name: "値", Value: a.prefix.String()})
			}
		}
	}
	num := func(r *model.Row, col string) (int, bool) {
		n, err := strconv.Atoi(r.Get(col))
		return n, err == nil
	}
	for _, r := range tab.Rows {
		dev := r.Get("device")
		if r.Get("enabled") == "true" {
			x.tally.Look("V-208", dev, 1)
			if r.Get("description") == "" {
				x.add("V-208", t, r, "description", "有効な IF の説明が空です")
			}
		}
		mss, ok := num(r, "tcp_mss")
		if !ok {
			continue
		}
		mtu, col, ok := 0, "ip_mtu", false
		if mtu, ok = num(r, "ip_mtu"); !ok {
			col = "mtu"
			mtu, ok = num(r, "mtu")
		}
		if !ok {
			continue
		}
		x.tally.Look("V-212", dev, 1)
		if mss > mtu-40 {
			x.add("V-212", t, r, "tcp_mss", fmt.Sprintf("TCP MSS 調整の値 %d が、MTU %d − 40 = %d より大きい値です", mss, mtu, mtu-40),
				model.KV{Name: "tcp_mss", Value: strconv.Itoa(mss)}, model.KV{Name: col, Value: strconv.Itoa(mtu)})
		}
	}
}

// routes looks at the next hops of the static routes (V-204). A route with
// an out interface is left alone, and so is one whose next hop another route
// of the device reaches (the default route does not count: it reaches every
// address). A device with an interface whose address is not known (dhcp,
// negotiated, unnumbered) in the VRF is not looked at.
func (x *values) routes() {
	t := x.in.Schema.Table("static_routes")
	tab := x.in.Data.Tables["static_routes"]
	if t == nil || tab == nil {
		return
	}
	list, dynamic := x.addrs()
	nets := map[string][]netip.Prefix{} // device and VRF → the subnets of its interfaces
	for _, a := range list {
		key := a.device + "\x00" + a.vrf
		nets[key] = append(nets[key], a.prefix.Masked())
	}
	for _, r := range tab.Rows {
		dev, vrf := r.Get("device"), r.Get("vrf")
		hop, err := netip.ParseAddr(r.Get("next_hop"))
		if err != nil || r.Get("out_if") != "" || dynamic[dev+"\x00"+vrf] {
			continue
		}
		x.tally.Look("V-204", dev, 1)
		reached := false
		for _, p := range nets[dev+"\x00"+vrf] {
			if p.Contains(hop) {
				reached = true
				break
			}
		}
		for _, o := range x.of("static_routes", dev) {
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
			x.add("V-204", t, r, "next_hop", fmt.Sprintf("ネクストホップ %s は、この機器のどの IF のサブネットにも入りません", hop), model.KV{Name: "値", Value: hop.String()})
		}
	}
}
