package ops

import (
	"strings"
	"syscall"
	"testing"

	"github.com/kojio145/configchief/internal/book"
)

// A book that another program holds open, as Excel does, is not written: the
// save stops, takes no copy and keeps the draft (spec 4.2, 14.6).
func TestEditBookHeldOpen(t *testing.T) {
	e := editCase(t)
	draft(t, e, book.DraftSheet{Sheet: "devices", Added: []book.DraftAdd{{Cells: map[string]string{"hostname": "rt-a02"}}}})
	copies := backups(t, e)
	before := read(t, e, e.Proj.Config.Book)

	name, err := syscall.UTF16PtrFromString(e.Proj.BookPath())
	if err != nil {
		t.Fatal(err)
	}
	// Others may read the file, as with Excel, but not write it.
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ, syscall.FILE_SHARE_READ, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.SaveEdits(nil, at)
	syscall.CloseHandle(h)
	if err == nil || !strings.Contains(err.Error(), "Excel を閉じてください") || !strings.Contains(err.Error(), "下書きは残っています") {
		t.Fatalf("err = %v", err)
	}
	if e.DraftCount() != 1 || backups(t, e) != copies || read(t, e, e.Proj.Config.Book) != before {
		t.Errorf("draft %d, backups %d → %d", e.DraftCount(), copies, backups(t, e))
	}
	// Closed: the same save goes through.
	saved, err := e.SaveEdits(nil, at)
	if err != nil || saved.Summary == nil || saved.Summary.Added != 1 || e.DraftCount() != 0 {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
}
