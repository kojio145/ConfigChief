# Cisco IOS / IOS-XE パック（cisco_ios）

Cisco IOS 15 M&T と IOS-XE 16・17 の running-config を読み取り、設計データからコンフィグを書き出すための決まりです。ルータのほか、ルータ内蔵のスイッチポートと Catalyst 系のスイッチ本体の VLAN・スイッチの行も扱います（「VLAN とスイッチ」の節）。

- 版: 0.2.0
- 読める行と入る表は「対応一覧」、書き方は「コンフィグの書き出し」の節です。「対応一覧」の表は、どれも書き出せます
- **実機の出力では確かめていません**。構文はすべて下の公開資料で確かめたものです。画面に出る語の順番や大文字・小文字は、資料の出力例で見た範囲にとどまります

## 参照した資料

どれも Cisco の公開ページで、2026-11-14 から 2026-11-21 の間に読みました。

### 技術別のコマンドリファレンス

IOS 15 M&T と IOS-XE のルータで共通のものです。右の列は、章ごとの PDF で読み直した所と、そこで確かめた行です（かっこの中の日付は、その章の最終更新）。

| 資料の名前 | 最終更新 | 章ごとに読んだ所 |
|---|---|---|
| Cisco IOS Interface and Hardware Component Command Reference | 2023-10-05 | `tunnel destination`、`tunnel mode`、`tunnel source`、`tunnel vrf`。`tunnel mode` の語の一覧（EtherIP が無いこと）は、検索結果に出た要約だけで見ました（章の本文は開けませんでした） |
| Cisco IOS IP Addressing Services Command Reference | 2018-01-11 | `ip dhcp snooping`、`ip nhrp …`（DMVPN を表にしないと決めるために読みました）。`ip name-server [vrf <VRF>] <アドレス> …` は、検索結果に出た要約だけで見ました。章「ip dhcp-client network-discovery through ip nat sip-sbc」（2018-01-11。`ip name-server` に IPv6 のアドレスを書ける） |
| Cisco IOS IP Application Services Command Reference | 2020-04-26 | 章「threshold metric through track timer」（2011-07-19。`track ip sla` の `state`・`reachability`、`track ip route`、`track list boolean`。回線監視の置き換えにも使いました）、章「ip tcp adjust-mss through ipv6 tcp adjust-mss」。同じ章の `track ip route` の `ipv6` の形（`track <番号> ipv6 route <プレフィックス> reachability`） |
| Cisco IOS IP Routing: Protocol-Independent Command Reference | 2018-04-22 | 章「A through R」「S through T」（ルートマップ、ポリシールーティング、`ip route` の `tag`・`permanent`） |
| Cisco IOS IP Routing: OSPF Command Reference | 2024-04-30 | 3 章とも。章「ip ospf fast-reroute per-prefix through R」（2014-10-08。`log-adjacency-changes`） |
| Cisco IOS IP Routing: BGP Command Reference | 2019-11-21 | 章「A through B」「C through I」「M through N」「O through show bgp」「show ip through Z」（ルートマップ・AS パスのリスト・コミュニティのリスト、`neighbor … send-community`。章「M through N」は、`neighbor activate` と `address-family ipv6` の例にも使いました） |
| Cisco IOS First Hop Redundancy Protocols Command Reference | 2020-06-21 | 章「A through I」（`fhrp version vrrp v3`: 「When VRRPv3 is in use, VRRPv2 is unavailable」）、章「Standby arp gratuitous through track vrrp」（`track (VRRP)`: `track <番号> [decrement <数>]`。VRRP の版 3 のグループの下）、章「VRRP authentication through VRRS mac-address」（古い形の `vrrp <グループ> timers advertise`。2026-11-21 に読みました） |
| Cisco IOS IP SLAs Command Reference | 2018-02-25 | `icmp-echo`（宛先は IPv4・IPv6 のアドレスか名前。`source-ip` に IPv6 を書けるかは書かれていません） |
| Cisco IOS Security Command Reference: Commands A to C | 2021-01-11 | `access-list (IP extended)` の項（ICMP・ポート・precedence・ToS の名前の一覧）。章「crypto aaa attribute list through crypto ipsec transform-set」「crypto isakmp aggressive-mode disable through crypto mib topn」（`crypto ipsec transform-set` の語の表（`esp-gcm`）、`crypto ipsec nat-transparency`、`crypto ipsec df-bit`、`crypto ipsec security-association …`、`crypto isakmp aggressive-mode disable`、`crypto isakmp invalid-spi-recovery`、`crypto isakmp identity`、`crypto isakmp key`、`crypto keyring`、`crypto isakmp profile`、`crypto ikev2 policy`・`proposal`（値の並び）、`crypto gdoi group`、`crypto map`） |
| 同 Commands D to L | 2019-12-17 | 章「database archive through dns」（`deny`・`deny (IP)`: ポートの名前と番号、ToS の名前と番号、TCP のフラグ）。`interface virtual-template`、`identity local`、`ivrf`、`keyring`、`key config-key password-encryption` |
| 同 Commands M to R | 2013-01-11 〜 2014-08-27 | 章「pac key through port-misuse」（`permit`: precedence・ToS の名前と番号）。`match identity`、`match (IKEv2 profile)`、`match (IKEv2 policy)`、`pre-shared-key`、`password encryption aes`、`proposal`、`ppp authentication`・`ppp authentication ms-chap-v2` |
| 同 Commands S to Z | 2024-08-28（章の PDF には、版の表記が無いものがあります） | 章「sa ipsec through sessions maximum」（`service password-encryption`）、「set aggressive-mode client-endpoint through show content-scan」、「traffic-export through zone security」（`username` の `password <種別> <値>`）。`set nat demux`、`set peer (IPsec)`、`set transform-set`（6 つまで）、`set security-association idle-time`、`tunnel protection`、`tunnel protection ipsec policy`、`virtual-template (IKEv2 profile)`、`vrf (isakmp profile)` |
| Cisco IOS SNMP Support Command Reference | 2017-11-13 | 章「snmp-server engineID local through snmp trap link-status」（2017-11-13。`snmp-server host` の宛先は IPv4 か IPv6 のアドレス） |
| Cisco IOS Basic System Management Command Reference | 2019-12-13 | 章「A through M Commands」（`absolute`）・「N through T Commands」（`show time-range` の出力例）（2020-06-10）。章「N through T Commands」（2011-07-20。`ntp server [vrf <VRF>] <IPv4 アドレス>\|<IPv6 アドレス>`） |
| Cisco IOS Configuration Fundamentals Command Reference | 2011-09-28 | 章「A through B」（`banner exec`・`banner login`・`banner motd`）、「R through setup」（`service timestamps`）、「monitor event-trace through Q」（`periodic`）。`errdisable recovery`（2019-12-16） |
| Cisco IOS Embedded Syslog Manager Command Reference | 2017-09-06 | 「A through Z Commands」（2011-07-27。`logging host ipv6 <アドレス>`。IPv6 の形に VRF は無い） |
| Cisco IOS Broadband Access Aggregation and DSL Command Reference | 2020-06-09 | `bba-group pppoe`、`pppoe enable [group <名前>]`、`pppoe-client dial-pool-number` の後ろの語 |
| Cisco IOS Multiprotocol Label Switching Command Reference | 2020-06-10 | — |
| Cisco IOS Dial Technologies Command Reference | 2017-09-20 | `peer default ip address`、`ppp ipcp dns`、`ip local pool` |
| Cisco IOS Terminal Services Command Reference | 2016-07-29 | — |
| Cisco IOS IPv6 Command Reference | 2019-12-13 | 章「a to clear ipv6 mld」「d to im」「ipv6 a to ipv6 g」「ipv6 mo to ipv6 ospf da」「ipv6 ospf de to ipv6 sp」「ipv6 su to m」「n to re」「ro to show bgp la」「sn to v」。章「ipv6 ospf de to ipv6 sp」（`ipv6 ospf encryption`: SPI は 256〜4294967295）、「ipv6 su to m」（`match ipv6 next-hop prefix-list`） |
| Cisco IOS Quality of Service Solutions Command Reference | 控えていません | 章「match access-group through mls ip pbr」（`match class-map`・`match cos`・`match input-interface`・`match not`・`match protocol`）、qos-n1 の章（`police` の 1 つの速さ・2 つの速さ・percent の形と動作の語）、qos-q1 の章（`queue-limit` の単位、`random-detect`）、qos-s1 の章（`set cos`・`set qos-group`・`shape` の average・peak・percent）。`match qos-group <0〜99>` は、検索結果に出た要約だけで見ました |
| Cisco IOS Wide-Area Networking Command Reference | 2015-04-05 | L2TPv3 の `l2tp-class`・`pseudowire-class`・`xconnect` と、その下の行 |
| Cisco IOS VPDN Command Reference | 2019-12-16 | `vpdn enable`・`vpdn-group` と、その下の行 |
| Cisco IOS LAN Switching Command Reference | 2020-06-15 | `encapsulation dot1q`（`second-dot1q` も）、`vlan`、`spanning-tree`。`mac-address-table aging-time` の古い書き方と `vlan internal allocation policy`（2014-11-25） |
| Cisco IOS Bridging Command Reference と、Configuring Transparent Bridging（Bridging and IBM Networking Configuration Guide） | 2009-12 ／ 2007-09-09 | `bridge <番号> protocol`・`priority`、IF の下の `bridge-group` の語 |
| Cisco IOS Cisco Discovery Protocol Command Reference | 2011-09-28 | ルータの `cdp run`・`cdp enable`・`cdp timer`・`cdp holdtime` |

### 設定ガイド・技術資料

| 資料の名前 | 最終更新 | 使った所 |
|---|---|---|
| MPLS VPN VRF CLI for IPv4 and IPv6 VPNs（MPLS: Layer 3 VPNs Configuration Guide, Cisco IOS Release 15M&T） | 版の表記なし | — |
| Using the Cisco IOS Command-Line Interface（Configuration Fundamentals Configuration Guide, Cisco IOS Release 15.1S） | 版の表記なし | 端末ログの形だけに使いました |
| Controlling Switch Access with Passwords and Privilege Levels（Security Configuration Guide, Cisco IOS XE 17.10.x） | 版の表記なし | ハッシュの種別の番号だけに使いました |
| Troubleshoot Border Gateway Protocol Routes that Do Not Advertise（Cisco の技術資料。Document ID 19345） | 2023-08-07 | `mask` の無い BGP の `network` の扱い |
| Cisco IOS XE Catalyst SD-WAN Qualified Command Reference Guide の章「BGP Commands」 | 2026-10-19 | `neighbor password [type] string` の種類 7 |
| CLI Book 1: Cisco Secure Firewall ASA Series General Operations CLI Configuration Guide, 9.23 の章「Addresses, Protocols, and Ports」（表「TCP and UDP Ports」） | 2026-10-26 | **別の製品の資料**で、UDP のポートの名前の番号だけに使いました（IOS のコマンドリファレンスは、UDP の名前を番号なしで並べています） |
| Mapping of Address and Port Using Encapsulation（IP Addressing Configuration Guide, Cisco IOS XE 17.x）と、同じ名前の章（IP Addressing: NAT Configuration Guide, Cisco IOS Release 15M&T。2015-11-23） | 2022-11-15 | MAP-E を表にしない判断だけに使いました（2 つの版の書き方が食い違っています） |
| VRRPv3 Protocol Support（Network Services Configuration Guide, Cisco IOS XE 17.x） | 版の表記なし | VRRP の版 3 の手順と例（`vrrp <グループ> address-family ipv4\|ipv6`、`address <アドレス> primary\|secondary`、`priority`、`preempt delay minimum`、`no preempt`、`timers advertise <ミリ秒>`（100〜40000）、`vrrpv2`、`exit-vrrp`）、既定値（優先度 100・切り戻し有効・1000 ミリ秒）、「VRRPv3 protocol does not support authentication」、IPv6 のグループにリンクローカルの primary が要ること。2026-11-21 に読みました |
| IPv6 Routing: OSPFv3 Authentication Support with IPsec（IP Routing: OSPF Configuration Guide） | 2026-10-14 | `ipv6 ospf authentication` と `area … authentication` の構文と例 |
| Layer 2 Tunneling Protocol Version 3（LAN and WAN Configuration Guide, Cisco IOS XE 17.x） | 版の表記なし | L2TPv3 の手順の構文と設定例 |
| Configuring NAS-Initiated Dial-In VPDN Tunneling（VPDN Configuration Guide, Cisco IOS XE 16） | 2016-02-09 | LAC・LNS の手順の構文と設定例 |
| IEEE 802.1Q-in-Q VLAN Tag Termination（LAN and WAN Configuration Guide, Cisco IOS XE 17.x） | 版の表記なし | 二重タグ（`encapsulation dot1q … second-dot1q`） |
| Configuring Internet Key Exchange Version 2 and FlexVPN Site-to-Site（FlexVPN and Internet Key Exchange Version 2 Configuration Guide, Cisco IOS XE Fuji 16.7.x） | 版の表記なし | IKEv2 の手順と設定例 |
| FlexVPN and Internet Key Exchange Version 2 Configuration Guide, Cisco IOS XE Gibraltar 16.12.x | 版の表記なし | IKEv2 の仮想テンプレートの例、`match fvrf`・`ivrf` の説明 |
| Appendix: IKEv2 and Legacy VPNs（Security and VPN Configuration Guide, Cisco IOS XE 17.x） | 版の表記なし | クリプトマップと動的なマップに `set ikev2-profile` を書いた例だけに使いました |
| Security and VPN Configuration Guide, Cisco IOS XE 17.x「IPv6 Virtual Tunnel Interface」「Session Initiation Protocol Triggered VPN」 | 2021-01-11 | IPv6 の例の食い違い、`nat force-encap` |
| Security for VPNs with IPsec Configuration Guide, Cisco IOS XE 16「IPsec Virtual Tunnel Interfaces」 | 2016-02-16 | 仮想テンプレートの設定例、VRF を仮想テンプレート・ISAKMP プロファイルに書いた例 |
| Internet Key Exchange for IPsec VPNs Configuration Guide, Cisco IOS XE 16「VRF-Aware IPsec」 | 2016-02-16 | キーリング・ISAKMP プロファイル・クリプトマップの例 |
| Security for VPNs with IPsec Configuration Guide, Cisco IOS XE Everest 16.6 | 版の表記なし | トランスフォームの語の表（GCM の決まり。SHA-2 の ESP の語は、この表にありません） |
| L2TP IPsec Support for NAT and PAT Windows Clients（Security for VPNs with IPsec Configuration Guide, Cisco IOS Release 15M&T） | 版の表記なし | L2TP/IPsec の受け口の設定例（トランスポートモードのトランスフォームセット・動的なクリプトマップ・`set nat demux`） |
| Migration from IPsec Static Crypto Maps and Dynamic Crypto Map to IPsec Virtual Tunnel Interface – Cisco IOS XE（White paper） | 2021 | 設定例の `esp-sha256-hmac`、IPsec プロファイルの下の `set isakmp-profile` |
| Configure a Multi-SA Virtual Tunnel Interface on a Cisco IOS XE Router（Document ID 214728） | 2022-07-22 | トンネル IF の `tunnel protection ipsec policy ipv4 <リスト>` の設定例 |
| Cisco cBR Converged Broadband Routers Layer 2 and Layer 3 VPN Configuration Guide for Cisco IOS XE Bengaluru 17.6.x「IPsec Security Support」 | 版の表記なし | `crypto ipsec transform-set <名前> esp-aes esp-sha256-hmac` の手順の行 |
| Cisco の「Network as Code」のデータモデルのページ（IOS-XE の crypto） | 版の表記なし | トランスフォームセットの ESP の認証の選択肢（`esp-sha384-hmac`・`esp-sha512-hmac`）。**この 2 語は、コマンドリファレンスの語の表にはありません**（「確かめていないこと」の IPsec） |

### Catalyst 系のスイッチと、内蔵スイッチ付きのルータ

| 資料の名前 | 最終更新 | 使った所 |
|---|---|---|
| Command Reference, Cisco IOS XE 17.12.x（Catalyst 系のスイッチ。Interface and Hardware・Layer 2/3・VLAN・Security・Network Management・Stack Manager and High Availability の章） | 2023-07-28 | `vlan` の並び・`state`・`private-vlan`、`switchport trunk allowed vlan` の語、`switchport port-security mac-address`・`aging`、`storm-control`、`spanning-tree guard`・`cost`・`port-priority`・`link-type`・`bpdufilter`・`loopguard`・`pathcost`・`mst`、`channel-group … non-silent`・`channel-protocol`・`port-channel load-balance`・`lacp`、`speed`、`vtp`、`switch … priority`・`renumber`、`errdisable`、`monitor session`、`lldp` |
| VLAN / Layer 2 / IP Routing / Security / Network Management / Interface and Hardware / System Management / IP Addressing / Stacking and High Availability Configuration Guide, Cisco IOS XE 17.12.x（Catalyst 系のスイッチ） | 2023-07-28 | VLAN・トランク・音声 VLAN・スパニングツリーの追加機能・IP ルーティング、スパニングツリーの優先度の刻み、MST、プライベート VLAN、UDLD、LLDP、CDP、DHCP スヌーピング、ARP 検査、SPAN、保護ポート、MAC アドレステーブル、システム MTU |
| Layer 2 Configuration Guide, Cisco IOS XE 17.9.x（Catalyst 系のスイッチ。Optional Spanning-Tree Features） | 2022-07-29 | PortFast の `edge` の語の出方だけに使いました |
| ルータに差すレイヤ 2 のスイッチモジュールの設定ガイド（Layer 2 Gigabit EtherSwitch の Configuration Guide） | 2017-12-21 | ルータ内蔵のスイッチポートの行の並び |
| 内蔵スイッチ付きの小型ルータのソフトウェア設定ガイド: Cisco 800 Series Integrated Services Routers Software Configuration Guide（Product Overview、Basic Router Configuration、Configuring Ethernet Switches、Configuring a LAN with DHCP and VLANs の章）と、Cisco 1000 Series Integrated Services Router Software Configuration Guide（Configure ethernet switch ports、Basic Router Configuration の章） | 2016-12-30 / 2026-04-28 | ポートの名前・既定の設定・VLAN の作り方・設定例 |

## 確かめた機種と版

- 実機: なし
- 資料: 上のコマンドリファレンスは版を分けていません。IOS-XE 16 にも当てはまると明記したページは見つかっていません
- Catalyst 系のスイッチは、別系統のコマンドリファレンス（上の「Catalyst 系のスイッチと、内蔵スイッチ付きのルータ」）で確かめた、「VLAN とスイッチ」の節の行だけを扱います

## 対応一覧

取り込める行と、入る表の一覧です。ここにある表は、どれも書き出せます（決まりは「コンフィグの書き出し」の節）。読み方の細かい決まりは「読み方の決まり」の節です。

見本は `tests/import/`（取り込み）と `tests/render/`（書き出し）のフォルダにあります。`_adapt`・`_failed`・`_unused` の付いたフォルダは、置き換え・止まる場合・使われなかった値の見本です。

設計データの表のうち、SNMP ユーザ（snmp_users）とログの対象ごとの重要度（log_subsystems）は、このパックが読まず、書かない表です（値があれば G-602 で知らせます）。

### 機器の基本

| 分野 | 読める行 | 入る表 |
|---|---|---|
| 機器全体 | `hostname`、`ip domain name` / `ip domain-name`、`ip name-server <アドレス>`、`clock timezone`、`enable secret`、`snmp-server location` / `contact` | devices |
| 時刻の付け方 | `service timestamps log\|debug uptime`、`service timestamps log\|debug datetime [msec] [localtime] [show-timezone] [year]` | devices の「ログの時刻の付け方」「ログの時刻に足すもの」「デバッグの時刻の付け方」「デバッグの時刻に足すもの」 |
| パスワードの暗号化 | `service password-encryption`、`no service password-encryption` | devices の「パスワードを暗号化して保存」 |
| 機器内のログ | `logging buffered [<バイト>] [<重要度>]`、`no logging buffered` | devices の「機器内にログを溜める」「機器内ログの大きさ」「ログに残す重要度」 |
| コンソール | `line con 0` の下の `exec-timeout <分> [<秒>]`（ほかの行は「読めなかった」のままです） | devices の「コンソールの無操作切断（分）」「（秒の端数）」 |
| バナー | `banner motd`、`banner login`、`banner exec` | devices の「バナー（接続したとき）」「バナー」「バナー（ログインの後）」 |
| DNS サーバの VRF | `ip name-server vrf <VRF> <アドレス> …` | devices の「DNS サーバ」「DNS サーバの VRF」 |
| ユーザ | `username <名前> [privilege <数>] secret <種類> <値>`、`… secret <平文>` | users |
| 元に戻せる形のユーザ | `username <名前> [privilege <数>] password [0] <値>`、`… password 7 <値>` | users（「元に戻せる形で持つ」が有効。`7` は「パスワードの形」が encrypted） |
| リモート接続 | `line vty` の下の `transport input`、`login local` / `login`、`access-class <リスト> in`、`exec-timeout <分> 0`、`password`。`ip ssh version` | remote_access |
| 無操作切断の秒 | `line vty` の下の `exec-timeout <分> <秒>` | remote_access の「無操作切断（秒の端数）」 |
| 暗号化した形の秘密 | `line vty` の下の `password 7 <値>`、`ip ospf authentication-key 7 <値>`、`ip ospf message-digest-key <番号> md5 7 <値>`、`ppp chap password 7 <値>`、`ppp pap sent-username <名前> password 7 <値>` | それぞれの秘密の列に、`7 <値>` の形のまま |
| NTP | `ntp server <アドレス> [prefer] [source <IF>]`、`ntp source <IF>` | ntp_servers |
| syslog | `logging host <アドレス>` / `logging <アドレス>`、`logging facility`、`logging trap`、`logging source-interface` | syslog_servers |
| SNMP | `snmp-server community <名前> ro\|rw [<リスト>]`、`snmp-server host <アドレス> [version 1\|2c] <コミュニティ>`、`snmp-server trap-source <IF>` | snmp_communities、snmp_hosts |

### インタフェース・トンネル・PPPoE

| 分野 | 読める行 | 入る表 |
|---|---|---|
| インタフェース | `description`、`ip address <アドレス> <マスク> [secondary]`、`ip address dhcp` / `negotiated`、`ip unnumbered <IF>`、`no ip address`、`ipv6 address`、`shutdown`、`vrf forwarding` / `ip vrf forwarding`、`encapsulation dot1Q <番号>`、`mtu`、`ip mtu`、`ip tcp adjust-mss`、`bandwidth`、`speed`、`duplex`、`ip access-group <リスト> in\|out`、`ip nat inside\|outside`、`ip helper-address`、`no ip proxy-arp` | interfaces |
| トンネル（GRE など） | `tunnel source`、`tunnel destination`、`tunnel key`、`tunnel mode ipip` / `tunnel mode ipsec ipv4`（行が無ければ GRE）、`keepalive <秒> <回数>` | tunnels |
| PPPoE | 物理側の `pppoe-client dial-pool-number <番号> [service-name <名前>]`。Dialer の下の `encapsulation ppp`、`dialer pool <番号>`、`ppp authentication chap\|pap\|chap pap callin`、`ppp chap hostname` / `password`、`ppp pap sent-username … password …` | pppoe |
| PPPoE（つなぎ方の細かい指定） | `pppoe-client dial-pool-number <番号> [dial-on-demand] [restart <秒>] [service-name <名前>\|"<空白のある名前>"] [mac-override]`、Dialer の下の `ppp authentication chap\|pap\|chap pap`（`callin` 無し）、`dialer-group <番号>` と `dialer-list <番号> protocol ip permit\|list <ACL の番号>` | pppoe の「通信があるときだけつなぐ」「つなぎ直すまでの待ち」「サービス名」「MAC アドレスを論理 IF のものにする」「相手も認証する」「つなぐきっかけの通信」 |

### 経路・VRF・OSPF・BGP・冗長化

| 分野 | 読める行 | 入る表 |
|---|---|---|
| スタティックルート | `ip route [vrf <VRF>] <宛先> <マスク> <次ホップ>\|<IF> [<距離>] [name <名前>] [track <番号>]` | static_routes |
| スタティックルート（`dhcp`・`global`・`multicast`） | `ip route … [<IF>] dhcp [<距離>]`（DHCP でもらったゲートウェイ）、`global`（VRF の経路のネクストホップを全体の経路表で引く）、`multicast` | static_routes（next_hop が `dhcp`、next_hop_global、multicast） |
| VRF | `vrf definition <名前>` / `ip vrf <名前>` の下の `rd`、`description` | vrfs |
| OSPF | `router ospf <番号> [vrf <VRF>]` の下の `router-id`、`auto-cost reference-bandwidth`、`network <アドレス> <ワイルドカード> area <エリア>`、`area <エリア> stub\|nssa [no-summary]`、`area <エリア> range`、`redistribute`、`default-information originate [always]`、`passive-interface default` / `<IF>`。インタフェースの下の `ip ospf <番号> area <エリア>`、`ip ospf cost` / `priority` / `hello-interval` / `dead-interval` / `network` / `authentication` / `authentication-key` / `message-digest-key` | ospf_processes、ospf_interfaces、ospf_areas、ospf_ranges、ospf_redistribute |
| OSPF（隣接の変化の記録） | `log-adjacency-changes detail`、`no log-adjacency-changes` | ospf_processes（log_adjacency） |
| BGP | `router bgp <AS>` の下、`address-family ipv4` の下、`address-family ipv4 vrf <VRF>` の下の `bgp router-id`、`bgp log-neighbor-changes`、`neighbor <アドレス> remote-as` / `description` / `password` / `ebgp-multihop` / `update-source` / `shutdown` / `next-hop-self` / `soft-reconfiguration inbound` / `prefix-list <名前> in\|out` / `route-map <名前> in\|out`、`network <アドレス> mask <マスク>`、`redistribute` | bgp_processes、bgp_neighbors、bgp_networks、bgp_redistribute |
| HSRP | インタフェースの下の `standby version`、`standby <番号> ip` / `priority` / `preempt` / `timers` / `authentication text` / `track` | fhrp |
| VRRP（インタフェースに書く古い形） | `vrrp <番号> ip` / `priority` / `preempt` / `timers advertise` / `track` | fhrp |
| VRRP の版 3（IPv4・IPv6） | `fhrp version vrrp v3`、インタフェースの下の `vrrp <番号> address-family ipv4\|ipv6` と、その下の `address <アドレス> primary`・`address <IPv6 アドレス>/<長さ>`・`priority`・`preempt delay minimum <秒>`・`no preempt`・`timers advertise <ミリ秒>`・`track <番号> [decrement <数>]`・`vrrpv2`（ipv4 だけ）・`exit-vrrp` | devices の「VRRP の版（IPv4）」、fhrp（「版 2 の相手とも話す」も）、ipv6_fhrp |
| 経路・VRF・再配布の細かい指定 | `ip route … tag <数>`・`permanent`、VRF の下（`address-family ipv4` の下も）の `route-target export\|import\|both <値>`、OSPF の `redistribute … tag <数>`、BGP の `neighbor <アドレス> send-community [standard]`、BGP の `redistribute ospf <番号>` の番号 | static_routes の tag・permanent 列、vrfs の rt_import・rt_export 列、ospf_redistribute の tag 列、bgp_neighbors の send_community 列、bgp_redistribute の process_id 列 |
| ルートマップ・BGP・OSPF のそのほかの行 | 下の「経路制御・BGP・OSPF の残り」の節 | route_policies、bgp_neighbors、bgp_processes、bgp_networks、bgp_aggregates、ospf_processes、ospf_virtual_links、ipv6_interfaces、devices |
| RIP・RIPng | 下の「RIP と RIPng」の節 | rip_processes、rip_interfaces、rip_redistribute |

