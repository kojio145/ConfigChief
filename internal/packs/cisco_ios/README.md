# Cisco IOS / IOS-XE パック（cisco_ios）

Cisco IOS 15 M&T と IOS-XE 16・17（ルータ）の running-config を読み取り、基本の設定を書き出すための決まりです。

- 版: 0.2.0
- 取り込みは下の「対応一覧」の全部。**書き出しは基本の 9 表と、M3 の表の全部（VRF・OSPF・BGP・冗長化・回線監視・アクセスリスト・プレフィックスリスト・NAT）**です（「コンフィグの書き出し」の節）。M4 の表も全部書けます（トンネル・DHCP・PPPoE・IPsec）
- **実機の出力では確かめていません**。構文はすべて下の公開資料で確かめたものです。画面に出る語の順番や大文字・小文字は、資料の出力例で見た範囲にとどまります

## 参照した資料

どれも Cisco の公開ページで、2026-11-14 に読みました。IOS 15 M&T と IOS-XE のルータで共通の、技術別のコマンドリファレンスです。

| 資料の名前 | 最終更新 |
|---|---|
| Cisco IOS Interface and Hardware Component Command Reference | 2023-10-05 |
| Cisco IOS IP Addressing Services Command Reference | 2018-01-11 |
| Cisco IOS IP Application Services Command Reference | 2020-04-26 |
| Cisco IOS IP Routing: Protocol-Independent Command Reference | 2018-04-22 |
| Cisco IOS IP Routing: OSPF Command Reference | 2024-04-30 |
| Cisco IOS IP Routing: BGP Command Reference | 2019-11-21 |
| Cisco IOS First Hop Redundancy Protocols Command Reference | 2020-06-21 |
| Cisco IOS IP SLAs Command Reference | 2018-02-25 |
| Cisco IOS Security Command Reference（A to C、D to L、M to R、S to Z） | 2013-01-11 〜 2024-08-28 |
| Cisco IOS SNMP Support Command Reference | 2017-11-13 |
| Cisco IOS Basic System Management Command Reference | 2019-12-13 |
| Cisco IOS Configuration Fundamentals Command Reference | 2011-09-28 |
| Cisco IOS Embedded Syslog Manager Command Reference | 2017-09-06 |
| Cisco IOS Broadband Access Aggregation and DSL Command Reference | 2020-06-09 |
| Cisco IOS Multiprotocol Label Switching Command Reference | 2020-06-10 |
| Cisco IOS Dial Technologies Command Reference | 2017-09-20 |
| Cisco IOS Terminal Services Command Reference | 2016-07-29 |
| Cisco IOS IPv6 Command Reference | 2019-12-13 |
| MPLS VPN VRF CLI for IPv4 and IPv6 VPNs（MPLS: Layer 3 VPNs Configuration Guide, Cisco IOS Release 15M&T） | 版の表記なし |
| Using the Cisco IOS Command-Line Interface（Configuration Fundamentals Configuration Guide, Cisco IOS Release 15.1S）。端末ログの形だけに使いました | 版の表記なし |
| Controlling Switch Access with Passwords and Privilege Levels（Security Configuration Guide, Cisco IOS XE 17.10.x）。ハッシュの種別の番号だけに使いました | 版の表記なし |

## 確かめた機種と版

- 実機: なし
- 資料: 上のコマンドリファレンスは版を分けていません。IOS-XE 16 にも当てはまると明記したページは見つかっていません
- Catalyst スイッチ（別系統のコマンドリファレンス）は対象外です

## 対応一覧

取り込める行の一覧です。このうち書き出せるのは、QoS・機器全体・ユーザ・リモート接続・NTP・syslog・SNMP・インタフェース・スタティックルートです（次の節）。

