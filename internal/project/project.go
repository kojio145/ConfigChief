package project

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Config is project.yaml (spec 4.1).
type Config struct {
	Version int    `yaml:"version"`
	Name    string `yaml:"name"`
	Kind    string `yaml:"kind"` // "migration" | "new"
	Book    string `yaml:"book"`
	Output  struct {
		Filename string `yaml:"filename"`
		Encoding string `yaml:"encoding"`
		Newline  string `yaml:"newline"`
	} `yaml:"output"`
	Secrets struct {
		MaskInReports    bool `yaml:"mask_in_reports"`
		DecodeReversible bool `yaml:"decode_reversible"`
	} `yaml:"secrets"`
	Checks struct {
		Disable  []string          `yaml:"disable"`
		Severity map[string]string `yaml:"severity"`
		// Placeholders are the words that stand for "not decided yet" in a
		// secret (V-218); empty = the words of the tool.
		Placeholders []string `yaml:"placeholders"`
	} `yaml:"checks"`
	Export struct {
		// KeepWords are the words a report of unhandled lines that leaves
		// the site keeps besides those of the pack (spec 8.6).
		KeepWords []string `yaml:"keep_words"`
	} `yaml:"export"`
	Language string `yaml:"language"`
}

// Project is an open case folder.
type Project struct {
	Dir    string
	Config Config
}

// ConfigFile is the name of the settings file of a case folder.
const ConfigFile = "project.yaml"

// Folders are the folders of a case folder (spec 4). The tool writes under
// state and out only; the rest is for the user to fill.
var Folders = []string{
	"input/old", "input/new", "input/customer", "input/logs",
	"extra", "mappings", "fill", "vars", "rules", "templates", "packs",
	"state/snapshots", "state/backup", "out",
}

const configTemplate = `version: 1
name: %s
kind: %s            # migration（更改）| new（新規）
book: parameters.xlsx
output:
  filename: "{{ device.hostname }}.txt"
  encoding: ""             # 空 = メーカーパックの既定
  newline: ""              # 空 = メーカーパックの既定（crlf | lf）
secrets:
  mask_in_reports: true    # 出力物で秘密の値を伏せる
  decode_reversible: false # 可逆な難読化を復号して取り込むか
checks:
  disable: []              # 止めるチェックの ID
  severity: {}             # 重大度の上書き（例: {V-208: warning}）
  placeholders: []         # 秘密の値の「仮の値」とみなす語（空 = ツールの既定）
export:
  keep_words: []           # 未対応行の持ち出し用レポートで、伏せずに残す語
language: ja
`

// Init makes a case folder: the folders and project.yaml. A folder that
// already is a case folder is left alone.
func Init(dir, name, kind string) (*Project, error) {
	if kind == "" {
		kind = "migration"
	}
	if kind != "migration" && kind != "new" {
		return nil, fmt.Errorf("案件の種類「%s」は使えません（migration か new）", kind)
	}
	if _, err := os.Stat(filepath.Join(dir, ConfigFile)); err == nil {
		return nil, fmt.Errorf("%s にはすでに %s があります（案件フォルダは作り直しません）", dir, ConfigFile)
	}
	if name == "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
		name = filepath.Base(abs)
	}
	for _, f := range Folders {
		if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(f)), 0o755); err != nil {
			return nil, fmt.Errorf("フォルダを作れません: %v", err)
		}
	}
	quoted, err := yaml.Marshal(name)
	if err != nil {
		return nil, err
	}
	text := fmt.Sprintf(configTemplate, strings.TrimSpace(string(quoted)), kind)
	if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(text), 0o644); err != nil {
		return nil, fmt.Errorf("%s を書けません: %v", ConfigFile, err)
	}
	return Open(dir)
}

