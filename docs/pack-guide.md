# メーカーパックの書き方

ConfigChief の本体は、メーカーを知りません。コンフィグの読み方・書き方・語の対応は、すべて「メーカーパック」というフォルダ（YAML とテンプレートのテキスト）にあります。新しいメーカーや新しい設定は、パックを足す・直すことで対応します。プログラムは直しません。

同梱のパック（`internal/packs/cisco_ios`・`internal/packs/nec_ix`）が、いちばんの見本です。このページと並べて読んでください。

## 1. はじめに決まりを 4 つ

1. **確かめていない書き方は、規則にしない。** 読めない行は「未対応」として人に見えます。誤って読むより安全です。メーカーが公開しているコマンドリファレンスで構文を確かめ、資料の名前と版を README に書きます
2. **取り込みの規則とテンプレートは、対で書く。** 書き出した文面を同じパックで読み直し、元の表と同じになることを、ツールが毎回確かめます（往復の確認）
3. **黙って落とさない。** 設定として意味のある行を「無視」に入れてはいけません。書けない値は、テンプレートで理由と直し方を付けて止めます
4. **見本は架空の値で書く。** アドレスは例示用（`192.0.2.0/24`・`198.51.100.0/24`・`203.0.113.0/24`・`2001:db8::/32`）、AS 番号は `64496`〜`64511`、名前は `example.com`・`rt-a01` のような自作のものを使います。現場で見た値は、形を変えても使いません

## 2. 作って試す

```
packs/<パックの ID>/
  pack.yaml        基本の情報、メーカーの判定、文面の形、IF 名、端末ログの切り出し
  ignore.yaml      無視する行
  import/*.yaml    取り込みの規則（ファイル名の順に読む）
  defaults.yaml    行が無いときの機器の動き（既定値）
  lexicon.yaml     語の対応（中立の語 ↔ メーカーの語）
  render.yaml      コンフィグを書く順番
  templates/*.j2   テンプレート
  adapt.yaml       このメーカーの書き方への置き換え（要るときだけ）
  errors.yaml      機器のエラーメッセージの形（投入ログのチェック用）
  words.txt        コマンドの語（未対応の行の持ち出し用レポートで残す語。無くてもよい）
  tests/import/<名前>/input.txt, expect.yaml
  tests/render/<名前>/data.yaml, expect.txt
  README.md        対応の一覧、参照した資料、確かめていないこと
```

- 案件フォルダの `packs/` に置いたパックは、同じ ID の同梱パックより優先されます。まず同梱のパックを写して、案件の中で直して試すのが早道です
- `ConfigChief.exe pack test <フォルダ>` が、パックの検査と試験を行います。`--lines` を付けると行ごとの状態、`--dump` を付けるとできた表が出ます
- `ConfigChief.exe pack columns <フォルダ>` が、パックが読む列・書く列の一覧を出します
- 表と列の一覧は [tables.md](tables.md) です。規則とテンプレートでは、内部名（`interfaces`・`ipv4` など）を使います

## 3. pack.yaml

```yaml
id: sample_os
title: {ja: "見本の OS", en: "Sample OS"}
version: 0.1.0
schema: 1
syntax: indent                  # indent = 字下げで親子が決まる。flat = すべての行が同じ階層
comment_prefixes: ["!"]
output: {encoding: utf-8, newline: lf, indent: " "}
end_marker: '^end$'             # 無ければ「途中で切れている疑い」と知らせる

detect:                         # 当たった行の点を足し、合計の高いパックを選ぶ
  - {match: '^interface GigabitEthernet\d', score: 2}
  - {match: '^line vty \d+', score: 2}

hostname: '^hostname (?P<name>\S+)$'

interfaces:
  types:
    - {name: GigabitEthernet, short: [Gi, Gig], kind: physical}
    - {name: Loopback, short: [Lo], kind: loopback}
    - {name: Tunnel, short: [Tu], kind: tunnel}
  subif: '^(?P<parent>.+)\.(?P<n>\d+)$'

session:                        # 端末ログから、設定の部分だけを切り出す
  prompt: '^(?P<host>[\w.\-]+)(\((?P<mode>[^)]+)\))?[>#]'
  show_config: ['^show running-config$']
  pager: ['--More--']
```

