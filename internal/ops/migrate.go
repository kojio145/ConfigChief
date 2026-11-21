package ops

import (
	"fmt"
	"strings"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/compare"
	"github.com/kojio145/configchief/internal/convert"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/report"
	"github.com/kojio145/configchief/internal/schema"
)

// Verdicts is the findings of a migration (appendix C 1.3: M-403 to M-408)
// with what the case says about them: the severities it changed and the
// approvals. A nil *Verdicts has no finding.
type Verdicts struct {
	Items []Judged
	// Hits counts the findings each approval covers; Refused tells why an
	// approval does not count for a finding it names.
	Hits    map[string]int
	Refused map[string]string

	approvals []project.Approval
	index     map[string]int
	// source tells that the findings are those of the customer's lists
	// (appendix C 1.4), not of a migration.
	source bool
}

// Of gives the finding with a fingerprint; nil if there is none (the check
// is switched off, or the difference is no finding).
func (v *Verdicts) Of(key string) *Judged {
	if v == nil {
		return nil
	}
	if i, ok := v.index[key]; ok {
		return &v.Items[i]
	}
	return nil
}

// Count counts the findings.
func (v *Verdicts) Count() Count {
	if v == nil {
		return Count{}
	}
	return countItems(v.Items)
}

// Open counts the errors and warnings that are not approved.
func (v *Verdicts) Open() int {
	c := v.Count()
	return c.Errors + c.Warnings
}

// Approvals lists the approvals of the checks of a migration, and the others
// that cover a finding here.
func (v *Verdicts) Approvals() []project.Approval {
	if v == nil {
		return nil
	}
	var out []project.Approval
	for _, a := range v.approvals {
		if (v.source && sourceApproval(a)) || (!v.source && migrationApproval(a)) || v.Hits[a.ID] > 0 || v.Refused[a.ID] != "" {
			out = append(out, a)
		}
	}
	return out
}

// sourceApproval tells whether an approval is for a check of the customer's
// lists.
func sourceApproval(a project.Approval) bool {
	if a.Scope != nil {
		return check.Source(a.Scope.Rule)
	}
	return check.Source(a.Rule)
}

// appliedApproval tells whether an approval is for a check of what is on
// the devices, or of the logs of the work.
func appliedApproval(a project.Approval) bool {
	rule := a.Rule
	if a.Scope != nil {
		rule = a.Scope.Rule
	}
	return check.Applied(rule) || check.Loaded(rule)
}

// migrationApproval tells whether an approval is for a check of a migration.
func migrationApproval(a project.Approval) bool {
	if a.Scope != nil {
		return check.Migration(a.Scope.Rule)
	}
	return check.Migration(a.Rule)
}

