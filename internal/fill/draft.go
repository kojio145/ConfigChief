package fill

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/kojio145/configchief/internal/conftree"
)

// A first template made from one device's configuration and its values
// (spec 9.7.5): where a value stands, the variable's name goes.

// DraftValue is a variable and its value for the device.
type DraftValue struct {
	Name  string // as the template writes it ({{Name}})
	Value string
}

// DraftEach is a variable rule with two or more rows for the device: a
// {{#each}} part when the lines of its rows are the same but for the values.
type DraftEach struct {
	Source string
	Names  []string            // the variables, as the rule writes them
	Rows   []map[string]string // name → value
}

// Draft is the first template.
type Draft struct {
	Text string
	// Replaced are the variables put in, with how many times.
	Replaced map[string]int
	Notes    []string
}

const minDraftValue = 3

// MakeDraft writes the values of a configuration as variables.
func MakeDraft(config string, values []DraftValue, eaches []DraftEach) *Draft {
	d := &Draft{Replaced: map[string]int{}}
	lines := conftree.Split(strings.TrimPrefix(config, string(rune(0xFEFF))))
	done := make([]bool, len(lines)) // lines inside a {{#each}} made here
	var outBlocks []block
	for _, e := range eaches {
		b, ok := d.each(lines, done, e)
		if ok {
			outBlocks = append(outBlocks, b)
			for i := b.from; i < b.to; i++ {
				done[i] = true
			}
		}
	}
	// The single values: the longest first, so that a value inside another
	// is not put in first.
	vals := append([]DraftValue{}, values...)
	sort.SliceStable(vals, func(i, j int) bool {
		return utf8.RuneCountInString(vals[i].Value) > utf8.RuneCountInString(vals[j].Value)
	})
	seen := map[string]string{}
	var use []DraftValue
	for _, v := range vals {
		v.Value = strings.TrimSpace(v.Value)
		switch {
		case v.Value == "":
			continue
		case utf8.RuneCountInString(v.Value) < minDraftValue:
			d.Notes = append(d.Notes, fmt.Sprintf("「%s」の値「%s」は短いので置き換えませんでした（ほかの所にも当たるため）", v.Name, v.Value))
			continue
		case boolish(v.Value):
			d.Notes = append(d.Notes, fmt.Sprintf("「%s」の値「%s」は真偽の値なので置き換えませんでした（{{#if %s}} で使えます）", v.Name, v.Value, v.Name))
			continue
		}
		if prev, ok := seen[strings.ToLower(v.Value)]; ok {
			d.Notes = append(d.Notes, fmt.Sprintf("「%s」の値「%s」は「%s」と同じなので、「%s」で置き換えました。違う所があれば直してください", v.Name, v.Value, prev, prev))
			continue
		}
		seen[strings.ToLower(v.Value)] = v.Name
		use = append(use, v)
	}
	for i := range lines {
		if done[i] {
			continue
		}
		for _, v := range use {
			var n int
			lines[i], n = replaceValue(lines[i], v.Value, "{{"+v.Name+"}}")
			d.Replaced[v.Name] += n
		}
	}
	for bi := range outBlocks {
		for li := range outBlocks[bi].lines {
			for _, v := range use {
				var n int
				outBlocks[bi].lines[li], n = replaceValue(outBlocks[bi].lines[li], v.Value, "{{"+v.Name+"}}")
				d.Replaced[v.Name] += n
			}
		}
	}
	for _, v := range use {
		if d.Replaced[v.Name] == 0 {
			delete(d.Replaced, v.Name)
		}
	}
	// Put the {{#each}} parts in, from the last so the line numbers hold.
	sort.Slice(outBlocks, func(i, j int) bool { return outBlocks[i].from > outBlocks[j].from })
	for _, b := range outBlocks {
		body := append([]string{"{{#each " + b.source + "}}"}, b.lines...)
		body = append(body, "{{/each}}")
		lines = append(lines[:b.from], append(body, lines[b.to:]...)...)
	}
	d.Text = strings.Join(lines, "\n") + "\n"
	return d
}

type block struct {
	from, to int // the lines of the configuration it stands for
	source   string
	lines    []string
}

