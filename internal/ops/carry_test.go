package ops

import (
	"os"
	"strings"
	"testing"
)

// The report that may leave the site: the names and the secrets of the
// unhandled lines are hidden, lines that differ only in a name are one
// shape, and nothing of the case is in the file.
func TestCarryOut(t *testing.T) {
	base := packSample(t)
	extra := func(name, secret string) string {
		return "zzfeature " + name + " mode 10\n" +
			"zzblock " + name + "\n" +
			" zzmember password " + secret + "\n" +
			" description kept for " + name + "\n" +
			"!\n"
	}
	files := map[string]string{
		"input/old/honsha-gw-file.txt": strings.Replace(base, "hostname rt-a01\n", "hostname rt-a01\n"+extra("HONSHA-GW", "himitsu-one"), 1),
		"input/old/shisha-gw-file.txt": strings.Replace(strings.Replace(base, "hostname rt-a01\n", "hostname rt-a02\n", 1), "hostname rt-a02\n", "hostname rt-a02\n"+extra("SHISHA-GW", "himitsu-two"), 1),
	}
	e := newCase(t, files)
	o, err := e.CarryOut("", at)
	if err != nil {
		t.Fatal(err)
	}
	if o.Source != "old" {
		t.Fatalf("the source: %s", o.Source)
	}
	var feature, member *CarryShape
	for i := range o.Shapes {
		s := &o.Shapes[i]
		switch {
		case s.Shape == "<name> <name> mode <n>" && len(s.Path) == 0 && s.Devices == 2:
			feature = s
		case strings.HasSuffix(s.Shape, "password <secret>"):
			member = s
		}
	}
	// "zzfeature HONSHA-GW mode 10" and "zzfeature SHISHA-GW mode 10" are one
	// shape; none of its words is known to the pack but the number.
	if feature == nil || feature.Lines != 2 {
		t.Fatalf("the shape of the two lines that differ in a name: %+v in %+v", feature, o.Shapes)
	}
	// The line below an unhandled line follows it and is not a shape of its
	// own; the secret after "password" never shows.
	for _, s := range o.Shapes {
		all := strings.Join(s.Path, " ") + " " + s.Shape
		for _, bad := range []string{"HONSHA", "SHISHA", "himitsu", "kept for"} {
			if strings.Contains(all, bad) {
				t.Errorf("the shape %q shows %q", all, bad)
			}
		}
	}
	_ = member
	hidden := map[string]int{}
	for _, h := range o.Hidden {
		hidden[h.Word] = h.Lines
	}
	if hidden["HONSHA-GW"] == 0 || hidden["SHISHA-GW"] == 0 || hidden["zzfeature"] != 2 {
		t.Errorf("the hidden words: %v", o.Hidden)
	}
	for _, f := range []string{o.Report.HTML, o.Report.XLSX} {
		data, err := os.ReadFile(e.Proj.Path(f))
		if err != nil {
			t.Fatal(err)
		}
		if f == o.Report.XLSX {
			continue // packed; the HTML form holds the same rows
		}
		for _, bad := range []string{"HONSHA", "SHISHA", "himitsu", "honsha-gw-file", "rt-a01", "rt-a02", "zzfeature", e.Proj.Dir} {
			if strings.Contains(string(data), bad) {
				t.Errorf("%s shows %q", f, bad)
			}
		}
		if !strings.Contains(string(data), "&lt;name&gt;") || !strings.Contains(string(data), "目で確かめてください") {
			t.Errorf("%s lacks the marks or the warning", f)
		}
	}

	// A word the user allows stays.
	e.Proj.Config.Export.KeepWords = []string{"zzfeature", "Mode"}
	o, err = e.CarryOut("old", at)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range o.Shapes {
		found = found || s.Shape == "zzfeature <name> mode <n>"
	}
	if !found {
		t.Errorf("the allowed words are not kept: %+v", o.Shapes)
	}
	if _, err := e.CarryOut("new", at); err == nil {
		t.Error("a source without files gave no error")
	}
}
