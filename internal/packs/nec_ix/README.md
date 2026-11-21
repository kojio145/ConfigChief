# NEC UNIVERGE IX パック（nec_ix）

NEC UNIVERGE IX2000 / IX3000 シリーズの running-config を読み取り、設計データからコンフィグを書き出すための決まりです。

- 版: 0.2.0
- 読む行と書く行は、下の「対応一覧」のとおりです。対応一覧にある表は、どれも取り込みと書き出しの両方ができます。書き出した文面は、同じパックで取り込み直して設計データと比べます（往復の確認）
- この機種に機能が無くて書けないもの（L2TPv3、BGP の IPv6、AS パスのリスト、スパニングツリー、LACP など）は、理由を付けて止まるか、外して知らせます（分野ごとの節と「読まない行」）
- **実機の出力では確かめていません**。構文はすべて下の公開資料で確かめたものです。画面に出る順番や字下げの桁数は、資料に決まりが書かれていないため、どの形でも読めるように書いてあります

この文書の並び: 参照した資料 → 対応一覧（読める行と入る表）→ 取り込みと書き出しの共通の決まり → 分野ごとの節（読み方の決まり・書き出し・別メーカーのデータの置き換え）→ 読まない行 → 確かめていないこと → 投入ログの読み方。

## 参照した資料

NEC の公開ページから 2026-11-14 に入手し、2026-11-20〜21 に同じ版を入手し直して読みました。

| 略号 | 資料の名前 | 版 |
|---|---|---|
| CRM | IX2000/IX3000 シリーズ コマンドリファレンスマニュアル | ソフトウェア ver.10.11 対応、第1.1版（2026年4月） |
| FD | IX2000/IX3000 シリーズ 機能説明書 | ver.10.11 対応、第1.1版（2026年5月） |
| EX | IX2000/IX3000 シリーズ 設定事例集 | ver.10.11 対応、10.11a版（2026年5月） |
| UM | IX2000 シリーズ・IX3000 シリーズ 取扱説明書（2 章・9 章のポートの数と名前）と、NEC 公式 Web のハードウェア仕様のページ。機種の定義（`models.yaml`）に使いました | ver.10.11 対応 |
| WEB | NEC 公式 Web「コンソールでの初期設定」「FAQ」 | 版の表記なし |
| FAQ | NEC 公式 Web の FAQ「VLAN」「ブリッジ」「Ethernet」 | 版の表記なし |

構文は CRM で確かめ、`defaults.yaml` の各行に CRM のページ番号を書いてあります。分野ごとに読んだ所は次のとおりです。

| 分野 | 読んだ所 |
|---|---|
| 機器の基本 | CRM の 2 章（`banner`・`service password-encryption`・`terminal timeout`・`username`）、25 章（`ip name-server`）、26 章（`ntp ip enable`）、32 章（`logging buffered`・`logging subsystem`・`logging timestamp`・`syslog ip source`） |
| インタフェース・VLAN・内蔵スイッチ・ブリッジ | CRM の 2 章（`system interfaces`・`system subinterfaces`）、3 章（デバイスコンフィグモードの全部: `mdi-mdix`・`port … mirror-port`・`port … qos default-priority`・`router-port`・`qos scheduler`、`dot1q min-frame-size`、`encapsulation dot1q`、`bandwidth`、`auto-connect`・`idle-time`・`device` の下の `description`）、6 章（ブリッジの全部）、8 章（`ip tcp adjust-mss`・`ip ufs-cache enable`）、20 章（ループ検出）。FD の 1 章の上限の表、2.6（SW-HUB: ミラー・リンクアグリゲーション）、2.8（ループ検出）、2.9（ブリッジ: タグ VLAN の透過、ポート VLAN どうしのブリッジ）、VLAN の節。EX の VLAN・ブリッジ・内蔵スイッチの章。FAQ「VLAN」「ブリッジ」「Ethernet」 |
| スタティックルート・OSPF・BGP・VRF | CRM の 8 章（`ip route`。p.8-11〜8-14）、10 章（OSPF の `area … virtual-link`・`distance`・`distribute-list`・`originate-default`、BGP の `aggregate-address`・`default-local-preference`・`distance`・`timers`・`neighbor … advertisement-interval`・`route-reflector-client`、p.10-34、p.10-55〜10-66、ピアグループの節 p.10-68〜10-72）。FD の 2.17（VRF-Lite の設定）、2.22（ルーティングの設定: BGP・ポリシールーティング、「ピアグループ設定」の節 = グループを使った設定と、ネイバーごとに書いた設定を「同等」として並べた例） |
| 経路制御の方針（ルートマップ・ポリシールーティング） | CRM の 9 章（`ipv6 policy route-map`・`ipv6 local policy route-map`）、23 章「高度な経路制御編」（`ip prefix-list … any`、`match ipv6 address`・`set ipv6 next-hop`・`set community … delete`） |
| アクセスリスト・NAT・回線監視・DHCP | CRM の 8 章（NAT・NAPT・DHCP）、13 章（`ip filter`）、17 章（ネットワークモニタ）、24 章（アクセスリスト。`ip access-list` の SEQUENCE）。FD の 2.28（ネットワークモニタ: 複数のイベントの動き）、2.30（パケットフィルタ: 複数のリストの評価の順）。EX（GRE・ESP のプロトコル番号の記述） |
| QoS・PPPoE | CRM の 7 章（PPP・PPPoE。`authentication request`・`pppoe server`）、14 章（QoS）。FD の 2.12.3（PPPoE サーバの設定） |
| トンネル・EtherIP | CRM の 12 章（`tunnel mode`・`tunnel df-bit`）。FD の 2.10（Ether over IP。2.10.1 EtherIP 機能）。EX の EtherIP の例（IKEv1・IKEv2） |
| IPsec・IKE | CRM の 13 章（p.13-24〜13-33 の `ipsec autokey-map`・`ipsec dynamic-map`・`ipsec local-id`・`ipsec policy`・`ipsec remote-id`、`ipsec anti-replay`、p.13-42 の `ike policy`、`ike local-id`、p.13-56〜13-66 の IKEv2 のプロファイルの行、`ikev2 authentication`）と NHRP の章。FD の 2.17（VRF-Lite: 対応する機能の表と多対地 IPsec 構成）、2.35（IPsec の設定。2.35.1 の「不定アドレス宛フェーズ 2 開始機構」、2.35.2 の「(b) 物理インタフェースを使用する場合」、2.35.6「トランスポートモード」）、2.38（IKEv2）、2.39（ダイナミック VPN）。EX の IPsec の章（トランスポートモードの例、`ipsec dynamic-map … ike <ポリシー>`・`ike-binding` の例） |
| L2TP | CRM の 7 章（PPP・PPPoE）、12 章（トンネル・L2TP。`l2tp password`）、13 章（`url-list`）。FD の 2.36（L2TP LNS 機能。2.36.4 L2TP LNS/IPsec の基本設定）、2.37（L2TP LAC 機能。2.37.3 L2TP LAC および PPPoE サーバ機能の基本設定）、対応していない機能の一覧（「L2TPv3 は対応しておりません」） |
| IPv6 | 「IPv6」の節に書いてあります |
| IPv6 の宛先・回線監視・RIP | CRM の 10 章（RIP・RIPng、OSPFv3 のコマンド一覧）、16 章（`vrrp <番号> ipv6`）、17 章（`event … ipv6 unreach-host`・`ipv6 unreach-route`）、22 章（`snmp-agent ipv6 …`）、23 章（`match ipv6 next-hop prefix-list`）、25 章（`ipv6 name-server`）、26 章（`ntp server`）、32 章（`syslog ipv6 host`・`syslog ipv6 source`）、12 章（`tunnel mode map-e`）。FD の 2.33.10（MAP-E の設定） |
| VRRP の版 3（IPv6 の VRRP） | CRM の 16 章（p.16-3〜16-5: `vrrp <番号> ipv6`（リンクローカルが 1 つ以上要る・複数書ける）、`vrrp <番号> timers advertisement [csec] <間隔>`（版 3 は 30 秒まで）、`vrrp <番号> authentication`（版 2 のときだけ有効））、17 章（p.17-4 の `action … ip decrement-vrrp-priority`）。FD の 2.27（2.27.1 基本設定: 「IPv4、IPv6 を含め、装置内で同じ VRID を使用することはできません」、「IPv4 と IPv6 の動作差分」の表: IPv4 は版 2（RFC 2338）・IPv6 は版 3（RFC 5798）・「VRRPv3 の IPv4 には対応していません」・IPv6 に認証は無い、設定例、2.27.6 タイマ設定）。2026-11-21 に読みました |

端末ログの形（プロンプトとページ送りの跡）だけは、資料の本文を読めていません。NEC の公式 Web の FAQ とトラブルシューティングのページを開こうとしましたが取得できず（2026-11-14）、検索結果に出た要約で見ただけです（「確かめていないこと」）。

## 確かめた機種と版

- 実機: なし
- 資料: ソフトウェア ver.10.11 のマニュアル。これより古い版で書き方が違うコマンドは、確かめていません

## 対応一覧

このパックが読む行と、その値が入る表です。ここにある表は、書き出しもできます（書き方は分野ごとの節）。VLAN・内蔵スイッチ・ブリッジの行は「VLAN とスイッチ」の節、IPv6 の行は「IPv6」の節、ルートマップ・BGP・OSPF の細かい行は「経路制御・BGP・OSPF の残り」の節に表があります。

### 機器の基本

| 分野 | 読める行 | 入る表 |
|---|---|---|
| 機器全体 | `hostname`、`timezone`、`ip name-server <アドレス>`、`ip name-server vrf <VRF> <アドレス>`（CRM 25 章）、`snmp-agent location` / `contact` | devices（DNS サーバの VRF は「DNS サーバの VRF」） |
| 機器内のログ | `logging buffered [<バイト>] [cyclic\|one-time]`（CRM p.32-17） | devices の「機器内にログを溜める」「機器内ログの大きさ」「機器内ログがいっぱいのとき」 |
| ログの重要度 | `logging subsystem all <重要度>`、`logging subsystem <対象> <重要度>`（重要度は debug・info・notice・warn・error。CRM 32 章） | devices の「ログに残す重要度」、log_subsystems |
| 時刻の付け方 | `logging timestamp timeofday\|uptime\|datetime`（CRM p.32-20。timeofday は `time`） | devices の「ログの時刻の付け方」 |
| パスワードの暗号化 | `service password-encryption`（CRM p.2-5） | devices の「パスワードを暗号化して保存」 |
| 機能の入り切り | `ip ufs-cache enable`（CRM p.8-17）、`ipv6 ufs-cache enable`（CRM p.9-19）、`ntp ip enable`（CRM p.26-3） | devices の「転送のキャッシュ」「転送のキャッシュ（IPv6）」「NTP サーバとして動く」 |
| ユーザ | `username <名前> password hash <値> [administrator\|operator\|monitor]`、`username <名前> password plain [0\|1] <値> [administrator\|operator\|monitor]`（入力の形。CRM p.2-12。書き出した文面を読み直すためのものです） | users |
| リモート接続 | `telnet-server ip enable`、`ssh-server ip enable`、`telnet-server ip access-list <名前>`・`ssh-server ip access-list <名前>`（別々のリストでも）、`terminal timeout <分>` | remote_access（リストは acl。Telnet と SSH で違えば acl_telnet・acl_ssh） |
| NTP | `ntp [vrf <VRF>] server <アドレス> [source <IF>]`（IPv6 のアドレスは VRF なしの形だけ）、`ntp source <IF>` | ntp_servers |
| syslog | `syslog [vrf <VRF>] ip host <アドレス>`、`syslog facility <名前>`、`syslog [vrf <VRF>] ip source <アドレス>`（CRM p.32-26） | syslog_servers（送信元は、そのアドレスを持つ IF の名前を「送信元 IF」に） |
| IPv6 の宛先 | `ipv6 name-server <アドレス>`、`syslog ipv6 host <アドレス>`・`syslog ipv6 source <アドレス>`、`snmp-agent ipv6 enable`・`snmp-agent ipv6 community <名前> [ro\|rw]`・`snmp-agent ipv6 host <アドレス> <コミュニティ> [version 1\|2]`・`snmp-agent ipv6 trap-source <IF>` | devices（DNS サーバ）、syslog_servers、snmp_communities、snmp_hosts |
| SNMP | `snmp-agent ip enable`、`snmp-agent ip community <名前> [<リスト>] [ro\|rw]`、`snmp-agent ip host <アドレス> <コミュニティ> [version <1\|2>]`、`snmp-agent ip trap-source <IF>` | snmp_communities、snmp_hosts |

### インタフェースと L2

| 分野 | 読める行 | 入る表 |
|---|---|---|
| ポート | `device <名前>` の下の `speed`、`duplex`、`description <文>`（CRM 3 章） | interfaces（`<名前>.0` の行。説明は「ポートの説明」） |
| インタフェース | `description`、`ip address <アドレス>/<長さ> [secondary]`、`ip address dhcp` / `ipcp`、`ip unnumbered <IF>`、`no ip address`、`ipv6 address`、`shutdown` / `no shutdown`、`ip vrf forwarding`、`encapsulation dot1q <番号>`、`ip mtu`、`ip tcp adjust-mss <数>\|auto`（CRM p.8-16）、`bandwidth <kbps>`、`ip filter <リスト> <番号> in\|out`（同じ向きに並べた 2 つ目以降も）、`ip proxy-arp`、`ip dhcp-relay server <アドレス>` | interfaces、vrfs |
| VLAN・内蔵スイッチ・LAG・ブリッジ・ポートミラー・ループ検出 | 「VLAN とスイッチ」の節 | switchports、lags、l2_settings、bridge_groups、mirror_sessions、interfaces |

### 経路

| 分野 | 読める行 | 入る表 |
|---|---|---|
| スタティックルート | `ip route [vrf <VRF>] <宛先>/<長さ>\|default <次ホップ>\|<IF> [connected] [metric <数>] [distance <数>] [tag <数>]`、`ip route <宛先> <IF> dhcp`（DHCP でもらったゲートウェイ。next_hop が `dhcp`） | static_routes |
| OSPF | `ip router ospf <番号>` の下の `router-id`、`network <アドレス>/<長さ>\|<IF> area <エリア>`、`area <エリア> stub\|nssa [no-summary]`、`area <エリア> range … advertise\|not-advertise`、`redistribute connected\|static\|rip\|bgp [metric <数>] [metric-type 1\|2] [tag <数>] [route-map <名前>]`、`originate-default [always] [metric <数>]`、`passive-interface <IF>`。インタフェースの下の `ip ospf cost` / `priority` / `hello-interval` / `dead-interval` / `network` / `authentication` / `authentication-key` / `message-digest-key`。`distance`・`distribute-list`・`area … virtual-link` などは「経路制御・BGP・OSPF の残り」の節 | ospf_processes、ospf_interfaces、ospf_areas、ospf_ranges、ospf_redistribute、ospf_virtual_links |
| RIP・RIPng | 「RIP と RIPng」の節 | rip_processes、rip_interfaces、rip_redistribute |
| BGP | `router bgp <AS>` の下の `router-id`、`neighbor <アドレス> remote-as` / `update-source` / `description` / `password` / `ebgp-multihop` / `timers` / `shutdown`。`address-family ipv4 unicast` の下の `network`、`redistribute`（`redistribute ospf <番号>` も）、`neighbor <アドレス> distribute-list <名前> in\|out` / `route-map <名前> in\|out` / `next-hop-self` / `send-default`。`router bgp` の下の `vrf <VRF>` のブロック（ネイバーと `address-family ipv4 unicast`）。ピアグループ・`aggregate-address`・`distance` などは「経路制御・BGP・OSPF の残り」の節 | bgp_processes、bgp_neighbors、bgp_networks、bgp_redistribute、bgp_aggregates（VRF のブロックは、それぞれの表の VRF の行） |
| VRRP | `vrrp enable`、インタフェースの下の `vrrp <番号> ip` / `priority` / `preempt` / `timers advertisement <秒>` / `authentication` | fhrp |
| VRRP の版 3（IPv6） | インタフェースの下の `vrrp <番号> ipv6 <アドレス>`（1 つ目のリンクローカルと、グローバルのアドレス）と、同じ番号の `priority` / `preempt` / `no … preempt` / `timers advertisement <秒>` / `timers advertisement csec <センチ秒>` | ipv6_fhrp |
| 経路制御の方針（ルートマップ） | `route-map <名前> permit\|deny <番号>` の下の `match ip address prefix-list <名前>`、`match ip address access-list <名前>`、`match ip next-hop prefix-list <名前>`、`match community <値> … [exact-match]`、`match interface <IF>`、`match metric <数>`、`match tag <数>`、`set local-preference`、`set metric [+\|-]<数>`、`set metric-type type-1\|type-2\|internal`、`set as-path prepend <AS> …`、`set community <値> … [additive]`、`set community none`、`set origin igp\|incomplete`、`set tag`、`set ip next-hop <アドレス>\|self`、`set ip default next-hop <アドレス>`、`set interface <IF>`、`set default interface <IF>`。インタフェースの下の `ip policy route-map <名前>`、`ip local policy route-map <名前>`。IPv6 の条件と動作・`set community … delete` は「経路制御・BGP・OSPF の残り」の節 | route_policies、interfaces の policy_route 列、devices の local_policy_route 列 |

### リスト・NAT・回線監視

| 分野 | 読める行 | 入る表 |
|---|---|---|
| アクセスリスト | `ip access-list <名前> [<番号>] permit\|deny ip\|tcp\|udp\|icmp\|<プロトコルの番号> src … [sport …] dest … [dport …]`。ポートは `eq` / `lt` / `gt` / `neq` と数字 1 つ、`range <開始> <終了>`、`any`。TCP のフラグ（`tcp` の後ろの `established`・`ack`・`fin`・`psh`・`rst`・`syn`・`urg` のどれか 1 つ）、ICMP の `type <番号> [code <番号>]` とメッセージの名前（`echo`・`echo-reply`・`unreachable`・`port-unreachable`・`time-exceeded` など 39 語）、末尾の `tos`・`precedence`・`dscp`・`fragments` | acls、acl_entries（flags・icmp_type・icmp_code・tos・precedence・dscp・src_port・dst_port・protocol） |
| アクセスリストの番号 | `ip access-list <名前> sequence-mode [<間隔>]` と、番号付きの行 | acls（seq_step）、acl_entries（seq） |
| フィルタ | 同じ向きに並べた `ip filter <リスト> <番号> in\|out`、`ipv6 filter …` | interfaces・ipv6_interfaces（acl_in・acl_in_more、acl_out・acl_out_more） |
| プレフィックスリスト | `ip prefix-list <名前> <番号> permit\|deny <アドレス>/<長さ> [min <数>] [max <数>]`、`ip prefix-list <名前> <番号> permit\|deny any` | prefix_lists |
| NAT | インタフェースの下の `ip napt enable`、`ip napt inside list <リスト>`、`ip napt inside list <リスト> outside <アドレス>`、`ip napt address <アドレス>`、`ip napt static <内側アドレス> <プロトコル> <ポート>`、`ip napt service <名前> <サーバ> <変換後のポート\|none> tcp\|udp <ポート>`、予約されたサービス名だけの形（`ip napt service dns\|http\|https\|snmp\|ssh\|telnet <サーバ>`）、`ip nat enable`、`ip nat static <内側アドレス> <外側アドレス>`、`ip nat static network <内側>/<長さ> <外側>/<長さ>`、`ip nat dynamic list <リスト> pool <名前>` と `ip nat pool <名前> <先頭> <末尾>` | nat_rules（種類 pat_interface・pat_pool・static_port・static・static_network・dynamic_pool） |
| 回線監視 | `watch-group <名前>` の下の `event <番号> [vrf <VRF>] ip unreach-host <宛先> [<IF> [<ネクストホップ>]] [source <IF>]`、`event <番号> [vrf <VRF>] ip unreach-route <宛先>/<長さ> [<ネクストホップ>\|<IF>]`、`event <番号> interface-down <IF>`、1 つの監視グループに複数の `event`（番号が違うもの、同じ番号で `sub` が違うもの）、`probe-counter variance\|restorer <回数>`、`probe-timer variance\|restorer <秒>`、`action <番号> ip shutdown-route <宛先>/<長さ> [<次ホップ>]`、`action <番号> shutdown-interface <IF>`、`action <番号> ip decrement-vrrp-priority <VR 番号> <減らす値>`。`network-monitor <名前> enable` | tracks（via_if・via_next_hop・vrf、type が icmp・interface・route・list、members・list_rule、restore_interval、shutdown_ifs）、static_routes の track 列、fhrp（track・track_decrement） |
| 回線監視（IPv6） | `watch-group <名前>` の下の `event <番号> [sub <番号>] ipv6 unreach-host <宛先> <IF> [<ネクストホップ>] [source <アドレス>]`、`event <番号> [sub <番号>] ipv6 unreach-route <宛先>/<長さ> [<ネクストホップ>\|<IF>]` | tracks |

### DHCP・PPPoE・QoS

| 分野 | 読める行 | 入る表 |
|---|---|---|
| DHCP サーバ | `ip dhcp enable`、`ip dhcp profile <名前>` の下の `assignable-range`、`subnet-mask`、`default-gateway`、`dns-server`、`domain-name`、`lease-time <秒>`（どの長さでも）、`lease-time infinite`。`ip dhcp excluded-address <開始> <終了>`、インタフェースの下の `ip dhcp binding <名前>` | dhcp_pools |
| DHCP リレー | `ip dhcp-relay enable` | interfaces の dhcp_relay 列 |
| PPPoE | `ppp profile <名前>` の下の `authentication myname` / `password` / `accept`、`authentication request chap\|pap\|chap-pap`（CRM 7 章）、`lcp echo-interval <秒>`。インタフェースの下の `ppp binding <名前>`、`pppoe service-name <名前>`、`no auto-connect`（CRM p.3-3）、`idle-time <秒>`（CRM p.3-4） | pppoe（「相手も認証する」「通信があるときだけつなぐ」「無通信で切る時間」も） |
| QoS のクラスと条件 | `class-map [match-all\|match-any] <名前>` の下の `match ip access-list <リスト> [cos <数>] [<キュー>]`、`match any [<キュー>]`、`match ipv6 access-list <リスト> [<キュー>]`、`match input-interface <IF> [<キュー>]`、`match cos <数> [<キュー>]`、`match qos-group <数> [<キュー>]`、`match local-generate-packet [<キュー>]`、`match rtp port <先頭> <幅> [<キュー>]`、`match ngn <トンネル IF> [qos-group <数>] [<キュー>]`。キューは high・medium・normal・low・sub-a〜sub-d。アクセスリストの末尾の `precedence <数>`、`dscp <数>` | qos_classes、qos_class_matches（「CoS」「IPv6 の ACL」「受信 IF」「QoS グループ」「機器自身が出す通信」「RTP のポートの先頭・幅」「NGN のトンネル IF」「キュー」、dscp・precedence）、acl_entries の dscp・precedence 列 |
| QoS のポリシー | `policy-map <名前>` の下の `class <名前>\|class-default\|class-local\|class-dynamic` と、その下の `priority [[mbps] <kbps> [<バースト>]]`、`priority percent <%> [<バースト>]`、`bandwidth [mbps] <kbps>`、`bandwidth percent <%>`、`shape [kbps\|mbps] <bps> [<Bc> [<Be>]]`、`set ip dscp`、`set ip precedence`、`set cos <数>`、`set qos-group <数>`、`set ipv6 dscp <数>`、`set ipv6 precedence <数>`、`queue-limit <数> …`（8 つまで）。インタフェースの下の `service-policy input\|output <名前>`、`service-policy output default-policy-map-ngn`、`traffic-shape rate [kbps\|mbps] <bps> [<Bc> [<Be>]]` | qos_policies（クラス `default`・`local`・`dynamic`、「付ける CoS」「付ける QoS グループ」「付ける DSCP（IPv6）」「付ける precedence（IPv6）」「キューの長さ」）、qos_bindings（「NGN 用の既定のポリシー」も） |

