package bulk

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestConfigs(t *testing.T) {
	files, err := Configs(300, 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 300 {
		t.Fatalf("%d files, want 300", len(files))
	}
	again, _ := Configs(300, 500)
	if !reflect.DeepEqual(files, again) {
		t.Error("the same call gave other files")
	}
	// Only the addresses kept for tests of network gear (appendix B 11.1),
	// apart from the masks and the default route.
	addr := regexp.MustCompile(`\b\d+\.\d+\.\d+\.\d+\b`)
	hosts, packs := map[string]bool{}, map[string]int{}
	for _, f := range files {
		if n := strings.Count(f.Text, "\n"); n != 500 {
			t.Errorf("%s has %d lines, want 500", f.Name, n)
		}
		if hosts[f.Host] || f.Name != f.Host+".txt" || !strings.Contains(f.Text, "hostname "+f.Host+"\n") {
			t.Errorf("the host name %s (file %s) is twice there, or not in its file", f.Host, f.Name)
		}
		hosts[f.Host] = true
		packs[f.Pack]++
		for _, a := range addr.FindAllString(f.Text, -1) {
			if !strings.HasPrefix(a, "198.18.") && !strings.HasPrefix(a, "198.19.") &&
				!strings.HasPrefix(a, "255.255.255.") && !strings.HasPrefix(a, "0.0.0.") {
				t.Fatalf("%s holds the address %s", f.Name, a)
			}
		}
	}
	if packs["cisco_ios"] != 150 || packs["nec_ix"] != 150 {
		t.Errorf("the packs: %v", packs)
	}
	// The two devices of a site share their LANs.
	if a, b := files[0].Text, files[1].Text; !strings.Contains(a, "vrrp 1 ip 198.19.0.1\n") || !strings.Contains(b, "vrrp 1 ip 198.19.0.1\n") {
		t.Error("the first two devices do not share the address of their first LAN")
	}

	// A device is never smaller than its fixed parts.
	small, err := Configs(1, 10)
	if err != nil || strings.Count(small[0].Text, "\n") < 300 {
		t.Errorf("a device of 10 lines: %v, %d lines", err, strings.Count(small[0].Text, "\n"))
	}
	for _, n := range []int{0, -1, Most + 1} {
		if _, err := Configs(n, 500); err == nil {
			t.Errorf("no error for %d devices", n)
		}
	}
	if _, err := Configs(Most, 400); err != nil {
		t.Errorf("%d devices: %v", Most, err)
	}
}
