package listimport

import (
	"fmt"
	"strings"

	"github.com/kojio145/configchief/internal/sheet"
)

// deviceWords are the words a header holds for the columns of the table of
// the devices, tried in this order, and the words that rule a header out.
var deviceWords = []struct {
	to       string
	words    []string
	notWords []string
}{
	{to: "hostname", words: []string{"ホスト名", "hostname", "機器名", "装置名", "ノード名", "nodename", "devicename", "機器名称", "ホスト"}, notWords: []string{"ip", "アドレス", "旧", "現行"}},
	{to: "site", words: []string{"拠点", "設置場所", "設置先", "サイト", "site"}, notWords: []string{"コード", "code"}},
	{to: "model", words: []string{"型番", "型式", "機種", "モデル", "model", "製品名"}, notWords: []string{"コード", "code", "数", "種別", "区分"}},
	{to: "os_version", words: []string{"osバージョン", "os版", "バージョン", "version", "ファームウェア", "firmware"}},
}

// Suggest guesses the start of a rule for a sheet: the header row (when the
// rule still has none) and, by the header names, the columns of the table of
// the devices. It is for a person to check, never taken as is.
func Suggest(s *sheet.Sheet, r Rule) (headerRow int, devices []Field, t *Table) {
	if r.HeaderRow < 1 {
		r.HeaderRow = GuessHeaderRow(FilledRows(s, &r))
		r.DataRow = 0
	}
	t, err := ReadTable(s, &r)
	if err != nil {
		return r.HeaderRow, nil, nil
	}
	used := map[int]bool{}
	for _, dw := range deviceWords {
		for _, h := range t.Headers {
			if used[h.Col] || h.Name == "" || isRepeated(h.Name) {
				continue
			}
			w := normWord(h.Name)
			if !hasAny(w, dw.words) || hasAny(w, dw.notWords) {
				continue
			}
			used[h.Col] = true
			devices = append(devices, Field{To: dw.to, Column: h.Name, Steps: []Step{{Type: "trim"}}})
			break
		}
	}
	return r.HeaderRow, devices, t
}

func yamlText(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// Draft writes the text of a first rule file for a sheet of a list: where
// the table is, the headers as comments, and the table of the devices if a
// header looks like the host name. The rest is for a person to write.
func Draft(s *sheet.Sheet, file string) string {
	row, devices, t := Suggest(s, Rule{})
	var sb strings.Builder
	sb.WriteString("# 外部一覧の列対応（仕様書 6.5）。下書きです: 見出しの行と行き先を確かめて直してください。\n")
	sb.WriteString("version: 1\n")
	fmt.Fprintf(&sb, "file: %s\n", yamlText(file))
	fmt.Fprintf(&sb, "sheet: %s\n", yamlText(s.Name))
	fmt.Fprintf(&sb, "header_row: %d\n", row)
	sb.WriteString("names: new          # 一覧の機器名が今の（旧の）名前なら old\n")
	if t != nil {
		sb.WriteString("# この表の見出し:\n")
		for _, h := range t.Headers {
			if h.Name != "" {
				fmt.Fprintf(&sb, "#   %s 列: %s\n", ColumnLetter(h.Col), h.Name)
			}
		}
	}
	sb.WriteString("rows:\n")
	host := ""
	for _, f := range devices {
		if f.To == "hostname" {
			host = f.Column
		}
	}
	if host == "" {
		sb.WriteString("  # 機器名らしい見出しが見つかりませんでした。column に見出しを書いてください。\n")
		sb.WriteString("  - table: devices\n    columns:\n      - {to: hostname, column: \"\", steps: [{type: trim}]}\n")
	} else {
		sb.WriteString("  - table: devices\n    columns:\n")
		for _, f := range devices {
			fmt.Fprintf(&sb, "      - {to: %s, column: %s, steps: [{type: trim}]}\n", f.To, yamlText(f.Column))
		}
	}
	sb.WriteString("  # IF の行の例（要るものだけ、見出しを書き換えて使います）:\n")
	sb.WriteString("  # - table: interfaces\n  #   label: WAN 側\n  #   columns:\n")
	fmt.Fprintf(&sb, "  #     - {to: device, column: %s, steps: [{type: trim}]}\n", yamlText(host))
	sb.WriteString("  #     - {to: name, value: GigabitEthernet0/0}\n")
	sb.WriteString("  #     - {to: ipv4, template: \"{アドレスの見出し}/{マスクの見出し}\", steps: [{type: cidr}]}\n")
	return sb.String()
}
