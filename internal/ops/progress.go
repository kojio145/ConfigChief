package ops

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// The steps of a case, from the first file to the check after the
// configurations went in (spec 14.12.1). The home screen shows them in a
// row, and the first one left to do is the next step.

// Step states.
const (
	StepDone     = "done"     // nothing left to do
	StepTodo     = "todo"     // something to do, with a count
	StepWait     = "wait"     // not reached yet
	StepOptional = "optional" // a step a case may leave out
	StepCounting = "counting" // the count is not known yet
)

// Step is one step of a case.
type Step struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	State string `json:"state"`
	Note  string `json:"note"` // a short count ("3 行")
	Text  string `json:"text"` // what to do, in a sentence
	Page  string `json:"page"`
	Act   string `json:"act"`
}

// Progress is the steps of the case and the next thing to do.
type Progress struct {
	Steps []Step `json:"steps"`
	Next  *Next  `json:"next"`
}

// OpenCount is the count of the unhandled lines not judged or on hold, of
// the old and the new side. Known false: not counted (yet).
type OpenCount struct {
	Lines int
	Known bool
	// IfsOpen is the count of the findings M-406 (a new interface name not
	// decided) that are not approved; IfsKnown tells it was counted (spec
	// 14.23.4: rows approved as they are count as decided).
	IfsOpen  int
	IfsKnown bool
}

// CountOpen reads the configurations of both sides again and counts the
// unhandled lines that are not judged or on hold. It writes nothing.
func (e *Env) CountOpen() OpenCount {
	n := 0
	for _, side := range []string{"old", "new"} {
		files, err := e.Proj.Inputs(side)
		if err != nil {
			return OpenCount{}
		}
		if len(files) == 0 {
			continue
		}
		j, err := e.Review(side)
		if err != nil {
			return OpenCount{}
		}
		n += j.Open + j.Hold
	}
	out := OpenCount{Lines: n, Known: true}
	if m, err := e.Mapping(); err == nil && m.Mig != nil {
		out.IfsKnown = true
		for _, f := range m.Mig.Items {
			if f.RuleID == "M-406" && f.Approval == nil {
				out.IfsOpen++
			}
		}
	}
	return out
}

// ProgressMark is a mark of what the count of the unhandled lines depends
// on: the names, sizes and times of the inputs, the book, the judgments,
// the choices of the files and the packs of the case. The same mark, the
// same count (spec 14.12.1).
func (e *Env) ProgressMark() string {
	h := sha256.New()
	add := func(rel string) {
		_ = filepath.WalkDir(e.Proj.Path(rel), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			fmt.Fprintf(h, "%s|%d|%d|%v\n", path, info.Size(), info.ModTime().UnixNano(), d.IsDir())
			return nil
		})
	}
	for _, rel := range []string{"input/old", "input/new", "state/judgments.yaml", "state/approvals.yaml", "state/sources.yaml", "packs", "extra", "project.yaml"} {
		add(rel)
	}
	add(e.Proj.Config.Book)
	return hex.EncodeToString(h.Sum(nil))
}

