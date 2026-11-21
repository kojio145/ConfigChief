# NEC QX シリーズ パック（nec_qx）

NEC の QX シリーズ（L2/L3 スイッチ）のコンフィグ（`display current-configuration` の出力、保存した設定ファイル）を読み取り、VLAN・スイッチポート・LAG・スパニングツリー・VLAN の IF・管理系・L3（経路・OSPF・BGP・VRRP・VPN インスタンス・IPv6）・ACL・QoS を書き出すための決まりです。

- 版: 0.2.0
- 取り込みも書き出しも、下の「対応一覧」の全部です
- **書き出しは、ソフトウェアが「Version 7」を名乗る世代の形**です（下の資料が「Version 7.x.x 以降のソフトウェアに対応」と書いています）。**取り込みは、それより前の世代（販売終了の機種。設定の頭が `version 5.…`）の書き方も読みます**（下の「Version 7 より前の世代」）。読めない行は「未対応」として画面に出ます
- **実機の出力では確かめていません**。構文はすべて下の公開資料で確かめたものです。保存した設定に出る語の順番や、どの行が省かれるかは、資料の出力例で見た範囲にとどまります（下の「確かめていないこと」にまとめています）

## 参照した資料

マニュアルはどれも NEC の公開のサポートページ（QX シリーズの製品マニュアル）にある PDF で、2026-11-21 に読みました。

| 資料の名前 | 版 | 使ったところ |
|---|---|---|
| QX-S5600G Series Ethernet Switches Command References（GVT-070107-001-00。英語） | 1.27 | 全部の行の構文・既定値（L3 の機種。この README で「コマンドマニュアル」と書くときは、この資料） |
| QX-S5600G シリーズ Ethernet スイッチ オペレーションマニュアル | 1.27 | コマンドラインのエラーの文の表、バナーの入力の形、PVST の動き、パケットフィルタ、コマンドの省略 |
| QX-S4200G シリーズ Ethernet スイッチ コマンドマニュアル | 1.0 | L2 の機種で同じ行があること（設定ファイルの例、`stp port bpdu-protection`、ACL のポートの比べ方 `lt`・`gt`・`eq`・`neq`・`range`） |
| QX-S4200G シリーズ Ethernet スイッチ オペレーションマニュアル | 1.2 | トランクの VLAN 1 を外す `undo port trunk permit vlan 1` の例、バナーの入力の形 |
| QX-S5300G シリーズ Ethernet スイッチ コマンドマニュアル | 1.10 | 世代の確認（Version 7）、ACL のポートの比べ方 |
| QX-S3400F/S4100G/S4500G シリーズ Ethernet スイッチ コマンドマニュアル | 1.13 | 世代の確認（Version 7）、`stp global enable`・`line vty` があること |
| QX-S3800 シリーズ Ethernet スイッチ コマンドマニュアル（販売終了製品のページ） | 1.7 | Version 7 より前の世代（Version 5.x）の書き方: `user-interface`、`local-user` の `password { cipher \| simple }`・`authorization-attribute level`、`super password [ level ]`、`stp enable`、`stp edged-port { enable \| disable }`、VRRP の時間が秒であること、`voice vlan`、`ntp-service … source-interface` |
| QX-S5600G シリーズ インスタレーションマニュアル | 2.1 | 機種ごとのポートの構成が載っていること（中は読めていません。機種の定義には、まだ使っていません） |
| NEC の公式 Web の、QX シリーズの製品のページ | 2026-11-21 に見たもの | 機種の定義（`models.yaml`）のポートの数（GE が 24 または 48、10GE が 4） |

## 確かめた機種と版

実機では確かめていません。見本のコンフィグの `version 7.1.070, Release 1201` の行は、上のコマンドマニュアルの設定ファイルの例にある文字です。古い世代の見本の `version 5.1.1` も、QX-S3800 シリーズのコマンドマニュアルの例にある文字です。

## 対応一覧

取り込みと書き出しの両方ができます（「取り込みだけ」と書いた行を除きます）。

