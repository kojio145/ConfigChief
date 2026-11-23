package ops

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The list of the outputs under out/ (spec 14.12.6).

// OutputDir is one folder of out/.
type OutputDir struct {
	Dir   string   `json:"dir"` // out/<time>
	When  string   `json:"when"`
	Kinds []string `json:"kinds"` // what made it, in words
	Files int      `json:"files"`
	Size  int64    `json:"size"`
	// Keep tells the folders kept when old ones are taken away: the newest
	// generation and the newest whole set.
	Keep bool `json:"keep"`
}

// outputKinds tells what made an output from the files it holds.
var outputKinds = []struct {
	rel, word string
}{
	{"configs", "生成とチェック"},
	{"reports/check_results.xlsx", "チェック"},
	{"reports/import_old.xlsx", "取り込み（旧）"},
	{"reports/import_new.xlsx", "取り込み（新）"},
	{"reports/customer.xlsx", "外部一覧"},
	{"sheets", "納品用シート"},
	{"reports/peer_parameters.xlsx", "相手に渡す表"},
	{"reports/load_logs.xlsx", "投入ログ"},
	{"reports/verify.xlsx", "反映確認"},
	{"reports/tree_diff.xlsx", "階層を見た比較"},
	{"reports/sites.xlsx", "多拠点の一覧"},
	{"reports/unhandled_shapes.xlsx", "持ち出し用レポート"},
	{"reports/fill.xlsx", "ひな形から作る"},
	{"reports/delta.xlsx", "差分の投入"},
	{"reports/rewrite.xlsx", "書き換えレポート"},
}

// Outputs lists the folders of out/, the newest first. It writes nothing.
func (e *Env) Outputs() []OutputDir {
	gen, deliv := e.latestGeneration(), ""
	list := []OutputDir{}
	for _, d := range e.outDirs() {
		o := OutputDir{Dir: d, When: Stamped(d), Kinds: []string{}}
		have := func(rel string) bool {
			_, err := os.Stat(e.Proj.Path(d + "/" + rel))
			return err == nil
		}
		gened := have("configs")
		for _, k := range outputKinds {
			if !have(k.rel) || (k.word == "チェック" && gened) {
				continue
			}
			if k.word == "チェック" {
				o.Kinds = append(o.Kinds, "チェックだけ")
				continue
			}
			o.Kinds = append(o.Kinds, k.word)
		}
		if entries, err := os.ReadDir(e.Proj.Path(d)); err == nil {
			for _, ent := range entries {
				if ent.IsDir() && strings.HasPrefix(ent.Name(), deliveryPrefix) {
					o.Kinds = append(o.Kinds, "納品一式")
					if deliv == "" {
						deliv = d
					}
					break
				}
			}
		}
		if rs, err := os.ReadDir(e.Proj.Path(d + "/reports")); err == nil {
			for _, r := range rs {
				if strings.HasPrefix(r.Name(), "compare_") && strings.HasSuffix(r.Name(), ".xlsx") {
					o.Kinds = append(o.Kinds, "比較")
					break
				}
			}
		}
		_ = filepath.WalkDir(e.Proj.Path(d), func(path string, ent fs.DirEntry, err error) error {
			if err == nil && !ent.IsDir() {
				o.Files++
				if info, err := ent.Info(); err == nil {
					o.Size += info.Size()
				}
			}
			return nil
		})
		if len(o.Kinds) == 0 {
			o.Kinds = append(o.Kinds, "そのほか")
		}
		o.Keep = d == gen || d == deliv
		list = append(list, o)
	}
	return list
}

// DropOutputs takes folders of out/ away (spec 14.12.6). Only folders right
// under out/ are taken.
func (e *Env) DropOutputs(dirs []string, now time.Time) (int, error) {
	known := map[string]bool{}
	for _, d := range e.outDirs() {
		known[d] = true
	}
	n := 0
	for _, d := range dirs {
		if !known[d] {
			return n, fmt.Errorf("%s は out の下の出力のフォルダではありません", d)
		}
		if err := os.RemoveAll(e.Proj.Path(d)); err != nil {
			return n, fmt.Errorf("%s を消せません: %v", d, err)
		}
		n++
	}
	if n > 0 {
		e.Proj.Log(now, "outputs: %d 個の出力のフォルダを消しました（%s）", n, strings.Join(dirs, "、"))
	}
	return n, nil
}
