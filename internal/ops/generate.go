package ops

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/caserule"
	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/render"
	"github.com/kojio145/configchief/internal/tmpl"
)

// What the generation did for a device.
const (
	GenWritten = "written" // the configuration is in configs/
	GenFailed  = "failed"  // the text is not whole: what there is of it is in configs/_failed/
	GenBlocked = "blocked" // not generated: a value is wrong (spec 9.1 step 2)
	GenChecked = "checked" // checked only: nothing was written
	GenSkipped = "skipped" // another device was asked for
)

// GenOptions says what Generate does.
type GenOptions struct {
	// Device, if set, is the one device to generate. The checks of the values
	// still look at every device.
	Device string
	// CheckOnly runs the checks and writes the check results, but no
	// configuration and no snapshot (the command "check").
	CheckOnly bool
	// Dry writes nothing at all.
	Dry bool
}

// DeviceRun is what the generation did for one device.
type DeviceRun struct {
	Host  string
	Pack  string
	State string
	File  string // the configuration, as a path from the case folder ("" = none)
	Why   string // why there is no configuration, or why it is in _failed
	// Output is the generated text with where its lines came from; nil for a
	// device that was not generated.
	Output *render.Output

	bytes []byte
	name  string
}

// Judged is a finding with what the case says about it.
type Judged struct {
	check.Finding
	// Approval is the approval that covers the finding; nil = none.
	Approval *project.Approval
}

// InputFile is a file the generation read, with its fingerprint.
type InputFile struct {
	Path   string
	SHA256 string
}

// GenerateOutcome is what Generate did.
type GenerateOutcome struct {
	Options GenOptions
	Dir     string // out/<time>
	Devices []DeviceRun
	Items   []Judged
	Tally   *check.Tally
	// BookNotes is what the book says that is no finding of a check (a sheet
	// that was not read, a heading written twice). It cannot be approved.
	BookNotes []string
	Notes     []string
	Inputs    []InputFile
	// CaseRules is the rules of the case that were read (rules/*.yaml).
	CaseRules []*caserule.Rule
	Approvals []project.Approval
	// Hits counts the findings each approval covers; Refused tells why an
	// approval does not count for a finding it names.
	Hits    map[string]int
	Refused map[string]string
	// Disabled is the checks the case switched off.
	Disabled map[string]bool
	// Unverified is the mark that was put ("" = none).
	Unverified string
	Report     Files
	Changes    *Changes

	hosts []string
	data  *model.Dataset
}

// Count is the findings by what is left to do about them.
type Count struct {
	Errors, Warnings int // not approved
	Approved         int
	Info             int
}

// Count counts the findings.
func (o *GenerateOutcome) Count() Count {
	var c Count
	for _, f := range o.Items {
		switch {
		case f.Severity == check.Info:
			c.Info++
		case f.Approval != nil:
			c.Approved++
		case f.Severity == check.Error:
			c.Errors++
		default:
			c.Warnings++
		}
	}
	return c
}

// Findings counts what a person has to look at: the errors and warnings
// that are not approved, and what the book says.
func (o *GenerateOutcome) Findings() int {
	c := o.Count()
	return c.Errors + c.Warnings + len(o.BookNotes)
}

func sha(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// extras reads the extra configurations of the case (spec 9.4).
func (e *Env) extras() ([]render.ExtraFile, []InputFile, error) {
	entries, err := os.ReadDir(e.Proj.Path("extra"))
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("extra を読めません: %v", err)
	}
	var files []render.ExtraFile
	var inputs []InputFile
	for _, ent := range entries {
		name := ent.Name()
		if !extraFile(ent) {
			continue
		}
		data, err := os.ReadFile(e.Proj.Path("extra/" + name))
		if err != nil {
			return nil, nil, fmt.Errorf("extra/%s を読めません: %v", name, err)
		}
		files = append(files, render.ExtraFile{Name: name, Text: project.Decode(data)})
		inputs = append(inputs, InputFile{Path: "extra/" + name, SHA256: sha(data)})
	}
	return files, inputs, nil
}

