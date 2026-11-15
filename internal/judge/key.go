package judge

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/kojio145/configchief/internal/project"
)

// All is the range of a judgment for every device.
const All = "*"

// Key tells a group of unhandled lines apart: where the lines come from, the
// pack that read them, the shapes of the lines above (the outermost first)
// and the shape of the line.
type Key struct {
	Source string // "old" | "new"
	Pack   string
	Path   []string
	Shape  string
}

// KeyOf gives the group a judgment is for.
func KeyOf(j project.Judgment) Key {
	src := j.Source
	if src == "" {
		src = "old"
	}
	return Key{Source: src, Pack: j.Pack, Path: j.Path, Shape: j.Shape}
}

// Same tells whether two keys are the same group.
func (k Key) Same(o Key) bool {
	if k.Source != o.Source || k.Pack != o.Pack || k.Shape != o.Shape || len(k.Path) != len(o.Path) {
		return false
	}
	for i := range k.Path {
		if k.Path[i] != o.Path[i] {
			return false
		}
	}
	return true
}

// Hash gives the hexadecimal text the ID of a group is cut from.
func (k Key) Hash() string {
	sum := sha256.Sum256([]byte(k.Source + "\x00" + k.Pack + "\x00" + strings.Join(k.Path, "\x01") + "\x00" + k.Shape))
	return hex.EncodeToString(sum[:])
}

// idLen is how long the ID of a group is when no other group begins the same.
const idLen = 8

// IDs gives each key its ID: the beginning of its hash, as long as it takes
// to tell every key of the list apart (spec 8.5).
func IDs(keys []Key) []string {
	hashes := make([]string, len(keys))
	for i, k := range keys {
		hashes[i] = k.Hash()
	}
	n := idLen
	for ; n < sha256.Size*2; n += 4 {
		seen := map[string]bool{}
		clash := false
		for _, h := range hashes {
			if seen[h[:n]] {
				clash = true
				break
			}
			seen[h[:n]] = true
		}
		if !clash {
			break
		}
	}
	out := make([]string, len(keys))
	for i, h := range hashes {
		out[i] = h[:n]
	}
	return out
}

// Devices splits the range of a judgment into host names. For every device
// it gives all = true.
func Devices(s string) (hosts []string, all bool) {
	if strings.TrimSpace(s) == All {
		return nil, true
	}
	for _, d := range strings.Split(s, ",") {
		if d = strings.TrimSpace(d); d != "" {
			hosts = append(hosts, d)
		}
	}
	return hosts, false
}

func holds(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Find gives the judgment for a device in a group (-1 if there is none), as
// the place in the list. A judgment that names the device comes before one
// for every device; among equals the later one wins (appendix C 3.3). own
// tells that the judgment names the device.
func Find(list []project.Judgment, k Key, device string) (at int, own bool) {
	at = -1
	for i, j := range list {
		if !KeyOf(j).Same(k) {
			continue
		}
		hosts, all := Devices(j.Devices)
		switch {
		case !all && device != "" && holds(hosts, device):
			at, own = i, true
		case all && !own:
			at = i
		}
	}
	return at, own
}

// Set puts a judgment on a group. With no device it is for every device and
// takes the place of the one for every device; the judgments that name a
// device stay. With devices it takes these devices out of the judgments that
// name them and adds one for them.
func Set(list []project.Judgment, k Key, devices []string, j project.Judgment) []project.Judgment {
	j.Source, j.Pack, j.Path, j.Shape = k.Source, k.Pack, k.Path, k.Shape
	j.Devices = All
	if len(devices) > 0 {
		j.Devices = strings.Join(devices, ",")
	}
	out, _ := drop(list, k, devices)
	return append(out, j)
}

// Clear takes the judgment of a group away: the one for every device (no
// device given), or what names the devices. missing tells what was not
// there: "*", or the devices no judgment names.
func Clear(list []project.Judgment, k Key, devices []string) (out []project.Judgment, missing []string) {
	out, found := drop(list, k, devices)
	if len(devices) == 0 {
		if len(found) == 0 {
			missing = []string{All}
		}
		return out, missing
	}
	for _, d := range devices {
		if !holds(found, d) {
			missing = append(missing, d)
		}
	}
	return out, missing
}

// drop gives the list without the judgments of a group for every device (no
// device given) or without the devices in the judgments that name them, and
// what it took out.
func drop(list []project.Judgment, k Key, devices []string) (out []project.Judgment, found []string) {
	for _, j := range list {
		if !KeyOf(j).Same(k) {
			out = append(out, j)
			continue
		}
		hosts, all := Devices(j.Devices)
		switch {
		case all && len(devices) == 0:
			found = append(found, All)
		case all || len(devices) == 0:
			out = append(out, j)
		default:
			var rest []string
			for _, h := range hosts {
				if holds(devices, h) {
					found = append(found, h)
				} else {
					rest = append(rest, h)
				}
			}
			if len(rest) > 0 {
				j.Devices = strings.Join(rest, ",")
				out = append(out, j)
			}
		}
	}
	return out, found
}
