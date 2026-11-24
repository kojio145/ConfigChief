package ops

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/project"
	"go.yaml.in/yaml/v3"
)

// The whole set of what is delivered, in one folder (spec 13.6), and the
// history of its versions (spec 13.7).

const revisionsFile = "state/revisions.yaml"

// deliveryNames are the names of the folder and the files of the delivery
// set, in the language the case chose (spec 14.15.4). done is written last
// into the set: a set without it was not finished.
type deliveryNames struct {
	prefix, done                      string
	checks, rewrite, changes, peer string
}

var deliveryWords = map[string]deliveryNames{
	"ja": {"納品_", "納品一式の内容.txt", "チェック結果表", "書き換えレポート", "変更一覧", "相手に渡すパラメータ表"},
	"en": {"delivery_", "contents.txt", "check_results", "rewrite_report", "changes", "peer_parameters"},
}

// DeliveryLangs are the languages the names of a delivery set can be in.
var DeliveryLangs = []string{"ja", "en"}

func (e *Env) deliveryNames() deliveryNames {
	if e.Proj.Config.Export.DeliveryNames == "en" {
		return deliveryWords["en"]
	}
	return deliveryWords["ja"]
}

// deliveryDoneOf gives the file that tells a set in the folder name was
// finished ("" = name is not a delivery set).
func deliveryDoneOf(name string) string {
	for _, w := range deliveryWords {
		if strings.HasPrefix(name, w.prefix) {
			return w.done
		}
	}
	return ""
}

// Revision is one version that was delivered (spec 13.7).
type Revision struct {
	Version  string `yaml:"version" json:"version"`
	Date     string `yaml:"date" json:"date"`
	By       string `yaml:"by,omitempty" json:"by"`
	Summary  string `yaml:"summary,omitempty" json:"summary"`
	Snapshot string `yaml:"snapshot,omitempty" json:"snapshot"` // the record of the generation
}

type revisionsDoc struct {
	Version   int        `yaml:"version"`
	Revisions []Revision `yaml:"revisions"`
}

// Revisions reads state/revisions.yaml (none = an empty list).
func (e *Env) Revisions() ([]Revision, error) {
	data, err := os.ReadFile(e.Proj.Path(revisionsFile))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s を読めません: %v", revisionsFile, err)
	}
	var doc revisionsDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s を読めません: %v", revisionsFile, err)
	}
	return doc.Revisions, nil
}

