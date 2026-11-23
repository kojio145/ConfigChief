# パラメータブックのシートと列

このページは、プログラムが使っている定義から作っています（`go run ./cmd/docgen`）。手で直さないでください。

- シートの名前と見出しは、表示名でも内部名でも読みます
- 「必」= 必須。「キー」= 行を見分ける列（同じキーの行は 2 つ書けません）
- 「明示」= 空のときに、書き出し先のメーカーの既定値を埋めて必ず書き出す列
- どのシートも「機器」の列に、ホスト名・`*`（全機器）・`@グループ名` を書けます（「機器」と「接続」のシートを除く）

## シートの一覧

| シート | 内部名 | キー |
|---|---|---|
| [機器](#devices) | devices | hostname |
| [インタフェース](#interfaces) | interfaces | device、name |
| [スタティック経路](#static_routes) | static_routes | device、vrf、prefix、next_hop、out_if |
| [ユーザ](#users) | users | device、name |
| [遠隔ログイン](#remote_access) | remote_access | device |
| [NTP サーバ](#ntp_servers) | ntp_servers | device、server |
| [ログ転送先](#syslog_servers) | syslog_servers | device、server |
| [SNMP コミュニティ](#snmp_communities) | snmp_communities | device、community |
| [SNMP 通知先](#snmp_hosts) | snmp_hosts | device、host |
| [VRF](#vrfs) | vrfs | device、name |
| [OSPF](#ospf_processes) | ospf_processes | device、process_id |
| [OSPF インタフェース](#ospf_interfaces) | ospf_interfaces | device、process_id、if_name |
| [OSPF エリア](#ospf_areas) | ospf_areas | device、process_id、area |
| [OSPF 集約](#ospf_ranges) | ospf_ranges | device、process_id、area、prefix |
| [OSPF 再配布](#ospf_redistribute) | ospf_redistribute | device、process_id、source |
| [BGP](#bgp_processes) | bgp_processes | device、vrf |
| [BGP ネイバー](#bgp_neighbors) | bgp_neighbors | device、vrf、neighbor |
| [BGP 広告経路](#bgp_networks) | bgp_networks | device、vrf、prefix |
| [BGP 再配布](#bgp_redistribute) | bgp_redistribute | device、vrf、source |
| [冗長化（VRRP・HSRP）](#fhrp) | fhrp | device、if_name、group |
| [ACL](#acls) | acls | device、name |
| [ACL 行](#acl_entries) | acl_entries | device、acl、seq |
| [プレフィックスリスト](#prefix_lists) | prefix_lists | device、name、seq |
| [NAT](#nat_rules) | nat_rules | device、name |
| [回線監視](#tracks) | tracks | device、id |
| [PPPoE](#pppoe) | pppoe | device、name |
| [IPsec](#ipsec_tunnels) | ipsec_tunnels | device、name |
| [IPsec 対象通信](#ipsec_selectors) | ipsec_selectors | device、tunnel、seq |
| [トンネル](#tunnels) | tunnels | device、if_name |
| [DHCP](#dhcp_pools) | dhcp_pools | device、name |
| [接続](#connections) | connections | name |
| [QoS の見分け方](#qos_classes) | qos_classes | device、name |
| [QoS の見分け方の条件](#qos_class_matches) | qos_class_matches | device、class、seq |
| [QoS のポリシー](#qos_policies) | qos_policies | device、policy、class |
| [QoS の適用](#qos_bindings) | qos_bindings | device、if_name、direction |
| [経路制御の方針](#route_policies) | route_policies | device、name、seq |
| [AS パスのリスト](#as_path_lists) | as_path_lists | device、name、seq |
| [コミュニティのリスト](#community_lists) | community_lists | device、name、seq |
| [IPsec 追加の提案](#ipsec_proposals) | ipsec_proposals | device、tunnel、phase、seq |
| [IPsec の全体設定](#ipsec_settings) | ipsec_settings | device |
| [L2 延伸（L2TPv3）](#l2tpv3) | l2tpv3 | device、if_name |
| [L2TP](#l2tp_groups) | l2tp_groups | device、name |
| [L2TP 接続ユーザ](#l2tp_users) | l2tp_users | device、group、name |
| [VLAN](#vlans) | vlans | device、vlan_id |
| [スイッチポート](#switchports) | switchports | device、if_name |
| [LAG のメンバー](#lags) | lags | device、if_name |
| [L2 の全体設定](#l2_settings) | l2_settings | device |
| [VLAN ごとのスパニングツリー](#stp_vlans) | stp_vlans | device、vlans |
| [スタックのメンバー](#stack_members) | stack_members | device、member |
| [IPv6 インタフェース](#ipv6_interfaces) | ipv6_interfaces | device、if_name |
| [RA のプレフィックス](#ipv6_ra_prefixes) | ipv6_ra_prefixes | device、if_name、prefix |
| [IPv6 スタティック経路](#ipv6_routes) | ipv6_routes | device、vrf、prefix、next_hop、out_if |
| [IPv6 フィルタ](#ipv6_acl_entries) | ipv6_acl_entries | device、acl、seq |
| [IPv6 プレフィックスリスト](#ipv6_prefix_lists) | ipv6_prefix_lists | device、name、seq |
| [OSPFv3](#ospfv3_processes) | ospfv3_processes | device、process_id |
| [OSPFv3 インタフェース](#ospfv3_interfaces) | ospfv3_interfaces | device、process_id、if_name |
| [OSPFv3 エリア](#ospfv3_areas) | ospfv3_areas | device、process_id、area |
| [BGP IPv6 ネイバー](#bgp_ipv6_neighbors) | bgp_ipv6_neighbors | device、vrf、neighbor |
| [SNMP ユーザ（版 3）](#snmp_users) | snmp_users | device、name |
| [MSTP のインスタンス](#stp_instances) | stp_instances | device、instance |
| [ポートミラー](#mirror_sessions) | mirror_sessions | device、session |
| [時間帯](#time_ranges) | time_ranges | device、name、seq |
| [ログの対象ごとの重要度](#log_subsystems) | log_subsystems | device、subsystem |
| [ブリッジグループ](#bridge_groups) | bridge_groups | device、group |
| [BGP 集約経路](#bgp_aggregates) | bgp_aggregates | device、vrf、prefix |
| [OSPF バーチャルリンク](#ospf_virtual_links) | ospf_virtual_links | device、process_id、area、router_id |
| [OSPFv3 再配布の指定](#ospfv3_redistribute) | ospfv3_redistribute | device、process_id、source |
| [BGP IPv6 再配布の指定](#bgp_ipv6_redistribute) | bgp_ipv6_redistribute | device、source |
| [RIP](#rip_processes) | rip_processes | device、family |
| [RIP インタフェース](#rip_interfaces) | rip_interfaces | device、family、if_name |
| [RIP 再配布](#rip_redistribute) | rip_redistribute | device、family、source |
| [冗長化（IPv6 の VRRP）](#ipv6_fhrp) | ipv6_fhrp | device、if_name、group |

## 機器

<a id="devices"></a>内部名 `devices`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| ホスト名 | hostname | 文字 | ○ | ○ |  | 機器の名前。全シートの「機器」列の参照先 |
| メーカー | vendor | 文字 | ○ |  |  | メーカーパックの ID。例 cisco_ios、nec_ix コンフィグには書きません |
| 機種 | model | 文字 |  |  |  | パックの機種定義の ID。空でも生成できるが、新旧対照表（ポート）の案とポートの検査が使えない コンフィグには書きません |
| OS 版 | os_version | 文字 |  |  |  | コンフィグには書きません |
| グループ | group | 文字。複数可 |  |  |  | @グループ名 で行をまとめて当てるための名前。複数はカンマで区切る コンフィグには書きません |
| 拠点 | site | 文字 |  |  |  | コンフィグには書きません |
| ドメイン名 | domain_name | 文字 |  |  |  |  |
| DNS サーバ | name_servers | IPv4 か IPv6 のアドレス。複数可 |  |  |  | IPv4 か IPv6 のアドレス |
| 時刻帯 | timezone | 文字 |  |  |  | 例 JST +9。名前は省ける（+9）。分があれば +5:30 |
| 特権パスワード | enable_secret | 秘密の値 |  |  |  | 画面とレポートでは伏せます |
| パスワードの形 | secret_form | 選択（plain、hashed、encrypted） |  |  |  | 取り込んだ値がハッシュのとき hashed、機器が戻せる形で暗号化した値のとき encrypted。別メーカーへは引き継げない |
| SNMP 設置場所 | snmp_location | 文字 |  |  |  |  |
| SNMP 連絡先 | snmp_contact | 文字 |  |  |  |  |
| IPv6 の転送 | ipv6_routing | 有効・無効 |  |  |  | IPv6 のパケットを転送する（ルータとして動かす）。メーカーによっては書く行が無い |
| バナー | banner | 文章 |  |  |  | ログインの前に出す文 |
| 機器自身の通信のポリシールーティング | local_policy_route | 文字 |  |  |  | 機器が自分で出す通信に当てる、経路制御の方針の名前 |
| バナー（接続したとき） | banner_motd | 文章 |  |  |  | つないだ端末に、ログインの前に出す文（motd） |
| バナー（ログインの後） | banner_exec | 文章 |  |  |  | ログインの後に出す文 |
| パスワードを暗号化して保存 | password_encryption | 有効・無効 |  |  |  | コンフィグの中のパスワードを、機器が暗号化して保存する |
| ログの時刻の付け方 | log_timestamp | 選択（datetime、time、uptime） |  |  |  | datetime = 日付と時刻、time = 時刻だけ、uptime = 起動してからの時間 |
| ログの時刻に足すもの | log_timestamp_options | 選択（msec、localtime、show_timezone、year）。複数可 |  |  |  | msec = ミリ秒、localtime = 機器の時刻帯、show_timezone = 時刻帯の名前、year = 年 |
| デバッグの時刻の付け方 | debug_timestamp | 選択（datetime、uptime） |  |  |  | デバッグの表示の時刻をログと分けるメーカーで使う |
| デバッグの時刻に足すもの | debug_timestamp_options | 選択（msec、localtime、show_timezone、year）。複数可 |  |  |  |  |
| 機器内にログを溜める | log_buffer | 有効・無効 |  |  |  |  |
| 機器内ログの大きさ（バイト） | log_buffer_size | 数（1〜） |  |  |  |  |
| 機器内ログがいっぱいのとき | log_buffer_mode | 選択（cyclic、one_time） |  |  |  | cyclic = 古いものから上書き、one_time = 止める |
| ログに残す重要度 | log_severity | 選択（emergencies、alerts、critical、errors、warnings、notifications、informational、debugging） |  |  |  | 機器がログに残す、いちばん低い重要度 |
| コンソールの無操作切断（分） | console_timeout | 数（0〜） |  |  |  |  |
| コンソールの無操作切断（秒の端数） | console_timeout_sec | 数（0〜59） |  |  |  |  |
| NTP サーバとして動く | ntp_serve | 有効・無効 |  |  |  | ほかの機器からの時刻の問い合わせに答える |
| 転送のキャッシュ | forward_cache | 有効・無効 |  |  |  | 転送を速くするキャッシュを使う（行が要るメーカーだけが書く） |
| 転送のキャッシュ（IPv6） | forward_cache_ipv6 | 有効・無効 |  |  |  | IPv6 の転送を速くするキャッシュを使う（行が要るメーカーだけが書く） |
| DNS サーバの VRF | name_servers_vrf | 参照（vrfs.name） |  |  |  | 空 = VRF なし。DNS サーバは全部が同じ VRF |
| 備考 | note | 文字 |  |  |  | 生成には使わない コンフィグには書きません |
| 機器自身の通信のポリシールーティング（IPv6） | local_policy_route_v6 | 文字 |  |  |  | 機器が自分で出す IPv6 の通信に当てる経路制御の方針の名前 |
| VRRP の版（IPv4） | vrrp_version | 選択（2、3） |  |  |  | 空 = 2。3 = 機器全体を VRRPv3 にする（Cisco の fhrp version vrrp v3。IPv6 の VRRP に要る） |

## インタフェース

<a id="interfaces"></a>内部名 `interfaces`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| IF 名 | name | IF 名 | ○ | ○ |  | メーカーの正式な書き方。例 GigabitEthernet0/0/0 |
| 種類 | kind | 選択（physical、subif、loopback、tunnel、dialer、vlan、lag、bvi、other） |  |  |  | 空なら名前から決める コンフィグには書きません |
| 説明 | description | 文字 |  |  |  |  |
| 有効 | enabled | 有効・無効 | ○ |  | ○ | 無効 = shutdown |
| IPv4 アドレス | ipv4 | IPv4 アドレス/長さ。ほかに dhcp、negotiated、unnumbered:* |  |  |  | 例 192.0.2.1/30。dhcp、negotiated、unnumbered:<IF 名> も書ける |
| IPv4 セカンダリ | ipv4_secondary | IPv4 アドレス/長さ。複数可 |  |  |  |  |
| IPv6 アドレス | ipv6 | IPv6 アドレス/長さ。複数可 |  |  |  |  |
| VRF | vrf | 参照（vrfs.name） |  |  |  |  |
| VLAN 番号（サブ IF） | encap_vlan | 数（1〜4094） |  |  |  |  |
| タグなしの VLAN（サブ IF） | encap_native | 有効・無効 |  |  |  | このサブ IF の VLAN を、タグを付けずに送る（ネイティブ VLAN） |
| タグの種別（TPID） | encap_tpid | 文字 |  |  |  | 16 進数。空 = 8100 |
| VLAN 番号（VLAN の IF） | vlan_id | 数（1〜4094） |  |  |  | この IF が L3 の口になる VLAN の番号。取り込みが IF 名から入れる。別メーカーへ書き換えるときに、ポートの VLAN と IF を結ぶ コンフィグには書きません |
| ブリッジグループ | bridge_group | 数（1〜65535） |  |  |  | 同じ番号の IF どうしを L2 でつなぐ。BVI の行に書くと、そのグループの L3 の口になる |
| 対向機器 | peer_device | 文字 |  |  |  | このポートの先につながる機器のホスト名。両端の VLAN・LAG のチェック（X-381〜）に使う。生成には使わない コンフィグには書きません |
| 対向 IF | peer_if | 文字 |  |  |  | 対向機器の側の IF 名 コンフィグには書きません |
| 内側の VLAN（二重タグ） | encap_inner_vlans | 文字。複数可 |  |  |  | サブ IF が受ける二重タグの内側の VLAN。例 200、100-199,201-600。any = ほかのサブ IF に無いものすべて。「VLAN 番号（サブ IF）」が外側 |
| タグ付きフレームの最小の長さ | dot1q_min_frame | 数（64〜1522） |  |  |  | タグ付きで送るフレームの最小の長さ（オクテット） |
| サブ IF の数の上限 | max_subinterfaces | 数（1〜） |  |  |  | このポートに作れるサブ IF の数を増やす（対応する機種だけ。ポートの基本 IF の行に書く） |
| ブリッジの保護ポート | bridge_protected | 有効・無効 |  |  |  | 同じブリッジグループの保護ポートどうしは通信させない |
| このポートで覚える MAC の数 | bridge_table_size | 数（1〜） |  |  |  | ブリッジグループの中で、このポートが覚える MAC アドレスの数の上限 |
| CDP | cdp | 有効・無効 |  |  |  | この IF で CDP を送受信する。空 = 機器の既定 |
| LLDP を送る | lldp_transmit | 有効・無効 |  |  |  | 空 = 機器の既定 |
| LLDP を受ける | lldp_receive | 有効・無効 |  |  |  | 空 = 機器の既定 |
| ループ検出 | loop_detect | 有効・無効 |  |  |  | 検出用のフレームを送り、戻ってきたらループとみなす |
| ループを見つけたときの動作 | loop_detect_action | 選択（shutdown、none） |  |  |  | shutdown = ポートを止める。none = 知らせるだけ |
| ループ検出からの復帰（秒） | loop_detect_restore | 数（0〜86400） |  |  |  | 止めたポートを自動で戻すまでの秒。0 = 戻さない |
| タグなしフレームの既定の優先度 | qos_default_priority | 数（0〜7） |  |  |  | 内蔵スイッチのポートが、優先度の無いフレームに付ける優先度（0〜7） |
| 内蔵スイッチの送信の順序 | switch_qos_scheduler | 選択（strict、wfq） |  |  |  | strict = 優先度の高い順。wfq = 重み付き。内蔵スイッチの基本 IF の行に書く |
| ルータ側ポートの既定の優先度 | switch_router_priority | 数（0〜7） |  |  |  | 内蔵スイッチとルータの間のポートの、既定の優先度（0〜7）。内蔵スイッチの基本 IF の行に書く |
| MTU | mtu | 数 |  |  |  |  |
| IP MTU | ip_mtu | 数 |  |  |  |  |
| TCP MSS 調整 | tcp_mss | 数。ほかに auto |  |  |  | 数か auto（MTU から機器が決める） |
| 速度 | speed | 選択（auto、10、100、1000、2500、5000、10000） |  |  |  | 単位は Mbps |
| 二重 | duplex | 選択（auto、full、half） |  |  |  |  |
| 自動交渉で名乗る速度 | speed_advertise | 選択（10、100、1000、2500、5000）。複数可 |  |  |  | 速度が auto のとき、相手に名乗る速度を絞る（Mbps） |
| 速度の自動交渉を止める | speed_nonegotiate | 有効・無効 |  |  |  | 光のポートなどで、自動交渉をせずに決まった速度で動かす |
| MDI の向き | mdi | 選択（mdi、mdix） |  |  |  | ケーブルの結線の向きを決め打ちにする。空 = 機器の既定 |
| 帯域（kbps） | bandwidth | 数 |  |  |  | 経路計算用の値 |
| 入力 ACL | acl_in | 参照（acls.name） |  |  |  |  |
| 出力 ACL | acl_out | 参照（acls.name） |  |  |  |  |
| 入力 ACL（2 つ目以降） | acl_in_more | 参照（acls.name）。複数可 |  |  |  | 入力 ACL の後に、書いた順で見る ACL |
| 出力 ACL（2 つ目以降） | acl_out_more | 参照（acls.name）。複数可 |  |  |  | 出力 ACL の後に、書いた順で見る ACL |
| NAT の向き | nat_role | 選択（inside、outside） |  |  |  | メーカーによっては使わない |
| DHCP 中継先 | dhcp_relay | IPv4 アドレス。複数可 |  |  |  |  |
| 代理 ARP | proxy_arp | 有効・無効 |  |  |  |  |
| ポリシールーティング | policy_route | 文字 |  |  |  | この IF に入ってくる通信に当てる、経路制御の方針の名前 |
| ポートの説明 | port_description | 文字 |  |  |  | ポートと IF が別のメーカーの、ポートの側の説明。「ポート名.0」の行に書く |
| 備考 | note | 文字 |  |  |  | 生成には使わない コンフィグには書きません |

## スタティック経路

<a id="static_routes"></a>内部名 `static_routes`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| VRF | vrf | 参照（vrfs.name） |  | ○ |  | 空 = VRF なし（全体の経路表） |
| 宛先 | prefix | IPv4 ネットワーク | ○ | ○ |  | 例 192.0.2.0/24。デフォルト経路は 0.0.0.0/0 |
| ネクストホップ | next_hop | IPv4 アドレス。ほかに dhcp |  | ○ |  | dhcp と書くと、DHCP でもらったゲートウェイ |
| 出力 IF | out_if | IF 名。ほかに Null* |  | ○ |  | 捨てる経路の Null0 などは、インタフェースの表に行がなくてもよい |
| 優先度（distance） | distance | 数（1〜255） |  |  |  |  |
| 回線監視 | track | 参照（tracks.id） |  |  |  |  |
| 説明 | description | 文字 |  |  |  |  |
| タグ | tag | 数（0〜） |  |  |  | 再配布のときに、経路制御の方針の「条件: タグ」で見分ける印 |
| メトリック | metric | 数（1〜255） |  |  |  | NEC IX だけ |
| IF が落ちても残す | permanent | 有効・無効 |  |  |  | Cisco だけ |
| つながっている間だけ有効 | connected_only | 有効・無効 |  |  |  | NEC IX だけ（つなぎに行く IF の経路） |
| ネクストホップを全体の経路表で引く | next_hop_global | 有効・無効 |  |  |  | VRF の経路だけ。Cisco だけ |
| マルチキャストの確認だけに使う | multicast | 有効・無効 |  |  |  | Cisco だけ |

## ユーザ

<a id="users"></a>内部名 `users`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| ユーザ名 | name | 文字 | ○ | ○ |  |  |
| 権限 | privilege | 文字 |  |  |  | admin、operator か、メーカーの値（数値など） |
| パスワード | secret | 秘密の値 | ○ |  |  | 画面とレポートでは伏せます |
| パスワードの形 | secret_form | 選択（plain、hashed、encrypted） |  |  |  | 取り込んだ値がハッシュのとき hashed。機器が戻せる形で暗号化した値は encrypted |
| 使える接続方式 | services | 選択（terminal、ssh、telnet、ftp）。複数可 |  |  |  | ユーザごとに使える方式を決める機器だけ。terminal = コンソール。空 = 機器の既定 |
| 元に戻せる形で持つ | reversible | 有効・無効 |  |  |  | 機器がパスワードそのものを使う認証（PPP の CHAP の相手）のユーザは有効にする |

## 遠隔ログイン

<a id="remote_access"></a>内部名 `remote_access`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 許可する方式 | protocols | 選択（ssh、telnet）。複数可 | ○ |  |  |  |
| SSH の版 | ssh_version | 選択（1、2） |  |  |  |  |
| 接続元の制限 | acl | 参照（acls.name） |  |  |  |  |
| 接続元の制限（SSH だけ） | acl_ssh | 参照（acls.name） |  |  |  | 方式で制限が違うときだけ。同じなら「接続元の制限」に書く |
| 接続元の制限（Telnet だけ） | acl_telnet | 参照（acls.name） |  |  |  | 方式で制限が違うときだけ。同じなら「接続元の制限」に書く |
| 無操作切断（分） | idle_timeout | 数 |  |  |  |  |
| 認証 | auth | 選択（local、password、aaa） |  |  |  | aaa の中身は追加コンフィグに書く |
| 回線パスワード | line_password | 秘密の値 |  |  |  | 画面とレポートでは伏せます |
| 無操作切断（秒の端数） | idle_timeout_sec | 数（0〜59） |  |  |  |  |

## NTP サーバ

<a id="ntp_servers"></a>内部名 `ntp_servers`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| サーバ | server | 文字 | ○ | ○ |  | アドレスか名前 |
| 優先 | prefer | 有効・無効 |  |  |  |  |
| 送信元 IF | source_if | IF 名 |  |  |  |  |
| VRF | vrf | 参照（vrfs.name） |  |  |  |  |
| 認証の鍵番号 | auth_key_id | 数（1〜4294967295） |  |  |  | このサーバを確かめる鍵の番号。空 = 認証しない |
| 認証の鍵 | auth_key | 秘密の値 |  |  |  | その番号の鍵（MD5）。同じ番号の行は同じ鍵にする 画面とレポートでは伏せます |

## ログ転送先

<a id="syslog_servers"></a>内部名 `syslog_servers`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 転送先 | server | 文字 | ○ | ○ |  | アドレスか名前 |
| ファシリティ | facility | 文字 |  |  |  |  |
| 重要度 | severity | 選択（emergencies、alerts、critical、errors、warnings、notifications、informational、debugging） |  |  |  | この重要度以上を送る |
| 送信元 IF | source_if | IF 名 |  |  |  |  |
| VRF | vrf | 参照（vrfs.name） |  |  |  |  |

## SNMP コミュニティ

<a id="snmp_communities"></a>内部名 `snmp_communities`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| コミュニティ | community | 秘密の値 | ○ | ○ |  | 画面とレポートでは伏せます |
| 権限 | access | 選択（ro、rw） |  |  |  |  |
| 接続元の制限 | acl | 参照（acls.name） |  |  |  |  |

## SNMP 通知先

<a id="snmp_hosts"></a>内部名 `snmp_hosts`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 通知先 | host | IPv4 か IPv6 のアドレス | ○ | ○ |  | IPv4 か IPv6 のアドレス |
| 版 | version | 選択（1、2c、3） |  |  |  |  |
| コミュニティ | community | 秘密の値 |  |  |  | 画面とレポートでは伏せます |
| 送信元 IF | source_if | IF 名 |  |  |  |  |
| セキュリティの段（版 3） | security_level | 選択（noauth、auth、priv） |  |  |  | 版 3 の通知で使う段。noauth = 認証なし、auth = 認証あり、priv = 認証と暗号化。版 3 の行の「コミュニティ」には SNMP ユーザの名前を書く |

## VRF

<a id="vrfs"></a>内部名 `vrfs`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  |  |
| RD | rd | 文字 |  |  |  |  |
| 説明 | description | 文字 |  |  |  |  |
| 取り込むルートターゲット | rt_import | 文字。複数可 |  |  |  | 64496:100 の形。Cisco だけ |
| 付けるルートターゲット | rt_export | 文字。複数可 |  |  |  | 64496:100 の形。Cisco だけ |

## OSPF

<a id="ospf_processes"></a>内部名 `ospf_processes`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| プロセス番号 | process_id | 数 | ○ | ○ |  | 番号を持たないメーカーでは 1 |
| VRF | vrf | 参照（vrfs.name） |  |  |  |  |
| ルータ ID | router_id | IPv4 アドレス |  |  | ○ |  |
| 基準帯域（Mbps） | reference_bandwidth | 数 |  |  |  |  |
| 既定を受動にする | passive_default | 有効・無効 |  |  |  |  |
| デフォルト経路の広告 | default_originate | 選択（none、when_present、always） |  |  |  |  |
| 広告のメトリック | default_metric | 数 |  |  |  |  |
| 隣接の変化の記録 | log_adjacency | 選択（detail、off） |  |  |  | 空 = 機器の既定。detail = 状態が変わるたびに記録、off = 記録しない（Cisco） |
| 広告のメトリックの種類 | default_metric_type | 選択（1、2） |  |  |  | デフォルト経路の広告に付ける外部経路の種類。空 = 機器の既定（2） |
| デフォルト経路の広告の条件 | default_filter | 文字 |  |  |  | 条件にする経路制御の方針の名前 |
| 通過させない（最大メトリックで広告） | max_metric | 有効・無効 |  |  |  | 有効 = 自分を通る経路を、ほかのルータに選ばせない（保守のとき）。Cisco だけ |
| 優先度（distance） | distance | 数（1〜255） |  |  |  | OSPF の経路ぜんぶの優先度。種類ごとの 3 列とどちらか |
| 優先度（エリア内の経路） | distance_intra | 数（1〜255） |  |  |  |  |
| 優先度（エリア間の経路） | distance_inter | 数（1〜255） |  |  |  |  |
| 優先度（外部経路） | distance_external | 数（1〜255） |  |  |  |  |
| 経路表に入れる経路の絞り込み（プレフィックスリスト） | filter_in_prefix_list | 参照（prefix_lists.name） |  |  |  | OSPF で学んだ経路のうち、経路表に入れるものを絞る |
| 経路表に入れる経路の絞り込み（ACL） | filter_in_acl | 参照（acls.name） |  |  |  | Cisco だけ |
| 経路表に入れる経路の絞り込み（経路制御の方針） | filter_in_policy | 文字 |  |  |  |  |

## OSPF インタフェース

<a id="ospf_interfaces"></a>内部名 `ospf_interfaces`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| プロセス番号 | process_id | 数 | ○ | ○ |  |  |
| IF 名 | if_name | IF 名 | ○ | ○ |  |  |
| エリア | area | 文字 | ○ |  | ○ | 0 と 0.0.0.0 は同じ値として扱う |
| コスト | cost | 数 |  |  |  |  |
| 優先度 | priority | 数（0〜255） |  |  |  |  |
| Hello 間隔（秒） | hello | 数 |  |  | ○ |  |
| Dead 間隔（秒） | dead | 数 |  |  | ○ |  |
| 網の種類 | network_type | 選択（broadcast、point_to_point、nbma、point_to_multipoint） |  |  | ○ |  |
| 受動 | passive | 有効・無効 |  |  |  |  |
| 認証 | auth_type | 選択（none、simple、md5、hmac_md5） |  |  | ○ | hmac_md5 は、md5 と別の語として持つ機器だけ（NEC QX の hmac-md5） |
| 鍵番号 | auth_key_id | 数 |  |  |  |  |
| 鍵 | auth_key | 秘密の値 |  |  |  | 画面とレポートでは伏せます |

## OSPF エリア

<a id="ospf_areas"></a>内部名 `ospf_areas`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| プロセス番号 | process_id | 数 | ○ | ○ |  |  |
| エリア | area | 文字 | ○ | ○ |  |  |
| 種類 | type | 選択（normal、stub、totally_stub、nssa、totally_nssa） |  |  | ○ |  |
| 既定経路のコスト | default_cost | 数 |  |  |  |  |

## OSPF 集約

<a id="ospf_ranges"></a>内部名 `ospf_ranges`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| プロセス番号 | process_id | 数 | ○ | ○ |  |  |
| エリア | area | 文字 | ○ | ○ |  |  |
| 集約する範囲 | prefix | IPv4 ネットワーク | ○ | ○ |  |  |
| 広告する | advertise | 有効・無効 |  |  |  |  |

## OSPF 再配布

<a id="ospf_redistribute"></a>内部名 `ospf_redistribute`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| プロセス番号 | process_id | 数 | ○ | ○ |  |  |
| 再配布の元 | source | 選択（connected、static、bgp、rip） | ○ | ○ |  |  |
| メトリック | metric | 数 |  |  |  |  |
| メトリックの種類 | metric_type | 選択（1、2） |  |  |  |  |
| 経路制御の条件 | filter | 文字 |  |  |  | 経路制御の方針の表の名前。方針を追加コンフィグに書くときは、その名前 |
| タグ | tag | 数（0〜） |  |  |  | 再配布した経路に付ける印 |

## BGP

<a id="bgp_processes"></a>内部名 `bgp_processes`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| VRF | vrf | 参照（vrfs.name） |  | ○ |  | 空 = VRF なし（全体の経路表） |
| AS 番号 | asn | 数 | ○ |  |  |  |
| ルータ ID | router_id | IPv4 アドレス |  |  | ○ |  |
| Keepalive（秒） | keepalive | 数 |  |  |  |  |
| Hold（秒） | holdtime | 数 |  |  |  |  |
| ネイバーの変化を記録 | log_neighbor_changes | 有効・無効 |  |  |  |  |
| IPv6 の広告経路 | ipv6_networks | IPv6 ネットワーク。複数可 |  |  |  |  |
| IPv6 の再配布 | ipv6_redistribute | 選択（connected、static、ospf）。複数可 |  |  |  |  |
| 優先度（eBGP の経路） | distance_ebgp | 数（1〜255） |  |  |  | 空 = 機器の既定（20）。Cisco は 3 つを組で書く |
| 優先度（iBGP の経路） | distance_ibgp | 数（1〜255） |  |  |  | 空 = 機器の既定（200） |
| 優先度（自分で作った経路） | distance_local | 数（1〜255） |  |  |  | 空 = 機器の既定（200）。Cisco だけ |
| ローカルプリファレンスの既定値 | default_local_preference | 数（0〜） |  |  |  | 空 = 機器の既定（100） |

## BGP ネイバー

<a id="bgp_neighbors"></a>内部名 `bgp_neighbors`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| VRF | vrf | 参照（vrfs.name） |  | ○ |  | 空 = VRF なし（全体の経路表） |
| ネイバーアドレス | neighbor | IPv4 アドレス | ○ | ○ |  |  |
| 相手 AS 番号 | remote_as | 数 | ○ |  |  |  |
| 説明 | description | 文字 |  |  |  |  |
| 有効 | enabled | 有効・無効 |  |  |  |  |
| 送信元 IF | update_source | IF 名 |  |  |  |  |
| eBGP マルチホップ | ebgp_multihop | 数 |  |  |  |  |
| パスワード | password | 秘密の値 |  |  |  | 画面とレポートでは伏せます |
| Keepalive（秒） | keepalive | 数 |  |  |  |  |
| Hold（秒） | holdtime | 数 |  |  |  |  |
| ネクストホップを自分にする | next_hop_self | 有効・無効 |  |  |  |  |
| デフォルト経路の広告 | default_originate | 有効・無効 |  |  |  |  |
| 受信経路の保持 | soft_reconfig_in | 有効・無効 |  |  |  |  |
| 受信フィルタ | prefix_list_in | 参照（prefix_lists.name） |  |  |  |  |
| 送信フィルタ | prefix_list_out | 参照（prefix_lists.name） |  |  |  |  |
| 受信の経路制御 | filter_in | 文字 |  |  |  | 経路制御の方針の表の名前。方針を追加コンフィグに書くときは、その名前 |
| 送信の経路制御 | filter_out | 文字 |  |  |  | 経路制御の方針の表の名前。方針を追加コンフィグに書くときは、その名前 |
| コミュニティを送る | send_community | 有効・無効 |  |  |  | Cisco は、これが無いと相手にコミュニティを送らない。NEC IX には当たる行が無い |
| デフォルト経路の広告の条件 | default_originate_filter | 文字 |  |  |  | デフォルト経路を送る条件にする経路制御の方針の名前。Cisco だけ |
| 受信の AS パスのリスト | as_path_filter_in | 参照（as_path_lists.name） |  |  |  | 受け取る経路を AS パスのリストで絞る。Cisco だけ |
| 送信の AS パスのリスト | as_path_filter_out | 参照（as_path_lists.name） |  |  |  | 送る経路を AS パスのリストで絞る。Cisco だけ |
| 受信フィルタ（ACL） | acl_in | 参照（acls.name） |  |  |  | 受け取る経路の宛先を ACL で絞る。Cisco だけ（NEC IX はプレフィックスリストで書く） |
| 送信フィルタ（ACL） | acl_out | 参照（acls.name） |  |  |  | 送る経路の宛先を ACL で絞る。Cisco だけ |
| 受け取る経路数の上限 | max_prefix | 数（1〜） |  |  |  | 超えるとセッションを切る（動作の列で変えられる）。Cisco だけ |
| 上限の警告を出す割合（%） | max_prefix_threshold | 数（1〜100） |  |  |  | 空 = 機器の既定（Cisco は 75） |
| 上限を超えたときの動作 | max_prefix_action | 選択（warning_only、discard_extra） |  |  |  | 空 = セッションを切る。warning_only = 記録するだけ。discard_extra = 超えた分を捨てる |
| 上限で切れた後に戻すまで（分） | max_prefix_restart | 数（1〜65535） |  |  |  |  |
| weight | weight | 数（0〜65535） |  |  |  | このネイバーから受け取った経路に付ける weight。Cisco だけ |
| 拡張コミュニティを送る | send_ext_community | 有効・無効 |  |  |  | Cisco の send-community extended（both は「コミュニティを送る」も有効） |
| ルートリフレクタのクライアント | route_reflector_client | 有効・無効 |  |  |  |  |
| 経路の更新を送る最小間隔（秒） | advertisement_interval | 数（0〜600） |  |  |  |  |

## BGP 広告経路

<a id="bgp_networks"></a>内部名 `bgp_networks`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| VRF | vrf | 参照（vrfs.name） |  | ○ |  | 空 = VRF なし（全体の経路表） |
| 広告する経路 | prefix | IPv4 ネットワーク | ○ | ○ |  |  |

## BGP 再配布

<a id="bgp_redistribute"></a>内部名 `bgp_redistribute`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| VRF | vrf | 参照（vrfs.name） |  | ○ |  | 空 = VRF なし（全体の経路表） |
| 再配布の元 | source | 選択（connected、static、ospf、rip） | ○ | ○ |  |  |
| メトリック | metric | 数 |  |  |  |  |
| 経路制御の条件 | filter | 文字 |  |  |  | 経路制御の方針の表の名前。方針を追加コンフィグに書くときは、その名前 |
| OSPF のプロセス番号 | process_id | 数 |  |  |  | 再配布の元が ospf のとき。空なら、同じ VRF にちょうど 1 つあるプロセス |

## 冗長化（VRRP・HSRP）

<a id="fhrp"></a>内部名 `fhrp`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| IF 名 | if_name | IF 名 | ○ | ○ |  |  |
| 方式 | protocol | 選択（vrrp、hsrp） | ○ |  |  | hsrp は Cisco のみ |
| グループ番号 | group | 数 | ○ | ○ |  |  |
| 版 | version | 選択（1、2、3） |  |  | ○ |  |
| 仮想 IP | virtual_ip | IPv4 アドレス | ○ |  |  |  |
| 優先度 | priority | 数（1〜254） |  |  | ○ |  |
| 切り戻し | preempt | 有効・無効 |  |  | ○ |  |
| 切り戻し待ち（秒） | preempt_delay | 数 |  |  |  |  |
| 広告・Hello 間隔（秒） | advert_interval | 数 |  |  | ○ |  |
| Hold（秒） | hold | 数 |  |  |  | hsrp のみ |
| 認証 | auth_type | 選択（none、simple、md5） |  |  |  |  |
| 鍵 | auth_key | 秘密の値 |  |  |  | 画面とレポートでは伏せます |
| 回線監視 | track | 参照（tracks.id） |  |  |  |  |
| 監視断のときの減点 | track_decrement | 数 |  |  |  |  |
| 版 2 の相手とも話す | vrrpv2 | 有効・無効 |  |  |  | VRRP の版（IPv4）が 3 の機器の IPv4 のグループを、版 2 だけの機器ともつなぐ（Cisco の vrrpv2） |

## ACL

<a id="acls"></a>内部名 `acls`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  | 番号の ACL は番号を名前として書く |
| 種類 | type | 選択（standard、extended） |  |  |  |  |
| 説明 | description | 文字 |  |  |  |  |
| 行の番号の刻み | seq_step | 数（1〜1000） |  |  |  | NEC IX の sequence-mode。値があれば、行に番号を付けて持つ。空なら、番号は 10、20、30 … で書く |

## ACL 行

<a id="acl_entries"></a>内部名 `acl_entries`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| ACL 名 | acl | 参照（acls.name） | ○ | ○ |  |  |
| 順番 | seq | 数 | ○ | ○ |  | 番号を持たない書き方では 10 刻みで振る |
| 動作 | action | 選択（permit、deny、remark） | ○ |  |  |  |
| プロトコル | protocol | 文字 |  |  |  | ip tcp udp icmp esp ah gre ospf か番号 |
| 送信元 | src | 文字 |  |  |  | any、192.0.2.1（ホスト）、192.0.2.0/24（ネットワーク） |
| 送信元ポート | src_port | 文字 |  |  |  | 80、1000-2000、>1023、<1024、!80 |
| 宛先 | dst | 文字 |  |  |  | 送信元と同じ書き方 |
| 宛先ポート | dst_port | 文字 |  |  |  | 送信元ポートと同じ書き方 |
| 付加条件 | flags | 選択（established、log、fragments、log_input、ack、fin、psh、rst、syn、urg）。複数可 |  |  |  | log_input = 入ってきた IF も記録する。ack fin psh rst syn urg = その TCP のフラグが立った通信だけに合う |
| ICMP の種類 | icmp_type | 文字 |  |  |  | 番号か、メーカーの語（echo、echo-reply、unreachable など） |
| 注記 | remark | 文字 |  |  |  | 動作が remark の行の本文 |
| DSCP | dscp | 数（0〜63） |  |  |  | この DSCP の通信だけに合う。数で書く（ef = 46） |
| precedence | precedence | 数（0〜7） |  |  |  | この precedence の通信だけに合う |
| ICMP のコード | icmp_code | 数（0〜255） |  |  |  | ICMP の種類を番号で書いた行だけ |
| ToS | tos | 数（0〜15） |  |  |  | この ToS の通信だけに合う。数で書く |
| 効かせる時間帯 | time_range | 参照（time_ranges.name） |  |  |  | この時間帯の間だけ効く行（シート「時間帯」の名前） |
| 行の注記 | comment | 文字 |  |  |  | この行に付ける注記（NEC QX の rule <番号> comment）。注記だけの行を持つメーカーでは、動作 remark の行を使う |

## プレフィックスリスト

<a id="prefix_lists"></a>内部名 `prefix_lists`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  |  |
| 順番 | seq | 数 | ○ | ○ |  |  |
| 動作 | action | 選択（permit、deny） | ○ |  |  |  |
| 範囲 | prefix | IPv4 ネットワーク | ○ |  |  |  |
| 長さの下限（ge） | ge | 数（0〜32） |  |  |  |  |
| 長さの上限（le） | le | 数（0〜32） |  |  |  |  |

## NAT

<a id="nat_rules"></a>内部名 `nat_rules`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  | 取り込み時は nat1 から自動で振る。比べるときは、名前でなく中身で行を合わせる |
| 種類 | type | 選択（pat_interface、pat_pool、dynamic_pool、static、static_port、static_network） | ○ |  |  | pat_interface = 外側 IF のアドレスへまとめて変換。static_network = ネットワークごとの 1 対 1（内側・外側アドレスにネットワークのアドレス、範囲のプレフィックス長に長さ） |
| 外側 IF | outside_if | IF 名 |  |  |  |  |
| 対象の通信 | match_acl | 参照（acls.name） |  |  |  | 空 = すべて |
| 変換後アドレスの先頭 | pool_start | IPv4 アドレス |  |  |  |  |
| 変換後アドレスの末尾 | pool_end | IPv4 アドレス |  |  |  |  |
| 範囲のプレフィックス長 | pool_prefix | 数（0〜32） |  |  |  |  |
| 内側アドレス | local_ip | IPv4 アドレス |  |  |  |  |
| 外側アドレス | global_ip | IPv4 アドレス。ほかに interface |  |  |  | interface と書くと外側 IF のアドレス |
| プロトコル | protocol | 選択（tcp、udp） |  |  |  |  |
| 内側ポート | local_port | 数（1〜65535） |  |  |  |  |
| 外側ポート | global_port | 数（1〜65535） |  |  |  |  |
| 複数の外側アドレスに対応させる | extendable | 有効・無効 |  |  |  | 同じ内側アドレスを、複数の外側アドレスに対応させる（Cisco の extendable） |

## 回線監視

<a id="tracks"></a>内部名 `tracks`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 番号・名前 | id | 文字 | ○ | ○ |  |  |
| 監視の方法 | type | 選択（icmp、interface、route、list） | ○ |  |  | icmp = 相手に届くか、interface = IF が上がっているか、route = 経路表に宛先があるか、list = ほかの監視をまとめる |
| 監視先 | target | 文字 |  |  |  | icmp はアドレス、interface は IF 名、route はネットワーク（アドレスとネットワークは IPv4 か IPv6） |
| 送信元 IF | source_if | IF 名 |  |  |  |  |
| VRF | vrf | 参照（vrfs.name） |  |  |  |  |
| 間隔（秒） | interval | 数 |  |  | ○ |  |
| 待ち時間（ミリ秒） | timeout | 数 |  |  |  |  |
| 断と判断する回数 | fail_count | 数 |  |  | ○ |  |
| 復旧と判断する回数 | restore_count | 数 |  |  |  |  |
| 断の反映待ち（秒） | down_delay | 数 |  |  |  |  |
| 復旧の反映待ち（秒） | up_delay | 数 |  |  |  |  |
| 送信元アドレス | source_ip | IPv4 か IPv6 のアドレス |  |  |  | 送信元 IF とどちらか 1 つ。監視先と同じ種類（IPv4 か IPv6） |
| 監視の出口の IF | via_if | IF 名 |  |  |  | 経路表によらず、この IF から確かめる（NEC IX） |
| 監視の出口のネクストホップ | via_next_hop | IPv4 か IPv6 のアドレス |  |  |  | NEC IX。監視先と同じ種類（IPv4 か IPv6） |
| しきい値（ミリ秒） | threshold | 数 |  |  |  | 応答がこれより遅いと「遅い」とみなす（Cisco） |
| 正常とみなす条件 | up_when | 選択（reachable、within_threshold） |  |  |  | 空・reachable = 届けば正常。within_threshold = しきい値より遅い応答も断とみなす |
| 断の間の監視間隔（秒） | restore_interval | 数 |  |  |  | NEC IX |
| まとめる監視 | members | 参照（tracks.id）。複数可 |  |  |  | 監視の方法が list の行だけ。まとめる行の「番号・名前」 |
| まとめ方 | list_rule | 選択（all_up、any_up） |  |  |  | all_up = 全部が正常のときだけ正常。any_up = 1 つでも正常なら正常 |
| 断のとき止める IF | shutdown_ifs | IF 名。複数可 |  |  |  | NEC IX |

## PPPoE

<a id="pppoe"></a>内部名 `pppoe`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  | メーカーに名前が無ければ、取り込み時に論理 IF の名前を入れる。比べるときは、名前でなく中身で行を合わせる |
| 物理 IF | physical_if | IF 名 | ○ |  |  | 回線につながる IF |
| 論理 IF | logical_if | IF 名 | ○ |  |  | アドレスや NAT を設定する IF。インタフェースのシートに行を持つ |
| 接続 ID | username | 文字 | ○ |  |  |  |
| 接続パスワード | password | 秘密の値 | ○ |  |  | 画面とレポートでは伏せます |
| 認証方式 | auth | 選択（chap、pap、chap_pap） |  |  |  |  |
| サービス名 | service_name | 文字 |  |  |  |  |
| デフォルト経路を作る | default_route | 有効・無効 |  |  |  |  |
| 監視間隔（秒） | keepalive | 数 |  |  |  |  |
| 相手も認証する | auth_both | 有効・無効 |  |  |  | こちらからも相手に認証を求める |
| 通信があるときだけつなぐ | on_demand | 有効・無効 |  |  |  |  |
| 無通信で切る時間（秒） | idle_timeout | 数（0〜） |  |  |  |  |
| つなぐきっかけの通信 | dial_trigger | 文字 |  |  |  | ip（IP の通信すべて）か、ACL の名前 |
| つなぎ直すまでの待ち（秒） | restart_delay | 数（1〜3600） |  |  |  |  |
| MAC アドレスを論理 IF のものにする | mac_override | 有効・無効 |  |  |  |  |

## IPsec

<a id="ipsec_tunnels"></a>内部名 `ipsec_tunnels`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  | 取り込み時は、トンネル IF の名前か、マップの名前（Cisco は「マップ名/番号」）。比べるときは、名前でなく中身で行を合わせる |
| 方式 | style | 選択（tunnel_if、policy） | ○ |  |  | tunnel_if = トンネル IF に経路を向ける。policy = 対象通信で選ぶ |
| トンネル IF | tunnel_if | IF 名 |  |  |  | 方式が tunnel_if のとき必須 |
| 自分側 IF | local_if | IF 名 |  |  |  | 方式が policy のとき: IPsec を当てる IF。L2TP の受け口を守る行は、NEC IX では空（受け口のトンネル IF に当てる） |
| 自分側アドレス | local_address | IPv4 アドレス |  |  |  |  |
| 相手アドレス | peer_address | 文字 | ○ |  |  | アドレス。相手が動的なら any。名前（FQDN）で書いてもよい |
| IKE の版 | ike_version | 選択（1、2） | ○ |  | ○ |  |
| IKE のモード | ike_mode | 選択（main、aggressive） |  |  | ○ | 版 1 のみ |
| 自分の ID | local_id | 文字 |  |  |  | 種類:値 の形。種類は address・fqdn・email・keyid（例 fqdn:rt-a01.example.com）。空 = 自分のアドレス |
| 相手の ID | remote_id | 文字 |  |  |  | 種類:値 の形（自分の ID と同じ）。空 = 相手のアドレス |
| 事前共有鍵 | psk | 秘密の値 | ○ |  |  | 画面とレポートでは伏せます |
| 相手を確かめる鍵 | psk_remote | 秘密の値 |  |  |  | 版 2 で、自分が名乗る鍵（事前共有鍵）と相手を確かめる鍵が別のとき。空 = 事前共有鍵と同じ 画面とレポートでは伏せます |
| 鍵の書き方 | psk_form | 選択（text、hex） |  |  |  | hex = 16 進で書いた鍵。空 = text（文字） |
| IKE 暗号 | ike_encryption | 選択（des、3des、aes128、aes192、aes256、aes128gcm、aes256gcm、null） |  |  | ○ | 版 1 では必須。版 2 で空なら、機器が持つ候補に任せる |
| IKE ハッシュ | ike_hash | 選択（md5、sha1、sha256、sha384、sha512） |  |  | ○ | 版 1 では必須。版 2 で空なら、機器が持つ候補に任せる |
| IKE DH グループ | ike_dh | 選択（dh1、dh2、dh5、dh14、dh15、dh16、dh19、dh20、dh21、dh24） |  |  | ○ | 版 1 では必須。版 2 で空なら、機器が持つ候補に任せる |
| IKE PRF | ike_prf | 選択（md5、sha1、sha256、sha384、sha512） |  |  |  | 版 2 のみ。空 = IKE ハッシュと同じ |
| IKE 有効時間（秒） | ike_lifetime | 数 |  |  | ○ |  |
| ESP 暗号 | esp_encryption | 選択（des、3des、aes128、aes192、aes256、aes128gcm、aes256gcm、null） |  |  | ○ | 版 1 では必須。版 2 で空なら、機器が持つ候補に任せる |
| ESP ハッシュ | esp_hash | 選択（md5、sha1、sha256、sha384、sha512、none） |  |  | ○ | none = ESP の認証を使わない。版 1 では必須。版 2 で空なら、機器が持つ候補に任せる |
| AH ハッシュ | ah_hash | 選択（md5、sha1、sha256、sha384、sha512） |  |  |  | AH（認証ヘッダ）も使うとき |
| IPsec のモード | ipsec_mode | 選択（tunnel、transport） |  |  |  | 空 = tunnel |
| PFS | pfs | 選択（none、dh1、dh2、dh5、dh14、dh15、dh16、dh19、dh20、dh21、dh24） |  |  | ○ |  |
| IPsec 有効時間（秒） | ipsec_lifetime | 数 |  |  | ○ |  |
| IPsec 有効量（キロバイト） | ipsec_lifetime_kb | 数 |  |  |  | この量を送ったら鍵を替える |
| いつもつないでおく | always_on | 有効・無効 |  |  |  | 通信が無くても、切れたらつなぎ直す。空 = 通信があったときにつなぐ |
| 相手監視の間隔（秒） | dpd_interval | 数 |  |  |  |  |
| 相手監視の再送回数 | dpd_retry | 数 |  |  |  |  |
| 相手監視の再送間隔（秒） | dpd_retry_interval | 数 |  |  |  | 返事が無いときに、聞き直す間隔 |
| 相手監視の送り方 | dpd_mode | 選択（on_demand、periodic） |  |  |  | on_demand = 通信が無いときだけ聞く。periodic = いつも間隔ごとに聞く |
| NAT 越え | nat_traversal | 有効・無効 |  |  |  |  |
| NAT 越えの維持間隔（秒） | nat_keepalive | 数 |  |  |  |  |
| NAT 越えを必ず使う | nat_force | 有効・無効 |  |  |  | 途中に NAT が無くても UDP で包む |
| この IKE の相手だけに使う | ike_bind | 有効・無効 |  |  |  | 相手が any のとき、この行の IKE で確かめた相手だけを受け付ける（NEC IX の ike-binding） |
| IKE 暗号（追加） | ike_encryption_more | 選択（3des、aes128、aes192、aes256、aes128gcm、aes256gcm）。複数可 |  |  |  | 版 2 のみ。同じ提案に並べる 2 つ目以降の値（機器は、並べた値のどの組み合わせでも折り合う） |
| IKE ハッシュ（追加） | ike_hash_more | 選択（md5、sha1、sha256、sha384、sha512）。複数可 |  |  |  | 版 2 のみ。同じ提案に並べる 2 つ目以降の値 |
| IKE DH グループ（追加） | ike_dh_more | 選択（dh1、dh2、dh5、dh14、dh15、dh16、dh19、dh20、dh21、dh24）。複数可 |  |  |  | 版 2 のみ。同じ提案に並べる 2 つ目以降の値 |
| IKE PRF（追加） | ike_prf_more | 選択（md5、sha1、sha256、sha384、sha512）。複数可 |  |  |  | 版 2 のみ。同じ提案に並べる 2 つ目以降の値 |
| ESP 暗号（追加） | esp_encryption_more | 選択（3des、aes128、aes192、aes256、aes128gcm、aes256gcm、null）。複数可 |  |  |  | 版 2 のみ。同じ提案に並べる 2 つ目以降の値（NEC IX の child-proposal。Cisco は、組み合わせごとのトランスフォームセットに置き換えて書く） |
| ESP ハッシュ（追加） | esp_hash_more | 選択（md5、sha1、sha256、sha384、sha512）。複数可 |  |  |  | 版 2 のみ。同じ提案に並べる 2 つ目以降の値 |
| PFS（追加） | pfs_more | 選択（dh1、dh2、dh5、dh14、dh15、dh16、dh19、dh20、dh21、dh24）。複数可 |  |  |  | 版 2 のみ。PFS に並べる 2 つ目以降のグループ（NEC IX の child-pfs） |
| 予備の相手 | peer_backup | 文字。複数可 |  |  |  | 相手アドレスの相手につながらないときに、順につなぎに行く相手（アドレスか名前）。鍵は同じ（Cisco のクリプトマップの 2 つ目以降の set peer） |
| 外側の VRF | outer_vrf | 参照（vrfs.name） |  |  |  | 暗号化した通信と IKE が通る VRF（Cisco の fvrf）。空 = VRF なし。方式が policy の行では、自分側 IF の VRF と同じにする |
| 内側の VRF | inner_vrf | 参照（vrfs.name） |  |  |  | 守る通信が属する VRF（Cisco の ivrf）。方式が policy の行だけに書く。空 = 外側の VRF と同じ。方式が tunnel_if の行では、トンネル IF の VRF（インタフェースのシート）が内側 |
| 同じ NAT の後ろの端末を見分ける | nat_demux | 有効・無効 |  |  |  | L2TP/IPsec の受け口で、同じ NAT の後ろから複数の端末がつなぐとき（Cisco の set nat demux） |
| つなぎ直しは最初の相手から | peer_default | 有効・無効 |  |  |  | 予備の相手があるとき、相手が応えなくなったら、次の相手の前に最初の相手（相手アドレス）をもう一度試す（Cisco の set peer … default） |

## IPsec 対象通信

<a id="ipsec_selectors"></a>内部名 `ipsec_selectors`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| IPsec の名前 | tunnel | 参照（ipsec_tunnels.name） | ○ | ○ |  |  |
| 順番 | seq | 数 | ○ | ○ |  |  |
| 自分側ネットワーク | local_net | IPv4 ネットワーク | ○ |  |  | すべてなら 0.0.0.0/0 |
| 相手側ネットワーク | remote_net | IPv4 ネットワーク | ○ |  |  | すべてなら 0.0.0.0/0 |
| プロトコル | protocol | 文字 |  |  |  | ip（すべて）・udp・tcp など。空 = ip |
| 自分側ポート | local_port | 数（1〜65535） |  |  |  | プロトコルが udp・tcp のとき。L2TP を守るなら udp の 1701 |
| 相手側ポート | remote_port | 数（1〜65535） |  |  |  | プロトコルが udp・tcp のとき |

## トンネル

<a id="tunnels"></a>内部名 `tunnels`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| トンネル IF | if_name | IF 名 | ○ | ○ |  | アドレスはインタフェースのシートに書く |
| 種類 | type | 選択（gre、ipip、ipsec、l2tpv3、ipv4_over_ipv6、ipv6_over_ipv4、map_e、auto_ds_lite、auto_ipip、etherip） | ○ |  |  | ipsec は IPsec のシートのトンネル IF と対になる。ipv4_over_ipv6 = IPv6 の網で IPv4 を運ぶ（DS-Lite・固定 IP の IPIP）、map_e = MAP-E、auto_ds_lite・auto_ipip = 回線の事業者から設定をもらう |
| 送信元 | source | 文字 |  |  |  | IF 名かアドレス。種類が ipsec 以外では必須 |
| 宛先 | destination | IPv4 アドレス |  |  |  | 種類が ipsec 以外では必須 |
| 鍵（GRE） | key | 数 |  |  |  |  |
| 監視間隔（秒） | keepalive_interval | 数 |  |  |  |  |
| 監視の再送回数 | keepalive_retry | 数 |  |  |  |  |
| 宛先（IPv6） | destination_v6 | IPv6 アドレス |  |  |  | IPv6 の網を通るトンネルの宛先 |
| 宛先（名前） | destination_fqdn | 文字 |  |  |  | 宛先を名前で書く（DS-Lite の AFTR など） |
| MAP-E の事業者 | provider | 選択（jpix、ocn、ocn_fixed） |  |  |  | 種類が map_e のとき。空 = 機器の既定 |
| MAP-E のホスト名 | provider_host | 文字 |  |  |  | 事業者が ocn_fixed のとき |
| LAN 側 IF（自動設定） | lan_if | IF 名 |  |  |  | 種類が auto_ds_lite・auto_ipip のとき、もらった IPv4 アドレスを使う LAN 側の IF |

## DHCP

<a id="dhcp_pools"></a>内部名 `dhcp_pools`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  |  |
| ネットワーク | network | IPv4 ネットワーク | ○ |  |  |  |
| 配る範囲の先頭 | range_start | IPv4 アドレス |  |  |  |  |
| 配る範囲の末尾 | range_end | IPv4 アドレス |  |  |  |  |
| 配らないアドレス | excluded | 文字。複数可 |  |  |  | 192.0.2.1-192.0.2.10 の形。複数はカンマで区切る |
| デフォルトゲートウェイ | default_router | IPv4 アドレス |  |  |  |  |
| デフォルトゲートウェイ（2 つ目以降） | default_routers_more | IPv4 アドレス。複数可 |  |  |  | Cisco だけ（8 つまで） |
| DNS サーバ | dns_servers | IPv4 アドレス。複数可 |  |  |  |  |
| ドメイン名 | domain | 文字 |  |  |  |  |
| 貸し出し時間 | lease | 文字 |  |  |  | 1d、12h、90m、100s の形か、infinite（無期限） |

## 接続

<a id="connections"></a>内部名 `connections`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 接続名 | name | 文字 | ○ | ○ |  |  |
| 種類 | type | 選択（ipsec、gre、gre_over_ipsec） | ○ |  |  |  |
| 機器 A | a_device | 参照（devices.hostname） | ○ |  |  |  |
| 機器 Z | z_device | 参照（devices.hostname）。ほかに (管理外) | ○ |  |  | 相手が管理外なら (管理外) |
| A の外向き IF | a_local_if | IF 名 |  |  |  |  |
| Z の外向き IF | z_local_if | IF 名 |  |  |  |  |
| A の外向きアドレス | a_address | IPv4 アドレス |  |  |  | 空なら IF のアドレス |
| Z の外向きアドレス | z_address | IPv4 アドレス |  |  |  | 空なら IF のアドレス |
| A のトンネル IF | a_tunnel_if | IF 名 |  |  |  |  |
| Z のトンネル IF | z_tunnel_if | IF 名 |  |  |  |  |
| A のトンネル IF アドレス | a_tunnel_ip | IPv4 アドレス/長さ |  |  |  |  |
| Z のトンネル IF アドレス | z_tunnel_ip | IPv4 アドレス/長さ |  |  |  |  |
| 事前共有鍵 | psk | 秘密の値 |  |  |  | 画面とレポートでは伏せます |
| IKE の版 | ike_version | 選択（1、2） |  |  |  |  |
| IKE のモード | ike_mode | 選択（main、aggressive） |  |  |  |  |
| IKE 暗号 | ike_encryption | 選択（des、3des、aes128、aes192、aes256、aes128gcm、aes256gcm、null） |  |  |  |  |
| IKE ハッシュ | ike_hash | 選択（md5、sha1、sha256、sha384、sha512） |  |  |  |  |
| IKE DH グループ | ike_dh | 選択（dh1、dh2、dh5、dh14、dh15、dh16、dh19、dh20、dh21、dh24） |  |  |  |  |
| IKE 有効時間（秒） | ike_lifetime | 数 |  |  |  |  |
| ESP 暗号 | esp_encryption | 選択（des、3des、aes128、aes192、aes256、aes128gcm、aes256gcm、null） |  |  |  |  |
| ESP ハッシュ | esp_hash | 選択（md5、sha1、sha256、sha384、sha512） |  |  |  |  |
| PFS | pfs | 選択（none、dh1、dh2、dh5、dh14、dh15、dh16、dh19、dh20、dh21、dh24） |  |  |  |  |
| IPsec 有効時間（秒） | ipsec_lifetime | 数 |  |  |  |  |

## QoS の見分け方

<a id="qos_classes"></a>内部名 `qos_classes`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| クラス名 | name | 文字 | ○ | ○ |  | 1〜31 字 |
| 合わせ方 | match | 選択（any、all） |  |  | ○ | any = どれかの条件に合えば、all = すべての条件に合えば |

## QoS の見分け方の条件

<a id="qos_class_matches"></a>内部名 `qos_class_matches`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| クラス名 | class | 参照（qos_classes.name） | ○ | ○ |  |  |
| 順番 | seq | 数 | ○ | ○ |  | 書いた順に 10 刻みで振る |
| ACL | acl | 参照（acls.name） |  |  |  | ACL・DSCP・precedence は 1 行に 1 つだけ。3 つとも空 = すべての通信 |
| DSCP | dscp | 数（0〜63）。複数可 |  |  |  | 数で書く（ef = 46、af41 = 34）。複数はカンマで区切る |
| precedence | precedence | 数（0〜7）。複数可 |  |  |  | 数で書く。複数はカンマで区切る |
| キュー | queue | 選択（high、medium、normal、low、sub_a、sub_b、sub_c、sub_d） |  |  |  | この条件に合った通信を入れる優先度のキュー |
| 受信 IF | input_if | IF 名 |  |  |  |  |
| CoS | cos | 数（0〜7）。複数可 |  |  |  | VLAN タグの優先度。複数はカンマで区切る |
| QoS グループ | qos_group | 数（0〜65535） |  |  |  | 受信側で付けた機器の中だけの印 |
| プロトコル | protocol | 文字 |  |  |  | 機器が見分けるプロトコルの名前 |
| 条件を逆にする | negate | 有効・無効 |  |  |  | この行の条件に合わない通信 |
| ほかのクラス | class_ref | 参照（qos_classes.name） |  |  |  | ほかのクラスに合う通信 |
| IPv6 の ACL | ipv6_acl | 文字 |  |  |  |  |
| 機器自身が出す通信 | local_generated | 有効・無効 |  |  |  |  |
| RTP のポートの先頭 | rtp_port_start | 数（1〜65535） |  |  |  |  |
| RTP のポートの幅 | rtp_port_range | 数（0〜65535） |  |  |  |  |
| NGN のトンネル IF | ngn_if | IF 名 |  |  |  |  |

## QoS のポリシー

<a id="qos_policies"></a>内部名 `qos_policies`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| ポリシー名 | policy | 文字 | ○ | ○ |  | 1〜31 字 |
| クラス名 | class | 参照（qos_classes.name）。ほかに default、local、dynamic | ○ | ○ |  | QoS の見分け方のクラス名か、default（どれにも当たらない通信）、local（機器自身が出す通信）、dynamic（ダイナミック QoS） |
| 順番 | seq | 数 | ○ |  |  | ポリシーの中で見る順（小さい順に書く。default はいつも最後） コンフィグには書きません |
| 優先して送る | priority | 有効・無効 |  |  |  |  |
| 優先の段 | priority_level | 数（1〜2） |  |  |  |  |
| 帯域（kbps） | bandwidth_kbps | 数（1〜） |  |  |  | 帯域の 3 列は 1 つだけ書く。優先の行では上限 |
| 帯域（%） | bandwidth_percent | 数（1〜100） |  |  |  |  |
| 帯域（残りの %） | bandwidth_remaining | 数（1〜100） |  |  |  |  |
| 優先のバースト（バイト） | priority_burst | 数（1〜） |  |  |  |  |
| 上限・遅らせる（bps） | shape_bps | 数（1〜） |  |  |  |  |
| 遅らせる側の通常バースト（ビット） | shape_bc | 数（1〜） |  |  |  |  |
| 遅らせる側の超過バースト（ビット） | shape_be | 数（0〜） |  |  |  |  |
| 上限・捨てる（bps） | police_bps | 数（1〜） |  |  |  |  |
| 捨てる側の通常バースト（バイト） | police_bc | 数（1〜） |  |  |  |  |
| 捨てる側の超過バースト（バイト） | police_be | 数（0〜） |  |  |  |  |
| 超えたとき | police_exceed | 選択（drop、transmit、set_dscp、set_precedence） |  |  |  | 空 = drop |
| 超えたときに付ける値 | police_exceed_value | 数（0〜63） |  |  |  |  |
| 付ける DSCP | set_dscp | 数（0〜63） |  |  |  |  |
| 付ける precedence | set_precedence | 数（0〜7） |  |  |  |  |
| キューの長さ（パケット） | queue_limit | 数（1〜）。複数可 |  |  |  | PQ では high・medium・normal・low の順 |
| 公平に送る | fair_queue | 有効・無効 |  |  |  |  |
| 付ける CoS | set_cos | 数（0〜7） |  |  |  |  |
| 付ける QoS グループ | set_qos_group | 数（0〜65535） |  |  |  |  |
| 付ける DSCP（IPv6） | set_ipv6_dscp | 数（0〜63） |  |  |  |  |
| 付ける precedence（IPv6） | set_ipv6_precedence | 数（0〜7） |  |  |  |  |
| 遅らせ方 | shape_mode | 選択（average、peak） |  |  |  | 空 = average |
| 上限・遅らせる（%） | shape_percent | 数（1〜100） |  |  |  |  |
| 上限・捨てる（%） | police_percent | 数（1〜100） |  |  |  |  |
| 最大の速さ・捨てる（bps） | police_pir_bps | 数（1〜） |  |  |  |  |
| 最大を超えたとき | police_violate | 選択（drop、transmit、set_dscp、set_precedence） |  |  |  |  |
| 最大を超えたときに付ける値 | police_violate_value | 数（0〜63） |  |  |  |  |
| 混む前に間引く（WRED） | random_detect | 選択（on、dscp、precedence） |  |  |  | on = 機器の既定、dscp・precedence = その値で間引き方を変える |
| キューの長さの単位 | queue_limit_unit | 選択（bytes、ms、us） |  |  |  | 空 = パケット |

## QoS の適用

<a id="qos_bindings"></a>内部名 `qos_bindings`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| IF 名 | if_name | IF 名 | ○ | ○ |  |  |
| 向き | direction | 選択（input、output） | ○ | ○ |  |  |
| ポリシー名 | policy | 参照（qos_policies.policy） |  |  |  | 空でもよい（IF 全体を絞るだけ） |
| IF 全体を絞る速さ（bps） | shape_bps | 数（1〜） |  |  |  | 出力だけ |
| IF 全体の通常バースト（ビット） | shape_bc | 数（1〜） |  |  |  |  |
| IF 全体の超過バースト（ビット） | shape_be | 数（0〜） |  |  |  |  |
| NGN 用の既定のポリシー | ngn_default | 有効・無効 |  |  |  | 機器が持つ NGN 用のポリシーを当てる（ポリシー名は空にする） |

## 経路制御の方針

<a id="route_policies"></a>内部名 `route_policies`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 方針名 | name | 文字 | ○ | ○ |  | BGP ネイバーの受信・送信の経路制御、再配布の経路制御の条件、IF のポリシールーティングから、この名前で指す |
| 順番 | seq | 数（0〜65535） | ○ | ○ |  | 小さい順に見て、最初に条件が合った行で決まる |
| 動作 | action | 選択（permit、deny） | ○ |  |  | permit = 条件に合ったら通して値を付ける。deny = 条件に合ったら落とす（ポリシールーティングでは、ふつうの経路表どおりに送る） |
| 条件: 宛先（プレフィックスリスト） | match_prefix_list | 参照（prefix_lists.name）。複数可 |  |  |  | 経路の宛先がリストに合う。複数はどれか 1 つに合えばよい |
| 条件: ACL | match_acl | 参照（acls.name）。複数可 |  |  |  | ポリシールーティングでは通信、経路の方針では経路の宛先を ACL で見る |
| 条件: ネクストホップ（プレフィックスリスト） | match_next_hop_prefix_list | 参照（prefix_lists.name） |  |  |  |  |
| 条件: AS パス | match_as_path | 参照（as_path_lists.name） |  |  |  |  |
| 条件: コミュニティ | match_community | 文字。複数可 |  |  |  | 経路が、書いたコミュニティをすべて持つ。64496:100 の形か、no-export などの語 |
| 条件: コミュニティのリスト | match_community_list | 参照（community_lists.name） |  |  |  | 「条件: コミュニティ」とは、1 行にどちらか 1 つ |
| 条件: コミュニティが完全に同じ | match_community_exact | 有効・無効 |  |  |  | 書いたもの以外のコミュニティを持つ経路には合わない |
| 条件: 出力 IF | match_interface | IF 名。複数可 |  |  |  | 経路の出口の IF |
| 条件: メトリック | match_metric | 数（0〜） |  |  |  |  |
| 条件: タグ | match_tag | 数（0〜）。複数可 |  |  |  |  |
| 付ける値: ローカルプリファレンス | set_local_preference | 数（0〜） |  |  |  |  |
| 付ける値: メトリック（MED） | set_metric | 文字 |  |  |  | 数で書く。+10・-10 と書くと、今の値に足す・引く |
| 付ける値: メトリックの種類 | set_metric_type | 選択（1、2、internal） |  |  |  | 1・2 = OSPF の外部経路の種類。internal = BGP の MED に IGP のメトリックを入れる |
| 付ける値: AS パスに足す AS | set_as_path_prepend | 数（1〜）。複数可 |  |  |  | 書いた順に足す |
| 付ける値: コミュニティ | set_community | 文字。複数可 |  |  |  | 64496:100 の形か、no-export などの語。none は、コミュニティをすべて外す |
| コミュニティを足す | set_community_additive | 有効・無効 |  |  |  | 有効 = 今のコミュニティに足す。空・無効 = 置き換える |
| 付ける値: weight | set_weight | 数（0〜65535） |  |  |  |  |
| 付ける値: オリジン | set_origin | 選択（igp、incomplete） |  |  |  |  |
| 付ける値: タグ | set_tag | 数（0〜） |  |  |  |  |
| 付ける値: ネクストホップ | set_next_hop | IPv4 アドレス。ほかに self。複数可 |  |  |  | 経路ではネクストホップの属性、ポリシールーティングでは送り先。self = 自分のアドレス（BGP） |
| 経路が無いときのネクストホップ | set_default_next_hop | IPv4 アドレス。複数可 |  |  |  | ポリシールーティング。経路表に宛先が無いときだけ使う |
| 付ける値: 出力 IF | set_interface | IF 名。ほかに Null*。複数可 |  |  |  | ポリシールーティング |
| 経路が無いときの出力 IF | set_default_interface | IF 名。ほかに Null*。複数可 |  |  |  | ポリシールーティング。経路表に宛先が無いときだけ使う |
| 条件: IPv6 の宛先（プレフィックスリスト） | match_ipv6_prefix_list | 参照（ipv6_prefix_lists.name）。複数可 |  |  |  | IPv6 の経路の宛先がリストに合う |
| 条件: IPv6 フィルタ | match_ipv6_acl | 参照（ipv6_acl_entries.acl）。複数可 |  |  |  | ポリシールーティングでは、IPv6 の通信をフィルタで見る |
| 条件: IPv6 のネクストホップ（プレフィックスリスト） | match_ipv6_next_hop_prefix_list | 参照（ipv6_prefix_lists.name） |  |  |  | IPv6 の経路のネクストホップがリストに合う |
| 条件: ネクストホップ（ACL） | match_next_hop_acl | 参照（acls.name）。複数可 |  |  |  | 経路のネクストホップを ACL で見る。Cisco だけ（NEC IX はプレフィックスリストで見る） |
| 条件: 経路の種類 | match_route_type | 選択（local、internal、external、external_type1、external_type2）。複数可 |  |  |  | local = 自分で作った BGP の経路。internal = OSPF のエリア内・エリア間。external = 外部経路（type1・type2 で分けられる）。Cisco だけ |
| 合った後に続けて見る順番 | continue_to | 文字 |  |  |  | 条件に合って値を付けた後、この順番の行も続けて見る。next = 次の行。Cisco だけ |
| 外すコミュニティ | set_community_delete | 文字。複数可 |  |  |  | 経路から、ここに書いたコミュニティだけを外す。Cisco はパックがリストを作って書く |
| 付ける値: 拡張コミュニティ（ルートターゲット） | set_extcommunity_rt | 文字。複数可 |  |  |  | 64496:100 の形。Cisco だけ |
| ルートターゲットを足す | set_extcommunity_rt_additive | 有効・無効 |  |  |  | 有効 = 今のルートターゲットに足す |
| 付ける値: 拡張コミュニティ（Site of Origin） | set_extcommunity_soo | 文字 |  |  |  | 64496:100 の形。Cisco だけ |
| 付ける値: IPv6 のネクストホップ | set_ipv6_next_hop | IPv6 アドレス。複数可 |  |  |  | IPv6 のポリシールーティングの送り先（経路ではネクストホップの属性） |
| 経路が無いときの IPv6 のネクストホップ | set_ipv6_default_next_hop | IPv6 アドレス。複数可 |  |  |  | IPv6 のポリシールーティング。経路表に宛先が無いときだけ使う |

## AS パスのリスト

<a id="as_path_lists"></a>内部名 `as_path_lists`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  | Cisco は番号（1〜500） |
| 順番 | seq | 数 | ○ | ○ |  | 書いた順に 10 刻みで振る |
| 動作 | action | 選択（permit、deny） | ○ |  |  |  |
| AS パスの正規表現 | regex | 文字 | ○ |  |  | 例 ^64496_ |

## コミュニティのリスト

<a id="community_lists"></a>内部名 `community_lists`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  | 番号も名前として持つ |
| 順番 | seq | 数 | ○ | ○ |  | 書いた順に 10 刻みで振る。行どうしは、どれかに合えばよい |
| 動作 | action | 選択（permit、deny） | ○ |  |  |  |
| 種類 | type | 選択（standard、expanded） | ○ |  |  | standard = コミュニティを並べる。expanded = 正規表現 |
| コミュニティ | communities | 文字。複数可 |  |  |  | standard の行。1 行に並べたものは「すべて持つ」。64496:100 の形か、no-export などの語 |
| 正規表現 | regex | 文字 |  |  |  | expanded の行 |

## IPsec 追加の提案

<a id="ipsec_proposals"></a>内部名 `ipsec_proposals`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| IPsec の名前 | tunnel | 参照（ipsec_tunnels.name） | ○ | ○ |  |  |
| 段 | phase | 選択（ike、ipsec） | ○ | ○ |  | ike = IKE の提案の候補、ipsec = IPsec（ESP・AH）の提案の候補 |
| 順番 | seq | 数 | ○ | ○ |  | IPsec の行に書いた提案の次に、小さい順に出す候補。番号は順番を決めるだけ（取り込みは 20・30… と振る） |
| 暗号 | encryption | 選択（des、3des、aes128、aes192、aes256、aes128gcm、aes256gcm、null） |  |  |  |  |
| ハッシュ | hash | 選択（md5、sha1、sha256、sha384、sha512） |  |  |  |  |
| DH グループ | dh | 選択（dh1、dh2、dh5、dh14、dh15、dh16、dh19、dh20、dh21、dh24） |  |  |  | 段が ike のとき |
| PRF | prf | 選択（md5、sha1、sha256、sha384、sha512） |  |  |  | 段が ike で、IKE の版 2 のとき |
| AH ハッシュ | ah_hash | 選択（md5、sha1、sha256、sha384、sha512） |  |  |  | 段が ipsec のとき |
| 有効時間（秒） | lifetime | 数 |  |  |  |  |
| 有効量（キロバイト） | lifetime_kb | 数 |  |  |  | 段が ipsec のとき |

## IPsec の全体設定

<a id="ipsec_settings"></a>内部名 `ipsec_settings`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 再送攻撃の検出 | anti_replay | 有効・無効 |  |  |  | 同じパケットをもう一度受けたら捨てる。空 = 機器の既定（有効） |
| 再送攻撃の検出の窓 | replay_window | 数（64〜1024） |  |  |  | 順番の入れ替わりを許すパケットの数（64・128・256・512・1024） |
| 外側の DF ビット | df_bit | 選択（copy、set、clear） |  |  |  | 暗号化したパケットの外側のヘッダの DF ビット。copy = 元のパケットから写す、set = 立てる、clear = 立てない（途中で分割できる） |
| 通信が無いときに切るまでの時間（秒） | idle_time | 数（60〜86400） |  |  |  | この時間だけ通信が無い相手の IPsec を切る。空 = 切らない |
| アグレッシブモードを受け付ける | aggressive_mode | 有効・無効 |  |  |  | 無効 = IKE 版 1 のアグレッシブモードを、受けることも始めることもしない。空 = 機器の既定（受け付ける） |
| 知らない SPI を受けたら IKE をつなぎ直す | invalid_spi_recovery | 有効・無効 |  |  |  | 相手が再起動した後などに、知らない SPI のパケットを受けたら、IKE をつないで相手に知らせる |

## L2 延伸（L2TPv3）

<a id="l2tpv3"></a>内部名 `l2tpv3`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 延伸する IF | if_name | IF 名 | ○ | ○ |  | この IF に入ったフレームを、そのまま相手へ運ぶ。インタフェースのシートに行を持つ |
| 相手アドレス | peer_address | IPv4 アドレス | ○ |  |  | 相手の機器の「送信元 IF」のアドレス |
| 回線番号（VC ID） | vc_id | 数（1〜4294967295） | ○ |  |  | 両端で同じ値 |
| 設定名 | profile | 文字 |  |  |  | pseudowire-class の名前。空ならパックが付ける |
| 送信元 IF | local_if | IF 名 | ○ |  |  | この IF のアドレスから送る |
| 張り方 | signaling | 選択（dynamic、manual） | ○ |  |  | dynamic = 両端が制御の通信で決める。manual = セッション ID とクッキーを手で決める |
| トンネルのパスワード | password | 秘密の値 |  |  |  | dynamic のとき。空 = 認証しない 画面とレポートでは伏せます |
| 自分の名前 | local_name | 文字 |  |  |  | 認証で名乗る名前。空なら機器のホスト名 |
| Hello 間隔（秒） | hello | 数（0〜） |  |  |  |  |
| 自分のセッション ID | local_session_id | 数（1〜） |  |  |  | manual のとき必須。相手の「相手のセッション ID」と同じ値 |
| 相手のセッション ID | remote_session_id | 数（1〜） |  |  |  | manual のとき必須 |
| 自分のクッキー | local_cookie | 文字 |  |  |  | manual のとき。大きさ（4 か 8）と値。例 4 54321 |
| 相手のクッキー | remote_cookie | 文字 |  |  |  | manual のとき。相手の「自分のクッキー」と同じ値 |
| 順序の確認 | sequencing | 選択（transmit、receive、both） |  |  |  |  |
| 経路 MTU の検出 | pmtu | 有効・無効 |  |  |  |  |
| DF ビットを立てる | df_bit | 有効・無効 |  |  |  |  |

## L2TP

<a id="l2tp_groups"></a>内部名 `l2tp_groups`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  | Cisco は vpdn-group の名前。NEC IX は、lns なら PPP プロファイル、lac なら URL リストの名前 |
| 役割 | role | 選択（lns、lac） | ○ |  |  | lns = 受ける側（リモートアクセスの受け口・回線を受ける側）。lac = PPP を LNS へ運ぶ側 |
| 受け口の IF | logical_ifs | IF 名。複数可 |  |  |  | lns: つないだ相手が乗る IF（Cisco は Virtual-Template を 1 つ、NEC IX はトンネル IF を同時につなぐ数だけ）。lac（NEC IX）: LNS へのトンネル IF |
| PPPoE を受ける IF | access_ifs | IF 名。複数可 |  |  |  | lac（NEC IX）: 配下の PPPoE を受けて L2TP へ渡す IF |
| 相手アドレス | peer_address | 文字 |  |  |  | lac のとき必須: LNS のアドレス |
| 相手の名前 | peer_name | 文字 |  |  |  | lns: 受け付ける相手（LAC）の名前 |
| 自分の名前 | local_name | 文字 |  |  |  | トンネルで名乗る名前 |
| トンネルの認証 | tunnel_auth | 有効・無効 |  |  | ○ | 無効 = トンネルを張るときに相手を確かめない |
| トンネルのパスワード | password | 秘密の値 |  |  |  | 画面とレポートでは伏せます |
| 運ぶドメイン | domains | 文字。複数可 |  |  |  | lac: 接続 ID の @ の後ろがこの名前の PPP を運ぶ |
| 利用者の認証 | auth | 選択（chap、pap、chap_pap、mschap_v2） |  |  |  | lns: つないできた利用者に求める認証 |
| 払い出すアドレスの先頭 | pool_start | IPv4 アドレス |  |  |  | lns: 利用者に配るアドレスの範囲 |
| 払い出すアドレスの末尾 | pool_end | IPv4 アドレス |  |  |  |  |
| 配る DNS サーバ | dns_servers | IPv4 アドレス。複数可 |  |  |  | lns。2 つまで |
| IPsec の設定名 | ipsec_policy | 参照（ipsec_tunnels.name） |  |  |  | lns: この受け口を守る IPsec の行の名前（IPsec のシート。相手が any で、モードが transport の行） |
| 無通信で切る時間（秒） | idle_timeout | 数（1〜） |  |  |  |  |

## L2TP 接続ユーザ

<a id="l2tp_users"></a>内部名 `l2tp_users`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| L2TP の名前 | group | 参照（l2tp_groups.name） | ○ | ○ |  |  |
| 接続 ID | name | 文字 | ○ | ○ |  |  |
| 接続パスワード | password | 秘密の値 | ○ |  |  | CHAP で確かめるので、ハッシュでは持てない 画面とレポートでは伏せます |
| 固定で払い出すアドレス | address | IPv4 アドレス |  |  |  |  |

## VLAN

<a id="vlans"></a>内部名 `vlans`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| VLAN 番号 | vlan_id | 数（1〜4094） | ○ | ○ |  |  |
| VLAN 名 | name | 文字 |  |  |  | 1〜32 字 |
| 説明 | description | 文字 |  |  |  | VLAN の説明の行を持つ機器だけ（名前とは別の文） |
| 状態 | state | 選択（active、suspend） |  |  |  | active = 使う。suspend = 定義は残して通信を止める。空 = 機器の既定（active） |
| この機器で止める | shutdown | 有効・無効 |  |  |  | この機器の中でだけ、VLAN の通信を止める |
| 遠隔ミラー用 | remote_span | 有効・無効 |  |  |  | ポートミラーの通信を、別のスイッチへ運ぶための VLAN |
| プライベート VLAN の種類 | private_vlan | 選択（primary、isolated、community） |  |  |  |  |
| プライベート VLAN のセカンダリ | private_vlan_secondaries | 文字。複数可 |  |  |  | primary の VLAN に結び付けるセカンダリの VLAN。例 501-503 |

## スイッチポート

<a id="switchports"></a>内部名 `switchports`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| IF 名 | if_name | IF 名 | ○ | ○ |  | ポート、または LAG の IF |
| モード | mode | 選択（access、trunk、dynamic_auto、dynamic_desirable、routed、hybrid、pvlan_host、pvlan_promiscuous） |  |  |  | access = 1 つの VLAN にタグなしで入る。trunk = 複数の VLAN をタグ付きで通す。routed = L3 のポートとして使う。空 = 機器の既定 |
| アクセス VLAN | access_vlan | 数（1〜4094） |  |  |  |  |
| ネイティブ VLAN | native_vlan | 数（1〜4094） |  |  |  | トランクでタグを付けずに通す VLAN。空 = 機器の既定（ふつう 1） |
| 許可 VLAN | allowed_vlans | 文字。複数可 |  |  |  | トランクで通す VLAN。例 10,20,30-40。all = すべて、none = なし。空 = 機器の既定（ふつう すべて） |
| トランクのタグ方式 | trunk_encapsulation | 選択（dot1q、isl、negotiate） |  |  |  | 方式を選べる機器だけ |
| モードの自動交渉を止める | nonegotiate | 有効・無効 |  |  |  |  |
| 音声 VLAN | voice_vlan | 数（1〜4094） |  |  |  |  |
| すぐ転送にする（エッジポート） | stp_edge | 選択（on、trunk、off） |  |  |  | on = 端末をつなぐポートとして、つないですぐ転送にする（PortFast）。trunk = トランクでも同じにする。off = 全体の既定を、このポートでは止める |
| BPDU ガード | stp_bpdu_guard | 有効・無効 |  |  |  | BPDU を受けたらポートを止める |
| ストーム制御・ブロードキャスト（%） | storm_broadcast | 文字 |  |  |  | 帯域に対する割合。例 10.00。速さで書くときは 500pps（1 秒あたりのパケット数）、64kbps、1500bps |
| ストーム制御・マルチキャスト（%） | storm_multicast | 文字 |  |  |  |  |
| ストーム制御・ユニキャスト（%） | storm_unicast | 文字 |  |  |  |  |
| ストームのときの動作 | storm_action | 選択（shutdown、trap） |  |  |  | 空 = 超えた分を捨てるだけ |
| ポートセキュリティ | port_security | 有効・無効 |  |  |  |  |
| 覚える MAC の数 | port_security_max | 数（1〜） |  |  |  |  |
| 違反のときの動作 | port_security_violation | 選択（protect、restrict、shutdown） |  |  |  |  |
| 覚えた MAC を設定に残す | port_security_sticky | 有効・無効 |  |  |  |  |
| タグ付きで通す VLAN（hybrid） | hybrid_tagged | 文字。複数可 |  |  |  | モードが hybrid のポートが、タグ付きで通す VLAN。10,20,30-40 の形 |
| タグなしで通す VLAN（hybrid） | hybrid_untagged | 文字。複数可 |  |  |  | モードが hybrid のポートが、タグなしで通す VLAN。入ってくるタグなしのフレームの VLAN は「ネイティブ VLAN」 |
| スパニングツリー（ポート） | stp_enabled | 有効・無効 |  |  |  | 無効 = このポートではスパニングツリーを止める。空 = 機器の既定（有効） |
| ガード | stp_guard | 選択（root、loop、none） |  |  |  | root = 上位の BPDU を受けたら止める（ルートガード）。loop = BPDU が来なくなっても転送にしない（ループガード）。none = 全体の既定のガードを、このポートでは止める |
| パスコスト | stp_cost | 数（1〜） |  |  |  |  |
| ポートの優先度 | stp_port_priority | 数（0〜240） |  |  |  |  |
| ループ検出をする VLAN | loop_detect_vlans | 文字。複数可 |  |  |  | このポートでループ検出をする VLAN。10,20,30-40 の形か all |
| ループのときの動作 | loop_detect_action | 選択（block、no_learning、shutdown） |  |  |  | 空 = 記録するだけ。block = 止める、no_learning = MAC を覚えない、shutdown = ポートを落とす |
| DHCP スヌーピングで信頼する | dhcp_snooping_trust | 有効・無効 |  |  |  | 有効 = DHCP サーバのいる側のポート |
| 音声 VLAN の伝え方 | voice_vlan_mode | 選択（dot1p、none、untagged） |  |  |  | 番号を使わないとき: dot1p = 優先度だけのタグ。untagged = タグなし。none = 電話機に何も伝えない |
| ストーム制御・ブロードキャストの下限 | storm_broadcast_low | 文字 |  |  |  | ここまで下がったら通信を戻す値。上の値と同じ書き方（10.00、500pps、64kbps）で、同じ単位にする |
| ストーム制御・マルチキャストの下限 | storm_multicast_low | 文字 |  |  |  |  |
| ストーム制御・ユニキャストの下限 | storm_unicast_low | 文字 |  |  |  |  |
| 許可する MAC アドレス | port_security_macs | 文字。複数可 |  |  |  | 手で決めた MAC アドレス。0000.5e00.5301 の形。VLAN を限るときは 0000.5e00.5301/10（トランクの VLAN 10）、0000.5e00.5301/access、0000.5e00.5301/voice |
| 覚えた MAC アドレス | port_security_sticky_macs | 文字。複数可 |  |  |  | 機器が覚えて設定に残したアドレス。書き方は「許可する MAC アドレス」と同じ |
| MAC を忘れるまでの時間（分） | port_security_aging_time | 数（0〜1440） |  |  |  | 0 = 忘れない |
| MAC の忘れ方 | port_security_aging_type | 選択（absolute、inactivity） |  |  |  | absolute = 覚えてからの時間。inactivity = 通信が無くなってからの時間 |
| 手で決めた MAC も忘れる | port_security_aging_static | 有効・無効 |  |  |  |  |
| 保護ポート | protected | 有効・無効 |  |  |  | 保護ポートどうしは、同じ VLAN でも通信させない |
| BPDU フィルタ | stp_bpdu_filter | 有効・無効 |  |  |  | BPDU を送らず、受けても捨てる。有効・無効のどちらも書ける |
| スパニングツリーのリンクの種類 | stp_link_type | 選択（point_to_point、shared） |  |  |  | 空 = 二重から機器が決める |
| プライベート VLAN のプライマリ | pvlan_primary | 数（1〜4094） |  |  |  | モードが pvlan_host・pvlan_promiscuous のポートが属するプライマリの VLAN |
| プライベート VLAN のセカンダリ（ポート） | pvlan_secondaries | 文字。複数可 |  |  |  | pvlan_host は 1 つ。pvlan_promiscuous は並び（501-503） |
| 片方向リンクの検出（UDLD） | udld | 選択（enable、aggressive、disable） |  |  |  | enable・aggressive = このポートで使う。disable = 全体の設定を、このポートでは止める |
| DHCP の受信の上限（毎秒） | dhcp_snooping_rate | 数（1〜） |  |  |  |  |
| ARP 検査の信頼ポート | arp_inspection_trust | 有効・無効 |  |  |  |  |

## LAG のメンバー

<a id="lags"></a>内部名 `lags`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| IF 名 | if_name | IF 名 | ○ | ○ |  | 束ねるポート |
| LAG の番号 | group | 数（1〜） | ○ |  |  | 同じ番号のポートを 1 本に束ねる。LAG の IF（Cisco は Port-channel<番号>）の設定は、インタフェースとスイッチポートの表に書く |
| 束ね方 | mode | 選択（on、active、passive、auto、desirable） |  |  |  | on = 固定。active・passive = LACP。auto・desirable = PAgP（Cisco） |
| LACP の短いタイムアウト | lacp_short | 有効・無効 |  |  |  | 有効 = 相手に 1 秒ごとの送信を求める |
| PAgP を黙らない相手として扱う | non_silent | 有効・無効 |  |  |  | 束ね方が auto・desirable のときだけ（Cisco の non-silent） |
| 束ねる手順の指定 | protocol | 選択（lacp、pagp） |  |  |  | ポートで使う手順を先に決める（Cisco の channel-protocol） |
| LACP のポート優先度 | lacp_port_priority | 数（1〜65535） |  |  |  |  |

## L2 の全体設定

<a id="l2_settings"></a>内部名 `l2_settings`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| スパニングツリーを使う | stp_enabled | 有効・無効 |  |  |  | 機器全体の有効・無効。空 = 機器の既定（メーカーで違う） |
| スパニングツリーの方式 | stp_mode | 選択（pvst、rapid_pvst、mst、rstp、stp） |  |  |  | pvst・rapid_pvst = VLAN ごとの木（rapid は収束の速い方式）。mst = MSTP。rstp・stp = 機器に 1 本の木（RSTP・STP） |
| 優先度（機器全体） | stp_priority | 数（0〜65535） |  |  |  | 機器に 1 本の木（rstp・stp）と、MSTP の共通の木での優先度。VLAN ごとの優先度は「VLAN ごとのスパニングツリー」に書く |
| アクセスポートをすぐ転送にする | stp_edge_default | 有効・無効 |  |  |  | 全体の既定（PortFast default） |
| エッジポートの BPDU ガード | stp_bpdu_guard_default | 有効・無効 |  |  |  | すぐ転送にするポートすべてに BPDU ガードを掛ける |
| 拡張システム ID | stp_extend_system_id | 有効・無効 |  |  |  |  |
| L3 の転送 | ip_routing | 有効・無効 |  |  |  | L3 スイッチで経路による転送を使う |
| デフォルトゲートウェイ | default_gateway | IPv4 アドレス |  |  |  | L3 の転送を使わない機器（L2 スイッチ）の管理通信の出口 |
| VTP のモード | vtp_mode | 選択（server、client、transparent、off） |  |  |  |  |
| VTP のドメイン名 | vtp_domain | 文字 |  |  |  |  |
| ブリッジと L3 の併用（IRB） | bridge_irb | 有効・無効 |  |  |  | ブリッジグループを BVI で L3 につなぐ |
| ルートブリッジの指定（機器全体） | stp_root | 選択（primary、secondary） |  |  |  | primary = ルートブリッジにする、secondary = 予備のルートブリッジにする（優先度を機器が決める） |
| Hello 間隔（秒） | stp_hello | 数 |  |  |  | 機器に 1 本の木と、MSTP の共通の木 |
| 転送遅延（秒） | stp_forward | 数 |  |  |  |  |
| 最大エージ（秒） | stp_max_age | 数 |  |  |  |  |
| MST リージョン名 | mst_name | 文字 |  |  |  |  |
| MST リビジョン | mst_revision | 数（0〜65535） |  |  |  |  |
| ループ検出をする VLAN（全体） | loop_detect_vlans | 文字。複数可 |  |  |  | 全部のポートでループ検出をする VLAN。10,20,30-40 の形か all |
| ループのときの動作（全体） | loop_detect_action | 選択（shutdown） |  |  |  | 空 = 記録するだけ |
| ループ検出の間隔（秒） | loop_detect_interval | 数 |  |  |  |  |
| DHCP スヌーピング | dhcp_snooping | 有効・無効 |  |  |  |  |
| ポートセキュリティを使う | port_security | 有効・無効 |  |  |  | 機器全体で入り切りする機器だけ（NEC QX の port-security enable） |
| LACP のシステム優先度 | lacp_system_priority | 数（0〜65535） |  |  |  |  |
| ループガードを全体の既定にする | stp_loopguard_default | 有効・無効 |  |  |  |  |
| エッジポートの BPDU フィルタ | stp_bpdu_filter_default | 有効・無効 |  |  |  | すぐ転送にするポートすべてに BPDU フィルタを掛ける |
| スパニングツリーのコストの桁 | stp_pathcost_method | 選択（short、long） |  |  |  | short = 16 ビット。long = 32 ビット |
| BackboneFast | stp_backbonefast | 有効・無効 |  |  |  |  |
| UplinkFast | stp_uplinkfast | 有効・無効 |  |  |  |  |
| LAG の振り分け方 | lag_load_balance | 選択（dst-ip、dst-mac、dst-mixed-ip-port、dst-port、src-dst-ip、src-dst-mac、src-dst-mixed-ip-port、src-dst-port、src-ip、src-mac、src-mixed-ip-port、src-port） |  |  |  | どの値でメンバーに振り分けるか |
| VTP の版 | vtp_version | 選択（1、2、3） |  |  |  |  |
| VTP のパスワード | vtp_password | 秘密の値 |  |  |  | 画面とレポートでは伏せます |
| VTP プルーニング | vtp_pruning | 有効・無効 |  |  |  |  |
| 自動で戻す停止の原因 | errdisable_recovery_causes | 文字。複数可 |  |  |  | 止まったポートを自動で戻す原因（機器の語。bpduguard、psecure-violation など。all = すべて） |
| 自動で戻すまでの時間（秒） | errdisable_recovery_interval | 数（30〜86400） |  |  |  |  |
| 片方向リンクの検出（UDLD） | udld | 選択（enable、aggressive） |  |  |  | 光のポート全体の既定 |
| UDLD の送信間隔（秒） | udld_message_time | 数（1〜90） |  |  |  |  |
| CDP | cdp | 有効・無効 |  |  |  | 機器全体の有効・無効。空 = 機器の既定 |
| CDP の送信間隔（秒） | cdp_timer | 数 |  |  |  |  |
| CDP の保持時間（秒） | cdp_holdtime | 数 |  |  |  |  |
| LLDP | lldp | 有効・無効 |  |  |  | 機器全体の有効・無効。空 = 機器の既定 |
| LLDP の送信間隔（秒） | lldp_timer | 数 |  |  |  |  |
| LLDP の保持時間（秒） | lldp_holdtime | 数 |  |  |  |  |
| LLDP の再初期化の待ち（秒） | lldp_reinit | 数 |  |  |  |  |
| DHCP スヌーピングの VLAN | dhcp_snooping_vlans | 文字。複数可 |  |  |  | 例 10,20,30-40 |
| DHCP スヌーピングの option 82 | dhcp_snooping_option82 | 有効・無効 |  |  |  | 有効・無効のどちらも書ける（無効 = 中継情報を付けない） |
| ARP 検査の VLAN | arp_inspection_vlans | 文字。複数可 |  |  |  |  |
| MAC アドレスを忘れるまでの時間（秒） | mac_aging_time | 数（0〜1000000） |  |  |  | 0 = 忘れない |
| 内部 VLAN の振り方 | vlan_internal_allocation | 選択（ascending、descending） |  |  |  | L3 のポートのために機器が内部で使う VLAN を、小さい番号から（ascending）か大きい番号から（descending）振る |
| スイッチ全体の MTU | system_mtu | 数 |  |  |  |  |
| ループとみなす回数 | loop_detect_threshold | 数（1〜10000） |  |  |  |  |
| ループ検出の窓の大きさ | loop_detect_window | 数（1〜5） |  |  |  |  |
| BVI の数 | bvi_count | 数（1〜64） |  |  |  | 機器が持つ BVI の数を変える（対応する機器だけ） |

## VLAN ごとのスパニングツリー

<a id="stp_vlans"></a>内部名 `stp_vlans`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| VLAN | vlans | 文字 | ○ | ○ |  | VLAN 番号。まとめて書くときは 10,20,30-40 |
| 優先度 | priority | 数（0〜65535） |  |  |  | 小さいほどルートブリッジになりやすい |
| Hello 間隔（秒） | hello_time | 数 |  |  |  |  |
| 転送遅延（秒） | forward_time | 数 |  |  |  |  |
| 最大エージ（秒） | max_age | 数 |  |  |  |  |
| ルートブリッジの指定 | root | 選択（primary、secondary） |  |  |  | primary・secondary（優先度を機器が決める） |
| スパニングツリーを使う | enabled | 有効・無効 |  |  |  | 無効 = この VLAN ではスパニングツリーを止める |
| ネットワークの直径 | root_diameter | 数（2〜7） |  |  |  | 「ルートブリッジにする」と一緒に書く。スイッチの段数（2〜7） |

## スタックのメンバー

<a id="stack_members"></a>内部名 `stack_members`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| メンバー番号 | member | 数（1〜） | ○ | ○ |  |  |
| 機種の種別 | provision | 文字 |  |  |  | そのメンバーの機種の種別（機器が設定に持つ語） |
| 優先度 | priority | 数（1〜） |  |  |  | 大きいほどマスターになりやすい |
| 説明 | description | 文字 |  |  |  |  |

## IPv6 インタフェース

<a id="ipv6_interfaces"></a>内部名 `ipv6_interfaces`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| IF 名 | if_name | IF 名 | ○ | ○ |  | 固定の IPv6 アドレスは、シート「インタフェース」の「IPv6 アドレス」に書く |
| IPv6 を使う | enabled | 有効・無効 |  |  |  | アドレスを書かずに IPv6 を使う IF（リンクローカルだけで動かす） |
| リンクローカルアドレス | link_local | IPv6 アドレス |  |  |  | 例 fe80::1。空 = 機器が自動で作る |
| EUI-64 で付けるプレフィックス | eui64 | IPv6 ネットワーク。複数可 |  |  |  | 例 2001:db8:0:1::/64。下 64 ビットは機器が MAC アドレスから作る |
| エニーキャストアドレス | anycast | IPv6 アドレス/長さ。複数可 |  |  |  |  |
| 自動設定（RA を受ける） | autoconfig | 有効・無効 |  |  |  | ルータ広告（RA）を受けてアドレスを作る。IPoE の WAN 側 |
| RA の送り元をデフォルト経路にする | autoconfig_default | 有効・無効 |  |  |  |  |
| DHCPv6 でアドレスをもらう | dhcp_address | 有効・無効 |  |  |  |  |
| IPv6 MTU | mtu | 数（1280〜） |  |  |  |  |
| 入力 IPv6 フィルタ | acl_in | 参照（ipv6_acl_entries.acl） |  |  |  |  |
| 出力 IPv6 フィルタ | acl_out | 参照（ipv6_acl_entries.acl） |  |  |  |  |
| 入力 IPv6 フィルタ（2 つ目以降） | acl_in_more | 参照（ipv6_acl_entries.acl）。複数可 |  |  |  | 入力 IPv6 フィルタの後に、書いた順で見るリスト |
| 出力 IPv6 フィルタ（2 つ目以降） | acl_out_more | 参照（ipv6_acl_entries.acl）。複数可 |  |  |  | 出力 IPv6 フィルタの後に、書いた順で見るリスト |
| RA を送る | ra | 選択（send、reply_only、off） |  |  |  | send = 送る、reply_only = 問い合わせにだけ答える、off = 送らない。空 = 機器の既定 |
| RA の間隔・最大（秒） | ra_interval | 数 |  |  |  |  |
| RA の間隔・最小（秒） | ra_interval_min | 数 |  |  |  |  |
| RA のルータ生存時間（秒） | ra_lifetime | 数 |  |  |  |  |
| RA の M フラグ | ra_managed | 有効・無効 |  |  |  | アドレスを DHCPv6 でもらうよう端末に伝える |
| RA の O フラグ | ra_other | 有効・無効 |  |  |  | DNS などを DHCPv6 でもらうよう端末に伝える |
| RA で配る DNS サーバ | ra_dns | IPv6 アドレス。複数可 |  |  |  |  |
| RA で配る DNS の生存時間（秒） | ra_dns_lifetime | 数 |  |  |  |  |
| ND プロキシの LAN 側 IF | nd_proxy | IF 名 |  |  |  | WAN 側の IF の行に、LAN 側の IF 名を書く（プレフィックスを LAN と共有する IPoE） |
| ND プロキシの対象外リスト | nd_proxy_acl | 参照（ipv6_acl_entries.acl） |  |  |  |  |
| プレフィックスをもらう（DHCPv6-PD） | pd_request | 文字 |  |  |  | もらったプレフィックスに付ける名前。WAN 側の IF の行に書く |
| もらったプレフィックスから付ける | pd_from | 文字 |  |  |  | プレフィックスの名前（「プレフィックスをもらう」に書いた名前）。LAN 側の IF の行に書く |
| 付けるアドレスの下の部分 | pd_suffix | IPv6 アドレス/長さ |  |  |  | 例 ::1/64。もらったプレフィックスの後ろに足す |
| 下 64 ビットを EUI-64 で作る | pd_eui64 | 有効・無効 |  |  |  |  |
| DHCPv6 の Rapid Commit | rapid_commit | 有効・無効 |  |  |  |  |
| DHCPv6 でもらう情報 | dhcp_request | 選択（dns、domain、ntp）。複数可 |  |  |  |  |
| DHCPv6 で情報だけもらう | dhcp_info_only | 有効・無効 |  |  |  | アドレスやプレフィックスはもらわず、DNS などだけをもらう |
| DHCPv6 サーバ（名前） | dhcp_server | 文字 |  |  |  | この IF で DHCPv6 サーバを動かす。設定の名前を書く |
| DHCPv6 で配る DNS サーバ | dhcp_dns | IPv6 アドレス。ほかに dhcp。複数可 |  |  |  | アドレス。dhcp と書くと、WAN 側の DHCPv6 でもらった DNS を配る |
| DHCPv6 で配るドメイン名 | dhcp_domain | 文字 |  |  |  |  |
| DHCPv6 で配る NTP サーバ | dhcp_ntp | IPv6 アドレス。ほかに dhcp。複数可 |  |  |  | アドレス。dhcp と書くと、WAN 側の DHCPv6 でもらった NTP サーバを配る（NEC IX） |
| 下流へ配るプレフィックスの元 | dhcp_pd_pool | IPv6 ネットワーク |  |  |  | 下流のルータへ配るプレフィックスを、ここから切り出す。例 2001:db8::/32。「上流からもらったプレフィックスから配る」があるときは、もらったプレフィックスの下の部分（先頭から切り出すなら ::/56。もらったプレフィックスの部分は 0 で書く） |
| 下流へ配る長さ | dhcp_pd_length | 数（1〜128） |  |  |  | 1 台に配るプレフィックスの長さ。例 48 |
| 上流からもらったプレフィックスから配る | dhcp_pd_from | 文字 |  |  |  | WAN 側の「プレフィックスをもらう（DHCPv6-PD）」の名前。もらったプレフィックスを分けて、下流へ配り直す（NEC IX） |
| 配るプレフィックスの有効時間（秒） | dhcp_pd_valid | 数（0〜）。ほかに infinite |  |  |  | 秒か infinite（無期限）。推奨時間と組で書く。空 = 機器の既定 |
| 配るプレフィックスの推奨時間（秒） | dhcp_pd_preferred | 数（0〜）。ほかに infinite |  |  |  |  |
| 端末へ配るアドレスのプレフィックス | dhcp_address_prefix | IPv6 ネットワーク。複数可 |  |  |  | 端末へ DHCPv6 でアドレスを配る（Cisco） |
| IPv6 のポリシールーティング | policy_route | 文字 |  |  |  | この IF に入ってくる IPv6 の通信に当てる経路制御の方針の名前 |
| IPv6 の TCP MSS 調整 | tcp_mss | 数。ほかに auto |  |  |  | 数か auto（MTU から機器が決める。NEC IX） |

## RA のプレフィックス

<a id="ipv6_ra_prefixes"></a>内部名 `ipv6_ra_prefixes`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| IF 名 | if_name | IF 名 | ○ | ○ |  |  |
| プレフィックス | prefix | IPv6 ネットワーク | ○ | ○ |  | 例 2001:db8:0:1::/64。IF のアドレスのプレフィックスは、行がなくても機器が RA に載せる |
| 有効時間（秒） | valid_lifetime | 数 |  |  | ○ |  |
| 推奨時間（秒） | preferred_lifetime | 数 |  |  | ○ |  |
| オンリンク | on_link | 有効・無効 |  |  | ○ | このプレフィックスが同じリンクにあると端末に伝える |
| 端末の自動設定に使う | autonomous | 有効・無効 |  |  | ○ | 端末がこのプレフィックスでアドレスを作ってよい |

## IPv6 スタティック経路

<a id="ipv6_routes"></a>内部名 `ipv6_routes`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| VRF | vrf | 参照（vrfs.name） |  | ○ |  | 空 = VRF なし（全体の経路表） |
| 宛先 | prefix | IPv6 ネットワーク | ○ | ○ |  | 例 2001:db8:10::/48。デフォルト経路は ::/0 |
| ネクストホップ | next_hop | IPv6 アドレス |  | ○ |  | リンクローカル（fe80::…）のときは出力 IF も書く |
| 出力 IF | out_if | IF 名。ほかに Null* |  | ○ |  | 捨てる経路の Null0 などは、インタフェースの表に行がなくてもよい |
| ネクストホップの決め方 | gateway_from | 選択（ra、dhcp） |  |  |  | 出力 IF だけを書いた経路で、RA の送り元（ra）か DHCPv6 サーバ（dhcp）をネクストホップにする |
| 優先度（distance） | distance | 数（1〜255） |  |  |  |  |
| 説明 | description | 文字 |  |  |  |  |

## IPv6 フィルタ

<a id="ipv6_acl_entries"></a>内部名 `ipv6_acl_entries`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| リスト名 | acl | 文字 | ○ | ○ |  | IPv4 の ACL とは別の名前の世界（同じ名前でもよい） |
| 順番 | seq | 数 | ○ | ○ |  | 番号を持たない書き方では 10 刻みで振る |
| 動作 | action | 選択（permit、deny、remark） | ○ |  |  |  |
| プロトコル | protocol | 文字 |  |  |  | ip（すべて）、tcp、udp、icmp か番号 |
| 送信元 | src | 文字 |  |  |  | any、2001:db8::1（ホスト）、2001:db8:1::/64（ネットワーク） |
| 送信元ポート | src_port | 文字 |  |  |  | 80、1000-2000、>1023、<1024、!80 |
| 宛先 | dst | 文字 |  |  |  | 送信元と同じ書き方 |
| 宛先ポート | dst_port | 文字 |  |  |  | 送信元ポートと同じ書き方 |
| 付加条件 | flags | 選択（established、log、fragments、log_input、ack、fin、psh、rst、syn、urg）。複数可 |  |  |  |  |
| ICMPv6 の種類 | icmp_type | 文字 |  |  |  | 番号か、メーカーの語（echo-request など） |
| 注記 | remark | 文字 |  |  |  |  |
| DSCP | dscp | 数（0〜63） |  |  |  |  |
| ICMPv6 のコード | icmp_code | 数（0〜255） |  |  |  | ICMPv6 の種類を番号で書いた行だけ |

## IPv6 プレフィックスリスト

<a id="ipv6_prefix_lists"></a>内部名 `ipv6_prefix_lists`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  |  |
| 順番 | seq | 数 | ○ | ○ |  |  |
| 動作 | action | 選択（permit、deny） | ○ |  |  |  |
| 範囲 | prefix | IPv6 ネットワーク | ○ |  |  | 例 2001:db8::/32。すべては ::/0 と「長さの上限」128 |
| 長さの下限（ge） | ge | 数（0〜128） |  |  |  |  |
| 長さの上限（le） | le | 数（0〜128） |  |  |  |  |

## OSPFv3

<a id="ospfv3_processes"></a>内部名 `ospfv3_processes`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| プロセス番号 | process_id | 数 | ○ | ○ |  |  |
| ルータ ID | router_id | IPv4 アドレス |  |  |  | IPv4 アドレスの形（例 192.0.2.1） |
| 既定を受動にする | passive_default | 有効・無効 |  |  |  |  |
| デフォルト経路の広告 | default_originate | 選択（none、when_present、always） |  |  |  |  |
| 広告のメトリック | default_metric | 数 |  |  |  |  |
| 再配布する経路 | redistribute | 選択（connected、static、bgp、rip）。複数可 |  |  |  |  |

## OSPFv3 インタフェース

<a id="ospfv3_interfaces"></a>内部名 `ospfv3_interfaces`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| プロセス番号 | process_id | 数 | ○ | ○ |  |  |
| IF 名 | if_name | IF 名 | ○ | ○ |  |  |
| エリア | area | 文字 | ○ |  |  | 0 と 0.0.0.0 は同じ値として扱う |
| コスト | cost | 数 |  |  |  |  |
| 優先度 | priority | 数（0〜255） |  |  |  |  |
| Hello 間隔（秒） | hello | 数 |  |  |  |  |
| Dead 間隔（秒） | dead | 数 |  |  |  |  |
| 網の種類 | network_type | 選択（broadcast、point_to_point、nbma、point_to_multipoint） |  |  |  |  |
| 受動 | passive | 有効・無効 |  |  |  |  |
| 認証のアルゴリズム | auth_hash | 選択（md5、sha1、none） |  |  |  | IPsec の AH で隣接を認証する（Cisco の ipv6 ospf authentication）。none = エリアの認証をこの IF では使わない |
| 認証の SPI | auth_spi | 数（256〜4294967295） |  |  |  | 両端で同じ値 |
| 認証の鍵 | auth_key | 秘密の値 |  |  |  | 16 進の数字（md5 は 32 字、sha1 は 40 字）。両端で同じ値 画面とレポートでは伏せます |

## OSPFv3 エリア

<a id="ospfv3_areas"></a>内部名 `ospfv3_areas`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| プロセス番号 | process_id | 数 | ○ | ○ |  |  |
| エリア | area | 文字 | ○ | ○ |  |  |
| 種類 | type | 選択（normal、stub、totally_stub、nssa、totally_nssa） |  |  |  |  |
| 既定経路のコスト | default_cost | 数 |  |  |  |  |
| 集約して広告する範囲 | ranges | IPv6 ネットワーク。複数可 |  |  |  |  |
| 集約して広告しない範囲 | ranges_hidden | IPv6 ネットワーク。複数可 |  |  |  |  |
| 認証のアルゴリズム | auth_hash | 選択（md5、sha1） |  |  |  | エリアの全部の IF の隣接を IPsec の AH で認証する（Cisco の area … authentication） |
| 認証の SPI | auth_spi | 数（256〜4294967295） |  |  |  |  |
| 認証の鍵 | auth_key | 秘密の値 |  |  |  | 16 進の数字（md5 は 32 字、sha1 は 40 字） 画面とレポートでは伏せます |

## BGP IPv6 ネイバー

<a id="bgp_ipv6_neighbors"></a>内部名 `bgp_ipv6_neighbors`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| VRF | vrf | 参照（vrfs.name） |  | ○ |  | 空 = VRF なし（全体の経路表） |
| ネイバーアドレス | neighbor | IPv6 アドレス | ○ | ○ |  |  |
| 相手 AS 番号 | remote_as | 数 | ○ |  |  |  |
| 説明 | description | 文字 |  |  |  |  |
| 有効 | enabled | 有効・無効 |  |  |  |  |
| 送信元 IF | update_source | IF 名 |  |  |  |  |
| eBGP マルチホップ | ebgp_multihop | 数 |  |  |  |  |
| パスワード | password | 秘密の値 |  |  |  | 画面とレポートでは伏せます |
| ネクストホップを自分にする | next_hop_self | 有効・無効 |  |  |  |  |
| 受信経路の保持 | soft_reconfig_in | 有効・無効 |  |  |  |  |
| 受信フィルタ | prefix_list_in | 参照（ipv6_prefix_lists.name） |  |  |  |  |
| 送信フィルタ | prefix_list_out | 参照（ipv6_prefix_lists.name） |  |  |  |  |
| 受信の経路制御 | filter_in | 文字 |  |  |  | 条件の名前 |
| 送信の経路制御 | filter_out | 文字 |  |  |  | 条件の名前 |
| コミュニティを送る | send_community | 有効・無効 |  |  |  |  |
| 拡張コミュニティを送る | send_ext_community | 有効・無効 |  |  |  |  |

## SNMP ユーザ（版 3）

<a id="snmp_users"></a>内部名 `snmp_users`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| ユーザ名 | name | 文字 | ○ | ○ |  |  |
| グループ | group | 文字 | ○ |  |  | SNMP のグループの名前。同じグループの行は、段・ビュー・グループの接続元の制限を同じにする |
| セキュリティの段 | security_level | 選択（noauth、auth、priv） | ○ |  |  | グループの段。noauth = 認証なし、auth = 認証あり、priv = 認証と暗号化 |
| 読めるビュー | read_view | 文字 |  |  |  | 空 = 機器の既定のビュー |
| 書けるビュー | write_view | 文字 |  |  |  | 空 = 書き込みを許さない |
| 通知のビュー | notify_view | 文字 |  |  |  |  |
| グループの接続元の制限 | group_acl | 参照（acls.name） |  |  |  |  |
| 認証の方式 | auth | 選択（md5、sha） |  |  |  |  |
| 認証のパスワード | auth_key | 秘密の値 |  |  |  | 画面とレポートでは伏せます |
| 暗号化の方式 | priv | 選択（des、3des、aes128） |  |  |  |  |
| 暗号化のパスワード | priv_key | 秘密の値 |  |  |  | 画面とレポートでは伏せます |
| 接続元の制限 | acl | 参照（acls.name） |  |  |  |  |

## MSTP のインスタンス

<a id="stp_instances"></a>内部名 `stp_instances`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| インスタンス番号 | instance | 数（0〜4094） | ○ | ○ |  | 0 は共通の木（CIST）。VLAN を割り当てない |
| VLAN | vlans | 文字。複数可 |  |  |  | このインスタンスに割り当てる VLAN。10,20,30-40 の形 |
| 優先度 | priority | 数（0〜65535） |  |  |  |  |
| ルートブリッジの指定 | root | 選択（primary、secondary） |  |  |  | primary・secondary（優先度を機器が決める） |

## ポートミラー

<a id="mirror_sessions"></a>内部名 `mirror_sessions`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 番号 | session | 数（1〜） | ○ | ○ |  |  |
| 写す先の IF | destination_if | IF 名 |  |  |  | 写した通信を出すポート（測定器をつなぐ） |
| 写す元（両方向） | sources_both | IF 名。複数可 |  |  |  |  |
| 写す元（受信） | sources_rx | IF 名。複数可 |  |  |  |  |
| 写す元（送信） | sources_tx | IF 名。複数可 |  |  |  |  |

## 時間帯

<a id="time_ranges"></a>内部名 `time_ranges`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  | ACL の行の「効かせる時間帯」から、この名前で指す |
| 順番 | seq | 数 | ○ | ○ |  | 取り込みでは 10 刻みで振る |
| 種類 | kind | 選択（periodic、absolute） | ○ |  |  | periodic = 毎週くり返す。absolute = 日時で区切る（1 つの名前に 1 行まで） |
| 曜日 | days | 選択（monday、tuesday、wednesday、thursday、friday、saturday、sunday、daily、weekdays、weekend）。複数可 |  |  |  | periodic だけ |
| 開始の時刻 | start_time | 文字 |  |  |  | 8:00 の形（24 時間） |
| 終了の曜日 | end_days | 選択（monday、tuesday、wednesday、thursday、friday、saturday、sunday、daily、weekdays、weekend）。複数可 |  |  |  | 開始と違う曜日に終わるときだけ |
| 終了の時刻 | end_time | 文字 |  |  |  | 18:00 の形 |
| 開始の日 | start_date | 文字 |  |  |  | absolute だけ。2026-01-31 の形 |
| 終了の日 | end_date | 文字 |  |  |  | absolute だけ。2026-12-31 の形 |

## ログの対象ごとの重要度

<a id="log_subsystems"></a>内部名 `log_subsystems`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 対象 | subsystem | 文字 | ○ | ○ |  | 機器の中の機能の名前（メーカーの語）。全部は、機器のシートの「ログに残す重要度」に書く |
| 重要度 | severity | 選択（emergencies、alerts、critical、errors、warnings、notifications、informational、debugging） | ○ |  |  |  |

## ブリッジグループ

<a id="bridge_groups"></a>内部名 `bridge_groups`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| ブリッジグループ | group | 数（1〜65535） | ○ | ○ |  | グループに入れる IF は、インタフェースの表の「ブリッジグループ」に書く。この表は、グループごとの設定があるときだけ |
| スパニングツリーの種類 | protocol | 選択（ieee、dec、vlan-bridge、ibm） |  |  |  | ブリッジグループで動かすスパニングツリー |
| 優先度 | priority | 数（0〜65535） |  |  |  |  |
| MAC アドレスを忘れるまでの時間（秒） | aging_time | 数（10〜1000000） |  |  |  |  |
| 覚える MAC の数 | table_size | 数（0〜65535）。ほかに unlimited |  |  |  | unlimited = 上限なし |
| IPv4 をブリッジする | bridge_ip | 有効・無効 |  |  |  | 無効 = このグループでは IPv4 をブリッジしない。空 = 機器の既定（する） |
| IPv6 をブリッジする | bridge_ipv6 | 有効・無効 |  |  |  |  |
| PPPoE だけをブリッジする | bridge_only_pppoe | 有効・無効 |  |  |  |  |

## BGP 集約経路

<a id="bgp_aggregates"></a>内部名 `bgp_aggregates`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| VRF | vrf | 参照（vrfs.name） |  | ○ |  | 空 = VRF なし（全体の経路表） |
| 集約した経路 | prefix | IPv4 ネットワーク | ○ | ○ |  | 中に入る経路が BGP の表に 1 つでもあれば、この経路を広告する |
| 細かい経路を広告しない | summary_only | 有効・無効 |  |  |  | 有効 = 集約した経路だけを広告する |
| 元の経路の AS を残す | as_set | 有効・無効 |  |  |  | 有効 = 集約した経路に、元の経路が通ってきた AS の集まりを付ける |

## OSPF バーチャルリンク

<a id="ospf_virtual_links"></a>内部名 `ospf_virtual_links`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| プロセス番号 | process_id | 数 | ○ | ○ |  |  |
| 通るエリア | area | 文字 | ○ | ○ |  | バックボーンにじかにつながらないエリアを、このエリアを通してつなぐ |
| 相手のルータ ID | router_id | IPv4 アドレス | ○ | ○ |  |  |
| Hello 間隔（秒） | hello | 数 |  |  |  |  |
| Dead 間隔（秒） | dead | 数 |  |  |  |  |

## OSPFv3 再配布の指定

<a id="ospfv3_redistribute"></a>内部名 `ospfv3_redistribute`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| プロセス番号 | process_id | 数 | ○ | ○ |  |  |
| 再配布の元 | source | 選択（connected、static、bgp、rip） | ○ | ○ |  | OSPFv3 の表の「再配布する経路」に入っている種類。行が無い種類は、指定なしで再配布する |
| メトリック | metric | 数 |  |  |  |  |
| メトリックの種類 | metric_type | 選択（1、2） |  |  |  |  |
| タグ | tag | 数 |  |  |  |  |
| 経路制御の条件 | filter | 文字 |  |  |  | 経路制御の方針の表の名前。方針を追加コンフィグに書くときは、その名前 |

## BGP IPv6 再配布の指定

<a id="bgp_ipv6_redistribute"></a>内部名 `bgp_ipv6_redistribute`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 再配布の元 | source | 選択（connected、static、ospf） | ○ | ○ |  | BGP の表の「IPv6 の再配布」に入っている種類。行が無い種類は、指定なしで再配布する |
| メトリック | metric | 数 |  |  |  |  |
| 経路制御の条件 | filter | 文字 |  |  |  | 経路制御の方針の表の名前。方針を追加コンフィグに書くときは、その名前 |

## RIP

<a id="rip_processes"></a>内部名 `rip_processes`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 種類 | family | 選択（ipv4、ipv6） | ○ | ○ |  | ipv4 = RIP、ipv6 = RIPng |
| 版 | version | 選択（1、2） |  |  |  | RIP（ipv4）だけ。IF ごとの「送る版」「受ける版」が空の IF に効く |
| クラスの境目で集約する | auto_summary | 有効・無効 |  |  |  | RIP（ipv4）だけ。何も書かないときの動きがメーカーで逆（Cisco は集約する、NEC IX はしない） |
| 既定を「送らない」にする | passive_default | 有効・無効 |  |  |  | IF の「送る」が空の IF では送らない（Cisco の passive-interface default） |
| デフォルト経路を広告する | default_originate | 有効・無効 |  |  |  |  |
| 広告するデフォルト経路のメトリック | default_originate_metric | 数（1〜15） |  |  |  |  |
| デフォルト経路の広告の条件 | default_originate_filter | 文字 |  |  |  | 経路制御の方針の名前（Cisco） |
| 再配布のメトリックの既定 | default_metric | 数（1〜16） |  |  |  |  |
| 優先度（distance） | distance | 数（10〜255） |  |  |  |  |
| 更新を送る間隔（秒） | timer_update | 数（1〜） |  |  |  |  |
| 経路を無効とみなすまで（秒） | timer_invalid | 数（1〜） |  |  |  | 最後の更新から数える |
| より良い経路を受け付けない時間（秒） | timer_holddown | 数（0〜） |  |  |  | Cisco だけ |
| 経路を消すまで（秒） | timer_flush | 数（1〜） |  |  |  | 最後の更新から数える（無効とみなすまで ＋ 無効になってから消すまで） |
| 受信フィルタ | prefix_list_in | 文字 |  |  |  | 受け取る経路を絞るプレフィックスリストの名前（ipv6 の行は IPv6 プレフィックスリスト） |
| 送信フィルタ | prefix_list_out | 文字 |  |  |  |  |

## RIP インタフェース

<a id="rip_interfaces"></a>内部名 `rip_interfaces`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 種類 | family | 選択（ipv4、ipv6） | ○ | ○ |  | ipv4 = RIP、ipv6 = RIPng |
| IF 名 | if_name | IF 名 | ○ | ○ |  | 行があれば、その IF で RIP が動く |
| 送る | send | 有効・無効 |  |  |  | 無効 = 経路を送らない（受けるだけ）。空 = 送る |
| 受ける | receive | 有効・無効 |  |  |  | 無効 = 経路を受けない。空 = 受ける |
| 送る版 | send_version | 選択（1、2）。複数可 |  |  |  | RIP（ipv4）だけ。空 = プロセスの「版」 |
| 受ける版 | receive_version | 選択（1、2）。複数可 |  |  |  |  |
| 版 2 をブロードキャストで送る | v2_broadcast | 有効・無効 |  |  |  |  |
| 認証 | auth_mode | 選択（text、md5） |  |  |  | RIP（ipv4）の版 2 だけ。NEC IX は text だけ |
| 認証キー | auth_key | 秘密の値 |  |  |  | 画面とレポートでは伏せます |
| 認証キーの番号 | auth_key_id | 数（0〜） |  |  |  | md5 のとき。空 = 1 |
| メトリックに足す値 | metric_offset | 数（1〜16） |  |  |  |  |
| スプリットホライズン | split_horizon | 選択（split_horizon、poisoned_reverse、none） |  |  |  | split_horizon = 受けた IF へは送らない、poisoned_reverse = 受けた IF へは届かない経路として送る、none = 何もしない |
| 集約して送る経路 | summary | IPv4 か IPv6 のネットワーク。複数可 |  |  |  |  |
| この IF でデフォルト経路を広告する | default_originate | 有効・無効 |  |  |  |  |
| 広告するデフォルト経路のメトリック | default_originate_metric | 数（1〜15） |  |  |  |  |
| 受信フィルタ | prefix_list_in | 文字 |  |  |  |  |
| 送信フィルタ | prefix_list_out | 文字 |  |  |  |  |

## RIP 再配布

<a id="rip_redistribute"></a>内部名 `rip_redistribute`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 種類 | family | 選択（ipv4、ipv6） | ○ | ○ |  | ipv4 = RIP、ipv6 = RIPng |
| 再配布する経路 | source | 選択（connected、static、ospf、bgp） | ○ | ○ |  | ipv6 の行の ospf は OSPFv3 |
| OSPF のプロセス番号 | process_id | 数 |  |  |  | 再配布する経路が ospf のとき。空なら、機器に 1 つだけあるプロセス |
| メトリック | metric | 数（1〜16） |  |  |  |  |
| タグ | tag | 数（0〜65535） |  |  |  | NEC IX だけ |
| 経路制御の条件 | filter | 文字 |  |  |  | 経路制御の方針の名前 |

## 冗長化（IPv6 の VRRP）

<a id="ipv6_fhrp"></a>内部名 `ipv6_fhrp`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| IF 名 | if_name | IF 名 | ○ | ○ |  |  |
| グループ番号 | group | 数（1〜255） | ○ | ○ |  | IPv6 の VRRP は版 3（VRRPv3）。NEC IX は、IPv4 の VRRP と同じ番号を機器の中で使えない |
| 仮想リンクローカルアドレス | link_local | IPv6 アドレス | ○ |  |  | 例 fe80::1。同じグループの機器で同じ値にする |
| 仮想グローバルアドレス | addresses | IPv6 アドレス。複数可 |  |  |  | 長さを付けずに書く（例 2001:db8:0:1::1） |
| 優先度 | priority | 数（1〜254） |  |  | ○ |  |
| 切り戻し | preempt | 有効・無効 |  |  | ○ |  |
| 切り戻し待ち（秒） | preempt_delay | 数 |  |  |  |  |
| 広告間隔（ミリ秒） | advert_interval | 数（10〜40000） |  |  | ○ | IPv4 の冗長化（秒）と単位が違う。例 1000 = 1 秒 |
| 回線監視 | track | 参照（tracks.id） |  |  |  |  |
| 監視断のときの減点 | track_decrement | 数（1〜255） |  |  |  |  |