### トンネル・IPsec・L2TP

| 分野 | 読める行 | 入る表 |
|---|---|---|
| トンネル（GRE） | インタフェースの下の `tunnel mode gre`、`tunnel source`、`tunnel destination`、`tunnel key`、`tunnel keepalive [period <秒>] [retries <回数>]` | tunnels |
| L2 延伸（EtherIP） | トンネル IF の下の `tunnel mode ether-ip ip`、`tunnel source`、`tunnel destination`、`bridge-group <番号>`（FD 2.10.1）。IPv6 の網を通る形は `tunnel mode ether-ip ipv6`、`tunnel source`、`tunnel destination <IPv6>` | tunnels（種類 `etherip`。IPv6 の網を通る形は「宛先（IPv6）」）、interfaces の「ブリッジグループ」 |
| IPsec（IKE 版 1・トンネル IF 型） | `ike proposal`、`ike policy <名前> {peer <アドレス>｜peer any｜peer-fqdn-ipv4 <名前>} key <鍵> [key-type char｜hex] [mode …] [<プロポーザル>]`（`key-type` は鍵の前でも読みます）、`ike local-id`・`ike remote-id <ポリシー> keyid｜address｜fqdn｜user-fqdn <値>`、`ike keepalive <ポリシー> <秒> [<回数>]`、`ike nat-traversal [policy <ポリシー>] [keepalive <秒>] [force]`、`ipsec autokey-proposal <名前> [esp-…] [esp-…] [ah-…] [lifetime time <秒>｜lifetime both <秒> <キロバイト>]`、`ipsec autokey-map <名前> <リスト> {peer <アドレス>｜peer-fqdn-ipv4 <名前>} [pfs …] [<プロポーザル>]`、`ipsec dynamic-map <名前> <リスト> [pfs …] [ike｜ike-binding <ポリシー>] [<プロポーザル>]`（ポリシーは、プロポーザルの前でも後ろでも読みます）、`ipsec local-id`・`ipsec remote-id <マップ> <アドレスか、ネットワーク/長さ>`（マップが相手に伝える範囲）。プロポーザルは、カンマで並べた 2 つ目以降も読みます（IKE は 4 つ、IPsec は 8 つまで）。インタフェースの下の `tunnel mode ipsec`、`ipsec policy tunnel <マップ>` | ipsec_tunnels、ipsec_selectors（マップが相手に伝える範囲）、ipsec_proposals（2 番目以降の候補）、tunnels |
| IPsec（IKE 版 2・トンネル IF 型） | `ikev2 authentication psk id fqdn｜keyid｜ipv4｜rfc822-addr <値> key char｜hex <鍵>`、`ikev2 default-profile`・`ikev2 profile <名前>` の下の `local-authentication psk [id …]`、`sa-proposal enc｜integrity｜prf｜dh <値> […]`（値は 1 つでも、空白で並べた 2 つ以上でも）、`sa-lifetime`、`child-proposal enc｜integrity <値> […]`（`child-proposal integrity off` は「ESP ハッシュ」の `none`）、`child-pfs <グループ> […]`、`local-ts`・`remote-ts ipv4 [protocol tcp｜udp] [address <ネットワーク/長さ>] [port <番号> <同じ番号>]`、`child-lifetime`、`dpd [interval <秒>｜off]`、`retransmit count <回数> interval <秒>｜backoff`、`nat-traversal [keepalive <秒>] [force]｜off`、`ipsec-mode tunnel｜transport`。インタフェースの下の `tunnel mode ipsec-ikev2`、`ikev2 binding <プロファイル>`、`ikev2 peer <アドレス>｜any [authentication [psk] [id …]]`、`ikev2 peer-fqdn-ipv4 <名前> […]`、`ikev2 connect-type auto｜trigger`、`ikev2 source-address <アドレスか IF>` と、プロファイルの下と同じ行に `ikev2` を付けた形（`ikev2 dpd interval <秒>` など） | ipsec_tunnels（並べた値は「（追加）」の列）、ipsec_selectors（トラフィックセレクタ）、tunnels |
| IPsec（IKE 版 1・方式 policy） | トンネル IF でないインタフェースの下の `ipsec policy tunnel <マップ> [out]`・`ipsec policy transport <マップ>`。マップ・IKE・プロポーザルの行は上と同じ。マップのリストの行（`ip access-list <名前> [<番号>] permit ip｜udp｜tcp src … [sport eq <番号>] dest … [dport eq <番号>]`）が対象通信 | ipsec_tunnels、ipsec_selectors、ipsec_proposals |
| L2TP を守る IPsec（L2TP/IPsec の受け口） | LNS のトンネル IF の下の `ipsec policy transport <マップ>` が指す `ipsec dynamic-map`、相手が any の `ike policy`、プロポーザル。LNS の行の「IPsec の設定名」が、その IPsec の行を指します | ipsec_tunnels、ipsec_proposals、l2tp_groups（IPsec の設定名） |
| IPsec で守る EtherIP | IKE 版 1: トンネル IF の下の `tunnel mode ether-ip ipsec` と `ipsec policy transport <マップ> with-id-payload`（マップ・IKE・プロポーザルは、トンネル IF 型の版 1 と同じ行）。IKE 版 2: `tunnel mode ether-ip ipsec-ikev2` と、トンネル IF 型の版 2 と同じ `ikev2 …` の行（`ikev2 ipsec-mode transport`） | tunnels（種類 etherip。版 1 は送信元と宛先が空、版 2 は `ikev2 peer`・`ikev2 source-address` が宛先・送信元）、ipsec_tunnels（方式 tunnel_if・IPsec のモード transport） |
| 暗号化して保存した鍵 | `ike policy … key <値> key-type secret 1`（`key-type` は鍵の前でも後ろでも）、`ikev2 authentication psk id … key secret <値>`、`l2tp password secret <値>`（CRM p.12-12） | 鍵・パスワードの列に `secret <値>` の形のまま（別紙 A 3.26）。`key-type secret 0` は平文の鍵 |
| VRF の中の IPsec | トンネル IF の下の `ip vrf forwarding <VRF>` | interfaces の「VRF」（トンネル IF 型の IPsec の「内側」。IPsec の表に列は足していません） |
| L2TP（LNS。リモートアクセスの受け口） | トンネル IF の下の `tunnel mode l2tp-lns ipsec`（古い表示の `tunnel mode l2tp ipsec` も）、`ppp binding <名前>`、`ipsec policy transport <名前>`、`l2tp password plain <値>`、`l2tp session idle-time <秒>`。`ppp profile <名前>` の下の `authentication request chap\|pap\|chap-pap`、`authentication password <接続 ID> <パスワード>`、`ipcp provide-ip-address range <先頭> <末尾>`、`ipcp provide-static-ip-address <接続 ID> <アドレス>`、`ipcp provide-remote-dns <アドレス> [<アドレス>]`、`lcp pfc`、`lcp acfc`、`ipcp ip-compression` | l2tp_groups、l2tp_users |
| L2TP（LAC） | トンネル IF の下の `tunnel mode l2tp-lac ip`、`tunnel destination <アドレス>` / `tunnel destination fqdn <名前>`、`l2tp password plain <値>`、`l2tp session idle-time <秒>`。PPPoE を受ける IF の下の `pppoe l2tp interface <トンネル IF> url-list <名前>`。`url-list <名前> permit domain *<ドメイン>` | l2tp_groups |
| PPPoE を受ける側（LAC の配下） | PPPoE を受ける IF の下の `pppoe server`（CRM p.7-27）と `ppp binding <プロファイル>`、そのプロファイルの `authentication request …`（FD 2.37.3） | l2tp_groups（役割 lac の行）の「利用者の認証」 |

### IPv6

| 分野 | 読める行 | 入る表 |
|---|---|---|
| IPv6（アドレス・RA・DHCPv6・経路・フィルタ・プレフィックスリスト・OSPFv3・IPv6 の網のトンネル） | 「IPv6」の節 | ipv6_interfaces、ipv6_ra_prefixes、ipv6_routes、ipv6_acl_entries、ipv6_prefix_lists、ospfv3_processes、ospfv3_interfaces、ospfv3_areas、tunnels、interfaces |
| IPv6 の MSS と OSPFv3 の再配布の指定 | `ipv6 tcp adjust-mss <数>\|auto`、OSPFv3 の `redistribute connected\|static\|bgp\|rip [metric <数>] [metric-type 1\|2] [tag <数>] [route-map <名前>]` | ipv6_interfaces（tcp_mss）、ospfv3_redistribute |
| DHCPv6 サーバが配るプレフィックス | 「IPv6」の節 | ipv6_interfaces |
| IPv6 のポリシールーティング | 「経路制御・BGP・OSPF の残り」の節 | ipv6_interfaces（policy_route）、devices（local_policy_route_v6） |

## 取り込みの共通の決まり

### この機種ならではの読み方

- **機能ごとに「有効にする行」があります。** `snmp-agent ip enable`、`vrrp enable`、`ip dhcp enable`、`ip dhcp-relay enable`、`network-monitor <名前> enable` が無いと、その機能の行は表に入らず「未対応」に戻ります
- **名前でつながる設定は、つながった先まで読めたときだけ表に入ります。** `ppp binding` と `ppp profile`、`ip dhcp binding` と `ip dhcp profile`、`ipsec policy tunnel` と `ipsec autokey-map` と `ike policy`（相手のアドレスが同じもの）がその例です。つながらない行は「未対応」に戻ります
- **ポートとインタフェースが別のブロックです。** `device GigaEthernet1` の速度と二重は、`GigaEthernet1.0` の行に入れます。`GigaEthernet1.1` 以降はサブインタフェースです
- **有効・無効の両方が画面に出ます。** `shutdown` も `no shutdown` も読みます。どちらも無いポート・サブインタフェース・トンネルは「停止」とみなします（既定が停止のため。CRM p.3-6）。Loopback と Null にはどちらの行も出ないので、「有効」とみなします（「確かめていないこと」を参照）
- そのほかの読み方は、分野ごとの節の「読み方の決まり」にあります

### 変更の投入の書き方（`change`。「ひな形から作る」の今との差）

`pack.yaml` の `change`（別紙 B 2.5）。設定モードへは `configure`、抜けるのは `exit`（グローバルコンフィグモードから動作モードへ）、ブロックを 1 段抜けるのも `exit`（機能説明書 2.1「すべてのモードは、exit コマンドで抜けることができます」）、行を消すのは `no` の形、`no shutdown` ⇄ `shutdown`（コマンドリファレンス）。`end` という命令は資料に無いので使わない。

- **確かめていないこと**: 行の全部に `no` を付けた形が通らない命令（引数を省く形しか取らない命令）が残っているか。保存（`write memory`）は付けない

## 端末ログの読み方

端末ソフトのログ（ログインから `show running-config` まで入ったもの）は、設定の部分だけを切り出して読みます。決まりは別紙 B 2.4、書いてある場所は `pack.yaml` の `session` です。

- **プロンプト。** 行の頭の「ホスト名」「かっこに入ったモード（あれば）」「`#` か `>`」です。この機種は設定のモード（`ホスト名(config)#`）で `show running-config` を打ちます
- **設定を表示するコマンド。** `show running-config` と `show startup-config` を、省かずに打った行だけです。短い形は確かめていないので読みません。短い形で取ったログは、全体をコンフィグとして読んだうえで「コマンドの行が見つかりません」と注意を出します
- **ページ送りの跡。** `--More--` です（大文字でも小文字でも読みます）。跡だけの行は「無視（ページ送りの跡）」にし、跡の後ろに設定が続く行は、跡を取り除いて読みます（そのときは注意を出します）
- 同じログにコマンドが何回もあるときは、最後のものを読みます

### 値を入れない規則

別紙 B 3.2 で認められている 2 つの場合だけです。IPv6 の 3 つ（`dhcp6.client`・`dhcp6.server`・`ospf3.area`）は「IPv6」の節の「読み方の決まり」にあり、合わせて 7 つです。

| 規則 | 行 | 理由 |
|---|---|---|
| `device` | `device <名前>` | 子の行を持つ入れ物の行 |
| `bgp.af` | `address-family ipv4 [unicast]` | 子の行を持つ入れ物の行 |
| `if.ipv4.none` | `no ip address` | 「ipv4 の列が空」と同じ意味の行 |
| `qos.policy` | `policy-map <名前>` | 子の行を持つ入れ物の行 |

### 読んで、表には入れない語

行は「取り込み済み」になりますが、次の語は入れる列が無いため捨てています。書き出すときは、決まった値に戻します。IPv6 の行の分は「IPv6」の節にあります。

| 行 | 捨てる語 |
|---|---|
| `ip filter <リスト> <番号> in\|out`、`ipv6 filter …` | 番号（リストの順番だけを持ちます。書き出すときは 1、2、3 … と振ります） |
| `ip access-list … sport any`・`dport any` | `sport any`・`dport any`（すべてのポート。ポートの列が空と同じ意味で、書き出すときは書きません） |
| `ip napt service <名前> …` | サービスの名前（書き出すときは、NAT の行の名前を使います） |
| `watch-group <名前> <番号>`、`event <番号> [sub <番号>] …`、`action <番号> …` | 番号（event の並びと、`sub` で組になっていることだけを持ちます） |
| `redistribute bgp <AS>`（OSPF の下） | AS 番号 |
| `ip prefix-list <名前> <番号> permit\|deny any` | `any` の語（「0.0.0.0/0、上限 32」として入れ、書き出すときは `0.0.0.0/0 max 32` と書きます） |
| `ike proposal … authentication pre-shared …` | `authentication pre-shared`（事前共有鍵だけを扱うため） |
| LNS の PPP プロファイルの `lcp pfc`、`lcp acfc`、`ipcp ip-compression` | 行そのもの（入れる列がありません。FD 2.36 が「設定例と同じ設定に」とする 3 行で、書き出しではどの LNS のプロファイルにも書きます。PPPoE のプロファイルにあるときは「未対応」です） |
| `tunnel mode l2tp ipsec` | 古い表示の形（Ver9.6 まで）。書き出しは `tunnel mode l2tp-lns ipsec` です |
| `ikev2 peer … authentication psk …`、`ike policy … key-type char` | `authentication psk`・`key-type char`（事前共有鍵・文字の鍵は既定で、ほかの値のときだけ意味を持ちます） |
| `ikev2 profile <名前>`、`ikev2 binding <名前>` | プロファイルの名前（表に列が無く、書き出しではパックが決めます。既定のプロファイルに書かれていた値は、書き出しではトンネルごとのプロファイルに書きます） |
| `ikev2 ipsec-mode tunnel`・`ipsec-mode tunnel` | 既定の値なので、書き出しでは書きません |
| `username <名前> password plain <モード> <値>` | モード（`0` = 旧方式・`1` = 新方式のハッシュ化。機器はどちらでも値をハッシュにして保存します。書き出しでは付けません） |
| LAC の配下の PPPoE サーバの `ppp binding <プロファイル>` | プロファイルの名前（書き出しでは `<L2TP の名前>-pppoe` にします） |
| PPP プロファイルを結んだ IF の `auto-connect`、`encapsulation pppoe` | 行そのもの（PPPoE の IF が自分ですること・サブ IF の既定です。PPPoE のクライアントでは書かず、PPPoE サーバでは資料の設定例どおりに書きます） |
| `ipsec dynamic-map … ike <ポリシー>`（LNS の受け口・方式 policy のマップ） | `ike <ポリシー>`（IKE のポリシーを指す語。書き出しでは、トンネル IF 型の行にだけ `ike` を書き、方式 policy の行はメーカーの設定例どおり書きません。「この IKE の相手だけに使う」が有効なら `ike-binding` を書きます） |
| `ipsec policy tunnel <マップ> out` | `out`（省いたときと同じ向き。書き出しでは付けます） |
| マップのリストの名前、`ip access-list <名前> <番号> …` の名前 | リストの名前（表に列が無く、書き出しでは `sec-<数>`・`sec-p<数>` にします） |
| `local-ts ipv4 protocol 6｜17 …` | 番号で書いたプロトコル（書き出しでは `tcp`・`udp` と書きます） |

### 「無視」にする行

`ignore.yaml` にあるのは、画面の表示だけの行です（`Current configuration : … bytes` など 3 種類）。インタフェースの下の `service-policy enable` は、書き出しのときにパックが書く行なので「無視（パックが書く）」にします（別紙 A 3.18）。`ipv6 dhcp enable` も同じ扱いです（「IPv6」の節）。`!` で始まる行は区切りとして数えます。

## コンフィグの書き出し（共通の決まり）

対応一覧の表を書き出します。書き出した文面は、同じパックで取り込み直して設計データと比べます（往復の確認）。分野ごとの書き方は、それぞれの節の「書き出し」にあります。書き出しの見本は `tests/render` にあります（名前に `_adapt`・`_failed`・`_unused` の付いたフォルダは、置き換え・止まる場合・使われなかった値の見本です）。

節の順（`render.yaml`）: `system`（ホスト名・時刻帯・機器内のログ・ユーザ・DNS）→ `policy`（アクセスリスト・プレフィックスリスト。工場出荷時のコンフィグで、アクセスリストが先頭の近くにあるため）→ `qos`（クラスが名前で指すアクセスリストの後ろ）→ `route_policy`（ルートマップ。名前で指すリストの後ろ、経路の前）→ `routing` → `ospf` → `bgp` → `ipv6`（IPv6 の経路・リスト・OSPFv3・RA のプレフィックス・DHCPv6）→ `tracking`（回線監視。止める経路が先にあるように、経路の後ろ）→ `ipsec` → `extra`（追加コンフィグの既定の位置）→ `services`（SNMP・NTP・syslog）→ `remote`（Telnet / SSH サーバ・無操作タイムアウト）→ `pppoe`（PPP プロファイル）→ `l2tp`（LNS の PPP プロファイル・LAC の URL リスト）→ `dhcp`（DHCP サーバ）→ `interfaces`（`device` のブロック → `interface` のブロック）。終わりの印はありません。

- 経路が先で、`device` と `interface` のブロックが最後なのは、公式の例の並びに合わせたものです。ユーザやサービスの位置は資料で確かめられていません
- 追加コンフィグの既定の位置を `services` より前にしたのは、そこに書いたアクセスリストが、それを名前で指す行（SNMP コミュニティ・Telnet / SSH サーバ・`ip filter`）より先に来るようにするためです
- 表の行は、キーの自然な順（`GigaEthernet…` は `Loopback…` より前）に書きます

この機種に当たる行が無い（または確かめられていない）ため、書けない列があります。値が入っていると「使われなかった値」（G-602、エラー）になります。分野ごとの節に「書けない列」があるものは、そちらも見てください。

| 表 | 書けない列 | 理由 |
|---|---|---|
| devices | ドメイン名、特権パスワードとその形、バナー | 機器のドメイン名と特権パスワードに当たるコマンドがありません。バナーは保存した文面での形を確かめていません |
| interfaces | MTU、NAT の向き | `mtu` に当たるコマンドが見つかっていません（`ip mtu` は書けます）。NAT は向きを書く形ではありません（別メーカーのデータの向きは、条件が合えば外して知らせます。「アクセスリスト・プレフィックスリスト・NAT」の節） |
| static_routes | 説明 | 経路に名前は付けられません（回線監視は、`watch-group` の側に書きます） |
| ospf_processes | VRF、基準帯域 | プロセスを VRF に結ぶ行、基準帯域の行がありません |
| ospf_areas | 既定経路のコスト | 取り込みの規則がありません |
| bgp_neighbors | 受信経路の保持 | 当たる行を確かめていません |
| fhrp | 版、切り戻し待ち、Hold | VRRP の行にありません |
| tracks | しきい値、待ち時間、断・復旧の反映待ち | 当たる行がありません |
| remote_access | SSH の版、認証、回線パスワード | 当たる行を確かめていません |
| ntp_servers | 優先 | `prefer` が無く、`priority`（数）です |

取り込みとの関係:

- 平文の `username … password plain …` は機器の画面には出ませんが、書き出した文面を読み直すために取り込みの規則があります
- `ntp source` と `syslog facility`、`snmp-agent ip trap-source` は、取り込みが各行に配り直します。書き出しは、各行の値がそろっていることを確かめて 1 行にまとめます
- `ip vrf forwarding` を読むと vrfs の表に名前だけの行ができます。設計データの vrfs にも同じ名前の行を書いておくと、往復の確認が合います

別メーカーのデータをこの機種の書き方に置き換える決まり（`adapt.yaml`）は、置き換えたことを G-614 の警告で知らせます。分野ごとの節の「置き換え」にあります。

## 機器の基本

機器全体・ユーザ・リモート接続・NTP・syslog・SNMP です。見本は `tests/import/basic`・`system` と、`tests/render` の `basic`・`system`・`system_adapt`・`system_failed` です。

### 読み方の決まり

