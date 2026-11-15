"use strict";

// ConfigChief: the page. It draws what the tool gives (internal/ui) and
// passes on what the user asks for. Every text reaches the page as a text
// node: nothing here builds HTML from a string.

(function () {
  const ROWS = 500; // rows of a list drawn at once
  const LINES = 3000; // lines of a file drawn at once

  const SRC = { old: "旧コンフィグ", new: "新コンフィグ", book: "パラメータブック" };

  const st = {
    version: "",
    cas: null, // the open case
    recent: [],
    page: "cases",
    error: "",
    jump: "", // the id of what the next drawing scrolls to
    home: null,
    create: { dir: "", name: "", kind: "migration" },
    imp: { source: "old", views: {}, tab: {}, lines: {}, pick: {}, onlyBad: false },
    jdg: newJdg(),
    map: { view: null, tab: "open", book: null, replace: false },
    cmp: { a: "old", b: "new", view: null, tab: 0 },
    diff: { a: "", b: "", view: null, shown: LINES, mode: "line", stamps: true, secrets: true },
    tree: newTree(),
    sites: { source: "book", view: null, tab: 1 },
    res: newRes(),
    mig: newMig(),
    cus: newCus(),
    logs: newLogs(),
    exp: { mask: "show", view: null },
    nb: { from: "", to: "", said: [] },
    filters: {},
  };

  // newJdg is the screen of the unhandled lines with nothing to show. changed
  // tells the sides that were judged after their last import.
  function newJdg() {
    return { source: "old", views: {}, sel: {}, changed: {}, only: true, scope: "all", picked: {}, verdict: "manual", reason: "", by: null, carry: {} };
  }

  // newMig is what the lists of a comparison and of the mapping keep about
  // the findings of a migration: the finding that is picked and the approval
  // that is being written.
  function newMig() {
    return { sel: "", scope: "", reason: "", by: null, msg: "" };
  }

  // newCus is the screen of the lists of the customer with nothing to show:
  // the comparison (view, the list that is shown) and the column mappings
  // (maps, the one that is being written).
  function newCus() {
    return { tab: "compare", view: null, vtab: 1, maps: null, edit: null, create: null, msg: "" };
  }

  // newTree is the comparison of every device with nothing to show.
  function newTree() {
    return { a: "old", b: "new", dirs: { a: "", b: "" }, stamps: true, secrets: true, view: null, one: null, shown: LINES };
  }

  // newLogs is the page of the logs with nothing to show.
  function newLogs() {
    return { tab: "load", views: {}, vtab: {} };
  }

  // newRes is the result screen with nothing to show.
  function newRes() {
    return { view: null, tab: "devices", ctab: 0, device: "", sev: "", state: "", sel: "", onlyBad: false, scope: "", reason: "", by: null };
  }

  const $ = (id) => document.getElementById(id);
  const api = () => window.go.main.App;
  const errText = (e) => (typeof e === "string" ? e : (e && e.message) || String(e));
  const cls = (...names) => names.filter(Boolean).join(" ");

  // h makes an element. Children that are strings become text nodes.
  function h(tag, props, ...kids) {
    const el = document.createElement(tag);
    if (props) {
      for (const k in props) {
        const v = props[k];
        if (v === false || v == null) continue;
        if (k === "class") el.className = v;
        else if (k.startsWith("on")) el.addEventListener(k.slice(2), v);
        else if (k === "disabled" || k === "checked" || k === "value" || k === "hidden") el[k] = v;
        else el.setAttribute(k, v === true ? "" : v);
      }
    }
    add(el, kids);
    return el;
  }

  function add(el, kids) {
    for (const k of kids) {
      if (k == null || k === false) continue;
      if (Array.isArray(k)) add(el, k);
      else el.appendChild(k instanceof Node ? k : document.createTextNode(String(k)));
    }
  }

  const button = (label, onclick, props) => h("button", Object.assign({ type: "button", onclick }, props), label);

  function select(options, value, onchange) {
    const el = h("select", { onchange: (e) => onchange(e.target.value) }, options.map((o) => h("option", { value: o[0] }, o[1])));
    el.value = value;
    return el;
  }

  // seg is a row of buttons of which one is on.
  function seg(options, value, onpick) {
    return h("span", { class: "seg" }, options.map((o) => button(o[1], () => onpick(o[0]), { class: o[0] === value ? "on" : null })));
  }

  function boxes(kind, list) {
    return list && list.length ? h("div", { class: "box " + kind }, list.join("\n")) : null;
  }

  function counts(list) {
    return h("div", { class: "counts" }, list.map((p) => {
      const bad = /未判定|保留|M-402|未承認のエラー|そろわなかった|生成していない/.test(p.label) && !/^0(\D|$)/.test(p.value);
      return h("div", { class: cls("count", bad && "bad") }, h("div", { class: "n" }, p.value), h("div", { class: "l" }, p.label));
    }));
  }

  // ---- what lies over the page ----

  let busy = false;
  let busyTimer = 0;
  let busyUnit = "ファイル"; // what the running step counts
  let afterDraw = []; // what the next drawing does once the page is in place

  function showBusy(label, canStop) {
    $("busy-label").textContent = label;
    $("busy-text").textContent = "";
    $("busy-bar").style.width = "0";
    $("busy-stop").hidden = !canStop;
    $("busy-stop").disabled = false;
    $("busy").dataset.running = "1";
    // A step that is over at once does not flash the dialog.
    busyTimer = setTimeout(() => { $("busy").hidden = false; }, 200);
  }

  function hideBusy() {
    clearTimeout(busyTimer);
    delete $("busy").dataset.running;
    $("busy").hidden = true;
  }

  function onProgress(p) {
    if (!busy || !p) return;
    // A step that does not know how many there are tells the file alone.
    $("busy-text").textContent = p.total ? `${p.done} / ${p.total} ${busyUnit}: ${p.file}` : p.file;
    $("busy-bar").style.width = p.total ? `${Math.round((p.done / p.total) * 100)}%` : "0";
  }

  // run does one step: the page waits for it, and what goes wrong is shown.
  // unit is what its progress counts (files, if it is not given).
  async function run(label, fn, canStop, unit) {
    if (busy) return;
    busy = true;
    busyUnit = unit || "ファイル";
    st.error = "";
    showBusy(label, canStop);
    try {
      await fn();
    } catch (e) {
      st.error = errText(e);
    } finally {
      busy = false;
      hideBusy();
      render();
    }
  }

  function ask(title, text, yes) {
    return new Promise((resolve) => {
      $("ask-title").textContent = title;
      $("ask-text").textContent = text;
      $("ask-yes").textContent = yes;
      $("ask").hidden = false;
      const done = (answer) => {
        $("ask").hidden = true;
        $("ask-yes").onclick = null;
        $("ask-no").onclick = null;
        resolve(answer);
      };
      $("ask-yes").onclick = () => done(true);
      $("ask-no").onclick = () => done(false);
    });
  }

  function show(rel) {
    api().Show(rel).catch((e) => {
      st.error = errText(e);
      render();
    });
  }

  // ---- the frame ----

  function setState(s) {
    st.version = s.version;
    st.cas = s.case;
    st.recent = s.recent;
  }

  // forgetViews drops what was drawn for the case that was open before.
  function forgetViews() {
    st.imp = { source: "old", views: {}, tab: {}, lines: {}, pick: {}, onlyBad: false };
    st.jdg = newJdg();
    st.map = { view: null, tab: "open", book: null, replace: false };
    st.cmp = { a: "old", b: "new", view: null, tab: 0 };
    st.tree = newTree();
    st.sites = { source: "book", view: null, tab: 1 };
    st.res = newRes();
    st.cus = newCus();
    st.logs = newLogs();
    st.exp = { mask: "show", view: null };
    st.nb = { from: "", to: "", said: [] };
    st.filters = {};
  }

  async function opened(home) {
    forgetViews();
    st.home = home;
    setState((await api().Start()).state);
    st.page = "home";
  }

  function go(page) {
    if (page === "home") {
      run("案件を読んでいます", async () => {
        st.home = await api().Home();
        st.page = "home";
      });
      return;
    }
    st.page = page;
    st.error = "";
    if (page === "mapping" && !st.map.view) {
      loadMapping();
      return;
    }
    if (page === "customer" && st.cus.tab === "maps" && !st.cus.maps) {
      loadMaps();
      return;
    }
    if (page === "judge" && !st.jdg.views[st.jdg.source]) {
      loadJudge();
      return;
    }
    render();
  }

  function drawNav() {
    const nav = $("nav");
    nav.textContent = "";
    const item = (page, label, needsCase) => button(label, () => go(page), { class: st.page === page ? "on" : null, disabled: needsCase && !st.cas });
    add(nav, [
      h("div", { class: "brand" }, "ConfigChief"),
      h("div", { class: "nav-case" }, st.cas ? st.cas.name : "案件を開いていません"),
      item("cases", "案件", false),
      item("home", "ホーム", true),
      item("import", "取り込み", true),
      item("judge", "未対応の行", true),
      item("mapping", "対応", true),
      item("customer", "お客様の一覧", true),
      item("result", "結果", true),
      item("logs", "ログ", true),
      item("compare", "比較", false),
      h("div", { class: "nav-foot" }, "版 " + st.version),
    ]);
  }

  function render() {
    drawNav();
    const main = $("main");
    const top = main.scrollTop;
    const same = main.dataset.page === st.page;
    main.textContent = "";
    main.dataset.page = st.page;
    if (st.error) {
      main.appendChild(h("div", { class: "box error" }, button("閉じる", () => { st.error = ""; render(); }, { class: "link close" }), st.error));
    }
    const pages = { cases: casesPage, home: homePage, import: importPage, judge: judgePage, mapping: mappingPage, customer: customerPage, result: resultPage, logs: logsPage, compare: comparePage };
    afterDraw = [];
    add(main, [pages[st.page]()]);
    main.scrollTop = same ? top : 0;
    const to = st.jump && $(st.jump);
    st.jump = "";
    if (to) to.scrollIntoView();
    for (const fn of afterDraw) fn();
    afterDraw = [];
  }

  // ---- lists ----

  // tableView draws a list with a filter over all its rows. opts: id (where
  // the filter is kept), onPick(i, tr), selected (the row that is picked).
  function tableView(t, opts) {
    opts = opts || {};
    const key = opts.id || t.name;
    const fs = st.filters[key] || (st.filters[key] = { text: "", shown: ROWS });
    if (!t.rows.length) return h("div", { class: "table-wrap" }, h("div", { class: "empty" }, "ありません"));
    const code = new Set(t.code || []);
    const whole = new Set(t.whole || []); // the columns that are not broken into lines
    const body = h("tbody");
    const note = h("span", { class: "dim small" });
    const more = button("さらに表示", () => { fs.shown += ROWS; fill(); });

    function fill() {
      const words = fs.text.toLowerCase().split(/\s+/).filter(Boolean);
      const hit = [];
      t.rows.forEach((row, i) => {
        if (words.length) {
          const all = row.join("\n").toLowerCase();
          if (!words.every((w) => all.includes(w))) return;
        }
        hit.push(i);
      });
      body.textContent = "";
      for (const i of hit.slice(0, fs.shown)) {
        const mark = (t.marks && t.marks[i]) || "";
        const tr = h("tr", { class: cls(mark && "mark-" + mark, opts.onPick && "pick", opts.selected === i && "sel") });
        t.rows[i].forEach((cell, c) => tr.appendChild(h("td", { class: cls(code.has(c) && "code", whole.has(c) && "whole") || null }, cell)));
        if (opts.onPick) tr.addEventListener("click", () => opts.onPick(i, tr));
        body.appendChild(tr);
      }
      const drawn = Math.min(hit.length, fs.shown);
      note.textContent = hit.length === t.rows.length && drawn === hit.length
        ? `${t.rows.length} 行`
        : `${t.rows.length} 行のうち ${hit.length} 行が当てはまり、${drawn} 行を出しています`;
      more.hidden = hit.length <= fs.shown;
    }

    const filter = h("input", { type: "text", value: fs.text, placeholder: "絞り込み（空白で区切ると、すべてを含む行）", oninput: (e) => { fs.text = e.target.value; fs.shown = ROWS; fill(); } });
    fill();
    return h("div", null,
      t.lead ? h("p", { class: "dim" }, t.lead) : null,
      h("div", { class: "row" }, filter, note, more),
      h("div", { class: "table-wrap" }, h("table", null, h("thead", null, h("tr", null, t.header.map((x) => h("th", null, x)))), body)));
  }

  // ---- 案件 ----

  function casesPage() {
    const c = st.create;
    const pick = async () => {
      try {
        const dir = await api().PickFolder("案件フォルダにするフォルダ（空のフォルダ、または新しく作る）");
        if (dir) c.dir = dir;
      } catch (e) {
        st.error = errText(e);
      }
      render();
    };
    const create = () => run("案件を作っています", async () => {
      await opened(await api().Create(c.dir, c.name, c.kind));
      st.create = { dir: "", name: "", kind: "migration" };
    });
    const open = (dir) => run("案件を開いています", async () => opened(await api().Open(dir)));
    const openPicked = async () => {
      let dir = "";
      try {
        dir = await api().PickFolder("開く案件フォルダ");
      } catch (e) {
        st.error = errText(e);
        render();
      }
      if (dir) open(dir);
    };
    const forget = (dir) => run("一覧を直しています", async () => setState(await api().Forget(dir)));

    return [
      h("h1", null, "案件"),
      h("p", { class: "dim" }, "案件フォルダ 1 つに、コンフィグ・パラメータブック・レポートをまとめて置きます。"),
      h("div", { class: "card" },
        h("h2", null, "案件を開く"),
        h("div", { class: "row" }, button("案件フォルダを選んで開く", openPicked, { class: "primary" })),
        st.recent.length ? h("div", { class: "table-wrap" }, h("table", null,
          h("thead", null, h("tr", null, h("th", null, "最近開いた案件"), h("th", null, "フォルダ"), h("th", null, ""))),
          h("tbody", null, st.recent.map((r) => h("tr", null,
            h("td", null, r.name || "（名前なし）"),
            h("td", { class: "path" }, r.dir, r.missing ? h("div", { class: "dim" }, "案件フォルダが見つかりません") : null),
            h("td", null, h("div", { class: "row" },
              button("開く", () => open(r.dir), { disabled: r.missing }),
              button("一覧から外す", () => forget(r.dir)))))))))
          : h("p", { class: "dim" }, "最近開いた案件はありません。")),
      h("div", { class: "card" },
        h("h2", null, "新しい案件を作る"),
        h("div", { class: "row" }, h("span", null, "フォルダ"), button("選ぶ", pick), h("span", { class: "path" }, c.dir || "（まだ選んでいません）")),
        h("div", { class: "row" }, h("span", null, "案件名"), h("input", { type: "text", value: c.name, placeholder: "空ならフォルダの名前", oninput: (e) => { c.name = e.target.value; } })),
        h("div", { class: "row" }, h("span", null, "種類"),
          seg([["migration", "更改（旧コンフィグから作る）"], ["new", "新規"]], c.kind, (k) => { c.kind = k; render(); })),
        h("div", { class: "row" }, button("作る", create, { class: "primary", disabled: !c.dir }))),
    ];
  }

  // ---- ホーム ----

  function homePage() {
    const m = st.home;
    if (!m) return [h("h1", null, "ホーム"), h("p", { class: "dim" }, "案件を開いてください。")];
    const toImport = (source) => { st.imp.source = source; go("import"); };
    const close = () => run("案件を閉じています", async () => {
      setState(await api().Close());
      forgetViews();
      st.home = null;
      st.page = "cases";
    });
    const exp = st.exp;
    const doExport = (what) => run(what === "sheet" ? "納品用パラメータシートを出しています" : "相手に渡すパラメータ表を出しています", async () => {
      exp.view = await api().Export(what, exp.mask === "mask");
      st.home = await api().Home();
    });
    // A new build: the empty book, and the copies of a device.
    const nb = st.nb;
    const newBook = () => run("空のパラメータブックを作っています", async () => {
      nb.said = await api().NewBook();
      st.home = await api().Home();
    });
    const cloneDevice = () => run("機器を複製しています", async () => {
      nb.said = await api().CloneDevice(nb.from, nb.to);
      nb.to = "";
      st.home = await api().Home();
    });
    const input = (source, n) => h("div", { class: "card" },
      h("h2", null, `${SRC[source]}（input/${source}）`),
      h("p", null, n ? `${n} ファイルあります。` : "まだファイルがありません。フォルダを開いて、コンフィグ（テキスト、端末ログでも可）を置いてください。"),
      h("div", { class: "row" },
        button("フォルダを開く", () => show("input/" + source)),
        button("取り込みへ", () => toImport(source), { class: n ? "primary" : null, disabled: !n })));

    return [
      h("h1", null, m.case.name),
      h("p", null, `種類: ${m.case.kind}`),
      h("div", { class: "row" }, h("span", { class: "path" }, m.case.dir), button("案件フォルダを開く", () => show(""))),
      boxes("warn", m.problems),
      input("old", m.old),
      h("div", { class: "card" },
        h("h2", null, `パラメータブック（${m.case.book}）`),
        h("p", null, m.hasBook ? `あります（最後に書かれた時刻: ${m.bookTime}）。` : "まだありません。旧コンフィグを取り込むと作られます。"),
        h("div", { class: "row" },
          button("Excel で開く", () => show(m.case.book), { disabled: !m.hasBook }),
          button("対応を見る", () => go("mapping")),
          m.hasBook ? null : button("空のブックを作る（新規構築）", newBook, { id: "new-book" })),
        m.hasBook ? h("div", { class: "row", id: "clone-row" },
          h("span", null, "機器の複製: "),
          h("input", { type: "text", id: "clone-from", value: nb.from, placeholder: "元のホスト名", oninput: (e) => { nb.from = e.target.value; } }),
          h("span", null, " → "),
          h("input", { type: "text", id: "clone-to", value: nb.to, placeholder: "新しいホスト名（カンマで区切って複数）", oninput: (e) => { nb.to = e.target.value; } }),
          button("複製する", cloneDevice, { id: "clone-go" })) : null,
        nb.said.length ? h("div", { id: "newbuild-result" }, nb.said.map((x) => h("p", null, x))) : null),
      input("new", m.new),
      h("div", { class: "card" },
        h("h2", null, "比較"),
        h("p", null, "旧コンフィグ・新コンフィグ・パラメータブックのうち 2 つを、設計データとして比べます。"),
        h("div", { class: "row" }, button("比較へ", () => go("compare")))),
      h("div", { class: "card" },
        h("h2", null, "生成とチェック"),
        h("p", null, "パラメータブックを読んでチェックし、全台のコンフィグ・チェック結果表・変更一覧（2 回目から）を出力フォルダに書きます。値の誤りがある機器のコンフィグは書きません。"),
        h("p", { class: "dim" }, !m.hasBook ? "パラメータブックがまだありません。" : m.lastGen ? `最後の生成: ${m.lastGen}` : "まだ生成していません。"),
        h("div", { class: "row" },
          button("生成とチェック", () => generate(false), { class: "primary", disabled: !m.hasBook }),
          button("チェックだけ", () => generate(true), { disabled: !m.hasBook }),
          st.res.view ? button("結果を見る", () => go("result")) : null)),
      h("div", { class: "card" },
        h("h2", null, "納品物"),
        h("p", null, "パラメータブックの値を、納品用パラメータシート（templates フォルダのひな形に流し込みます。ひな形が無ければ標準の形）と、相手に渡すパラメータ表（管理外の相手と合わせる値）に書き出します。チェックはしません。"),
        h("div", { class: "row" },
          h("span", null, "秘密の値: "),
          select([["show", "そのまま書く"], ["mask", "*** に伏せる"]], exp.mask, (v) => { exp.mask = v; })),
        h("div", { class: "row" },
          button("納品用パラメータシートを出す", () => doExport("sheet"), { disabled: !m.hasBook }),
          button("相手に渡すパラメータ表を出す", () => doExport("peer"), { disabled: !m.hasBook }),
          button("ひな形のフォルダを開く", () => show("templates"))),
        exp.view ? h("div", { id: "export-result" },
          h("p", null, exp.view.summary),
          boxes("warn", exp.view.notes),
          boxes("warn", exp.view.problems),
          h("div", { class: "row" },
            exp.view.files.map((f) => button(f.slice(f.lastIndexOf("/") + 1), () => show(f))),
            exp.view.dir ? button("フォルダを開く", () => show(exp.view.dir)) : null)) : null),
      h("div", { class: "card" },
        h("h2", null, "レポート"),
        h("p", null, m.lastOut ? `いちばん新しい出力: ${m.lastOut}` : "まだ出力はありません。"),
        h("div", { class: "row" }, button("出力フォルダを開く", () => show(m.lastOut), { disabled: !m.lastOut }))),
      h("div", { class: "card" },
        h("h2", null, "使えるベンダーパック"),
        h("p", null, m.packs.length ? m.packs.map((p) => `${p.id} ${p.version}`).join("、") : "ありません。")),
      h("div", { class: "row" }, button("案件を閉じる", close)),
    ];
  }

  // ---- 取り込み ----

  function importPage() {
    const im = st.imp;
    const s = im.source;
    const v = im.views[s];
    const doImport = () => run(`${SRC[s]}を取り込んでいます`, async () => {
      im.views[s] = await api().Import(s);
      delete im.lines[s];
      im.tab[s] = "files";
      // The files may have changed: the unhandled lines are read anew.
      delete st.jdg.views[s];
      delete st.jdg.changed[s];
    }, true);
    const toJudge = () => { st.jdg.source = s; go("judge"); };
    const out = [
      h("h1", null, "取り込み"),
      h("div", { class: "row" },
        seg([["old", "旧コンフィグ（input/old）"], ["new", "新コンフィグ（input/new）"]], s, (x) => { im.source = x; render(); }),
        button("取り込む", doImport, { class: "primary" }),
        button("フォルダを開く", () => show("input/" + s))),
      h("p", { class: "dim" }, s === "old"
        ? "input/old のファイルを読んで、取り込みレポートを書きます。パラメータブックの機器対応・IF 対応に、足りない行を足します。"
        : "input/new のファイルを読んで、取り込みレポートを書きます。"),
      h("p", { class: "dim small" }, "画面では、秘密の列の値をいつも伏せて出します。"),
    ];
    if (!v) return out;

    out.push(
      h("h2", null, v.title),
      boxes("info", v.messages),
      st.jdg.changed[s] ? h("div", { class: "box warn", id: "judged-since" }, "この取り込みの後に、未対応の行の判定を変えました。下の数と取り込みレポートは、取り込み直すと新しくなります。") : null,
      counts(v.counts),
      boxes("warn", v.notes),
      h("div", { class: "row" },
        button("レポートを Excel で開く", () => show(v.report.xlsx)),
        button("レポートをブラウザで開く", () => show(v.report.html)),
        v.files.some((f) => f.unknown) ? button("未対応の行に判定を付ける", toJudge) : null,
        s === "new" ? button("新コンフィグからパラメータブックを作る", () => makeBook("new", false)) : null),
      st.map.book && st.map.book.from === "new" && s === "new" ? bookResult(st.map.book) : null,
      needPacks(s, v));

    const tab = im.tab[s] || "files";
    const lv = im.lines[s];
    const tabs = [["files", `ファイル（${v.files.length}）`]];
    v.tables.forEach((t, i) => tabs.push(["t" + i, `${t.name}（${t.rows.length}）`]));
    if (lv) tabs.push(["lines", `行: ${lv.view.path}`]);
    out.push(h("div", { class: "row", id: "tabs" }, seg(tabs, tab, (x) => { im.tab[s] = x; render(); })));

    if (tab === "files") out.push(filesTable(s, v));
    else if (tab === "lines" && lv) out.push(linesView(s, lv));
    else {
      const t = v.tables[Number(tab.slice(1))];
      if (t) out.push(tableView(t, { id: `import/${s}/${t.name}` }));
    }
    return out;
  }

  // needPacks lists the files whose vendor a person has to choose.
  function needPacks(s, v) {
    const list = v.files.filter((f) => f.needsPack);
    if (!list.length) return null;
    const im = st.imp;
    return h("div", { class: "box warn" },
      h("div", null, "メーカーを自動で決められないファイルがあります。パックを選んでください（選んだ結果は案件に残ります）。"),
      list.map((f) => {
        const key = s + "/" + f.path;
        const first = f.candidates.length ? f.candidates[0].pack : v.packs.length ? v.packs[0].id : "";
        if (!im.pick[key]) im.pick[key] = first;
        const choose = () => run(`${f.path} を読み直しています`, async () => {
          im.views[s] = await api().ChoosePack(s, f.path, im.pick[key]);
          delete im.lines[s];
        }, true);
        return h("div", { class: "row" },
          h("span", { class: "path" }, f.path),
          h("span", { class: "dim small" }, f.candidates.length ? "候補: " + f.candidates.map((c) => `${c.pack}（${c.score} 点）`).join("、") : ""),
          select(v.packs.map((p) => [p.id, `${p.id} ${p.version}`]), im.pick[key], (x) => { im.pick[key] = x; }),
          button("このパックで読む", choose));
      }));
  }

  function filesTable(s, v) {
    const im = st.imp;
    const files = im.onlyBad ? v.files.filter((f) => f.mark) : v.files;
    const header = ["ファイル", "機器", "パック", "行", "取り込み済み", "無視", "コメント・空行", "未対応", "未判定・保留", "取り込み率"];
    if (s === "old") header.push("新ホスト名");
    header.push("指摘");
    const t = { name: "ファイル", lead: "", header, code: [], marks: files.map((f) => f.mark), rows: files.map((f) => {
      const pack = f.pack ? `${f.pack} ${f.version}${f.chosen ? "（人が選んだ）" : ""}` : "";
      const told = f.err ? [f.err] : f.warnings.concat(f.notes);
      const row = [f.path, f.device, pack];
      if (f.imported) row.push(String(f.total), String(f.taken), String(f.ignored), String(f.blank), String(f.unknown), String(f.open), f.ratio);
      else row.push("", "", "", "", "", "", "");
      if (s === "old") row.push(f.newHost);
      row.push(told.join(" / "));
      return row;
    }) };
    const pick = (i) => {
      const f = files[i];
      if (!f.imported) {
        st.error = `${f.path} は取り込めていません: ${f.err}`;
        render();
        return;
      }
      run(`${f.path} の行を読んでいます`, async () => {
        im.lines[s] = { view: await api().Lines(s, f.path), only: false, shown: LINES, sel: -1 };
        im.tab[s] = "lines";
        st.jump = "tabs"; // the lines need the height of the window
      });
    };
    return h("div", null,
      h("div", { class: "row" },
        h("label", null, h("input", { type: "checkbox", checked: im.onlyBad, onchange: (e) => { im.onlyBad = e.target.checked; render(); } }), " 指摘がある台だけ"),
        h("span", { class: "dim small" }, "ファイルを押すと、行ごとの扱いを見られます。")),
      tableView(t, { id: `import/${s}/files/${im.onlyBad}`, onPick: pick }));
  }

  // linesView draws the lines of a file: grey for an ignored line, red for
  // an unhandled one (yellow once it is judged), and what a line became
  // beside it.
  function linesView(s, lv) {
    const v = lv.view;
    const list = h("div", { class: "lines-list" });
    const side = h("div", { class: "lines-side" });
    const note = h("span", { class: "dim small" });
    const more = button("さらに表示", () => { lv.shown += LINES; fill(); });
    const unknown = v.lines.filter((l) => l.state === "unknown").length;
    const open = v.lines.filter((l) => l.open).length;

    function detail(l) {
      side.textContent = "";
      if (!l) {
        side.appendChild(h("p", { class: "dim" }, "行を押すと、その行の扱いを出します。"));
        return;
      }
      add(side, [h("dl", null,
        h("dt", null, "行"), h("dd", null, String(l.no)),
        h("dt", null, "状態"), h("dd", null, h("span", { class: "tag " + l.state }, l.word)),
        l.judged ? [h("dt", null, "判定"), h("dd", null,
          h("span", { class: cls("tag", l.open ? "unknown" : "judged") }, l.judged), " ",
          button("このかたまりを見る", () => openJudge(s, l.group), { class: "link" }))] : null,
        l.reason ? [h("dt", null, "理由"), h("dd", null, l.reason)] : null,
        l.rule ? [h("dt", null, "当たった規則"), h("dd", { class: "code" }, l.rule)] : null,
        l.made.length ? [h("dt", null, "入った先と値"), l.made.map((m) => h("dd", null, m.where, h("div", { class: "code" }, m.value)))] : null,
        h("dt", null, "行の文字"), h("dd", { class: "code" }, l.text))]);
    }

    function fill() {
      const hit = lv.only ? v.lines.filter((l) => l.state === "unknown") : v.lines;
      list.textContent = "";
      for (const l of hit.slice(0, lv.shown)) {
        const row = h("div", { class: cls("line", l.state, l.state === "unknown" && !l.open && "judged", lv.sel === l.no && "sel") },
          h("span", { class: "no" }, String(l.no)), h("span", { class: "tx" }, l.text));
        row.addEventListener("click", () => {
          const old = list.querySelector(".line.sel");
          if (old) old.classList.remove("sel");
          row.classList.add("sel");
          lv.sel = l.no;
          detail(l);
        });
        list.appendChild(row);
      }
      if (!hit.length) list.appendChild(h("div", { class: "empty" }, lv.only ? "未対応の行はありません" : "行がありません"));
      note.textContent = hit.length > lv.shown ? `${hit.length} 行のうち ${lv.shown} 行を出しています` : `${hit.length} 行`;
      more.hidden = hit.length <= lv.shown;
    }

    fill();
    detail(v.lines.find((l) => l.no === lv.sel));
    return h("div", null,
      h("div", { class: "row" },
        h("span", null, `${v.path}（機器 ${v.device || "不明"}）　未対応 ${unknown} 行（うち未判定・保留 ${open} 行）`),
        h("label", null, h("input", { type: "checkbox", checked: lv.only, onchange: (e) => { lv.only = e.target.checked; lv.shown = LINES; fill(); } }), " 未対応だけ"),
        note, more),
      h("div", { class: "lines" }, list, side));
  }

  // ---- 未対応の行 ----

  const VERDICTS = [["manual", "手動で対応"], ["not_needed", "対応不要"], ["hold", "保留"]];

  function loadJudge() {
    const s = st.jdg.source;
    run(`${SRC[s]}を読んでいます`, async () => {
      showJudge(s, await api().Unhandled(s));
    }, true);
  }

  // showJudge takes the judgments of a side and keeps what was picked, if it
  // is still there.
  function showJudge(s, v) {
    const jd = st.jdg;
    jd.views[s] = v;
    if (jd.by == null || !jd.by.trim()) jd.by = v.by;
    if (!v.groups.some((g) => g.id === jd.sel[s])) jd.sel[s] = "";
  }

  // openJudge shows the group an unhandled line is in.
  function openJudge(s, group) {
    const jd = st.jdg;
    run(`${SRC[s]}を読んでいます`, async () => {
      const v = await api().Unhandled(s);
      jd.source = s;
      showJudge(s, v);
      if (v.groups.some((g) => g.id === group)) {
        if (jd.sel[s] !== group) pickGroup(s, group);
        jd.only = false;
      }
      st.page = "judge";
    }, true);
  }

  // pickGroup picks a group: what was filled in for the one before goes.
  function pickGroup(s, id) {
    const jd = st.jdg;
    jd.sel[s] = id;
    jd.scope = "all";
    jd.picked = {};
    jd.reason = "";
  }

  // carryPart is the report of the unhandled lines that may leave the site:
  // the values and the names hidden.
  function carryPart(jd, s) {
    const c = jd.carry[s];
    const go = () => run("持ち出し用レポートを作っています", async () => {
      jd.carry[s] = await api().CarryOut(s);
      st.jump = "carry-result";
    }, true);
    const out = [h("div", { class: "row" },
      button("持ち出し用レポートを出す", go, { id: "carry-run" }),
      h("span", { class: "dim small" }, "メーカーパックを直す人に渡すためのものです。値と名前を伏せた形だけを書きます（案件・機器・ファイルの名前は入りません）。"))];
    if (!c) return out;
    out.push(h("div", { id: "carry-result" },
      h("div", { class: "box info" }, c.summary),
      h("div", { class: "row" },
        button("レポートを Excel で開く", () => show(c.report.xlsx)),
        button("レポートをブラウザで開く", () => show(c.report.html))),
      boxes("warn", c.notes),
      c.hidden.rows.length ? [
        h("h3", null, `名前として伏せた語（${c.hidden.rows.length}）`),
        h("p", { class: "dim small" }, "この一覧は現場の語です。レポートには書いていません（人に渡さないでください）。コマンドの語が伏せられていたら、project.yaml の export.keep_words に足して、出し直します。"),
        tableView(c.hidden, { id: "carry/hidden/" + s }),
      ] : null));
    return out;
  }

  function judgePage() {
    const jd = st.jdg;
    const s = jd.source;
    const v = jd.views[s];
    const out = [
      h("h1", null, "未対応の行"),
      h("div", { class: "row" },
        seg([["old", "旧コンフィグ（input/old）"], ["new", "新コンフィグ（input/new）"]], s, (x) => {
          jd.source = x;
          if (jd.views[x]) render();
          else loadJudge();
        }),
        button(v ? "読み直す" : "読み込む", loadJudge, { class: v ? null : "primary" })),
      h("p", { class: "dim" }, "どの規則にも当たらなかった行を、形（数字・アドレスなどを印に置き換えた文面）ごとのかたまりにまとめて出します。かたまりごとに「手動で対応」「対応不要」「保留」のどれかを付けます。判定は案件フォルダ（state/judgments.yaml）に残り、同じ形の行には、次の取り込みでも当たります。"),
      h("p", { class: "dim small" }, "未対応の行は、どの値が秘密かをツールが見分けられないため、文面をそのまま出します。"),
    ];
    out.push(carryPart(jd, s));
    if (!v) return out;

    const list = v.groups.filter((g) => !jd.only || g.open || g.id === jd.sel[s]);
    const t = {
      name: "かたまり", lead: "", code: [1, 2], whole: [3, 4, 5], marks: list.map((g) => (g.open ? "unknown" : "")),
      header: ["判定", "形", "親の行", "台数", "行数", "ID"],
      rows: list.map((g) => [g.verdict, g.shape, g.path, String(g.devices.length), g.below ? `${g.heads}（下に ${g.below}）` : String(g.heads), g.id]),
    };
    const side = h("div", { class: "lines-side" });
    const detail = (g) => {
      side.textContent = "";
      add(side, [g ? judgeDetail(s, g) : h("p", { class: "dim" }, "かたまりを押すと、機器ごとの行を出します。判定もここで付けます。")]);
    };
    const pick = (i, tr) => {
      const old = tr.parentNode.querySelector("tr.sel");
      if (old) old.classList.remove("sel");
      tr.classList.add("sel");
      if (jd.sel[s] !== list[i].id) pickGroup(s, list[i].id);
      detail(list[i]);
    };
    const sel = list.findIndex((g) => g.id === jd.sel[s]);
    detail(list[sel]);
    afterDraw.push(() => {
      const tr = document.querySelector("#judge tr.sel");
      const wrap = tr && tr.closest(".table-wrap");
      if (wrap) wrap.scrollTop = Math.max(0, tr.offsetTop - wrap.clientHeight / 2);
    });
    out.push(
      h("h2", null, v.title),
      v.message ? h("div", { class: "box info", id: "judge-done" }, v.message) : null,
      counts(v.counts),
      boxes("warn", v.notes),
      h("div", { class: "row" },
        h("label", null, h("input", { type: "checkbox", checked: jd.only, onchange: (e) => { jd.only = e.target.checked; render(); } }), " 未判定と保留だけ"),
        h("span", { class: "dim small" }, v.groups.length
          ? `かたまり ${v.groups.length} 個のうち、未判定・保留があるのは ${v.groups.filter((g) => g.open).length} 個です。`
          : "未対応の行はありません。")),
      h("div", { class: "lines", id: "judge" },
        h("div", { class: "findings-list" }, tableView(t, { id: `judge/${s}/${jd.only}`, onPick: pick, selected: sel })),
        side));
    return out;
  }

  function judgeDetail(s, g) {
    const jd = st.jdg;
    const lineText = (l) => `${l.no}: ${l.text}` + (l.below ? `　（下に ${l.below} 行）` : "");
    const example = [g.example.text].concat(g.exampleText);
    if (g.exampleMore) example.push(`…（ほか ${g.exampleMore} 行）`);
    const name = (d) => d.device || `（機器名なし: ${d.file}）`;
    const out = [h("dl", null,
      h("dt", null, "形"), h("dd", { class: "code" }, g.shape),
      g.path ? [h("dt", null, "親の行（外側から）"), h("dd", { class: "code" }, g.path)] : null,
      h("dt", null, "パックと ID"), h("dd", null, `${g.pack}　${g.id}`),
      h("dt", null, "判定"), h("dd", null, h("span", { class: cls("tag", g.open ? "unknown" : "judged") }, g.verdict)),
      g.reason ? [h("dt", null, "判定の理由"), h("dd", null, g.reason)] : null,
      g.by ? [h("dt", null, "判定した人"), h("dd", null, g.by)] : null,
      h("dt", null, `例（${g.exampleFile} の ${g.example.no} 行目から）`), h("dd", { class: "code" }, example.join("\n")))];
    // The lines of each device come last: there may be many of them.
    const perDevice = h("div", { class: "approve" },
      h("h2", null, `機器ごとの行（${g.devices.length} 台、${g.heads} 行${g.below ? `、その下に ${g.below} 行` : ""}）`),
      g.devices.map((d) => h("div", { class: "judge-dev" },
        h("div", null, h("b", null, name(d)), " ",
          h("span", { class: cls("tag", d.verdict && d.verdict !== "hold" ? "judged" : "unknown") }, d.word),
          d.verdict ? h("span", { class: "dim small" }, ` ${d.own ? "この機器への判定、" : ""}${d.by}、${when(d.at)}${d.reason ? "、理由: " + d.reason : ""}`) : null),
        h("div", { class: "code" }, d.heads.map(lineText).join("\n") + (d.more ? `\n…（ほか ${d.more} 行）` : "")))));

    const some = jd.scope === "some";
    const hosts = g.devices.filter((d) => d.device && jd.picked[d.device]).map((d) => d.device);
    const devices = some ? hosts.join(",") : "*";
    const put = button("判定を付ける", () => judgeGroup(s, g.id, devices, jd.verdict), { class: "primary" });
    const clear = button("判定を取り消す", async () => {
      const whose = some ? hosts.join("、") + " の" : "全台への";
      if (await ask("判定を取り消しますか", `かたまり ${g.id} の、${whose}判定を取り消します。`, "取り消す")) judgeGroup(s, g.id, devices, "none");
    });
    const check = () => {
      const nobody = some && !hosts.length;
      put.disabled = nobody || !(jd.by || "").trim() || (jd.verdict === "not_needed" && !jd.reason.trim());
      clear.disabled = nobody || !g.devices.some((d) => d.verdict);
    };
    check();
    out.push(h("div", { class: "approve judge-form" },
      h("h2", null, "判定を付ける"),
      h("p", { class: "dim small" }, "手動で対応: 追加コンフィグ（extra）に人が書きます。対応不要: 新しい機器には要りません（理由を書きます）。保留: まだ決めません（未判定と同じく、残りに数えます）。"),
      h("div", { class: "field" }, h("span", null, "範囲"),
        select([["all", "このかたまりの全台"], ["some", "選んだ機器だけ"]], jd.scope, (x) => { jd.scope = x; render(); })),
      some ? h("div", { class: "judge-pick" }, g.devices.map((d) => h("label", null,
        h("input", { type: "checkbox", checked: !!jd.picked[d.device], disabled: !d.device, onchange: (e) => { jd.picked[d.device] = e.target.checked; render(); } }),
        " " + name(d)))) : null,
      h("div", { class: "field" }, h("span", null, "判定"), seg(VERDICTS, jd.verdict, (x) => { jd.verdict = x; render(); })),
      h("div", { class: "field" }, h("span", null, "理由"),
        h("input", { type: "text", id: "judge-reason", value: jd.reason, placeholder: jd.verdict === "not_needed" ? "対応不要には、理由が要ります" : "書かなくても付けられます", oninput: (e) => { jd.reason = e.target.value; check(); } })),
      h("div", { class: "field" }, h("span", null, "判定した人"),
        h("input", { type: "text", id: "judge-by", value: jd.by || "", oninput: (e) => { jd.by = e.target.value; check(); } })),
      h("div", { class: "row" }, put, clear)));
    out.push(perDevice);
    return out;
  }

  function judgeGroup(s, id, devices, verdict) {
    const jd = st.jdg;
    run(verdict === "none" ? "判定を取り消しています" : "判定を付けています", async () => {
      showJudge(s, await api().Judge(s, id, devices, verdict, jd.reason, jd.by || ""));
      jd.reason = "";
      // The import that is shown is older than the judgments now.
      if (st.imp.views[s]) jd.changed[s] = true;
      delete st.imp.lines[s];
      if (st.imp.tab[s] === "lines") st.imp.tab[s] = "files";
    }, true);
  }

  // ---- 対応 ----

  function loadMapping() {
    run("旧コンフィグとパラメータブックを読んでいます", async () => {
      st.mig.msg = "";
      st.map.view = await api().Mapping();
    }, true);
  }

  function makeBook(from, replace) {
    run("パラメータブックに書いています", async () => {
      st.map.book = await api().MakeBook(from, replace);
      if (st.map.view) st.map.view = await api().Mapping();
    }, true);
  }

  function bookResult(b) {
    return h("div", null,
      boxes("info", b.messages),
      boxes("warn", b.problems),
      b.open.rows.length ? [h("h2", null, `${b.open.name}（${b.open.rows.length}）`), tableView(b.open, { id: "book/open/" + b.from })] : null);
  }

  function mappingPage() {
    const mp = st.map;
    const v = mp.view;
    const out = [
      h("h1", null, "対応"),
      h("p", { class: "dim" }, "機器対応と IF 対応は、パラメータブックを Excel で開いて書きます。この画面では、まだ決まっていないところを確かめ、決まったら旧コンフィグの値をパラメータブックに書きます。"),
      h("div", { class: "row" },
        button("読み直す", loadMapping),
        button("Excel で開く", () => show(st.cas.book), { disabled: !v || !v.hasBook })),
    ];
    if (!v) return out;
    out.push(boxes("warn", v.notes));
    if (!v.hasBook) {
      out.push(h("div", { class: "box info" }, `パラメータブック ${v.book} はまだありません。旧コンフィグを取り込むと作られます。`),
        h("div", { class: "row" }, button("取り込みへ", () => { st.imp.source = "old"; go("import"); })));
      return out;
    }
    const tables = { open: v.open, devices: v.devices, ifs: v.ifs };
    const mg = st.mig;
    if (mg.by == null || !mg.by.trim()) mg.by = v.by;
    const again = async () => { st.map.view = await api().Mapping(); };
    out.push(
      counts(v.counts),
      boxes("info", mg.msg ? [mg.msg] : []),
      boxes("warn", v.problems),
      h("div", { class: "row" }, seg([
        ["open", `未決（${v.open.rows.length}）`],
        ["devices", `機器対応（${v.devices.rows.length}）`],
        ["ifs", `IF 対応（${v.ifs.rows.length}）`]], mp.tab, (x) => { mp.tab = x; render(); })),
      mp.tab === "open" ? migTable(v.open, v.findings, "mapping/open", again) : tableView(tables[mp.tab], { id: "mapping/" + mp.tab }),
      h("div", { class: "card" },
        h("h2", null, "パラメータブックに書く"),
        h("p", null, "旧コンフィグの値に対応を当てて（ホスト名と IF 名を新しい名前に替えて）、パラメータブックの設計データのシートに書きます。すでにある行は変えず、足りない行を足します。"),
        h("div", { class: "row" },
          h("label", null, h("input", { type: "checkbox", checked: mp.replace, onchange: (e) => { mp.replace = e.target.checked; } }),
            " 設計データのシートを空にしてから書く（作り直し）")),
        h("div", { class: "row" }, button("対応を当ててパラメータブックに書く", async () => {
          if (mp.replace && !(await ask("作り直しますか", "設計データのシートを空にしてから書きます。Excel で手で直した値は消えます（書く前の控えは残ります）。機器対応・IF 対応のシートは変えません。", "作り直す"))) return;
          makeBook("old", mp.replace);
        }, { class: "primary" }))),
      mp.book && mp.book.from === "old" ? bookResult(mp.book) : null);
    return out;
  }

  // ---- 結果 ----

  const NO_DEVICE = "-"; // the findings that belong to no one device

  function generate(checkOnly) {
    run(checkOnly ? "チェックしています" : "コンフィグを生成してチェックしています", async () => {
      showResult(checkOnly ? await api().Check() : await api().Generate());
      st.res.tab = "devices";
      st.page = "result";
    }, true, "台");
  }

  // showResult takes a new result and keeps what was picked, if it is still there.
  function showResult(v) {
    const rs = st.res;
    rs.view = v;
    if (rs.by == null || !rs.by.trim()) rs.by = v.by;
    if (!v.findings.some((f) => f.key === rs.sel)) rs.sel = "";
    if (rs.device && rs.device !== NO_DEVICE && !v.devices.some((d) => d.host === rs.device)) rs.device = "";
  }

  // when gives a time of the tool as the clock of this machine shows it.
  function when(text) {
    const d = new Date(text);
    if (!text || isNaN(d.getTime())) return text || "";
    const two = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${two(d.getMonth() + 1)}-${two(d.getDate())} ${two(d.getHours())}:${two(d.getMinutes())}`;
  }

  function resultPage() {
    const rs = st.res;
    const v = rs.view;
    const hasBook = !!(st.home && st.home.hasBook);
    const again = [
      button("生成とチェック", () => generate(false), { class: v ? null : "primary", disabled: !hasBook }),
      button("チェックだけ", () => generate(true), { disabled: !hasBook }),
    ];
    if (!v) {
      return [
        h("h1", null, "結果"),
        h("p", { class: "dim" }, hasBook
          ? "まだ結果がありません。「生成とチェック」か「チェックだけ」を押すと、ここに出ます（結果は、案件を開いている間だけ覚えています）。"
          : "パラメータブックがまだありません。旧コンフィグを取り込むと作られます。"),
        h("div", { class: "row" }, again),
      ];
    }
    const todo = v.findings.filter((f) => f.state === "open" || f.state === "fix").length;
    const tabs = [
      ["devices", `機器（${v.devices.length}）`],
      ["findings", `指摘（未承認 ${todo} / 全 ${v.findings.length}）`],
      ["checks", `チェック項目（${v.checks.rows.length}）`],
      ["approvals", `承認（${v.approvals.length}）`],
      ["changes", "変更"],
    ];
    const out = [
      h("h1", null, "結果"),
      h("h2", null, v.title),
      v.message ? h("div", { class: "box info" }, v.message) : null,
      v.notice ? h("div", { class: "box warn" }, v.notice) : null,
      counts(v.counts),
      boxes("warn", v.notes),
      h("div", { class: "row" },
        button("チェック結果表を Excel で開く", () => show(v.report.xlsx)),
        button("チェック結果表をブラウザで開く", () => show(v.report.html)),
        v.changes.has ? button("変更一覧を Excel で開く", () => show(v.changes.report.xlsx)) : null,
        v.changes.has ? button("変更一覧をブラウザで開く", () => show(v.changes.report.html)) : null,
        button("出力フォルダを開く", () => show(v.dir)),
        button("パラメータブックを Excel で開く", () => show(st.cas.book))),
      h("div", { class: "row" }, h("span", { class: "dim small" }, "パラメータブックを直したら、もう一度:"), again),
      h("p", { class: "dim small" }, "画面では、秘密の列の値をいつも伏せて出します。書き出したコンフィグのファイルには、値がそのまま入っています。"),
      h("div", { class: "row", id: "tabs" }, seg(tabs, rs.tab, (x) => { rs.tab = x; render(); })),
    ];
    const draw = { devices: resultDevices, findings: resultFindings, checks: () => tableView(v.checks, { id: "result/checks" }), approvals: resultApprovals, changes: resultChanges };
    out.push(draw[rs.tab](v));
    return out;
  }

  function resultDevices(v) {
    const rs = st.res;
    const list = rs.onlyBad ? v.devices.filter((d) => d.mark) : v.devices;
    const t = {
      name: "機器", lead: "", code: [], whole: [0, 1, 2], marks: list.map((d) => d.mark),
      header: ["機器", "パック", "状態", "未承認のエラー", "未承認の警告", "承認済み", "情報", "コンフィグ", "備考"],
      rows: list.map((d) => [d.label, d.pack, d.word, String(d.errors), String(d.warnings), String(d.approved), String(d["info"]), d.file, d.why]),
    };
    const pick = (i) => {
      rs.device = list[i].host || NO_DEVICE;
      rs.tab = "findings";
      rs.sel = "";
      render();
    };
    return h("div", null,
      h("div", { class: "row" },
        h("label", null, h("input", { type: "checkbox", checked: rs.onlyBad, onchange: (e) => { rs.onlyBad = e.target.checked; render(); } }), " 指摘がある台だけ"),
        h("span", { class: "dim small" }, "機器を押すと、その機器の指摘を見られます。")),
      tableView(t, { id: `result/devices/${rs.onlyBad}`, onPick: pick }));
  }

  function resultFindings(v) {
    const rs = st.res;
    const list = v.findings.filter((f) => {
      if (rs.device && (rs.device === NO_DEVICE ? f.device !== "" : f.device !== rs.device)) return false;
      if (rs.sev && f.severity !== rs.sev) return false;
      if (rs.state === "todo") return f.state === "open" || f.state === "fix";
      return !rs.state || f.state === rs.state;
    });
    const set = (name) => (x) => { rs[name] = x; render(); };
    const devices = [["", "すべての機器"]].concat(v.devices.map((d) => [d.host || NO_DEVICE, d.label]));
    const side = h("div", { class: "lines-side" });
    const detail = (f) => {
      side.textContent = "";
      add(side, [f ? findingDetail(v, f) : h("p", { class: "dim" }, "指摘を押すと、内容・場所・根拠を出します。承認もここで行います。")]);
    };
    const t = {
      name: "指摘", lead: "", code: [], whole: [0, 1, 3], marks: list.map((f) => f.mark),
      header: ["チェック", "重大度", "状態", "機器", "表", "行", "列", "内容"],
      rows: list.map((f) => [f.rule, f.word, f.stateWord, f.device, f.table, f.row, f.column, f.message]),
    };
    const pick = (i, tr) => {
      const old = tr.parentNode.querySelector("tr.sel");
      if (old) old.classList.remove("sel");
      tr.classList.add("sel");
      if (rs.sel !== list[i].key) rs.reason = "";
      rs.sel = list[i].key;
      detail(list[i]);
    };
    const sel = list.findIndex((f) => f.key === rs.sel);
    detail(list[sel]);
    // The row that is picked stays in sight when the list is drawn again.
    afterDraw.push(() => {
      const tr = document.querySelector("#findings tr.sel");
      const wrap = tr && tr.closest(".table-wrap");
      if (wrap) wrap.scrollTop = Math.max(0, tr.offsetTop - wrap.clientHeight / 2);
    });
    return h("div", null,
      h("div", { class: "row" },
        select(devices, rs.device, set("device")),
        select([["", "すべての重大度"], ["error", "エラー"], ["warning", "警告"], ["info", "情報"]], rs.sev, set("sev")),
        select([["", "すべての状態"], ["todo", "未承認・要修正"], ["approved", "承認済み"], ["info", "情報"]], rs.state, set("state")),
        rs.device || rs.sev || rs.state ? button("絞り込みを外す", () => { rs.device = ""; rs.sev = ""; rs.state = ""; render(); }, { class: "link" }) : null),
      h("div", { class: "lines", id: "findings" },
        h("div", { class: "findings-list" }, tableView(t, { id: "result/findings", onPick: pick, selected: sel })),
        side));
  }

  // findingDetail shows a finding and lets it be approved. o, for a finding
  // of a migration: rs (where the approval being written is kept), approve
  // and unapprove.
  function findingDetail(v, f, o) {
    o = o || {};
    const rs = o.rs || st.res;
    const place = [f.table, f.row, f.column].filter(Boolean).join(" / ");
    const out = [h("dl", null,
      h("dt", null, "チェック"), h("dd", null, `${f.rule} ${f.title}（${f.kind}）`),
      h("dt", null, "重大度と状態"), h("dd", null, h("span", { class: "tag sev-" + f.severity }, f.word), " ", f.stateWord),
      h("dt", null, "機器"), h("dd", null, f.device || "（案件全体）"),
      place ? [h("dt", null, "場所"), h("dd", null, place)] : null,
      f.at ? [h("dt", null, "パラメータブックのセル"), h("dd", { class: "code" }, f.at)] : null,
      h("dt", null, "内容"), h("dd", null, f.message),
      f.evidence.length ? [h("dt", null, "根拠"), f.evidence.map((p) => h("dd", { class: "code" }, `${p.label}: ${p.value}`))] : null,
      f.line ? [h("dt", null, `コンフィグの ${f.line} 行目`), h("dd", { class: "code" }, f.text)] : null,
      f.state === "approved" ? [
        h("dt", null, "承認"), h("dd", null, `${f.approval}（${f.by}、${when(f.when)}）`),
        h("dt", null, "承認の理由"), h("dd", null, f.reason)] : null)];

    if (o.extra) out.push(o.extra(f));
    if (f.state === "approved") {
      out.push(h("div", { class: "row" }, button("この承認を取り消す", () => (o.unapprove || unapprove)(f.approval))));
    } else if (f.state === "fix") {
      out.push(h("div", { class: "box warn" }, "この指摘は承認できません。パラメータブックの値（またはテンプレート・追加コンフィグ）を直して、もう一度チェックしてください。"),
        h("div", { class: "row" }, button("パラメータブックを Excel で開く", () => show(st.cas.book))));
    } else if (f.state === "open") {
      const what = [f.rule, f.table && `表「${f.table}」`, o.migration && f.column && `列「${f.column}」`].filter(Boolean).join("・");
      const scopes = [["", "この指摘だけ"]];
      if (f.canScope && f.device) scopes.push(["device", `${f.device} の、${what}の指摘すべて`]);
      if (f.canScope) scopes.push(["all", `全機器の、${what}の指摘すべて`]);
      if (!scopes.some((s) => s[0] === rs.scope)) rs.scope = "";
      const ok = button("承認する", () => (o.approve || approve)(f.key), { class: "primary" });
      const check = () => { ok.disabled = !rs.reason.trim() || !(rs.by || "").trim(); };
      check();
      out.push(h("div", { class: "approve" },
        h("h2", null, "承認する"),
        h("p", { class: "dim small" }, "指摘の内容を確かめて、そのままでよいと判断したときに、理由を書いて承認します。承認は案件フォルダ（state/approvals.yaml）に残ります。"),
        h("div", { class: "field" }, h("span", null, "範囲"),
          scopes.length > 1
            ? select(scopes, rs.scope, (x) => { rs.scope = x; })
            : h("span", null, "この指摘だけ（この指摘は、まとめては承認できません）")),
        h("div", { class: "field" }, h("span", null, "理由"),
          h("input", { type: "text", id: "approve-reason", value: rs.reason, oninput: (e) => { rs.reason = e.target.value; check(); } })),
        h("div", { class: "field" }, h("span", null, "承認者"),
          h("input", { type: "text", id: "approve-by", value: rs.by || "", oninput: (e) => { rs.by = e.target.value; check(); } })),
        h("div", { class: "row" }, ok)));
    }
    return out;
  }

  // migTable draws a list of a comparison or of the mapping. A row is tied
  // to a finding of a migration by the fingerprint in the column 指紋: the
  // row is picked, and the finding is shown and approved beside the list.
  // reload asks for the list again after an approval.
  function migTable(t, items, id, reload, extra) {
    const mg = st.mig;
    const col = t.header.indexOf("指紋");
    if (col < 0 || !items || !items.length) return tableView(t, { id });
    const byKey = new Map(items.map((f) => [f.key, f]));
    const side = h("div", { class: "lines-side" });
    const detail = (f) => {
      side.textContent = "";
      add(side, [f
        ? findingDetail(null, f, { rs: mg, migration: true, extra, approve: (key) => approveMig(key, reload), unapprove: (id) => unapproveMig(id, reload) })
        : h("p", { class: "dim" }, "「指紋」の欄に字がある行を押すと、指摘の内容と根拠を出します。承認もここで行います。")]);
    };
    const pick = (i, tr) => {
      const key = t.rows[i][col];
      const old = tr.parentNode.querySelector("tr.sel");
      if (old) old.classList.remove("sel");
      tr.classList.add("sel");
      if (mg.sel !== key) mg.reason = "";
      mg.sel = key;
      detail(byKey.get(key));
    };
    const sel = mg.sel ? t.rows.findIndex((r) => r[col] === mg.sel) : -1;
    detail(sel >= 0 ? byKey.get(mg.sel) : null);
    return h("div", { class: "lines", id: "mig-findings" },
      h("div", { class: "findings-list" }, tableView(t, { id, onPick: pick, selected: sel })),
      side);
  }

  function approveMig(key, reload) {
    const mg = st.mig;
    run("承認して、一覧を作り直しています", async () => {
      mg.msg = await api().ApproveMigration(key, mg.scope, mg.reason, mg.by || "");
      mg.reason = "";
      mg.scope = "";
      await reload();
    }, true);
  }

  async function unapproveMig(id, reload) {
    if (!(await ask("承認を取り消しますか", `承認 ${id} を取り消します。この承認が当たっていた指摘は、未承認に戻ります。`, "取り消す"))) return;
    run("承認を取り消して、一覧を作り直しています", async () => {
      await api().UnapproveMigration(id);
      st.mig.msg = `承認 ${id} を取り消しました`;
      await reload();
    }, true);
  }

  function approve(key) {
    const rs = st.res;
    run("承認して、チェックをやり直しています", async () => {
      showResult(await api().Approve(key, rs.scope, rs.reason, rs.by || ""));
      rs.reason = "";
      rs.scope = "";
    }, true, "台");
  }

  async function unapprove(id) {
    if (!(await ask("承認を取り消しますか", `承認 ${id} を取り消します。この承認が当たっていた指摘は、未承認に戻ります。`, "取り消す"))) return;
    run("承認を取り消して、チェックをやり直しています", async () => {
      showResult(await api().Unapprove(id));
    }, true, "台");
  }

  function resultApprovals(v) {
    if (!v.approvals.length) {
      return h("div", { class: "table-wrap" }, h("div", { class: "empty" }, "承認はありません。「指摘」で指摘を選ぶと、承認できます。"));
    }
    const header = ["ID", "対象", "チェック", "機器", "理由", "承認者", "日時", "当たった件数", "備考", ""];
    return h("div", null,
      h("p", { class: "dim" }, "この案件の承認（state/approvals.yaml）と、今回それぞれが当たった指摘の件数です。"),
      h("div", { class: "table-wrap" }, h("table", null,
        h("thead", null, h("tr", null, header.map((x) => h("th", null, x)))),
        h("tbody", null, v.approvals.map((a) => h("tr", { class: a.mark ? "mark-" + a.mark : null },
          h("td", null, a.id), h("td", null, a.target), h("td", null, a.rule), h("td", null, a.devices),
          h("td", null, a.reason), h("td", null, a.by), h("td", null, when(a.at)),
          h("td", { class: "num" }, String(a.hits)), h("td", null, a.note),
          h("td", null, button("取り消す", () => unapprove(a.id)))))))));
  }

  function resultChanges(v) {
    const rs = st.res;
    const c = v.changes;
    if (!c.has) {
      return h("div", { class: "table-wrap" }, h("div", { class: "empty" },
        v.configs ? "比べる相手（前の生成）がないので、変更一覧はありません。2 回目の生成から出ます。" : "変更一覧は、「生成とチェック」のときに出ます（「チェックだけ」では作りません）。"));
    }
    const t = c.tables[rs.ctab] || c.tables[0];
    return h("div", null,
      h("p", { class: "dim" }, `前の生成（${c.before}）から変わったところです。`),
      counts(c.counts),
      boxes("warn", c.notes),
      h("div", { class: "row" }, seg(c.tables.map((x, i) => [i, `${x.name}（${x.rows.length}）`]), c.tables.indexOf(t), (i) => { rs.ctab = i; render(); })),
      t ? tableView(t, { id: "result/changes/" + t.name }) : null);
  }

  // ---- 比較 ----

  // ---- お客様の一覧 ----

  const OPS = { empty: "が空", notempty: "が空でない", contains: "が次を含む", notcontains: "が次を含まない", equals: "が次と同じ", notequals: "が次と違う" };
  const STEPS = {
    trim: "前後の空白を除く", width: "全角を半角にする", lower: "小文字にする", upper: "大文字にする",
    split: "区切って片方を残す", replace: "文字を置き換える", regex: "正規表現で置き換える", map: "置き換え表",
    cidr: "マスクをプレフィックス長にする",
  };
  const SOURCES = [["column", "一覧の列"], ["template", "組み立て"], ["value", "決まった値"]];

  function loadMaps() {
    run("列対応ファイルを読んでいます", async () => {
      st.cus.maps = await api().Mappings();
    });
  }

  function customerPage() {
    const cu = st.cus;
    const pickTab = (x) => {
      cu.tab = x;
      if (x === "maps" && !cu.maps) {
        loadMaps();
        return;
      }
      render();
    };
    return [
      h("h1", null, "お客様の一覧"),
      h("p", { class: "dim" }, "お客様からもらった一覧（input/customer/）を、列対応ファイル（mappings/）のとおりに読んで、パラメータブックと比べます。"),
      h("div", { class: "row" },
        seg([["compare", "比べる"], ["maps", "列対応"]], cu.tab, pickTab),
        button("input/customer を開く", () => show("input/customer")),
        button("mappings を開く", () => show("mappings"))),
      cu.tab === "compare" ? customerCompare() : customerMaps(),
    ];
  }

  // ---- ログ ----

  function logsPage() {
    const lg = st.logs;
    const what = lg.tab;
    const v = lg.views[what];
    const mg = st.mig;
    const again = async () => { lg.views[what] = await api().Logs(what); };
    const label = what === "load" ? "調べる" : "比べる";
    const doing = what === "load" ? "投入ログを調べています" : "投入後の設定を、パラメータブックと比べています";
    const start = () => run(doing, async () => {
      lg.views[what] = null;
      mg.msg = "";
      mg.sel = "";
      await again();
      lg.vtab[what] = 1;
    }, true, "ログ");
    const out = [
      h("h1", null, "ログ"),
      h("p", { class: "dim" }, "検証機や本番機で取った端末ログ（input/logs/。1 台で 1 ファイル）を読みます。「投入ログ」は機器に弾かれたコマンドを探し、「反映確認」は投入後の設定全体（show running-config）をパラメータブックと比べます。"),
      h("div", { class: "row" },
        seg([["load", "投入ログ"], ["verify", "反映確認"]], what, (x) => { lg.tab = x; mg.sel = ""; render(); }),
        button("input/logs を開く", () => show("input/logs"))),
      h("div", { class: "row" }, button(label, start, { class: "primary", id: "logs-run" })),
    ];
    if (!v) return out;
    if (mg.by == null || !mg.by.trim()) mg.by = v.by;
    const i = lg.vtab[what] || 0;
    const t = v.tables[i];
    out.push(
      h("h2", null, v.title),
      counts(v.counts),
      boxes("info", mg.msg ? [mg.msg + "\nもう一度読み直して、レポートも新しく書きました。"] : []),
      boxes("info", v.notes),
      h("div", { class: "row" },
        button("レポートを Excel で開く", () => show(v.report.xlsx)),
        button("レポートをブラウザで開く", () => show(v.report.html))),
      h("div", { class: "row" }, seg(v.tables.map((x, n) => [n, `${x.name}（${x.rows.length}）`]), i, (n) => { lg.vtab[what] = n; mg.sel = ""; render(); })),
      t ? migTable(t, v.items, "logs/" + what + "/" + t.name, again) : null);
    return out;
  }

  // reflect writes the values of the lists into the book, after asking.
  async function reflect(keys, rule, n, again) {
    const text = `お客様の一覧の値を、パラメータブックに書きます（${n} 件）。書く前に、今のブックの控えを state/backup に取ります。Excel でブックを開いているときは、先に閉じてください。`;
    if (!(await ask("パラメータブックに反映しますか", text, "反映する"))) return;
    run("パラメータブックに反映して、比べ直しています", async () => {
      st.mig.msg = await api().Reflect(keys, rule);
      await again();
    }, true);
  }

  function customerCompare() {
    const cu = st.cus;
    const v = cu.view;
    const again = async () => { cu.view = await api().Customer(); };
    const read = () => run("お客様の一覧を読んで、パラメータブックと比べています", async () => {
      cu.view = null;
      st.mig.msg = "";
      st.mig.sel = "";
      await again();
      if (cu.vtab >= cu.view.tables.length) cu.vtab = 1;
    }, true);
    const out = [h("div", { class: "row" }, button("読んで比べる", read, { class: "primary", id: "customer-read" }))];
    if (!v) return out;
    const t = v.tables[cu.vtab];
    const mg = st.mig;
    if (mg.by == null || !mg.by.trim()) mg.by = v.by;
    const ruleOf = { "一覧にだけある行": "S-501", "値が違うセル": "S-503" }[t ? t.name : ""];
    const open = ruleOf ? v.items.filter((f) => f.rule === ruleOf && f.state === "open").length : 0;
    const extra = (f) => (f.state === "open" && (f.rule === "S-501" || f.rule === "S-503")
      ? h("div", { class: "approve" },
        h("h2", null, "ブックに反映する"),
        h("p", { class: "dim small" }, f.rule === "S-501"
          ? "一覧が正しいとき: この行を、パラメータブックのシートの末尾に足します。"
          : "一覧が正しいとき: パラメータブックのこのセルを、一覧の値に書き換えます（その機器の名前で書いた行のセルだけ）。"),
        h("div", { class: "row" }, button("一覧の値をブックに反映する", () => reflect([f.key], "", 1, again), { class: "primary", id: "reflect-one" })))
      : null);
    out.push(
      h("h2", null, v.title),
      counts(v.counts),
      boxes("info", mg.msg ? [mg.msg + "\n読んで比べ直して、レポートも新しく書きました。"] : []),
      boxes("info", v.notes),
      h("div", { class: "row" },
        button("レポートを Excel で開く", () => show(v.report.xlsx)),
        button("レポートをブラウザで開く", () => show(v.report.html)),
        button("パラメータブックを Excel で開く", () => show(st.cas.book))),
      h("div", { class: "row" }, seg(v.tables.map((x, i) => [i, `${x.name}（${x.rows.length}）`]), cu.vtab, (i) => { cu.vtab = i; render(); })),
      open ? h("div", { class: "row" }, button(`未承認の ${open} 件をすべてブックに反映する`, () => reflect([], ruleOf, open, again), { id: "reflect-all" })) : null,
      t ? migTable(t, v.items, "customer/" + t.name, again, extra) : null);
    return out;
  }

  // ---- 列対応 ----

  const tableInfo = (name) => st.cus.maps.tables.find((t) => t.name === name);

  // keyFields gives the fields a target starts with: the key columns of its
  // table. The device comes from the column the first target reads it from.
  function keyFields(table, rule) {
    const t = tableInfo(table);
    let host = null;
    for (const tg of rule.rows) {
      const ti = tableInfo(tg.table);
      const dev = ti && ti.columns.find((c) => c.name === (tg.table === "devices" ? "hostname" : "device"));
      host = host || (dev && tg.columns.find((f) => f.to === dev.name && f.column));
    }
    return t.columns.filter((c) => c.key).map((c) => {
      const f = newField(c.name);
      if (host && (c.name === "device" || c.name === "hostname")) {
        f.column = host.column;
        f.steps = JSON.parse(JSON.stringify(host.steps));
      }
      return f;
    });
  }

  const newField = (to) => ({ to, column: "", fallback: [], template: "", value: "", steps: [], default: "", _src: "column" });

  // srcOf tells where a field takes its value from: what the user chose, or
  // what the rule file says.
  function srcOf(f) {
    if (!f._src) f._src = f.template ? "template" : f.value ? "value" : "column";
    return f._src;
  }

  // cleanRule gives the rule as the tool takes it: of the three sources of a
  // field only the chosen one.
  function cleanRule(r) {
    const c = JSON.parse(JSON.stringify(r));
    c.headerRow = Number(c.headerRow) || 0;
    c.dataRow = Number(c.dataRow) || 0;
    c.sheetIndex = Number(c.sheetIndex) || 0;
    for (const tg of c.rows) {
      for (const f of tg.columns) {
        const src = srcOf(f);
        delete f._src;
        if (src !== "column") { f.column = ""; f.fallback = []; }
        if (src !== "template") f.template = "";
        if (src !== "value") f.value = "";
      }
    }
    return c;
  }

  // sample asks for the top of the list again (the sheet, the header row or
  // the mapping of a secret column changed).
  async function sample(ed) {
    ed.tryv = null;
    try {
      ed.sample = await api().ListSample(cleanRule(ed.rule));
      ed.err = "";
      if (!Number(ed.rule.headerRow)) ed.rule.headerRow = ed.sample.headerRow;
      ed.rule.sheet = ed.sample.sheet;
    } catch (e) {
      ed.sample = null;
      ed.err = errText(e);
    }
  }
  const resample = () => run("一覧を読んでいます", () => sample(st.cus.edit));

  function newMap(file) {
    run("一覧を読んでいます", async () => {
      const rule = { version: 1, file, sheet: "", sheetIndex: 0, headerRow: 0, dataRow: 0, startCol: "", mergeOnlyFirst: false, names: "new", exclude: [], rows: [], path: "" };
      const ed = { path: "", rule, sample: null, tryv: null, err: "", open: "" };
      await sample(ed);
      const sug = (ed.sample && ed.sample.suggest) || [];
      rule.rows.push({ table: "devices", label: "", columns: sug.length ? sug : keyFields("devices", rule) });
      st.cus.edit = ed;
      st.cus.msg = "";
    });
  }

  function openMap(info) {
    run("一覧を読んでいます", async () => {
      const ed = { path: info.path, rule: JSON.parse(JSON.stringify(info.rule)), sample: null, tryv: null, err: "", open: "" };
      await sample(ed);
      st.cus.edit = ed;
      st.cus.msg = "";
    });
  }

  function customerMaps() {
    const cu = st.cus;
    const m = cu.maps;
    if (!m) return h("p", { class: "dim" }, "列対応ファイルを読んでいます。");
    if (cu.edit) return mapEditor();
    const t = {
      name: "列対応ファイル", header: ["ファイル", "一覧", "シート", "機器名", "行き先", "状態"], code: [],
      rows: m.mappings.map((x) => {
        const r = x.rule;
        return [x.path, r ? r.file : "", r ? r.sheet || (r.sheetIndex ? `${r.sheetIndex} 枚目` : "") : "",
          r ? (r.names === "old" ? "今の（旧の）名前" : "新しい名前") : "",
          r ? r.rows.map((tg) => (tableInfo(tg.table) || { title: tg.table }).title + (tg.label ? `（${tg.label}）` : "")).join("、") : "",
          x.problems.length ? "誤りがあります:\n" + x.problems.join("\n") : "読めます"];
      }),
      marks: m.mappings.map((x) => (x.problems.length ? "unknown" : "")),
    };
    const cs = cu.create || (cu.create = { file: m.files[0] || "" });
    if (!m.files.includes(cs.file)) cs.file = m.files[0] || "";
    return [
      boxes("info", cu.msg ? [cu.msg] : []),
      h("h2", null, "列対応ファイル"),
      h("p", { class: "dim" }, "「一覧のどの見出しが、設計データのどの表のどの列か」を書いたファイルです。一覧 1 つ（シート 1 枚の表 1 つ）につき 1 つ作ります。行を押すと開きます。誤りがあるファイルが 1 つでもあると「読んで比べる」は始まりません。"),
      m.mappings.length
        ? tableView(t, { id: "customer/maps", onPick: (i) => {
          if (m.mappings[i].rule) openMap(m.mappings[i]);
          else { st.error = `${m.mappings[i].path} は、列対応ファイルとして読めません。テキストエディタで直してください:\n${m.mappings[i].problems.join("\n")}`; render(); }
        } })
        : h("p", { class: "dim" }, "まだありません。"),
      h("div", { class: "row" }, button("読み直す", loadMaps)),
      h("h2", null, "新しく作る"),
      m.files.length
        ? h("div", { class: "row" },
          h("span", null, "一覧のファイル"), select(m.files.map((f) => [f, f]), cs.file, (x) => { cs.file = x; }),
          button("この一覧の列対応を作る", () => newMap(cs.file), { class: "primary", id: "map-new" }))
        : h("p", { class: "dim" }, "input/customer にファイルがありません。「input/customer を開く」で、お客様の一覧（xlsx・xls・csv）を置いてから「読み直す」を押してください。"),
    ];
  }

  function mapEditor() {
    const cu = st.cus;
    const ed = cu.edit;
    const r = ed.rule;
    const sm = ed.sample;
    const m = cu.maps;
    const headers = sm ? sm.headers.map((x) => x.name) : [];
    const text = (obj, key, props) => h("input", Object.assign({ type: "text", value: obj[key] || "", oninput: (e) => { obj[key] = e.target.value; } }, props));
    // headerSelect chooses a column of the list; one the list lacks stays.
    const headerSelect = (value, onpick) => {
      const opts = [["", "（選ぶ）"]].concat(headers.map((x) => [x, x]));
      if (value && !headers.includes(value)) opts.push([value, value + "（一覧にありません）"]);
      return select(opts, value || "", onpick);
    };
    const close = async () => {
      if (!(await ask("列対応を閉じますか", "保存していない変更は消えます。", "閉じる"))) return;
      cu.edit = null;
      render();
    };
    const tryIt = () => run("試しに読んでいます", async () => { ed.tryv = await api().ListTry(cleanRule(r)); st.jump = "map-try"; });
    const save = () => run("列対応を保存しています", async () => {
      ed.path = await api().ListSave(ed.path, cleanRule(r));
      cu.msg = `保存しました: ${ed.path}`;
      cu.maps = await api().Mappings();
    });

    const out = [
      boxes("info", cu.msg ? [cu.msg] : []),
      h("h2", null, ed.path ? `列対応 ${ed.path}` : "新しい列対応（まだ保存していません）"),
      h("div", { class: "row" },
        button("試しに読む", tryIt, { id: "map-try-button" }),
        button("保存", save, { class: "primary", id: "map-save" }),
        button("閉じる", close)),
      boxes("error", ed.err ? [ed.err] : []),

      h("h3", null, "一覧のどこを読むか"),
      h("div", { class: "form-grid" },
        h("label", null, "一覧のファイル", text(r, "file", { onchange: resample })),
        h("label", null, "シート", sm && sm.sheets.length
          ? select(sm.sheets.map((x) => [x, x]), sm.sheet, (x) => { r.sheet = x; r.sheetIndex = 0; r.headerRow = 0; resample(); })
          : text(r, "sheet", { onchange: resample })),
        h("label", null, "見出しの行", h("input", { type: "number", min: "1", id: "map-header-row", value: r.headerRow || "", onchange: (e) => { r.headerRow = Number(e.target.value) || 0; resample(); } })),
        h("label", null, "値の最初の行", h("input", { type: "number", min: "0", value: r.dataRow || "", placeholder: "見出しの次の行", onchange: (e) => { r.dataRow = Number(e.target.value) || 0; resample(); } })),
        h("label", null, "表の左端の列", text(r, "startCol", { placeholder: "自動（A、D、AB…）", onchange: resample })),
        h("label", null, "一覧の機器名・IF 名", select([["new", "新しい機器の名前"], ["old", "今の（旧の）機器の名前"]], r.names || "new", (x) => { r.names = x; })),
        h("label", null, "結合セル", select([["", "範囲の全部のセルに値がある"], ["1", "左上のセルだけに値がある"]], r.mergeOnlyFirst ? "1" : "", (x) => { r.mergeOnlyFirst = x === "1"; resample(); }))),
    ];

    if (sm) {
      const width = Math.max(1, ...sm.grid.map((row) => row.length));
      const cols = [];
      for (let c = 0; c < width; c++) cols.push(c);
      const letter = (n) => { let s = ""; for (n += 1; n > 0; n = Math.floor((n - 1) / 26)) s = String.fromCharCode(65 + ((n - 1) % 26)) + s; return s; };
      out.push(
        boxes("warn", sm.err ? [sm.err] : []),
        boxes("warn", sm.warnings),
        h("p", { class: "dim small" }, `${sm.file} のシート「${sm.sheet}」の上の ${sm.grid.length} 行です（全部で ${sm.rows} 行）。色の付いた行が見出しです。行番号を押すと、その行を見出しの行にします。`),
        h("div", { class: "table-wrap sample", id: "map-sample" }, h("table", null,
          h("thead", null, h("tr", null, h("th", null, ""), cols.map((c) => h("th", null, letter(c))))),
          h("tbody", null, sm.grid.map((row, i) => h("tr", { class: cls(i + 1 === Number(r.headerRow) && "hdr", i + 1 < Number(r.headerRow) && "above") },
            h("th", { class: "pick", onclick: () => { r.headerRow = i + 1; r.dataRow = 0; resample(); } }, String(i + 1)),
            cols.map((c) => h("td", { title: row[c] || null }, row[c] || "")))))))
      );
    }

    // The conditions that leave lines out.
    out.push(
      h("h3", null, "読まない行（除外）"),
      r.exclude.map((c, i) => h("div", { class: "row" },
        headerSelect(c.column, (x) => { c.column = x; }),
        select(m.ops.map((o) => [o, OPS[o] || o]), c.op, (x) => { c.op = x; render(); }),
        c.op === "empty" || c.op === "notempty" ? null : text(c, "value"),
        button("消す", () => { r.exclude.splice(i, 1); render(); }, { class: "link" }))),
      h("div", { class: "row" }, button("条件を足す", () => { r.exclude.push({ column: "", op: "equals", value: "" }); render(); })));

    // The rows of the design data a line gives.
    out.push(
      h("h3", null, "一覧の 1 行から作る行（行き先）"),
      h("p", { class: "dim small" }, "一覧の 1 行から、機器の行・WAN 側の IF の行・LAN 側の IF の行…と、いくつでも作れます。★ はキーの列で、必ず要ります（一覧に無ければ「決まった値」で書きます。例: 「WAN 側 IP」の列しか無い一覧の IF 名）。キー以外の列に一覧から読めた値が無い行は、作りません。"));
    r.rows.forEach((tg, ti) => {
      const t = tableInfo(tg.table);
      const used = new Set(tg.columns.map((f) => f.to));
      const rows = [];
      tg.columns.forEach((f, fi) => {
        const c = (t && t.columns.find((x) => x.name === f.to)) || { name: f.to, title: f.to };
        const src = srcOf(f);
        const id = `${ti}/${fi}`;
        const colOpts = (t ? t.columns : []).filter((x) => x.name === f.to || !used.has(x.name))
          .map((x) => [x.name, (x.key ? "★ " : "") + x.title + (x.secret ? "（秘密）" : "")]);
        rows.push(h("tr", null,
          h("td", null, select(colOpts, f.to, (x) => { f.to = x; resample(); })),
          h("td", null, select(SOURCES, src, (x) => { f._src = x; render(); })),
          h("td", null, src === "column"
            ? headerSelect(f.column, (x) => { f.column = x; if (c.secret) resample(); })
            : src === "template"
              ? text(f, "template", { placeholder: "{見出し}/{見出し}" })
              : text(f, "value", { placeholder: c.values && c.values.length ? c.values.join(" | ") : "" })),
          h("td", null,
            h("span", { class: "small" }, f.steps.length ? f.steps.map((s) => STEPS[s.type] || s.type).join(" → ") : "（なし）"), " ",
            button(ed.open === id ? "たたむ" : "直す", () => { ed.open = ed.open === id ? "" : id; render(); }, { class: "link" })),
          h("td", null, text(f, "default")),
          h("td", null, c.key ? null : button("消す", () => { tg.columns.splice(fi, 1); ed.open = ""; render(); }, { class: "link" }))));
        if (ed.open === id) rows.push(h("tr", { class: "steps" }, h("td", { colspan: "6" }, stepEditor(f), c.note ? h("p", { class: "dim small" }, `列「${c.title}」（${c.type}）: ${c.note}`) : null)));
      });
      const free = t ? t.columns.filter((x) => !used.has(x.name)) : [];
      out.push(h("div", { class: "target-card" },
        h("div", { class: "row" },
          h("span", null, "表"),
          select(m.tables.map((x) => [x.name, x.title]), tg.table, (x) => { tg.table = x; tg.columns = []; tg.columns = keyFields(x, r); ed.open = ""; resample(); }),
          h("span", null, "ラベル"), text(tg, "label", { placeholder: "例: WAN 側（人のためのメモ）" }),
          button("この行き先を消す", () => { r.rows.splice(ti, 1); ed.open = ""; render(); }, { class: "link" })),
        h("div", { class: "table-wrap" }, h("table", { class: "fields" },
          h("thead", null, h("tr", null, ["行き先の列", "値の元", "値", "加工", "空のときの値", ""].map((x) => h("th", null, x)))),
          h("tbody", null, rows))),
        h("div", { class: "row" }, button("列を足す", () => { tg.columns.push(newField(free[0].name)); render(); }, { disabled: !free.length }))));
    });
    out.push(h("div", { class: "row" }, button("行き先を足す", () => {
      r.rows.push({ table: "interfaces", label: "", columns: keyFields("interfaces", r) });
      render();
    }, { id: "map-add-target" })));

    if (ed.tryv) out.push(tryView(ed.tryv));
    return out;
  }

  // stepEditor edits the processing of one field.
  function stepEditor(f) {
    const m = st.cus.maps;
    const text = (obj, key, placeholder) => h("input", { type: "text", value: obj[key] || "", placeholder, oninput: (e) => { obj[key] = e.target.value; } });
    const params = (s) => {
      switch (s.type) {
        case "split":
          return [text(s, "sep", "区切りの文字"),
            select([["first", "最初の区切りで"], ["last", "最後の区切りで"]], s.pos || "first", (x) => { s.pos = x; }),
            select([["right", "右を残す"], ["left", "左を残す"]], s.keep || "right", (x) => { s.keep = x; }),
            select([["", "区切りが無ければそのまま"], ["empty", "区切りが無ければ空"]], s.miss || "", (x) => { s.miss = x; })];
        case "replace":
          return [text(s, "from", "置き換える文字"), "→", text(s, "to", "置き換えた後（空なら削る）")];
        case "regex":
          return [text(s, "pattern", "正規表現"), "→", text(s, "to", "置き換えた後（$1 が使えます）")];
        case "map":
          return [h("textarea", {
            rows: "4", placeholder: "1 行に 1 つ: 元の値 => 置き換える値",
            oninput: (e) => {
              s.rows = e.target.value.split("\n").filter((l) => l.includes("=>")).map((l) => {
                const i = l.indexOf("=>");
                return { match: "equals", from: l.slice(0, i).trim(), to: l.slice(i + 2).trim() };
              });
            },
          }, (s.rows || []).map((x) => `${x.from} => ${x.to}`).join("\n")),
          select([["keep", "当たらなければそのまま"], ["empty", "当たらなければ空"]], s.miss || "keep", (x) => { s.miss = x; })];
      }
      return [];
    };
    const add = { type: m.steps[0] };
    return h("div", { class: "steps-box" },
      f.steps.map((s, i) => h("div", { class: "row" },
        h("span", { class: "dim" }, `${i + 1}.`),
        select(m.steps.map((x) => [x, STEPS[x] || x]), s.type, (x) => { f.steps[i] = { type: x, rows: [] }; render(); }),
        params(s),
        button("消す", () => { f.steps.splice(i, 1); render(); }, { class: "link" }))),
      h("div", { class: "row" },
        select(m.steps.map((x) => [x, STEPS[x] || x]), add.type, (x) => { add.type = x; }),
        button("加工を足す", () => { f.steps.push({ type: add.type, rows: [] }); render(); })),
      h("p", { class: "dim small" }, "上から順に当てます。結果は「試しに読む」で確かめます。"));
  }

  function tryView(v) {
    const out = [h("h3", { id: "map-try" }, "試しに読んだ結果（何も書いていません）")];
    if (v.faults.length) {
      out.push(h("div", { class: "box error" }, "この列対応には誤りがあります。直してから、もう一度押してください:\n" + v.faults.join("\n")));
      return out;
    }
    out.push(
      h("p", null, `${v.file} のシート「${v.sheet}」: ${v.lines} 行（うち除外 ${v.excluded} 行）から、設計データの行を ${v.rows} 行作りました。`),
      boxes("warn", v.missing.length ? [`一覧に、見出し「${v.missing.join("」「")}」がありません。その列は空として読みました。`] : []),
      boxes("warn", v.warnings));
    if (v.problems.rows.length) {
      out.push(h("h3", null, `読めなかった値（${v.problems.rows.length}）`), tableView(v.problems, { id: "customer/try/problems" }));
    }
    v.tables.forEach((t, i) => out.push(h("h3", null, t.name), tableView(t, { id: "customer/try/" + i })));
    return out;
  }

  function comparePage() {
    const cp = st.cmp;
    const df = st.diff;
    const v = cp.view;
    const sources = [["old", SRC.old], ["new", SRC.new], ["book", SRC.book]];
    const compare = () => run(`${SRC[cp.a]}と${SRC[cp.b]}を比べています`, async () => {
      cp.view = null;
      st.mig.msg = "";
      cp.view = await api().Compare(cp.a, cp.b);
      cp.tab = 0;
    }, true);
    const out = [h("h1", null, "比較"), h("h2", null, "設計データの比較")];
    if (!st.cas) {
      out.push(h("p", { class: "dim" }, "案件を開くと使えます。"));
    } else {
      out.push(
        h("p", { class: "dim" }, "2 つを設計データ（表）にして比べ、比較レポートを書きます。書き方や順番の違いは差になりません。"),
        h("div", { class: "row" },
          h("span", null, "A"), select(sources, cp.a, (x) => { cp.a = x; }),
          h("span", null, "B"), select(sources, cp.b, (x) => { cp.b = x; }),
          button("比べる", compare, { class: "primary" })));
      if (v) {
        const t = v.tables[cp.tab];
        const mg = st.mig;
        if (mg.by == null || !mg.by.trim()) mg.by = v.by;
        const again = async () => { cp.view = await api().Compare(cp.a, cp.b); };
        out.push(
          h("h2", null, v.title),
          counts(v.counts),
          boxes("info", mg.msg ? [mg.msg + "。比べ直して、レポートも新しく書きました。"] : []),
          boxes("warn", v.notes),
          h("div", { class: "row" },
            button("レポートを Excel で開く", () => show(v.report.xlsx)),
            button("レポートをブラウザで開く", () => show(v.report.html))),
          h("div", { class: "row" }, seg(v.tables.map((x, i) => [i, `${x.name}（${x.rows.length}）`]), cp.tab, (i) => { cp.tab = i; render(); })),
          t ? migTable(t, v.items, "compare/" + t.name, again) : null);
      }
    }

    const pick = async (side) => {
      try {
        const path = await api().PickFile(side === "a" ? "比べるファイル A" : "比べるファイル B");
        if (path) {
          df[side] = path;
          df.view = null;
        }
      } catch (e) {
        st.error = errText(e);
      }
      render();
    };
    const diff = () => run("テキストを比べています", async () => {
      df.view = df.mode === "tree" ? await api().Tree(df.a, df.b, df.stamps, df.secrets) : await api().Diff(df.a, df.b);
      df.shown = LINES;
    });
    out.push(treeAllPart());
    out.push(sitesPart());
    out.push(
      h("h2", null, "ファイル 2 つの比較"),
      h("p", { class: "dim" }, "2 つのファイルを比べます。案件を開いていなくても使えます。「行ごと」はテキストをそのまま、「階層を見て」はコンフィグを interface などのブロックごとに突き合わせて比べます。"),
      h("div", { class: "box warn" }, "この比較は、ファイルの文字をそのまま画面に出します（秘密の値も伏せません）。"),
      h("div", { class: "row" }, h("span", null, "A"), button("選ぶ", () => pick("a")), h("span", { class: "path" }, df.a || "（まだ選んでいません）")),
      h("div", { class: "row" }, h("span", null, "B"), button("選ぶ", () => pick("b")), h("span", { class: "path" }, df.b || "（まだ選んでいません）")),
      h("div", { class: "row" },
        seg([["line", "行ごと"], ["tree", "階層を見て"]], df.mode, (m) => { df.mode = m; df.view = null; render(); }),
        df.mode === "tree" ? treeChecks(df) : null,
        button("比べる", diff, { class: "primary", id: "diff-run", disabled: !df.a || !df.b })));
    if (df.view) out.push(df.mode === "tree" ? treeView(df.view, df) : diffView(df));
    return out;
  }

  // treeChecks is the two choices of what a structured comparison leaves out.
  function treeChecks(o) {
    return [
      h("label", null, h("input", { type: "checkbox", checked: o.stamps, onchange: (e) => { o.stamps = e.target.checked; } }), " 時刻の行を無視"),
      h("label", null, h("input", { type: "checkbox", checked: o.secrets, onchange: (e) => { o.secrets = e.target.checked; } }), " 証明書・暗号化値を無視"),
    ];
  }

  const SIDES = [["old", "旧コンフィグ（input/old）"], ["new", "新コンフィグ（input/new）"], ["logs", "ログ（input/logs）"], ["generated", "ブックから生成したコンフィグ"], ["folder", "フォルダを選ぶ"]];

  // treeAllPart is the structured comparison of every device of two sides.
  function treeAllPart() {
    const tr = st.tree;
    const out = [h("h2", null, "階層を見た比較（全台まとめて）")];
    if (!st.cas) {
      out.push(h("p", { class: "dim" }, "案件を開くと使えます。"));
      return out;
    }
    const side = (k) => (tr[k] === "folder" ? tr.dirs[k] : tr[k]);
    const pickDir = async (k) => {
      try {
        const dir = await api().PickFolder(k === "a" ? "比べるフォルダ A" : "比べるフォルダ B");
        if (dir) {
          tr.dirs[k] = dir;
          tr.view = null;
          tr.one = null;
        }
      } catch (e) {
        st.error = errText(e);
      }
      render();
    };
    const go = () => run("コンフィグを全台まとめて比べています", async () => {
      tr.view = null;
      tr.one = null;
      tr.view = await api().TreeAll(side("a"), side("b"), tr.stamps, tr.secrets);
    }, true);
    const sideRow = (k, label) => h("div", { class: "row" },
      h("span", null, label), select(SIDES, tr[k], (x) => { tr[k] = x; tr.view = null; tr.one = null; render(); }),
      tr[k] === "folder" ? [button("選ぶ", () => pickDir(k)), h("span", { class: "path" }, tr.dirs[k] || "（まだ選んでいません）")] : null);
    out.push(
      h("p", { class: "dim" }, "2 つの側で同じホスト名の機器を組にして、コンフィグを interface などのブロックごとに突き合わせて比べ、レポートを書きます。同じメーカーどうしの、作業前と作業後・旧と新・生成したものと手で書いたものに使います。"),
      sideRow("a", "A"), sideRow("b", "B"),
      h("div", { class: "row" }, treeChecks(tr), button("比べる", go, { class: "primary", id: "tree-run", disabled: !side("a") || !side("b") })));
    const v = tr.view;
    if (!v) return out;
    const open = (i) => run("行を出しています", async () => {
      tr.one = await api().TreeOf(i);
      tr.shown = LINES;
      st.jump = "tree-dev";
    });
    out.push(
      h("h3", null, v.title),
      counts(v.counts),
      boxes("warn", v.notes),
      h("div", { class: "row" },
        button("レポートを Excel で開く", () => show(v.report.xlsx)),
        button("レポートをブラウザで開く", () => show(v.report.html))),
      h("div", { class: "table-wrap" }, h("table", { id: "tree-devices" },
        h("thead", null, h("tr", null, ["機器", "A のファイル", "B のファイル", "追加", "削除", "変更", "順序", "状態", ""].map((x) => h("th", null, x)))),
        h("tbody", null, v.devices.map((d, i) => h("tr", { class: d.differs ? "mark-warn" : null },
          h("td", null, d.device), h("td", null, d.a), h("td", null, d.b),
          d.both ? [d.added, d.removed, d.changed, d.moved].map((n) => h("td", null, String(n))) : h("td", { colspan: "4" }, ""),
          h("td", null, !d.both ? (d.a ? "A にだけあります" : "B にだけあります") : d.differs ? "違いがあります" : "違いはありません", d.notes ? h("div", { class: "dim small" }, d.notes) : null),
          h("td", null, d.both && d.differs ? button("行を見る", () => open(i)) : null)))))));
    if (v.failed.rows.length) out.push(h("h3", null, `比べなかったファイル（${v.failed.rows.length}）`), tableView(v.failed, { id: "tree/failed" }));
    if (tr.one) out.push(h("h3", { id: "tree-dev" }, tr.one.title), treeView(tr.one, tr));
    return out;
  }

  // sitesPart is the list of many sites: the devices of one source side by
  // side.
  function sitesPart() {
    const si = st.sites;
    const out = [h("h2", null, "多拠点の一覧")];
    if (!st.cas) {
      out.push(h("p", { class: "dim" }, "案件を開くと使えます。"));
      return out;
    }
    const sources = [["book", "パラメータブック"], ["old", "旧コンフィグ（input/old）"], ["new", "新コンフィグ（input/new）"], ["logs", "ログ（input/logs）"]];
    const go = () => run("機器を横に並べています", async () => {
      si.view = null;
      si.view = await api().Sites(si.source);
      si.tab = si.view.tables.length > 1 ? 1 : 0;
    }, true);
    out.push(
      h("p", { class: "dim" }, "1 つの表を機器の横並びで見ます。全台で同じ値の項目は畳み、いちばん多い値と違う機器だけを出します。外れ値の拠点を見つけるのに使います。"),
      h("div", { class: "row" },
        h("span", null, "元"), select(sources, si.source, (x) => { si.source = x; si.view = null; render(); }),
        button("並べる", go, { class: "primary", id: "sites-run" })));
    const v = si.view;
    if (!v) return out;
    const t = v.tables[si.tab];
    out.push(
      h("h3", null, v.title),
      counts(v.counts),
      boxes("warn", v.notes),
      h("div", { class: "row" },
        button("レポートを Excel で開く", () => show(v.report.xlsx)),
        button("レポートをブラウザで開く", () => show(v.report.html))),
      // A case has many tables: a list to choose from, not a row of buttons.
      h("div", { class: "row", id: "sites-tabs" }, h("span", null, "見る一覧"),
        select(v.tables.map((x, i) => [String(i), `${x.name}（${x.rows.length}）`]), String(si.tab), (i) => { si.tab = Number(i); render(); })),
      t ? [t.lead ? h("p", { class: "dim small" }, t.lead) : null, tableView(t, { id: "sites/" + t.name })] : null);
    return out;
  }

  // treeView draws two configurations compared block by block: A on the
  // left, B on the right.
  function treeView(d, keep) {
    const head = h("div", { class: "dim small" }, `A: ${d.a}　B: ${d.b}`);
    const notes = boxes("warn", d.notes);
    if (d.equal) return h("div", null, head, notes, h("div", { class: "box info" }, "違いはありません。" + d.leftOut + "。"));
    const text = (ps) => ps.map((p) => (p.h ? h("mark", null, p.t) : p.t));
    const mark = (r) => (r.noise ? "" : r.op === "chg" ? (r.moved ? "変更・順序" : "変更") : r.op === "del" ? "A だけ" : r.op === "add" ? "B だけ" : r.moved ? "順序" : "");
    const rows = d.rows.slice(0, keep.shown).map((r) => r.op === "gap"
      ? h("div", { class: "dl gap" }, "… " + r.text + " …")
      : h("div", { class: cls("tl", r.op, r.moved && "moved", r.noise && "noise", r.head && "head") },
        h("span", { class: "no" }, r.a ? String(r.a) : ""),
        h("span", { class: "tx a" }, text(r.ta)),
        h("span", { class: "no" }, r.b ? String(r.b) : ""),
        h("span", { class: "tx b" }, text(r.tb)),
        h("span", { class: "kind" }, mark(r))));
    return h("div", null, head, notes,
      h("div", { class: "row" },
        h("span", null, `B だけの行 ${d.added}、A だけの行 ${d.removed}、変更 ${d.changed}、順序 ${d.moved}`),
        h("span", { class: "dim small" }, d.leftOut),
        d.rows.length > keep.shown ? [h("span", { class: "dim small" }, `${d.rows.length} 行のうち ${keep.shown} 行を出しています`), button("さらに表示", () => { keep.shown += LINES; render(); })] : null),
      h("div", { class: "diff tree" }, rows));
  }

  function diffView(df) {
    const d = df.view;
    if (d.equal) return h("div", { class: "box info" }, "2 つのファイルは同じです。");
    const sign = { same: " ", del: "-", add: "+" };
    const lines = d.lines.slice(0, df.shown).map((l) => l.op === "gap"
      ? h("div", { class: "dl gap" }, "… " + l.text + " …")
      : h("div", { class: "dl " + l.op },
        h("span", { class: "no" }, l.a ? String(l.a) : ""),
        h("span", { class: "no" }, l.b ? String(l.b) : ""),
        h("span", { class: "sign" }, sign[l.op]),
        h("span", { class: "tx" }, l.text)));
    return h("div", null,
      h("div", { class: "row" },
        h("span", null, `A にだけある行 ${d.removed} 行、B にだけある行 ${d.added} 行`),
        d.lines.length > df.shown ? [h("span", { class: "dim small" }, `${d.lines.length} 行のうち ${df.shown} 行を出しています`), button("さらに表示", () => { df.shown += LINES; render(); })] : null),
      h("div", { class: "diff" }, lines));
  }

  // ---- the start ----

  async function boot() {
    $("busy-stop").addEventListener("click", () => {
      $("busy-stop").disabled = true;
      $("busy-text").textContent = "止めています…";
      api().Cancel();
    });
    try {
      window.runtime.EventsOn("progress", onProgress);
      const s = await api().Start();
      setState(s.state);
      st.error = s.notice || "";
      if (st.cas) {
        st.home = await api().Home();
        st.page = "home";
      }
    } catch (e) {
      st.error = errText(e);
    }
    render();
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", boot);
  else boot();
})();