// Progress tells the steps of the case (spec 14.12.1). open is the count
// of the unhandled lines (CountOpen; it may not be known yet). It writes
// nothing.
func (e *Env) Progress(open OpenCount) (*Progress, error) {
	old, err := e.Proj.Inputs("old")
	if err != nil {
		return nil, err
	}
	_, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	migration := e.Proj.Config.Kind != "new"
	p := &Progress{}
	add := func(s Step) { p.Steps = append(p.Steps, s) }

	// The steps up to the design data in the book.
	bookDone := false
	if migration {
		s := Step{ID: "old", Label: "旧コンフィグ", Page: "home", Act: "add-old"}
		if len(old) > 0 {
			s.State, s.Note = StepDone, fmt.Sprintf("%d ファイル", len(old))
		} else {
			s.State, s.Text = StepTodo, "旧コンフィグを置いてください（ファイルをここに落とすか「ファイルを足す」）"
		}
		add(s)
		s = Step{ID: "import", Label: "取り込み", Page: "import", Act: "import-run"}
		switch {
		case bk != nil:
			s.State = StepDone
		case len(old) > 0:
			s.State, s.Text = StepTodo, fmt.Sprintf("旧コンフィグ %d ファイルを取り込んでください", len(old))
		default:
			s.State = StepWait
		}
		add(s)
		dev := Step{ID: "devices", Label: "新旧対照表（機器）", Page: "mapping", Act: "edit-devices", State: StepWait}
		ifs := Step{ID: "ifs", Label: "新旧対照表（ポート）", Page: "mapping", Act: "edit-ifs", State: StepWait}
		book := Step{ID: "book", Label: "パラメータブックに書く", Page: "mapping", Act: "make-book", State: StepWait}
		if bk != nil {
			empty, hosts := 0, map[string]bool{}
			for _, d := range bk.Devices {
				if d.NewHost == "" && d.Note == "" && !d.Out {
					empty++
				}
			}
			for _, m := range bk.Ifs {
				hosts[m.NewHost] = true
			}
			var noIfs []string
			for _, d := range bk.Devices {
				if d.NewHost != "" && !hosts[d.NewHost] {
					noIfs = append(noIfs, d.NewHost)
				}
			}
			undecided := 0
			for _, m := range bk.Ifs {
				if m.NewIf == "" {
					undecided++
				}
			}
			if empty > 0 {
				dev.State, dev.Note = StepTodo, fmt.Sprintf("%d 台", empty)
				dev.Text = fmt.Sprintf("「新旧対照表（機器）」の新ホスト名が %d 台空です。新ホスト名・新メーカー・新機種を書いてください（持っていかない機器は、新ホスト名に - を書きます）", empty)
			} else {
				dev.State = StepDone
			}
			switch {
			case empty > 0:
			case len(noIfs) > 0 && len(old) > 0:
				sort.Strings(noIfs)
				ifs.State, ifs.Note, ifs.Act = StepTodo, fmt.Sprintf("%d 台", len(noIfs)), "propose-ifs"
				ifs.Text = fmt.Sprintf("新ホスト名を書いた %d 台（%s）に、新旧対照表（ポート）の行がありません。新旧対照表（ポート）の案を作ってください", len(noIfs), short(noIfs))
			case undecided > 0 && open.IfsKnown && open.IfsOpen == 0:
				// Every row left empty is approved as it is.
				ifs.State, ifs.Note = StepDone, "承認済み"
			case undecided > 0:
				ifs.State, ifs.Note = StepTodo, fmt.Sprintf("%d 行", undecided)
				ifs.Text = fmt.Sprintf("「新旧対照表（ポート）」の新 IF 名が %d 行空です。新しい IF 名を書いてください（持っていかない IF は -）", undecided)
			default:
				ifs.State = StepDone
			}
			t := bk.Data.Tables["devices"]
			switch {
			case t != nil && len(t.Rows) > 0:
				book.State, bookDone = StepDone, true
			case dev.State == StepDone && ifs.State == StepDone && len(old) > 0:
				book.State, book.Text = StepTodo, "新旧対照表が決まりました。「パラメータブックに書く」を押してください"
			}
		}
		add(dev)
		add(ifs)
		add(book)
		j := Step{ID: "judge", Label: "読めなかった行", Page: "judge", State: StepWait}
		switch {
		case len(old) == 0 || bk == nil:
		case !open.Known:
			j.State = StepCounting
		case open.Lines > 0:
			j.State, j.Note = StepTodo, fmt.Sprintf("%d 行", open.Lines)
			j.Text = fmt.Sprintf("未判定・保留の読めなかった行が %d 行あります。「追加コンフィグで書く」「要らない」を付けてください", open.Lines)
		default:
			j.State = StepDone
		}
		add(j)
	} else {
		s := Step{ID: "newbook", Label: "空のブック", Page: "home", Act: "new-book"}
		if bk != nil {
			s.State = StepDone
		} else {
			s.State, s.Text = StepTodo, "空のパラメータブックを作ってください（新規構築）"
		}
		add(s)
		s = Step{ID: "first", Label: "1 台目を書く", Page: "edit", Act: "edit-first", State: StepWait}
		if bk != nil {
			if t := bk.Data.Tables["devices"]; t != nil && len(t.Rows) > 0 {
				s.State, s.Note, bookDone = StepDone, fmt.Sprintf("%d 台", len(t.Rows)), true
			} else {
				s.State, s.Text = StepTodo, "パラメータブックの「機器」のシートに、1 台目を書いてください"
			}
		}
		add(s)
	}

	// The steps from the generation on.
	snap, _, _ := e.lastSnapshot()
	genAt := e.lastSnapshotTime()
	gen := Step{ID: "generate", Label: "生成とチェック", Page: "home", Act: "generate", State: StepWait}
	fresh := false
	if bookDone {
		switch changed := e.changedSince(genAt); {
		case genAt.IsZero():
			gen.State, gen.Text = StepTodo, "「生成とチェック」を押してください"
		case changed != "":
			gen.State, gen.Note = StepTodo, "やり直し"
			gen.Text = changed + "が、最後の生成（" + e.LastGeneration() + "）の後に変わりました。もう一度「生成とチェック」を押してください"
		default:
			gen.State, fresh = StepDone, true
		}
	}
	add(gen)
	fin := Step{ID: "findings", Label: "指摘を片付ける", Page: "result", Act: "findings", State: StepWait}
	if fresh {
		switch {
		case snap == nil || !snap.Counted:
			fin.State, fin.Text = StepTodo, "結果の指摘を確かめてください（直すか承認）"
		case snap.Errors+snap.Warnings > 0:
			fin.State, fin.Note = StepTodo, fmt.Sprintf("%d 件", snap.Errors+snap.Warnings)
			fin.Text = fmt.Sprintf("未承認の指摘が %d 件あります（エラー %d・警告 %d）。値を直すか、承認してください", snap.Errors+snap.Warnings, snap.Errors, snap.Warnings)
		default:
			fin.State = StepDone
		}
	}
	add(fin)

	// The log of a check device: no log is no step to do; a log newer than
	// its check is.
	logs := Step{ID: "logs", Label: "検証機のログ", Page: "logs", Act: "logs-run", State: StepOptional}
	if fresh {
		logs.Text = "1 台分を検証機に流して、その端末ログを「検証機のログ」の画面に置き、「調べる」を押してください（機器に弾かれた行が出ます）"
		newest := e.newestIn("input/logs")
		checked := e.newestReport("load_logs.xlsx")
		switch {
		case newest.IsZero():
		case checked.IsZero() || newest.After(checked):
			logs.State, logs.Text = StepTodo, "「検証機のログ」の画面で、置いた端末ログを調べてください（機器に弾かれた行が出ます）"
		default:
			logs.State = StepDone
		}
	}
	add(logs)
	del := Step{ID: "deliver", Label: "納品物", Page: "home", Act: "make-delivery", State: StepWait}
	delivered := time.Time{}
	if fresh {
		delivered = e.newestDelivery()
		if !delivered.IsZero() && delivered.After(genAt) {
			del.State = StepDone
		} else {
			del.State, del.Text = StepTodo, "納品物を出してください（ホームの「納品物」の「納品一式を作る」）"
		}
	}
	add(del)
	ver := Step{ID: "verify", Label: "投入後の確認", Page: "logs", Act: "verify", State: StepOptional}
	if del.State == StepDone {
		ver.Text = "投入の後に、投入後の設定のログを「検証機のログ」の画面に置き、「反映確認」で比べてください"
		if at := e.newestReport("verify.xlsx"); !at.IsZero() && at.After(delivered) {
			ver.State = StepDone
		}
	}
	add(ver)

	// The next step: the draft of the table first, then the first step to
	// do; a step a case may leave out only when nothing else is left.
	if n := e.DraftCount(); n > 0 {
		p.Next = &Next{fmt.Sprintf("表の編集に、保存していない変更が %d 件あります。保存してください", n), "edit", "g-save"}
		return p, nil
	}
	for _, s := range p.Steps {
		if s.State == StepTodo {
			p.Next = &Next{s.Text, s.Page, s.Act}
			return p, nil
		}
		if s.State == StepCounting {
			p.Next = &Next{"読めなかった行を数えています…", "", ""}
			return p, nil
		}
	}
	for _, s := range p.Steps {
		if s.State == StepOptional && s.Text != "" {
			p.Next = &Next{s.Text + "（しない案件では、このままでかまいません）", s.Page, s.Act}
			return p, nil
		}
	}
	if del.State == StepDone {
		p.Next = &Next{"ひととおり済みました", "", ""}
	}
	return p, nil
}

