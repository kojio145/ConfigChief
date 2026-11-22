package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/kojio145/configchief/internal/project"
)

// The search over the files of a case (spec 14.8.8; the model is PalaTerm's
// ログの grep). The texts are shown as they stand, as in the comparison of
// two files: the tool does not know which words of a log are secret.

// maxHits is how many lines a search gives at most.
const maxHits = 5000

// maxFileLines is how many lines of a file the viewer gives at most.
const maxFileLines = 200000

// SearchScopes are the places a search looks in, in the order of the screen.
var SearchScopes = []SearchScope{
	{ID: "old", Label: "旧コンフィグ", Dir: "input/old"},
	{ID: "new", Label: "新コンフィグ", Dir: "input/new"},
	{ID: "logs", Label: "ログ", Dir: "input/logs"},
	{ID: "extra", Label: "追加コンフィグ", Dir: "extra"},
	{ID: "fill", Label: "ひな形", Dir: "fill"},
	{ID: "out", Label: "作ったコンフィグ", Dir: "out"},
}

// SearchScope is a place a search looks in.
type SearchScope struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Dir   string `json:"dir"`
}

// SearchOptions is what the screen asks for.
type SearchOptions struct {
	Query  string   `json:"query"`
	Regex  bool     `json:"regex"`
	Case   bool     `json:"case"`   // tell capitals from small letters
	Scopes []string `json:"scopes"` // ids of SearchScopes (none = all)
	Out    string   `json:"out"`    // the folder of out/ ("" = the newest)
	Misses bool     `json:"misses"` // list the files without a hit too
}

// SearchHit is a line that matched. Spans are the matches as [from, to) in
// UTF-16 units of Text (what the page counts in).
type SearchHit struct {
	Line  int      `json:"line"`
	Text  string   `json:"text"`
	Spans [][2]int `json:"spans"`
}

// SearchFile is a file with its hits.
type SearchFile struct {
	Rel   string      `json:"rel"`   // from the case folder, with "/"
	Scope string      `json:"scope"` // the label of its place
	Hits  []SearchHit `json:"hits"`
}

// SearchView is the outcome of a search (or, with no query, the places and
// the folders of out/ to choose from).
type SearchView struct {
	Scopes []SearchScope `json:"scopes"`
	Outs   []string      `json:"outs"` // the folders of out/, the newest first
	Out    string        `json:"out"`  // the folder that was looked in ("" = none)
	Files  []SearchFile  `json:"files"`
	Missed []string      `json:"missed"` // the files without a hit (when asked)
	Read   int           `json:"read"`   // files read
	Hits   int           `json:"hits"`
	Capped bool          `json:"capped"` // more than maxHits lines matched
	Errors []string      `json:"errors"` // files that could not be read
}

// Search looks for a text in the files of the open case.
func (s *Session) Search(o SearchOptions) (*SearchView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil {
		return nil, errors.New("案件を開いていません")
	}
	root := s.env.Proj.Dir
	v := &SearchView{Scopes: SearchScopes, Outs: outFolders(root), Files: []SearchFile{}, Missed: []string{}, Errors: []string{}}
	if strings.TrimSpace(o.Query) == "" {
		return v, nil
	}
	match, err := matcher(o)
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, id := range o.Scopes {
		want[id] = true
	}
	for _, sc := range SearchScopes {
		if len(want) > 0 && !want[sc.ID] {
			continue
		}
		var files []string
		if sc.ID == "out" {
			out := o.Out
			if out == "" && len(v.Outs) > 0 {
				out = v.Outs[0]
			}
			if out == "" || strings.ContainsAny(out, `/\:`) || out == ".." {
				continue
			}
			v.Out = out
			files = deepFiles(root, "out/"+out+"/configs")
			files = append(files, deepFiles(root, "out/"+out+"/fill")...)
		} else {
			files = flatFiles(root, sc.Dir)
		}
		for _, rel := range files {
			text, err := project.ReadText(filepath.Join(root, filepath.FromSlash(rel)))
			if err != nil {
				v.Errors = append(v.Errors, rel+": "+err.Error())
				continue
			}
			v.Read++
			f := SearchFile{Rel: rel, Scope: sc.Label, Hits: []SearchHit{}}
			for i, line := range strings.Split(text, "\n") {
				line = strings.TrimRight(line, "\r")
				spans := match(line)
				if len(spans) == 0 {
					continue
				}
				v.Hits++
				if v.Hits > maxHits {
					v.Capped = true
					break
				}
				f.Hits = append(f.Hits, SearchHit{Line: i + 1, Text: line, Spans: spans})
			}
			if len(f.Hits) > 0 {
				v.Files = append(v.Files, f)
			} else if o.Misses && !v.Capped {
				v.Missed = append(v.Missed, rel)
			}
			if v.Capped {
				v.Hits = maxHits
				return v, nil
			}
		}
	}
	return v, nil
}