| 分野 | 行 | 入る表と列 |
|---|---|---|
| 機器 | `sysname`、`version <版>[, Release <番号>]`（「OS 版」。取り込みだけ）、`clock timezone <名前> add\|minus <時[:分]>`、`dns domain`、`dns server <アドレス> [vpn-instance <名前>]`、`header login\|motd\|shell <区切り><文><区切り>`、`super password [role network-admin] hash\|simple <値>`、`snmp-agent sys-info location\|contact` | devices（バナーは、login = 「バナー」、motd = 「バナー（接続したとき）」、shell = 「バナー（ログインの後）」） |
| 機器のログ | `info-center timestamp boot\|date`、`info-center logbuffer`（`undo` も）、`info-center source default logbuffer level <重要度>`、`info-center source <モジュール> logbuffer level <重要度>` | devices の「ログの時刻の付け方」「機器内にログを溜める」「ログに残す重要度」、log_subsystems |
| コンソール | `line aux <番号>` の下の `idle-timeout <分> [<秒>]` | devices の「コンソールの無操作切断」 |
| ユーザ | `local-user <名前> [class manage]` と、その下の `password hash\|simple <値>`、`service-type ssh\|telnet\|terminal …`・`service-type ftp`、`authorization-attribute user-role <ロール>` | users |
| 遠隔ログイン | `telnet server enable`、`ssh server enable`、`telnet\|ssh server acl <番号>`（番号が違えば「SSH だけ」「Telnet だけ」の列）、`line vty <始め> [<終わり>]` と、その下の `authentication-mode scheme\|password`、`protocol inbound all\|ssh\|telnet`、`idle-timeout <分> [<秒>]`、`set authentication password simple\|hash <値>` | remote_access |
| NTP | `ntp-service enable`、`ntp-service unicast-server <サーバ> [vpn-instance <名前>] [priority] [source <IF>] [authentication-keyid <番号>]`、`ntp-service source <IF>`、`ntp-service authentication enable`、`ntp-service authentication-keyid <番号> authentication-mode md5 cipher\|simple <鍵>`、`ntp-service reliable authentication-keyid <番号>` | ntp_servers（認証は「認証の鍵番号」「認証の鍵」） |
| syslog | `info-center loghost [vpn-instance <名前>] <転送先> [facility local0〜7]`、`info-center loghost source <IF>`、`info-center source default loghost level <重要度>` | syslog_servers |
| SNMP | `snmp-agent`、`snmp-agent community read\|write [simple\|cipher] <名前> [acl <番号>\|name <名前>]`、`snmp-agent sys-info version …`、`snmp-agent target-host trap address udp-domain <アドレス> params securityname <名前> [v1\|v2c\|v3 [authentication\|privacy]]`、`snmp-agent trap source <IF>`、`snmp-agent group v3 <グループ> [authentication\|privacy] [read-view …] [write-view …] [notify-view …] [acl …]`、`snmp-agent usm-user v3 <ユーザ> <グループ> [cipher\|simple authentication-mode md5\|sha <パスワード> [privacy-mode 3des\|aes128\|des56 <パスワード>]] [acl …]` | snmp_communities、snmp_hosts、snmp_users |
| VLAN | `vlan <番号>` と、その下の `name`・`description`。`vlan 10 to 20`・`vlan 2 4 10 to 20`（番号を並べた行。1 つずつの行になります。書き出しは 1 つずつ） | vlans |
| インタフェース | `interface <名前>` と、その下の `description`、`shutdown`、`speed 10\|100\|1000\|10000\|auto`、`duplex auto\|full\|half`、`ip address <アドレス> <マスク\|長さ> [sub]`、`ip address dhcp-alloc`、`ip binding vpn-instance <名前>`。IF の行は、下の「IF 名」に無い名前（`Route-Aggregation<番号>` など）でも作ります | interfaces |
| スイッチポート | `port link-mode bridge\|route`、`port link-type access\|trunk\|hybrid`、`port access vlan <番号>`、`port trunk permit vlan <並び>\|all`、`undo port trunk permit vlan 1`、`port trunk pvid vlan <番号>`、`port hybrid vlan <並び> tagged\|untagged`、`undo port hybrid vlan 1`、`port hybrid pvid vlan <番号>`、`voice-vlan <番号> enable` | switchports |
| スパニングツリー（ポート） | `stp edged-port`、`stp port bpdu-protection enable\|disable`、`stp enable`（`undo` も）、`stp root-protection`、`stp loop-protection`、`stp cost <数>`、`stp port priority <数>` | switchports |
| ストーム制御 | `broadcast-suppression\|multicast-suppression\|unicast-suppression <%>`・`… pps <数>`・`… kbps <数>` | switchports（速さは `500pps`・`64kbps` の形） |
| ループ検出・DHCP スヌーピング | `loopback-detection global enable vlan <並び>\|all`、`loopback-detection global action shutdown`、`loopback-detection interval-time <秒>`、ポートの下の `loopback-detection enable vlan <並び>\|all`・`loopback-detection action block\|no-learning\|shutdown`、`dhcp snooping enable`、ポートの下の `dhcp snooping trust` | l2_settings、switchports |
| ポートセキュリティ | `port-security enable`、ポートの下の `port-security port-mode autolearn`・`port-security max-mac-count <数>`・`port-security intrusion-mode blockmac\|disableport` | l2_settings の「ポートセキュリティを使う」、switchports |
| LAG | `interface Bridge-Aggregation<番号>` と、その下の `link-aggregation mode dynamic`、ポートの下の `port link-aggregation group <番号> [force]`・`lacp period short`、`lacp system-priority <数>` | interfaces、switchports、lags、l2_settings |
| スパニングツリー（全体） | `stp global enable`（`undo` も）、`stp mode mstp\|pvst\|rstp\|stp`、`stp bpdu-protection`、`stp priority <数>`、`stp root primary\|secondary`、`stp timer hello\|forward-delay\|max-age <センチ秒>`、`stp vlan <並び> priority <数>`・`… timer …`・`… root primary\|secondary`・`… enable`（`undo` も） | l2_settings、stp_vlans |
| MSTP | `stp region-configuration` と、その下の `region-name`・`revision-level`・`instance <番号> vlan <並び>`・`active region-configuration`、`stp instance <番号> priority <数>`、`stp instance <番号> root primary\|secondary` | l2_settings の「MST リージョン名」「MST リビジョン」、stp_instances |
| スタック（IRF） | `irf member <番号> priority <数>`、`irf member <番号> description <文>` | stack_members |
| ポートミラー | `mirroring-group <番号> local`、`mirroring-group <番号> mirroring-port <IF> both\|inbound\|outbound`、`mirroring-group <番号> monitor-port <IF>`（IF の下に書く形も読みます。書き出しは機器全体の形） | mirror_sessions |
| VLAN の IF | `interface Vlan-interface<番号>`。番号を「VLAN 番号（VLAN の IF）」に入れます | interfaces |
| VPN インスタンス | `ip vpn-instance <名前>` と、その下の `route-distinguisher`・`description`・`vpn-target <並び> import-extcommunity\|export-extcommunity\|both` | vrfs |
| スタティック経路 | `ip route-static [vpn-instance <名前>] <宛先> <マスク\|長さ> <ネクストホップ>\|<IF> [<ネクストホップ>] [permanent\|track <番号>] [preference <数>] [tag <数>] [description <文>]` | static_routes |
| 回線監視 | `nqa entry <名前> <タグ>` と、その下の `type icmp-echo`（`destination ip`・`frequency`・`probe timeout`・`source ip`・`source interface`・`next-hop ip`・`vpn-instance`・`reaction <番号> checked-element probe-fail threshold-type consecutive <回数> action-type trigger-only`）、`nqa schedule <名前> <タグ> start-time now lifetime forever`、`track <番号> nqa entry <名前> <タグ> reaction <番号>`、`track <番号> interface <IF> [delay negative <秒> positive <秒>]` | tracks |
| ACL | `acl basic\|advanced <番号>\|name <名前>`、`acl number <番号> [match-order config]`、その下の `description`・`rule <番号> permit\|deny …`（送信元・宛先・ポート・`established`・TCP のフラグ `ack 1` など・`icmp-type <種類> [<コード>]`・`dscp`・`precedence`・`tos <数>`・`time-range <名前>`・`fragment`・`logging`。並ぶ順は問いません）・`rule <番号> comment <文>`（その番号の行の「行の注記」に入れます）、IF の下の `packet-filter <番号>\|name <名前> inbound\|outbound` | acls、acl_entries、interfaces の「入力 ACL」「出力 ACL」 |
| 時間帯 | `time-range <名前> <時刻> to <時刻> <曜日…>`、`time-range <名前> [from <時刻> <日>] [to <時刻> <日>]` | time_ranges |
| プレフィックスリストなど | `ip prefix-list <名前> [index <番号>] permit\|deny <アドレス> <長さ> [greater-equal <数>] [less-equal <数>]`、`ipv6 prefix-list …`、`ip as-path <番号> permit\|deny <正規表現>`、`ip community-list <番号>\|basic <名前> permit\|deny <コミュニティ…>`、`ip community-list <番号>\|advanced <名前> permit\|deny <正規表現>` | prefix_lists、ipv6_prefix_lists、as_path_lists、community_lists |
| 経路制御の方針 | `route-policy <名前> permit\|deny node <番号>` と、その下の `if-match ip address prefix-list\|acl`・`if-match ip next-hop prefix-list`・`if-match as-path`・`if-match community [name] <リスト> [whole-match]`・`if-match interface`・`if-match cost`・`if-match tag`、`apply local-preference`・`apply cost [+\|-]<数>`・`apply cost-type`・`apply as-path`・`apply community … [additive]`・`apply preferred-value`・`apply origin igp\|incomplete`・`apply tag`・`apply ip-address next-hop` | route_policies |
| ポリシールーティング | `policy-based-route <名前> permit\|deny node <番号>` と、その下の `if-match acl`・`apply next-hop <アドレス> [<アドレス>]`、IF の下の `ip policy-based-route <名前>`、`ip local policy-based-route <名前>` | route_policies、interfaces の「ポリシールーティング」、devices の「機器自身の通信のポリシールーティング」 |
| OSPF | `ospf [<番号>] [router-id <ID>] [vpn-instance <名前>]` と、その下の `bandwidth-reference`、`import-route direct\|static\|bgp\|rip [cost] [tag] [type] [route-policy <名前>]`、`default-route-advertise [always] [cost]`、`silent-interface <IF>\|all`、`area <エリア>`（その下の `network <アドレス> <ワイルドカード>`、`stub [no-summary]`、`nssa [no-summary]`、`abr-summary <アドレス> <マスク\|長さ> [advertise\|not-advertise]`）。IF の下の `ospf cost`・`ospf dr-priority`・`ospf timer hello\|dead`・`ospf network-type`・`ospf authentication-mode simple cipher\|plain <鍵>`・`… md5\|hmac-md5 <番号> cipher\|plain <鍵>` | ospf_processes、ospf_interfaces、ospf_areas、ospf_ranges、ospf_redistribute |
| BGP | `bgp <AS 番号>` と、その下の `router-id`・`timer keepalive <秒> hold <秒>`・`peer <アドレス> as-number\|description\|connect-interface\|ebgp-max-hop <数>\|password cipher\|simple <値>\|timer keepalive … hold …\|ignore`、`address-family ipv4 [unicast]`（`network <アドレス> <マスク\|長さ>`、`import-route direct\|static\|ospf [<番号>]\|rip [med] [route-policy]`、`peer <アドレス> enable\|next-hop-local\|default-route-advertise\|keep-all-routes\|advertise-community\|prefix-list <名前> import\|export\|route-policy <名前> import\|export`）、`address-family ipv6 [unicast]`（`network <プレフィックス> <長さ>`、`import-route direct\|static\|ospfv3`、`peer …`） | bgp_processes、bgp_neighbors、bgp_networks、bgp_redistribute、bgp_ipv6_neighbors |
| VRRP | IF の下の `vrrp version 2\|3`、`vrrp vrid <番号> virtual-ip <アドレス>`、`… priority <数>`、`… preempt-mode [delay <センチ秒>]`（`undo` も）、`… timer advertise <センチ秒>`、`… authentication-mode simple\|md5 cipher\|plain <鍵>`、`… track <番号> priority reduced <数>` | fhrp |
| IPv6 | IF の下の `ipv6 address <アドレス>/<長さ> [eui-64\|anycast]`・`ipv6 address <アドレス> link-local`・`ipv6 address auto [link-local]`・`ipv6 mtu`・`ipv6 nd ra halt`（`undo` も）・`ipv6 nd ra interval <最大> <最小>`・`ipv6 nd ra router-lifetime`・`ipv6 nd autoconfig managed-address-flag\|other-flag`・`ipv6 nd ra prefix <プレフィックス> <有効> <推奨> [no-autoconfig] [off-link]`・`packet-filter ipv6 <番号>\|name <名前> inbound\|outbound`、`ipv6 route-static [vpn-instance <名前>] <宛先> <長さ> <ネクストホップ>\|<IF> [<ネクストホップ>] [preference] [description]`、`acl ipv6 basic\|advanced <番号>\|name <名前>` と `rule …` | interfaces の「IPv6 アドレス」、ipv6_interfaces、ipv6_ra_prefixes、ipv6_routes、ipv6_acl_entries |
| OSPFv3 | `ospfv3 [<番号>]` と、その下の `router-id`・`import-route direct\|static\|bgp4+\|ripng`・`silent-interface <IF>\|all`・`area <エリア>`（`stub [no-summary]`）、IF の下の `ospfv3 <番号> area <エリア>`・`ospfv3 cost\|dr-priority\|timer hello\|timer dead\|network-type` | ospfv3_processes、ospfv3_interfaces、ospfv3_areas |
| QoS | `traffic classifier <名前> [operator and\|or]` と、その下の `if-match any\|acl [name] <リスト>\|acl ipv6 …\|dscp <値…>\|service-dot1p <値…>\|protocol ip\|ipv6`、`traffic behavior <名前>` と、その下の `car cir <kbps> [cbs <バイト>] [ebs <バイト>] [red discard\|pass\|remark-dscp-pass <値>]`・`remark dscp`・`remark dot1p`、`qos policy <名前>` と `classifier <クラス> behavior <動作>`、IF の下の `qos apply policy <名前> inbound\|outbound`・`qos lr outbound cir <kbps>` | qos_classes、qos_class_matches、qos_policies、qos_bindings |
| DHCP | `dhcp enable`、`dhcp server forbidden-ip <始め> [<終わり>]`、`dhcp server ip-pool <名前>` と、その下の `network <アドレス> mask <マスク>`（長さも）、`address range`、`gateway-list`（2 つ目以降は「デフォルトゲートウェイ（2 つ目以降）」）、`dns-list`、`domain-name`、`expired day <日> [hour <時> [minute <分> [second <秒>]]]`・`expired unlimited`。IF の下の `dhcp select relay`、`dhcp relay server-address <アドレス>` | dhcp_pools、interfaces の「DHCP 中継先」 |

