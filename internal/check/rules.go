package check

import "strings"

// Rule is one check of appendix C.
type Rule struct {
	ID string
	// Severity is the one the check has unless the case says otherwise
	// (checks.severity of project.yaml).
	Severity string
	Title    string
	// PerDevice tells that the check looks at a device as a whole: what it
	// looked at is counted in devices, not in cells and rows.
	PerDevice bool
}

// The checks of the generation (appendix C 1.1 and 2.1, the ones of M2), in
// the order of the list.
var rules = []Rule{
	{ID: "V-101", Severity: Error, Title: "必須の値が空"},
	{ID: "V-102", Severity: Error, Title: "値の形式が型に合わない"},
	{ID: "V-103", Severity: Error, Title: "値が範囲の外"},
	{ID: "V-104", Severity: Error, Title: "選択肢にない語"},
	{ID: "V-105", Severity: Error, Title: "行のキーが重複している"},
	{ID: "V-106", Severity: Error, Title: "参照先がない（ACL 名・IF 名・回線監視の番号など）"},
	{ID: "V-107", Severity: Error, Title: "「機器」列のホスト名が機器シートにない。@グループ名 に当たる機器が 1 台もない"},
	{ID: "V-108", Severity: Info, Title: "定義にない見出しがある（自由列として読んだ）"},
	{ID: "V-109", Severity: Error, Title: "機器シートのメーカーに当たるパックがない", PerDevice: true},
	{ID: "V-110", Severity: Warning, Title: "値に全角の英数字、前後の空白、制御文字が含まれる"},
	{ID: "V-111", Severity: Error, Title: "IF 名が、機種のポート一覧にも論理 IF の書き方にも当たらない"},
	{ID: "V-112", Severity: Error, Title: "生成先のメーカーで書けない語"},
	{ID: "V-113", Severity: Info, Title: "「明示」列が空のため、生成先の既定値で書き出す"},
	{ID: "V-114", Severity: Warning, Title: "機種が空、または機種の定義がない", PerDevice: true},
	{ID: "V-115", Severity: Warning, Title: "参照先がないが、参照先の表をその機器のメーカーパックがまだ読まない"},
	{ID: "V-201", Severity: Error, Title: "同じ機器・同じ VRF の中で、IF のサブネットが重なっている"},
	{ID: "V-202", Severity: Warning, Title: "案件の中で同じ IP アドレスが複数の IF に付いている"},
	{ID: "V-203", Severity: Error, Title: "IF のアドレスがネットワークアドレスかブロードキャストアドレス"},
	{ID: "V-204", Severity: Warning, Title: "スタティック経路のネクストホップが、その機器のどの IF のサブネットにも入らない"},
	{ID: "V-205", Severity: Error, Title: "仮想 IP が、その IF のサブネットに入らない"},
	{ID: "V-206", Severity: Error, Title: "ホスト名に使えない文字がある・長すぎる", PerDevice: true},
	{ID: "V-207", Severity: Error, Title: "ルータ ID（OSPF・BGP）が案件の中で重複している"},
	{ID: "V-208", Severity: Info, Title: "有効な IF の説明が空"},
	{ID: "V-209", Severity: Error, Title: "ハッシュ化された秘密の値を、別メーカーの機器に引き継ごうとしている"},
	{ID: "V-210", Severity: Warning, Title: "ACL・プレフィックスリスト・回線監視が、定義されているがどこからも使われていない"},
	{ID: "V-211", Severity: Warning, Title: "ACL に行が 1 つもない。permit の行が 1 つもない"},
	{ID: "V-212", Severity: Warning, Title: "TCP MSS 調整の値が MTU − 40 より大きい"},
	{ID: "V-213", Severity: Error, Title: "DHCP の配布範囲・デフォルトルータがネットワークの外。配布範囲が IF のアドレスを含む"},
	{ID: "V-214", Severity: Warning, Title: "スタティック経路・冗長化が参照する回線監視の送信元 IF が無効"},
	{ID: "V-215", Severity: Error, Title: "IPsec の方式が tunnel_if なのにトンネル IF がない。policy なのに対象通信の行がない"},
	{ID: "V-216", Severity: Warning, Title: "弱い暗号（des・md5・dh1・dh2・dh5）を使っている"},
	{ID: "V-217", Severity: Error, Title: "PPPoE の論理 IF・物理 IF がインタフェース表にない"},
	{ID: "V-218", Severity: Warning, Title: "秘密の値が仮の値のまま"},

	{ID: "X-300", Severity: Info, Title: "相手の機器が案件の中にない（管理外）"},
	{ID: "X-301", Severity: Warning, Title: "IPsec・トンネル: 相手アドレスは案件の中の機器のものだが、相手側に対になる行がない"},
	{ID: "X-302", Severity: Error, Title: "IPsec: IKE の版、モードが両端で違う"},
	{ID: "X-303", Severity: Error, Title: "IPsec: IKE の暗号・ハッシュ・DH グループが両端で違う"},
	{ID: "X-304", Severity: Error, Title: "IPsec: ESP の暗号・ハッシュ・PFS が両端で違う"},
	{ID: "X-305", Severity: Error, Title: "IPsec: 事前共有鍵が両端で違う"},
	{ID: "X-306", Severity: Error, Title: "IPsec: 自分の ID と、相手が期待する相手の ID が合わない"},
	{ID: "X-307", Severity: Error, Title: "IPsec: 対象通信が両端で鏡写しになっていない"},
	{ID: "X-308", Severity: Warning, Title: "IPsec: 有効時間が両端で違う"},
	{ID: "X-309", Severity: Info, Title: "IPsec: 相手監視・NAT 越えの設定が片側だけ"},
	{ID: "X-311", Severity: Error, Title: "トンネル: 送信元と宛先が両端で鏡写しになっていない"},
	{ID: "X-312", Severity: Error, Title: "トンネル: 種類・鍵が両端で違う"},
	{ID: "X-313", Severity: Error, Title: "トンネル: 両端のトンネル IF のアドレスが同じサブネットにない"},
	{ID: "X-314", Severity: Warning, Title: "トンネル: MTU・監視の設定が両端で違う"},
	{ID: "X-321", Severity: Error, Title: "BGP: 相手 AS 番号が、相手機器の AS 番号と違う"},
	{ID: "X-322", Severity: Error, Title: "BGP: 片側にしかネイバーの行がない"},
	{ID: "X-323", Severity: Error, Title: "BGP: パスワードが両端で違う・片側だけ"},
	{ID: "X-324", Severity: Warning, Title: "BGP: 送信元 IF の指定と、相手が指すアドレスが合わない。マルチホップの指定が片側だけ"},
	{ID: "X-325", Severity: Info, Title: "BGP: タイマーが両端で違う"},
	{ID: "X-331", Severity: Error, Title: "OSPF: 同じサブネットの IF で、エリアが違う"},
	{ID: "X-332", Severity: Error, Title: "OSPF: Hello・Dead 間隔が違う"},
	{ID: "X-333", Severity: Error, Title: "OSPF: 網の種類が違う"},
	{ID: "X-334", Severity: Error, Title: "OSPF: 認証の方式・鍵番号・鍵が違う"},
	{ID: "X-335", Severity: Error, Title: "OSPF: エリアの種類（stub・nssa）が違う"},
	{ID: "X-336", Severity: Warning, Title: "OSPF: IF の MTU が違う"},
	{ID: "X-337", Severity: Warning, Title: "OSPF: 片側だけ受動になっている"},
	{ID: "X-338", Severity: Info, Title: "OSPF: 同じサブネットで、片側だけ OSPF が有効"},
	{ID: "X-341", Severity: Error, Title: "VRRP・HSRP: 同じサブネット・同じグループ番号で、仮想 IP が違う"},
	{ID: "X-342", Severity: Error, Title: "VRRP・HSRP: 方式・版が違う"},
	{ID: "X-343", Severity: Warning, Title: "VRRP・HSRP: 優先度が同じ"},
	{ID: "X-344", Severity: Error, Title: "VRRP・HSRP: 認証が違う"},
	{ID: "X-345", Severity: Warning, Title: "VRRP・HSRP: 間隔が違う"},
	{ID: "X-346", Severity: Warning, Title: "VRRP・HSRP: グループに機器が 1 台しかない"},
	{ID: "X-351", Severity: Warning, Title: "同じサブネットの IF どうしで、プレフィックス長・MTU が違う"},

	{ID: "G-601", Severity: Error, Title: "テンプレートの実行エラー", PerDevice: true},
	{ID: "G-602", Severity: Error, Title: "設計データに値があるのに、生成に使われなかった", PerDevice: true},
	{ID: "G-603", Severity: Error, Title: "語の対応がない", PerDevice: true},
	{ID: "G-604", Severity: Error, Title: "往復の不一致（生成したコンフィグを取り込み直した結果が、設計データと違う）", PerDevice: true},
	{ID: "G-605", Severity: Warning, Title: "生成したコンフィグに、取り込み直しで未対応になる行がある", PerDevice: true},
	{ID: "G-606", Severity: Error, Title: "生成先のメーカーで必要な値が空", PerDevice: true},
	{ID: "G-607", Severity: Error, Title: "機種が対応していない機能を使っている", PerDevice: true},
	{ID: "G-608", Severity: Error, Title: "追加コンフィグの実行エラー", PerDevice: true},
	{ID: "G-609", Severity: Warning, Title: "追加コンフィグの行が、パックが生成した行と二重になっている", PerDevice: true},
	{ID: "G-610", Severity: "", Title: "案件のルール: 含むべき行がない", PerDevice: true},
	{ID: "G-611", Severity: "", Title: "案件のルール: 含んではいけない行がある", PerDevice: true},
	{ID: "G-612", Severity: "", Title: "案件のルール: 式が偽・値が重複・値がそろっていない"},
	{ID: "G-613", Severity: Error, Title: "出力ファイル名が重複する・ファイル名に使えない文字がある", PerDevice: true},
	{ID: "G-614", Severity: Warning, Title: "パックが、このメーカーの書き方に置き換えて書いた設定がある", PerDevice: true},
}

