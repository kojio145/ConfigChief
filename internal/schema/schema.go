package schema

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/netip"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Text is a text in the languages of the tool.
type Text struct {
	JA string `yaml:"ja"`
	EN string `yaml:"en"`
}

// In gives the text in a language ("ja" or "en"); the other one stands in if
// it is empty.
func (t Text) In(lang string) string {
	if lang == "en" && t.EN != "" {
		return t.EN
	}
	if t.JA != "" {
		return t.JA
	}
	return t.EN
}

// Column is one column of a table (appendix A 1.1).
type Column struct {
	Name     string
	Title    Text
	Type     string // the type of one value: of each item for a list
	List     bool   // list<Type>
	Required bool
	Default  string
	Explicit bool
	Values   []string // the choices of an enum
	Also     []string // words allowed besides the type; a trailing * stands for any rest
	OrIfname bool     // a value may be the name of an interface of the device
	// AutoName marks a key the import makes up ("nat1", "nat2"): no line of a
	// configuration holds it, and rows are matched by what else they hold.
	AutoName bool
	// OrderOnly marks a key that is a number which only orders the rows
	// (the further proposals of IPsec): no line of a configuration holds
	// the number itself, and rows are compared by their place.
	OrderOnly bool
	Min, Max  *int
	Ref       string // "table.column"
	Note      Text
	Secret    bool
	// Generate is false for a column kept for people and for the tool: no
	// line is written from it (spec 9.6).
	Generate bool
	// Compare is false for a column only a person fills in and no
	// configuration can tell: two sources are not compared in it (spec 12.3).
	Compare bool
}

// Check is a check of one table, written as an expression.
type Check struct {
	ID      string `yaml:"id"`
	When    string `yaml:"when"`
	Message Text   `yaml:"message"`
}

// Table is the definition of one table.
type Table struct {
	Name    string
	Title   Text
	Since   string
	Key     []string
	Columns []*Column
	Checks  []Check

	byName map[string]*Column
}

// Column gives a column by its internal name, or nil.
func (t *Table) Column(name string) *Column {
	if t == nil {
		return nil
	}
	return t.byName[name]
}

// IsKey tells whether the column is part of the key.
func (t *Table) IsKey(name string) bool {
	for _, k := range t.Key {
		if k == name {
			return true
		}
	}
	return false
}

// AutoName is the name of the key column the import makes up ("" = none).
func (t *Table) AutoName() string {
	if t == nil {
		return ""
	}
	for _, c := range t.Columns {
		if c.AutoName {
			return c.Name
		}
	}
	return ""
}

// DeviceColumn is the name of the column that holds the device ("" for a
// table that is not per device).
func (t *Table) DeviceColumn() string {
	switch {
	case t.Name == "devices":
		return "hostname"
	case t.byName["device"] != nil:
		return "device"
	}
	return ""
}

// Set is every table definition, in the order of the sheets.
type Set struct {
	Tables []*Table

	byName map[string]*Table
}

// Table gives a table by its internal name, or nil.
func (s *Set) Table(name string) *Table {
	if s == nil {
		return nil
	}
	return s.byName[name]
}

// The value types (spec 5.2).
var types = map[string]bool{
	"string": true, "int": true, "bool": true, "enum": true,
	"ip4": true, "ip6": true, "cidr4": true, "cidr6": true, "net4": true, "net6": true,
	// An address or a network of either family (2026-11-21): the columns
	// both vendors can write with IPv6 too. An IPv4 value is written exactly
	// as an ip4 / net4 column writes it, so a key compares as before.
	"ip": true, "net": true,
	"ifname": true, "ref": true, "secret": true, "text": true,
}

// The sheets of the book that are not tables (appendix A 6).
var reserved = []string{"新旧対照表（機器）", "新旧対照表（ポート）", "共通値", "機器対応", "IF 対応", "Device map", "Interface map", "Common values"}

var (
	reName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	reList = regexp.MustCompile(`^list<(\w+)>$`)
)

