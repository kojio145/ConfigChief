# NEC UNIVERGE IX パック（nec_ix）

NEC UNIVERGE IX2000 / IX3000 シリーズの running-config を読み取り、基本の設定を書き出すための決まりです。

- 版: 0.2.0
- 取り込みは下の「対応一覧」の全部。**書き出しは基本の 9 表と、M3 の表の全部（VRF・OSPF・BGP・冗長化・回線監視・アクセスリスト・プレフィックスリスト・NAT）**です（「コンフィグの書き出し」の節）。M4 の表は、トンネル（GRE）・DHCP・PPPoE・IPsec（トンネル IF 型）が書けます
- **実機の出力では確かめていません**。構文はすべて下の公開資料で確かめたものです。画面に出る順番や字下げの桁数は、資料に決まりが書かれていないため、どの形でも読めるように書いてあります

## 参照した資料

どれも 2026-11-14 に NEC の公開ページから入手しました。

| 略号 | 資料の名前 | 版 |
|---|---|---|
| CRM | IX2000/IX3000 シリーズ コマンドリファレンスマニュアル | ソフトウェア ver.10.11 対応、第1.1版（2026年4月） |
| FD | IX2000/IX3000 シリーズ 機能説明書 | ver.10.11 対応、第1.1版（2026年5月） |
| EX | IX2000/IX3000 シリーズ 設定事例集 | ver.10.11 対応、10.11a版（2026年5月） |
| WEB | NEC 公式 Web「コンソールでの初期設定」「FAQ」 | 版の表記なし |

構文は CRM で確かめ、`defaults.yaml` の各行に CRM のページ番号を書いてあります。

端末ログの形（プロンプトとページ送りの跡）だけは、資料の本文を読めていません。NEC の公式 Web の FAQ とトラブルシューティングのページを開こうとしましたが取得できず（2026-11-14）、検索結果に出た要約で見ただけです（下の「確かめていないこと」）。

## 確かめた機種と版

- 実機: なし
- 資料: ソフトウェア ver.10.11 のマニュアル。これより古い版で書き方が違うコマンドは、確かめていません

## 対応一覧

取り込める行の一覧です。このうち書き出せるのは、QoS・機器全体・ユーザ・リモート接続・NTP・syslog・SNMP・インタフェース・スタティックルートです（「コンフィグの書き出し」の節）。