- **時刻の帯。** `timezone +09 00` は `+9` として入れます（この機種は帯の名前を持ちません）
- **ログ転送の送信元は、アドレスから IF の名前に直します。** `syslog ip source <アドレス>` のアドレスを持つ IF が無いと、その行は「未対応」に戻ります。VRF ごとに 1 つです
- **`logging subsystem all <重要度>` は機器の「ログに残す重要度」、ほかの対象は 1 行ずつ「ログの対象ごとの重要度」の表に入ります。** メッセージの番号を選ぶ形（`include`・`exclude`）は「未対応」のままです
- **`terminal timeout` は、遠隔ログインの「無操作切断」に入ります。** コマンドリファレンスの説明は「コンソールのログインタイマ」で、機器に 1 つの値です。別メーカーの「コンソールの無操作切断」は、これと合わせて 1 つにします（下の置き換え）
- **DNS サーバは、全部が同じ VRF（か、全部が VRF なし）です。** 先に読んだ行と VRF が違う行は「未対応」になります
- **遠隔ログインのリスト**は、Telnet と SSH で同じなら（または、リストを書いた側のサーバだけが有効なら）「接続元の制限」、違えば方式ごとの列に入ります

### 書き出し

| 項目 | 書き方 |
|---|---|
| 時刻帯 | `+9` → `timezone +09 00`、`+5:30` → `timezone +05 30`。帯の名前は持てないので、`JST +9` のような名前付きの形はエラーにします（`+9` と書いてください） |
| 機器内のログ・機能の入り切り | 節 `system` に `logging buffered`・`logging subsystem`・`logging timestamp`・`service password-encryption`（時刻帯のすぐ後ろ。工場出荷時のコンフィグで、ログの行が先頭の近くにあるため）と `ip ufs-cache enable`・`ipv6 ufs-cache enable`、節 `services` に `ntp ip enable` を書きます。「機器内にログを溜める」「転送のキャッシュ」「NTP サーバとして動く」の「無効」は書けません（止まります。止める行が保存した文面に出るかを確かめていません。セルを空にしてください） |
| ハッシュ化された秘密（ユーザ） | `username <名前> password hash <値>` と書きます。`$` で始まるハッシュは別メーカーの形なので、エラーにします（ハッシュは別メーカーへ引き継げません）。この機種のハッシュの形は資料に無いため、それ以外の値が本当にこの機種のハッシュかどうかは見分けられません |
| 平文の秘密（ユーザ） | `username <名前> password plain <値>` と書きます。形が空のセルは平文として扱います。空白を含む値はエラーにします |
| ユーザの権限 | `admin` → `administrator`、`operator` → `operator`、`monitor` → `monitor`。空なら語を付けません。数（`15` など）は、語の対応がないエラー（G-603）になります |
| リモート接続 | プロトコルごとに `telnet-server ip enable` / `ssh-server ip enable`。「接続元の制限」があれば両方のサーバに同じ名前を、方式ごとの列があればそのサーバにだけ書きます（両方に値があると止まります）。無操作タイムアウトは `terminal timeout <分>`（機器に 1 つ） |
| DNS サーバ | `ip name-server [vrf <VRF>] <アドレス>`、IPv6 のアドレスは `ipv6 name-server <アドレス>`。VRF だけがあってサーバが無いと止まります。IPv6 のサーバに VRF は付けられません（止まります） |
| NTP | サーバは IPv4 か IPv6 のアドレスです（名前はエラー。IPv6 のサーバに VRF は付けられません）。送信元 IF が全部の行で同じなら `ntp source`、違えばサーバごとの `source` で書きます |
| syslog | 転送先は IPv4 か IPv6 のアドレスです（IPv6 は `syslog ipv6 host <アドレス>`。VRF は付けられません）。ファシリティは機器に 1 つなので、全部の行で同じ値でなければエラーにします。送信元 IF は、その IF のアドレスで書きます（`syslog [vrf <VRF>] ip source <アドレス>`。VRF ごとに 1 つ。IF にアドレスが無いと止まります。IPv6 の転送先は `syslog ipv6 source <その IF の固定の IPv6 アドレス（1 つ目）>`） |
| SNMP | コミュニティか通知先があれば `snmp-agent ip enable` を書きます。コミュニティはアクセスリスト → `ro` / `rw` の順。通知先の版は `1` と `2c`（`version 2` と書きます）だけで、版 3 はエラーです。通知先のコミュニティは必須で、SNMP コミュニティの表に無い名前はエラーにします（先に定義が要るため。CRM p.22-15）。通知の送信元 IF は機器に 1 つです（そろっていなければエラー）。連絡先は空白なしの 1 語です。IPv6 の通知先があれば、`snmp-agent ipv6 enable`、その通知先が使うコミュニティの `snmp-agent ipv6 community <名前> [ro\|rw]`、`snmp-agent ipv6 host …`、`snmp-agent ipv6 trap-source <IF>` も書きます（IPv6 の SNMP は IPv4 と別に設定が要ります。CRM p.22-17〜22-19）。そのコミュニティに「接続元の制限」（IPv4 の ACL）があれば止まります |

- **書けないもの**（理由を付けて止まります。G-601）: 別メーカーの機器が暗号化した形の秘密（OSPF の鍵・BGP のパスワード・PPPoE の接続パスワードなど。L2TP のトンネルのパスワードと IPsec の鍵の `secret <値>` だけは書けます）と、形が encrypted のユーザ
- **書けない列**（値があれば G-602）: バナーの 3 つ（保存した文面での形を確かめていません）と、上の「書けない列」の表の devices・remote_access・ntp_servers の列

### 置き換え（adapt.yaml。G-614 で知らせます）

- 「ログの時刻に足すもの」「デバッグの時刻の付け方」は、NEC IX に無いので書きません
- ログ転送先の「重要度」は、機器の「ログに残す重要度」に置き換えます（全部の転送先が同じ値で、機器の値が空のとき。`logging subsystem all` で、機器内のログも同じ重要度になります）
- 無操作切断の秒の端数は、分に切り上げます
- 「コンソールの無操作切断」は、遠隔ログインの「無操作切断」が空ならそこへ入れ、あれば遠隔ログインの値を使います
- ユーザの「元に戻せる形で持つ」は書きません（NEC IX のユーザはいつもハッシュです）

## インタフェースとトンネル

インタフェース・VRF・GRE と EtherIP のトンネルです。IPsec のトンネルは「IPsec」の節、L2TP のトンネルは「L2TP」の節、IPv6 の網のトンネル（MAP-E・DS-Lite など）は「IPv6」の節です。見本は `tests/import/basic`・`etherip` と、`tests/render` の `basic`・`dhcp_tunnel`・`etherip`・`etherip_failed` です。

### 読み方の決まり

- **VRF は定義のブロックがありません。** `ip vrf forwarding <名前>` が出てきた時点で、vrfs の表に名前だけの行を作ります
- **ポートの説明**（`device` の下の `description`）は、`<ポート名>.0` の行の「ポートの説明」に入ります（インタフェースの説明とは別です）
- **`ip tcp adjust-mss auto`** は、「TCP MSS 調整」に `auto` と入ります
- **EtherIP は、IPv4 で運ぶ形（`tunnel mode ether-ip ip`）のほかに、IPv6 で運ぶ形（`ether-ip ipv6`）と IPsec で守る形（`ether-ip ipsec`・`ether-ip ipsec-ikev2`）も読みます。** IPsec で守る形の決まりは「IPsec」の節です

### 書き出し

| 項目 | 書き方 |
|---|---|
| インタフェースの有効・無効 | GigaEthernet（基本・サブ）と Tunnel には、有効なら `no shutdown`、無効なら `shutdown` を必ず書きます。Loopback と Null には、どちらも書きません（「無効」はエラー）。BVI にも、有効なら `no shutdown`、無効なら `shutdown` を書きます（FD の例に `no shutdown` があります）。そのほかの種類（USB-Serial など）は、行の出方を確かめていないのでエラーにします |
| アドレス | `ip address <アドレス>/<長さ>`、`dhcp` → `ip address dhcp`、`negotiated` → `ip address ipcp`、`unnumbered:<IF>` → `ip unnumbered <IF>`。アドレスの無い IF に `no ip address` は書きません |
| VRF | `ip vrf forwarding` を、アドレスより前に書きます。この機種は VRF の定義のブロックを持たず、この行が VRF を作ります。そのため vrfs の表の「名前」はこの行で書いたものとして扱い、どのインタフェースにも使われていない VRF はエラーにします |
| 速度・二重・ポートの説明 | ポートの設定なので、`device <ポート名>` のブロックに書きます（説明はブロックの先頭。79 字まで）。値は基本インタフェース（`GigaEthernet0.0` のように `.0` で終わる行）に書いてください。速度・二重がサブインタフェースやほかの種類の行にあるとエラーにします |
| VLAN のタグ | `encapsulation dot1q <番号> [tpid <16 進数>]`。サブインタフェース（`.1` 以降）以外にあるとエラーにします。くわしくは「VLAN とスイッチ」の節 |
| 帯域・MTU・MSS | `bandwidth <kbps>`、`ip mtu`、`ip tcp adjust-mss <数>\|auto` |
| DHCP リレー | 転送先が 1 つでもあれば、機器全体の `ip dhcp-relay enable` を書きます |
| 代理 ARP | 「有効」は `ip proxy-arp`。「無効」は書けません（既定と、止める行の出方を確かめていないため、エラーにします。セルを空にしてください） |
| トンネル | トンネル IF の下に `tunnel mode gre｜ipsec`。gre は `tunnel source`・`tunnel destination`（必須）・`tunnel key`・`tunnel keepalive period <間隔> retries <回数>`（2 つは組）。種類が ipsec の行は、IPsec の表に、そのトンネル IF の行が要ります（書き方は「IPsec」の節）。種類 ipip・l2tpv3 は語の対応がないエラー（G-603） |
| EtherIP | `tunnel mode ether-ip ip` → `tunnel source` → `tunnel destination` → `bridge-group`。トンネル IF に「ブリッジグループ」が無いと止まります。「宛先（IPv6）」のあるトンネルは、`tunnel mode ether-ip ipv6`・`tunnel source`・`tunnel destination <IPv6>` |

### 置き換え（adapt.yaml。G-614 で知らせます）

- EtherIP のトンネルがあるのに「ブリッジと L3 の併用」が有効でないときは、有効にして `bridge irb enable` を書きます

## VLAN とスイッチ

タグ VLAN（サブ IF）、内蔵スイッチ（スイッチングハブ）のポート VLAN・ポートの設定・リンクアグリゲーション・ポートミラー、ブリッジ（BVI）、ループ検出を読み書きします。NEC のスイッチ（別シリーズ）は、このパックの対象ではありません（別のメーカーパックになります）。内蔵スイッチ付きの小型ルータ（Cisco など）からこの機種へ書き換えるときの決まりも、この節にあります。見本は `tests/import/switch`・`switch_more` と、`tests/render` の `switch`・`switch_adapt`・`switch_failed`・`switch_model_failed`・`switch_more`・`switch_more_failed` です。

### 内蔵スイッチのポートの名前

内蔵スイッチのポートは、この機種では IF ではありません（`device` の下の `port <番号>` で指します）。表では 1 行ずつ持つので、パックが名前を付けます: **`<デバイス名>:port<番号>`**（`device GigaEthernet2` の `port 1` は `GigaEthernet2:port1`）。機器が使う名前ではないので、コンフィグには出ません。別メーカーから書き換えるときは、IF 対応の「新 IF 名」に、旧機器のスイッチポートごとにこの名前を書きます。

### 読める行と入る表

| 分野 | 行 | 入る表と列 |
|---|---|---|
| タグ VLAN（サブ IF） | `encapsulation dot1q <番号> [tpid <16 進数>]`（CRM p.3-4。サブ IF だけ。`GigaEthernet2:1.1` のような VLAN グループのサブ IF にも書けます） | interfaces の「VLAN 番号（サブ IF）」「タグの種別（TPID）」 |
| タグ付きフレーム | IF の下の `dot1q min-frame-size <64〜1522>`（CRM p.3-10） | interfaces の「タグ付きフレームの最小の長さ」 |
| ポート VLAN | `device` の下の `vlan-group <グループ> port <ポート> …`（CRM p.3-14）。ポートごとに、モード access・アクセス VLAN = グループの番号の行になります。グループの IF `<デバイス名>:<グループ>.0` の「VLAN 番号（VLAN の IF）」にも、グループの番号が入ります | switchports、interfaces |
| グループに入れないポート | 名前の出たポートが、どの VLAN グループにも入っていなければ、モード routed の行になります（デバイスの基本 IF `<デバイス名>.0` に属するポート） | switchports |
| ポートの設定 | `device` の下の `port <番号> speed 10\|100\|1000\|auto`、`port <番号> duplex full\|half\|auto`、`port <番号> shutdown`（CRM p.3-11〜3-13）、`port <番号> mdi-mdix mdi\|mdix`、`port <番号> qos default-priority <0〜7>`（CRM p.3-12） | interfaces（ポートの行）の「速度」「二重」「有効」「MDI の向き」「タグなしフレームの既定の優先度」 |
| デバイス全体 | `mdi-mdix mdi\|mdix`（CRM p.3-11）、`qos scheduler strict\|wfq`・`router-port qos default-priority <0〜7>`（CRM p.3-12〜3-13）、`speed 2500\|5000`（CRM p.3-14） | interfaces（デバイスの基本 IF `<デバイス名>.0` の行）の「MDI の向き」「内蔵スイッチの送信の順序」「ルータ側ポートの既定の優先度」「速度」 |
| LAG | `device` の下の `port <番号> link-aggregation <LAG の番号>`（CRM p.3-11）。束ね方は on（固定）です。この機種に LACP はありません（FD p.2-20） | lags |
| ポートミラー | `port <写す元のポート> mirror-port <写す先のポート> both\|out`（CRM p.3-12）。both は「写す元（両方向）」、out は「写す元（送信）」に入ります。番号は、写す先のポートの番号 | mirror_sessions |
| ブリッジ | `bridge irb enable`（CRM p.6-6）、IF の下の `bridge-group <番号>`（CRM p.6-5。Ethernet・Tunnel・BVI）。BVI は、自分の `bridge-group` の行でグループとつながります（BVI の番号とグループの番号は別） | l2_settings の「ブリッジと L3 の併用」、interfaces の「ブリッジグループ」 |
| ブリッジグループ | `bridge <番号> aging-time <秒>`、`bridge <番号> table-size <数>\|unlimited`、`[no] bridge <番号> bridge ip\|ipv6`、`bridge <番号> bridge-only pppoe`（CRM p.6-3、6-6） | bridge_groups |
| ブリッジに入れた IF | `bridge-group <番号> port-protected`、`bridge-group <番号> port-table-size <数>`（CRM p.6-6） | interfaces の「ブリッジの保護ポート」「このポートで覚える MAC の数」 |
| ループ検出 | IF の下の `loop-detection enable`、`loop-detection action shutdown\|none [restore-time <秒>]`、全体の `loop-detection interval-time`・`threshold`・`window-size`（CRM p.20-3〜20-4） | interfaces、l2_settings |
| IF の数 | `system interfaces bvi <数>`（CRM p.2-7）、`system subinterfaces <デバイス名> <数>`（CRM p.2-8） | l2_settings の「BVI の数」、interfaces（基本 IF の行）の「サブ IF の数の上限」 |

- デバイス全体の行（`mdi-mdix`・`qos scheduler`・`router-port …`）は、デバイスの速度と同じく、基本 IF `<デバイス名>.0` の行に入ります。その IF のブロックが設定に無いと、行は「未対応」になります（機器は、設定していない基本 IF も見せます。FD p.2-6）
- この機種のデータ（取り込んだコンフィグ）には、下の置き換えは何もしません（グループの IF の「VLAN 番号（VLAN の IF）」が、グループの番号と同じため）
- 別メーカー（Cisco）へ書き換えると、MDI の向き・優先度・送信の順序・ループ検出・保護ポートなどこの機種だけの列は、G-602 で知らせます（Cisco の側に当たる行を確かめていません）。ブリッジグループの `bridge ip`・エージングも同じです

### 書き出しで決めたこと

| 項目 | 書き方 |
|---|---|
| `device` のブロック | デバイスごとに 1 つ。デバイス自身の行（説明・`speed`・`duplex`・`mdi-mdix`・`qos scheduler`・`router-port …`）→ `vlan-group`（グループの番号の順。ポートは番号の順に空白で区切ります）→ ポートごとの `port <番号> speed`・`duplex`・`mdi-mdix`・`qos default-priority`・`mirror-port`・`shutdown`・`link-aggregation` の順です。書くものが無いデバイスのブロックは書きません。ブロックの中の行の順は、資料では決まっていません（例によって違います） |
| ポートの行 | ポート（`…:port<番号>`）の行は、`interface` のブロックにはしません。説明は書けません（下の置き換えで外して知らせます）。アドレスなど、ほかの列に値があれば G-602 で知らせます。速度は 10・100・1000・auto だけです（ほかは止まります） |
| アクセス VLAN | この機種のデータでは、VLAN グループの番号です（その番号のグループの IF `<デバイス名>:<番号>.0` がインタフェースの表にあるとき）。グループの IF が無い番号は、別メーカーの VLAN の番号として扱い、下の置き換えで VLAN グループにまとめます |
| 書けないスイッチポート | IF 名がポートの名前の形でない行、モードが trunk・dynamic の行、束ね方が on 以外の LAG は、理由と直し方を付けてエラーにします |
| タグなしの VLAN | サブ IF に「タグなしの VLAN」は書けません（この機種では、タグなしの通信は基本 IF `ポート名.0` が受けます。FD p.2-66）。IF 対応で、そのサブ IF の新 IF 名を `ポート名.0` にすると、下の置き換えで基本 IF として書きます |
| ブリッジ | 「ブリッジと L3 の併用」が有効なら `bridge irb enable` を、インタフェースの前に書きます。BVI に「ブリッジグループ」があるのに併用が無効なら、エラーにします。「保護ポート」「覚える MAC の数」は、ブリッジグループのある IF にだけ書けます |
| ポートミラー | `port <写し取るポート> mirror-port <出力先のポート> both\|out`。写し取る IF と出力先が同じ内蔵スイッチのポートでない行、受信だけを写す行（「写す元（受信）」）は、エラーにします |
| 「無効」と書けない列 | 「ブリッジと L3 の併用」「PPPoE だけをブリッジする」「ブリッジの保護ポート」「ループ検出」の「無効」は書けません（無効は、行が無いのと同じです。セルを空にしてください）。「サブ IF の数の上限」は基本 IF の行にだけ書けます |
| IF の並び | IF 名の順です（置き換えで足した行も、同じ順に入ります） |

### 別メーカーの VLAN・スイッチポートの置き換え（adapt.yaml。G-614 で知らせます）

別メーカー（Cisco など）は、ポートごとに VLAN の番号を書き、VLAN ごとに IF（`Vlan10`）を持ちます。この機種は、ポートを VLAN グループに入れ、グループをまたぐ VLAN はブリッジグループでつなぎます。パックは、パラメータブックの値（VLAN の番号・トランク）をそのままにして、書き出しのたびに次のように置き換えます。**人が決めるのは、IF 対応の新 IF 名だけです。**

| 旧機器の設定 | IF 対応で決めること | パックの書き方 |
|---|---|---|
| スイッチポート | 新 IF 名を `<デバイス名>:port<番号>` にします | — |
| VLAN の IF（トランクに通さない VLAN） | 新 IF 名を、ポート VLAN の IF `<デバイス名>:<グループ>.0` にします（グループは 1〜ポートの数で、好きな番号） | その VLAN のアクセスポートを、そのグループの `vlan-group` に入れます。アドレスはグループの IF に付きます |
| VLAN の IF（トランクに通す VLAN、または複数のグループにまたがらせたい VLAN） | 新 IF 名を BVI（`BVI0` など）にします | BVI の「ブリッジグループ」を VLAN の番号にし、`bridge irb enable` を書きます。その VLAN のアクセスポートは、1 つの VLAN グループ（番号は、いちばん若いポートの番号）にまとめ、グループの IF をブリッジグループに入れます |
| トランクのポート | — | ポート 1 つの VLAN グループ（番号はポートの番号）にし、通す VLAN ごとにタグ付きのサブ IF（`<デバイス名>:<グループ>.1` 〜）を作って、その VLAN のブリッジグループに入れます。タグなしの通信（ネイティブ VLAN。空なら 1）は、グループの基本 IF（`.0`）で受け、その VLAN がこの機器にあればブリッジグループに入れます |
| スイッチポートの行が無いポート、アクセス VLAN が空のアクセスポート | — | VLAN 1 のアクセスポートとして書きます（元の機器の既定）。行が無いポートは、VLAN 1 の IF がデータにあるときだけです |
| IF の無い VLAN（L2 だけの VLAN） | — | BVI にする VLAN と同じ形（BVI なしのブリッジグループ）で書きます |

つまり、旧機器の `Vlan<n>` は、**その VLAN がトランクを通らなければポート VLAN の IF、通るなら BVI** にします（下の「機種の定義」を使うと、この案をツールが作ります）。

- トランクが通せるのは、**この機器にアクセスポートか BVI がある VLAN** と、**同じ内蔵スイッチの 2 つ以上のトランクが通す VLAN**（通り抜けるだけの VLAN）です。通り抜ける VLAN は、トランクごとのタグ付きのサブ IF を 1 つのブリッジグループ（番号は VLAN の番号）に入れて書きます（CRM p.6-5「入出力インタフェースでタグが等しい場合は透過されます」、FD の 2.9.6 節のタグ VLAN の透過ブリッジ）。許可 VLAN が空・all のトランクは、「VLAN」の表にある VLAN と VLAN 1 を通すものとして数えます。1 つのトランクしか通さず、この機器にポートも BVI も無い VLAN は、行き先が無いので書かず、知らせます
- パックがブリッジグループを作ったとき（アクセスポートをまとめたグループ、トランクのタグ付きのサブ IF）は、BVI が無くても `bridge irb enable` を書いて知らせます（この行が無いとブリッジが働きません。CRM p.6-6）
- ポート VLAN の IF にした VLAN は、トランクに通せません。その VLAN を許可している（許可 VLAN が空・all の場合も含む）トランクは置き換えず、エラーにして「VLAN の IF を BVI に」と知らせます
- パックが作る VLAN グループの IF（`<デバイス名>:<番号>.0`）が、IF 対応で決めた IF とぶつかるときは、エラー（G-601）にして、別の番号にするよう知らせます
- サブ IF は、VLAN グループ 1 つに 8 つまでです（FD の上限の表、FAQ）。9 つ以上になるトランク（通す VLAN が 9 つ以上）は、エラー（G-601）にして、通す VLAN を減らすよう知らせます。`system subinterfaces` は、資料では一部の機種だけで効くコマンドで、VLAN グループの上限を増やせるとは書かれていないので、数え方には使いません
- 機種が分かるとき（機器の「機種」が下の「機種の定義」の ID のとき）は、内蔵スイッチのポートの番号と、VLAN グループの番号（1〜ポートの数。CRM p.3-14）を確かめ、合わなければエラー（G-601）にします
- **この機種に無い設定は、外して知らせます**: スパニングツリー（ポートの「すぐ転送にする」「BPDU ガード」、ガード・BPDU フィルタ・コスト・ポート優先度・リンクの種類・ループガード・MST、全体設定、VLAN ごとの優先度。この機種はスパニングツリーに対応していません。FD p.2-78）、VLAN の定義と名前（定義する行がありません）、ポートの説明、トランクのタグ方式・自動交渉の設定、UDLD、errdisable recovery、LAG の振り分け方・LACP のシステム優先度、CDP・LLDP（CRM に `lldp`・`cdp` のコマンドはありません）
- **知らせずに外すもの**: もとから無効の CDP・LLDP（`no cdp enable`・`no cdp run`）と、内部 VLAN の振り方（この機種の動きと同じで、人が確かめることが無いため）
- **外さずに G-602 で知らせるもの**: ポートセキュリティ・ストーム制御・DHCP スヌーピング・ARP 検査・プライベート VLAN・二重タグ・VTP（通信を絞る設定が黙って消えないように）
- 別メーカーのポートミラーは、上の `port … mirror-port …` の形に置き換えます。セッション番号は出力先のポートの番号にします（知らせません）
- ブリッジでつなぐ通信はソフトウェアで転送します。同じ VLAN グループの中のポートどうしの転送とは、速さが違います

