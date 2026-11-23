package ops

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/kojio145/configchief/internal/casefile"
)

func caseWithSecrets(t *testing.T) *Env {
	t.Helper()
	cfg, err := os.ReadFile("testdata/basics/rt-g01.txt")
	if err != nil {
		t.Fatal(err)
	}
	e := newCase(t, map[string]string{
		"input/old/rt-g01.txt":  string(cfg),
		"extra/rt-g01.txt":      "! a note that repeats the secret: dummy-ppp-password\n",
		"out/x.txt":             "made\n",
		"state/backup/old.xlsx": "copy",
		"state/draft.yaml":      "version: 1\n",
	})
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	// The book gets the rows of a device once it has its new name.
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", "rt-g01", "新ホスト名", "rt-g01")
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	if _, err := e.MakeBook("old", true, at); err != nil {
		t.Fatal(err)
	}
	return e
}

func filesOf(t *testing.T, data []byte, password string) map[string][]byte {
	t.Helper()
	_, files, err := casefile.Unpack(data, password)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	for _, f := range files {
		out[f.Path] = f.Data
	}
	return out
}

func TestExportCase(t *testing.T) {
	e := caseWithSecrets(t)
	x, err := e.ExportCase(false, "", at)
	if err != nil {
		t.Fatal(err)
	}
	files := filesOf(t, x.Data, "")
	for _, want := range []string{"project.yaml", "parameters.xlsx", "input/old/rt-g01.txt", "extra/rt-g01.txt", "state/draft.yaml"} {
		if files[want] == nil {
			t.Errorf("%s is written out", want)
		}
	}
	for rel := range files {
		if strings.HasPrefix(rel, "out/") || strings.HasPrefix(rel, "state/backup/") {
			t.Errorf("%s is not written out", rel)
		}
	}
	if !strings.HasSuffix(x.Name, ".ccase") || x.Encrypted || !bytes.Contains(files["input/old/rt-g01.txt"], []byte("dummy-admin-secret-1")) {
		t.Errorf("as it is: %s %v", x.Name, x.Encrypted)
	}
}

func TestExportCaseWithoutSecrets(t *testing.T) {
	e := caseWithSecrets(t)
	x, err := e.ExportCase(true, "correct horse", at)
	if err != nil {
		t.Fatal(err)
	}
	if !x.Encrypted || !x.NoSecrets || x.Values == 0 || x.Masked == 0 {
		t.Fatalf("%+v", x)
	}
	if _, _, err := casefile.Unpack(x.Data, "wrong horse!"); err == nil {
		t.Fatal("a wrong password")
	}
	files := filesOf(t, x.Data, "correct horse")
	if files["state/draft.yaml"] != nil {
		t.Error("the draft is left out")
	}
	for rel, data := range files {
		for _, secret := range []string{"dummy-admin-secret-1", "dummy-ppp-password"} {
			if bytes.Contains(data, []byte(secret)) {
				t.Errorf("%s still holds %s", rel, secret)
			}
		}
	}
	if !bytes.Contains(files["input/old/rt-g01.txt"], []byte("secret 0 ****")) || !bytes.Contains(files["extra/rt-g01.txt"], []byte("secret: ****")) {
		t.Errorf("masked: %s", files["input/old/rt-g01.txt"])
	}
	// The book: no secret left in any cell, the masks in their place.
	path := filepath.Join(t.TempDir(), "b.xlsx")
	os.WriteFile(path, files["parameters.xlsx"], 0o644)
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	masks := 0
	for _, sh := range f.GetSheetList() {
		rows, _ := f.GetRows(sh)
		for _, r := range rows {
			for _, c := range r {
				if strings.Contains(c, "dummy-admin-secret-1") || strings.Contains(c, "dummy-ppp-password") {
					t.Errorf("sheet %s holds a secret", sh)
				}
				if c == "****" {
					masks++
				}
			}
		}
	}
	if masks == 0 {
		t.Error("the secret cells are masked")
	}
	f.Close()
}

func TestImportCase(t *testing.T) {
	e := caseWithSecrets(t)
	x, err := e.ExportCase(false, "correct horse", at)
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	if _, err := ImportCase(x.Data, "", parent); err == nil {
		t.Fatal("a sealed file needs its password")
	}
	got, err := ImportCase(x.Data, "correct horse", parent)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(got.Dir) != parent || got.Name != e.Proj.Config.Name || got.NoSecrets {
		t.Errorf("%+v", got)
	}
	e2, err := Open(got.Dir)
	if err != nil {
		t.Fatalf("the case opens: %v", err)
	}
	if paths, _ := e2.Proj.Inputs("old"); len(paths) != 1 {
		t.Errorf("the inputs: %v", paths)
	}
	if st, err := os.Stat(filepath.Join(got.Dir, "out")); err != nil || !st.IsDir() {
		t.Error("the folders of a case are made")
	}
	again, err := ImportCase(x.Data, "correct horse", parent)
	if err != nil || again.Dir == got.Dir || !strings.HasSuffix(again.Dir, "_2") {
		t.Errorf("a second import goes beside the first: %+v %v", again, err)
	}
	if _, err := ImportCase([]byte("not a case"), "", parent); err == nil {
		t.Error("not a case file")
	}
}

func TestSafeFileName(t *testing.T) {
	if got := SafeFileName(` a/b:c*? `, "case"); got != "a_b_c__" {
		t.Errorf("%q", got)
	}
	if got := SafeFileName(" .. ", "case"); got != "case" {
		t.Errorf("%q", got)
	}
}