| 分野 | 読める行 | 入る表 |
|---|---|---|
| 機器全体 | `hostname`、`timezone`、`ip name-server <アドレス>`、`snmp-agent location` / `contact` | devices |
| ユーザ | `username <名前> password hash <値> [administrator\|operator\|monitor]`、`username <名前> password plain <値> [administrator\|operator\|monitor]`（入力の形。書き出した文面を読み直すためのものです） | users |
| リモート接続 | `telnet-server ip enable`、`ssh-server ip enable`、`…-server ip access-list <名前>`、`terminal timeout <分>` | remote_access |
| NTP | `ntp [vrf <VRF>] server <アドレス> [source <IF>]`、`ntp source <IF>` | ntp_servers |
| syslog | `syslog [vrf <VRF>] ip host <アドレス>`、`syslog facility <名前>` | syslog_servers |
| SNMP | `snmp-agent ip enable`、`snmp-agent ip community <名前> [<リスト>] [ro\|rw]`、`snmp-agent ip host <アドレス> <コミュニティ> [version <1\|2>]`、`snmp-agent ip trap-source <IF>` | snmp_communities、snmp_hosts |
| ポート | `device <名前>` の下の `speed`、`duplex` | interfaces（`<名前>.0` の行） |
| インタフェース | `description`、`ip address <アドレス>/<長さ> [secondary]`、`ip address dhcp` / `ipcp`、`ip unnumbered <IF>`、`no ip address`、`ipv6 address`、`shutdown` / `no shutdown`、`ip vrf forwarding`、`encapsulation dot1q <番号>`、`ip mtu`、`ip tcp adjust-mss <数>`、`ip filter <リスト> <番号> in\|out`、`ip proxy-arp`、`ip dhcp-relay server <アドレス>` | interfaces、vrfs |
| スタティックルート | `ip route [vrf <VRF>] <宛先>/<長さ>\|default <次ホップ>\|<IF> [distance <数>]` | static_routes |
| OSPF | `ip router ospf <番号>` の下の `router-id`、`network <アドレス>/<長さ>\|<IF> area <エリア>`、`area <エリア> stub\|nssa [no-summary]`、`area <エリア> range … advertise\|not-advertise`、`redistribute`、`originate-default [always] [metric <数>]`、`passive-interface <IF>`。インタフェースの下の `ip ospf cost` / `priority` / `hello-interval` / `dead-interval` / `network` / `authentication` / `authentication-key` / `message-digest-key` | ospf_processes、ospf_interfaces、ospf_areas、ospf_ranges、ospf_redistribute |
| BGP | `router bgp <AS>` の下の `router-id`、`neighbor <アドレス> remote-as` / `update-source` / `description` / `password` / `ebgp-multihop` / `timers` / `shutdown`。`address-family ipv4 unicast` の下の `network`、`redistribute`、`neighbor <アドレス> distribute-list <名前> in\|out` / `next-hop-self` / `send-default` | bgp_processes、bgp_neighbors、bgp_networks、bgp_redistribute |
| VRRP | `vrrp enable`、インタフェースの下の `vrrp <番号> ip` / `priority` / `preempt` / `timers advertisement <秒>` / `authentication` | fhrp |
| アクセスリスト | `ip access-list <名前> [<番号>] permit\|deny ip\|tcp\|udp\|icmp src … [sport …] dest … [dport …]`（ポートは eq / lt / gt / neq と数字 1 つ） | acls、acl_entries |
| プレフィックスリスト | `ip prefix-list <名前> <番号> permit\|deny <アドレス>/<長さ> [min <数>] [max <数>]` | prefix_lists |
| NAT | インタフェースの下の `ip napt enable`、`ip napt inside list <リスト>`、`ip napt static <内側アドレス> <プロトコル> <ポート>`、`ip nat enable`、`ip nat static <内側アドレス> <外側アドレス>` | nat_rules |
| 回線監視 | `watch-group <名前>` の下の `event <番号> ip unreach-host <アドレス> [source <IF>]`、`event <番号> interface-down <IF>`、`probe-counter variance\|restorer <回数>`、`probe-timer variance <秒>`、`action <番号> ip shutdown-route <宛先>/<長さ> [<次ホップ>]`。`network-monitor <名前> enable` | tracks、static_routes の track 列 |
| PPPoE | `ppp profile <名前>` の下の `authentication myname` / `password` / `accept`、`lcp echo-interval <秒>`。インタフェースの下の `ppp binding <名前>`、`pppoe service-name <名前>` | pppoe |
| IPsec（IKEv1・トンネル IF 型） | `ike proposal`、`ike policy <名前> peer <アドレス> key <鍵> [mode …] [<プロポーザル>]`、`ike keepalive <ポリシー> <秒> [<回数>]`、`ike nat-traversal`、`ipsec autokey-proposal`、`ipsec autokey-map <名前> <リスト> peer <アドレス> [pfs …] [<プロポーザル>]`。インタフェースの下の `tunnel mode ipsec`、`ipsec policy tunnel <マップ>` | ipsec_tunnels、tunnels |
| トンネル（GRE） | インタフェースの下の `tunnel mode gre`、`tunnel source`、`tunnel destination`、`tunnel key`、`tunnel keepalive [period <秒>] [retries <回数>]` | tunnels |
| DHCP サーバ | `ip dhcp enable`、`ip dhcp profile <名前>` の下の `assignable-range`、`subnet-mask`、`default-gateway`、`dns-server`、`domain-name`、`lease-time <秒>`。`ip dhcp excluded-address <開始> <終了>`、インタフェースの下の `ip dhcp binding <名前>` | dhcp_pools |
| DHCP リレー | `ip dhcp-relay enable` | interfaces の dhcp_relay 列 |
| QoS（M6） | `class-map [match-all\|match-any] <名前>` の下の `match ip access-list <リスト> [high\|medium\|normal\|low]`、`match any [<キュー>]`。`policy-map <名前>` の下の `class <名前>\|class-default\|class-local` と、その下の `priority [[mbps] <kbps> [<バースト>]]`、`priority percent <%> [<バースト>]`、`bandwidth [mbps] <kbps>`、`bandwidth percent <%>`、`shape [kbps\|mbps] <bps> [<Bc> [<Be>]]`、`set ip dscp`、`set ip precedence`、`queue-limit <数> …`（4 つまで）。インタフェースの下の `service-policy input\|output <名前>`、`traffic-shape rate [kbps\|mbps] <bps> [<Bc> [<Be>]]`。アクセスリストの末尾の `precedence <数>`、`dscp <数>` | qos_classes、qos_class_matches、qos_policies、qos_bindings、acl_entries の dscp・precedence 列 |

## この機種ならではの読み方

- **ポートとインタフェースが別のブロックです。** `device GigaEthernet1` の速度と二重は、`GigaEthernet1.0` の行に入れます。`GigaEthernet1.1` 以降はサブインタフェースです
- **有効・無効の両方が画面に出ます。** `shutdown` も `no shutdown` も読みます。どちらも無いポート・サブインタフェース・トンネルは「停止」とみなします（既定が停止のため。CRM p.3-6）。Loopback と Null にはどちらの行も出ないので、「有効」とみなします（下の「確かめていないこと」を参照）
- **機能ごとに「有効にする行」があります。** `snmp-agent ip enable`、`vrrp enable`、`ip dhcp enable`、`ip dhcp-relay enable`、`network-monitor <名前> enable` が無いと、その機能の行は表に入らず「未対応」に戻ります
- **名前でつながる設定は、つながった先まで読めたときだけ表に入ります。** `ppp binding` と `ppp profile`、`ip dhcp binding` と `ip dhcp profile`、`ipsec policy tunnel` と `ipsec autokey-map` と `ike policy`（相手のアドレスが同じもの）がその例です。つながらない行は「未対応」に戻ります
- **PPPoE の行の名前は、PPP プロファイルの名前です。** 論理 IF は `ppp binding` を書いたサブインタフェース、物理 IF は同じポートの「ポート名.0」です。1 つのプロファイルを 2 つのインタフェースが使う設定は、1 つ目だけが行になり、注意が出ます
- **経路の切り替えは、監視の側に書かれています。** `watch-group` の `action … ip shutdown-route <宛先>` は、同じ宛先のスタティックルートの track 列に、その監視の名前を入れます
- **VRF は定義のブロックがありません。** `ip vrf forwarding <名前>` が出てきた時点で、vrfs の表に名前だけの行を作ります
- **アクセスリストの番号。** 番号の無い行は、行の順に 10、20、30 … と番号を付けます
- **時刻の帯。** `timezone +09 00` は `+9` として入れます（この機種は帯の名前を持ちません）
- **DHCP の配るネットワーク。** プロファイルに `subnet-mask` があればそこから、無ければ結び付けたインタフェースのアドレスから決めます
- **IPsec のトンネル IF 型。** マップが指すアクセスリストの 1 行目が「すべて許可（`permit ip src any dest any`）」のときだけ読みます。それ以外は「未対応」に戻ります。行の名前はトンネル IF の名前です。**その 1 行目は、アクセスリストの表（acl_entries）から外します。行が残らなければ、リスト（acls）も外します**（フィルタ・NAT・SNMP も使っているリストは、そのまま残します）。トンネル IF の下の行は `ipsec policy tunnel <マップ名>` と、後ろに `out` が付いた形の両方を読みます
- **QoS の DSCP・precedence。** この機種は DSCP・precedence をアクセスリストの末尾で見ます。クラスが指すリストのすべての行が `permit ip src any dest any dscp <数>`（または `precedence <数>`）だけのときは、その値を見分け方の条件（qos_class_matches の dscp・precedence 列）に入れ、**リストはアクセスリストの表から外します**（フィルタ・NAT・SNMP・IPsec も使っているリストは、リストのまま残します）。アドレスなどほかの条件があるリストも、リストのまま残します
- **QoS の既定のクラス。** `class-default` は `default`、`class-local` は `local` という名前で入れます。動作の行が無い `class` の行も、行として入れます。単位の語（`kbps`・`mbps`）は数に直します（`shape`・`traffic-shape rate` は bps、`priority`・`bandwidth` は kbps）
- **NAT の外側。** `ip napt …` と `ip nat …` は、その行が書かれたインタフェースを「外側」として入れます（下の「確かめていないこと」を参照）