### 機種の定義（`models.yaml`。別紙 B 6 章）

「機器対応」の「新機種」に下の ID を書くと、IF 対応の新 IF 名の案を、機種のポートの並びから作ります。ID は製品の名前ではなく、ポートの並びを言う名前です（数字はポートの数）。

| ID | ポート | 根拠 |
|---|---|---|
| `ix-1ge-4sw` | L3 のポート `GigaEthernet0.0`、内蔵スイッチ `GigaEthernet1:port1`〜`4` | UM（5 ポートのうち 4 ポートが SW-HUB）、FAQ の設定例 |
| `ix-2ge-8sw` | L3 のポート `GigaEthernet0.0`・`GigaEthernet1.0`、内蔵スイッチ `GigaEthernet2:port1`〜`8` | UM（10 ポートのうち 8 ポートが SW-HUB） |
| `ix-4ge` | L3 のポート `GigaEthernet0.0`〜`GigaEthernet3.0`（内蔵スイッチなし） | UM |

案の作り方（別紙 B 6.3）:

- 旧機器のスイッチのポートは、名前の順に内蔵スイッチのポートへ。止めてあるポートは `-`（持っていかない）
- VLAN の IF は、その VLAN がトランクを通らなければ、ポート VLAN の IF `<デバイス名>:<いちばん若いアクセスポートの番号>.0`。トランクを通るなら BVI（`BVI0` から）
- サブ IF と PPPoE の論理 IF は、親のポートの `.1` から順に。ループバックとトンネルは `Loopback<番号>.0`・`Tunnel<番号>.0`
- 内蔵スイッチを 2 つ持つ形（上位の機種）は、スイッチ 1 つあたりのポートの数を資料で確かめられなかったので、定義していません

VLAN・内蔵スイッチ・ブリッジの「読まない行」と「確かめていないこと」は、後ろの同じ名前の節にまとめてあります。

## IPv6（別紙 A 3.23）

構文は、上の「参照した資料」の CRM（9 章 IPv6・10 章 ルーティング・12 章 トンネル・13 章 セキュリティ・23 章 高度な経路制御・24 章 アクセスリスト）で確かめました。**実機の出力では確かめていません。**

### 読める行と入る表

| 分野 | 読める行 | 入る表 |
|---|---|---|
| インタフェース | `ipv6 enable`、`ipv6 address <プレフィックス> eui-64`、`ipv6 address <アドレス>/<長さ> anycast`、`ipv6 address autoconfig [receive-default]`、`ipv6 mtu`、`ipv6 filter <リスト> <番号> in\|out`、`ipv6 nd proxy <LAN 側 IF> [<リスト>]`、`ipv6 dhcp client <プロファイル>`、`ipv6 dhcp server <プロファイル>` | ipv6_interfaces（固定の `ipv6 address <アドレス>/<長さ>` は interfaces） |
| RA の送信 | `ipv6 nd ra enable`、`no ipv6 nd ra enable`、`ipv6 nd ra max-interval` / `min-interval` / `lifetime`、`ipv6 nd ra managed-config-flag` / `other-config-flag`、`ipv6 nd ra dns-server <アドレス> [<アドレス>] [<秒>]`、`ipv6 nd ra prefix-advertisement <名前>` と `ipv6 prefix <名前> <プレフィックス> [<有効> <推奨>] [on-link] [autonomous]` | ipv6_interfaces、ipv6_ra_prefixes |
| DHCPv6 クライアント | `ipv6 dhcp client-profile <名前>` の下の `option-request dns-servers\|domain-search-list\|ntp-servers`、`ia-pd subscriber <IF> [<下の部分>/<長さ>] [eui-64]`、`information-request`、`rapid-commit` | ipv6_interfaces（プロファイルを付けた WAN 側の IF の行と、subscriber の LAN 側の IF の行） |
| DHCPv6 サーバ | `ipv6 dhcp enable`、`ipv6 dhcp server-profile <名前>` の下の `dns-server <アドレス>\|dhcp` | ipv6_interfaces |
| DHCPv6 サーバ（プレフィックスの配布） | `ipv6 dhcp server-profile` の下の `ia-pd prefix <配る長さのプレフィックス> nla-length <ビット数> [life-time <有効> <推奨>]`、`ntp-server <アドレス>\|dhcp`。もらったプレフィックスの配り直し: クライアントのプロファイルの `ia-pd redistribute pool <プール名> <下の部分>/<配る長さ> [nla-length <ビット数>]` と、サーバのプロファイルの `ia-pd redistribute-prefix pool <プール名>` | ipv6_interfaces（dhcp_pd_pool・dhcp_pd_length・dhcp_pd_from・dhcp_pd_valid・dhcp_pd_preferred・dhcp_ntp） |
| スタティック経路 | `ipv6 route <宛先>/<長さ>\|default <次ホップ>[%<IF>]\|<IF> [ra\|dhcp] [distance <数>]` | ipv6_routes |
| フィルタ | `ipv6 access-list <名前> [<番号>] permit\|deny ip\|<番号> src … dest …`、`… tcp [established]\|udp src … [sport eq\|lt\|gt\|neq <数>\|range <数> <数>] dest … [dport …]`、`… icmp [type <数>\|<メッセージ名>] src … dest …`、末尾の `dscp <数>`・`fragments` | ipv6_acl_entries |
| プレフィックスリスト | `ipv6 prefix-list <名前> <番号> permit\|deny any\|<プレフィックス> [min <数>] [max <数>]` | ipv6_prefix_lists |
| OSPFv3 | `ipv6 router ospf <番号>` の下の `router-id`、`network <IF> area <エリア>`、`area <エリア> stub [no-summary]`、`area <エリア> default-cost`、`area <エリア> range <プレフィックス> advertise\|not-advertise`、`passive-interface <IF>`、`originate-default [metric <数>]`、`redistribute connected\|static\|bgp\|rip`。IF の `ipv6 ospf cost` / `priority` / `hello-interval` / `dead-interval` | ospfv3_processes、ospfv3_interfaces、ospfv3_areas |
| トンネル | `tunnel mode 4-over-6`（IPv6 の網で IPv4 を運ぶ。DS-Lite・固定 IP の IPIP）、`tunnel mode 6-over-4`、`tunnel mode map-e [jpix\|ocn\|ocn-fixed [hostname <名前>]]` と `ip address map-e`、`tunnel mode v6pv ds-lite\|ipip` と `ip address v6pv lan <IF>`、`tunnel destination <IPv6 アドレス>`、`tunnel destination fqdn <名前>` | tunnels |

### 読み方の決まり

- **値を入れない規則**: `ipv6 dhcp client-profile <名前>`・`ipv6 dhcp server-profile <名前>`（下の行を持つ入れ物の行）、OSPFv3 の下の `area <エリア>` だけの行（何も設定していないエリア。行が無いのと同じ）
- **「無視」にする行**: `ipv6 dhcp enable`（DHCPv6 サーバを起動する印。表に列が無く、DHCPv6 サーバを使う機器の書き出しでパックが書きます）
- **読んで、表には入れない語**: フィルタの `sport any`・`dport any`（何も絞らない）、`ipv6 filter` の番号（書き出しでは 1）、RA のプレフィックスの名前（書き出しでは `ra1`・`ra2`… とパックが付けます）、情報だけをもらう DHCPv6 のプロファイルの名前（書き出しでは `dhcpv6-<番号>`）
- **プレフィックスをもらう設定は、名前で 2 つの IF の行に分けて入れます。** プロファイルを付けた WAN 側の IF の行の「プレフィックスをもらう（DHCPv6-PD）」にプロファイルの名前、`ia-pd subscriber` が指す LAN 側の IF の行の「もらったプレフィックスから付ける」に同じ名前が入ります（Cisco の `ipv6 dhcp client pd <名前>` と `ipv6 address <名前> …` と同じ持ち方）
- `ipv6 filter` は、向きごとに最初の 1 つだけを読みます（IPv4 の `ip filter` と同じ）
- `ia-pd prefix <プレフィックス> nla-length <ビット数>` は、「配る長さ」= プレフィックスの長さ、「元」= 長さからビット数を引いたプレフィックスとして入れます（機能説明書 2.21.3 の図と設定例）。配り直しのプール名は表に持ちません（書き出しでは DHCPv6 サーバの名前を付けます）
- `ip address map-e` は「このトンネルは MAP-E」という意味だけを持つので、トンネルの種類として読みます。`ip address v6pv lan <IF>` は「LAN 側 IF（自動設定）」に入れます
- OSPFv3 の `originate-default` には `always` がありません（CRM）。表では when_present として持ちます（下の「確かめていないこと」）

### 読まない行（「未対応」として画面に出ます）

| 行 | 理由 |
|---|---|
| `ipv6 ufs-cache` の `enable` 以外の行（`hash`・`max-entries`・`timeout`）、`ipv6 hop-limit`、`ipv6 redirects`、`ipv6 nd ns-interval`、`ipv6 interface-identifier`、`ipv6 unnumbered`、`ipv6 autoselect …`、`ipv6 name-server <リンクローカル>%<IF>` など、上の表に無い `ipv6 …` の行 | 入れる列がありません |
| `ipv6 nd ra prefix-advertisement <名前> expire`、`ipv6 prefix` の `infinity`、`no ipv6 nd ra import-prefix`、`ipv6 nd ra linkmtu`・`cur-hoplimit`・`reachable-time`・`retrans-timer`・`domain-name`、`ipv6 nd local-proxy` | 入れる列がありません |
| `ipv6 route` に `metric`・`tag` を付けた形、`ipv6 route vrf …` | 入れる列がありません。VRF の中の IPv6 の経路の書き方を確かめていません |
| フィルタの行のうち、`src-domain`・`dest-domain`、ワイルドカード、`linkmgr`、`tc`、`precedence`、語の対応表に無いメッセージ名（TCP のフラグ 1 つと、ICMPv6 の `type <番号> code <番号>` は読みます）、`ipv6 access-list dynamic …`・`cache`・`option`・`sequence-mode` | 入れる列がありません |
| `ipv6 filter` に `suppress-logging` を付けた形 | 入れる列がありません（同じ向きの 2 つ目以降のフィルタは、「入力 IPv6 フィルタ（2 つ目以降）」などに入ります） |
| DHCPv6 クライアントの `ia-option`・`client-identifier`・`no auto-default`・`no reconfigure-accept`、サーバの `dns-server autoconfig`・`ia-pd static`・`ia-pd redistribute-static`・`no auto-static`、`ipv6 dhcp duid-list`・`ipv6 dhcp code` | 入れる列がありません（相手ごとに決めたプレフィックスを配る形は、相手の見分け方がメーカーで違うので表にしていません） |
| OSPFv3 の `redistribute ospf <番号> …`（ほかのプロセスの経路）、`originate-default` に `metric-type`・`route-map`・`tag` を付けた形、`distance`、`timers`、`ipv6 ospf neighbor`・`retransmit-interval`・`transmit-delay` | 入れる列がありません（`redistribute` にメトリック・メトリックの種類・タグ・ルートマップを付けた形は、「OSPFv3 再配布の指定」の表に入ります） |
| `tunnel mode 6-over-6`・`ether-ip`・`l2tp-…`・`mgre`・`ipsec-ikev2`、`tunnel v6pv url`・`tunnel v6pv user`、`tunnel outgoing-interface`、`tunnel adjust-mtu` | 入れる列がありません（L2TP と IKEv2 のトンネルは「L2TP」「IPsec」の節で読みます） |

| 同じ IPv6 のグループの 2 つ目以降のリンクローカルアドレス（`vrrp <番号> ipv6 fe80::…`）、IPv6 のグループの `vrrp <番号> authentication`、`vrrp <番号> timers delay`、IPv4 のグループの `timers advertisement csec` | 入れる列がありません（表の仮想リンクローカルアドレスは 1 つです。版 3 に認証は無く、機器でも働きません（FD 2.27）。切り戻りタイマは IPv4 と同じく読みません） |
| `syslog ipv6 host` に `port`・`match`・`unmatch` を付けた形、リンクローカルアドレスの宛先（`fe80::1%<IF>`）、`snmp-agent ipv6 community` に `view`・リストを付けた形、`snmpv3 ipv6 host` | 入れる列がありません（IPv6 の SNMP のリストは IPv6 のアクセスリストで、表の「接続元の制限」は IPv4 の ACL です） |

### 書き出し

- 経路・フィルタ・プレフィックスリスト・OSPFv3・RA のプレフィックス・DHCPv6 のプロファイルは BGP の後ろ（`ipv6.j2`）、IF の中の行は `ipv6_if.j2` に書きます
- フィルタは IPv4 と同じ決まりです: 行の番号は書かず、順番は 10・20・30… でなければ止めます。名前は 15 字まで。注記（remark）と log は書けません。プロトコルは ip・tcp・udp・icmp か番号です（esp は 50、ah は 51 と書きます）
- プレフィックスリスト・DHCPv6 のプロファイルの名前は、小文字でなければ止めます（機器が小文字で保存するため）
- `::/0` で「長さの上限」が 128 のプレフィックスリストの行は、`any` と書きます
- DHCPv6 サーバを使う IF があれば、`ipv6 dhcp enable` を書きます
- **置き換えて知らせるもの（G-614）**: 機器の「IPv6 の転送」が有効でも、行は書きません（NEC IX には当たる行が無く、IPv6 を使う IF があれば転送します）
- DHCPv6 サーバが配るプレフィックスは、`ia-pd prefix <「元」の頭に配る長さを付けたプレフィックス> nla-length <配る長さ − 元の長さ>` と書きます。ビット数は 0〜32 なので、差が 32 を超える行は止まります。もらったプレフィックスを配り直す行には、有効時間・推奨時間を書けません
- **書けないもの（理由を付けて止めます。G-601）**: **BGP の IPv6（ネイバー・広告経路・再配布。NEC IX の BGP のアドレスファミリは IPv4 だけです）**、リンクローカルアドレスを手で決める形、「DHCPv6 でアドレスをもらう」、「RA を送る」の reply_only、3 つ以上の RA の DNS サーバ、「DHCPv6 で配るドメイン名」、同じ DHCPv6 サーバの名前を 2 つの IF に付けた形、OSPFv3 の「網の種類」・「既定を受動にする」・NSSA・デフォルト経路の広告の always、次ホップ（リンクローカルでない）と出力 IF の両方を書いた経路、VRF の中の IPv6 の経路、OSPFv3 の認証（この機器の OSPFv3 に認証はありません。CRM 10 章）、「端末へ配るアドレスのプレフィックス」（DHCPv6 サーバは、プレフィックスと DNS・NTP だけを配ります）
- **書かない列（値があれば G-602）**: 経路の「説明」（NEC IX の経路に名前はありません）

### 確かめていないこと

- 機器が `show running-config` に出す形と順番（プロファイルの中の行の順、`tunnel mode` と `ip address map-e` の順、`area <エリア>` だけの行が出るか）
- `tunnel mode v6pv ds-lite\|ipip` の語順: CRM の構文の行（`tunnel mode TUNNEL_MODE [DELIVERY_PROTOCOL] [hostname HOSTNAME] [PVMODE]`）から組み立てた形で、入力例は載っていません
- OSPFv3 の `originate-default`（`always` の無い形）が、「デフォルト経路があるときだけ広告する」動きかどうか（IPv4 の OSPF の `originate-default` と同じ意味として、when_present で持っています）
- `ipv6 address autoconfig` や `ipv6 dhcp client` だけの IF で `ipv6 enable` が要るか（表に値があるときだけ書きます）
- DHCPv6 クライアントの既定（`ia-option ia-pd` を書かなくてもプレフィックスをもらうか）: プロファイルに `ia-pd subscriber` を書く形で書き出しています
- MAP-E・DS-Lite のトンネルに要るほかの行（NAPT・MSS・DNS の設定）は、IPv4 の表（NAT・インタフェース）と追加コンフィグで書きます

## アクセスリスト・プレフィックスリスト・NAT

読める行は「対応一覧」の「リスト・NAT・回線監視」です（別紙 A 3.25）。見本は `tests/import/acl_nat`、`tests/render` の `acl_nat`・`acl_adapt`・`nat_role_adapt`・`policy`・`policy_failed` です。

### 読み方の決まり

- **アクセスリストの番号。** 番号の無い行は、行の順に 10、20、30 … と番号を付けます。**`sequence-mode` に間隔が無い行**は、機器の既定の 100 として入れます（CRM p.24-8）
- **プロトコルの番号**は、資料に名前が出てくる 47（GRE）と 50（ESP）を、設計データの語（`gre`・`esp`）に直します。ほかの番号は番号のままです
- **ICMP のメッセージの名前**は、語のまま「ICMP の種類」に入れます（Cisco も同じ語を使います）
- **同じ向きのフィルタ**は、番号の小さい順に並べて、1 つ目を「入力 ACL」（「出力 ACL」）、2 つ目以降を「入力 ACL（2 つ目以降）」（「出力 ACL（2 つ目以降）」）に入れます。機器は、番号の小さいリストから見て、どの行にも合わなければ次のリストを見ます。どのリストにも合わない通信は捨てます（FD 2.30）
- **`ip prefix-list … any`** は、「0.0.0.0/0、長さの上限 32」の行として入れます（CRM p.23-14 に「0.0.0.0/0 min 0 max 32 と同等」とあります。書き出すときは `0.0.0.0/0 max 32` の形です）
- **NAT の外側。** `ip napt …` と `ip nat …` は、その行が書かれたインタフェースを「外側」として入れます（「確かめていないこと」を参照）
- **NAPT の変換後のアドレス。** `ip napt address <アドレス>` がある IF の `ip napt enable` は、「そのアドレス 1 つへまとめる」行（種類 pat_pool。先頭と末尾が同じアドレス）になります。`ip napt inside list <リスト> outside <アドレス>` も、同じ形の行です。NEC IX のプールは長さを持たないので、「範囲のプレフィックス長」は空です
- **`ip napt service`** は、内側のポート（変換後のポート。`none` なら外側と同じ）と外側のポートを持つ static_port の行です。予約されたサービス名（`dns` は udp 53、`http` は tcp 80、`https` は tcp 443、`snmp` は udp 161、`ssh` は tcp 22、`telnet` は tcp 23。CRM p.8-38）だけの行は、その値で入れます
- **IPsec と QoS が使うリスト。** IPsec のマップのリストと、QoS のクラスが DSCP・precedence を見るためだけに使うリストは、アクセスリストの表から外します（「IPsec」「QoS」の節）

### 書き出し

| 項目 | 書き方 |
|---|---|
| アクセスリスト | 1 行に 1 エントリ（`ip access-list <名前> [<番号>] permit｜deny <プロトコル> [<TCP のフラグ>]｜[<ICMP の種類>] src … [sport …] dest … [dport …] [tos …] [precedence …] [dscp …] [fragments]`）。リストそのものの行はありません。プロトコルは ip・tcp・udp・icmp か番号（`gre` は 47、`esp` は 50 と書きます）、アドレスは `any` か アドレス/長さ（ホストは /32）、ポートは `80` → `eq 80`、`1000-2000` → `range 1000 2000`、`>1023` → `gt 1023`、`<1024` → `lt 1024`、`!80` → `neq 80` |
| アクセスリストの種類 | 機器の ACL は 1 種類なので、種類は extended だけです（空なら extended として扱います）。standard のリストは止まります。別メーカーから持ってきた標準 ACL は、種類を extended にし、各行にプロトコル ip と宛先 any を書いてください |
| アクセスリストの行の番号 | 「行の番号の刻み」があるリストは、`ip access-list <名前> sequence-mode <刻み>` を書いてから、行を番号付きで書きます。刻みが空のリストは、番号を書かずに上から順に書きます（番号は 10、20、30 … でなければなりません。評価の順は変わりません） |
| アクセスリストの注記・説明 | 注記（remark）の行は書けません（止まります）。説明の列も書けません（値があれば G-602） |
| 行のないアクセスリスト | 何も書きません（エントリを追加コンフィグに書くときのため）。名前が 15 字を超えるリストは止まります |
| フィルタ | 並びの順に `ip filter <リスト> 1 in\|out`、`ip filter <リスト> 2 in\|out` … と番号を振って書きます。「（2 つ目以降）」だけがあって 1 つ目が空の行は止まります |
| プレフィックスリスト | `ip prefix-list <名前> <番号> …`、下限・上限は `min`・`max`。名前に大文字があると止まります（機器が小文字で保存するため） |
| NAT | 外側 IF のブロックの中に書きます。外側 IF は必須です（空なら止まります）。`pat_interface` → `ip napt enable`（対象の通信があれば `ip napt inside list <ACL>`）、先頭と末尾が同じ `pat_pool` → `ip napt enable` と `ip napt address <アドレス>`（2 つ目以降は `ip napt inside list <ACL> outside <アドレス>`）、`static_port` → ポートを変えない転送は `ip napt static <内側アドレス> <プロトコル> <ポート>`、ポートを変える転送（内側ポート ≠ 外側ポート）は `ip napt service <NAT の行の名前> …`、`static` → `ip nat enable` と `ip nat static <内側> <外側>`、`static_network` → `ip nat static network <内側>/<長さ> <外側>/<長さ>`、`dynamic_pool` → `ip nat pool <NAT の行の名前> <先頭> <末尾>` と IF の下の `ip nat dynamic list <ACL> pool <NAT の行の名前>` |
| ポート転送の外側アドレス | 空なら `interface`（外側 IF のアドレス）として扱います（defaults.yaml）。`interface` 以外は書けません |

