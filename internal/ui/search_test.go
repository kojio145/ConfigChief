package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearch(t *testing.T) {
	s, dir := newSession(t, map[string]string{
		"input/old/rt-a01.txt":  "hostname rt-a01\ninterface GigabitEthernet0/1\n ip address 192.0.2.1 255.255.255.0\n",
		"input/old/rt-a02.txt":  "hostname rt-a02\n",
		"input/logs/rt-a01.log": "rt-a01#show ip interface brief\nGigabitEthernet0/1 192.0.2.1 up\n",
		"input/old/~$lock.txt":  "192.0.2.1\n",
	})
	os.MkdirAll(filepath.Join(dir, "out", "2026-01-01_000000", "configs"), 0o755)
	os.MkdirAll(filepath.Join(dir, "out", "2026-02-01_000000", "configs"), 0o755)
	writeTestFile(filepath.Join(dir, "out", "2026-02-01_000000", "configs", "rt-a01.txt"), []byte("ip address 192.0.2.1 255.255.255.0\n"), 0o644)

	v, err := s.Search(SearchOptions{})
	if err != nil || len(v.Outs) != 2 || v.Outs[0] != "2026-02-01_000000" || len(v.Files) != 0 {
		t.Fatalf("no query: %+v %v", v, err)
	}
	v, err = s.Search(SearchOptions{Query: "192.0.2.1", Misses: true})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range v.Files {
		got = append(got, f.Rel)
	}
	if strings.Join(got, ",") != "input/old/rt-a01.txt,input/logs/rt-a01.log,out/2026-02-01_000000/configs/rt-a01.txt" || v.Hits != 3 || v.Out != "2026-02-01_000000" {
		t.Errorf("files: %v hits %d out %q", got, v.Hits, v.Out)
	}
	if len(v.Missed) != 1 || v.Missed[0] != "input/old/rt-a02.txt" {
		t.Errorf("missed: %v", v.Missed)
	}
	if h := v.Files[0].Hits[0]; h.Line != 3 || h.Spans[0] != [2]int{12, 21} {
		t.Errorf("hit: %+v", h)
	}
	v, _ = s.Search(SearchOptions{Query: "gigabit", Scopes: []string{"logs"}})
	if len(v.Files) != 1 || v.Files[0].Scope != "ログ" {
		t.Errorf("a place and small letters: %+v", v.Files)
	}
	v, _ = s.Search(SearchOptions{Query: "gigabit", Case: true})
	if len(v.Files) != 0 {
		t.Errorf("capitals told apart: %+v", v.Files)
	}
	v, _ = s.Search(SearchOptions{Query: `^hostname rt-a0[12]$`, Regex: true})
	if v.Hits != 2 {
		t.Errorf("a regular expression: %d", v.Hits)
	}
	if _, err := s.Search(SearchOptions{Query: "(", Regex: true}); err == nil {
		t.Error("a broken regular expression is an error")
	}
	f, err := s.ReadFile("input/old/rt-a01.txt")
	if err != nil || len(f.Lines) != 3 || f.Lines[1] != "interface GigabitEthernet0/1" {
		t.Errorf("ReadFile: %+v %v", f, err)
	}
	for _, bad := range []string{"../x.txt", `input\old\rt-a01.txt`, "C:/x", "/etc"} {
		if _, err := s.ReadFile(bad); err == nil {
			t.Errorf("ReadFile(%q) is refused", bad)
		}
	}
}

func TestSearchSpansCountUTF16(t *testing.T) {
	m, _ := matcher(SearchOptions{Query: "ab"})
	if got := m("拠点ab"); len(got) != 1 || got[0] != [2]int{2, 4} {
		t.Errorf("spans: %v", got)
	}
}

func TestDiffSide(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")
	writeTestFile(a, []byte("hostname rt-a01\nip address 192.0.2.1\nold only\nsame\n"), 0o644)
	writeTestFile(b, []byte("hostname rt-a01\nip address 192.0.2.9\nsame\nnew <one>\n"), 0o644)
	s := New("1.2.3", "")
	v, err := s.Diff(a, b)
	if err != nil {
		t.Fatal(err)
	}
	var ops []string
	for _, r := range v.Side {
		ops = append(ops, r.Op)
	}
	if strings.Join(ops, ",") != "same,change,del,same,add" {
		t.Fatalf("rows: %v", ops)
	}
	if c := v.Side[1]; c.SA[0] != [2]int{19, 20} || c.SB[0] != [2]int{19, 20} || c.A != 2 || c.B != 2 {
		t.Errorf("the pair: %+v", c)
	}
	page, err := s.DiffHTML(v)
	if err != nil {
		t.Fatal(err)
	}
	html := string(page)
	if !strings.Contains(html, "<mark>1</mark>") || !strings.Contains(html, "new &lt;one&gt;") || strings.Contains(html, "<script") {
		t.Errorf("the page: %s", html)
	}
}
