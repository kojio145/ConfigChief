package project

import (
	"strings"
	"testing"
)

func TestCleanName(t *testing.T) {
	for in, want := range map[string]string{
		"  拠点ルータ更改 ": "拠点ルータ更改",
		"rt-a01 update":    "rt-a01 update",
		"CONSOLE":          "CONSOLE",
		"案件.v2":            "案件.v2",
	} {
		if got, err := CleanName(in); err != nil || got != want {
			t.Errorf("%q: %q, %v (want %q)", in, got, err, want)
		}
	}
	for in, word := range map[string]string{
		"":                       "入れてください",
		"   ":                    "入れてください",
		"a/b":                    "「/」",
		`a\b`:                    `「\」`,
		"a:b":                    "「:」",
		"what?":                  "「?」",
		"a|b":                    "「|」",
		"end.":                   "「.」で終え",
		"..":                     "使えません",
		"con":                    "Windows",
		"COM1.txt":               "Windows",
		"lpt9":                   "Windows",
		"a\tb":                   "制御文字",
		strings.Repeat("あ", 101): "長すぎ",
	} {
		if _, err := CleanName(in); err == nil || !strings.Contains(err.Error(), word) {
			t.Errorf("%q: %v (want %q)", in, err, word)
		}
	}
}
