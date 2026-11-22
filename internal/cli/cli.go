// Package cli is the command line of ConfigChief (spec 15): it makes a case
// folder, imports configurations, makes the parameter book, compares,
// generates and checks, keeps the judgments of unhandled lines, and tests a
// vendor pack. The program of the window runs it when it is started with a
// command (see main.go of the root package).
package cli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/errlog"
	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/packtest"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/schemas"
)

// Version is the version the commands tell; the program sets it.
var Version = "0.0.0-dev"

// Commands are the first words that make an invocation a command. Anything
// else given to the program of the window is a case folder to open.
var Commands = []string{"version", "--version", "help", "--help", "init", "import", "book", "clone", "compare", "generate", "check", "approve",
	"judge", "customer", "fill", "case", "sites", "logs", "verify", "export", "diff", "pack"}

// IsCommand tells whether the arguments of the program are a command.
func IsCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	for _, c := range Commands {
		if args[0] == c {
			return true
		}
	}
	return false
}

const usage = `usage:
  ConfigChief.exe [<案件フォルダ>]          (画面を開く)
  ConfigChief.exe version
  ConfigChief.exe init    <dir> [--name <案件名>] [--kind migration|new]
  ConfigChief.exe import  --project <dir> [--source old|new] [--pack <ID>] [--use <ファイル>]
  ConfigChief.exe book    --project <dir> [--from old|new] [--replace]
  ConfigChief.exe book    --project <dir> --new
  ConfigChief.exe clone   --project <dir> --from <ホスト名> --to <ホスト名>[,<ホスト名>…]
  ConfigChief.exe compare --project <dir> --a old|new|book --b old|new|book
  ConfigChief.exe generate --project <dir> [--device <hostname>]
  ConfigChief.exe check   --project <dir>
  ConfigChief.exe approve --project <dir> --finding <指紋> --reason <理由> --by <名前>
  ConfigChief.exe approve --project <dir> --rule <ID> --devices <*|@グループ|ホスト名,...> [--table <表>] [--column <列>] --reason <理由> --by <名前>
  ConfigChief.exe approve --project <dir> --remove <承認の ID>
  ConfigChief.exe judge   --project <dir> [--source old|new] [--all]
  ConfigChief.exe judge   --project <dir> [--source old|new] --group <ID> --verdict manual|not_needed|hold|none [--devices <*|ホスト名,...>] [--reason <理由>] --by <名前>
  ConfigChief.exe customer --project <dir>
  ConfigChief.exe customer --project <dir> --draft <一覧のファイル名> [--sheet <シート名>]
  ConfigChief.exe customer --project <dir> --apply <指紋>[,<指紋>...]
  ConfigChief.exe customer --project <dir> --apply-rule S-501|S-503 [--devices <*|@グループ|ホスト名,...>]
  ConfigChief.exe fill    --project <dir> --list
  ConfigChief.exe fill    --project <dir> --template <名前> [--diff old|new|logs] [--device <機器>[,<機器>...]]
  ConfigChief.exe fill    --project <dir> --draft <コンフィグのファイル> --device <機器> [--name <ひな形の名前>]
  ConfigChief.exe case export --project <dir> --out <ファイル> [--no-secrets] [--password-env <環境変数名>]
  ConfigChief.exe case import <ファイル> --to <親フォルダ> [--password-env <環境変数名>]
  ConfigChief.exe export  --project <dir> --what sheet|peer [--secrets show|mask]
  ConfigChief.exe export  --project <dir> --what unhandled [--source old|new]
  ConfigChief.exe logs    --project <dir>
  ConfigChief.exe verify  --project <dir>
  ConfigChief.exe sites   --project <dir> [--source book|old|new|logs]
  ConfigChief.exe diff    <fileA> <fileB>
  ConfigChief.exe diff    --tree <fileA> <fileB> [--keep-stamps] [--keep-secrets]
  ConfigChief.exe diff    --tree --project <dir> --a <old|new|logs|generated|フォルダ> --b <…> [--keep-stamps] [--keep-secrets]
  ConfigChief.exe pack test <packdir> [--lines] [--dump]
  ConfigChief.exe pack columns <packdir>
`

