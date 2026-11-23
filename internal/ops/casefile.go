package ops

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/casefile"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/project"
)

// The case written out to one file and read back (spec 14.9).

// CaseExport is a case written out.
type CaseExport struct {
	Data      []byte
	Name      string // the name of the file it is offered under
	Files     int    // files put in
	Encrypted bool
	NoSecrets bool
	// With the secrets taken out: how many values were found, how many
	// places of the texts were masked and in which files, and what could not
	// be masked.
	Values      int
	Masked      int
	MaskedFiles []string
	Left        []string
	Notes       []string
}

// Not written out (spec 14.9.1): what the case makes again, the copies of the
// book, the mark of the case in use.
func leftOut(rel string) bool {
	base := rel[strings.LastIndex(rel, "/")+1:]
	switch {
	case rel == "out" || strings.HasPrefix(rel, "out/"),
		rel == "state/backup" || strings.HasPrefix(rel, "state/backup/"),
		rel == project.LockFile,
		strings.HasPrefix(base, "~$"),
		base == ".git":
		return true
	}
	return false
}

// textFile tells a file whose text can hold a secret and be masked.
func textFile(rel string) bool {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".xlsx", ".xls", ".xlsm", ".zip", ".png", ".jpg", ".jpeg", ".gif", ".pdf", ".exe", ".dll":
		return false
	}
	return true
}

// ExportCase writes the open case out to one file (spec 14.9.1). password ""
// leaves it unsealed; noSecrets masks the secret values of the book and of
// the texts.
func (e *Env) ExportCase(noSecrets bool, password string, now time.Time) (*CaseExport, error) {
	root := e.Proj.Dir
	var rels []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." {
			return err
		}
		rel = filepath.ToSlash(rel)
		if leftOut(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil // a link may lead out of the case
		}
		if !d.IsDir() {
			rels = append(rels, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("案件のフォルダを読めません: %v", err)
	}
	out := &CaseExport{Encrypted: password != "", NoSecrets: noSecrets}
	bookRel := filepath.ToSlash(e.Proj.Config.Book)
	var mask *secretSet
	var bookData []byte
	if noSecrets {
		mask = &secretSet{byLine: map[string]map[int][]string{}}
		if data, err := os.ReadFile(e.Proj.BookPath()); err == nil {
			masked, taken, err := book.MaskSecrets(data, filepath.Base(e.Proj.BookPath()), e.Schema, e.Proj.Config.Language)
			if err != nil {
				return nil, fmt.Errorf("パラメータブックの秘密の値を伏せられません: %v", err)
			}
			bookData = masked
			for _, v := range taken {
				mask.add(v, "", 0)
			}
		}
		left := e.collectSecrets(mask)
		out.Left = append(out.Left, left...)
		out.Values = len(mask.all)
	}
	var files []casefile.File
	for _, rel := range rels {
		if noSecrets && rel == "state/draft.yaml" {
			out.Notes = append(out.Notes, "保存していない下書き（state/draft.yaml）は入れていません（秘密の値が入っていることがあるため）")
			continue
		}
		var data []byte
		if noSecrets && rel == bookRel && bookData != nil {
			data = bookData
		} else {
			data, err = os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if err != nil {
				return nil, fmt.Errorf("%s を読めません: %v", rel, err)
			}
		}
		if noSecrets && rel != bookRel {
			if textFile(rel) {
				if m, n := mask.apply(rel, data); n > 0 {
					data = m
					out.Masked += n
					out.MaskedFiles = append(out.MaskedFiles, rel)
				}
			} else if strings.HasPrefix(rel, "input/") || strings.HasPrefix(rel, "templates/") {
				out.Left = append(out.Left, rel+"（Excel のファイルの中は伏せられません）")
			}
		}
		files = append(files, casefile.File{Path: rel, Data: data})
	}
	m := casefile.Manifest{Name: e.Proj.Config.Name, Written: now.Format(time.RFC3339), Version: e.Version, NoSecret: noSecrets}
	out.Data, err = casefile.Pack(m, files, password, now)
	if err != nil {
		return nil, err
	}
	out.Files = len(files)
	out.Name = SafeFileName(e.Proj.Config.Name, "case") + "_" + project.Stamp(now) + ".ccase"
	e.Proj.Log(now, "案件を書き出した（%d ファイル・暗号化 %v・秘密の値を外す %v）", out.Files, out.Encrypted, noSecrets)
	return out, nil
}

// collectSecrets reads the configurations of input/old, input/new and
// input/logs and keeps the values of their secret columns, with the line
// each came from. It gives what it could not look into: the files with
// unhandled lines (a secret there is not known as one).
func (e *Env) collectSecrets(mask *secretSet) (left []string) {
	tick := e.Tick
	e.Tick = nil
	defer func() { e.Tick = tick }()
	var sources []*Source
	for _, name := range []string{"old", "new"} {
		if s, err := e.Import(name, ""); err == nil {
			sources = append(sources, s)
		}
	}
	if s, _, err := e.importLogs(); err == nil {
		sources = append(sources, s)
	}
	for _, s := range sources {
		for _, f := range s.Files {
			if f.Result == nil {
				continue
			}
			if n := f.Result.Count().Unknown; n > 0 {
				left = append(left, fmt.Sprintf("%s（読めなかった行 %d 行。その中の秘密の値は伏せられません）", f.Path, n))
			}
			addSecrets(mask, e, f.Result.Data)
		}
	}
	return left
}

