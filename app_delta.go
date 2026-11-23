package main

import "github.com/kojio145/configchief/internal/ui"

// The change commands made from the parameter book (spec 9.8, 14.12).

// Delta tells the sides the configurations now may be taken from and the
// devices of the book.
func (a *App) Delta() (r0 *ui.DeltaView, err error) {
	defer a.guard("Delta", &err)
	return a.s.Delta()
}

// DeltaRun makes the change commands against the configurations now of a
// side (old | new | logs), for the devices given (all when empty).
func (a *App) DeltaRun(side string, devices []string) (r0 *ui.DeltaRunView, err error) {
	defer a.guard("DeltaRun", &err)
	return a.s.DeltaRun(side, devices)
}