## Version 7 より前の世代（取り込みだけ）

設定の頭の `version <版>` を「OS 版」に入れ、`5.` で始まるときは、次の書き方を古い世代のものとして読みます（QX-S3800 シリーズのコマンドマニュアルで確かめた範囲）。書き出しは Version 7 の形だけです。

| 行 | 読み方 |
|---|---|
| `user-interface vty\|aux <番号> …` | `line vty`・`line aux` と同じ |
| `local-user <名前>`（`class` なし）、`password cipher <値>`、`authorization-attribute level 0〜3` | `cipher` は「パスワードの形」が encrypted。level 3 は admin、0〜2 は数のまま |
| `super password [level 3] simple\|cipher <値>` | 特権パスワード（level 1・2 の行は「未対応」）。`cipher` は encrypted |
| `set authentication password cipher <値>` | 回線パスワードが `cipher <値>` |
| `stp enable`（機器全体。`undo` も）、ポートの下の `stp edged-port enable\|disable` | `stp global enable`、`stp edged-port`（`disable` は値を入れません） |
| `vrrp vrid <番号> timer advertise <秒>`、`… preempt-mode timer delay <秒>`、`… authentication-mode simple\|md5 [cipher] <鍵>` | **時間は秒**（Version 7 はセンチ秒）。`version` の行が無い文面は、Version 7 として読みます |
| `ospf authentication-mode … [cipher\|plain] <鍵>`（`cipher`・`plain` を省いた形） | 鍵は平文として読みます |
| `ntp-service unicast-server <サーバ> source-interface <IF>` | 送信元 IF |
| `info-center timestamp log\|debugging boot\|date` | 「ログの時刻の付け方」「デバッグの時刻の付け方」 |
| `voice vlan <番号> enable`、IF 名 `Ethernet<番号>` | 音声 VLAN、100 Mbps のポート |

encrypted のユーザ・特権パスワード、`cipher` の回線パスワードは、Version 7 の形では書けません（書き出しは理由を付けて止まります。平文を入れ直します）。

## このメーカーならではの読み方