type columnY struct {
	Name      string    `yaml:"name"`
	Title     Text      `yaml:"title"`
	Type      string    `yaml:"type"`
	Required  bool      `yaml:"required"`
	Default   string    `yaml:"default"`
	Explicit  bool      `yaml:"explicit"`
	Values    []*string `yaml:"values"` // a pointer: a bare null must not vanish
	Also      []*string `yaml:"also"`
	OrIfname  bool      `yaml:"or_ifname"`
	AutoName  bool      `yaml:"auto_name"`
	OrderOnly bool      `yaml:"order_only"`
	Min       *int      `yaml:"min"`
	Max       *int      `yaml:"max"`
	Ref       string    `yaml:"ref"`
	Note      Text      `yaml:"note"`
	Secret    bool      `yaml:"secret"`
	Generate  *bool     `yaml:"generate"`
	Compare   *bool     `yaml:"compare"`
}

type tableY struct {
	Table   string    `yaml:"table"`
	Title   Text      `yaml:"title"`
	Since   string    `yaml:"since"`
	Key     []string  `yaml:"key"`
	Columns []columnY `yaml:"columns"`
	Checks  []Check   `yaml:"checks"`
}

// Load reads the definition files (*.yaml) of a folder, by file name: the
// order of the files is the order of the sheets. All problems are reported
// together in the error.
func Load(fsys fs.FS) (*Set, error) {
	files, err := fs.Glob(fsys, "*.yaml")
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	s := &Set{byName: map[string]*Table{}}
	var errs []string
	errf := func(format string, a ...any) { errs = append(errs, fmt.Sprintf(format, a...)) }

	for _, f := range files {
		data, err := fs.ReadFile(fsys, f)
		if err != nil {
			errf("%s: %v", f, err)
			continue
		}
		var y tableY
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(&y); err != nil {
			errf("%s: %v", f, err)
			continue
		}
		t := &Table{Name: y.Table, Title: y.Title, Since: y.Since, Key: y.Key, Checks: y.Checks, byName: map[string]*Column{}}
		switch {
		case !reName.MatchString(t.Name):
			errf("%s: table「%s」は英小文字・数字・_ で書きます", f, t.Name)
			continue
		case strings.HasPrefix(t.Name, "x_"):
			errf("%s: x_ で始まる名前は作業用の表のものです", f)
			continue
		case s.byName[t.Name] != nil:
			errf("%s: 表 %s の定義が 2 つあります", f, t.Name)
			continue
		}
		if t.Title.JA == "" || t.Title.EN == "" {
			errf("%s: title の ja と en が要ります", f)
		}
		titles := map[string]string{}
		for i := range y.Columns {
			c, err := column(&y.Columns[i])
			if err != nil {
				errf("%s: 列 %s: %v", f, y.Columns[i].Name, err)
				continue
			}
			if t.byName[c.Name] != nil {
				errf("%s: 列 %s が 2 つあります", f, c.Name)
				continue
			}
			// A heading of the book may be any of the three names: none may
			// point at two columns.
			for _, h := range []string{c.Name, c.Title.JA, c.Title.EN} {
				if prev, dup := titles[fold(h)]; dup && prev != c.Name {
					errf("%s: 見出し「%s」が列 %s と %s の両方に当たります", f, h, prev, c.Name)
				}
				titles[fold(h)] = c.Name
			}
			t.Columns = append(t.Columns, c)
			t.byName[c.Name] = c
		}
		if len(t.Key) == 0 {
			errf("%s: key がありません", f)
		}
		for _, k := range t.Key {
			if t.byName[k] == nil {
				errf("%s: key の列 %s がありません", f, k)
			} else if t.byName[k].List {
				errf("%s: key の列 %s は list にできません", f, k)
			}
		}
		autos := 0
		for _, c := range t.Columns {
			if c.AutoName {
				autos++
				if !t.IsKey(c.Name) {
					errf("%s: auto_name の列 %s は key の列にしてください", f, c.Name)
				}
			}
		}
		if autos > 1 {
			errf("%s: auto_name の列は 1 つまでです", f)
		}
		seen := map[string]bool{}
		for _, c := range t.Checks {
			if c.ID == "" || c.When == "" || c.Message.JA == "" {
				errf("%s: checks には id・when・message.ja が要ります", f)
			}
			if seen[c.ID+c.When] {
				errf("%s: 同じ検査が 2 つあります（%s）", f, c.ID)
			}
			seen[c.ID+c.When] = true
		}
		s.Tables = append(s.Tables, t)
		s.byName[t.Name] = t
	}

	sheets := map[string]string{}
	for _, name := range reserved {
		sheets[fold(name)] = "管理用のシート"
	}
	for _, t := range s.Tables {
		for _, h := range []string{t.Title.JA, t.Title.EN} {
			if h == "" {
				continue
			}
			if len([]rune(h)) > 31 || strings.ContainsAny(h, `:\/?*[]`) {
				errf("表 %s: シート名「%s」は Excel で使えません（31 文字まで。: \\ / ? * [ ] は不可）", t.Name, h)
			}
			if prev, dup := sheets[fold(h)]; dup && prev != t.Name {
				errf("表 %s: シート名「%s」が %s と重なっています", t.Name, h, prev)
			}
			sheets[fold(h)] = t.Name
		}
		for _, c := range t.Columns {
			if c.Ref == "" {
				continue
			}
			tn, cn, _ := strings.Cut(c.Ref, ".")
			if s.byName[tn].Column(cn) == nil {
				errf("表 %s の列 %s: ref「%s」の先がありません", t.Name, c.Name, c.Ref)
			}
		}
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}
	return s, nil
}

