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

## 機器

<a id="devices"></a>内部名 `devices`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| ホスト名 | hostname | 文字 | ○ | ○ |  | 機器の名前。全シートの「機器」列の参照先 |
| メーカー | vendor | 文字 | ○ |  |  | メーカーパックの ID。例 cisco_ios、nec_ix コンフィグには書きません |
| 機種 | model | 文字 |  |  |  | パックの機種定義の ID。空でも生成できるが、IF 対応の案とポートの検査が使えない コンフィグには書きません |
| OS 版 | os_version | 文字 |  |  |  | コンフィグには書きません |
| グループ | group | 文字。複数可 |  |  |  | @グループ名 で行をまとめて当てるための名前。複数はカンマで区切る コンフィグには書きません |
| 拠点 | site | 文字 |  |  |  | コンフィグには書きません |
| ドメイン名 | domain_name | 文字 |  |  |  |  |
| DNS サーバ | name_servers | IPv4 アドレス。複数可 |  |  |  |  |
| 時刻帯 | timezone | 文字 |  |  |  | 例 JST +9。名前は省ける（+9）。分があれば +5:30 |
| 特権パスワード | enable_secret | 秘密の値 |  |  |  | 画面とレポートでは伏せます |
| パスワードの形 | secret_form | 選択（plain、hashed） |  |  |  | 取り込んだ値がハッシュのとき hashed。別メーカーへは引き継げない |
| SNMP 設置場所 | snmp_location | 文字 |  |  |  |  |
| SNMP 連絡先 | snmp_contact | 文字 |  |  |  |  |
| バナー | banner | 文章 |  |  |  | ログインの前に出す文 |
| 備考 | note | 文字 |  |  |  | 生成には使わない コンフィグには書きません |

## インタフェース

<a id="interfaces"></a>内部名 `interfaces`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| IF 名 | name | IF 名 | ○ | ○ |  | メーカーの正式な書き方。例 GigabitEthernet0/0/0 |
| 種類 | kind | 選択（physical、subif、loopback、tunnel、dialer、vlan、lag、other） |  |  |  | 空なら名前から決める コンフィグには書きません |
| 説明 | description | 文字 |  |  |  |  |
| 有効 | enabled | 有効・無効 | ○ |  | ○ | 無効 = shutdown |
| IPv4 アドレス | ipv4 | IPv4 アドレス/長さ。ほかに dhcp、negotiated、unnumbered:* |  |  |  | 例 192.0.2.1/30。dhcp、negotiated、unnumbered:<IF 名> も書ける |
| IPv4 セカンダリ | ipv4_secondary | IPv4 アドレス/長さ。複数可 |  |  |  |  |
| IPv6 アドレス | ipv6 | IPv6 アドレス/長さ。複数可 |  |  |  |  |
| VRF | vrf | 参照（vrfs.name） |  |  |  |  |
| VLAN 番号（サブ IF） | encap_vlan | 数（1〜4094） |  |  |  |  |
| MTU | mtu | 数 |  |  |  |  |
| IP MTU | ip_mtu | 数 |  |  |  |  |
| TCP MSS 調整 | tcp_mss | 数 |  |  |  |  |
| 速度 | speed | 選択（auto、10、100、1000、10000） |  |  |  | 単位は Mbps |
| 二重 | duplex | 選択（auto、full、half） |  |  |  |  |
| 帯域（kbps） | bandwidth | 数 |  |  |  | 経路計算用の値 |
| 入力 ACL | acl_in | 参照（acls.name） |  |  |  |  |
| 出力 ACL | acl_out | 参照（acls.name） |  |  |  |  |
| NAT の向き | nat_role | 選択（inside、outside） |  |  |  | メーカーによっては使わない |
| DHCP 中継先 | dhcp_relay | IPv4 アドレス。複数可 |  |  |  |  |
| 代理 ARP | proxy_arp | 有効・無効 |  |  |  |  |
| 備考 | note | 文字 |  |  |  | 生成には使わない コンフィグには書きません |

## スタティック経路

<a id="static_routes"></a>内部名 `static_routes`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| VRF | vrf | 参照（vrfs.name） |  | ○ |  | 空 = VRF なし（全体の経路表） |
| 宛先 | prefix | IPv4 ネットワーク | ○ | ○ |  | 例 192.0.2.0/24。デフォルト経路は 0.0.0.0/0 |
| ネクストホップ | next_hop | IPv4 アドレス |  | ○ |  |  |
| 出力 IF | out_if | IF 名。ほかに Null* |  | ○ |  | 捨てる経路の Null0 などは、インタフェースの表に行がなくてもよい |
| 優先度（distance） | distance | 数（1〜255） |  |  |  |  |
| 回線監視 | track | 参照（tracks.id） |  |  |  |  |
| 説明 | description | 文字 |  |  |  |  |