- **理由を付けて止まるもの**（G-601）: アクセスリストの行の `log`・`log_input`・「効かせる時間帯」（NEC IX のアクセスリストに、行ごとの記録と時間帯はありません。時間で動かす仕組みは別にありますが、置き換えは作っていません）、NEC IX に無い ICMP の語（`traceroute` など。番号で書きます）、名前の無いプロトコルの語（`ah`・`ospf`。番号で書きます）、TCP のフラグが 2 つ以上、先頭と末尾が違う pat_pool、NAT の「複数の外側アドレスに対応させる」、同じ外側 IF に `pat_interface`（か pat_pool）の行がないポート転送、同じ外側 IF に 2 つ目の `pat_interface`、空白のある・31 字を超える NAT の行の名前（プール・サービスの名前に使うため。予約されたサービス名 any・dns・ftp・http・https・ping・snmp・ssh・telnet も不可）

### 置き換え（adapt.yaml。G-614 で知らせます）

- **番号が 10 刻みでないリスト**は、パックが刻みを 10 にして、番号を持つ書き方で書きます（別メーカーから来た番号を、そのまま残すためです）
- **「NAT の向き」**（Cisco の `ip nat inside`・`outside`）: NEC IX は、NAT の行を外側 IF の下に書き、IF に向きは書きません。NAT の行がすべて外側 IF を持ち、向きがそれと合っていれば（outside の IF が外側 IF、inside の IF はそうでない）、向きを外して知らせます（内側の IF は選べません: 外側 IF を通る通信すべてに当たります）。合っていなければ G-602 です。**外側 IF が空の NAT の行**（Cisco の静的 NAT・ポート転送）には、向きが outside の IF がちょうど 1 つなら、その IF を入れて知らせます

## 回線監視

読める行は「対応一覧」の「回線監視」です。見本は `tests/import/acl_nat`、`tests/render` の `nat_track`・`nat_track_failed`・`track_rename` です。

### 読み方の決まり

- **経路の切り替えは、監視の側に書かれています。** `watch-group` の `action … ip shutdown-route <宛先>` は、同じ宛先のスタティックルートの track 列に、その監視の名前を入れます
- **複数の `event`**（FD 2.28.3）: 番号が違う event は「どれかが起きたら動く」（全部が正常のときだけ正常 = all_up）、同じ番号で `sub` が違う event は「全部が起きたら動く」（1 つでも正常なら正常 = any_up）です。event ごとに `<グループ名>-1`、`<グループ名>-2` …（番号の小さい順）の行を作り、グループの名前の行（監視の方法 list）がそれをまとめます。間隔・回数は、まとめられる行に入ります。event が 1 つのグループは 1 行です
- **`action … ip decrement-vrrp-priority <VR 番号> <数>`** の 2 つ目の数は、下げる幅です（CRM p.17-4「減算プライオリティ値」）。その VR 番号の冗長化の行の「回線監視」「監視断のときの減点」に入れます

### 書き出し

| 項目 | 書き方 |
|---|---|
| 回線監視 | `watch-group <名前> 10` のブロックと、`network-monitor <名前> enable` を書きます。監視（ping）は `event 10 [vrf <VRF>] ip unreach-host <アドレス> [<出口の IF> [<ネクストホップ>]] [source <IF>]`、経路の監視は `event 10 [vrf <VRF>] ip unreach-route <宛先>/<長さ> [<ネクストホップ>\|<IF>]`、IPv6 の監視は `event 10 ipv6 unreach-host <アドレス> <出口の IF> [<ネクストホップ>] [source <アドレス>]`・`event 10 ipv6 unreach-route <宛先>/<長さ> [<ネクストホップ>\|<IF>]`（VRF は付けられません）、IF の監視は `event 10 interface-down <IF>`。ping の出口の IF は、値があるときだけ書きます（省けるのは Ver.10.6 以降です）。名前に空白があると止まります |
| 回数と間隔 | 値がある行に、`probe-counter variance <断と判断する回数>`・`probe-counter restorer <復旧と判断する回数>`・`probe-timer variance <間隔>`・`probe-timer restorer <復旧を待つ間の間隔>` を書きます |
| 止める経路 | 「回線監視」の列がその監視を指しているスタティックルートを、`action <10 刻みの番号> ip shutdown-route <宛先>/<長さ> [<次ホップ>]` で書きます。VRF の経路は書けません（止まります） |
| VRRP・IF を動かす | その監視を指す冗長化の行は `action <番号> ip decrement-vrrp-priority <VR 番号> <減点>`（「監視断のときの減点」が必須）、「止める IF」は `action <番号> shutdown-interface <IF>` |
| list の行 | 1 つの監視グループになり、まとめる行が event になります（all_up は `event 10`・`event 20` …、any_up は `event 10 sub 10`・`event 10 sub 20` …）。まとめられる行が、ほかの経路・冗長化から指されていなければ、その行だけの監視グループは作りません。まとめる行の間隔・回数は、同じ値でなければなりません。list の中に list は入れられません |

- **理由を付けて止まるもの**（G-601）: 「正常とみなす条件」が within_threshold の行、ネクストホップだけがあって出口の IF が無い ping の監視、出口の IF とネクストホップの両方を書いた経路の監視
- **書けない列**（値があれば G-602）: 回線監視の「しきい値」「待ち時間」「反映の待ち」（NEC IX に当たる行がありません）

### 置き換え（adapt.yaml。G-614 で知らせます）

- **回線監視の「送信元アドレス」**（Cisco の `source-ip`）: NEC IX は IPv4 の監視の送信元を IF 名で書くので、そのアドレスを持つ IF に置き換えます。持つ IF が無ければ、理由を付けて止まります
- **IPv6 の監視の「送信元 IF」**: IPv6 の監視は送信元をアドレスで書くので、その IF の固定の IPv6 アドレス（1 つ目）に置き換えます
- **IPv6 の監視の「監視の出口の IF」が空のとき**: IPv6 の監視には出力 IF が要るので、監視先を含む IPv6 のスタティック経路の出力 IF を入れます。入れられなければ、理由を付けて止まります
- **list がまとめる行の名前が `<list の名前>-1`、`-2` … でないとき**（Cisco の番号のままのときなど）は、パックがその名前の写しを作り、list がそれを指すようにして書きます（取り込み直すとその名前になるためです）。元の行は、経路・冗長化・ほかの list から指されていなければ書きません（何もしない監視グループになるためです）。指されていれば、その名前の監視グループも書きます。写しの名前の行がもうあるときだけ、理由を付けて止まります

## DHCP

### 読み方の決まり

- **DHCP の配るネットワーク。** プロファイルに `subnet-mask` があればそこから、無ければ結び付けたインタフェースのアドレスから決めます
- **`lease-time`** は、割り切れるいちばん大きい単位にします（86400 → `1d`、43200 → `12h`、5400 → `90m`、100 → `100s`）。`lease-time infinite` は `infinite` です

### 書き出し

| 項目 | 書き方 |
|---|---|
| DHCP | 行があれば `ip dhcp enable` を書きます。プールは `ip dhcp profile <名前>` で、そのネットワークにアドレスを持つインタフェース 1 つに `ip dhcp binding <名前>` で結び付けます（そういうインタフェースが無い・2 つ以上あると止まります）。配る範囲があれば `assignable-range` と `subnet-mask` を書き、無ければ書きません（ネットワークは、結び付けたインタフェースのアドレスから決まります） |
| DHCP の値 | `default-gateway`・`dns-server`（2 つまで。3 つ以上は止まります）・`domain-name`・`lease-time <秒>`（`2d` → 172800、`12h` → 43200、`90m` → 5400、`100s` → 100。`infinite` はそのまま）。配らないアドレスは `ip dhcp excluded-address <先頭> <末尾>` |

- **書けない列**（値があれば G-602）: 「デフォルトゲートウェイ（2 つ目以降）」

## PPPoE

読める行は「対応一覧」の「PPPoE」です（別紙 A 3.26）。PPPoE を受ける側（LAC の配下の PPPoE サーバ）は「L2TP」の節です。見本は `tests/import/basic`・`pppoe_more`、`tests/render` の `pppoe`・`pppoe_failed`・`pppoe_more`・`pppoe_more_adapt` です。

### 読み方の決まり

- **PPPoE の行の名前は、PPP プロファイルの名前です。** 論理 IF は `ppp binding` を書いたサブインタフェース、物理 IF は同じポートの「ポート名.0」です。1 つのプロファイルを 2 つのインタフェースが使う設定は、1 つ目だけが行になり、注意が出ます
- **PPP プロファイルの `authentication password` は、接続 ID ごとに読みます。** PPPoE の行は、`authentication myname` と同じ接続 ID のパスワードを使います。LNS のプロファイルでは、1 行が利用者 1 人（l2tp_users）になります
- **PPP プロファイルの `authentication request`** は、LNS のプロファイルのほかに、PPPoE のクライアントのプロファイルでは「相手も認証する」、LAC の配下の PPPoE サーバのプロファイルでは「利用者の認証」として読みます。クライアントのプロファイルで `authentication accept` と方式が違うときは、PPPoE の行を作りません（表の「認証方式」は 1 つです）

### 書き出し

| 項目 | 書き方 |
|---|---|
| PPPoE | 行ごとに `ppp profile <行の名前>` を書き、論理 IF の下に `ppp binding <行の名前>` を書きます。論理 IF は、ポートのサブインタフェース（「ポート名.1」以降）で、インタフェースの表に行が要ります。物理 IF は同じポートの「ポート名.0」でなければ止まります。名前は空白のない 31 字まで。`encapsulation pppoe` と `auto-connect` は書きません（サブインタフェースの既定です） |
| PPPoE の値 | `authentication myname <接続 ID>`・`authentication password <接続 ID> <パスワード>`（平文）、認証方式があれば `authentication accept chap｜pap｜chap-pap`、「相手も認証する」が有効なら `authentication request <認証方式>`、監視間隔があれば `lcp echo-interval <秒>`、サービス名があれば論理 IF の下に `pppoe service-name <名前>` |
| 通信があるときだけつなぐ | 有効な論理 IF に `no auto-connect`（と、あれば `idle-time <秒>`）を書きます。「無通信で切る時間」は、「通信があるときだけつなぐ」が有効のときだけ書けます |
| PPPoE のアドレス・MSS | 論理 IF のインタフェースの行に書きます（アドレス `negotiated` → `ip address ipcp`） |
| PPPoE で書けない列 | 「デフォルト経路を作る」（値があれば G-602。NEC IX に当たる行がありません。出力 IF を論理 IF にしたスタティックルートの行で書いてください → `ip route default <IF>`）、「つなぎ直すまでの待ち」「MAC アドレスを論理 IF のものにする」（G-602） |

### 置き換え（adapt.yaml。G-614 で知らせます）

- PPPoE の「つなぐきっかけの通信」は書きません（NEC IX は、自分でつなぐか、どの通信でもつなぐかのどちらかです）

## QoS

読める行は「対応一覧」の「QoS のクラスと条件」「QoS のポリシー」です（別紙 A 3.18・3.26）。見本は `tests/import/qos`・`qos_more`、`tests/render` の `qos`・`qos_adapt`・`qos_more`・`qos_more_failed`・`qos_more_unused` です。

### 読み方の決まり

- **QoS の DSCP・precedence。** この機種は DSCP・precedence をアクセスリストの末尾で見ます。クラスが指すリストのすべての行が `permit ip src any dest any dscp <数>`（または `precedence <数>`）だけのときは、その値を見分け方の条件（qos_class_matches の dscp・precedence 列）に入れ、**リストはアクセスリストの表から外します**（フィルタ・NAT・SNMP・IPsec も使っているリストは、リストのまま残します）。アドレスなどほかの条件があるリストも、リストのまま残します
- **QoS の既定のクラス。** `class-default` は `default`、`class-local` は `local`、`class-dynamic` は `dynamic` という名前で入れます。動作の行が無い `class` の行も、行として入れます（何も書いていない `class-default`・`class-local` は行にしません）。単位の語（`kbps`・`mbps`）は数に直します（`shape`・`traffic-shape rate` は bps、`priority`・`bandwidth` は kbps）

### 書き出し

- **書く行**: `class-map match-any|match-all <名前>` と「対応一覧」の `match …` の行、`policy-map` と `class`、`priority`・`priority percent`・`bandwidth`・`bandwidth percent`・`shape`・`set ip dscp`・`set ip precedence`・`set cos`・`set qos-group`・`set ipv6 dscp`・`set ipv6 precedence`・`queue-limit`、インタフェースの下の `service-policy input|output`・`service-policy output default-policy-map-ngn`・`traffic-shape rate`
- **DSCP・precedence の条件**: この機種には DSCP をじかに見る条件が無いので、条件ごとに `qos-d<番号>` というアクセスリストを作って書きます（取り込みは、これを条件の列に戻します）。同じ名前のリストを自分で作ると止まります
- **`class class-local` と `class class-default`** は、どのポリシーにも必ず書きます（機器にいつもある 2 つのクラスです）
- **`service-policy enable`** は、出力の行で、IF 全体を絞るか、ポリシーに優先・帯域・シェーピング・キューがあるときに書きます
- **順番**: クラスは「順番」の小さい順、その後ろに `class class-dynamic` → `class class-local` → `class class-default` の順で書きます。条件の順番は 10 刻みでなければ止まります。キューの `sub_a`〜`sub_d` は `sub-a`〜`sub-d` と書きます
- **理由を付けて止まるもの**（G-601）: プロトコル・ほかのクラス・「条件を逆にする」の条件、2 つ以上の CoS、1 行に 2 つ以上の条件（一緒に書けるのは「ACL と CoS」「NGN のトンネル IF と QoS グループ」だけ）、帯域（%）が 98 を超える行、9 つ以上のキューの長さ
- **書けない列**（値があれば G-602。人が決めます）: ポリシング（上限・捨てる、超えたときの動作）、公平に送る（fair-queue）、「遅らせ方」「上限・遅らせる（%）」「上限・捨てる（%）」「最大の速さ・捨てる」「最大を超えたとき」「混む前に間引く」「キューの長さの単位」

### 置き換え（adapt.yaml。G-614 で知らせます）

- 「帯域（残りの %）」は同じ数の「帯域（%）」として書きます
- 「優先の段」は、段のない優先として書きます
- 帯域（%）の合計が、機器の 2 つのクラスの各 10% を足して 100% を超えるときは、値を変えずに知らせます

## スタティックルート・OSPF・BGP・VRRP

読める行は「対応一覧」の「経路」です。ルートマップは次の節、ピアグループ・集約経路・バーチャルリンクなどの細かい行は「経路制御・BGP・OSPF の残り」の節です。見本は `tests/import/basic`、`tests/render` の `basic`・`dynamic`・`dynamic_failed` です。

### 読み方の決まり

- **VRF の BGP。** `router bgp` の下の `vrf <VRF>` のブロックを、その VRF の行（bgp_processes の VRF の行と、ネイバー・広告経路・再配布）にします。AS 番号は、`router bgp` の番号です
- **VRF の中の OSPF は、機器に機能がありません**（「読まない行」）。OSPF のプロセスは VRF に結べないので、VRF の中のインタフェースは OSPF の行にしません

### 書き出し

| 項目 | 書き方 |
|---|---|
| スタティックルート | `ip route [vrf <VRF>] <宛先>/<長さ> <次ホップ> [<IF>] [connected] [metric <数>] [distance <数>] [tag <数>]`（CRM p.8-14 の構文の順）。`0.0.0.0/0` は `default` と書きます。ネクストホップが `dhcp` の経路は `ip route <宛先> <IF> dhcp`（出力 IF が無いと止まります）。次ホップも出力 IF も無い経路は止まります |
| OSPF | `ip router ospf <番号>` の下に、`router-id`、`area … stub｜nssa [no-summary]`、`area … range <範囲> advertise｜not-advertise`（「広告する」が空なら advertise）、`redistribute`、`originate-default`、`passive-interface <IF>`、`network <IF> area <エリア>` を書きます。インタフェースは名前で入れます（アドレスの形の `network` は書きません） |
| OSPF で書けない形 | 「既定を受動にする」（この機種にありません。止まります）、VRF の中のインタフェース（止まります）、1 つのインタフェースを 2 つのプロセスに入れること（止まります）、網の種類 `point_to_multipoint`（語の対応がないエラー G-603） |
| OSPF のインタフェース | `ip ospf cost`・`priority`・`hello-interval`・`dead-interval`・`network`・`authentication [message-digest｜null]`・`message-digest-key <鍵番号> <鍵>`（`md5` の語はありません）・`authentication-key <鍵>`。Hello と Dead は「明示」の列なので、空なら既定値（10・40）を書きます |
| BGP | `router bgp <AS>` の下に `router-id` と、ネイバーの `remote-as`・`update-source`・`description`・`password`・`ebgp-multihop`・`timers <Keepalive> <Hold>`（2 つは組）・`shutdown`。`address-family ipv4 unicast` の下に `network`・`redistribute`・ネイバーの `distribute-list <プレフィックスリスト> in｜out`・`route-map <名前> in｜out`・`next-hop-self`・`send-default`。下に書く行がなければ `address-family` の行も書きません |
| BGP で書けない形 | 65535 を超える AS 番号（止まります）、VRF の行のルータ ID（この機種は VRF ごとに決められません。止まります）、VRF が空の行が無い BGP（止まります） |
| VRF の BGP | `router bgp <AS>` の下に `vrf <VRF>` のブロックを書き、その中にネイバーと `address-family ipv4 unicast` を書きます（機能説明書 2.17.7 の例の形）。AS 番号は機器に 1 つなので、全部の行で同じでなければ止まります |
| BGP の再配布 | 元が ospf の行は、「OSPF のプロセス番号」があれば `redistribute ospf <番号>` と書きます |
| VRRP | 行が 1 つでもあれば `vrrp enable` を書きます。インタフェースの下に `vrrp <番号> ip`・`priority`・`preempt`（無効なら `no vrrp <番号> preempt`）・`timers advertisement <秒>`・`authentication <鍵>`。方式が hsrp の行、認証 md5 は止まります。回線監視との連動は「回線監視」の節です |
| VRRP の版 3（IPv6） | 冗長化（IPv6 の VRRP）の行は、インタフェースの下に `vrrp <番号> ipv6 <仮想リンクローカルアドレス>`、仮想グローバルアドレスごとに `vrrp <番号> ipv6 <アドレス>`、`priority`・`preempt`（無効なら `no vrrp <番号> preempt`）、広告間隔は秒で割り切れれば `timers advertisement <秒>`、そうでなければ `timers advertisement csec <センチ秒>`。行が 1 つでもあれば `vrrp enable` を書きます |
| VRRP の版 3 で書けない形（G-601） | IPv4 の VRRP か、ほかの IF の IPv6 の VRRP と同じ番号（機器の中で番号を重ねられません。FD 2.27.1）、30 秒を超える広告間隔・10 ミリ秒で割り切れない広告間隔、回線監視（`action … ip decrement-vrrp-priority` が IPv6 のグループに効くかを資料で確かめられません）、fe80::/10 に無い仮想リンクローカルアドレス。切り戻し待ちは書きません（値があれば G-602） |
| VRRP の版の置き換え | 機器の「VRRP の版（IPv4）」が 3 なら、列を空にして、IPv4 のグループを版 2 で書きます。IPv4 の VRRP の行があれば G-614 で知らせます（同じグループの相手が版 3 だけの機器ならつながらないため。X-405）。冗長化の「版 2 の相手とも話す」は、この機器の IPv4 の VRRP がいつもそう動くので、知らせずに空にします |

- **書けない列**（値があれば G-602）: スタティック経路の「IF が落ちても残す」「ネクストホップを全体の経路表で引く」「マルチキャストの確認だけに使う」、OSPF の「隣接の変化の記録」、VRF の「取り込む／付けるルートターゲット」（VRF-Lite で、当たる設定がありません）と、共通の「書けない列」の表の列

## 経路制御の方針（ルートマップ・ポリシールーティング）

読める行は「対応一覧」の「経路制御の方針」です（別紙 A 3.19）。IPv6 の条件と動作、`set community … delete` は「経路制御・BGP・OSPF の残り」の節です。見本は `tests/import/route_policy`、`tests/render` の `route_policy`・`route_policy_adapt`・`route_policy_failed` です。

### 読み方の決まり

- **ルートマップ。** この機種には、AS パスのリストもコミュニティのリストもありません。コミュニティは、ルートマップの行にじかに書きます（設計データも、その形で持ちます。別紙 A 3.19）。決まった語は、`no-export-subconfed` を中立の語 `local-as` として入れます（`lexicon.yaml`）。条件と動作は、1 行に名前・値を 1 つずつです

### 書き出し

節 `route_policy` は、アクセスリスト・プレフィックスリスト・QoS の後ろ、経路（`routing`・`ospf`・`bgp`）の前です（機能説明書に「redistribute は、ルートマップ・プレフィックスリストの設定の後に行う」とあります）。ルートマップ → `ip local policy route-map` の順に書きます。

- **書く行**: 取り込みが読める行だけです。`route-map` は、動作と番号を必ず書きます
- **書けないもの**（理由を付けて止まります）: 方針の条件に使われている AS パスのリストと「条件: AS パス」（この機種に、AS パスで経路を見分ける設定がありません）、2 行以上・deny・正規表現のコミュニティのリスト、1 行に 2 つ以上のプレフィックスリスト・ACL・出力 IF・タグ・ネクストホップ、プレフィックスリストと ACL の両方を条件にした行、65535 を超える「AS パスに足す AS」、33 個以上のコミュニティ、31 字を超える方針の名前
- **書けない語**（語の対応がないエラー G-603）: コミュニティの `internet`・`gshut`（この機種の決まった語は `no-advertise`・`no-export`・`no-export-subconfed` の 3 つです）
- **書けない列**（値があれば G-602）: 方針の「付ける値: weight」、コミュニティのリストの表の行（置き換えられなかったもの）

### 置き換え（adapt.yaml。G-614 で知らせます）

- permit の 1 行だけの標準のコミュニティのリストを指す行は、リストの中身を条件にじかに書きます（意味は同じです）
- ネイバーの「コミュニティを送る」は、当たる行が無いので書きません（コミュニティが相手に届くかどうかは、機器で確かめてください）
- **どの方針の条件にも使われていない AS パスのリスト**は、書かずに知らせます（「条件: AS パス」を別の条件に書き直した後に、リストの行を人が消さなくて済むようにするためです）

## 経路制御・BGP・OSPF の残り（別紙 A 3.19 の「経路制御の残り」）