| 項目 | 読み方 |
|---|---|
| 文面の形 | 設定の部分ごとに `#` の行で区切られ、機器全体の 1 行の設定は 1 字下がっています（` sysname sw-a01`）。`#` の行で、開いている階層をすべて閉じて読みます（`pack.yaml` の `comment_closes`。別紙 B 2.2）。終わりの印は `return` |
| IF 名 | `GigabitEthernet`・`Ethernet`・`Ten-GigabitEthernet`・`FortyGigE`・`M-GigabitEthernet`・`Bridge-Aggregation`・`Vlan-interface`・`LoopBack`・`Tunnel`・`NULL`。表示コマンドに出る短い名前 `GE`・`XGE`・`BAGG` も正式な名前に直します。捨てる経路の出力 IF `NULL0` は、表では `Null0` です。ここに無い名前の IF（`Route-Aggregation<番号>` など）も、IF の行として読みますが、種別は付きません（書き出しでは止まります。下の「コンフィグの書き出し」の「IF 名」） |
| IF に当てた ACL（`packet-filter`） | **QX のパケットフィルタは、どの `rule` にも当たらない通信を通します**（コマンドマニュアルからの読み取りです。根拠と、機器で確かめてほしいことは、下の「確かめていないこと」）。表の決まりは「どの行にも当たらない通信は捨てる」なので、IF に当てているリストが「すべての通信」の行で終わっていなければ、取り込みが最後に「残りをすべて通す」行（最後の番号 + 5）を足します。IF のほかにも使っているリスト（遠隔ログイン・SNMP の接続元の制限）は、足す行でそちらの意味が変わるので足さず、`packet-filter` の行を「未対応」にします。**機器で既定を止める設定にしている場合**（この資料に無い行は「未対応」として出ます）は、足された行を消してください |
| スイッチポートの行 | Ethernet のポートと `Bridge-Aggregation` の IF は、下に行が無くてもスイッチポートです。1 行ずつ行を作り、「モード」が無ければ access を入れます（`port link-type` の既定。`defaults.yaml`）。`port link-mode route` のポートはモード routed です |
| トランク・hybrid の VLAN 1 | トランクは、何も書かなくても VLAN 1 を通します（`port trunk permit vlan` の既定）。hybrid のポートも同じで、`undo port hybrid vlan 1` が無ければ「タグなしで通す VLAN」に 1 を足します。`undo port trunk permit vlan 1` が無いトランクは、「許可 VLAN」に 1 を足します（並びの最後に付きます）。`undo …` だけで `permit` の行が無いトランクは none です |
| 許可 VLAN の並び | `2 4 50 to 100` を `2,4,50-100` として持ちます。複数の行は足し合わせます |
| LAG の束ね方 | 固定か LACP かは、LAG の IF（`Bridge-Aggregation<番号>`）の `link-aggregation mode dynamic` で決まります。メンバーの「束ね方」は、その行があれば active、無ければ on です |
| スパニングツリー | `stp global enable` が無い機器は「スパニングツリーを使う」が無効です（既定。スイッチポートがある機器だけ行を作ります）。`stp mode pvst` は rapid_pvst として持ちます（QX の PVST は VLAN ごとに RSTP を動かします）。時間はセンチ秒を秒に直します（100 の倍数でない行は「未対応」） |
| VRRP | `timer advertise`・`preempt-mode delay` はセンチ秒です。秒に直します（100 の倍数でない行は「未対応」）。回線監視は、`vrrp vrid … track <番号> priority reduced <数>` の形だけです（`switchover` は「未対応」） |
| 遠隔ログイン | 「許可する方式」は、`telnet server enable`・`ssh server enable` のあるものから、`protocol inbound` が外したものを除いて決めます。`authentication-mode scheme` は local として持ちます（AAA の既定がローカルユーザであることを前提にしています。RADIUS などを使う設定は、その行が「未対応」として出ます）。`telnet server acl` と `ssh server acl` の番号が同じ（か、動いているサーバが片方だけ）なら「接続元の制限」、違えば「SSH だけの接続元の制限」「Telnet だけの接続元の制限」に入れます |
| ユーザの権限 | `network-admin` は admin、`network-operator` は operator、ほかのロールは名前のまま。ロールが複数あるユーザは、`network-admin` があれば admin、無ければ最初の 1 つです |
| ACL の名前と番号 | 番号の ACL は番号を名前として持ちます（basic 2000〜2999 = standard、advanced 3000〜3999 = extended）。`rule` の番号を「順番」にそのまま入れます（0 から始まります） |
| ACL のアドレス | `source`・`destination` が無い側は any。ワイルドカード `0` はホスト。ポートの名前（`www`・`dns` など、コマンドマニュアルの表の名前）は番号に直します。プロトコル 50・51 は esp・ah |
| OSPF | エリアの下の `network <アドレス> <ワイルドカード>` に入る IF を、そのエリアの OSPF の IF として行にします（ほかのパックと同じ） |
| DHCP の除外 | `dhcp server forbidden-ip` は、アドレスが入るプールの「除外範囲」に入れます |
| パスワード・鍵 | 平文で入れた値を、機器はハッシュ（`hash`）や暗号化した形（`cipher`）で保存します。ユーザと特権パスワードの `hash` は、値を入れて「パスワードの形」を hashed にします。ほかの秘密の値は、機器が出したとおりに「種別 値」でセルに入れます（別紙 A 3.26）: `cipher <値>`（SNMP のコミュニティ・SNMPv3 のパスワード・VRRP と OSPF の鍵・NTP の鍵・BGP のパスワード）、`hash <値>`（回線のパスワード）。QX へはそのまま書き戻し、別メーカーへは引き継げません（チェック V-283 が知らせ、書き出しは止まります） |
| ログの重要度 | `info-center source default logbuffer level` は機器の「ログに残す重要度」、`… loghost level` はログ転送先の全部の行の「重要度」、モジュールごとの `logbuffer level` は log_subsystems。コンソールなどほかの出力先と `deny` は「未対応」です |
| SNMPv3 | ユーザの行に、グループの段・ビュー・グループの接続元の制限を写します（グループの行だけがあってユーザがいないものは「未対応」） |
| NTP の認証 | 認証の有効化・鍵・信頼する鍵の 3 つがそろったサーバに、鍵を入れます。そろわない行は「未対応」です |
| 時間帯 | 曜日は語（`Mon`・`working-day`・`off-day`・`daily` など）で読みます。数字の曜日、くり返しと日時の両方を持つ行は「未対応」です。時刻は `8:00` の形にそろえます |
| 回線監視 | ping の NQA・その予定（`start-time now lifetime forever`）・`track` の 3 つで 1 行です。間隔はミリ秒を秒に直します（1000 の倍数でなければ「未対応」）。NQA の名前とタグは持ちません |
| 経路制御の方針 | 1 つの項目の `if-match` は「すべてに合う」の意味です（コマンドマニュアル）。表の 1 つのセルに並べた名前は「どれかに合う」なので、同じ種類の `if-match` は 1 つ目だけを読み、2 つ目は「未対応」です。`apply preferred-value` は「付ける値: weight」に入れます |
| ポリシールーティング | `policy-based-route` の項目は、経路制御の方針と同じ表に入ります（IF の「ポリシールーティング」・機器の「機器自身の通信のポリシールーティング」が名前で指します） |
| BGP のネイバー | アドレスファミリの下の `peer <アドレス> enable` が無いネイバーと、`peer <アドレス> ignore` のあるネイバーは、「有効」を無効として読みます |
| QoS | クラスに何をするか（`traffic behavior`）は、ポリシーの行に入れます（動作の名前は持ちません）。`car cir` は kbps を bps に直して「上限・捨てる」、`red …` は「超えたとき」。`service-dot1p` は CoS。`qos lr outbound cir` は「IF 全体を絞る速さ」に入れます |
| ポートセキュリティ | `port-security port-mode autolearn` を「ポートセキュリティ」の有効として読み、「覚えた MAC を設定に残す」も有効にします（このモードが覚えたアドレスは、設定に残る secure MAC になります）。`intrusion-mode blockmac` は protect、`disableport` は shutdown |
| DHCP のリース | いちばん大きい単位で持ちます（`expired day 0 hour 1 minute 30` は `90m`、`expired unlimited` は `infinite`） |

## 端末ログの読み方

| 項目 | 読み方 |
|---|---|
| プロンプト | `<名前>`（user view）と `[名前]`・`[名前-view]`（設定を変える view）。機器の名前にハイフンがあると view との境目が分からないので、名前は `<名前>` の形からだけ取ります。投入ログのチェックでは、`[…]` のプロンプトの後ろに打った行を「設定のコマンド」として見ます（レポートの「モード」には、角かっこの中の全部が出ます） |
| 設定を出すコマンド | `display current-configuration`、`display saved-configuration`。語は省略できます（`dis cu`）。`display this`・`display current-configuration interface …` は、設定の全体として拾いません |
| ページ送りの跡 | `---- More ----` |

## 値を入れない規則

| 規則 | 行 | 理由 |
|---|---|---|
| `sys.sysname` | `sysname <名前>` | 機器の行を開くだけです（名前は行のキー） |
| `if.sw.bridge` | `port link-mode bridge` | スイッチポートの行を開くだけです（モードは `port link-type` か既定で決まります） |
| `if.sw.edge.disable` | `stp edged-port disable`（古い世代） | 「すぐ転送にする」が空であることと同じ行です |
| `ospf.area`・`ospfv3.area` | `area <エリア>` | 子の行を持つ入れ物です |
| `bgp.af4`・`bgp.af6` | `address-family ipv4\|ipv6 [unicast]` | 子の行を持つ入れ物です |
| `acl6.basic`・`acl6.advanced` | `acl ipv6 basic\|advanced …` | 子の行（rule）がリストの名前を持ちます（IPv6 のリストそのものの行は、表にありません） |
| `qos.policy` | `qos policy <名前>` | 子の行を持つ入れ物です |
| `stp.region.active` | `active region-configuration` | リージョンの設定を働かせる合図で、値を持ちません（書き出しでは、リージョンの設定の最後にいつも書きます） |
| `con`・`mirror.group`・`sw.hybrid.no1` ほか | `line aux …`、`mirroring-group <番号> local`、`undo port hybrid vlan 1` | 行を開くだけ、または「VLAN 1 を足さない」ことを表す行です |