### 経路制御の方針

| 分野 | 読める行 | 入る表 |
|---|---|---|
| 経路制御の方針（ルートマップ） | `route-map <名前> permit\|deny <番号>` の下の `match ip address prefix-list <名前> …`、`match ip address <ACL> …`、`match as-path <番号>`、`match community <リスト> [exact]`、`match interface <IF> …`、`match metric <数>`、`match tag <数> …`、`set local-preference`、`set metric <数>`、`set metric-type type-1\|type-2\|internal`、`set as-path prepend <AS> …`、`set community <値> … [additive]`、`set community none`、`set weight`、`set origin igp\|incomplete`、`set tag`、`set ip next-hop <アドレス> …\|self`、`set ip default next-hop <アドレス> …`、`set interface <IF> …`、`set default interface <IF> …`。インタフェースの下の `ip policy route-map <名前>`、`ip local policy route-map <名前>` | route_policies、interfaces の policy_route 列、devices の local_policy_route 列 |
| AS パスのリスト | `ip as-path access-list <番号> permit\|deny <正規表現>` | as_path_lists |
| コミュニティのリスト | `ip community-list <1〜99> permit\|deny <値> …`、`ip community-list standard <名前> …`、`ip community-list <100〜500> permit\|deny <正規表現>`、`ip community-list expanded <名前> …` | community_lists（permit の 1 行だけの標準のリストは、方針の match_community 列） |

### アクセスリスト・NAT・回線監視・DHCP

| 分野 | 読める行 | 入る表 |
|---|---|---|
| アクセスリスト | `ip access-list standard\|extended <名前>` の下の各行（番号あり・なしの両方）、`remark`。`access-list <番号> …`（標準と拡張） | acls、acl_entries |
| アクセスリストの行の細かい条件 | TCP・UDP のポートの名前（`eq www`・`range bootps bootpc` など。「読み方の決まり」の「ポートの名前」）、`established`、TCP のフラグ（`ack` `fin` `psh` `rst` `syn` `urg`。1 行に 1 つ）、ICMP の `<種類> [<コード>]`（番号）とメッセージの名前（`echo`・`echo-reply`・`unreachable`・`port-unreachable`・`time-exceeded` など 44 語）、`dscp`・`precedence`・`tos`（名前は数に直します）、`time-range <名前>`、`fragments`、`log`・`log-input`。どのプロトコルの行でも、順番を問わずに読みます | acl_entries（flags・icmp_type・icmp_code・dscp・precedence・tos・time_range） |
| 番号のアクセスリスト | `access-list <100〜199・2000〜2699> permit\|deny …` の、名前付きのリストの行と同じ形（ポート・ICMP・フラグなど） | acls（名前が番号）、acl_entries |
| 時間帯 | `time-range <名前>` の下の `periodic <曜日…> <時:分> to [<曜日…>] <時:分>`、`absolute [start <時:分> <日> <月> <年>] [end <時:分> <日> <月> <年>]` | time_ranges |
| プレフィックスリスト | `ip prefix-list <名前> seq <番号> permit\|deny <アドレス>/<長さ> [ge <数>] [le <数>]` | prefix_lists |
| NAT | `ip nat inside source list <リスト> interface <IF> overload`、`ip nat inside source list <リスト> pool <名前> [overload]`、`ip nat pool`、`ip nat inside source static <内側> <外側>`、`ip nat inside source static tcp\|udp <内側> <ポート> <外側> <ポート>`、`ip nat inside source static tcp\|udp <内側> <ポート> interface <IF> <ポート>` | nat_rules |
| NAT（`extendable`・ネットワークの静的 NAT） | 静的 NAT・ポート転送の末尾の `extendable`、`ip nat inside source static network <内側> <外側> /<長さ>\|<マスク>` | nat_rules（extendable。種類 static_network） |
| 回線監視 | `track <番号> ip sla <番号> reachability`、`track <番号> interface <IF> line-protocol`、その下の `delay`。`ip sla <番号>` の下の `icmp-echo <アドレス> [source-interface <IF>]`、`frequency`、`timeout`。`ip sla schedule <番号> life forever start-time now` | tracks |
| 回線監視（`state`・経路・まとめる監視・しきい値） | `track <番号> ip sla <番号> state`、`track <番号> ip route <宛先>/<長さ> reachability`、`track <番号> list boolean and\|or` と下の `object <番号>`、`icmp-echo <宛先> source-ip <アドレス>`、`threshold <ミリ秒>` | tracks（up_when、type が route・list、members、list_rule、source_ip、threshold） |
| DHCP サーバ | `ip dhcp pool <名前>` の下の `network`、`default-router`、`dns-server`、`domain-name`、`lease`。`ip dhcp excluded-address <開始> [<終了>]` | dhcp_pools |
| DHCP サーバ（複数のゲートウェイ・貸し出し時間） | `default-router` に並べた 2 つ目以降（8 つまで）、分まで書いた `lease <日> <時> <分>`、`lease infinite` | dhcp_pools（default_routers_more、lease） |

### IPsec

仮想テンプレート・VRF・暗号化して保存した鍵などの見本は `tests/import/ipsec_vrf`・`ipsec_third`、`tests/render/ipsec_vrf`・`ipsec_third`・`ipsec_third_failed`、2 メーカーの間は `internal/ops/testdata/ipsec_h2` です。

| 分野 | 読める行 | 入る表 |
|---|---|---|
| IPsec（IKE 版 1・トンネル IF 型） | `crypto isakmp policy <番号>` の下の `encr` / `encryption`、`hash`、`authentication pre-share`、`group`、`lifetime`。`crypto isakmp key <鍵> address <アドレス>`（`0.0.0.0` は、アドレスの決まっていない相手の鍵）・`… hostname <名前>`、`crypto isakmp keepalive <秒> [<再送の間隔>] [periodic｜on-demand]`、`crypto isakmp nat keepalive <秒>`、`crypto ipsec transform-set <名前> [ah-…] <esp-…> [<esp-…>]` とその下の `mode tunnel｜transport`、`crypto ipsec profile <名前>` の下の `set transform-set`、`set pfs`、`set security-association lifetime seconds｜kilobytes`・`set isakmp-profile <名前>`。トンネルの下の `tunnel protection ipsec profile <名前> [isakmp-profile｜ikev2-profile <名前>]`、`tunnel protection ipsec policy ipv4 <リスト>`（トンネルが相手に伝える範囲）。トランスフォームセットの語は `esp-gcm`・`esp-gcm 256`、`esp-sha256-hmac`・`esp-sha384-hmac`・`esp-sha512-hmac` も読みます。機器全体の `crypto isakmp identity address｜hostname`、`no crypto ipsec nat-transparency udp-encaps` | ipsec_tunnels、ipsec_selectors（トンネルが相手に伝える範囲）、ipsec_proposals（2 番目以降の候補）、tunnels |
| IPsec（IKE 版 1・クリプトマップ型） | `crypto map <名前> <番号> ipsec-isakmp` の下の `set peer <アドレス>`・`set peer <名前> [dynamic]`（2 行目以降は「予備の相手」）、`set transform-set <名前> …`（6 つまで）、`set pfs`、`set security-association lifetime seconds｜kilobytes`、`match address <リスト>`（リストの行は `permit ip｜udp｜tcp`、`any`・ホスト・ネットワーク、ポートは `eq <番号>`）、`set isakmp-profile <名前>`。インタフェースの下の `crypto map <名前>`。IKE のポリシー・鍵・トランスフォームセットは上と同じ | ipsec_tunnels、ipsec_selectors、ipsec_proposals |
| IPsec（動的なクリプトマップ） | `crypto map <名前> <番号> ipsec-isakmp dynamic <動的なマップ>` と、`crypto dynamic-map <名前> <番号>` の下の `set transform-set`、`set pfs`、`set security-association lifetime …`、`match address`、`set isakmp-profile`、`set ikev2-profile`。相手アドレスは `any` になります | ipsec_tunnels、ipsec_selectors |
| IPsec（ISAKMP プロファイル） | `crypto keyring <名前>` の下の `pre-shared-key address <アドレス>｜hostname <名前> key <鍵>`、`crypto isakmp profile <名前>` の下の `keyring`、`match identity address｜host｜user …`、`self-identity address｜fqdn｜user-fqdn <値>`、`keepalive <秒> retry <秒>` | ipsec_tunnels（鍵・自分の ID・相手の ID・相手監視） |
| IPsec（IKE 版 2） | `crypto ikev2 proposal <名前>` の下の `encryption`・`integrity`・`group`・`prf`（値は 1 つでも、空白で並べた 2 つ以上でも）、`crypto ikev2 policy <名前>` の下の `proposal`（2 つ目以降も）、`crypto ikev2 keyring <名前>` の下の `peer <名前>` と、その下の `address`・`hostname`・`pre-shared-key [local｜remote] [hex] <鍵>`、`crypto ikev2 profile <名前>` の下の `match identity remote address｜fqdn｜email｜key-id｜any …`、`identity local address｜fqdn｜email｜key-id <値>`、`authentication local｜remote pre-share`、`keyring [local] <名前>`、`lifetime`、`dpd <秒> <再送の間隔> on-demand｜periodic`、`nat keepalive <秒>`。機器全体の `crypto ikev2 dpd …`・`crypto ikev2 nat keepalive <秒>`。IPsec プロファイル・クリプトマップ・動的なマップの下の `set ikev2-profile <名前>` | ipsec_tunnels（並べた値は「（追加）」の列）、ipsec_proposals（2 つ目以降のプロポーザル） |
| L2TP を守る IPsec（L2TP/IPsec の受け口） | 上の IPsec の行のうち、トランスポートモードのトランスフォームセット（`mode transport`）・動的なクリプトマップ・`crypto map … ipsec-isakmp dynamic`・`address 0.0.0.0` の鍵、L2TP を選ぶアクセスリスト（`permit udp … eq 1701 …`）。LNS の行の「IPsec の設定名」が、その IPsec の行を指します | ipsec_tunnels、ipsec_selectors、l2tp_groups（IPsec の設定名） |
| 仮想テンプレート（dVTI） | `interface Virtual-Template<数> type tunnel` と、その下の `ip unnumbered`・`tunnel mode ipsec ipv4`・`tunnel source`・`tunnel protection ipsec profile`。`crypto isakmp profile` の下の `virtual-template <数>`・`match identity address 0.0.0.0`、`crypto ikev2 profile` の下の `virtual-template <数>` | IPsec の行（方式 tunnel_if・相手 any・トンネル IF が `Virtual-Template<数>`）、トンネルの行（種類 ipsec・宛先なし）、インタフェースの行 |
| トンネル IF 型で相手が名前 | `tunnel destination <名前>`（点のある名前）と、名前の鍵（`crypto isakmp key … hostname`、キーリングの `hostname`） | IPsec の行の相手アドレス（名前）、トンネルの「宛先（名前）」 |
| 外側の VRF | トンネル IF の `tunnel vrf <VRF>`、`crypto keyring <名前> vrf <VRF>`、ISAKMP プロファイルの `match identity address <アドレス> [<マスク>] <VRF>`、IKEv2 プロファイル・ポリシーの `match fvrf <VRF>｜any` | IPsec の行の「外側の VRF」（方式 policy の行は、マップを当てた IF の VRF） |
| 内側の VRF | トンネル IF の `vrf forwarding`（インタフェースの行の「VRF」）。方式 policy は、ISAKMP プロファイルの `vrf <VRF>`、IKEv2 プロファイルの `ivrf <VRF>` | インタフェースの「VRF」／IPsec の行の「内側の VRF」 |
| 暗号化して保存した鍵 | `crypto isakmp key 6 <値> …`、キーリングの `… key 6 <値>`、IKEv2 キーリングの `pre-shared-key [local｜remote] 6 <値>` | 鍵の列に `6 <値>` のまま（別紙 A 3.26） |
| クリプトマップの細かい指定 | 1 行目の `set peer <相手> default`、`set nat demux` | 「つなぎ直しは最初の相手から」「同じ NAT の後ろの端末を見分ける」 |
| NAT 越えを必ず使う（版 2） | IKEv2 プロファイルの `nat force-encap` | 「NAT 越えを必ず使う」 |
| 機器全体の有効時間 | `crypto ipsec security-association lifetime seconds｜kilobytes <数>` | 自分の有効時間を書いていない行の「IPsec 有効時間」「IPsec 有効量」 |
| 機器全体の動き | `crypto ipsec security-association replay disable`・`replay window-size <数>`、`crypto ipsec df-bit clear｜set｜copy`、`crypto ipsec security-association idle-time <秒>`、`crypto isakmp aggressive-mode disable`、`crypto isakmp invalid-spi-recovery` | IPsec の全体設定（ipsec_settings）の 1 行 |

### QoS

| 分野 | 読める行 | 入る表 |
|---|---|---|
| QoS（MQC） | `class-map [match-all\|match-any] <名前>` の下の `match access-group name <リスト>` / `<番号>`、`match [ip] dscp <値> …`（8 つまで）、`match [ip] precedence <値> …`（4 つまで）、`match any`。`policy-map <名前>` の下の `class <名前>\|class-default` と、その下の `priority [<kbps> [<バースト>]]`、`priority percent <%> [<バースト>]`、`priority level 1\|2`、`bandwidth <kbps>`、`bandwidth percent <%>`、`bandwidth remaining percent <%>`、`shape average <bps>[k\|m\|g] [<Bc> [<Be>]]`、`police [cir\|rate] <bps> [[bc] <バースト> [[be] <バースト>]] [conform-action transmit] [exceed-action drop\|transmit\|set-dscp-transmit <値>\|set-prec-transmit <値>]`（動作を下の行に書く形も）、`set [ip] dscp`、`set [ip] precedence`、`queue-limit <数> [packets]`、`fair-queue`、`service-policy <子>`。インタフェースの下の `service-policy input\|output <名前>`。拡張アクセスリストの `ip` などの行の末尾の `dscp <値>`・`precedence <値>` | qos_classes、qos_class_matches、qos_policies、qos_bindings、acl_entries の dscp・precedence 列 |
| QoS の条件 | `match [not] input-interface <IF>`、`match [not] cos <値> …`（4 つまで）、`match [not] qos-group <数>`、`match [not] protocol <名前>`、`match [not] class-map <名前>`。上の「QoS（MQC）」の条件（`access-group`・`dscp`・`precedence`）の `not` 付き | qos_class_matches の「受信 IF」「CoS」「QoS グループ」「プロトコル」「ほかのクラス」「条件を逆にする」 |
| QoS の動作 | `shape peak <bps> [<Bc> [<Be>]]`、`shape average\|peak percent <%>`、`police cir percent <%> …`、`police [cir] <bps> [bc <バイト>] pir <bps> [be <バイト>] …`、`violate-action drop\|transmit\|set-dscp-transmit <値>\|set-prec-transmit <値>`（1 行の形と、下の行に書く形）、`set cos <数>`、`set qos-group <数>`、`random-detect [dscp-based\|prec-based]`、`queue-limit <数> bytes\|ms\|us` | qos_policies の「遅らせ方」「上限・遅らせる（%）」「上限・捨てる（%）」「最大の速さ・捨てる」「最大を超えたとき」「付ける CoS」「付ける QoS グループ」「混む前に間引く」「キューの長さの単位」 |
| QoS（MQC より前の形） | インタフェースの下の `traffic-shape rate <bps> [<Bc> [<Be>]]`、`rate-limit input\|output [access-group <番号>] <bps> <通常のバースト> <最大のバースト> conform-action transmit exceed-action drop\|transmit\|set-dscp-transmit <数>\|set-prec-transmit <数>`、`priority-group <番号>`、`custom-queue-list <番号>`。`priority-list <番号> protocol ip high\|medium\|normal\|low [list <リスト> \| tcp <ポート> \| udp <ポート>]`、`priority-list <番号> default <キュー>`、`priority-list <番号> queue-limit <4 つの数>`、`queue-list <番号> protocol ip <キュー> [list <リスト> \| tcp <ポート> \| udp <ポート>]`、`queue-list <番号> default <キュー>`、`queue-list <番号> queue <キュー> [byte-count <バイト>] [limit <数>]` | qos_classes、qos_class_matches、qos_policies、qos_bindings（`tcp`・`udp <ポート>` は acls、acl_entries も） |

### L2TP

| 分野 | 読める行 | 入る表 |
|---|---|---|
| L2 延伸（L2TPv3） | `l2tp-class <名前>` の下の `authentication`、`password [0] <値>`、`hostname <名前>`、`hello <秒>`。`pseudowire-class <名前>` の下の `encapsulation l2tpv3`、`protocol l2tpv3 [<クラス>]` / `protocol none`、`ip local interface <IF>`、`ip pmtu`、`ip dfbit set`、`sequencing transmit\|receive\|both`。インタフェースの下の `xconnect <相手> <回線番号> [encapsulation l2tpv3 [manual]] pw-class <名前>` と、その下の `l2tp id <自分> <相手>`、`l2tp cookie local\|remote <大きさ> <値> [<値>]`、`l2tp hello <クラス>` | l2tpv3 |
| L2TP（LNS・LAC） | `vpdn enable`。`vpdn-group <名前>` の下の `accept-dialin`（その下の `protocol l2tp`、`virtual-template <番号>`）、`request-dialin`（その下の `protocol l2tp`、`domain <名前>`）、`terminate-from hostname <名前>`、`initiate-to ip <アドレス>`、`local name <名前>`、`l2tp tunnel password [0] <値>`、`no l2tp tunnel authentication`。`interface Virtual-Template<番号>` の下の `peer default ip address pool <名前>`、`ppp authentication chap\|pap\|chap pap\|ms-chap-v2`、`ppp ipcp dns <アドレス> [<アドレス>]`。`ip local pool <名前> <先頭> <末尾>` | l2tp_groups |
| L2TP の利用者 | LNS がある機器の `username <名前> password [0] <値>`（権限なし） | l2tp_users |
| PPPoE を受ける側（LAC の配下） | `bba-group pppoe <名前>` の下の `virtual-template <番号>`、ポートの下の `pppoe enable group <名前>`、その `Virtual-Template` の下の `ppp authentication …` | l2tp_groups（役割 lac の行）の「PPPoE を受ける IF」「受け口の IF」「利用者の認証」 |

### VLAN とスイッチ、IPv6

| 分野 | 読める行 | 入る表 |
|---|---|---|
| VLAN・スイッチポート・LAG・スパニングツリー・ブリッジ・ポートミラー・CDP・LLDP・UDLD・DHCP スヌーピング | 下の「VLAN とスイッチ」の節 | vlans、switchports、lags、l2_settings、stp_vlans、stp_instances、mirror_sessions、bridge_groups、stack_members、interfaces |
| IPv6（アドレス・RA・DHCPv6・経路・フィルタ・プレフィックスリスト・OSPFv3・BGP・トンネル） | 下の「IPv6」の節 | ipv6_interfaces、ipv6_ra_prefixes、ipv6_routes、ipv6_acl_entries、ipv6_prefix_lists、ospfv3_processes、ospfv3_interfaces、ospfv3_areas、bgp_ipv6_neighbors、bgp_processes、tunnels、devices、interfaces |
| IPv6（TCP MSS・再配布の指定） | `ipv6 tcp adjust-mss <数>`、OSPFv3 の `redistribute connected\|static\|bgp <AS> [metric <数>] [metric-type 1\|2] [tag <数>] [route-map <名前>]`、BGP の `address-family ipv6` の下の `redistribute connected\|static\|ospf <番号> [metric <数>] [route-map <名前>]` | ipv6_interfaces（tcp_mss）、ospfv3_redistribute、bgp_ipv6_redistribute |
| IPv6（DHCPv6 サーバが配るプレフィックスとアドレス・OSPFv3 の認証） | 下の「IPv6」の節 | ipv6_interfaces、ospfv3_interfaces、ospfv3_areas |
| IPv6 の宛先（DNS・NTP・syslog・SNMP・回線監視） | 下の「IPv6」の節の「IPv6 の宛先」 | devices、ntp_servers、syslog_servers、snmp_hosts、tracks |

## 読み方の決まり

### 全般

- **古い書き方と新しい書き方の両方を読みます。** `ip domain name` と `ip domain-name`、`vrf definition` と `ip vrf`、`vrf forwarding` と `ip vrf forwarding`、`logging host <アドレス>` と `logging <アドレス>` などです。OS の違いではなく出た時期の違いで、1 台に両方が残ることがあります
- **マスクとワイルドカードは、長さの形に直して入れます。** とびとびのワイルドカード（長さに直せないもの）の行は「読めなかった」になります
- **全体に 1 行だけ書く設定**（`ntp source`、`logging facility`、`logging trap`、`logging source-interface`、`snmp-server trap-source`）は、それぞれのサーバの行に同じ値を入れます

### 機器の基本

- **`password` で書いたユーザは「元に戻せる形で持つ」が有効になります。** `secret` はハッシュで保存され、PPP の CHAP の相手として使えないため、`password` で書いたユーザは `password` のまま書き出します。LNS がある機器の、権限の無いこの形のユーザは「L2TP 接続ユーザ」に入ります（下の「L2TP」）
- **暗号化した形の秘密は、`7 <値>` のままセルに入ります**（別紙 A 3.26。「形」の列を持つユーザだけは、値だけを入れて形を encrypted にします）。平文に空白は入らないので、空白のある値は暗号化した形と分かります。チェック V-283 が、どのセルがその形かを知らせます
- **DNS サーバは、全部が同じ VRF（か、全部が VRF なし）です。** 先に読んだ行と VRF が違う `ip name-server` の行は「読めなかった」になります

### 経路・VRF・OSPF・BGP

- **OSPF を動かすインタフェース。** `network` の行にアドレスが入るインタフェースと、`ip ospf <番号> area` を書いたインタフェースを、ospf_interfaces の行にします。どのインタフェースにも当たらない `network` の行は「読めなかった」に戻ります
- **`route-target both`** は、取り込むルートターゲットと付けるルートターゲットの両方に入れます
- **BGP の `redistribute ospf <番号>`** は、番号を「OSPF のプロセス番号」の列に入れます
- **ルートマップのコミュニティの条件。** `match community <リスト>` が指すリストが、permit の 1 行だけの標準のリスト（「この値をすべて持つ経路」）のときは、その値を方針の行（route_policies の match_community 列）に入れ、**リストはコミュニティのリストの表から外します**（別紙 A 3.19。設計データは、コミュニティをじかの値で持ちます）。2 行以上のリスト、deny の行、正規表現のリストは、リストのまま残して、方針の行には名前（match_community_list 列）が入ります。`exact` は `exact-match` と書いた形も読みます
- **AS パスのリスト・コミュニティのリストの行の順番。** 機器は行に番号を持たないので、書いてある順に 10、20、30 … と番号を付けます
- ルートマップ・BGP・OSPF のそのほかの決まり（ピアグループ、`mask` の無い `network`、`set comm-list … delete` など）は、下の「経路制御・BGP・OSPF の残り」の節にあります

### アクセスリスト・NAT・回線監視・DHCP

見本は `tests/import/acl_nat/`、`tests/render/acl_nat/`、`tests/render/acl_adapt/` です。