「読まない行」に残っていた、ルートマップ・BGP・OSPF の書き方を読み書きできるようにしたものです。見本は `tests/import/bgp_more/`・`tests/import/routing_more/`、`tests/render/bgp_more/`・`tests/render/routing_more/` です。**実機の出力では確かめていません。**

### 読める行と入る表

| 分野 | 読める行 | 入る表・列 |
|---|---|---|
| ルートマップ | `match ipv6 address prefix-list <名前>`、`match ipv6 address access-list <名前>`、`set ipv6 next-hop <アドレス>`、`set ipv6 default next-hop <アドレス>`、`set community delete <値> …`、`set community <値> … additive delete <値> …` | route_policies（match_ipv6_prefix_list・match_ipv6_acl・set_ipv6_next_hop・set_ipv6_default_next_hop・set_community_delete） |
| ルートマップ（IPv6 のネクストホップ） | `match ipv6 next-hop prefix-list <名前>`（CRM 23-5） | route_policies（match_ipv6_next_hop_prefix_list） |
| IPv6 のポリシールーティング | IF の `ipv6 policy route-map <名前>`、`ipv6 local policy route-map <名前>` | ipv6_interfaces（policy_route）、devices（local_policy_route_v6） |
| BGP のネイバー | `neighbor <アドレス> advertisement-interval <秒>`、`neighbor <アドレス> route-reflector-client` | bgp_neighbors |
| BGP のピアグループ | `peer-group <名前> remote-as <AS>` と、その下の `neighbor <アドレス> [remote-as <AS>]`、`description`、`password`、`ebgp-multihop`、`update-source`、`timers`、`shutdown`、`advertisement-interval`、`route-reflector-client`、`address-family ipv4 distribute-list <名前> in\|out`、`address-family ipv4 route-map <名前> in\|out`、`address-family ipv4 next-hop-self`、`address-family ipv4 send-default` | bgp_neighbors（メンバーの行） |
| BGP の全体 | `timers <Keepalive> <Hold>`、`default-local-preference <数>`。`address-family ipv4 unicast` の下の `aggregate-address <プレフィックス> [as-set] [summary-only]`、`distance [ebgp <数>] [ibgp <数>]` | bgp_processes、bgp_aggregates |
| OSPF | `originate-default … [metric-type 1\|2] [route-map <名前>]`、`distance [external <数>] [inter-area <数>] [intra-area <数>]`、`distribute-list prefix <名前>`・`distribute-list route-map <名前>`、`area <エリア> virtual-link <ルータ ID> [dead-interval <秒>] [hello-interval <秒>]` | ospf_processes、ospf_virtual_links |

### 読み方の決まり

- **ピアグループは、表に持ちません。** グループの設定を、メンバーの行のうち自分で値を持たない列に入れます。FD の 2.22 章の「ピアグループ設定」の節が、グループを使った設定と、ネイバーごとに書いた設定を「同等の設定」として並べています。メンバーの AS 番号は、メンバーの行に書いた番号、無ければグループの番号です。書き出しは、いつもネイバーごとの行です
- **`set community … additive delete …`** は、足す値を「付ける値: コミュニティ」（「コミュニティを足す」が有効）、外す値を「外すコミュニティ」に入れます

### 書き出しと置き換え（adapt.yaml）

- BGP の `timers`・`default-local-preference` は、VRF が空の行だけに書けます。`aggregate-address` と `distance` は、`address-family ipv4 unicast` の下に書きます（CRM の実行モード）
- **OSPF の「優先度（distance）」**（経路ぜんぶの 1 つの値。Cisco の `distance <数>`）は、種類ごとの 3 つの列に写して `distance external … inter-area … intra-area …` と書きます（同じ意味なので、知らせません）
- **BGP の「優先度（自分で作った経路）」**（Cisco の `distance bgp` の 3 つ目）は、当たる設定がありません。200（Cisco の既定）なら知らせずに外し、ほかの値は外して G-614 で知らせます
- 「外すコミュニティ」は、コミュニティを足す行（`additive`）か、外すだけの行に書けます。置き換える行（`additive` の無い `set community <値>`）と一緒には書けません（理由を付けて止まります）
- **書けないもの**（理由を付けて止まります）: 「条件: ネクストホップ（ACL）」（プレフィックスリストで見ます）、「条件: 経路の種類」、「合った後に続けて見る順番」（最初に合った行で終わります）、ネイバーの「受信 / 送信の AS パスのリスト」「受信 / 送信フィルタ（ACL）」、OSPF の「経路表に入れる経路の絞り込み（ACL）」、2 つ以上の IPv6 のネクストホップ
- **BGP の「ネイバーの変化を記録」**（Cisco の `bgp log-neighbor-changes`）は、当たる行がありません。外して G-614 で知らせます（経路には影響しません）
- **書けない列**（値があれば G-602）: 方針の拡張コミュニティ（ルートターゲット・SoO）、ネイバーの「デフォルト経路の広告の条件」「受け取る経路数の上限」と関連の 3 列・「weight」「拡張コミュニティを送る」、OSPF の「通過させない（最大メトリックで広告）」

### 読まない行（「未対応」として画面に出ます）

| 行 | 理由 |
|---|---|
| ピアグループの下の `listen range …`、`passive`、`connect-interval`、`send-capability`、メンバーが 1 つも無いグループ | 範囲で受けるネイバーは、ネイバーごとの行にできません。ほかは入れる列がありません |
| BGP の `originate-default [always]`、`default-metric`、`multipath`、`cluster-id`、`redistribute-internal`、`bgp listen limit`、`neighbor … passive`・`connect-interval`・`outgoing-interface`・`send-capability` | 入れる列がありません（`originate-default` は、Cisco の BGP の `default-information originate` と意味がそろいません） |
| 空白を含む `neighbor <アドレス> password <値>` | 秘密の値に空白は入れられません（空白のある値は「機器が暗号化した形」として扱うため。別紙 A 3.26） |
| OSPF の `distance … nssa-external <数>` の付いた形 | NSSA の外部経路の値を入れる列がありません |
| `area <エリア> virtual-link` に `authentication`・`authentication-key`・`message-digest-key`・`retransmit-interval`・`transmit-delay` を付けた形 | 入れる列がありません |
| OSPF の `originate-default … tag <数>` など、上の表に無い語の付いた形 | 入れる列がありません |

### 公開資料で確かめられず、読まないもの

公開資料で書き方か意味を確かめられなかったもの、設計データの形に入らないものです。どれも「未対応」として画面に出ます（黙って落としません）。

| 書き方 | 探した資料と、読まない理由 |
|---|---|
| アクセスリストのアドレスをワイルドカードで書いた形（`src 192.0.2.0 0.0.0.255`） | CRM 24 章は、入力の形としてワイルドカードを示していますが、入れた行を機器が running-config にどう表示するか（プレフィックス長に直すか、ワイルドカードのままか）は書いていません。とびとびのワイルドカード（`0.0.255.0` など）は設計データのアドレスの型に入らないので、読んでいません |
| 同じ名前の `ip nat pool` が 2 行以上（とびとびの範囲） | 設計データのプールは、先頭と末尾の 1 組です。行を増やすと、NAT の行の名前（自動で振る名前）と Cisco の側のプール（1 行）に合わなくなるので、読んでいません |
| VRF の中の IPv6 の経路（`ipv6 route vrf …`） | 機能説明書 2.17 の VRF 機能の対応状況の一覧に「IPv6 はルーティング機能を含めて VRF 未対応です」とあります。機器に無い設定なので、読み書きしません |
| MAP-E の規則を手で書く形（Cisco の `nat64 map-e domain …`・`basic-mapping-rule` に当たるもの） | この機器の MAP-E は、事業者を選んで規則を自動でもらう形（`tunnel mode map-e jpix\|ocn\|ocn-fixed`。CRM 12 章、FD 2.33.10）だけで、規則を手で書く行がありません。Cisco の規則の値を書く先が無く、事業者の規則に置き換えると値が使われないまま別の規則で動くので、置き換えもしません |
| OSPFv3 の認証・バーチャルリンク | CRM 10 章の OSPFv3 のコマンド一覧にありません（Cisco の OSPFv3 の認証を書こうとすると止まります） |

### 確かめていないこと

- **running-config での形。** ピアグループの下の行の字下げと順番（FD の設定例の形で読みます）、`distance` と `area … virtual-link` の語の順（CRM の入力形式の順で書きます）、`set community … additive delete …` の表示は確かめていません
- **ルートマップと IPv6 のリストの順番。** IPv6 のプレフィックスリスト・フィルタは、ルートマップより後ろに書き出しています（IPv6 の節をまとめて BGP の後ろに書くためです）。ルートマップが、まだ無いリストの名前を指せるかどうかは、資料では確かめられていません。投入で弾かれる場合は、IPv6 のリストを先に入れてください
- **VRRP の版 3。** running-config での行の順（`vrrp <番号> ipv6` の行と `priority` などの順。どの順でも読みます）、リンクローカルを先に書く必要があるか（書き出しでは先に書きます）。`action … ip decrement-vrrp-priority <番号>` が IPv6 のグループに効くか（番号は IPv4 と IPv6 で重ならないので効く見込みはありますが、`ip` の語があり、資料に IPv6 の例がありません。読み書きしません）
- **OSPF の `distribute-list` の向き。** CRM の説明（「ルーティングテーブルに内容を格納する時、フィルタリングに使用」）から、Cisco の `distribute-list … in` と同じ「経路表に入れる側」として読んでいます

## RIP と RIPng（別紙 A 3.27）

構文は CRM の 10 章（RIP・RIPng）で確かめました。**実機の出力では確かめていません。** 見本は `tests/import/rip/`・`tests/render/rip/`・`tests/render/rip_adapt/` です。

### 読める行と入る表

| 分野 | 読める行 | 入る表・列 |
|---|---|---|
| RIP・RIPng のプロセス | `ip router rip`・`ipv6 router rip` の下の `[no] auto-summary`（RIP だけ）、`default-metric <数>`（RIP だけ）、`distance <数>`、`distribute-list prefix <名前> in\|out`、`originate-default [<メトリック>]`、`timers <更新> <無効> <消すまで>`（RIP だけ）、`redistribute connected\|static\|ospf <番号>\|bgp [metric <数>] [tag <数>] [route-map <名前>]` | rip_processes・rip_redistribute（種類 ipv4・ipv6） |
| IF | `ip rip enable`・`ipv6 rip enable`、`[no] ip rip send`・`receive`（RIPng も同じ形）、`ip rip send version 1\|2\|1 2\|compatible`、`ip rip receive version 1\|2\|1 2`、`ip rip authentication text <鍵>`、`ip rip metric-offset`・`split-type`・`summary-address`・`originate-default [<メトリック>]`・`distribute-list prefix <名前> in\|out`（RIPng は `ipv6 rip …` の同じ形） | rip_interfaces |

### 読み方の決まり

- `ip rip send version compatible` は、「版 2 をブロードキャストで送る」（Cisco の `ip rip v2-broadcast`）として入れます
- `timers` の 3 つ目（無効になってから消すまで）は、「無効とみなすまで」を足した値（最後の更新から消すまで）にして入れます
- 「クラスの境目で集約する」は、既定がメーカーで逆なので、取り込みでは必ず値が入ります（この機器の既定はしない）

### 書き出し

- 版はプロセスに書く列ですが、この機器は IF ごとに書くので、IF の「送る版」「受ける版」が空の IF に、プロセスの版を書きます
- **置き換えて知らせるもの（G-614）**: 「既定を『送らない』にする」は、「送る」が空の IF の行を「送らない」にして書きます。「より良い経路を受け付けない時間」（Cisco だけ）は外します
- **書けないもの（理由を付けて止めます。G-601）**: 認証の md5、RIPng の「クラスの境目で集約する」・再配布のメトリックの既定・タイマ（RIPng に無い設定です）、RIPng の再配布の元 bgp
- **書かない列（値があれば G-602）**: 「デフォルト経路の広告の条件」（Cisco のルートマップ）

### 読まない行（「未対応」として画面に出ます）

| 行 | 理由 |
|---|---|
| `ip rip neighbor`・`send unicast`、`send-default`・`receive-default`、IF の `originate-default` のネクストホップ | 入れる列がありません |
| VRF の中の RIP | 機器に機能がありません（機能説明書 2.17 の対応状況の一覧で、VRF の RIP は「×」です） |

## IPsec

読める行は「対応一覧」の「トンネル・IPsec・L2TP」です（別紙 A 3.20）。扱う形は、トンネル IF 型（IKE の版 1 と版 2）、トンネル IF を使わない方式 policy（版 1）、L2TP/IPsec の受け口を守る IPsec、IPsec で守る EtherIP です。認証は事前共有鍵だけです。見本は `tests/import` の `ipsec2`・`ipsec_lists`・`ipsec_more`・`ipsec_policy`・`ipsec_rest`・`ipsec_third`・`etherip_ipsec`、`tests/render` の `ipsec`・`ipsec2`・`ipsec_adapt`・`ipsec_failed`・`ipsec_lists`・`ipsec_more`・`ipsec_policy`・`ipsec_rest`・`ipsec_third`・`ipsec_third_failed`・`etherip_ipsec`、2 メーカーの間は `internal/ops/testdata/ipsec_h2` です。

### 読み方の決まり

- **IPsec のトンネル IF 型。** マップが指すアクセスリストの 1 行目が「すべて許可（`permit ip src any dest any`）」のときだけ読みます。それ以外は「未対応」に戻ります。行の名前はトンネル IF の名前です。**その 1 行目は、アクセスリストの表（acl_entries）から外します。行が残らなければ、リスト（acls）も外します**（フィルタ・NAT・SNMP も使っているリストは、そのまま残します）。トンネル IF の下の行は `ipsec policy tunnel <マップ名>` と、後ろに `out` が付いた形の両方を読みます
- **相手・ID・鍵・寿命（IKE 版 1）。** 相手は、アドレス・`any`（`ipsec dynamic-map`）・名前（`peer-fqdn-ipv4`）のどれかで、相手アドレスの列にそのまま入れます。動的なマップは、`ike <ポリシー>` が指す IKE のポリシー（書いていなければ、相手が `any` のポリシー）とつなぎます。`ike local-id`・`ike remote-id` は、自分の ID・相手の ID に「種類:値」で入れます（`user-fqdn` は email）。`key-type hex` の鍵は「鍵の書き方」が hex です。`ike nat-traversal` は、ポリシーを指した行があればその行のもの、無ければ機器全体のものを読みます（`keepalive` は「NAT 越えの維持間隔」、`force` は「NAT 越えを必ず使う」）。`lifetime both <秒> <キロバイト>` は、有効時間と有効量の両方に入れます
- **相手が any の IKE ポリシーが 2 つ以上ある機器**の、IKE ポリシーを指さない `ipsec dynamic-map` は、行にしません（どのポリシーとつなぐかを決められないため。「読まない行」）
- **`ipsec dynamic-map` の `ike-binding`。** `ike <ポリシー>` と同じく IKE のポリシーを指し、「この IKE の相手だけに使う」（ike_bind）を true にします
- **IPsec の IKE 版 2。** トンネル IF の `ikev2 peer`（か `ikev2 peer-fqdn-ipv4`）の行が、ipsec_tunnels の 1 行になります（版は 2）。**トンネルが自分で言わないことは、割り当てたプロファイル（`ikev2 binding`）から、それにも無ければ既定のプロファイル（`ikev2 default-profile`）から、項目ごとに読みます。** どこにも無い項目は空です（IKE の暗号・ハッシュ・DH グループ・PRF、ESP の暗号・ハッシュ、PFS は、空なら機器が持つ候補に任せる。別紙 A 3.20）。鍵は `ikev2 authentication psk id …` の行を ID で引きます: 自分の鍵は自分の ID（`local-authentication psk id …`。ID が無ければ、`ikev2 source-address` のアドレスか、その IF のアドレス）、相手を確かめる鍵は相手の ID（`ikev2 peer … id …`。ID が無ければ相手のアドレス、名前の相手なら名前）の行です。2 つの鍵が同じなら「相手を確かめる鍵」は空です。鍵が見つからない・片方が hex で片方が文字のときは「未対応」に戻ります。`ikev2 peer <アドレス>` は tunnels の宛先、`ikev2 source-address` は送信元にも入ります。`ikev2 connect-type auto` は「いつもつないでおく」の有効、`trigger` は無効です。`dpd` だけの行は 10 秒（機器の既定）、`retransmit count <回数> interval <秒>` は相手監視の再送回数・再送間隔（相手監視があるときだけ読みます）
- **候補を並べた設定。** IKE の版 1: `ike policy … <プロポーザル>,<プロポーザル>,…`（4 つまで）と `ipsec autokey-map`（`ipsec dynamic-map`）`… <プロポーザル>,<プロポーザル>,…`（8 つまで。どちらも機器の上限です）の 2 番目以降は、「IPsec 追加の提案」（ipsec_proposals。段 ike・段 ipsec、順番 20・30…）の行になります。何も書いていないプロポーザル（`ike proposal <名前>` だけ）も 1 行です（値は空 = 機器の既定）。上限を超えて並べた行は「未対応」です。IKE の版 2: `sa-proposal enc aes-cbc-256 aes-cbc-128` のように 1 行に並べた値は、1 つ目が IPsec の行、2 つ目以降がその行の「IKE 暗号（追加）」などの「（追加）」の列に入ります（`sa-proposal` の enc・integrity・prf・dh、`child-proposal` の enc・integrity、`child-pfs`。トンネル・プロファイル・既定のプロファイルのうち、1 つ目の値を読んだ所の並びを読みます）
- **IPsec の方式 policy（トンネル IF を使わない形）。** マップを、トンネル IF でないインタフェースに `ipsec policy tunnel <マップ>`（トンネルモード）か `ipsec policy transport <マップ>`（トランスポートモード）で当てた設定は、**マップの名前を行の名前にした、方式 policy の行**になります（自分側 IF は、そのインタフェース）。マップのリストの `permit` の行が対象通信（ipsec_selectors）で、`ip`・`udp`・`tcp`、ポートは `eq <番号>`、`any` は 0.0.0.0/0 です（1 行だけの「すべて許可」は、対象通信の行にしません）。**リストに、対象通信にできない行（`deny`、`icmp`、`eq` 以外のポート、`dscp`・`precedence`）が 1 行でもあるか、リストをほかのマップも使っているか、マップを 2 つ以上のインタフェースに当てていると、行にせず「未対応」に戻します。** 写したリストは、アクセスリストの表から外します（トンネル IF 型と同じ決まり）。メーカーは、この形（FD 2.35.2 の (b)）より、トンネル IF を使う形を勧めています
- **L2TP を守る IPsec（L2TP/IPsec の受け口）。** LNS のトンネル IF（`tunnel mode l2tp-lns ipsec`）が `ipsec policy transport <マップ>` で指すマップは、**方式 policy・相手アドレス any・IPsec のモード transport の行**になります（行の名前はマップの名前。自分側 IF は空）。LNS の行（l2tp_groups）の「IPsec の設定名」が、この行の名前です。動的なマップに `ike <ポリシー>` が無ければ、相手が `any` の IKE のポリシーとつなぎます（そのポリシーが 1 つだけのとき）
- **マップが相手に伝える範囲（フェーズ 2 の ID）。** トンネル IF に当てたマップの `ipsec local-id <マップ> …`・`ipsec remote-id <マップ> …` は、そのトンネルの IPsec の行の対象通信（ipsec_selectors）の 1 行になります（順番 10。書いていない側は 0.0.0.0/0 = リストの any）。アドレスだけの形はホスト（/32）として読みます。**`/32` と書いた形と IPv6 は読みません**（アドレスだけの形とは、相手に送る ID の種類が違い、表では見分けられないため）。方式 policy のマップに付いた `ipsec local-id`・`remote-id` は読みません（その行だけ「未対応」に残ります）
- **IKEv2 のトラフィックセレクタ。** `local-ts ipv4 …`・`remote-ts ipv4 …`（プロファイルの下。トンネル IF の下は `ikev2 local-ts …`）も、そのトンネルの対象通信の 1 行になります。読むのは、ネットワーク（`address <ネットワーク/長さ>`）、`protocol tcp｜udp`（番号の 6・17 も）、ポート 1 つ（`port <番号> <同じ番号>`）です。アドレスの範囲・ポートの範囲・ほかのプロトコル・`ipv6` は「未対応」です。自分側と相手側でプロトコルが違う組は、行にしません
- **相手監視の送り方。** この機種は選べません。版 1 は on_demand（通信が無いときだけ聞く）、版 2 は periodic（いつも間隔ごとに聞く）を、相手監視のある行に入れます（FD 2.38 章）
- **IPsec で守る EtherIP のトンネルの行は、IPsec の行と一緒に作ります。** IPsec の行を作れないとき（マップのリストが「すべて許可」の 1 行でない、鍵が見つからない、`with-id-payload` が無い など）は、`tunnel mode ether-ip ipsec` の行ごと「未対応」に残ります（IPsec の無い EtherIP として読んでしまうと、守られていない設定が黙ってできるため）。設定事例集の版 1 の例の形（マップのリストが両端のアドレスどうしの 1 行、`ipsec local-id`・`remote-id` が `/32`）は、この決まりで「未対応」のままです（`/32` の ID を表せません）
- **`with-id-payload`** は列にしません。EtherIP のトンネルの下の `ipsec policy transport` に付いた形だけを読み、その形の行にいつも書きます
- **VRF は、内側だけです。** この機種は、トンネルの中を VRF にできます（FD 2.17 の表「IPsec / ダイナミック VPN Inner 対応（トンネル内を VRF 化）○」）。トンネルの外側（暗号化した通信が通る側）は VRF にできません（同じ表の「IPsec Outer 対応（トンネル外を VRF 化）×」）

### 書き出し

