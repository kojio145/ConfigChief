package ops

import (
	"os"
	"strings"
	"testing"
)

// The lines of a group of one device are put at the end of the extra
// configuration of the device, with the lines above them; nothing is
// written (spec 14.11.3).
func TestDraftExtra(t *testing.T) {
	e := judgeCase(t)
	j := review(t, e, "old")
	mop := groupOf(t, j, "no mop enabled")
	d, err := e.DraftExtra("old", mop.ID, "rt-a01")
	if err != nil {
		t.Fatal(err)
	}
	// No book yet: the host name itself.
	if d.Name != "rt-a01.j2" || d.Lines != 2 {
		t.Fatalf("draft = %+v", d)
	}
	for _, want := range []string{"{# 旧 rt-a01 の読めなかった行（input/old/rt-a01.txt", "interface GigabitEthernet0/0\n no mop enabled\n", "interface GigabitEthernet0/1\n no mop enabled\n", "上の行の名前"} {
		if !strings.Contains(d.Text, want) {
			t.Errorf("the draft lacks %q:\n%s", want, d.Text)
		}
	}
	sla := groupOf(t, j, "ip sla <n>")
	d, err = e.DraftExtra("old", sla.ID, "rt-a01")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(d.Text, "ip sla 10\n icmp-echo 192.0.2.254\n frequency 30\n") || strings.Contains(d.Text, "上の行の名前") {
		t.Errorf("the block:\n%s", d.Text)
	}
	if _, err := os.Stat(e.Proj.Path("extra/rt-a01.j2")); err == nil {
		t.Error("a draft wrote the file")
	}
	if _, err := e.DraftExtra("old", "nothing", "rt-a01"); err == nil {
		t.Error("a group that is not there")
	}
}

// An extra configuration is written only as a template, the old one kept;
// a deleted one goes to state/backup (spec 4.2, 14.11.3).
func TestSaveExtra(t *testing.T) {
	e := migCase(t)
	if _, err := e.MakeBook("old", false, at); err != nil {
		t.Fatal(err)
	}
	places, err := e.ExtraPlaces()
	if err != nil {
		t.Fatal(err)
	}
	if len(places.Devices) == 0 || !contains(places.Positions, "interfaces") || contains(places.Positions, "extra") {
		t.Fatalf("places = %+v", places)
	}
	if n, err := ExtraName("device", places.Devices[0]); err != nil || n != places.Devices[0]+".j2" {
		t.Errorf("name: %q, %v", n, err)
	}
	if n, _ := ExtraName("group", "tokyo"); n != "@tokyo.j2" {
		t.Errorf("group: %q", n)
	}
	if _, err := ExtraName("device", "a/b"); err == nil {
		t.Error("a name with a slash")
	}
	if _, err := e.SaveExtra("_all.j2", "line one\n{{ device.hostname\n", at); err == nil || !strings.Contains(err.Error(), "2 行目") {
		t.Errorf("a broken template: %v", err)
	}
	if _, err := e.SaveExtra("_all.j2", "{# position: after nowhere #}\nx\n", at); err == nil {
		t.Error("a position no pack has")
	}
	s, err := e.SaveExtra("_all.j2", "{# position: after interfaces #}\nip domain lookup source-interface {{ device.hostname }}\n", at)
	if err != nil || s.Backup != "" || len(s.Notes) != 0 {
		t.Fatalf("save: %+v, %v", s, err)
	}
	s, err = e.SaveExtra("_all.j2", "snmp-server location sample\r\n", at)
	if err != nil || s.Backup == "" {
		t.Fatalf("again: %+v, %v", s, err)
	}
	if !strings.Contains(read(t, e, s.Backup), "position: after interfaces") || read(t, e, "extra/_all.j2") != "snmp-server location sample\n" {
		t.Error("the copy or the file")
	}
	if text, err := e.ReadExtra("_all.j2"); err != nil || text != "snmp-server location sample\n" {
		t.Errorf("read: %q, %v", text, err)
	}
	if s, err = e.SaveExtra("rt-zz9.j2", "x\n", at); err != nil || len(s.Notes) != 1 {
		t.Errorf("a host the book has not: %+v, %v", s, err)
	}
	gone, err := e.DeleteExtra("rt-zz9.j2", at)
	if err != nil || !strings.HasPrefix(gone, "state/backup/extra_rt-zz9_") {
		t.Fatalf("delete: %q, %v", gone, err)
	}
	if _, err := os.Stat(e.Proj.Path("extra/rt-zz9.j2")); err == nil {
		t.Error("the file is still there")
	}
	if _, err := e.ReadExtra("../project.yaml"); err == nil {
		t.Error("a file outside extra/")
	}
	// The generation takes what was written.
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Devices) == 0 || o.Devices[0].Output == nil || !strings.Contains(o.Devices[0].Output.Text, "snmp-server location sample") {
		t.Errorf("the generated text lacks the extra configuration")
	}
}