## 端末ログの読み方

端末ソフトのログ（ログインから `show running-config` まで入ったもの）は、設定の部分だけを切り出して読みます。決まりは別紙 B 2.4、書いてある場所は `pack.yaml` の `session` です。

- **プロンプト。** 行の頭の「ホスト名」「かっこに入ったモード（あれば）」「`#` か `>`」です。この機種は設定のモード（`ホスト名(config)#`）で `show running-config` を打ちます
- **設定を表示するコマンド。** `show running-config` と `show startup-config` を、省かずに打った行だけです。短い形は確かめていないので読みません。短い形で取ったログは、全体をコンフィグとして読んだうえで「コマンドの行が見つかりません」と注意を出します
- **ページ送りの跡。** `--More--` です（大文字でも小文字でも読みます）。跡だけの行は「無視（ページ送りの跡）」にし、跡の後ろに設定が続く行は、跡を取り除いて読みます（そのときは注意を出します）
- 同じログにコマンドが何回もあるときは、最後のものを読みます

## 値を入れない規則

別紙 B 3.2 で認められている 2 つの場合だけです。

| 規則 | 行 | 理由 |
|---|---|---|
| `device` | `device <名前>` | 子の行を持つ入れ物の行 |
| `bgp.af` | `address-family ipv4 [unicast]` | 子の行を持つ入れ物の行 |
| `if.ipv4.none` | `no ip address` | 「ipv4 の列が空」と同じ意味の行 |
| `qos.policy` | `policy-map <名前>` | 子の行を持つ入れ物の行 |

## 読んで、表には入れない語

行は「取り込み済み」になりますが、次の語は入れる列が無いため捨てています。書き出すときは、決まった値に戻します（いま書き出せるのは `ip filter` だけで、番号は 1 にします）。

| 行 | 捨てる語 |
|---|---|
| `ip filter <リスト> <番号> in\|out` | 番号（1 方向に 1 リストだけ読むため） |
| `watch-group <名前> <番号>` | 番号 |
| `event <番号> …`、`action <番号> …` | 番号 |
| `redistribute bgp <AS>`（OSPF の下） | AS 番号 |
| `ike proposal … authentication pre-shared …` | `authentication pre-shared`（事前共有鍵だけを扱うため） |

## 「無視」にする行

`ignore.yaml` にあるのは、画面の表示だけの行です（`Current configuration : … bytes` など 3 種類）。インタフェースの下の `service-policy enable` は、書き出しのときにパックが書く行なので「無視（パックが書く）」にします（別紙 A 3.18）。`!` で始まる行は区切りとして数えます。

## 読まない行（「未対応」として画面に出ます）

試験の見本（`tests/import/basic/input.txt`）にある 13 行と、その理由です。

| 行 | 理由 |
|---|---|
| `logging buffered …`、`logging subsystem …`、`logging timestamp …` | 入れる列がありません |
| `ip ufs-cache enable` | 入れる列がありません |
| `ip access-list <名前> sequence-mode <間隔>` | リストに番号を付ける設定で、入れる列がありません |
| `ip access-list … dport range <開始> <終了>` | ポートの範囲は、まだ読んでいません |
| `ntp ip enable` | NTP サーバとして動かす設定で、入れる列がありません |
| `action <番号> ip decrement-vrrp-priority <VR 番号> <数>` | 2 つ目の数の意味（下げる幅か、下げた後の値か）を資料で確かめられていません |
| `probe-timer restorer <秒>` | 復旧を待つ間の間隔で、入れる列がありません |
| `device` の下の `description` | ポートの説明で、入れる列がありません（インタフェースの説明とは別です） |
| `ipv6 enable` | 入れる列がありません |
| 同じ向きの 2 つ目の `ip filter` | 1 方向に 1 つのリストしか入れられません |
| `ip tcp adjust-mss auto` | 列は数だけを受けます |

