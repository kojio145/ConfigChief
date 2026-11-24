package ops

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// importSigFile keeps what input/<source> held when it was last imported
// (spec 14.19.3).
func importSigFile(source string) string { return "state/imported_" + source + ".txt" }

// inputSig is a line per file of input/<source>: its name, size and time.
func (e *Env) inputSig(source string) (string, error) {
	files, err := e.Proj.Inputs(source)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, f := range files {
		st, err := os.Stat(e.Proj.Path(f))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&sb, "%s\t%d\t%d\n", f, st.Size(), st.ModTime().UnixNano())
	}
	return sb.String(), nil
}

// ImportStale tells whether input/<source> has files that are not imported
// as they are now: files were added, taken away or changed since the last
// import, or there was none (spec 14.19.3). No files is never stale. It
// writes nothing.
func (e *Env) ImportStale(source string) (bool, error) {
	sig, err := e.inputSig(source)
	if err != nil || sig == "" {
		return false, err
	}
	was, err := os.ReadFile(e.Proj.Path(importSigFile(source)))
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return string(was) != sig, nil
}