// The checks of a migration that can be approved (appendix C 1.3). They come
// from the steps that read the old configurations (import, book, compare),
// not from the generation.
var migration = []Rule{
	{ID: "M-403", Severity: Error, Title: "旧にあって新にない行"},
	{ID: "M-404", Severity: Warning, Title: "新にあって旧にない行"},
	{ID: "M-405", Severity: Warning, Title: "旧と新で値が違うセル"},
	{ID: "M-406", Severity: Error, Title: "IF 対応が決まっていない旧 IF がある"},
	{ID: "M-407", Severity: Error, Title: "機器対応が決まっていない旧機器がある"},
	{ID: "M-408", Severity: Error, Title: "複数の旧 IF が、同じ新 IF に対応している"},
}

// The checks of the customer's lists (appendix C 1.4). They come from the
// step that reads the lists and compares them with the book.
var source = []Rule{
	{ID: "S-501", Severity: Warning, Title: "お客様の一覧にあって、パラメータブックにない行"},
	{ID: "S-502", Severity: Info, Title: "パラメータブックにあって、お客様の一覧にない行"},
	{ID: "S-503", Severity: Error, Title: "お客様の一覧とパラメータブックで値が違う"},
	{ID: "S-504", Severity: Warning, Title: "お客様の一覧の値が読めなかった"},
	{ID: "S-505", Severity: Info, Title: "お客様の一覧のファイルが、前回の読み込みから変わった"},
}