- **アクセスリストの番号。** 番号が画面に出るかどうかは版によって違うため、両方を読みます。番号の無い行は、行の順に 10、20、30 … と番号を付けます
- **ポートの名前**は、番号に直して入れます（設計データは番号で持ちます）。番号を資料で確かめられた名前だけです。TCP は、コマンドリファレンスの `deny` の項にある一覧（`bgp` 179、`chargen` 19、`cmd` 514、`daytime` 13、`discard` 9、`domain` 53、`echo` 7、`exec` 512、`finger` 79、`ftp` 21、`ftp-data` 20、`gopher` 70、`hostname` 101、`ident` 113、`irc` 194、`klogin` 543、`kshell` 544、`login` 513、`lpd` 515、`nntp` 119、`pim-auto-rp` 496、`pop2` 109、`pop3` 110、`smtp` 25、`sunrpc` 111、`syslog` 514、`tacacs` 49、`talk` 517、`telnet` 23、`time` 37、`uucp` 540、`whois` 43、`www` 80）。UDP は、IOS のコマンドリファレンスが名前を番号なしで並べているので、ASA の設定ガイドの表にも同じ名前がある語だけです（`biff` 512、`bootpc` 68、`bootps` 67、`discard` 9、`dnsix` 195、`domain` 53、`echo` 7、`isakmp` 500、`mobile-ip` 434、`nameserver` 42、`netbios-dgm` 138、`netbios-ns` 137、`ntp` 123、`pim-auto-rp` 496、`rip` 520、`snmp` 161、`snmptrap` 162、`sunrpc` 111、`syslog` 514、`tacacs` 49、`talk` 517、`tftp` 69、`time` 37、`who` 513、`xdmcp` 177）。**書き出しは番号です**（機器は、番号を名前に直して表示します）
- **ICMP のメッセージの名前**は、語のまま「ICMP の種類」に入れます（NEC IX も同じ語を使います）。種類とコードを番号で書いた行は、番号のまま入れます
- **ToS の名前**は数に直します（`normal` 0、`min-monetary-cost` 1、`max-reliability` 2、`max-throughput` 4、`min-delay` 8）
- **時間帯の曜日**は、小文字にして入れます（`Monday` → `monday`）。日付は `2026-01-31` の形に直します
- **NAT の規則の名前。** 機器に名前が無いため、行の順に `nat1`、`nat2` … と付けます
- **`track … ip sla … state`** は「正常とみなす条件」が within_threshold、`reachability` は空です。**`threshold` が `timeout` と同じ値の行**は、しきい値を列に入れません（短い `timeout` を書くために、このパックが自分で書く形だからです）
- **`track … list boolean and`** は「まとめ方」all_up、`or` は any_up です。`object` の番号が「まとめる監視」に並びます
- **`lease`** は、割り切れるいちばん大きい単位にします（`lease 2` → `2d`、`lease 1 12` → `36h`、`lease 0 1 30` → `90m`）

### IPsec

- **IPsec のトンネル IF 型。** トンネルが指すプロファイル、プロファイルが指すトランスフォームセット、相手のアドレスが同じ鍵、番号がいちばん小さい IKE のポリシーをつないで 1 行にします。2〜4 番目の IKE のポリシーと、プロファイル・クリプトマップに並べた 2〜4 番目のトランスフォームセットは、「IPsec 追加の提案」（ipsec_proposals）の行になります（下の「候補を並べた設定」）。行の名前はトンネル IF の名前です。自分側（トンネルの送信元）は tunnels の行にあり、ipsec_tunnels の「自分側 IF」には入れません
- **IPsec のクリプトマップ型。** インタフェースが使っているマップの 1 項目を、ipsec_tunnels の 1 行にします。機器には項目ごとの名前が無いため、行の名前は `<マップ名>/<番号>` です。`match address` が指す拡張アクセスリストの `permit` の行が、ipsec_selectors の行になります（`ip`、または `udp`・`tcp` で、ポートは片側に 1 つまでの `eq <番号>`。`any` は 0.0.0.0/0）。**ipsec_selectors に写した行は、アクセスリストの表（acl_entries）から外します。行が残らなければ、リスト（acls）も外します**（フィルタ・NAT・`line vty`・SNMP も使っているリストは、そのまま残します）。**リストに、対象通信の行にできない行（`deny`、範囲や名前で書いたポート、`icmp` など）が 1 行でもあると、そのマップの項目は行にせず、関係する行が「読めなかった」に戻ります**（一部だけを読むと、守る範囲の違う設定になるため。注記 `remark` の行は数えません）。どのインタフェースも使っていないマップは「読めなかった」に戻ります
- **IPsec の IKE 版 2。** IPsec プロファイル（かクリプトマップの項目）が指す IKEv2 プロファイルから、相手の見分け方（相手の ID）・自分の ID・有効時間・相手監視を読みます。鍵は、プロファイルが指すキーリングの中の、相手のアドレス（か名前）が同じ `peer` から読みます（無ければ、`address 0.0.0.0` の `peer`）。`pre-shared-key local` は「事前共有鍵」、`remote` は「相手を確かめる鍵」に入ります（1 行だけの鍵は、両方に同じ鍵です）。IKE の提案は、`match` の行が無い最初のポリシーの、最初のプロポーザルです。**ポリシーが 1 つも無ければ、IKE の暗号・ハッシュ・DH グループは空**（機器が持つ候補に任せる。別紙 A 3.20）になります。`match identity remote address` が相手アドレスと同じなら、相手の ID は空（アドレスで確かめる）です。両方向とも `pre-share` のプロファイルだけを読み、証明書・EAP のプロファイルを指す行は「読めなかった」に戻ります
- **ISAKMP プロファイル（IKE 版 1）。** クリプトマップの項目が `set isakmp-profile` でプロファイルを指すとき、鍵はそのプロファイルのキーリングから、相手の ID は `match identity`（最初の 1 行）から、自分の ID は `self-identity` から読みます。`self-identity fqdn` は値を持たない行なので、機器の表の「ホスト名.ドメイン名」を自分の ID に入れます（`ip domain name` が無ければ「読めなかった」に戻ります）。プロファイルの `keepalive <秒> retry <秒>` は、機器全体の `crypto isakmp keepalive` より先に使います
- **動的なクリプトマップ。** `ipsec-isakmp dynamic <名前>` の項目は、動的なマップの最初の項目の中身（トランスフォームセット・PFS・有効時間・アクセスリスト）を持つ、相手が `any` の行になります。鍵は `address 0.0.0.0` の鍵です。アクセスリストは無くてもかまいません（相手が言ってきた範囲を受け入れる行）
- **相手を名前で書いた行。** `set peer <名前>` は、相手アドレスの列に名前のまま入れます。鍵は `crypto isakmp key … hostname <名前>`（IKE 版 2 はキーリングの `hostname <名前>`）から読みます
- **相手を複数並べたクリプトマップ。** 2 行目以降の `set peer` は、1 行目の相手につながらないときに順につなぐ相手として、「予備の相手」（peer_backup）に並べます。読むのは、IKE の版 1 で ISAKMP プロファイルを使わない項目の、**鍵が 1 行目の相手と同じ**相手だけです（表は 1 行に鍵を 1 つ持ちます）
- **相手監視（IKE 版 1）。** `crypto isakmp keepalive` の 2 つ目の数は「再送間隔（秒）」です（回数ではありません。回数の列は NEC IX の書き方のためのものです）。`periodic` は「送り方」に入り、語が無ければ機器の既定の on_demand を入れます
- **候補を並べた設定（別紙 A 3.20）。** 組み合わせを並べる書き方は、2 番目以降が「IPsec 追加の提案」（ipsec_proposals）の行になります（順番は 20・30… と振ります）: `set transform-set A B C …` の 2 番目以降（段 ipsec。6 つまで）、`crypto isakmp policy` の 2 番目以降（番号の小さい順。段 ike。**IKE の版 1 の全部の行**に同じ行が入ります。ポリシーは機器全体のものだからです）、最初の IKEv2 ポリシーの 2 つ目以降の `proposal`（段 ike。**版 2 の全部の行**に入ります）。1 つの提案に値を並べる書き方（IKEv2 のプロポーザルの `encryption aes-cbc-256 aes-cbc-128` など）は、1 つ目の値が IPsec の行、2 つ目以降がその行の「IKE 暗号（追加）」などの列に入ります。**値を並べたプロポーザルを、ポリシーの 2 つ目以降に置いた形は読みません**（そのプロポーザルの行と、ポリシーの `proposal` の行が「読めなかった」に残ります）。2 番目以降のトランスフォームセットの `mode` は読みません（その行は「読めなかった」）
- **トランスフォームセットの `mode`。** `mode tunnel`・`mode transport` を「IPsec のモード」に入れます。行が無ければ空（tunnel と同じ意味）です
- **トランスフォームセットの GCM と SHA-2。** `esp-gcm` は aes128gcm、`esp-gcm 256` は aes256gcm です（GCM は ESP ハッシュを並べられません。並べた行は「読めなかった」）。`esp-sha256-hmac`・`esp-sha384-hmac`・`esp-sha512-hmac` は ESP ハッシュの sha256・sha384・sha512 です
- **トンネル IF が相手に伝える範囲。** トンネル IF の `tunnel protection ipsec policy ipv4 <リスト>` は、「any any」の代わりに相手へ伝える範囲です。リストの行を、そのトンネルの IPsec の行の対象通信（ipsec_selectors）に入れ、リストはアクセスリストの表から外します（クリプトマップ型と同じ決まり）
- **IKE 版 1 の自分の ID。** ISAKMP プロファイルを使わない行は、機器全体の `crypto isakmp identity hostname` があれば、自分の ID を「ホスト名.ドメイン名」（`fqdn:…`）として読みます（`ip domain name` が無ければ「読めなかった」に戻ります）。`address`（機器の既定）は空のままです。プロファイルの `self-identity address` は、その行が送り出す IF（クリプトマップを当てた IF・トンネルの送信元）のアドレスを `address:<アドレス>` として入れます
- **トンネル IF 型の行の ISAKMP プロファイル・IKEv2 プロファイル。** IPsec プロファイルの下の `set isakmp-profile`・`set ikev2-profile` と、`tunnel protection ipsec profile <名前> isakmp-profile｜ikev2-profile <名前>` の後ろの指定の、どちらも読みます（同じ意味として扱います）
- **NAT 越え。** 機器は、途中に NAT を見つけると自分で UDP に包みます（`crypto ipsec nat-transparency` の既定）。そこで、**IPsec の行の「NAT 越え」は、行が無ければ既定値の true** です。`no crypto ipsec nat-transparency udp-encaps` があれば、全部の行が false になります
- **仮想テンプレートは、相手 any の 1 行です。** 機器がつないできた相手ごとに作る IF（Virtual-Access）は、表に持ちません。IKE 版 1 は、`virtual-template <数>` を持つ ISAKMP プロファイルと、そのキーリング（か機器全体）の「どの相手でも」の鍵（`address 0.0.0.0 0.0.0.0`）をつなぎます。版 2 は、`virtual-template <数>` を持つ IKEv2 プロファイルです。アドレスごとの鍵しか無いひな形は、行にしません（「読めなかった」に残ります）
- **外側の VRF は、鍵と相手の見分け方がそろっているときだけ読みます。** トンネルの `tunnel vrf` と、キーリングの `vrf`（版 1）・IKEv2 プロファイルの `match fvrf`（版 2。`any` はどの VRF でも）が同じでなければ、行にしません（機器でもつながらない組み合わせです）。機器全体の `crypto isakmp key` は VRF なしの相手の鍵なので、外側の VRF がある行の鍵にはしません
- **IKEv2 ポリシーは、行の外側の VRF に当たるものを使います**: `match fvrf <その VRF>` のポリシー、無ければ `match fvrf any` のポリシー。VRF なしの行は、`match fvrf` の無い最初のポリシー、無ければ `any` のポリシー
- **`ivrf` が外側と同じ・`vrf` が外側と同じなら、「内側の VRF」は空**（空 = 外側と同じ）です
- **`crypto isakmp key 6 <値>` の `6 <値>` は、鍵そのものではありません**（機器の中の親の鍵で暗号化した値）。セルにそのまま持ち、Cisco へはそのまま書き戻します。チェック V-283 が、どのセルがその形かを知らせます。対向の鍵の比較（X-305）は、この形のセルを比べません
- **L2TP を守る IPsec（L2TP/IPsec）。** 機器には、L2TP の設定から IPsec を指す行がありません。LNS（`accept-dialin` の `vpdn-group`）がある機器では、**「方式 policy・相手アドレス any・IPsec のモード transport」の IPsec の行のうち、名前の順で最初の行**を、LNS の行の「IPsec の設定名」に入れます（別紙 A 3.21。NEC IX の `ipsec policy transport <名前>` に当たるつながりを、表の上で持つためです）

### PPPoE

- **PPPoE。** プールの番号が同じ物理インタフェースと Dialer を 1 行にします。行の名前は Dialer の名前です（Cisco の PPPoE に名前はありません）。CHAP と PAP の両方の行があるときは、接続 ID とパスワードが同じときだけ読みます（違えば、関係する行が「読めなかった」に戻ります）
- **`dialer-group` と `dialer-list` は番号でつながります。** 番号は表に持たず、書き出しでは PPPoE の行の順に 1、2、3 … と振ります。`dialer-group` があるのに同じ番号の `dialer-list` が読めないときは、その PPPoE の行を作りません（関係する行が「読めなかった」に戻ります）
- **`ppp authentication` の `callin` の有無。** `callin` 付きは「認証方式」だけ、無しは「相手も認証する」も有効にします。同じ Dialer に両方の行があるときは、行を作りません

### QoS

- **QoS の二段のポリシー。** 親のポリシーが `class class-default` だけで、その下が `shape average` と子の `service-policy` だけのときは、親を「IF 全体を絞る速さ」として適用の行（qos_bindings）に畳みます。適用の行のポリシー名は子の名前になり、親のポリシーの行は表から外します（親の行は「取り込み済み」のままです）。それ以外の二段の形（親にほかのクラスや動作があるもの、入力の向き）は、`service-policy` の行が「読めなかった」に戻ります
- **QoS の MQC より前の形。** 機器に名前が無いものは、パックが名前を付けます。`rate-limit` はポリシー `rl-<IF の短い名前>-in|out`（`/` と `.` は `-` にします。例 `rl-Gi0-1-in`）（ACL 付きの行はクラス `rl-acl<番号>`、無い行は `default`）、`priority-list <n>` はクラスとポリシー `pq<n>`（条件の行ごとにキュー）、`queue-list <n>` はポリシー `cq<n>` とキューごとのクラス `cq<n>-<キュー>`、`tcp <ポート>`・`udp <ポート>` はリスト `pq<n>-tcp<ポート>` などです（送信元か宛先のどちらかがそのポートの通信に合う 2 行）。`default` の行は順番 9990 の条件（すべての通信）にします。`priority-list` に `default` の行が無いときは、機器の既定（normal）の条件を足します
- **カスタムキューの帯域。** キューの `byte-count`（無ければ 1500）を、使っているキューの合計で割った割合（% の整数。切り捨て、1 より小さければ 1）を帯域の % にします。機器はバイト数で順に送るので、**動きは近いものの同じではありません**。どのキューにも当たらない通信の行き先は、`default` の行が無ければ書かれていません（機器の既定はキュー 1 ですが、足していません）
- **`traffic-shape rate` の単位**は bps とビットとして読みます（コマンドリファレンスの該当の章が取れず、設定ガイドの手順から読みました。下の「確かめていないこと」）
- **QoS の DSCP・precedence の名前**（`ef`・`af41`・`critical` など）は数に直して入れます（別紙 A 5.2）。`class-default` は `default` という名前で入れます。`shape`・`police` の `k`・`m`・`g` の付いた値は bps に直します

### L2TP

- **L2 延伸（L2TPv3）は、つながった先まで読めたときだけ表に入ります。** `xconnect` → `pseudowire-class` → `l2tp-class` を名前でたどります。`encapsulation l2tpv3`・`ip local interface`・`protocol` の行がそろっていないクラス、`authentication` と `password` の片方しか無いクラス、`manual` なのに `protocol none` でないクラスは行を作らず、関係する行がすべて「読めなかった」に戻ります。1 つの `pseudowire-class` を複数の IF が使うと、同じ設定の行が IF の数だけできます（設定名が同じ行になります）
- **`l2tp-class` の名前と `ip local pool` の名前は、表に持ちません。** 書き出しでは、クラスに `pseudowire-class` と同じ名前、プールに `l2tp-<L2TP の名前>` を付けます
- **VPDN のグループは、LNS（`accept-dialin`）か LAC（`request-dialin`）のどちらかです。** 両方を持つグループ、`protocol l2tp` の無いグループ、`vpdn enable` の無い機器のグループは行を作りません。`accept-dialin`・`request-dialin` の下の行は、機器の出力どおり、グループの行より 1 段深い字下げで読みます（資料の設定例には字下げが 1 段のものがあり、その形は「読めなかった」になります）
- **LNS がある機器の、`password` で書いた権限なしのユーザは、L2TP の利用者として読みます**（ユーザの表ではなく「L2TP 接続ユーザ」に入ります。別紙 A 3.21）。CHAP にはパスワードそのものが要り、`secret`（ハッシュ）では確かめられないためです。トンネルの認証のために相手の名前で作ったユーザも、ここに入ります。LNS が 2 つ以上ある機器では、名前の順で最初の LNS に入れます。`secret` で書いたユーザと、権限（`privilege`）を持つユーザは、ユーザの表です
- **「トンネルの認証」は、`no l2tp tunnel authentication` が無ければ有効です**（機器の既定）。有効で `l2tp tunnel password` が無いグループは、「自分の名前（無ければホスト名）のユーザのパスワードを使う」の意味で、パスワードの列は空になります
- **PPPoE を受けるポートは、機器に LAC がちょうど 1 つあるときだけ読みます**（Cisco は、PPPoE のグループと VPDN のグループを結び付けません）。グループの名前は表に持たず、書き出しでは L2TP の名前にします

## 変更の投入の書き方（`change`。「ひな形から作る」の今との差）

`pack.yaml` の `change`（別紙 B 2.5）。設定モードへは `configure terminal`、抜けるのは `end`、ブロックを 1 段抜けるのは `exit`、行を消すのは `no` の形、`no` で始まる行は `no` を外して戻す（Configuration Fundamentals「Using the Cisco IOS Command-Line Interface」）。`address-family` は、ガイドの見本のコンフィグの形どおり `exit-address-family` で抜ける。`username` は `no username <名前>`、`banner` は `no banner <種類>`、`description` は `no description` で消す。

- **確かめていないこと**: `address-family` を `exit` でも抜けられるか。行の全部に `no` を付けた形が通らない命令が残っているか（実機で弾かれた行は、投入ログのチェックで分かる）。保存のコマンドは付けない

## 端末ログの読み方

端末ソフトのログ（ログインから `show running-config` まで入ったもの）は、設定の部分だけを切り出して読みます。決まりは別紙 B 2.4、書いてある場所は `pack.yaml` の `session` です。

- **プロンプト。** 行の頭の「ホスト名」「かっこに入ったモード（あれば）」「`>` か `#`」です
- **設定を表示するコマンド。** `show running-config` と `show startup-config`、その短い形（`sh run`、`sho run`、`show run`、`show start` など。`show` は `sh` まで、後ろは `run`・`start` まで縮めた形）、頭に `do` を付けた形です。後ろに語が続く形（`show running-config all`、`| include …`、`interface …` など）は、全体が出ないので設定として読みません
- **ページ送りの跡。** ` --More-- ` です。跡だけの行は「無視（ページ送りの跡）」にし、跡の後ろに設定が続く行は、跡を取り除いて読みます（そのときは注意を出します）
- 同じログにコマンドが何回もあるときは、最後のものを読みます

## 投入ログの読み方

コンフィグを流したときの端末ログから、機器に弾かれたコマンドを探します（仕様書 11.6）。機器の返事の形は `errors.yaml` にあります。

- 資料で確かめた「弾かれた」形は 3 つ（`% Invalid input detected at '^' marker.`・`% Incomplete command.`・`% Ambiguous command:`）、「受け付けた後の通知」は `% Warning:` と鍵の生成の 2 つです。ほかの `%` の返事（アドレスの重なりなど）は「パックが知らないメッセージ」として、人が確かめる指摘（L-701）になります。機器のログの行（`%LINK-3-UPDOWN: …`）は返事として数えません
- プロンプトにかっこがある行（設定のモード）で打たれたコマンドを、生成したコンフィグの行と突き合わせます

## 値を入れない規則

別紙 B 3.2 で認められている 2 つの場合だけです。

| 規則 | 行 | 理由 |
|---|---|---|
| `bgp.af` | `router bgp` の下の `address-family ipv4` | 子の行を持つ入れ物の行 |
| `if.ipv4.none` | `no ip address` | 「ipv4 の列が空」と同じ意味の行 |
| `qos.policy` | `policy-map <名前>` | 子の行を持つ入れ物の行 |
| `qos.policy.police.conform` | `police` の下の `conform-action transmit` | 機器の既定の動作で、「列が空」と同じ意味の行 |
| `isakmp.keyring` | `crypto keyring <名前>` | 子の行を持つ入れ物の行 |
| `ikev2.keyring` | `crypto ikev2 keyring <名前>` | 子の行を持つ入れ物の行 |
| `time_range` | `time-range <名前>` | 子の行を持つ入れ物の行 |
| `mst` | `spanning-tree mst configuration` | 子の行を持つ入れ物の行 |
| `ospf.log_adjacency` | `router ospf` の下の `log-adjacency-changes`（`detail` なし） | 機器の既定の動作（隣接が上がった・落ちたときに記録する）で、「隣接の変化の記録」の列が空と同じ意味の行。`detail` と `no …` は列に入ります |
| `bgp.neighbor.activate` | BGP の `neighbor <IPv4 アドレス> activate` | 機器の既定の動作（IPv4 のアドレスファミリは、ネイバーごとに初めから有効）で、「列が空」と同じ意味の行 |
| `if.vrrp3.ipv6` | インタフェースの下の `vrrp <番号> address-family ipv6` | 子の行を持つ入れ物の行（VRRP の版 3 の IPv6 のグループ） |

IPv6 の規則（`acl6`・`dhcp6.pool`・`bgp.af6`・`ospf3.new.af`）は、下の「IPv6」の節の「読み方の決まり」にあります。

## 読んで、表には入れない語

| 行 | 捨てる語 |
|---|---|
| OSPF の `redistribute … subnets` | `subnets`（入れる列がありません） |
| `l2tp-class <名前>`、`protocol l2tpv3 <クラス>`、`l2tp hello <クラス>` | クラスの名前（書き出しでは `pseudowire-class` と同じ名前にします） |
| `ip local pool <名前> …`、`peer default ip address pool <名前>` | プールの名前（書き出しでは `l2tp-<L2TP の名前>` にします） |
| `password 0 <値>`（`l2tp-class` の下）、`l2tp tunnel password 0 <値>`、`username … password 0 <値>` | 種別の `0`（平文の印。L2TP のパスワードは種別なしで、利用者は `password 0` で書き出します） |
| クリプトマップの `set peer <名前> dynamic` | `dynamic`（つなぐたびに名前を引き直す指定。入れる列がありません。書き出しでは、名前の相手にいつも付けます） |
| `crypto isakmp key 0 <鍵> …`、キーリングの `pre-shared-key … key 0 <鍵>`・`pre-shared-key 0 <鍵>` | `0`（平文の印。書き出しでは付けません） |
| `crypto ikev2 keyring` の下の `peer <名前>`、`crypto ikev2 policy <名前>`・`proposal <名前>`・`profile <名前>`、`crypto keyring <名前>`、`crypto isakmp profile <名前>`、`crypto dynamic-map <名前> <番号>` | 名前と番号（表に列が無く、書き出しではパックが決めます） |
| アドレスの後ろの `255.255.255.255`（`crypto isakmp key … address`、キーリングの `address`、`match identity … address`） | 1 台を指すマスク（書き出しでは、`match identity` にだけ付けます） |
| `tunnel protection ipsec profile <名前> isakmp-profile｜ikev2-profile <名前>` | 後ろの指定の位置（書き出しでは、IPsec プロファイルの下の `set isakmp-profile`・`set ikev2-profile` として書きます） |
| `crypto isakmp identity hostname｜address` | 行そのもの（自分の ID の列に入れ、書き出しでは、その行の ISAKMP プロファイルの `self-identity fqdn` として書きます。`address` は機器の既定なので、書きません） |
| `tunnel protection ipsec policy ipv4 <リスト>`、`match address <リスト>` | リストの名前（表に列が無く、書き出しでは `ACL-<トンネル IF>`・`ACL-<マップ名>-<番号>` にします） |
| `crypto ikev2 proposal` の 2 つ目以降の名前、`crypto isakmp policy <番号>` の番号 | 名前と番号（順番だけを持ちます。書き出しでは `IKEV2-PROP-2`…、10・20・30… と振ります） |

IPv6 の行で捨てる語は、下の「IPv6」の節の「読み方の決まり」にあります。

## 「無視」にする行

- `ignore.yaml`: `Building configuration...`、`Current configuration : … bytes`（表示上の行）、`version <数>`（機器が自動で付ける行）、`end`（終わりの印）
- `exit-address-family`（終わりの印。VRF と BGP のブロックの中）
- `exit-vrrp`（終わりの印。VRRP の版 3 のグループの中。書き出しでは必ず付けます）
- `vrf definition` の下の `address-family ipv4`（VRF で IPv4 を使う印。入れる列がありません。書き出しでは必ず付けます。その下の `route-target` は読みます。`address-family ipv6` は読みません）
- 物理インタフェースの下の `pppoe enable`・`pppoe enable group global`（PPPoE を使う印。入れる列がありません。書き出しでは必ず付けます）
- `!` で始まる行は区切りとして数えます
- IPv6 の行（`address-family ipv6` の下の `neighbor <アドレス> activate` など）は、下の「IPv6」の節の「読み方の決まり」にあります

## 読まない行（「読めなかった」として画面に出ます）

当たる規則が無い行は、黙って落とさず、「読めなかった」の一覧に出ます。分野ごとの、読まない書き方とその理由です。VLAN とスイッチ、IPv6、ルートマップ・BGP・OSPF のそのほかの行は、それぞれの節の「読まない行」にあります。

### 機器の基本

- `enable password`（読むのは `enable secret` だけです）
- `banner incoming` など、motd・login・exec 以外のバナー
- `line con 0` の下の、`exec-timeout` 以外の行（`logging synchronous` など。コンソールの設定で列があるのは、無操作切断だけです）
- 先に読んだ行と VRF が違う `ip name-server`（DNS サーバは、全部が同じ VRF か、全部が VRF なしです）
- 名前で書いた `ntp server`・`logging host`（アドレスの形だけを読みます）

### インタフェース・トンネル・PPPoE

