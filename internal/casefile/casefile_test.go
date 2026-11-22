package casefile

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

var at = time.Date(2026, 11, 22, 21, 0, 0, 0, time.UTC)

func sample() []File {
	return []File{
		{Path: "project.yaml", Data: []byte("version: 1\nname: sample\n")},
		{Path: "input/old/rt-a01.txt", Data: []byte("hostname rt-a01\n")},
	}
}

func TestRoundTrip(t *testing.T) {
	for _, pw := range []string{"", "correct horse"} {
		data, err := Pack(Manifest{Name: "sample", Version: "1.3.0"}, sample(), pw, at)
		if err != nil {
			t.Fatal(err)
		}
		if Sealed(data) != (pw != "") || !Is(data) {
			t.Fatalf("sealed %v for %q", Sealed(data), pw)
		}
		if pw != "" && bytes.Contains(data, []byte("hostname rt-a01")) {
			t.Error("a sealed file shows its text")
		}
		m, files, err := Unpack(data, pw)
		if err != nil {
			t.Fatal(err)
		}
		if m.Name != "sample" || m.Format != Format || len(files) != 2 || files[0].Path != "input/old/rt-a01.txt" || string(files[1].Data) != "version: 1\nname: sample\n" {
			t.Errorf("%+v %+v", m, files)
		}
	}
}

func TestPasswords(t *testing.T) {
	if _, err := Pack(Manifest{}, sample(), "short", at); err == nil {
		t.Error("a short password is refused")
	}
	data, _ := Pack(Manifest{}, sample(), "correct horse", at)
	if _, _, err := Unpack(data, ""); !errors.Is(err, ErrNeedPassword) {
		t.Errorf("no password: %v", err)
	}
	if _, _, err := Unpack(data, "wrong horse!"); !errors.Is(err, ErrPassword) {
		t.Errorf("a wrong password: %v", err)
	}
	data[len(data)-1] ^= 1
	if _, _, err := Unpack(data, "correct horse"); !errors.Is(err, ErrPassword) {
		t.Errorf("a broken file: %v", err)
	}
	if _, _, err := Unpack([]byte("PK\x03\x04"), ""); err == nil || !strings.Contains(err.Error(), ".ccase") {
		t.Errorf("not a case file: %v", err)
	}
}

func TestBadNames(t *testing.T) {
	for _, p := range []string{"../x", "/etc/x", `C:\x`, "a/../../x", "a//b", "./a", ManifestName} {
		if _, err := Pack(Manifest{}, []File{{Path: p}}, "", at); err == nil {
			t.Errorf("Pack(%q) is refused", p)
		}
	}
	// A file made elsewhere with a name that leaves the folder.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("../evil.txt")
	w.Write([]byte("x"))
	w, _ = zw.Create(ManifestName)
	w.Write([]byte(`{"format":1,"files":["../evil.txt"]}`))
	zw.Close()
	if _, _, err := Unpack(append(append([]byte{}, magicPlain...), buf.Bytes()...), ""); err == nil {
		t.Error("a name that leaves the folder stops the reading")
	}
	// A file the manifest does not name is not given back.
	buf.Reset()
	zw = zip.NewWriter(&buf)
	w, _ = zw.Create("extra.txt")
	w.Write([]byte("x"))
	w, _ = zw.Create(ManifestName)
	w.Write([]byte(`{"format":1,"files":[]}`))
	zw.Close()
	_, files, err := Unpack(append(append([]byte{}, magicPlain...), buf.Bytes()...), "")
	if err != nil || len(files) != 0 {
		t.Errorf("unlisted file: %v %v", files, err)
	}
	// A newer format.
	buf.Reset()
	zw = zip.NewWriter(&buf)
	w, _ = zw.Create(ManifestName)
	w.Write([]byte(`{"format":9,"version":"9.0.0","files":[]}`))
	zw.Close()
	if _, _, err := Unpack(append(append([]byte{}, magicPlain...), buf.Bytes()...), ""); err == nil || !strings.Contains(err.Error(), "9.0.0") {
		t.Errorf("a newer format: %v", err)
	}
}
