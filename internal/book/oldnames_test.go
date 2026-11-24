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

// TestGoneNewOSColumn: a book made before 2026-11-24 has the column 新 OS 版
// in 新旧対照表（機器）(spec 14.17.3). It is not read, not shown in the table
// editor and not a free column; the other columns are read as before.
func TestGoneNewOSColumn(t *testing.T) {
	sc := defs(t)
	first, _ := mustUpdate(t, nil, sc, "ja", Change{
		Data:    sampleData(),
		Devices: []DeviceMap{{OldHost: "rt-a01", OldVendor: "cisco_ios", NewHost: "rt-b01", NewModel: "ix-4ge", Note: "sample note"}},
	})
	sheet := "新旧対照表（機器）"
	old := edit(t, first, func(f *excelize.File) {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) == 0 {
			t.Fatal(err)
		}
		if slices.Contains(rows[0], "新 OS 版") {
			t.Fatal("a new book has the column 新 OS 版")
		}
		at := len(rows[0]) // 備考 is the last column: put 新 OS 版 before it
		col, _ := excelize.ColumnNumberToName(at)
		if err := f.InsertCols(sheet, col, 1); err != nil {
			t.Fatal(err)
		}
		cell, _ := excelize.CoordinatesToCellName(at, 1)
		f.SetCellValue(sheet, cell, "新 OS 版")
		cell, _ = excelize.CoordinatesToCellName(at, 2)
		f.SetCellValue(sheet, cell, "9.9.9")
	})
	b := mustRead(t, old, sc)
	if len(b.Problems) != 0 {
		t.Errorf("problems = %v", b.Problems)
	}
	if len(b.Devices) != 1 || b.Devices[0].NewModel != "ix-4ge" || b.Devices[0].Note != "sample note" {
		t.Fatalf("devices = %+v", b.Devices)
	}
	sheets, err := ReadSheets(old, editFile, sc, "ja")
	if err != nil {
		t.Fatal(err)
	}
	for _, es := range sheets {
		if es.ID != "@devices" {
			continue
		}
		for _, c := range es.Columns {
			if c.Title == "新 OS 版" {
				t.Errorf("the table editor shows 新 OS 版: %+v", es.Columns)
			}
		}
	}
	second, _ := mustUpdate(t, old, sc, "ja", Change{Data: sampleData(), Devices: []DeviceMap{{OldHost: "rt-a02", OldVendor: "cisco_ios", Note: "second note"}}})
	b = mustRead(t, second, sc)
	if len(b.Devices) != 2 || b.Devices[1].Note != "second note" {
		t.Errorf("devices = %+v", b.Devices)
	}
}

// TestNotCarriedOver: a new host name "-" says the old device is not carried
// over (spec 14.18.1). It reads as an empty new host name that is decided,
// and is written back as "-".
func TestNotCarriedOver(t *testing.T) {
	sc := defs(t)
	data, _ := mustUpdate(t, nil, sc, "ja", Change{
		Data:    sampleData(),
		Devices: []DeviceMap{{OldHost: "rt-a01", OldVendor: "cisco_ios", Out: true}, {OldHost: "rt-a02", OldVendor: "cisco_ios"}},
	})
	b := mustRead(t, data, sc)
	if len(b.Devices) != 2 || !b.Devices[0].Out || b.Devices[0].NewHost != "" || b.Devices[1].Out {
		t.Fatalf("devices = %+v", b.Devices)
	}
}