// The characters a file name cannot have (Windows), and the names it cannot be.
const badNameChars = `<>:"/\|?*`

var reservedNames = map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true}

// nameProblem tells why a text cannot be the name of a file ("" = it can).
func nameProblem(name string) string {
	base, _, _ := strings.Cut(name, ".")
	switch {
	case name == "":
		return "ファイル名が空です"
	case len(name) > 200:
		return "ファイル名が長すぎます"
	case strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") || strings.HasPrefix(name, " "):
		return "ファイル名の端に空白やピリオドがあります"
	case reservedNames[strings.ToUpper(base)]:
		return fmt.Sprintf("「%s」は Windows でファイル名に使えません", base)
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(badNameChars, r) {
			return fmt.Sprintf("ファイル名に使えない文字があります（%q）", string(r))
		}
	}
	return ""
}

// safeName makes a file name of a host name whatever it holds.
func safeName(host string) string {
	var sb strings.Builder
	for _, r := range host {
		if r < 0x20 || r == 0x7f || r == ' ' || r == '.' || strings.ContainsRune(badNameChars, r) {
			r = '_'
		}
		sb.WriteRune(r)
	}
	if sb.Len() == 0 || reservedNames[strings.ToUpper(sb.String())] {
		return "device_" + sb.String()
	}
	return sb.String()
}

// fileName gives the name of the configuration of a device (project.yaml
// output.filename).
func (e *Env) fileName(eng *tmpl.Engine, row *model.Row) (string, error) {
	pattern := e.Proj.Config.Output.Filename
	if strings.TrimSpace(pattern) == "" {
		pattern = "{{ device.hostname }}.txt"
	}
	dev := map[string]any{}
	if t := e.Schema.Table("devices"); t != nil {
		for _, c := range t.Columns {
			if c.Secret {
				continue
			}
			var vals []string
			if cell := row.Cells[c.Name]; cell != nil {
				vals = cell.Values
			}
			dev[c.Name] = c.Value(vals)
		}
	}
	name, err := eng.Expand(pattern, map[string]any{"device": dev, "project": e.projectVars()})
	if err != nil {
		return "", fmt.Errorf("output.filename を当てられません: %v", err)
	}
	name = strings.TrimRight(name, "\r\n")
	if why := nameProblem(name); why != "" {
		return "", fmt.Errorf("%s: %s", why, name)
	}
	return name, nil
}

func (e *Env) projectVars() map[string]any {
	return map[string]any{"name": e.Proj.Config.Name, "kind": e.Proj.Config.Kind}
}

// The checks of the generated text, in the order they are told about.
var genRules = []string{"G-601", "G-602", "G-603", "G-604", "G-605", "G-606", "G-607", "G-608", "G-609", "G-610", "G-611", "G-612", "G-613", "G-614"}

const (
	noModels    = "どのメーカーパックにも、機種の定義（models.yaml）がまだありません"
	notWhole    = "テンプレートが失敗して文面がそろっていないため、行っていません"
	notThisTime = "今回は、ほかの機器だけを対象にしました（--device）"
)