- **判定**: 文面の先頭 500 行に `detect` を当てます。合計が 3 点未満か、1 位と 2 位の差が 2 点未満なら、人が選びます
- **IF 名**: フィルタ `ifname` が省略形を正式な名前に直します。`Null`・`on`・`yes`・`no` のような名前は、YAML が別の値として読むので引用符で囲みます（`name: "Null"`）
- **端末ログ**: `show_config` は行の終わり（`$`）まで書きます（`show running-config interface …` のような一部の表示を、全体として拾わないため）。`pager` は跡そのものだけを書きます

## 4. 取り込みの規則（import/*.yaml）

```yaml
rules:
  - id: if
    match: '^interface (?P<name>\S+)$'
    row: {table: interfaces, key: {name: "{{ name | ifname }}"}}
    children:
      - id: if.description
        match: '^description (?P<v>.+)$'
        set: {description: "{{ v }}"}
      - id: if.ipv4
        match: '^ip address (?P<ip>\S+) (?P<mask>\S+)$'
        set: {ipv4: "{{ ip | with_mask(mask) }}"}
      - id: if.ipv4.secondary
        match: '^ip address (?P<ip>\S+) (?P<mask>\S+) secondary$'
        append: {ipv4_secondary: "{{ ip | with_mask(mask) }}"}
      - id: if.shutdown
        match: '^shutdown$'
        set: {enabled: false}
```

| 項目 | 内容 |
|---|---|
| id | 規則の名前（パックの中で 1 つ）。行の対応と試験に出ます |
| match | 正規表現（Go の RE2）。行の全体に当てるので `^` と `$` を書きます。`(?P<名前>…)` で取った値が変数になります |
| when | 追加の条件（式）。偽なら、当たらなかったことにします |
| row | 行を開きます（`table` と `key`）。同じキーの行があれば、その行に足します |
| set | 開いている行の列に値を入れます。空になった値は入れません |
| append | 複数の値を持つ列に値を足します（空白とカンマで区切って、その数だけ） |
| emit | 別の表に行を作ります（`table`・`key`・`set`・`append`） |
| children | 子の行に当てる規則 |
| children_from | 別の規則の ID。その規則の `children` を、ここでも使います |
| ignore | 理由の文。この行を「無視」にします（子の行の中で使います） |

当て方の決まり:

- 照合する文面は、行頭の字下げと行末の空白を除き、連続した空白を 1 つにしたものです
- 同じ階層の規則は、ファイル名の順 → 書いた順に試し、**最初に当たった 1 つ**を使います。限定的な規則を先に書きます
- どの規則にも当たらない行は「未対応」です。その子もすべて「未対応」です
- 値は、入れるときに列の型の書き方に直します（`01500` → `1500`、`Gi0/1` → 正式な IF 名）。型に合わない値の行は、何も入れずに「未対応」になります
- `row` のキーには、機器（ホスト名）が自動で入ります
- `match` だけで値を入れない規則が書けるのは、子の行を持つ入れ物の行（`address-family ipv4` など）と、「列が空であること」と同じ意味の行（`no ip address` など）だけです。README に一覧を書きます

式（`{{ }}` の中と `when`）は Jinja2 の書き方です。取り出した名前のほか、`parent`（直近の親の規則で取った値）、`row`（開いている行）、`device`（ホスト名）が使えます。

### 4.1 無視する行（ignore.yaml）

```yaml
ignore:
  - {match: '^Building configuration', reason: 表示上の行}
  - {match: '^version \d', reason: 機器が自動で付ける行}
  - {match: '^end$', reason: 終わりの印}
```

入れてよいのは、表示上の行・機器が自動で付ける行・終わりの印だけです。

### 4.2 分かれて書かれた設定をまとめる（derive）

