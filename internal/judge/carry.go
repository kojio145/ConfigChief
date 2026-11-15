package judge

import (
	"regexp"
	"strings"
)

// The shape of a line for a report that leaves the site (spec 8.6): the
// values hidden as in Shape, and every word hidden that is not known to be a
// word of the commands. What cannot be told apart is hidden.

// The marks of a carried shape, besides those of Shape.
const (
	MarkName   = "<name>"
	MarkSecret = "<secret>"
	MarkText   = "<text>"
)

// A word with one of secretRuns among its runs of letters ("password",
// "key-string", "pre-shared-key") is followed by a secret, a word of
// textWords by what a person wrote: the rest of the line is one mark.
var (
	secretRuns = map[string]bool{"password": true, "secret": true, "key": true, "community": true, "psk": true, "passphrase": true}
	textWords  = map[string]bool{"description": true, "remark": true, "banner": true, "contact": true, "location": true, "comment": true}

	reMark = regexp.MustCompile(`<(?:str|mac|ip6?|hex|n)>`)
)

// Carry gives the shape of a line with its names hidden too. known tells
// whether a run of letters (in lower case) is a word that may stay. hidden
// lists the words that were hidden as names, as they were written (values
// already made marks): they are names of the site, never to be written into
// what leaves it.
func Carry(text string, known func(word string) bool) (shape string, hidden []string) {
	return CarryShape(Shape(text), known)
}

// CarryShape is Carry for a text that is a shape already.
func CarryShape(shaped string, known func(word string) bool) (shape string, hidden []string) {
	words := strings.Fields(shaped)
	var out []string
	for i, w := range words {
		runs, ascii := letterRuns(w)
		secret, text := false, textWords[strings.ToLower(w)]
		for _, r := range runs {
			secret = secret || secretRuns[r]
		}
		ok := ascii
		for _, r := range runs {
			// The words that open a secret or a text are words of commands.
			if !(known(r) || secretRuns[r] || text) {
				ok = false
			}
		}
		if ok {
			out = append(out, w)
		} else {
			out = append(out, MarkName)
			hidden = append(hidden, w)
		}
		if i == len(words)-1 {
			break
		}
		// Whether the word itself stays or not, what follows it is hidden.
		if secret {
			out = append(out, MarkSecret)
			break
		}
		if text {
			out = append(out, MarkText)
			break
		}
	}
	return strings.Join(out, " "), hidden
}

// letterRuns gives the runs of ASCII letters of a word, in lower case, the
// marks of values left out. ascii is false for a word that holds a letter of
// another script: a name.
func letterRuns(w string) (runs []string, ascii bool) {
	rest := reMark.ReplaceAllString(w, " ")
	start := -1
	for i := 0; i <= len(rest); i++ {
		if i < len(rest) && rest[i] >= 0x80 {
			return nil, false
		}
		letter := i < len(rest) && (rest[i] >= 'a' && rest[i] <= 'z' || rest[i] >= 'A' && rest[i] <= 'Z')
		switch {
		case letter && start < 0:
			start = i
		case !letter && start >= 0:
			runs = append(runs, strings.ToLower(rest[start:i]))
			start = -1
		}
	}
	return runs, true
}
