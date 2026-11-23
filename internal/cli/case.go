package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/convert"
	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/textdiff"
)

// now gives the time of a run; a test puts a fixed one in.
var now = time.Now

// options reads the arguments of a subcommand: "--name value" for the names
// in values, "--name" for those in switches, anything else is positional.
func options(args, values, switches []string) (pos []string, val map[string]string, on map[string]bool, err error) {
	val, on = map[string]string{}, map[string]bool{}
	has := func(list []string, s string) bool {
		for _, x := range list {
			if x == s {
				return true
			}
		}
		return false
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case has(values, a):
			if i+1 >= len(args) {
				return nil, nil, nil, fmt.Errorf("%s の後に値がありません", a)
			}
			i++
			val[a] = args[i]
		case has(switches, a):
			on[a] = true
		case strings.HasPrefix(a, "--"):
			return nil, nil, nil, fmt.Errorf("%s という指定はありません", a)
		default:
			pos = append(pos, a)
		}
	}
	return pos, val, on, nil
}

// openCase opens the case folder named by --project.
func openCase(dir string, out, errOut io.Writer) (*ops.Env, bool) {
	if dir == "" {
		fmt.Fprintln(errOut, "--project <案件フォルダ> を指定してください")
		return nil, false
	}
	// The mark of a case in use (spec 14.8.2): a case open in the window is
	// left alone.
	if held == nil {
		h, err := project.Lock(dir)
		if err != nil {
			fmt.Fprintln(errOut, strings.Replace(err.Error(), "別の場所で開かれています", "画面など別の場所で開かれています", 1))
			return nil, false
		}
		held = h
	}
	e, err := ops.Open(dir)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return nil, false
	}
	if newer := project.NoteTool(dir, Version); newer != "" && writing {
		fmt.Fprintln(errOut, (&project.ReadOnlyError{Newer: newer, Mine: Version}).Error())
		return nil, false
	}
	e.Version = Version
	for _, p := range e.PackProblems {
		fmt.Fprintln(out, "注意: "+p)
	}
	return e, true
}

func sourceWord(name string) string {
	switch name {
	case "old":
		return "旧コンフィグ"
	case "new":
		return "新コンフィグ"
	case "book":
		return "パラメータブック"
	}
	return name
}

func cmdInit(args []string, out, errOut io.Writer) int {
	pos, val, on, err := options(args, []string{"--name", "--kind", "--from"}, []string{"--sample"})
	if err != nil || len(pos) != 1 || (on["--sample"] && len(val) > 0) {
		if err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprint(errOut, usage)
		return 2
	}
	dir := pos[0]
	if on["--sample"] {
		// The sample case (spec 14.12.7).
		if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
			fmt.Fprintf(errOut, "%s は空のフォルダではありません。見本の案件は、空のフォルダか、まだ無いフォルダに作ります\n", dir)
			return 2
		}
		n, err := ops.MakeSampleCase(dir)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 2
		}
		fmt.Fprintf(out, "見本の案件: %s（%s）\n", dir, ops.SampleName)
		fmt.Fprintf(out, "  架空の旧コンフィグ %d ファイルを input/old に置きました\n", n)
		fmt.Fprintf(out, "  次に: ConfigChief.exe import --project %s\n", dir)
		return 0
	}
	name := val["--name"]
	if name == "" {
		if abs, err := filepath.Abs(dir); err == nil {
			name = filepath.Base(abs)
		}
	}
	from := val["--from"]
	if from != "" {
		if _, err := project.Open(from); err != nil {
			fmt.Fprintf(errOut, "設定を写す元の案件を開けません: %v\n", err)
			return 2
		}
	}
	p, err := project.Init(dir, name, val["--kind"])
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	fmt.Fprintf(out, "案件フォルダ: %s（案件名 %s）\n", p.Dir, p.Config.Name)
	if from != "" {
		// The settings of a case before (spec 14.12.8): all but the extra
		// configurations of single devices.
		var parts []string
		for _, x := range ops.CaseParts {
			if x.Default {
				parts = append(parts, x.ID)
			}
		}
		c, err := ops.CopyCaseSettings(from, dir, parts)
		if err != nil {
			fmt.Fprintf(errOut, "設定を写す途中で止まりました: %v\n", err)
			return 2
		}
		if len(c.Parts) == 0 {
			fmt.Fprintf(out, "  元の案件 %s に、写す設定はありませんでした\n", from)
		} else {
			fmt.Fprintf(out, "  元の案件 %s から写しました（ファイル %d 個）: %s\n", from, c.Files, strings.Join(c.Parts, "、"))
		}
	}
	fmt.Fprintln(out, "  旧コンフィグを input/old に、手で書いた新コンフィグがあれば input/new に置きます（1 台 1 ファイル）")
	fmt.Fprintf(out, "  次に: ConfigChief.exe import --project %s\n", dir)
	return 0
}

