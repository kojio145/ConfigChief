package ops

import (
	"fmt"
	"os"
	"strings"

	"github.com/kojio145/configchief/internal/caserule"
	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/render"
	"github.com/kojio145/configchief/internal/tmpl"
)

const noCaseRules = "rules/*.yaml に、この種類のルールがありません"

// The checks the rules of a case go under.
var caseChecks = []string{"G-610", "G-611", "G-612"}

// caseRules reads the rules of the case. A fault in a file stops the
// generation and the checks: a rule that is passed over without a word would
// read as "nothing found" (appendix C 4.1).
func (e *Env) caseRules() ([]*caserule.Rule, []InputFile, error) {
	entries, err := os.ReadDir(e.Proj.Path("rules"))
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("rules を読めません: %v", err)
	}
	var rules []*caserule.Rule
	var inputs []InputFile
	var problems []string
	eng := tmpl.New(tmpl.Options{})
	for _, ent := range entries {
		name := ent.Name()
		if ent.IsDir() || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "~$") {
			continue
		}
		if strings.HasSuffix(strings.ToLower(name), ".yml") {
			problems = append(problems, fmt.Sprintf("rules/%s: ルールのファイルの名前は .yaml で終わらせます（.yml は読みません）", name))
			continue
		}
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		data, err := os.ReadFile(e.Proj.Path("rules/" + name))
		if err != nil {
			return nil, nil, fmt.Errorf("rules/%s を読めません: %v", name, err)
		}
		rs, ps := caserule.Parse("rules/"+name, []byte(project.Decode(data)), e.Schema, eng)
		rules = append(rules, rs...)
		problems = append(problems, ps...)
		inputs = append(inputs, InputFile{Path: "rules/" + name, SHA256: sha(data)})
	}
	problems = append(problems, caserule.Doubled(rules)...)
	if len(problems) > 0 {
		return nil, nil, fmt.Errorf("案件のルール（rules/）に書き方の誤りがあるため、始めていません。直してから、もう一度実行してください。\n- %s",
			strings.Join(problems, "\n- "))
	}
	return rules, inputs, nil
}

// caseRun runs the rules of the case during a generation and keeps count of
// what they looked at: each rule under its own ID, and each kind under the
// check its findings go under.
type caseRun struct {
	e       *Env
	rules   []*caserule.Rule
	targets map[string][]string // rule → the devices it is for, in the order of the book
	has     map[string]bool     // the checks that have a rule
	tally   *check.Tally
}

func (e *Env) startCaseRules(rules []*caserule.Rule, ds *model.Dataset, devRows map[string]*model.Row, out *GenerateOutcome) *caseRun {
	c := &caseRun{e: e, rules: rules, targets: map[string][]string{}, has: map[string]bool{}, tally: out.Tally}
	vendor := func(host string) string { return devRows[host].Get("vendor") }
	for _, r := range rules {
		c.has[r.CheckID()] = true
		c.tally.Look(r.ID, "", 0)
		c.targets[r.ID] = r.Targets(ds, vendor)
		if len(c.targets[r.ID]) == 0 {
			out.Notes = append(out.Notes, fmt.Sprintf("案件のルール %s（%s）の devices「%s」は、どの機器にも当たりません",
				r.ID, r.File, strings.Join(r.Devices, ", ")))
		}
	}
	for _, id := range caseChecks {
		if c.has[id] {
			c.tally.Look(id, "", 0)
		} else {
			c.tally.Skip(id, "", noCaseRules)
		}
	}
	return c
}

// skip notes that the rules for a device did not run: all of them, or only
// those that look at the generated text.
func (c *caseRun) skip(host, why string, linesOnly bool) {
	for _, r := range c.rules {
		if (r.OfLines() || !linesOnly) && has(c.targets[r.ID], host) {
			c.tally.Skip(r.ID, host, why)
		}
	}
}

// lines runs the rules of lines on the generated text of a device.
func (c *caseRun) lines(host string, o *render.Output, p *pack.Pack) []check.Finding {
	if o.Failed {
		for _, id := range caseChecks[:2] {
			c.tally.Skip(id, host, notWhole)
		}
		c.skip(host, notWhole, true)
		return nil
	}
	for _, id := range caseChecks[:2] {
		if c.has[id] {
			c.tally.Look(id, host, 1)
		}
	}
	text := caserule.Text{
		Shown: o.Shown,
		Comment: func(body string) bool {
			for _, pre := range p.Options.CommentPrefixes {
				if strings.HasPrefix(body, pre) {
					return true
				}
			}
			return false
		},
	}
	for _, l := range o.Lines {
		text.Lines = append(text.Lines, l.Text)
	}
	var findings []check.Finding
	for _, r := range c.rules {
		if !r.OfLines() || !has(c.targets[r.ID], host) {
			continue
		}
		c.tally.Look(r.ID, host, 1)
		findings = append(findings, r.Lines(host, text)...)
	}
	return findings
}

// rows runs the rules of rows on the design data. The devices whose values
// are wrong are left out; a generation of one device looks at all the others
// too, as the checks of the values do.
func (c *caseRun) rows(ds *model.Dataset, blocked map[string][]string, devRows map[string]*model.Row) []check.Finding {
	engines := map[string]*tmpl.Engine{}
	data := &caserule.Data{Schema: c.e.Schema, All: ds, Project: c.e.projectVars(),
		Engine: func(host string) *tmpl.Engine {
			id := devRows[host].Get("vendor")
			if engines[id] == nil {
				if p := c.e.Packs[id]; p != nil {
					engines[id] = p.Engine()
				} else {
					engines[id] = tmpl.New(tmpl.Options{})
				}
			}
			return engines[id]
		}}
	var findings []check.Finding
	for _, r := range c.rules {
		if r.OfLines() {
			continue
		}
		var hosts []string
		for _, h := range c.targets[r.ID] {
			if len(blocked[h]) == 0 {
				hosts = append(hosts, h)
			}
		}
		findings = append(findings, r.Rows(data, hosts, func(host string, n int) {
			c.tally.Look(r.ID, host, n)
			c.tally.Look("G-612", host, n)
		})...)
	}
	return findings
}
