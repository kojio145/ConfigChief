package book

import "testing"

// The columns of 新旧対照表（ポート） the tool writes for people to read
// (spec 14.19.5): a new row gets them, and a row in the book gets them anew
// with the next import, without touching what a person wrote.
func TestIfRefColumns(t *testing.T) {
	sc := defs(t)
	first, _ := mustUpdate(t, nil, sc, "ja", Change{
		Data:    sampleData(),
		Devices: []DeviceMap{{OldHost: "rt-a01", OldVendor: "cisco_ios", NewHost: "rt-b01"}},
		Ifs:     []IfMap{{NewHost: "rt-b01", OldIf: "GigabitEthernet0/0", OldHost: "rt-a01", Now: "192.0.2.1/30"}},
	})
	b := mustRead(t, first, sc)
	if len(b.Ifs) != 1 || b.Ifs[0].OldHost != "rt-a01" || b.Ifs[0].Now != "192.0.2.1/30" {
		t.Fatalf("new row: %+v", b.Ifs)
	}
	// A person writes the new name; the next import has other settings.
	second, sum := mustUpdate(t, first, sc, "ja", Change{
		Ifs: []IfMap{{NewHost: "rt-b01", OldIf: "GigabitEthernet0/0", OldHost: "rt-a01", Now: "192.0.2.5/30 / 無効（shutdown）"}},
	})
	if sum.IfRef != 1 || !sum.Changed() {
		t.Errorf("summary: %+v", sum)
	}
	b = mustRead(t, second, sc)
	if len(b.Ifs) != 1 || b.Ifs[0].Now != "192.0.2.5/30 / 無効（shutdown）" {
		t.Errorf("row written anew: %+v", b.Ifs)
	}
	// The same values again change nothing.
	if _, sum := mustUpdate(t, second, sc, "ja", Change{
		Ifs: []IfMap{{NewHost: "rt-b01", OldIf: "GigabitEthernet0/0", OldHost: "rt-a01", Now: "192.0.2.5/30 / 無効（shutdown）"}},
	}); sum.Changed() {
		t.Errorf("nothing to write: %+v", sum)
	}
}
