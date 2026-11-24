package book

import (
	"strings"
	"unicode"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// DeviceMap is a row of the sheet 新旧対照表（機器） (appendix A 6.1).
type DeviceMap struct {
	OldHost, OldVendor, NewHost, NewVendor, NewModel, Note string
	Row                                                    int // row of the sheet; 0 = not in the book
	// Out: the new host name is "-": the old device is not carried over
	// (spec 14.18.1). NewHost is "" then, as for an empty cell.
	Out bool
}

// IfMap is a row of the sheet 新旧対照表（ポート） (appendix A 6.2). NewIf "" = not
// decided yet, "-" = not carried over.
type IfMap struct {
	NewHost, OldIf, NewIf, Basis string
	Row                          int
	// OldHost and Now are written by the tool for people to read (spec
	// 14.19.5): the old device of the interface and its settings now. They
	// are written anew with every import; nothing reads them back.
	OldHost, Now string
}

// Common is a row of the sheet 共通値 (appendix A 6.3).
type Common struct {
	Scope, Table, Column, Value string
	Row                         int
}

// mgmt describes a sheet of the book that is not a table.
type mgmt struct {
	id    string // the name in Book.Sheets
	title schema.Text
	cols  []mgmtCol
	// old are the names the sheet had before (spec 14.14.1): a book made
	// by an older version is read as it is.
	old []string
}

type mgmtCol struct {
	name  string
	title schema.Text
	note  schema.Text
}

var (
	deviceSheet = &mgmt{"@devices", schema.Text{JA: "新旧対照表（機器）", EN: "Device map"}, []mgmtCol{
		{"old_host", schema.Text{JA: "旧ホスト名", EN: "Old hostname"}, schema.Text{JA: "旧コンフィグのホスト名（取り込んだ値）", EN: "The hostname in the old configuration"}},
		{"old_vendor", schema.Text{JA: "旧メーカー", EN: "Old vendor"}, schema.Text{JA: "取り込みに使ったメーカーパック（表示用）", EN: "The vendor pack that read the old configuration"}},
		{"new_host", schema.Text{JA: "新ホスト名", EN: "New hostname"}, schema.Text{JA: "新しい機器の名前。空 = まだ決めていない。- と書くと、その旧機器は新しい機器に持っていかない（理由は備考に）", EN: "The name of the new device. Empty = not decided yet. Write - for an old device that is not carried over (the reason in Note)"}},
		{"new_vendor", schema.Text{JA: "新メーカー", EN: "New vendor"}, schema.Text{JA: "▼ から選ぶ（メーカーパックの ID）", EN: "Pick from the list (the id of the vendor pack)"}},
		{"new_model", schema.Text{JA: "新機種", EN: "New model"}, schema.Text{JA: "▼ から選ぶ（新メーカーの機種だけが出る）。選ぶと新旧対照表（ポート）の案が機種のポートの並びから入る", EN: "Pick from the list (the models of the new vendor). The interface map is proposed from its ports"}},
		{"note", schema.Text{JA: "備考", EN: "Note"}, schema.Text{JA: "持っていかない機器は、その理由", EN: "Why the old device is not carried over"}},
	}, []string{"機器対応"}}
	ifSheet = &mgmt{"@ifs", schema.Text{JA: "新旧対照表（ポート）", EN: "Interface map"}, []mgmtCol{
		{"new_host", schema.Text{JA: "新ホスト名", EN: "New hostname"}, schema.Text{}},
		{"old_host", schema.Text{JA: "旧ホスト名", EN: "Old hostname"}, schema.Text{JA: "ツールが書く（表示用）。その旧 IF を持つ旧機器", EN: "Written by the tool (to read): the old device of the interface"}},
		{"old_if", schema.Text{JA: "旧 IF 名", EN: "Old interface"}, schema.Text{}},
		{"now", schema.Text{JA: "旧 IF の設定", EN: "Old interface settings"}, schema.Text{JA: "ツールが書く（表示用）。旧 IF の今の設定。取り込むたびに書き直す", EN: "Written by the tool (to read): the settings of the old interface, written anew with every import"}},
		{"new_if", schema.Text{JA: "新 IF 名", EN: "New interface"}, schema.Text{JA: "空なら未決。- と書くと「新機器には持っていかない」", EN: "Empty = not decided. Write - for an interface that is not carried over"}},
		{"basis", schema.Text{JA: "案の根拠", EN: "Basis"}, schema.Text{JA: "ツールが書く。人が直したら空にする", EN: "Written by the tool. Clear it when you change the name"}},
	}, []string{"IF 対応"}}
	commonSheet = &mgmt{"@common", schema.Text{JA: "共通値", EN: "Common values"}, []mgmtCol{
		{"scope", schema.Text{JA: "範囲", EN: "Scope"}, schema.Text{JA: "* か @グループ名", EN: "* or @group"}},
		{"table", schema.Text{JA: "表", EN: "Table"}, schema.Text{JA: "表のシート名か内部名", EN: "The sheet name or the internal name of the table"}},
		{"column", schema.Text{JA: "列", EN: "Column"}, schema.Text{JA: "列の表示名か内部名", EN: "The heading or the internal name of the column"}},
		{"value", schema.Text{JA: "値", EN: "Value"}, schema.Text{JA: "行の値が空のときに使う値", EN: "The value used when the cell of a row is empty"}},
	}, nil}
	mgmtSheets = []*mgmt{deviceSheet, ifSheet, commonSheet}
	// goneCols are the headings of columns a sheet had before and has no
	// more (spec 14.17.3): a book made by an older version keeps the column,
	// which is not read and not shown in the table editor.
	goneCols = map[*mgmt][]string{deviceSheet: {"新 OS 版", "New OS version"}}
)

func (m *mgmt) col(name string) int {
	for i, c := range m.cols {
		if c.name == name {
			return i
		}
	}
	return -1
}

func (m *mgmt) grid(lang string, rows [][]string) Grid {
	g := Grid{Name: m.title.In(lang), Rows: rows, Notes: map[int]string{}}
	for i, c := range m.cols {
		g.Header = append(g.Header, c.title.In(lang))
		if n := c.note.In(lang); n != "" {
			g.Notes[i] = n
		}
	}
	return g
}

func (d DeviceMap) cells() []string {
	host := d.NewHost
	if d.Out && host == "" {
		host = "-"
	}
	return []string{d.OldHost, d.OldVendor, host, d.NewVendor, d.NewModel, d.Note}
}

func (m IfMap) cells() []string {
	return []string{m.NewHost, m.OldHost, m.OldIf, m.Now, m.NewIf, m.Basis}
}

// foldName makes a sheet name or a heading comparable: case and spaces do
// not matter ("IF 対応" = "IF対応").
func foldName(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if !unicode.IsSpace(r) {
			sb.WriteRune(unicode.ToLower(r))
		}
	}
	return sb.String()
}