func addSecrets(mask *secretSet, e *Env, ds *model.Dataset) {
	if ds == nil {
		return
	}
	for name, t := range ds.Tables {
		st := e.Schema.Table(name)
		if st == nil {
			continue
		}
		for _, r := range t.Rows {
			for col, c := range r.Cells {
				sc := st.Column(col)
				if sc == nil || !sc.Secret || c == nil {
					continue
				}
				for i, v := range c.Values {
					file, line := "", 0
					if i < len(c.Origins) {
						file, line = c.Origins[i].File, c.Origins[i].Line
					} else if len(c.Origins) > 0 {
						file, line = c.Origins[0].File, c.Origins[0].Line
					}
					mask.add(v, filepath.ToSlash(file), line)
				}
			}
		}
	}
}

// secretSet is the secret values to mask: the long ones wherever they are,
// the short ones (under minMask letters, too likely to be a word of
// something else) only on the line they were read from.
type secretSet struct {
	all    []string
	long   []string
	byLine map[string]map[int][]string // file → line → short values
	seen   map[string]bool
}

const minMask = 4

func (m *secretSet) add(v, file string, line int) {
	v = strings.TrimSpace(v)
	if v == "" || v == book.SecretMask {
		return
	}
	if m.seen == nil {
		m.seen = map[string]bool{}
	}
	if len([]rune(v)) >= minMask {
		if !m.seen[v] {
			m.seen[v] = true
			m.all = append(m.all, v)
			m.long = append(m.long, v)
		}
		return
	}
	if !m.seen[v] {
		m.seen[v] = true
		m.all = append(m.all, v)
	}
	if file == "" || line <= 0 {
		return
	}
	if m.byLine[file] == nil {
		m.byLine[file] = map[int][]string{}
	}
	m.byLine[file][line] = append(m.byLine[file][line], v)
}

// apply masks the values in the text of a file and gives how many places.
func (m *secretSet) apply(rel string, data []byte) ([]byte, int) {
	if len(m.long) == 0 && len(m.byLine[rel]) == 0 {
		return data, 0
	}
	text := string(data)
	n := 0
	long := append([]string{}, m.long...)
	sort.SliceStable(long, func(i, j int) bool { return len(long[i]) > len(long[j]) })
	for _, v := range long {
		if c := strings.Count(text, v); c > 0 {
			n += c
			text = strings.ReplaceAll(text, v, book.SecretMask)
		}
	}
	if short := m.byLine[rel]; len(short) > 0 {
		lines := strings.SplitAfter(text, "\n")
		for no, vals := range short {
			if no-1 >= len(lines) {
				continue
			}
			for _, v := range vals {
				re := regexp.MustCompile(`(^|[\s"'])` + regexp.QuoteMeta(v) + `($|[\s"'\r\n])`)
				l := lines[no-1]
				if re.MatchString(l) {
					n += len(re.FindAllStringIndex(l, -1))
					lines[no-1] = re.ReplaceAllString(l, "${1}"+book.SecretMask+"${2}")
				}
			}
		}
		text = strings.Join(lines, "")
	}
	return []byte(text), n
}

// SafeFileName keeps a name to what a Windows file name may hold (def when
// nothing is left).
func SafeFileName(name, def string) string {
	name = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, strings.TrimSpace(name))
	name = strings.TrimRight(name, ". ")
	if name == "" {
		return def
	}
	return name
}

// CaseImport is a case read back from its file.
type CaseImport struct {
	Dir       string // the case folder made
	Name      string
	Files     int
	NoSecrets bool   // its secrets were taken out when it was written
	Written   string // when it was written
	Version   string // the ConfigChief that wrote it
}

// ImportCase spreads the file of a case under parent, in a new folder named
// after the case (spec 14.9.2). A folder that is there is never written into.
func ImportCase(data []byte, password, parent string) (*CaseImport, error) {
	m, files, err := casefile.Unpack(data, password)
	if err != nil {
		return nil, err
	}
	has := false
	for _, f := range files {
		has = has || f.Path == project.ConfigFile
	}
	if !has {
		return nil, errors.New("ファイルに project.yaml がありません（案件のファイルではありません）")
	}
	if st, err := os.Stat(parent); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("広げる場所のフォルダがありません: %s", parent)
	}
	base := SafeFileName(m.Name, "case")
	dir := filepath.Join(parent, base)
	for i := 2; ; i++ {
		if _, err := os.Lstat(dir); errors.Is(err, fs.ErrNotExist) {
			break
		}
		if i > 999 {
			return nil, fmt.Errorf("%s に同じ名前のフォルダが多すぎます", parent)
		}
		dir = filepath.Join(parent, fmt.Sprintf("%s_%d", base, i))
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		return nil, fmt.Errorf("フォルダを作れません: %v", err)
	}
	for _, f := range files {
		p := filepath.Join(dir, filepath.FromSlash(f.Path))
		if rel, err := filepath.Rel(dir, p); err != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("ファイルの名前 %q は使えません", f.Path)
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(p, f.Data, 0o644); err != nil {
			return nil, fmt.Errorf("%s を書けません: %v", f.Path, err)
		}
	}
	for _, d := range project.Folders {
		os.MkdirAll(filepath.Join(dir, filepath.FromSlash(d)), 0o755)
	}
	return &CaseImport{Dir: dir, Name: m.Name, Files: len(files), NoSecrets: m.NoSecret, Written: m.Written, Version: m.Version}, nil
}
