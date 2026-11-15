package listimport

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"regexp/syntax"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/kojio145/configchief/internal/schema"
)

// Rule is one column mapping (mappings/<name>.yaml, spec 6.5.1): where the
// table of a customer's list is, and which rows of the design data one line
// of it gives.
type Rule struct {
	Version int `yaml:"version" json:"version"`
	// File is the name of the list under input/customer; it may hold "*".
	File string `yaml:"file" json:"file"`

	// Where the table is: the sheet (by name, or failing that its number,
	// 1-based; 0 = the first), the header row and the first value row
	// (1-based; 0 = the row after the header), and the column the table
	// starts at ("" = the header row's first filled cell, else a letter).
	Sheet      string `yaml:"sheet,omitempty" json:"sheet"`
	SheetIndex int    `yaml:"sheet_index,omitempty" json:"sheetIndex"`
	HeaderRow  int    `yaml:"header_row" json:"headerRow"`
	DataRow    int    `yaml:"data_row,omitempty" json:"dataRow"`
	StartCol   string `yaml:"start_col,omitempty" json:"startCol"`
	// MergeOnlyFirst leaves a merged area's value in its top-left cell only;
	// by default every cell of the area reads the value (a site written once
	// for five rows applies to all five).
	MergeOnlyFirst bool `yaml:"merge_only_first,omitempty" json:"mergeOnlyFirst"`

	// Names tells whose names the list holds: "new" (the default) or "old",
	// for which the mapping of the book is put on before comparing.
	Names string `yaml:"names,omitempty" json:"names"`

	Exclude []Cond   `yaml:"exclude,omitempty" json:"exclude"`
	Rows    []Target `yaml:"rows" json:"rows"`

	// Path is the file the rule was read from, from the case folder.
	Path string `yaml:"-" json:"path"`
}

// Old tells that the list holds the names of the old devices.
func (r *Rule) Old() bool { return r.Names == "old" }

// Target is one row of the design data a line of the list gives.
type Target struct {
	Table string `yaml:"table" json:"table"`
	// Label names the target for people ("WAN 側"); it is not used otherwise.
	Label   string  `yaml:"label,omitempty" json:"label"`
	Columns []Field `yaml:"columns" json:"columns"`
}

// Field fills one column of a target. Its value comes from one of: a column
// of the list (by its header; Fallback are further columns read in turn
// while the value is empty), a template (text with {header} placeholders),
// or a fixed value.
type Field struct {
	To       string   `yaml:"to" json:"to"`
	Column   string   `yaml:"column,omitempty" json:"column"`
	Fallback []string `yaml:"fallback,omitempty" json:"fallback"`
	Template string   `yaml:"template,omitempty" json:"template"`
	Value    string   `yaml:"value,omitempty" json:"value"`
	Steps    []Step   `yaml:"steps,omitempty" json:"steps"`
	// Default fills the column when the value is empty after the steps.
	Default string `yaml:"default,omitempty" json:"default"`
}

// Source is where the value of the field comes from: "column", "template"
// or "value"; "" if the field names none or more than one.
func (f *Field) Source() string {
	n, src := 0, ""
	if f.Column != "" {
		n, src = n+1, "column"
	}
	if f.Template != "" {
		n, src = n+1, "template"
	}
	if f.Value != "" {
		n, src = n+1, "value"
	}
	if n != 1 {
		return ""
	}
	return src
}

// Step is one step of a field's processing, applied in order.
//
//	trim     前後の空白を除く
//	split    区切って残す: Sep, Pos first|last, Keep left|right; no Sep in
//	         the value: unchanged, or empty with Miss "empty"
//	replace  文字を置き換える・削る: From → To
//	regex    正規表現: Pattern → To ($1 …)
//	map      置き換え表: Rows; Miss keep|empty for a value no row matches
//	lower / upper  小文字・大文字にする
//	width    全角を半角にする (NFKC)
//	cidr     アドレスとマスクをプレフィックス長の形にする
type Step struct {
	Type    string   `yaml:"type" json:"type"`
	Sep     string   `yaml:"sep,omitempty" json:"sep"`
	Pos     string   `yaml:"pos,omitempty" json:"pos"`
	Keep    string   `yaml:"keep,omitempty" json:"keep"`
	From    string   `yaml:"from,omitempty" json:"from"`
	To      string   `yaml:"to,omitempty" json:"to"`
	Pattern string   `yaml:"pattern,omitempty" json:"pattern"`
	Rows    []MapRow `yaml:"rows,omitempty" json:"rows"`
	Miss    string   `yaml:"miss,omitempty" json:"miss"`
}

// MapRow is one line of a 置き換え表: Match "equals" or "contains" (both
// ignore case).
type MapRow struct {
	Match string `yaml:"match,omitempty" json:"match"`
	From  string `yaml:"from" json:"from"`
	To    string `yaml:"to" json:"to"`
}