// Generate reads the parameter book, checks the values, generates the
// configuration of each device, reads it back, and writes the configurations,
// the check results and the list of changes (spec 9, 11, 12.4, 13).
func (e *Env) Generate(opt GenOptions, now time.Time) (*GenerateOutcome, error) {
	raw, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	if bk == nil {
		return nil, fmt.Errorf("%s がありません。先にパラメータブックを作ってください", e.Proj.Config.Book)
	}
	ds, expandProblems := bk.Expand(e.Schema)
	hosts := book.Hosts(ds)
	if len(hosts) == 0 {
		return nil, fmt.Errorf("%s の機器のシートに、機器が 1 台もありません", e.Proj.Config.Book)
	}
	if opt.Device != "" && index(hosts, opt.Device) == len(hosts) {
		return nil, fmt.Errorf("機器「%s」は、%s の機器のシートにありません", opt.Device, e.Proj.Config.Book)
	}
	approvals, err := e.Proj.Approvals()
	if err != nil {
		return nil, err
	}
	extras, extraInputs, err := e.extras()
	if err != nil {
		return nil, err
	}
	rules, ruleInputs, err := e.caseRules()
	if err != nil {
		return nil, err
	}

	out := &GenerateOutcome{Options: opt, Dir: "out/" + project.Stamp(now), Tally: check.NewTally(),
		Approvals: approvals, Hits: map[string]int{}, Refused: map[string]string{}, Disabled: map[string]bool{},
		CaseRules: rules, hosts: hosts, data: ds}
	out.Inputs = append(out.Inputs, InputFile{Path: e.Proj.Config.Book, SHA256: sha(raw)})
	if cfg, err := os.ReadFile(e.Proj.Path(project.ConfigFile)); err == nil {
		out.Inputs = append(out.Inputs, InputFile{Path: project.ConfigFile, SHA256: sha(cfg)})
	}
	out.Inputs = append(out.Inputs, extraInputs...)
	out.Inputs = append(out.Inputs, ruleInputs...)
	tally := out.Tally

	// The values (spec 9.1 step 2).
	in := check.Input{
		Schema: e.Schema, Data: ds, Book: bk.Data,
		HasPack:      func(id string) bool { return e.Packs[id] != nil },
		Reads:        e.packReads,
		Placeholders: e.Proj.Config.Checks.Placeholders,
	}
	findings := check.Values(in, tally)
	// The pairs of devices (appendix C 1.2), on the values with the defaults
	// of each device's pack.
	in.Filled, _, _ = e.bookData(bk)
	findings = append(findings, check.Cross(in, tally)...)
	findings = append(findings, e.bookFindings(bk, ds, expandProblems, out)...)

	blocked := map[string][]string{}
	for _, f := range findings {
		if check.Blocks(f.RuleID) && f.Device != "" && !has(blocked[f.Device], f.RuleID) {
			blocked[f.Device] = append(blocked[f.Device], f.RuleID)
		}
	}

	// The configurations (spec 9.1 steps 3 to 5).
	tally.Skip("G-607", "", noModels)
	names := map[string]string{} // file name in small letters → the device that has it
	devRows := map[string]*model.Row{}
	for _, r := range ds.Tables["devices"].Rows {
		devRows[r.Get("hostname")] = r
	}
	cases := e.startCaseRules(rules, ds, devRows, out)
	// The devices are written side by side (none needs the text of
	// another), and taken in their order. Each goroutine has renderers of
	// its own: a renderer notes what the templates read.
	generated := func(h string) bool {
		return (opt.Device == "" || h == opt.Device) && len(blocked[h]) == 0
	}
	type written struct {
		o  *render.Output
		fs []check.Finding
	}
	texts := make([]written, len(hosts))
	renderers := make([]map[string]*render.Renderer, workers(len(hosts)))
	for i := range renderers {
		renderers[i] = map[string]*render.Renderer{}
	}
	vars := e.projectVars()
	work := newSpread(len(hosts), func(w, i int) {
		h := hosts[i]
		if !generated(h) {
			return
		}
		id := devRows[h].Get("vendor")
		r := renderers[w][id]
		if r == nil {
			r = render.New(e.Packs[id], e.Schema, extras)
			renderers[w][id] = r
		}
		o, fs := r.Render(ds, h, vars)
		texts[i] = written{o, append(fs, r.RoundTrip(o)...)}
	})
	defer work.stop()
	for i, h := range hosts {
		if e.Tick != nil {
			if err := e.Tick(i, len(hosts), h); err != nil {
				return nil, err
			}
		}
		work.wait(i)
		run := DeviceRun{Host: h, Pack: devRows[h].Get("vendor")}
		switch {
		case opt.Device != "" && h != opt.Device:
			run.State = GenSkipped
			for _, id := range genRules {
				if id != "G-612" { // looks at the design data, whatever is generated
					tally.Skip(id, h, notThisTime)
				}
			}
			cases.skip(h, notThisTime, true)
		case len(blocked[h]) > 0:
			sort.Strings(blocked[h])
			run.State = GenBlocked
			run.Why = "値の誤り（" + strings.Join(blocked[h], "・") + "）があるため、生成していません"
			for _, id := range genRules {
				tally.Skip(id, h, run.Why)
			}
			tally.Skip("V-113", h, run.Why)
			cases.skip(h, run.Why, false)
		default:
			p := e.Packs[run.Pack]
			o, fs := texts[i].o, texts[i].fs
			fs = append(fs, cases.lines(h, o, p)...)
			run.Output = o
			for _, id := range []string{"G-601", "G-603", "G-606", "G-608", "G-609", "G-613"} {
				tally.Look(id, h, 1)
			}
			for _, id := range []string{"G-602", "G-604", "G-605"} {
				if o.Failed {
					tally.Skip(id, h, notWhole)
				} else {
					tally.Look(id, h, 1)
				}
			}
			tally.Look("V-113", h, explicitCells(e, o.Data))

			name, err := e.fileName(p.Engine(), devRows[h])
			if err == nil {
				if first, dup := names[strings.ToLower(name)]; dup {
					err = fmt.Errorf("ファイル名 %s が、%s と重なります", name, first)
				} else {
					names[strings.ToLower(name)] = h
				}
			}
			if err != nil {
				fs = append(fs, check.New("G-613", check.Error, h, model.Ref{}, err.Error()))
				run.Why = err.Error()
			} else {
				run.name = name
			}

			enc, nl := e.Proj.Config.Output.Encoding, e.Proj.Config.Output.Newline
			if enc == "" {
				enc = p.Output.Encoding
			}
			if nl == "" {
				nl = p.Output.Newline
			}
			run.bytes, err = project.Encode(o.Text, enc, nl)
			if err != nil {
				fs = append(fs, check.New("G-601", check.Error, h, model.Ref{}, "文面を書き出せません: "+err.Error()))
				run.bytes, _ = project.Encode(o.Text, "utf-8", nl)
				run.name = ""
				run.Why = err.Error()
			}
			switch {
			case o.Failed || run.name == "":
				run.State = GenFailed
				if run.Why == "" {
					run.Why = "テンプレートが失敗したため、文面がそろっていません"
				}
			case opt.CheckOnly || opt.Dry:
				run.State = GenChecked
			default:
				run.State = GenWritten
			}
			findings = append(findings, fs...)
		}
		out.Devices = append(out.Devices, run)
	}
	for _, x := range extras {
		if !e.extraApplies(ds, x.Name) {
			out.Notes = append(out.Notes, fmt.Sprintf("extra/%s は、どの機器にも当たりません（_all.j2、@グループ名.j2、ホスト名.j2 のどれかの名前にします）", x.Name))
		}
	}

	findings = append(findings, cases.rows(ds, blocked, devRows)...)

	e.judge(out, findings)
	if opt.Dry {
		return out, nil
	}
	if !opt.CheckOnly {
		if err := e.writeConfigs(out, now); err != nil {
			return nil, err
		}
		if out.Changes, err = e.snapshot(out, now); err != nil {
			return nil, err
		}
	}
	name := "check_results"
	if out.Report, err = e.writeReport(now, name, e.checkDoc(out, now)); err != nil {
		return nil, err
	}
	if out.Changes != nil && out.Changes.Before != "" {
		if out.Changes.Report, err = e.writeReport(now, "changes", e.changesDoc(out, now)); err != nil {
			return nil, err
		}
	}
	c := out.Count()
	what := "generate"
	if opt.CheckOnly {
		what = "check"
	}
	n := map[string]int{}
	for _, d := range out.Devices {
		n[d.State]++
	}
	e.Proj.Log(now, "%s: %d 台（書き出し %d、失敗 %d、値の誤りで生成せず %d）、未承認のエラー %d、警告 %d、承認済み %d、レポート %s",
		what, len(out.Devices)-n[GenSkipped], n[GenWritten], n[GenFailed], n[GenBlocked], c.Errors, c.Warnings, c.Approved, out.Report.XLSX)
	return out, nil
}

