package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// The mark of a case folder in use (spec 14.8.2) and the newest tool that
// opened it (spec 14.8.3).

// LockFile is the mark a window or a command holds while the case is open.
const LockFile = "state/inuse.lock"

// ToolFile records the newest version of ConfigChief that opened the case.
const ToolFile = "state/tool.yaml"

// ErrInUse is a case held by another ConfigChief; the error's text names
// the holder when it can be read.
var ErrInUse = errors.New("in use")

// InUseError is ErrInUse with the holder: "PC|user|pid|time".
type InUseError struct{ Holder string }

func (e *InUseError) Error() string {
	return "この案件は別の場所で開かれています" + holderText(e.Holder) + "。同じ案件を 2 か所で開くと、あとから保存した側が上書きします。先に開いている ConfigChief で案件を閉じてから開いてください"
}

func (e *InUseError) Is(target error) bool { return target == ErrInUse }

func holderText(line string) string {
	f := strings.Split(line, "|")
	if len(f) < 4 || f[0] == "" {
		return ""
	}
	return fmt.Sprintf("（PC: %s、ユーザー: %s、%s から）", f[0], f[1], f[3])
}

// Held is a case's mark held by this process. Release empties and lets it go.
type Held struct{ h lockHandle }

// Release empties the mark (the PC and the user are for the refusal while the
// case is open, not for whoever gets the folder next) and lets it go. Safe on
// nil.
func (l *Held) Release() {
	if l == nil {
		return
	}
	releaseLock(l.h)
	l.h = noLock
}

// Lock takes the mark of the case folder dir. A case held elsewhere gives an
// *InUseError (errors.Is ErrInUse). A folder that cannot hold the mark
// (read-only, not Windows) gives a nil Held and no error: the case opens
// without it.
func Lock(dir string) (*Held, error) {
	path := filepath.Join(dir, filepath.FromSlash(LockFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil
	}
	h, holder, err := acquireLock(path)
	if err != nil {
		if errors.Is(err, ErrInUse) {
			return nil, &InUseError{Holder: holder}
		}
		return nil, nil
	}
	if h == noLock {
		return nil, nil
	}
	return &Held{h: h}, nil
}

type toolRecord struct {
	Version string `yaml:"version"`
}

// ToolVersion is the version recorded in state/tool.yaml ("" = none).
func ToolVersion(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(ToolFile)))
	if err != nil {
		return ""
	}
	var r toolRecord
	if yaml.Unmarshal(data, &r) != nil {
		return ""
	}
	return strings.TrimSpace(r.Version)
}

// IsRelease tells a version of a public build (not "0.0.0-dev", not empty).
func IsRelease(v string) bool {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	return v != "" && !strings.Contains(v, "-") && versionParts(v) != [3]int{}
}

// NoteTool records version in state/tool.yaml when it is a public build at
// least as new as the one recorded, and tells the version recorded when it
// is newer than this one ("" = this tool may write the case).
func NoteTool(dir, version string) (newer string) {
	had := ToolVersion(dir)
	if !IsRelease(version) {
		return "" // a development build neither records nor refuses
	}
	if had != "" && VersionNewer(had, version) {
		return had
	}
	if had != strings.TrimPrefix(version, "v") {
		data, _ := yaml.Marshal(toolRecord{Version: strings.TrimPrefix(version, "v")})
		path := filepath.Join(dir, filepath.FromSlash(ToolFile))
		if os.MkdirAll(filepath.Dir(path), 0o755) == nil {
			os.WriteFile(path, data, 0o644)
		}
	}
	return ""
}

// ReadOnlyError is a case used by a newer ConfigChief: this one only reads it.
type ReadOnlyError struct{ Newer, Mine string }

func (e *ReadOnlyError) Error() string {
	return fmt.Sprintf("この案件は v%s で使われています。この版（v%s）では読むだけにしています（新しい版が足した設定を消さないため）。バージョン情報から最新版にしてください",
		strings.TrimPrefix(e.Newer, "v"), strings.TrimPrefix(e.Mine, "v"))
}

func versionParts(v string) [3]int {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, " -+"); i >= 0 {
		v = v[:i]
	}
	for i, p := range strings.SplitN(v, ".", 3) {
		n := 0
		for _, r := range p {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		out[i] = n
	}
	return out
}

// VersionNewer reports whether a is a later version than b.
func VersionNewer(a, b string) bool {
	x, y := versionParts(a), versionParts(b)
	for i := range x {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}

// holderGoneOn reports that the holder line names a process of this PC that
// is not running. A line from another PC cannot be checked and is trusted.
func holderGoneOn(line, host string, alive func(uint32) bool) bool {
	f := strings.Split(line, "|")
	if len(f) < 4 || host == "" || !strings.EqualFold(f[0], host) {
		return false
	}
	pid, err := strconv.ParseUint(f[2], 10, 32)
	if err != nil {
		return false
	}
	return !alive(uint32(pid))
}
