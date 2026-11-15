package pack

import (
	"reflect"
	"testing"
	"testing/fstest"
)

func TestLoadWords(t *testing.T) {
	fsys := fstest.MapFS{
		"pack.yaml":                  {Data: []byte("id: sample_os\nhostname: '^hostname (\\S+)'\n")},
		"templates/system.j2":        {Data: []byte("ip domain-name {{ device.domain }}\n")},
		"tests/import/one/input.txt": {Data: []byte("interface GigabitEthernet0/1\n no shutdown\n")},
		"words.txt":                  {Data: []byte("# words of the reference\nsla  track # the tracking\nMultilink\n")},
		"README.md":                  {Data: []byte("prose that is not read\n")},
		"notes.csv":                  {Data: []byte("other,kind\n")},
	}
	words, err := LoadWords(fsys)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"hostname", "ip", "domain", "name", "device", "interface", "gigabitethernet", "no", "shutdown", "sla", "track", "multilink", "sample", "os"} {
		if !words[w] {
			t.Errorf("%q is not a word of the pack", w)
		}
	}
	for _, w := range []string{"prose", "reference", "tracking", "other", "kind", "GigabitEthernet", ""} {
		if words[w] {
			t.Errorf("%q is a word of the pack", w)
		}
	}
	if got, want := Letters("ip-access-list ACL_In 10.5 é"), []string{"ip", "access", "list", "acl", "in"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Letters = %v, want %v", got, want)
	}
}
