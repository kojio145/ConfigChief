package ops

import (
	"os"
	"testing"
	"time"
)

// The screen of the import imports again when the files changed since the
// last import (spec 14.19.3).
func TestImportStale(t *testing.T) {
	e := newCase(t, map[string]string{"input/old/rt-a01.txt": sample(t, "cisco_ios")})
	stale := func(want bool, what string) {
		t.Helper()
		got, err := e.ImportStale("old")
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s: stale = %v, want %v", what, got, want)
		}
	}
	stale(true, "never imported")
	if got, _ := e.ImportStale("new"); got {
		t.Error("no files is never stale")
	}
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	stale(false, "just imported")
	if err := os.WriteFile(e.Proj.Path("input/old/rt-a02.txt"), []byte(sample(t, "cisco_ios")), 0o644); err != nil {
		t.Fatal(err)
	}
	stale(true, "a file added")
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	stale(false, "imported again")
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(e.Proj.Path("input/old/rt-a02.txt"), later, later); err != nil {
		t.Fatal(err)
	}
	stale(true, "a file changed")
	if err := os.Remove(e.Proj.Path("input/old/rt-a02.txt")); err != nil {
		t.Fatal(err)
	}
	stale(true, "a file taken away")
}
