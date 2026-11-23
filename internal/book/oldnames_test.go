package book

import (
	"slices"
	"testing"

	"github.com/xuri/excelize/v2"
)

// TestOldSheetNames: a book made by an older version names the mapping
// sheets 機器対応 and IF 対応 (spec 14.14.1). It is read as it is, and rows
// are added to those sheets; no sheet of the new name is made.
func TestOldSheetNames(t *testing.T) {
	sc := defs(t)
	first, _ := mustUpdate(t, nil, sc, "ja", Change{
		Data:    sampleData(),
		Devices: []DeviceMap{{OldHost: "rt-a01", OldVendor: "cisco_ios", NewHost: "rt-b01"}},
		Ifs:     []IfMap{{NewHost: "rt-b01", OldIf: "GigabitEthernet0/0", NewIf: "GigabitEthernet0/0/0"}},
	})
	old := edit(t, first, func(f *excelize.File) {
		if err := f.SetSheetName("新旧対照表（機器）", "機器対応"); err != nil {
			t.Fatal(err)
		}
		if err := f.SetSheetName("新旧対照表（ポート）", "IF対応"); err != nil {
			t.Fatal(err)
		}
	})
	b := mustRead(t, old, sc)
	if len(b.Problems) != 0 {
		t.Errorf("problems = %v", b.Problems)
	}
	if len(b.Devices) != 1 || b.Devices[0].NewHost != "rt-b01" || len(b.Ifs) != 1 || b.Ifs[0].NewIf != "GigabitEthernet0/0/0" {
		t.Fatalf("devices = %+v, ifs = %+v", b.Devices, b.Ifs)
	}

	second, sum := mustUpdate(t, old, sc, "ja", Change{
		Data:    sampleData(),
		Devices: []DeviceMap{{OldHost: "rt-a02", OldVendor: "cisco_ios"}},
		Ifs:     []IfMap{{NewHost: "rt-b01", OldIf: "GigabitEthernet0/1"}},
	})
	if sum.DeviceRows != 1 || sum.IfRows != 1 {
		t.Errorf("summary = %+v", sum)
	}
	names := sheetNames(t, second)
	if !slices.Contains(names, "機器対応") || !slices.Contains(names, "IF対応") || slices.Contains(names, "新旧対照表（機器）") || slices.Contains(names, "新旧対照表（ポート）") {
		t.Errorf("sheets = %v", names)
	}
	b = mustRead(t, second, sc)
	if len(b.Devices) != 2 || len(b.Ifs) != 2 {
		t.Errorf("devices = %+v, ifs = %+v", b.Devices, b.Ifs)
	}
}
