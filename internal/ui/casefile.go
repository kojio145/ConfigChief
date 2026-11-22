package ui

import (
	"errors"
	"os"

	"github.com/kojio145/configchief/internal/casefile"
	"github.com/kojio145/configchief/internal/ops"
)

// The case written out to one file and read back (spec 14.9).

// CaseExportView is what the screen says of a case written out.
type CaseExportView struct {
	Path        string   `json:"path"` // where the file was saved ("" = not yet)
	Name        string   `json:"name"`
	Size        int      `json:"size"`
	Files       int      `json:"files"`
	Encrypted   bool     `json:"encrypted"`
	NoSecrets   bool     `json:"noSecrets"`
	Values      int      `json:"values"`
	Masked      int      `json:"masked"`
	MaskedFiles []string `json:"maskedFiles"`
	Left        []string `json:"left"`
	Notes       []string `json:"notes"`
}

// CaseExport writes the open case out (spec 14.9.1) and gives the bytes to
// save with what to tell. Writing a case out reads it: a case that is only
// read can be written out.
func (s *Session) CaseExport(noSecrets bool, password string) (*CaseExportView, []byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, nil, err
	}
	x, err := e.ExportCase(noSecrets, password, s.now())
	if err != nil {
		return nil, nil, err
	}
	return &CaseExportView{Name: x.Name, Size: len(x.Data), Files: x.Files, Encrypted: x.Encrypted, NoSecrets: x.NoSecrets,
		Values: x.Values, Masked: x.Masked, MaskedFiles: strs(x.MaskedFiles), Left: strs(x.Left), Notes: strs(x.Notes)}, x.Data, nil
}

// CaseFileView is a case file chosen to be read.
type CaseFileView struct {
	Path   string `json:"path"`
	Sealed bool   `json:"sealed"` // a password is needed
}

// CaseFile looks at a chosen file: whether it is a case file, and sealed.
func CaseFile(path string) (*CaseFileView, error) {
	data, err := readCaseFile(path)
	if err != nil {
		return nil, err
	}
	if !casefile.Is(data) {
		return nil, errors.New("ConfigChief の案件のファイル（.ccase）ではありません")
	}
	return &CaseFileView{Path: path, Sealed: casefile.Sealed(data)}, nil
}

// CaseCheck tells whether the password opens the file (nil = it does).
func CaseCheck(path, password string) error {
	data, err := readCaseFile(path)
	if err != nil {
		return err
	}
	_, _, err = casefile.Unpack(data, password)
	return err
}

func readCaseFile(path string) ([]byte, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.Size() > 5<<30 {
		return nil, errors.New("ファイルが大きすぎます")
	}
	return os.ReadFile(path)
}

// CaseImportView is a case read back and opened.
type CaseImportView struct {
	Home      *Home  `json:"home"`
	Dir       string `json:"dir"`
	Name      string `json:"name"`
	Files     int    `json:"files"`
	NoSecrets bool   `json:"noSecrets"`
	Written   string `json:"written"`
	Version   string `json:"version"`
}

// CaseImport spreads a case file under parent and opens the case (spec
// 14.9.2).
func (s *Session) CaseImport(path, password, parent string) (*CaseImportView, error) {
	data, err := readCaseFile(path)
	if err != nil {
		return nil, err
	}
	got, err := ops.ImportCase(data, password, parent)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h, err := s.open(got.Dir)
	if err != nil {
		return nil, err
	}
	return &CaseImportView{Home: h, Dir: got.Dir, Name: got.Name, Files: got.Files, NoSecrets: got.NoSecrets, Written: got.Written, Version: got.Version}, nil
}