- `pppoe-client dial-pool-number` の後ろの語を、コマンドリファレンスの順（`dial-on-demand` → `restart <秒>` → `service-name <名前>` → `mac-override`）と違う順に並べた形
- `dialer-list <番号> protocol ip deny`・`access-group` の形と `ip` 以外のプロトコル、`dialer idle-timeout`
- `bba-group pppoe global`（PPPoE を使うポートの `pppoe enable group global` と見分けられません）と、`bba-group` の下の `virtual-template` 以外の行
- GRE など IPsec でないトンネルの `tunnel vrf`（下の「IPsec」）

### 経路・VRF・OSPF・BGP

- 向きの語（`export`・`import`・`both`）を省いた `route-target`、`vrf definition` の下の `address-family ipv6`
- OSPF の `area <エリア> default-cost`（IPv4 の OSPF には、規則がありません）
- BGP の `redistribute ospf <番号> match …` など、番号の後ろに語が続く形
- VRRP の版 3 のグループの下の `address <アドレス> secondary`（IPv4 の 2 つ目の仮想 IP）・`address <アドレス>`（ipv4 で primary の無い形）、`description`、`match-address`、`shutdown`、`track <番号> shutdown`、`vrrs leader`、`fhrp delay`。入れる列がありません。IPv4 のグループの `timers advertise` で 1000 で割り切れない値（表の広告間隔は秒です）も読みません

### 経路制御の方針

- ルートマップの下の、列の無い行: `set dampening`・`set level`・`set automatic-tag`・`set origin egp`、`set ip next-hop` の `peer-address`・`verify-availability`・`recursive`、`match as-path` に番号を 2 つ以上並べた形
- 動作と番号を省いた `route-map <名前>` の行（機器は `permit 10` を補って表示するはずですが、資料の文では確かめていません）
- `ip bgp-community new-format`
- 符号の付いた `set metric`、`match ip route-source`・`match local-preference`・`match extcommunity`、拡張コミュニティのリスト（`ip extcommunity-list`）などは、下の「経路制御・BGP・OSPF の残り」の節の「読まない行」にあります

### アクセスリスト・NAT・回線監視・DHCP

| 行 | 理由 |
|---|---|
| ポートの名前のうち、番号を確かめられなかったもの（UDP の `non500-isakmp`・`tacacs-ds`、TCP の `drip`・`tacacs-ds` など） | IOS のコマンドリファレンスは名前だけを並べていて、番号がありません。ほかの製品の資料で見た番号は、確かめたことにしていません |
| `eq`・`neq` にポートを 2 つ以上並べた形（`eq ftp ftp-data`） | 設計データのポートの列は、値が 1 つか範囲です |
| TCP のフラグの `match-all`・`match-any` の形（`match-all +ack -syn`）、フラグを 2 つ以上並べた形 | 複数のフラグを組み合わせた条件は、列で表せません（NEC IX も 1 行に 1 つです） |
| アクセスリストの `option`・`ttl`・`reflect`・`evaluate`・`dynamic`・オブジェクトグループ | 入れる列がありません |
| `time-range` の下の `absolute` の 2 行目以降 | 機器が 1 つしか持てません |
| `track <番号> ip sla <番号>`（`state` も `reachability` も無い形） | 省いたときにどちらになるかを、コマンドリファレンスで確かめられていません。IP Application Services Command Reference の章「threshold metric through track timer」の `track ip sla` の項は、構文の行で 2 つの語を省けることを示し、既定を「IP SLAs の追跡は無効」と書いているだけで、省いたときにどちらを見るかは書いていません |
| `ip sla schedule` の、`life forever start-time now` 以外の形（開始を遅らせる・寿命がある・`recurring`） | ずっと動かす監視ではないので、回線監視の行にしません |
| `track … ip route … metric threshold`、`track … list threshold …`、`object <番号> not`、`track … interface … ip routing` | 入れる列がありません |
| NAT の `no-alias`・`no-payload`・`route-map`・`vrf`・`redundancy`・`add-route`、`ip nat pool` の `type`、`ip nat outside source …`、`ip nat inside destination …` | 入れる列がありません |
| IPv6 のアクセスリストの ICMPv6 のコード（`<種類> <コード>` の形）、`time-range`・`routing`・`mobility` など | ICMPv6 のコードは、IPv6 の行での書き方を資料で確かめていません（値があれば G-602 です）。ポートの名前（IPv4 と同じ語）・TCP のフラグ（1 行に 1 つ）・`log-input` は読みます |

### IPsec

- 相手を複数並べたクリプトマップのうち、**2 行目以降の `set peer … default`**、**ISAKMP プロファイル・IKE 版 2 の項目の 2 行目以降の `set peer`**、**鍵が 1 行目の相手と違う相手**: 相手ごとの鍵・ID を持つ列がありません（1 行は、鍵 1 つ・ID 1 組です。プロファイルを使う項目は、相手ごとに `match identity` と鍵が要り、1 行に写せません）。その `set peer` の行と、その相手の鍵の行が「読めなかった」に残ります。IKE 版 1 で ID の無い項目の、同じ鍵の相手は「予備の相手」として読みます（1 行目の `default` は「つなぎ直しは最初の相手から」）
- **仮想テンプレートのうち**、IKEv2 プロファイルの `virtual-template <数> mode auto`（トンネルの種類を機器が選ぶ形）、相手ごとに鍵の違うキーリング（アドレスごとの鍵）で 1 つのひな形を使う形、`match identity group`（Easy VPN）: 行は「相手 any・鍵 1 つ」で、相手ごとの値を持てません。ISAKMP プロファイルの `vrf` をひな形と組にした形は、`vrf` の行だけが「読めなかった」に残ります（トンネル IF 型の内側は、ひな形の `vrf forwarding` で持ちます）
- **VRF のうち**、IKEv2 ポリシーの `match address local`（そのポリシーが当たる IKE 版 2 の行は、行にしません。「読めなかった」に戻ります）、IKEv2 プロファイルの `ivrf global` を VRF の中の IF のクリプトマップで使う形（「内側は VRF なし」を表す語がありません。行にしません）、GRE など IPsec でないトンネルの `tunnel vrf`（トンネルの表に外側の VRF の列がありません。IPsec の行の列です）
- **証明書・EAP**（`authentication … rsa-sig`・`eap`、`pki trustpoint`、`crypto isakmp policy` の `authentication` が `pre-share` でないもの、`crypto isakmp identity dn`）: 表にしません。認証に使う証明書は、機器の中で鍵を作り、認証局に登録して入れるもので（`crypto pki …` の手順と、機器の外の認証局）、コンフィグの行を表に写しても、別の機器では動きません。EAP は、利用者の確かめを外のサーバ（AAA）に任せる設定で、同じく機器の外に及びます。このパックが扱う認証は、事前共有鍵だけです
- **IPv6 の相手・ID・対象通信**（`tunnel mode ipsec ipv6`、`crypto isakmp key … address ipv6 …`、キーリングの `pre-shared-key ipv6 …`、`match identity address ipv6 …`、`tunnel protection ipsec policy ipv6`、`crypto map ipv6 …`）: 読まず、書きません。表は相手・対象通信を IPv4 で持ちます。そのうえ、公開資料の書き方がそろいません: `crypto isakmp key` は、構文の行が `{address <アドレス> | ipv6 <アドレス>/<長さ>}`、例が `address ipv6 <アドレス>/<長さ>` です。キーリングの IPv6 の例は `key` の無いまま印刷されています。IKEv2 のキーリング・プロファイルの「ipv6-address prefix」は、区切りが空白か `/` かを決められる例がありません。確かめられた形（IKE 版 1 の機器全体の鍵とトンネル IF）だけでは、NEC IX との往復を作れませんでした
- **マルチポイント GRE と NHRP（DMVPN）**（`tunnel mode gre multipoint`、`ip nhrp …`、`tunnel protection ipsec profile <名前> shared`）: 表にしません。NEC IX にも似た機能（ダイナミック VPN: `tunnel mode mgre ipsec-ikev2`、`nhrp nhs`、`nhrp local`）がありますが、メーカーの機能説明書に「IX と WA のみ接続可能です。それ以外の装置とは接続できません」とあり、Cisco の DMVPN とはつながりません。設定の考え方も違います（NEC IX は IKEv2 だけ・ハブに NHRP の設定が要らない・LAN 側の IF を `nhrp local` で指す。Cisco は NHRP の網番号・認証・対応づけを両側に書く）。片方を読んでもう片方へ書く決まりを作れないので、どちらのパックも読みません（表にするなら: トンネルの種類に `mgre` を足し、NHRP の表を「トンネル IF・役割（hub／spoke）・ハブのトンネル内アドレス・ハブの外側アドレス・保持時間・網番号・認証の文字列・ショートカットを使う」で持つ案。同じメーカーどうしの移し替えにしか使えません）
- **グループ鍵の VPN（GETVPN）**（`crypto gdoi group …`、`crypto map <名前> <番号> gdoi`）: 鍵サーバが配る鍵で守る別の仕組みで、相手 1 つ = 1 行の表に写せません
- **AH の SHA-2 の語**: メーカーの公開資料に見つかりません。コマンドリファレンス（A to C、2021-01-11）の「Allowed Transform Combinations」の表の AH は `ah-md5-hmac`・`ah-sha-hmac` だけです。AH ハッシュ sha256 以上の行は書き出しで止まります
- **IKE 版 1 の「NAT 越えを必ず使う」**: 当たる行が公開資料にありません（`crypto ipsec nat-transparency` は `spi-matching`・`udp-encaps` の 2 つだけ）。値があれば G-602 です。版 2 は IKEv2 プロファイルの `nat force-encap` で読み書きします
- 値を並べた IKEv2 プロポーザルを、ポリシーの 2 つ目以降に置いた形（「組み合わせの並び」の中に「値の並び」が入り、行に書けません）、7 つ以上のトランスフォームセットを並べた行（機器が受け付けるのは 6 つまで）
- `crypto map … ipsec-isakmp profile <名前>`（マップのひな形）、`ipsec-manual`、IKEv2 のキーリングの `identity`・`description`、プロファイルの `description`、`identity local dn`、`match identity remote … domain`、`match identity group`、`crypto ipsec nat-transparency spi-matching`、`crypto isakmp key … no-xauth`、`crypto isakmp fragmentation`・`crypto ipsec fragmentation`、IF の下の `crypto ipsec df-bit`、クリプトマップの下の `set security-association idle-time`・`set security-association replay …`
- トランスフォームセットの `esp-gmac`・`esp-seal`・`comp-lzs`、ESP を持たない（AH だけの）トランスフォームセット、`set security-association lifetime days`・`kilobytes disable`

### QoS

- `match protocol <名前>` の後ろに語が続く形、`random-detect dscp <値> <しきい値> …` などのしきい値の行、ミリ秒でバーストを書く `shape … percent`・`police cir percent` の形、`pir percent`、`set-dscp-transmit`・`set-prec-transmit` 以外の印を付ける動作、`match ip rtp`・`match packet length`・`match discard-class`
- インタフェースの下の `qos pre-classify`（トンネルで、アドレスやポートで見分けるときに要る行です。入れる列が無く、書き出しもしません。要るときは追加コンフィグに書いてください）
- 二段のポリシーのうち、「読み方の決まり」の「QoS の二段のポリシー」の形でないもの（`service-policy` の行が「読めなかった」に戻ります）

### L2TP

見本は `tests/import/l2tp/input.txt` です（別紙 A 3.21）。

- `l2tp-class` の下の `digest`・`hidden`・`cookie size`・`retransmit`・`timeout setup`・`password 7 <値>`（暗号化した形）
- `pseudowire-class` の下の `ip tos`・`ip ttl`・`ip protocol`、`encapsulation mpls`
- `xconnect` の行の `sequencing`、MPLS の形、`backup peer`
- `vpdn-group` の下の `session-limit`、`force-local-chap`、`lcp renegotiation`、`source-ip`、`l2tp tunnel hello` などの `l2tp tunnel …`（読むのは `l2tp tunnel password [0] <値>` と `no l2tp tunnel authentication` だけです。`l2tp tunnel password 7 <値>` は、読み書きを確かめていない形です）、`l2tp security crypto-profile`、`initiate-to` の `limit`・`priority` と 2 つ目以降、`dnis`、`request-dialout`・`accept-dialout`、`protocol` の `l2tp` 以外
- `vpdn-template`、`vpdn search-order` などの `vpdn …`
- Virtual-Template の下の `encapsulation ppp`、`no peer default ip address`、`ppp authentication` のほかの形（`callin`・リスト名・`eap`・`ms-chap`）、`ppp ipcp dns` の `accept` などの付いた形
- `ip local pool` の 2 行目以降と `group` などの付いた形
- LAC を通さずに自分で PPPoE を終える形（利用者とアドレスの払い出しを持つ `bba-group`）

## コンフィグの書き出し

「対応一覧」の表を書き出します。書き出した文面は、同じパックで取り込み直して設計データと比べます（往復の確認）。VLAN とスイッチ、IPv6、ルートマップ・BGP・OSPF のそのほかの行の書き方は、それぞれの節にあります。

節の順: `system`（ホスト名・特権パスワード・時刻帯・DNS・ユーザ・時刻の付け方）→ `vrf` → `ipv6_global`（`ipv6 unicast-routing` と DHCPv6 のプール）→ `ipsec`（IKE・トランスフォームセット・プロファイル・クリプトマップ。それを指すインタフェースより前）→ `qos` → `l2tp` → `switch`（VLAN・スパニングツリー・ブリッジ）→ `interfaces`（PPPoE もここ）→ `tracking`（回線監視）→ `routing` → `ospf` → `bgp` → `ipv6`（IPv6 の経路・OSPFv3・プレフィックスリスト・フィルタ）→ `policy`（時間帯・アクセスリスト・プレフィックスリスト）→ `route_policy`（AS パスのリスト・コミュニティのリスト・ルートマップ）→ `nat` → `dhcp` → `services`（syslog・SNMP・NTP・機器内のログ）→ `extra`（追加コンフィグの既定の位置）→ `lines`（バナー・`line con 0`・`line vty`）→ `footer`（`end`）。

表の行は、キーの自然な順（`GigabitEthernet0/2` は `GigabitEthernet0/10` より前。`Loopback0` は `GigabitEthernet…` より後）で書きます。

### 機器の基本

| 項目 | 書き方 |
|---|---|
| ハッシュ化された秘密（特権パスワード・ユーザ） | ハッシュの始まりで種別の番号を決めます。`$1$` → `5`、`$8$` → `8`、`$9$`・`$14$` → `9`。それ以外の始まりはエラーにします（別メーカーのハッシュは引き継げません） |
| 平文の秘密 | `secret 0 <値>` と書きます。形が空のセルは平文として扱います。「元に戻せる形で持つ」が有効なユーザは `password 0 <値>`（形が encrypted なら `password 7 <値>`）と書きます（PPP の CHAP の相手のユーザ。`secret` では通りません） |
| ユーザの権限 | `admin` は `privilege 15`、数はそのまま。`operator` に当たる数は決めていないので、新しいメーカーに同じ語がないエラー（G-603）になります。数で書いてください |
| 時刻帯 | 「名前 +時[:分]」の形（`JST +9` → `clock timezone JST 9 0`）。名前のない形はエラーにします |
| バナー | 「バナー（接続したとき）」は `banner motd`、「バナー」は `banner login`、「バナー（ログインの後）」は `banner exec` で書きます。区切りの文字は `#` `@` `%` `~` `\|` のうち文面に無い最初のものを使います |
| リモート接続 | `line vty 0 4` に書きます。5 番以降の線は追加コンフィグに書いてください。認証 `aaa` は書けません（エラー）。認証 `password` にはログインのパスワードが要ります |
| syslog | ファシリティ・重要度・送信元 IF は機器に 1 つなので、全部の行で同じ値でなければエラーにします |
| SNMP | 通知の送信元 IF も機器に 1 つです（そろっていなければエラー）。通知先の版 3 は書けません（エラー）。通知先のコミュニティは必須です |
| NTP | 送信元 IF が全部の行で同じなら `ntp source`、違えばサーバごとの `source` で書きます |
| 時刻の付け方・パスワードの暗号化・機器内のログ | 節 `system` に `service timestamps`・`service password-encryption`、節 `services` に `logging buffered` を書きます |
| バナーとコンソール | 節 `lines` に、バナー（motd → login → exec の順）と `line con 0` を書きます。`line con 0` に書くのは `exec-timeout` だけです |

取り込みとの関係: `ntp server`・`logging host` は、取り込みがアドレスの形だけを読みます。名前で書いたサーバは、往復の確認で指摘になります。

- **自動で置き換えて知らせるもの**（`adapt.yaml`。G-614 の警告）: ログの時刻の付け方 `time` → `datetime`。機器内ログの `cyclic`（いつもそうなので書きません）。「NTP サーバとして動く」「転送のキャッシュ」「転送のキャッシュ（IPv6）」（Cisco に行がありません）
- **書けないもの**（理由を付けて止まります。G-601）: 別メーカーの機器が暗号化した形の秘密（平文を入れ直します）。ハッシュなのに「元に戻せる形で持つ」が有効なユーザ。機器内ログの `one_time`
- **書けない列**（値があれば G-602）: ログの対象ごとの重要度（log_subsystems）

### インタフェース・トンネル・PPPoE

| 項目 | 書き方 |
|---|---|
| インタフェース | 有効な IF には `no shutdown`、無効な IF には `shutdown` を必ず書きます。アドレスの無い IF に `no ip address` は書きません。VRF は `vrf forwarding` の形で、アドレスより前に書きます |
| トンネル | トンネル IF の下に `tunnel source`・`tunnel destination`・`tunnel key`（種類が gre のときだけ）・`keepalive <間隔> <回数>`（2 つは組）。種類が gre なら `tunnel mode` の行は書きません（機器の既定）。ipip は `tunnel mode ipip`、ipsec は `tunnel mode ipsec ipv4` と `tunnel protection ipsec profile <名前>`（IPsec の表に、そのトンネル IF の行が要ります。送信元と宛先は ipsec でも必須です）。l2tpv3 は新しいメーカーに同じ語がないエラー（G-603） |
| PPPoE | 論理 IF は `Dialer1`〜`Dialer255`。その番号をプールの番号に使います（表に番号の列がありません）。物理 IF の下に `pppoe enable group global` と `pppoe-client dial-pool-number <番号> [service-name <名前>]`、Dialer の下に `encapsulation ppp`・`dialer pool <番号>`・認証の行を書きます。物理 IF と論理 IF は、どちらもインタフェースの表に行が要ります。同じ物理 IF・同じ論理 IF を 2 つの行で使うと止まります |
| PPPoE の認証 | 認証方式があれば `ppp authentication <方式> callin`（chap_pap は `chap pap`）。「相手も認証する」が有効なら、`callin` を付けません。chap は `ppp chap hostname`・`ppp chap password`、pap は `ppp pap sent-username … password …`、chap_pap と空は両方を書きます。パスワードは平文で、種別の番号を付けません |
| PPPoE のアドレス・MTU | 論理 IF のインタフェースの行に書きます（アドレス `negotiated` → `ip address negotiated`、MTU → `mtu`、IP MTU → `ip mtu`、MSS → `ip tcp adjust-mss`） |
| PPPoE のつなぎ方 | 物理 IF の `pppoe-client dial-pool-number <番号>` の後ろに `dial-on-demand`・`restart <秒>`・`service-name`（空白があれば引用符で囲みます）・`mac-override` の順で書きます。「つなぐきっかけの通信」があれば Dialer の下に `dialer-group <n>`、IF の後ろに `dialer-list <n> protocol ip permit`（値が `ip`）か `… list <番号>`（値が ACL の番号。名前は止まります）を書きます。番号 n は、PPPoE の行の順に 1、2、3 … と振ります |
| PPPoE で書けない列 | 「デフォルト経路を作る」「監視間隔」（値があれば G-602）。デフォルト経路は、出力 IF を Dialer にしたスタティックルートの行で書いてください。「無通信で切る時間」も書けません（`dialer idle-timeout` の書き方を確かめていません。値があれば G-602） |

- **自動で置き換えて知らせるもの**（`adapt.yaml`。G-614 の警告）: TCP MSS 調整の `auto` → 「IP MTU − 40」（無ければ「MTU − 40」）。「ポートの説明」→ IF の「説明」が空ならそこへ、あれば書きません
- **書けないもの**（理由を付けて止まります。G-601）: 種類が `etherip` のトンネル（`tunnel mode` に EtherIP がありません）。MTU の無い IF の MSS の `auto`

### 経路・VRF・OSPF・BGP・冗長化

| 項目 | 書き方 |
|---|---|
| スタティックルート | 説明は `name` に書きます。空白を含む説明はエラーにします（名前は 1 語） |
| VRF | `vrf definition <名前>` の形で書き、`address-family ipv4` 〜 `exit-address-family` を必ず付けます（この形の VRF は、これが無いと IPv4 を運びません）。`ip vrf <名前>` で取り込んだ VRF も、この形で書きます |
| OSPF のインタフェース | インタフェースの下に `ip ospf <プロセス番号> area <エリア>` で書きます（`network … area …` の行は書きません。`network` で取り込んだ設定も、この形になります）。1 つのインタフェースは 1 つのプロセスだけ。インタフェースの VRF とプロセスの VRF が違えばエラー |
| OSPF の受動 | 「既定を受動にする」が有効なら `passive-interface default`。インタフェースの行の「受動」は、有効 → `passive-interface <IF>`、無効 → `no passive-interface <IF>`、空 → 書きません |
| OSPF の再配布 | `subnets` を必ず付けます（付けないとクラスフルの経路しか入りません。表に列がありません）。元が bgp のときは、BGP の表の AS 番号を付けます |
| OSPF の認証 | md5 → `ip ospf authentication message-digest` と `ip ospf message-digest-key <鍵番号> md5 <鍵>`、simple → `ip ospf authentication` と `ip ospf authentication-key <鍵>`、none → `ip ospf authentication null`。鍵は平文で書きます |
| OSPF で書けない列 | エリアの「既定経路のコスト」（取り込みの規則がありません。値があれば G-602） |
| BGP | AS 番号は機器に 1 つです（行によって違えばエラー）。VRF が空の行のネイバー・広告経路・再配布は `router bgp` の直下に、VRF の行のものは `address-family ipv4 vrf <VRF>` 〜 `exit-address-family` に書きます |
| BGP の再配布 | 元が ospf のときは、同じ VRF の OSPF のプロセス番号を付けます（プロセスがちょうど 1 つのときだけ。それ以外はエラー） |
| BGP で書けない列 | VRF の行のルータ ID（取り込みの規則がありません。値があれば G-602）。Keepalive・Hold・「ネイバーの変化を記録」などの機器全体の設定は、VRF が空の行にだけ書けます（下の「経路制御・BGP・OSPF の残り」の節） |
| HSRP | `standby <グループ> ip`・`priority`・`preempt [delay minimum <秒>]`・`timers <Hello> <Hold>`（2 つは組）・`authentication text <鍵>`・`track <番号> [decrement <数>]`。版 2 は `standby version 2`（インタフェースに 1 つ。グループで違えばエラー）。認証 md5 は書けません（エラー） |
| VRRP | インタフェースの下の古い形（`vrrp <グループ> ip`・`priority`・`preempt [delay minimum <秒>]`・`timers advertise <秒>`・`track`）で書きます。版・Hold・認証は書けません（値があれば G-602） |
| 切り戻しの無効 | 「切り戻し」が無効の行は `no standby <グループ> preempt` / `no vrrp <グループ> preempt` と書きます（取り込みもこの行を読みます） |
| VRRP の版 3 | 機器の「VRRP の版（IPv4）」が 3 なら、IF の前に `fhrp version vrrp v3` を書き、方式 vrrp の行を `vrrp <グループ> address-family ipv4` 〜 `exit-vrrp` の形で書きます（`address <仮想 IP> primary`、`priority`、切り戻し待ちがあれば `preempt delay minimum <秒>`、無効なら `no preempt`、`timers advertise <秒 × 1000>`、`track`、「版 2 の相手とも話す」が有効なら `vrrpv2`）。切り戻しが有効で待ちの無い行は、何も書きません（機器の既定が有効。`preempt` だけの形は資料で確かめていません）。版 3 に認証は無いので、認証のある行は止まります（G-601）。冗長化（IPv6 の VRRP）の行は `vrrp <グループ> address-family ipv6` 〜 `exit-vrrp` の形で、`address <仮想リンクローカルアドレス> primary` と、仮想グローバルアドレスごとに `address <アドレス>/<長さ>`（長さは、その IF の固定の IPv6 アドレスのうち、仮想アドレスを含むものの長さ。無ければ止まります）、広告間隔はミリ秒のまま（100〜40000。外れたら止まります） |
| VRRP の版の置き換え（G-614） | 冗長化（IPv6 の VRRP）に行があるのに「VRRP の版（IPv4）」が 3 でない機器は、3 にして書きます。その機器の方式 vrrp の行には「版 2 の相手とも話す」を有効にして書きます（NEC IX などから来た IPv4 のグループは版 2 で、相手もそうなので、版 2 の相手とつながるままにします） |

- **スタティックルート**: `[<距離>] [name <名前>] [permanent\|track <番号>] [tag <数>]` の順です。「IF が落ちても残す」と「回線監視」の両方がある行は止まります
- **VRF**: `route-target export <値>`・`route-target import <値>` を、`vrf definition` のすぐ下に書きます（`both` で読んだ値も、2 行に分かれます）
- **BGP の再配布**: 元が ospf の行は、「OSPF のプロセス番号」があればその番号、無ければ同じ VRF にちょうど 1 つあるプロセスの番号を付けます
- **書けない列**（値があれば G-602）: スタティックルートの「メトリック」「つながっている間だけ有効」（NEC IX だけの設定です）

### 経路制御の方針（別紙 A 3.19）

節 `route_policy` は、アクセスリスト・プレフィックスリスト（`policy`）の後ろです。AS パスのリスト → コミュニティのリスト → ルートマップ → `ip local policy route-map` の順に書きます。

