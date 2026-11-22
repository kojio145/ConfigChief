package book

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Every sample template pours as it is, with the values of the devices in
// it (spec 14.11.2).
func TestSamplePours(t *testing.T) {
	sc := defs(t)
	for _, kind := range SampleKinds {
		data, err := Sample(kind, sc, "ja")
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		made, bad := Deliver(SampleName(kind), data, sc, sheetsData(), DeliverOptions{Project: "sample", Lang: "ja"})
		if len(bad) > 0 {
			t.Fatalf("%s: %v", kind, bad)
		}
		var all strings.Builder
		for _, m := range made {
			f, err := excelize.OpenReader(bytes.NewReader(m.Data))
			if err != nil {
				t.Fatal(err)
			}
			for _, sh := range f.GetSheetList() {
				all.WriteString(sh + "\n")
				rows, _ := f.GetRows(sh)
				for _, r := range rows {
					all.WriteString(strings.Join(r, "\t") + "\n")
				}
			}
			f.Close()
		}
		text := all.String()
		for _, want := range []string{"sample", "rt-a02", "GigabitEthernet0/10", "198.51.100.1/24", "0.0.0.0/0", "203.0.113.2", "IF 名", "宛先"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s lacks %q:\n%s", kind, want, text)
			}
		}
		if strings.Contains(text, "{{") {
			t.Errorf("%s keeps an expression:\n%s", kind, text)
		}
		want := map[string]int{"sheets": 1, "files": 2, "list": 1}[kind]
		if len(made) != want {
			t.Errorf("%s: %d files", kind, len(made))
		}
	}
	if _, err := Sample("other", sc, "ja"); err == nil {
		t.Error("a kind that is not there")
	}
}