| 分野 | 読める行 | 入る表 |
|---|---|---|
| 機器全体 | `hostname`、`ip domain name` / `ip domain-name`、`ip name-server <アドレス>`、`clock timezone`、`enable secret` / `enable password`、`snmp-server location` / `contact`、`banner motd` / `banner login` | devices |
| ユーザ | `username <名前> [privilege <数>] secret <種類> <値>`、`… secret <平文>`、`… password <平文>` | users |
| リモート接続 | `line vty` の下の `transport input`、`login local` / `login`、`access-class <リスト> in`、`exec-timeout <分> 0`、`password`。`ip ssh version` | remote_access |
| NTP | `ntp server <アドレス> [prefer] [source <IF>]`、`ntp source <IF>` | ntp_servers |
| syslog | `logging host <アドレス>` / `logging <アドレス>`、`logging facility`、`logging trap`、`logging source-interface` | syslog_servers |
| SNMP | `snmp-server community <名前> ro\|rw [<リスト>]`、`snmp-server host <アドレス> [version 1\|2c] <コミュニティ>`、`snmp-server trap-source <IF>` | snmp_communities、snmp_hosts |
| インタフェース | `description`、`ip address <アドレス> <マスク> [secondary]`、`ip address dhcp` / `negotiated`、`ip unnumbered <IF>`、`no ip address`、`ipv6 address`、`shutdown`、`vrf forwarding` / `ip vrf forwarding`、`encapsulation dot1Q <番号>`、`mtu`、`ip mtu`、`ip tcp adjust-mss`、`bandwidth`、`speed`、`duplex`、`ip access-group <リスト> in\|out`、`ip nat inside\|outside`、`ip helper-address`、`no ip proxy-arp` | interfaces |
| スタティックルート | `ip route [vrf <VRF>] <宛先> <マスク> <次ホップ>\|<IF> [<距離>] [name <名前>] [track <番号>]` | static_routes |
| VRF | `vrf definition <名前>` / `ip vrf <名前>` の下の `rd`、`description` | vrfs |
| OSPF | `router ospf <番号> [vrf <VRF>]` の下の `router-id`、`auto-cost reference-bandwidth`、`network <アドレス> <ワイルドカード> area <エリア>`、`area <エリア> stub\|nssa [no-summary]`、`area <エリア> range`、`redistribute`、`default-information originate [always]`、`passive-interface default` / `<IF>`。インタフェースの下の `ip ospf <番号> area <エリア>`、`ip ospf cost` / `priority` / `hello-interval` / `dead-interval` / `network` / `authentication` / `authentication-key` / `message-digest-key` | ospf_processes、ospf_interfaces、ospf_areas、ospf_ranges、ospf_redistribute |
| BGP | `router bgp <AS>` の下、`address-family ipv4` の下、`address-family ipv4 vrf <VRF>` の下の `bgp router-id`、`bgp log-neighbor-changes`、`neighbor <アドレス> remote-as` / `description` / `password` / `ebgp-multihop` / `update-source` / `shutdown` / `next-hop-self` / `soft-reconfiguration inbound` / `prefix-list <名前> in\|out` / `route-map <名前> in\|out`、`network <アドレス> mask <マスク>`、`redistribute` | bgp_processes、bgp_neighbors、bgp_networks、bgp_redistribute |
| HSRP | インタフェースの下の `standby version`、`standby <番号> ip` / `priority` / `preempt` / `timers` / `authentication text` / `track` | fhrp |
| VRRP（インタフェースに書く古い形） | `vrrp <番号> ip` / `priority` / `preempt` / `timers advertise` / `track` | fhrp |
| アクセスリスト | `ip access-list standard\|extended <名前>` の下の各行（番号あり・なしの両方）、`remark`。`access-list <番号> …`（標準と、ポートの無い拡張） | acls、acl_entries |
| プレフィックスリスト | `ip prefix-list <名前> seq <番号> permit\|deny <アドレス>/<長さ> [ge <数>] [le <数>]` | prefix_lists |
| NAT | `ip nat inside source list <リスト> interface <IF> overload`、`ip nat inside source list <リスト> pool <名前> [overload]`、`ip nat pool`、`ip nat inside source static <内側> <外側>`、`ip nat inside source static tcp\|udp <内側> <ポート> <外側> <ポート>`、`ip nat inside source static tcp\|udp <内側> <ポート> interface <IF> <ポート>` | nat_rules |
| 回線監視 | `track <番号> ip sla <番号> reachability`、`track <番号> interface <IF> line-protocol`、その下の `delay`。`ip sla <番号>` の下の `icmp-echo <アドレス> [source-interface <IF>]`、`frequency`、`timeout`。`ip sla schedule <番号> life forever start-time now` | tracks |
| PPPoE | 物理側の `pppoe-client dial-pool-number <番号> [service-name <名前>]`。Dialer の下の `encapsulation ppp`、`dialer pool <番号>`、`ppp authentication chap\|pap\|chap pap callin`、`ppp chap hostname` / `password`、`ppp pap sent-username … password …` | pppoe |
| IPsec（IKEv1・トンネル IF 型） | `crypto isakmp policy <番号>` の下の `encr` / `encryption`、`hash`、`authentication pre-share`、`group`、`lifetime`。`crypto isakmp key <鍵> address <アドレス>`、`crypto isakmp keepalive <秒>`、`crypto ipsec transform-set`、`crypto ipsec profile <名前>` の下の `set transform-set`、`set pfs`、`set security-association lifetime seconds`。トンネルの下の `tunnel protection ipsec profile <名前>` | ipsec_tunnels、tunnels |
| IPsec（IKEv1・クリプトマップ型） | `crypto map <名前> <番号> ipsec-isakmp` の下の `set peer <アドレス>`、`set transform-set <名前>`、`set pfs`、`set security-association lifetime seconds`、`match address <リスト>`。インタフェースの下の `crypto map <名前>`。IKE のポリシー・鍵・トランスフォームセットは上と同じ | ipsec_tunnels、ipsec_selectors |
| トンネル（GRE など） | `tunnel source`、`tunnel destination`、`tunnel key`、`tunnel mode ipip` / `tunnel mode ipsec ipv4`（行が無ければ GRE）、`keepalive <秒> <回数>` | tunnels |
| DHCP サーバ | `ip dhcp pool <名前>` の下の `network`、`default-router`、`dns-server`、`domain-name`、`lease`。`ip dhcp excluded-address <開始> [<終了>]` | dhcp_pools |
| QoS（M6。MQC） | `class-map [match-all\|match-any] <名前>` の下の `match access-group name <リスト>` / `<番号>`、`match [ip] dscp <値> …`（8 つまで）、`match [ip] precedence <値> …`（4 つまで）、`match any`。`policy-map <名前>` の下の `class <名前>\|class-default` と、その下の `priority [<kbps> [<バースト>]]`、`priority percent <%> [<バースト>]`、`priority level 1\|2`、`bandwidth <kbps>`、`bandwidth percent <%>`、`bandwidth remaining percent <%>`、`shape average <bps>[k\|m\|g] [<Bc> [<Be>]]`、`police [cir\|rate] <bps> [[bc] <バースト> [[be] <バースト>]] [conform-action transmit] [exceed-action drop\|transmit\|set-dscp-transmit <値>\|set-prec-transmit <値>]`（動作を下の行に書く形も）、`set [ip] dscp`、`set [ip] precedence`、`queue-limit <数> [packets]`、`fair-queue`、`service-policy <子>`。インタフェースの下の `service-policy input\|output <名前>`。拡張アクセスリストの `ip` などの行の末尾の `dscp <値>`・`precedence <値>` | qos_classes、qos_class_matches、qos_policies、qos_bindings、acl_entries の dscp・precedence 列 |
| QoS（M6。MQC より前の形） | インタフェースの下の `traffic-shape rate <bps> [<Bc> [<Be>]]`、`rate-limit input\|output [access-group <番号>] <bps> <通常のバースト> <最大のバースト> conform-action transmit exceed-action drop\|transmit\|set-dscp-transmit <数>\|set-prec-transmit <数>`、`priority-group <番号>`、`custom-queue-list <番号>`。`priority-list <番号> protocol ip high\|medium\|normal\|low [list <リスト> \| tcp <ポート> \| udp <ポート>]`、`priority-list <番号> default <キュー>`、`priority-list <番号> queue-limit <4 つの数>`、`queue-list <番号> protocol ip <キュー> [list <リスト> \| tcp <ポート> \| udp <ポート>]`、`queue-list <番号> default <キュー>`、`queue-list <番号> queue <キュー> [byte-count <バイト>] [limit <数>]` | qos_classes、qos_class_matches、qos_policies、qos_bindings（`tcp`・`udp <ポート>` は acls、acl_entries も） |