- **書く行**: 取り込みが読める行だけです（対応一覧の「経路制御の方針」「AS パスのリスト」「コミュニティのリスト」「経路・VRF・再配布の細かい指定」）。`route-map` は、動作と番号を必ず書きます
- **コミュニティの条件**: 方針の行にじかに書いた値（match_community 列）は、Cisco に書く形が無いので、`CL-<方針名>-<順番>` という標準のリストを作って書きます（取り込みは、これを値に戻します）。同じ名前のリストを自分で作ると止まります。permit の 1 行だけの標準のリストを指している行も、同じ形に直して書きます（意味は同じです。知らせません）。完全一致は `exact` と書きます（コマンドリファレンスの構文の語です）
- **リストの名前**: AS パスのリストは 1〜500 の番号だけです（名前は止まります）。コミュニティのリストは、名前が数なら番号の形（standard は 1〜99、expanded は 100〜500）、それ以外は `standard <名前>`・`expanded <名前>` の形で書きます
- **リストの行の順番**: 10、20、30 … と並んでいないリストは止まります（機器が行に番号を持たないので、読み直すと 10 刻みになります。順番を振り直してください。見る順は変わりません）
- **知らせるだけのもの**（`adapt.yaml`。G-614 の警告）: 送信の経路制御がコミュニティを付けるのに、そのネイバーの「コミュニティを送る」が有効でないとき（Cisco は `neighbor … send-community` が無いと、コミュニティを相手に送りません）。値は変えません
- **書けないもの**（理由を付けて止まります）: 「条件: ネクストホップ（プレフィックスリスト）」（コマンドリファレンスにあるのは、ACL で見る形だけです）、符号の付いたメトリック（`+10`・`-10`）、`none` とほかのコミュニティを並べた行、`self` とアドレスを並べたネクストホップ、17 個以上のコミュニティを並べた条件・標準のリストの行（機器が持てるのは 16 個までです）

### アクセスリスト・NAT・回線監視・DHCP

| 項目 | 書き方 |
|---|---|
| アクセスリスト | 名前付きの形（`ip access-list standard｜extended <名前>`）で書きます。名前が番号のリストも同じ形です（`ip access-list standard 10`）。アドレスは `any`・`host <アドレス>`・アドレスとワイルドカード。ポートは番号だけ（`80` → `eq 80`、`1000-2000` → `range 1000 2000`、`>1023` → `gt 1023`、`<1024` → `lt 1024`、`!80` → `neq 80`）。プロトコル `ah` は `ahp` と書きます。`established` は tcp の行だけ、標準のリストに書ける付加条件は `log` だけです（ほかはエラー） |
| アクセスリストの行の番号 | 行が 10、20、30… と並んでいるリストは、番号を書きません（番号を保存しない版の機器の出方と同じ）。それ以外の番号のリストは、全部の行に番号を書きます（注記の行にも） |
| アクセスリストの説明 | 機器にリストの説明の行がないので、「説明」の列は書けません（値があれば G-602）。注記は「動作」が remark の行で書きます |
| プレフィックスリスト | `ip prefix-list <名前> seq <番号> …` と、番号を必ず書きます |
| 回線監視（ping） | `ip sla <番号>`・`ip sla schedule <番号> life forever start-time now`・`track <番号> ip sla <番号> reachability` の 3 つを書きます（「正常とみなす条件」が within_threshold なら `state`）。送信元は `source-interface <IF>` か `source-ip <アドレス>` のどちらか 1 つです。SLA の番号は、回線監視の番号と同じにします。番号は 1〜1000 の数だけ（名前はエラー）。監視先は IPv4 アドレスだけ |
| 回線監視の待ち時間 | 5000 ミリ秒より短い待ち時間は、同じ値の `threshold` を先に書きます（機器は、しきい値（既定 5000）より短い待ち時間を受け付けません）。待ち時間が間隔より長い行はエラーにします |
| 回線監視（IF） | `track <番号> interface <IF> line-protocol` |
| 回線監視（経路・まとめる監視） | 監視の方法 `route` は `track <番号> ip route <宛先>/<長さ> reachability`、`list` は `track <番号> list boolean and｜or` と、その下の `object <番号>` で書きます。まとめる行（list）は、まとめられる行の後に書きます |
| 回線監視で書けない列 | 「VRF」は、当たる行を確かめていないので読みません（値があれば G-602）。反映待ちは `delay up <秒> down <秒>` で書きます。「断と判断する回数」「復旧と判断する回数」は当たる行が無いので、反映待ちに置き換えて書きます（表の下の「回線監視の回数」） |
| NAT | 種類ごとに、取り込みが読める形だけを書きます。`pat_interface` → `ip nat inside source list <ACL> interface <外側 IF> overload`、`pat_pool`・`dynamic_pool` → `ip nat pool <名前> <先頭> <末尾> prefix-length <長さ>` と `ip nat inside source list <ACL> pool <名前> [overload]`（プールの名前は、行の名前）、`static`、`static_port`（外側アドレスが `interface` のときは `interface <外側 IF>` の形） |
| NAT の向き | NAT の行があるのに、NAT の向きが inside と outside のインタフェースがそろっていなければ、エラーにします。`pat_interface` の外側 IF は、向きが outside の IF でなければエラーです。どの IF にも向きが無いデータ（NEC IX から来たもの）は、パックが向きを補います（表の下の「どの IF にも『NAT の向き』が無いデータ」） |
| NAT の対象の通信 | `pat_interface`・`pat_pool`・`dynamic_pool` は ACL の名前が必須です（空 = すべて、は Cisco では書けません） |
| NAT の外側 IF | `pat_interface` と、外側アドレスが `interface` の `static_port` だけが読みます。ほかの種類の行に外側 IF を書くと G-602 になります（Cisco の行に、外側 IF を書く場所がありません。セルを空にしてください） |
| DHCP | `ip dhcp pool <名前>` の下に `network <アドレス> <マスク>`・`default-router`（8 つまで）・`dns-server`・`domain-name`・`lease <日> [<時> [<分>]]`（か `lease infinite`）。配らないアドレスは `ip dhcp excluded-address <先頭> [<末尾>]`（プールのネットワークの外なら止まります） |
| DHCP の貸し出し時間 | `2d` → `lease 2`、`36h` → `lease 1 12`、`90m` → `lease 0 1 30`、`infinite` → `lease infinite`。割り切れるいちばん大きい単位で書いていない値（`24h` など）は、書き直してもらいます（取り込み直すと `1d` になり、往復が合わないため）。秒の端数は止まります（Cisco は分まで） |
| DHCP で書けない列 | 「配る範囲の先頭・末尾」（Cisco は、ネットワークから配らないアドレスを引いた残りを配ります。範囲の行がありません。値があれば G-602。配らないアドレスに書き直してください） |

置き換え（`adapt.yaml`）は G-614 で知らせます。見本は `tests/render/acl_nat/`、`tests/render/acl_adapt/` です。

- アクセスリストの行の末尾は、ICMP の種類・コード → TCP のフラグ → `dscp`／`precedence` → `tos` → `time-range` → `fragments` → `log`／`log-input` の順に書きます（`log` の後ろには何も書けないためです）。`time-range` のブロックは、アクセスリストの前に書きます
- **1 つの向きに ACL が複数ある IF**（NEC IX の `ip filter` を並べた形。「入力 ACL（2 つ目以降）」など）: Cisco は 1 つの向きに 1 つなので、並んだリストの行を順につないだリスト（`<IF の短い名前>-in`・`-out`。例 `Gi0-0-in`）を作って当てます。NEC IX は、どのリストにも合わない通信を捨てるので、意味は同じです。つなげるのは 5 つまでです。元のリストも、そのまま書きます。IPv6 フィルタも同じです
- **プールの長さが空の NAT**（NEC IX のプールは長さを持ちません）: プールが外側 IF のネットワークに入っていれば、その長さで書きます。入っていなければ、空のまま G-606 で止まります
- **「断の間の監視間隔」**（NEC IX の `probe-timer restorer`）: Cisco に無いので、書きません（断の間も同じ間隔で監視します）
- **どの IF にも「NAT の向き」が無いデータ**（NEC IX は IF に向きを書きません）: NAT の行が「外側 IF」に指している IF を outside、ほかの IPv4 アドレスを持つ IF（ループバックを除く。トンネルも入ります）を inside にして書きます。NEC IX の NAT は外側 IF を通る通信すべてに当たるので、これがいちばん近い形です。NAT を通さない IF があるときは、内側にする IF の「NAT の向き」に inside を自分で書きます（1 つでも向きが書いてあれば、パックは補いません）
- **回線監視の回数。「断と判断する回数」「復旧と判断する回数」**（NEC IX の `probe-counter`）: Cisco の監視は 1 回の失敗で断になり、回数の設定はありません。「断の反映の待ち」が空なら、「間隔 ×（回数 − 1）」秒を入れて `delay down <秒>` と書きます（その間ずっと届かないときに断にする、という近い動きです。同じではありません）。「復旧」も同じで `delay up <秒>` です。待ちがもう書いてあれば、回数だけを外します。回数が 1 のときは、機器の動きと同じなので、知らせずに外します
- 知らせずに行うもの: 「行の番号の刻み」（NEC IX の `sequence-mode`）は、Cisco がいつも番号を持つので、何も書きません。NAT の「外側 IF」は、その IF の「NAT の向き」が outside なら、行に要らないので読みません
- **書けないもの**（G-602。値があるのに使われなかった、と出ます）: 回線監視の「監視の出口の IF」「監視の出口のネクストホップ」「断のとき止める IF」（Cisco の `ip sla`・`track` に当たる書き方がありません。出口を決めたいときは、監視先へのホスト経路を別に書きます）、遠隔ログインの「接続元の制限（SSH だけ）」「（Telnet だけ）」（`line vty` は方式で分けられません）
- 理由を付けて止まるもの（G-601）: TCP のフラグが 1 行に 2 つ以上、貸し出し時間の秒の端数（Cisco は分まで）、list の「まとめる監視」が番号でない、など

### IPsec

| 項目 | 書き方 |
|---|---|
| IPsec（IKE 版 1） | 事前共有鍵だけ。`crypto isakmp policy 10`（`encryption`・`hash`・`authentication pre-share`・`group`・`lifetime`）を 1 つ書きます。**版 1 の行の IKE の暗号・ハッシュ・DH グループ・有効時間、相手監視（間隔・再送間隔・送り方）、NAT 越えの維持間隔は、全部の行で同じでなければ止まります**（機器全体のもので、取り込みも 1 組だけを読みます）。鍵は相手ごとに 1 つ（同じ相手で鍵が違えば止まります）: `crypto isakmp key <鍵> address <相手>`、相手が `any` なら `address 0.0.0.0 0.0.0.0`、名前なら `hostname <名前>`。相手監視は `crypto isakmp keepalive <秒> [<再送間隔>] [periodic]`、NAT 越えの維持間隔は `crypto isakmp nat keepalive <秒>`。鍵の書き方 hex は止まります（版 1 の鍵は文字だけ） |
| IPsec（IKE 版 1 の ID） | 自分の ID か相手の ID がある行は、その行だけ `crypto keyring KR-<マップ名>-<番号>`（トンネル IF 型は `KR-<トンネル IF>`。プロファイルの名前も同じ決まり）（`pre-shared-key address｜hostname … key <鍵>`）と `crypto isakmp profile ISAKMP-<マップ名>-<番号>`（`keyring`・`self-identity`・`match identity`）を書き、クリプトマップ（トンネル IF 型は IPsec プロファイル）に `set isakmp-profile` を書きます。自分の ID は `email:` → `self-identity user-fqdn <値>`、`fqdn:` → `self-identity fqdn`（値が機器の「ホスト名.ドメイン名」と同じときだけ。違えば止まります）、`address:` → `self-identity address`（値が自分側 IF・トンネルの送信元のアドレスと同じときだけ）。相手の ID は `address:` → `match identity address <値> 255.255.255.255`、`fqdn:` → `match identity host <値>`、`email:` → `match identity user <値>`。**`keyid:` は止まります**（ISAKMP プロファイルに、書く形がありません）。機器全体の `crypto isakmp identity` は書きません（取り込みでは読み、行ごとのプロファイルにして書きます） |
| IPsec（IKE 版 2） | 版 2 の行があれば、`crypto ikev2 proposal IKEV2-PROP`（`encryption`・`integrity`・`group`・`prf`。「（追加）」の列の値を、同じ行に空白で並べます）と、「IPsec 追加の提案」の段 ike の行ごとの `crypto ikev2 proposal IKEV2-PROP-2`…、それらを順に並べた `crypto ikev2 policy IKEV2-POL` を 1 組（**版 2 の行の IKE の暗号・ハッシュ・DH グループ・PRF と、その「（追加）」の列・段 ike の追加の提案は、全部の行で同じでなければ止まります**。3 つとも空なら、プロポーザルとポリシーを書かず、機器の既定の提案に任せます。一部だけ空は止まります）、`crypto ikev2 keyring IKEV2-KEYS`（行ごとに `peer` を 1 つ: `address <相手>`・`hostname <名前>`・`address 0.0.0.0 0.0.0.0`。鍵は `pre-shared-key <鍵>`、相手を確かめる鍵が別なら `pre-shared-key local <鍵>` と `remote <鍵>`、書き方 hex なら鍵の前に `hex`）、行ごとの `crypto ikev2 profile IKEV2-<トンネル IF か マップ名-番号>`（`match identity remote …`・`identity local …`・`authentication remote pre-share`・`authentication local pre-share`・`keyring local IKEV2-KEYS`・`lifetime`・`dpd <間隔> <再送間隔> on-demand｜periodic`・`nat keepalive <秒>`）を書き、IPsec プロファイルかクリプトマップに `set ikev2-profile` を書きます。**相手監視は、間隔・再送間隔・送り方の 3 つがそろわなければ止まります**（G-606）。同じ相手（アドレス）の行が 2 つあると止まります。相手が `any` の行が 2 つ以上あるときは、どれも相手の ID が要り、鍵は同じでなければ止まります。相手が名前の行は、相手の ID が要ります |
| IPsec（トンネル IF 型） | `crypto ipsec transform-set TS-<トンネル IF>`、`crypto ipsec profile PROF-<トンネル IF>`（`set transform-set`・`set pfs group<数>`・`set security-association lifetime seconds`）、トンネル IF の下に `tunnel protection ipsec profile PROF-<トンネル IF>`。**対象通信の行があるトンネル IF 型の行**（相手に伝える範囲。NEC IX の `ipsec local-id`・`remote-id`、IKEv2 の `local-ts`・`remote-ts` から来ます）は、`ip access-list extended ACL-<トンネル IF>` を作り、トンネル IF の下に `tunnel protection ipsec policy ipv4 ACL-<トンネル IF>` を書きます（IOS XE 16.12 以降のコマンドです）。トンネルの表に種類 ipsec の行が要り、その宛先が相手アドレスと同じでなければ止まります。**トンネルの「宛先」が空なら IPsec の相手アドレスを宛先に書き（知らせません）、「送信元」が空なら、デフォルト経路の出口のインタフェースを送信元に書いて知らせます**（G-614。NEC IX の IKE 版 1 のトンネルは、どちらも持たないため） |
| IPsec（クリプトマップ型） | 行の名前が `<マップ名>/<番号>` なら、その項目として書きます。**ほかの名前の行（別メーカーから来た行）は、`CMAP-<自分側 IF の名前>` というマップに、その IF のそういう行の順に 10・20… の番号で入れます**（往復の確認は、名前でなく中身で行を合わせます）。対象通信の行から `ip access-list extended ACL-<マップ名>-<番号>` を作り（`<順番> permit ip｜udp｜tcp <自分側> [eq <ポート>] <相手側> [eq <ポート>]`。0.0.0.0/0 は `any`）、`crypto map <マップ名> <番号> ipsec-isakmp` に `set peer <アドレス>`（名前なら `set peer <名前> dynamic`）・`set transform-set TS-<マップ名>-<番号>`・`set pfs`・`set security-association lifetime seconds`・`… kilobytes`・`match address` を書きます。自分側 IF の下に `crypto map <マップ名>`。1 つのインタフェースに 1 つのマップ、1 つのマップは 1 つのインタフェースです。対象通信のプロトコルは ip・udp・tcp |
| IPsec の予備の相手 | 「予備の相手」の値を、`set peer <相手アドレス>` の後ろに `set peer <アドレス>`（名前なら `set peer <名前> dynamic`）と並べ、その相手の鍵（行の事前共有鍵）を `crypto isakmp key <鍵> address｜hostname <相手>` で書きます。IKE の版 1・方式 policy・ID の無い行だけ（ほかは止まります） |
| IPsec（相手が any） | 方式 policy の行は次のとおり（方式 tunnel_if の行は、仮想テンプレートで書きます。表の下の「相手 any のトンネル IF 型」）。`crypto dynamic-map DYN-<マップ名>-<番号> 10` に中身を書き、`crypto map <マップ名> <番号> ipsec-isakmp dynamic DYN-<マップ名>-<番号>` で指します。対象通信の行は無くてもかまいません（無ければ `match address` を書きません）。**「自分側 IF」が空の行（NEC IX の L2TP の受け口を守る行）は、デフォルト経路の出口のインタフェースに当てて知らせます**（G-614。決まらなければ止まります） |
| IPsec の追加の提案 | ipsec_proposals の行を、順番の小さい順に書きます（番号そのものは書きません）。段 ike は、版 1 の行なら `crypto isakmp policy 20`・`30`…（値のある項目だけ。`authentication pre-share` はいつも書きます）、版 2 の行なら `crypto ikev2 proposal IKEV2-PROP-2`…（暗号・ハッシュ・DH グループの 3 つがそろわなければ止まります）。**その版の全部の行に、同じ並びの行が無ければ止まります。** 段 ipsec は、`crypto ipsec transform-set TS-…-2`・`-3`… を作り、`set transform-set TS-… TS-…-2 …` と並べます（版 1・版 2 の両方。全部で 6 つまで。超えると止まります）。段 ipsec の有効時間・有効量、版 2 の段 ike の有効時間は書けません（値があれば G-602） |
| IPsec の「（追加）」の列（版 2） | IKE の 4 つの列は、IKEv2 のプロポーザルの行に並べて書きます。**「ESP 暗号（追加）」「ESP ハッシュ（追加）」は、組み合わせごとのトランスフォームセットにして書き（「IPsec 追加の提案」の行に置き換えます。意味は同じです）、「PFS（追加）」は書きません。どちらも G-614 で知らせます**（Cisco のトランスフォームセットは 1 つが 1 つの組み合わせで、PFS は項目に 1 つです）。版 1 の行にあると止まります |
| IPsec のトランスフォームセット | `crypto ipsec transform-set TS-… [ah-md5-hmac｜ah-sha-hmac] <ESP 暗号> [<ESP ハッシュ>]`。「IPsec のモード」に値があれば、その下に `mode tunnel｜transport`（空なら書きません。機器の既定は tunnel）。ESP 暗号の aes128gcm・aes256gcm は `esp-gcm`・`esp-gcm 256`、ESP ハッシュの sha256・sha384・sha512 は `esp-sha256-hmac`・`esp-sha384-hmac`・`esp-sha512-hmac`。**GCM の行に ESP ハッシュがあると、ハッシュを書かずに知らせます**（G-614。GCM は認証を兼ね、ほかの ESP の語と並べられません）。AH ハッシュ sha256 以上は止まります（語を確かめていません） |
| IPsec の NAT 越え | 行が無いことが有効です（機器の既定）。**全部の行が無効なら `no crypto ipsec nat-transparency udp-encaps` を 1 行書きます。行によって違えば止まります**（機器に 1 つです）。「NAT 越えの維持間隔」は、版 1 は `crypto isakmp nat keepalive`、版 2 は IKEv2 プロファイルの `nat keepalive` で書きます。版 2 の「NAT 越えを必ず使う」は、IKEv2 プロファイルの `nat force-encap` です |
| L2TP を守る IPsec | 「方式 policy・相手 any・モード transport」の行を、トランスポートモードのトランスフォームセット・動的なクリプトマップ・`crypto map … ipsec-isakmp dynamic`・`address 0.0.0.0 0.0.0.0` の鍵として書きます（上の決まりのとおり）。L2TP の行の「IPsec の設定名」は、取り込みが決める行（名前の順で最初のそういう行）と違えば、その行に置き換えて知らせ、そういう行が無ければ書かずに知らせます（G-614）。指す先がそういう行でないと止まります |
| IPsec の名前 | トランスフォームセット・プロファイル・対象通信のアクセスリストの名前は、表に列が無いのでパックが決めます。取り込んだ元のコンフィグの名前（`IPSEC-PROF-…` など）は引き継ぎません |
| IPsec で書けない列 | IKE のモード・相手監視の再送回数・自分側アドレス、トンネル IF 型の行の自分側 IF、版 1 の行の PRF・相手を確かめる鍵・「NAT 越えを必ず使う」（値があれば G-602）。「この IKE の相手だけに使う」は、書かずに知らせます（G-614。Cisco に当たる行がありません）。IKE 版 1 のハッシュ sha512 は、語を資料で確かめていないのでエラー（G-603）になります |
| 「いつもつないでおく」 | Cisco に書く行がありません。トンネル IF 型の「有効」と、クリプトマップ型の「無効」は、機器の動きそのものなので、何も書かず、何も知らせません。トンネル IF 型の「無効」は、書かずに G-614 で知らせます。クリプトマップ型の「有効」は G-602 です |

- 相手 any のトンネル IF 型: `crypto keyring KR-Virtual-Template<数>`（`pre-shared-key address 0.0.0.0 0.0.0.0 key <鍵>`）、`crypto isakmp profile ISAKMP-Virtual-Template<数>`（`keyring`・`match identity address 0.0.0.0`（相手の ID があればその行）・`virtual-template <数>`）、`interface Virtual-Template<数> type tunnel`（`ip unnumbered`・`tunnel mode ipsec ipv4`・`tunnel protection ipsec profile`。「送信元」があれば `tunnel source`）。版 2 は IKEv2 プロファイルに `virtual-template <数>`。**「トンネル IF」が `Virtual-Template<数>` でない相手 any の行は止まります**（NEC IX から書き換えるときは、新旧対照表（ポート）で `Tunnel0.0` → `Virtual-Template1` のように対にします）
- 相手が名前のトンネル IF 型: `tunnel destination <名前>`。トンネルの「宛先（名前）」が空なら、IPsec の相手の名前を書きます（知らせません）。版 2 は相手の ID が要ります
- 外側の VRF: トンネル IF に `tunnel vrf`。版 1 は `crypto keyring … vrf <VRF>` と ISAKMP プロファイル（`match identity address <相手> 255.255.255.255 <VRF>`）を行ごとに書きます（ID の無い行でも）。版 2 は、IKEv2 プロファイルに `match fvrf <VRF>`、機器に 1 つの IKEv2 ポリシーに `match fvrf any`（版 2 の行に外側の VRF が 1 つでもあるとき）。**外側の VRF がある版 2 の行で IKE の暗号・ハッシュ・DH グループが空なら止まります**（ポリシーにはプロポーザルが要り、ポリシーが無ければ VRF なしの相手にしか当たらないため）。方式 policy の行の「外側の VRF」が自分側 IF の VRF と違えば止まります
- 内側の VRF（方式 policy）: ISAKMP プロファイルの `vrf <VRF>`、IKEv2 プロファイルの `ivrf <VRF>`
- 「ESP ハッシュ」の `none` は、ESP の認証の語の無いトランスフォームセットとして書きます（空と同じ。知らせません）
- 「IPsec の全体設定」は、`ipsec.j2` の終わりに 1 行ずつ書きます。行が無いことと同じ値（再送攻撃の検出 = 有効 など）は止まります
- 別メーカーの形の鍵（`secret <値>`）は止まります（G-601。平文を入れ直します）。種類が etherip のトンネルと、それを守る IPsec の行は止まります（EtherIP がありません）

### QoS

- **書く行**: MQC の形だけです（古い `traffic-shape`・`rate-limit`・`priority-list`・`queue-list` は読むだけで、書きません）。`class-map match-any|match-all` と `match access-group [name]`・`match dscp`・`match precedence`・`match any`、`policy-map` と `class`、`priority`・`priority percent`・`priority level`・`bandwidth`・`bandwidth percent`・`bandwidth remaining percent`・`shape average`・`police cir … conform-action transmit exceed-action …`・`set dscp`・`set precedence`・`queue-limit <数> packets`・`fair-queue`、インタフェースの下の `service-policy input|output`
- **条件と動作のほかの形**（対応一覧の「QoS の条件」「QoS の動作」）も書きます。`match class-map` で指されるクラスを、指すクラスより前に書きます。`shape` は「遅らせ方」が空なら `average` です。`police` は `police cir <bps> [bc …] [pir …] [be …] conform-action transmit exceed-action … [violate-action …]` の 1 行で書きます。`queue-limit` は単位が空なら `packets` を付けます
- **DSCP・precedence** は数で書きます
- **IF 全体を絞る速さ**: `shape-<IF の短い名前>` という親のポリシー（`class class-default` に `shape average` と、子の `service-policy`）を作り、インタフェースにはその親を当てます（取り込みは、これを適用の行に戻します）
- **順番**: クラスは「順番」の小さい順、`class-default` はいつも最後に書きます。条件の順番は 10 刻みでなければ止まります
- **自動で置き換えて知らせるもの**（`adapt.yaml`。G-614 の警告で知らせます）: 条件ごとのキュー（high〜low。NEC IX と、古い `priority-list` を読んだもの）は MQC に無いので、キューごとのクラス（`<クラス名>-high` など）に分けて書きます。high は `priority level 1`、medium は `priority level 2`、normal は残りの帯域の 75%、low は 25% です。**動きは近いものの同じではありません。** 割合を変えたいときは、分けた形のクラスの行をパラメータブックに自分で書きます。キューがどれも同じクラスは、キューを書かずに 1 つのクラスとして書きます。そのクラスに自分の動作（帯域・優先など）があるポリシーでは分けず、キューの値を G-602 で知らせます
- **書けないもの**: 機器自身の通信のクラス（local）の行、4 つ並べたキューの長さ、IPv6 の ACL・機器自身が出す通信・RTP のポート・NGN の QoS の条件、QoS のクラス `dynamic` は止まります（G-601）
- **書けない列**（値があれば G-602）: 「付ける DSCP（IPv6）」「付ける precedence（IPv6）」「NGN 用の既定のポリシー」

