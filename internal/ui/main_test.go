package ui

import (
	"os"
	"testing"
)

// The tests open one case from several sessions of one process, which the
// mark of a case in use (spec 14.8.2) would refuse; TestCaseInUse turns it on.
func TestMain(m *testing.M) {
	lockCases = false
	os.Exit(m.Run())
}