func (e *Env) saveRevisions(list []Revision) error {
	data, err := yaml.Marshal(revisionsDoc{Version: 1, Revisions: list})
	if err != nil {
		return err
	}
	head := []byte("# 改版履歴（納品一式を作るときに書いた版。13.7）\n")
	if err := os.MkdirAll(e.Proj.Path("state"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(e.Proj.Path(revisionsFile), append(head, data...), 0o644)
}

// DeliveryPlan is what the screen shows before the set is made.
type DeliveryPlan struct {
	Ready bool   `json:"ready"`
	Why   string `json:"why"` // why it cannot be made
	// Generation is the output of the newest generation the set takes.
	Generation string `json:"generation"`
	When       string `json:"when"`
	Configs    int    `json:"configs"`
	// Errors and Warnings are the findings not approved at that generation
	// (Known false: an older record that did not keep them).
	Errors   int  `json:"errors"`
	Warnings int  `json:"warnings"`
	Known    bool `json:"known"`
	// The history of the versions, the version proposed next and what
	// changed since the last one.
	Revisions   []Revision `json:"revisions"`
	NextVersion string     `json:"nextVersion"`
	Summary     string     `json:"summary"`
}

// latestGeneration finds the newest output of "generate": the folder that
// holds the configurations and the check results ("" = none).
func (e *Env) latestGeneration() string {
	for _, d := range e.outDirs() {
		if info, err := os.Stat(e.Proj.Path(d + "/configs")); err != nil || !info.IsDir() {
			continue
		}
		if _, err := os.Stat(e.Proj.Path(d + "/reports/check_results.xlsx")); err == nil {
			return d
		}
	}
	return ""
}

// PlanDelivery tells whether the set can be made, from what (spec 13.6).
// It writes nothing.
func (e *Env) PlanDelivery() (*DeliveryPlan, error) {
	p := &DeliveryPlan{Revisions: []Revision{}}
	revs, err := e.Revisions()
	if err != nil {
		return nil, err
	}
	if revs != nil {
		p.Revisions = revs
	}
	p.NextVersion = nextVersion(revs)
	genAt := e.lastSnapshotTime()
	dir := e.latestGeneration()
	switch {
	case genAt.IsZero() || dir == "":
		p.Why = "まだ「生成とチェック」をしていません。先に「生成とチェック」を押してください"
		return p, nil
	}
	if changed := e.changedSince(genAt); changed != "" {
		p.Why = changed + "が、最後の生成（" + e.LastGeneration() + "）の後に変わりました。先にもう一度「生成とチェック」を押してください"
		return p, nil
	}
	snap, name, _ := e.lastSnapshot()
	// The newest output folder must be the one of the newest generation: a
	// folder deleted on the outputs page, or a generation that stopped
	// before its report, leaves an older folder that is not what was
	// checked last.
	if name == "" || Stamped(strings.TrimSuffix(name, ".json")) != Stamped(dir) {
		p.Why = "最後の生成（" + e.LastGeneration() + "）の出力フォルダが見つかりません（消したか、途中で止まりました）。もう一度「生成とチェック」を押してください"
		return p, nil
	}
	p.Generation, p.When = dir, Stamped(dir)
	p.Configs = len(e.configFiles(dir))
	if p.Configs == 0 {
		p.Why = "最後の生成で、コンフィグを書いた機器がありません（値の誤りのある機器は書きません）。指摘を直して、もう一度「生成とチェック」を押してください"
		return p, nil
	}
	if snap != nil && snap.Counted {
		p.Errors, p.Warnings, p.Known = snap.Errors, snap.Warnings, true
	}
	p.Summary = e.revisionSummary(revs, snap, strings.TrimSuffix(name, ".json"))
	p.Ready = true
	return p, nil
}

// configFiles lists the configurations of an output (not those of
// configs/_failed).
func (e *Env) configFiles(dir string) []string {
	entries, err := os.ReadDir(e.Proj.Path(dir + "/configs"))
	if err != nil {
		return nil
	}
	var out []string
	for _, ent := range entries {
		if !ent.IsDir() {
			out = append(out, ent.Name())
		}
	}
	sort.Strings(out)
	return out
}

var lastNumber = regexp.MustCompile(`(\d+)(\D*)$`)

// nextVersion proposes the version after the last one: its last number
// counted up ("1.0" → "1.1"). "1.0" for the first.
func nextVersion(revs []Revision) string {
	if len(revs) == 0 {
		return "1.0"
	}
	last := revs[len(revs)-1].Version
	m := lastNumber.FindStringSubmatchIndex(last)
	if m == nil {
		return ""
	}
	n, err := strconv.Atoi(last[m[2]:m[3]])
	if err != nil {
		return ""
	}
	return last[:m[2]] + strconv.Itoa(n+1) + last[m[4]:m[5]]
}

// revisionSummary proposes what changed since the last version: the cells
// of each table that are not what they were then (spec 13.7).
func (e *Env) revisionSummary(revs []Revision, now *snapFile, nowName string) string {
	if len(revs) == 0 {
		return "初版"
	}
	last := revs[len(revs)-1]
	if now == nil || last.Snapshot == "" {
		return ""
	}
	if last.Snapshot == nowName {
		return "前の版と同じ生成です（値の変更はありません）"
	}
	data, err := os.ReadFile(e.Proj.Path(snapshotDir + "/" + last.Snapshot + ".json"))
	if err != nil {
		return ""
	}
	var before snapFile
	if err := json.Unmarshal(data, &before); err != nil {
		return ""
	}
	cells := func(f *snapFile) map[string]string {
		m := map[string]string{}
		for _, d := range f.Devices {
			for _, c := range d.Cells {
				m[d.Host+"\x00"+cellID(c)] = c.Value
			}
		}
		return m
	}
	tableOf := map[string]string{}
	for _, f := range []*snapFile{&before, now} {
		for _, d := range f.Devices {
			for _, c := range d.Cells {
				tableOf[d.Host+"\x00"+cellID(c)] = c.Table
			}
		}
	}
	a, b := cells(&before), cells(now)
	count := map[string]int{}
	for k, v := range b {
		if w, ok := a[k]; !ok || w != v {
			count[tableOf[k]]++
		}
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			count[tableOf[k]]++
		}
	}
	if len(count) == 0 {
		return "値の変更はありません"
	}
	var parts []string
	for _, t := range e.Schema.Tables {
		if n := count[t.Name]; n > 0 {
			name := t.Title.JA
			if name == "" {
				name = t.Name
			}
			parts = append(parts, fmt.Sprintf("%s %d セル", name, n))
		}
	}
	return strings.Join(parts, "、") + "を変更"
}

// DeliveryOptions are the choices of one set.
type DeliveryOptions struct {
	Mask            bool // write *** for the secret values of the sheets
	AllowUnverified bool // make it although findings are not approved
	// Version "" = no entry in the history.
	Version, Summary, By string
}

// DeliveryOutcome is what MakeDelivery made.
type DeliveryOutcome struct {
	Dir        string   // out/<time>/納品_<case>
	Zip        string   // out/<time>/納品_<case>_<date>.zip
	Files      []string // in Dir, with their folders
	Configs    int
	Unverified bool
	Errors     int
	Warnings   int
	Revision   *Revision
	Notes      []string
}

// MakeDelivery gathers the configurations and the check results of the
// newest generation, and the sheets and the table for the peers made now,
// into one folder and a zip of it (spec 13.6).
func (e *Env) MakeDelivery(opt DeliveryOptions, now time.Time) (*DeliveryOutcome, error) {
	plan, err := e.PlanDelivery()
	if err != nil {
		return nil, err
	}
	if !plan.Ready {
		return nil, fmt.Errorf("%s", plan.Why)
	}
	unverified := false
	if _, err := os.Stat(e.Proj.Path(plan.Generation + "/UNVERIFIED.txt")); err == nil {
		unverified = true
	}
	if plan.Known && plan.Errors+plan.Warnings > 0 {
		unverified = true
	}
	if unverified && !opt.AllowUnverified {
		return nil, fmt.Errorf("未承認の指摘が残っています（エラー %d・警告 %d）。直すか承認してから作るか、残したまま作ることを選んでください", plan.Errors, plan.Warnings)
	}
	words := e.deliveryNames()
	name := words.prefix + fileSafe(e.Proj.Config.Name)
	base := e.outDir(now)
	dir := base + "/" + name
	if _, err := os.Stat(e.Proj.Path(dir)); err == nil {
		return nil, fmt.Errorf("%s がもうあります。少し待ってから、もう一度作ってください", dir)
	}
	out := &DeliveryOutcome{Dir: dir, Unverified: unverified, Errors: plan.Errors, Warnings: plan.Warnings}
	_, had := os.Stat(e.Proj.Path(base))
	fail := func(err error) (*DeliveryOutcome, error) {
		// What this run wrote goes: the folder of the set, and the whole
		// folder of the run when it was made for it (the table for the
		// peers is written beside the set).
		if os.IsNotExist(had) {
			_ = os.RemoveAll(e.Proj.Path(base))
		} else {
			_ = os.RemoveAll(e.Proj.Path(dir))
		}
		return nil, err
	}

	// The history first: the sheets made now show this version.
	revs := plan.Revisions
	var cur *Revision
	if v := strings.TrimSpace(opt.Version); v != "" {
		for _, r := range revs {
			if r.Version == v {
				return nil, fmt.Errorf("版 %s は、もう改版履歴にあります（%s）。別の版にしてください", v, r.Date)
			}
		}
		_, snapName, _ := e.lastSnapshot()
		cur = &Revision{Version: v, Date: now.Format("2006-01-02"), By: strings.TrimSpace(opt.By), Summary: strings.TrimSpace(opt.Summary), Snapshot: strings.TrimSuffix(snapName, ".json")}
		revs = append(append([]Revision{}, revs...), *cur)
	}

	copyFile := func(from, to string) error {
		data, err := os.ReadFile(e.Proj.Path(from))
		if err != nil {
			return fmt.Errorf("%s を読めません: %v", from, err)
		}
		if err := os.MkdirAll(filepath.Dir(e.Proj.Path(to)), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(e.Proj.Path(to), data, 0o644); err != nil {
			return fmt.Errorf("%s を書けません: %v", to, err)
		}
		out.Files = append(out.Files, to)
		return nil
	}
	for _, f := range e.configFiles(plan.Generation) {
		if err := copyFile(plan.Generation+"/configs/"+f, dir+"/configs/"+f); err != nil {
			return fail(err)
		}
		out.Configs++
	}
	reports := []struct{ from, to string }{
		{"check_results", words.checks},
		{"rewrite", words.rewrite},
		{"changes", words.changes},
	}
	for _, r := range reports {
		for _, ext := range []string{".xlsx", ".html"} {
			from := plan.Generation + "/reports/" + r.from + ext
			if _, err := os.Stat(e.Proj.Path(from)); err != nil {
				continue
			}
			if err := copyFile(from, dir+"/"+r.to+ext); err != nil {
				return fail(err)
			}
		}
	}
	if unverified {
		text := fmt.Sprintf("未承認の指摘が残っています（エラー %d・警告 %d）。チェック結果表を見てください。\r\n", plan.Errors, plan.Warnings)
		if err := os.WriteFile(e.Proj.Path(dir+"/UNVERIFIED.txt"), []byte(text), 0o644); err != nil {
			return fail(err)
		}
		out.Files = append(out.Files, dir+"/UNVERIFIED.txt")
	}

	// The sheets and the table for the peers, made from the book now.
	sheets, err := e.exportSheets(opt.Mask, now, revs, cur, dir)
	if err != nil {
		return fail(err)
	}
	if len(sheets.Problems) > 0 {
		return fail(fmt.Errorf("納品用シートのひな形に誤りがあるため、納品一式を作っていません:\n%s", strings.Join(sheets.Problems, "\n")))
	}
	out.Files = append(out.Files, sheets.Files...)
	out.Notes = append(out.Notes, sheets.Notes...)
	peer, err := e.ExportPeer(opt.Mask, now)
	if err != nil {
		return fail(err)
	}
	for _, f := range []string{peer.Files.XLSX, peer.Files.HTML} {
		if f == "" {
			continue
		}
		to := dir + "/" + words.peer + filepath.Ext(f)
		if err := os.Rename(e.Proj.Path(f), e.Proj.Path(to)); err != nil {
			return fail(fmt.Errorf("%s を写せません: %v", f, err))
		}
		out.Files = append(out.Files, to)
	}
	_ = os.Remove(e.Proj.Path(base + "/reports")) // only when the table for the peers left it empty

	// What the set holds, written last.
	var sb strings.Builder
	fmt.Fprintf(&sb, "案件: %s\r\n作った日時: %s\r\nもとにした生成: %s（%s）\r\n", e.Proj.Config.Name, now.Format("2006-01-02 15:04:05"), plan.When, plan.Generation)
	if cur != nil {
		fmt.Fprintf(&sb, "版: %s\r\n", cur.Version)
	}
	if unverified {
		fmt.Fprintf(&sb, "未承認の指摘: エラー %d・警告 %d（UNVERIFIED.txt）\r\n", plan.Errors, plan.Warnings)
	}
	fmt.Fprintf(&sb, "秘密の値: 納品用シート・相手に渡す表は%s。コンフィグにはそのまま入っています\r\n\r\nファイル:\r\n", secretWord(opt.Mask))
	for _, f := range out.Files {
		fmt.Fprintf(&sb, "  %s\r\n", strings.TrimPrefix(f, dir+"/"))
	}
	if err := os.WriteFile(e.Proj.Path(dir+"/"+words.done), []byte(sb.String()), 0o644); err != nil {
		return fail(err)
	}
	out.Files = append(out.Files, dir+"/"+words.done)

	out.Zip = base + "/" + name + "_" + now.Format("20060102") + ".zip"
	if err := zipFolder(e.Proj.Path(dir), e.Proj.Path(out.Zip), name, now); err != nil {
		_ = os.Remove(e.Proj.Path(out.Zip))
		return fail(fmt.Errorf("zip を作れません: %v", err))
	}
	if cur != nil {
		if err := e.saveRevisions(revs); err != nil {
			return nil, fmt.Errorf("%s を書けません: %v（納品一式はできています）", revisionsFile, err)
		}
		out.Revision = cur
	}
	e.Proj.Log(now, "export delivery: %s（コンフィグ %d、ファイル %d、未承認 %v、版 %s）", dir, out.Configs, len(out.Files), unverified, opt.Version)
	return out, nil
}

// zipFolder writes a zip of a folder, its files under the name root. The
// times in the zip are now: the same set, the same zip.
func zipFolder(dir, to, root string, now time.Time) error {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	var paths []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(paths)
	for _, path := range paths {
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		h := &zip.FileHeader{Name: root + "/" + filepath.ToSlash(rel), Method: zip.Deflate, Modified: now}
		h.Flags |= 0x800 // the names are UTF-8
		fw, err := w.CreateHeader(h)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(fw, f)
		f.Close()
		if err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}
	return os.WriteFile(to, buf.Bytes(), 0o644)
}

// fileSafe makes a name that Windows takes as a file name.
func fileSafe(name string) string {
	name = strings.TrimSpace(name)
	var sb strings.Builder
	for _, r := range name {
		if r < 0x20 || strings.ContainsRune(`<>:"/\|?*`, r) {
			r = '_'
		}
		sb.WriteRune(r)
	}
	s := strings.TrimRight(sb.String(), ". ")
	if s == "" {
		return "案件"
	}
	return s
}

// bookRevisions gives the history for the sheets: the versions, and the
// one made now (cur nil = the newest of the history).
func bookRevisions(revs []Revision, cur *Revision) ([]book.Revision, *book.Revision) {
	var list []book.Revision
	for _, r := range revs {
		list = append(list, book.Revision{Version: r.Version, Date: r.Date, By: r.By, Summary: r.Summary})
	}
	switch {
	case cur != nil:
		return list, &book.Revision{Version: cur.Version, Date: cur.Date, By: cur.By, Summary: cur.Summary}
	case len(list) > 0:
		last := list[len(list)-1]
		return list, &last
	}
	return list, nil
}

// BookHosts lists the host names of the sheet 機器 of the book, in its
// order (none = no book).
func (e *Env) BookHosts() ([]string, error) {
	_, bk, err := e.readBook()
	if err != nil || bk == nil {
		return []string{}, err
	}
	ds, _ := bk.Expand(e.Schema)
	hosts := book.Hosts(ds)
	if hosts == nil {
		hosts = []string{}
	}
	return hosts, nil
}

// SetDeliveryNames writes the language of the names of the delivery set
// into project.yaml (spec 14.15.4): export.delivery_names only, the other
// lines and their comments staying. The file before goes to state/backup.
// It gives the backup ("" = nothing changed).
func (e *Env) SetDeliveryNames(lang string, now time.Time) (string, error) {
	ok := false
	for _, l := range DeliveryLangs {
		ok = ok || l == lang
	}
	if !ok {
		return "", fmt.Errorf("納品一式の名前の言葉「%s」は使えません（ja か en）", lang)
	}
	was := e.Proj.Config.Export.DeliveryNames
	if was == lang || (was == "" && lang == "ja") {
		return "", nil
	}
	path := e.Proj.Path(project.ConfigFile)
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("%s を読めません: %v", project.ConfigFile, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s を読めません: %v", project.ConfigFile, err)
	}
	next, err := putValue(data, "export", "delivery_names", lang)
	if err != nil {
		return "", fmt.Errorf("%s を書き換えられません: %v", project.ConfigFile, err)
	}
	copyTo, err := e.copyToBackup(project.ConfigFile, "project", data, now)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, next, 0o644); err != nil {
		return "", fmt.Errorf("%s を書けません: %v", project.ConfigFile, err)
	}
	// The names of the set do not change what is generated: the time of
	// the file stays, so the last generation is not taken as old (spec
	// 13.6).
	_ = os.Chtimes(path, info.ModTime(), info.ModTime())
	e.Proj.Config.Export.DeliveryNames = lang
	e.Proj.Log(now, "納品一式の名前の言葉を %s にしました（控え %s）", lang, copyTo)
	return copyTo, nil
}

// putValue gives a YAML file with group.key set to a plain value; the other
// lines and their comments stay.
func putValue(data []byte, group, key, value string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("先頭が設定の並びではありません")
	}
	top := doc.Content[0]
	g := mapValue(top, group)
	if g == nil {
		g = &yaml.Node{Kind: yaml.MappingNode}
		top.Content = append(top.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: group}, g)
	}
	if g.Kind != yaml.MappingNode {
		g.Kind, g.Tag, g.Value, g.Content = yaml.MappingNode, "", "", nil
	}
	setMap(g, key, &yaml.Node{Kind: yaml.ScalarNode, Value: value})
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SetCaseName writes the name of the case into project.yaml (spec 14.16.3).
// The folder is renamed by the caller; this only writes the name.
func (e *Env) SetCaseName(name string, now time.Time) error {
	if e.Proj.Config.Name == name {
		return nil
	}
	path := e.Proj.Path(project.ConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s を読めません: %v", project.ConfigFile, err)
	}
	next, err := putTop(data, "name", name)
	if err != nil {
		return fmt.Errorf("%s を書き換えられません: %v", project.ConfigFile, err)
	}
	copyTo, err := e.copyToBackup(project.ConfigFile, "project", data, now)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, next, 0o644); err != nil {
		return fmt.Errorf("%s を書けません: %v", project.ConfigFile, err)
	}
	was := e.Proj.Config.Name
	e.Proj.Config.Name = name
	e.Proj.Log(now, "案件名を「%s」から「%s」に変えました（控え %s）", was, name, copyTo)
	return nil
}

// putTop sets a key at the top of a YAML file, keeping the other lines and
// their comments.
func putTop(data []byte, key, value string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("先頭が設定の並びではありません")
	}
	setMap(doc.Content[0], key, &yaml.Node{Kind: yaml.ScalarNode, Value: value})
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