### L2TP

| 項目 | 書き方 |
|---|---|
| L2 延伸（L2TPv3） | 設定名ごとに `pseudowire-class <設定名>`（`encapsulation l2tpv3`、`protocol l2tpv3 [<設定名>]` か `protocol none`、`ip local interface`、`ip pmtu`、`ip dfbit set`、`sequencing`）を 1 つ書きます。パスワード・名前・Hello のどれかがあれば、同じ名前の `l2tp-class`（`authentication`・`password`・`hostname`・`hello`）を先に書きます。延伸する IF の下に、dynamic は `xconnect <相手> <回線番号> pw-class <設定名>`、manual は `xconnect <相手> <回線番号> encapsulation l2tpv3 manual pw-class <設定名>` と、その下の `l2tp id`・`l2tp cookie local`・`l2tp cookie remote`・`l2tp hello`。設定名が空なら `l2tpv3-<回線番号>` を付けて知らせます（G-614）。同じ設定名で設定が違う行、「経路 MTU の検出」「DF ビットを立てる」の「無効」は止まります |
| L2TP（LNS） | `vpdn enable`、`vpdn-group <名前>` の下に `accept-dialin`・`protocol l2tp`・`virtual-template <番号>`、`terminate-from hostname`、`local name`、認証が無効なら `no l2tp tunnel authentication`、パスワードがあれば `l2tp tunnel password`。払い出すアドレスは `ip local pool l2tp-<名前> <先頭> <末尾>`。受け口の IF（`Virtual-Template<番号>` を 1 つ。ほかの名前・2 つ以上は止まります）の下に `peer default ip address pool`、`ppp authentication`、`ppp ipcp dns`。IF のアドレス（`unnumbered:<IF>` など）は、インタフェースの行に書きます |
| L2TP（LAC） | `vpdn-group <名前>` の下に `request-dialin`・`protocol l2tp`・`domain <名前>`（運ぶドメインの数だけ）、`initiate-to ip <相手アドレス>`（名前では書けません。止まります）、`local name`、認証とパスワードは LNS と同じ |
| L2TP の利用者 | `username <接続 ID> password 0 <パスワード>`。同じ名前がユーザの表にもあると止まります |
| PPPoE を受ける LAC | `bba-group pppoe <L2TP の名前>` と `virtual-template <番号>`、PPPoE を受ける IF の下に `pppoe enable group <L2TP の名前>`、Virtual-Template の下に `ppp authentication <利用者の認証>` を書きます。「受け口の IF」が Virtual-Template でないとき（空のときも）は、`Virtual-Template<n>` の行を作って使い、知らせます（G-614） |
| L2TP で書けない列 | PPPoE を受けない LAC の「受け口の IF」と、LAC の行の「IPsec の設定名」は、書かずに知らせます（G-614）。LNS の行の「IPsec の設定名」は、上の「IPsec」の「L2TP を守る IPsec」のとおりです。「無通信で切る時間」、利用者の「固定で払い出すアドレス」、LNS の「相手アドレス」「運ぶドメイン」、LAC の「相手の名前」は、値があれば G-602。暗号化した形のパスワード（L2TP・L2TPv3 のパスワード、利用者のパスワード）は、読み書きを確かめていないので止まります（G-601） |

## VLAN とスイッチ

ルータのタグ VLAN（サブ IF）、ルータ内蔵のスイッチポート、LAG、Catalyst 系のスイッチ本体（IOS / IOS-XE）、古い書き方のブリッジ（BVI）を、このパック 1 つで読み書きします。スイッチ用の別のパックにしなかった理由: 判定（`pack.yaml` の `detect`）に使う行（`Current configuration`・`version`・`interface GigabitEthernet…`・`line vty`）がルータとスイッチで同じで、2 つに分けると点数が並んで人が選ぶことになるため。また、内蔵スイッチ付きのルータは両方の行を 1 つのコンフィグに持つため。

### 読める行と書き出す行

| 分野 | 行 | 入る表と列 |
|---|---|---|
| タグ VLAN（サブ IF） | `encapsulation dot1Q <番号> [native]`（`dot1q` とも）、二重タグの `encapsulation dot1Q <外> second-dot1q <内の並び>\|any` | interfaces の「VLAN 番号（サブ IF）」「タグなしの VLAN」「内側の VLAN（二重タグ）」 |
| VLAN の定義 | `vlan <番号>` と、その下の `name <名前>`、`state active\|suspend`、`shutdown`、`remote-span`、`private-vlan primary\|isolated\|community`、`private-vlan association <並び>`。`vlan 30,40`・`vlan 2-4`（番号を並べた行。番号ごとの行になります） | vlans |
| VLAN の IF | `interface Vlan<番号>`。番号を「VLAN 番号（VLAN の IF）」に入れます（別メーカーへ書き換えるときに、ポートの VLAN と IF を結ぶため） | interfaces |
| スイッチポート | `switchport`（それだけの行）、`no switchport`（モード routed）、`switchport mode access\|trunk\|dynamic auto\|dynamic desirable`、`switchport access vlan <番号>`、`switchport trunk native vlan <番号>`、`switchport trunk allowed vlan <並び>\|all\|none`、`… allowed vlan add <並び>`（前の行の続きとして足します）、`switchport trunk encapsulation dot1q\|isl\|negotiate`、`switchport nonegotiate`、`switchport voice vlan <番号>`、`switchport voice vlan dot1p\|none\|untagged`、`switchport protected`、`switchport mode private-vlan host\|promiscuous`、`switchport private-vlan host-association <プライマリ> <セカンダリ>`、`switchport private-vlan mapping <プライマリ> <並び>` | switchports |
| スパニングツリー（ポート） | `spanning-tree portfast`（`edge` の語が有っても無くても読みます）、`… portfast [edge] trunk`、`… portfast disable`、`spanning-tree bpduguard enable\|disable`、`spanning-tree guard root\|loop\|none`、`spanning-tree bpdufilter enable\|disable`、`spanning-tree cost <数>`、`spanning-tree port-priority <数>`、`spanning-tree link-type point-to-point\|shared` | switchports |
| ストーム制御 | `storm-control broadcast\|multicast\|unicast level <%>`、`… level <上> <下>`、`… level bps\|pps <上> [<下>]`（整数と、その後ろの k・m・g。`pps 2k` は `2000pps`、`bps 100m` は `100000kbps` として持ちます。小数の付いた数は読みません）、`storm-control action shutdown\|trap` | switchports |
| ポートセキュリティ | `switchport port-security`、`… maximum <数>`、`… violation protect\|restrict\|shutdown`、`… mac-address sticky`、`switchport port-security mac-address <MAC> [vlan <番号>\|access\|voice]`、`… mac-address sticky <MAC> [vlan …]`（覚えたアドレス）、`switchport port-security aging time <分>`・`aging type absolute\|inactivity`・`aging static` | switchports |
| LAG | `channel-group <番号> [mode active\|passive\|on\|auto\|desirable]`。LAG の IF（`interface Port-channel<番号>`）は、ふつうの IF として interfaces と switchports に入ります。`channel-group <番号> mode auto\|desirable non-silent`、`channel-protocol lacp\|pagp`、`lacp port-priority <数>`、`lacp rate normal\|fast`、`port-channel load-balance <方式>`、`lacp system-priority <数>` | lags、l2_settings |
| スパニングツリー（全体） | `spanning-tree mode pvst\|rapid-pvst\|mst`、`spanning-tree portfast [edge] default`、`spanning-tree portfast [edge] bpduguard default`、`spanning-tree extend system-id`、`spanning-tree vlan <並び> priority\|hello-time\|forward-time\|max-age <数>`、`spanning-tree loopguard default`、`spanning-tree portfast [edge] bpdufilter default`、`spanning-tree pathcost method long\|short`、`spanning-tree backbonefast`、`spanning-tree uplinkfast`、`spanning-tree vlan <並び> root primary\|secondary [diameter <数>]`、`no spanning-tree vlan <並び>` | l2_settings、stp_vlans |
| MST | `spanning-tree mst configuration` と、その下の `name`・`revision`・`instance <番号> vlan <並び>`（`vlans` とも）、`spanning-tree mst <並び> priority <数>`、`spanning-tree mst <並び> root primary\|secondary`（並べた行は、インスタンスごとの行になります）、`spanning-tree mst hello-time\|forward-time\|max-age <秒>`（機器に 1 本の木のタイマー） | l2_settings、stp_instances |
| スイッチ全体 | `ip routing` / `no ip routing`、`ip default-gateway <アドレス>`、`mac address-table aging-time <秒>`（`mac-address-table` とも）、`vlan internal allocation policy ascending\|descending`、`system mtu <数>` | l2_settings |
| VTP | `vtp mode server\|client\|transparent\|off`、`vtp domain <名前>`、`vtp version 1\|2\|3`、`vtp password <文字>`、`vtp pruning` | l2_settings |
| 速度 | `speed 2500\|5000\|10000`、`speed auto <速度の並び>`、`speed nonegotiate` | interfaces |
| ポートの見張り | `errdisable recovery cause <原因>`、`errdisable recovery interval <秒>`、`udld enable\|aggressive`、`udld message time <秒>`、IF の下の `udld port [aggressive\|disable]` | l2_settings、switchports |
| CDP・LLDP | `[no] cdp run`、`cdp timer`、`cdp holdtime`、`[no] lldp run`、`lldp timer`・`holdtime`・`reinit`、IF の下の `[no] cdp enable`、`[no] lldp transmit\|receive`（ルータの IF でも読みます） | l2_settings、interfaces |
| DHCP スヌーピング・ARP 検査 | `ip dhcp snooping`、`ip dhcp snooping vlan <並び>`、`[no] ip dhcp snooping information option`、`ip arp inspection vlan <並び>`、IF の下の `ip dhcp snooping trust`・`ip dhcp snooping limit rate <数>`・`ip arp inspection trust` | l2_settings、switchports |
| ポートミラー | `monitor session <番号> source interface <IF> [both\|rx\|tx]`（1 行に IF 1 つ）、`monitor session <番号> destination interface <IF>` | mirror_sessions |
| スタック | `switch <番号> provision <種別>` | stack_members |
| ブリッジ | `bridge irb`、`bridge <番号> route ip`（`interface BVI<番号>` があるとき、その BVI の「ブリッジグループ」になります）、IF の下の `bridge-group <番号>`、`bridge <番号> protocol ieee\|dec\|vlan-bridge\|ibm`、`bridge <番号> priority <数>` | l2_settings、interfaces、bridge_groups |

### 書き出しで決めたこと

| 項目 | 書き方 |
|---|---|
| 節の位置 | `switch`（スタック・スイッチ全体・スパニングツリー・VLAN の定義・ブリッジ）を、インタフェースの前に書きます。ポートの行は、そのインタフェースの下（`description`・`encapsulation` の次、アドレスの前）です。L3 のポートは `no switchport` をアドレスより前に書きます |
| PortFast | `edge` の語を付けずに書きます（`spanning-tree portfast`・`… portfast trunk`・`… portfast default`・`… portfast bpduguard default`）。Catalyst 系の IOS XE 17.12 の設定ガイドと、ルータのスイッチモジュールの設定ガイドがこの形です。`edge` 付きで保存する機種があること（下の「確かめていないこと」）は、取り込みで両方を読むことで吸収します |
| 許可 VLAN | 1 行にまとめて `switchport trunk allowed vlan 10,20,30-40` と書きます（`add` の続きの行には分けません）。`all`・`none` は、それだけを書きます（番号と一緒ならエラー） |
| それだけの `switchport` | スイッチポートの行に何も値が無いときだけ書きます |
| VLAN の IF | IF 名は `Vlan<番号>` です。「VLAN 番号（VLAN の IF）」が IF 名の番号と違う行、`Vlan` で始まらない IF にこの番号がある行は、エラーにして直し方を知らせます（別メーカーから書き換えたときに、新旧対照表（ポート）の名前の付け間違いに気づくため） |
| ブリッジ | 「ブリッジと L3 の併用」が有効なら `bridge irb`、BVI ごとに `bridge <番号> route ip` を書きます。BVI の「ブリッジグループ」は、BVI の番号と同じでなければエラーです（Cisco では、グループ n の L3 の口は BVIn）。グループの番号は 1〜255 です |
| 「無効」が書けない列 | 行が無いのと同じ意味になる「無効」は、エラーにします（セルを空にしてください）: 「アクセスポートをすぐ転送にする」「エッジポートの BPDU ガード」「拡張システム ID」「モードの自動交渉を止める」「ポートセキュリティ」「覚えた MAC を設定に残す」「ブリッジと L3 の併用」、ループガードの既定・BackboneFast・UplinkFast・VTP プルーニング・DHCP スヌーピング・保護ポート・信頼ポート・VLAN の shutdown など。有効・無効のどちらも書けるのは、CDP・LLDP・BPDU フィルタ（ポート）・DHCP スヌーピングの option 82 です |
| LAG | `channel-group <番号> mode <束ね方>`。束ね方が空なら `channel-group <番号>` だけを書きます |
| 番号を並べた VLAN | 1 つずつの `vlan <番号>` で書きます（並べた形には戻しません。機器はどちらも受け付けます） |
| MST のインスタンス | `instance <番号> vlan <並び>` と書きます（設定ガイドの形。コマンドリファレンスは `vlans`）。取り込みは両方を読みます |
| BPDU フィルタの既定 | `spanning-tree portfast bpdufilter default` と、`edge` の語を付けずに書きます（PortFast と同じ決まり） |
| ストーム制御の値 | 割合はそのまま、速さ（`500pps`・`64kbps`・`1500bps`）は `level pps <数>`・`level bps <数>` と、k・m・g を使わない数で書きます。下限は同じ行の後ろに付けます（上の値と単位が違えばエラー） |
| MAC アドレステーブル | `mac address-table aging-time` と書きます（古いソフトウェアは `mac-address-table`。取り込みは両方を読みます） |
| ポートミラー | IF 名は正式な名前で書きます（機器は省略形で見せますが、取り込みは両方を読みます）。両方向の行は、向きの語を付けずに書きます |
| MST の優先度 | インスタンス 1 つに 1 行で書きます（`spanning-tree mst 0 priority 4096`。並べた形には戻しません） |
| 機器に 1 本の木の設定 | 別メーカーの「ルートブリッジの指定（機器全体）」「優先度（機器全体）」は、方式が mst のとき、インスタンス 0（共通の木）の `spanning-tree mst 0 root …`・`priority …` に置き換えて知らせます。タイマー（Hello・転送遅延・最大エージ）は、方式が mst のときだけ `spanning-tree mst hello-time` などで書きます（VLAN ごとの木では、VLAN ごとの行に書きます） |
| LACP の送信間隔 | 「LACP の短いタイムアウト」が有効なら `lacp rate fast`、無効なら `lacp rate normal` |
| 優先度の刻み | ポート優先度が 16 の倍数でない、MST の優先度が 4096 の倍数でないときは、エラーにします（機器が受け付けません。VLAN ごとの優先度は V-259 が見ます） |
| 別メーカーの VLAN | VLAN を使っているのに「VLAN」の表が空の機器（スパニングツリーの方式も VTP のモードも空の機器だけ）には、`vlan <番号>` を足して知らせます（G-614。VLAN 1 は足しません） |
| NEC IX だけの列 | MDI の向き、ループ検出、ブリッジグループのエージング・テーブルの大きさ、ポートの既定の優先度などは、当たる行を確かめていないので書きません（G-602 で知らせます） |

### 別メーカーのスイッチ（NEC QX）のデータを書くとき

NEC QX のパック（`nec_qx`）から来るデータ（別紙 A 3.4・3.22・3.24）を、Cisco で書くときの置き換えです（`adapt.yaml`。G-614 で知らせます。試験は `tests/render/other_switch`・`other_switch_failed`）。

| 元のデータ | Cisco での書き方 |
|---|---|
| 「スパニングツリーを使う」が有効 | 書きません（このパックに、スパニングツリーを有効にする行はありません）。無効は置き換えず、G-602（使われなかった値）で知らせます |
| 方式 rstp・stp（機器に 1 本の木） | mst（`spanning-tree mode mst`） |
| 「優先度（機器全体）」「ルートブリッジの指定（機器全体）」 | 方式が mst のとき（上の置き換えで mst になったときも）、MST のインスタンス 0（共通の木）の `spanning-tree mst 0 priority`・`root` として書きます（上の「機器に 1 本の木の設定」） |
| VLAN の「説明」、ユーザの「使える接続方式」 | 書きません |
| ユーザの権限 `level-<数>` | 特権レベルの数 |
| 0 から始まる ACL の順番 | 同じ並びのまま 10・20・30 … に振り直します（Cisco の順番は 1 以上。振り直した ACL は、番号を付けずに書かれます。行のほかの列（ICMP のコード・ToS・時間帯・行の注記）は、そのまま引き継ぎます） |
| 名前が番号で、Cisco の ACL の番号の範囲の外（NEC QX の 2000〜2999 の標準、3000〜3999 の拡張など） | 名前 `ACL-<番号>` に付け替えて書きます（リスト・その行・リストを指す IF／遠隔ログイン／SNMP／QoS の列）。経路制御の方針がそのリストを指しているとき、`ACL-<番号>` がすでにあるときは、エラーです |
| スイッチポートのモード hybrid | タグなしで通す VLAN が「ネイティブ VLAN」の 1 つだけなら、trunk として書きます。ほかはエラー |
| 機器が保存した形の値（SNMP コミュニティ・VRRP の鍵の `cipher …`）、「パスワードの形」が encrypted の特権パスワード、OSPF の認証 hmac_md5 | エラー（G-601） |
| ACL の「行の注記」、SNMP ユーザ（版 3）、NTP の認証、ループ検出など、別紙 A 3.24 の列のうち Cisco に当たる行を確かめていないもの | 書きません（G-602） |

### 読まない行

| 行 | 理由 |
|---|---|
| `switchport trunk allowed vlan remove …`・`… except …` | 資料では、機器が元の並びから計算する入力の形です（`remove` は前の並びから引く、`except` は並びを裏返す）。計算した後の並びが running-config にどう出るかを資料で確かめられず、`remove` は元の並びが無いと値になりません |
| `vlan` の下の `mtu`・`name` 以外の古い媒体の語（`media`・`said`・`ring` ほか） | 入れる列がありません（`mtu` は、構文と範囲を資料で確かめられませんでした） |
| IF の下の `spanning-tree vlan <並び> cost\|port-priority …`・`spanning-tree mst <並び> cost\|port-priority …`（VLAN・インスタンスごとのポートの値） | 入れる表がありません。ポート全体の `spanning-tree cost`・`port-priority` は読みます |
| `spanning-tree mst max-hops`、`spanning-tree uplinkfast max-update-rate …`、`spanning-tree vlan … root … hello-time`、`spanning-tree transmit hold-count`、`spanning-tree etherchannel guard misconfig` | 入れる列がありません |
| `storm-control unknown-unicast …`、`switchport port-security violation shutdown vlan`、`switchport port-security maximum <数> vlan …`、`switchport block unicast\|multicast` | 入れる列がありません |
| `port-channel load-balance extended …`、`lacp max-bundle`、`port-channel min-links`、`pagp port-priority` | 入れる列がありません |
| `vtp password <文字> hidden\|secret`、`vtp mode … mst\|unknown\|vlan`、`vtp file`、`vtp interface` | 入れる列がありません。資料では、VTP のパスワード・プルーニング・版は設定ファイルに保存されません（`hidden` のときだけ、鍵が 16 進数で出ます） |
| `switch <番号> priority <数>`・`switch <番号> renumber <数>` | 設定の行ではありません（資料では EXEC のコマンドで、running-config に出るという記述がありません） |
| `monitor session <番号> source vlan …`・`… source remote vlan`・`… destination remote vlan`・`… filter …`、出力先の後ろの `encapsulation`・`ingress`、IF を `,`・`-` で並べた行 | 入れる列がありません（1 行に IF 1 つの形だけを読みます） |
| `switchport mode private-vlan trunk …`、`switchport private-vlan … trunk …`、VLAN の IF の下の `private-vlan mapping` | 入れる列がありません |
| `errdisable detect cause …`、`udld reset`、`lldp tlv-select`・`lldp med-tlv-select`、`cdp advertise-v2`・`cdp log mismatch duplex`・`cdp source-interface` | 入れる列がありません |
| `ip dhcp snooping database …`・`ip dhcp snooping verify mac-address`・`ip arp inspection validate\|limit\|filter`、`ip source binding`、`ip verify source` | 入れる列がありません |
| `mac address-table aging-time <秒> vlan <番号>`・`mac address-table static …`・`mac address-table notification …`、古い小型ルータの `mac-address-table secure …` | 入れる列がありません |
| `mdix auto`、`flowcontrol`、`power inline`、`load-interval`、`keepalive`、`switchport host`、`ip igmp snooping …` | 入れる列がありません |
| `bridge <番号> hello-time\|forward-time\|max-age`、ブリッジグループのエージング、IF の下の `bridge-group <番号> spanning-disabled\|path-cost\|priority` | 入れる列がありません。エージングは、資料の見出しが `bridge-group <番号> aging-time`（グローバル）となっていて、`bridge <番号> aging-time` の形を確かめられませんでした |
| `encapsulation dot1Q <番号> second-dot1q … from-bd`、基本 IF の `dot1q tunneling ethertype` | 入れる列がありません |

### 確かめていないこと

- running-config の中の `dot1Q`・`Vlan`・`Port-channel`・`BVI` の大文字小文字。資料の構文の行は小文字（`interface vlan`・`interface port-channel`・`interface bvi`）です。小型ルータの設定ガイドの running-config の例は `interface Vlan1` でした。取り込みは IF 名の大文字小文字を区別せず、書き出しは `Vlan10`・`Port-channel1`・`BVI1` の形です。`encapsulation dot1Q` が running-config に出る形は、どの資料にもありません（二重タグの例はどれも小文字の `dot1q`。取り込みは両方を読みます）
- PortFast・BPDU フィルタの `edge` の語が、どの機種・どの版で running-config に出るか。17.12 のコマンドリファレンスは `spanning-tree portfast edge bpdufilter default`、同じ版の設定ガイドは `edge` なしです
- 長い許可 VLAN の並びが、running-config で `switchport trunk allowed vlan add …` の行に分かれること。メーカーの公開の Q&A で見ただけで、コマンドリファレンスの文では確かめていません。1 行に書いた長い並びを機器が受け付けるかも確かめていません
- 番号を並べた VLAN の定義が running-config に出る形。Catalyst 系の資料には、並びを入力できること（`vlan vlan-id` の説明）しかありません。小型ルータの設定ガイドの running-config の例に `vlan 2-4` があったので、その形で読みます。並べた行の下に `name` などの行が続く形は読みません
- スパニングツリーの優先度。コマンドリファレンスの範囲は 0〜65535、設定ガイドは「0〜61440 の 4096 刻み。ほかの値は受け付けない」です。設定ガイドに合わせて V-259 がエラーにします。`root primary`・`secondary` が、running-config に `root …` のまま出るか、機器が決めた `priority` の行になるかは、資料にありません（2 つの資料で、機器が入れる値も違います）。パックは `root …` の行をそのまま読み書きします
- MST の `instance <番号> vlan` と `vlans` のどちらが running-config に出るか（2 つの資料で違います）
- VLAN の定義が running-config に出る条件。資料では、VTP が transparent のとき（と、拡張範囲の VLAN）に出ます。server・client のときは出ないので、取り込んだ vlans の表は空になります（そのため V-251 は、vlans に行がある機器だけを見ます）。VTP が off のときに出るかは、資料にありません
- スイッチポートの既定のモード。機種と版で違います（Catalyst 系は dynamic auto、ルータのスイッチモジュールは dynamic desirable。内蔵スイッチ付きの小型ルータは、古い版では access、新しい版では dynamic auto と設定ガイドにあります）。そのため `defaults.yaml` に既定を書いていません
- 内蔵スイッチ付きの小型ルータの設定ガイドには、トランクの行（`switchport trunk native vlan`・`allowed vlan`・`encapsulation`）と `spanning-tree portfast` の例がありません。この 2 つは、共通のコマンドリファレンスと、ルータに差すスイッチモジュールの設定ガイドで確かめた形のままです。設定ガイドにある上限（VLAN の数が機種で 2〜8、ポートセキュリティの MAC の数）は、パックは確かめません。古い機種の、設定ファイルの外（VLAN データベース）で VLAN を作る書き方は扱いません
- VTP が server・client のスイッチのデータ（vlans の表が空）を、同じ Cisco へ書き出すとき、パックは VLAN の定義を足しません（スパニングツリーの方式が書かれているため）。新しい機器に VLAN が要るなら、シート「VLAN」に行を足してください。内蔵スイッチ付きのルータでは、スパニングツリーの方式も VTP のモードも書かれていないことが多く、パックが定義を足します（機器が VLAN を設定の外に持つ場合、比較では「ブックにだけある行」として出ます）
- 古い書き方のブリッジで、`bridge <番号> protocol …` の行が無いブリッジグループが働くか
- IOS XE の小型ルータのスイッチポートに `monitor session <番号> source <IF>`（`interface` の語なし）と書く例が設定ガイドにあります。この形は読みません
- Catalyst 系で、何も書かないときにスパニングツリーが有効であること（Cisco IOS LAN Switching Command Reference の `spanning-tree vlan` の既定の欄には、時間と優先度だけがあり、有効・無効の記述がありません）。方式 mst を、リージョンの設定なしで書いたときの動き（1 本の木として働くこと）も、資料の文では確かめていません

