package check

import "testing"

// Every check tells how its findings are put right, and no text is left
// over for a check that is gone.
func TestEveryRuleHasAFix(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range AllRules() {
		seen[r.ID] = true
		if r.Fix == "" {
			t.Errorf("%s has no fix", r.ID)
		}
		if got, _ := Lookup(r.ID); got.Fix != r.Fix {
			t.Errorf("%s: Lookup gives another fix", r.ID)
		}
	}
	for id := range fixes {
		if !seen[id] {
			t.Errorf("a fix for %s, which is not a check", id)
		}
	}
}