このほか、次の書き方も読みません。

- ルートマップ（`route-map` のブロック）
- IKEv2、`ike policy` の `key-type` 付き・プロポーザルを複数並べた形、`ike local-id` / `remote-id`、`ike nat-traversal` のオプション付き
- `ipsec autokey-proposal` の `ah-…`、バイト数での寿命
- `ip nat dynamic list … pool …`、`ip napt service …`
- `username <名前> password plain <モード> <値>` の形（`plain` の後ろにモードの語が付く形。語の種類を確かめていません）
- アクセスリストの `established`、ICMP の種類、`range` と `any` のポート指定
- `ip prefix-list … any`
- `event … ip unreach-host` で出口のインタフェースや次ホップを付けた形、`event … ip unreach-route`、`action … shutdown-interface`
- `ip route` の `connected`・`metric`・`tag`・`dhcp`
- インタフェースの `bandwidth`（単位を資料で確かめられていません）
- VRF の中の OSPF・BGP
- 1 つの監視グループに 2 つ目の `event`
- 3,600 秒で割り切れない `lease-time`、`lease-time infinite`
- QoS の `match` のうち `ipv6 access-list`・`input-interface`・`cos`・`qos-group`・`local-generate-packet`・`rtp port`・`ngn`、キューの `sub-a`〜`sub-d`、`class class-dynamic`、`set cos`・`set qos-group`・`set ipv6 …`、`service-policy output default-policy-map-ngn` は表にしないため、そのまま「未対応」になります。アクセスリストの `tos` も読みません

## コンフィグの書き出し

基本の 9 表（devices、interfaces、static_routes、users、remote_access、ntp_servers、syslog_servers、snmp_communities、snmp_hosts）と、VRF（vrfs）・回線監視（tracks）・アクセスリスト（acls、acl_entries）・プレフィックスリスト（prefix_lists）・NAT（nat_rules）、OSPF（ospf_processes、ospf_interfaces、ospf_areas、ospf_ranges、ospf_redistribute）・BGP（bgp_processes、bgp_neighbors、bgp_networks、bgp_redistribute）・冗長化（fhrp）、トンネル（tunnels）・DHCP（dhcp_pools）・PPPoE（pppoe）・IPsec（ipsec_tunnels。トンネル IF 型だけ）を書き出します。書き出した文面は、同じパックで取り込み直して設計データと比べます（往復の確認）。

節の順: `system`（ホスト名・時刻帯・ユーザ・DNS）→ `policy`（アクセスリスト・プレフィックスリスト。工場出荷時のコンフィグで、アクセスリストが先頭の近くにあるため）→ `routing` → `ospf` → `bgp` → `tracking`（回線監視。止める経路が先にあるように、経路の後ろ）→ `ipsec` → `extra`（追加コンフィグの既定の位置）→ `services`（SNMP・NTP・syslog）→ `remote`（Telnet / SSH サーバ・無操作タイムアウト）→ `pppoe`（PPP プロファイル）→ `dhcp`（DHCP サーバ）→ `interfaces`（`device` のブロック → `interface` のブロック）。終わりの印はありません。

- 経路が先で、`device` と `interface` のブロックが最後なのは、公式の例の並びに合わせたものです。ユーザやサービスの位置は資料で確かめられていません
- 追加コンフィグの既定の位置を `services` より前にしたのは、そこに書いたアクセスリストが、それを名前で指す行（SNMP コミュニティ・Telnet / SSH サーバ・`ip filter`）より先に来るようにするためです

書き方について決めたこと:

| 項目 | 書き方 |
|---|---|
| 時刻帯 | `+9` → `timezone +09 00`、`+5:30` → `timezone +05 30`。帯の名前は持てないので、`JST +9` のような名前付きの形はエラーにします（`+9` と書いてください） |
| ハッシュ化された秘密（ユーザ） | `username <名前> password hash <値>` と書きます。`$` で始まるハッシュは別メーカーの形なので、エラーにします（ハッシュは別メーカーへ引き継げません）。この機種のハッシュの形は資料に無いため、それ以外の値が本当にこの機種のハッシュかどうかは見分けられません |
| 平文の秘密（ユーザ） | `username <名前> password plain <値>` と書きます。形が空のセルは平文として扱います。空白を含む値はエラーにします |
| ユーザの権限 | `admin` → `administrator`、`operator` → `operator`、`monitor` → `monitor`。空なら語を付けません。数（`15` など）は、語の対応がないエラー（G-603）になります |
| リモート接続 | プロトコルごとに `telnet-server ip enable` / `ssh-server ip enable`、アクセスリストがあれば両方のサーバに同じ名前を書きます。無操作タイムアウトは `terminal timeout <分>`（機器に 1 つ） |
| インタフェースの有効・無効 | GigaEthernet（基本・サブ）と Tunnel には、有効なら `no shutdown`、無効なら `shutdown` を必ず書きます。Loopback と Null には、どちらも書きません（「無効」はエラー）。そのほかの種類（BVI、USB-Serial など）は、行の出方を確かめていないのでエラーにします |
| アドレス | `ip address <アドレス>/<長さ>`、`dhcp` → `ip address dhcp`、`negotiated` → `ip address ipcp`、`unnumbered:<IF>` → `ip unnumbered <IF>`。アドレスの無い IF に `no ip address` は書きません |
| VRF | `ip vrf forwarding` を、アドレスより前に書きます。この機種は VRF の定義のブロックを持たず、この行が VRF を作ります。そのため vrfs の表の「名前」はこの行で書いたものとして扱い、どのインタフェースにも使われていない VRF はエラーにします |
| 速度・二重 | ポートの設定なので、`device <ポート名>` のブロックに書きます。値は基本インタフェース（`GigaEthernet0.0` のように `.0` で終わる行）に書いてください。サブインタフェースやほかの種類の行にあるとエラーにします |
| VLAN のタグ | `encapsulation dot1q <番号>`。サブインタフェース（`.1` 以降）以外にあるとエラーにします |
| フィルタ | `ip filter <リスト> 1 in\|out`。番号は 1 に決めています |
| DHCP リレー | 転送先が 1 つでもあれば、機器全体の `ip dhcp-relay enable` を書きます |
| 代理 ARP | 「有効」は `ip proxy-arp`。「無効」は書けません（既定と、止める行の出方を確かめていないため、エラーにします。セルを空にしてください） |
| スタティックルート | `ip route [vrf <VRF>] <宛先>/<長さ> <次ホップ> [<IF>] [distance <数>]`。`0.0.0.0/0` は `default` と書きます |
| NTP | サーバは IPv4 アドレスだけです（名前・IPv6 はエラー）。送信元 IF が全部の行で同じなら `ntp source`、違えばサーバごとの `source` で書きます |
| syslog | 転送先は IPv4 アドレスだけです。ファシリティは機器に 1 つなので、全部の行で同じ値でなければエラーにします |
| SNMP | コミュニティか通知先があれば `snmp-agent ip enable` を書きます。コミュニティはアクセスリスト → `ro` / `rw` の順。通知先の版は `1` と `2c`（`version 2` と書きます）だけで、版 3 はエラーです。通知先のコミュニティは必須で、SNMP コミュニティの表に無い名前はエラーにします（先に定義が要るため。CRM p.22-15）。通知の送信元 IF は機器に 1 つです（そろっていなければエラー）。連絡先は空白なしの 1 語です |
| アクセスリスト | 1 行に 1 エントリ（`ip access-list <名前> permit｜deny <プロトコル> src … dest …`）。リストそのものの行はありません。書けるのは、取り込みが読める形だけです: プロトコルは ip・tcp・udp・icmp、アドレスは `any` か アドレス/長さ（ホストは /32）、ポートは番号 1 つ（`80` → `eq 80`、`>1023`、`<1024`、`!80`）。ポートの範囲・付加条件（established など）・ICMP の種類・ほかのプロトコルは、理由を出して止まります（追加コンフィグに書いてください） |
| アクセスリストの種類 | 機器の ACL は 1 種類なので、種類は extended だけです（空なら extended として扱います）。standard のリストは止まります。別メーカーから持ってきた標準 ACL は、種類を extended にし、各行にプロトコル ip と宛先 any を書いてください |
| アクセスリストの行の番号 | 番号は書きません（番号を持つのは `sequence-mode` のリストだけで、その書き出しは作っていません）。行が 10、20、30… と並んでいないリストは止まります（順番を振り直してください。評価の順は変わりません） |
| アクセスリストの注記・説明 | 注記（remark）の行は書けません（止まります）。説明の列も書けません（値があれば G-602） |
| 行のないアクセスリスト | 何も書きません（エントリを追加コンフィグに書くときのため）。名前が 15 字を超えるリストは止まります |
| プレフィックスリスト | `ip prefix-list <名前> <番号> …`、下限・上限は `min`・`max`。名前に大文字があると止まります（機器が小文字で保存するため） |
| 回線監視 | `watch-group <名前> 10` のブロックと、`network-monitor <名前> enable` を書きます。監視（ping）は `event 10 ip unreach-host <アドレス> [source <IF>]`、IF の監視は `event 10 interface-down <IF>`。監視の方法 `route` は書けません（止まります）。ping の出力インタフェースは書きません（省けるのは Ver.10.6 以降です） |
| 回線監視の回数と間隔 | 監視（ping）の行だけ、`probe-counter variance <断と判断する回数>`・`probe-counter restorer <復旧と判断する回数>`・`probe-timer variance <間隔>` を書きます。IF の監視の行にこれらの値があると G-602 になります |
| 回線監視で止める経路 | 「回線監視」の列がその監視を指しているスタティックルートを、`action <10 刻みの番号> ip shutdown-route <宛先>/<長さ> [<次ホップ>]` で書きます。VRF の経路は書けません（止まります） |
| NAT | 外側 IF のブロックの中に書きます。外側 IF は必須です（空なら止まります）。`pat_interface` → `ip napt enable`（対象の通信があれば `ip napt inside list <ACL>`）、`static_port` → `ip napt static <内側アドレス> <プロトコル> <ポート>`、`static` → `ip nat enable` と `ip nat static <内側> <外側>`。同じ IF では、`ip napt enable` → ポート転送 → 静的 NAT の順です |
| NAT で書けない形 | アドレスの範囲を使う種類（`pat_pool`・`dynamic_pool`）、内側ポートと外側ポートが違うポート転送、外側アドレスが `interface` でないポート転送、同じ外側 IF に `pat_interface` の行がないポート転送、同じ外側 IF に 2 つ目の `pat_interface`。どれも理由を付けて止まります |
| ポート転送の外側アドレス | 空なら `interface`（外側 IF のアドレス）として扱います（defaults.yaml） |
| OSPF | `ip router ospf <番号>` の下に、`router-id`、`area … stub｜nssa [no-summary]`、`area … range <範囲> advertise｜not-advertise`（「広告する」が空なら advertise）、`redistribute`、`originate-default`、`passive-interface <IF>`、`network <IF> area <エリア>` を書きます。インタフェースは名前で入れます（アドレスの形の `network` は書きません） |
| OSPF で書けない形 | 「既定を受動にする」（この機種にありません。止まります）、VRF の中のインタフェース（止まります）、1 つのインタフェースを 2 つのプロセスに入れること（止まります）、網の種類 `point_to_multipoint`（語の対応がないエラー G-603） |
| OSPF のインタフェース | `ip ospf cost`・`priority`・`hello-interval`・`dead-interval`・`network`・`authentication [message-digest｜null]`・`message-digest-key <鍵番号> <鍵>`（`md5` の語はありません）・`authentication-key <鍵>`。Hello と Dead は「明示」の列なので、空なら既定値（10・40）を書きます |
| BGP | `router bgp <AS>` の下に `router-id` と、ネイバーの `remote-as`・`update-source`・`description`・`password`・`ebgp-multihop`・`timers <Keepalive> <Hold>`（2 つは組）・`shutdown`。`address-family ipv4 unicast` の下に `network`・`redistribute`・ネイバーの `distribute-list <プレフィックスリスト> in｜out`・`route-map <名前> in｜out`・`next-hop-self`・`send-default`。下に書く行がなければ `address-family` の行も書きません |
| BGP で書けない形 | VRF の中の BGP（止まります）、65535 を超える AS 番号（止まります） |
| VRRP | 行が 1 つでもあれば `vrrp enable` を書きます。インタフェースの下に `vrrp <番号> ip`・`priority`・`preempt`（無効なら `no vrrp <番号> preempt`）・`timers advertisement <秒>`・`authentication <鍵>`。方式が hsrp の行、認証 md5 は止まります |
| トンネル | トンネル IF の下に `tunnel mode gre｜ipsec`。gre は `tunnel source`・`tunnel destination`（必須）・`tunnel key`・`tunnel keepalive period <間隔> retries <回数>`（2 つは組）。**ipsec は `ipsec policy tunnel <マップ名> out` だけを書き、送信元・宛先・鍵・監視は読みません**（NEC IX の IPsec のトンネルには、その行がありません。値があれば G-602。セルを空にしてください）。IPsec の表に、そのトンネル IF の行が要ります。種類 ipip・l2tpv3 は語の対応がないエラー（G-603） |
| DHCP | 行があれば `ip dhcp enable` を書きます。プールは `ip dhcp profile <名前>` で、そのネットワークにアドレスを持つインタフェース 1 つに `ip dhcp binding <名前>` で結び付けます（そういうインタフェースが無い・2 つ以上あると止まります）。配る範囲があれば `assignable-range` と `subnet-mask` を書き、無ければ書きません（ネットワークは、結び付けたインタフェースのアドレスから決まります） |
| DHCP の値 | `default-gateway`・`dns-server`（2 つまで。3 つ以上は止まります）・`domain-name`・`lease-time <秒>`（`2d` → 172800、`12h` → 43200）。配らないアドレスは `ip dhcp excluded-address <先頭> <末尾>` |
| PPPoE | 行ごとに `ppp profile <行の名前>` を書き、論理 IF の下に `ppp binding <行の名前>` を書きます。論理 IF は、ポートのサブインタフェース（「ポート名.1」以降）で、インタフェースの表に行が要ります。物理 IF は同じポートの「ポート名.0」でなければ止まります。名前は空白のない 31 字まで。`encapsulation pppoe` と `auto-connect` は書きません（サブインタフェースの既定です） |
| PPPoE の値 | `authentication myname <接続 ID>`・`authentication password <接続 ID> <パスワード>`（平文）、認証方式があれば `authentication accept chap｜pap｜chap-pap`、監視間隔があれば `lcp echo-interval <秒>`、サービス名があれば論理 IF の下に `pppoe service-name <名前>` |
| PPPoE のアドレス・MSS | 論理 IF のインタフェースの行に書きます（アドレス `negotiated` → `ip address ipcp`） |
| PPPoE で書けない列 | 「デフォルト経路を作る」（値があれば G-602。NEC IX に当たる行がありません。出力 IF を論理 IF にしたスタティックルートの行で書いてください → `ip route default <IF>`） |
| IPsec | 方式は tunnel_if だけ、IKE は版 1・事前共有鍵だけ（ほかは止まります）。トンネル IF `Tunnel<数>.0` の行ごとに、設定事例集の形で 6 行を書きます: `ip access-list sec-<数> permit ip src any dest any`、`ike proposal ike-prop-<数> encryption … hash … group … [lifetime <秒>]`、`ike policy ike-<数> peer <相手> key <鍵> [mode main｜aggressive] ike-prop-<数>`、`ike keepalive ike-<数> <秒> [<回数>]`（相手監視の間隔があるとき）、`ipsec autokey-proposal ipsec-prop-<数> esp-… esp-… [lifetime time <秒>]`、`ipsec autokey-map ipsec-<数> sec-<数> peer <相手> [pfs off｜<ビット数>] ipsec-prop-<数>`。トンネル IF の下に `ipsec policy tunnel ipsec-<数> out` |
| IPsec の NAT 越え | 機器に 1 つの設定（`ike nat-traversal`）です。全部の行が有効なら書き、全部が空なら書きません。行によって違う・「無効」と書いた行がある、は止まります |
| IPsec の名前 | プロポーザル・ポリシー・マップ・アクセスリストの名前は、表に列が無いのでパックがトンネルの番号から決めます。取り込んだ元のコンフィグの名前は引き継ぎません。ACL の表に `sec-<数>` という名前のリストがあると止まります |
| IPsec で書けない値 | 方式 policy（対象通信で選ぶ形。取り込みの規則もまだありません）、相手 `any`、自分の ID・相手の ID・自分側 IF・自分側アドレス（値があれば G-602）、DH グループ dh15 以上・暗号 GCM・ESP のハッシュ sha384 以上（語の対応がないエラー G-603） |
| 並び | 表の行は、キーの自然な順（`GigaEthernet…` は `Loopback…` より前）です |