func column(y *columnY) (*Column, error) {
	c := &Column{Name: y.Name, Title: y.Title, Type: y.Type, Required: y.Required, Default: y.Default, Explicit: y.Explicit,
		Min: y.Min, Max: y.Max, Ref: y.Ref, Note: y.Note, Secret: y.Secret, OrIfname: y.OrIfname, AutoName: y.AutoName, OrderOnly: y.OrderOnly}
	c.Generate = y.Generate == nil || *y.Generate
	c.Compare = y.Compare == nil || *y.Compare
	for _, v := range y.Values {
		if v == nil || *v == "" {
			return nil, errors.New("values に空の語があります（null・yes などの語は引用符で囲みます）")
		}
		c.Values = append(c.Values, *v)
	}
	for _, v := range y.Also {
		if v == nil || *v == "" {
			return nil, errors.New("also に空の語があります（null・yes などの語は引用符で囲みます）")
		}
		c.Also = append(c.Also, *v)
	}
	if !reName.MatchString(c.Name) {
		return nil, errors.New("name は英小文字・数字・_ で書きます")
	}
	if c.Title.JA == "" || c.Title.EN == "" {
		return nil, errors.New("title の ja と en が要ります")
	}
	if m := reList.FindStringSubmatch(c.Type); m != nil {
		c.Type, c.List = m[1], true
	}
	if !types[c.Type] {
		return nil, fmt.Errorf("type「%s」はありません", y.Type)
	}
	if c.Type == "text" && c.List {
		return nil, errors.New("text の list はありません")
	}
	if (c.Type == "enum") != (len(c.Values) > 0) {
		return nil, errors.New("values は enum に、enum には values が要ります")
	}
	if (c.Min != nil || c.Max != nil) && c.Type != "int" {
		return nil, errors.New("min・max は int だけに書けます")
	}
	if (c.Type == "ref") != (c.Ref != "") {
		return nil, errors.New("ref は type: ref に、type: ref には ref が要ります")
	}
	if c.Ref != "" && strings.Count(c.Ref, ".") != 1 {
		return nil, errors.New("ref は「表.列」の形で書きます")
	}
	if c.AutoName && (c.Type != "string" || c.List) {
		return nil, errors.New("auto_name は string の列だけに書けます")
	}
	if c.OrderOnly && (c.Type != "int" || c.List) {
		return nil, errors.New("order_only は int の列だけに書けます")
	}
	if !c.Generate && c.Explicit {
		return nil, errors.New("generate: false の列に explicit は書けません")
	}
	if c.Type == "secret" {
		c.Secret = true
	}
	if c.Default != "" {
		v, err := c.Normalize(c.Default)
		if err != nil {
			return nil, fmt.Errorf("default: %v", err)
		}
		c.Default = v
	}
	return c, nil
}

