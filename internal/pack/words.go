package pack

import (
	"io/fs"
	"path"
	"strings"
)

// The words a pack knows (spec 8.6): every run of letters in its own files —
// the rules, the templates, the word lists, the messages of the devices, the
// samples of its tests — and in words.txt, where words of the vendor's
// public command reference can be added. A report of unhandled lines that is
// carried out of a site keeps these words and masks every other.

// wordFiles are the kinds of files the words are taken from. The README is
// prose and is left out.
var wordFiles = map[string]bool{".yaml": true, ".j2": true, ".txt": true}

// LoadWords reads the words of the pack from its files.
func LoadWords(fsys fs.FS) (map[string]bool, error) {
	words := map[string]bool{}
	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !wordFiles[path.Ext(name)] {
			return nil
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		text := string(data)
		if name == "words.txt" {
			// "#" starts a note.
			var lines []string
			for _, l := range strings.Split(text, "\n") {
				if i := strings.Index(l, "#"); i >= 0 {
					l = l[:i]
				}
				lines = append(lines, l)
			}
			text = strings.Join(lines, "\n")
		}
		for _, w := range Letters(text) {
			words[w] = true
		}
		return nil
	})
	return words, err
}

// Letters gives the runs of ASCII letters of a text, in lower case.
func Letters(text string) []string {
	var out []string
	start := -1
	for i := 0; i <= len(text); i++ {
		letter := i < len(text) && (text[i] >= 'a' && text[i] <= 'z' || text[i] >= 'A' && text[i] <= 'Z')
		switch {
		case letter && start < 0:
			start = i
		case !letter && start >= 0:
			out = append(out, strings.ToLower(text[start:i]))
			start = -1
		}
	}
	return out
}