// packReads tells whether a pack has a rule that fills a table.
func (e *Env) packReads(id, table string) bool {
	p := e.Packs[id]
	if p == nil {
		return true // V-109 tells about the pack that is not there
	}
	for _, w := range p.Writes() {
		if w.Table == table {
			return true
		}
	}
	return false
}

// explicitCells counts the cells of the "explicit" columns of a device.
func explicitCells(e *Env, data *model.Dataset) int {
	n := 0
	for _, t := range e.Schema.Tables {
		tab := data.Tables[t.Name]
		if tab == nil {
			continue
		}
		for _, c := range t.Columns {
			if c.Explicit {
				n += len(tab.Rows)
			}
		}
	}
	return n
}

// extraApplies tells whether an extra configuration is for a device of the
// book.
func (e *Env) extraApplies(ds *model.Dataset, file string) bool {
	name := strings.TrimSuffix(file, ".j2")
	if name == "_all" {
		return true
	}
	return len(book.Targets(ds, name)) > 0 && name != "*"
}

// sheetTable gives the table a sheet of the book holds ("" = none).
func sheetTable(bk *book.Book, sheet string) string {
	for table, info := range bk.Sheets {
		if info.Name == sheet && !strings.HasPrefix(table, "@") {
			return table
		}
	}
	return ""
}