### 機種の定義（`models.yaml`。別紙 B 6 章）

「新旧対照表（機器）」の「新機種」に下の ID を書くと、新旧対照表（ポート）の新 IF 名の案を、機種のポートの並びから作ります（内蔵スイッチのポートはスイッチのポートへ、L3 のポートは WAN のポートへ。VLAN の IF は `Vlan<番号>`、サブ IF は `<親>.<VLAN 番号>`、PPPoE の論理 IF は `Dialer<番号>`）。ID は製品の名前ではなく、ポートの並びを言う名前です（数字はポートの数）。

| ID | ポート | 根拠 |
|---|---|---|
| `isr-4fe-1fe` | スイッチ `FastEthernet0`〜`3`、WAN `FastEthernet4` | 小型ルータの設定ガイド（4 ポートの FE のスイッチと FE の WAN） |
| `isr-8fe-1fe-1ge` | スイッチ `FastEthernet0`〜`7`、WAN `GigabitEthernet0`・`FastEthernet8` | 同じ設定ガイド（8 ポートの FE のスイッチ、FE と GE の WAN）。スイッチのポートの名前 `FastEthernet0`〜`7` は、「8 ポート」と「WAN は FE8」からの読み取りです |
| `isr-xe-8ge-2ge` | スイッチ `GigabitEthernet0/1/0`〜`7`、WAN `GigabitEthernet0/0/0`・`0/0/1` | IOS XE の小型ルータの設定ガイドの既定の running-config。`0/0/x` が WAN であることは、設定ガイドの本文にはありません |

GE のスイッチ 8 ポートに `GigabitEthernet8` の WAN を持つ形は、読んだ設定ガイドに無かったので定義していません（案件フォルダのパックに `models.yaml` を置けば足せます）。

## IPv6（別紙 A 3.23）

構文は、上の「参照した資料」の Cisco IOS IPv6 Command Reference と BGP Command Reference で確かめました。**実機の出力では確かめていません。**

### 読める行と入る表

| 分野 | 読める行 | 入る表 |
|---|---|---|
| 機器全体 | `ipv6 unicast-routing` | devices の「IPv6 の転送」 |
| インタフェース | `ipv6 enable`、`ipv6 address <アドレス> link-local`、`ipv6 address <プレフィックス> eui-64`、`ipv6 address <アドレス>/<長さ> anycast`、`ipv6 address autoconfig [default]`、`ipv6 address dhcp [rapid-commit]`、`ipv6 address <プレフィックスの名前> <下の部分>/<長さ>`、`ipv6 dhcp client pd <名前> [rapid-commit]`、`ipv6 mtu`、`ipv6 traffic-filter <リスト> in\|out`、`ipv6 dhcp server <プール>` | ipv6_interfaces（固定の `ipv6 address <アドレス>/<長さ>` は interfaces） |
| RA の送信 | `ipv6 nd ra suppress [all]`、`no ipv6 nd ra suppress`、`ipv6 nd ra interval <最大> [<最小>]`、`ipv6 nd ra lifetime`、`ipv6 nd managed-config-flag`、`ipv6 nd other-config-flag`、`ipv6 nd ra dns server <アドレス> [<秒>]`、`ipv6 nd prefix <プレフィックス> [<有効> <推奨> [no-autoconfig\|no-onlink]]` | ipv6_interfaces、ipv6_ra_prefixes |
| DHCPv6 サーバ | `ipv6 dhcp pool <名前>` の下の `dns-server <アドレス>`、`import dns-server`、`domain-name` | ipv6_interfaces（プールを付けた IF の行） |
| DHCPv6 サーバ（プレフィックスとアドレスの配布） | `ipv6 local pool <名前> <プレフィックス> <配る長さ>` と、`ipv6 dhcp pool` の下の `prefix-delegation pool <名前> [lifetime <有効> <推奨>]`、`address prefix <プレフィックス>`、`sntp address <アドレス>` | ipv6_interfaces（dhcp_pd_pool・dhcp_pd_length・dhcp_pd_valid・dhcp_pd_preferred・dhcp_address_prefix・dhcp_ntp） |
| スタティック経路 | `ipv6 route [vrf <VRF>] <宛先>/<長さ> [<IF>] [<次ホップ>] [<distance>] [name <名前>]` | ipv6_routes |
| フィルタ | `ipv6 access-list <名前>` の下の `[sequence <番号>] permit\|deny ipv6\|tcp\|udp\|icmp\|esp\|ahp\|<番号> any\|host <アドレス>\|<プレフィックス> [eq\|gt\|lt\|neq <ポート>\|range <ポート> <ポート>] …  [dscp <数>] [established] [fragments] [log] [sequence <番号>]`、`remark` | ipv6_acl_entries |
| プレフィックスリスト | `ipv6 prefix-list <名前> [seq <番号>] permit\|deny <プレフィックス> [ge <数>] [le <数>]` | ipv6_prefix_lists |
| OSPFv3 | `ipv6 router ospf <番号>` の下の `router-id`、`area <エリア> stub\|nssa [no-summary]`、`area <エリア> default-cost`、`area <エリア> range <プレフィックス> [advertise\|not-advertise]`、`passive-interface default\|<IF>`、`default-information originate [always] [metric <数>]`、`redistribute connected\|static\|bgp <AS>`。IF の `ipv6 ospf <番号> area <エリア>`、`ipv6 ospf cost` / `priority` / `hello-interval` / `dead-interval` / `network`。新しい形の `router ospfv3 <番号>`（その下の `address-family ipv6 [unicast]`）と IF の `ospfv3 <番号> ipv6 area <エリア>`（取り込みのみ） | ospfv3_processes、ospfv3_interfaces、ospfv3_areas |
| OSPFv3 の認証 | IF の `ipv6 ospf authentication ipsec spi <SPI> md5\|sha1 [0\|7] <鍵>`、`ipv6 ospf authentication null`、`ipv6 router ospf` の下の `area <エリア> authentication ipsec spi <SPI> md5\|sha1 [0\|7] <鍵>`（IP Routing: OSPF Configuration Guide「IPv6 Routing: OSPFv3 Authentication Support with IPsec」） | ospfv3_interfaces・ospfv3_areas（認証のアルゴリズム・SPI・鍵。`null` は none） |
| BGP（IPv6） | `router bgp` の下の `neighbor <IPv6 アドレス> remote-as` / `description` / `password` / `ebgp-multihop` / `update-source` / `shutdown`。`address-family ipv6 [unicast]` の下の `neighbor <IPv6 アドレス> activate` / `next-hop-self` / `soft-reconfiguration inbound` / `prefix-list <名前> in\|out` / `route-map <名前> in\|out`、`network <プレフィックス>`、`redistribute connected\|static\|ospf <番号>` | bgp_ipv6_neighbors、bgp_processes の「IPv6 の広告経路」「IPv6 の再配布」 |
| トンネル | `tunnel mode ipv6`（IPv6 の網で IPv4 を運ぶ）、`tunnel mode ipv6ip`（その逆）、`tunnel destination <IPv6 アドレス>`、`tunnel destination <ホスト名>` | tunnels |

### 読み方の決まり

- **値を入れない規則**: `ipv6 access-list <名前>`・`ipv6 dhcp pool <名前>`・`address-family ipv6`（`router bgp` と `router ospfv3` の下）は、下の行を持つ入れ物の行です
- **「無視」にする行**: `address-family ipv6` の下の `neighbor <アドレス> activate`（IPv6 の経路を交換する印。表に列が無く、書き出しでは必ず付けます）、`router ospfv3` の下の `exit-address-family`
- **読んで、表には入れない語**: BGP の `redistribute ospf <番号>` の番号（書き出しでは、機器に 1 つだけある OSPFv3 のプロセスの番号を書きます）、OSPFv3 の `redistribute bgp <AS>` の AS 番号（BGP の表の AS 番号を書きます）
- フィルタの行の番号は、行の頭（`sequence 10 permit …`）・行の末尾（`… sequence 10`）・番号なしのどれでも読みます。番号の無い行には 10 刻みで振ります。プレフィックスリストの `seq` が無い行には 5 刻みで振ります（機器の振り方と同じ）
- フィルタの `ipv6`（すべてのプロトコル）は、表では `ip` です。ICMPv6 の種類は、番号か、語の対応表（`icmp6_type`）にある語だけを読みます
- DHCPv6 のプールは、名前で IF に結び付けて、IF の行に入れます。1 つのプールを 2 つ以上の IF に付けた設定は、最初の IF の行にだけ DNS が入ります（取り込みの警告が出ます）
- `ipv6 local pool` の名前は表に持ちません（書き出しでは `PD-<DHCPv6 サーバの名前>` と付けます）
- OSPFv3 の認証の鍵の `7 <値>`（機器が暗号化して保存した形）は、その形のまま入れます。`0 <鍵>` は鍵そのものです

### 読まない行（「読めなかった」として画面に出ます）

| 行 | 理由 |
|---|---|
| `ipv6 cef`、`ipv6 nd router-preference`、`ipv6 nd ns-interval` など、上の表に無い `ipv6 …` の行 | 入れる列がありません |
| `ipv6 dhcp client pd hint …`、`ipv6 dhcp server automatic`、`ipv6 dhcp server <プール>` にオプションを付けた形、`ipv6 dhcp relay destination` | 入れる列がありません |
| `ipv6 dhcp pool` の下の `prefix-delegation <プレフィックス> <DUID>`（相手ごとに決めたプレフィックス）、`address prefix … lifetime …`、`link-address` など | 入れる列がありません（相手の見分け方がメーカーで違うため、相手ごとの配布は表にしていません） |
| `ipv6 nd prefix` の `default`・`no-advertise`・`off-link`・`no-rtr-address`・日付の形、`ipv6 nd ra interval msec …` | 入れる列がありません |
| `ipv6 route` に `tag`・`nexthop-vrf`・`unicast`・`multicast`・2 つ目の distance を付けた形 | 入れる列がありません |
| フィルタの行のうち、DSCP を名前で書いた形 | IPv6 のコマンドリファレンス（`permit (IPv6)`・`deny (IPv6)`）は、`dscp value` を「0 から 63」の数としてだけ示していて、名前の一覧がありません。IPv4 の一覧を当てはめることはしていません |
| フィルタの行のうち、`flow-label`・`routing`・`mobility`・`reflect`・`evaluate`・`time-range`・`auth`、ICMPv6 のコード、TCP のフラグを 2 つ以上並べた形（ポートの名前・TCP のフラグ 1 つ・`log-input` は読みます） | 入れる列がありません（`reflect`・`evaluate` は、通信の向きを覚えて戻りを通す仕組みで、NEC IX のダイナミックフィルタと形がそろいません。フラグの組み合わせは、NEC IX も 1 行に 1 つです） |
| OSPFv3 の `redistribute` の `match …`・`include-connected`・`metric transparent` の付いた形、`redistribute rip <名前>`、`area … range … cost`、`router ospfv3` の下の `address-family ipv4` | 入れる列がありません（メトリック・メトリックの種類・タグ・ルートマップを付けた形は、「OSPFv3 再配布の指定」の表に入ります） |
| OSPFv3 の `area … virtual-link` | IPv6 のコマンドリファレンスに、`ipv6 router ospf` の下のバーチャルリンクの項がありません（OSPF のコマンドリファレンスの `area virtual-link` は、エリアの番号に「IPv6 のプレフィックスも可」と書いていますが、OSPFv3 での構文の行と例がありません）。確かめられないので読みません（IPv4 の OSPF のバーチャルリンクは読みます） |
| `ipv6 ospf encryption …`、`area … encryption …`（OSPFv3 の暗号化） | 入れる列がありません（認証は読みます。暗号化は、アルゴリズムと鍵の組が多く、NEC IX の OSPFv3 にも無いので表にしていません。追加コンフィグに書きます） |
| BGP の `address-family ipv6` の下の `aggregate-address`、`neighbor … maximum-prefix`・`filter-list`・`weight`・`default-originate` など（`send-community [both\|extended]` は読みます）、`address-family ipv6 vrf …`・`vpnv6`・`multicast` | 入れる列がありません（`redistribute` にメトリック・ルートマップを付けた形は、「BGP IPv6 再配布の指定」の表に入ります。VRF の中の IPv6 の BGP は表にしていません。IPv6 のネイバーのこれらの行は、コマンドリファレンスの構文の行が IPv4 のアドレスとピアグループの名前だけを示しています） |
| `tunnel mode gre ipv6`・`ipsec ipv6`、`tunnel mode ipv6ip` の `6rd`・`6to4`・`isatap`・`auto-tunnel`、`tunnel 6rd …` | 入れる列がありません |
| MAP-E（`nat64 map-e domain …` と、その下の `basic-mapping-rule` など）、`nat64 …` | 下の「公開資料で確かめられず、読まないもの」の MAP-E の行のとおりです |
| `icmp-echo <IPv6 アドレス> source-ip <アドレス>` | IPv6 の監視の `source-ip` に IPv6 アドレスを書けるかを、コマンドリファレンス（`icmp-echo` の項は、宛先が IPv4・IPv6 のどちらでもよいとだけ書いています）で確かめられませんでした。`source-interface` の形は読みます |
| `logging host ipv6 <アドレス>` に `transport`・`discriminator` などを付けた形、`ntp server ipv6 <名前>`・`ntp server ip <名前>` | 入れる列がありません（名前の宛先は、IPv4 と同じく読みません） |
| IF の `ospfv3 <番号> area <エリア> ipv6` の語順 | コマンドリファレンスの構文の行（`ospfv3 <番号> ipv6 area <エリア>`）と例の語順が食い違っています。構文の行の形だけを読みます |

### 書き出し

- 機器全体の `ipv6 unicast-routing` と DHCPv6 のプールは IF の前（`ipv6_global.j2`）、IF の中の行は `ipv6_if.j2`、経路・OSPFv3・プレフィックスリスト・フィルタは BGP の後ろ（`ipv6.j2`）、BGP の IPv6 は `router bgp` の中（`bgp_ipv6.j2`）に書きます
- **OSPFv3 は `ipv6 router ospf` と IF の `ipv6 ospf <番号> area <エリア>` の形で書きます**（`router ospfv3` の形で取り込んだ設定も、この形になります）
- **RA を送る**: send は `no ipv6 nd ra suppress`、reply_only は `ipv6 nd ra suppress`、off は `ipv6 nd ra suppress all` と書きます（空なら何も書きません）
- フィルタの行の番号は、10・20・30… なら書きません。それ以外は行の末尾に `sequence <番号>` と書きます。注記（remark）の行には番号を書けないので、注記の行の順番が上から 10 刻みで数えた番号と違うときは、理由を付けて止めます
- **置き換えて知らせるもの（G-614）**: 「IPv6 の転送」が空で、IPv6 の経路・OSPFv3・BGP・RA の送信があるとき、`ipv6 unicast-routing` を書きます。「RA で配る DNS の生存時間」が空のとき、1800 秒で書きます
- OSPFv3 の認証は、IF の `ipv6 ospf authentication ipsec spi …`（none は `ipv6 ospf authentication null`）と、`ipv6 router ospf` の下の `area <エリア> authentication ipsec spi …` に書きます
- **書けないもの（理由を付けて止めます。G-601）**: ND プロキシ、「DHCPv6 で情報だけもらう」、もらったプレフィックスに EUI-64 で付ける形、スタティック経路の「ネクストホップの決め方」（ra・dhcp）、RA のプレフィックスの 2 つのフラグを両方無効にする形、トンネルの種類 map_e・auto_ds_lite・auto_ipip、OSPFv3 の再配布の元 rip、VRF の中の IPv6 の BGP、VRF を付けた IPv6 のログ転送先
- **書かない列（値があれば G-602）**: 「DHCPv6 でもらう情報」（Cisco には当たる行がありません）、トンネルの「MAP-E の事業者」「MAP-E のホスト名」「LAN 側 IF（自動設定）」

### IPv6 の宛先（DNS・NTP・syslog・SNMP・回線監視）

| 分野 | 読める行 | 入る表 |
|---|---|---|
| DNS サーバ | `ip name-server [vrf <VRF>] <アドレス> …` の IPv6 アドレス（IP Addressing Services Command Reference。IPv4 と同じ行に並べられます） | devices（DNS サーバ） |
| NTP | `ntp server [vrf <VRF>] <IPv6 アドレス> [prefer] [source <IF>]`（Basic System Management Command Reference） | ntp_servers |
| syslog | `logging host ipv6 <アドレス>`（Embedded Syslog Manager Command Reference。IPv6 の形に VRF はありません） | syslog_servers |
| SNMP | `snmp-server host <IPv6 アドレス> [version 1\|2c] <コミュニティ>`（SNMP Support Command Reference） | snmp_hosts |
| 回線監視 | `ip sla <番号>` の下の `icmp-echo <IPv6 アドレス> [source-interface <IF>]`（IP SLAs Command Reference）、`track <番号> ipv6 route <プレフィックス> reachability`（IP Application Services Command Reference の `track ip route`） | tracks |

- IPv6 アドレスは、機器が大文字で表示しても、小文字に直して入れます
- 書き出しは、読める行と同じ形です。IPv6 のログ転送先に VRF があれば、理由を付けて止まります（G-601）
- **置き換えて知らせるもの（G-614）**: IPv6 の監視の「送信元アドレス」は、そのアドレスを持つ IF を「送信元 IF」にして書きます（持つ IF が無ければ止まります）。「監視の出口の IF」（NEC IX の IPv6 の監視には必ずあります）は、監視先への IPv6 のスタティック経路の出力 IF と同じなら、書かずに知らせます（Cisco の監視は経路表に従って送ります。違うときは G-602 として残ります）
- **名前の回線監視（G-614）**: Cisco の回線監視は 1〜1000 の番号です。名前の行（NEC IX のウォッチグループの名前など）は、表にある番号のいちばん大きいものの次から、名前の順に番号を振って書きます。その監視を指すスタティック経路・冗長化・まとめる監視の行も、その番号にします（IPv4 の監視も同じです）

### 確かめていないこと

- 機器が `show running-config` に出す形: IPv6 の宛先の行（`ntp server` の IPv6 アドレスの大文字・小文字など）、OSPFv3 の認証の鍵の表示（`7` を付けて暗号化して表示するか）
- 機器が `show running-config` に出す形: フィルタの行の番号の位置、`ipv6 nd ra dns server` の秒数が省けるか（構文の行には角かっこがありません）、`no ipv6 nd ra suppress` が表示されるか（書き出した文面を読み直すための形として読みます）
- `ipv6 router ospf` の下の `area <エリア> stub`・`nssa`・`default-cost` と `default-information originate` の `metric` の省略: IPv6 のコマンドリファレンスには `area range` と `area (OSPFv3)` の節しかなく、IPv4 の OSPF と同じ形として読み書きしています
- IPv6 のネイバーを足すと、`address-family ipv4` の側にも `no neighbor <IPv6 アドレス> activate` が要るかどうか（機器の既定による）。この行は書いていません
- `ipv6 tcp adjust-mss <数>` を読み書きします（IP Application Services Command Reference の章「ip tcp adjust-mss through ipv6 tcp adjust-mss」にありました。値は 40〜1940）。NEC IX の `auto` は、パックが「IPv6 MTU（無ければ MTU）− 60」に置き換えて G-614 で知らせます。どちらも空なら、理由を付けて止まります
- `tunnel mode ipv6` のトンネルに要るほかの行（MTU など）

## 経路制御・BGP・OSPF の残り（別紙 A 3.19 の「経路制御の残り」）

「読まない行」に残っていた、ルートマップ・BGP・OSPF の書き方を読み書きできるようにしたものです。見本は `tests/import/bgp_more/`・`tests/import/routing_more/`、`tests/render/bgp_more/`・`tests/render/routing_more/` です。構文は、上の「参照した資料」の章ごとの PDF で確かめました。**実機の出力では確かめていません。**

### 読める行と入る表

| 分野 | 読める行 | 入る表・列 |
|---|---|---|
| ルートマップ | `match ip next-hop <ACL> …`、`match route-type local\|internal\|external [type-1\|type-2]`、`continue [<順番>]`、`set comm-list <リスト> delete`、`set extcommunity rt <値> … [additive]`、`set extcommunity soo <値>`、`match ipv6 address prefix-list <名前>`、`match ipv6 address <IPv6 のリスト>`、`set ipv6 next-hop <アドレス> …`、`set ipv6 default next-hop <アドレス> …` | route_policies（match_next_hop_acl・match_route_type・continue_to・set_community_delete・set_extcommunity_rt・set_extcommunity_soo・match_ipv6_prefix_list・match_ipv6_acl・set_ipv6_next_hop・set_ipv6_default_next_hop） |
| ルートマップ（IPv6 のネクストホップ） | `match ipv6 next-hop prefix-list <名前>`（IPv6 Command Reference） | route_policies（match_ipv6_next_hop_prefix_list） |
| IPv6 のポリシールーティング | IF の `ipv6 policy route-map <名前>`、`ipv6 local policy route-map <名前>` | ipv6_interfaces（policy_route）、devices（local_policy_route_v6） |
| BGP のネイバー | `neighbor <アドレス> timers <Keepalive> <Hold>`、`default-originate [route-map <名前>]`、`filter-list <番号> in\|out`、`distribute-list <ACL> in\|out`、`maximum-prefix <数> [<割合>] [discard-extra] [restart <分>] [warning-only]`、`weight <数>`、`send-community both\|extended`、`route-reflector-client`、`advertisement-interval <秒>`、`password 7 <値>` | bgp_neighbors |
| BGP のピアグループ | `neighbor <名前> peer-group`、`neighbor <名前> <上と「対応一覧」のネイバーの設定>`、`neighbor <アドレス> peer-group <名前>` | bgp_neighbors（メンバーの行） |
| BGP の全体 | `timers bgp <Keepalive> <Hold>`、`distance bgp <外> <内> <ローカル>`、`bgp default local-preference <数>`、`no bgp log-neighbor-changes`、`mask` の無い `network <アドレス>`、`aggregate-address <アドレス> <マスク> [as-set] [summary-only]` | bgp_processes、bgp_networks、bgp_aggregates |
| BGP（IPv6 のネイバー） | `neighbor <IPv6 アドレス> password 7 <値>`、`address-family ipv6` の下の `neighbor <IPv6 アドレス> send-community [standard\|both\|extended]` | bgp_ipv6_neighbors |
| OSPF | `default-information originate … [metric-type 1\|2] [route-map <名前>]`、`max-metric router-lsa`、`distance <数>`、`distance ospf [intra-area <数>] [inter-area <数>] [external <数>]`、`distribute-list prefix <名前> in`・`distribute-list <ACL> in`・`distribute-list route-map <名前> in`、`area <エリア> virtual-link <ルータ ID> [hello-interval <秒>] [dead-interval <秒>]` | ospf_processes、ospf_virtual_links |

### 読み方の決まり

- **ピアグループは、表に持ちません。** グループの設定を、メンバーの行のうち自分で値を持たない列に入れます（グループを使わずにネイバーごとに書いた設定と同じ意味です）。書き出しは、いつもネイバーごとの行です（取り込んだ元のグループの名前は引き継ぎません）。メンバーが 1 つも無いグループの行は「読めなかった」のままです。IPv6 のアドレスのメンバー（`neighbor <IPv6 アドレス> peer-group <名前>`）は読みません
- **`mask` の無い `network <アドレス>`** は、アドレスのクラスの長さで入れます（先頭の数が 1〜126 なら /8、128〜191 なら /16、192〜223 なら /24）。Cisco の技術資料（Document ID 19345）に「クラスの境目（255.0.0.0・255.255.0.0・255.255.255.0）のネットワークは mask を書かなくてよい」「クラス B のネットワークは、mask の無い `network <アドレス>` だけで /16 として BGP の表に入る」（資料の例）とあります。コマンドリファレンスの `network (BGP and multiprotocol BGP)` の項は、`mask` を省けることだけを示しています。そのクラスのネットワークのアドレスでない行（`network 192.0.2.64`）は「読めなかった」です。**書き出しは、いつも `mask` を付けます**
- **`set comm-list <リスト> delete`** は、リストが「permit の、1 行に 1 つのコミュニティを書いた標準のリスト」のときだけ読みます（コマンドリファレンスが、この命令に使えるリストの形として示しているものです）。リストの中身を方針の行の「外すコミュニティ」に入れ、リストはコミュニティのリストの表から外します。ほかの形のリスト（正規表現・deny・1 行に 2 つ以上）を指す行は「読めなかった」です
- **`continue`** の番号を省いた形は、`next`（次の行）として入れます
- **`password 7 <値>`** は、「7 <値>」のままセルに入れます（別紙 A 3.26）
- **`send-community both`** は「コミュニティを送る」と「拡張コミュニティを送る」の両方、`extended` は後者だけが有効です
- **`distance ospf`** は、3 つの種類を 1 行に並べた形も、1 つずつ別の行に書いた形も読みます

### 書き出しと置き換え（adapt.yaml）

- BGP の全体の行（`timers bgp`・`bgp default local-preference`・`no bgp log-neighbor-changes`）は、VRF が空の行だけに書けます（VRF の行に値があれば、理由を付けて止まります）
- **`distance bgp`** は 3 つの数を組で書きます。空の列には、機器の既定（20・200・200。コマンドリファレンスの `distance bgp` の項）を入れて書きます（意味は変わらないので、知らせません）
- 「外すコミュニティ」は、パックが `CD-<方針名>-<順番>` という標準のリスト（1 行に 1 つ）を作って、`set comm-list CD-… delete` と書きます。同じ名前のリストを自分で作ると止まります
- ネイバーの「受信フィルタ」（プレフィックスリスト）と「受信フィルタ（ACL）」は、同じ向きにどちらか 1 つです（両方あれば止まります）
- OSPF の「経路表に入れる経路の絞り込み」は、3 つの列のどれか 1 つです

