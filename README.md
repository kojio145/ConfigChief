# ConfigChief

**Network device configurations ⇄ a vendor-neutral Excel workbook.**
ConfigChief reads router and switch configurations into a vendor-neutral table (an Excel "parameter book"), and writes configurations, a check report and delivery sheets back from that table. It is made for replacing devices, also across vendors.

- One Windows exe. No installer, no network access, no listening port
- The same input always gives the same output (no AI, no randomness inside)
- A line that could not be read is always listed as "unknown" — nothing is dropped silently
- The program knows no vendor: everything about a vendor lives in a text "vendor pack"
- Bundled packs: Cisco IOS / IOS-XE (routers, small routers with a built-in switch, Catalyst switches), NEC UNIVERGE IX (routers) and NEC QX series (switches)
- The parameter book can be edited in Excel or in a table editor inside the program (with a draft that survives a restart)
- The user interface and the documents are in Japanese (for now)

> **Not verified on real devices.** The bundled packs were written from the vendors' public command references. Run every generated configuration on a lab device before you use it.

The exe is not code-signed, so Microsoft Defender SmartScreen may warn on the first start ("More info" → "Run anyway"). Compare the SHA-256 of your download with the one on the release page. This is a spare-time project: issues are read, answers come on an irregular schedule. Never paste real configurations into an issue.

License: [MIT](LICENSE). Third-party notices: [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md).

---

# ConfigChief（日本語）

ネットワーク機器のコンフィグを読んで、メーカーによらない表（Excel のパラメータブック）にし、その表からコンフィグ・チェック結果表・納品用のシートを出すツールです。機器の更改（同じメーカーでも、別メーカーでも）と新規構築に使えます。

![結果の画面。Cisco の拠点ルータを NEC IX に書き換え、書けない値を指摘として出している（値はすべて架空の見本）](docs/img/result-findings.png)

## できること

- **取り込み**: 旧機器のコンフィグ（端末ログのままでも可）を読んで、表にします。行ごとに「取り込み済み・無視・未対応」が分かります
- **書き換え**: 同じ表から、別メーカーのコンフィグを出します（Cisco IOS ⇄ NEC IX、Catalyst 系 ⇄ NEC QX、内蔵スイッチ付きの小型ルータ → NEC IX など）。書けない値・置き換えて書いた設定は、必ず指摘として出し、直し方も出します
- **IF 対応の案**: 新しい機種の ID を書くと、機種のポートの並びから、旧 IF 名 → 新 IF 名の案を作ります
- **表の編集**: パラメータブックは、Excel でも、画面の表でも直せます（下書き・検索と置換・行の挿入と並べ替え。Excel 側の変更とぶつかったら、どちらを残すかを選べます）
- **チェック**: 値の誤り、2 台の間の食い違い（IPsec・BGP・OSPF・VRRP・VLAN・LAG など）、旧との違いを調べ、提出用のチェック結果表を出します。どの指摘にも直し方が付きます。指摘は理由を付けて承認できます
- **自己点検**: 生成したコンフィグを読み直して、元の表と同じになることを毎回確かめます
- **検証と投入の後**: 検証機の投入ログから弾かれた行を探し、投入後の設定とパラメータブックの違いを出します
- **納品物**: Excel のひな形に値を流し込んだ納品用パラメータシート、相手に渡すパラメータ表
- **新規構築**: 空のパラメータブックから 1 台目を作り、複製して増やせます

同梱のパック: Cisco IOS / IOS-XE（ルータ、内蔵スイッチ付きの小型ルータ、Catalyst 系のスイッチ）、NEC UNIVERGE IX（ルータ）、NEC QX シリーズ（スイッチ）。

対応している設定: 機器の基本（ユーザ・遠隔ログイン・NTP・syslog・SNMP〈版 3 を含む〉）・インタフェース・VLAN・スイッチポート・LAG・スパニングツリー・ブリッジ・ポートミラー・スタティック経路・VRF・OSPF・BGP・VRRP/HSRP・経路制御の方針（ルートマップ・ポリシールーティング）・ACL・プレフィックスリスト・NAT・回線監視・PPPoE・DHCP・IPsec（IKEv1・IKEv2）・GRE・EtherIP・L2TP・QoS・IPv6（アドレス・RA・経路・フィルタ・OSPFv3・BGP・IPoE のトンネル）。メーカーによって、機能が無くて書けないものや、パックが読まない書き方があります（[使い方](docs/manual.md) の 11 章と、各パックの README）。

## 守っていること

- exe 1 つ。インストール不要。**外部と通信しません。待ち受けポートを開きません**
- 同じ入力からは、いつも同じ出力（AI や乱数を使いません）
- 読めなかった行は「未対応」として必ず一覧に出します
- 本体はメーカーを知りません。メーカーの知識は、テキストのメーカーパックにあります

## ⚠️ 実機では確かめていません

同梱のパックは、メーカーが公開しているコマンドリファレンスを読んで作りました。実機の出力では確かめていません。**生成したコンフィグは、必ず検証機に流して確かめてから使ってください。** 何を確かめていないかは、各パックの README にあります（[Cisco IOS](internal/packs/cisco_ios/README.md)・[NEC IX](internal/packs/nec_ix/README.md)・[NEC QX](internal/packs/nec_qx/README.md)）。

## 使い方

1. [Releases](../../releases) から `ConfigChief.exe` を取り、好きなフォルダに置いて起動します（WebView2 ランタイムが要ります。Windows 11 には初めから入っています）
2. 「案件」で案件フォルダを作り、旧機器のコンフィグを `input/old/` に置いて「取り込み」
3. 対応（旧ホスト名 → 新ホスト名・新メーカー・新機種、旧 IF 名 → 新 IF 名）を、画面の表か Excel で埋めて、パラメータブックにする
4. 「生成とチェック」で、全台のコンフィグとチェック結果表が出ます

署名の無い exe なので、初めて起動するときに Microsoft Defender SmartScreen の警告が出ることがあります。「詳細情報」→「実行」で起動できます。取った exe が本物かどうかは、リリースのページにある SHA-256 と比べて確かめられます（PowerShell: `Get-FileHash ConfigChief.exe`）。

くわしくは:

- [使い方](docs/manual.md)
- [シートと列の一覧](docs/tables.md)
- [チェックの一覧](docs/checks.md)
- [メーカーパックの書き方](docs/pack-guide.md)
- [開発記（Zenn）](https://zenn.dev/kjo145/articles/configchief-config-to-excel-and-back)

## ビルド

Go 1.26 系が要ります。npm も wails コマンドも要りません。

```
build.bat
```

`go vet` と `go test` の後、`build\bin\ConfigChief.exe` ができます。

コードとパックのコメントにある「spec 9.6」「appendix B 3.6」「cmdref 7.15」などは、作者の手元の設計メモの節番号です（設計メモは公開していません。決まりの中身は、このリポジトリの文書と各パックの README に書いてあります）。

## サポート方針

個人が余暇で作っているツールです。Issue と要望は読みますが、対応は不定期です。**Issue に実際のコンフィグを貼らないでください**（ホスト名・アドレス・パスワード・拠点名は伏せてください）。脆弱性の報告は [SECURITY.md](SECURITY.md) のとおり、非公開でお願いします。

本番の機器に入れる前に、生成したコンフィグを検証機で確かめるのは、使う人の責任でお願いします（MIT ライセンスのとおり、無保証です）。

## ライセンス

[MIT](LICENSE)。使っている部品のライセンスは [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md) にあります。

作者: KJO