// bookFindings turns what the book says into findings: V-105 (a key written
// twice), V-106 and V-107 (the device column and the common values), V-108
// (headings the definition does not have).
func (e *Env) bookFindings(bk *book.Book, ds *model.Dataset, expanded []book.Problem, out *GenerateOutcome) []check.Finding {
	tally := out.Tally
	var findings []check.Finding
	finding := func(p book.Problem, device string) {
		rule, _ := check.Lookup(p.Kind)
		at := p.Sheet
		if p.Addr != "" {
			at += "!" + p.Addr
		}
		f := check.New(p.Kind, rule.Severity, device, model.Ref{Table: sheetTable(bk, p.Sheet)}, p.Text, model.KV{Name: "セル", Value: at})
		f.At = at
		findings = append(findings, f)
	}
	rows := 0
	for _, name := range bk.Data.TableNames() {
		rows += len(bk.Data.Tables[name].Rows)
	}
	skipSheet := map[string]bool{}
	for _, id := range []string{"@devices", "@ifs"} {
		if info := bk.Sheets[id]; info != nil {
			skipSheet[info.Name] = true
		}
	}
	for _, p := range bk.Problems {
		switch p.Kind {
		case "V-102":
			// The checks of the values told about it, by device.
		case "V-105":
			rows++
			targets := book.Targets(ds, p.Device)
			if len(targets) == 0 {
				targets = []string{""}
			}
			for _, h := range targets {
				finding(p, h)
			}
		case "":
			// The sheets of the mapping are not read for the generation.
			if !skipSheet[p.Sheet] {
				out.BookNotes = append(out.BookNotes, p.String())
			}
		default:
			finding(p, "")
		}
	}
	for _, p := range expanded {
		finding(p, "")
	}
	tally.Look("V-105", "", rows)
	tally.Look("V-107", "", rows+len(bk.Common))

	sheets := 0
	for _, t := range e.Schema.Tables {
		info := bk.Sheets[t.Name]
		if info == nil {
			continue
		}
		sheets++
		for _, h := range info.Free {
			f := check.New("V-108", check.Info, "", model.Ref{Table: t.Name},
				fmt.Sprintf("シート「%s」の見出し「%s」は、表の定義にありません。自由列として読みました", info.Name, h), model.KV{Name: "見出し", Value: h})
			f.At = info.Name
			findings = append(findings, f)
		}
	}
	tally.Look("V-108", "", sheets)
	return findings
}