// changedSince tells what the generation depends on that changed after t
// ("" = nothing): the book, the approvals, the rules and settings of the
// case, the extra configurations, the packs of the case.
func (e *Env) changedSince(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	var what []string
	newer := func(rel string) bool {
		at := e.newestIn(rel)
		return !at.IsZero() && at.After(t)
	}
	if newer(e.Proj.Config.Book) {
		what = append(what, "パラメータブック")
	}
	if newer("state/approvals.yaml") {
		what = append(what, "承認")
	}
	if newer("project.yaml") || newer("rules") {
		what = append(what, "案件の設定・ルール")
	}
	if newer("extra") {
		what = append(what, "追加コンフィグ")
	}
	if newer("packs") {
		what = append(what, "案件のパック")
	}
	return strings.Join(what, "・")
}

// newestIn is the newest time of a file, or of the files under a folder
// (zero = none).
func (e *Env) newestIn(rel string) time.Time {
	var at time.Time
	_ = filepath.WalkDir(e.Proj.Path(rel), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "~$") {
			return nil
		}
		if info, err := d.Info(); err == nil && info.ModTime().After(at) {
			at = info.ModTime()
		}
		return nil
	})
	return at
}

// outDirs are the folders under out/, the newest first.
func (e *Env) outDirs() []string {
	entries, err := os.ReadDir(e.Proj.Path("out"))
	if err != nil {
		return nil
	}
	var dirs []string
	for _, ent := range entries {
		if ent.IsDir() {
			dirs = append(dirs, "out/"+ent.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	return dirs
}

// newestReport is the time of the newest reports/<name> under out/ (zero =
// none).
func (e *Env) newestReport(name string) time.Time {
	var at time.Time
	for _, d := range e.outDirs() {
		if info, err := os.Stat(e.Proj.Path(d + "/reports/" + name)); err == nil && info.ModTime().After(at) {
			at = info.ModTime()
		}
	}
	return at
}

// newestDelivery is when the deliverables were last put out: the sheets,
// the table for the peers, or the whole set (zero = never).
func (e *Env) newestDelivery() time.Time {
	var at time.Time
	later := func(rel string) {
		if info, err := os.Stat(e.Proj.Path(rel)); err == nil && info.ModTime().After(at) {
			at = info.ModTime()
		}
	}
	for _, d := range e.outDirs() {
		later(d + "/reports/peer_parameters.xlsx")
		if t := e.newestIn(d + "/sheets"); t.After(at) {
			at = t
		}
		entries, _ := os.ReadDir(e.Proj.Path(d))
		for _, ent := range entries {
			if done := deliveryDoneOf(ent.Name()); done != "" && ent.IsDir() {
				later(d + "/" + ent.Name() + "/" + done)
			}
		}
	}
	return at
}
