package ops

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// Next is the one thing the home screen tells a person to do next (spec
// 14.10.4).
type Next struct {
	Text string `json:"text"`
	Page string `json:"page"` // the screen to go to
	// Act names the control on that screen to point at ("" = none).
	Act string `json:"act"`
}

// NextStep tells what to do next, from what the case folder holds. The
// first of the rules of spec 14.10.4 that holds wins. It writes nothing.
func (e *Env) NextStep() (*Next, error) {
	old, err := e.Proj.Inputs("old")
	if err != nil {
		return nil, err
	}
	_, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	migration := e.Proj.Config.Kind != "new"
	if bk == nil {
		switch {
		case migration && len(old) == 0:
			return &Next{"旧コンフィグを置いてください（ファイルをここに落とすか「ファイルを足す」）", "home", "add-old"}, nil
		case migration:
			return &Next{fmt.Sprintf("旧コンフィグ %d ファイルを取り込んでください", len(old)), "import", "import-run"}, nil
		default:
			return &Next{"空のパラメータブックを作ってください（新規構築）", "home", "new-book"}, nil
		}
	}
	if len(bk.Devices) > 0 {
		empty, hosts := 0, map[string]bool{}
		for _, d := range bk.Devices {
			if d.NewHost == "" && d.Note == "" {
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
		switch {
		case empty > 0:
			return &Next{fmt.Sprintf("「機器対応」の新ホスト名が %d 台空です。新ホスト名・新メーカー（分かれば新機種）を書いてください（持っていかない機器は、備考に理由を書きます）", empty), "mapping", "edit-devices"}, nil
		case len(noIfs) > 0 && len(old) > 0:
			sort.Strings(noIfs)
			return &Next{fmt.Sprintf("新ホスト名を書いた %d 台（%s）に、IF 対応の行がありません。IF 対応の案を作ってください", len(noIfs), short(noIfs)), "mapping", "propose-ifs"}, nil
		case undecided > 0:
			return &Next{fmt.Sprintf("「IF 対応」の新 IF 名が %d 行空です。新しい IF 名を書いてください（持っていかない IF は -）", undecided), "mapping", "edit-ifs"}, nil
		}
		if t := bk.Data.Tables["devices"]; (t == nil || len(t.Rows) == 0) && len(old) > 0 {
			return &Next{"対応が決まりました。「対応を当ててパラメータブックに書く」を押してください", "mapping", "make-book"}, nil
		}
	}
	if n := e.DraftCount(); n > 0 {
		return &Next{fmt.Sprintf("表の編集に、保存していない変更が %d 件あります。保存してください", n), "edit", "g-save"}, nil
	}
	if t := bk.Data.Tables["devices"]; t == nil || len(t.Rows) == 0 {
		return &Next{"パラメータブックの「機器」のシートに、1 台目を書いてください", "edit", ""}, nil
	}
	last := e.LastGeneration()
	if last == "" {
		return &Next{"「生成とチェック」を押してください", "home", "generate"}, nil
	}
	if info, err := os.Stat(e.Proj.BookPath()); err == nil {
		if gen := e.lastSnapshotTime(); !gen.IsZero() && info.ModTime().After(gen) {
			return &Next{"パラメータブックが、最後の生成（" + last + "）の後に変わりました。もう一度「生成とチェック」を押してください", "home", "generate"}, nil
		}
	}
	return &Next{"結果の指摘を確かめて（直すか承認）、納品物を出してください", "result", ""}, nil
}

// lastSnapshotTime is when the newest record of a generation was written
// (the time of the file: the name tells the time of the generation, the file
// is written after the book was read).
func (e *Env) lastSnapshotTime() time.Time {
	entries, err := os.ReadDir(e.Proj.Path(snapshotDir))
	if err != nil {
		return time.Time{}
	}
	var last time.Time
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		if info, err := ent.Info(); err == nil && info.ModTime().After(last) {
			last = info.ModTime()
		}
	}
	return last
}

// LastOutput finds the newest output of "generate" or "check": the folder
// under out/ that holds reports/check_results.xlsx ("" = none).
func (e *Env) LastOutput() string {
	entries, err := os.ReadDir(e.Proj.Path("out"))
	if err != nil {
		return ""
	}
	var dirs []string
	for _, ent := range entries {
		if ent.IsDir() {
			dirs = append(dirs, ent.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	for _, d := range dirs {
		if _, err := os.Stat(e.Proj.Path("out/" + d + "/reports/check_results.xlsx")); err == nil {
			return "out/" + d
		}
	}
	return ""
}

// Attach points the devices of an outcome that wrote nothing (Dry) at the
// files of an output folder of before (spec 14.10.6): the configurations
// that are there. It tells whether the folder has configurations, and the
// mark UNVERIFIED.txt ("" = none).
func (o *GenerateOutcome) Attach(e *Env, dir string) (configs bool, unverified string) {
	o.Dir = dir
	if info, err := os.Stat(e.Proj.Path(dir + "/configs")); err == nil && info.IsDir() {
		configs = true
	}
	for i := range o.Devices {
		d := &o.Devices[i]
		d.File = ""
		// Only where the check of now also gives a text: a device that is
		// blocked now does not point at a file of before.
		var try []string
		switch {
		case (d.State == GenWritten || d.State == GenChecked) && d.name != "":
			try = append(try, dir+"/configs/"+d.name)
		case d.State == GenFailed:
			try = append(try, dir+"/configs/_failed/"+safeName(d.Host)+".txt")
		}
		for _, rel := range try {
			if _, err := os.Stat(e.Proj.Path(rel)); err == nil {
				d.File = rel
				break
			}
		}
	}
	if _, err := os.Stat(e.Proj.Path(dir + "/UNVERIFIED.txt")); err == nil {
		unverified = dir + "/UNVERIFIED.txt"
	}
	o.Report = Files{XLSX: dir + "/reports/check_results.xlsx", HTML: dir + "/reports/check_results.html"}
	return configs, unverified
}

// Stamped gives the time of an output folder out/<time> in words.
func Stamped(dir string) string {
	stamp := strings.TrimPrefix(dir, "out/")
	if len(stamp) > 17 {
		stamp = stamp[:17]
	}
	if t, err := time.Parse("2006-01-02_150405", stamp); err == nil {
		return t.Format("2006-01-02 15:04:05")
	}
	return stamp
}
