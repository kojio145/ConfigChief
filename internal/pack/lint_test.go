package pack

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/schemas"
)

func lintOf(t *testing.T, dir string) (*Pack, *LintResult) {
	t.Helper()
	sc, err := schemas.Load()
	if err != nil {
		t.Fatalf("schemas: %v", err)
	}
	p, err := Load(os.DirFS(dir))
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	return p, Lint(p, sc)
}

// The check names every fault of the pack once, with its rule, and nothing
// that is right.
func TestLint(t *testing.T) {
	p, res := lintOf(t, "testdata/lint")
	want := []string{
		"規則 if.fast: interfaces の speed: 「fast」は選択肢（auto, 10, 100, 1000, 2500, 5000, 10000）にありません",
		"規則 if.shelf: 表 interfaces に、shelf_{{ dir }} に当たる列がありません",
		"規則 if.relay: interfaces の dhcp_relay: 「standard」は IPv4 アドレスではありません",
		"規則 route.default: static_routes の prefix: 「everything」は IPv4 の「アドレス/長さ」の形ではありません",
		"規則 note: 表 shelves は設計データにありません",
		"規則 note: interfaces の dhcp_relay は値を並べる列です。append で入れます",
		"derive boxes: for の表 boxes は設計データにありません",
		"derive boxes: 表 vrfs に列 colour はありません",
		"defaults interfaces.mtu: interfaces の mtu: 「big」は整数ではありません",
		"defaults interfaces.colour: 表 interfaces に列 colour はありません",
		"defaults shelves.name: 表 shelves は設計データにありません",
		"規則 if.colour: 語の種類「colour」が lexicon.yaml にありません",
	}
	if !reflect.DeepEqual(res.Problems, want) {
		t.Errorf("problems:\n  %s\nwant:\n  %s", strings.Join(res.Problems, "\n  "), strings.Join(want, "\n  "))
	}
	if want := []string{"if.box（import/10_rules.yaml）"}; !reflect.DeepEqual(res.Valueless, want) {
		t.Errorf("valueless = %v, want %v", res.Valueless, want)
	}

	// A work table that a when reads is in use; one that nothing reads is not.
	if want := []string{"作業用の表 x_lost に入れる規則があるのに、使う derive がありません"}; !reflect.DeepEqual(p.Warnings, want) {
		t.Errorf("warnings = %v, want %v", p.Warnings, want)
	}
}

// What the importer finds line by line in the typed pack, the check finds
// without a config.
func TestLintTyped(t *testing.T) {
	_, res := lintOf(t, "../importer/testdata/typed")
	want := []string{
		"規則 if.colour: 表 interfaces に列 colour はありません",
		"規則 if.relay-one: interfaces の dhcp_relay は値を並べる列です。append で入れます",
		"規則 if.mtu-more: interfaces の mtu は値が 1 つの列です。set で入れます",
		"規則 if.rename: interfaces の name はキーの列です。値は row の key に書きます",
		"規則 route.by: distance は表 static_routes のキーの列ではありません（キーは device, vrf, prefix, next_hop, out_if）",
		"規則 shelf: 表 shelves は設計データにありません",
	}
	if !reflect.DeepEqual(res.Problems, want) {
		t.Errorf("problems:\n  %s\nwant:\n  %s", strings.Join(res.Problems, "\n  "), strings.Join(want, "\n  "))
	}
	if len(res.Valueless) != 0 {
		t.Errorf("valueless = %v", res.Valueless)
	}
}
