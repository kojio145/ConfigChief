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
	// Fix tells how a finding of the check is put right (fixes.go).
	Fix string
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
	{ID: "V-114", Severity: Info, Title: "機種が空、または書いた機種がパックの機種の定義にない", PerDevice: true},
	{ID: "V-115", Severity: Warning, Title: "参照先がないが、参照先の表をその機器のメーカーパックがまだ読まない"},
	{ID: "V-116", Severity: Warning, Title: "書いた機種が、パックの機種の定義の ID に似ているが当たらない（打ち間違い）", PerDevice: true},
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
	{ID: "V-215", Severity: Error, Title: "IPsec の方式が tunnel_if なのにトンネル IF がない。policy なのに対象通信の行がない（相手が any の行を除く）"},
	{ID: "V-216", Severity: Warning, Title: "弱い暗号（des・md5・dh1・dh2・dh5）を使っている"},
	{ID: "V-217", Severity: Error, Title: "PPPoE の論理 IF・物理 IF がインタフェース表にない"},
	{ID: "V-218", Severity: Warning, Title: "秘密の値が仮の値のまま"},
	{ID: "V-221", Severity: Warning, Title: "経路制御の方針の名前（BGP ネイバー・再配布・ポリシールーティング）が、方針の表にない"},
	{ID: "V-222", Severity: Warning, Title: "経路制御の方針が、定義されているがどこからも使われていない"},
	{ID: "V-223", Severity: Warning, Title: "AS パスのリスト・コミュニティのリストが、どの方針からも使われていない"},
	{ID: "V-224", Severity: Warning, Title: "経路制御の方針に permit の行が 1 つもない"},
	{ID: "V-225", Severity: Warning, Title: "経路制御の方針の行に、その使い道では効かない列の値がある"},
	{ID: "V-226", Severity: Warning, Title: "経路制御の方針の行に、一緒には書けない値・使われない値がある"},
	{ID: "V-227", Severity: Error, Title: "IPv6: DHCPv6 サーバが配るプレフィックス・アドレスの列の組み合わせが合わない"},
	{ID: "V-228", Severity: Error, Title: "RIP: IF・再配布の行に RIP の行がない。認証の列の組み合わせ・RIPng の行に書けない列"},
	{ID: "V-229", Severity: Error, Title: "回線監視: 監視先がアドレス・ネットワークの形でない。送信元アドレス・監視の出口のネクストホップが監視先と別の種類（IPv4・IPv6）"},
	{ID: "V-231", Severity: Error, Title: "IPsec の自分の ID・相手の ID が「種類:値」の形でない"},
	{ID: "V-232", Severity: Warning, Title: "IPsec の行に、その IKE の版では使わない値がある"},
	{ID: "V-233", Severity: Warning, Title: "IPsec の相手監視の再送回数・再送間隔・送り方があるのに、間隔がない"},
	{ID: "V-234", Severity: Error, Title: "IPsec の相手アドレスが、アドレス・any・名前のどれでもない"},
	{ID: "V-235", Severity: Error, Title: "L2TP の「IPsec の設定名」が指す IPsec の行が、受け口を守る形（方式 policy・相手 any・モード transport）でない"},
	{ID: "V-236", Severity: Error, Title: "IPsec: 「（追加）」の列が、IKE の版 2 でない行にある・1 つ目の値が空"},
	{ID: "V-237", Severity: Error, Title: "IPsec 対象通信: ポートがあるのに、プロトコルが udp・tcp でない"},
	{ID: "V-238", Severity: Error, Title: "IPsec: 方式が tunnel_if の行に「内側の VRF」がある"},
	{ID: "V-239", Severity: Error, Title: "IPsec: 方式が policy の行の「外側の VRF」が、自分側 IF の VRF と違う"},
	{ID: "V-241", Severity: Error, Title: "L2TP: 役割に要る値がない（lac の相手アドレス、lns の受け口の IF）"},
	{ID: "V-242", Severity: Warning, Title: "L2TP: トンネルの認証が無効なのに、トンネルのパスワードがある"},
	{ID: "V-243", Severity: Error, Title: "L2TP: 払い出すアドレスの先頭・末尾が片方だけ・逆。接続ユーザの L2TP が lns でない"},
	{ID: "V-244", Severity: Error, Title: "L2 延伸: 張り方（dynamic・manual）と、セッション ID・クッキー・パスワードが合わない。クッキーの形が違う"},
	{ID: "V-245", Severity: Error, Title: "L2 延伸: 同じ設定名で設定が違う。同じ相手・同じ回線番号の行が 2 つある"},
	{ID: "V-251", Severity: Warning, Title: "スイッチポート・サブ IF・VLAN の IF が、VLAN の表に無い VLAN を使っている"},
	{ID: "V-252", Severity: Warning, Title: "トランクのネイティブ VLAN が、許可 VLAN に入っていない"},
	{ID: "V-253", Severity: Error, Title: "同じ LAG のメンバーで、束ね方（固定・LACP・PAgP）が混ざっている"},
	{ID: "V-254", Severity: Warning, Title: "同じ LAG のメンバーで、スイッチポートの設定（モード・VLAN）がそろっていない"},
	{ID: "V-255", Severity: Error, Title: "同じポートのサブ IF で、VLAN 番号が重なっている。タグなしの VLAN が 2 つある"},
	{ID: "V-256", Severity: Warning, Title: "ブリッジグループに入っている IF が、1 つだけ"},
	{ID: "V-257", Severity: Error, Title: "許可 VLAN の書き方が読めない"},
	{ID: "V-258", Severity: Error, Title: "L3 のポート（routed）に、VLAN の設定がある"},
	{ID: "V-261", Severity: Error, Title: "同じ機器・同じ VRF の中で、IF の IPv6 のサブネットが重なっている"},
	{ID: "V-262", Severity: Warning, Title: "案件の中で同じ IPv6 アドレスが複数の IF に付いている"},
	{ID: "V-263", Severity: Warning, Title: "IPv6 スタティック経路のネクストホップが、その機器のどの IF のサブネットにも入らない"},
	{ID: "V-264", Severity: Error, Title: "IPv6 スタティック経路のネクストホップがリンクローカルなのに、出力 IF がない"},
	{ID: "V-265", Severity: Error, Title: "IPv6 の表（IPv6 インタフェース・RA のプレフィックス・OSPFv3 インタフェース）の IF がインタフェース表にない"},
	{ID: "V-266", Severity: Error, Title: "もらったプレフィックスの名前を、その機器のどの IF ももらっていない（DHCPv6-PD）"},
	{ID: "V-267", Severity: Error, Title: "OSPFv3 インタフェース・エリアのプロセスが、OSPFv3 の表にない"},
	{ID: "V-268", Severity: Error, Title: "トンネルの宛先（IPv6・名前）が、種類に合わない"},
	{ID: "V-269", Severity: Info, Title: "IPv6 のサブネットを持つ IF の「RA を送る」が空（既定がメーカーで違う）"},
	{ID: "V-259", Severity: Error, Title: "スパニングツリーの優先度が刻みに合わない（ブリッジは 4096 刻み、ポートは 16 刻み）"},
	{ID: "V-271", Severity: Warning, Title: "VLAN ごとのスパニングツリーの行があるのに、方式が VLAN ごとの木でない"},
	{ID: "V-272", Severity: Warning, Title: "機器全体のスパニングツリーの優先度があるのに、方式が VLAN ごとの木"},
	{ID: "V-276", Severity: Error, Title: "ACL の行の TCP のフラグ・established が、プロトコル tcp 以外の行にある。TCP のフラグが 1 行に 2 つ以上ある"},
	{ID: "V-277", Severity: Error, Title: "ACL の行の ICMP の種類・コードが、プロトコル icmp 以外の行にある。コードがあるのに種類が番号でない"},
	{ID: "V-278", Severity: Error, Title: "遠隔ログインの「接続元の制限」と、方式ごとの制限の両方が書かれている"},
	{ID: "V-279", Severity: Error, Title: "時間帯の行が、種類に合わない（曜日・時刻・日が足りない、形が読めない、absolute が 2 行ある）"},
	{ID: "V-280", Severity: Error, Title: "NAT の行の形が合わない（プールの末尾が先頭より前、static_network の長さ・アドレス、extendable の種類）"},
	{ID: "V-281", Severity: Error, Title: "回線監視の行の形が合わない（list の「まとめる監視」「まとめ方」、送信元 IF と送信元アドレスの両方、icmp だけの列）"},
	{ID: "V-282", Severity: Error, Title: "DHCP の貸し出し時間の形が読めない。デフォルトゲートウェイが 2 つ目以降だけにある"},
	{ID: "V-283", Severity: Info, Title: "秘密の値が、機器が暗号化して保存した形（種別と値）で入っている（別メーカーへは引き継げない）"},
	{ID: "V-284", Severity: Error, Title: "ユーザ: 元に戻せる形で持つのにハッシュ。暗号化した形なのに、元に戻せる形で持たない"},
	{ID: "V-285", Severity: Error, Title: "QoS: 一緒に書けない動作がある（上限の bps と %、キューの長さと WRED など）。元になる値のない動作がある"},
	{ID: "V-286", Severity: Error, Title: "QoS: 1 行に条件が 2 つ以上ある。逆にする条件がない。ほかのクラスが自分自身"},
	{ID: "V-287", Severity: Error, Title: "EtherIP: トンネル IF がブリッジグループに入っていない・延伸する IF がない"},
	{ID: "V-288", Severity: Warning, Title: "PPPoE: 相手も認証するのに認証方式がない。いつもつなぐ回線に、無通信で切る時間がある"},
	{ID: "V-289", Severity: Error, Title: "機器の基本: 元になる値のない値がある（時刻に足すもの、機器内ログの大きさ、秒の端数、DNS サーバの VRF）"},
	{ID: "V-290", Severity: Error, Title: "IPv6 の VRRP: 仮想リンクローカルアドレスが fe80::/10 にない。仮想グローバルアドレスが IF の IPv6 のサブネットに入らない"},
	{ID: "V-291", Severity: Warning, Title: "冗長化の「版 2 の相手とも話す」が、効かない行（方式 vrrp でない・VRRP の版が 3 でない機器）にある"},

	{ID: "X-300", Severity: Info, Title: "相手の機器が案件の中にない（管理外）"},
	{ID: "X-301", Severity: Warning, Title: "IPsec・トンネル: 相手アドレスは案件の中の機器のものだが、相手側に対になる行がない"},
	{ID: "X-302", Severity: Error, Title: "IPsec: IKE の版、モードが両端で違う"},
	{ID: "X-303", Severity: Error, Title: "IPsec: IKE の暗号・ハッシュ・DH グループ・PRF が両端で違う"},
	{ID: "X-304", Severity: Error, Title: "IPsec: ESP の暗号・ハッシュ・PFS・AH・モードが両端で違う"},
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
	{ID: "X-361", Severity: Warning, Title: "再配布: 2 台以上が BGP と OSPF の間で互いに再配布していて、経路制御の条件がない"},
	{ID: "X-362", Severity: Warning, Title: "BGP: 受信の方針が待つコミュニティを、相手の送信の方針が送らない"},
	{ID: "X-363", Severity: Error, Title: "RIP: 送る版を、同じサブネットの相手が受けない"},
	{ID: "X-364", Severity: Error, Title: "RIP: 認証（方式・鍵・鍵の番号）が両端で違う"},
	{ID: "X-365", Severity: Warning, Title: "IPsec: 鍵の書き方（文字・16 進）が両端で違い、鍵を比べられない"},
	{ID: "X-366", Severity: Warning, Title: "IPsec: 提案の候補（追加の値・追加の提案）が両端で違う（折り合う候補はある）"},
	{ID: "X-371", Severity: Warning, Title: "L2 延伸: 相手アドレスは案件の中の機器のものだが、相手側に対になる行がない"},
	{ID: "X-372", Severity: Error, Title: "L2 延伸: 回線番号・張り方・セッション ID・クッキーが両端で合わない"},
	{ID: "X-373", Severity: Error, Title: "L2 延伸・L2TP: トンネルのパスワード・認証が両端で違う・片側だけ"},
	{ID: "X-374", Severity: Error, Title: "L2 延伸: 送信元 IF のアドレスが、相手の指す相手アドレスと違う"},
	{ID: "X-375", Severity: Warning, Title: "L2TP: LAC の相手（LNS）の機器に、受ける側の行がない・名前が合わない"},
	{ID: "X-381", Severity: Warning, Title: "対向: 対向の機器にその IF がない。対向の側が、別の IF を対向としている"},
	{ID: "X-382", Severity: Error, Title: "対向: トランクの許可 VLAN が両端で違う。対向のサブ IF の VLAN が許可 VLAN に入っていない"},
	{ID: "X-383", Severity: Error, Title: "対向: トランクのネイティブ VLAN が両端で違う"},
	{ID: "X-384", Severity: Error, Title: "対向: スイッチポートのモード（アクセスとトランク）・アクセス VLAN が両端で違う"},
	{ID: "X-385", Severity: Error, Title: "対向: LAG の束ね方が両端で合わない。片側だけ LAG"},
	{ID: "X-386", Severity: Warning, Title: "同じ番号の VLAN に、機器によって違う名前が付いている"},
	{ID: "X-387", Severity: Warning, Title: "対向: 速度・二重が両端で違う"},
	{ID: "X-388", Severity: Warning, Title: "対向: 片方向リンクの検出（UDLD）が片側だけ。方式が両端で違う"},
	{ID: "X-391", Severity: Error, Title: "OSPFv3: 同じ IPv6 のサブネットの IF で、エリアが違う"},
	{ID: "X-392", Severity: Error, Title: "OSPFv3: Hello・Dead 間隔が違う"},
	{ID: "X-393", Severity: Error, Title: "OSPFv3: 網の種類が違う"},
	{ID: "X-394", Severity: Error, Title: "OSPFv3: エリアの種類（stub・nssa）が違う"},
	{ID: "X-395", Severity: Warning, Title: "OSPFv3: 片側だけ受動になっている"},
	{ID: "X-396", Severity: Info, Title: "OSPFv3: 同じ IPv6 のサブネットで、片側だけ OSPFv3 が有効"},
	{ID: "X-397", Severity: Error, Title: "BGP（IPv6）: 相手 AS 番号が、相手機器の AS 番号と違う"},
	{ID: "X-398", Severity: Error, Title: "BGP（IPv6）: 片側にしかネイバーの行がない"},
	{ID: "X-399", Severity: Error, Title: "BGP（IPv6）: パスワードが両端で違う・片側だけ"},
	{ID: "X-404", Severity: Error, Title: "OSPFv3: 認証（アルゴリズム・SPI・鍵）が両端で違う"},
	{ID: "X-405", Severity: Error, Title: "VRRP: 同じグループの両端で、片方が版 3 だけ・もう片方が版 2（つながらない）"},
	{ID: "X-406", Severity: Error, Title: "IPv6 の VRRP: 仮想アドレス（リンクローカル・グローバル）・広告間隔が両端で違う"},
	{ID: "X-407", Severity: Error, Title: "EtherIP: 相手の機器のトンネルが、別の種類"},

	{ID: "G-601", Severity: Error, Title: "テンプレートの実行エラー", PerDevice: true},
	{ID: "G-602", Severity: Error, Title: "設計データに値があるのに、生成に使われなかった", PerDevice: true},
	{ID: "G-603", Severity: Error, Title: "新しいメーカーに同じ語がない", PerDevice: true},
	{ID: "G-604", Severity: Error, Title: "往復の不一致（生成したコンフィグを取り込み直した結果が、設計データと違う）", PerDevice: true},
	{ID: "G-605", Severity: Warning, Title: "生成したコンフィグに、取り込み直しで読めなかった行になる行がある", PerDevice: true},
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
	{ID: "M-406", Severity: Error, Title: "新旧対照表（ポート）が決まっていない旧 IF がある"},
	{ID: "M-407", Severity: Error, Title: "新旧対照表（機器）が決まっていない旧機器がある"},
	{ID: "M-408", Severity: Error, Title: "複数の旧 IF が、同じ新 IF に当たっている"},
}

