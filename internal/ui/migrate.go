package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kojio145/configchief/internal/ops"
)

// migFindings gives the findings of a migration (appendix C 1.3) for a
// screen: the comparison and the mapping show them in their lists, and a
// row is tied to its finding by the fingerprint in the column 指紋.
func migFindings(e *ops.Env, v *ops.Verdicts) []FindingView {
	out := []FindingView{}
	if v == nil {
		return out
	}
	for _, f := range v.Items {
		out = append(out, findingView(e, nil, f))
	}
	return out
}

// ApproveMigration approves a finding of a migration that the comparison or
// the mapping shows. scope: "" = this finding only, "device" = the findings
// of the same check, table and column on its device, "all" = those on every
// device. Nothing but state/approvals.yaml is written: the screen asks for
// the list again.
func (s *Session) ApproveMigration(key, scope, reason, by string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil {
		return "", errors.New("案件を開いていません")
	}
	f := s.mig.Of(key)
	if f == nil {
		return "", errors.New("その指摘は、今の一覧にありません。比べ直すか、読み直してください")
	}
	req := ops.ApproveRequest{Reason: reason, By: by}
	switch scope {
	case "":
		req.Finding = key
	case "device":
		if f.Device == "" {
			return "", errors.New("機器を 1 台に決められない指摘です。「この指摘だけ」か「全機器」で承認してください")
		}
		req.Rule, req.Devices, req.Table, req.Column = f.RuleID, f.Device, f.Ref.Table, f.Ref.Column
	case "all":
		req.Rule, req.Devices, req.Table, req.Column = f.RuleID, "*", f.Ref.Table, f.Ref.Column
	default:
		return "", fmt.Errorf("承認の範囲「%s」は使えません", scope)
	}
	e, err := s.beginWrite()
	if err != nil {
		return "", err
	}
	a, err := e.Approve(req, s.now())
	if err != nil {
		return "", err
	}
	if name := strings.TrimSpace(by); name != s.name {
		s.name = name
		s.save()
	}
	return fmt.Sprintf("承認しました（%s。当たった指摘 %d 件）", a.Approval.ID, a.Hits), nil
}

// UnapproveMigration takes an approval back; the screen asks for the list
// again.
func (s *Session) UnapproveMigration(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil {
		return errors.New("案件を開いていません")
	}
	e, err := s.beginWrite()
	if err != nil {
		return err
	}
	return e.Unapprove(id, s.now())
}

// ApproveMigrationMany approves findings of a migration that the mapping or
// the comparison shows, picked together: one approval each, with the same
// reason (spec 14.22.10; appendix C 3.2). The screen asks for the list again.
func (s *Session) ApproveMigrationMany(keys []string, reason, by string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil {
		return "", errors.New("案件を開いていません")
	}
	if s.mig == nil {
		return "", errors.New("指摘の一覧がありません。読み直してください")
	}
	if len(keys) == 0 {
		return "", errors.New("承認する指摘を選んでください")
	}
	e, err := s.beginWrite()
	if err != nil {
		return "", err
	}
	done, left, err := e.ApproveFindings(s.mig.Items, keys, reason, by, s.now())
	if err != nil {
		return "", err
	}
	if name := strings.TrimSpace(by); name != s.name {
		s.name = name
		s.save()
	}
	msg := fmt.Sprintf("%d 件を承認しました", len(done))
	if len(left) > 0 {
		msg += fmt.Sprintf("。%d 件は承認できませんでした: %s", len(left), firstKinds(left))
	}
	return msg, nil
}