// Cond leaves a line out (除外条件) by the raw text of a column. Op: empty,
// notempty, contains, notcontains, equals, notequals (ignoring case and
// surrounding spaces).
type Cond struct {
	Column string `yaml:"column" json:"column"`
	Op     string `yaml:"op" json:"op"`
	Value  string `yaml:"value,omitempty" json:"value"`
}

// StepTypes are the step types a rule may use.
var StepTypes = []string{"trim", "width", "lower", "upper", "split", "replace", "regex", "map", "cidr"}

// CondOps are the operators an exclusion condition may use.
var CondOps = []string{"empty", "notempty", "contains", "notcontains", "equals", "notequals"}

var reColLetters = regexp.MustCompile(`^[A-Za-z]{1,3}$`)

// ColumnNumber reads a column letter (A = 1, Z = 26, AA = 27); 0 if it is
// not one.
func ColumnNumber(s string) int {
	s = strings.TrimSpace(s)
	if !reColLetters.MatchString(s) {
		return 0
	}
	n := 0
	for _, r := range strings.ToUpper(s) {
		n = n*26 + int(r-'A'+1)
	}
	return n
}

// ColumnLetter is ColumnNumber's inverse ("" for n < 1).
func ColumnLetter(n int) string {
	s := ""
	for n > 0 {
		n--
		s = string(rune('A'+n%26)) + s
		n /= 26
	}
	return s
}

// The size a rule may have: generous for any real list, small enough that a
// rule file someone hands over cannot keep a reading busy for hours or
// compile a huge regexp.
const (
	maxTargets     = 50
	maxSteps       = 30
	maxMapRows     = 5000
	maxTemplate    = 1000
	maxFallbacks   = 10
	maxExcludes    = 100
	maxRegexpInsts = 10000
)

func tableOf(sc *schema.Set, name string) *schema.Table {
	name = strings.TrimSpace(name)
	for _, t := range sc.Tables {
		if t.Name == name || t.Title.JA == name || t.Title.EN == name {
			return t
		}
	}
	return nil
}

func columnOf(t *schema.Table, name string) *schema.Column {
	name = strings.TrimSpace(name)
	for _, c := range t.Columns {
		if c.Name == name || c.Title.JA == name || c.Title.EN == name {
			return c
		}
	}
	return nil
}

