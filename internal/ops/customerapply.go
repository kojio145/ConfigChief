package ops

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/listimport"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/sheet"
)

// ReflectRequest names the findings of the customer's lists whose values go
// into the book: by their fingerprints (or enough of their beginnings), or
// every finding of a check that is not approved, of the devices if given.
type ReflectRequest struct {
	Findings      []string
	Rule, Devices string
}

// Reflected is one finding with what became of it.
type Reflected struct {
	Finding Judged
	// Skipped tells why its value was not written ("" = it was).
	Skipped string
}

// ReflectOutcome is what Reflect did.
type ReflectOutcome struct {
	Items  []Reflected
	Rows   int // rows added to the book
	Cells  int // cells of the book that were changed
	Backup string
}

// Skipped counts the findings whose value was not written.
func (o *ReflectOutcome) Skipped() int {
	n := 0
	for _, it := range o.Items {
		if it.Skipped != "" {
			n++
		}
	}
	return n
}

// Reflect writes what the customer's lists say into the parameter book, for
// the findings that are named (spec 6.5.4): a row the book lacks is added
// (S-501), a cell that differs is changed (S-503). Only a cell of a row
// written for the device itself is changed.
func (e *Env) Reflect(req ReflectRequest, now time.Time) (*ReflectOutcome, error) {
	if (len(req.Findings) == 0) == (req.Rule == "") {
		return nil, fmt.Errorf("反映する指摘の指紋か、チェックの ID（S-501・S-503）のどちらか一方を指定してください")
	}
	if req.Rule != "" && req.Rule != "S-501" && req.Rule != "S-503" {
		return nil, fmt.Errorf("ブックに反映できるのは、S-501（一覧にだけある行）と S-503（値が違うセル）です")
	}
	o, err := e.Customer(false, now)
	if err != nil {
		return nil, err
	}
	if o.Book == nil {
		return nil, fmt.Errorf("%s がありません。先にパラメータブックを作ってください", e.Proj.Config.Book)
	}
	var picked []*Judged
	if req.Rule != "" {
		devices := strings.TrimSpace(req.Devices)
		if devices == "" {
			devices = "*"
		}
		for i := range o.Verdicts.Items {
			f := &o.Verdicts.Items[i]
			if f.RuleID == req.Rule && f.Approval == nil && scopeHas(o.Book.Data, devices, f.Device) {
				picked = append(picked, f)
			}
		}
		if len(picked) == 0 {
			return nil, fmt.Errorf("その範囲に、反映できる未承認の指摘はありません")
		}
	}
	seen := map[string]bool{}
	for _, want := range req.Findings {
		want = strings.TrimSpace(want)
		var found []*Judged
		for i := range o.Verdicts.Items {
			f := &o.Verdicts.Items[i]
			if f.Key == want {
				found = []*Judged{f}
				break
			}
			if len(want) >= 6 && strings.HasPrefix(f.Key, want) {
				found = append(found, f)
			}
		}
		switch {
		case len(found) == 0:
			return nil, fmt.Errorf("指紋 %s の指摘は、今の結果にありません（値を直したか、根拠の値が変わりました）", want)
		case len(found) > 1:
			return nil, fmt.Errorf("指紋 %s で始まる指摘が %d 件あります。もう少し長く書いてください", want, len(found))
		}
		if !seen[found[0].Key] {
			seen[found[0].Key] = true
			picked = append(picked, found[0])
		}
	}

	rows := map[string]*RowFinding{}
	for i := range o.OnlyList {
		rows[o.OnlyList[i].Finding.Key] = &o.OnlyList[i]
	}
	cells := map[string]*CellFinding{}
	for i := range o.Cells {
		cells[o.Cells[i].Finding.Key] = &o.Cells[i]
	}
	out := &ReflectOutcome{}
	ch := book.Change{Data: model.NewDataset("customer")}
	for _, f := range picked {
		it := Reflected{Finding: *f}
		switch {
		case rows[f.Key] != nil:
			ch.Data.Adopt(rows[f.Key].Row)
		case cells[f.Key] != nil:
			x := cells[f.Key]
			// The row the book has for the device itself (not one for "*" or
			// a group, which other devices share).
			if o.Book.Data.Tables[x.Table].Find(x.ARow.Key) == nil {
				it.Skipped = fmt.Sprintf("ブックに、機器 %s の名前で書いた行がありません（値は * や @グループ の行から来ています）。1 台だけ変えるには、ブックでその機器の行を分けてから、もう一度反映してください", x.Device)
				break
			}
			ch.Cells = append(ch.Cells, book.CellSet{Table: x.Table, Key: x.ARow.Key, Column: x.Column, Cell: x.A})
		default:
			it.Skipped = fmt.Sprintf("%s は、ブックに反映する指摘ではありません（反映できるのは S-501 と S-503）", f.RuleID)
		}
		out.Items = append(out.Items, it)
	}
	if len(ch.Cells) == 0 && len(ch.Data.Tables) == 0 {
		return out, nil
	}
	old, _, err := e.readBook()
	if err != nil {
		return nil, err
	}
	next, sum, err := book.Update(old, e.Proj.Config.Book, e.Schema, e.lang(), ch)
	if err != nil {
		return nil, err
	}
	out.Rows, out.Cells = sum.Rows(), sum.Cells
	if sum.Changed() {
		if out.Backup, err = e.saveBook(now, next); err != nil {
			return nil, err
		}
	}
	e.Proj.Log(now, "customer --apply: 行 %d、セル %d をブックに反映しました（反映しなかった指摘 %d 件、控え %s）", out.Rows, out.Cells, out.Skipped(), out.Backup)
	return out, nil
}

// DraftMapping writes a first column mapping for a list of input/customer
// (spec 15): mappings/<name of the list>.yaml. It gives the path. A file that
// is there is not written over.
func (e *Env) DraftMapping(file, sheetName string) (string, error) {
	file = path.Base(strings.ReplaceAll(strings.TrimSpace(file), `\`, "/"))
	inputs, err := e.Proj.Inputs("customer")
	if err != nil {
		return "", err
	}
	rel := ""
	for _, p := range inputs {
		if strings.EqualFold(path.Base(p), file) {
			rel = p
		}
	}
	if rel == "" {
		return "", fmt.Errorf("input/customer に「%s」がありません", file)
	}
	bk, err := sheet.ReadFile(e.Proj.Path(rel))
	if err != nil {
		return "", fmt.Errorf("%s: %v", rel, err)
	}
	s, err := listimport.PickSheet(bk, &listimport.Rule{Sheet: sheetName})
	if err != nil {
		return "", fmt.Errorf("%s: %v", rel, err)
	}
	name := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
	out := "mappings/" + name + ".yaml"
	if _, err := os.Stat(e.Proj.Path(out)); err == nil {
		return "", fmt.Errorf("%s は、すでにあります（書き換えません。直すか、消してからやり直してください）", out)
	}
	if err := os.MkdirAll(e.Proj.Path("mappings"), 0o755); err != nil {
		return "", fmt.Errorf("mappings を作れません: %v", err)
	}
	if err := os.WriteFile(e.Proj.Path(out), []byte(listimport.Draft(s, path.Base(rel))), 0o644); err != nil {
		return "", fmt.Errorf("%s を書けません: %v", out, err)
	}
	return out, nil
}
