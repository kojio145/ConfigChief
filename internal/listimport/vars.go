package listimport

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
	"golang.org/x/text/unicode/norm"
)

// VarRule is a variable rule (vars/<name>.yaml, spec 9.7.2): where the table
// of a list is, which column names the device, and which columns give which
// variable. It reads a list as PalaTerm's 変数ルール does (its
// internal/model/fillrule.go and readFill of app_fill.go, v1.10.5): a row
// whose device cell is empty continues the device above, more devices on a
// row (Also), exclusions on a column or on a variable, and the differences
// a device name may have from the book's.
type VarRule struct {
	Version        int    `yaml:"version" json:"version"`
	File           string `yaml:"file" json:"file"`
	Sheet          string `yaml:"sheet,omitempty" json:"sheet"`
	SheetIndex     int    `yaml:"sheet_index,omitempty" json:"sheetIndex"`
	HeaderRow      int    `yaml:"header_row" json:"headerRow"`
	DataRow        int    `yaml:"data_row,omitempty" json:"dataRow"`
	StartCol       string `yaml:"start_col,omitempty" json:"startCol"`
	MergeOnlyFirst bool   `yaml:"merge_only_first,omitempty" json:"mergeOnlyFirst"`

	// Match: the differences left out when a device name of the list is
	// matched with the book's (case and the spaces around always are).
	Match   VarMatch  `yaml:"match,omitempty" json:"match"`
	Exclude []VarCond `yaml:"exclude,omitempty" json:"exclude"`
	Device  Field     `yaml:"device" json:"device"`
	Vars    []Var     `yaml:"vars" json:"vars"`
	// Also are the second and later devices a line lists (a redundant
	// pair's backup): their name and the variables they take from columns
	// of their own; the others they read from the first device's columns.
	Also []AlsoDevice `yaml:"also,omitempty" json:"also"`

	// Name is the rule's name (the file name without ".yaml"); Path the file
	// from the case folder.
	Name string `yaml:"-" json:"name"`
	Path string `yaml:"-" json:"path"`
}

// VarMatch are the differences of device names a matching leaves out, as
// PalaTerm's ImportMatch: 全角・半角, the spaces inside, the domain part
// (from the first "."), and "-" against "_".
type VarMatch struct {
	Width  bool `yaml:"width,omitempty" json:"width"`
	Space  bool `yaml:"space,omitempty" json:"space"`
	Domain bool `yaml:"domain,omitempty" json:"domain"`
	Dash   bool `yaml:"dash,omitempty" json:"dash"`
}

// VarCond leaves a row out: by the raw text of a column (Column) or by the
// value of a variable after its steps (Var). Op as Cond's.
type VarCond struct {
	Column string `yaml:"column,omitempty" json:"column"`
	Var    string `yaml:"var,omitempty" json:"var"`
	Op     string `yaml:"op" json:"op"`
	Value  string `yaml:"value,omitempty" json:"value"`
}

// Var is one variable of a rule. The value comes as a column mapping's
// field's does (Field.To is not used); Default fills an empty value first.
// Empty tells what a value still empty does: "" or "stop" (the device is
// not made) or "line" (the lines using it are left out).
type Var struct {
	Name  string `yaml:"name" json:"name"`
	Field `yaml:",inline"`
	Empty string `yaml:"empty,omitempty" json:"empty"`
}

// AlsoDevice is one more device on a line.
type AlsoDevice struct {
	Label  string `yaml:"label,omitempty" json:"label"`
	Device Field  `yaml:"device" json:"device"`
	Vars   []Var  `yaml:"vars,omitempty" json:"vars"`
}

// varNameRe is the form of a variable's name (spec 9.7.1): no "." (that is a
// table of the book) and nothing a template could not read.
var varNameRe = regexp.MustCompile(`^[\p{L}_][\p{L}\p{N}_\-]*$`)

const (
	maxVars     = 200
	maxAlso     = 10
	maxVarName  = 64
	maxVarRows  = 100000
	ruleFileMax = 1 << 20
	sampleLines = 8 // the lines a try shows, and 4 more per further device
)

// excludedMark is the device a continuing row would join when the
// device's own row was left out.
var excludedMark = string(rune(0)) + "excluded"