func countItems(items []Judged) Count {
	var c Count
	for _, f := range items {
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

// verdicts puts what the case says on the findings of a migration. ds is the
// design data of the book: the groups of a scope are looked up in it.
func (e *Env) verdicts(findings []check.Finding, ds *model.Dataset) (*Verdicts, error) {
	approvals, err := e.Proj.Approvals()
	if err != nil {
		return nil, err
	}
	off := map[string]bool{}
	for _, id := range e.Proj.Config.Checks.Disable {
		if id = strings.TrimSpace(id); check.Migration(id) || check.Source(id) || check.Applied(id) || check.Loaded(id) {
			off[id] = true
		}
	}
	v := &Verdicts{Hits: map[string]int{}, Refused: map[string]string{}, approvals: approvals, index: map[string]int{}}
	for _, f := range findings {
		if _, dup := v.index[f.Key]; dup || off[f.RuleID] {
			continue
		}
		v.index[f.Key] = len(v.Items)
		v.Items = append(v.Items, e.settle(f, approvals, ds, v.Hits, v.Refused))
	}
	return v, nil
}

func migFinding(rule, device string, ref model.Ref, message string, evidence ...model.KV) check.Finding {
	r, _ := check.Lookup(rule)
	return check.New(rule, r.Severity, device, ref, message, evidence...)
}

// rowRef names a row for a finding, and gives what it holds besides its key
// (secrets sealed): the evidence of a row that only one side has.
func (e *Env) rowRef(r *model.Row) (model.Ref, []model.KV) {
	t := e.Schema.Table(r.Table)
	if t == nil {
		return model.Ref{Table: r.Table, Key: r.KeyText()}, nil
	}
	key := map[string]bool{}
	for _, k := range r.Key {
		key[k.Name] = true
	}
	var ev []model.KV
	for _, col := range r.Columns() {
		if key[col] {
			continue
		}
		if v := check.Shown(t.Column(col), r.Cells[col]); v != "" {
			ev = append(ev, model.KV{Name: col, Value: v})
		}
	}
	return check.RefOf(t, r, ""), ev
}

// rowFinding is the finding of a row that only one side of a comparison with
// the old side has: M-403 for a row of the old side, M-404 for the other.
func (e *Env) rowFinding(x compare.RowDiff, old bool) check.Finding {
	ref, ev := e.rowRef(x.Row)
	if old {
		return migFinding("M-403", x.Device, ref, "旧にあって、新にない行です", ev...)
	}
	return migFinding("M-404", x.Device, ref, "新にあって、旧にない行です", ev...)
}

// cellFinding is the finding of a cell that differs between the old side and
// the other (M-405).
func (e *Env) cellFinding(x compare.CellDiff, oldIsA bool) check.Finding {
	ref := model.Ref{Table: x.Table, Key: x.ARow.KeyText(), Column: x.Column}
	var col *schema.Column
	if t := e.Schema.Table(x.Table); t != nil {
		ref, col = check.RefOf(t, x.ARow, x.Column), t.Column(x.Column)
	}
	o, n := x.A, x.B
	if !oldIsA {
		o, n = x.B, x.A
	}
	return migFinding("M-405", x.Device, ref, "旧と新で値が違います",
		model.KV{Name: "旧", Value: check.Shown(col, o)}, model.KV{Name: "新", Value: check.Shown(col, n)})
}

// diffFindings gives the findings of a comparison with the old side.
func (e *Env) diffFindings(r *compare.Result, oldIsA bool) []check.Finding {
	var out []check.Finding
	for _, x := range r.OnlyA {
		out = append(out, e.rowFinding(x, oldIsA))
	}
	for _, x := range r.OnlyB {
		out = append(out, e.rowFinding(x, !oldIsA))
	}
	for _, x := range r.Cells {
		out = append(out, e.cellFinding(x, oldIsA))
	}
	return out
}

// The findings of the mapping (M-406 to M-408). The device of M-406, of
// M-407 and of a row that was left out is the old host name.
func noReasonFinding(host string) check.Finding {
	return migFinding("M-407", host, model.Ref{}, "新ホスト名が空で、備考もありません。「機器対応」に新ホスト名か、対象外にする理由（備考）を書いてください")
}

func undecidedFinding(n convert.Note) check.Finding {
	return migFinding("M-406", n.Device, model.Ref{}, "新 IF 名が決まっていません（旧の名前のままです）。「IF 対応」に新 IF 名を書いてください。持っていかない IF は - と書きます",
		model.KV{Name: "旧 IF", Value: n.Value})
}

func sharedFinding(n convert.Note) check.Finding {
	return migFinding("M-408", n.Device, model.Ref{}, "新機器 "+n.Device+" で、旧 IF "+n.Key+" が同じ新 IF になっています",
		model.KV{Name: "新 IF", Value: n.Value}, model.KV{Name: "旧 IF", Value: n.Key})
}

func clashFinding(n convert.Note) check.Finding {
	return migFinding("M-408", n.Device, model.Ref{Table: n.Table, Key: n.Key}, "名前を直すと、先にある行とキーが同じになります。この行は書いていません")
}

// mappingFindings gives the findings of the mapping put on the old side.
func mappingFindings(a *convert.Applied) []check.Finding {
	var out []check.Finding
	for _, h := range a.NoReason {
		out = append(out, noReasonFinding(h))
	}
	for _, n := range a.Undecided {
		out = append(out, undecidedFinding(n))
	}
	for _, n := range a.Shared {
		out = append(out, sharedFinding(n))
	}
	for _, n := range a.Clashes {
		out = append(out, clashFinding(n))
	}
	return out
}

// Migration runs the checks of a migration as they stand now, writing
// nothing: the mapping of the book put on the old side, the old side against
// the book and, if input/new holds a file, against the new side. An approval
// of such a finding is for a finding that is there (appendix C 1.3).
func (e *Env) Migration() (*Verdicts, error) {
	src, err := e.importSide("old", "")
	if err != nil {
		return nil, err
	}
	_, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	if bk == nil {
		return nil, fmt.Errorf("%s がありません。先に import --source old を実行してください", e.Proj.Config.Book)
	}
	a := convert.Apply(e.Schema, src.Data, bk.Devices, bk.Ifs)
	findings := mappingFindings(a)
	design, _, _ := e.bookData(bk)
	findings = append(findings, e.diffFindings(compare.Datasets(e.Schema, a.Data, design), true)...)
	nw, err := e.newSide()
	if err != nil {
		return nil, err
	}
	if nw != nil {
		findings = append(findings, e.diffFindings(compare.Datasets(e.Schema, a.Data, nw.Data), true)...)
	}
	return e.verdicts(findings, bk.Data)
}

// stateCells gives the cells of a finding for a row of a report: the check,
// the state, the approval, the fingerprint and how the finding is put
// right, with the mark of the row. All are empty if the difference is no
// finding.
func stateCells(v *Verdicts, f check.Finding) ([]string, string) {
	j := v.Of(f.Key)
	if j == nil {
		return []string{"", "", "", "", ""}, ""
	}
	cells, mark := stateCellsOf(j)
	return append(cells, check.Fix(j.RuleID)), mark
}

func stateCellsOf(j *Judged) ([]string, string) {
	sev := check.SeverityTitle(j.Severity)
	switch {
	case j.Severity == check.Info:
		return []string{j.RuleID, "情報", "", j.Key}, ""
	case j.Approval != nil:
		a := j.Approval
		return []string{j.RuleID, "承認済み", a.ID + " " + a.By + ": " + a.Reason, j.Key}, ""
	case j.Severity == check.Error:
		return []string{j.RuleID, "未承認（" + sev + "）", "", j.Key}, report.MarkUnknown
	}
	return []string{j.RuleID, "未承認（" + sev + "）", "", j.Key}, report.MarkWarn
}

var stateHeader = []string{"チェック", "状態", "承認", "指紋", "直し方"}

// approvalSheet lists the approvals of the checks of a migration with the
// findings each covers here.
func approvalSheet(v *Verdicts) report.Sheet {
	s := report.Sheet{Name: "承認", Lead: "state/approvals.yaml にある移行のチェック（M-403〜M-408）の承認と、今回それぞれが当たった指摘の件数です。",
		Header: []string{"ID", "対象", "チェック", "機器", "理由", "承認者", "日時", "当たった件数", "備考"}}
	for _, a := range v.Approvals() {
		rule, dev := a.Rule, a.Device
		if a.Scope != nil {
			rule, dev = a.Scope.Rule, a.Scope.Devices
		}
		note, mark := v.Refused[a.ID], ""
		if note == "" && v.Hits[a.ID] == 0 {
			note = "今回、当たる指摘はありません（指摘がなくなったか、根拠の値が変わったか、別の比較の指摘です）"
		}
		if note != "" {
			mark = report.MarkWarn
		}
		s.Rows = append(s.Rows, []string{a.ID, approvalTarget(a), rule, dev, a.Reason, a.By, a.At, num(v.Hits[a.ID]), note})
		s.Marks = append(s.Marks, mark)
	}
	return s
}