## 読んで、表には入れない語

| 行 | 入れない語 | 書き出しでは |
|---|---|---|
| `port link-aggregation group <番号> force` | `force`（入るときに LAG の IF の設定を写す指定） | 付けません |
| `line vty <始め> <終わり>`、`line aux <番号>` | 行の番号 | `line vty 0 63`・`line aux 0` と書きます |
| `version <版>, Release <番号>` | Release の番号 | 書きません（「OS 版」は生成に使わない列です） |
| `nqa entry <名前> <タグ>`、`traffic behavior <名前>` | NQA の名前とタグ、動作の名前 | `nqa entry track <番号>`、`traffic behavior <ポリシー名>-<クラス名>` と書きます |
| `acl number <番号> match-order config` | `match-order config`（機器の既定と同じ） | 書きません |
| `ip prefix-list <名前> …`（`index` なし） | — | 10 刻みで振った番号を `index` として書きます |
| `authorization-attribute user-role …` が複数 | 2 つ目以降のロール（`network-admin` があればそれを残します） | 1 つだけ書きます |
| `snmp-agent`、`snmp-agent sys-info version …` | エージェントの有効化と、話す版 | コミュニティか通知先があれば `snmp-agent` と `snmp-agent sys-info version v2c`（版 1 か、版が空の通知先があれば `v1 v2c`）を書きます |
| `ntp-service enable`、`dhcp enable` | 機能の有効化 | NTP サーバ・DHCP プール・DHCP 中継があれば書きます |
| `idle-timeout <分> 0` | 秒の 0 | 分だけを書きます |
| `car cir … green pass` | `green pass`（機器の既定と同じ） | 書きません |

## 「無視」にする行

| 行 | 理由 |
|---|---|
| `return` | 終わりの印 |

## 読まない行（「未対応」として画面に出ます）

上の「対応一覧」に無い行は、「未対応」として画面に出ます。資料で見て、読まないと決めた主な行は次のとおりです。

| 行 | 理由 |
|---|---|
| `vlan all` | 4094 個の行になるので、読みません |
| L3 の LAG（`interface Route-Aggregation<番号>`）の下の `link-aggregation mode dynamic` | 読んだコマンドマニュアル（L3 の機種）に、この IF を作る行がありません。書き方を確かめられないので、LAG としては読みません。`interface Route-Aggregation<番号>` の行そのものと、下のアドレスなどの行は、種別の無い IF として読みます。メンバーのポートの `port link-aggregation group <番号>` は、L2 の LAG（`Bridge-Aggregation<番号>`）のメンバーと同じ行になるので、取り込んだ後に LAG の表を見直してください。書き出しでは、この名前の IF は止まります |
| `irf domain`・`irf-port <番号>` と、その下の `port group interface …`・`irf mac-address …`・`irf auto-update …`（スタックの口のつなぎ方） | 入れる列がありません（スタックの口は、機種と配線で決まります。メンバーの優先度と説明だけを読みます） |
| `stp instance <並び> …`（番号を並べた行）、`stp [instance\|vlan …] cost`・`… port priority`（インスタンス・VLAN ごとのポートの値）、`stp pathcost-standard`、`stp timer-factor` ほか | 入れる列がありません |
| `voice-vlan mac-address …`・`voice-vlan mode …`・`voice-vlan security …`・`voice-vlan qos …`・`voice-vlan aging` | 入れる列がありません（音声 VLAN の番号だけを読みます） |
| `port-security port-mode <autolearn のほか>`（802.1X・MAC 認証と組のモード）、`port-security intrusion-mode disableport-temporarily`、`port-security mac-address security …`（覚えた・手で書いたアドレス）、`port-security ntk-mode`、`port-security timer …` | 入れる列がありません（覚えたアドレスそのものは、表に持たない決まりです。別紙 A 3.22） |
| `mirroring-group <番号> remote-source\|remote-destination`・`… remote-probe vlan`・`… monitor-egress`・`… reflector-port`、ポートを並べた `mirroring-port`（`… to …`） | リモートのミラーは、入れる列がありません |
| `link-aggregation load-sharing …`・`link-aggregation selected-port …` | 入れる列がありません |
| `speed 40000`、`speed auto 10 100 …`、`jumboframe enable`、`combo enable`、`flow-control`、`mdix-mode` | 「速度」の選択肢に無い値と、入れる列が無い行 |
| `header legal\|incoming …`、区切りの文字を最初の行に書かないバナー（`header login` だけの行で始まり `%` で終わる形） | 入れる列がありません。区切りを書かない形は、今の書式（`blocks`）で読めません |
| `clock timezone …` の、秒のある時差（`add 05:30:30`）、`dns domain <名前> vpn-instance …` | 表に、その形がありません |
| `local-user … class network`、ユーザの `service-type lan-access\|web\|portal`、`password-control …`、`role …`、`domain …`、`radius scheme …`、`super password role <network-admin のほかのロール> …`、`super password level 1\|2 …`（古い世代） | 入れる列がありません |
| `line vty`・`line aux` の下の `user-role …`・`authentication-mode none`、`user privilege level`（古い世代）、`line class …` | 入れる列がありません |
| `ssh server …`（`enable`・`acl <番号>` のほか）、`telnet server acl mac …`、`telnet server ipv6 acl …`、`ssh user …`、`public-key …` | 入れる列がありません |
| `ntp-service unicast-server … version\|maxpoll\|minpoll …`、`ntp-service refclock-master`、古い世代の `ntp-service unicast-server vpn-instance <名前> <サーバ>`（VPN インスタンスを先に書く形） | 入れる列がありません |
| `info-center loghost … port <番号>`・`… ipv6 …`・`… channel …`（古い世代）、`info-center source … console\|monitor\|logfile …`・`… deny`、`info-center logbuffer size <件数>`（表の大きさはバイト）、`info-center timestamp none`・`info-center timestamp loghost …`、`undo info-center enable`、`info-center logfile …`・`info-center synchronous` | 入れる列がありません |
| `snmp-agent community … mib-view …`、`snmp-agent mib-view …`、`snmp-agent local-engineid`、`snmp-agent group v1\|v2c …`・`snmp-agent usm-user v1\|v2c …`、`snmp-agent target-host inform …`・`… udp-port …`・`… dscp …`・`… vpn-instance …`・`… ipv6 …`、`snmp-agent trap enable …`、IPv6 のリストを指す `acl ipv6 …` | 入れる列がありません |
| `ip route-static … bfd …`、ネクストホップが別の VPN インスタンスにある経路（`… vpn-instance <名前> <ネクストホップ>`）、`ip route-static default-preference` | 入れる列がありません |
| `acl number <番号> name <名前>`（番号と名前の両方を持つリスト。古い世代）、`acl … match-order auto`、`acl mac …`、`acl ipv6 number …`、`step`（番号の刻みを変えると、機器が rule の番号を振り直します。書き出す位置と結果を確かめていません）、`rule … counting`・`vpn-instance …`・`tos <名前>`・TCP のフラグの `ack 0` など、IPv6 の rule の `time-range`・`flow-label`・`routing`・ICMPv6 の名前（`icmp6-type echo-reply` など）・`ipv6-esp`・`ipv6-ah` | 入れる列がありません（IPv6 の行は、番号の種類とコードの組だけを読みます） |
| `packet-filter mac …`、`packet-filter … hardware-count`、`packet-filter filter …`、IF のほかにも使っているリストの `packet-filter`（上の「IF に当てた ACL」） | 入れる列がありません |
| `time-range` の、数字の曜日・くり返しと日時の両方を持つ行 | 表の 1 行は、くり返しか日時のどちらかです |
| `ospf` の `import-route … nssa-only\|allow-ibgp`、`abr-summary … cost …`、`stub\|nssa default-route-advertise…`、`authentication-mode …`（エリア）、`ospf network-type p2mp unicast`、`filter …`、`preference …`、`router id` | 入れる列がありません |
| `bgp` の `group …`（ピアグループ）、`ip vpn-instance …`（VPN インスタンスの中の BGP）、`network <アドレス>`（マスク無し。省いたときの長さを確かめていません）、`network … route-policy`、`import-route … allow-direct` ほか、`peer … route-update-interval\|fake-as\|substitute-as\|capability-advertise …`、IPv6 のネイバーの `peer … timer`、`ebgp-max-hop`（数なし）、ほかのアドレスファミリ | 入れる列がありません |
| `route-policy` の `if-match route-type\|extcommunity\|ipv6 …\|ip route-source`、`apply extcommunity\|comm-list … delete\|origin egp\|preference\|ipv6 next-hop`、`apply as-path … replace`、同じ種類の 2 つ目の `if-match`。`policy-based-route` の `apply next-hop … direct\|track …\|vpn-instance …`、IPv6 のポリシールーティング | 入れる列がありません |
| `nqa entry` の `type icmp-echo` のほか（`tcp`・`udp-echo`・`http` など）、`probe count`、`reaction … threshold-type accumulate`・`action-type none\|trap-only`、`nqa schedule` のほかの形、`track … bfd …`・`track … interface … protocol ipv4\|ipv6` | 入れる列がありません |
| `vrrp vrid … track … switchover`、`vrrp ipv6 …` | 入れる列がありません |
| `ip vpn-instance` の下の `vpn-target`（向きの語が無い行）・`import route-policy` ほか、`vpn-instance` の下の `address-family` | 向きを省いたときの意味を確かめていません。ほかは入れる列がありません |
| `ipv6 address dhcp-alloc`・`ipv6 dhcp …`（DHCPv6）、`ipv6 nd ra prefix`（時間を省いた形）、`ipv6 nd …` のほかの行、`ospfv3` の `default-route-advertise`・`abr-summary`・`import-route … cost\|route-policy`、`ospfv3 … instance <番号>` | 入れる列がありません（DHCPv6 は、書き方を確かめていません） |
| `traffic classifier` の `if-match customer-dot1p\|customer-vlan-id\|service-vlan-id\|source-mac\|destination-mac\|acl mac …`、名前を並べた `if-match dscp`、`traffic behavior` の `filter deny\|permit`・`accounting`・`redirect`・`mirror-to`・`remark local-precedence` ほか・`car … pir …`・`yellow …`、`classifier … insert-before`、`qos apply policy … global`、`qos lr inbound`、`qos gts`・`qos wrr`・`qos map-table`・`qos trust`・`qos priority` | 入れる列がありません（キューの割り当ては、QX では別の設定です。確かめていません） |
| DHCP の残り: `network … secondary`、プールの中の `forbidden-ip`、`static-bind …`、`option …`、`dhcp server apply ip-pool`、`dhcp relay … class\|vpn-instance`、古い世代の `dhcp relay server-group`・`dhcp relay server-select`、`dhcp snooping` のほかの行 | 入れる列がありません |
| `lldp …`、`rip …`・`ripng …`、`pim …`、`igmp-snooping …`、`dot1x …`・`mac-authentication …`、`interface Tunnel<番号>` の下のトンネルの行（`source`・`destination` など。IF の行とアドレスは読みます。`interface Tunnel<番号> mode …` と書いた行は、IF の行も作りません）、`vxlan`・`evpn`・`mpls` ほか | このパックでは読みません（表が無いか、QX の書き方を確かめていません） |

