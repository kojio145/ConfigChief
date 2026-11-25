package ops

import (
	"fmt"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/judge"
	"github.com/kojio145/configchief/internal/project"
)

// JudgeManyOutcome is what JudgeMany did.
type JudgeManyOutcome struct {
	Judging *Judging
	Verdict string
	Groups  int // the groups judged
	Lines   int // their lines on every device
}

// JudgeMany puts one judgment on several groups at once, for every device
// (spec 14.19.8): the lines of hundreds of groups are judged with one step.
// The configurations are read once and the judgments written once.
func (e *Env) JudgeMany(source string, groups []string, verdict, reason, by string, now time.Time) (*JudgeManyOutcome, error) {
	reason, by, verdict = strings.TrimSpace(reason), strings.TrimSpace(by), strings.TrimSpace(verdict)
	if source == "" {
		source = "old"
	}
	switch verdict {
	case project.VerdictManual, project.VerdictHold, project.VerdictNotNeeded:
	default:
		return nil, fmt.Errorf("まとめて付ける判定は manual（追加コンフィグで書く）・not_needed（要らない）・hold（保留）のどれかです（「%s」）", verdict)
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("判定を付けるかたまりがありません")
	}
	src, err := e.importSide(source, "")
	if err != nil {
		return nil, err
	}
	view := e.sideView(src)
	before, err := e.judging(src, view)
	if err != nil {
		return nil, err
	}
	list, err := e.Proj.Judgments()
	if err != nil {
		return nil, err
	}
	out := &JudgeManyOutcome{Verdict: verdict}
	seen := map[string]bool{}
	var ids []string
	for _, id := range groups {
		g, err := before.Group(id)
		if err != nil {
			return nil, err
		}
		if seen[g.ID] {
			continue
		}
		seen[g.ID] = true
		ids = append(ids, g.ID)
		list = judge.Set(list, g.Key, nil, project.Judgment{Verdict: verdict, Reason: reason, By: by, At: now.Format(time.RFC3339)})
		out.Groups++
		for _, d := range g.Devices {
			if !d.Own {
				out.Lines += d.Lines()
			}
		}
	}
	if err := e.Proj.SaveJudgments(list); err != nil {
		return nil, err
	}
	if out.Judging, err = e.judging(src, view); err != nil {
		return nil, err
	}
	e.Proj.Log(now, "judge: %s かたまり %d 個に %s（全台、%d 行、判定した人 %s）: %s", source, out.Groups, VerdictWord(verdict), out.Lines, by, strings.Join(ids, " "))
	return out, nil
}