## このメーカーならではの読み方

- **古い書き方と新しい書き方の両方を読みます。** `ip domain name` と `ip domain-name`、`vrf definition` と `ip vrf`、`vrf forwarding` と `ip vrf forwarding`、`logging host <アドレス>` と `logging <アドレス>` などです。OS の違いではなく出た時期の違いで、1 台に両方が残ることがあります
- **マスクとワイルドカードは、長さの形に直して入れます。** とびとびのワイルドカード（長さに直せないもの）の行は「未対応」になります
- **アクセスリストの番号。** 番号が画面に出るかどうかは版によって違うため、両方を読みます。番号の無い行は、行の順に 10、20、30 … と番号を付けます
- **OSPF を動かすインタフェース。** `network` の行にアドレスが入るインタフェースと、`ip ospf <番号> area` を書いたインタフェースを、ospf_interfaces の行にします。どのインタフェースにも当たらない `network` の行は「未対応」に戻ります
- **NAT の規則の名前。** 機器に名前が無いため、行の順に `nat1`、`nat2` … と付けます
- **IPsec のトンネル IF 型。** トンネルが指すプロファイル、プロファイルが指すトランスフォームセット、相手のアドレスが同じ鍵、番号がいちばん小さい IKE のポリシーをつないで 1 行にします。2 つ目以降の IKE のポリシーは「未対応」に戻ります。行の名前はトンネル IF の名前です。自分側（トンネルの送信元）は tunnels の行にあり、ipsec_tunnels の「自分側 IF」には入れません
- **IPsec のクリプトマップ型。** インタフェースが使っているマップの 1 項目を、ipsec_tunnels の 1 行にします。機器には項目ごとの名前が無いため、行の名前は `<マップ名>/<番号>` です。`match address` が指す拡張アクセスリストの `permit ip <ネットワークかホスト> <ネットワークかホスト>` の行が、ipsec_selectors の行になります。**ipsec_selectors に写した行は、アクセスリストの表（acl_entries）から外します。行が残らなければ、リスト（acls）も外します**（フィルタ・NAT・`line vty`・SNMP も使っているリストは、そのまま残します）。`deny` の行、`any`、ポート付きの行は ipsec_selectors に入らず、アクセスリストの表に残ります。どのインタフェースも使っていないマップは「未対応」に戻ります
- **PPPoE。** プールの番号が同じ物理インタフェースと Dialer を 1 行にします。行の名前は Dialer の名前です（Cisco の PPPoE に名前はありません）。CHAP と PAP の両方の行があるときは、接続 ID とパスワードが同じときだけ読みます（違えば、関係する行が「未対応」に戻ります）
- **QoS の二段のポリシー。** 親のポリシーが `class class-default` だけで、その下が `shape average` と子の `service-policy` だけのときは、親を「IF 全体を絞る速さ」として適用の行（qos_bindings）に畳みます。適用の行のポリシー名は子の名前になり、親のポリシーの行は表から外します（親の行は「取り込み済み」のままです）。それ以外の二段の形（親にほかのクラスや動作があるもの、入力の向き）は、`service-policy` の行が「未対応」に戻ります
- **QoS の MQC より前の形。** 機器に名前が無いものは、パックが名前を付けます。`rate-limit` はポリシー `rl-<IF の短い名前>-in|out`（`/` と `.` は `-` にします。例 `rl-Gi0-1-in`）（ACL 付きの行はクラス `rl-acl<番号>`、無い行は `default`）、`priority-list <n>` はクラスとポリシー `pq<n>`（条件の行ごとにキュー）、`queue-list <n>` はポリシー `cq<n>` とキューごとのクラス `cq<n>-<キュー>`、`tcp <ポート>`・`udp <ポート>` はリスト `pq<n>-tcp<ポート>` などです（送信元か宛先のどちらかがそのポートの通信に合う 2 行）。`default` の行は順番 9990 の条件（すべての通信）にします。`priority-list` に `default` の行が無いときは、機器の既定（normal）の条件を足します
- **カスタムキューの帯域。** キューの `byte-count`（無ければ 1500）を、使っているキューの合計で割った割合（% の整数。切り捨て、1 より小さければ 1）を帯域の % にします。機器はバイト数で順に送るので、**動きは近いものの同じではありません**。どのキューにも当たらない通信の行き先は、`default` の行が無ければ書かれていません（機器の既定はキュー 1 ですが、足していません）
- **`traffic-shape rate` の単位**は bps とビットとして読みます（コマンドリファレンスの該当の章が取れず、設定ガイドの手順から読みました。下の「確かめていないこと」）
- **QoS の DSCP・precedence の名前**（`ef`・`af41`・`critical` など）は数に直して入れます（別紙 A 5.2）。`class-default` は `default` という名前で入れます。`shape`・`police` の `k`・`m`・`g` の付いた値は bps に直します
- **全体に 1 行だけ書く設定**（`ntp source`、`logging facility`、`logging trap`、`logging source-interface`、`snmp-server trap-source`）は、それぞれのサーバの行に同じ値を入れます