// Location is the rule as a column mapping that only says where the table
// is: what the sample and the table reading take.
func (r *VarRule) Location() *Rule {
	return &Rule{Version: 1, File: r.File, Sheet: r.Sheet, SheetIndex: r.SheetIndex, HeaderRow: r.HeaderRow,
		DataRow: r.DataRow, StartCol: r.StartCol, MergeOnlyFirst: r.MergeOnlyFirst}
}

// ParseVars reads a variable rule file.
func ParseVars(data []byte) (*VarRule, error) {
	if len(data) > ruleFileMax {
		return nil, fmt.Errorf("ファイルが大きすぎます（%d バイトまで）", ruleFileMax)
	}
	var r VarRule
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&r); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("中身がありません")
		}
		return nil, fmt.Errorf("YAML として読めません: %v", err)
	}
	return &r, nil
}

// MarshalVars writes a rule as its file holds it.
func MarshalVars(r *VarRule) ([]byte, error) {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(r); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Validate gives the faults of a rule; it also trims what it reads.
func (r *VarRule) Validate() []string {
	var out []string
	errf := func(format string, a ...any) { out = append(out, fmt.Sprintf(format, a...)) }
	out = append(out, r.Location().validateLocation()...)
	if r.Device.Source() == "" {
		errf("device（機器名の列）を column・template・value のどれか 1 つで書きます")
	} else {
		for _, p := range r.Device.validate() {
			errf("device: %s", p)
		}
	}
	if len(r.Vars) == 0 {
		errf("vars（変数）がありません")
	}
	if len(r.Vars) > maxVars {
		errf("vars は %d 個までです", maxVars)
	}
	seen := map[string]bool{}
	for i := range r.Vars {
		v := &r.Vars[i]
		v.Name = strings.TrimSpace(v.Name)
		at := fmt.Sprintf("vars %d", i+1)
		if v.Name != "" {
			at = fmt.Sprintf("変数「%s」", v.Name)
		}
		switch {
		case v.Name == "":
			errf("%s: name（変数の名前）がありません", at)
		case len([]rune(v.Name)) > maxVarName:
			errf("%s: 名前は %d 文字までです", at, maxVarName)
		case !varNameRe.MatchString(v.Name):
			errf("%s: 名前は文字か _ で始め、文字・数字・_・- で書きます（. は使えません。ブックの表と見分けるため）", at)
		case reservedVar(v.Name):
			errf("%s: この名前は書き方と重なるため使えません", at)
		}
		low := strings.ToLower(v.Name)
		if v.Name != "" && seen[low] {
			errf("%s: 同じ名前の変数が 2 つあります（大文字と小文字は区別しません）", at)
		}
		seen[low] = true
		if v.Source() == "" && v.Default == "" {
			errf("%s: 値の元を column・template・value のどれか 1 つで書きます", at)
		}
		for _, p := range v.validate() {
			errf("%s: %s", at, p)
		}
		switch v.Empty {
		case "", "stop", "line":
		default:
			errf("%s: empty「%s」は使えません（stop・line。決まった値は default に書きます）", at, v.Empty)
		}
	}
	if len(r.Exclude) > maxExcludes {
		errf("exclude は %d 件までです", maxExcludes)
	}
	for i := range r.Exclude {
		c := &r.Exclude[i]
		c.Column, c.Var = strings.TrimSpace(c.Column), strings.TrimSpace(c.Var)
		switch {
		case c.Column == "" && c.Var == "":
			errf("exclude %d: column（一覧の見出し）か var（変数）を書きます", i+1)
		case c.Column != "" && c.Var != "":
			errf("exclude %d: column と var は、どちらか 1 つだけ書きます", i+1)
		case c.Var != "" && !seen[strings.ToLower(c.Var)]:
			errf("exclude %d: 変数「%s」は vars にありません", i+1, c.Var)
		}
		ok := false
		for _, op := range CondOps {
			ok = ok || c.Op == op
		}
		if !ok {
			errf("exclude %d: op「%s」は使えません（%s）", i+1, c.Op, strings.Join(CondOps, "、"))
		}
	}
	if len(r.Also) > maxAlso {
		errf("also は %d 台分までです", maxAlso)
	}
	for i := range r.Also {
		a := &r.Also[i]
		at := fmt.Sprintf("also %d（%d 台目）", i+1, i+2)
		if a.Device.Source() == "" {
			errf("%s: device（機器名の列）を書きます", at)
		}
		for _, p := range a.Device.validate() {
			errf("%s の device: %s", at, p)
		}
		for j := range a.Vars {
			v := &a.Vars[j]
			v.Name = strings.TrimSpace(v.Name)
			if !seen[strings.ToLower(v.Name)] {
				errf("%s: 変数「%s」は vars にありません（2 台目の列は、vars にある変数だけ書けます）", at, v.Name)
			}
			if v.Source() == "" {
				errf("%s の変数「%s」: 値の元を column・template・value のどれか 1 つで書きます", at, v.Name)
			}
			if v.Empty != "" {
				errf("%s の変数「%s」: empty は 1 台目の変数に書きます", at, v.Name)
			}
			for _, p := range v.validate() {
				errf("%s の変数「%s」: %s", at, v.Name, p)
			}
		}
	}
	return out
}

// reservedVar tells a name the template's own words would read.
func reservedVar(name string) bool {
	switch strings.ToLower(name) {
	case "each", "if", "else", "sync":
		return true
	}
	return false
}

// validateLocation gives the faults of where the table is: the first part of
// Validate, without the targets.
func (r *Rule) validateLocation() []string {
	var out []string
	for _, p := range r.Validate(nil) {
		if strings.HasPrefix(p, "rows") || strings.HasPrefix(p, "names") {
			continue
		}
		out = append(out, p)
	}
	return out
}

// validate gives the faults of one field's source and steps (what Validate
// of a column mapping checks for each column).
func (f *Field) validate() []string {
	var out []string
	if len(f.Fallback) > 0 && f.Column == "" {
		out = append(out, "fallback は column と一緒に書きます")
	}
	if len(f.Fallback) > maxFallbacks {
		out = append(out, fmt.Sprintf("fallback は %d 個までです", maxFallbacks))
	}
	if len([]rune(f.Template)) > maxTemplate {
		out = append(out, fmt.Sprintf("template は %d 文字までです", maxTemplate))
	}
	if len(f.Steps) > maxSteps {
		out = append(out, fmt.Sprintf("steps は %d 個までです", maxSteps))
	}
	for k := range f.Steps {
		if err := f.Steps[k].Validate(); err != nil {
			out = append(out, fmt.Sprintf("steps %d: %v", k+1, err))
		}
	}
	return out
}

// VarRow is one device's values from one line of a list.
type VarRow struct {
	Line   int               // the row of the sheet
	Values map[string]string // variable name (as the rule writes it) → value
	// Lines are the variables whose lines are left out (empty with
	// empty: line).
	Lines map[string]bool
	// Empty are the variables that are empty and stop the device.
	Empty map[string]bool
}

// VarLine is one line of the list as a try shows it (PalaTerm's
// FillTestRow): which device of the line, its name, the raw text and the
// value of each variable, whether it continues the device above, and why it
// was left out.
type VarLine struct {
	Line      int               `json:"line"`
	Unit      int               `json:"unit"` // 1 = the first device of the line
	Device    string            `json:"device"`
	RawDevice string            `json:"rawDevice"`
	Cont      bool              `json:"cont"`
	Excluded  string            `json:"excluded"`
	Values    map[string]string `json:"values"`
	Raw       map[string]string `json:"raw"`
}

// VarResult is what a rule read from its table.
type VarResult struct {
	// Devices are the device names in the order the list first gives them.
	Devices []string
	// Rows are each device's rows, by its folded name (DeviceKey).
	Rows map[string][]VarRow
	// Names are the device names as the list writes them, by folded name.
	Names    map[string]string
	Lines    int // the value rows read (each device of a line once)
	Excluded int
	// Missing are the headers the rule names that the table lacks.
	Missing []string
	// Warnings are what could not be read.
	Warnings []string
	// Sample are the first lines as a try shows them.
	Sample []VarLine
}

// DeviceKey folds a device name as every match does: case and the spaces
// around do not count (PalaTerm's FillKey).
func DeviceKey(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// MatchName folds a device name with the differences a rule leaves out, as
// PalaTerm's NormName.
func MatchName(s string, m VarMatch) string {
	s = strings.TrimSpace(s)
	if m.Width {
		s = norm.NFKC.String(s)
	}
	if m.Space {
		s = strings.Join(strings.Fields(s), "")
	}
	s = strings.ToLower(s)
	if m.Dash {
		s = strings.ReplaceAll(s, "_", "-")
	}
	if m.Domain && strings.Trim(s, "0123456789.") != "" && !strings.Contains(s, ":") {
		if i := strings.IndexByte(s, '.'); i > 0 {
			s = s[:i]
		}
	}
	return s
}

// Align gives the devices of the list the names known devices have (the
// book's, the configurations'), when the rule's match finds exactly one;
// rows of list names that come to one device are joined in the list's
// order (PalaTerm's matchFillNames). A name matching none, or two, stays.
func (res *VarResult) Align(known []string, m VarMatch) {
	if !m.Width && !m.Space && !m.Domain && !m.Dash {
		return
	}
	by := map[string]string{}
	for _, k := range known {
		n := MatchName(k, m)
		if prev, dup := by[n]; dup && DeviceKey(prev) != DeviceKey(k) {
			by[n] = ""
		} else if !dup {
			by[n] = k
		}
	}
	rows := map[string][]VarRow{}
	names := map[string]string{}
	var devices []string
	for _, d := range res.Devices {
		name := d
		if to := by[MatchName(d, m)]; to != "" {
			name = to
		}
		k := DeviceKey(name)
		if _, ok := rows[k]; !ok {
			devices = append(devices, name)
			names[k] = name
		}
		rows[k] = append(rows[k], res.Rows[DeviceKey(d)]...)
	}
	for k := range rows {
		sort.SliceStable(rows[k], func(i, j int) bool { return rows[k][i].Line < rows[k][j].Line })
	}
	res.Devices, res.Rows, res.Names = devices, rows, names
}

// condText says why a condition left a row out.
func condText(c VarCond) string {
	what := "列「" + c.Column + "」"
	if c.Var != "" {
		what = "変数「" + c.Var + "」"
	}
	switch c.Op {
	case "empty":
		return what + "が空"
	case "notempty":
		return what + "が空でない"
	case "contains":
		return what + "が「" + c.Value + "」を含む"
	case "notcontains":
		return what + "が「" + c.Value + "」を含まない"
	case "equals":
		return what + "が「" + c.Value + "」と同じ"
	case "notequals":
		return what + "が「" + c.Value + "」と違う"
	}
	return what
}

// ReadVars reads the values of a rule from its table (spec 9.7.2; PalaTerm's
// readFill). A row whose device cell is empty is one more row of the device
// above; one with no value at all is skipped. With more devices on a row, a
// row whose first device cell is empty continues each device of the row
// above; one whose first device is named but a further one is not has no
// such device, and the rows after it do not continue it.
func ReadVars(t *Table, r *VarRule) *VarResult {
	out := &VarResult{Rows: map[string][]VarRow{}, Names: map[string]string{}}
	missing := map[string]bool{}
	read := func(f *Field, row SrcRow) (raw, val string) {
		v, _, miss, _ := t.sourceOf(f, row)
		for _, m := range miss {
			missing[m] = true
		}
		raw = v
		v = strings.TrimSpace(ApplySteps(v, f.Steps))
		if v == "" {
			v = strings.TrimSpace(f.Default)
		}
		return raw, v
	}
	// The fields of each device of a line: the first device's, with a
	// further device's own in their place (its default the first's when it
	// has none).
	type unit struct {
		device Field
		vars   []Field
	}
	units := []unit{{device: r.Device}}
	for i := range r.Vars {
		units[0].vars = append(units[0].vars, r.Vars[i].Field)
	}
	for _, a := range r.Also {
		u := unit{device: a.Device, vars: append([]Field{}, units[0].vars...)}
		for _, o := range a.Vars {
			for i := range r.Vars {
				if strings.EqualFold(r.Vars[i].Name, o.Name) {
					f := o.Field
					if f.Default == "" {
						f.Default = r.Vars[i].Default
					}
					u.vars[i] = f
				}
			}
		}
		units = append(units, u)
	}
	// excluded gives why a device of a row is left out ("" = it is not).
	excluded := func(src SrcRow, vals map[string]string) string {
		for _, c := range r.Exclude {
			v := ""
			if c.Var != "" {
				for k, x := range vals {
					if strings.EqualFold(k, c.Var) {
						v = x
					}
				}
			} else {
				col := t.column(c.Column, 0)
				if col == 0 {
					missing[c.Column] = true
					continue
				}
				v = src.Cells[col]
			}
			if condMet(Cond{Op: c.Op, Value: c.Value}, v) {
				return condText(c)
			}
		}
		return ""
	}
	limit := sampleLines + 4*len(r.Also)
	cur := make([]string, len(units)) // the device each unit continues ("" = none)
	full := false
	for _, src := range t.Rows {
		_, first := read(&units[0].device, src)
		lineCont := first == ""
		for u := range units {
			rawName, name := read(&units[u].device, src)
			vals, raws := map[string]string{}, map[string]string{}
			any := false
			for i := range r.Vars {
				rw, v := read(&units[u].vars[i], src)
				vals[r.Vars[i].Name], raws[r.Vars[i].Name] = v, rw
				any = any || v != ""
			}
			cont := name == ""
			if cont && (!any || cur[u] == "" || cur[u] == excludedMark || (u > 0 && !lineCont)) {
				// A blank line, nothing above to continue (or a device left
				// out, whose rows go with it), or a line without this device.
				if u > 0 && !lineCont {
					cur[u] = ""
				}
				if u == 0 && any && cur[u] == "" {
					out.Warnings = append(out.Warnings, fmt.Sprintf("%d 行目: 機器名が空で、上に続ける機器も無いので読みませんでした", src.Line))
				}
				continue
			}
			why := excluded(src, vals)
			if !cont {
				cur[u] = name
				if why != "" {
					cur[u] = excludedMark // its continuing rows are left out too
				}
			}
			dev := cur[u]
			if dev == "" || dev == excludedMark {
				dev = name
			}
			if len(out.Sample) < limit {
				out.Sample = append(out.Sample, VarLine{Line: src.Line, Unit: u + 1, Device: dev, RawDevice: rawName, Cont: cont, Excluded: why, Values: vals, Raw: raws})
			}
			if why != "" {
				out.Excluded++
				continue
			}
			if out.Lines >= maxVarRows {
				full = true
				break
			}
			out.Lines++
			row := VarRow{Line: src.Line, Values: vals, Lines: map[string]bool{}, Empty: map[string]bool{}}
			for i := range r.Vars {
				v := &r.Vars[i]
				if vals[v.Name] == "" {
					if v.Empty == "line" {
						row.Lines[v.Name] = true
					} else {
						row.Empty[v.Name] = true
					}
				}
			}
			key := DeviceKey(dev)
			if _, ok := out.Rows[key]; !ok {
				out.Devices = append(out.Devices, dev)
				out.Names[key] = dev
			}
			out.Rows[key] = append(out.Rows[key], row)
		}
		if full {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%d 行を超える分は読みませんでした", maxVarRows))
			break
		}
	}
	for m := range missing {
		out.Missing = append(out.Missing, m)
	}
	sort.Strings(out.Missing)
	return out
}

// VarFile is one variable rule file with its faults.
type VarFile struct {
	Path     string
	Rule     *VarRule
	Problems []string
}

// VarFiles reads the variable rules of a folder (vars/ of a case folder), by
// name. Like Files: ".yml" is a fault, names starting "." or "~$" are not
// read.
func VarFiles(fsys fs.FS, dir string) ([]VarFile, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("%s を読めません: %v", dir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var out []VarFile
	seen := map[string]string{}
	for _, name := range names {
		f := VarFile{Path: dir + "/" + name}
		lower := strings.ToLower(name)
		switch {
		case strings.HasPrefix(name, ".") || strings.HasPrefix(name, "~$"):
			continue
		case strings.HasSuffix(lower, ".yml"):
			f.Problems = []string{"拡張子は .yaml にしてください（.yml は読みません）"}
			out = append(out, f)
			continue
		case !strings.HasSuffix(lower, ".yaml"):
			continue
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			f.Problems = []string{fmt.Sprintf("読めません: %v", err)}
			out = append(out, f)
			continue
		}
		r, err := ParseVars(data)
		if err != nil {
			f.Problems = []string{err.Error()}
			out = append(out, f)
			continue
		}
		r.Path = f.Path
		r.Name = name[:len(name)-len(".yaml")]
		f.Rule, f.Problems = r, r.Validate()
		if !varNameRe.MatchString(r.Name) {
			f.Problems = append(f.Problems, "ファイルの名前は、ひな形から {{#each 名前}} で呼べるよう、文字か _ で始め、文字・数字・_・- で付けます")
		}
		if prev, ok := seen[strings.ToLower(r.Name)]; ok {
			f.Problems = append(f.Problems, fmt.Sprintf("%s と名前が大文字と小文字だけ違います", prev))
		}
		seen[strings.ToLower(r.Name)] = f.Path
		out = append(out, f)
	}
	return out, nil
}