| 項目 | 書き方 |
|---|---|
| トンネル IF（種類 ipsec） | **IKE の版 1 の行は `tunnel mode ipsec` と `ipsec policy tunnel <マップ名> out` だけを書き、送信元・宛先・鍵・監視は読みません**（NEC IX の IKE 版 1 の IPsec のトンネルには、その行がありません。値があれば G-602。セルを空にしてください。別メーカーのデータの宛先・送信元は、下の置き換え）。**IKE の版 2 の行は `tunnel mode ipsec-ikev2` と書き、送信元を `ikev2 source-address <アドレスか IF>` として書きます。宛先は IPsec の相手アドレスと同じでなければ止まります**（空でもかまいません） |
| IPsec（IKE 版 1・トンネル IF 型） | 事前共有鍵だけ（ほかは止まります）。トンネル IF `Tunnel<数>.0` の行ごとに、設定事例集の形で書きます: `ip access-list sec-<数> permit ip src any dest any`、`ike proposal ike-prop-<数> encryption … hash … group … [lifetime <秒>]`、`ike policy ike-<数> {peer <相手>｜peer any｜peer-fqdn-ipv4 <名前>} key <鍵> [key-type hex] [mode main｜aggressive] ike-prop-<数>`、`ike local-id`・`ike remote-id ike-<数> <種類> <値>`（ID があるとき。email は `user-fqdn`）、`ike keepalive ike-<数> <秒> [<回数>]`（相手監視の間隔があるとき）、`ipsec autokey-proposal ipsec-prop-<数> esp-… [esp-…] [ah-…] [lifetime time <秒>｜lifetime both <秒> <キロバイト>]`、`ipsec autokey-map ipsec-<数> sec-<数> {peer <相手>｜peer-fqdn-ipv4 <名前>} [pfs off｜<ビット数>] ipsec-prop-<数>`（相手が any なら `ipsec dynamic-map ipsec-<数> sec-<数> [pfs …] ike ike-<数> ipsec-prop-<数>`。「この IKE の相手だけに使う」が有効なら `ike` の代わりに `ike-binding`）。**対象通信の行が 1 つあるトンネル IF 型の行**（相手に伝える範囲。Cisco の `tunnel protection ipsec policy ipv4` から来ます）は、マップの後ろに `ipsec local-id ipsec-<数> <自分側>`・`ipsec remote-id ipsec-<数> <相手側>` を書きます（0.0.0.0/0 の側は書きません。/32 はアドレスだけで書きます。2 行以上、プロトコル・ポート付きは止まります）。有効量（キロバイト）は、有効時間が無いと止まります。IPsec のモード transport は止まります（方式 policy の行にします。EtherIP を守る行だけは transport です） |
| IPsec（IKE 版 1・方式 policy） | 行の名前をマップの名前にします（半角の英数字・`-`・`_` で 30 字まで。**「/」は「-」に置き換えて書きます**: Cisco の `CMAP-A/10` は `CMAP-A-10`。往復の確認は、名前でなく中身で行を合わせます）。ほかの部品は、方式 policy の行の順に `p1`・`p2`… と番号を付けます: `ip access-list sec-p<数> <順番> permit ip｜udp｜tcp src <自分側> [sport eq <ポート>] dest <相手側> [dport eq <ポート>]`（対象通信の行ごと。行が無ければ「すべて許可」の 1 行）、`ike proposal ike-prop-p<数> …`、`ike policy ike-p<数> …`、`ipsec autokey-proposal ipsec-prop-p<数> …`、`ipsec autokey-map <行の名前> sec-p<数> peer … ipsec-prop-p<数>`（相手が any なら `ipsec dynamic-map <行の名前> sec-p<数> … ipsec-prop-p<数>`）。自分側 IF（必須。トンネル IF は止まります）の下に、IPsec のモードが transport なら `ipsec policy transport <行の名前>`、ほかは `ipsec policy tunnel <行の名前> out`。IKE の版 2 の行は止まります（この機種の IKEv2 はトンネル IF 型だけです） |
| L2TP を守る IPsec | LNS の行の「IPsec の設定名」が指す行（方式 policy・相手 any・モード transport。違えば止まります）を、上の方式 policy の形の動的なマップとして書き、受け口のトンネル IF の下に `ipsec policy transport <行の名前>` を書きます（FD 2.36.4 の設定例の形） |
| IPsec（IKE 版 2） | 方式は tunnel_if だけ、事前共有鍵だけ。先に、鍵を ID ごとに 1 行ずつ書きます: `ikev2 authentication psk id <種類> <値> key char｜hex <鍵>`（自分の ID に「事前共有鍵」、相手の ID に「相手を確かめる鍵」。空なら事前共有鍵。同じ ID で鍵が違う行があると止まります）。**自分の ID が空の行は、トンネルの送信元（アドレスか、アドレスを持つ IF）から自分のアドレスを決めて、`id ipv4 <アドレス>` の鍵として書きます。決まらなければ止まります。** 相手の ID が空なら、相手のアドレス（`id ipv4`）か名前（`id fqdn`）です。相手が any の行は、相手の ID が要ります。続けて、行ごとに `ikev2 profile ikev2-<数>` を書きます（値がある項目だけ）: `child-lifetime`、`child-pfs off｜<ビット数>`、`child-proposal enc`・`integrity`、`dpd interval <秒>`、`ipsec-mode transport`、`local-authentication psk [id <種類> <値>]`、`nat-traversal [keepalive <秒>] [force]`（無効なら `nat-traversal off`）、`retransmit count <回数> interval <秒>｜backoff`、`sa-lifetime`、`sa-proposal dh`・`enc`・`integrity`・`prf`。トンネル IF の下に `tunnel mode ipsec-ikev2`、`ikev2 binding ikev2-<数>`、`ikev2 connect-type auto｜trigger`（「いつもつないでおく」に値があるとき）、`ikev2 peer <相手>｜any authentication psk [id …]`（名前なら `ikev2 peer-fqdn-ipv4`）、`ikev2 source-address`。**「（追加）」の列の値は、1 つ目の値に続けて同じ行に並べます**（`sa-proposal enc aes-cbc-256 aes-cbc-128` など。1 つ目の値が空だと止まります）。**対象通信の行が 1 つある行**は、プロファイルに `local-ts ipv4 [protocol tcp｜udp] [address <自分側>] [port <番号> <番号>]`・`remote-ts ipv4 …` を書きます（0.0.0.0/0 でプロトコルも無い側は書きません。2 行以上は止まります） |
| IPsec の追加の提案 | ipsec_proposals の行を、順番の小さい順に、IKE の版 1 の行に書きます（番号そのものは書きません）。段 ike は `ike proposal ike-prop-<数>-2`（`-3`・`-4`。全部で 4 つまで）、段 ipsec は `ipsec autokey-proposal ipsec-prop-<数>-2`…（全部で 8 つまで）を作り（値のある項目だけ）、ポリシーとマップの最後に `ike-prop-<数>,ike-prop-<数>-2,…` とカンマで並べます。上限を超えると止まります |
| IPsec の相手監視 | 版 1 は `ike keepalive`（間隔と回数）、版 2 は `dpd interval` と `retransmit`（回数と間隔の組）。版 1 の再送間隔は書けません（値があれば G-602） |
| IPsec の NAT 越え | 版 1: 版 1 の全部の行が有効で、維持間隔・「必ず使う」も同じなら、機器に 1 行（`ike nat-traversal [keepalive <秒>] [force]`）を書きます。行によって違うときは、有効な行ごとに `ike nat-traversal policy ike-<数> …` を書きます。「無効」と書いた行は止まります（行が無いことが無効です）。版 2: プロファイルの下に `nat-traversal …`、無効なら `nat-traversal off` |
| IPsec で守る EtherIP | 種類が etherip のトンネルを指す IPsec の行（方式 tunnel_if）があれば、トンネル IF に `tunnel mode ether-ip ipsec` と `ipsec policy transport ipsec-<番号> with-id-payload`（版 1）、`tunnel mode ether-ip ipsec-ikev2` と `ikev2 …` の行（版 2）を書きます。**その行の「IPsec のモード」が transport でなければ止まります**。守られる EtherIP のトンネルは、`tunnel source`・`tunnel destination` を書きません |
| ESP の認証を使わない | 「ESP ハッシュ」の `none`: 版 2 は `child-proposal integrity off`。**版 1 は止まります**（プロポーザルに「使わない」と書く語がありません） |
| 暗号化して保存した鍵 | 鍵が `secret <値>` の行: 版 1 は `key <値> key-type secret 1`、版 2 は `key secret <値>`。別メーカーの形（`6 <値>` など）は止まります（G-601。平文を入れ直します） |
| VRF | **「外側の VRF」のある行は止まります**（FD 2.17）。**「内側の VRF」のある方式 policy の行も止まります**（VRF で IPsec を使う形として資料にあるのは、トンネル IF 型だけです。方式を tunnel_if にして、トンネル IF の「VRF」に書きます） |
| 相手が any のトンネル IF 型 | Cisco の仮想テンプレートから来た行は、動的なマップ（版 1）か `ikev2 peer any`（版 2）で書きます。**この機種のトンネル IF 1 つがつなぐ相手は 1 つ**なので、相手の数だけ行とトンネル IF を足してください（パックは増やしません。版 2 は相手の ID が要ります） |
| IPsec の名前 | プロポーザル・ポリシー・マップ・アクセスリストの名前は、表に列が無いのでパックがトンネルの番号から決めます。取り込んだ元のコンフィグの名前は引き継ぎません。ACL の表に `sec-<数>` という名前のリストがあると止まります |
| IPsec で書けない値 | 自分側アドレス、トンネル IF 型の行の自分側 IF（値があれば G-602）、版 1 の DH グループ dh15 以上・暗号 GCM、版 2 の DH グループ dh16 以上（語の対応がないエラー G-603）。版 2 の AH ハッシュ・有効量（キロバイト）・IKE のモード、版 1 の PRF・相手を確かめる鍵・「いつもつないでおく」（値があれば G-602）。「予備の相手」（Cisco の 2 つ目以降の `set peer`。この機種のマップ・IKE のポリシーは相手を 1 つだけ持ち、当たる書き方を資料で見つけられていません。値があれば G-602）。表「IPsec の全体設定」（Cisco の機器全体の行）と、「同じ NAT の後ろの端末を見分ける」「つなぎ直しは最初の相手から」も、この機種に当たる行が無く、値があれば G-602 です。方式 policy で IKE の版 2 の行、版 1 の行の「（追加）」の列は止まります |

### 置き換え（adapt.yaml。G-614 で知らせます）

- **相手監視の送り方**はこの機種では選べないので、違う値（版 1 の periodic、版 2 の on_demand）は、機種の送り方に置き換えます。**版 2 で再送間隔だけがある行**は、回数を機種の既定の 3 回として書きます
- **版 2 の行の追加の提案**（Cisco の 2 つ目以降の IKEv2 プロポーザル・トランスフォームセット）は、値を「（追加）」の並びに足して書きます（この機種の IKEv2 は、候補を値の並びで持ち、並べた値のどの組み合わせでも折り合うので、行に書いた組み合わせより広くなります）。有効時間・有効量・AH ハッシュのある行は足せず、止まります
- **トンネルの「宛先」と「送信元」（版 1）。** 「宛先」が IPsec の相手アドレスと同じなら書かず（知らせません）、「送信元」は書かずに知らせます（この機種の IKE 版 1 のトンネルは、どちらも持ちません）。トンネルの「宛先（名前）」が IPsec の相手の名前と同じなら、書きません（相手は `peer-fqdn-ipv4` に書きます。知らせません）
- **L2TP を守る IPsec の行の「自分側 IF」**（Cisco がクリプトマップを当てる IF）と、**UDP 1701 を選ぶ対象通信の行**は、知らせずに外します（この機種は、L2TP だけが通る受け口のトンネル IF に当て、リストは「すべて許可」で書きます）。ほかの対象通信の行は、外して知らせます

置き換えではありませんが、Cisco から書き換えたトンネル IF 型の行は「いつもつないでおく」が空です（Cisco に、その行が無いため）。NEC IX では「通信があったときにつなぐ」になります。いつもつないでおくなら、セルに有効を書いてください。

## L2TP

読める行は「対応一覧」の「L2TP（LNS）」「L2TP（LAC）」「PPPoE を受ける側（LAC の配下）」です（別紙 A 3.21）。**L2TPv3 は、この機器にありません**（FD の対応していない機能の一覧）。L2 をつなぐのは EtherIP です（「インタフェースとトンネル」「IPsec」の節）。見本は `tests/import/l2tp`、`tests/render` の `l2tp`・`l2tp_adapt`・`l2tp_failed`・`l2tp_unused` です。

### 読み方の決まり

- **L2TP の LNS は、PPP プロファイルが 1 行になります。** `tunnel mode l2tp-lns ipsec` のトンネル IF が `ppp binding` で指すプロファイルが行で、同じプロファイルを指すトンネル IF を「受け口の IF」に並べます（NEC IX は、同時につなぐ数だけトンネル IF を持ちます）。`ipsec policy transport` の無いトンネル IF は行を作りません（LNS は IPsec と組でしか動きません）。パスワード・無通信で切る時間・IPsec の設定名がトンネル IF によって違うと、後の IF の値を使い、取り込みの警告を出します
- **L2TP の LAC は、URL リストの名前が行の名前になります。** PPPoE を受ける IF の `pppoe l2tp interface <トンネル IF> url-list <名前>` から、トンネル IF（`tunnel mode l2tp-lac ip`・`tunnel destination`）と URL リストをたどります。たどれないトンネル IF・URL リストの行は「未対応」に戻ります。運ぶドメインは、リストの `permit domain *<名前>`（後ろが一致）の `*` を除いた名前です。Cisco の `domain <名前>` は接続 ID の `@` の後ろ全体と比べるので、意味は同じではありません
- **「トンネルの認証」は、`l2tp password` があれば有効、無ければ無効として読みます**（「確かめていないこと」）。`l2tp password secret <値>`（暗号化した形）は、「トンネルのパスワード」に `secret <値>` の形のまま入れます
- **L2TP を守る IPsec**（`ipsec dynamic-map`、相手が `any` の `ike policy`、`ike nat-traversal` など）は、IPsec の規則が読みます（「IPsec」の節）。L2TP の規則は、トンネル IF の下の `ipsec policy transport <名前>` の名前を読みます。IKE のポリシー・プロポーザルの行がそろっていない設定（見本の `tests/import/l2tp/input.txt`）では、動的なマップの行は「未対応」です

### 書き出し

| 項目 | 書き方 |
|---|---|
| L2TP（LNS） | `ppp profile <行の名前>` に `authentication request <認証>`（必須）、利用者ごとの `authentication password <接続 ID> <パスワード>`、`lcp pfc`・`lcp acfc`・`ipcp ip-compression`（いつも書きます）、利用者ごとの `ipcp provide-static-ip-address`、`ipcp provide-ip-address range`、`ipcp provide-remote-dns`（2 つまで）。受け口の IF（トンネル IF。インタフェースの表に行が要ります）ごとに `ppp binding <行の名前>`、`tunnel mode l2tp-lns ipsec`、`ipsec policy transport <IPsec の設定名>`（**必須**。空なら G-606）、`l2tp password plain`（`secret <値>` の形はそのまま `l2tp password secret <値>`）、`l2tp session idle-time` |
| L2TP（LAC） | `url-list <行の名前> permit domain *<ドメイン>`（運ぶドメインの数だけ。必須）。受け口の IF（トンネル IF を 1 つ）の下に `tunnel mode l2tp-lac ip`、`tunnel destination <アドレス>`（アドレスの形でなければ `tunnel destination fqdn <名前>`）、`l2tp password plain`、`l2tp session idle-time`。PPPoE を受ける IF（必須）の下に `pppoe l2tp interface <トンネル IF> url-list <行の名前>`。「利用者の認証」があれば、その IF を PPPoE サーバにする行（`encapsulation pppoe`・`auto-connect`・`pppoe server`・`ppp binding <名前>-pppoe`）と、そのプロファイル（`authentication request <認証>`）も書きます。空なら書きません（追加コンフィグに書きます） |
| L2TP で書けない値 | L2 延伸（L2TPv3）の行は、理由を付けて止まります（機器にありません）。利用者の認証 `mschap_v2`（G-603）、「トンネルの認証」が有効で「トンネルのパスワード」が空の行、31 字を超える名前・パスワードも止まります。「相手の名前」「自分の名前」は書く行が無く、値があれば G-602 |

### 置き換え（adapt.yaml。G-614 で知らせます）

- LNS の受け口の IF が 1 つのときは、同時に 1 台しかつなげないことを知らせます（値は変えません）

## 別メーカーのスイッチ（NEC QX）のデータを書くとき

NEC QX のパック（`nec_qx`）のデータにある列（別紙 A 3.4・3.22・3.24）を、NEC IX で書くときの決まりです（`adapt.yaml` の `sw.stp.qx`・`user.services` など。置き換えは G-614 で知らせます。試験は `tests/render/other_switch`）。

| 元のデータ | NEC IX での書き方 |
|---|---|
| 「スパニングツリーを使う」「優先度（機器全体）」 | 書きません（NEC IX にスパニングツリーはありません） |
| ユーザの「使える接続方式」 | 書きません（ユーザごとに方式を決める行がありません） |
| VLAN の「説明」 | VLAN の定義ごと書きません（「VLAN とスイッチ」の節） |
| 機器が保存した形の値（SNMP コミュニティ・回線パスワードの `cipher …`・`hash …`）、「パスワードの形」が encrypted の特権パスワード、OSPF の認証 hmac_md5、スイッチポートのモード hybrid | エラー（G-601）で止めます |
| 別紙 A 3.24 のほかの列（SNMP ユーザ・NTP の認証・MSTP・ポートミラーなど） | 書きません（G-602） |

## 読まない行（「未対応」として画面に出ます）

対応一覧に無い行は、「未対応」として画面に出ます（黙って落としません）。分野ごとに、分かっているものと理由を挙げます。**「機器に機能がありません」と書いたものは、この機種では書き出しもできません。** IPv6 の行は「IPv6」の節、ルートマップ・BGP・OSPF の細かい行は「経路制御・BGP・OSPF の残り」の節の「読まない行」にあります。

### 機器の基本

| 行 | 理由 |
|---|---|
| `logging subsystem <対象> include\|exclude <番号> …` | メッセージの番号を選ぶ形は、入れる列がありません |
| VRF の付いた行と付かない行が混ざった `ip name-server` | DNS サーバの VRF は 1 つです。先に読んだ行と違う行が「未対応」になります |
| `ntp server … priority <数>` など、`retry`・`version`・`timeout`・`priority` の付いた形 | 入れる列がありません |
| `syslog ip host … port <番号>` など、オプションの付いた形 | 入れる列がありません |
| `snmp-agent ip community … view …` | 入れる列がありません |
| `service password-encryption` を入れた機器が表示する、暗号化した形の秘密（PPP・SNMP など） | 資料で形を確かめられたのは `l2tp password secret <値>` と、IPsec の鍵の `key-type secret 1`・`key secret <値>` だけです（「確かめていないこと」） |

### インタフェースとトンネル

| 行 | 理由 |
|---|---|
| インタフェースの `mtu` | 入れる列に当たる行を確かめていません（`ip mtu` は読みます） |
| `no ip proxy-arp` | 既定と、止める行の出方を確かめていません |
| `tunnel mode gre ipsec` などの組み合わせのモード | 入れる列がありません |
| `tunnel mode` の `hostname <名前>` の付いた形 | 何の名前かの説明が資料にありません（MAP-E の `hostname` は読みます。「IPv6」の節） |

### VLAN・内蔵スイッチ・ブリッジ

| 行 | 理由 |
|---|---|
| `device` の下の `connector-type`・`ifspeed-change`・`shutdown`・`keepalive`・`output-buffer`・`igmp-snooping …`・`sflow …` | 入れる列がありません（`device` の `shutdown` は、別紙 A 3.17 のとおり） |
| `bridge-group <番号> permanent-address <MAC>` | 入れる列がありません（MAC アドレスを 1 つ持つだけの列を足していません） |
| IF の下の `bridge ip filter …`・`bridge ipv6 filter …`・`bridge ip tcp adjust-mss`・`bridge ipv6 tcp adjust-mss` | 入れる列がありません（ブリッジのフィルタと MSS は、表にしていません） |
| `system interfaces tunnel\|loopback\|null <数>` | 入れる列がありません |
| LLDP・CDP の行 | 機器に機能がありません（CRM に `lldp`・`cdp` のコマンドはありません。OpenFlow の設定の中の語だけです）。別メーカーのデータの CDP・LLDP は、外して知らせます |
| スパニングツリー、LACP | 機器に機能がありません（FD p.2-78、p.2-20）。別メーカーのデータは、「VLAN とスイッチ」の節の置き換えのとおりです |

### 経路

| 行 | 理由 |
|---|---|
| VRF の中の OSPF | 機器に機能がありません。機能説明書 2.17 の対応状況の一覧で、VRF の OSPF・RIP は「×」です。VRF で使える経路制御は、スタティックと BGP だけです |
| AS パスのリスト、コミュニティのリスト | 機器に機能がありません（コミュニティは、ルートマップの行にじかに書きます。AS パスで経路を見分ける設定はありません） |
| ルートマップの下の `match ip url-offload` | 入れる列がありません |
| 動作（`permit`・`deny`）か番号を省いた `route-map` の行 | 入力では省けますが、省いたときの扱いと表示を資料で確かめられていません。その下の行も「未対応」になります |
| ピアグループの `listen range` など、ルートマップ・BGP・OSPF のそのほかの行 | 「経路制御・BGP・OSPF の残り」の節の「読まない行」 |

### アクセスリスト・NAT

| 行 | 理由 |
|---|---|
| アクセスリストのアドレスをワイルドカードで書いた形（`src 192.0.2.0 0.0.0.255`）、`src-domain`・`dest-domain`、`linkmgr …` | ワイルドカードで入れた行を機器がどう表示するかを確かめていません。ドメイン名とリンクマネージャは、入れる列がありません |
| TCP のフラグを 2 つ以上並べた行 | 入力の形（CRM p.24-3）は `[FLAGS]` を 1 つだけ示しています |
| `ip access-list dynamic …`、`ip access-list cache …`・`option …` | ダイナミックフィルタと機器の調整の設定で、入れる表がありません |
| `ip filter … suppress-logging` | 入れる列がありません |
| 同じ名前の `ip nat pool` が 2 行以上（とびとびの範囲）と、それを使う `ip nat dynamic list` | 設計データのプールは、先頭と末尾の 1 組です |
| `ip napt service` の `icmp`・`any`・ポートの範囲（`ping`・`ftp`・`any` の予約名も）、`ip napt static` のプロトコル番号・ポートの範囲・IF 名の形、`ip napt address <IF>` | 設計データのポート転送は、tcp・udp のポート 1 つです |
| `ip nat static network` の、内側と外側で長さが違う形 | 設計データは長さを 1 つ持ちます |

### 回線監視