// each finds the lines of each row of a rule and makes them one {{#each}}
// when they follow each other and read the same with the names put in.
func (d *Draft) each(lines []string, done []bool, e DraftEach) (block, bool) {
	fail := func(why string) (block, bool) {
		d.Notes = append(d.Notes, fmt.Sprintf("変数ルール「%s」の %d 行: %s。{{#each}} にしなかったので、その値は置き換えていません", e.Source, len(e.Rows), why))
		return block{}, false
	}
	// The values that tell the rows apart: those not the same in all.
	differ := map[string]bool{}
	for _, n := range e.Names {
		for _, r := range e.Rows[1:] {
			if r[n] != e.Rows[0][n] {
				differ[n] = true
			}
		}
	}
	var groups [][]int
	for ri, r := range e.Rows {
		var at []int
		for i, l := range lines {
			if done[i] {
				continue
			}
			for n := range differ {
				v := strings.TrimSpace(r[n])
				if utf8.RuneCountInString(v) < minDraftValue {
					continue
				}
				if _, k := replaceValue(l, v, "x"); k > 0 {
					at = append(at, i)
					break
				}
			}
		}
		if len(at) == 0 {
			return fail(fmt.Sprintf("%d 行目の値がコンフィグに見つかりません", ri+1))
		}
		groups = append(groups, at)
	}
	size := len(groups[0])
	for i, g := range groups {
		if len(g) != size || g[len(g)-1]-g[0]+1 != size {
			return fail("行ごとの行の数がそろっていないか、続いていません")
		}
		if i > 0 && g[0] != groups[i-1][len(groups[i-1])-1]+1 {
			return fail("行ごとのかたまりが続いていません")
		}
	}
	// The names in, the longest value first; every row must read the same.
	var want []string
	for ri, g := range groups {
		names := append([]string{}, e.Names...)
		sort.SliceStable(names, func(i, j int) bool {
			return utf8.RuneCountInString(e.Rows[ri][names[i]]) > utf8.RuneCountInString(e.Rows[ri][names[j]])
		})
		var got []string
		for _, i := range g {
			l := lines[i]
			for _, n := range names {
				v := strings.TrimSpace(e.Rows[ri][n])
				if utf8.RuneCountInString(v) < minDraftValue || boolish(v) {
					continue
				}
				l, _ = replaceValue(l, v, "{{"+n+"}}")
			}
			got = append(got, l)
		}
		if ri == 0 {
			want = got
			continue
		}
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			return fail("値を変数にしても、行ごとの形がそろいません")
		}
	}
	for _, l := range want {
		for _, n := range e.Names {
			if c := strings.Count(l, "{{"+n+"}}"); c > 0 {
				d.Replaced[n] += c
			}
		}
	}
	return block{from: groups[0][0], to: groups[len(groups)-1][size-1] + 1, source: e.Source, lines: want}, true
}

func boolish(v string) bool {
	switch strings.ToLower(v) {
	case "true", "false", "yes", "no", "on", "off", "有", "無", "あり", "なし":
		return true
	}
	return false
}

// wordByte tells a byte that continues a word next to a value: a value is put
// in only where it stands as a whole (192.0.2.1 not in 192.0.2.10).
func wordByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '-' || b >= 0x80
}

// replaceValue puts with in for each whole v in line, outside the {{…}}
// already there; the count is how many.
func replaceValue(line, v, with string) (string, int) {
	if v == "" {
		return line, 0
	}
	var sb strings.Builder
	n, i := 0, 0
	for {
		j := strings.Index(line[i:], v)
		if j < 0 {
			break
		}
		j += i
		end := j + len(v)
		ok := true
		if j > 0 {
			p := line[j-1]
			ok = !wordByte(p) && p != '.'
		}
		if ok && end < len(line) {
			q := line[end]
			if wordByte(q) {
				ok = false
			} else if q == '.' && end+1 < len(line) && wordByte(line[end+1]) {
				ok = false
			}
		}
		if ok && inBraces(line, j) {
			ok = false
		}
		if !ok {
			sb.WriteString(line[i : j+1])
			i = j + 1
			continue
		}
		sb.WriteString(line[i:j])
		sb.WriteString(with)
		i = end
		n++
	}
	sb.WriteString(line[i:])
	return sb.String(), n
}

func inBraces(line string, at int) bool {
	open := strings.LastIndex(line[:at], "{{")
	if open < 0 {
		return false
	}
	return !strings.Contains(line[open:at], "}}")
}
