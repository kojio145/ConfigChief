package cli

import (
	"fmt"
	"io"
)

// cmdSites sets the devices of one source side by side (spec 12.5, 15).
func cmdSites(args []string, out, errOut io.Writer) int {
	pos, val, _, err := options(args, []string{"--project", "--source"}, nil)
	if err != nil || len(pos) != 0 {
		if err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprint(errOut, usage)
		return 2
	}
	source := val["--source"]
	if source == "" {
		source = "book"
	}
	e, ok := openCase(val["--project"], out, errOut)
	if !ok {
		return 2
	}
	o, err := e.Sites(source, now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	fmt.Fprintf(out, "多拠点の一覧: %s、機器 %d 台\n", source, len(o.Devices))
	for _, t := range o.Tables {
		differ, split := 0, 0
		for _, it := range t.Items {
			if it.Split {
				split++
			} else {
				differ++
			}
		}
		if differ+split > 0 {
			fmt.Fprintf(out, "  %s: 違う機器がある項目 %d、ばらばらの項目 %d（全台で同じ項目 %d）\n", t.Title, differ, split, t.Same)
		}
	}
	const show = 20
	for i, x := range o.Outliers {
		if i == show {
			fmt.Fprintf(out, "  …ほか %d 件（レポートの「外れ値」にあります）\n", len(o.Outliers)-show)
			break
		}
		row := x.Row
		if row != "" {
			row += " の "
		}
		fmt.Fprintf(out, "  外れ値 %s %s: %s%s = %s（%d 台中 %d 台は %s）\n", x.Device, x.Table, row, x.Column, x.Value, x.Of, x.CommonN, x.Common)
	}
	for _, f := range o.Failed {
		fmt.Fprintf(out, "  読めなかったファイル %s\n", f)
	}
	for _, n := range o.Notes {
		fmt.Fprintf(out, "注意: %s\n", n)
	}
	fmt.Fprintf(out, "外れ値 %d 件\n", len(o.Outliers))
	fmt.Fprintf(out, "レポート: %s\n          %s\n", o.Report.XLSX, o.Report.HTML)
	if len(o.Failed) > 0 {
		return 1
	}
	return 0
}