// The checks of what is on the devices after the work (appendix C 2.3).
// They come from the step that reads the logs of input/logs.
var applied = []Rule{
	{ID: "R-801", Severity: Error, Title: "パラメータブックにあるが、機器の設定にない行"},
	{ID: "R-802", Severity: Warning, Title: "機器の設定にあるが、パラメータブックにない行"},
	{ID: "R-803", Severity: Error, Title: "パラメータブックと機器の設定で値が違う"},
	{ID: "R-804", Severity: Error, Title: "追加コンフィグの行が、機器の設定に見つからない"},
	{ID: "R-805", Severity: Info, Title: "機器の設定に未対応の行があり、追加コンフィグにも当たらない"},
	{ID: "R-806", Severity: Error, Title: "ログのホスト名が、案件のどの機器にも当たらない・別の機器のものである"},
	{ID: "R-807", Severity: Info, Title: "秘密の値は、機器側でハッシュ化されているため比べられなかった"},
}

// Applied tells whether the check is one of what is on the devices.
func Applied(id string) bool {
	for _, r := range applied {
		if r.ID == id {
			return true
		}
	}
	return false
}

// The checks of the logs of the work (appendix C 2.2). They come from the
// step that reads the commands typed into the devices.
var loaded = []Rule{
	{ID: "L-701", Severity: Error, Title: "機器に弾かれた行（エラーメッセージと、その直前のコマンド）"},
	{ID: "L-702", Severity: Warning, Title: "ログのコマンドの並びが、生成したコンフィグと合わない（別の版を流した疑い）"},
	{ID: "L-703", Severity: Warning, Title: "ログに現れたコマンドが、生成したコンフィグより少ない（途中で切れた疑い）"},
	{ID: "L-704", Severity: Info, Title: "機器の警告メッセージ"},
	{ID: "L-705", Severity: Error, Title: "ログのプロンプトのホスト名と、流したコンフィグのホスト名・機器の対応が合わない（機器の取り違えの疑い）"},
}

// Loaded tells whether the check is one of the logs of the work.
func Loaded(id string) bool {
	for _, r := range loaded {
		if r.ID == id {
			return true
		}
	}
	return false
}

