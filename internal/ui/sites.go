package ui

// SitesView is the list of many sites (spec 12.5): the devices of one
// source side by side.
type SitesView struct {
	Title  string   `json:"title"`
	Counts []Pair   `json:"counts"`
	Notes  []string `json:"notes"`
	Tables []Table  `json:"tables"`
	Report Files    `json:"report"`
}

// Sites sets the devices of one source ("book", "old", "new" or "logs")
// side by side, as the command "sites" does: the report is written. The
// secrets are masked on the screen whatever the case says.
func (s *Session) Sites(source string) (*SitesView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	now := s.now()
	o, err := e.Sites(source, now)
	if err != nil {
		return nil, err
	}
	d := e.SitesDoc(o, now)
	v := &SitesView{Title: d.Title, Counts: pairs(d.Counts), Notes: strs(d.Notes), Tables: []Table{}, Report: Files{o.Report.XLSX, o.Report.HTML}}
	for _, sh := range d.Sheets {
		v.Tables = append(v.Tables, table(sh))
	}
	return v, nil
}
