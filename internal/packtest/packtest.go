// Package packtest runs the tests that a vendor pack carries with it
// (tests/import/<name>/input.txt and expect.yaml).
//
// Spec: appendix B 7. The render tests join in M2.
package packtest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/kojio145/configchief/internal/conftree"
	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/schema"
)

// Case is the outcome of one import test.
type Case struct {
	Name     string
	Result   *importer.Result
	Count    importer.Count
	Failures []string // empty: the test passed
}

// value is one expected cell: a text, or a list (compared joined by ",").
type value string

func (v *value) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		if n.Tag != "!!null" {
			*v = value(n.Value)
		}
		return nil
	case yaml.SequenceNode:
		items := make([]string, len(n.Content))
		for i, c := range n.Content {
			if c.Kind != yaml.ScalarNode {
				return fmt.Errorf("%d 行目: リストの中は 1 つの値で書いてください", c.Line)
			}
			items[i] = c.Value
		}
		*v = value(strings.Join(items, ","))
		return nil
	}
	return fmt.Errorf("%d 行目: 値は文字列かリストで書いてください", n.Line)
}

type expect struct {
	Tables     map[string][]map[string]value `yaml:"tables"`
	Unknown    []string                      `yaml:"unknown"`
	IgnoredMin int                           `yaml:"ignored_min"`
}

// RunImport runs every test under tests/import of the pack's folder. A pack
// without tests gives no cases.
func RunImport(p *pack.Pack, sc *schema.Set, fsys fs.FS) ([]Case, error) {
	inputs, _ := fs.Glob(fsys, "tests/import/*/input.txt")
	sort.Strings(inputs)
	var out []Case
	for _, in := range inputs {
		dir := path.Dir(in)
		name := path.Base(dir)
		text, err := fs.ReadFile(fsys, in)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", in, err)
		}
		var ex expect
		data, err := fs.ReadFile(fsys, dir+"/expect.yaml")
		if err != nil {
			return nil, fmt.Errorf("%s/expect.yaml: 読めません（%w）", dir, err)
		}
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(&ex); err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%s/expect.yaml: %w", dir, err)
		}
		res, err := importer.Import(p, sc, name+".txt", string(text))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", in, err)
		}
		c := Case{Name: name, Result: res, Count: res.Count()}
		c.Failures = compare(res, &ex)
		out = append(out, c)
	}
	return out, nil
}

func compare(res *importer.Result, ex *expect) []string {
	var fails []string
	failf := func(format string, a ...any) { fails = append(fails, fmt.Sprintf(format, a...)) }

	// Every expected row must be there with the expected values. Rows and
	// columns the test does not name are not looked at.
	tables := make([]string, 0, len(ex.Tables))
	for t := range ex.Tables {
		tables = append(tables, t)
	}
	sort.Strings(tables)
	for _, t := range tables {
		got := res.Data.Tables[t]
		for i, want := range ex.Tables[t] {
			if msg := findRow(got, want); msg != "" {
				failf("%s の %d 件目: %s", t, i+1, msg)
			}
		}
	}

	// The unknown lines must be exactly the listed ones.
	left := map[string]int{}
	for _, u := range ex.Unknown {
		left[conftree.Normalize(u)]++
	}
	for _, l := range res.Lines {
		if l.State != importer.Unknown {
			continue
		}
		norm := conftree.Normalize(l.Text)
		if left[norm] > 0 {
			left[norm]--
			continue
		}
		reason := l.Reason
		if reason == "" {
			reason = "当たる規則がありません"
		}
		failf("%d 行目が「読めなかった」になりました: %s（%s）", l.Line, strings.TrimSpace(l.Text), reason)
	}
	var missing []string
	for u, n := range left {
		for ; n > 0; n-- {
			missing = append(missing, u)
		}
	}
	sort.Strings(missing)
	for _, u := range missing {
		failf("「読めなかった」のはずの行が、そうなっていません: %s", u)
	}

	c := res.Count()
	if c.Ignored < ex.IgnoredMin {
		failf("「無視」が %d 行で、下限の %d 行に足りません", c.Ignored, ex.IgnoredMin)
	}
	return fails
}