## ユーザ

<a id="users"></a>内部名 `users`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| ユーザ名 | name | 文字 | ○ | ○ |  |  |
| 権限 | privilege | 文字 |  |  |  | admin、operator か、メーカーの値（数値など） |
| パスワード | secret | 秘密の値 | ○ |  |  | 画面とレポートでは伏せます |
| パスワードの形 | secret_form | 選択（plain、hashed） |  |  |  | 取り込んだ値がハッシュのとき hashed |

## 遠隔ログイン

<a id="remote_access"></a>内部名 `remote_access`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 許可する方式 | protocols | 選択（ssh、telnet）。複数可 | ○ |  |  |  |
| SSH の版 | ssh_version | 選択（1、2） |  |  |  |  |
| 接続元の制限 | acl | 参照（acls.name） |  |  |  |  |
| 無操作切断（分） | idle_timeout | 数 |  |  |  |  |
| 認証 | auth | 選択（local、password、aaa） |  |  |  | aaa の中身は追加コンフィグに書く |
| 回線パスワード | line_password | 秘密の値 |  |  |  | 画面とレポートでは伏せます |

## NTP サーバ

<a id="ntp_servers"></a>内部名 `ntp_servers`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| サーバ | server | 文字 | ○ | ○ |  | アドレスか名前 |
| 優先 | prefer | 有効・無効 |  |  |  |  |
| 送信元 IF | source_if | IF 名 |  |  |  |  |
| VRF | vrf | 参照（vrfs.name） |  |  |  |  |

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
| 通知先 | host | IPv4 アドレス | ○ | ○ |  |  |
| 版 | version | 選択（1、2c、3） |  |  |  |  |
| コミュニティ | community | 秘密の値 |  |  |  | 画面とレポートでは伏せます |
| 送信元 IF | source_if | IF 名 |  |  |  |  |

## VRF

<a id="vrfs"></a>内部名 `vrfs`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  |  |
| RD | rd | 文字 |  |  |  |  |
| 説明 | description | 文字 |  |  |  |  |

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
| 認証 | auth_type | 選択（none、simple、md5） |  |  | ○ |  |
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
| 経路制御の条件 | filter | 文字 |  |  |  | 条件の名前。中身は追加コンフィグに書く |

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
| 受信の経路制御 | filter_in | 文字 |  |  |  | 条件の名前。中身は追加コンフィグに書く |
| 送信の経路制御 | filter_out | 文字 |  |  |  | 条件の名前。中身は追加コンフィグに書く |

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
| 経路制御の条件 | filter | 文字 |  |  |  | 条件の名前。中身は追加コンフィグに書く |

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

## ACL

<a id="acls"></a>内部名 `acls`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  | 番号の ACL は番号を名前として書く |
| 種類 | type | 選択（standard、extended） |  |  |  |  |
| 説明 | description | 文字 |  |  |  |  |

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
| 付加条件 | flags | 選択（established、log、fragments）。複数可 |  |  |  |  |
| ICMP の種類 | icmp_type | 文字 |  |  |  |  |
| 注記 | remark | 文字 |  |  |  | 動作が remark の行の本文 |
| DSCP | dscp | 数（0〜63） |  |  |  | この DSCP の通信だけに合う。数で書く（ef = 46） |
| precedence | precedence | 数（0〜7） |  |  |  | この precedence の通信だけに合う |

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
| 種類 | type | 選択（pat_interface、pat_pool、dynamic_pool、static、static_port） | ○ |  |  | pat_interface = 外側 IF のアドレスへまとめて変換 |
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

## 回線監視

<a id="tracks"></a>内部名 `tracks`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 番号・名前 | id | 文字 | ○ | ○ |  |  |
| 監視の方法 | type | 選択（icmp、interface、route） | ○ |  |  |  |
| 監視先 | target | 文字 |  |  |  | icmp はアドレス、interface は IF 名、route はネットワーク |
| 送信元 IF | source_if | IF 名 |  |  |  |  |
| VRF | vrf | 参照（vrfs.name） |  |  |  |  |
| 間隔（秒） | interval | 数 |  |  | ○ |  |
| 待ち時間（ミリ秒） | timeout | 数 |  |  |  |  |
| 断と判断する回数 | fail_count | 数 |  |  | ○ |  |
| 復旧と判断する回数 | restore_count | 数 |  |  |  |  |
| 断の反映待ち（秒） | down_delay | 数 |  |  |  |  |
| 復旧の反映待ち（秒） | up_delay | 数 |  |  |  |  |

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

## IPsec

