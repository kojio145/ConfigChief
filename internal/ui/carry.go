package ui

import (
	"fmt"
	"strconv"
)

// CarryView is the report of unhandled lines that may leave the site (spec
// 8.6), as the page of the unhandled lines tells of it.
type CarryView struct {
	Summary string   `json:"summary"`
	Notes   []string `json:"notes"`
	// Hidden are the words that were hidden as names: words of the site,
	// shown here and written to no file.
	Hidden Table `json:"hidden"`
	Report Files `json:"report"`
}

// CarryOut writes the report of the unhandled lines of a source ("old" or
// "new") with the values and the names hidden, as the command "export --what
// unhandled" does.
func (s *Session) CarryOut(source string) (*CarryView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := checkSource(source); err != nil {
		return nil, err
	}
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	o, err := e.CarryOut(source, s.now())
	if err != nil {
		return nil, err
	}
	lines := 0
	for _, x := range o.Shapes {
		lines += x.Lines
	}
	v := &CarryView{
		Summary: fmt.Sprintf("持ち出し用レポートを出しました: 形 %d、未対応の行 %d。人に渡す前に、すべての行を目で確かめてください。", len(o.Shapes), lines),
		Notes:   []string{},
		Hidden:  Table{Name: "名前として伏せた語", Header: []string{"語", "行数"}, Rows: [][]string{}, Marks: []string{}, Code: []int{0}},
		Report:  Files{o.Report.XLSX, o.Report.HTML},
	}
	for _, f := range o.Unread {
		v.Notes = append(v.Notes, f+" は取り込めなかったので、その行は入っていません")
	}
	for _, h := range o.Hidden {
		v.Hidden.Rows = append(v.Hidden.Rows, []string{h.Word, strconv.Itoa(h.Lines)})
		v.Hidden.Marks = append(v.Hidden.Marks, "")
	}
	return v, nil
}
