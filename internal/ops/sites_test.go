package ops

import (
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/packs"
)

func packSample(t *testing.T) string {
	t.Helper()
	fsys, err := packs.FS("cisco_ios")
	if err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(fsys, "tests/import/basic/input.txt")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func siteItemOf(t *testing.T, o *SitesOutcome, table, row, column string) *SiteItem {
	t.Helper()
	for i := range o.Tables {
		if o.Tables[i].Name != table {
			continue
		}
		for j := range o.Tables[i].Items {
			it := &o.Tables[i].Items[j]
			if strings.Contains(it.Row, row) && it.Column == column {
				return it
			}
		}
	}
	t.Fatalf("no item %s / %s / %s", table, row, column)
	return nil
}

// Four devices side by side: a value one device has differently, a row one
// device lacks, a value every device has differently, a secret.
func TestSites(t *testing.T) {
	base := packSample(t)
	files := map[string]string{}
	for _, h := range []string{"rt-a01", "rt-a02", "rt-a03", "rt-a04"} {
		text := strings.Replace(base, "hostname rt-a01\n", "hostname "+h+"\n", 1)
		// Every device has its own description on the loopback.
		text = strings.Replace(text, " description sample loopback\n", " description loopback of "+h+"\n", 1)
		switch h {
		case "rt-a02":
			text = strings.Replace(text, " description sample lan\n", " description sample lan 2\n", 1)
		case "rt-a03":
			text = strings.Replace(text, "ntp server 192.0.2.124\n", "", 1)
		case "rt-a04":
			text = strings.Replace(text, "enable secret 5 $1$dummy$SampleHashForTestsOnly000\n", "enable secret 5 $1$dummy$SampleHashForTestsOnly999\n", 1)
		}
		files["input/old/"+h+".txt"] = text
	}
	e := newCase(t, files)
	if _, err := e.Sites("book", at); err == nil || !strings.Contains(err.Error(), "パラメータブック") {
		t.Fatalf("no book: %v", err)
	}
	if _, err := e.Sites("other", at); err == nil {
		t.Fatal("an unknown source gave no error")
	}
	o, err := e.Sites("old", at)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Devices) != 4 || len(o.Failed) != 0 {
		t.Fatalf("devices %v, failed %v", o.Devices, o.Failed)
	}

	// One device says something else: the others are blank.
	it := siteItemOf(t, o, "interfaces", "", "説明")
	var lan *SiteItem
	for i := range o.Tables {
		for j := range o.Tables[i].Items {
			x := &o.Tables[i].Items[j]
			if x.Column == "説明" && x.Common == "sample lan" {
				lan = x
			}
		}
	}
	if lan == nil {
		t.Fatalf("no item with the common value of the description (first: %+v)", it)
	}
	if lan.Split || lan.CommonN != 3 || lan.Of != 4 || lan.Differ != 1 || strings.Join(lan.Values, "|") != "|sample lan 2||" {
		t.Errorf("the description of one device: %+v", lan)
	}
	// Every device says something else: all values are written, and it is
	// no outlier.
	var loop *SiteItem
	for i := range o.Tables {
		for j := range o.Tables[i].Items {
			x := &o.Tables[i].Items[j]
			if x.Column == "説明" && x.Split {
				loop = x
			}
		}
	}
	if loop == nil || loop.Common != siteSplit || loop.Values[0] != "loopback of rt-a01" || loop.Values[3] != "loopback of rt-a04" {
		t.Errorf("the description every device has its own of: %+v", loop)
	}
	// A row one device lacks.
	row := siteItemOf(t, o, "ntp_servers", "192.0.2.124", siteRow)
	if row.Common != siteHas || row.CommonN != 3 || strings.Join(row.Values, "|") != "||"+siteHasNot+"|" {
		t.Errorf("the row one device lacks: %+v", row)
	}
	// A secret: compared by its value, shown masked.
	e.Proj.Config.Secrets.MaskInReports = true
	o, err = e.Sites("old", at)
	if err != nil {
		t.Fatal(err)
	}
	var secret *SiteItem
	for i := range o.Tables {
		for j := range o.Tables[i].Items {
			if x := &o.Tables[i].Items[j]; strings.Join(x.Values, "|") == "|||"+siteOther {
				secret = x
			}
		}
	}
	if secret == nil || secret.Common != masked {
		t.Fatalf("the secret one device has differently: %+v", secret)
	}
	var outliers []string
	for _, x := range o.Outliers {
		outliers = append(outliers, x.Device+" "+x.Column+" "+x.Value)
	}
	got := strings.Join(outliers, "\n")
	for _, want := range []string{"rt-a02 説明 sample lan 2", "rt-a03 " + siteRow + " " + siteHasNot, "rt-a04 " + secret.Column + " " + siteOther} {
		if !strings.Contains(got, want) {
			t.Errorf("the outliers lack %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "loopback of") {
		t.Errorf("a value every device has its own of is an outlier:\n%s", got)
	}
	html, err := os.ReadFile(e.Proj.Path(o.Report.HTML))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(html), "SampleHashForTestsOnly") {
		t.Error("the report shows a secret")
	}
	for _, want := range []string{"外れ値", "sample lan 2", siteSplit, "rt-a04"} {
		if !strings.Contains(string(html), want) {
			t.Errorf("the report does not hold %q", want)
		}
	}

	// One device alone has nothing beside it.
	one := newCase(t, map[string]string{"input/old/rt-a01.txt": base})
	if _, err := one.Sites("old", at); err == nil || !strings.Contains(err.Error(), "2 台以上") {
		t.Fatalf("one device: %v", err)
	}
}

// The book and the logs as sources.
func TestSitesBookAndLogs(t *testing.T) {
	e := exportCase(t)
	o, err := e.Sites("book", at)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Devices) != 2 || len(o.Tables) == 0 {
		t.Fatalf("the book: %v, %d tables", o.Devices, len(o.Tables))
	}
	if _, err := e.Sites("logs", at); err == nil || !strings.Contains(err.Error(), "input/logs") {
		t.Fatalf("no log: %v", err)
	}
	gen, err := e.Generate(GenOptions{Dry: true}, at)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(e.Proj.Path("input/logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, d := range gen.Devices {
		log := d.Host + "#show running-config\nBuilding configuration...\n\n" + d.Output.Text + d.Host + "#exit\n"
		if err := writeTestFile(e.Proj.Path("input/logs/"+d.Host+".txt"), []byte(log), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A log that shows no configuration is told and left out.
	if err := writeTestFile(e.Proj.Path("input/logs/zz-state.txt"), []byte("rt-s01#show clock\n10:00:00.000 JST Mon Nov 2 2026\nrt-s01#exit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	o, err = e.Sites("logs", at)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Devices) != 2 || len(o.Failed) != 0 || len(o.Notes) != 1 || !strings.Contains(o.Notes[0], "zz-state.txt") {
		t.Fatalf("the logs: %v, failed %v, notes %v", o.Devices, o.Failed, o.Notes)
	}
}