## 端末ログの読み方

端末ソフトのログ（ログインから `show running-config` まで入ったもの）は、設定の部分だけを切り出して読みます。決まりは別紙 B 2.4、書いてある場所は `pack.yaml` の `session` です。

- **プロンプト。** 行の頭の「ホスト名」「かっこに入ったモード（あれば）」「`>` か `#`」です
- **設定を表示するコマンド。** `show running-config` と `show startup-config`、その短い形（`sh run`、`sho run`、`show run`、`show start` など。`show` は `sh` まで、後ろは `run`・`start` まで縮めた形）、頭に `do` を付けた形です。後ろに語が続く形（`show running-config all`、`| include …`、`interface …` など）は、全体が出ないので設定として読みません
- **ページ送りの跡。** ` --More-- ` です。跡だけの行は「無視（ページ送りの跡）」にし、跡の後ろに設定が続く行は、跡を取り除いて読みます（そのときは注意を出します）
- 同じログにコマンドが何回もあるときは、最後のものを読みます

## 値を入れない規則

別紙 B 3.2 で認められている 2 つの場合だけです。

| 規則 | 行 | 理由 |
|---|---|---|
| `bgp.af` | `router bgp` の下の `address-family ipv4` | 子の行を持つ入れ物の行 |
| `if.ipv4.none` | `no ip address` | 「ipv4 の列が空」と同じ意味の行 |
| `qos.policy` | `policy-map <名前>` | 子の行を持つ入れ物の行 |
| `qos.policy.police.conform` | `police` の下の `conform-action transmit` | 機器の既定の動作で、「列が空」と同じ意味の行 |

## 読んで、表には入れない語

| 行 | 捨てる語 |
|---|---|
| OSPF の `redistribute … subnets` | `subnets`（入れる列がありません） |

## 「無視」にする行

- `ignore.yaml`: `Building configuration...`、`Current configuration : … bytes`（表示上の行）、`version <数>`（機器が自動で付ける行）、`end`（終わりの印）
- `exit-address-family`（終わりの印。VRF と BGP のブロックの中）
- `vrf definition` の下の `address-family ipv4`（VRF で IPv4 を使う印。入れる列がありません。書き出しでは必ず付けます。`address-family ipv6` と `route-target` は読みません）
- `ip sla` の下の `threshold <ミリ秒>`（入れる列がありません。`track … reachability` の監視では、しきい値を超えても「届く」のままなので、結果を変えません）
- 物理インタフェースの下の `pppoe enable`・`pppoe enable group global`（PPPoE を使う印。入れる列がありません。書き出しでは必ず付けます）
- インタフェースの下の `qos pre-classify`（トンネルで、アドレスやポートで見分けるときに要る行。書き出しのときにパックが書きます。別紙 A 3.18）
- `!` で始まる行は区切りとして数えます

## 読まない行（「未対応」として画面に出ます）

試験の見本（`tests/import/basic/input.txt`）にある 13 行と、その理由です。

| 行 | 理由 |
|---|---|
| `service timestamps debug …`、`service timestamps log …`、`no service password-encryption` | 入れる列がありません |
| `crypto isakmp keepalive <秒> <再送の間隔>` | 2 つ目の数は再送の間隔（秒）で、データセットが持つのは回数です |
| トランスフォームセットの下の `mode tunnel` | 入れる列がありません |
| `dialer-group <番号>`、`dialer-list <番号> protocol ip permit` | 入れる列がありません（通信があるときだけつなぐ設定の行で、IOS XE の PPPoE クライアントの手順にはありません。書き出しもしません） |
| OSPF の下の `log-adjacency-changes` | 入れる列がありません |
| `neighbor <アドレス> activate`（3 行） | 入れる列がありません |
| `line con 0` と、その下の `exec-timeout` | コンソールの設定で、入れる表がありません（remote_access は遠隔の接続だけです） |

このほか、次の書き方も読みません。

- ルートマップ（`route-map` のブロック）、IKEv2
- クリプトマップのうち、動的なマップ（`dynamic`）、プロファイル付き、相手を名前で書く形、2 つ目以降のトランスフォームセット
- VRF の `route-target`、VRF 付きの `ip name-server`
- `banner exec` など motd・login 以外のバナー
- `username … password 7 …`、OSPF の鍵の `7 <値>` の形（戻せる形の暗号で、形を表す語がありません）
- 秒まで指定した `exec-timeout`
- アクセスリストのポートの名前（`www` など）、`time-range`、ポート付きの番号リスト（拡張）。`dscp`・`precedence` は `ip` などポートの無い行の末尾だけを読みます
- QoS の `match` のうち `input-interface`・`qos-group`・`cos`・`protocol`・`not`・`class-map`、`shape peak`・`shape average percent`・`police cir percent`・`pir` の付いた `police`、`violate-action`、`set cos`・`set qos-group`、`random-detect`、`bytes`・`ms` の `queue-limit`
- `mask` の無い BGP の `network`、`redistribute ospf <番号>`（BGP の下）
- `track <番号> ip sla <番号>` の `reachability` が無い形、`ip sla` の `source-ip`、`ip sla schedule` のほかの形
- `callin` の無い `ppp authentication`、引用符で囲んだ `service-name`、`pppoe-client` のほかの語（`dial-on-demand`・`restart`・`mac-override`）
- DHCP の `default-router` にアドレスを複数並べた形、分まで指定した `lease`、`lease infinite`
- `ip route` の `tag`・`permanent`・`dhcp`・`global`・`multicast`

## コンフィグの書き出し

