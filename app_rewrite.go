package main

import "github.com/kojio145/configchief/internal/ui"

// Rewrite gives the list of the tab 書き換え of the result (spec 13.4):
// the lines of the old configurations with where they went, narrowed to an
// old device ("" = all) and a kind ("" = all: auto, unread, cannot, model,
// changed, unsure, ignored). It reads the result that is shown and writes
// nothing.
func (a *App) Rewrite(device, kind string) (r0 *ui.RewriteView, err error) {
	defer a.guard("Rewrite", &err)
	return a.s.Rewrite(device, kind)
}
