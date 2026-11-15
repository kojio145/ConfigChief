package textdiff

import (
	"strings"
	"testing"
)

const treeA = `hostname rt-a01
!
interface GigabitEthernet0/0
 description uplink
 ip address 192.0.2.1 255.255.255.0
!
interface GigabitEthernet0/1
 ip address 198.51.100.1 255.255.255.0
 shutdown
!
ip access-list extended ACL-IN
 permit tcp any host 192.0.2.10 eq 22
 permit tcp any host 192.0.2.10 eq 443
 deny ip any any
!
ntp server 203.0.113.1
`

func tree(t *testing.T, a, b string, o TreeOptions) TreeResult {
	t.Helper()
	return Tree(Text(a), Text(b), o)
}

func rowsOf(r TreeResult, op Op) []Row {
	var out []Row
	for _, row := range r.Rows {
		if row.Op == op && !row.Noise {
			out = append(out, row)
		}
	}
	return out
}

func TestTreeSame(t *testing.T) {
	r := tree(t, treeA, treeA, TreeOptions{})
	if r.Differs() || r.Same == 0 {
		t.Fatalf("the same text differs: %+v", r)
	}
}

// A line put into one block changes that block alone, and a changed value
// is one changed row under its heading.
func TestTreeChangedUnderParent(t *testing.T) {
	b := strings.Replace(treeA, " description uplink\n", " description uplink\n mtu 1400\n", 1)
	b = strings.Replace(b, "198.51.100.1 255", "198.51.100.2 255", 1)
	r := tree(t, treeA, b, TreeOptions{})
	if r.Added != 1 || r.Removed != 0 || r.Changed != 1 || r.Moved != 0 {
		t.Fatalf("added %d removed %d changed %d moved %d", r.Added, r.Removed, r.Changed, r.Moved)
	}
	found := 0
	r.Walk(func(_ int, row Row, parents []Row) {
		switch {
		case row.Op == Add:
			found++
			if len(parents) != 1 || strings.TrimSpace(parents[0].B) != "interface GigabitEthernet0/0" {
				t.Errorf("the added line stands below %v", parents)
			}
		case row.Op == Chg:
			found++
			if len(parents) != 1 || strings.TrimSpace(parents[0].B) != "interface GigabitEthernet0/1" {
				t.Errorf("the changed line stands below %v", parents)
			}
			if len(row.BSpans) != 1 || row.B[row.BSpans[0][0]:row.BSpans[0][1]] != "2" {
				t.Errorf("the mark of the changed line: %v in %q", row.BSpans, row.B)
			}
			if row.ALine != 8 || row.BLine != 9 {
				t.Errorf("lines %d and %d", row.ALine, row.BLine)
			}
		}
	})
	if found != 2 {
		t.Fatalf("%d rows that differ", found)
	}
}

// A block at another place is the same block, moved; an entry of an access
// list at another place is a moved line. Neither counts as added or removed.
func TestTreeMoved(t *testing.T) {
	b := `hostname rt-a01
!
interface GigabitEthernet0/1
 ip address 198.51.100.1 255.255.255.0
 shutdown
!
interface GigabitEthernet0/0
 description uplink
 ip address 192.0.2.1 255.255.255.0
!
ip access-list extended ACL-IN
 permit tcp any host 192.0.2.10 eq 443
 permit tcp any host 192.0.2.10 eq 22
 deny ip any any
!
ntp server 203.0.113.1
`
	r := tree(t, treeA, b, TreeOptions{})
	if r.Added != 0 || r.Removed != 0 || r.Changed != 0 {
		t.Fatalf("added %d removed %d changed %d", r.Added, r.Removed, r.Changed)
	}
	if r.Moved != 2 {
		t.Fatalf("moved %d, want the block and the entry", r.Moved)
	}
	for _, row := range r.Rows {
		if row.Moved && row.Op != Same {
			t.Errorf("a moved row that is not the same: %+v", row)
		}
	}
}

// A block only on one side gives its heading and every line below it.
func TestTreeBlockOnOneSide(t *testing.T) {
	b := treeA + "router ospf 1\n router-id 192.0.2.1\n network 192.0.2.0 0.0.0.255 area 0\n"
	r := tree(t, treeA, b, TreeOptions{})
	if r.Added != 3 || r.Removed != 0 || r.Changed != 0 {
		t.Fatalf("added %d removed %d changed %d", r.Added, r.Removed, r.Changed)
	}
	add := rowsOf(r, Add)
	if !add[0].Head || add[0].Level != 0 || add[1].Level != 1 {
		t.Fatalf("the rows of the block: %+v", add)
	}
}