基本の 9 表（devices、interfaces、static_routes、users、remote_access、ntp_servers、syslog_servers、snmp_communities、snmp_hosts）と、VRF（vrfs）・回線監視（tracks）・アクセスリスト（acls、acl_entries）・プレフィックスリスト（prefix_lists）・NAT（nat_rules）、OSPF（ospf_processes、ospf_interfaces、ospf_areas、ospf_ranges、ospf_redistribute）・BGP（bgp_processes、bgp_neighbors、bgp_networks、bgp_redistribute）・冗長化（fhrp）、トンネル（tunnels）・DHCP（dhcp_pools）・PPPoE（pppoe）・IPsec（ipsec_tunnels、ipsec_selectors）を書き出します。書き出した文面は、同じパックで取り込み直して設計データと比べます（往復の確認）。

節の順: `system`（ホスト名・特権パスワード・時刻帯・DNS・ユーザ）→ `vrf` → `ipsec`（IKE・トランスフォームセット・プロファイル・クリプトマップ。それを指すインタフェースより前）→ `interfaces` → `tracking`（回線監視）→ `routing` → `ospf` → `bgp` → `policy`（アクセスリスト・プレフィックスリスト）→ `nat` → `dhcp` → `services`（syslog・SNMP・NTP）→ `extra`（追加コンフィグの既定の位置）→ `lines`（バナー・`line vty`）→ `footer`（`end`）。

書き方について決めたこと:

| 項目 | 書き方 |
|---|---|
| ハッシュ化された秘密（特権パスワード・ユーザ） | ハッシュの始まりで種別の番号を決めます。`$1$` → `5`、`$8$` → `8`、`$9$`・`$14$` → `9`。それ以外の始まりはエラーにします（別メーカーのハッシュは引き継げません） |
| 平文の秘密 | `secret 0 <値>` と書きます。形が空のセルは平文として扱います |
| ユーザの権限 | `admin` は `privilege 15`、数はそのまま。`operator` に当たる数は決めていないので、語の対応がないエラー（G-603）になります。数で書いてください |
| 時刻帯 | 「名前 +時[:分]」の形（`JST +9` → `clock timezone JST 9 0`）。名前のない形はエラーにします |
| バナー | `banner login` で書き、区切りの文字は `#` `@` `%` `~` `\|` のうち文面に無い最初のものを使います |
| リモート接続 | `line vty 0 4` に書きます。5 番以降の線は追加コンフィグに書いてください。認証 `aaa` は書けません（エラー）。認証 `password` にはログインのパスワードが要ります |
| インタフェース | 有効な IF には `no shutdown`、無効な IF には `shutdown` を必ず書きます。アドレスの無い IF に `no ip address` は書きません。VRF は `vrf forwarding` の形で、アドレスより前に書きます。速度 `10000` は書けません（エラー） |
| syslog | ファシリティ・重要度・送信元 IF は機器に 1 つなので、全部の行で同じ値でなければエラーにします |
| SNMP | 通知の送信元 IF も機器に 1 つです（そろっていなければエラー）。通知先の版 3 は書けません（エラー）。通知先のコミュニティは必須です |
| NTP | 送信元 IF が全部の行で同じなら `ntp source`、違えばサーバごとの `source` で書きます |
| スタティックルート | 説明は `name` に書きます。空白を含む説明はエラーにします（名前は 1 語） |
| アクセスリスト | 名前付きの形（`ip access-list standard｜extended <名前>`）で書きます。名前が番号のリストも同じ形です（`ip access-list standard 10`）。アドレスは `any`・`host <アドレス>`・アドレスとワイルドカード。ポートは番号だけ（`80` → `eq 80`、`1000-2000` → `range 1000 2000`、`>1023` → `gt 1023`、`<1024` → `lt 1024`、`!80` → `neq 80`）。プロトコル `ah` は `ahp` と書きます。`established` は tcp の行だけ、標準のリストに書ける付加条件は `log` だけです（ほかはエラー） |
| アクセスリストの行の番号 | 行が 10、20、30… と並んでいるリストは、番号を書きません（番号を保存しない版の機器の出方と同じ）。それ以外の番号のリストは、全部の行に番号を書きます（注記の行にも） |
| アクセスリストの説明 | 機器にリストの説明の行がないので、「説明」の列は書けません（値があれば G-602）。注記は「動作」が remark の行で書きます |
| プレフィックスリスト | `ip prefix-list <名前> seq <番号> …` と、番号を必ず書きます |
| VRF | `vrf definition <名前>` の形で書き、`address-family ipv4` 〜 `exit-address-family` を必ず付けます（この形の VRF は、これが無いと IPv4 を運びません）。`ip vrf <名前>` で取り込んだ VRF も、この形で書きます |
| 回線監視（ping） | `ip sla <番号>`・`ip sla schedule <番号> life forever start-time now`・`track <番号> ip sla <番号> reachability` の 3 つを書きます。SLA の番号は、回線監視の番号と同じにします。番号は 1〜1000 の数だけ（名前はエラー）。監視先は IPv4 アドレスだけ |
| 回線監視の待ち時間 | 5000 ミリ秒より短い待ち時間は、同じ値の `threshold` を先に書きます（機器は、しきい値（既定 5000）より短い待ち時間を受け付けません）。待ち時間が間隔より長い行はエラーにします |
| 回線監視（IF） | `track <番号> interface <IF> line-protocol`。監視の方法 `route` は書けません（エラー） |
| 回線監視で書けない列 | 「断と判断する回数」「復旧と判断する回数」「VRF」は、当たる行を確かめていないので読みません（値があれば G-602）。反映待ちは `delay up <秒> down <秒>` で書きます |
| NAT | 種類ごとに、取り込みが読める形だけを書きます。`pat_interface` → `ip nat inside source list <ACL> interface <外側 IF> overload`、`pat_pool`・`dynamic_pool` → `ip nat pool <名前> <先頭> <末尾> prefix-length <長さ>` と `ip nat inside source list <ACL> pool <名前> [overload]`（プールの名前は、行の名前）、`static`、`static_port`（外側アドレスが `interface` のときは `interface <外側 IF>` の形） |
| NAT の向き | NAT の行があるのに、NAT の向きが inside と outside のインタフェースがそろっていなければ、エラーにします。`pat_interface` の外側 IF は、向きが outside の IF でなければエラーです |
| NAT の対象の通信 | `pat_interface`・`pat_pool`・`dynamic_pool` は ACL の名前が必須です（空 = すべて、は Cisco では書けません） |
| NAT の外側 IF | `pat_interface` と、外側アドレスが `interface` の `static_port` だけが読みます。ほかの種類の行に外側 IF を書くと G-602 になります（Cisco の行に、外側 IF を書く場所がありません。セルを空にしてください） |
| OSPF のインタフェース | インタフェースの下に `ip ospf <プロセス番号> area <エリア>` で書きます（`network … area …` の行は書きません。`network` で取り込んだ設定も、この形になります）。1 つのインタフェースは 1 つのプロセスだけ。インタフェースの VRF とプロセスの VRF が違えばエラー |
| OSPF の受動 | 「既定を受動にする」が有効なら `passive-interface default`。インタフェースの行の「受動」は、有効 → `passive-interface <IF>`、無効 → `no passive-interface <IF>`、空 → 書きません |
| OSPF の再配布 | `subnets` を必ず付けます（付けないとクラスフルの経路しか入りません。表に列がありません）。元が bgp のときは、BGP の表の AS 番号を付けます |
| OSPF の認証 | md5 → `ip ospf authentication message-digest` と `ip ospf message-digest-key <鍵番号> md5 <鍵>`、simple → `ip ospf authentication` と `ip ospf authentication-key <鍵>`、none → `ip ospf authentication null`。鍵は平文で書きます |
| OSPF で書けない列 | エリアの「既定経路のコスト」（取り込みの規則がありません。値があれば G-602） |
| BGP | AS 番号は機器に 1 つです（行によって違えばエラー）。VRF が空の行のネイバー・広告経路・再配布は `router bgp` の直下に、VRF の行のものは `address-family ipv4 vrf <VRF>` 〜 `exit-address-family` に書きます |
| BGP の再配布 | 元が ospf のときは、同じ VRF の OSPF のプロセス番号を付けます（プロセスがちょうど 1 つのときだけ。それ以外はエラー） |
| BGP で書けない列 | プロセスとネイバーの Keepalive・Hold、ネイバーの「デフォルト経路の広告」、VRF の行のルータ ID と「ネイバーの変化を記録」（取り込みの規則がありません。値があれば G-602） |
| HSRP | `standby <グループ> ip`・`priority`・`preempt [delay minimum <秒>]`・`timers <Hello> <Hold>`（2 つは組）・`authentication text <鍵>`・`track <番号> [decrement <数>]`。版 2 は `standby version 2`（インタフェースに 1 つ。グループで違えばエラー）。認証 md5 は書けません（エラー） |
| VRRP | インタフェースの下の古い形（`vrrp <グループ> ip`・`priority`・`preempt [delay minimum <秒>]`・`timers advertise <秒>`・`track`）で書きます。版・Hold・認証は書けません（値があれば G-602） |
| 切り戻しの無効 | 「切り戻し」が無効の行は `no standby <グループ> preempt` / `no vrrp <グループ> preempt` と書きます（取り込みもこの行を読みます） |
| トンネル | トンネル IF の下に `tunnel source`・`tunnel destination`・`tunnel key`（種類が gre のときだけ）・`keepalive <間隔> <回数>`（2 つは組）。種類が gre なら `tunnel mode` の行は書きません（機器の既定）。ipip は `tunnel mode ipip`、ipsec は `tunnel mode ipsec ipv4` と `tunnel protection ipsec profile <名前>`（IPsec の表に、そのトンネル IF の行が要ります。送信元と宛先は ipsec でも必須です）。l2tpv3 は語の対応がないエラー（G-603） |
| DHCP | `ip dhcp pool <名前>` の下に `network <アドレス> <マスク>`・`default-router`・`dns-server`・`domain-name`・`lease <日> [<時>]`。配らないアドレスは `ip dhcp excluded-address <先頭> [<末尾>]`（プールのネットワークの外なら止まります） |
| DHCP の貸し出し時間 | `2d` → `lease 2`、`36h` → `lease 1 12`。24 の倍数の時間（`24h`）は日の形に書き直してもらいます（取り込み直すと `1d` になり、往復が合わないため） |
| DHCP で書けない列 | 「配る範囲の先頭・末尾」（Cisco は、ネットワークから配らないアドレスを引いた残りを配ります。範囲の行がありません。値があれば G-602。配らないアドレスに書き直してください） |
| PPPoE | 論理 IF は `Dialer1`〜`Dialer255`。その番号をプールの番号に使います（表に番号の列がありません）。物理 IF の下に `pppoe enable group global` と `pppoe-client dial-pool-number <番号> [service-name <名前>]`、Dialer の下に `encapsulation ppp`・`dialer pool <番号>`・認証の行を書きます。物理 IF と論理 IF は、どちらもインタフェースの表に行が要ります。同じ物理 IF・同じ論理 IF を 2 つの行で使うと止まります |
| PPPoE の認証 | 認証方式があれば `ppp authentication <方式> callin`（chap_pap は `chap pap`）。chap は `ppp chap hostname`・`ppp chap password`、pap は `ppp pap sent-username … password …`、chap_pap と空は両方を書きます。パスワードは平文で、種別の番号を付けません |
| PPPoE のアドレス・MTU | 論理 IF のインタフェースの行に書きます（アドレス `negotiated` → `ip address negotiated`、MTU → `mtu`、IP MTU → `ip mtu`、MSS → `ip tcp adjust-mss`） |
| PPPoE で書けない列 | 「デフォルト経路を作る」「監視間隔」（値があれば G-602）。デフォルト経路は、出力 IF を Dialer にしたスタティックルートの行で書いてください。`dialer-group`・`dialer-list` は書きません |
| IPsec（共通） | IKE は版 1・事前共有鍵だけ。`crypto isakmp policy 10`（`encryption`・`hash`・`authentication pre-share`・`group`・`lifetime`）を 1 つ書きます。**IKE の暗号・ハッシュ・DH グループ・有効時間と、相手監視の間隔は、全部の行で同じでなければ止まります**（Cisco の IKE の提案は機器全体のもので、取り込みも 1 組だけを読みます）。`crypto isakmp key <鍵> address <相手>` は相手ごとに 1 つ（同じ相手で鍵が違えば止まります）。相手監視の間隔は `crypto isakmp keepalive <秒>`。相手はアドレスだけ（`any` や名前は止まります） |
| IPsec（トンネル IF 型） | `crypto ipsec transform-set TS-<トンネル IF>`、`crypto ipsec profile PROF-<トンネル IF>`（`set transform-set`・`set pfs group<数>`・`set security-association lifetime seconds`）、トンネル IF の下に `tunnel protection ipsec profile PROF-<トンネル IF>`。トンネルの表に種類 ipsec の行が要り、その宛先が相手アドレスと同じでなければ止まります |
| IPsec（クリプトマップ型） | 行の名前を `<マップ名>/<番号>` にします（ほかの形は止まります）。対象通信の行から `ip access-list extended ACL-<マップ名>-<番号>` を作り（`<順番> permit ip …`）、`crypto map <マップ名> <番号> ipsec-isakmp` に `set peer`・`set transform-set TS-<マップ名>-<番号>`・`set pfs`・`set security-association lifetime seconds`・`match address` を書きます。自分側 IF の下に `crypto map <マップ名>`。1 つのインタフェースに 1 つのマップ、1 つのマップは 1 つのインタフェースです。対象通信のプロトコルは ip だけ |
| IPsec の名前 | トランスフォームセット・プロファイル・対象通信のアクセスリストの名前は、表に列が無いのでパックが決めます。取り込んだ元のコンフィグの名前（`IPSEC-PROF-…` など）は引き継ぎません |
| IPsec で書けない列 | 自分の ID・相手の ID・IKE のモード・相手監視の再送回数・NAT 越え・自分側アドレス、トンネル IF 型の行の自分側 IF（値があれば G-602）。ESP のハッシュ sha256 以上・暗号 GCM と、IKE のハッシュ sha512 は、語を資料で確かめていないのでエラー（G-603）になります。トランスフォームセットの `mode tunnel` は書きません（既定） |
| 並び | 表の行は、キーの自然な順（`GigabitEthernet0/2` は `GigabitEthernet0/10` より前。`Loopback0` は `GigabitEthernet…` より後）です |

