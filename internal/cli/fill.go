package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/kojio145/configchief/internal/ops"
)

// cmdFill fills a template with variables (spec 9.7, 15): the list of the
// templates and the rules, the files of each device, a change against the
// configuration now, and a first template from one device.
func cmdFill(args []string, out, errOut io.Writer) int {
	pos, val, on, err := options(args, []string{"--project", "--template", "--diff", "--device", "--draft", "--name"}, []string{"--list"})
	if err != nil || len(pos) != 0 {
		if err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprint(errOut, usage)
		return 2
	}
	modes := 0
	for _, m := range []bool{on["--list"], val["--template"] != "", val["--draft"] != ""} {
		if m {
			modes++
		}
	}
	if modes != 1 {
		fmt.Fprintln(errOut, "fill: --list・--template・--draft のどれか 1 つを書きます")
		fmt.Fprint(errOut, usage)
		return 2
	}
	e, ok := openCase(val["--project"], out, errOut)
	if !ok {
		return 2
	}
	switch {
	case on["--list"]:
		ts, err := e.FillTemplates()
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 2
		}
		vs, err := e.VarFiles()
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 2
		}
		bad := 0
		fmt.Fprintf(out, "ひな形 %d\n", len(ts))
		for _, t := range ts {
			fmt.Fprintf(out, "  %s\n", t.Path)
			if t.Err != "" {
				bad++
				fmt.Fprintf(out, "    誤り: %s\n", t.Err)
			}
		}
		fmt.Fprintf(out, "変数ルール %d\n", len(vs))
		for _, f := range vs {
			line := "  " + f.Path
			if f.Rule != nil {
				var names []string
				for _, v := range f.Rule.Vars {
					names = append(names, v.Name)
				}
				line += fmt.Sprintf("（一覧 %s、変数 %s）", f.Rule.File, strings.Join(names, "・"))
			}
			fmt.Fprintln(out, line)
			for _, p := range f.Problems {
				bad++
				fmt.Fprintf(out, "    誤り: %s\n", p)
			}
		}
		if bad > 0 {
			return 1
		}
		return 0
	case val["--draft"] != "":
		if val["--device"] == "" {
			fmt.Fprintln(errOut, "fill --draft: --device（値を取る機器）を書きます")
			return 2
		}
		d, err := e.FillDraft(val["--draft"], val["--device"])
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 2
		}
		name := val["--name"]
		if name == "" {
			name = "draft"
		}
		rel, fault, err := e.SaveTemplate("", name, d.Text)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 2
		}
		fmt.Fprintf(out, "ひな形の案: %s（%s の %s から）\n", rel, d.File, d.Device)
		for n, c := range d.Replaced {
			fmt.Fprintf(out, "  {{%s}} %d か所\n", n, c)
		}
		for _, n := range d.Notes {
			fmt.Fprintf(out, "注意: %s\n", n)
		}
		if fault != "" {
			fmt.Fprintf(out, "誤り: %s\n", fault)
		}
		if len(d.Notes) > 0 || fault != "" {
			return 1
		}
		return 0
	}
	var devices []string
	for _, d := range strings.Split(val["--device"], ",") {
		if d = strings.TrimSpace(d); d != "" {
			devices = append(devices, d)
		}
	}
	o, err := e.Fill(fillOptions(val["--template"], val["--diff"], devices), now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	made := 0
	for _, d := range o.Devices {
		switch {
		case d.Made:
			made++
			fmt.Fprintf(out, "  作った %s: %s（%d 行）\n", d.Name, d.File, d.Lines)
		case d.Unchanged:
			fmt.Fprintf(out, "  変更なし %s\n", d.Name)
		default:
			fmt.Fprintf(out, "  作らなかった %s: %s\n", d.Name, d.Why)
		}
	}
	for _, n := range o.Notes {
		fmt.Fprintf(out, "注意: %s\n", n)
	}
	fmt.Fprintf(out, "ひな形 %s: 機器 %d、作った %d、作らなかった %d\n", o.Template, len(o.Devices), made, o.NotMade())
	fmt.Fprintf(out, "レポート: %s\n          %s\n", o.Report.XLSX, o.Report.HTML)
	if o.NotMade() > 0 || len(o.Notes) > 0 {
		return 1
	}
	return 0
}

func fillOptions(template, diff string, devices []string) ops.FillOptions {
	return ops.FillOptions{Template: template, Diff: diff, Devices: devices}
}