// Open reads project.yaml of a case folder.
func Open(dir string) (*Project, error) {
	path := filepath.Join(dir, ConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s は案件フォルダではありません（%s がありません）", dir, ConfigFile)
		}
		return nil, fmt.Errorf("%s を読めません: %v", path, err)
	}
	p := &Project{Dir: dir}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&p.Config); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	c := &p.Config
	var problems []string
	if c.Version != 1 {
		problems = append(problems, fmt.Sprintf("version は 1 と書きます（今は %d）", c.Version))
	}
	if c.Kind == "" {
		c.Kind = "migration"
	}
	if c.Kind != "migration" && c.Kind != "new" {
		problems = append(problems, fmt.Sprintf("kind「%s」は使えません（migration か new）", c.Kind))
	}
	if c.Book == "" {
		c.Book = "parameters.xlsx"
	}
	if c.Book != filepath.Base(c.Book) || !strings.EqualFold(filepath.Ext(c.Book), ".xlsx") {
		problems = append(problems, fmt.Sprintf("book「%s」は、案件フォルダの直下の .xlsx のファイル名で書きます", c.Book))
	}
	if c.Language == "" {
		c.Language = "ja"
	}
	if c.Language != "ja" && c.Language != "en" {
		problems = append(problems, fmt.Sprintf("language「%s」は使えません（ja か en）", c.Language))
	}
	switch c.Output.Newline {
	case "", "crlf", "lf":
	default:
		problems = append(problems, fmt.Sprintf("output.newline「%s」は使えません（crlf か lf、または空）", c.Output.Newline))
	}
	if !KnownEncoding(c.Output.Encoding) {
		problems = append(problems, fmt.Sprintf("output.encoding「%s」は使えません（utf-8、utf-8-bom、shift_jis、または空）", c.Output.Encoding))
	}
	for id, s := range c.Checks.Severity {
		if s != "error" && s != "warning" && s != "info" {
			problems = append(problems, fmt.Sprintf("checks.severity の %s:「%s」は使えません（error、warning、info）", id, s))
		}
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("%s:\n  %s", path, strings.Join(problems, "\n  "))
	}
	return p, nil
}

// Path gives the path of a file of the case folder; rel is written with "/".
func (p *Project) Path(rel string) string {
	return filepath.Join(p.Dir, filepath.FromSlash(rel))
}

// BookPath is the path of the parameter book.
func (p *Project) BookPath() string { return filepath.Join(p.Dir, p.Config.Book) }

// Inputs lists the files right under input/<source>, by name, as paths from
// the case folder written with "/". Files whose name begins with "." or "~$"
// are left out (spec 4.2).
func (p *Project) Inputs(source string) ([]string, error) {
	rel := "input/" + source
	entries, err := os.ReadDir(p.Path(rel))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("%s を読めません: %v", rel, err)
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "~$") {
			continue
		}
		out = append(out, rel+"/"+name)
	}
	sort.Strings(out)
	return out, nil
}

// Log adds a line to state/log.txt. The caller keeps secret values out of
// the message. A log that cannot be written does not stop the work.
func (p *Project) Log(now time.Time, format string, a ...any) {
	path := p.Path("state/log.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	msg := strings.ReplaceAll(fmt.Sprintf(format, a...), "\n", " ")
	fmt.Fprintf(f, "%s %s\n", now.Format("2006-01-02 15:04:05"), msg)
}

// Stamp is the name of the folder of one run under out/ (spec 4).
func Stamp(now time.Time) string { return now.Format("2006-01-02_150405") }

// Backup copies the parameter book into state/backup before it is written
// over, and gives the path of the copy ("" if there is no book yet).
func (p *Project) Backup(now time.Time) (string, error) {
	data, err := os.ReadFile(p.BookPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("%s を読めません: %v", p.Config.Book, err)
	}
	ext := filepath.Ext(p.Config.Book)
	base := "state/backup/" + strings.TrimSuffix(p.Config.Book, ext) + "_" + Stamp(now)
	rel := base + ext
	// A second copy within the same second gets a number: no copy is lost.
	for n := 2; ; n++ {
		if _, err := os.Stat(p.Path(rel)); err != nil {
			break
		}
		rel = fmt.Sprintf("%s_%d%s", base, n, ext)
	}
	path := p.Path(rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("控えを書けません: %v", err)
	}
	return rel, nil
}

// WriteBook puts the bytes of the parameter book in place. The bytes go to a
// file beside the book first, so that a failure leaves the old book whole; a
// book that Excel holds open cannot be replaced, and nothing is written then
// (spec 4.2).
func (p *Project) WriteBook(data []byte) error {
	path := p.BookPath()
	if f, err := os.OpenFile(path, os.O_RDWR, 0); err == nil {
		f.Close()
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s に書けません。Excel を閉じてください", p.Config.Book)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("%s を書けません: %v", p.Config.Book, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("%s に書けません。Excel を閉じてください", p.Config.Book)
	}
	return nil
}
