package ops

import (
	"testing"
)

// Several groups get one judgment with one step (spec 14.19.8).
func TestJudgeMany(t *testing.T) {
	e := judgeCase(t)
	j := review(t, e, "old")
	if len(j.Groups) < 2 {
		t.Fatalf("groups: %d", len(j.Groups))
	}
	if _, err := e.JudgeMany("old", nil, "hold", "", "admin", at); err == nil {
		t.Error("no group is refused")
	}
	if _, err := e.JudgeMany("old", []string{j.Groups[0].ID}, "none", "", "admin", at); err == nil {
		t.Error("none is not for many")
	}
	var ids []string
	for _, g := range j.Groups {
		ids = append(ids, g.ID, g.ID) // twice: counted once
	}
	o, err := e.JudgeMany("old", ids, "not_needed", "memo lines of the sheet", "admin", at)
	if err != nil {
		t.Fatal(err)
	}
	if o.Groups != len(j.Groups) || o.Lines == 0 {
		t.Errorf("outcome: %+v", o)
	}
	if o.Judging.Open+o.Judging.Hold != 0 {
		t.Errorf("left open: %d, on hold: %d", o.Judging.Open, o.Judging.Hold)
	}
	if after := review(t, e, "old"); after.Open != 0 {
		t.Errorf("open after a review: %d", after.Open)
	}
}