// findRow looks for a row that has every expected value. It gives "" if there
// is one, or else what is wrong with the nearest row.
func findRow(t *model.Table, want map[string]value) string {
	cols := make([]string, 0, len(want))
	for c := range want {
		cols = append(cols, c)
	}
	sort.Strings(cols)
	if t == nil || len(t.Rows) == 0 {
		return "表に行がありません"
	}
	best, bestHits := -1, -1
	var bestDiff []string
	for i, r := range t.Rows {
		hits := 0
		var diff []string
		for _, c := range cols {
			if got := r.Get(c); got == string(want[c]) {
				hits++
			} else {
				diff = append(diff, fmt.Sprintf("%s は「%s」（期待は「%s」）", c, got, want[c]))
			}
		}
		if len(diff) == 0 {
			return ""
		}
		if hits > bestHits {
			best, bestHits, bestDiff = i, hits, diff
		}
	}
	return fmt.Sprintf("合う行がありません。いちばん近い行（%s）では、%s", t.Rows[best].KeyText(), strings.Join(bestDiff, "、"))
}

// Report writes the outcome of a case as text: the counts, the lines that
// stayed unknown, the warnings and the failures.
func Report(c Case) string {
	var sb strings.Builder
	n := c.Count
	lines := n.Total - n.Separators
	fmt.Fprintf(&sb, "試験 %s（機器 %s）\n", c.Name, c.Result.Device)
	fmt.Fprintf(&sb, "  全 %d 行（うち区切り %d 行）: 取り込み済み %d・無視 %d・読めなかった %d\n", n.Total, n.Separators, n.Taken, n.Ignored, n.Unknown)
	if n.Outside > 0 {
		fmt.Fprintf(&sb, "  区切りのうち、端末ログの設定の外: %d 行\n", n.Outside)
	}
	fmt.Fprintf(&sb, "  取り込み済み＋無視の割合: %.1f%%（区切りを除く %d 行のうち %d 行）\n", n.Covered()*100, lines, n.Taken+n.Ignored)
	if n.Total > 0 {
		fmt.Fprintf(&sb, "  区切りを含めた割合: %.1f%%\n", float64(n.Taken+n.Ignored+n.Separators)*100/float64(n.Total))
	}
	if n.Unknown > 0 {
		sb.WriteString("  読めなかった行:\n")
		for _, l := range c.Result.Lines {
			if l.State != importer.Unknown {
				continue
			}
			reason := l.Reason
			if reason == "" {
				reason = "当たる規則がありません"
			}
			fmt.Fprintf(&sb, "    %4d: %s    ← %s\n", l.Line, strings.TrimRight(l.Text, " \t"), reason)
		}
	}
	for _, s := range c.Result.Notes {
		sb.WriteString("  お知らせ: " + s + "\n")
	}
	if len(c.Result.Warnings) > 0 {
		sb.WriteString("  取り込みの警告:\n")
		for _, w := range c.Result.Warnings {
			sb.WriteString("    " + w + "\n")
		}
	}
	if len(c.Failures) == 0 {
		sb.WriteString("  結果: 合格\n")
	} else {
		fmt.Fprintf(&sb, "  結果: 不合格（%d 件）\n", len(c.Failures))
		for _, f := range c.Failures {
			sb.WriteString("    " + f + "\n")
		}
	}
	return sb.String()
}

// Lines writes every line with its state, for looking into a pack.
func Lines(c Case) string {
	mark := map[string]string{importer.Taken: "取込", importer.Ignored: "無視", importer.Unknown: "読めなかった"}
	var sb strings.Builder
	for _, l := range c.Result.Lines {
		state := mark[l.State]
		if l.Separator {
			state = "区切"
		}
		if l.Outside {
			state = "外"
		}
		fmt.Fprintf(&sb, "%4d %-6s %-28s %s\n", l.Line, state, l.RuleID, strings.TrimRight(l.Text, " \t"))
	}
	return sb.String()
}
