package ops

import (
	"fmt"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/model"
)

// The helps for a new build (spec 3.2 and 6.6): an empty parameter book, and
// the rows of one device copied to new host names.

// NewBook writes an empty parameter book: a sheet for every table, with the
// headings, their notes and their drop-down lists. A book that is there is
// never replaced.
func (e *Env) NewBook(now time.Time) (*book.Summary, error) {
	old, _, err := e.readBook()
	if err != nil {
		return nil, err
	}
	file := e.Proj.Config.Book
	if old != nil {
		return nil, fmt.Errorf("パラメータブック %s が既にあります（空のブックは、ブックが無い案件にだけ作ります）", file)
	}
	next, sum, err := book.Update(nil, file, e.Schema, e.lang(), book.Change{})
	if err != nil {
		return nil, err
	}
	if _, err := e.saveBook(now, next); err != nil {
		return nil, err
	}
	e.Proj.Log(now, "book --new: 空のパラメータブックを作りました")
	return sum, nil
}

// CloneOutcome tells what CloneDevice did.
type CloneOutcome struct {
	From    string
	To      []string
	Summary *book.Summary
	Backup  string
	// Shared counts the rows that reach the device through "*" or a group:
	// they are not copied (the new devices get them the same way if they are
	// in the group).
	Shared int
}

// CloneDevice copies every row of one device, on every sheet of the tables,
// to new host names (spec 6.6). The values are copied as they are: what has
// to differ between the devices (addresses, descriptions) is for a person to
// change, and the checks tell what clashes.
func (e *Env) CloneDevice(from string, to []string, now time.Time) (*CloneOutcome, error) {
	old, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	if bk == nil {
		return nil, fmt.Errorf("パラメータブック %s がありません", e.Proj.Config.Book)
	}
	from = strings.TrimSpace(from)
	hosts := map[string]bool{}
	if t := bk.Data.Tables["devices"]; t != nil {
		for _, r := range t.Rows {
			hosts[r.Get("hostname")] = true
		}
	}
	if from == "" || !hosts[from] {
		return nil, fmt.Errorf("複製元の機器「%s」が、シート「機器」にありません", from)
	}
	var names []string
	seen := map[string]bool{}
	for _, h := range to {
		h = strings.TrimSpace(h)
		switch {
		case h == "":
			continue
		case strings.ContainsAny(h, " \t*@,") || h == "(管理外)":
			return nil, fmt.Errorf("新しいホスト名「%s」は使えません（空白・*・@・カンマは書けません）", h)
		case hosts[h]:
			return nil, fmt.Errorf("新しいホスト名「%s」は、シート「機器」に既にあります", h)
		case seen[h]:
			return nil, fmt.Errorf("新しいホスト名「%s」が 2 回あります", h)
		}
		seen[h] = true
		names = append(names, h)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("新しいホスト名がありません")
	}

	out := &CloneOutcome{From: from, To: names}
	data := model.NewDataset("book")
	for _, t := range e.Schema.Tables {
		tab := bk.Data.Tables[t.Name]
		dev := t.DeviceColumn()
		if tab == nil || dev == "" {
			continue // connections: a row of two devices is not of one device
		}
		for _, r := range tab.Rows {
			v := r.Get(dev)
			if v != from {
				if t.Name != "devices" && (v == "*" || strings.HasPrefix(v, "@")) {
					out.Shared++
				}
				continue
			}
			for _, h := range names {
				c := &model.Row{Table: r.Table, Cells: map[string]*model.Cell{}, Extra: r.Extra}
				for name, cell := range r.Cells {
					cp := *cell
					cp.Values = append([]string{}, cell.Values...)
					c.Cells[name] = &cp
				}
				for _, k := range r.Key {
					if k.Name == dev {
						k.Value = h
						c.Cells[dev].Values = []string{h}
					}
					c.Key = append(c.Key, k)
				}
				data.Adopt(c)
			}
		}
	}
	file := e.Proj.Config.Book
	next, sum, err := book.Update(old, file, e.Schema, e.lang(), book.Change{Data: data})
	if err != nil {
		return nil, err
	}
	out.Summary = sum
	if sum.Changed() {
		if out.Backup, err = e.saveBook(now, next); err != nil {
			return nil, err
		}
	}
	e.Proj.Log(now, "clone --from %s --to %s: %d 行を足しました（控え %s）", from, strings.Join(names, ","), sum.Rows(), out.Backup)
	return out, nil
}