## コンフィグの書き出し

機器が保存する形に合わせて、部分ごとに `#` の行で区切り、機器全体の 1 行の設定は 1 字下げて書きます。

| 項目 | 書き方 |
|---|---|
| 順番 | 機器・ログ・ユーザ → VPN インスタンス → スパニングツリーと VLAN → 時間帯・ACL・プレフィックスリスト・経路制御の方針 → 回線監視（NQA と track）→ QoS → DHCP → インタフェース（LAG の IF が先。ポートは、ある LAG にしか入れないため）→ ポートミラー → スタティック経路・OSPF・IPv6 の経路・OSPFv3・BGP → syslog・SNMP・NTP → 追加コンフィグ → バナー・コンソール・遠隔ログイン → `return` |
| 時刻帯 | `clock timezone JST add 9`（分があれば `add 5:30`）。名前の無い値はエラーです |
| パスワード | 平文は `password simple <値>`、ハッシュは `password hash <値>`。ハッシュは `$h$` で始まる値だけを書きます（別メーカーのハッシュはエラー）。形が encrypted（古い世代の `cipher`、別メーカーの暗号化した形）はエラーです |
| 機器が保存した形の値 | `cipher <値>`・`hash <値>`（回線パスワード）は、そのまま書き戻します。空白のあるほかの値（別メーカーの形）は、置き換えの手順がエラーにします |
| 特権パスワード | `super password role network-admin simple\|hash <値>` |
| ユーザ | `local-user <名前> class manage`。`service-type` は「使える接続方式」のとおり（ftp は別の行）。権限は admin・operator か、QX のロールの名前（数だけの値はエラー。0〜15 は置き換えで `level-<数>` になります） |
| スイッチポート | access は `port link-type access` と `port access vlan`。hybrid は `port link-type hybrid`、VLAN 1 を通さないなら `undo port hybrid vlan 1`、`port hybrid vlan <並び> tagged\|untagged`、`port hybrid pvid vlan`。trunk は `port link-type trunk`、VLAN 1 を通さないなら `undo port trunk permit vlan 1`、`port trunk permit vlan <並び>`（10 個ずつの行）、`port trunk pvid vlan`。モードが routed のポートは `port link-mode route` |
| LAG | LAG の IF に、メンバーの束ね方が active なら `link-aggregation mode dynamic`。メンバーに `port link-aggregation group <番号>`。同じ LAG で on と active が混ざるとエラーです。LAG の IF の行が無ければ、置き換えで足します |
| スパニングツリー | `stp global enable`（無効なら `undo stp global enable`）、`stp mode`、`stp bpdu-protection`、`stp priority`、`stp vlan <並び> priority`・`timer …`（秒をセンチ秒に）。優先度は 4096 の倍数（0〜61440）でなければエラーです |
| IF 名 | 書けるのは `GigabitEthernet`・`Ethernet`・`Ten-GigabitEthernet`・`FortyGigE`・`M-GigabitEthernet`・`Bridge-Aggregation`・`Vlan-interface`・`LoopBack`・`NULL` で始まる名前です。ほかの名前（`Tunnel<番号>`・`Route-Aggregation<番号>`、別メーカーの IF 名）と、サブ IF（名前に `.` がある IF）はエラーです。QX から取り込んだデータでも、`Tunnel`・`Route-Aggregation` の IF があれば止まります（行を消すか、追加コンフィグに書きます） |
| VLAN の IF | IF 名は `Vlan-interface<番号>`。「VLAN 番号（VLAN の IF）」が IF 名の番号と違う行、ほかの名前の IF にこの番号がある行はエラーです（別メーカーから書き換えたときに、IF 対応の名前の付け間違いに気づくため） |
| IF のアドレス | `ip address <アドレス> <マスク>`（セカンダリは `sub`）。L2 のポートにアドレスがあるとエラーです |
| MSTP | `stp region-configuration` の下に、リージョン名・リビジョン・インスタンスの VLAN を書き、最後に `active region-configuration` を書きます |
| 回線監視 | ping の行は `nqa entry track <番号>`（`type icmp-echo`）と `nqa schedule track <番号> start-time now lifetime forever`、`track <番号> nqa entry track <番号> reaction 1`。間隔と「断と判断する回数」が要ります。方法が route・list の行はエラーです |
| 経路制御の方針 | IF の「ポリシールーティング」か機器の「機器自身の通信のポリシールーティング」が指す名前の方針は `policy-based-route`、ほかは `route-policy`。同じ種類の条件が 2 つ以上ある項目はエラーです（QX では「すべてに合う」の意味になるため） |
| QoS | クラスごとに `traffic behavior <ポリシー名>-<クラス名>` を作ります。クラス default・local・dynamic、帯域・優先・シェーピングの列は書けません（エラーか G-602） |
| BGP | ネイバーは `peer … as-number` と、アドレスファミリの下の `peer … enable`。「有効」が無効のネイバーは `peer … ignore` を書き、`enable` を書きません。アドレスファミリの前に ` #` を書きます |
| IPv6 の ACL | 番号が 2000〜2999 か、名前のリストで送信元だけを見るものは `acl ipv6 basic`、ほかは `acl ipv6 advanced` |
| ACL | 名前が番号なら `acl basic\|advanced <番号>`（basic 2000〜2999、advanced 3000〜3999 の外はエラー）、ほかは `acl basic\|advanced name <名前>`（英字で始まる名前）。行は `rule <順番> …`。プロトコル esp・ah は 50・51。ポートは番号で書きます。ICMP の種類は、番号（コードも）か、コマンドマニュアルの表で確かめた名前だけ。行の注記は `rule <番号> comment <文>` |
| スタティック経路 | `ip route-static [vpn-instance <名前>] <宛先> <長さ> [<IF>] [<ネクストホップ>] [permanent\|track <番号>] [preference] [tag] [description]`。出力 IF の `Null0` は `NULL0` |
| OSPF | `ospf <番号> [router-id <ID>]`。エリアの下に、OSPF の IF のアドレスを `network <アドレス> 0.0.0.0` と書きます（IF に固定のアドレスが無ければエラー）。受動は `silent-interface` |
| VRRP | 版は IF に 1 つ（`vrrp version`。グループで違えばエラー、1 はエラー）。時間は秒をセンチ秒にします |
| DHCP | プールか中継があれば `dhcp enable`。除外範囲は `dhcp server forbidden-ip`。リースは `expired day <日> [hour <時> [minute <分> [second <秒>]]]`、infinite は `expired unlimited` |
| 遠隔ログイン | 「許可する方式」にある `telnet server enable`・`ssh server enable`、`接続元の制限` は番号だけ（`telnet\|ssh server acl <番号>`）。`line vty 0 63` に `authentication-mode scheme\|password`、方式が 1 つなら `protocol inbound ssh\|telnet`。認証 aaa と SSH の版 1 はエラーです。「コンソールの無操作切断」は `line aux 0` の `idle-timeout` |
| バナー | `header motd\|login\|shell <区切り><文><区切り>`。区切りは、文に無い 1 文字（`%` `@` `~` `\|` `^` の順に探します） |
| 書かない列（値があれば G-602） | インタフェースのサブ IF・MTU・帯域・NAT・代理 ARP・ブリッジグループ・「入力／出力 ACL（2 つ目以降）」、スイッチポートのストームのときの動作、スタティック経路の「メトリック」ほか、OSPF の「隣接の変化の記録」、回線監視のしきい値・復旧の回数など、QoS の帯域・優先・シェーピングの列、ipv6_interfaces の DHCPv6・ND プロキシ・RA の DNS、L2 の全体設定の「ブリッジと L3 の併用」、そのほかこのパックが読まない表（NAT・トンネル・PPPoE・IPsec・L2TP・ブリッジグループ） |