// Run gives the exit code: 0 = fine, 1 = something to look at, 2 = could not
// run (spec 15).
func Run(args []string, out, errOut io.Writer) (code int) {
	// An internal error ends the command with 2 and a record (spec 14.8.4).
	defer func() {
		if r := recover(); r != nil {
			errlog.Version = Version
			fmt.Fprintln(errOut, errlog.Panic(strings.Join(head(args, 1), " "), r))
			code = 2
		}
		held.Release()
		held = nil
	}()
	writing = len(args) > 0 && (writes[args[0]] || (args[0] == "import" && hasArg(args, "--use")))
	switch {
	case len(args) == 1 && (args[0] == "version" || args[0] == "--version"):
		fmt.Fprintln(out, "ConfigChief "+Version)
		return 0
	case len(args) >= 1 && args[0] == "init":
		return cmdInit(args[1:], out, errOut)
	case len(args) >= 1 && args[0] == "import":
		return cmdImport(args[1:], out, errOut)
	case len(args) >= 1 && args[0] == "book":
		return cmdBook(args[1:], out, errOut)
	case len(args) >= 1 && args[0] == "clone":
		return cmdClone(args[1:], out, errOut)
	case len(args) >= 1 && args[0] == "compare":
		return cmdCompare(args[1:], out, errOut)
	case len(args) >= 1 && args[0] == "generate":
		return cmdGenerate(args[1:], out, errOut)
	case len(args) >= 1 && args[0] == "check":
		return cmdCheck(args[1:], out, errOut)
	case len(args) >= 1 && args[0] == "approve":
		return cmdApprove(args[1:], out, errOut)
	case len(args) >= 1 && args[0] == "judge":
		return cmdJudge(args[1:], out, errOut)
	case len(args) >= 1 && args[0] == "customer":
		return cmdCustomer(args[1:], out, errOut)
	case len(args) >= 1 && args[0] == "fill":
		return cmdFill(args[1:], out, errOut)
	case len(args) >= 2 && args[0] == "case" && args[1] == "export":
		return cmdCaseExport(args[2:], out, errOut)
	case len(args) >= 2 && args[0] == "case" && args[1] == "import":
		return cmdCaseImport(args[2:], out, errOut)
	case len(args) >= 1 && args[0] == "sites":
		return cmdSites(args[1:], out, errOut)
	case len(args) >= 1 && args[0] == "logs":
		return cmdLogs(args[1:], out, errOut)
	case len(args) >= 1 && args[0] == "verify":
		return cmdVerify(args[1:], out, errOut)
	case len(args) >= 1 && args[0] == "export":
		return cmdExport(args[1:], out, errOut)
	case len(args) >= 1 && args[0] == "diff":
		return cmdDiff(args[1:], out, errOut)
	case len(args) >= 3 && args[0] == "pack" && args[1] == "test":
		return packTest(args[2], args[3:], out, errOut)
	case len(args) == 3 && args[0] == "pack" && args[1] == "columns":
		return packColumns(args[2], out, errOut)
	}
	if len(args) == 1 && (args[0] == "help" || args[0] == "--help") {
		fmt.Fprint(out, "ConfigChief "+Version+"\n"+usage)
		return 0
	}
	fmt.Fprint(errOut, "ConfigChief "+Version+"\n"+usage)
	return 2
}

func loadPack(dir string, errOut io.Writer) (*pack.Pack, bool) {
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		fmt.Fprintf(errOut, "%s: パックのフォルダがありません\n", dir)
		return nil, false
	}
	p, err := pack.Load(os.DirFS(dir))
	if err != nil {
		fmt.Fprintf(errOut, "パックを読めません:\n%v\n", err)
		return nil, false
	}
	return p, true
}

// packTest checks a pack and runs the tests in its folder (appendix B 7, 8).
func packTest(dir string, flags []string, out, errOut io.Writer) int {
	var lines, dump bool
	for _, f := range flags {
		switch f {
		case "--lines":
			lines = true
		case "--dump":
			dump = true
		default:
			fmt.Fprint(errOut, usage)
			return 2
		}
	}
	p, ok := loadPack(dir, errOut)
	if !ok {
		return 2
	}
	fmt.Fprintf(out, "パック %s %s\n", p.ID, p.Version)
	for _, w := range p.Warnings {
		fmt.Fprintf(out, "  警告: %s\n", w)
	}
	sc, err := schemas.Load()
	if err != nil {
		fmt.Fprintf(errOut, "表の定義を読めません:\n%v\n", err)
		return 2
	}
	lint := pack.Lint(p, sc)
	for _, s := range lint.Problems {
		fmt.Fprintf(out, "  誤り: %s\n", s)
	}
	if len(lint.Valueless) > 0 {
		fmt.Fprintf(out, "  値を入れない規則 %d 件（README.md の一覧と見比べます）:\n", len(lint.Valueless))
		for _, s := range lint.Valueless {
			fmt.Fprintf(out, "    %s\n", s)
		}
	}
	cases, err := packtest.RunImport(p, sc, os.DirFS(dir))
	if err != nil {
		fmt.Fprintf(errOut, "試験を実行できません: %v\n", err)
		return 2
	}
	if len(cases) == 0 {
		fmt.Fprintln(out, "tests/import に試験がありません")
	}
	failed := len(lint.Problems)
	for _, c := range cases {
		fmt.Fprint(out, packtest.Report(c))
		if lines {
			fmt.Fprint(out, packtest.Lines(c))
		}
		if dump {
			fmt.Fprint(out, c.Result.Data.Dump())
		}
		if len(c.Failures) > 0 {
			failed++
		}
	}
	renders, err := packtest.RunRender(p, sc, os.DirFS(dir))
	if err != nil {
		fmt.Fprintf(errOut, "生成の試験を実行できません: %v\n", err)
		return 2
	}
	if len(p.Sections) > 0 && len(renders) == 0 {
		fmt.Fprintln(out, "tests/render に試験がありません")
	}
	for _, c := range renders {
		fmt.Fprint(out, packtest.ReportRender(c))
		if dump {
			fmt.Fprint(out, c.Output.Text)
		}
		if len(c.Failures) > 0 {
			failed++
		}
	}
	if failed > 0 {
		return 1
	}
	return 0
}

// packColumns lists the tables and columns a pack fills, for comparing with
// the table definitions.
func packColumns(dir string, out, errOut io.Writer) int {
	p, ok := loadPack(dir, errOut)
	if !ok {
		return 2
	}
	seen := map[string]bool{}
	var names []string
	for _, w := range p.Writes() {
		name := w.Table + "." + w.Column
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintln(out, n)
	}
	return 0
}

// writes are the commands that write the case beyond a new folder of out/
// (spec 14.8.3): a case used by a newer ConfigChief stops them.
var writes = map[string]bool{"book": true, "clone": true, "generate": true, "check": true, "approve": true, "judge": true, "customer": true, "fill": true}

// writing: the command that runs writes the case.
var writing bool

// held is the mark of the case the command has open (spec 14.8.2).
var held *project.Held

func hasArg(args []string, a string) bool {
	for _, x := range args {
		if x == a {
			return true
		}
	}
	return false
}

func head(args []string, n int) []string {
	if len(args) < n {
		return args
	}
	return args[:n]
}
