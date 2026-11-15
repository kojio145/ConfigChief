package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func sample() *Doc {
	return &Doc{
		Title:  "取り込みの結果",
		Facts:  []Pair{{"案件", "sample <case>"}},
		Counts: []Pair{{"未対応", "1"}},
		Notes:  []string{"未対応の行にある秘密の値は伏せられません"},
		Sheets: []Sheet{
			{
				Name:   "行の一覧",
				Lead:   "旧コンフィグのすべての行です",
				Header: []string{"機器", "行番号", "文面", "状態"},
				Rows: [][]string{
					{"rt-a01", "1", "hostname rt-a01", "取り込み済み"},
					{"rt-a01", "2", "!", "無視"},
					{"rt-a01", "3", " sample <b>&\"x\"", "未対応"},
				},
				Marks: []string{"", MarkIgnored, MarkUnknown},
				Code:  []int{2},
			},
			{Name: "注意", Header: []string{"機器", "内容"}},
		},
	}
}

func TestHTML(t *testing.T) {
	h := sample().HTML()
	for _, want := range []string{
		"<title>取り込みの結果</title>",
		"<td>sample &lt;case&gt;</td>",
		"<a href=\"#s1\">行の一覧（3）</a>",
		"<h2 id=\"s1\">行の一覧（3 件）</h2>",
		"<tr class=\"unknown\"><td>rt-a01</td><td>3</td><td class=\"code\"> sample &lt;b&gt;&amp;&#34;x&#34;</td><td>未対応</td></tr>",
		"<tr class=\"ignored\">",
		"<p class=\"none\">ありません</p>",
	} {
		if !strings.Contains(h, want) {
			t.Errorf("the HTML lacks %s", want)
		}
	}
	// Nothing is loaded from outside.
	for _, bad := range []string{"<script", "<link", "<img", "http://", "https://", "url("} {
		if strings.Contains(h, bad) {
			t.Errorf("the HTML holds %s", bad)
		}
	}
	if h != sample().HTML() {
		t.Error("the same report gave another text")
	}
}

func TestXLSX(t *testing.T) {
	var a, b bytes.Buffer
	if err := sample().WriteXLSX(&a); err != nil {
		t.Fatal(err)
	}
	if err := sample().WriteXLSX(&b); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Error("the same report gave other bytes")
	}
	f, err := excelize.OpenReader(bytes.NewReader(a.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got := strings.Join(f.GetSheetList(), ","); got != "まとめ,行の一覧,注意" {
		t.Errorf("sheets = %s", got)
	}
	rows, err := f.GetRows("まとめ")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range rows {
		got = append(got, strings.Join(r, "="))
	}
	if want := "項目=内容|レポート=取り込みの結果|案件=sample <case>|未対応=1|注意=未対応の行にある秘密の値は伏せられません"; strings.Join(got, "|") != want {
		t.Errorf("summary = %q", got)
	}
	if v, _ := f.GetCellValue("行の一覧", "C4"); v != " sample <b>&\"x\"" {
		t.Errorf("C4 = %q", v)
	}
}
