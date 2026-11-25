package ops

import (
	"fmt"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/project"
)

// ApproveRequest asks for one approval: of the finding with a fingerprint,
// or of a scope (appendix C 3.2).
type ApproveRequest struct {
	Finding string // the fingerprint, or enough of its beginning to tell it apart

	// The scope: a check, the devices ("*", "@group" or host names with
	// commas), and if wanted a table and a column, by their internal names or
	// by the names the book shows.
	Rule, Devices, Table, Column string

	Reason, By string
}

// Approved is what Approve did.
type Approved struct {
	Approval project.Approval
	Hits     int // the findings the approval covers now
	// Migration tells that the approval is for a check of a migration: it
	// counts from the next import or comparison, not from the generation.
	Migration bool
	// Source tells that the approval is for a check of the customer's lists:
	// it counts from the next reading of the lists.
	Source bool
}

// table and column find the internal names for what a person wrote.
func (e *Env) tableName(s string) (string, bool) {
	for _, t := range e.Schema.Tables {
		if t.Name == s || t.Title.JA == s || t.Title.EN == s {
			return t.Name, true
		}
	}
	return "", false
}

func (e *Env) columnName(table, s string) (string, bool) {
	t := e.Schema.Table(table)
	if t == nil {
		return "", false
	}
	for _, c := range t.Columns {
		if c.Name == s || c.Title.JA == s || c.Title.EN == s {
			return c.Name, true
		}
	}
	return "", false
}

