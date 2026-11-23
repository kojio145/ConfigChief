package ops

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/project"
	"go.yaml.in/yaml/v3"
)

// Two ways to start a case (spec 14.12.7, 14.12.8): the sample case, and a
// case that takes the settings of one before.

//go:embed sample/*.txt
var sampleFiles embed.FS

// SampleName is the name of the sample case.
const SampleName = "見本の更改（Cisco → NEC IX）"

// MakeSampleCase makes a case folder in dir with the sample configurations
// in input/old: two branch routers of Cisco with switch ports, made up from
// the addresses kept for examples. It gives the number of files.
func MakeSampleCase(dir string) (int, error) {
	if _, err := project.Init(dir, SampleName, "migration"); err != nil {
		return 0, err
	}
	entries, err := fs.ReadDir(sampleFiles, "sample")
	if err != nil {
		return 0, err
	}
	n := 0
	for _, ent := range entries {
		data, err := sampleFiles.ReadFile("sample/" + ent.Name())
		if err != nil {
			return n, err
		}
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		if err := os.WriteFile(filepath.Join(dir, "input", "old", ent.Name()), data, 0o644); err != nil {
			return n, fmt.Errorf("見本のコンフィグを書けません: %v", err)
		}
		n++
	}
	return n, nil
}

// The parts of a case that can be taken into a new one (spec 14.12.8).
const (
	PartTemplates  = "templates"
	PartRules      = "rules"
	PartChecks     = "checks"
	PartExtra      = "extra"       // _all.j2 and @<group>.j2
	PartExtraHosts = "extra_hosts" // <host>.j2
	PartMappings   = "mappings"
	PartFill       = "fill" // fill/ and vars/
	PartPacks      = "packs"
	PartKeepWords  = "keep_words"
)

// CaseParts lists the parts, in the order the screen shows them, with
// their words and whether they are taken unless a person says otherwise.
var CaseParts = []struct {
	ID, Word string
	Default  bool
}{
	{PartTemplates, "納品用シートのひな形（templates）", true},
	{PartRules, "案件のルール（rules）", true},
	{PartChecks, "チェックの設定（project.yaml の checks）", true},
	{PartExtra, "追加コンフィグの全機器・グループの分（extra の _all.j2・@グループ名.j2）", true},
	{PartExtraHosts, "追加コンフィグの機器ごとの分（extra の ホスト名.j2）", false},
	{PartMappings, "外部一覧の列対応（mappings）", true},
	{PartFill, "ひな形と変数ルール（fill・vars）", true},
	{PartPacks, "案件のパック（packs）", true},
	{PartKeepWords, "持ち出し用レポートで残す語（project.yaml の export.keep_words）", true},
}

// CaseCopied is what was taken from the case before.
type CaseCopied struct {
	Files int      // files written
	Parts []string // the words of the parts that had something
}

// CopyCaseSettings takes the settings of the case in from into the case in
// to (just made): the parts asked for. The case before is only read (it may
// be open somewhere else). The inputs, the book, state/ and out/ are never
// taken.
func CopyCaseSettings(from, to string, parts []string) (*CaseCopied, error) {
	src, err := project.Open(from)
	if err != nil {
		return nil, fmt.Errorf("設定を写す元の案件を開けません: %v", err)
	}
	if same, _ := sameDir(from, to); same {
		return nil, fmt.Errorf("設定を写す元の案件と、新しい案件が同じフォルダです")
	}
	want := map[string]bool{}
	for _, p := range parts {
		want[p] = true
	}
	out := &CaseCopied{}
	words := map[string]string{}
	for _, p := range CaseParts {
		words[p.ID] = p.Word
	}
	had := map[string]bool{}
	copyDir := func(part, rel string, take func(name string) bool) error {
		root := src.Path(rel)
		return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if d.IsDir() {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "~$") || (take != nil && !take(name)) {
				return nil
			}
			sub, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("%s を読めません: %v", path, err)
			}
			dst := filepath.Join(to, filepath.FromSlash(rel), sub)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(dst, data, 0o644); err != nil {
				return fmt.Errorf("%s を書けません: %v", dst, err)
			}
			out.Files++
			had[part] = true
			return nil
		})
	}
	shared := func(name string) bool { return name == "_all.j2" || strings.HasPrefix(name, "@") }
	steps := []struct {
		part, rel string
		take      func(string) bool
	}{
		{PartTemplates, "templates", nil},
		{PartRules, "rules", nil},
		{PartExtra, "extra", shared},
		{PartExtraHosts, "extra", func(n string) bool { return !shared(n) }},
		{PartMappings, "mappings", nil},
		{PartFill, "fill", nil},
		{PartFill, "vars", nil},
		{PartPacks, "packs", nil},
	}
	for _, s := range steps {
		if !want[s.part] {
			continue
		}
		if err := copyDir(s.part, s.rel, s.take); err != nil {
			return out, err
		}
	}
	if want[PartChecks] || want[PartKeepWords] {
		took, err := copyProjectParts(src.Path(project.ConfigFile), filepath.Join(to, project.ConfigFile), want[PartChecks], want[PartKeepWords])
		if err != nil {
			return out, err
		}
		for _, p := range took {
			had[p] = true
		}
	}
	for _, p := range CaseParts {
		if had[p.ID] {
			out.Parts = append(out.Parts, p.Word)
		}
	}
	return out, nil
}

// copyProjectParts takes checks and export.keep_words of project.yaml from
// one case into another; the other lines of the new one and their comments
// stay. It tells the parts that had something.
func copyProjectParts(from, to string, checks, keep bool) ([]string, error) {
	read := func(path string) (*yaml.Node, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var doc yaml.Node
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("%s: %v", path, err)
		}
		if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
			return nil, fmt.Errorf("%s: 先頭が設定の並びではありません", path)
		}
		return &doc, nil
	}
	a, err := read(from)
	if err != nil {
		return nil, err
	}
	b, err := read(to)
	if err != nil {
		return nil, err
	}
	var took []string
	if checks {
		if v := mapValue(a.Content[0], "checks"); v != nil && v.Kind == yaml.MappingNode && len(v.Content) > 0 {
			setMap(b.Content[0], "checks", v)
			took = append(took, PartChecks)
		}
	}
	if keep {
		if ex := mapValue(a.Content[0], "export"); ex != nil && ex.Kind == yaml.MappingNode {
			if kw := mapValue(ex, "keep_words"); kw != nil && len(kw.Content) > 0 {
				dst := mapValue(b.Content[0], "export")
				if dst == nil || dst.Kind != yaml.MappingNode {
					dst = &yaml.Node{Kind: yaml.MappingNode}
					setMap(b.Content[0], "export", dst)
					dst = mapValue(b.Content[0], "export")
				}
				setMap(dst, "keep_words", kw)
				took = append(took, PartKeepWords)
			}
		}
	}
	if len(took) == 0 {
		return nil, nil
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(b); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	if err := os.WriteFile(to, buf.Bytes(), 0o644); err != nil {
		return nil, err
	}
	sort.Strings(took)
	return took, nil
}

// sameDir tells whether two paths name the same folder.
func sameDir(a, b string) (bool, error) {
	x, err := filepath.Abs(a)
	if err != nil {
		return false, err
	}
	y, err := filepath.Abs(b)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(filepath.Clean(x), filepath.Clean(y)), nil
}