| 行 | 理由 |
|---|---|
| 番号が違う event と `sub` の付いた event が混ざった監視グループ | 「どれか」と「全部」の組み合わせは、列で表せません。グループの行がすべて「未対応」に戻ります |
| `event … ip reach-host`・`reach-route`（届くようになったら動く）、`vr-active`・`vr-inactive`、`watch-group-status`、`always`、`interface-up`、`ipv6 reach-host`・`ipv6 reach-route`、`ngn …`、`nm …` | 入れる列がありません |
| `action …` のうち、`ip shutdown-route`・`shutdown-interface`・`ip decrement-vrrp-priority` 以外（`ip shutdown-policy`・`ipsec clear-sa`・`invoke-watch-group`・`command-action-list` など）、`suppress-restoration` の付いた形 | 入れる列がありません |
| `probe-timer … msec <ミリ秒>`、`probe-timer wait`、`probe-counter watch`、`probe-counter … percent <率>`、`probe-size`、`probe-mode`、`suppress …` | 入れる列がありません（「間隔」は秒の整数です） |
| VR 番号に当たる VRRP の行が無い `action … ip decrement-vrrp-priority` | 入れる先の行がありません |

### PPPoE・QoS

| 行 | 理由 |
|---|---|
| `idle-time` の向きの語（`inbound`・`outbound`）の付いた形 | 入れる列がありません |
| PPP プロファイルの `authentication secret-password`（暗号化した形のパスワード） | 取り込みも書き出しもしていません |
| `authentication accept` と `authentication request` の方式が違う PPPoE のクライアントのプロファイル | 表の「認証方式」は 1 つです。PPPoE の行を作りません |
| QoS の `match access-list <MAC のリスト>`（ブリッジ専用） | 表にしていません |

### IPsec

- **ID の種類 `ipv6`、相手・ID が IPv6 の形**（`ikev2 authentication psk id ipv6 …`、`local-authentication psk id ipv6 …`、`ikev2 peer-fqdn-ipv6`、`ike policy` の IPv6 の `peer`・`peer-fqdn-ipv6`、`ike local-id … address <IPv6>`、`ipsec local-id`・`remote-id` の IPv6、`local-ts`・`remote-ts ipv6`、`ikev2 source-address <IPv6>`）: 読まず、書きません。この機種の書き方はコマンドリファレンスで確かめられましたが（例: `ike policy ipolicy1 peer 2001:db8::111 key …`）、IPsec の表は相手・対象通信を IPv4 で持ち、もう片方のメーカー（Cisco）の IPv6 の鍵の行の書き方が公開資料でそろわず、往復を作れませんでした（Cisco のパックの README）
- **IKEv2 の証明書・EAP**（`local-authentication rsa｜eap-md5`、`ikev2 authentication eap-md5 …`、`ikev2 peer … authentication rsa`）: 表にしません。証明書は、機器の中で鍵を作って認証局に登録して入れるもので、コンフィグの行を表に写しても別の機器では動きません。EAP は利用者の確かめの設定で、相手の機器の種類に左右されます。このパックが扱う認証は、事前共有鍵だけです（IKE の版 1 は、機器も事前共有鍵だけです。CRM p.13-43）
- **データコネクト（NGN）の相手**（`ike policy … peer ngn-dynamic`、`ikev2 peer-ngn-dynamic`）と、そのための設定: 相手のアドレスを NTT の網のサービスから取る、回線のサービスに固有の決め方で、ほかのメーカーに当たるものが無く、入れる列がありません
- **ダイナミック VPN**（`tunnel mode mgre ipsec-ikev2`、`nhrp nhs`、`nhrp local` など。FD 2.39）: 表にしません。「IX と WA のみ接続可能です。それ以外の装置とは接続できません」（FD 2.39.1 の「制限事項」）とあり、Cisco の DMVPN とつながらず、設定の考え方も違います（Cisco のパックの README に、表にするならの案）
- EtherIP 以外で使う `ipsec policy transport <マップ> with-id-payload`: 入れる列がありません（`with-id-payload` を読むのは、EtherIP のトンネルの下だけです。メーカーが「EtherIP 使用時は設定してください」と書いている形で、列にせず、その形の行にいつも書きます）
- `ipsec policy` の `in`・`df-bit`・`without-id-payload`・`pre-fragment`、`ipsec autokey-map`・`ipsec dynamic-map` の `esp-level`・`ah-level`、`ipsec anti-replay <マップ>`・`anti-replay off`: 入れる列がありません。`df-bit` と再送攻撃の検出は、Cisco では機器全体に 1 つの設定（表「IPsec の全体設定」）で、この機種の行ごと・マップごとの指定と粒度が合いません。`esp-level`・`ah-level`（守られていない受信を捨てるかどうか）と `in` は、Cisco に当たる行がありません。こうした語の付いた行は「未対応」に残り、そのマップの IPsec の行も作りません
- 方式 policy のマップに付いた `ipsec local-id`・`ipsec remote-id`、**`/32` と書いた `ipsec local-id`・`remote-id`**（相手へ送る ID の種類がアドレス 1 つの形と違います＝コマンドリファレンスの `ipsec local-id` のノート。対象通信の行は `/32` を 1 つの形でしか持てず、書き戻すと種類が変わります）、**アドレスの範囲・ポートの範囲で書いた `local-ts`・`remote-ts`**（対象通信の行はネットワークと 1 つのポートで持ちます）
- IKEv2 の `outgoing-interface`、`anti-replay`、`negotiation-direction`、`initial-contact`、`nonce-size`、`suppress`、`ikev2 ipsec mtu ignore`・`pre-fragment`、`ipsec source-address`、`ike commit-bit`、`ike suppress-dangling`、`ike initial-contact`、`ike rekey …`、`ipsec commit-bit`、`ipsec rekey …` などの細かい調整: 入れる列がありません
- `ike policy`・`ipsec autokey-map`・`ipsec dynamic-map` に、機器の上限（IKE 4 つ・IPsec 8 つ）を超えてプロポーザルを並べた形
- **相手が any の IKE ポリシーが 2 つ以上ある機器の、IKE ポリシーを指さない `ipsec dynamic-map`**: 行にしません。`ike <ポリシー>` も `ike-binding <ポリシー>` も無いマップは、どの IKE ポリシーともつながっていません（コマンドリファレンスの `ipsec dynamic-map`: `ike` は「指定 IKE ポリシによる IPsec の Rekey を開始を許可」、`ike-binding` は「ネゴシエーションを指定 IKE ポリシに固定」）。相手が any のポリシーが 1 つなら、その鍵と ID の行として読みますが、2 つ以上あると、どのポリシーで確かめた相手もそのマップを使えるので、鍵と ID を 1 組しか持てない行では表せません。マップと、どの行にも使われなかった IKE ポリシーの行が「未対応」に残ります（マップに `ike <ポリシー>` を書けば読めます）
- 相手監視の無い行の `retransmit`（相手監視があるときだけ読みます）

### L2TP

- **L2TPv3: 機器に機能がありません**（FD の対応していない機能の一覧）
- 機器全体の `l2tp hello-interval`・`l2tp retransmit`・`l2tp history max-records`
- `url-list` の `deny`・`ip`・`any` の行と、`*` が頭以外にあるドメイン（LAC が使うリストでも、その行だけ「未対応」です）
- PPP プロファイルの `authentication request … callin`、`authentication list`（AAA）、`ipcp provide-ip-address <アドレス>`（範囲でない形）、アドレスなしの `ipcp provide-remote-dns`
- `pppoe access-concentrator`、LAC を通さずに自分で PPPoE を終える PPPoE サーバ（利用者とアドレスの払い出しを持つプロファイル）
- PPPoE のプロファイルにある `lcp pfc`・`lcp acfc`・`ipcp ip-compression`（LNS のプロファイルの行としてだけ読みます）

## 確かめていないこと

資料に決まりが書かれていないか、見つけられなかった点です。実機の出力が手に入ったら、ここから確かめます。IPv6 は「IPv6」の節、ルートマップ・BGP・OSPF の細かい行は「経路制御・BGP・OSPF の残り」の節の「確かめていないこと」にあります。

### 全体と端末ログ

- running-config 全体でのブロックの並び順と、先頭行の並び
- 字下げの桁数（資料によって 1〜4 桁とばらつきます。何桁でも読めます）
- 末尾に終わりの印があるかどうか（無いものとして扱っています）
- 1 行に並ぶオプションの順（`ike proposal`、`ipsec autokey-map` など。コマンドリファレンスの構文の順で読んでいます）
- 端末ログ: プロンプトの形（`ホスト名(config)#`）と、ページ送りの跡の字（`--More--` か `--more--` か）。検索結果の要約で見ただけです
- 端末ログ: `show running-config` の短い形（どこまで縮められるか）。読んでいません
- 端末ログ: 機器がページ送りの跡をどう消すか。制御文字が残っているログも、残っていないログも読めるようにしてありますが、実機のログでは確かめていません
- 端末ログ: 行ごとに時刻が付くログ（端末ソフトの設定）は読めません
- 書き出した文面を実機に入れて確かめていません。とくに、節の順（ユーザ・SNMP・NTP・syslog・Telnet / SSH サーバの位置）と、ブロックの中の行の順

### 機器の基本

- この機種のハッシュの形（`$` で始まる値を断るだけで、ほかの値は確かめずに書きます）
- `username … password plain` の後ろに付けられる「モード」の語（パスワードがその語と同じだと、読み違えられるおそれがあります）
- 空白を含む `snmp-agent location` が、そのまま入るか
- SNMP の通知に `snmp-agent ip trap …` の行が要るか（書いていません）
- `ssh-server ip enable` の前に要る鍵の生成（設定の行ではないので、書いていません）
- `timezone` の時の範囲と、分の書き方（`+05 30` の形は、`+09 00` の観察からの類推です）
- `service password-encryption` を入れた機器が、PPP・IKE・SNMP などの秘密をどの形で表示するか。資料で形を確かめられたのは `l2tp password secret <値>` と IPsec の鍵の形だけなので、ほかの秘密の暗号化した形の行は「未対応」のままです（`authentication password … secret …` のような形があるかどうかも分かりません）
- `logging buffered`・`ip ufs-cache enable`・`ntp ip enable` を止める行（`no …`）が、保存した文面に出るかどうか
- `logging …`・`service password-encryption`・`ip ufs-cache enable`・`ntp ip enable`・`syslog ip source` が、保存した文面のどこに出るか（取り込みは順番によりません）

### インタフェースとトンネル

- Loopback と Null が、`no shutdown` の行なしで「有効」であること（どちらの行も出ないという観察からの読みです。資料の文ではありません）
- `ip proxy-arp` の既定（既定値は入れていません）
- `ipv6 address` だけで IPv6 が働くか（`ipv6 enable` は、表に値があるときだけ書きます。「IPv6」の節）
- IPv6 の網を通る EtherIP の `tunnel source` に書けるもの（IPv6 のアドレスか IF 名を、表の値のまま書いています）

### VLAN・内蔵スイッチ・ブリッジ

- `show running-config` の丸ごとの出力の中で、`vlan-group`・`port …` の行がどう並ぶか。資料に、`device` のブロックを含む running-config の出力はありません。設定例は、どれも `device` の下に字下げして書いています
- `encapsulation dot1q <番号>` が、保存された設定で `tpid 8100` 付きで出るか。資料の例には両方の形があります。取り込みは両方を読み、TPID が空の行は語を付けずに書きます
- BVI の既定の有効・無効。資料の例に `no shutdown` があることと、インタフェースの既定が停止であること（CRM p.3-6）から、Ethernet と同じ扱いにしています
- 別々の VLAN グループのタグ付きのサブ IF どうし（`GigaEthernet2:3.1` と `GigaEthernet2:4.1`）を、アクセスポートも BVI も無しに 1 つのブリッジグループに入れる形（通り抜ける VLAN）そのものの設定例は、資料にありません。資料にあるのは、L3 のポートのタグ付きのサブ IF どうしの透過ブリッジ（FD の 2.9.6 節。BVI なし）、ポート VLAN の基本 IF どうしのブリッジ（FD 2.9.2。BVI なし）、トランクとアクセスの組（FD p.2-99）で、パックの書き方はこの組み合わせです。同じ内蔵スイッチのポート VLAN どうしのブリッジは、内蔵スイッチが 1 Gbps の機種でだけ働きます（FD 2.9.2）
- ブリッジグループあたりの IF の数（機種で 10〜200）、ブリッジグループの数（255。機種によって 1000）の上限（FD の上限の表）。パックは確かめません
- リンクアグリゲーションとミラーの制約（同じ LAG のポートは同じ VLAN グループ、LAG のポートはミラーに使えない、ミラーの出力先はほかの通信に使えない、機種によって写し取るポートは 1 つ。CRM p.3-11〜3-12、FD p.2-19〜2-20）。パックは確かめません
- VLAN グループのサブ IF の上限が「8」であること。FD の上限の表と FAQ は 8、FD の VLAN の節は「VLAN グループが無いときの上限を、SW-HUB のポートの数で等分したもの」と書いています。パックは 8 で確かめます

### 経路

- `ip route` の `connected`・`metric`・`tag` が、既定値と同じ値のときに表示されるか（`metric 1`・`tag 0`）
- `ip route` で次ホップと IF の両方を書いたときの順（構文の 1 つ目の形で書いています）
- `ip router ospf`・`router bgp`・`vrrp enable` が running-config のどこに出るか（経路の後ろ、`vrrp enable` はインタフェースの前に書いています）
- `address-family ipv4 unicast` のブロックの閉じ方（何も書いていません。資料の例では、空白と `!` の行で閉じています）
- `vrf <VRF>` のブロックが、`router bgp` の下のどの位置に出るか（VRF が空の行の `address-family` の後ろに書いています。どこにあっても読めます）
- `no vrrp <番号> preempt` が running-config に出るか（既定が有効なので、無効にした機器では出るはずです）
- Loopback に `ip ospf hello-interval` などを書いてよいか（Hello と Dead が「明示」の列なので、OSPF に入れた全部のインタフェースに書いています）
- ルートマップが running-config のどこに出るか（アクセスリスト・プレフィックスリストの後ろ、経路の前に書いています）と、ルートマップの下の行の並び（どの順でも読めます）
- `route-map` の動作や番号を省いて入れたときに、機器が何を補って表示するか（省いた行は読みません。書き出しは必ず両方を書きます）
- 1 つのルートマップの行に、`match ip address prefix-list` と `match ip address access-list` の両方を書けるか（書き出しでは、両方がある行を止めています）
- `match community`・`set community` の値が、`AA:NN` と 1 つの数のどちらで表示されるか（どちらも読めます）
- `no-export-subconfed` が、Cisco の `local-as` と同じものであること。CRM は NO_EXPORT_SUBCONFED（0xFFFFFF03）と書き、Cisco のコマンドリファレンスは `local-as` を「自分の AS の中（コンフェデレーションの自分の小さい AS の中）にだけ広告する」と説明しています。説明が同じなので、同じ語として扱っています（Cisco の資料に、値そのものは書かれていません）
- コミュニティを相手に送るかどうかを決める行（見つかっていません。機能説明書の属性の一覧には、COMMUNITY は「オプション通過」とあります）

### アクセスリスト・NAT

- **`sequence-mode` の行と番号付きの行の順番。** 書き出しは、`sequence-mode` の行を先に、番号付きの行を後に書きます。コマンドリファレンスは「既存の設定に番号を付ける」と説明しており、空のリストに先に入れられるかどうかは、資料では確かめられていません。機器が running-config に出す順番も確かめていません
- アクセスリストのポートの範囲（`range`）・`established`・ICMP の種類の、running-config での出方（コマンドリファレンスの構文のとおりに読み書きしています）
- **`sport any`・`dport any` と、TCP のフラグを複数並べられるか。** 入力の形（CRM p.24-3）は `[FLAGS]` を 1 つだけ示しています。複数を並べた行は読みません
- 標準 ACL に当たる書き方を `permit ip src … dest any` と読み替えてよいか（自動では読み替えず、人に書き直してもらっています）
- `ip filter` の番号を 1 から振ってよいか（番号の範囲を確かめていません）
- **プロトコルの番号の 51（AH）・89（OSPF）** は、NEC の資料に名前との対応が出てこないので、語に直していません
- `ip nat static` を書くインタフェースが「外側」であること
- インタフェースのブロックの中で、NAT の行がどこに出るか（アドレスの行の後ろ、`shutdown` の前に書いています）
- `ip napt static` が、`ip napt enable` と同じインタフェースのアドレスで受けること（コマンドリファレンスの説明から、そう読んでいます）
- **`ip napt enable` と `outside` 付きのリストだけがある IF。** 「2 つ目以降の NAPT に使う」とあるので、`ip napt enable` の行そのものは「IF のアドレスへまとめる」行（pat_interface）として読みます。1 つ目の NAPT を持たない設定での動きは確かめていません
- **`ip napt service` と `ip napt static` の違い。** 内側と外側のポートが同じ `ip napt service` の行は、書き出しでは `ip napt static` になります（どちらも「外から来たそのポートを、内側のアドレスへ送る」設定として扱っています。細かい動きの違いは確かめていません）

### 回線監視

- `watch-group`・`network-monitor` が running-config のどこに出るか（経路の後ろに書いています）。ウォッチグループの名前に使える字と長さ
- 動作（action）のない `watch-group`、数字だけの名前（Cisco から持ってきた番号）を機器が受け付けるか
- VRF の経路を止める行（確かめていないので、VRF の経路の回線監視は書き出しで止めています）

### PPPoE・QoS

- PPP プロファイルと PPPoE のサブインタフェースが、running-config でどの位置・どの並びで出るか（設定事例集の入力の順と FAQ の字下げから書いています）
- `lcp echo-interval` を既定値（30 秒）と同じ値で入れたときに、running-config に出るか
- PPPoE サーバの IF に `encapsulation pppoe`・`auto-connect` が要るかどうか（サブ IF の既定のはずですが、FD の設定例には書いてあるので、PPPoE サーバでは書いています）
- `match rtp port` の 2 つ目の数（幅）の意味の細目、`match ngn`・`default-policy-map-ngn` を使うときにほかに要る設定（NGN の機能の設定は、表にしていません）

### IPsec

- IPsec の行（`ike proposal` など）が running-config でどの位置に出るか（回線監視の後ろに書いています）。`ike proposal` の `authentication pre-shared` は書いていません（設定事例集の例に無く、選べる値が 1 つだけです）
- `ipsec dynamic-map` の `ike`・`ike-binding` とプロポーザルの順（コマンドリファレンスの構文は IKE のポリシーが先、設定事例集の例はプロポーザルが先です。どちらも読み、構文の順で書いています）
- 1 つのインタフェースに `ipsec policy` を 2 つ以上書けるか（書けるものとして、行ごとに 1 行ずつ書いています。機能説明書の例は 1 つです）。方式 policy のマップのリストに複数の行を書いたときの動き（行ごとに SA を張るものとして、対象通信の行をそのまま並べています）
- L2TP の受け口を守る動的なマップに `ike <ポリシー>` を付けたときの動き（メーカーの設定例に合わせて、付けずに書いています）
- `ipsec local-id`・`ipsec remote-id` の行が running-config でどの位置に出るか（マップの行のすぐ後ろに書いています）。`local-ts`・`remote-ts` は「イニシエータ動作時のみ有効」（CRM）で、受ける側の動きには効きません
- IKEv2 のプロファイルに値を並べたとき、running-config が並びをどの順で表示するか（書いた順のまま、と読んでいます）
- `ipsec autokey-map` の `pfs` を書かないときの既定（既定値として入れていません）
- `ike policy` の `key-type` の位置。コマンドリファレンスの構文の行は鍵の後ろ、使用例は鍵の前です（どちらも読み、後ろに書いています）
- `key-type secret 1` が running-config でどう表示されるか（コマンドリファレンスの構文は `key <鍵> [key-type char | hex | secret { 0 | 1 }]` で、暗号化した鍵の表示の例がありません）
- IKEv2 の行（`ikev2 authentication`、`ikev2 profile` など）が running-config でどの位置・どの順に出るか（鍵 → IKE 版 1 の行とプロファイルを行の順、と書いています。プロファイルの中は、機能説明書の例に合わせて英字の順です）
- `sa-proposal integrity`・`prf`・`child-proposal integrity` の SHA-1 の語。コマンドリファレンスの構文の行と使用例は `sha1`、パラメータの説明は `sha` です（`sha1` を読み書きします）
- `nat-traversal` の `keepalive` と `force` の順（構文の行は keepalive が先、使用例は force が先。どちらも読み、keepalive を先に書いています）
- `retransmit` は相手監視だけでなく、IKEv2 のやり取り全体の再送の設定です（相手監視の再送は、これに従います。CRM p.13-57）
- IKEv2 で自分の ID を書かないときに、機器が鍵を引く ID（「送信元のアドレス」として、`id ipv4 <アドレス>` の鍵を書いています。コマンドリファレンスの既定値の記述からの読みです）
- 「有効量（キロバイト）」の範囲はメーカーで違います（NEC IX は 1000〜4000000）。範囲の外の値も、表のままを書きます
- `tunnel mode ether-ip ipsec` のトンネル IF の下の行の並び（モード → `ipsec policy transport` → `bridge-group` の順で書いています。FD の例は `no ip address` をはさみます）
- VRF の中のトンネル IF 型で、IKE の版 1（`tunnel mode ipsec` と `ip vrf forwarding`）が使えるか（FD 2.17.6 の例は版 2 で、「鍵交換プロトコルには IKEv2 を用い、センタ側の設定は peer any としてください」とあります。同じ相手と複数の VRF でつなぐ場合の注意と読み、版 1 を止めてはいません）

### L2TP

- `l2tp password` を書かないトンネルが相手を認証しないこと（CRM の既定値「なし」からの読みで、そう書いた文は見ていません。`defaults.yaml` の「トンネルの認証」はこの読みです）
- `url-list` が running-config のどこに出るか（PPP プロファイルの前に書いています）
- トンネル IF の下の行の並び（FD 2.36・2.37 の設定例の順ではなく、パックの決まった順で書いています）
- `tunnel mode` の `hostname <名前>` が何の名前か（CRM の入力形式にだけあり、説明がありません。読まず、書きません）
- `tunnel mode l2tp-lns ipsec` は設定の後に再起動が要ります（FD 2.36。投入の手順はこのツールの外です）

## 投入ログの読み方

コンフィグを流したときの端末ログから、機器に弾かれたコマンドを探します（仕様書 11.6）。機器の返事の形は `errors.yaml` にあります。

- 資料で確かめた「弾かれた」形は `% <語> -- Invalid command.` と、登録済みのユーザ名・ほかのユーザが設定モードにいる、の 3 つです。「受け付けた後の通知」は、再起動が要る旨・パスワードの強さ・ユーザの追加と削除・`% Warning:`・BGP の反映の案内です。ほかの `%` の返事は「パックが知らないメッセージ」として、人が確かめる指摘（L-701）になります。引数が足りない・値が範囲の外、というときのメッセージの形は、資料で確かめられていません
- プロンプトにかっこがある行（設定のモード）で打たれたコマンドを、生成したコンフィグの行と突き合わせます