// matcher gives the matches of a line as UTF-16 spans.
func matcher(o SearchOptions) (func(string) [][2]int, error) {
	pat := o.Query
	if !o.Regex {
		pat = regexp.QuoteMeta(pat)
	}
	if !o.Case {
		pat = "(?i)" + pat
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return nil, fmt.Errorf("正規表現として読めません: %v", err)
	}
	return func(line string) [][2]int {
		locs := re.FindAllStringIndex(line, 50)
		var out [][2]int
		for _, l := range locs {
			if l[1] == l[0] {
				continue // an empty match marks nothing
			}
			out = append(out, [2]int{u16len(line[:l[0]]), u16len(line[:l[1]])})
		}
		return out
	}, nil
}

func u16len(s string) int { return len(utf16.Encode([]rune(s))) }

// readable tells a file a search reads (spec 4.2: not one starting with "."
// or "~$", not a book).
func readable(name string) bool {
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "~$") {
		return false
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".xlsx", ".xls", ".xlsm", ".zip", ".exe", ".png", ".jpg", ".pdf":
		return false
	}
	return true
}

// flatFiles are the files right under dir (from the case folder), by name.
func flatFiles(root, dir string) []string {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && readable(e.Name()) {
			out = append(out, dir+"/"+e.Name())
		}
	}
	return out
}

// deepFiles are the files under dir and its folders, by path.
func deepFiles(root, dir string) []string {
	base := filepath.Join(root, filepath.FromSlash(dir))
	var out []string
	filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !readable(d.Name()) {
			return nil
		}
		if rel, err := filepath.Rel(root, path); err == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// outFolders are the folders of out/, the newest first.
func outFolders(root string) []string {
	entries, err := os.ReadDir(filepath.Join(root, "out"))
	if err != nil {
		return []string{}
	}
	out := []string{}
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(out)))
	return out
}

// TextView is a text file of the case, line by line, as it stands.
type TextView struct {
	Rel   string   `json:"rel"`
	Lines []string `json:"lines"`
	Cut   bool     `json:"cut"` // more than maxFileLines lines: the rest is not given
}

// ReadFile gives a text file of the open case to read (spec 14.8.8). rel is
// from the case folder, with "/".
func (s *Session) ReadFile(rel string) (*TextView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil {
		return nil, errors.New("案件を開いていません")
	}
	if rel == "" || strings.ContainsAny(rel, `\:`) || strings.HasPrefix(rel, "/") {
		return nil, fmt.Errorf("%s は開けません（案件フォルダの中の場所ではありません）", rel)
	}
	for _, part := range strings.Split(rel, "/") {
		if part == ".." {
			return nil, fmt.Errorf("%s は開けません（案件フォルダの中の場所ではありません）", rel)
		}
	}
	text, err := project.ReadText(s.env.Proj.Path(rel))
	if err != nil {
		return nil, fmt.Errorf("%s: %v", rel, err)
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	v := &TextView{Rel: rel, Lines: lines}
	if len(lines) > maxFileLines {
		v.Lines, v.Cut = lines[:maxFileLines], true
	}
	return v, nil
}
