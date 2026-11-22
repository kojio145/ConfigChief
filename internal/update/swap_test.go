package update

import (
	"os"
	"path/filepath"
	"testing"
)

// The downloaded exe takes the name of the running one, which keeps the
// name of its version; Undo puts both back (spec 14.11.1).
func TestSwap(t *testing.T) {
	dir := t.TempDir()
	put := func(name, text string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	readText := func(p string) string {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	running := put("ConfigChief.exe", "old")
	down := put("ConfigChief-v1.4.0.exe", "new")
	put("ConfigChief-v1.3.1.exe", "older copy")
	sum, err := FileSHA256(down)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Swap(running, down, "00", "1.3.1"); err == nil {
		t.Error("a file that changed was put in")
	}
	if _, err := Swap(running, "", sum, "1.3.1"); err == nil {
		t.Error("nothing downloaded")
	}
	other := filepath.Join(t.TempDir(), "ConfigChief-v1.4.0.exe")
	if err := os.WriteFile(other, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Swap(running, other, sum, "1.3.1"); err == nil {
		t.Error("a file of another folder was put in")
	}
	s, err := Swap(running, down, sum, "1.3.1")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(s.Kept) != "ConfigChief-v1.3.1_2.exe" || readText(running) != "new" || readText(s.Kept) != "old" {
		t.Fatalf("swapped = %+v", s)
	}
	if _, err := os.Stat(down); err == nil {
		t.Error("the downloaded file is still there by its name")
	}
	if err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if readText(running) != "old" || readText(down) != "new" {
		t.Error("undo")
	}
}