取り込みとの関係: `ntp server`・`logging host` は、取り込みがアドレスの形だけを読みます。名前で書いたサーバは、往復の確認で指摘になります。

### QoS の書き出し（M6）

- **書く行**: MQC の形だけです（古い `traffic-shape`・`rate-limit`・`priority-list`・`queue-list` は読むだけで、書きません）。`class-map match-any|match-all` と `match access-group [name]`・`match dscp`・`match precedence`・`match any`、`policy-map` と `class`、`priority`・`priority percent`・`priority level`・`bandwidth`・`bandwidth percent`・`bandwidth remaining percent`・`shape average`・`police cir … conform-action transmit exceed-action …`・`set dscp`・`set precedence`・`queue-limit <数> packets`・`fair-queue`、インタフェースの下の `service-policy input|output`
- **DSCP・precedence** は数で書きます
- **IF 全体を絞る速さ**: `shape-<IF の短い名前>` という親のポリシー（`class class-default` に `shape average` と、子の `service-policy`）を作り、インタフェースにはその親を当てます（取り込みは、これを適用の行に戻します）
- **順番**: クラスは「順番」の小さい順、`class-default` はいつも最後に書きます。条件の順番は 10 刻みでなければ止まります
- **自動で置き換えて知らせるもの**（`adapt.yaml`。G-614 の警告で知らせます）: 条件ごとのキュー（high〜low。NEC IX と、古い `priority-list` を読んだもの）は MQC に無いので、キューごとのクラス（`<クラス名>-high` など）に分けて書きます。high は `priority level 1`、medium は `priority level 2`、normal は残りの帯域の 75%、low は 25% です。**動きは近いものの同じではありません。** 割合を変えたいときは、分けた形のクラスの行をパラメータブックに自分で書きます。キューがどれも同じクラスは、キューを書かずに 1 つのクラスとして書きます。そのクラスに自分の動作（帯域・優先など）があるポリシーでは分けず、キューの値を G-602 で知らせます
- **書けないもの**: 機器自身の通信のクラス（local）の行、4 つ並べたキューの長さは止まります
- **確かめていないこと**: サブインタフェースやトンネルに、絞る速さの無いキュー付きのポリシーを当てられるか（古い資料では親のシェーピングが要ります。パックは足しません）。`qos pre-classify` は読まず、書きません（要るときは追加コンフィグに書いてください）

## 投入ログの読み方

コンフィグを流したときの端末ログから、機器に弾かれたコマンドを探します（仕様書 11.6）。機器の返事の形は `errors.yaml` にあります。

- 資料で確かめた「弾かれた」形は 3 つ（`% Invalid input detected at '^' marker.`・`% Incomplete command.`・`% Ambiguous command:`）、「受け付けた後の通知」は `% Warning:` と鍵の生成の 2 つです。ほかの `%` の返事（アドレスの重なりなど）は「パックが知らないメッセージ」として、人が確かめる指摘（L-701）になります。機器のログの行（`%LINK-3-UPDOWN: …`）は返事として数えません
- プロンプトにかっこがある行（設定のモード）で打たれたコマンドを、生成したコンフィグの行と突き合わせます

