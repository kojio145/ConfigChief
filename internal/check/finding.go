package check

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// The severities (spec 11.2).
const (
	Error   = "error"
	Warning = "warning"
	Info    = "info"
)

// Finding is one thing a check found (spec 11.3). It never holds a secret
// value: Seal stands in for it.
type Finding struct {
	RuleID   string
	Severity string
	Device   string    // "" = the case as a whole
	Ref      model.Ref // the place in the design data; the key is the one shown (RefOf)
	Line     int       // line of the generated configuration; 0 if none
	Message  string
	Evidence []model.KV
	// At is the cell of the parameter book the finding is about, as
	// "sheet!C5"; "" if none. It is not part of the key: a row that moved is
	// the same finding.
	At string
	// CaseRule is the rule of the case the finding came from (appendix C 4);
	// "" for a check of the tool.
	CaseRule string
	// Firm tells that the severity is not for the case to change
	// (checks.severity): a rule of the case that could not be run.
	Firm bool
	// Key tells the same finding apart from the others, and stays the same
	// from run to run while nothing about it changes (appendix C 3).
	Key string
}

// New makes a finding and gives it its key.
func New(rule, severity, device string, ref model.Ref, message string, evidence ...model.KV) Finding {
	f := Finding{RuleID: rule, Severity: severity, Device: device, Ref: ref, Message: message, Evidence: evidence}
	f.Key = f.fingerprint()
	return f
}

func (f *Finding) fingerprint() string {
	parts := []string{f.RuleID, f.Device, f.Ref.Table, f.Ref.Key, f.Ref.Column}
	for _, e := range f.Evidence {
		parts = append(parts, e.Name, e.Value)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:8])
}

// Seal is what stands for a secret value in a finding and in a report: the
// same value gives the same text, and the value cannot be read from it.
func Seal(secret string) string {
	if secret == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("configchief\x00" + secret))
	return "（伏せ字 #" + hex.EncodeToString(sum[:4]) + "）"
}

// Shown gives the text of a cell for a finding: sealed for a secret column.
func Shown(c *schema.Column, cell *model.Cell) string {
	if cell == nil {
		return ""
	}
	if c != nil && c.Secret {
		vals := make([]string, len(cell.Values))
		for i, v := range cell.Values {
			vals[i] = Seal(v)
		}
		return strings.Join(vals, ",")
	}
	return cell.Text()
}

// RefOf names a cell of a row for a finding. The key leaves out the device
// (the finding has it) and seals the secret columns.
func RefOf(t *schema.Table, r *model.Row, column string) model.Ref {
	var parts []string
	for _, k := range r.Key {
		if t != nil && k.Name == t.DeviceColumn() {
			continue
		}
		v := k.Value
		if c := t.Column(k.Name); c != nil && c.Secret {
			v = Seal(v)
		}
		parts = append(parts, k.Name+"="+v)
	}
	return model.Ref{Table: r.Table, Key: strings.Join(parts, ","), Column: column}
}

// CellAt gives the place in the parameter book of a cell of a row (of the row
// itself if the cell is empty), as "sheet!C5"; "" if it did not come from the
// book.
func CellAt(r *model.Row, column string) string {
	origins := r.Origins
	if c := r.Cells[column]; c != nil && len(c.Origins) > 0 {
		origins = c.Origins
	}
	for _, o := range origins {
		if (o.Kind == "book" || o.Kind == "common") && o.Sheet != "" {
			return o.Sheet + "!" + o.Addr
		}
	}
	return ""
}
