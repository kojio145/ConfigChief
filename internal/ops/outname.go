package ops

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kojio145/configchief/internal/project"
)

// The folder of one run under out/ (spec 4). Two runs within the same
// second must not write into one folder (a delivery would then mix the
// configurations of one run with the check results of the other), so a
// run whose folder is already there gets "_2", "_3"… One run asks for its
// folder many times with the same time; it gets the same folder each time.
var (
	outMu    sync.Mutex
	outNames = map[string]string{}
)

// outDir gives "out/<stamp>" (or "out/<stamp>_<n>") for the run at now.
func (e *Env) outDir(now time.Time) string {
	key := e.Proj.Dir + "\x00" + now.Format(time.RFC3339Nano)
	outMu.Lock()
	defer outMu.Unlock()
	if d, ok := outNames[key]; ok {
		return d
	}
	base := "out/" + project.Stamp(now)
	d := base
	for n := 2; ; n++ {
		if _, err := os.Stat(e.Proj.Path(d)); os.IsNotExist(err) {
			break
		}
		d = fmt.Sprintf("%s_%d", base, n)
	}
	outNames[key] = d
	return d
}
