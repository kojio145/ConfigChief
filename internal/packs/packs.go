package packs

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

// The folders of the bundled packs. A new pack is added here.
//
//go:embed cisco_ios nec_ix
var bundled embed.FS

// IDs lists the bundled packs by the name of their folder, in a fixed order.
func IDs() []string {
	entries, err := fs.ReadDir(bundled, ".")
	if err != nil {
		return nil
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	sort.Strings(ids)
	return ids
}

// FS gives the folder of one bundled pack, in the form pack.Load reads.
func FS(id string) (fs.FS, error) {
	if _, err := fs.Stat(bundled, id+"/pack.yaml"); err != nil {
		return nil, fmt.Errorf("同梱のパック %q はありません", id)
	}
	return fs.Sub(bundled, id)
}