この機種に当たる行が無い（または確かめられていない）ため、書けない列があります。値が入っていると「使われなかった値」（G-602、エラー）になります。

| 表 | 書けない列 | 理由 |
|---|---|---|
| devices | ドメイン名、特権パスワードとその形、バナー | 機器のドメイン名と特権パスワードに当たるコマンドがありません。バナーは保存した文面での形を確かめていません |
| interfaces | MTU、帯域、NAT の向き | `mtu` に当たるコマンドが見つかっていません（`ip mtu` は書けます）。`bandwidth` は単位を確かめていません。NAT は向きを書く形ではありません |
| static_routes | 説明 | 経路に名前は付けられません（回線監視は、`watch-group` の側に書きます） |
| ospf_processes | VRF、基準帯域 | プロセスを VRF に結ぶ行、基準帯域の行がありません |
| ospf_areas | 既定経路のコスト | 取り込みの規則がありません |
| bgp_processes | Keepalive、Hold、ネイバーの変化を記録 | 当たる行を確かめていません（Keepalive と Hold はネイバーごとに書けます） |
| bgp_neighbors | 受信経路の保持 | 当たる行を確かめていません |
| fhrp | 版、切り戻し待ち、Hold、回線監視、監視断のときの減点 | VRRP の行にありません。回線監視との連動は `watch-group` の `action … ip decrement-vrrp-priority` ですが、2 つ目の数の意味を確かめていないので書いていません（追加コンフィグに書いてください） |
| tracks | 待ち時間、断・復旧の反映待ち、VRF | 当たる行を確かめていません（VRF は `event <番号> vrf <名前> …` の形がありますが、取り込みの規則をまだ書いていません） |
| remote_access | SSH の版、認証、回線パスワード | 当たる行を確かめていません |
| ntp_servers | 優先 | `prefer` が無く、`priority`（数）です |
| syslog_servers | 重要度、送信元 IF | 重要度は転送先ごとに決められません（`logging subsystem`）。送信元はインタフェースではなくアドレスで書く機種です |