// Source tells whether the check is one of the customer's lists.
func Source(id string) bool {
	for _, r := range source {
		if r.ID == id {
			return true
		}
	}
	return false
}

var ruleByID = func() map[string]Rule {
	m := map[string]Rule{}
	for _, r := range rules {
		m[r.ID] = r
	}
	for _, r := range source {
		m[r.ID] = r
	}
	for _, r := range migration {
		m[r.ID] = r
	}
	for _, r := range applied {
		m[r.ID] = r
	}
	for _, r := range loaded {
		m[r.ID] = r
	}
	return m
}()

// Migration tells whether the check is one of a migration that can be
// approved: its findings come from the import and the comparison.
func Migration(id string) bool {
	for _, r := range migration {
		if r.ID == id {
			return true
		}
	}
	return false
}

// Rules lists the checks of the generation, in the order of appendix C.
func Rules() []Rule { return append([]Rule{}, rules...) }

// Lookup gives a check by its ID: one of the generation or of a migration.
func Lookup(id string) (Rule, bool) {
	r, ok := ruleByID[id]
	return r, ok
}

// Kind is the letter of a check: "V" for V-101.
func Kind(id string) string {
	k, _, _ := strings.Cut(id, "-")
	return k
}

// KindTitle names a kind of check (spec 11.1).
func KindTitle(kind string) string {
	switch kind {
	case "V":
		return "値"
	case "X":
		return "対向"
	case "M":
		return "移行"
	case "S":
		return "資料"
	case "G":
		return "生成物"
	case "L":
		return "投入ログ"
	case "R":
		return "反映"
	}
	return kind
}

// SeverityTitle is the word for a severity in the reports.
func SeverityTitle(s string) string {
	switch s {
	case Error:
		return "エラー"
	case Warning:
		return "警告"
	case Info:
		return "情報"
	}
	return s
}

// Blocks tells whether a finding of the check keeps the device from being
// generated (spec 9.1 step 2).
func Blocks(id string) bool {
	switch id {
	case "V-101", "V-102", "V-103", "V-104", "V-105", "V-106", "V-107", "V-109":
		return true
	}
	return false
}

// Approvable tells whether a finding of the check can be approved. The ones
// that keep a device from being generated, and the ones that leave its text
// not whole or without a file, cannot: the value or the template is to be put
// right (appendix C 3.2).
func Approvable(id string) bool {
	switch id {
	case "G-601", "G-603", "G-606", "G-608", "G-613":
		return false
	}
	return !Blocks(id)
}

// ScopeOK tells whether the errors of the check can be approved by a scope
// (appendix C 3.2).
func ScopeOK(id string) bool {
	switch id {
	case "M-403", "M-404", "M-405", "S-503":
		return true
	}
	return false
}

// Tally counts what the checks looked at, and keeps why a check did not run:
// a check that was not run is never shown as passed (appendix C 5).
type Tally struct {
	looked  map[string]map[string]int
	skipped map[string]map[string]string
}

// NewTally makes an empty count.
func NewTally() *Tally {
	return &Tally{looked: map[string]map[string]int{}, skipped: map[string]map[string]string{}}
}

// Look notes that a check looked at n things of a device ("" = of the case
// as a whole). Look with 0 tells that the check ran and had nothing to look
// at.
func (t *Tally) Look(rule, device string, n int) {
	if t.looked[rule] == nil {
		t.looked[rule] = map[string]int{}
	}
	t.looked[rule][device] += n
}

// Skip notes why a check did not run for a device ("" = for every device).
// The first reason stays.
func (t *Tally) Skip(rule, device, reason string) {
	if t.skipped[rule] == nil {
		t.skipped[rule] = map[string]string{}
	}
	if _, ok := t.skipped[rule][device]; !ok {
		t.skipped[rule][device] = reason
	}
}

// Looked gives what a check looked at, by device.
func (t *Tally) Looked(rule string) map[string]int { return t.looked[rule] }

// Skipped gives why a check did not run, by device.
func (t *Tally) Skipped(rule string) map[string]string { return t.skipped[rule] }

// Ran tells whether the check ran at all.
func (t *Tally) Ran(rule string) bool { return len(t.looked[rule]) > 0 }

// AllRules lists every check the tool has: the ones of the generation, of a
// migration, of the customer's lists, of the logs of the work and of what is
// on the devices afterwards. For the list of checks in the manual.
func AllRules() []Rule {
	out := append([]Rule{}, rules...)
	out = append(out, migration...)
	out = append(out, source...)
	out = append(out, loaded...)
	return append(out, applied...)
}
