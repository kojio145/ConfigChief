package listimport

import (
	"net/netip"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/text/unicode/norm"
)

// Copied from PalaTerm internal/listimport/steps.go (see ORIGIN.txt), without
// the steps that guess PalaTerm's OS types, and with the step cidr.

// ApplySteps runs a field's steps over v in order.
func ApplySteps(v string, steps []Step) string {
	for _, s := range steps {
		v = applyStep(v, s)
	}
	return v
}

// TraceSteps is ApplySteps keeping the value after each step (the rule
// editor shows them under each step).
func TraceSteps(v string, steps []Step) []string {
	out := make([]string, 0, len(steps))
	for _, s := range steps {
		v = applyStep(v, s)
		out = append(out, v)
	}
	return out
}

// reCache keeps compiled patterns: a rule runs the same few patterns over
// every row. Each bound call runs on its own goroutine (the step editor
// traces at every key while a preview may be running), hence the lock: a
// map written from two goroutines at once ends the process.
var (
	reMu    sync.Mutex
	reCache = map[string]*regexp.Regexp{}
)

func compile(p string) *regexp.Regexp {
	reMu.Lock()
	defer reMu.Unlock()
	if re, ok := reCache[p]; ok {
		return re
	}
	re, err := regexp.Compile(p)
	if err != nil {
		re = nil
	}
	if len(reCache) > 256 {
		reCache = map[string]*regexp.Regexp{}
	}
	reCache[p] = re
	return re
}

func applyStep(v string, s Step) string {
	switch s.Type {
	case "trim":
		return strings.TrimSpace(v)
	case "lower":
		return strings.ToLower(v)
	case "upper":
		return strings.ToUpper(v)
	case "width":
		return norm.NFKC.String(v)
	case "cidr":
		return cidr(v)
	case "split":
		if s.Sep == "" {
			return v
		}
		i := strings.Index(v, s.Sep)
		if s.Pos == "last" {
			i = strings.LastIndex(v, s.Sep)
		}
		if i < 0 {
			// No separator: the value as it is (core-sw04 alone stays
			// core-sw04), or empty when asked — a cell holding only the
			// user "ops" has no password to give.
			if s.Miss == "empty" {
				return ""
			}
			return v
		}
		if s.Keep == "left" {
			return v[:i]
		}
		return v[i+len(s.Sep):]
	case "replace":
		if s.From == "" {
			return v
		}
		return strings.ReplaceAll(v, s.From, s.To)
	case "regex":
		if re := compileSafe(s.Pattern); re != nil {
			return re.ReplaceAllString(v, s.To)
		}
		return v
	case "map":
		t := strings.TrimSpace(v)
		for _, r := range s.Rows {
			// A row's From is trimmed in either mode: stray spaces around a
			// word in the 置き換え表 must not keep it from matching.
			from := strings.TrimSpace(r.From)
			if from == "" {
				continue
			}
			if r.Match == "contains" {
				if strings.Contains(strings.ToLower(t), strings.ToLower(from)) {
					return r.To
				}
			} else if strings.EqualFold(t, from) {
				return r.To
			}
		}
		if s.Miss == "empty" {
			return ""
		}
		return v
	}
	return v
}

// compileSafe is compile for patterns from a rule file someone else wrote:
// Go's regexp runs in linear time, so only the length is capped.
func compileSafe(p string) *regexp.Regexp {
	if p == "" || len(p) > 500 {
		return nil
	}
	return compile(p)
}

var reMaskSep = regexp.MustCompile(`^(\S+?)(?:\s*/\s*|\s+)(\S+)$`)

// cidr turns an address with a mask ("192.0.2.1 255.255.255.0",
// "192.0.2.1/255.255.255.0") into the address with a prefix length
// ("192.0.2.1/24"). Anything it cannot read that way stays as it is, so that
// the check of the value names the cell.
func cidr(v string) string {
	m := reMaskSep.FindStringSubmatch(strings.TrimSpace(v))
	if m == nil {
		return v
	}
	addr, err := netip.ParseAddr(m[1])
	if err != nil || !addr.Is4() {
		return v
	}
	mask, err := netip.ParseAddr(m[2])
	if err != nil || !mask.Is4() {
		return v
	}
	b := mask.As4()
	bits := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	n := 0
	for n < 32 && bits&(1<<(31-n)) != 0 {
		n++
	}
	if n < 32 && bits<<n != 0 {
		return v // not a run of ones: no mask
	}
	return netip.PrefixFrom(addr, n).String()
}