取り込みとの関係:

- 平文の `username … password plain …` は機器の画面には出ませんが、書き出した文面を読み直すために取り込みの規則を足しました
- `ntp source` と `syslog facility`、`snmp-agent ip trap-source` は、取り込みが各行に配り直します。書き出しは、各行の値がそろっていることを確かめて 1 行にまとめます
- `ip vrf forwarding` を読むと vrfs の表に名前だけの行ができます。設計データの vrfs にも同じ名前の行を書いておくと、往復の確認が合います

### QoS の書き出し（M6）

- **書く行**: `class-map match-any|match-all <名前>` と `match ip access-list <リスト> [キュー]`・`match any [キュー]`、`policy-map` と `class`、`priority`・`priority percent`・`bandwidth`・`bandwidth percent`・`shape`・`set ip dscp`・`set ip precedence`・`queue-limit`、インタフェースの下の `service-policy input|output`・`traffic-shape rate`
- **DSCP・precedence の条件**: この機種には DSCP をじかに見る条件が無いので、条件ごとに `qos-d<番号>` というアクセスリストを作って書きます（取り込みは、これを条件の列に戻します）。同じ名前のリストを自分で作ると止まります
- **`class class-local` と `class class-default`** は、どのポリシーにも必ず書きます（機器にいつもある 2 つのクラスです）
- **`service-policy enable`** は、出力の行で、IF 全体を絞るか、ポリシーに優先・帯域・シェーピング・キューがあるときに書きます
- **順番**: クラスは「順番」の小さい順、その後ろに local、default の順で書きます。条件の順番は 10 刻みでなければ止まります
- **自動で置き換えて知らせるもの**（`adapt.yaml`。G-614 の警告で知らせます）: 「帯域（残りの %）」は同じ数の「帯域（%）」として書きます。「優先の段」は、段のない優先として書きます。帯域（%）の合計が、機器の 2 つのクラスの各 10% を足して 100% を超えるときは、値を変えずに知らせます
- **書けないもの**（G-602 で知らせます。人が決めます）: ポリシング（上限・捨てる、超えたときの動作）、公平に送る（fair-queue）。帯域（%）が 98 を超える行、5 つ以上のキューの長さは止まります

## 投入ログの読み方

コンフィグを流したときの端末ログから、機器に弾かれたコマンドを探します（仕様書 11.6）。機器の返事の形は `errors.yaml` にあります。