// scopeHas tells whether the devices of an approval of a scope hold a device.
func scopeHas(ds *model.Dataset, devices, host string) bool {
	devices = strings.TrimSpace(devices)
	if devices == "*" {
		return true
	}
	if strings.HasPrefix(devices, "@") {
		return ds != nil && has(book.Targets(ds, devices), host)
	}
	for _, d := range strings.Split(devices, ",") {
		if strings.TrimSpace(d) == host && host != "" {
			return true
		}
	}
	return false
}

// judge puts what the case says on the findings: the checks it switched off,
// the severities it changed, and the approvals (spec 11.2, 11.4).
func (e *Env) judge(out *GenerateOutcome, findings []check.Finding) {
	cfg := e.Proj.Config.Checks
	for _, id := range cfg.Disable {
		id = strings.TrimSpace(id)
		switch _, known := check.Lookup(id); {
		case id == "":
		case !known:
			out.Notes = append(out.Notes, fmt.Sprintf("checks.disable の %s は、この版にないチェックです", id))
		case check.Blocks(id):
			out.Notes = append(out.Notes, fmt.Sprintf("checks.disable の %s は止められません（値の誤りがある機器を生成しないためのチェックです）", id))
		default:
			out.Disabled[id] = true
		}
	}
	ids := make([]string, 0, len(cfg.Severity))
	for id := range cfg.Severity {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if _, known := check.Lookup(id); !known {
			out.Notes = append(out.Notes, fmt.Sprintf("checks.severity の %s は、この版にないチェックです", id))
		}
	}

	order := map[string]int{"": -1}
	for i, h := range out.hosts {
		order[h] = i
	}
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		switch {
		case order[a.Device] != order[b.Device]:
			return order[a.Device] < order[b.Device]
		case a.RuleID != b.RuleID:
			return a.RuleID < b.RuleID
		case a.Ref.Table != b.Ref.Table:
			return a.Ref.Table < b.Ref.Table
		case a.Ref.Key != b.Ref.Key:
			return tmpl.NaturalLess(a.Ref.Key, b.Ref.Key)
		case a.Ref.Column != b.Ref.Column:
			return a.Ref.Column < b.Ref.Column
		}
		return a.Line < b.Line
	})

	for _, f := range findings {
		if out.Disabled[f.RuleID] {
			continue
		}
		out.Items = append(out.Items, e.settle(f, out.Approvals, out.data, out.Hits, out.Refused))
	}
}

// settle puts the severity of the case and the approvals on one finding
// (spec 11.2, 11.4). hits counts the findings each approval covers; refused
// tells why an approval does not count for a finding it names.
func (e *Env) settle(f check.Finding, approvals []project.Approval, ds *model.Dataset, hits map[string]int, refused map[string]string) Judged {
	if s, ok := e.Proj.Config.Checks.Severity[f.RuleID]; ok && !f.Firm {
		f.Severity = s
	}
	j := Judged{Finding: f}
	for i := range approvals {
		a := &approvals[i]
		hit := a.Finding != "" && a.Finding == f.Key
		if s := a.Scope; s != nil {
			hit = s.Rule == f.RuleID && (s.Table == "" || s.Table == f.Ref.Table) &&
				(s.Column == "" || s.Column == f.Ref.Column) && scopeHas(ds, s.Devices, f.Device)
		}
		if !hit {
			continue
		}
		switch {
		case !check.Approvable(f.RuleID):
			refused[a.ID] = fmt.Sprintf("%s は承認できません（値や文面を直す指摘です）", f.RuleID)
		case a.Scope != nil && f.Severity == check.Error && !check.ScopeOK(f.RuleID):
			refused[a.ID] = fmt.Sprintf("エラーの指摘（%s）は、範囲ではなく 1 件ずつ承認します", f.RuleID)
		case f.Severity == check.Info:
			// nothing to approve
		default:
			hits[a.ID]++
			if j.Approval == nil {
				j.Approval = a
			}
		}
	}
	return j
}

