package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/importer"
)

// site is a way of one site router from one vendor to the other.
type site struct {
	dir                         string // the folder of the samples under testdata
	old, oldHost, newHost, pack string
	ifs                         map[string]string
	// mend is what a person writes in the book for the new vendor: sheet,
	// the heading and the value that find the row, the heading to set, the
	// value.
	mend [][5]string
	// stops is what the new pack tells before the book is mended.
	stops []string
	// unused is what the new pack has no line for (G-602), by the headings.
	unused []string
	// first is how many errors and warnings the first run tells.
	first int
	// notes is what the new pack wrote in another way and told (G-614): it
	// stays told after the book is mended, until a person approves it.
	notes []string
}

func siteFile(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

// unsettled lists the errors and warnings of an outcome that are not approved.
func unsettled(o *GenerateOutcome) []string {
	var out []string
	for _, f := range o.Items {
		if f.Approval == nil && f.Severity != check.Info {
			out = append(out, f.RuleID+" "+f.Ref.Table+" "+f.Ref.Key+": "+f.Message)
		}
	}
	return out
}

// A site router with routes, OSPF, BGP, a redundancy group, a watch of its
// line and NAT goes from one vendor to the other, both ways (the end of
// stage M3, spec 18). What a person has to write for the new vendor is
// listed in the test: nothing else stands between the old configuration and
// the new one, and the new one reads back as the book says.
func TestCrossVendorSite(t *testing.T) {
	ways := []site{
		{
			old: "rt-s01.txt", oldHost: "rt-s01", newHost: "ix-s01", pack: "nec_ix",
			ifs: map[string]string{"GigabitEthernet0/0": "GigaEthernet0.0", "GigabitEthernet0/1": "GigaEthernet1.0", "Loopback0": "Loopback0.0"},
			// The new vendor has no direction of NAT on an interface: the pack
			// leaves it out and tells.
			first: 1,
			notes: []string{"GigaEthernet0.0: 「NAT の向き」は書きません"},
		},
		{
			old: "ix-s01.txt", oldHost: "ix-s01", newHost: "rt-s01", pack: "cisco_ios",
			ifs: map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet1.0": "GigabitEthernet0/1", "Loopback0.0": "Loopback0"},
			// The new vendor numbers its watches: the pack gives the named
			// watch the next free number, and the route that names it
			// follows (2026-11-21). The marks of the interfaces inside and
			// outside of NAT, and the count of failed probes (written as a
			// delay of the watch), come from the pack too.
			first: 4,
			notes: []string{
				"回線監視 wg1: Cisco の監視は 1〜1000 の番号なので、1 番として書きます",
				"GigabitEthernet0/0: 「NAT の向き」が空なので、outside として書きます",
				"GigabitEthernet0/1: 「NAT の向き」が空なので、inside として書きます",
				"「断と判断する回数」（6 回）は Cisco に無いので、「断の反映の待ち」25 秒（間隔 5 秒 × 5）にして書きます",
			},
		},
	}
	crossSites(t, "site", ways)
}

// crossSites takes each sample from its old vendor to the new one: import,
// the mappings, the book, a first run, what a person mends, a clean run.
func crossSites(t *testing.T, dir string, ways []site) {
	for _, w := range ways {
		t.Run(w.oldHost+" to "+w.pack, func(t *testing.T) {
			e := newCase(t, map[string]string{"input/old/" + w.old: siteFile(t, dir, w.old)})
			in, err := e.RunImport("old", "", at)
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range in.Source.Files {
				if f.Result == nil {
					t.Fatalf("%s was not imported: %s", f.Path, f.Err)
				}
				for _, l := range f.Result.Lines {
					if l.State == importer.Unknown {
						t.Errorf("a line of the old configuration was not read: %s", l.Text)
					}
				}
			}
			setCell(t, e, "新旧対照表（機器）", "旧ホスト名", w.oldHost, "新ホスト名", w.newHost)
			setCell(t, e, "新旧対照表（機器）", "旧ホスト名", w.oldHost, "新メーカー", w.pack)
			if _, err := e.RunImport("old", "", at); err != nil {
				t.Fatal(err)
			}
			for old, name := range w.ifs {
				setCell(t, e, "新旧対照表（ポート）", "旧 IF 名", old, "新 IF 名", name)
			}
			b, err := e.MakeBook("old", false, at)
			if err != nil {
				t.Fatal(err)
			}
			if a := b.Applied; len(a.Undecided)+len(a.Dropped)+len(a.Cleared)+len(a.Clashes)+len(b.Problems) != 0 {
				t.Fatalf("the mapping: %+v, problems %v", a, b.Problems)
			}

			// Before a person mends the book.
			o, err := e.Generate(GenOptions{}, at)
			if err != nil {
				t.Fatal(err)
			}
			told := strings.Join(unsettled(o), "\n")
			for _, want := range append(append([]string{}, w.stops...), w.unused...) {
				if !strings.Contains(told, want) {
					t.Errorf("the first run lacks %q:\n%s", want, told)
				}
			}
			if n := len(unsettled(o)); n != w.first {
				t.Errorf("the first run told %d things, not %d:\n%s", n, w.first, told)
			}

			for _, m := range w.mend {
				setCell(t, e, m[0], m[1], m[2], m[3], m[4])
			}
			if o, err = e.Generate(GenOptions{}, at); err != nil {
				t.Fatal(err)
			}
			var left, noted []string
			for _, l := range unsettled(o) {
				if strings.HasPrefix(l, "G-614 ") {
					noted = append(noted, l)
				} else {
					left = append(left, l)
				}
			}
			if o.Devices[0].State != GenWritten || len(left) != 0 || (o.Unverified != "") != (len(w.notes) > 0) {
				t.Fatalf("after the book was mended: %s, mark %q:\n%s", o.Devices[0].State, o.Unverified, strings.Join(left, "\n"))
			}
			if len(noted) != len(w.notes) {
				t.Errorf("told as written in another way: %d, not %d:\n%s", len(noted), len(w.notes), strings.Join(noted, "\n"))
			}
			for _, want := range w.notes {
				if !strings.Contains(strings.Join(noted, "\n"), want) {
					t.Errorf("not told: %q:\n%s", want, strings.Join(noted, "\n"))
				}
			}
			got := strings.ReplaceAll(read(t, e, o.Devices[0].File), "\r\n", "\n")
			file := filepath.Join("testdata", dir, "to_"+w.newHost+".txt")
			if os.Getenv("CROSS_WRITE") != "" {
				os.WriteFile(file, []byte(got), 0o644)
			}
			want, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
				t.Errorf("the configuration for the new vendor:\n%s", got)
			}
		})
	}
}