1 つの設定がいくつかの場所に分かれているメーカーでは、いったん作業用の表（名前が `x_` で始まる。列は自由）に入れ、すべての行を読み終えてからまとめます。

```yaml
derive:
  - id: nat.pool.join
    for: x_nat_use                # 作業用の表の 1 行ずつを u として回す
    as: u
    let:
      pool: "lookup('x_nat_pool', name=u.pool)"
    when: "pool.start"
    row: {table: nat_rules, key: {name: "{{ u.name }}"}}
    set: {pool_start: "{{ pool.start }}", pool_end: "{{ pool.end }}"}
```

- `lookup(表, 列=値, …)` は 1 行、`first(表, …)` は最初の 1 行、`rows(表, …)` はすべての行を返します。`covering(表, 列, アドレス)` は、アドレスを含むいちばん細かいネットワークの行を返します
- **どの derive にも使われなかった作業用の行は、元の行が「未対応」に戻ります。** 使われなかった値も同じです。作業用の表に入れただけでは「取り込み済み」になりません
- `remove: true` の derive は、別の表に写し終えた行を表から外します（例: IPsec が通信を選ぶためだけに使うアクセスリスト）。`when` に「写し終えたこと」と「ほかの設定が使っていないこと」を必ず書きます

### 4.3 既定値（defaults.yaml）

```yaml
defaults:
  - {table: interfaces, column: enabled, value: true}
  - {table: ospf_interfaces, column: hello, value: 10, when: "row.network_type in ['broadcast', '']"}
```

行が無いときの機器の動きを書きます。取り込みでは空の列を埋め、書き出しでは「明示」の列が空のときに使います。**コマンドリファレンスで確かめた値だけ**を書きます。

### 4.4 語の対応（lexicon.yaml）

```yaml
ike_encryption:          # 種類の名前
  aes256: "aes 256"      # 中立の語: メーカーの語
  aes128: aes
```

テンプレートでは `{{ r.ike_encryption | lex('ike_encryption') }}`（中立 → メーカー）、取り込みでは `{{ v | unlex('ike_encryption') }}`（メーカー → 中立）と書きます。対応が無い語は、取り込みでは「未対応」、書き出しではエラー（G-603）になります。

## 5. 書き出し（render.yaml と templates/）

```yaml
# render.yaml
sections:
  - {id: system,     template: system.j2}
  - {id: interfaces, template: interfaces.j2}
  - {id: extra,      slot: extra}       # 追加コンフィグが入る場所
  - {id: footer,     template: footer.j2}
```

```jinja
{# templates/interfaces.j2 #}
{% for i in t.interfaces %}
interface {{ i.name }}
{% if i.description %}
 description {{ i.description }}
{% endif %}
{% if i.ipv4 is cidr %}
 ip address {{ i.ipv4 | ip }} {{ i.ipv4 | mask }}
{% endif %}
{% if i.enabled == false %}
 shutdown
{% endif %}
!
{% endfor %}
```

テンプレートで使える値: `device`（機器の行）、`t.表の名前`（その機器の行の並び）、`all.表の名前`（全機器の行。相手の値を引くとき）、`project`（案件の設定）。

| フィルタ・関数 | 内容 |
|---|---|
| ip / mask / wildcard / prefixlen / network / broadcast | `192.0.2.1/30` から、アドレス・マスク・ワイルドカード・長さ・ネットワーク・ブロードキャスト |
| with_mask(m) / with_wildcard(w) | アドレスとマスク（ワイルドカード）から `アドレス/長さ` を作る |
| ifname / ifshort | IF 名を正式な形／省略形にする |
| lex(種類) / unlex(種類) | 語の対応 |
| required(文) | 空ならエラーにする（G-606）。例 `{{ r.outside_if \| required('外側 IF') }}` |
| fail(文) | エラーにして止める（G-601）。文には、理由と直し方を書きます |
| is cidr / is digits | 値が「アドレス/長さ」の形か／数字だけか |

書くときの決まり:

- ブロック（`{% … %}`）は 1 行に 1 つ、単独で書きます（その行は出力に残りません）
- 真偽の値をそのまま出しません（`True` と出ます）。`{{ 'no ' if not x }}` のように文字にします
- 文字の加工は、メソッド（`.split()`）ではなくフィルタ（`| split`）で書きます
- 割り算 `/` は小数になります。整数にするなら `//` を使います
- `default` フィルタは使いません（空の値を黙って別の値にするため）

**読まなかった値は、G-602 として指摘されます。** 表に値があるのに、どのテンプレートも読まなかったセルです。別メーカーから来た値を、このパックが書けないときに出ます。「このメーカーには無い設定」と宣言して黙らせる仕組みは、わざと作っていません。

### 5.1 このメーカーの書き方への置き換え（adapt.yaml）

表にある設定を、このメーカーが**別の形でなら書ける**とき、パックが書き出しの前に行を置き換えられます。置き換えたことは、必ず G-614（警告）で知らせます。

```yaml
adapt:
  - id: qos.remaining
    for: qos_policies
    as: p
    when: "p.bandwidth_remaining"
    row: {table: qos_policies, key: {policy: "{{ p.policy }}", class: "{{ p.class }}"}}
    set: {bandwidth_percent: "{{ p.bandwidth_remaining }}"}
    clear: [bandwidth_remaining]
    note: "「帯域（残りの %）」はこの機種に無いので、「帯域（%）」として書きます"
```

- 書き方は derive と同じです。足したのは `clear`（`for` の行の列を空にする）と `note`（知らせる文。必須。空の文になった回は知らせません）
- 扱えるのは設計データの表だけです。パラメータブックは変えません。テンプレートと往復の確認は、置き換えた後の行を見ます
- **書けない値を黙って落とすためには使いません。** 動きが変わる置き換えは、使う人が「それでよい」と言ったものだけにします

## 6. 試験

```yaml
# tests/import/basic/expect.yaml
tables:
  interfaces:
    - {name: GigabitEthernet0/0, description: sample wan, ipv4: 192.0.2.1/30, enabled: true}
unknown:                 # 「未対応」になるはずの行（これ以外が未対応になると不合格）
  - "ip sla 10"
```

```yaml
# tests/render/basic/data.yaml
tables:
  devices:
    - {hostname: rt-a01, vendor: sample_os}
  interfaces:
    - {name: GigabitEthernet0/0, ipv4: 192.0.2.1/30, enabled: true}
findings: []             # 出るはずの指摘の ID（例 [G-601]）。書かなければ「指摘なし」を期待
```

- 取り込みの試験は、`input.txt` を読んで、`expect.yaml` に書いた行と値があること、未対応の行が一覧と同じであることを確かめます
- 書き出しの試験は、`data.yaml` から作った文面が `expect.txt` と同じであること、読み直した結果が表と同じであることを確かめます
- うまく書けないときの形（`fail` で止まること）も、試験にしておきます

## 7. パックの検査

`pack test` と起動のときに、次を確かめます。誤りがあるパックは読み込みません。

- YAML として読める。正規表現が読める（`^` `$` の無い規則は警告）
- 規則・derive・既定値が指す表と列が、定義にある。キーの列がその表のキーである。複数の値の列に `append`、1 つの値の列に `set` を使っている
- 決まった値（`set: {speed: auto}`）と既定値が、列の型に合う
- `lex`・`unlex` の種類が lexicon.yaml にある。メーカーの語が重なっていない
- 規則と derive の ID が重なっていない
- 入れる規則があるのに、どこからも使われない作業用の表がない

## 8. README に書くこと

使う人が「取り込み済み」の意味を確かめられるように、次を書きます。

- 対応の一覧（読める行と、入る表）。取り込みだけで書き出せないものは、そう書く
- 参照した資料の名前と版、確かめた機種と版
- **値を入れない規則**と、その理由
- **読んで、表には入れない語**（行は「取り込み済み」になるが、行の一部を入れる列が無いもの）
- **読まない行**（見本で「未対応」になる行と、その理由）
- **確かめていないこと**（資料に決まりが無かった点。実機の出力が手に入ったら、ここから確かめる）