// Approve adds an approval to state/approvals.yaml. The checks run first
// (nothing is written by them): an approval is for a finding that is there.
func (e *Env) Approve(req ApproveRequest, now time.Time) (*Approved, error) {
	req.Reason, req.By = strings.TrimSpace(req.Reason), strings.TrimSpace(req.By)
	// The reason and the name may be empty (appendix C 3.2, 2026-11-25).
	switch {
	case (req.Finding == "") == (req.Rule == ""):
		return nil, fmt.Errorf("指摘の指紋か、範囲（チェックの ID と機器）のどちらか一方を指定してください")
	}
	// The findings as they stand now: of the generation, of the migration
	// (appendix C 1.3) or of the customer's lists (1.4). A fingerprint does
	// not tell which, so they are asked in that order until one has the
	// finding. A step that cannot run (no old configuration, no mapping) is
	// passed over; the reason of the first is given if none could run.
	var items []Judged
	var data *model.Dataset
	var found []Judged
	var firstErr error
	ran := false // a step gave its findings
	pick := func() []Judged {
		var found []Judged
		for _, f := range items {
			if f.Key == req.Finding {
				return []Judged{f}
			}
			if len(req.Finding) >= 6 && strings.HasPrefix(f.Key, req.Finding) {
				found = append(found, f)
			}
		}
		return found
	}
	bookData := func() *model.Dataset {
		if _, bk, err := e.readBook(); err == nil && bk != nil {
			return bk.Data
		}
		return nil
	}
	steps := []struct {
		on  bool
		run func() error
	}{
		{!check.Migration(req.Rule) && !check.Source(req.Rule) && !check.Applied(req.Rule) && !check.Loaded(req.Rule), func() error {
			o, err := e.Generate(GenOptions{Dry: true}, now)
			if err == nil {
				items, data = o.Items, o.data
			}
			return err
		}},
		{req.Rule == "" || check.Migration(req.Rule), func() error {
			v, err := e.Migration()
			if err == nil {
				items, data = v.Items, bookData()
			}
			return err
		}},
		{req.Rule == "" || check.Source(req.Rule), func() error {
			o, err := e.Customer(false, now)
			if err == nil {
				items, data = o.Verdicts.Items, bookData()
			}
			return err
		}},
		{req.Rule == "" || check.Applied(req.Rule), func() error {
			o, err := e.verify(now, false)
			if err == nil {
				items, data = o.Verdicts.Items, bookData()
			}
			return err
		}},
		{req.Rule == "" || check.Loaded(req.Rule), func() error {
			o, err := e.loadCheck(now, false)
			if err == nil {
				items, data = o.Verdicts.Items, bookData()
			}
			return err
		}},
	}
	for _, st := range steps {
		if !st.on {
			continue
		}
		items, data = nil, nil
		if err := st.run(); err != nil {
			if req.Rule != "" {
				return nil, err
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		ran = true
		if req.Rule != "" {
			break
		}
		if found = pick(); len(found) > 0 {
			break
		}
	}
	if req.Finding != "" && len(found) == 0 && firstErr != nil && !ran {
		return nil, firstErr
	}
	all, err := e.Proj.Approvals()
	if err != nil {
		return nil, err
	}
	a := project.Approval{ID: project.NextApprovalID(all), Reason: req.Reason, By: req.By, At: now.Format(time.RFC3339)}
	hits := 0
	if req.Finding != "" {
		switch {
		case len(found) == 0:
			return nil, fmt.Errorf("指紋 %s の指摘は、今の結果にありません（値を直したか、根拠の値が変わりました）", req.Finding)
		case len(found) > 1:
			return nil, fmt.Errorf("指紋 %s で始まる指摘が %d 件あります。もう少し長く書いてください", req.Finding, len(found))
		}
		f := found[0]
		switch {
		case f.Severity == check.Info:
			return nil, fmt.Errorf("%s は情報です。承認は要りません", f.RuleID)
		case !check.Approvable(f.RuleID):
			return nil, fmt.Errorf("%s は承認できません。値や文面を直してください（%s）", f.RuleID, f.Message)
		case f.Approval != nil:
			return nil, fmt.Errorf("この指摘は、すでに承認されています（%s）", f.Approval.ID)
		}
		a.Finding, a.Rule, a.Device = f.Key, f.RuleID, f.Device
		hits = 1
	} else {
		if _, ok := check.Lookup(req.Rule); !ok {
			return nil, fmt.Errorf("%s というチェックは、この版にありません", req.Rule)
		}
		if !check.Approvable(req.Rule) {
			return nil, fmt.Errorf("%s は承認できません。値や文面を直してください", req.Rule)
		}
		sc := &project.ApprovalScope{Rule: req.Rule, Devices: strings.TrimSpace(req.Devices)}
		if sc.Devices == "" {
			return nil, fmt.Errorf("範囲の承認には、機器（*、@グループ名、ホスト名の並び）を指定してください")
		}
		if req.Table != "" {
			name, ok := e.tableName(req.Table)
			if !ok {
				return nil, fmt.Errorf("「%s」という表はありません", req.Table)
			}
			sc.Table = name
		}
		if req.Column != "" {
			if sc.Table == "" {
				return nil, fmt.Errorf("列を指定するときは、表も指定してください")
			}
			name, ok := e.columnName(sc.Table, req.Column)
			if !ok {
				return nil, fmt.Errorf("表「%s」に「%s」という列はありません", req.Table, req.Column)
			}
			sc.Column = name
		}
		for _, f := range items {
			if f.RuleID != sc.Rule || (sc.Table != "" && sc.Table != f.Ref.Table) || (sc.Column != "" && sc.Column != f.Ref.Column) ||
				!scopeHas(data, sc.Devices, f.Device) || f.Severity == check.Info {
				continue
			}
			if f.Severity == check.Error && !check.ScopeOK(f.RuleID) {
				return nil, fmt.Errorf("エラーの指摘（%s）は、範囲ではなく 1 件ずつ承認します（指紋を指定してください）", f.RuleID)
			}
			hits++
		}
		if hits == 0 {
			return nil, fmt.Errorf("その範囲に当たる指摘は、今の結果にありません")
		}
		for _, x := range all {
			if x.Scope != nil && *x.Scope == *sc {
				return nil, fmt.Errorf("同じ範囲の承認が、もうあります（%s）。変えるなら、その承認を取り消してから承認し直してください", x.ID)
			}
		}
		a.Scope = sc
	}
	if err := e.Proj.SaveApprovals(append(all, a)); err != nil {
		return nil, err
	}
	e.Proj.Log(now, "approve: %s（%s、当たった指摘 %d 件、承認者 %s）", a.ID, approvalTarget(a), hits, a.By)
	return &Approved{Approval: a, Hits: hits, Migration: check.Migration(a.Rule) || migrationApproval(a), Source: check.Source(a.Rule) || sourceApproval(a)}, nil
}

// Unapprove takes an approval out of state/approvals.yaml.
func (e *Env) Unapprove(id string, now time.Time) error {
	list, err := e.Proj.Approvals()
	if err != nil {
		return err
	}
	var rest []project.Approval
	for _, a := range list {
		if !strings.EqualFold(a.ID, strings.TrimSpace(id)) {
			rest = append(rest, a)
		}
	}
	if len(rest) == len(list) {
		return fmt.Errorf("承認 %s はありません", id)
	}
	if err := e.Proj.SaveApprovals(rest); err != nil {
		return err
	}
	e.Proj.Log(now, "approve: %s を取り消しました", id)
	return nil
}

// ApproveFindings approves findings of a result that is shown, one approval
// each with the same reason (appendix C 3.2: picked and approved together).
// The checks do not run again for each: items are the findings the screen
// has. It gives the approvals written and, for each key that was left out,
// why.
func (e *Env) ApproveFindings(items []Judged, keys []string, reason, by string, now time.Time) (done []project.Approval, left []string, err error) {
	reason, by = strings.TrimSpace(reason), strings.TrimSpace(by)
	byKey := map[string]Judged{}
	for _, f := range items {
		byKey[f.Key] = f
	}
	all, err := e.Proj.Approvals()
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]bool{}
	for _, k := range keys {
		if seen[k] {
			continue
		}
		seen[k] = true
		f, ok := byKey[k]
		switch {
		case !ok:
			left = append(left, "今の結果に無い指摘です（チェックをやり直してください）")
			continue
		case f.Severity == check.Info:
			left = append(left, f.RuleID+" は情報です。承認は要りません")
			continue
		case !check.Approvable(f.RuleID):
			left = append(left, f.RuleID+" は承認できません。値や文面を直してください")
			continue
		case f.Approval != nil:
			left = append(left, "すでに承認されています")
			continue
		}
		a := project.Approval{ID: project.NextApprovalID(all), Finding: f.Key, Rule: f.RuleID, Device: f.Device,
			Reason: reason, By: by, At: now.Format(time.RFC3339)}
		all = append(all, a)
		done = append(done, a)
	}
	if len(done) == 0 {
		return nil, left, nil
	}
	if err := e.Proj.SaveApprovals(all); err != nil {
		return nil, nil, err
	}
	e.Proj.Log(now, "approve: %d 件をまとめて承認しました（%s〜%s、1 件ずつ、承認者 %s）", len(done), done[0].ID, done[len(done)-1].ID, by)
	return done, left, nil
}
