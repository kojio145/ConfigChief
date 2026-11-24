package project

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// maxName is the longest name of a case, in characters (spec 14.15.2).
const maxName = 100

// CleanName gives the name of a case as the name of its folder (spec
// 14.15.2): the spaces around it go; a name Windows cannot take as a folder
// is refused with the reason.
func CleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", fmt.Errorf("案件名を入れてください")
	case name == "." || name == "..":
		return "", fmt.Errorf("案件名「%s」は使えません", name)
	case utf8.RuneCountInString(name) > maxName:
		return "", fmt.Errorf("案件名が長すぎます（%d 文字まで）", maxName)
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("案件名に、制御文字は使えません")
		}
		if strings.ContainsRune(`\/:*?"<>|`, r) {
			return "", fmt.Errorf("案件名に「%c」は使えません（フォルダ名に使えない字: \\ / : * ? \" < > |）", r)
		}
	}
	if strings.HasSuffix(name, ".") {
		return "", fmt.Errorf("案件名を「.」で終えることはできません")
	}
	base := strings.ToUpper(name)
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	base = strings.TrimRight(base, " ")
	switch base {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return "", fmt.Errorf("案件名「%s」は Windows が使う名前なので、フォルダ名にできません", name)
	}
	return name, nil
}
