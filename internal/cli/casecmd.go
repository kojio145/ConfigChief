package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kojio145/configchief/internal/ops"
)

// The case written out to one file and read back (spec 14.9.1, 14.9.2). The
// password is read from an environment variable the user names, never from
// the command line (where it would stay in the history of the shell).

func passwordFrom(env string, errOut io.Writer) (string, bool) {
	if env == "" {
		return "", true
	}
	p := os.Getenv(env)
	if p == "" {
		fmt.Fprintf(errOut, "環境変数 %s が空です（パスワードを入れてから実行してください）\n", env)
		return "", false
	}
	return p, true
}

func cmdCaseExport(args []string, out, errOut io.Writer) int {
	pos, val, on, err := options(args, []string{"--project", "--out", "--password-env"}, []string{"--no-secrets"})
	if err != nil || len(pos) != 0 || val["--out"] == "" {
		if err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprint(errOut, usage)
		return 2
	}
	password, ok := passwordFrom(val["--password-env"], errOut)
	if !ok {
		return 2
	}
	e, ok := openCase(val["--project"], out, errOut)
	if !ok {
		return 2
	}
	x, err := e.ExportCase(on["--no-secrets"], password, now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	path := val["--out"]
	if !strings.HasSuffix(strings.ToLower(path), ".ccase") {
		path += ".ccase"
	}
	if err := os.WriteFile(path, x.Data, 0o600); err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	sealed := "暗号化していません"
	if x.Encrypted {
		sealed = "パスワードで暗号化しました"
	}
	fmt.Fprintf(out, "%s に書き出しました（%d ファイル・%d バイト。%s）\n", path, x.Files, len(x.Data), sealed)
	if x.NoSecrets {
		fmt.Fprintf(out, "秘密の値を外しました: 値 %d 個、テキストの %d か所（%d ファイル）\n", x.Values, x.Masked, len(x.MaskedFiles))
	}
	for _, n := range x.Notes {
		fmt.Fprintln(out, "注意: "+n)
	}
	if len(x.Left) > 0 {
		fmt.Fprintln(out, "伏せられなかったもの（渡す前に中身を見てください）:")
		for _, l := range x.Left {
			fmt.Fprintln(out, "  "+l)
		}
		return 1
	}
	return 0
}

func cmdCaseImport(args []string, out, errOut io.Writer) int {
	pos, val, _, err := options(args, []string{"--to", "--password-env"}, nil)
	if err != nil || len(pos) != 1 || val["--to"] == "" {
		if err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprint(errOut, usage)
		return 2
	}
	password, ok := passwordFrom(val["--password-env"], errOut)
	if !ok {
		return 2
	}
	data, err := os.ReadFile(pos[0])
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	got, err := ops.ImportCase(data, password, val["--to"])
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	fmt.Fprintf(out, "案件「%s」を %s に広げました（%d ファイル。%s に ConfigChief %s で書き出したもの）\n", got.Name, got.Dir, got.Files, got.Written, got.Version)
	if got.NoSecrets {
		fmt.Fprintln(out, "秘密の値を外した案件です。パラメータブックの **** を本当の値に書き換えてください")
	}
	return 0
}
