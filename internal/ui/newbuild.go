package ui

import (
	"fmt"
	"strings"
)

// The helps for a new build (spec 3.2 and 6.6), as the commands "book --new"
// and "clone" do them. Each gives the lines to show.

// NewBook writes the empty parameter book.
func (s *Session) NewBook() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	if _, err := e.NewBook(s.now()); err != nil {
		return nil, err
	}
	return []string{
		fmt.Sprintf("空のパラメータブック %s を作りました（見出しのセルに、列の説明があります）", e.Proj.Config.Book),
		"1 台目をシート「機器」から書いてください。似た機器は「機器の複製」で増やせます",
	}, nil
}

// CloneDevice copies the rows of one device to new host names (separated by
// commas, spaces or line breaks).
func (s *Session) CloneDevice(from, to string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	names := strings.FieldsFunc(to, func(r rune) bool {
		return r == ',' || r == '、' || r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\u3000'
	})
	o, err := e.CloneDevice(from, names, s.now())
	if err != nil {
		return nil, err
	}
	out := []string{fmt.Sprintf("機器 %s の行を、%s に複製しました（%d 行。控え: %s）", o.From, strings.Join(o.To, "、"), o.Summary.Rows(), o.Backup)}
	if o.Shared > 0 {
		out = append(out, fmt.Sprintf("機器の列が * か @グループ の行（%d 行）は複製していません（新しい機器にも、同じ決まりで当たります）", o.Shared))
	}
	return append(out, "値は元の機器のままです。アドレスや説明など、機器ごとに違う値を、画面の表（「表で編集」）か Excel で直してください（重なりは「チェックだけ」で分かります）"), nil
}
