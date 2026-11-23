# メーカーパックの書き方

ConfigChief の本体は、メーカーを知りません。コンフィグの読み方・書き方・語の対応は、すべて「メーカーパック」というフォルダ（YAML とテンプレートのテキスト）にあります。新しいメーカーや新しい設定は、パックを足す・直すことで対応します。プログラムは直しません。

同梱のパック（`internal/packs/cisco_ios`・`internal/packs/nec_ix`・`internal/packs/nec_qx`）が、いちばんの見本です。このページと並べて読んでください。

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
  models.yaml      機種とポートの並び（IF 対応の案に使う。無くてもよい）
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
- **区切りの行で階層を閉じる**: `comment_closes: true` と書くと、`comment_prefixes` で始まる行が、開いている階層をすべて閉じます（区切りの次の行は、字下げがあっても一番上の階層）。設定の部分ごとに `#` の行で区切り、機器全体の 1 行の設定を 1 字下げて保存するメーカー（同梱の `nec_qx`）のためのものです。字下げのある区切りの行（` #`）は、その字下げと同じか、それより深い階層だけを閉じます。書かなければ、区切りの行は階層に関わりません
- **IF 名**: フィルタ `ifname` が省略形を正式な名前に直します。`Null`・`on`・`yes`・`no` のような名前は、YAML が別の値として読むので引用符で囲みます（`name: "Null"`）
- **端末ログ**: `show_config` は行の終わり（`$`）まで書きます（`show running-config interface …` のような一部の表示を、全体として拾わないため）。`pager` は跡そのものだけを書きます
- **変更の投入**（`change`。書かなくてもよい）: 「ひな形から作る」で、今のコンフィグとの差から投入を作るときの書き方です（使い方 8 章）。書かないパックの機器には、差の投入を作りません

```yaml
change:
  begin: [configure terminal]     # 設定モードに入る行
  end: [end]                      # 設定モードから出る行
  exit: exit                      # ブロック（interface など）を 1 段抜ける行
  exits:                          # 別の行で抜けるブロック（最初に当たったもの）
    - {parent: '^address-family ', exit: exit-address-family}
  negate: "no "                   # 行を消すとき前に付ける語
  unnegate: true                  # 「no …」の行は「no」を外して消す（no shutdown → shutdown）
  forms:                          # 別の形で消す行（最初に当たったもの。$1… は当たった所）
    - {match: '^description ', remove: 'no description'}
```

- `begin`・`end`・`exit` は、メーカーの公開資料で確かめた行だけを書きます。保存のコマンドは書きません（保存するかは人が決めます）

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
- `each: {名前: 式}` を書くと、式が返す並びの 1 つずつについて行を作ります（1 行から複数の行。例: `each: {p: "g.ports | split(' ')"}`）。並びの中身は `名前`、1 から数えた順番は `名前_n` で読めます。`let` は先に 1 度だけ実行されます（`let` から `名前` は読めません）。書けるのは 1 組だけで、`remove` の手順には書けません
- 値を並べる列を `列=値` で引くと、並びの中にその値を持つ行に当たります（`rows('tracks', members='5')`）。行が無いときに `lookup(…).列` を式の中でじかに読むと誤りになるので、`let` に受けるか、`rows(…) | length` で数えます
- `let` の値が空（行が見つからない）なら、その列を読んでも空の文字です。`when` があって `each` が無い手順では、`when` が読む `let` だけが先に評価され、`when` が偽ならそこで終わります
- **`only_if: 式`**: その手順を、機器ごとに 1 度だけ聞き、偽ならその機器ではまるごと飛ばします（`for` の表の行を 1 つも見ません）。行の多い表（アクセスリストの行など）を回る手順を、関係する設定の無い機器で動かさないための、速さのための指定です。例: `only_if: "rows('x_crypto_map') or rows('x_tunnel_protection')"`。式から見えるのは `device` と、行を引く関数（`rows`・`lookup`・`first`・`covering`）だけです。**手順が動く機器での結果を変えてはいけません**（`when` が必ず偽になる機器を先に見分けるだけで、`when` の代わりではありません）。derive にも adapt にも書けます
- 決まった書き方でしか意味を持たない行（Cisco の `ip sla schedule 1 life forever start-time now` など）は、作業用の表の列に入れて `when` で読みます。その行が欠けた設定では行を作らず、関係する行がすべて「未対応」に戻ります

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
| host(n) / in_net(n) | n 番目のアドレス（`192.0.2.0/30 \| host(2)` → `192.0.2.2`）／ネットワークに含まれるか |
| version_ge(版) | 版の比較（`device.os_version \| version_ge('16.9')`） |
| ifname / ifshort | IF 名を正式な形／省略形にする |
| natural / natural(attribute='列') | 番号を数として並べる／行の並びをその列の値で並べる |
| ranges | `10,20,30-32` の形の文字（またはその並び）を、番号の並びにする |
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