- 資料で確かめた「弾かれた」形は `% <語> -- Invalid command.` と、登録済みのユーザ名・ほかのユーザが設定モードにいる、の 3 つです。「受け付けた後の通知」は、再起動が要る旨・パスワードの強さ・ユーザの追加と削除・`% Warning:`・BGP の反映の案内です。ほかの `%` の返事は「パックが知らないメッセージ」として、人が確かめる指摘（L-701）になります。引数が足りない・値が範囲の外、というときのメッセージの形は、資料で確かめられていません
- プロンプトにかっこがある行（設定のモード）で打たれたコマンドを、生成したコンフィグの行と突き合わせます

## 確かめていないこと

資料に決まりが書かれていないか、見つけられなかった点です。実機の出力が手に入ったら、ここから確かめます。

- IPsec の行（`ike proposal` など）が running-config でどの位置に出るか（回線監視の後ろに書いています）。`ike proposal` の `authentication pre-shared` は書いていません（設定事例集の例に無く、選べる値が 1 つだけです）
- `ipsec autokey-map` の `pfs` を書かないときの既定（既定値として入れていません）
- PPP プロファイルと PPPoE のサブインタフェースが、running-config でどの位置・どの並びで出るか（設定事例集の入力の順と FAQ の字下げから書いています）
- `authentication secret-password`（暗号化した形のパスワード）は、取り込みも書き出しもしていません
- `lcp echo-interval` を既定値（30 秒）と同じ値で入れたときに、running-config に出るか
- アクセスリストの `sequence-mode`（番号つきのリスト）の書き出し。番号つきの行は取り込めますが、書き出しは番号なしだけです
- アクセスリストのポートの範囲（`range`）・`established`・ICMP の種類の、running-config での出方（コマンドリファレンスに構文はありますが、取り込みの規則をまだ書いていません）
- 標準 ACL に当たる書き方を `permit ip src … dest any` と読み替えてよいか（自動では読み替えず、人に書き直してもらっています）
- running-config 全体でのブロックの並び順と、先頭行の並び
- 字下げの桁数（資料によって 1〜4 桁とばらつきます。何桁でも読めます）
- 末尾に終わりの印があるかどうか（無いものとして扱っています）
- 1 行に並ぶオプションの順（`ike proposal`、`ipsec autokey-map` など。コマンドリファレンスの構文の順で読んでいます）
- `ip nat static` を書くインタフェースが「外側」であること
- `ip router ospf`・`router bgp`・`vrrp enable` が running-config のどこに出るか（経路の後ろ、`vrrp enable` はインタフェースの前に書いています）
- `address-family ipv4 unicast` のブロックの閉じ方（何も書いていません。資料の例では、空白と `!` の行で閉じています）
- `no vrrp <番号> preempt` が running-config に出るか（既定が有効なので、無効にした機器では出るはずです）
- Loopback に `ip ospf hello-interval` などを書いてよいか（Hello と Dead が「明示」の列なので、OSPF に入れた全部のインタフェースに書いています）
- `watch-group`・`network-monitor` が running-config のどこに出るか（経路の後ろに書いています）。ウォッチグループの名前に使える字と長さ
- 動作（action）のない `watch-group`、数字だけの名前（Cisco から持ってきた番号）を機器が受け付けるか
- インタフェースのブロックの中で、NAT の行がどこに出るか（アドレスの行の後ろ、`shutdown` の前に書いています）
- `ip napt static` が、`ip napt enable` と同じインタフェースのアドレスで受けること（コマンドリファレンスの説明から、そう読んでいます）
- `ip proxy-arp` の既定（既定値は入れていません）
- `bandwidth` の単位
- `decrement-vrrp-priority` の 2 つ目の数の意味
- 端末ログ: プロンプトの形（`ホスト名(config)#`）と、ページ送りの跡の字（`--More--` か `--more--` か）。検索結果の要約で見ただけです
- 端末ログ: `show running-config` の短い形（どこまで縮められるか）。読んでいません
- 端末ログ: 機器がページ送りの跡をどう消すか。制御文字が残っているログも、残っていないログも読めるようにしてありますが、実機のログでは確かめていません
- 端末ログ: 行ごとに時刻が付くログ（端末ソフトの設定）は読めません
- Loopback と Null が、`no shutdown` の行なしで「有効」であること（どちらの行も出ないという観察からの読みです。資料の文ではありません）
- 書き出し: 書き出した文面を実機に入れて確かめていません。とくに次の点です
  - 節の順（ユーザ・SNMP・NTP・syslog・Telnet / SSH サーバの位置）と、ブロックの中の行の順
  - この機種のハッシュの形（`$` で始まる値を断るだけで、ほかの値は確かめずに書きます）
  - `username … password plain` の後ろに付けられる「モード」の語（パスワードがその語と同じだと、読み違えられるおそれがあります）
  - `ip filter` の番号を 1 にしてよいか（番号の範囲を確かめていません）
  - 空白を含む `snmp-agent location` が、そのまま入るか
  - `ipv6 address` だけで IPv6 が働くか（`ipv6 enable` は書いていません）
  - SNMP の通知に `snmp-agent ip trap …` の行が要るか（書いていません）
  - `ssh-server ip enable` の前に要る鍵の生成（設定の行ではないので、書いていません）
  - `timezone` の時の範囲と、分の書き方（`+05 30` の形は、`+09 00` の観察からの類推です）
  - `ip route` で次ホップと IF の両方を書いたときの順（構文の 1 つ目の形で書いています）
