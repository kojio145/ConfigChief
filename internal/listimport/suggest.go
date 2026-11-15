package listimport

import (
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Copied from PalaTerm internal/listimport/suggest.go (see ORIGIN.txt): the
// guess of the header row and the helpers. The words of PalaTerm's device
// fields are left out; the ones of the design data are in draft.go.

// GuessHeaderRow picks the row of the first 30 that looks most like a
// header: the most distinct short texts that are not numbers. A group row
// above the header (the same word over several columns) counts once per
// word, so the row of names below it wins.
func GuessHeaderRow(rows [][]string) int {
	best, bestScore := 1, -1
	for i := 0; i < len(rows) && i < 30; i++ {
		seen := map[string]bool{}
		for _, v := range rows[i] {
			v = foldHeader(v)
			if v == "" || len([]rune(v)) > 40 || isNumberish(v) {
				continue
			}
			seen[v] = true
		}
		score := len(seen)
		// A header has values under it.
		if i+1 >= len(rows) || filled(rows[i+1]) == 0 {
			score /= 2
		}
		if score > bestScore {
			best, bestScore = i+1, score
		}
	}
	return best
}

func filled(row []string) int {
	n := 0
	for _, v := range row {
		if strings.TrimSpace(v) != "" {
			n++
		}
	}
	return n
}

func isNumberish(v string) bool {
	v = strings.NewReplacer("-", "", "/", "", ".", "", ":", "", " ", "").Replace(v)
	if v == "" {
		return true
	}
	_, err := strconv.ParseFloat(v, 64)
	return err == nil
}

func normWord(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(norm.NFKC.String(s)), ""))
}

// reRepeated is the mark ReadTable gives the second and later columns of a
// repeated header ("Router (2)") — only that: "ポート (TCP)" is a header of
// its own.
var reRepeated = regexp.MustCompile(` \(\d+\)$`)

func isRepeated(name string) bool { return reRepeated.MatchString(name) }

func hasAny(w string, words []string) bool {
	for _, x := range words {
		if strings.Contains(w, normWord(x)) {
			return true
		}
	}
	return false
}
