package update

import (
	"testing"
	"time"
)

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"v1.3.0", "1.2.0", true},
		{"1.2.10", "1.2.9", true},
		{"1.2.0", "1.2.0", false},
		{"1.1.9", "1.2.0", false},
		{"v1.0.0", "0.0.0-dev", true}, // the development build: every release is newer
		{"2", "1.9.9", true},
	} {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%q, %q) = %v", c.a, c.b, got)
		}
	}
}

func TestNoteBody(t *testing.T) {
	body := "---\r\n**Title:** x\r\n\r\n- English\r\n\r\n## 日本語\r\n\r\n### 見出し\r\n\r\n- 日本語の変更\r\n\r\n## Download\r\n\r\n| File | SHA-256 |\r\n"
	if got := NoteBody(body); got != "### 見出し\n\n- 日本語の変更" {
		t.Errorf("NoteBody = %q", got)
	}
	if got := NoteBody("- only English\n\n## Download\nx"); got != "- only English" {
		t.Errorf("NoteBody without Japanese = %q", got)
	}
}

func TestNotesOf(t *testing.T) {
	at := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	rels := []ghRelease{
		{TagName: "v1.4.0", Body: "next", PublishedAt: at},            // past the latest: not yet what a download gives
		{TagName: "v1.3.1", Body: "b", PublishedAt: at},               //
		{TagName: "v1.3.0", Body: "a", PublishedAt: at},               //
		{TagName: "v1.3.2-rc", Body: "rc", Prerelease: true},          // a pre-release
		{TagName: "v1.3.5", Body: "draft", Draft: true},               // a draft
		{TagName: "v1.2.0", Body: "this build", PublishedAt: at},      // not newer
		{TagName: "v1.1.0", Body: "older than this", PublishedAt: at}, //
	}
	got := notesOf(rels, "v1.3.1", "1.2.0")
	if len(got) != 2 || got[0].Version != "1.3.1" || got[1].Version != "1.3.0" || got[0].Published == "" {
		t.Fatalf("notesOf = %+v", got)
	}
}

func TestFileNameAndPages(t *testing.T) {
	if got := FileName("v1.3.0"); got != "ConfigChief-v1.3.0.exe" {
		t.Errorf("FileName = %q", got)
	}
	if got := FileName("1.3.0/../x"); got != "ConfigChief-v1.3.0_.._x.exe" {
		t.Errorf("FileName keeps the name to safe letters: %q", got)
	}
	if got := NotesPage("1.3.0"); got != "https://github.com/kojio145/ConfigChief/releases/tag/v1.3.0" {
		t.Errorf("NotesPage = %q", got)
	}
	if got := NotesPage(""); got != ReleasePage {
		t.Errorf("NotesPage of nothing = %q", got)
	}
}

func TestSaveDir(t *testing.T) {
	dir := t.TempDir()
	if got := SaveDir(dir); got != dir {
		t.Errorf("SaveDir = %q, want the folder of the exe", got)
	}
	if !sameDir(dir, dir+"/") {
		t.Error("sameDir")
	}
}
