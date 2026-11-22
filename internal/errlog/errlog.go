// Package errlog records an internal error (a Go panic) for a bug report
// (spec 14.8.4): %APPDATA%\ConfigChief\error.log, next to the settings. Only
// the time, the version, where it happened, the panic and the stack are
// written: no configuration, no value of the book. The model is PalaTerm's
// panicguard.go (v1.10.8).
package errlog

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"
)

// maxSize is where the log is moved to error.log.old: a log that grows
// forever on a bug in a loop is a bug too.
const maxSize = 1 << 20

var mu sync.Mutex

// Path is the log file ("" = the system names no place). A variable for the
// tests.
var Path = func() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "ConfigChief", "error.log")
}

// Version is written with every record.
var Version = "0.0.0-dev"

// Panic records a recovered panic and gives the message for the user.
func Panic(where string, r any) string {
	mu.Lock()
	defer mu.Unlock()
	p := Path()
	if p != "" {
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if st, err := os.Stat(p); err == nil && st.Size() > maxSize {
			_ = os.Rename(p, p+".old")
		}
		if f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			fmt.Fprintf(f, "%s ConfigChief %s panic in %s: %v\n%s\n", time.Now().Format(time.RFC3339), Version, where, r, debug.Stack())
			f.Close()
		}
	}
	return fmt.Sprintf("内部エラーが発生しました（%s: %v）。%s に記録しました。お手数ですが、この文と記録を添えて知らせてください", where, r, shown(p))
}

func shown(p string) string {
	if p == "" {
		return "error.log"
	}
	return p
}