## 確かめていないこと

資料に決まりが書かれていないか、見つけられなかった点です。実機の出力が手に入ったら、ここから確かめます。

- アクセスリスト・プレフィックスリストが running-config のどこに出るか（経路の後ろに書いています）
- `crypto map` の `match address` が指すアクセスリストを、マップより先に定義しておく必要があるか（先に書いています）
- IKE の提案が 2 つ以上ある機器で、どの提案がどの相手に使われるか（機器が相手と折り合いを付けます）。取り込みは番号がいちばん小さい 1 つだけを読み、書き出しは 1 つだけを書きます
- `crypto isakmp key` の種別の番号（`0`・`6`）。取り込みは `0` 付きと番号なしを読み、書き出しは番号なしです
- PPPoE のクライアントに `pppoe enable group global` が要るか。IOS XE の設定ガイドの手順にはあり、古い IOS 12.2T の例にはありません。要ると言い切る文は見つけていません（必ず書いています）
- `dialer-group`・`dialer-list` が無くても PPPoE がつながり続けること。IOS XE の設定ガイドの手順と例に出てこないことと、コマンドリファレンスの「通信があるときだけつなぐ設定をしたときは、これらも要る」という文から、書いていません。古い IOS では要るかもしれません（要るなら追加コンフィグに書いてください）
- `ppp chap password`・`ppp pap sent-username … password` の種別の番号（`0`・`7`）。構文の行には無く、例にだけあります。取り込みは `0` 付きと番号なしを読み、書き出しは番号なしです
- PPPoE のデフォルト経路を自動で作る行。コマンドリファレンスは `ppp ipcp default route`、新しい設定例は `ppp ipcp route default` と、語の順が違います。どちらも読まず、書きません
- Dialer（PPP）の下の `keepalive` の働き。資料に記述がないので、「監視間隔」は読まず、書きません
- Dialer の下の行の並び（`encapsulation ppp` をアドレスの後ろに書いています）
- VRF・回線監視・NAT が running-config のどこに出るか。VRF はインタフェースの前、回線監視はインタフェースの後ろで経路の前、NAT はアクセスリストの後ろに書いています（使われる前に定義がある順）
- `vrf definition` の下の `description`（取り込みと書き出しの両方にありますが、読んだ資料に構文の行がありません）
- `ip nat inside source static tcp|udp <内側> <ポート> interface <IF> <ポート>` の形。コマンドリファレンスの構文の行が、この箇所で崩れています（`interface` の語があることだけ読めます）
- IP SLA の `timeout` が `threshold` より短いと受け付けられないこと。資料には「threshold は timeout を超えられない」とあり、そこから `threshold` を先に書いています。実機では確かめていません
- `track` を、それを使う `ip route … track` より先に書く必要があるか（先に書いています）
- OSPF の `area … range` の既定が「広告する」であること（構文の行で `advertise` が省けることから、既定値として入れています。既定を述べた文は見つけていません）
- `no standby <グループ> preempt`・`no vrrp <グループ> preempt` が running-config に出るか（入力としては受け付けられる形です。VRRP は既定が有効なので、無効にした機器では出るはずです）
- `router bgp` の直下に書いたネイバーを、IOS XE が `address-family ipv4` の下に移して表示すること（どちらの形でも読めます）
- VRRPv3 の形（`fhrp version vrrp v3` と `vrrp <グループ> address-family ipv4`）は、取り込みも書き出しもしていません
- 注記の行に番号を付ける書き方（`15 remark …`）は IOS XE 16.12 以降の設定ガイドにあるもので、それより前の版で受け付けられるかは確かめていません。行が 10 刻みのリストでは番号を書かないので、当たりません
- 名前が番号のリストを `ip access-list standard 10` と入れたとき、機器が `access-list 10 …` と `ip access-list standard 10` のどちらで出すか
- running-config の体裁（先頭行、`!` の入り方、字下げ 1 桁、`end`、banner の区切り）。資料の出力例で見ただけです
- 1 行に並ぶオプションの順（`ip route` の `name` と `track` など。どちらの順でも読めます）
- アクセスリストの番号が出る版と出ない版の境目
- `ip sla` の `frequency` と `timeout` が、`icmp-echo` の下に出るか横に出るか（両方を読めます）
- 番号リスト（拡張）のポート付きの形
- `crypto map` の下の行の並び（どの順でも読めます）
- 端末ログ: 機器がページ送りの跡をどう消すか（後退の文字で消すのか、空白を重ねるのか）。資料に書かれていません。制御文字が残っているログも、残っていないログも読めるようにしてありますが、実機のログでは確かめていません
- 端末ログ: 行ごとに時刻が付くログ（端末ソフトの設定）は読めません
- 書き出し: 書き出した文面を実機に入れて確かめていません。とくに、ハッシュの始まりと種別の番号の対応（資料の例と Cisco Community の記事で見た範囲です）、`line vty 0 4` の範囲が機種に合うか、バナーの区切りの文字、すべての有効な IF に `no shutdown` を書くことの影響
- QoS の表示の形（cmdref 7.16・7.17）: 既定の `match-all` が出るか、DSCP・precedence が名前と数のどちらで出るか、`match ip dscp` の `ip` が付くか、`police` が 1 行か複数行か、`shape`・`police` の値が bps に直って出るか、クラスの下の動作の順。取り込みはどの形でも読みます
- `traffic-shape rate` の値の単位と、省いたバーストが表示で補われるか（4 つ目の数（バッファの上限）は読みません）。`rate-limit` の条件と bps の並び（「条件の後ろに bps」として読みます）。隠されたコマンド（`priority-group`・`custom-queue-list`）が 15.x の running-config に残るか
- 二段のポリシーを、サブインタフェースやトンネルで今の IOS XE でも要るか（古いテックノートでは要る）