// writeConfigs writes the configurations and, if something is left to look
// at, the mark UNVERIFIED.txt (spec 4, 9.6, 11.2).
func (e *Env) writeConfigs(out *GenerateOutcome, now time.Time) error {
	dir := out.Dir + "/configs"
	if err := os.MkdirAll(e.Proj.Path(dir), 0o755); err != nil {
		return fmt.Errorf("%s を作れません: %v", dir, err)
	}
	failedNames := map[string]bool{}
	for i := range out.Devices {
		d := &out.Devices[i]
		switch d.State {
		case GenWritten:
			d.File = dir + "/" + d.name
		case GenFailed:
			if err := os.MkdirAll(e.Proj.Path(dir+"/_failed"), 0o755); err != nil {
				return fmt.Errorf("%s/_failed を作れません: %v", dir, err)
			}
			name := safeName(d.Host)
			for n := 2; failedNames[strings.ToLower(name)]; n++ {
				name = fmt.Sprintf("%s_%d", safeName(d.Host), n)
			}
			failedNames[strings.ToLower(name)] = true
			d.File = dir + "/_failed/" + name + ".txt"
		default:
			continue
		}
		if err := os.WriteFile(e.Proj.Path(d.File), d.bytes, 0o644); err != nil {
			return fmt.Errorf("%s を書けません: %v", d.File, err)
		}
	}
	if out.Findings() == 0 {
		return nil
	}
	c := out.Count()
	var sb strings.Builder
	sb.WriteString("このフォルダのコンフィグには、未承認の指摘が残っています。\r\n")
	sb.WriteString("機器に入れる前に、reports/check_results.xlsx（または .html）の「指摘」を確かめてください。\r\n\r\n")
	fmt.Fprintf(&sb, "実施日時: %s\r\n", now.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&sb, "未承認のエラー: %d 件\r\n未承認の警告: %d 件\r\n", c.Errors, c.Warnings)
	if n := len(out.BookNotes); n > 0 {
		fmt.Fprintf(&sb, "パラメータブックの注意: %d 件\r\n", n)
	}
	sb.WriteString("\r\n機器ごとの未承認の件数（エラー / 警告）\r\n")
	type pair struct{ e, w int }
	per := map[string]*pair{}
	for _, f := range out.Items {
		if f.Approval != nil || f.Severity == check.Info {
			continue
		}
		if per[f.Device] == nil {
			per[f.Device] = &pair{}
		}
		if f.Severity == check.Error {
			per[f.Device].e++
		} else {
			per[f.Device].w++
		}
	}
	if p := per[""]; p != nil {
		fmt.Fprintf(&sb, "  （案件全体）  %d / %d\r\n", p.e, p.w)
	}
	for _, d := range out.Devices {
		p := per[d.Host]
		if p == nil && d.State != GenBlocked && d.State != GenFailed {
			continue
		}
		if p == nil {
			p = &pair{}
		}
		note := ""
		switch d.State {
		case GenBlocked:
			note = "  生成していません"
		case GenFailed:
			note = "  文面がそろっていません（configs/_failed）"
		}
		fmt.Fprintf(&sb, "  %s  %d / %d%s\r\n", d.Host, p.e, p.w, note)
	}
	out.Unverified = out.Dir + "/UNVERIFIED.txt"
	if err := os.WriteFile(e.Proj.Path(out.Unverified), append([]byte{0xEF, 0xBB, 0xBF}, sb.String()...), 0o644); err != nil {
		return fmt.Errorf("%s を書けません: %v", out.Unverified, err)
	}
	return nil
}

func has(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// index gives the place of s in the list; len(list) if it is not there.
func index(list []string, s string) int {
	for i, x := range list {
		if x == s {
			return i
		}
	}
	return len(list)
}
