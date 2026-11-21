package check

import (
	"fmt"
	"strconv"
)

// The checks of how spanning tree is set for a device (appendix C 1.1):
// V-271 and V-272. A device runs either a tree per VLAN (pvst, rapid_pvst)
// or one tree for the device (rstp, stp; with mst the common tree): what is
// written for the other kind does nothing, or cannot be written at all.

// perVLAN tells whether a mode runs a tree per VLAN; known is false for an
// empty cell (the default of the device, which differs by vendor).
func perVLAN(mode string) (per, known bool) {
	switch mode {
	case "pvst", "rapid_pvst":
		return true, true
	case "mst", "rstp", "stp":
		return false, true
	}
	return false, false
}

// stpKind looks at the row of layer-2 settings of one device.
func (x *values) stpKind(dev string) {
	l2 := x.in.Schema.Table("l2_settings")
	sv := x.in.Schema.Table("stp_vlans")
	if l2 == nil || sv == nil || l2.Column("stp_priority") == nil {
		return
	}
	for _, g := range x.of("l2_settings", dev) {
		per, known := perVLAN(g.Get("stp_mode"))
		if !known {
			continue
		}
		x.tally.Look("V-271", dev, 1)
		x.tally.Look("V-272", dev, 1)
		if !per {
			for _, r := range x.of("stp_vlans", dev) {
				x.add("V-271", sv, r, "vlans", fmt.Sprintf("VLAN %s のスパニングツリーの設定がありますが、方式が %s です（VLAN ごとの設定は、方式が pvst・rapid_pvst のときだけ働きます）", r.Get("vlans"), g.Get("stp_mode")))
			}
		} else if g.Get("stp_priority") != "" {
			x.add("V-272", l2, g, "stp_priority", fmt.Sprintf("「優先度（機器全体）」%s がありますが、方式が %s です（VLAN ごとの木では、優先度を「VLAN ごとのスパニングツリー」に書きます）", g.Get("stp_priority"), g.Get("stp_mode")))
		}
	}
}

// stpSteps looks at the priorities of spanning tree (V-259): those of a
// bridge go in steps of 4096 up to 61440, that of a port in steps of 16 up
// to 240. A device takes no other value.
func (x *values) stpSteps(dev string) {
	for _, c := range []struct {
		table, column, name string
		step, max           int
	}{
		{"stp_vlans", "priority", "vlans", 4096, 61440},
		{"stp_instances", "priority", "instance", 4096, 61440},
		{"l2_settings", "stp_priority", "", 4096, 61440},
		{"switchports", "stp_port_priority", "if_name", 16, 240},
	} {
		t := x.in.Schema.Table(c.table)
		if t == nil || t.Column(c.column) == nil {
			continue
		}
		for _, r := range x.of(c.table, dev) {
			v := r.Get(c.column)
			if v == "" {
				continue
			}
			x.tally.Look("V-259", dev, 1)
			n, err := strconv.Atoi(v)
			if err != nil || (n%c.step == 0 && n <= c.max) {
				continue
			}
			what := t.Title.JA
			if c.name != "" {
				what += " " + r.Get(c.name)
			}
			x.add("V-259", t, r, c.column, fmt.Sprintf("%s の「%s」%s は書けません（0〜%d の %d 刻みです。近い値は %d）", what, t.Column(c.column).Title.JA, v, c.max, c.step, nearStep(n, c.step, c.max)))
		}
	}
}

func nearStep(n, step, max int) int {
	n = (n + step/2) / step * step
	if n > max {
		n = max
	}
	return n
}
