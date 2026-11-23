// Command docgen writes the two reference pages of the manual from the
// definitions the program itself uses: docs/tables.md (the sheets and the
// columns of the parameter book) and docs/checks.md (the checks). It also
// makes the HTML pages of the project site from the Markdown of docs/.
//
//	go run ./cmd/docgen          (from the root of the repository)
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/schemas"
)

func main() {
	pages, err := Pages()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for name, text := range pages {
		if err := os.WriteFile(filepath.Join(docsDir, name), []byte(text), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("docs/" + name)
	}
}

// Pages gives the pages by their file names.
func Pages() (map[string]string, error) {
	sc, err := schemas.Load()
	if err != nil {
		return nil, err
	}
	pages := map[string]string{"tables.md": tables(sc), "checks.md": checks()}
	// The pages of the site, from the Markdown of docs/ (html.go). The two
	// lists are made above; the others are written by hand.
	for _, name := range []string{"manual.md", "pack-guide.md", "tables.md", "checks.md", "approval.md"} {
		md, ok := pages[name]
		if !ok {
			data, err := os.ReadFile(filepath.Join(docsDir, name))
			if err != nil {
				return nil, err
			}
			md = strings.ReplaceAll(string(data), "\r\n", "\n")
		}
		body, title := render(md)
		pages[strings.TrimSuffix(name, ".md")+".html"] = page(title, body)
	}
	return pages, nil
}

// docsDir is where docs/ is, from the folder the command runs in: the root
// of the repository (go run ./cmd/docgen), or cmd/docgen (go test).
var docsDir = "docs"

func cell(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "|", `\|`), "\n", " ")
}

// kind tells the type of a column in the words of the manual.
func kind(c *schema.Column) string {
	words := map[string]string{
		"string": "文字", "text": "文章", "int": "数", "bool": "有効・無効", "enum": "選択", "secret": "秘密の値",
		"ip4": "IPv4 アドレス", "ip6": "IPv6 アドレス", "cidr4": "IPv4 アドレス/長さ", "cidr6": "IPv6 アドレス/長さ",
		"net4": "IPv4 ネットワーク", "net6": "IPv6 ネットワーク", "ifname": "IF 名", "ref": "参照",
		"ip": "IPv4 か IPv6 のアドレス", "net": "IPv4 か IPv6 のネットワーク",
	}
	s := words[c.Type]
	if s == "" {
		s = c.Type
	}
	switch {
	case c.Type == "enum":
		s += "（" + strings.Join(c.Values, "、") + "）"
	case c.Type == "ref" && c.Ref != "":
		s += "（" + c.Ref + "）"
	case c.Type == "int" && (c.Min != nil || c.Max != nil):
		lo, hi := "", ""
		if c.Min != nil {
			lo = strconv.Itoa(*c.Min)
		}
		if c.Max != nil {
			hi = strconv.Itoa(*c.Max)
		}
		s += "（" + lo + "〜" + hi + "）"
	}
	if len(c.Also) > 0 {
		s += "。ほかに " + strings.Join(c.Also, "、")
	}
	if c.List {
		s += "。複数可"
	}
	return s
}

func tables(sc *schema.Set) string {
	var b strings.Builder
	b.WriteString("# パラメータブックのシートと列\n\n")
	b.WriteString("このページは、プログラムが使っている定義から作っています（`go run ./cmd/docgen`）。手で直さないでください。\n\n")
	b.WriteString("- シートの名前と見出しは、表示名でも内部名でも読みます\n")
	b.WriteString("- 「必」= 必須。「キー」= 行を見分ける列（同じキーの行は 2 つ書けません）\n")
	b.WriteString("- 「明示」= 空のときに、書き出し先のメーカーの既定値を埋めて必ず書き出す列\n")
	b.WriteString("- どのシートも「機器」の列に、ホスト名・`*`（全機器）・`@グループ名` を書けます（「機器」と「接続」のシートを除く）\n\n")
	b.WriteString("## シートの一覧\n\n| シート | 内部名 | キー |\n|---|---|---|\n")
	for _, t := range sc.Tables {
		fmt.Fprintf(&b, "| [%s](#%s) | %s | %s |\n", t.Title.JA, t.Name, t.Name, strings.Join(t.Key, "、"))
	}
	for _, t := range sc.Tables {
		fmt.Fprintf(&b, "\n## %s\n\n<a id=\"%s\"></a>内部名 `%s`\n\n| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |\n|---|---|---|---|---|---|---|\n", t.Title.JA, t.Name, t.Name)
		for _, c := range t.Columns {
			mark := func(on bool) string {
				if on {
					return "○"
				}
				return ""
			}
			note := c.Note.JA
			if c.Secret || c.Type == "secret" {
				note = strings.TrimSpace(note + " 画面とレポートでは伏せます")
			}
			if !c.Generate {
				note = strings.TrimSpace(note + " コンフィグには書きません")
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n", cell(c.Title.JA), c.Name, cell(kind(c)), mark(c.Required), mark(t.IsKey(c.Name)), mark(c.Explicit), cell(note))
		}
	}
	return b.String()
}

func checks() string {
	var b strings.Builder
	b.WriteString("# チェックの一覧\n\n")
	b.WriteString("このページは、プログラムが使っている定義から作っています（`go run ./cmd/docgen`）。手で直さないでください。\n\n")
	b.WriteString("- 重大度: E = エラー、W = 警告、I = 情報。「指定」= 案件のルールに書いた重大度\n")
	b.WriteString("- 頭の字: V = 値、X = 対向（2 台の間）、G = 生成、M = 移行（旧との違い・対応）、S = 外部一覧、L = 投入ログと反映確認\n\n")
	b.WriteString("| ID | 重大度 | 内容 | 直し方 |\n|---|---|---|---|\n")
	sev := map[string]string{check.Error: "E", check.Warning: "W", check.Info: "I", "": "指定"}
	for _, r := range check.AllRules() {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", r.ID, sev[r.Severity], cell(r.Title), cell(r.Fix))
	}
	return b.String()
}