func cmdImport(args []string, out, errOut io.Writer) int {
	pos, val, _, err := options(args, []string{"--project", "--source", "--pack", "--use"}, nil)
	if err != nil || len(pos) != 0 {
		if err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprint(errOut, usage)
		return 2
	}
	e, ok := openCase(val["--project"], out, errOut)
	if !ok {
		return 2
	}
	source := val["--source"]
	if source == "" {
		source = "old"
	}
	if use := val["--use"]; use != "" {
		// The file is named as the list shows it (input/old/a.txt) or by its
		// name alone.
		use = strings.ReplaceAll(use, "\\", "/")
		if !strings.Contains(use, "/") {
			use = "input/" + source + "/" + use
		}
		if err := e.ChooseFile(source, use, now()); err != nil {
			fmt.Fprintln(errOut, err)
			return 2
		}
	}
	o, err := e.RunImport(source, val["--pack"], now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	src := o.Source
	fmt.Fprintf(out, "取り込み（%s）: %d ファイル\n", sourceWord(source), len(src.Files))
	for _, f := range src.Files {
		if f.Err != "" {
			fmt.Fprintf(out, "  %s: 取り込めません: %s\n", f.Path, f.Err)
			if len(f.Same) > 1 {
				fmt.Fprintf(out, "    選ぶには: ConfigChief.exe import --project %s --source %s --use <ファイル>\n", val["--project"], source)
			}
			continue
		}
		if f.Left != "" {
			fmt.Fprintf(out, "  %s: %s\n", f.Path, f.Left)
			continue
		}
		n := f.Result.Count()
		pack := f.Pack
		if f.Chosen {
			pack += "（人が選んだ）"
		}
		fmt.Fprintf(out, "  %s: %s %s %d 行（取り込み済み %d、無視 %d、読めなかった %d）\n",
			f.Path, f.Result.Device, pack, n.Total, n.Taken, n.Ignored+n.Separators, n.Unknown)
		for _, w := range f.Result.Warnings {
			fmt.Fprintf(out, "    注意: %s\n", w)
		}
		for _, s := range f.Result.Notes {
			fmt.Fprintf(out, "    お知らせ: %s\n", s)
		}
	}
	if n := src.Count().Unknown; n > 0 {
		j := o.Judging
		fmt.Fprintf(out, "読めなかった行: %d 行（未判定 %d、保留 %d、追加コンフィグで書く %d、要らない %d）。レポートの「読めなかった行の形」「読めなかった行」に一覧があります\n",
			n, j.Open, j.Hold, j.Manual, j.NotNeeded)
		if j.Open+j.Hold > 0 {
			fmt.Fprintln(out, "  判定は、画面の「読めなかった行」か judge で付けます")
		}
	}
	printNoExtra(out, o.Judging)
	if o.New != nil {
		for _, f := range o.New.Files {
			if f.Err != "" {
				fmt.Fprintf(out, "注意: %s は新旧対照表の案に使えません: %s\n", f.Path, f.Err)
			}
		}
	}
	if s := o.Book; s != nil {
		book := e.Proj.Config.Book
		switch {
		case s.Created:
			fmt.Fprintf(out, "パラメータブック %s を作りました（新旧対照表（機器） %d 行、新旧対照表（ポート） %d 行）\n", book, s.DeviceRows, s.IfRows)
		case s.Changed():
			fmt.Fprintf(out, "パラメータブック %s に足しました（新旧対照表（機器） %d 行、新旧対照表（ポート） %d 行）。控え: %s\n", book, s.DeviceRows, s.IfRows, o.Backup)
		default:
			fmt.Fprintf(out, "パラメータブック %s は変えていません（足す行がありません）\n", book)
		}
	}
	for _, p := range o.Problems {
		fmt.Fprintf(out, "  ブックの注意: %s\n", p)
	}
	if a := o.Applied; a != nil {
		printMapping(out, e, a, o.Mig, false)
	}
	fmt.Fprintf(out, "レポート: %s\n          %s\n", o.Report.XLSX, o.Report.HTML)
	if o.Findings() > 0 {
		return 1
	}
	return 0
}

// printMapping tells what of the mapping is open; with all set, also what
// was left out of the data.
func printMapping(out io.Writer, e *ops.Env, a *convert.Applied, v *ops.Verdicts, all bool) {
	if c := v.Count(); c.Errors+c.Warnings+c.Approved > 0 {
		fmt.Fprintf(out, "新旧対照表で決まっていないところの指摘（M-406〜M-408）: 未承認 %d 件、承認済み %d 件（指紋はレポートの「新旧対照表で決まっていないところ」「新旧対照表のメモ」にあります）\n", c.Errors+c.Warnings, c.Approved)
	}
	if len(a.Unmapped) > 0 {
		fmt.Fprintf(out, "新ホスト名が空の旧機器（書き換えの対象外）: %d 台（うち備考もない %d 台）\n", len(a.Unmapped), len(a.NoReason))
		fmt.Fprintf(out, "  %s\n", strings.Join(a.Unmapped, ", "))
	}
	if len(a.Undecided) > 0 {
		fmt.Fprintf(out, "新 IF 名が決まっていない旧 IF: %d 個（旧の名前のままです）\n", len(a.Undecided))
		for _, n := range a.Undecided {
			fmt.Fprintf(out, "  %s %s\n", n.Device, n.Value)
		}
	}
	for _, n := range a.Shared {
		fmt.Fprintf(out, "新機器 %s で、旧 IF %s が同じ新 IF %s になっています\n", n.Device, n.Key, n.Value)
	}
	for _, n := range a.Clashes {
		t, r, _ := e.Place(n)
		fmt.Fprintf(out, "書かなかった行（先の行とキーが同じ）: %s %s %s\n", n.Device, t, r)
	}
	if !all {
		return
	}
	for _, n := range a.Dropped {
		t, r, _ := e.Place(n)
		fmt.Fprintf(out, "書かなかった行（新 IF 名が -）: %s %s %s\n", n.Device, t, r)
	}
	for _, n := range a.Cleared {
		t, r, c := e.Place(n)
		fmt.Fprintf(out, "空にした列（新 IF 名が - の IF を指していた）: %s %s %s の %s（%s）\n", n.Device, t, r, c, n.Value)
	}
}

func cmdBook(args []string, out, errOut io.Writer) int {
	pos, val, on, err := options(args, []string{"--project", "--from"}, []string{"--replace", "--new"})
	if err != nil || len(pos) != 0 {
		if err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprint(errOut, usage)
		return 2
	}
	e, ok := openCase(val["--project"], out, errOut)
	if !ok {
		return 2
	}
	if on["--new"] {
		// The empty book of a new build (spec 3.2).
		if val["--from"] != "" || on["--replace"] {
			fmt.Fprintln(errOut, "--new は、--from・--replace と一緒には使えません")
			return 2
		}
		if _, err := e.NewBook(now()); err != nil {
			fmt.Fprintln(errOut, err)
			return 2
		}
		fmt.Fprintf(out, "空のパラメータブック %s を作りました（見出しのセルに、列の説明があります）\n", e.Proj.Config.Book)
		fmt.Fprintln(out, "1 台目をシート「機器」から書き、似た機器は clone で増やせます")
		return 0
	}
	from := val["--from"]
	if from == "" {
		from = "old"
	}
	o, err := e.MakeBook(from, on["--replace"], now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	for _, f := range o.Source.Files {
		if f.Err != "" {
			fmt.Fprintf(out, "  %s: 取り込めません: %s\n", f.Path, f.Err)
		}
	}
	s, book := o.Summary, e.Proj.Config.Book
	switch {
	case s.Created:
		fmt.Fprintf(out, "パラメータブック %s を作りました（%s から %d 行）\n", book, sourceWord(from), s.Rows())
	case s.Changed():
		how := "足しました"
		if s.Replaced {
			how = "設計データのシートを空にしてから書きました"
		}
		fmt.Fprintf(out, "パラメータブック %s に%s（%s から %d 行）。控え: %s\n", book, how, sourceWord(from), s.Rows(), o.Backup)
	default:
		fmt.Fprintf(out, "パラメータブック %s は変えていません（足す行がありません）\n", book)
	}
	var names []string
	seen := map[string]bool{}
	for _, t := range e.Schema.Tables {
		if s.Added[t.Name] > 0 || s.Kept[t.Name] > 0 {
			names = append(names, t.Name)
			seen[t.Name] = true
		}
	}
	var rest []string
	for name := range s.Added {
		if !seen[name] {
			rest = append(rest, name)
		}
	}
	sort.Strings(rest)
	for _, name := range append(names, rest...) {
		title := name
		if t := e.Schema.Table(name); t != nil {
			title = t.Title.In(e.Proj.Config.Language)
		}
		fmt.Fprintf(out, "  %s: 足した %d 行、すでにあった %d 行\n", title, s.Added[name], s.Kept[name])
	}
	if s.DeviceRows > 0 || s.IfRows > 0 {
		fmt.Fprintf(out, "  新旧対照表（機器）に %d 行、新旧対照表（ポート）に %d 行を足しました\n", s.DeviceRows, s.IfRows)
	}
	for _, p := range o.Problems {
		fmt.Fprintf(out, "  ブックの注意: %s\n", p)
	}
	if o.Applied != nil {
		printMapping(out, e, o.Applied, o.Mig, true)
	}
	if o.Findings() > 0 {
		return 1
	}
	return 0
}

func cmdCompare(args []string, out, errOut io.Writer) int {
	pos, val, _, err := options(args, []string{"--project", "--a", "--b"}, nil)
	if err != nil || len(pos) != 0 || val["--a"] == "" || val["--b"] == "" {
		if err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprint(errOut, usage)
		return 2
	}
	e, ok := openCase(val["--project"], out, errOut)
	if !ok {
		return 2
	}
	o, err := e.Compare(val["--a"], val["--b"], now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	la, lb := sourceWord(o.A.Name), sourceWord(o.B.Name)
	r := o.Result
	fmt.Fprintf(out, "比較（A = %s、B = %s）\n", la, lb)
	for _, s := range []*ops.Side{o.A, o.B} {
		if s.Source == nil {
			continue
		}
		for _, f := range s.Source.Files {
			if f.Err != "" {
				fmt.Fprintf(out, "  %s: 取り込めません: %s\n", f.Path, f.Err)
			}
		}
		if n := s.Source.Count().Unknown; n > 0 {
			fmt.Fprintf(out, "  %s に読めなかった行が %d 行あります。その行の設定は、比べる対象に入っていません\n", sourceWord(s.Name), n)
		}
	}
	fmt.Fprintf(out, "  A にだけある行: %d\n  B にだけある行: %d\n  値が違うセル: %d\n  両方にある行: %d（うち値がすべて同じ %d）\n",
		len(r.OnlyA), len(r.OnlyB), len(r.Cells), r.Rows, r.Same)
	if o.Mig != nil {
		c := o.Mig.Count()
		fmt.Fprintf(out, "  移行のチェック（M-403〜M-408）: 未承認のエラー %d 件、未承認の警告 %d 件、承認済み %d 件（指紋はレポートの各行にあります）\n", c.Errors, c.Warnings, c.Approved)
	}
	for _, s := range []*ops.Side{o.A, o.B} {
		if a := s.Applied; a != nil && len(a.Unmapped)+len(a.Undecided)+len(a.Shared) > 0 {
			fmt.Fprintf(out, "  新旧対照表で決まっていないところ: 新ホスト名が空の旧機器 %d 台、新 IF 名が決まっていない旧 IF %d 個、同じ新 IF に当たっている組 %d（レポートの「新旧対照表のメモ」に一覧があります）\n",
				len(a.Unmapped), len(a.Undecided), len(a.Shared))
		}
		if s.Book != nil && len(s.Book.Problems)+len(s.Expand) > 0 {
			fmt.Fprintf(out, "  ブックの注意: %d 件（レポートの「注意」に一覧があります）\n", len(s.Book.Problems)+len(s.Expand))
		}
	}
	fmt.Fprintf(out, "レポート: %s\n          %s\n", o.Report.XLSX, o.Report.HTML)
	if o.Findings() > 0 {
		return 1
	}
	return 0
}

// cmdDiff compares two text files line by line (spec 12.1).
func cmdDiff(args []string, out, errOut io.Writer) int {
	for _, a := range args {
		if a == "--tree" {
			return cmdTreeDiff(args, out, errOut)
		}
	}
	if len(args) != 2 {
		fmt.Fprint(errOut, usage)
		return 2
	}
	var texts [2]string
	for i, path := range args {
		t, err := project.ReadText(path)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 2
		}
		texts[i] = t
	}
	r := textdiff.Diff(texts[0], texts[1])
	if r.Equal() {
		fmt.Fprintln(out, "違いはありません")
		return 0
	}
	fmt.Fprintf(out, "--- %s\n+++ %s\n%s", args[0], args[1], textdiff.Unified(r, 3))
	fmt.Fprintf(out, "減った行 %d、増えた行 %d\n", r.Removed, r.Added)
	return 1
}

// cmdClone copies the rows of one device to new host names (spec 6.6).
func cmdClone(args []string, out, errOut io.Writer) int {
	pos, val, _, err := options(args, []string{"--project", "--from", "--to"}, nil)
	if err != nil || len(pos) != 0 || val["--from"] == "" || val["--to"] == "" {
		if err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprint(errOut, usage)
		return 2
	}
	e, ok := openCase(val["--project"], out, errOut)
	if !ok {
		return 2
	}
	o, err := e.CloneDevice(val["--from"], strings.Split(val["--to"], ","), now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	fmt.Fprintf(out, "機器 %s の行を、%s に複製しました（%d 行。控え: %s）\n", o.From, strings.Join(o.To, "、"), o.Summary.Rows(), o.Backup)
	for _, t := range e.Schema.Tables {
		if n := o.Summary.Added[t.Name]; n > 0 {
			fmt.Fprintf(out, "  %s: %d 行\n", t.Title.In(e.Proj.Config.Language), n)
		}
	}
	if o.Shared > 0 {
		fmt.Fprintf(out, "機器の列が * か @グループ の行（%d 行）は複製していません（新しい機器にも、同じ決まりで当たります）\n", o.Shared)
	}
	fmt.Fprintln(out, "値は元の機器のままです。アドレスや説明など、機器ごとに違う値を直してください（重なりは check が指摘します）")
	return 0
}