- 書き方は derive と同じです（`for`・`as`・`let`・`when`・`each`・`only_if`・`row`・`set`・`append`・`remove`）。adapt だけの項目は次のとおりです
  - `note`: 知らせる文（必須）。空の文になった回は知らせません（`note: "{{ '' }}"`）。知らせずに置き換えてよいのは、置き換えた後も設定の意味が何も変わらず、人が確かめることも無い場合だけです
  - `clear: [列, …]`: `for` の行の列を空にします。`row` も `remove` も無い手順は、`clear` だけ、または知らせるだけの手順になります
  - `replace: {列: 値}`: `row` の行の、値を並べる列の中身を入れ替えます（区切りは `append` と同じ。空の値なら列が空になります）。並びの中の 1 つの名前を付け替えるときに使います。`row` のある手順にだけ書けます
  - `drop: {table: 表, key: {…}}`: 手順の終わりに、その行を設計データから外します（行が無ければ何もしません）。`row` も `remove` も無い手順にだけ書けます。`each` で並びを回しながら、並びの名前が指す行を外すときに使います。外すのは、別の行に写し終えて、どこからも指されていない行だけです（`when` に書きます）
  - `remove: true`: `for` の行を外します（derive と同じ）
- 置き換えができない組み合わせは、式の中で `fail('文')` を呼んで知らせます（その手順は行を作らず、文が G-601 のエラーになります）
- 見えるのは、その機器の行（`*`・`@グループ`・共通値を当てた後）だけです。作業用の表は使えません
- 値のチェック（V）と対向のチェック（X）は、置き換える前の行（パラメータブックの値）を見ます。テンプレート・G-602・往復の確認は、置き換えた後の行を見ます
- 足した行は表の終わりに付きます。並びが要るテンプレートは `t.interfaces | natural(attribute='name')` のように自分で並べます
- 扱えるのは設計データの表だけです。パラメータブックは変えません。テンプレートと往復の確認は、置き換えた後の行を見ます
- **書けない値を黙って落とすためには使いません。** 動きが変わる置き換えは、使う人が「それでよい」と言ったものだけにします

### 5.2 機器が暗号化して保存した値

機器が暗号化して保存した秘密の値（Cisco の `7 <値>`・`6 <値>`、NEC IX の `secret <値>`、NEC QX の `cipher <値>` など）は、機器が出したとおりに「種別 値」と空白で区切って、秘密の値の列に入れます（「形」の列は足しません。「ユーザ」と「機器」の表だけは、値を入れて「パスワードの形」を encrypted にします）。平文の値に空白は入らないので、空白のある値は暗号化した形だと分かります。

- 取り込み: 機器の形のまま、空白を含めて列に入れます。平文に直しません
- 書き出し: **自分のメーカーの形で、取り込みが読める列だけ**を、そのまま書き戻します
- ほかの形（別メーカーの形、自分の形でも書き戻せない列）は、adapt の手順で `fail` を呼んで止めます（G-601。「平文を入れ直してください」と知らせます）。例:

```yaml
  - id: secret.form.ospf
    for: ospf_interfaces
    as: x
    when: "' ' in x.auth_key"
    note: "{{ fail(x.if_name ~ ' の OSPF の鍵は、機器が暗号化した形です（平文を入れ直してください）') }}"
```

### 5.3 表の定義で、知っておくこと

表と列の定義はプログラムの側にあり、パックからは変えません（[tables.md](tables.md)）。規則とテンプレートを書くときに効く指定は次のとおりです。

| 列の指定 | 意味 |
|---|---|
| `auto_name` | 取り込みでパックが振る名前（コンフィグに書かれていない名前。NAT の規則・PPPoE・IPsec の行の名前など）。往復の確認と比較は、名前ではなく中身で行を突き合わせます。その名前を指すほかの表の行も、指している行が同じなら同じとみなします。だから、パックは別メーカーの形の名前の行を、自分の決まりの名前で書いてかまいません |
| `order_only` | 行の順番を決めるだけの番号（IPsec の追加の提案の順番）。コンフィグに番号そのものが無いので、往復の確認と比較は「何番目か」で行を突き合わせます。書き出しは、番号の小さい順に並べるだけで、番号は書きません |
| `compare: false` | 比較で見ない列（人だけが書く列。インタフェースの対向の機器・IF）。コンフィグからは分からないので、取り込みの規則は書きません |
| `generate: false` | 生成に使わない列（管理のための列）。G-602 と往復の確認の対象になりません |
| 「明示」の列 | 空のとき、書き出し先のパックの既定値（`defaults.yaml`）を埋めて必ず書く列 |

## 6. 機種の定義（models.yaml）と IF 対応の案

`models.yaml` に機種のポートの並びを書くと、「機器対応」の「新機種」にその ID を書いた機器について、IF 対応の新 IF 名の案を、機種のポートから作ります。無くてもパックは動きます（案が入らないだけです）。

```yaml
models:
  - id: sample-2wan-8sw            # 機種の ID。製品の名前ではなく、ポートの並びを言う名前にする
    title: {ja: "WAN×2 ＋ スイッチ×8", en: "2 WAN ports and an 8-port switch"}
    family: ios-xe                 # 書かなくてもよい（families を書いたパックだけ）
    ports:                         # 割り当てに使う順に並べる
      - {name: GigabitEthernet0/0/0, role: wan}
      - {name: GigabitEthernet0/0/1, role: wan}
      - {range: "GigabitEthernet0/1/[0-7]", role: switch}
    logical:
      loopback: "Loopback{n}"
      tunnel:   "Tunnel{n}"
      subif:    "{parent}.{vlan}"
      pppoe:    "Dialer{i}"
      vlan:     "Vlan{vlan}"
      lag:      "Port-channel{n}"
    other_names: []                # ほかに、この機種にあってよい IF の名前の形（チェック V-111）
    unsupported:                   # この機種に無い機能（チェック G-607）。資料で確かめたものだけ
      - {name: 内蔵スイッチ, table: switchports, when: "row.mode != 'routed'"}
```

