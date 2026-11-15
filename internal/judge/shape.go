// Package judge holds what is common to the judgments of unhandled lines
// (spec 8.4, 8.5): the shape of a line with its values hidden, the key of a
// group of lines, and how a judgment is found, put and taken away.
//
// It knows no vendor: the way a shape is made is the same for every pack.
package judge

import (
	"net/netip"
	"regexp"
	"strconv"
	"strings"
)

// The marks a shape puts where a value was.
const (
	markStr = "<str>"
	markMAC = "<mac>"
	markIP  = "<ip>"
	markIP6 = "<ip6>"
	markHex = "<hex>"
	markN   = "<n>"
)

var (
	reQuoted = regexp.MustCompile(`"[^"]*"`)
	reMAC    = regexp.MustCompile(`(?i)\b(?:[0-9a-f]{4}\.[0-9a-f]{4}\.[0-9a-f]{4}|[0-9a-f]{2}(?::[0-9a-f]{2}){5}|[0-9a-f]{2}(?:-[0-9a-f]{2}){5})\b`)
	reHex    = regexp.MustCompile(`^[0-9A-Fa-f]{8,}$`)
	reDigits = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)*`)
	reDigit  = regexp.MustCompile(`[0-9]+`)
	// Numbers joined by "/", "," or "-" are one value: a port, a list, a range.
	reJoined = regexp.MustCompile(`<n>(?:[/,-]<n>)+`)
	// Words whose digits are a part of the word and not a value: the names of
	// protocols and methods. Without them "address-family ipv4" and
	// "address-family ipv6" would be one shape.
	reKeep = regexp.MustCompile(`(?i)\b(?:ipv[46](?:cp)?|dot1q|dot1x|dot1ad|dot3ad|802\.1q|802\.1x|802\.1ad|802\.3ad|md5|sha(?:1|2|256|384|512)|3des|aes(?:128|192|256)|l2tp(?:v[23])?|l2vpn|l3vpn|ikev[12]|ospfv[23]|v1|v2c?|v3)\b`)
)

// The first of the runes that stand for the kept words while the digits are
// replaced (the private use area: no configuration holds them).
const keepBase = 0xE000

// macHold stands for a MAC address while the digits are replaced.
var macHold = string(rune(0xF000))

// Shape gives the text of a line with its values hidden (spec 8.5): the white
// space is made one, and quoted texts, addresses, long hexadecimal words and
// numbers become marks. Names and words stay.
func Shape(text string) string {
	s := strings.Join(strings.Fields(text), " ")
	s = reQuoted.ReplaceAllString(s, markStr)
	words := strings.Split(s, " ")
	for i, w := range words {
		words[i] = word(w)
	}
	return reJoined.ReplaceAllString(strings.Join(words, " "), markN)
}

func word(w string) string {
	var kept []string
	w = reKeep.ReplaceAllStringFunc(w, func(m string) string {
		kept = append(kept, m)
		return string(rune(keepBase + len(kept) - 1))
	})
	// An IPv6 address is a whole word; a "/length" behind it is a number.
	head, rest, cut := strings.Cut(w, "/")
	if strings.Count(head, ":") >= 2 {
		if a, err := netip.ParseAddr(head); err == nil && a.Is6() {
			w = markIP6
			if cut {
				w += "/" + digits(rest)
			}
			return restore(w, kept)
		}
	}
	w = reMAC.ReplaceAllString(w, macHold)
	if reHex.MatchString(w) && strings.ContainsAny(w, "0123456789") && strings.ContainsAny(w, "abcdefABCDEF") {
		return markHex
	}
	return restore(digits(w), kept)
}

// digits replaces the runs of digits: an IPv4 address by its mark, any other
// run by the mark of a number.
func digits(w string) string {
	return reDigits.ReplaceAllStringFunc(w, func(m string) string {
		if isIPv4(m) {
			return markIP
		}
		return reDigit.ReplaceAllString(m, markN)
	})
}

func isIPv4(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if n, err := strconv.Atoi(p); err != nil || n > 255 || len(p) > 3 {
			return false
		}
	}
	return true
}

// restore puts back what was set aside while the digits were replaced.
func restore(w string, kept []string) string {
	w = strings.ReplaceAll(w, macHold, markMAC)
	for i, k := range kept {
		w = strings.Replace(w, string(rune(keepBase+i)), k, 1)
	}
	return w
}
