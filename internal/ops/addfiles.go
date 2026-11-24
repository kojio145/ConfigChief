package ops

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/project"
)

// The folders a person can add files to from the screen (spec 4.2, 14.10.1).
var addFolders = map[string]string{
	"old":      "旧コンフィグ",
	"new":      "新コンフィグ",
	"logs":     "ログ",
	"customer": "外部一覧",
}

// maxAddSize is the largest file that is copied.
const maxAddSize = 100 << 20

// Added is what AddInputs did.
type Added struct {
	Folder string   // input/<source>
	Copied []string // the names copied (written over included)
	Same   []string // names that were there with the same content: not copied
	// Clash are names that are there with another content: not copied. A
	// person is asked, and the same files come again with overwrite.
	Clash   []string
	Skipped []string // "<name>: why", the files that were left out
	Backups []string // the copies of the files written over
	// Excel are the Excel files that came for the configurations (old,
	// new): they are not copied; a person picks the columns that are made
	// into files (spec 14.16.4).
	Excel []string
}

// AddInputs copies files a person chose into input/<source> (spec 4.2,
// 14.10.1). It only copies: the files chosen stay as they are, and nothing
// in input/ is deleted or renamed. A folder brings the files right in it.
// A file of the same name with another content is written over only with
// overwrite, and is copied to state/backup first.
func (e *Env) AddInputs(source string, paths []string, overwrite bool, now time.Time) (*Added, error) {
	_, ok := addFolders[source]
	if !ok {
		return nil, fmt.Errorf("「%s」にはファイルを足せません（old・new・logs・customer）", source)
	}
	rel := "input/" + source
	out := newAdded(rel)
	type item struct{ name, path string }
	var items []item
	seen := map[string]bool{}
	add := func(name, path string) {
		key := strings.ToLower(name)
		if seen[key] {
			out.Skipped = append(out.Skipped, name+": 同じ名前のファイルを 2 つ選んでいます（先の 1 つだけを写します）")
			return
		}
		seen[key] = true
		items = append(items, item{name, path})
	}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			out.Skipped = append(out.Skipped, filepath.Base(p)+": 読めません")
			continue
		}
		if !info.IsDir() {
			add(filepath.Base(p), p)
			continue
		}
		// A folder: the files right in it, as input/ is read.
		entries, err := os.ReadDir(p)
		if err != nil {
			out.Skipped = append(out.Skipped, info.Name()+": フォルダを読めません")
			continue
		}
		dirs := 0
		for _, ent := range entries {
			if ent.IsDir() {
				dirs++
				continue
			}
			add(ent.Name(), filepath.Join(p, ent.Name()))
		}
		if dirs > 0 {
			out.Skipped = append(out.Skipped, fmt.Sprintf("%s: 中のフォルダ %d 個は写しません（%s は、フォルダの直下のファイルだけを読みます）", info.Name(), dirs, rel))
		}
	}
	if err := os.MkdirAll(e.Proj.Path(rel), 0o755); err != nil {
		return nil, fmt.Errorf("%s を作れません: %v", rel, err)
	}
	for _, it := range items {
		name := it.name
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "~$") {
			out.Skipped = append(out.Skipped, name+": . か ~$ で始まる名前のファイルは読まないので、写しません")
			continue
		}
		if (source == "old" || source == "new") && IsExcel(name) {
			out.Excel = append(out.Excel, it.path)
			continue
		}
		if source == "customer" {
			if ext := strings.ToLower(filepath.Ext(name)); ext != ".xlsx" && ext != ".xls" && ext != ".csv" {
				out.Skipped = append(out.Skipped, name+": 外部一覧に置けるのは xlsx・xls・csv だけです")
				continue
			}
		}
		info, err := os.Stat(it.path)
		if err != nil {
			out.Skipped = append(out.Skipped, name+": 読めません")
			continue
		}
		if info.Size() > maxAddSize {
			out.Skipped = append(out.Skipped, fmt.Sprintf("%s: 大きすぎます（%d MB。100 MB まで）", name, info.Size()>>20))
			continue
		}
		data, err := os.ReadFile(it.path)
		if err != nil {
			out.Skipped = append(out.Skipped, name+": 読めません")
			continue
		}
		same := false
		if a, err := filepath.Abs(it.path); err == nil {
			if d, err := filepath.Abs(e.Proj.Path(rel + "/" + name)); err == nil && strings.EqualFold(a, d) {
				same = true
			}
		}
		if same {
			out.Same = append(out.Same, name)
			continue
		}
		if err := e.putInput(out, source, name, data, overwrite, now); err != nil {
			return nil, err
		}
	}
	sort.Strings(out.Copied)
	if len(out.Copied) > 0 {
		e.Proj.Log(now, "add: %s に %d ファイルを写しました（上書き %d）", rel, len(out.Copied), len(out.Backups))
	}
	return out, nil
}

func newAdded(rel string) *Added {
	return &Added{Folder: rel, Copied: []string{}, Same: []string{}, Clash: []string{}, Skipped: []string{}, Backups: []string{}, Excel: []string{}}
}

// putInput writes one file into input/<source>: a file of the same content
// is left as it is, another content is written over only with overwrite and
// is copied to state/backup first (spec 14.10.1).
func (e *Env) putInput(out *Added, source, name string, data []byte, overwrite bool, now time.Time) error {
	rel := "input/" + source
	dst := e.Proj.Path(rel + "/" + name)
	if have, err := os.ReadFile(dst); err == nil {
		if bytes.Equal(have, data) {
			out.Same = append(out.Same, name)
			return nil
		}
		if !overwrite {
			out.Clash = append(out.Clash, name)
			return nil
		}
		bk := fmt.Sprintf("state/backup/input_%s_%s_%s", source, project.Stamp(now), name)
		if err := os.MkdirAll(e.Proj.Path("state/backup"), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(e.Proj.Path(bk), have, 0o644); err != nil {
			return fmt.Errorf("%s の控えを書けません: %v", name, err)
		}
		out.Backups = append(out.Backups, bk)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("%s/%s を書けません: %v", rel, name, err)
	}
	out.Copied = append(out.Copied, name)
	return nil
}

// Said tells what AddInputs did, in a few lines.
func (a *Added) Said() string {
	var lines []string
	if len(a.Copied) > 0 {
		lines = append(lines, fmt.Sprintf("%s に %d ファイルを写しました: %s", a.Folder, len(a.Copied), short(a.Copied)))
	}
	if len(a.Backups) > 0 {
		lines = append(lines, fmt.Sprintf("上書きした %d ファイルの元は、state/backup に控えました", len(a.Backups)))
	}
	if len(a.Same) > 0 {
		lines = append(lines, fmt.Sprintf("同じ中身のファイルがあるため、写さなかったもの %d: %s", len(a.Same), short(a.Same)))
	}
	if len(a.Skipped) > 0 {
		lines = append(lines, "写さなかったもの:\n  "+strings.Join(a.Skipped, "\n  "))
	}
	if len(a.Excel) > 0 {
		lines = append(lines, fmt.Sprintf("Excel のファイル %d 個は写さずに、コンフィグにする列を選びます", len(a.Excel)))
	}
	if len(lines) == 0 {
		return "写すファイルはありませんでした"
	}
	return strings.Join(lines, "\n")
}

func short(names []string) string {
	if len(names) <= 5 {
		return strings.Join(names, "、")
	}
	return strings.Join(names[:5], "、") + fmt.Sprintf(" ほか %d", len(names)-5)
}