## 書き換えで置き換えること（adapt.yaml）

別メーカーのデータを QX で書くとき、QX に同じ行が無いものを、パックが置き換えて書きます。置き換えたことは G-614 で知らせます（パラメータブックの値は変えません）。

| 元のデータ | QX での書き方 |
|---|---|
| 「スパニングツリーを使う」が空で、スイッチポートがある | 有効にして `stp global enable` を書きます（QX は、書かなければスパニングツリーが働きません） |
| 方式 pvst | rapid_pvst（`stp mode pvst`） |
| 「アクセスポートをすぐ転送にする」（全体の既定） | アクセスポートごとに `stp edged-port`。ポートの off は、その行を書かないだけです |
| 「すぐ転送にする」の trunk | on（`stp edged-port` は 1 つ） |
| 拡張システム ID、VTP、「L3 の転送」 | 書きません（当たる行がコマンドマニュアルにありません） |
| デフォルトゲートウェイ | デフォルト経路（`ip route-static 0.0.0.0 0 <アドレス>`）。0.0.0.0/0 の経路がすでにあればエラーです |
| スイッチポートの行が無いポート・LAG の IF、モードが空の行 | access として書きます（QX の既定） |
| トランクの「許可 VLAN」が空 | all（`port trunk permit vlan all`。QX は、書かなければ VLAN 1 だけです） |
| 「モードの自動交渉を止める」、タグ方式 dot1q、ポートのスパニングツリーのガード none | 書きません（知らせません。QX は交渉せず、タグは 802.1Q だけです。ガードを機器全体の既定にする行は、QX にありません） |
| ストーム制御 `5.00` | `5`（整数の %。小数のある値はエラー） |
| LAG の IF の行が無い | `interface Bridge-Aggregation<番号>` を足し、最初のメンバーのモード・VLAN を写します |
| LAG の束ね方 passive・auto・desirable・空 | passive は active（passive を選ぶ行がコマンドマニュアルにありません）、auto・desirable（PAgP）は active（相手も LACP に変える必要があります）、空は on |
| ユーザの「使える接続方式」が空 | terminal・ssh・telnet |
| ユーザの権限が 0〜15 の数 | `level-<数>` |
| SSH の版 2 | 書きません（知らせません。QX の SSH サーバは、`ssh server compatible-ssh1x enable` を書かなければ SSH2 だけです） |
| 番号の ACL（Cisco の番号） | QX の番号に付け替えます: 標準 1〜99 → 2001〜2099（2000 を足す）、標準 1300〜1999 → 2100〜2799、拡張 100〜199 → 3000〜3099、拡張 2000〜2699 → 3100〜3799。リスト・その行と、リストを指す列（IF の入力／出力 ACL、遠隔ログイン・SNMP の接続元の制限、QoS の条件）を読み替えます。付け替えた先の番号がすでにあるとき、経路制御の方針がそのリストを指しているときは、エラーです（別紙 A 3.24） |
| ACL の注記の行 | 次の行の「行の注記」にして書きます（`rule <番号> comment`）。後ろに行が無い注記は書きません |
| IF に当てた ACL の最後が「すべて」の行でない | 最後に「残りをすべて止める」行（`rule <最後の番号 + 5> deny`）を足します（QX のパケットフィルタは、どの行にも当たらない通信を通すため） |
| 経路制御の方針の、じかに書いたコミュニティの条件 | コミュニティのリスト `CL-<方針名>-<順番>` を作って、それを指します |
| 回線監視（ping）の「断と判断する回数」が空 | 1 回 |
| ポートセキュリティの「覚えた MAC を設定に残す」が有効でない | 有効として書きます（QX の autolearn は、覚えたアドレスを残します）。違反のときの動作 restrict は protect。ポートセキュリティを使うポートがあれば `port-security enable` を足します |
| スタックのメンバーの「機種の種別」、ユーザの「元に戻せる形で持つ」 | 書きません |
| 「パスワードを暗号化して保存」、ログの時刻に足すもの・デバッグの時刻、機器内ログの大きさ（バイト）・いっぱいのとき、「NTP サーバとして動く」「転送のキャッシュ」 | 書きません（当たる行がありません）。ログの時刻の付け方 time は datetime |
| 別メーカーの機器が暗号化した形の値（`7 <値>`・`secret <値>` など） | エラーです（平文を入れ直します） |

書けないものはエラー（G-601）で止め、直し方を知らせます: モード dynamic_auto・dynamic_desirable、タグ方式 isl・negotiate、名前の ACL を遠隔ログインの「接続元の制限」に使う、HSRP、サブ IF、QoS のクラス default・local、ほかのメーカーの IF 名（IF 対応で名前を決めます）。