### 読まない行（「読めなかった」として画面に出ます）

| 行 | 理由 |
|---|---|
| 符号の付いた `set metric`（`+10`・`-10`） | コマンドリファレンス（`set metric (BGP-OSPF-RIP)`）は、値を「-294967295 から 294967295 の整数」と書いているだけで、符号が「今の値に足す・引く」を表すとは書いていません。意味を確かめられないので、読みも書きもしません（書こうとすると、理由を付けて止まります） |
| `match route-type level-1\|level-2`、`match ip route-source`、`match local-preference`、`match extcommunity`、`match ipv6 next-hop`・`match ipv6 route-source` | 入れる列がありません（IS-IS は表にしていません。拡張コミュニティのリストは表にしていません） |
| `set extcommunity rt range …`・`cost`・`vpn-distinguisher`、`set ipv6 next-hop` の `peer-address`・リンクローカルを並べた形・`encapsulate` | 入れる列がありません |
| `ip extcommunity-list …` | 拡張コミュニティのリストは表にしていません（付ける値のルートターゲット・SoO だけを読みます） |
| `neighbor … timers <Keepalive> <Hold> <下限>`、`timers bgp <Keepalive> <Hold> <下限>` | 3 つ目の数（受け入れる Hold の下限）を入れる列がありません |
| `aggregate-address` の `as-confed-set`・`suppress-map`・`advertise-map`・`attribute-map` | 入れる列がありません |
| `neighbor … remove-private-as`・`local-as`・`fall-over`・`transport`・`ttl-security`・`unsuppress-map`・`next-hop-unchanged`・`version` など | 入れる列がありません |
| IPv6 のアドレスのネイバーの `peer-group`、`neighbor <名前> peer-group` だけがあってメンバーが無いグループ | メンバーの行に入れる先がありません |
| OSPF の `distribute-list … in <IF>`、`distribute-list … out [<経路の元>]`、`gateway` の付いた形 | 入れる列がありません（機器全体の、経路表に入れる側だけを読みます） |
| `max-metric router-lsa` に `external-lsa`・`include-stub`・`on-startup`・`summary-lsa` を付けた形 | 入れる列がありません |
| `area <エリア> virtual-link` に `authentication`・`retransmit-interval`・`transmit-delay`・`ttl-security` を付けた形 | 入れる列がありません |
| `default-information originate` (BGP)、`bgp default ipv4-unicast`、`bgp bestpath …` | 入れる列がありません（BGP の `default-information originate` は「0.0.0.0 の再配布を許す」設定で、NEC IX の `originate-default`（デフォルト経路を広告する）と意味がそろいません） |

### 公開資料で確かめられず、読まないもの

公開資料で書き方か意味を確かめられなかったもの、設計データの形に入らないものです。どれも「読めなかった」として画面に出ます（黙って落としません）。

| 書き方 | 探した資料と、読まない理由 |
|---|---|
| VRF の中の IPv6（`ipv6 route vrf …` 以外の、`address-family ipv6 vrf …` の BGP、VRF の OSPFv3） | NEC IX は「IPv6 はルーティング機能を含めて VRF 未対応」です（NEC IX の機能説明書 2.17）。BGP の IPv6 の表は VRF を持たない形で、VRF の中の IPv6 の BGP・OSPFv3 は表にしていません（VRF の中の IPv6 のスタティック経路は読み書きします） |
| ND プロキシ | IPv6 Command Reference の章「ipv6 mo to ipv6 ospf da」（`ipv6 nd …`）にあるのは `ipv6 nd dad-proxy`・`ipv6 nd na glean` などで、NEC IX の `ipv6 nd proxy <LAN 側 IF>`（WAN 側で LAN の端末の代わりに答える）に当たる行は見つかりませんでした。書けないままです（G-601） |
| MAP-E（`nat64 map-e domain <番号>` と、その下の `border-relay-address`・`basic-mapping-rule`・`default-mapping-rule` など） | Cisco の MAP-E は、規則（IPv6 のプレフィックス・IPv4 のプレフィックス・ポートの決め方）を手で書く形です。公開資料の 2 つの版（IP Addressing Configuration Guide, IOS XE 17.x の「Mapping of Address and Port Using Encapsulation」と、IP Addressing: NAT Configuration Guide, 15M&T の同じ名前の章）は、境界の装置の書き方（`border-relay-address` と `default-mapping-rule`）、IF での有効にし方（`nat64 enable` とトンネル IF）、行の置き場所（モードの名前）が食い違っていて、往復できる形を決められませんでした。NEC IX の MAP-E は「事業者を選ぶと、規則を自動でもらう」形だけで（CRM 12 章の `tunnel mode map-e jpix\|ocn\|ocn-fixed`）、規則を手で書く行がありません。Cisco の規則を NEC IX へ書くと、規則の値が使われず事業者の規則で動くので、置き換えもしません。表にせず、読めなかった行のままです |
| ポートの名前のうち番号を確かめられなかったもの、`eq` に複数のポート、同じ名前の `ip nat pool` の複数行 | 上の「ACL・NAT・経路・回線監視・DHCP の残り」の節の「読まない行」のとおりです。`eq <ポート> <ポート>` を 1 行ずつに分ける置き換えは、行の番号を振り直すことになる（10・20 の間に入れられない）ので作っていません |

### 確かめていないこと

- **running-config での形。** `distance ospf` が 1 行にまとまって表示されるか、`neighbor … maximum-prefix` の語の順、`set extcommunity rt … additive` の表示は、資料の構文の行のとおりに書いています。機器が表示する形は確かめていません
- **`mask` の無い `network` と auto-summary。** 技術資料は「auto-summary が有効なら、そのクラスのサブネットが経路表に 1 つあれば、クラスのネットワークを BGP の表に入れる」とも書いています。このパックは auto-summary を表にしていないので、広告される経路そのもの（クラスの長さ）だけを読みます
- **ピアグループと、メンバーが自分で書いた設定の優先。** メンバーの行の値を優先し、空の列だけにグループの値を入れています。機器が「メンバーの設定がグループの設定に勝つ」ことは、コマンドリファレンスの文（「メンバーはグループの設定を引き継ぐ」）から読み取ったもので、すべての設定で確かめてはいません
- **IPv6 のネイバーの `password 7`** は、IPv4 と同じ形として読み書きしています

## RIP と RIPng（別紙 A 3.27）

構文は、IP Routing: RIP Command Reference・IP Routing: Protocol-Independent Command Reference・IPv6 Command Reference で確かめました。**実機の出力では確かめていません。** 見本は `tests/import/rip/`・`tests/render/rip/` です。

### 読める行と入る表

| 分野 | 読める行 | 入る表・列 |
|---|---|---|
| RIP（IPv4） | `router rip` の下の `version 1\|2`、`[no] auto-summary`、`network <クラスのネットワーク>`、`passive-interface default`・`[no] passive-interface <IF>`、`default-information originate [route-map <名前>]`、`default-metric <数>`、`distance <数> 0.0.0.0 255.255.255.255`、`timers basic <更新> <無効> <保留> <消す>`、`distribute-list prefix <名前> in\|out [<IF>]`、`redistribute connected\|static\|ospf <番号>\|bgp <AS> [metric <数>] [route-map <名前>]` | rip_processes・rip_interfaces・rip_redistribute（種類 ipv4） |
| RIP の IF | `ip rip send version …`・`ip rip receive version …`、`ip rip v2-broadcast`、`ip rip authentication mode text\|md5`・`ip rip authentication key-chain <名前>` と `key chain <名前>` の下の `key <番号>`・`key-string <鍵>`、`[no] ip split-horizon`、`ip summary-address rip <アドレス> <マスク>` | rip_interfaces |
| RIPng | `ipv6 router rip <名前>` の下の `distance <数>`、`timers <更新> <無効> <保留> <消すまで>`、`distribute-list prefix-list <名前> in\|out [<IF>]`、`no split-horizon`・`poison-reverse`、`redistribute …`。IF の `ipv6 rip <名前> enable`・`default-information originate [metric <数>]`・`metric-offset <数>`・`summary-address <プレフィックス>` | rip_processes・rip_interfaces・rip_redistribute（種類 ipv6） |

### 読み方の決まり

- RIP（IPv4）は、`network` に含まれるアドレスを持つ IF ごとに、RIP インタフェースの行を作ります。`network` の行そのものは表に持ちません
- キーチェーンの名前は表に持ちません（書き出しでは `RIP-<IF の短い名前>` と付けます）
- RIPng は、IF が最初に指したプロセスの名前の設定だけを読みます（ほかの名前の行は「読めなかった」）。書き出しでは `ripng` という名前で書きます
- RIPng の `timers` の 4 つ目（無効になってから消すまで）は、「無効とみなすまで」を足した値（最後の更新から消すまで）にして入れます

### 書き出し

- 行のある IF のアドレスから、`network` の行を作ります。同じクラスのネットワークのアドレスを持つのに行が無い IF があると、その IF でも RIP が動いてしまうので、理由を付けて止めます（G-601）
- **書けないもの（理由を付けて止めます。G-601）**: 「受けない」、RIPng の「送らない」、RIP（IPv4）の poisoned_reverse と IF ごとのデフォルト経路の広告、RIPng のプロセスの行のデフォルト経路の広告・既定の「送らない」・再配布のメトリックの既定、IF ごとに違う RIPng のスプリットホライズン
- **書かない列（値があれば G-602）**: RIP（IPv4）の IF の「メトリックに足す値」（`offset-list` は ACL を使う形です）、再配布の「タグ」

### 読まない行（「読めなかった」として画面に出ます）

| 行 | 理由 |
|---|---|
| `neighbor <アドレス>`、`offset-list`、`ip rip triggered`、`maximum-paths`、`address-family ipv4 vrf …`（VRF の中の RIP）、RIPng の `port`・`default-information only` | 入れる列がありません |
| 2 つ目以降の RIPng のプロセス | 表は、機器に種類ごとに 1 つのプロセスを持ちます |

## 確かめていないこと

資料に決まりが書かれていないか、見つけられなかった点です。実機の出力が手に入ったら、ここから確かめます。

### 全般

- running-config の体裁（先頭行、`!` の入り方、字下げ 1 桁、`end`、banner の区切り）。資料の出力例で見ただけです
- 1 行に並ぶオプションの順（`ip route` の `name` と `track` など。どちらの順でも読めます）
- VRF・回線監視・NAT が running-config のどこに出るか。VRF はインタフェースの前、回線監視はインタフェースの後ろで経路の前、NAT はアクセスリストの後ろに書いています（使われる前に定義がある順）
- 端末ログ: 機器がページ送りの跡をどう消すか（後退の文字で消すのか、空白を重ねるのか）。資料に書かれていません。制御文字が残っているログも、残っていないログも読めるようにしてありますが、実機のログでは確かめていません
- 端末ログ: 行ごとに時刻が付くログ（端末ソフトの設定）は読めません
- 書き出し: 書き出した文面を実機に入れて確かめていません。とくに、ハッシュの始まりと種別の番号の対応（資料の例と Cisco Community の記事で見た範囲です）、`line vty 0 4` の範囲が機種に合うか、バナーの区切りの文字、すべての有効な IF に `no shutdown` を書くことの影響
- `match qos-group`・`ip name-server vrf`・`tunnel mode` の語の一覧は、検索結果の要約で見ただけです（「参照した資料」）

### 機器の基本

- `service timestamps` の語が、running-config でこの順（`msec` → `localtime` → `show-timezone` → `year`）に出ること（構文の行の順です）。別の順に出る機器の行は「読めなかった」になります

### インタフェース・トンネル・PPPoE

- PPPoE のクライアントに `pppoe enable group global` が要るか。IOS XE の設定ガイドの手順にはあり、古い IOS 12.2T の例にはありません。要ると言い切る文は見つけていません（必ず書いています）
- `dialer-group`・`dialer-list` が無くても PPPoE がつながり続けること。IOS XE の設定ガイドの手順と例に出てこないことと、コマンドリファレンスの「通信があるときだけつなぐ設定をしたときは、これらも要る」という文から、「つなぐきっかけの通信」が空の行には書いていません。古い IOS では要るかもしれません（要るなら「つなぐきっかけの通信」に `ip` と書くと、`dialer-group` と `dialer-list … protocol ip permit` を書きます）
- `ppp chap password`・`ppp pap sent-username … password` の種別の番号（`0`・`7`）。構文の行には無く、例にだけあります。取り込みは `0` 付きと番号なしを読み、書き出しは番号なしです
- PPPoE のデフォルト経路を自動で作る行。コマンドリファレンスは `ppp ipcp default route`、新しい設定例は `ppp ipcp route default` と、語の順が違います。どちらも読まず、書きません
- Dialer（PPP）の下の `keepalive` の働き。資料に記述がないので、「監視間隔」は読まず、書きません
- Dialer の下の行の並び（`encapsulation ppp` をアドレスの後ろに書いています）
- `pppoe-client dial-pool-number` の後ろの語を 2 つ以上並べられること・その順（コマンドリファレンスの構文は、語を `|` で区切って 1 つだけ選ぶ形です）。引用符で囲んだ `service-name` を機器が受けること
- `dialer-list` の番号の範囲（1 から順に振ります）

### 経路・VRF・OSPF・BGP・冗長化

- `vrf definition` の下の `description`（取り込みと書き出しの両方にありますが、読んだ資料に構文の行がありません）
- `route-target` を `address-family ipv4` の下に書いた機器で、`vrf definition` のすぐ下に書き直してよいか（どちらにも書けると資料にあります。すぐ下に書いています）
- `ip route` の `permanent` と `tag` の表示の順（構文の順で書き、どの順でも読めます）
- OSPF の `area … range` の既定が「広告する」であること（構文の行で `advertise` が省けることから、既定値として入れています。既定を述べた文は見つけていません）
- `router bgp` の直下に書いたネイバーを、IOS XE が `address-family ipv4` の下に移して表示すること（どちらの形でも読めます）
- VRRP の版 3: running-config での形（グループの下の行の順、`exit-vrrp` が出るか、IPv6 のアドレスを大文字で出すか）。どの順・どちらの形でも読めるように書いてあります。`preempt` だけの行（遅れなしの有効）を受け付けるか（書き出しでは書きません）。`track` の `decrement` を省いたときに下げる値（コマンドリファレンスに既定の値が書かれていません）。`vrrpv2` を付けたグループが、版 2 の相手と実際にどう話すか（設定ガイドの「VRRPv2 しか話さない機器とつなぐため」という説明だけで入れています）
- `no standby <グループ> preempt`・`no vrrp <グループ> preempt` が running-config に出るか（入力としては受け付けられる形です。VRRP は既定が有効なので、無効にした機器では出るはずです）

### 経路制御の方針

- ルートマップ・AS パスのリスト・コミュニティのリストが running-config のどこに出るか（プレフィックスリストの後ろに書いています）。ルートマップの下の行の並び（どの順でも読めます）
- `match community` の完全一致の語。コマンドリファレンスの構文は `exact` です。機器が `exact-match` と表示するかもしれないので、取り込みは両方を読み、書き出しは `exact` です
- `set community` に値を並べたときの `additive` の位置（構文は値の後ろです。行の最後に書いています）と、コミュニティが `AA:NN` と 1 つの数のどちらで表示されるか（`ip bgp-community new-format` によります。どちらも読めます。この行そのものは読みません）
- `set ip next-hop self`（BGP Command Reference にあります）を、ポリシールーティングの方針に書いたときの働き
- `match ip next-hop prefix-list <名前>` の形（読んだコマンドリファレンスの構文には、ACL の形しかありません。読まず、書きません）

### アクセスリスト・NAT・回線監視・DHCP

- アクセスリスト・プレフィックスリストが running-config のどこに出るか（経路の後ろに書いています）
- アクセスリストの番号が出る版と出ない版の境目
- 注記の行に番号を付ける書き方（`15 remark …`）は IOS XE 16.12 以降の設定ガイドにあるもので、それより前の版で受け付けられるかは確かめていません。行が 10 刻みのリストでは番号を書かないので、当たりません
- 名前が番号のリストを `ip access-list standard 10` と入れたとき、機器が `access-list 10 …` と `ip access-list standard 10` のどちらで出すか
- 番号のアクセスリスト（拡張）のポート付きの行が、running-config に出る形（名前付きのリストの行と同じ形として読みます）
- **running-config での語の順番。** アクセスリストの行の末尾の語は、順番を問わずに読みます。書き出しの順番は、上に書いたとおりで、資料の構文の行に合わせています。機器が表示する順番は確かめていません
- **UDP のポートの名前の番号**は、IOS の資料ではなく、ASA の設定ガイドの表によります（同じ名前が同じ番号を指す、という前提です）
- **`ip nat inside source static network` の長さ**は、`/26` の形で書き出します。機器が `/26` とマスクのどちらで表示するかは確かめていません（取り込みは両方を読みます）
- **つないだリストの名前**（`Gi0-0-in`）は、このパックが決めたものです。同じ名前の ACL がすでにあれば、理由を付けて止まります
- **時間帯の時刻**は、書かれたとおりの字（`8:00`）で持ちます。機器が `08:00` と表示する版があれば、往復の確認（G-604）で違いが出ます
- `ip nat inside source static tcp|udp <内側> <ポート> interface <IF> <ポート>` の形。コマンドリファレンスの構文の行が、この箇所で崩れています（`interface` の語があることだけ読めます）
- IP SLA の `timeout` が `threshold` より短いと受け付けられないこと。資料には「threshold は timeout を超えられない」とあり、そこから `threshold` を先に書いています。実機では確かめていません
- `track` を、それを使う `ip route … track` より先に書く必要があるか（先に書いています）
- `ip sla` の `frequency` と `timeout` が、`icmp-echo` の下に出るか横に出るか（両方を読めます）

### IPsec

- `crypto map` の `match address` が指すアクセスリストを、マップより先に定義しておく必要があるか（先に書いています）
- IKE の提案が 2 つ以上ある機器で、どの提案がどの相手に使われるか（機器が相手と折り合いを付けます）。取り込みは、番号の小さい順に 1 番目を IPsec の行、2 番目以降を「IPsec 追加の提案」に入れ、書き出しは 10・20・30… の番号で書きます
- トランスフォームセットの `esp-sha256-hmac`・`esp-sha384-hmac`・`esp-sha512-hmac`。コマンドリファレンスの語の表（A to C、2021-01-11）には SHA-2 の語が 1 つもありません。メーカーの文書に出てくるのは、例と表示だけです（`esn` の項の例 `esp-aes esp-sha512-hmac`、ASR 920 の設定ガイドの表示 `esp-sha384-hmac`・`esp-sha512-hmac`、cBR の設定ガイドの `esp-aes esp-sha256-hmac`、SD-WAN の設定ガイドの `[esp-sha384-hmac | esp-sha256-hmac]`、データモデルのページの選択肢）。3 語とも、その字面で読み書きしています
- `esp-gcm` の 128 ビットを running-config がどう表示するか（`esp-gcm` と書き、`esp-gcm 256` だけを 256 と読んでいます。コマンドリファレンスの「既定は 128 ビット」「`esp-gcm 256`」からの読みです）
- `no crypto ipsec nat-transparency udp-encaps` が IKE の版 2 の NAT 越えも止めるか（コマンドリファレンスは版を分けて書いていません。機器に 1 つの設定として、全部の行に当てています）
- `tunnel protection ipsec policy ipv4` と `tunnel protection ipsec profile` の行の並び（設定例の順で、policy を先に書いています）。このコマンドは IOS XE 16.12.1 で入ったもので、それより前の版・IOS 15 にはありません（版で止める決まりは入れていません）
- L2TP/IPsec の受け口で、動的なマップにアクセスリスト（UDP 1701）を付けない形と付ける形の、どちらが機器の推す形か（メーカーの設定例は付けていません。取り込みはどちらも読み、書き出しは対象通信の行があるときだけ付けます）
- 名前が「マップ名/番号」でない行を入れるマップの名前（`CMAP-<IF 名>`）は、このパックの決まりです（31 字などの長さの上限は確かめていません）
- `crypto isakmp key` の種別の番号 `0`（平文の印）。取り込みは `0` 付きと番号なしを読み、書き出しは番号なしです（`6` は暗号化した形で、`6 <値>` のまま持ちます。「読み方の決まり」の「IPsec」）
- IKE 版 2 の行（`crypto ikev2 …`）が running-config でどの位置・どの順に出るか、キーリングの `peer` の下の字下げ、`keyring local <名前>` と `keyring <名前>` のどちらで出るか（どちらも読めます。書き出しは `keyring local`）
- IKEv2 のキーリングの `pre-shared-key <鍵>`（`local`・`remote` の無い形）。コマンドリファレンスの構文の行は `{local | remote}` を省けない形で、「既定は対称の鍵」という文と設定例に、語の無い形があります。書き出しは、鍵が 1 つなら語の無い形です
- IKEv2 の `dpd` の再送間隔の範囲（コマンドリファレンスに範囲がありません。値は表のままを書きます）
- IKE 版 1 で、名前で書いた相手の鍵（`crypto isakmp key … hostname`）がメインモードで使われるか。コマンドリファレンスには「メインモードでは、鍵はアドレスだけで引く」とあります。書き出しはこの形で書きますが、つながるかどうかは確かめていません
- `crypto isakmp key <鍵> address 0.0.0.0`（マスクなし）と `0.0.0.0 0.0.0.0` のどちらで表示されるか（両方を読み、マスク付きで書きます）
- 動的なクリプトマップを指す項目を、マップの中でいちばん大きい番号にする決まり（コマンドリファレンスの勧め）。書き出しは、表の行の番号のままです
- `self-identity address` が送るアドレス（自分側 IF のアドレスと同じときだけ書いています）
- ISAKMP プロファイルの下の `keepalive <秒> retry <秒>` は読むだけで、書き出しは機器全体の `crypto isakmp keepalive` です
- 「有効量（キロバイト）」の範囲はメーカーで違います（Cisco は 2560 以上）。範囲の外の値も、表のままを書きます
- `match identity address 0.0.0.0`（どの相手でも）の書き方。コマンドリファレンスの構文は `address <アドレス> [<マスク>] [<fvrf>]` で、`0.0.0.0` の意味を書いた文は読めていません（`0.0.0.0` だけの形と、マスク `0.0.0.0` を付けた形の両方を読み、マスクなしで書きます）
- 外側の VRF を名前で書いた `crypto keyring <名前> vrf <VRF>` と `match identity address <アドレス> <マスク> <VRF>` の設定例（構文の行と説明だけで、値の入った例は公開資料に見つかりませんでした）。`tunnel vrf` を IPsec のトンネル IF で使った設定例も同じです（`tunnel vrf` の例は、IPsec でないトンネルです）
- IKE 版 1 の仮想テンプレートに `tunnel source` を書いてよいか（メーカーの版 1 の例には無く、版 2 の例にあります。表に「送信元」があるときだけ書きます）。仮想テンプレートの下の `no shutdown`（ほかの IF と同じに書いています）
- ISAKMP プロファイルの下の行の並び（`vrf`・`keyring`・`self-identity`・`match identity`・`virtual-template` の順で書いています）
- `nat force-encap`: コマンドリファレンスに項目が無く、設定ガイド（Session Initiation Protocol Triggered VPN）の本文で確かめました
- 種別 `6` の鍵を、`key config-key password-encryption` の親の鍵が違う機器に入れたときの動き（元に戻せないはずです。同じ機器へ書き戻す使い方を想定しています）

### QoS

- QoS の表示の形（cmdref 7.16・7.17）: 既定の `match-all` が出るか、DSCP・precedence が名前と数のどちらで出るか、`match ip dscp` の `ip` が付くか、`police` が 1 行か複数行か、`shape`・`police` の値が bps に直って出るか、クラスの下の動作の順。取り込みはどの形でも読みます
- `traffic-shape rate` の値の単位と、省いたバーストが表示で補われるか（4 つ目の数（バッファの上限）は読みません）。`rate-limit` の条件と bps の並び（「条件の後ろに bps」として読みます）。隠されたコマンド（`priority-group`・`custom-queue-list`）が 15.x の running-config に残るか
- サブインタフェースやトンネルに、絞る速さの無いキュー付きのポリシーを当てられるか（古い資料では、親のシェーピングを持つ二段のポリシーが要ります。今の IOS XE でも要るかは確かめていません。パックは足しません）
- QoS の新しい行（`police … pir …`、`shape peak`、`random-detect`、`queue-limit … bytes`）が、running-config にどの形で出るか（1 行か複数行か、既定の語が出るか）。取り込みは、コマンドリファレンスの構文の形を読みます

### L2TP

- L2TP（別紙 A 3.21）: `l2tp-class`・`pseudowire-class`・`vpdn-group` とその下の行が running-config でどの位置・どの並びで出るか（設定ガイドの例の順で書いています）。`protocol` の行が無い `pseudowire-class` を機器がどう扱うか（行が無い設定は読みません）。`cookie size` の書き方（手順の表と設定例で形が違うため、読みも書きもしません）。Virtual-Template の `encapsulation ppp` が表示されるか（設定ガイドの例にはあります。読まず、書きません）。Virtual-Template に `no shutdown` を書いてよいか（ほかの IF と同じに書いています）。`l2tp tunnel password`・`password`（`l2tp-class`）の種別の番号（`0`・`7`）は構文の行に無く、例にだけあります
- L2TP の利用者を `password`（`secret` でなく）で書いたユーザと見なす決まりは、資料の決まりではなく、このパックの読み方です（Cisco は利用者とログインのユーザを分けて持ちません）
- `Virtual-Template` の下に、LAC が PPPoE を受けるために要る行がほかにあるか（`ppp authentication` だけを書きます）。Cisco の LAC で PPPoE を受ける設定例は、資料で確かめられていません（`bba-group` と `vpdn-group` の `request-dialin` を、それぞれの構文で書いています）

### VLAN とスイッチ、IPv6、ルートマップ・BGP・OSPF

それぞれの節の「確かめていないこと」にあります。