// The words of a bool in the book (spec 6.2).
func boolWords(lang string) (yes, no string) {
	if lang == "en" {
		return "yes", "no"
	}
	return "有効", "無効"
}

// Show gives the text of a cell as the book shows it: a bool as 有効/無効,
// the items of a list joined by ", ".
func Show(c *schema.Column, cell *model.Cell, lang string) string {
	if cell == nil {
		return ""
	}
	vals := cell.Values
	if c != nil && c.Type == "bool" {
		yes, no := boolWords(lang)
		vals = make([]string, len(cell.Values))
		for i, v := range cell.Values {
			switch v {
			case "true":
				vals[i] = yes
			case "false":
				vals[i] = no
			default:
				vals[i] = v
			}
		}
	}
	return strings.Join(vals, ", ")
}

// heading is the header cell of a column, and note the comment on it.
func heading(c *schema.Column, lang string) string { return c.Title.In(lang) }

func note(t *schema.Table, c *schema.Column, lang string) string {
	var parts []string
	if n := c.Note.In(lang); n != "" {
		parts = append(parts, n)
	}
	switch {
	case t.IsKey(c.Name):
		if lang == "en" {
			parts = append(parts, "Part of the key of the row")
		} else {
			parts = append(parts, "行を見分ける列（キー）")
		}
	case c.Required:
		if lang == "en" {
			parts = append(parts, "Required")
		} else {
			parts = append(parts, "必須")
		}
	}
	// An example of the way of writing (spec 6.6), where the type has one
	// and the note of the column does not show it already.
	if ex := map[string]string{"ip4": "192.0.2.1", "cidr4": "192.0.2.1/24", "net4": "192.0.2.0/24",
		"ip6": "2001:db8::1", "cidr6": "2001:db8::1/64", "net6": "2001:db8::/64",
		"ip": "192.0.2.1", "net": "192.0.2.0/24"}[c.Type]; ex != "" && !strings.Contains(c.Note.In(lang), ex) {
		if lang == "en" {
			parts = append(parts, "Example: "+ex)
		} else {
			parts = append(parts, "例: "+ex)
		}
	}
	if c.List {
		if lang == "en" {
			parts = append(parts, "Several values: separate with commas or line breaks")
		} else {
			parts = append(parts, "複数の値はカンマか改行で区切る")
		}
	}
	return strings.Join(parts, "\n")
}

// choices gives the drop-down list of a column, or nil: the words of an enum
// or of a bool. A column that also takes free words, a list column, and a
// list too long for Excel get none.
func choices(c *schema.Column, lang string) []string {
	if c.List {
		return nil
	}
	var list []string
	switch c.Type {
	case "bool":
		yes, no := boolWords(lang)
		list = []string{yes, no}
	case "enum":
		list = append(list, c.Values...)
	default:
		return nil
	}
	for _, a := range c.Also {
		if strings.HasSuffix(a, "*") {
			return nil
		}
		list = append(list, a)
	}
	n := len(list) - 1
	for _, s := range list {
		n += len(s)
		if strings.Contains(s, ",") {
			return nil
		}
	}
	if n > maxChoiceLen {
		return nil
	}
	return list
}

// TableGrid lays the rows of a table out as its sheet: the columns in the
// order of the definition, the free columns of the rows after them.
func TableGrid(t *schema.Table, rows []*model.Row, lang string) Grid {
	g := Grid{Name: t.Title.In(lang), Notes: map[int]string{}, Choices: map[int][]string{}}
	for i, c := range t.Columns {
		g.Header = append(g.Header, heading(c, lang))
		if n := note(t, c, lang); n != "" {
			g.Notes[i] = n
		}
		if list := choices(c, lang); list != nil {
			g.Choices[i] = list
		}
	}
	extra := map[string]int{}
	for _, r := range rows {
		for _, kv := range r.Extra {
			if _, ok := extra[kv.Name]; !ok {
				extra[kv.Name] = len(g.Header)
				g.Header = append(g.Header, kv.Name)
			}
		}
	}
	for _, r := range rows {
		line := make([]string, len(g.Header))
		for i, c := range t.Columns {
			line[i] = Show(c, r.Cells[c.Name], lang)
		}
		for _, kv := range r.Extra {
			line[extra[kv.Name]] = kv.Value
		}
		g.Rows = append(g.Rows, line)
	}
	return g
}