QX のデータを Cisco・NEC IX で書くときの置き換えは、その 2 つのパックの `adapt.yaml` にあります（それぞれの README の、NEC QX のデータを書くときの節）。主なものは次のとおりです。

| QX のデータ | Cisco・NEC IX での書き方 |
|---|---|
| 「スパニングツリーを使う」の有効 | 書かずに知らせます（どちらのパックにも、スパニングツリーを有効にする行がありません。NEC IX は「優先度（機器全体）」も書きません） |
| 方式 rstp・stp | Cisco では mst として書きます。「優先度（機器全体）」は、MST のインスタンス 0 の優先度になります |
| VLAN の説明、ユーザの「使える接続方式」 | 書かずに知らせます（VLAN の説明は Cisco） |
| ユーザのロール `level-<数>` | Cisco では、特権レベルの数にします |
| 0 から始まる ACL の順番 | Cisco では、同じ並びのまま 10・20・30 … に振り直します（ICMP のコード・ToS・時間帯・行の注記も写します） |
| Cisco の番号でない番号の ACL（QX の 2000〜3999 など） | Cisco では、名前 `ACL-<番号>` に付け替えます（リストを指す列も読み替えます。経路制御の方針が指しているときはエラー） |
| hybrid のポート | Cisco では、タグなしで通す VLAN がネイティブ VLAN の 1 つだけのときに trunk に置き換えます。ほかはエラーです。NEC IX はエラーです |
| 機器が保存した形の値（`cipher …`・`hash …`）、encrypted の特権パスワード、OSPF の認証 hmac_md5 | エラーです（平文を入れ直します。hmac_md5 は、両端の機器で確かめたうえで md5 に書き直します） |
| 速さで書いたストーム制御（`500pps`・`64kbps`） | Cisco では、そのまま書けます |

## 投入ログの読み方

| 機器の返事 | 扱い |
|---|---|
| `% Unrecognized command found at '^' position.`、`% Incomplete command found at '^' position.`、`% Ambiguous command found at '^' position.`、`% Too many parameters.`、`% Wrong parameter found at '^' position.` | 弾かれた（オペレーションマニュアルの「コマンドラインのエラーメッセージ」の表の全部） |
| ほかの `% ` で始まる行 | 分からない返事として人に見せます |
| `%Feb 24 15:04:29:663 2013 …` の形の行（機器のログ） | 指摘にしません（`%` の後ろに空白がありません） |

## 確かめていないこと

- **実機の出力の全部**。特に、保存した設定での語の順番（`rule` の中の語は、どの順でも読みます。書き出しは資料の例の順）、既定値の行が省かれること、`interface` の下の行の並び
- **IF に当てた ACL（`packet-filter`）で、どの `rule` にも当たらない通信を通すこと**は、コマンドマニュアルの `display packet-filter` の表示項目の説明から読み取ったものです（既定の動作は Permit か Deny で、Deny を適用できなかったときは「permit のまま働く」。既定を Deny に変える行は、この資料にありません。既定の動作を言い切る文は、コマンドマニュアルにもオペレーションマニュアルにも見つけていません）。取り込みが足す「残りをすべて通す」行と、書き出しが足す「残りをすべて止める」行は、この読み方によります。**機器で 1 度確かめてください**（`display packet-filter interface …` の default action）
- `acl` を遠隔ログイン・SNMP の接続元の制限に使ったときに、どの rule にも当たらない接続元がどうなるか（資料に見つけていません。このパックは何も足しません）
- Version 7 より前の世代は、1 つの機種（販売終了の L3 の機種）のコマンドマニュアルで確かめた行だけです。その世代の VRRP の版（資料に版を選ぶ行が無く、取り込みでは Version 7 の既定の 3 が入ります）、`stp` の時間の単位（資料ではセンチ秒）のほかの違いは見ていません
- ACL のポートの比べ方。L3 の機種のコマンドマニュアルは `eq` だけを挙げ、ほかの 3 つの資料は `lt`・`gt`・`eq`・`neq`・`range` を挙げています。5 つとも読み書きします。機種が受け付けない比べ方は、投入のときに機器が弾きます
- `super password` をロール無しで書いた行が、`network-admin` のためのものであること（`super default role` を変えていれば別のロールです）
- `undo vrrp vrid <番号> preempt-mode`・`undo stp global enable`・`info-center logbuffer`（既定と同じ行）が、保存した設定に出ること
- `header` の、保存した設定での形（入力の形 `header login %文%` と、区切りを最初の行に書く複数行の形で読み書きします）
- `port link-aggregation group` を書く位置、`port-security max-mac-count` と `port-mode autolearn` の順（数を先に書いています）、`ip binding vpn-instance` の位置（アドレスの前に書いています）
- `bgp` の下の区切り（` #`）と `address-family ipv4 unicast` の形は、コマンドマニュアルの入力例から。保存した設定で `unicast` が付くかは、どちらでも読みます
- `apply cost + 10` の、符号と数の間の空白（空白ありで書き、どちらでも読みます）
- `mirroring-group` の行が、保存した設定で機器全体の形か IF の下の形か（どちらも読み、機器全体の形で書きます）
- QX の `hmac-md5`（OSPF）が、別メーカーの MD5 認証と同じ動きか（別メーカーへは止めています）、`apply preferred-value` と Cisco の weight が同じ扱いであること（どちらも、その機器の中だけで効く、いちばん先に比べる値として対応させました）
- 短い IF 名（`GE`・`XGE`・`BAGG`）が、表示コマンドのどこに出るか。端末ログで省略したコマンド（`dis cu` など）が、どこまで短くできるか
- QX シリーズの中の、機種ごとの違い（L2 だけの機種に無い行、ポートの数、`port link-mode` の有無）。機種の定義（`models.yaml`）は、下の「機種の定義」の 2 つの形だけです
- 機種の定義の、**ポートの番号の付け方（GE の続きの番号が 10GE に付くこと）**。資料の文では確かめていません（インスタレーションマニュアルを読めていません。コマンドマニュアルの例の `Ten-GigabitEthernet1/0/49` からの読み取りです）。ポートの数は、製品のページで確かめています
- L3 の LAG（`Route-Aggregation`）の書き方（読んだコマンドマニュアルに、この IF を作る行がありません）

## 機種の定義（`models.yaml`。別紙 B 6 章）

「機器対応」の「新機種」に下の ID を書くと、IF 対応の新 IF 名の案を、機種のポートの並びから作ります（旧機器のポートと同じ名前のポートがあれば、その名前のまま。無ければ、空いているポートに名前の順に。止めてあるポートは `-`。VLAN の IF は `Vlan-interface<VLAN 番号>`、LAG の IF は `Bridge-Aggregation<番号>`、ループバックは `LoopBack<番号>`）。ID は製品の名前ではなく、ポートの並びを言う名前です（数字はポートの数）。

| ID | ポート |
|---|---|
| `qx-24ge-4xge` | `GigabitEthernet1/0/1`〜`24`、`Ten-GigabitEthernet1/0/25`〜`28` |
| `qx-48ge-4xge` | `GigabitEthernet1/0/1`〜`48`、`Ten-GigabitEthernet1/0/49`〜`52` |

- 根拠: ポートの数は、製品のページ（上の「参照した資料」）。**ポートの番号の付け方は、資料の文では確かめていません**（上の「確かめていないこと」）。案は人が見て直す前提のものです
- 同じ名前のポートが無い旧機器のポート（`TenGigabitEthernet1/1/1` など）は、空いているポートに名前の順に割り当てます（GE のポートから）。アップリンク（10GE）へつなぐポートは、案を直してください
- 光のポートを別に持つ形、40GE を持つ形、スタックの 2 台目（`GigabitEthernet2/0/…`）は定義していません（案件フォルダのパックに `models.yaml` を置けば足せます）