// Validate checks a rule and gives every fault of it. The names of tables
// and columns written as the book shows them become the internal ones.
func (r *Rule) Validate(sc *schema.Set) []string {
	var out []string
	errf := func(format string, a ...any) { out = append(out, fmt.Sprintf(format, a...)) }
	if r.Version != 1 {
		errf("version は 1 と書きます")
	}
	r.File = strings.TrimSpace(r.File)
	switch {
	case r.File == "":
		errf("file（input/customer の中のファイル名）がありません")
	case strings.ContainsAny(r.File, `/\`):
		errf("file「%s」: フォルダは書けません（input/customer の直下のファイル名だけ）", r.File)
	}
	if r.HeaderRow < 1 {
		errf("header_row（見出しの行）は 1 以上にします")
	}
	if r.DataRow != 0 && r.DataRow <= r.HeaderRow {
		errf("data_row（値の最初の行）は、見出しの行より後にします")
	}
	if r.SheetIndex < 0 {
		errf("sheet_index は 1 以上にします")
	}
	if c := strings.TrimSpace(r.StartCol); c != "" && ColumnNumber(c) == 0 {
		errf("start_col「%s」: A、D、AB のような列の記号で書きます", r.StartCol)
	}
	if r.Names != "" && r.Names != "new" && r.Names != "old" {
		errf("names「%s」: new か old と書きます", r.Names)
	}
	if len(r.Exclude) > maxExcludes {
		errf("exclude は %d 件までです", maxExcludes)
	}
	for i := range r.Exclude {
		c := &r.Exclude[i]
		if strings.TrimSpace(c.Column) == "" {
			errf("exclude %d: column（一覧の見出し）がありません", i+1)
		}
		ok := false
		for _, op := range CondOps {
			ok = ok || c.Op == op
		}
		if !ok {
			errf("exclude %d: op「%s」は使えません（%s）", i+1, c.Op, strings.Join(CondOps, "、"))
		}
	}
	if len(r.Rows) == 0 {
		errf("rows（一覧の 1 行から作る行）がありません")
	}
	if len(r.Rows) > maxTargets {
		errf("rows は %d 個までです", maxTargets)
	}
	for i := range r.Rows {
		tg := &r.Rows[i]
		where := fmt.Sprintf("rows %d", i+1)
		t := tableOf(sc, tg.Table)
		if t == nil {
			errf("%s: 「%s」という表はありません", where, tg.Table)
			continue
		}
		tg.Table = t.Name
		where += "（" + t.Title.JA + "）"
		if len(tg.Columns) == 0 {
			errf("%s: columns がありません", where)
		}
		seen := map[string]bool{}
		for j := range tg.Columns {
			f := &tg.Columns[j]
			c := columnOf(t, f.To)
			if c == nil {
				errf("%s: 表「%s」に「%s」という列はありません", where, t.Title.JA, f.To)
				continue
			}
			f.To = c.Name
			at := where + " の列「" + c.Title.JA + "」"
			if seen[c.Name] {
				errf("%s: 2 回書かれています", at)
			}
			seen[c.Name] = true
			if f.Source() == "" {
				errf("%s: 値の元を column・template・value のどれか 1 つで書きます", at)
			}
			if len(f.Fallback) > 0 && f.Column == "" {
				errf("%s: fallback は column と一緒に書きます", at)
			}
			if len(f.Fallback) > maxFallbacks {
				errf("%s: fallback は %d 個までです", at, maxFallbacks)
			}
			if len([]rune(f.Template)) > maxTemplate {
				errf("%s: template は %d 文字までです", at, maxTemplate)
			}
			if len(f.Steps) > maxSteps {
				errf("%s: steps は %d 個までです", at, maxSteps)
			}
			for k := range f.Steps {
				if err := f.Steps[k].Validate(); err != nil {
					errf("%s の steps %d: %v", at, k+1, err)
				}
			}
		}
		for _, k := range t.Key {
			if !seen[k] {
				errf("%s: キーの列「%s」がありません（一覧に無ければ value で決まった値を書きます）", where, t.Column(k).Title.JA)
			}
		}
	}
	return out
}

// Validate checks one step and fills in what it leaves out.
func (s *Step) Validate() error {
	switch s.Type {
	case "trim", "lower", "upper", "width", "cidr":
	case "split":
		if s.Sep == "" {
			return errors.New("split には sep（区切り文字）が要ります")
		}
		if s.Pos != "first" && s.Pos != "last" {
			s.Pos = "first"
		}
		if s.Keep != "left" && s.Keep != "right" {
			s.Keep = "right"
		}
		if s.Miss != "empty" {
			s.Miss = ""
		}
	case "replace":
		if s.From == "" {
			return errors.New("replace には from（置き換える文字）が要ります")
		}
	case "regex":
		if s.Pattern == "" {
			return errors.New("regex には pattern（正規表現）が要ります")
		}
		if len(s.Pattern) > 500 {
			return errors.New("正規表現が長すぎます（500 文字まで）")
		}
		if _, err := regexp.Compile(s.Pattern); err != nil {
			return fmt.Errorf("正規表現の書き方が正しくありません（%v）", err)
		}
		// Short but huge once compiled ((?:a{1000}){1000}): refused.
		if re, err := syntax.Parse(s.Pattern, syntax.Perl); err == nil {
			if prog, err := syntax.Compile(re.Simplify()); err != nil || len(prog.Inst) > maxRegexpInsts {
				return errors.New("正規表現が複雑すぎます")
			}
		}
	case "map":
		if len(s.Rows) > maxMapRows {
			return fmt.Errorf("置き換え表は %d 行までです", maxMapRows)
		}
		if s.Miss != "keep" && s.Miss != "empty" {
			s.Miss = "keep"
		}
		for i := range s.Rows {
			if s.Rows[i].Match != "contains" {
				s.Rows[i].Match = "equals"
			}
		}
	default:
		return fmt.Errorf("加工「%s」は使えません（%s）", s.Type, strings.Join(StepTypes, "、"))
	}
	return nil
}

// Parse reads the text of a rule file; an unknown word in it is a fault.
func Parse(data []byte) (*Rule, error) {
	r := &Rule{}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(r); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return r, nil
}

// Marshal writes a rule as the text of a rule file.
func Marshal(r *Rule) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// File is one rule file of a folder with what is wrong with it. Rule is nil
// for a file that could not be read as a rule at all.
type File struct {
	Path     string
	Rule     *Rule
	Problems []string
}

// Files reads the rule files of a folder (mappings/ of a case folder), by
// name, each with its faults. dir is how the folder is named in the paths
// and the messages. A file that is no rule file by its name is left out; a
// ".yml" is a fault (nobody would see that it is not read).
func Files(fsys fs.FS, dir string, sc *schema.Set) ([]File, error) {
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
	var out []File
	for _, name := range names {
		f := File{Path: dir + "/" + name}
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
		r, err := Parse(data)
		if err != nil {
			f.Problems = []string{err.Error()}
			out = append(out, f)
			continue
		}
		r.Path = f.Path
		f.Rule, f.Problems = r, r.Validate(sc)
		out = append(out, f)
	}
	return out, nil
}

// Load reads the rule files of a folder. Every fault of every file is given
// together: with one fault no list is read (spec 6.5.1).
func Load(fsys fs.FS, dir string, sc *schema.Set) ([]*Rule, []string) {
	files, err := Files(fsys, dir, sc)
	if err != nil {
		return nil, []string{err.Error()}
	}
	var rules []*Rule
	var problems []string
	for _, f := range files {
		for _, p := range f.Problems {
			problems = append(problems, f.Path+": "+p)
		}
		if f.Rule != nil {
			rules = append(rules, f.Rule)
		}
	}
	if len(problems) > 0 {
		return nil, problems
	}
	return rules, nil
}