// fold makes a heading comparable: case, and the spaces around it, do not
// matter.
func fold(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// The spellings of a bool in the book and in the packs (spec 6.2).
var bools = map[string]string{
	"true": "true", "false": "false", "yes": "true", "no": "false", "1": "true", "0": "false",
	"有効": "true", "無効": "false", "○": "true", "×": "false", "〇": "true", "on": "true", "off": "false",
}

// Normalize checks one value against the type of the column and gives its
// one way of writing: 010 → 10, Yes → true, 192.0.2.1/24 stays. For a list
// column it takes one item. An empty value stays empty; a text and a secret
// are never touched.
func (c *Column) Normalize(val string) (string, error) {
	if c.Type == "text" || c.Type == "secret" {
		return val, nil // every character counts: the spaces too
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return "", nil
	}
	for _, a := range c.Also {
		if rest, ok := strings.CutSuffix(a, "*"); ok {
			if len(val) > len(rest) && strings.EqualFold(val[:len(rest)], rest) {
				return rest + val[len(rest):], nil
			}
		} else if strings.EqualFold(val, a) {
			return a, nil
		}
	}
	switch c.Type {
	case "int":
		n, err := strconv.Atoi(val)
		if err != nil {
			return "", fmt.Errorf("「%s」は整数ではありません", val)
		}
		if (c.Min != nil && n < *c.Min) || (c.Max != nil && n > *c.Max) {
			return "", fmt.Errorf("「%s」は範囲（%s）の外です", val, c.rangeText())
		}
		return strconv.Itoa(n), nil
	case "bool":
		if b, ok := bools[strings.ToLower(val)]; ok {
			return b, nil
		}
		return "", fmt.Errorf("「%s」は有効・無効として読めません", val)
	case "enum":
		for _, v := range c.Values {
			if strings.EqualFold(val, v) {
				return v, nil
			}
		}
		return "", fmt.Errorf("「%s」は選択肢（%s）にありません", val, strings.Join(c.Values, ", "))
	case "ip4", "ip6":
		a, err := netip.ParseAddr(val)
		if err != nil || a.Zone() != "" || a.Is4() != (c.Type == "ip4") {
			return "", fmt.Errorf("「%s」は %s アドレスではありません", val, family(c.Type))
		}
		return a.String(), nil
	case "ip":
		a, err := netip.ParseAddr(val)
		if err != nil || a.Zone() != "" {
			return "", fmt.Errorf("「%s」は IPv4 アドレスでも IPv6 アドレスでもありません", val)
		}
		return a.String(), nil
	case "net":
		p, err := netip.ParsePrefix(val)
		if err != nil {
			return "", fmt.Errorf("「%s」は「アドレス/長さ」の形ではありません", val)
		}
		if p != p.Masked() {
			return "", fmt.Errorf("「%s」はネットワークのアドレスではありません（%s と書きます）", val, p.Masked())
		}
		return p.String(), nil
	case "cidr4", "cidr6", "net4", "net6":
		p, err := netip.ParsePrefix(val)
		if err != nil || p.Addr().Is4() != strings.HasSuffix(c.Type, "4") {
			return "", fmt.Errorf("「%s」は %s の「アドレス/長さ」の形ではありません", val, family(c.Type))
		}
		if strings.HasPrefix(c.Type, "net") && p != p.Masked() {
			return "", fmt.Errorf("「%s」はネットワークのアドレスではありません（%s と書きます）", val, p.Masked())
		}
		return p.String(), nil
	}
	return val, nil
}

func family(typ string) string {
	if strings.HasSuffix(typ, "6") {
		return "IPv6"
	}
	return "IPv4"
}

func (c *Column) rangeText() string {
	lo, hi := "", ""
	if c.Min != nil {
		lo = strconv.Itoa(*c.Min)
	}
	if c.Max != nil {
		hi = strconv.Itoa(*c.Max)
	}
	return lo + "〜" + hi
}

// IsAlso tells whether a value is one of the words the column allows besides
// its type.
func (c *Column) IsAlso(val string) bool {
	for _, a := range c.Also {
		if rest, ok := strings.CutSuffix(a, "*"); ok {
			if len(val) > len(rest) && strings.EqualFold(val[:len(rest)], rest) {
				return true
			}
		} else if strings.EqualFold(val, a) {
			return true
		}
	}
	return false
}

// Value gives the values of a cell in the form the expressions see (spec
// 9.6): a list for a list column, a bool or a number where the column is one
// and the text reads as one, the text otherwise. An empty cell is "" (an
// empty list for a list column).
func (c *Column) Value(vals []string) any {
	if c.List {
		return append([]string{}, vals...)
	}
	v := strings.Join(vals, ",")
	switch c.Type {
	case "bool":
		switch v {
		case "true":
			return true
		case "false":
			return false
		}
	case "int":
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return v
}
