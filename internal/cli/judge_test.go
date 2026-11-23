package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// groupID gives the ID in the line of the list that ends with a shape.
func groupID(t *testing.T, list, shape string) string {
	t.Helper()
	m := regexp.MustCompile(`(?m)^  ([0-9a-f]{8,})  .*  ` + regexp.QuoteMeta(shape) + `$`).FindStringSubmatch(list)
	if m == nil {
		t.Fatalf("no group %q in the list:\n%s", shape, list)
	}
	return m[1]
}

func TestJudgeCommand(t *testing.T) {
	tick := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	now = func() time.Time {
		tick = tick.Add(time.Minute)
		return tick
	}
	defer func() { now = time.Now }()
	dir := filepath.Join(t.TempDir(), "sample-case")
	if code, _, errOut := do("init", dir, "--kind", "migration"); code != 0 {
		t.Fatalf("init = %d: %s", code, errOut)
	}
	for _, name := range []string{"rt-a01.txt", "rt-a02.txt"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "internal", "ops", "testdata", "judge", name))
		if err != nil {
			t.Fatal(err)
		}
		put(t, filepath.Join(dir, "input", "old", name), string(data))
		put(t, filepath.Join(dir, "input", "new", name), string(data))
	}

	// The list writes nothing and says with its code that lines are open.
	code, list, errOut := do("judge", "--project", dir)
	if code != 1 {
		t.Fatalf("judge = %d: %s%s", code, list, errOut)
	}
	wants(t, "the list", list,
		"読めなかった行（旧コンフィグ）: 未判定 12 行、保留 0 行、追加コンフィグで書く 0 行、要らない 0 行。かたまり 3 個",
		"  未判定  2 台  3 行  cisco_ios  interface GigabitEthernet<n> > no mop enabled",
		"  未判定  2 台  3 行（下に 4 行）  cisco_ios  ip sla <n>",
	)
	if _, err := os.Stat(filepath.Join(dir, "state", "judgments.yaml")); err == nil {
		t.Error("the list wrote the judgments")
	}
	sla := groupID(t, list, "ip sla <n>")
	policy := groupID(t, list, "interface GigabitEthernet<n> > no mop enabled")
	schedule := groupID(t, list, "ip sla schedule <n> life forever start-time now")

	for _, c := range []struct {
		want string
		args []string
	}{
		{"--group <ID> と --verdict", []string{"--by", "admin"}},
		{"両方を指定", []string{"--group", sla}},
		{"両方を指定", []string{"--verdict", "manual", "--by", "admin"}},
		{"両方を指定", []string{"--group", sla, "--verdict", "manual", "--by", "admin", "--all"}},
		{"名前を書いてください", []string{"--group", sla, "--verdict", "manual"}},
		{"理由を書いてください", []string{"--group", sla, "--verdict", "not_needed", "--by", "admin"}},
		{"どれかです", []string{"--group", sla, "--verdict", "later", "--by", "admin"}},
		{"読めなかった行にありません", []string{"--group", "00000000", "--verdict", "hold", "--by", "admin"}},
		{"読めなかった行にありません", []string{"--group", sla, "--verdict", "hold", "--by", "admin", "--source", "new", "--devices", "rt-a09"}},
		{"行がありません: rt-a09", []string{"--group", sla, "--verdict", "hold", "--by", "admin", "--devices", "rt-a09"}},
		{"判定はありません", []string{"--group", sla, "--verdict", "none"}},
		{"", []string{"extra"}},
		{"", []string{"--colour", "red"}},
	} {
		code, out, errOut := do(append([]string{"judge", "--project", dir}, c.args...)...)
		if code != 2 || errOut == "" || !strings.Contains(errOut, c.want) {
			t.Errorf("judge %v = %d, want 2 with %q:\n%s%s", c.args, code, c.want, out, errOut)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "state", "judgments.yaml")); err == nil {
		t.Error("a refused judgment was written")
	}

	// One device first, then every device: the judgment that names a device stays.
	code, out, errOut := do("judge", "--project", dir, "--group", sla, "--verdict", "hold", "--devices", "rt-a02", "--by", "admin")
	if code != 0 {
		t.Fatalf("judge for a device = %d: %s%s", code, out, errOut)
	}
	wants(t, "judge for a device", out,
		"かたまり "+sla+" に「保留」を付けました（機器 1 台、2 行）",
		"このかたまりの判定: 未判定 1 台、保留 1 台",
		"残り（旧コンフィグ）: 未判定 10 行、保留 2 行",
	)
	// The beginning of an ID is enough when no other group begins the same.
	code, out, errOut = do("judge", "--project", dir, "--group", strings.ToUpper(sla[:6]), "--verdict", "manual", "--by", "admin")
	if code != 0 {
		t.Fatalf("judge for all = %d: %s%s", code, out, errOut)
	}
	wants(t, "judge for all", out,
		"かたまり "+sla+" に「追加コンフィグで書く」を付けました（機器 1 台、5 行）",
		"機器を指定した判定が 1 台に残っています",
		"このかたまりの判定: 保留 1 台、追加コンフィグで書く 1 台",
		"残り（旧コンフィグ）: 未判定 5 行、保留 2 行",
		"パラメータブックがまだ無い",
	)
	for _, id := range []string{policy, schedule} {
		if code, out, errOut = do("judge", "--project", dir, "--group", id, "--verdict", "not_needed", "--reason", "sample reason", "--by", "admin"); code != 0 {
			t.Fatalf("judge %s = %d: %s%s", id, code, out, errOut)
		}
	}
	wants(t, "the last judgment", out, "残り（旧コンフィグ）: 未判定 0 行、保留 2 行")

	code, list, _ = do("judge", "--project", dir)
	if code != 1 {
		t.Errorf("the list with a line on hold = %d:\n%s", code, list)
	}
	wants(t, "the list of the open groups", list,
		"未判定 0 行、保留 2 行、追加コンフィグで書く 5 行、要らない 5 行",
		sla+"  保留 1 台、追加コンフィグで書く 1 台  2 台",
		"判定の済んだかたまりが 2 個あります",
	)
	if strings.Contains(list, policy) {
		t.Errorf("a judged group is in the list without --all:\n%s", list)
	}
	if _, list, _ = do("judge", "--project", dir, "--all"); !strings.Contains(list, policy+"  要らない  2 台") {
		t.Errorf("--all lacks the judged group:\n%s", list)
	}

	// Taking the judgment of the device away leaves the one for every device.
	if code, out, errOut = do("judge", "--project", dir, "--group", sla, "--verdict", "none", "--devices", "rt-a02"); code != 0 {
		t.Fatalf("clear = %d: %s%s", code, out, errOut)
	}
	wants(t, "clear", out, "かたまり "+sla+" の判定を取り消しました（機器 1 台、2 行）", "このかたまりの判定: 追加コンフィグで書く")
	if code, list, _ = do("judge", "--project", dir); code != 0 || !strings.Contains(list, "未判定・保留のかたまりはありません") {
		t.Errorf("the list with every line judged = %d:\n%s", code, list)
	}

	// The import makes the book: now the devices to handle by hand need their
	// additional configuration (M-402).
	code, out, errOut = do("import", "--project", dir)
	if code != 1 {
		t.Fatalf("import = %d: %s%s", code, out, errOut)
	}
	wants(t, "import", out,
		"読めなかった行: 12 行（未判定 0、保留 0、追加コンフィグで書く 7、要らない 5）",
		"M-402: rt-a01 に「追加コンフィグで書く」の行が 5 行ありますが",
		"M-402: rt-a02 に「追加コンフィグで書く」の行が 2 行ありますが",
	)
	if strings.Contains(out, "judge で付けます") {
		t.Errorf("the import asks for judgments although every line is judged:\n%s", out)
	}
	if code, list, _ = do("judge", "--project", dir); code != 1 || !strings.Contains(list, "M-402: rt-a01") {
		t.Errorf("the list with M-402 = %d:\n%s", code, list)
	}
	put(t, filepath.Join(dir, "extra", "_all.j2"), "! sample addition\n")
	if code, list, _ = do("judge", "--project", dir); code != 0 || strings.Contains(list, "M-402") {
		t.Errorf("the list with the additional configuration = %d:\n%s", code, list)
	}

	// The new side has its own judgments.
	code, list, _ = do("judge", "--project", dir, "--source", "new")
	if code != 1 || !strings.Contains(list, "読めなかった行（新コンフィグ）: 未判定 12 行") {
		t.Errorf("the list of the new side = %d:\n%s", code, list)
	}

	// The log tells the group and the verdict, not the lines.
	log, err := os.ReadFile(filepath.Join(dir, "state", "log.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(log), "judge: old "+sla) || strings.Contains(string(log), "ip sla") {
		t.Errorf("the log:\n%s", log)
	}
}