func TestTreeStamps(t *testing.T) {
	a := "Current configuration : 1200 bytes\n!\n! Last configuration change at 10:00:00 JST Mon Nov 2 2026\nversion 15.9\nntp clock-period 17180000\n" + treeA
	b := "Current configuration : 1300 bytes\n!\n! Last configuration change at 11:30:00 JST Tue Nov 3 2026\nversion 15.9\nntp clock-period 17180555\n" + treeA
	r := tree(t, a, b, TreeOptions{Stamps: true})
	if r.Differs() {
		t.Fatalf("the stamps differ: %+v", rowsOf(r, Chg))
	}
	if r.Stamps != 6 {
		t.Fatalf("%d stamps left out, want 6", r.Stamps)
	}
	if r := tree(t, a, b, TreeOptions{}); r.Changed != 3 {
		t.Fatalf("with the stamps kept: changed %d, want 3", r.Changed)
	}
	// The version is compared.
	if r := tree(t, a, strings.Replace(b, "version 15.9", "version 17.3", 1), TreeOptions{Stamps: true}); r.Changed != 1 {
		t.Fatalf("another version: changed %d", r.Changed)
	}
}

func TestTreeSecrets(t *testing.T) {
	cert := func(hex string) string {
		return "crypto pki certificate chain TP-EXAMPLE\n certificate self-signed 01\n  " + hex + " 0A0B0C0D\n  quit\n"
	}
	a, b := treeA+cert("30820122"), treeA+cert("30820999")
	r := tree(t, a, b, TreeOptions{Secrets: true})
	if r.Differs() {
		t.Fatalf("the certificates differ: %+v", r)
	}
	if r.Secrets != 6 {
		t.Fatalf("%d secret lines, want 6", r.Secrets)
	}
	dim := 0
	for _, row := range r.Rows {
		if row.Noise {
			dim++
			if row.A == "" || row.B == "" {
				t.Errorf("a dimmed row with one side: %+v", row)
			}
		}
	}
	if dim != 3 {
		t.Fatalf("%d dimmed rows, want 3", dim)
	}
	if r := tree(t, a, b, TreeOptions{}); r.Changed != 1 {
		t.Fatalf("with the secrets kept: changed %d", r.Changed)
	}
}

// The comment lines a caller marks are not compared; the lines outside the
// listing are not there at all. The numbers are those of the file.
func TestTreeSource(t *testing.T) {
	a := Source{Lines: []string{"rt-a01#show running-config", "! made by hand", "hostname rt-a01", "!", "ntp server 203.0.113.1", "rt-a01#"}}
	a.Outside = []bool{true, false, false, false, false, true}
	a.Comment = []bool{false, true, false, true, false, false}
	b := Source{Lines: []string{"hostname rt-a01", "ntp server 203.0.113.2"}}
	r := Tree(a, b, TreeOptions{})
	if r.Changed != 1 || r.Added+r.Removed != 0 {
		t.Fatalf("%+v", r)
	}
	if r.Comments != 2 {
		t.Fatalf("%d comment lines", r.Comments)
	}
	chg := rowsOf(r, Chg)[0]
	if chg.ALine != 5 || chg.BLine != 2 {
		t.Fatalf("lines %d and %d", chg.ALine, chg.BLine)
	}
	for _, row := range r.Rows {
		if strings.Contains(row.A, "show running-config") || row.A == "rt-a01#" {
			t.Fatalf("a line outside the listing is shown: %+v", row)
		}
	}
}

// Separators that differ in number do not make lines differ in a block.
func TestTreeBannerAndBlank(t *testing.T) {
	a := "banner motd ^C\n end of the day\n!\n^C\nhostname rt-a01\n"
	b := "banner motd ^C\n end of the day\n!\n^C\n\n\nhostname rt-a02\n"
	r := tree(t, a, b, TreeOptions{})
	if r.Changed != 1 || r.Added+r.Removed != 0 {
		t.Fatalf("%+v", r.Rows)
	}
}