<a id="ipsec_tunnels"></a>内部名 `ipsec_tunnels`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| 名前 | name | 文字 | ○ | ○ |  | 取り込み時は、トンネル IF の名前か「マップ名/番号」。比べるときは、名前でなく中身で行を合わせる |
| 方式 | style | 選択（tunnel_if、policy） | ○ |  |  | tunnel_if = トンネル IF に経路を向ける。policy = 対象通信で選ぶ |
| トンネル IF | tunnel_if | IF 名 |  |  |  | 方式が tunnel_if のとき必須 |
| 自分側 IF | local_if | IF 名 |  |  |  | 自分側 IF か自分側アドレスのどちらか |
| 自分側アドレス | local_address | IPv4 アドレス |  |  |  |  |
| 相手アドレス | peer_address | 文字 | ○ |  |  | アドレス。相手が動的なら any |
| IKE の版 | ike_version | 選択（1、2） | ○ |  | ○ |  |
| IKE のモード | ike_mode | 選択（main、aggressive） |  |  | ○ | 版 1 のみ |
| 自分の ID | local_id | 文字 |  |  |  |  |
| 相手の ID | remote_id | 文字 |  |  |  |  |
| 事前共有鍵 | psk | 秘密の値 | ○ |  |  | 画面とレポートでは伏せます |
| IKE 暗号 | ike_encryption | 選択（des、3des、aes128、aes192、aes256、aes128gcm、aes256gcm、null） | ○ |  | ○ |  |
| IKE ハッシュ | ike_hash | 選択（md5、sha1、sha256、sha384、sha512） | ○ |  | ○ |  |
| IKE DH グループ | ike_dh | 選択（dh1、dh2、dh5、dh14、dh15、dh16、dh19、dh20、dh21、dh24） | ○ |  | ○ |  |
| IKE 有効時間（秒） | ike_lifetime | 数 |  |  | ○ |  |
| ESP 暗号 | esp_encryption | 選択（des、3des、aes128、aes192、aes256、aes128gcm、aes256gcm、null） | ○ |  | ○ |  |
| ESP ハッシュ | esp_hash | 選択（md5、sha1、sha256、sha384、sha512） | ○ |  | ○ |  |
| PFS | pfs | 選択（none、dh1、dh2、dh5、dh14、dh15、dh16、dh19、dh20、dh21、dh24） |  |  | ○ |  |
| IPsec 有効時間（秒） | ipsec_lifetime | 数 |  |  | ○ |  |
| 相手監視の間隔（秒） | dpd_interval | 数 |  |  |  |  |
| 相手監視の再送回数 | dpd_retry | 数 |  |  |  |  |
| NAT 越え | nat_traversal | 有効・無効 |  |  |  |  |

## IPsec 対象通信

<a id="ipsec_selectors"></a>内部名 `ipsec_selectors`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| IPsec の名前 | tunnel | 参照（ipsec_tunnels.name） | ○ | ○ |  |  |
| 順番 | seq | 数 | ○ | ○ |  |  |
| 自分側ネットワーク | local_net | IPv4 ネットワーク | ○ |  |  |  |
| 相手側ネットワーク | remote_net | IPv4 ネットワーク | ○ |  |  |  |
| プロトコル | protocol | 文字 |  |  |  |  |

## トンネル

<a id="tunnels"></a>内部名 `tunnels`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| トンネル IF | if_name | IF 名 | ○ | ○ |  | アドレスはインタフェースのシートに書く |
| 種類 | type | 選択（gre、ipip、ipsec、l2tpv3） | ○ |  |  | ipsec は IPsec のシートのトンネル IF と対になる |
| 送信元 | source | 文字 |  |  |  | IF 名かアドレス。種類が ipsec 以外では必須 |
| 宛先 | destination | IPv4 アドレス |  |  |  | 種類が ipsec 以外では必須 |
| 鍵（GRE） | key | 数 |  |  |  |  |
| 監視間隔（秒） | keepalive_interval | 数 |  |  |  |  |
| 監視の再送回数 | keepalive_retry | 数 |  |  |  |  |

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
| DNS サーバ | dns_servers | IPv4 アドレス。複数可 |  |  |  |  |
| ドメイン名 | domain | 文字 |  |  |  |  |
| 貸し出し時間 | lease | 文字 |  |  |  | 1d、12h の形 |

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
| キュー | queue | 選択（high、medium、normal、low） |  |  |  | この条件に合った通信を入れる優先度のキュー |

## QoS のポリシー

<a id="qos_policies"></a>内部名 `qos_policies`

| 見出し | 内部名 | 型 | 必 | キー | 明示 | 説明 |
|---|---|---|---|---|---|---|
| 機器 | device | 参照（devices.hostname） | ○ | ○ |  | ホスト名。* は全機器、@グループ名 はそのグループの機器 |
| ポリシー名 | policy | 文字 | ○ | ○ |  | 1〜31 字 |
| クラス名 | class | 参照（qos_classes.name）。ほかに default、local | ○ | ○ |  | QoS の見分け方のクラス名か、default（どれにも当たらない通信）、local（機器自身が出す通信） |
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