// The checks of the customer's lists (appendix C 1.4). They come from the
// step that reads the lists and compares them with the book.
var source = []Rule{
	{ID: "S-501", Severity: Warning, Title: "外部一覧にあって、パラメータブックにない行"},
	{ID: "S-502", Severity: Info, Title: "パラメータブックにあって、外部一覧にない行"},
	{ID: "S-503", Severity: Error, Title: "外部一覧とパラメータブックで値が違う"},
	{ID: "S-504", Severity: Warning, Title: "外部一覧の値が読めなかった"},
	{ID: "S-505", Severity: Info, Title: "外部一覧のファイルが、前回の読み込みから変わった"},
}

// The checks of what is on the devices after the work (appendix C 2.3).
// They come from the step that reads the logs of input/logs.
var applied = []Rule{
	{ID: "R-801", Severity: Error, Title: "パラメータブックにあるが、機器の設定にない行"},
	{ID: "R-802", Severity: Warning, Title: "機器の設定にあるが、パラメータブックにない行"},
	{ID: "R-803", Severity: Error, Title: "パラメータブックと機器の設定で値が違う"},
	{ID: "R-804", Severity: Error, Title: "追加コンフィグの行が、機器の設定に見つからない"},
	{ID: "R-805", Severity: Info, Title: "機器の設定に読めなかった行があり、追加コンフィグにも当たらない"},
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
	{ID: "L-705", Severity: Error, Title: "ログのプロンプトのホスト名と、流したコンフィグのホスト名が合わない（機器の取り違えの疑い）"},
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
