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

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/render"
	"github.com/kojio145/configchief/internal/schema"
)

// RenderCase is the outcome of one generation test
// (tests/render/<name>/data.yaml and expect.txt).
type RenderCase struct {
	Name     string
	Device   string
	Output   *render.Output
	Findings []check.Finding // of the generation and of the way back
	Failures []string        // empty: the test passed
}

// renderData is data.yaml: the design data of the devices, and what the test
// expects besides the text.
type renderData struct {
	// Device is the device to generate; the first row of devices if empty.
	Device string                        `yaml:"device"`
	Tables map[string][]map[string]value `yaml:"tables"`
	// Extra holds extra configurations by file name ("_all.j2" ...).
	Extra map[string]string `yaml:"extra"`
	// Findings lists the rules that have to be told, each once however many
	// findings it has. Without it, no finding is expected.
	Findings []string `yaml:"findings"`
}

// RunRender runs every test under tests/render of the pack's folder: the
// configuration of the device is generated and compared with expect.txt, and
// read back with the same pack (appendix B 5.6 and 7).
func RunRender(p *pack.Pack, sc *schema.Set, fsys fs.FS) ([]RenderCase, error) {
	files, _ := fs.Glob(fsys, "tests/render/*/data.yaml")
	sort.Strings(files)
	var out []RenderCase
	for _, f := range files {
		dir := path.Dir(f)
		raw, err := fs.ReadFile(fsys, f)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		var d renderData
		dec := yaml.NewDecoder(bytes.NewReader(raw))
		dec.KnownFields(true)
		if err := dec.Decode(&d); err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		want, err := fs.ReadFile(fsys, dir+"/expect.txt")
		if err != nil {
			return nil, fmt.Errorf("%s/expect.txt: 読めません（%w）", dir, err)
		}
		ds, device, err := dataset(sc, &d)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		var extras []render.ExtraFile
		for name, text := range d.Extra {
			extras = append(extras, render.ExtraFile{Name: name, Text: text})
		}
		r := render.New(p, sc, extras)
		o, findings := r.Render(ds, device, map[string]any{"name": "test"})
		findings = append(findings, r.RoundTrip(o)...)
		c := RenderCase{Name: path.Base(dir), Device: device, Output: o, Findings: findings}
		c.Failures = compareRender(o, findings, strings.ReplaceAll(string(want), "\r\n", "\n"), d.Findings)
		out = append(out, c)
	}
	return out, nil
}

// dataset makes the design data of a test. A row without "device" belongs to
// the device of the test.
func dataset(sc *schema.Set, d *renderData) (*model.Dataset, string, error) {
	device := d.Device
	if device == "" {
		if rows := d.Tables["devices"]; len(rows) > 0 {
			device = string(rows[0]["hostname"])
		}
	}
	if device == "" {
		return nil, "", errors.New("devices に行がありません")
	}
	ds := model.NewDataset("test")
	o := model.Origin{Kind: "book"}
	for name := range d.Tables {
		if sc.Table(name) == nil {
			return nil, "", fmt.Errorf("表 %s はありません", name)
		}
	}
	for _, t := range sc.Tables {
		for i, vals := range d.Tables[t.Name] {
			norm := map[string][]string{}
			for name, raw := range vals {
				c := t.Column(name)
				if c == nil {
					return nil, "", fmt.Errorf("%s の %d 件目: 列 %s はありません", t.Name, i+1, name)
				}
				items := []string{string(raw)}
				if c.List {
					items = strings.Split(string(raw), ",")
				}
				for _, item := range items {
					v, err := c.Normalize(item)
					if err != nil {
						return nil, "", fmt.Errorf("%s の %d 件目の %s: %w", t.Name, i+1, name, err)
					}
					if v != "" {
						norm[name] = append(norm[name], v)
					}
				}
			}
			if dc := t.DeviceColumn(); dc == "device" && len(norm[dc]) == 0 {
				norm[dc] = []string{device}
			}
			var key []model.KV
			for _, k := range t.Key {
				key = append(key, model.KV{Name: k, Value: strings.Join(norm[k], ",")})
			}
			row := ds.Open(t.Name, key, o)
			for _, c := range t.Columns {
				if t.IsKey(c.Name) || len(norm[c.Name]) == 0 {
					continue
				}
				if c.List {
					for _, v := range norm[c.Name] {
						row.Append(c.Name, v, o)
					}
				} else {
					row.Set(c.Name, norm[c.Name][0], o)
				}
			}
		}
	}
	return ds, device, nil
}

func compareRender(o *render.Output, findings []check.Finding, want string, rules []string) []string {
	var fails []string
	if o.Text != want {
		got, exp := strings.Split(o.Text, "\n"), strings.Split(want, "\n")
		n := 0
		for n < len(got) && n < len(exp) && got[n] == exp[n] {
			n++
		}
		line := func(l []string) string {
			if n < len(l) {
				return l[n]
			}
			return "（終わり）"
		}
		fails = append(fails, fmt.Sprintf("文面が違います。%d 行目: 「%s」（期待は「%s」）", n+1, line(got), line(exp)))
	}
	told := map[string]bool{}
	for _, f := range findings {
		told[f.RuleID] = true
	}
	wanted := map[string]bool{}
	for _, r := range rules {
		wanted[r] = true
		if !told[r] {
			fails = append(fails, fmt.Sprintf("%s の指摘が出るはずですが、出ていません", r))
		}
	}
	for _, f := range findings {
		if !wanted[f.RuleID] {
			fails = append(fails, fmt.Sprintf("思っていない指摘: %s %s %s", f.RuleID, refText(f.Ref), f.Message))
		}
	}
	return fails
}

func refText(r model.Ref) string {
	var parts []string
	for _, s := range []string{r.Table, r.Key, r.Column} {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " / ")
}

// ReportRender writes the outcome of a generation test as text.
func ReportRender(c RenderCase) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "生成の試験 %s（機器 %s）\n", c.Name, c.Device)
	fmt.Fprintf(&sb, "  %d 行・指摘 %d 件\n", len(c.Output.Lines), len(c.Findings))
	for _, f := range c.Findings {
		fmt.Fprintf(&sb, "    %s %s %s\n", f.RuleID, refText(f.Ref), f.Message)
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