- `ports`: `name` は 1 つのポート、`range` は終わりの `[a-b]` を番号に広げた並び。`role` は `wan`・`lan`・`switch`（内蔵スイッチのポート）
- ポートの指定 `sub`: サブ IF の親としての名前（NEC IX の `GigaEthernet0.0` のサブ IF は `GigaEthernet0.1` なので `sub: GigaEthernet0`）。`vlan_if`: 内蔵スイッチのポートで、そのポートがいちばん若いアクセスポートである VLAN の IF の名前（`range` では `{p}` がポートの番号。NEC IX の `"GigaEthernet2:{p}.0"`）
- `logical` の印: `{n}`（旧 IF 名の番号）、`{parent}`（親のポートの新しい名前）、`{vlan}`（VLAN の番号）、`{i}`（1 から、まだ使っていない番号）、`{j}`（0 から、まだ使っていない番号）。印の値が決まらないときは、案を出しません
- テンプレートからは `model.id`・`model.family`・`model.ports`・`model.limits.名前` で読めます。機種が空か定義に無い機器では、どれも空です。機種が分かるときだけ確かめる書き方にします（`{% if model.ports %}`）
- 機器の「機種」がこの ID の機器では、インタフェースの表の IF 名を確かめます（チェック V-111）。当たってよいのは、`ports` のポートと `vlan_if`、`logical` と `other_names` の名前の形、`pack.yaml` の `interfaces.types` で種類が physical でない IF（ループバック・トンネル・VLAN の IF など）、それらのサブ IF です。ほかに、その機種だけの名前（内蔵スイッチの基本 IF など）があれば `other_names` に書きます（印は `{n}` = 数字の並び、`{parent}` = ポートの `sub` か名前）
- `unsupported`: その機種に無い機能。`name`（指摘に出す名前）・`table`（設計データの表）・`when`（行ごとの式。`row.列名` で値を読む。無ければすべての行）。生成のとき、当たる行があればエラー（G-607）にします。書いていない機種では G-607 は「未実施」です。資料で「無い」と確かめた機能だけを書きます（使える設定を止めないため）
- 機種が空か、定義に無い機器は、情報（V-114）を出すだけです。定義の ID に似た打ち間違い（英字と数字だけで 2 字以内の違い）は警告（V-116）にします
- ID の重なり、空の `ports`、`range` の形、知らない `role`・`logical` の名前、`other_names` の知らない印、`unsupported` の空の `name`・`table` は、パックの検査で誤りになります。`unsupported` の知らない表と読めない `when` は、パックの試験（`ConfigChief pack test <フォルダ>`）で誤りになります
- 同梱のパックの機種は、それぞれの README の「機種の定義」の節にあります。案件フォルダのパックに `models.yaml` を置けば、機種を足せます

案の作り方:

1. 物理ポート: 旧機器の使っているポート（「有効」が無効でないもの）を名前の順に並べ、新しい機種の `ports` の順に割り当てます。旧機器のポートは「内蔵スイッチのポート」と「L3 のポート」に分け、スイッチのポートは `role: switch` のポートへ、L3 のポートはそれ以外へ割り当てます。新しい機種に同じ名前のポートがあれば、その名前のままにします。無効のポートは `-`（持っていかない）です。新しい機種に `role: switch` のポートしか無いとき（スイッチ本体）は、分けずに全部を名前の順に割り当てます
2. PPPoE の論理 IF・サブ IF・ループバック・トンネル・LAG の IF は `logical` の形で作ります
3. VLAN の IF は、その VLAN がどのトランクにも通っておらず、アクセスポートがあり、その新しいポートのうちいちばん若いものに `vlan_if` があれば、その名前。そうでなければ `logical.vlan` の形です
4. ポートが足りない、当たる形が無いときは、空のままにします

## 7. 試験

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

## 8. パックの検査

`pack test` と起動のときに、次を確かめます。誤りがあるパックは読み込みません。

- YAML として読める。正規表現が読める（`^` `$` の無い規則は警告）
- 規則・derive・既定値が指す表と列が、定義にある。キーの列がその表のキーである。複数の値の列に `append`、1 つの値の列に `set` を使っている
- 決まった値（`set: {speed: auto}`）と既定値が、列の型に合う
- `lex`・`unlex` の種類が lexicon.yaml にある。メーカーの語が重なっていない
- 規則と derive の ID が重なっていない
- 入れる規則があるのに、どこからも使われない作業用の表がない

## 9. README に書くこと

使う人が「取り込み済み」の意味を確かめられるように、次を書きます。

- 対応の一覧（読める行と、入る表）。取り込みだけで書き出せないものは、そう書く
- 参照した資料の名前と版、確かめた機種と版
- **値を入れない規則**と、その理由
- **読んで、表には入れない語**（行は「取り込み済み」になるが、行の一部を入れる列が無いもの）
- **読まない行**（見本で「未対応」になる行と、その理由）
- **確かめていないこと**（資料に決まりが無かった点。実機の出力が手に入ったら、ここから確かめる）
