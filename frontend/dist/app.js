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
    create: { name: "", kind: "migration" },
    imp: { source: "old", views: {}, tab: {}, lines: {}, pick: {}, onlyBad: false },
    jdg: newJdg(),
    folds: {}, // the folded rows of buttons that are open (spec 14.18.8)
    map: { view: null, tab: "open", pin: "", book: null, replace: false, redo: null, link: null }, // link: the panel that ties the ports (spec 14.20); pin: the part the next visit opens (spec 14.17.2); redo: making the proposals of 新旧対照表（ポート） again (spec 14.11.4)
    cmp: { a: "old", b: "new", view: null, tab: 0 },
    diff: { a: "", b: "", view: null, shown: LINES, mode: "line", stamps: true, secrets: true, layout: "stack", cur: -1 },
    tree: newTree(),
    sites: { source: "book", view: null, tab: 1 },
    res: newRes(),
    mig: newMig(),
    cus: newCus(),
    fil: newFil(),
    xtra: newXtra(), // the extra configurations (spec 14.11.3)
    logs: newLogs(),
    exp: { mask: "show", view: null },
    nb: { from: "", to: "", said: [] },
    dlt: { side: "old", devices: [], view: null, run: null, sel: "" },
    edit: { list: null, want: null, msg: "", prefs: {}, find: { open: false, replace: false, q: "", r: "", cs: false, whole: false, scope: "sheet", onlySel: false, range: null, said: "", all: 0, focus: false } }, // as newEdit() makes it
    filters: {},
    srch: newSrch(),
    upd: { found: null, busy: "", error: "", path: "" },
    readOnly: "", // why the open case is only read (spec 14.8.3)
    cx: { noSecrets: false, seal: true, pw: "", pw2: "", view: null }, // the case written out (spec 14.9)
    cks: null, // the settings of the checks: { list, set, q, only, msg } (spec 14.11.5)
    caseNote: "", // what the home tells once of a case read back
    zoom: 100,
    added: null, // what adding files told: { source, text } (spec 14.10.1)
    next: null, // the one thing to do next (spec 14.10.4)
    nextFor: null, // the home view the next step was asked for
    hint: "", // the id of the control the next step points at
    nextSeq: 0, // the asking of the next step that is the newest
    xl: null, // the Excel files whose columns are picked: { source, queue, book, picked, err } (spec 14.16.4)
    rn: null, // the new name of the case being typed: { name } (spec 14.16.3)
  };

  // newSrch is the search over the files of a case with nothing to show
  // (spec 14.8.8).
  function newSrch() {
    return { q: "", regex: false, cs: false, misses: false, scopes: null, out: "", view: null, file: null, fq: "", fhits: [], fcur: -1 };
  }

  // newXtra is the screen of the extra configurations with nothing open
  // (spec 14.11.3).
  function newXtra() {
    return { places: null, name: "", text: "", saved: "", isNew: false, kind: "device", target: "", from: null, msg: "", notes: [] };
  }

  // newJdg is the screen of the unhandled lines with nothing to show. changed
  // tells the sides that were judged after their last import.
  function newJdg() {
    return { source: "old", views: {}, sel: {}, changed: {}, only: true, scope: "all", picked: {}, verdict: "manual", reason: "", by: null, carry: {}, q: "", memo: false, many: { verdict: "not_needed", reason: "" } };
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
    return { view: null, tab: "devices", ctab: 0, device: "", sev: "", state: "", sel: "", onlyBad: false, scope: "", reason: "", by: null,
      many: {}, manyReason: "", // the findings picked to approve together (spec 14.10.8)
      conf: null, confHost: "", confPick: null, // the configuration shown (spec 14.10.5)
      lastTried: false }; // the result of the newest output was asked for (spec 14.10.6)
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
    const began = Date.now();
    try {
      for (;;) {
        try {
          await fn();
          // A step the user waited for tells it is over (spec 14.8.10).
          if (Date.now() - began >= TOAST_AFTER) toast(doneText(label));
        } catch (e) {
          const text = errText(e);
          // Excel holds the book: wait for it to let go, then do it again
          // (spec 14.10.7).
          if (text.includes(EXCEL_OPEN) && st.cas && (await waitBook(label, canStop))) continue;
          st.error = text;
          toast(st.error, "err");
        }
        break;
      }
    } finally {
      busy = false;
      hideBusy();
      render();
      if (st.cas) askNext();
    }
  }

  // ---- waiting for Excel to let the book go (spec 14.10.7) ----

  const EXCEL_OPEN = "Excel を閉じてください";
  let bookWait = null; // { stop } while the page waits

  // waitBook shows that the step waits for Excel, and tells true when the
  // book can be written (the step is then done again), false when the user
  // stopped waiting.
  async function waitBook(label, canStop) {
    clearTimeout(busyTimer);
    $("busy-label").textContent = "Excel でパラメータブックを閉じると、続けます";
    $("busy-text").textContent = `「${label}」は、パラメータブックを書くところで止まっています。Excel でブックを閉じてください（保存してから閉じてください）。まだ何も書いていません。`;
    $("busy-bar").style.width = "0";
    $("busy-stop").textContent = "やめる";
    $("busy-stop").hidden = false;
    $("busy-stop").disabled = false;
    $("busy").hidden = false;
    bookWait = { stop: false };
    const w = bookWait;
    try {
      for (;;) {
        await new Promise((r) => setTimeout(r, 1000));
        if (w.stop) return false;
        let ok = false;
        try {
          ok = await api().BookWritable();
        } catch (e) {
          return false;
        }
        if (ok) {
          showBusy(label, canStop);
          $("busy").hidden = false;
          return true;
        }
      }
    } finally {
      bookWait = null;
      $("busy-stop").textContent = "止める";
    }
  }

  // ---- adding files to input/ (spec 14.10.1) ----

  const ADD_WHERE = { old: "旧コンフィグ", new: "新コンフィグ", logs: "検証機のログ", customer: "外部一覧" };

  // dropZone is a place files and folders can be dropped on, with the
  // button that chooses them.
  function dropZone(source) {
    const said = st.added && st.added.source === source ? st.added.text : "";
    return h("div", { class: "drop", "data-drop": source, style: "--wails-drop-target: drop", id: "drop-" + source },
      h("div", { class: "row" },
        h("span", { class: "dim" }, `ここにファイルかフォルダを落とすと、input/${source} に写します（元のファイルはそのままです）。`),
        button("ファイルを足す", () => pickAdd(source), { id: "add-" + source })),
      said ? h("div", { class: "box info small" }, button("閉じる", () => { st.added = null; render(); }, { class: "link close" }), said) : null);
  }

  async function pickAdd(source) {
    let paths = null;
    try {
      paths = await api().PickFiles(`${ADD_WHERE[source]}のファイルを選ぶ（input/${source} に写します）`);
    } catch (e) {
      toast(errText(e), "err");
      return;
    }
    if (paths && paths.length) await addFiles(source, paths);
  }

  // addFiles copies the files; a name that is there with another content is
  // written over only when the user says so.
  async function addFiles(source, paths) {
    const label = `${ADD_WHERE[source]}のファイルを写しています`;
    let v = null;
    await run(label, async () => { v = await api().AddFiles(source, paths, false); });
    if (!v) return;
    const c = await clashes(source, v, label, () => api().AddFiles(source, paths, true));
    // Excel files are not copied: their columns are picked (spec 14.16.4).
    if (v.excel && v.excel.length) {
      st.xl = { source, queue: v.excel.slice(), book: null, picked: {}, err: "", wrote: 0 };
      xlLoad("", 0);
    }
    await told(source, c.said, v.copied + c.wrote);
  }

  // clashes asks whether the files of the same name with another content are
  // written over, and tells what was done and how many files it wrote.
  async function clashes(source, v, label, again) {
    let said = v.message, wrote = 0;
    if (v.clash.length) {
      const list = v.clash.slice(0, 10).join("\n") + (v.clash.length > 10 ? `\nほか ${v.clash.length - 10}` : "");
      const yes = await ask("同じ名前のファイル", `input/${source} に、同じ名前で中身の違うファイルが ${v.clash.length} 個あります。\n${list}\n\n上書きしますか？（元のファイルは state/backup に控えます）`, "上書きする");
      if (yes) {
        let w = null;
        await run(label, async () => { w = await again(); });
        if (w) {
          said = (said ? said + "\n" : "") + `上書きしました: ${w.copied} ファイル（元のファイルは state/backup に控えました）`;
          wrote = w.copied;
        }
      } else {
        said = (said ? said + "\n" : "") + `同じ名前で中身の違う ${v.clash.length} ファイルは、写していません`;
      }
    }
    return { said, wrote };
  }

  // told shows what adding files did, and counts the steps again. Files that
  // were written are imported right away, on the screen of the import (spec
  // 14.19.3); while Excel files wait for their columns, the import waits for
  // them.
  async function told(source, said, wrote) {
    if (st.xl && st.xl.source === source) {
      st.xl.wrote += wrote || 0;
      wrote = 0;
    }
    if (wrote > 0 && (source === "old" || source === "new")) {
      if (said) toast(said.split("\n")[0]);
      st.added = null;
      st.imp.source = source;
      goTo("import");
      return;
    }
    st.added = { source, text: said };
    try {
      st.home = await api().Home();
    } catch (e) {
      // the home screen is drawn again the next time
    }
    if (said) toast(said.split("\n")[0]);
    render();
    loadNext();
  }

  // ---- configurations kept in Excel, one device a column (spec 14.16.4) ----

  async function xlLoad(sheet, header) {
    const x = st.xl;
    if (!x || !x.queue.length) return;
    try {
      x.book = await api().ExcelBook(x.queue[0], sheet || "", header || 0);
      x.err = "";
    } catch (e) {
      x.book = null;
      x.err = errText(e);
    }
    x.picked = {};
    render();
  }

  // xlNext goes on to the next Excel file, or closes the box.
  function xlNext() {
    const x = st.xl;
    if (!x) return;
    x.queue.shift();
    if (!x.queue.length) {
      st.xl = null;
      // Files put before the Excel ones are imported now.
      if (x.wrote > 0) {
        st.imp.source = x.source;
        goTo("import");
        return;
      }
      render();
      return;
    }
    xlLoad("", 0);
  }

  async function xlPut() {
    const x = st.xl, b = x && x.book;
    if (!b) return;
    const cols = b.columns.filter((c) => x.picked[c.col]).map((c) => c.col);
    if (!cols.length) return;
    const label = "Excel の列からコンフィグを作っています";
    let v = null;
    await run(label, async () => { v = await api().AddExcel(x.source, b.path, b.sheet, b.header, cols, false); });
    if (!v) return;
    const c = await clashes(x.source, v, label, () => api().AddExcel(x.source, b.path, b.sheet, b.header, cols, true));
    const source = x.source;
    const wrote = x.wrote + v.copied + c.wrote;
    x.wrote = 0;
    xlNext();
    await told(source, c.said, wrote);
  }

  function excelCard() {
    const x = st.xl, b = x.book;
    const n = b ? b.columns.filter((c) => x.picked[c.col]).length : 0;
    const pickAll = (on) => { b.columns.forEach((c) => { if (on && c.lines > 0) x.picked[c.col] = true; else delete x.picked[c.col]; }); render(); };
    const fileName = b ? b.file : x.queue[0].split(/[\\/]/).pop();
    return h("div", { class: "card excel-pick", id: "excel-pick" },
      h("h2", null, `Excel の列からコンフィグを読み込む（${ADD_WHERE[x.source]}）`),
      h("p", { class: "dim" }, "1 列を 1 台のコンフィグとして、選んだ列ごとにコンフィグのファイルを作って読み込みます。見出しの下のセルを、上から 1 セル 1 行でそのまま写します（字下げもそのまま）。メモの行も写すので、取り込んだ後に「読めなかった行」で「メモらしい行だけ」に絞り、まとめて「要らない」を付けてください。ホスト名はコンフィグの hostname の行から取り、無ければファイル名になります。"),
      h("div", { class: "row" }, h("b", null, fileName),
        x.queue.length > 1 ? h("span", { class: "dim" }, `（このあと Excel があと ${x.queue.length - 1} 個）`) : null),
      boxes("error", x.err ? [x.err] : []),
      b ? [
        h("div", { class: "form-grid" },
          h("label", null, "シート", Object.assign(select(b.sheets.map((s) => [s, s]), b.sheet, (s) => xlLoad(s, 0)), { id: "excel-sheet" })),
          h("label", null, "見出しの行", h("input", { type: "number", min: "1", id: "excel-header", value: b.header, onchange: (e) => xlLoad(b.sheet, Number(e.target.value) || 0) }))),
        h("p", { class: "dim small" }, `シート「${b.sheet}」は ${b.rows} 行です。見出しの行の、値のある列が並びます。見出しの行が違うときは、数を変えてください。`),
        b.columns.length ? h("div", { class: "table-wrap" }, h("table", null,
          h("thead", null, h("tr", null, h("th", null, ""), h("th", null, "列"), h("th", null, "見出し"), h("th", null, "行数"), h("th", null, "初めの行"), h("th", null, "作るファイル"))),
          h("tbody", null, b.columns.map((c) => h("tr", null,
            h("td", null, h("input", { type: "checkbox", "data-col": c.col, checked: !!x.picked[c.col], disabled: c.lines === 0,
              onchange: (e) => { if (e.target.checked) x.picked[c.col] = true; else delete x.picked[c.col]; render(); } })),
            h("td", null, c.letter),
            h("td", null, c.header || h("span", { class: "dim" }, "（空）")),
            h("td", null, String(c.lines)),
            h("td", { class: "sample-lines" }, c.sample.join("\n")),
            h("td", { class: "path" }, c.name)))))) : h("div", { class: "box info" }, "見出しの行の下に、値のある列がありません。見出しの行を変えてください。"),
      ] : null,
      h("div", { class: "row" },
        b ? button("全部選ぶ", () => pickAll(true)) : null,
        b ? button("全部外す", () => pickAll(false)) : null,
        b ? button(`選んだ ${n} 列のコンフィグを読み込む`, xlPut, { class: "primary", id: "excel-put", disabled: n === 0 }) : null,
        button(x.queue.length > 1 ? "このファイルはやめて次へ" : "やめる", xlNext, { id: "excel-skip" })));
  }

  // onFileDrop takes files dropped on a place that takes them.
  function onFileDrop(x, y, paths) {
    const el = document.elementFromPoint(x, y);
    const zone = el && el.closest("[data-drop]");
    if (!zone || !paths || !paths.length || busy || !st.cas) return;
    addFiles(zone.dataset.drop, paths);
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

  // ---- toasts, tips and the zoom (spec 14.8.6, 14.8.10) ----

  const TOAST_AFTER = 2500; // a step that took this long (ms) tells it is over

  // toast shows a short message at the bottom right: an error for 9 s, any
  // other for 4 s.
  function toast(text, kind) {
    let box = $("toasts");
    if (!box) {
      box = h("div", { id: "toasts" });
      document.body.appendChild(box);
    }
    const el = h("div", { class: cls("toast", kind === "err" && "err") }, text);
    el.addEventListener("click", () => el.remove());
    box.appendChild(el);
    while (box.children.length > 4) box.firstChild.remove();
    setTimeout(() => el.remove(), kind === "err" ? 9000 : 4000);
  }

  // doneText turns the label of a step ("生成しています") into what it did.
  function doneText(label) {
    if (/ています$/.test(label)) return label.replace(/ています$/, "ました");
    return "終わりました（" + label + "）";
  }

  // tip is the ⓘ that shows a longer explanation when the mouse is on it.
  const tip = (text) => h("span", { class: "tip", tabindex: "0", "data-tip": text }, "ⓘ");

  // menu is a button that opens a short list of buttons; a click on one of
  // them closes it (spec 14.12.5).
  function menu(label, id, items) {
    const d = h("details", { class: "menu", id }, h("summary", null, label + " ▾"), h("div", { class: "menu-items" }, items));
    d.addEventListener("click", (e) => { if (e.target.closest("button")) d.open = false; });
    return d;
  }

  const ZOOMS = [70, 80, 90, 100, 110, 120, 130, 140, 150];

  function applyZoom(z) {
    st.zoom = z;
    document.body.style.zoom = z === 100 ? "" : String(z / 100);
  }

  // zoomBy changes the zoom of the page by steps (0 = back to 100 %) and
  // keeps it in the settings.
  function zoomBy(step) {
    let i = ZOOMS.indexOf(st.zoom);
    if (i < 0) i = 3;
    const z = step === 0 ? 100 : ZOOMS[Math.max(0, Math.min(ZOOMS.length - 1, i + step))];
    applyZoom(z);
    toast(`画面の拡大率 ${z}%`);
    api().SetZoom(z).catch(() => {});
    if (typeof gFit === "function" && G) gFit();
  }

  function zoomKey(e) {
    if (!(e.ctrlKey || e.metaKey) || e.altKey) return;
    if (e.key === "+" || e.key === "=" || e.key === ";" || e.code === "NumpadAdd") zoomBy(1);
    else if (e.key === "-" || e.code === "NumpadSubtract") zoomBy(-1);
    else if (e.key === "0" || e.code === "Numpad0") zoomBy(0);
    else return;
    e.preventDefault();
    e.stopPropagation();
  }

  function zoomWheel(e) {
    if (!e.ctrlKey) return;
    e.preventDefault();
    zoomBy(e.deltaY < 0 ? 1 : -1);
  }

  // ---- the frame ----

  function setState(s) {
    st.version = s.version;
    // An error told for the case that was open before is not about this one.
    if ((st.cas && st.cas.dir) !== (s.case && s.case.dir)) document.querySelectorAll("#toasts .toast.err").forEach((x) => x.remove());
    st.cas = s.case;
    st.recent = s.recent;
    st.readOnly = s.readOnly || "";
    if (s.zoom) applyZoom(s.zoom);
    // The name of the case in the title of the window (spec 14.12.5).
    try {
      api().SetTitle(s.case ? s.case.name : "").catch(() => {});
    } catch (e) {
      // no window to tell
    }
  }

  // forgetViews drops what was drawn for the case that was open before.
  function forgetViews() {
    st.next = null; // the next step, the files told and the pointer are of that case
    st.steps = null;
    st.outs = null;
    st.dlt = { side: "old", devices: [], view: null, run: null, sel: "" };
    st.nextFor = null;
    st.added = null;
    st.hint = "";
    st.imp = { source: "old", views: {}, tab: {}, lines: {}, pick: {}, onlyBad: false };
    st.jdg = newJdg();
    st.map = { view: null, tab: "open", pin: "", book: null, replace: false, redo: null, link: null };
    st.xtra = newXtra();
    st.cks = null;
    st.cmp = { a: "old", b: "new", view: null, tab: 0 };
    st.tree = newTree();
    st.sites = { source: "book", view: null, tab: 1 };
    st.res = newRes();
    st.cus = newCus();
    st.fil = newFil();
    st.logs = newLogs();
    st.exp = { mask: "show", view: null };
    st.nb = { from: "", to: "", said: [] };
    st.edit = newEdit();
    G = null;
    st.filters = {};
    st.srch = newSrch();
    st.caseNote = "";
    st.mig = newMig();
    st.cx = { noSecrets: false, seal: true, pw: "", pw2: "", view: null };
  }

  async function opened(home) {
    forgetViews();
    st.place = null; // the list of the place is read again
    st.home = home;
    setState((await api().Start()).state);
    st.page = "home";
  }

  // go shows a page. The table of the book keeps what is being typed, and
  // its draft, before it goes out of sight.
  function go(page) {
    if (st.page === "edit" && G) {
      gLeave().then(() => goTo(page));
      return;
    }
    goTo(page);
  }

  function goTo(page) {
    // The pointer of the next step stays only on the screen it points into.
    if (st.hint && !(st.next && page === st.next.page)) st.hint = "";
    // What the home told once goes when the home is left (spec 14.12.2).
    if (st.page === "home" && page !== "home") {
      st.nb.said = [];
      st.exp.sampleMsg = "";
      st.exp.done = null;
      st.exp.plan = null;
    }
    if (page === "edit") {
      st.edit.back = ""; // opened from the menu: the save stays here
      openEdit(null);
      return;
    }
    if (st.page === "edit") st.edit.back = "";
    if (page === "home") {
      st.nb.devices = null;
      run("案件を読んでいます", async () => {
        st.home = await api().Home();
        st.page = "home";
      });
      return;
    }
    if (st.page !== page) {
      // The lists of the mapping, the comparison, the lists and the logs
      // share st.mig: what one told is not about the next (spec 14.10).
      st.mig.msg = "";
      st.mig.sel = "";
      st.mig.reason = "";
    }
    if (page === "mapping" && st.page !== "mapping") {
      // The part with something left is opened, unless the way here chose
      // one (spec 14.17.2).
      if (st.map.pin) st.map.tab = st.map.pin;
      else st.map.tab = mapTabNow();
      st.map.pin = "";
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
    if (page === "fill" && !st.fil.view) {
      loadFill();
      return;
    }
    if (page === "extra" && !st.xtra.places) {
      loadExtra();
      return;
    }
    if (page === "judge" && !st.jdg.views[st.jdg.source]) {
      loadJudge();
      return;
    }
    if (page === "search" && !st.srch.view) {
      loadSearch();
      return;
    }
    if (page === "outputs") {
      loadOutputs();
      return;
    }
    if (page === "delta" && !st.dlt.view) {
      loadDelta();
      return;
    }
    render();
    if (page === "import") importIfStale();
  }

  function drawNav() {
    const nav = $("nav");
    nav.textContent = "";
    const ns = navSteps();
    // The state of the steps of each screen is drawn from data-badge, so
    // that the name of the button stays its text (spec 14.16.5).
    const item = (page, label, needsCase) => {
      const b = navBadge(page, ns);
      return button(label, () => go(page), { class: cls(st.page === page && "on", b && b.cls), disabled: needsCase && !st.cas,
        "data-badge": b ? b.word : null, "data-page": page });
    };
    // The menu in the order of the steps of the case, the rest folded
    // (spec 14.14.2); a heading is not a button.
    const group = (title, ...items) => h("div", { class: "nav-group" }, h("div", { class: "nav-group-title" }, title), items);
    const migration = !st.cas || st.cas.kind !== "新規";
    const I = {
      home: ["ホーム", true], import: ["取り込み", true], mapping: ["新旧対照表", true], judge: ["読めなかった行", true],
      edit: ["表の編集", true], result: ["結果", true], logs: ["検証機のログ", true],
      extra: ["追加コンフィグ", true], fill: ["ひな形から作る", true], delta: ["差分の投入", true], customer: ["外部一覧", true],
      compare: ["比較", false], search: ["検索", true], outputs: ["出力", true],
    };
    // A migration opens the table from the home, the mapping and the findings:
    // the menu has it no more (spec 14.19.7). A new build writes its book there.
    const steps = migration ? ["home", "import", "mapping", "judge", "result", "logs"] : ["home", "edit", "result", "logs"];
    const rest = (migration ? [] : ["import", "mapping", "judge"]).concat(["extra", "fill", "delta", "customer", "compare", "search", "outputs"]);
    const items = (ids) => ids.map((id) => item(id, I[id][0], I[id][1]));
    const fold = h("details", { class: "nav-group nav-others", id: "nav-others", open: rest.includes(st.page) || navOthersOpen() },
      h("summary", { class: "nav-group-title" }, "そのほか"), items(rest));
    fold.addEventListener("toggle", () => { if (!rest.includes(st.page)) navOthersOpen(fold.open); });
    add(nav, [
      h("div", { class: "brand" }, "ConfigChief",
        // The version under the name, as PalaTerm has it (spec 14.19.2).
        button("v" + st.version, () => go("about"), { class: "side-ver", id: "side-ver", title: "バージョン情報を開く" })),
      h("div", { class: "nav-case" }, st.cas ? st.cas.name : "案件を開いていません"),
      navNow(ns),
      group("工程", items(steps)),
      fold,
      group("管理",
        item("cases", "案件", false),
        item("about", "バージョン情報", false)),
    ]);
  }

  // navSteps gathers the steps of the open case by the screen they are done
  // on, and which one is next (spec 14.16.5).
  function navSteps() {
    const p = st.steps;
    if (!st.cas || !p || !p.steps || !p.steps.length) return null;
    const next = p.next || {};
    const i = p.steps.findIndex((x) => next.page && x.page === next.page && x.act === next.act && (x.state === "todo" || x.state === "optional"));
    const by = {};
    p.steps.forEach((x) => { (by[x.page] = by[x.page] || []).push(x); });
    return { p, i, by };
  }

  // navBadge is the state of the steps of one screen of the menu: next, what
  // is left, or done.
  function navBadge(page, ns) {
    const list = ns && ns.by[page];
    if (!list) return null;
    const nx = ns.i >= 0 ? ns.p.steps[ns.i] : null;
    if (nx && nx.page === page) return { word: "次" + (nx.note ? " " + nx.note : ""), cls: "nb-next" };
    const todo = list.find((x) => x.state === "todo");
    if (todo) return { word: todo.note || "残りあり", cls: "nb-todo" };
    if (list.some((x) => x.state === "counting")) return { word: "…", cls: "nb-wait" };
    const must = list.filter((x) => x.state !== "optional");
    if (must.length && must.every((x) => x.state === "done")) return { word: "済み", cls: "nb-done" };
    return null;
  }

  // navNow tells under the name of the case which step the case is at.
  function navNow(ns) {
    if (!ns) return null;
    const n = ns.p.steps.length;
    if (ns.i >= 0) {
      const x = ns.p.steps[ns.i];
      return button(`いま ${ns.i + 1}/${n}: ${x.label}${x.note ? `（${x.note}）` : ""}`, () => goStep(x.page, x.act),
        { class: "nav-now", id: "nav-now", title: (ns.p.next && ns.p.next.text) || "" });
    }
    const t = ns.p.next && ns.p.next.text;
    return t ? h("div", { class: "nav-now", id: "nav-now", title: t }, t) : null;
  }

  // navOthersOpen tells (or keeps, given a value) whether そのほか of the
  // menu is open: a choice of this PC (spec 14.14.2).
  function navOthersOpen(v) {
    try {
      if (v === undefined) return localStorage.getItem("cc.nav.others") === "1";
      localStorage.setItem("cc.nav.others", v ? "1" : "0");
    } catch (e) {
      // the choice is not kept
    }
    return !!v;
  }

  function render() {
    drawNav();
    const main = $("main");
    const top = main.scrollTop;
    const same = main.dataset.page === st.page;
    main.textContent = "";
    main.dataset.page = st.page;
    if (st.readOnly && st.cas) {
      main.appendChild(h("div", { class: "box warn", id: "read-only" }, st.readOnly));
    }
    if (st.error) {
      main.appendChild(h("div", { class: "box error" }, button("閉じる", () => { st.error = ""; render(); }, { class: "link close" }), st.error));
    }
    // The next step on every screen of the steps (spec 14.16.1).
    if (st.cas && !NO_NEXT_LINE.has(st.page)) {
      main.appendChild(nextLine());
      if (!same) askNext();
    }
    if (st.cas && st.xl) main.appendChild(excelCard());
    const pages = { cases: casesPage, home: homePage, import: importPage, judge: judgePage, extra: extraPage, mapping: mappingPage, edit: editPage, customer: customerPage, fill: fillPage, result: resultPage, logs: logsPage, search: searchPage, compare: comparePage, about: aboutPage, outputs: outputsPage, delta: deltaPage };
    afterDraw = [];
    add(main, [pages[st.page]()]);
    main.scrollTop = same ? top : 0;
    const to = st.jump && $(st.jump);
    st.jump = "";
    if (to) to.scrollIntoView();
    for (const fn of afterDraw) fn();
    afterDraw = [];
    // The control the next step points at (spec 14.10.4).
    const hint = st.hint && $(st.hint);
    if (hint) {
      hint.classList.add("hint");
      hint.scrollIntoView({ block: "nearest" });
    }
  }

  // ---- lists ----

  // tableView draws a list with a filter over all its rows. opts: id (where
  // the filter is kept), onPick(i, tr), selected (the row that is picked),
  // checks (a box in front of each row: can(i), on(i), set(i, on), and
  // changed() after the boxes changed; spec 14.10.8).
  function tableView(t, opts) {
    opts = opts || {};
    const key = opts.id || t.name;
    const fs = st.filters[key] || (st.filters[key] = { text: "", shown: ROWS, sort: { ci: -1, dir: 1 }, widths: {} });
    if (!fs.sort) fs.sort = { ci: -1, dir: 1 };
    if (!fs.widths) fs.widths = {};
    if (!t.rows.length) return h("div", { class: "table-wrap" }, h("div", { class: "empty" }, "ありません"));
    const code = new Set(t.code || []);
    const whole = new Set(t.whole || []); // the columns that are not broken into lines
    // The columns empty in every row are not drawn, and a column of long
    // texts keeps a width it is read in (spec 14.12.4).
    const cellText = (x) => (x == null ? "" : String(x));
    const empty = new Set(), wide = new Set();
    t.header.forEach((_, c) => {
      let longest = 0, any = false;
      for (const row of t.rows) {
        const x = cellText(row[c]);
        if (x !== "") any = true;
        if (x.length > longest) longest = x.length;
        if (any && longest > 24) break;
      }
      if (!any && !fs.showEmpty) empty.add(c);
      if (longest > 24) wide.add(c);
    });
    const cols = t.header.map((_, c) => c).filter((c) => !empty.has(c));
    const body = h("tbody");
    const note = h("span", { class: "dim small" });
    const more = button("さらに表示", () => { fs.shown += ROWS; fill(); });
    const ck = opts.checks;
    let shownRows = []; // the rows drawn, for the box of the heading
    const allBox = ck ? h("input", { type: "checkbox", title: "見えている行をすべて選ぶ・外す" }) : null;
    if (allBox) {
      allBox.addEventListener("click", (e) => e.stopPropagation());
      allBox.addEventListener("change", () => {
        shownRows.filter((i) => ck.can(i)).forEach((i) => ck.set(i, allBox.checked));
        fill();
        ck.changed();
      });
    }

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
      if (fs.sort.ci >= 0) {
        const c = fs.sort.ci;
        hit.sort((x, y) => fs.sort.dir * listSorter.compare(String(t.rows[x][c] == null ? "" : t.rows[x][c]), String(t.rows[y][c] == null ? "" : t.rows[y][c])) || x - y);
      }
      body.textContent = "";
      for (const i of hit.slice(0, fs.shown)) {
        const mark = (t.marks && t.marks[i]) || "";
        const tr = h("tr", { class: cls(mark && "mark-" + mark, opts.onPick && "pick", opts.selected === i && "sel") });
        if (ck) {
          const box = h("input", { type: "checkbox", checked: ck.on(i), disabled: !ck.can(i), class: "pick-one" });
          box.addEventListener("click", (e) => e.stopPropagation());
          box.addEventListener("change", () => { ck.set(i, box.checked); ck.changed(); });
          const td = h("td", { class: "pickbox" }, box);
          td.addEventListener("click", (e) => { e.stopPropagation(); if (!box.disabled) { box.checked = !box.checked; ck.set(i, box.checked); ck.changed(); } });
          tr.appendChild(td);
        }
        for (const c of cols) tr.appendChild(h("td", { class: cls(code.has(c) && "code", whole.has(c) && "whole", wide.has(c) && "wide") || null }, t.rows[i][c]));
        if (opts.onPick) tr.addEventListener("click", () => opts.onPick(i, tr));
        body.appendChild(tr);
      }
      const drawn = Math.min(hit.length, fs.shown);
      shownRows = hit.slice(0, fs.shown);
      if (allBox) {
        const can = shownRows.filter((i) => ck.can(i));
        allBox.checked = can.length > 0 && can.every((i) => ck.on(i));
      }
      note.textContent = hit.length === t.rows.length && drawn === hit.length
        ? `${t.rows.length} 行`
        : `${t.rows.length} 行のうち ${hit.length} 行が当てはまり、${drawn} 行を出しています`;
      more.hidden = hit.length <= fs.shown;
    }

    const filter = h("input", { type: "text", value: fs.text, placeholder: "絞り込み（空白で区切ると、すべてを含む行）", oninput: (e) => { fs.text = e.target.value; fs.shown = ROWS; fill(); } });
    fill();
    // The headings sort (ascending, descending, as given) and their right
    // edges are dragged to widen a column; a double click puts it back.
    const heads = cols.map((c) => {
      const x = t.header[c];
      const mark = fs.sort.ci === c ? (fs.sort.dir > 0 ? " ▲" : " ▼") : "";
      const th = h("th", { class: "sortable", title: "押すと並べ替えます（昇順 → 降順 → 元の順）" }, x, h("span", { class: "sort-mark" }, mark));
      if (fs.widths[c]) th.style.width = th.style.minWidth = th.style.maxWidth = fs.widths[c] + "px";
      th.addEventListener("click", (e) => {
        if (e.target.classList.contains("rz")) return;
        if (fs.sort.ci !== c) fs.sort = { ci: c, dir: 1 };
        else if (fs.sort.dir > 0) fs.sort.dir = -1;
        else fs.sort = { ci: -1, dir: 1 };
        heads.forEach((th2, k) => { th2.querySelector(".sort-mark").textContent = fs.sort.ci === cols[k] ? (fs.sort.dir > 0 ? " ▲" : " ▼") : ""; });
        fill();
      });
      const rz = h("span", { class: "rz", title: "ドラッグで幅を変えます（ダブルクリックで元に戻す）" });
      rz.addEventListener("mousedown", (e) => {
        e.preventDefault();
        e.stopPropagation();
        const x0 = e.clientX;
        const w0 = th.getBoundingClientRect().width / (st.zoom / 100);
        const move = (m) => {
          const w = Math.max(40, Math.round(w0 + (m.clientX - x0) / (st.zoom / 100)));
          fs.widths[c] = w;
          th.style.width = th.style.minWidth = th.style.maxWidth = w + "px";
        };
        const up = () => { document.removeEventListener("mousemove", move); document.removeEventListener("mouseup", up); };
        document.addEventListener("mousemove", move);
        document.addEventListener("mouseup", up);
      });
      rz.addEventListener("dblclick", (e) => {
        e.stopPropagation();
        delete fs.widths[c];
        th.style.width = th.style.minWidth = th.style.maxWidth = "";
      });
      th.appendChild(rz);
      return th;
    });
    const emptyNote = fs.showEmpty ? button("空の列を隠す", () => { fs.showEmpty = false; render(); }, { class: "link small" })
      : empty.size ? button(`空の列 ${empty.size} 個を隠しています（出す）`, () => { fs.showEmpty = true; render(); }, { class: "link small", title: [...empty].map((c) => t.header[c]).join("、") }) : null;
    return h("div", null,
      t.lead ? h("p", { class: "dim" }, t.lead) : null,
      h("div", { class: "row" }, filter, note, more, emptyNote),
      h("div", { class: "table-wrap" }, h("table", { class: "listed" }, h("thead", null, h("tr", null, allBox ? h("th", { class: "pickbox" }, allBox) : null, heads)), body)));
  }

  const listSorter = new Intl.Collator("ja", { numeric: true });

  // ---- 案件 ----

  function casesPage() {
    const c = st.create;
    // The place of the cases (spec 14.15.1): projects next to the exe, or
    // the one chosen instead.
    if (!st.place && !st.placeLoading) {
      st.placeLoading = true;
      api().Place().then((v) => { st.place = v; st.placeLoading = false; if (st.page === "cases") render(); })
        .catch((e) => { st.placeLoading = false; st.error = errText(e); render(); });
    }
    const pl = st.place;
    const sep = pl && pl.dir.includes("/") && !pl.dir.includes("\\") ? "/" : "\\";
    const where = (name) => pl ? pl.dir + sep + (name.trim() || "（案件名）") : "";
    const create = () => run("案件を作っています", async () => {
      if (c.from) {
        const parts = (st.caseParts || []).filter((p) => (c.parts && p.id in c.parts ? c.parts[p.id] : p.default)).map((p) => p.id);
        const made = await api().CreateFrom(c.name, c.kind, c.from, parts);
        await opened(made.home);
        st.caseNote = made.said;
      } else {
        await opened(await api().Create(c.name, c.kind));
      }
      st.create = { name: "", kind: "migration" };
    });
    // The settings of a case before (spec 14.12.8).
    if (!st.caseParts) {
      st.caseParts = [];
      api().CaseParts().then((list) => { st.caseParts = list || []; if (st.page === "cases") render(); }).catch(() => {});
    }
    const pickFrom = async () => {
      try {
        const dir = await api().PickFolder("設定を写す元の案件フォルダ");
        if (dir) c.from = dir;
      } catch (e) {
        st.error = errText(e);
      }
      render();
    };
    // The cases of the place by their names; those outside it with their folders.
    const fromChoices = [["", "写さない（空の案件）"]]
      .concat((pl ? pl.cases : []).map((r) => [r.dir, r.name]))
      .concat(st.recent.filter((r) => !r.missing).map((r) => [r.dir, `${r.name || "（名前なし）"}（${r.dir}）`]));
    if (c.from && !fromChoices.some((x) => x[0] === c.from)) fromChoices.push([c.from, c.from]);
    // The sample case (spec 14.12.7, 14.15.3).
    const sample = () => run("見本の案件を作っています", async () => {
      const made = await api().SampleCase();
      await opened(made.home);
      st.caseNote = made.said;
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
    const pickPlace = () => run("置き場所を変えています", async () => {
      const v = await api().PickPlace();
      if (v) st.place = v;
    });
    const resetPlace = () => run("置き場所を戻しています", async () => { st.place = await api().ResetPlace(); });
    const showPlace = () => run("置き場所を開いています", async () => { await api().ShowPlace(); });
    const typed = (e) => {
      c.name = e.target.value;
      const w = document.getElementById("create-where");
      if (w) w.textContent = where(c.name);
      const b = document.getElementById("create-go");
      if (b) b.disabled = !c.name.trim();
    };

    return [
      h("h1", null, "案件"),
      h("p", { class: "dim" }, "案件 1 つが 1 つのフォルダです。コンフィグ・パラメータブック・レポートを、その中にまとめて置きます。"),
      h("div", { class: "card", id: "place-card" },
        h("h2", null, "案件の一覧"),
        pl ? h("p", { class: "dim small" }, "置き場所: ", h("span", { class: "path", id: "place-dir" }, pl.dir), pl.default ? "（ConfigChief.exe の隣）" : "") : null,
        pl && pl.shared ? h("div", { class: "box warn" }, `置き場所が共有の場所です（${pl.shared}）。案件は手元のフォルダで持つ決まりです。同期や、ほかの人の保存とぶつかることがあります。`) : null,
        pl && pl.cases.length ? h("div", { class: "table-wrap" }, h("table", { id: "place-cases" },
          h("thead", null, h("tr", null, h("th", null, "案件"), h("th", null, "フォルダ"), h("th", null, ""))),
          h("tbody", null, pl.cases.map((r) => h("tr", null,
            h("td", null, r.name),
            h("td", { class: "path" }, r.dir.split(/[\\/]/).pop()),
            h("td", null, button("開く", () => open(r.dir))))))))
          : h("p", { class: "dim" }, pl ? "まだ案件がありません。下の「新しい案件を作る」か「見本の案件を作る」で作ります。" : "読んでいます…"),
        h("div", { class: "row" },
          button("置き場所をエクスプローラで開く", showPlace),
          button("ほかの案件フォルダを選んで開く", openPicked)),
        h("details", { class: "others", id: "place-more" }, h("summary", null, "置き場所を変える"),
          h("p", { class: "dim small" }, "ふだんは ConfigChief.exe の隣の projects に作ります。ほかの場所に集めたいときだけ変えます（この PC に覚えます）。"),
          h("div", { class: "row" },
            button("置き場所を変える", pickPlace, { id: "place-pick" }),
            pl && !pl.default ? button("ConfigChief.exe の隣に戻す", resetPlace, { id: "place-reset" }) : null))),
      st.recent.length ? h("div", { class: "card" },
        h("h2", null, "最近開いた、置き場所の外の案件"),
        h("div", { class: "table-wrap" }, h("table", null,
          h("thead", null, h("tr", null, h("th", null, "案件"), h("th", null, "フォルダ"), h("th", null, ""))),
          h("tbody", null, st.recent.map((r) => h("tr", null,
            h("td", null, r.name || "（名前なし）"),
            h("td", { class: "path" }, r.dir, r.missing ? h("div", { class: "dim" }, "案件フォルダが見つかりません") : null),
            h("td", null, h("div", { class: "row" },
              button("開く", () => open(r.dir), { disabled: r.missing }),
              button("一覧から外す", () => forget(r.dir)))))))))) : null,
      h("div", { class: "card" },
        h("h2", null, "新しい案件を作る"),
        h("div", { class: "row" }, h("span", null, "案件名"), h("input", { type: "text", id: "create-name", value: c.name, placeholder: "例: 拠点ルータ更改", oninput: typed })),
        h("div", { class: "row" }, h("span", null, "種類"),
          seg([["migration", "更改（旧コンフィグから作る）"], ["new", "新規"]], c.kind, (k) => { c.kind = k; render(); })),
        h("div", { class: "row" }, h("span", null, "設定を写す元の案件"),
          Object.assign(select(fromChoices, c.from || "", (x) => { c.from = x; render(); }), { id: "create-from" }),
          button("フォルダを選ぶ", pickFrom),
          tip("前の案件の、納品用シートのひな形・案件のルール・チェックの設定・追加コンフィグ・列の割り当て・ひな形と変数ルール・パックを、新しい案件に写します。コンフィグ・パラメータブック・出力は写しません。元の案件は変えません。")),
        c.from ? h("div", { class: "row create-parts" }, (st.caseParts || []).map((p) => h("label", null,
          h("input", { type: "checkbox", "data-part": p.id, checked: c.parts && p.id in c.parts ? c.parts[p.id] : p.default, onchange: (e) => { c.parts = c.parts || {}; c.parts[p.id] = e.target.checked; } }), " " + p.word))) : null,
        h("div", { class: "row" }, h("span", { class: "dim" }, "作るフォルダ: "), h("span", { class: "path", id: "create-where" }, where(c.name))),
        h("div", { class: "row" }, button("作る", create, { class: "primary", id: "create-go", disabled: !c.name.trim() }))),
      h("div", { class: "card" },
        h("h2", null, "見本の案件"),
        h("p", null, "初めて使うときや、人に使い方を見せるときに。置き場所の sample に、架空の旧コンフィグ（Cisco の拠点ルータ 2 台）を置いた更改の案件を作ります。ホームの「次にやること」に沿って、取り込みから納品物まで一通り試せます。現場のデータは使いません。"),
        h("div", { class: "row" }, button("見本の案件を作る", sample, { id: "sample-case" }))),
      casePart(),
      st.cas ? checksPart() : null,
    ];
  }

  // checksPart sets, for the case, how heavy each check is or stops it: it
  // writes checks.severity and checks.disable of project.yaml (spec 14.11.5).
  function checksPart() {
    const k = st.cks;
    const SEV = { error: "エラー", warning: "警告", info: "情報" };
    const load = () => run("チェックの一覧を読んでいます", async () => {
      const list = await api().CheckSettings();
      const set = {};
      for (const c of list.checks) set[c.id] = c.now;
      st.cks = { list, set, q: "", only: false, msg: "" };
    });
    if (!k) {
      return h("div", { class: "card", id: "checks-card" },
        h("h2", null, "チェックの設定"),
        h("p", { class: "dim" }, "この案件で、チェックごとに重さ（エラー・警告・情報）を変えたり、止めたりします。この案件だけに効きます。"),
        h("div", { class: "row" }, button("チェックの設定を開く", load, { id: "checks-open" })));
    }
    const changed = k.list.checks.filter((c) => (k.set[c.id] || "") !== (c.now || ""));
    const shown = k.list.checks.filter((c) => !k.only || k.set[c.id]);
    const count = h("span", { class: "dim" });
    // The text filter hides rows in place: the field keeps the focus.
    const narrow = () => {
      const q = k.q.trim().toLowerCase();
      let n = 0;
      for (const tr of body.children) {
        const hit = !q || tr.dataset.text.includes(q);
        tr.hidden = !hit;
        if (hit) n++;
      }
      count.textContent = `${n} / ${k.list.checks.length} 個`;
    };
    const save = () => run("project.yaml に書いています", async () => {
      k.msg = await api().SaveCheckSettings(k.set);
      const list = await api().CheckSettings();
      k.list = list;
      for (const c of list.checks) k.set[c.id] = c.now;
    });
    const choices = (c) => [["", `既定のまま（${SEV[c.default] || c.default}）`], ["error", "エラー"], ["warning", "警告"], ["info", "情報"]].concat(c.stays ? [] : [["off", "止める"]]);
    const body = h("tbody", null, shown.map((c) => {
      const tr = h("tr", { class: k.set[c.id] ? "marked" : null },
        h("td", null, c.id), h("td", null, c.title), h("td", null, SEV[c.default] || c.default),
        h("td", null, (() => { const el = select(choices(c), k.set[c.id] || "", (x) => { k.set[c.id] = x; render(); }); el.dataset.check = c.id; return el; })()));
      tr.dataset.text = (c.id + " " + c.title).toLowerCase();
      return tr;
    }));
    narrow();
    return h("div", { class: "card", id: "checks-card" },
      h("h2", null, "チェックの設定"),
      h("p", { class: "dim" }, "重さを変えた・止めたチェックは、次のチェックから効きます。値の誤りがある機器を生成しないためのチェック（V-101〜V-107・V-109）は止められません。 ",
        tip("書くのは project.yaml の checks.severity と checks.disable だけです（ほかの行とコメントは変えません）。書く前の project.yaml は state/backup に控えます。この版に無い ID の設定は、消さずに残します。")),
      boxes("info", k.msg ? [k.msg] : []),
      k.list.unknown.length ? boxes("warn", [`この版に無いチェックの設定（そのまま残します）: ${k.list.unknown.join("、")}`]) : null,
      h("div", { class: "row" },
        h("input", { type: "text", placeholder: "ID・名前で絞り込む", value: k.q, id: "checks-q", oninput: (e) => { k.q = e.target.value; narrow(); } }),
        h("label", null, h("input", { type: "checkbox", checked: k.only, onchange: (e) => { k.only = e.target.checked; render(); } }), " 既定と違うものだけ"),
        count),
      h("div", { class: "table-wrap checks-set" }, h("table", null,
        h("thead", null, h("tr", null, h("th", null, "ID"), h("th", null, "名前"), h("th", null, "既定"), h("th", null, "この案件"))),
        body)),
      h("div", { class: "row" },
        button(changed.length ? `保存する（${changed.length} 個）` : "保存する", save, { class: "primary", disabled: !changed.length, id: "checks-save" }),
        button("元に戻す", () => { for (const c of k.list.checks) k.set[c.id] = c.now; render(); }, { disabled: !changed.length }),
        button("閉じる", () => { st.cks = null; render(); })));
  }

  // casePart writes the open case out to one file and reads such a file
  // back (spec 14.9; PalaTerm's 一式書き出し・読み込み).
  function casePart() {
    const x = st.cx;
    const write = () => {
      if (x.seal) {
        if (x.pw.length < 8) { st.error = "パスワードは 8 文字以上にしてください"; render(); return; }
        if (x.pw !== x.pw2) { st.error = "パスワードが 2 回で違います"; render(); return; }
      }
      run("案件を書き出しています", async () => {
        if (!x.seal && !(await ask("暗号化しない書き出し", "パスワードで暗号化せずに書き出します。受け取った人でなくても中身を読めます。よろしいですか？", "暗号化せずに書き出す"))) return;
        const v = await api().CaseExport(x.noSecrets, x.seal ? x.pw : "");
        if (!v || !v.path) return; // the dialog was closed
        x.view = v;
        x.pw = x.pw2 = "";
        toast("書き出しました: " + v.path);
      });
    };
    const read = async () => {
      let f = null;
      try {
        f = await api().CasePick();
      } catch (e) {
        st.error = errText(e);
        render();
        return;
      }
      if (!f) return;
      let pw = "";
      if (f.sealed) {
        pw = await askText("パスワード", "このファイルはパスワードで暗号化されています。書き出した人から聞いたパスワードを入れてください。", "", "読み込む", true);
        if (pw == null) return;
      }
      run("案件を読み込んでいます", async () => {
        const got = await api().CaseImport(f.path, pw);
        if (!got) return; // the dialog was closed
        await opened(got.home);
        st.caseNote = `案件「${got.name}」を ${got.dir} に広げて開きました（${got.files} ファイル。${got.written.slice(0, 16).replace("T", " ")} に ConfigChief ${got.version} で書き出したもの）。`
          + (got.noSecrets ? "秘密の値を外した案件です。パラメータブックの **** を本当の値に書き換えてください（チェックで V-218 が出ます）。" : "");
      });
    };
    const v = x.view;
    return h("div", { class: "card" },
      h("h2", null, "案件の書き出し・読み込み"),
      h("p", { class: "dim" }, "案件は手元のフォルダで持ちます。人に渡すときは、フォルダごとコピーするか、ここで 1 つのファイル（.ccase）に書き出します。受け取った人は「読み込む」で、手元のフォルダに案件として広げます。 ",
        tip("入れないもの: 作ったコンフィグとレポート（out）、ブックの控え（state/backup）、使用中の印。受け取った人が作り直します。パスワードは覚えません。忘れると読めません。")),
      h("h3", null, "書き出す"),
      st.cas ? [
        h("div", { class: "row" }, h("span", null, "案件"), h("b", null, st.cas.name)),
        h("div", { class: "row" },
          h("label", null, h("input", { type: "checkbox", checked: x.seal, onchange: (e) => { x.seal = e.target.checked; render(); } }), " パスワードで暗号化する（おすすめ）"),
          h("label", null, h("input", { type: "checkbox", checked: x.noSecrets, onchange: (e) => { x.noSecrets = e.target.checked; } }), " 秘密の値（パスワード・鍵）を外す"),
          tip("外すと、パラメータブックの秘密の列と、取り込めたコンフィグ・ログの中の秘密の値を **** にします。取り込めなかった行（読めなかった行）と、Excel の一覧の中は伏せられません。書き出した後に一覧が出るので、渡す前に中身を見てください。")),
        x.seal ? h("div", { class: "row" },
          h("input", { type: "password", id: "cx-pw", value: x.pw, placeholder: "パスワード（8 文字以上）", autocomplete: "new-password", oninput: (e) => { x.pw = e.target.value; } }),
          h("input", { type: "password", id: "cx-pw2", value: x.pw2, placeholder: "もう一度", autocomplete: "new-password", oninput: (e) => { x.pw2 = e.target.value; } })) : null,
        h("div", { class: "row" }, button("書き出す", write, { class: "primary", id: "cx-write" })),
      ] : h("p", { class: "dim" }, "案件を開くと書き出せます。"),
      v ? h("div", { class: "box info", id: "cx-done" },
        `${v.path} に書き出しました（${v.files} ファイル・${(v.size / 1048576).toFixed(1)} MB・${v.encrypted ? "パスワードで暗号化" : "暗号化なし"}）。`
        + (v.noSecrets ? `\n秘密の値を外しました: 値 ${v.values} 個、テキストの ${v.masked} か所（${v.maskedFiles.length} ファイル）。` : "")
        + (v.notes.length ? "\n" + v.notes.join("\n") : "")) : null,
      v && v.left.length ? h("div", { class: "box warn" }, "伏せられなかったもの（渡す前に中身を見てください）:\n" + v.left.join("\n")) : null,
      h("h3", null, "読み込む"),
      h("div", { class: "row" }, button("ファイルを選んで読み込む", read, { id: "cx-read" }), h("span", { class: "dim small" }, "読み込んだ案件は、案件の置き場所に新しいフォルダを作って広げ、そのまま開きます。")));
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
      h("p", null, n ? `${n} ファイルあります。` : "まだファイルがありません。コンフィグ（テキスト、端末ログでも可）を下の枠に落とすか、「ファイルを足す」で選んでください。"),
      dropZone(source),
      h("div", { class: "row" },
        button("フォルダを開く", () => show("input/" + source)),
        button("取り込みへ", () => toImport(source), { class: n ? "primary" : null, disabled: !n, id: "to-import-" + source })));

    // The cards in the order of the work of the case; what a case may not
    // need is folded under そのほか (spec 14.12.2).
    const migration = m.case.kind !== "新規";
    const bookCard = h("div", { class: "card" },
      h("h2", null, `パラメータブック（${m.case.book}）`),
      h("p", null, m.hasBook ? `あります（最後に書かれた時刻: ${m.bookTime}）。値は、画面の表（「表で編集」）でも、Excel でも直せます。` : migration ? "まだありません。旧コンフィグを取り込むと作られます。" : "まだありません。「空のブックを作る」で作ります。"),
      h("div", { class: "row" },
        button("表で編集", () => openEdit(null), { disabled: !m.hasBook, id: "edit-book" }),
        button("Excel で開く", () => show(m.case.book), { disabled: !m.hasBook }),
        migration ? button("新旧対照表を見る", () => go("mapping")) : null,
        m.hasBook ? null : button("空のブックを作る（新規構築）", newBook, { id: "new-book", class: migration ? null : "primary" })),
      m.hasBook ? h("div", { class: "row", id: "clone-row" },
        h("span", null, "機器の複製: "),
        h("input", { type: "text", id: "clone-from", value: nb.from, list: "clone-devices", placeholder: "元のホスト名（一覧から選べます）", oninput: (e) => { nb.from = e.target.value; } }),
        h("datalist", { id: "clone-devices" }, (nb.devices || []).map((d) => h("option", { value: d }))),
        h("span", null, " → "),
        h("input", { type: "text", id: "clone-to", value: nb.to, placeholder: "新しいホスト名（カンマで区切って複数）", oninput: (e) => { nb.to = e.target.value; } }),
        button("複製する", cloneDevice, { id: "clone-go" })) : null,
      nb.said.length ? h("div", { id: "newbuild-result" }, nb.said.map((x) => h("p", null, x))) : null);
    if (m.hasBook && !nb.devices && !nb.loading) {
      nb.loading = true;
      api().BookDevices().then((list) => { nb.devices = list || []; nb.loading = false; if (st.page === "home" && !busy) render(); }).catch(() => { nb.devices = []; nb.loading = false; });
    }
    const genCard = h("div", { class: "card" },
      h("h2", null, "生成とチェック"),
      h("p", null, "パラメータブックを読んでチェックし、全台のコンフィグ・チェック結果表・変更一覧（2 回目から）を出力フォルダに書きます。値の誤りがある機器のコンフィグは書きません。"),
      h("p", { class: "dim" }, !m.hasBook ? "パラメータブックがまだありません。" : m.lastGen ? `最後の生成: ${m.lastGen}` : "まだ生成していません。"),
      h("div", { class: "row" },
        button("生成とチェック", () => generate(false), { class: "primary", disabled: !m.hasBook, id: "generate" }),
        button("チェックだけ", () => generate(true), { disabled: !m.hasBook }),
        m.lastGen || st.res.view ? button("結果を見る", () => go("result")) : null),
      h("p", { class: "dim small" }, "変えるコマンドだけの投入（設定変更の案件）は、左のメニューの「そのほか」の「差分の投入」で作ります。"));
    const deliverCard = h("div", { class: "card" },
      h("h2", null, "納品物"),
      h("p", null, "「納品一式を作る」で、いちばん新しい生成のコンフィグ・チェック結果表と、パラメータブックから作る納品用パラメータシート・相手に渡すパラメータ表を、1 つのフォルダ（と zip）にまとめます。シートや表だけを出すこともできます（チェックはしません）。"),
      h("div", { class: "row" },
        h("span", null, "秘密の値（シート・表）: "),
        select([["show", "そのまま書く"], ["mask", "*** に伏せる"]], exp.mask, (v) => { exp.mask = v; })),
      h("div", { class: "row" },
        button("納品一式を作る", openDelivery, { class: "primary", disabled: !m.hasBook, id: "make-delivery" })),
      deliveryPart(),
      // The rest is folded under one line (spec 14.18.8).
      h("details", { class: "fold-menu", id: "deliver-more", open: !!st.folds["deliver-more"], ontoggle: (e) => { st.folds["deliver-more"] = e.target.open; } }, h("summary", null, "シート・表だけを出す・ファイル名・ひな形"),
        h("div", { class: "row" },
          button("納品用パラメータシートを出す", () => doExport("sheet"), { disabled: !m.hasBook }),
          button("相手に渡すパラメータ表を出す", () => doExport("peer"), { disabled: !m.hasBook }),
          button("ひな形のフォルダを開く", () => show("templates"))),
      h("div", { class: "row" },
        h("span", null, "納品一式のファイル名: "),
        Object.assign(select([["ja", "日本語（納品_<案件名>・チェック結果表.xlsx …）"], ["en", "英語（delivery_<案件名>・check_results.xlsx …）"]], m.deliveryNames || "ja",
          (v) => run("ファイル名の言葉を変えています", async () => { st.home = await api().SetDeliveryNames(v); })), { id: "delivery-names" }),
        tip("お客さまに渡すフォルダとファイルの名前です。この案件だけに効きます（project.yaml の export.delivery_names）。ファイルの中の文は変わりません。")),
      h("div", { class: "row" },
        h("span", null, "ひな形の見本: "),
        select([["sheets", "1 ブックで機器ごとのシート"], ["files", "機器ごとに 1 ファイル"], ["list", "全台を 1 枚の一覧"]], exp.sample || "sheets", (v) => { exp.sample = v; }),
        button("ひな形の見本を作る", () => run("ひな形の見本を作っています", async () => {
          const rel = await api().MakeSample(exp.sample || "sheets");
          exp.sampleMsg = `${rel} を作りました。templates フォルダのひな形は、すべて使います（ひな形 1 つにつき、納品用シートが 1 つ出ます）。見本を直して使うか、要らなければ消してください。書き方は使い方の「納品用パラメータシートのひな形」にあります。`;
          exp.sampleFile = rel;
        }), { id: "make-sample" }),
        tip("見出し・罫線・色の付いたひな形を templates フォルダに作ります。機器の基本・インタフェース・スタティック経路の欄があり、そのまま「納品用パラメータシートを出す」で出せます。自社の書式に合わせて直して使います。改版履歴の欄は {{#rows revisions}} で書けます（使い方の「改版履歴」）"))),
      exp.sampleMsg ? h("div", { class: "box info", id: "sample-msg" }, exp.sampleMsg,
        h("div", { class: "row" }, button("見本を開く", () => show(exp.sampleFile)), button("ひな形のフォルダを開く", () => show("templates")))) : null,
      exp.view ? h("div", { id: "export-result" },
        h("p", null, exp.view.summary),
        boxes("warn", exp.view.notes),
        boxes("warn", exp.view.problems),
        h("div", { class: "row" },
          exp.view.files.map((f) => button(f.slice(f.lastIndexOf("/") + 1), () => show(f))),
          exp.view.dir ? button("フォルダを開く", () => show(exp.view.dir)) : null)) : null);
    const compareCard = h("div", { class: "card" },
      h("h2", null, "比較"),
      h("p", null, "旧コンフィグ・新コンフィグ・パラメータブックのうち 2 つを、設計データとして比べます。"),
      h("div", { class: "row" }, button("比較へ", () => go("compare"))));
    const reportCard = h("div", { class: "card" },
      h("h2", null, "レポート"),
      h("p", null, m.lastOut ? `いちばん新しい出力: ${m.lastOut}` : "まだ出力はありません。"),
      h("div", { class: "row" },
        button("出力フォルダを開く", () => show(m.lastOut), { disabled: !m.lastOut }),
        button("出力の一覧", () => go("outputs"))));
    const packCard = h("div", { class: "card" },
      h("h2", null, "使えるベンダーパック"),
      h("p", null, m.packs.length ? m.packs.map((p) => p.title ? `${p.title}（${p.id} ${p.version}）` : `${p.id} ${p.version}`).join("、") : "ありません。"));
    const main = migration ? [input("old", m.old), bookCard, genCard, deliverCard] : [bookCard, genCard, deliverCard];
    const others = migration ? [input("new", m.new), compareCard, reportCard, packCard] : [input("old", m.old), input("new", m.new), compareCard, reportCard, packCard];
    const fold = h("details", { class: "others", id: "home-others", open: othersOpen() }, h("summary", null, "そのほか（" + (migration ? "新コンフィグ・比較・レポート・パック" : "旧コンフィグ・新コンフィグ・比較・レポート・パック") + "）"), others);
    fold.addEventListener("toggle", () => othersOpen(fold.open));

    const rn = st.rn;
    const rename = async () => {
      let home = null;
      await run("案件名を変えています", async () => { home = await api().RenameCase(rn.name); });
      if (!home) return;
      st.rn = null;
      await opened(home);
      st.caseNote = `案件名を「${home.case.name}」に変えました（フォルダ: ${home.case.dir}）`;
      render();
    };
    return [
      rn
        ? h("div", { class: "row rename", id: "rename" },
          h("input", { type: "text", id: "rename-name", value: rn.name, oninput: (e) => { rn.name = e.target.value; },
            onkeydown: (e) => { if (e.key === "Enter") rename(); if (e.key === "Escape") { st.rn = null; render(); } } }),
          button("変更", rename, { class: "primary", id: "rename-go" }),
          button("キャンセル", () => { st.rn = null; render(); }),
          h("span", { class: "dim small" }, "案件フォルダの名前も一緒に変わります。"))
        : h("div", { class: "row case-title" }, h("h1", null, m.case.name),
          st.readOnly ? null : button("名前を変更", () => { st.rn = { name: m.case.name }; render(); }, { id: "rename-open" })),
      h("p", null, `種類: ${m.case.kind}`),
      stepsBox(),
      nextBox(),
      h("div", { class: "row" }, h("span", { class: "path" }, m.case.dir), button("案件フォルダを開く", () => show(""))),
      m.shared ? h("div", { class: "box warn", id: "shared-note" }, m.shared) : null,
      st.caseNote ? h("div", { class: "box info" }, button("閉じる", () => { st.caseNote = ""; render(); }, { class: "link close" }), st.caseNote) : null,
      boxes("warn", m.problems),
      draftNote(),
      main,
      fold,
      h("div", { class: "row" }, button("案件を閉じる", close)),
    ];
  }

  // othersOpen tells (or keeps, given a value) whether そのほか of the home
  // is open: a choice of this PC (spec 14.12.2).
  function othersOpen(v) {
    try {
      if (v === undefined) return localStorage.getItem("cc.home.others") === "1";
      localStorage.setItem("cc.home.others", v ? "1" : "0");
    } catch (e) {
      // the choice is not kept
    }
    return !!v;
  }

  // ---- 納品一式 (spec 13.6, 13.7) ----

  function openDelivery() {
    const exp = st.exp;
    run("納品一式の準備をしています", async () => {
      const p = await api().DeliveryPlan();
      exp.plan = p;
      exp.done = null;
      exp.rev = { on: p.revisions.length > 0, version: p.nextVersion, summary: p.summary, by: p.by || "" };
      st.jump = "delivery";
    });
  }

  function deliveryPart() {
    const exp = st.exp;
    const p = exp.plan;
    if (exp.done) {
      const d = exp.done;
      return h("div", { class: "box info", id: "delivery-done" }, button("閉じる", () => { exp.done = null; render(); }, { class: "link close" }), d.summary,
        d.notes.length ? "\n" + d.notes.join("\n") : "",
        h("div", { class: "row" }, button("フォルダを開く", () => show(d.dir), { class: "primary" }), button("zip のある場所を開く", () => show(d.dir.slice(0, d.dir.lastIndexOf("/"))))));
    }
    if (!p) return null;
    if (!p.ready) {
      return h("div", { class: "box warn", id: "delivery" }, button("閉じる", () => { exp.plan = null; render(); }, { class: "link close" }), "納品一式は作れません: " + p.why);
    }
    const r = exp.rev;
    const left = p.known ? p.errors + p.warnings : 0;
    const make = async () => {
      if (left > 0 && !(await ask("未承認の指摘が残っています", `未承認の指摘が ${left} 件あります（エラー ${p.errors}・警告 ${p.warnings}）。それでも納品一式を作りますか？\n作ると、UNVERIFIED.txt を入れます。`, "それでも作る"))) return;
      if (r.on && !r.version.trim()) { st.error = "改版履歴に残すときは、版を書いてください"; render(); return; }
      run("納品一式を作っています", async () => {
        exp.done = await api().MakeDelivery(exp.mask === "mask", left > 0, r.on ? r.version.trim() : "", r.on ? r.summary : "", r.on ? r.by : "");
        exp.plan = null;
        st.home = await api().Home();
        st.jump = "delivery-done";
      });
    };
    return h("div", { class: "card", id: "delivery" },
      h("h3", null, "納品一式を作る"),
      h("p", null, `もとにする生成: ${p.when}（コンフィグ ${p.configs} 台）。` + (p.known ? (left ? `未承認の指摘が ${left} 件あります（エラー ${p.errors}・警告 ${p.warnings}）。` : "未承認の指摘はありません。") : "")),
      h("div", { class: "row" },
        h("label", null, h("input", { type: "checkbox", id: "rev-on", checked: r.on, onchange: (e) => { r.on = e.target.checked; render(); } }), " 改版履歴に残す"),
        tip("改版履歴（版・日付・変更内容・作った人）を state/revisions.yaml に残し、納品用シートのひな形に {{#rows revisions}} や {{ revision.version }} を書いたときに出します。使わない案件では外したままにします。")),
      r.on ? [
        h("div", { class: "row" }, h("span", null, "版"), h("input", { type: "text", id: "rev-version", value: r.version, oninput: (e) => { r.version = e.target.value; } }),
          h("span", null, "作った人"), h("input", { type: "text", id: "rev-by", value: r.by, oninput: (e) => { r.by = e.target.value; } })),
        h("div", { class: "row" }, h("span", null, "変更内容"), h("input", { type: "text", id: "rev-summary", class: "wide-input", value: r.summary, oninput: (e) => { r.summary = e.target.value; } })),
        p.revisions.length ? h("p", { class: "dim small" }, "これまでの版: " + p.revisions.map((x) => `${x.version}（${x.date}）`).join("、")) : null,
      ] : null,
      h("div", { class: "row" },
        button("作る", make, { class: "primary", id: "delivery-go" }),
        button("やめる", () => { exp.plan = null; render(); })));
  }

  // ---- 取り込み ----

  // importRun imports a side and shows what it gave on the screen of the
  // import.
  function importRun(s) {
    const im = st.imp;
    return run(`${SRC[s]}を取り込んでいます`, async () => {
      im.views[s] = await api().Import(s);
      delete im.lines[s];
      im.tab[s] = "files";
      // The files may have changed: the unhandled lines are read anew.
      delete st.jdg.views[s];
      delete st.jdg.changed[s];
    }, true);
  }

  // importIfStale imports the side of the screen of the import when its
  // files changed since the last import (spec 14.19.3): nobody has to press
  // the button for files that were just put.
  async function importIfStale() {
    if (busy || !st.cas || st.page !== "import") return;
    const s = st.imp.source;
    let stale = false;
    try {
      stale = await api().ImportStale(s);
    } catch (e) {
      return; // the button is still there
    }
    if (stale && st.page === "import" && st.imp.source === s) await importRun(s);
  }

  function importPage() {
    const im = st.imp;
    const s = im.source;
    const v = im.views[s];
    const doImport = () => importRun(s);
    const toJudge = () => { st.jdg.source = s; go("judge"); };
    const out = [
      h("h1", null, "取り込み"),
      h("div", { class: "row" },
        seg([["old", "旧コンフィグ（input/old）"], ["new", "新コンフィグ（input/new）"]], s, (x) => { im.source = x; render(); importIfStale(); }),
        button(v ? "取り込み直す" : "取り込む", doImport, { class: "primary", id: "import-run" }),
        button("フォルダを開く", () => show("input/" + s))),
      dropZone(s),
      h("p", { class: "dim" }, s === "old"
        ? "input/old のファイルを読んで、取り込みレポートを書きます。パラメータブックの新旧対照表（機器）・新旧対照表（ポート）に、足りない行を足します。"
        : "input/new のファイルを読んで、取り込みレポートを書きます。"),
      h("p", { class: "dim small" }, "画面では、秘密の列の値をいつも伏せて出します。"),
    ];
    if (!v) return out;

    out.push(
      h("h2", null, v.title),
      boxes("info", v.messages),
      st.jdg.changed[s] ? h("div", { class: "box warn", id: "judged-since" }, "この取り込みの後に、読めなかった行の判定を変えました。下の数と取り込みレポートは、取り込み直すと新しくなります。") : null,
      counts(v.counts),
      boxes("warn", v.notes),
      h("div", { class: "row" },
        // In a migration, the mapping comes next (spec 14.18.6); the lines
        // not read are judged after it.
        s === "old" && (!st.cas || st.cas.kind !== "新規") ? button("次へ: 新旧対照表", () => goStep("mapping", ""), { class: "primary", id: "import-next" }) : null,
        button("レポートを Excel で開く", () => show(v.report.xlsx)),
        button("レポートをブラウザで開く", () => show(v.report.html)),
        v.files.some((f) => f.unknown) ? button("読めなかった行に判定を付ける", toJudge) : null,
        s === "new" ? button("新コンフィグからパラメータブックを作る", () => makeBook("new", false)) : null),
      st.map.book && st.map.book.from === "new" && s === "new" ? bookResult(st.map.book) : null,
      needPacks(s, v), needFiles(s, v));

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
          select(v.packs.map((p) => [p.id, p.title ? `${p.title}（${p.id} ${p.version}）` : `${p.id} ${p.version}`]), im.pick[key], (x) => { im.pick[key] = x; }),
          button("このパックで読む", choose));
      }));
  }

  // needFiles lists the host names that two or more files have: a person
  // chooses the file to take (and can choose again later).
  function needFiles(s, v) {
    const groups = new Map();
    for (const f of v.files) {
      if (!f.same || f.same.length < 2) continue;
      if (!groups.has(f.device)) groups.set(f.device, { device: f.device, files: f.same, open: false, used: "" });
      const g = groups.get(f.device);
      if (f.needsFile) g.open = true;
      if (f.imported) g.used = f.path;
    }
    if (!groups.size) return null;
    const im = st.imp;
    const list = [...groups.values()];
    const open = list.some((g) => g.open);
    return h("div", { class: open ? "box warn" : "box" },
      h("div", null, open
        ? "同じホスト名のファイルが 2 つ以上あります。どれを取り込むか選んでください（選ぶまで、そのホスト名はどれも取り込みません。選んだ結果は案件に残ります）。"
        : "同じホスト名のファイルがあります。取り込むファイルは選んであります（選び直せます）。"),
      list.map((g) => {
        const key = s + "/use/" + g.device;
        if (!im.pick[key] || !g.files.includes(im.pick[key])) im.pick[key] = g.used || g.files[0];
        const choose = () => run(`${g.device} のファイルを選んでいます`, async () => {
          im.views[s] = await api().ChooseFile(s, im.pick[key]);
          delete im.lines[s];
        }, true);
        return h("div", { class: "row" },
          h("span", { class: "path" }, g.device),
          select(g.files.map((p) => [p, p + (p === g.used ? "（今これを取り込んでいます）" : "")]), im.pick[key], (x) => { im.pick[key] = x; }),
          button("このファイルを取り込む", choose));
      }));
  }

  function filesTable(s, v) {
    const im = st.imp;
    const files = im.onlyBad ? v.files.filter((f) => f.mark) : v.files;
    const header = ["ファイル", "機器", "パック", "行", "取り込み済み", "無視", "コメント・空行", "読めなかった", "未判定・保留", "取り込み率"];
    if (s === "old") header.push("新ホスト名");
    header.push("指摘");
    const t = { name: "ファイル", lead: "", header, code: [], marks: files.map((f) => f.mark), rows: files.map((f) => {
      const pack = f.pack ? `${f.pack} ${f.version}${f.chosen ? "（人が選んだ）" : ""}` : "";
      const told = f.err ? [f.err] : f.left ? [f.left] : f.warnings.concat(f.notes);
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
        st.error = f.left ? `${f.path}: ${f.left}` : `${f.path} は取り込めていません: ${f.err}`;
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
      if (!hit.length) list.appendChild(h("div", { class: "empty" }, lv.only ? "読めなかった行はありません" : "行がありません"));
      note.textContent = hit.length > lv.shown ? `${hit.length} 行のうち ${lv.shown} 行を出しています` : `${hit.length} 行`;
      more.hidden = hit.length <= lv.shown;
    }

    fill();
    detail(v.lines.find((l) => l.no === lv.sel));
    return h("div", null,
      h("div", { class: "row" },
        h("span", null, `${v.path}（機器 ${v.device || "不明"}）　読めなかった ${unknown} 行（うち未判定・保留 ${open} 行）`),
        h("label", null, h("input", { type: "checkbox", checked: lv.only, onchange: (e) => { lv.only = e.target.checked; lv.shown = LINES; fill(); } }), " 読めなかった行だけ"),
        note, more),
      h("div", { class: "lines" }, list, side));
  }

  // ---- 読めなかった行 ----

  const VERDICTS = [["manual", "追加コンフィグで書く"], ["not_needed", "要らない"], ["hold", "保留"]];

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
      h("h1", null, "読めなかった行"),
      h("div", { class: "row" },
        seg([["old", "旧コンフィグ（input/old）"], ["new", "新コンフィグ（input/new）"]], s, (x) => {
          jd.source = x;
          if (jd.views[x]) render();
          else loadJudge();
        }),
        button(v ? "読み直す" : "読み込む", loadJudge, { class: v ? null : "primary" })),
      h("p", { class: "dim" }, "どの規則にも当たらなかった行を、形（数字・アドレスなどを印に置き換えた文面）ごとのかたまりにまとめて出します。かたまりごとに「追加コンフィグで書く」「要らない」「保留」のどれかを付けます。判定は案件に残り、同じ形の行には、次の取り込みでも当たります。"),
      h("p", { class: "dim small" }, "読めなかった行は、どの値が秘密かをツールが見分けられないため、文面をそのまま出します。"),
    ];
    if (!v) return out;

    // Narrowed by words and by memo lines (spec 14.19.8): what is left can
    // be judged at once.
    const q = (jd.q || "").trim().toLowerCase();
    const list = v.groups.filter((g) => (!jd.only || g.open || g.id === jd.sel[s])
      && (!q || (g.shape + "\n" + g.path + "\n" + g.example.text).toLowerCase().includes(q))
      && (!jd.memo || memoLike(g)));
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
        h("label", null, h("input", { type: "checkbox", id: "judge-memo", checked: jd.memo, onchange: (e) => { jd.memo = e.target.checked; render(); } }), " メモらしい行だけ"),
        tip("日本語などの文字（※・■・【 などの印も）を含む行と、# で始まる行・==== や ---- のような区切りの行です。Excel のコンフィグに混ざったメモの行は、ここで絞ってまとめて「要らない」にできます。"),
        h("input", { type: "search", id: "judge-q", placeholder: "文字で絞る（形・親の行・例）", value: jd.q || "", onchange: (e) => { jd.q = e.target.value; render(); } }),
        h("span", { class: "dim small" }, v.groups.length
          ? `かたまり ${v.groups.length} 個のうち、未判定・保留があるのは ${v.groups.filter((g) => g.open).length} 個です。表示は ${list.length} 個です。`
          : "読めなかった行はありません。")),
      list.length > 1 ? judgeManyBox(s, list) : null,
      h("div", { class: "lines", id: "judge" },
        h("div", { class: "findings-list" }, tableView(t, { id: `judge/${s}/${jd.only}`, onPick: pick, selected: sel })),
        side),
      // For the people who mend the packs, under the work (spec 14.18.8).
      h("details", { class: "fold-menu", id: "carry-fold", open: !!(st.folds["carry-fold"] || jd.carry[s]), ontoggle: (e) => { st.folds["carry-fold"] = e.target.open; } },
        h("summary", null, "メーカーパックを直す人に渡すレポート（持ち出し用）"), carryPart(jd, s)));
    return out;
  }

  // memoLike tells a group whose lines look like notes rather than settings:
  // letters other than ASCII, or a mark that opens a note (spec 14.19.8).
  function memoLike(g) {
    const t = (g.example && g.example.text ? g.example.text : g.shape).trim();
    return /[^\x00-\x7f]/.test(g.shape + "\n" + t) || /^#/.test(t) || /^[=*\-_~]{3,}/.test(t);
  }

  // judgeManyBox judges every group the list shows (spec 14.19.8).
  function judgeManyBox(s, list) {
    const jd = st.jdg, m = jd.many;
    const lines = list.reduce((n, g) => n + g.heads + g.below, 0);
    const go = async () => {
      const word = VERDICTS.find((x) => x[0] === m.verdict)[1];
      if (!(await ask("まとめて判定を付けますか", `表示しているかたまり ${list.length} 個（${lines} 行）の全台に「${word}」を付けます。機器を指定した判定は、そのまま残ります。`, "付ける"))) return;
      await run("判定をまとめて付けています", async () => {
        showJudge(s, await api().JudgeMany(s, list.map((g) => g.id), m.verdict, m.reason, jd.by || ""));
        if (st.imp.views[s]) jd.changed[s] = true;
        delete st.imp.lines[s];
        if (st.imp.tab[s] === "lines") st.imp.tab[s] = "files";
      }, true);
    };
    const btn = button(`表示している ${list.length} 個にまとめて付ける`, go, { class: "primary", id: "judge-many" });
    const check = () => { btn.disabled = !(jd.by || "").trim() || (m.verdict === "not_needed" && !m.reason.trim()); };
    check();
    return h("details", { class: "fold-menu", id: "judge-many-fold", open: !!st.folds["judge-many-fold"], ontoggle: (e) => { st.folds["judge-many-fold"] = e.target.open; } },
      h("summary", null, `表示しているかたまり ${list.length} 個（${lines} 行）に、まとめて判定を付ける`),
      h("div", { class: "approve judge-form judge-many" },
        h("p", { class: "dim small" }, "上の「未判定と保留だけ」「メモらしい行だけ」と文字で絞ってから付けます。全台に付けます。"),
        h("div", { class: "field" }, h("span", null, "判定"), seg(VERDICTS, m.verdict, (x) => { m.verdict = x; render(); })),
        h("div", { class: "field" }, h("span", null, "理由"),
          h("input", { type: "text", id: "judge-many-reason", value: m.reason, placeholder: m.verdict === "not_needed" ? "要らないには、理由が要ります（例: Excel のメモの行）" : "書かなくても付けられます", oninput: (e) => { m.reason = e.target.value; check(); } })),
        h("div", { class: "field" }, h("span", null, "判定した人"),
          h("input", { type: "text", id: "judge-many-by", value: jd.by || "", oninput: (e) => { jd.by = e.target.value; check(); } })),
        h("div", { class: "row" }, btn)));
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
        h("div", { class: "code" }, d.heads.map(lineText).join("\n") + (d.more ? `\n…（ほか ${d.more} 行）` : "")),
        d.device ? h("div", { class: "row" }, button("この行で追加コンフィグの下書きを作る", () => draftExtra(s, g.id, d.device), { class: "small", "data-extra": d.device,
          title: "この機器のこのかたまりの行（下の行も全部）を、この機器の追加コンフィグの末尾に足した下書きを開きます。保存すると、この機器のこの行に「追加コンフィグで書く」を付けます" })) : null)));

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
      h("p", { class: "dim small" }, "追加コンフィグで書く: 追加コンフィグ（extra）に人が書きます。要らない: 新しい機器には要りません（理由を書きます）。保留: まだ決めません（未判定と同じく、残りに数えます）。"),
      h("div", { class: "field" }, h("span", null, "範囲"),
        select([["all", "このかたまりの全台"], ["some", "選んだ機器だけ"]], jd.scope, (x) => { jd.scope = x; render(); })),
      some ? h("div", { class: "judge-pick" }, g.devices.map((d) => h("label", null,
        h("input", { type: "checkbox", checked: !!jd.picked[d.device], disabled: !d.device, onchange: (e) => { jd.picked[d.device] = e.target.checked; render(); } }),
        " " + name(d)))) : null,
      h("div", { class: "field" }, h("span", null, "判定"), seg(VERDICTS, jd.verdict, (x) => { jd.verdict = x; render(); })),
      h("div", { class: "field" }, h("span", null, "理由"),
        h("input", { type: "text", id: "judge-reason", value: jd.reason, placeholder: jd.verdict === "not_needed" ? "要らないには、理由が要ります" : "書かなくても付けられます", oninput: (e) => { jd.reason = e.target.value; check(); } })),
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

  // ---- 追加コンフィグ ----

  function loadExtra() {
    run("追加コンフィグを読んでいます", async () => {
      st.xtra.places = await api().ExtraPlaces();
      st.page = "extra";
    }, true);
  }

  // xtraLeave asks before the text that is not saved goes.
  async function xtraLeave() {
    const x = st.xtra;
    if (!x.name || x.text === x.saved) return true;
    return ask("編集をやめますか", `extra/${x.name} の保存していない変更は消えます。`, "やめる");
  }

  async function openExtra(name, isNew) {
    if (!(await xtraLeave())) return;
    const x = st.xtra;
    await run("追加コンフィグを読んでいます", async () => {
      const text = isNew ? "" : await api().ReadExtra(name);
      Object.assign(x, { name, text, saved: text, isNew: !!isNew, from: null, msg: "", notes: [] });
    });
  }

  // draftExtra opens the extra configuration of a device with the unhandled
  // lines of a group added at its end; nothing is written until it is saved.
  async function draftExtra(source, group, device) {
    if (!(await xtraLeave())) return;
    await run("追加コンフィグの下書きを作っています", async () => {
      const d = await api().DraftExtra(source, group, device);
      const x = st.xtra;
      x.places = await api().ExtraPlaces();
      const there = x.places.files.some((f) => f.name === d.name);
      Object.assign(x, { name: d.name, text: d.text, saved: there ? await api().ReadExtra(d.name) : "", isNew: !there,
        from: { source, group, device, by: st.jdg.by || "" }, notes: [],
        msg: `読めなかった行 ${d.lines} 行を、extra/${d.name} の末尾に足しました（まだ保存していません）。メーカーの書き方に合っているかを見て、直してから保存してください。` });
      st.page = "extra";
    });
  }

  const POS_LINE = /^\{#\s*position:\s*(after|before)\s+(\S+)\s*#\}\s*$/;

  function extraPage() {
    const x = st.xtra;
    const pl = x.places;
    const out = [
      h("h1", null, "追加コンフィグ"),
      h("p", { class: "dim" }, "パックが書けない設定を、extra フォルダのファイルに書いて、生成したコンフィグに差し込みます。全機器・グループ・機器ごとに 1 ファイルです。パラメータブックの値は {{ device.hostname }} のように書けます。 ",
        tip("ファイル名の決まり: _all.j2（全機器）・@グループ名.j2（そのグループ）・ホスト名.j2（その機器）。当てる先を選べば、名前はツールが決めます。保存する前の中身は state/backup に控えます。")),
      h("div", { class: "row" }, button("読み直す", loadExtra), button("フォルダを開く", () => show("extra"))),
    ];
    if (!pl) return out;
    const kinds = [["device", "機器"], ["group", "グループ"], ["all", "全機器"]];
    const targets = x.kind === "device" ? pl.devices : x.kind === "group" ? pl.groups : [];
    if (x.kind !== "all" && !targets.includes(x.target)) x.target = targets[0] || "";
    const kindWord = { all: "全機器", group: "グループ", device: "機器" };
    const create = async () => {
      let name = "";
      try {
        name = await api().ExtraName(x.kind, x.target);
      } catch (e) {
        st.error = errText(e);
        render();
        return;
      }
      openExtra(name, !pl.files.some((f) => f.name === name));
    };
    out.push(h("div", { class: "card" },
      h("h2", null, `ファイル（${pl.files.length}）`),
      pl.files.length ? h("div", { class: "table-wrap" }, h("table", null,
        h("thead", null, h("tr", null, h("th", null, "ファイル"), h("th", null, "当てる先"), h("th", null, ""))),
        h("tbody", null, pl.files.map((f) => h("tr", { class: f.name === x.name ? "sel" : null },
          h("td", { class: "code" }, "extra/" + f.name),
          h("td", null, `${kindWord[f.kind]}${f.target ? " " + f.target : ""}`, f.applies ? null : h("span", { class: "dim small" }, "（ブックのどの機器にも当たりません）")),
          h("td", null, button("開く", () => openExtra(f.name, false), { "data-open": f.name })))))))
        : h("p", { class: "dim" }, "まだありません。"),
      h("div", { class: "row" },
        h("span", null, "新しく作る・開く: "),
        select(kinds, x.kind, (v) => { x.kind = v; x.target = ""; render(); }),
        x.kind === "all" ? null : targets.length ? select(targets.map((t) => [t, t]), x.target, (v) => { x.target = v; })
          : h("span", { class: "dim" }, x.kind === "group" ? "（ブックの機器にグループがありません）" : "（ブックに機器がありません）"),
        button("開く", create, { disabled: x.kind !== "all" && !x.target, id: "extra-new" }))));
    if (!x.name) return out;

    const lines = x.text.split("\n");
    const m = POS_LINE.exec(lines[0] || "");
    const pos = m ? `${m[1]}:${m[2]}` : "";
    const posOpts = [["", "末尾の手前（既定）"]];
    for (const id of pl.positions) posOpts.push([`after:${id}`, `${id} の後`], [`before:${id}`, `${id} の前`]);
    if (pos && !posOpts.some((o) => o[0] === pos)) posOpts.push([pos, `${m[2]} の${m[1] === "after" ? "後" : "前"}（どのパックにもありません）`]);
    const setPos = (v) => {
      // The text as it is now: typing does not draw the page again.
      const now = x.text.split("\n");
      const rest = POS_LINE.test(now[0] || "") ? now.slice(1) : now;
      x.text = (v ? [`{# position: ${v.replace(":", " ")} #}`].concat(rest) : rest).join("\n");
      render();
    };
    const save = () => run("追加コンフィグを保存しています", async () => {
      const r = await api().SaveExtra(x.name, x.text, x.from);
      x.saved = x.text;
      x.isNew = false;
      x.msg = r.message;
      x.notes = r.notes || [];
      if (x.from) {
        x.backTo = x.from.source; // a way back to the lines not read (spec 14.18.7)
        x.from = null;
        st.jdg.views = {}; // the judgments changed: read again when the screen opens
      }
      x.places = await api().ExtraPlaces();
    });
    const del = async () => {
      if (!(await ask("追加コンフィグを消しますか", `extra/${x.name} を state/backup に移します（生成に使わなくなります）。`, "消す"))) return;
      run("追加コンフィグを消しています", async () => {
        x.msg = await api().DeleteExtra(x.name);
        Object.assign(x, { name: "", text: "", saved: "", from: null, notes: [] });
        x.places = await api().ExtraPlaces();
      });
    };
    const dirty = h("span", { class: "dim small" }, "（保存していない変更があります）");
    dirty.hidden = x.text === x.saved;
    const ta = h("textarea", { class: "fill-text", spellcheck: "false", rows: "24", value: x.text, id: "extra-text",
      oninput: (e) => { x.text = e.target.value; dirty.hidden = x.text === x.saved; } });
    out.push(h("div", { class: "card", id: "extra-edit" },
      h("h2", null, `extra/${x.name}`, x.isNew ? h("span", { class: "dim small" }, "（新しいファイル）") : null, " ", dirty),
      boxes("info", x.msg ? [x.msg] : []),
      x.backTo ? h("div", { class: "row" }, button("読めなかった行へ戻る", () => { st.jdg.source = x.backTo; x.backTo = ""; go("judge"); }, { class: "primary", id: "extra-back" })) : null,
      boxes("warn", x.notes),
      h("div", { class: "row" }, h("span", null, "差し込む位置: "), select(posOpts, pos, setPos),
        tip("節の名前は、生成したコンフィグの並びの区切り（system・interfaces・routing など）です。選ぶと、ファイルの 1 行目に差し込む位置を書きます")),
      ta,
      h("div", { class: "row" },
        button("保存", save, { class: "primary", id: "extra-save" }),
        x.isNew ? null : button("消す", del, { id: "extra-delete" }),
        button("閉じる", async () => { if (await xtraLeave()) { Object.assign(x, { name: "", text: "", saved: "", from: null, msg: "", notes: [] }); render(); } }))));
    if (!x.name) return out;
    return out;
  }

  // ---- 新旧対照表 ----

  function loadMapping() {
    run("旧コンフィグとパラメータブックを読んでいます", async () => {
      st.mig.msg = "";
      st.map.view = await api().Mapping();
    }, true);
  }

  // proposeIfs makes the proposals of 新旧対照表（ポート） without a second import
  // (spec 14.10.2).
  function proposeIfs() {
    run("新旧対照表（ポート）の案を作っています", async () => {
      st.mig.msg = await api().ProposeIfs();
      st.map.view = await api().Mapping();
      st.map.tab = "ifs";
    }, true);
  }

  // openRedo shows the card that makes the proposals of 新旧対照表（ポート） again for
  // the new host names picked (spec 14.11.4).
  function openRedo(hosts) {
    const picked = {};
    for (const x of hosts) picked[x] = true;
    st.map.redo = { picked, view: null };
    st.map.tab = "ifs";
    st.map.pin = "ifs";
    if (st.page !== "mapping") go("mapping");
    else render();
  }

  function redoCard(v) {
    const rd = st.map.redo;
    const hosts = v.hosts || [];
    const chosen = () => hosts.filter((x) => rd.picked[x]);
    const look = (write) => run(write ? "新旧対照表（ポート）の案を作り直しています" : "作り直した案を作っています", async () => {
      const r = await api().RedoIfs(chosen(), write);
      if (write) {
        st.mig.msg = r.message;
        st.map.redo = null;
        st.map.view = await api().Mapping();
      } else rd.view = r;
    }, true);
    const all = hosts.length > 0 && hosts.every((x) => rd.picked[x]);
    const rv = rd.view;
    return h("div", { class: "card", id: "redo" },
      h("h2", null, "新旧対照表（ポート）の案を作り直す"),
      h("p", null, "選んだ機器の 新旧対照表（ポート）の新 IF 名を、今の「新機種」・新コンフィグから作り直します。入れ替えるのは、根拠の入っている行（ツールの案のまま）と、新 IF 名が空の行です。根拠が空で新 IF 名のある行（人が書いた名前と -）は変えません。新しい案の無い行は、新 IF 名と根拠を空にします。"),
      h("p", { class: "dim small" }, "Excel で新 IF 名を直したときは、根拠も空にしておくと、作り直しで上書きされません（画面の表で直したときは、保存で根拠を空にします）。下の一覧で、書く前に確かめられます。"),
      h("div", { class: "row redo-hosts" },
        h("label", null, h("input", { type: "checkbox", checked: all, onchange: (e) => { for (const x of hosts) rd.picked[x] = e.target.checked; rd.view = null; render(); } }), " 全部"),
        hosts.map((x) => h("label", null, h("input", { type: "checkbox", checked: !!rd.picked[x], "data-host": x, onchange: (e) => { rd.picked[x] = e.target.checked; rd.view = null; render(); } }), " " + x))),
      h("div", { class: "row" },
        button("前と後を見る", () => look(false), { class: rv ? null : "primary", disabled: !chosen().length, id: "redo-look" }),
        button("書く", async () => {
          if (!(await ask("新旧対照表（ポート）の案を作り直す", `新旧対照表（ポート）の ${rv.rows.length} 行の新 IF 名と根拠を書き換えます。書く前に、今のブックの控えを state/backup に取ります。`, "書く"))) return;
          look(true);
        }, { class: rv ? "primary" : null, disabled: !rv || !rv.rows.length, id: "redo-write" }),
        button("やめる", () => { st.map.redo = null; render(); })),
      rv ? h("div", { id: "redo-result" },
        h("p", null, rv.message),
        rv.rows.length ? tableView({ name: "作り直す行", header: ["新ホスト名", "旧 IF 名", "今の新 IF 名", "今の根拠", "→ 新 IF 名", "→ 根拠"],
          rows: rv.rows.map((r) => [r.newHost, r.oldIf, r.was || "（空）", r.wasBasis, r.now || "（空）", r.basis]) }, { id: "redo/rows" }) : null) : null);
  }

  // ---- 機器ごとに結ぶ (spec 14.20) ----
  //
  // st.map.link holds the panel: ifs and devs are the sheets 新旧対照表（ポート）
  // and 新旧対照表（機器） as the book has them, pend the new interface names
  // picked here (sheet row → name) that are not saved yet.

  const BARE_NOW = ["（設定なし）", "無効（shutdown）"];

  async function openLink(host) {
    if (st.home && st.home.draft) {
      st.edit.msg = "表の編集に、保存していない変更があります。先に「保存」するか「下書きを捨てる」を押してから、もう一度「機器ごとに結ぶ」を押してください";
      openEdit({ sheet: "@ifs" });
      return;
    }
    const lk = st.map.link || { host: "", sel: 0, pend: {}, copy: null, typed: "" };
    await run("新旧対照表（ポート）を読んでいます", async () => {
      lk.ifs = await api().EditSheet("@ifs");
      lk.devs = await api().EditSheet("@devices");
    });
    if (!lk.ifs) return;
    st.map.link = lk;
    const hosts = linkHosts(lk);
    if (!hosts.includes(lk.host)) lk.host = hosts.find((x) => linkLeft(lk, x) > 0) || hosts[0] || "";
    linkPickFirst(lk);
    render();
  }

  function linkCols(v) {
    const k = (key) => v.columns.findIndex((c) => c.key === key);
    return { host: k("new_host"), oldHost: k("old_host"), old: k("old_if"), now: k("now"), neu: k("new_if"), basis: k("basis") };
  }

  // linkRows are the rows of 新旧対照表（ポート） of a new host, in the order of the sheet.
  function linkRows(lk, host) {
    const c = linkCols(lk.ifs);
    return lk.ifs.rows.filter((r) => r.row && (r.cells[c.host] || "").trim() === host);
  }

  function linkHosts(lk) {
    const c = linkCols(lk.ifs), seen = {}, out = [];
    for (const r of lk.ifs.rows) {
      const x = (r.cells[c.host] || "").trim();
      if (x && !seen[x]) { seen[x] = true; out.push(x); }
    }
    return out;
  }

  // linkVal is the new interface name of a row with what was picked here.
  function linkVal(lk, r) {
    if (r.row in lk.pend) return lk.pend[r.row];
    return (r.cells[linkCols(lk.ifs).neu] || "").trim();
  }

  function linkLeft(lk, host) {
    return linkRows(lk, host).filter((r) => !linkVal(lk, r)).length;
  }

  function linkPickFirst(lk) {
    const rows = linkRows(lk, lk.host);
    if (rows.some((r) => r.row === lk.sel)) return;
    const empty = rows.find((r) => !linkVal(lk, r));
    lk.sel = empty ? empty.row : rows.length ? rows[0].row : 0;
  }

  // linkSet puts a name into a row; "" empties it.
  function linkSet(lk, r, name) {
    const base = (r.cells[linkCols(lk.ifs).neu] || "").trim();
    if (name === base) delete lk.pend[r.row];
    else lk.pend[r.row] = name;
  }

  // linkNext moves the pick to the next row without a new name: below it
  // first, then from the top.
  function linkNext(lk) {
    const rows = linkRows(lk, lk.host);
    const i = rows.findIndex((r) => r.row === lk.sel);
    const after = rows.slice(i + 1).concat(rows.slice(0, i + 1));
    const next = after.find((r) => !linkVal(lk, r));
    if (next) lk.sel = next.row;
  }

  // linkTie gives the picked old interface the name; one that had it is
  // emptied (spec 14.20.1).
  function linkTie(name) {
    const lk = st.map.link;
    name = (name || "").trim();
    const rows = linkRows(lk, lk.host);
    const r = rows.find((x) => x.row === lk.sel);
    if (!r || !name) return;
    let moved = "";
    if (name !== "-") {
      const c = linkCols(lk.ifs);
      for (const o of rows) {
        if (o !== r && linkVal(lk, o) === name) {
          linkSet(lk, o, "");
          moved = (o.cells[c.old] || "").trim();
        }
      }
    }
    linkSet(lk, r, name);
    lk.msg = moved ? `${name} は旧 ${moved} から外して、こちらに結びました（旧 ${moved} は未決になりました）` : "";
    linkNext(lk);
    render();
  }

  function linkClear() {
    const lk = st.map.link;
    const r = linkRows(lk, lk.host).find((x) => x.row === lk.sel);
    if (!r) return;
    linkSet(lk, r, "");
    lk.msg = "";
    render();
  }

  function linkDropBare() {
    const lk = st.map.link, c = linkCols(lk.ifs);
    let n = 0;
    for (const r of linkRows(lk, lk.host)) {
      if (linkVal(lk, r) || !BARE_NOW.includes((r.cells[c.now] || "").trim())) continue;
      linkSet(lk, r, "-");
      n++;
    }
    lk.msg = n ? `設定のない旧 IF ${n} 行に - を入れました（保存すると書きます）` : "設定のない旧 IF で、新 IF 名が空の行はありませんでした";
    linkNext(lk);
    render();
  }

  // linkDevice gives the new vendor and model of a new host.
  function linkDevice(lk, host) {
    const v = lk.devs;
    if (!v) return { vendor: "", model: "" };
    const k = (key) => v.columns.findIndex((c) => c.key === key);
    const h = k("new_host"), ve = k("new_vendor"), m = k("new_model");
    const r = v.rows.find((x) => (x.cells[h] || "").trim() === host);
    return r ? { vendor: (r.cells[ve] || "").trim(), model: (r.cells[m] || "").trim() } : { vendor: "", model: "" };
  }

  // linkShapes are the devices the names of a host can be copied to (spec
  // 14.20.2): the same new model, with some old interface names alike.
  function linkShapes(lk) {
    const c = linkCols(lk.ifs);
    const src = linkDevice(lk, lk.host);
    const mine = new Set(linkRows(lk, lk.host).map((r) => (r.cells[c.old] || "").trim()));
    const out = [];
    for (const x of linkHosts(lk)) {
      if (x === lk.host) continue;
      const d = linkDevice(lk, x);
      if (d.model !== src.model || (!src.model && d.vendor !== src.vendor)) continue;
      const olds = linkRows(lk, x).map((r) => (r.cells[c.old] || "").trim());
      const hit = olds.filter((o) => mine.has(o)).length;
      if (!hit) continue;
      const same = hit === olds.length && new Set(olds).size === mine.size;
      out.push({ host: x, hit, all: olds.length, same });
    }
    return out;
  }

  function linkCopyOpen() {
    const lk = st.map.link;
    const picked = {};
    for (const s of linkShapes(lk)) picked[s.host] = s.same;
    lk.copy = { picked };
    render();
  }

  function linkCopy() {
    const lk = st.map.link, c = linkCols(lk.ifs);
    const names = {};
    for (const r of linkRows(lk, lk.host)) {
      const v = linkVal(lk, r);
      if (v) names[(r.cells[c.old] || "").trim()] = v;
    }
    let hosts = 0, n = 0, kept = 0;
    for (const x of Object.keys(lk.copy.picked).filter((y) => lk.copy.picked[y])) {
      let any = false;
      for (const r of linkRows(lk, x)) {
        const v = names[(r.cells[c.old] || "").trim()];
        if (!v) continue;
        const base = (r.cells[c.neu] || "").trim(), basis = (r.cells[c.basis] || "").trim();
        // A name a person wrote stays: in the book (no basis), or here.
        if (r.row in lk.pend || (base && !basis)) {
          if (linkVal(lk, r) !== v) kept++;
          continue;
        }
        if (base === v) continue;
        linkSet(lk, r, v);
        n++;
        any = true;
      }
      if (any) hosts++;
    }
    lk.copy = null;
    lk.msg = (n ? `${hosts} 台の ${n} 行に、${lk.host} の新 IF 名を写しました（保存すると書きます）` : "写す行はありませんでした")
      + (kept ? `。人が書いた新 IF 名の ${kept} 行は変えていません` : "");
    render();
  }

  async function linkSave() {
    const lk = st.map.link, c = linkCols(lk.ifs);
    const changed = [];
    for (const r of lk.ifs.rows) {
      if (!(r.row in lk.pend)) continue;
      changed.push({ row: r.row,
        key: { new_host: (r.cells[c.host] || "").trim(), old_if: (r.cells[c.old] || "").trim() },
        old: { new_if: r.cells[c.neu] || "" }, new: { new_if: lk.pend[r.row] } });
    }
    if (!changed.length) {
      lk.msg = "保存する変更はありません";
      render();
      return;
    }
    if (!(await ask("新旧対照表（ポート）を保存", `結んだ ${changed.length} 行の新 IF 名を、パラメータブック ${st.cas.book} に書きます。書く前に、今のブックの控えを state/backup に取ります。Excel でブックを開いているときは、先に閉じてください。`, "保存する"))) return;
    let res = null;
    await run("新旧対照表（ポート）を保存しています", async () => {
      await api().EditDraft({ sheet: "@ifs", changed, added: [], deleted: [], columns: [] });
      res = await api().EditSave({});
    });
    if (!res) return; // what went wrong is shown; the draft is in the table editor
    if (!res.saved) {
      // The book was changed elsewhere: the table editor shows what clashes.
      st.edit.msg = "パラメータブックが、結ぶ欄を開いた後にほかで変わっていました。下の表で確かめて「保存」を押してください";
      st.map.link = null;
      openEdit({ sheet: "@ifs" });
      return;
    }
    lk.pend = {};
    lk.msg = `${changed.length} 行を保存しました`;
    await run("新旧対照表を読み直しています", async () => {
      lk.ifs = await api().EditSheet("@ifs");
      lk.devs = await api().EditSheet("@devices");
      st.map.view = await api().Mapping();
      if (st.home) st.home = await api().Home();
    });
    askNext();
  }

  async function linkClose() {
    const lk = st.map.link;
    const n = Object.keys(lk.pend).length;
    if (n && !(await ask("結ぶ欄を閉じる", `保存していない ${n} 行の割り当てを捨てます。`, "捨てて閉じる"))) return;
    st.map.link = null;
    render();
  }

  function linkCard() {
    const lk = st.map.link, c = linkCols(lk.ifs);
    const hosts = linkHosts(lk);
    if (!hosts.length) {
      return h("div", { class: "card", id: "link" }, h("h2", null, "機器ごとに結ぶ"),
        h("p", null, "新旧対照表（ポート）に行がありません。先に「1. 新旧対照表（機器）」で新ホスト名を書いて保存してください。"),
        h("div", { class: "row" }, button("閉じる", linkClose)));
    }
    const rows = linkRows(lk, lk.host);
    const col = lk.ifs.columns[c.neu] || {};
    const choices = ((col.choiceSets && col.choiceSets[lk.host]) || []).filter((x) => x !== "-");
    // Names written here that the list does not have (typed ones).
    for (const r of rows) {
      const v = linkVal(lk, r);
      if (v && v !== "-" && !choices.includes(v)) choices.push(v);
    }
    const holder = {};
    for (const r of rows) {
      const v = linkVal(lk, r);
      if (v && v !== "-") (holder[v] = holder[v] || []).push((r.cells[c.old] || "").trim());
    }
    const pendN = Object.keys(lk.pend).length;
    const nextHost = hosts.find((x) => x !== lk.host && linkLeft(lk, x) > 0);
    const dev = linkDevice(lk, lk.host);
    const sel = rows.find((r) => r.row === lk.sel);
    const shapes = lk.copy ? linkShapes(lk) : [];
    const left = rows.map((r) => {
      const v = linkVal(lk, r), now = (r.cells[c.now] || "").trim();
      return h("div", { class: cls("link-item", r.row === lk.sel && "on", BARE_NOW.includes(now) && "bare", r.row in lk.pend && "pend"),
        "data-old": (r.cells[c.old] || "").trim(), onclick: () => { lk.sel = r.row; lk.msg = ""; render(); } },
      h("div", { class: "link-name" }, (r.cells[c.old] || "").trim(), h("span", { class: "link-to" }, "→ " + (v === "-" ? "-（持っていかない）" : v || "（未決）"))),
      h("div", { class: "link-now dim small" }, now));
    });
    const right = choices.map((x) => {
      const by = holder[x];
      return h("div", { class: cls("link-item", !by && "free"), "data-port": x, onclick: () => linkTie(x) },
        h("div", { class: "link-name" }, x, col.choiceNotes && col.choiceNotes[x] ? h("span", { class: "dim small" }, " " + col.choiceNotes[x]) : null),
        h("div", { class: "dim small" }, by ? "← 旧 " + by.join(", ") : "空き"));
    });
    return h("div", { class: "card", id: "link" },
      h("h2", null, "機器ごとに結ぶ"),
      h("p", null, "左で旧 IF を押してから、右で新しい IF を押すと結ばれます。結ぶと、左は次の未決の旧 IF へ進みます。全部結んだら「保存」を押してください。"),
      h("div", { class: "row" },
        h("span", null, "機器: "),
        select(hosts.map((x) => { const n = linkLeft(lk, x); return [x, `${x}（${n ? "残り " + n : "済み"}）`]; }), lk.host,
          (x) => { lk.host = x; lk.sel = 0; lk.copy = null; lk.msg = ""; linkPickFirst(lk); render(); }),
        nextHost ? button("残りのある次の機器へ", () => { lk.host = nextHost; lk.sel = 0; lk.copy = null; lk.msg = ""; linkPickFirst(lk); render(); }, { id: "link-next-host" }) : null,
        h("span", { class: "dim small" }, dev.model ? `新機種: ${dev.model}` : "新機種が空です（右に出るのは、この機器に書かれている名前だけです）")),
      lk.msg ? h("div", { class: "box info", id: "link-msg" }, lk.msg) : null,
      h("div", { class: "link-grid" },
        h("div", null, h("h3", null, "旧 IF"), h("div", { class: "link-list", id: "link-old" }, left),
          h("div", { class: "row" },
            button("-（持っていかない）", () => linkTie("-"), { disabled: !sel, id: "link-drop" }),
            button("外す（空にする）", linkClear, { disabled: !sel, id: "link-clear" }),
            button("設定のない旧 IF を「-」にする", linkDropBare, { id: "link-bare" }))),
        h("div", null, h("h3", null, "新しい IF"), h("div", { class: "link-list", id: "link-new" }, right.length ? right : h("p", { class: "dim" }, "候補がありません。下に名前を打って結んでください。")),
          h("div", { class: "row" },
            h("input", { type: "text", value: lk.typed, placeholder: "一覧に無い名前", id: "link-typed", oninput: (e) => { lk.typed = e.target.value; },
              onkeydown: (e) => { if (e.key === "Enter") { linkTie(lk.typed); lk.typed = ""; } } }),
            button("結ぶ", () => { linkTie(lk.typed); lk.typed = ""; }, { disabled: !sel, id: "link-typed-go" })))),
      lk.copy ? h("div", { class: "box", id: "link-copy" },
        h("p", null, `${lk.host} の新 IF 名を、同じ旧 IF 名の行に写します（新機種が同じ機器だけ）。空の行と、ツールの案のままの行に入れます。人が書いた新 IF 名は変えません。`),
        shapes.length ? h("div", { class: "row redo-hosts" }, shapes.map((s) => h("label", null,
          h("input", { type: "checkbox", checked: !!lk.copy.picked[s.host], "data-host": s.host, onchange: (e) => { lk.copy.picked[s.host] = e.target.checked; render(); } }),
          ` ${s.host}（${s.same ? "同じ形" : `重なる旧 IF ${s.hit} / ${s.all}`}）`))) : h("p", { class: "dim" }, "写せる機器がありません（新機種が同じで、旧 IF 名が重なる機器がありません）。"),
        h("div", { class: "row" },
          button("写す", linkCopy, { class: "primary", disabled: !shapes.some((s) => lk.copy.picked[s.host]), id: "link-copy-go" }),
          button("やめる", () => { lk.copy = null; render(); }))) : null,
      h("div", { class: "row" },
        button(pendN ? `保存（${pendN} 行）` : "保存", linkSave, { class: "primary", disabled: !pendN, id: "link-save" }),
        button("この割り当てを同じ形の機器へ写す", linkCopyOpen, { id: "link-copy-open", disabled: !!lk.copy }),
        button(pendN ? "やめる（捨てる）" : "閉じる", linkClose, { id: "link-close" })));
  }

  // loadNext asks for the steps of the case and the next thing to do (spec
  // 14.10.4, 14.12.1). The home is drawn first; the answer comes after it,
  // at once without counting the unhandled lines, then with them.
  async function loadNext() {
    if (!st.cas) return;
    st.nextFor = st.home;
    // An answer that comes after the case changed, or after a newer asking,
    // is dropped.
    const dir = st.cas.dir, seq = ++st.nextSeq;
    const gone = () => !st.cas || st.cas.dir !== dir || st.nextSeq !== seq;
    const ask = async (quick) => {
      try {
        return await api().Steps(quick);
      } catch (e) {
        return null;
      }
    };
    let p = await ask(true);
    if (gone()) return;
    st.steps = p;
    st.next = p ? p.next : null;
    drawNext();
    if (p && p.steps.some((x) => x.state === "counting")) {
      p = await ask(false);
      if (gone()) return;
      st.steps = p;
      st.next = p ? p.next : null;
      drawNext();
    }
  }

  // askNext asks for the next step a little later, once for asks that come
  // together (a step that ends and the screen it goes to), so that the
  // screen's own calls go first (spec 14.16.1).
  let nextTimer = 0;
  function askNext() {
    clearTimeout(nextTimer);
    nextTimer = setTimeout(loadNext, 300);
  }

  // drawNext shows the next step that was told: the home is drawn again,
  // another screen has only its line put in place (what is being typed in
  // the table stays; spec 14.16.1).
  function drawNext() {
    if (busy) return;
    if (st.page === "home") {
      render();
      return;
    }
    drawNav();
    const old = $("next-line");
    if (old) old.replaceWith(nextLine());
  }

  // The screens that show no line of the next step.
  const NO_NEXT_LINE = new Set(["home", "cases", "about"]);

  // nextLine is the next step at the top of a screen other than the home
  // (spec 14.16.1). On the screen it points into, it marks the button.
  function nextLine() {
    const n = st.next;
    if (!n) return h("div", { id: "next-line", hidden: true });
    const here = n.page === st.page;
    let go = null;
    if (n.page && !(n.page === "home" && !n.act)) {
      go = here
        ? (n.act ? button("ここ", () => { stepTarget(n.page, n.act); if (n.page === "edit" && n.act === "edit-first") { openEdit({ sheet: "devices" }); return; } render(); }, { id: "next-here", title: "この画面の、押すボタンに枠を付けます" }) : null)
        : button(n.page === "home" ? "そこへ" : "移る", () => goStep(n.page, n.act), { class: "primary", id: "next-go" });
    }
    return h("div", { class: "next next-line", id: "next-line" }, h("b", null, "次にやること"), h("span", null, n.text), go);
  }

  // stepTarget makes the screen of a step show the place of its operation
  // (the tab, the side) and points at its button (spec 14.18.6). The
  // 「移る」 and 「ここ」 of the next step both use it.
  function stepTarget(page, act) {
    st.hint = act;
    if (page === "mapping" && mapTabOf(act)) {
      st.map.tab = st.map.pin = mapTabOf(act);
      if (st.page === "mapping") st.map.pin = "";
    }
    if (page === "import") st.imp.source = "old";
    if (page === "judge") st.jdg.source = "old";
    if (page === "logs") st.logs.tab = act === "verify" ? "verify" : "load";
    if (act === "verify") st.hint = "logs-run";
    if (page === "result" && act === "findings") {
      st.res.tab = "findings";
      st.res.state = "todo";
      st.res.want = "findings";
      st.hint = "";
    }
    if (act === "edit-first") st.hint = "";
    // The ports are tied in the panel first (spec 14.20.1).
    if (act === "edit-ifs") st.hint = "link-open";
  }

  // goStep goes to the screen of a step, and points at its button.
  function goStep(page, act) {
    stepTarget(page, act);
    if (page === "home") { render(); return; }
    if (page === "edit" && act === "edit-first") { st.edit.back = ""; openEdit({ sheet: "devices" }); return; }
    go(page);
  }

  const STEP_WORDS = { done: "済み", todo: "残りあり", wait: "まだ", optional: "任意", counting: "数えています…" };

  // stepsBox is the row of the steps of the case (spec 14.12.1).
  function stepsBox() {
    const p = st.steps;
    if (!p || !p.steps || !p.steps.length) return null;
    const next = p.next || {};
    let marked = false;
    return h("div", { class: "steps", id: "steps" }, p.steps.map((x, i) => {
      const isNext = !marked && next.page && x.page === next.page && x.act === next.act && (x.state === "todo" || x.state === "optional");
      if (isNext) marked = true;
      return h("button", { type: "button", onclick: () => goStep(x.page, x.act),
        class: cls("step", x.state, isNext && "next"), "data-step": x.id, title: x.text || x.label },
      h("span", null, `${i + 1}. ${x.label}`),
      h("span", { class: "st" }, x.state === "todo" && x.note ? x.note : STEP_WORDS[x.state] || ""));
    }));
  }

  // nextBox is the one thing to do next, at the top of the home.
  function nextBox() {
    if (st.nextFor !== st.home) {
      st.nextFor = st.home;
      setTimeout(loadNext, 0);
    }
    const n = st.next;
    if (!n) return null;
    return h("div", { class: "next", id: "next" },
      h("b", null, "次にやること"),
      h("span", null, n.text),
      !n.page || (n.page === "home" && !n.act) ? null : button(n.page === "home" ? "そこへ" : "移る", () => goStep(n.page, n.act), { class: "primary", id: "next-go" }));
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

  // The three parts of the mapping screen, in the order of the work (spec
  // 14.17.1), and the steps of the case they do (spec 14.12.1).
  const MAP_PARTS = [
    { tab: "devices", step: "devices", label: "1. 新旧対照表（機器）" },
    { tab: "ifs", step: "ifs", label: "2. 新旧対照表（ポート）" },
    { tab: "open", step: "book", label: "3. 確かめて書く" },
  ];

  // mapTabOf is the part of the mapping screen an operation is on.
  function mapTabOf(act) {
    return { "edit-devices": "devices", "edit-ifs": "ifs", "propose-ifs": "ifs", "redo-ifs": "ifs", "make-book": "open" }[act] || "";
  }

  // mapTabNow is the part the mapping screen opens at: the first of its
  // steps with something left, else the last (spec 14.17.2).
  function mapTabNow() {
    const steps = (st.steps && st.steps.steps) || [];
    for (const part of MAP_PARTS) {
      const x = steps.find((y) => y.id === part.step);
      if (x && x.state === "todo") return part.tab;
    }
    return "open";
  }

  function mapParts() {
    const mp = st.map;
    const steps = (st.steps && st.steps.steps) || [];
    return h("div", { class: "steps map-parts", id: "map-parts" }, MAP_PARTS.map((part) => {
      const x = steps.find((y) => y.id === part.step) || {};
      const word = x.state === "todo" && x.note ? x.note : STEP_WORDS[x.state] || "";
      return h("button", { type: "button", onclick: () => { mp.tab = part.tab; st.hint = ""; render(); },
        class: cls("step", x.state, mp.tab === part.tab && "on"), "data-part": part.tab, id: "map-part-" + part.tab },
      h("span", null, part.label), h("span", { class: "st" }, word));
    }));
  }

  // copyHosts puts the old host name into each empty new host name, as a
  // draft of the table, and opens the table to look and save (spec 14.18.4).
  async function copyHosts() {
    if (st.home && st.home.draft) {
      // A draft of the sheet would be put over: save or throw it first.
      st.edit.msg = "表の編集に、保存していない変更があります。先に「保存」するか「下書きを捨てる」を押してから、もう一度「新ホスト名を旧ホスト名のままにする」を押してください";
      openEdit({ sheet: "@devices" });
      return;
    }
    let said = "";
    await run("下書きを作っています", async () => {
      const v = await api().EditSheet("@devices");
      const ci = (k) => v.columns.findIndex((c) => c.key === k);
      const changed = [];
      for (const r of v.rows) {
        const host = r.cells[ci("old_host")] || "", now = r.cells[ci("new_host")] || "", note = r.cells[ci("note")] || "";
        if (host && !now.trim() && !note.trim()) changed.push({ row: r.row, key: { old_host: host }, old: { new_host: now }, new: { new_host: host } });
      }
      if (changed.length) {
        const all = await api().EditDraft({ sheet: "@devices", changed, added: [], deleted: [], columns: [] });
        if (st.home) st.home.draft = all;
      }
      said = changed.length ? `${changed.length} 台の新ホスト名に、旧ホスト名を入れました（下書き）。確かめて「保存」を押してください` : "新ホスト名が空の行はありません";
    });
    if (!said) return;
    st.edit.msg = said;
    openEdit({ sheet: "@devices" });
  }

  function mappingPage() {
    const mp = st.map;
    const v = mp.view;
    const out = [
      h("h1", null, "新旧対照表"),
      h("p", { class: "dim" }, "旧機器と新しい機器の割り当てを、1 → 2 → 3 の順に決めます。押した段の中に、することとボタンが出ます。"),
      draftNote(),
    ];
    if (!v) return out;
    out.push(boxes("warn", v.notes));
    if (!v.hasBook) {
      out.push(h("div", { class: "box info" }, `パラメータブック ${v.book} はまだありません。旧コンフィグを取り込むと作られます。`),
        h("div", { class: "row" }, button("取り込みへ", () => { st.imp.source = "old"; go("import"); })));
      return out;
    }
    const n = (label) => {
      const x = (v.counts || []).find((p) => p.label === label);
      return x ? +x.value : 0;
    };
    const again = async () => { st.map.view = await api().Mapping(); };
    const reload = button("読み直す", loadMapping, { class: "link", title: "Excel で直したときなどに、旧コンフィグとパラメータブックを読み直します" });
    const excel = button("Excel で開く", () => show(st.cas.book), { title: "パラメータブックを Excel で開きます。Excel で書いたら、保存して閉じてから「読み直す」を押してください" });
    out.push(mapParts(), boxes("info", st.mig.msg ? [st.mig.msg] : []), boxes("warn", v.problems));
    if (mp.tab === "devices") {
      const left = n("うち新ホスト名がまだ空");
      const gone = n("うち持っていかない（新ホスト名が - か、空で備考あり）");
      out.push(h("div", { class: "card", id: "map-devices" },
        h("h2", null, "1. 新旧対照表（機器）"),
        h("p", null, "旧機器 1 台ごとに、置き換える新しい機器を書きます。「表で編集する」を押して、次の 3 つの列を埋めて保存してください。"),
        h("ul", null,
          h("li", null, h("b", null, "新ホスト名"), ": 新しい機器の名前"),
          h("li", null, h("b", null, "新メーカー"), ": ▼ から選びます（NEC IX は nec_ix、NEC QX は nec_qx、Cisco は cisco_ios）"),
          h("li", null, h("b", null, "新機種"), ": ▼ から選びます。選ぶと、2 のポートの割り当ての案が機種のポートの並びから入ります")),
        h("p", { class: "dim" }, "持っていかない機器は、新ホスト名に - と書きます（理由は備考に）。表では、列の中の複数のセルを選んでから ▼ で選ぶと、まとめて入ります。保存すると、2. 新旧対照表（ポート）へ進みます。"),
        h("p", null, (left ? `まだ書いていない機器: ${left} 台` : "すべての機器が決まっています。") + (gone ? `（持っていかない機器: ${gone} 台）` : "")),
        h("div", { class: "row" },
          button("表で編集する", () => openEdit({ sheet: "@devices" }), { class: "primary", id: "edit-devices" }),
          button("新ホスト名を旧ホスト名のままにする", copyHosts, { id: "copy-hosts", disabled: !left,
            title: "新ホスト名が空の行に、旧ホスト名を写して表を開きます（下書き。確かめて「保存」を押すまで、パラメータブックは変わりません）。名前を変えない更改のときに使います" }),
          excel, reload),
        tableView(v.devices, { id: "mapping/devices" })));
    } else if (mp.tab === "ifs") {
      const left = n("うち新 IF 名が空");
      const card = h("div", { class: "card", id: "map-ifs" },
        h("h2", null, "2. 新旧対照表（ポート）"),
        h("p", null, "旧機器の IF（旧 IF 名）を、新しい機器のどの IF（新 IF 名）にするかを確かめます。新機種を選んだ機器には、ツールの案が入っています（「案の根拠」の列）。"),
        h("ul", null,
          h("li", null, "案が合っていれば、そのままで大丈夫です"),
          h("li", null, "新 IF 名が空の行は、ツールが決められなかった所です。「表で編集する」で新 IF 名を書きます"),
          h("li", null, "新しい機器に持っていかない IF は、新 IF 名に - と書きます")),
        h("p", null, left ? `新 IF 名が空の行: ${left} 行` : "新 IF 名はすべて決まっています。"),
        h("p", null, "「機器ごとに結ぶ」で、旧 IF と新しい IF を左右に並べて、押して結べます。形が同じ機器が何台もあるときは、1 台を結んでから「この割り当てを同じ形の機器へ写す」を押します。"),
        h("div", { class: "row" },
          button("機器ごとに結ぶ", () => openLink(""), { class: mp.link ? null : "primary", id: "link-open", disabled: !!mp.link }),
          button("表で編集する", () => openEdit({ sheet: "@ifs" }), { id: "edit-ifs" }),
          button("案を作る", proposeIfs, { id: "propose-ifs",
            title: "input/old を読み直し、新ホスト名を書いた機器の 新旧対照表（ポート）の行を足し、空の新 IF 名を案で埋めます（人が書いた名前と - は変えません）。Excel で新旧対照表（機器）を書いたときに押してください" }),
          button("案を作り直す", () => openRedo([]), { disabled: !(v.hosts || []).length, id: "redo-ifs",
            title: "後から新機種を書いた・変えた機器の案を、今の機種で作り直します。ツールの案（根拠のある行）だけを入れ替え、人が書いた名前は変えません" }),
          excel, reload),
        h("p", { class: "dim small" }, "直して保存すると、3. 確かめて書く へ進みます。"),
        mp.redo ? redoCard(v) : null,
        mp.link ? linkCard() : null,
        mp.link ? null : tableView(v.ifs, { id: "mapping/ifs" }));
      out.push(card);
    } else {
      const open = v.open.rows.length;
      out.push(h("div", { class: "card", id: "map-open" },
        h("h2", null, "3. 確かめて書く"),
        h("p", null, open
          ? `新旧対照表で決まっていないところが ${open} 件あります。行を押すと、内容と直し方が出ます。直すときは、1 か 2 に戻って「表で編集する」で直してください（このまま進めてよいものは、理由を付けて承認できます）。`
          : "新旧対照表で決まっていないところはありません。下の「新しい名前に置き換えてパラメータブックに書く」を押してください。"),
        open ? migTable(v.open, v.findings, "mapping/open", again) : null,
        h("div", { class: "row" }, reload)),
      h("div", { class: "card" },
        h("h2", null, "パラメータブックに書く"),
        h("p", null, "旧コンフィグの値を、新旧対照表で新しいホスト名と IF 名に置き換えて、パラメータブックの設計データのシートに書きます。すでにある行は変えず、足りない行を足します。"),
        h("div", { class: "row" },
          h("label", null, h("input", { type: "checkbox", checked: mp.replace, onchange: (e) => { mp.replace = e.target.checked; } }),
            " 設計データのシートを空にしてから書く（作り直し）")),
        h("div", { class: "row" }, button("新しい名前に置き換えてパラメータブックに書く", async () => {
          if (mp.replace && !(await ask("作り直しますか", "設計データのシートを空にしてから書きます。画面の表や Excel で手で直した値は消えます（書く前の控えは残ります）。新旧対照表（機器）・新旧対照表（ポート）のシートは変えません。", "作り直す"))) return;
          makeBook("old", mp.replace);
        }, { class: "primary", id: "make-book" }))),
      mp.book && mp.book.from === "old" ? bookResult(mp.book) : null);
    }
    return out;
  }

  // ---- 結果 ----

  const NO_DEVICE = "-"; // the findings that belong to no one device

  function generate(checkOnly) {
    run(checkOnly ? "チェックしています" : "コンフィグを生成してチェックしています", async () => {
      showResult(checkOnly ? await api().Check() : await api().Generate());
      // The findings to fix or approve come first, as the next step is
      // to clear them (spec 14.18.6).
      const todo = st.res.view.findings.some((f) => (f.state === "open" || f.state === "fix") && (f.severity === "error" || f.severity === "warning"));
      st.res.tab = todo ? "findings" : "devices";
      if (todo) st.res.state = "todo";
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
      // After the case was opened again: the newest output is shown (spec
      // 14.10.6).
      if (hasBook && !rs.lastTried && st.cas) {
        rs.lastTried = true;
        setTimeout(loadLastResult, 0);
      }
      return [
        h("h1", null, "結果"),
        h("p", { class: "dim" }, hasBook
          ? "まだ結果がありません。「生成とチェック」か「チェックだけ」を押すと、ここに出ます。"
          : (st.cas && st.cas.kind === "新規" ? "パラメータブックがまだありません。ホームの「空のブックを作る」で作ります。" : "パラメータブックがまだありません。旧コンフィグを取り込むと作られます。")),
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
    if (v.rewrite) tabs.splice(1, 0, ["rewrite", "書き換え"]);
    if (rs.tab === "rewrite" && !v.rewrite) rs.tab = "devices";
    const out = [
      h("h1", null, "結果"),
      h("h2", null, v.title),
      v.message ? h("div", { class: "box info" }, v.message) : null,
      v.notice ? h("div", { class: "box warn" }, v.notice) : null,
      draftNote(),
      counts(v.counts),
      boxes("warn", v.notes),
      v.draft ? h("div", { class: "row", id: "draft-result" },
        button("表の編集へ戻る", () => openEdit(null), { class: "primary" }),
        h("span", { class: "dim small" }, "直したら、表の編集の「保存してチェック」で保存して、チェックし直します。")) : h("div", { class: "row" },
        button("パラメータブックを表で編集", () => openEdit(null)),
        button("パラメータブックを Excel で開く", () => show(st.cas.book)),
        // The reports and the folder are folded: the findings below are the
        // work (spec 14.18.8).
        h("details", { class: "fold-menu", id: "result-files", open: !!st.folds["result-files"], ontoggle: (e) => { st.folds["result-files"] = e.target.open; } }, h("summary", null, "レポート・出力フォルダを開く"),
          h("div", { class: "row" },
            button("チェック結果表を Excel で開く", () => show(v.report.xlsx)),
            button("チェック結果表をブラウザで開く", () => show(v.report.html)),
            v.rewriteReport && v.rewriteReport.xlsx ? button("書き換えレポートを Excel で開く", () => show(v.rewriteReport.xlsx)) : null,
            v.changes.has ? button("変更一覧を Excel で開く", () => show(v.changes.report.xlsx)) : null,
            v.changes.has ? button("変更一覧をブラウザで開く", () => show(v.changes.report.html)) : null,
            button("出力フォルダを開く", () => show(v.dir))))),
      h("div", { class: "row" }, h("span", { class: "dim small" }, "パラメータブックを直したら（画面の表なら「保存」をしてから）、もう一度:"), again),
      h("p", { class: "dim small" }, "画面では、秘密の列の値をいつも伏せて出します。書き出したコンフィグのファイルには、値がそのまま入っています。"),
      h("div", { class: "row", id: "tabs" }, seg(tabs, rs.tab, (x) => { rs.tab = x; render(); })),
    ];
    const draw = { devices: resultDevices, findings: resultFindings, checks: () => tableView(v.checks, { id: "result/checks" }), approvals: resultApprovals, changes: resultChanges, rewrite: resultRewrite };
    out.push(draw[rs.tab](v));
    return out;
  }

  // loadLastResult shows the result of the newest output of generate or
  // check, with the checks run again (spec 14.10.6).
  function loadLastResult() {
    run("前回の結果を読んでいます", async () => {
      const r = await api().LastResult();
      if (r) {
        showResult(r);
        st.res.tab = st.res.want || "devices";
        st.res.want = "";
      }
    }, true, "台");
  }

  // showConf shows the configuration of a device written to the output
  // folder, to read and copy (spec 14.10.5).
  function showConf(d) {
    const rs = st.res;
    run("コンフィグを読んでいます", async () => {
      rs.conf = await api().ReadFile(d.file);
      rs.confHost = d.label;
      st.jump = "conf-view";
    });
  }

  function confView(v) {
    const rs = st.res;
    const c = rs.conf;
    if (!c) return null;
    const failed = c.rel.includes("/_failed/");
    const copy = () => api().ClipboardSet(c.lines.join("\r\n") + "\r\n")
      .then(() => toast(`${rs.confHost} のコンフィグ（${c.lines.length} 行）をコピーしました`))
      .catch((e) => toast(errText(e), "err"));
    const host = (v.devices.find((d) => d.file === c.rel) || {}).host || "";
    return h("div", { class: "card", id: "conf-view" },
      h("h2", null, `${rs.confHost} のコンフィグ`),
      h("p", { class: "dim small" }, `${c.rel}（${c.lines.length} 行）。ファイルの中身そのままです（秘密の値も入っています）。`
        + (failed ? "文面がそろわなかった機器のファイルです。このまま機器に入れないでください。" : "")
        + (c.cut ? "長いため、先頭の部分だけを出しています（コピーはファイルを開いて行ってください）。" : "")),
      h("div", { class: "row" },
        button("全部コピー", copy, { class: "primary", id: "conf-copy", disabled: c.cut }),
        host ? button("この機器の指摘を見る", () => { rs.device = host; rs.tab = "findings"; rs.sel = ""; render(); }) : null,
        button("閉じる", () => { rs.conf = null; render(); })),
      h("pre", { class: "conf-text", id: "conf-text" }, c.lines.join("\n")));
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
    const withFile = v.devices.filter((d) => d.file);
    if (rs.confPick == null || !withFile.some((d) => d.file === rs.confPick)) rs.confPick = withFile.length ? withFile[0].file : "";
    return h("div", null,
      h("div", { class: "row" },
        h("label", null, h("input", { type: "checkbox", checked: rs.onlyBad, onchange: (e) => { rs.onlyBad = e.target.checked; render(); } }), " 指摘がある台だけ"),
        h("span", { class: "dim small" }, "機器を押すと、その機器の指摘を見られます。")),
      withFile.length ? h("div", { class: "row", id: "conf-row" },
        h("span", null, "コンフィグを見る:"),
        select(withFile.map((d) => [d.file, d.label]), rs.confPick, (x) => { rs.confPick = x; }),
        button("見る・コピー", () => showConf(withFile.find((d) => d.file === rs.confPick) || withFile[0]), { id: "conf-show" })) :
        h("p", { class: "dim small" }, v.configs ? "書き出したコンフィグはありません（値の誤りのある機器は書きません）。" : "「チェックだけ」ではコンフィグを書きません。コンフィグを見るには「生成とチェック」を押してください。"),
      confView(v),
      tableView(t, { id: `result/devices/${rs.onlyBad}`, onPick: pick }));
  }

  // resultRewrite is the tab 書き換え: every line of the old configurations
  // with where it went (spec 10.4, 13.4).
  function resultRewrite(v) {
    const rs = st.res;
    const rw = rs.rw || (rs.rw = { device: "", kind: "", view: null, loading: false, for: null });
    const load = () => {
      rw.loading = true;
      rw.for = v;
      api().Rewrite(rw.device, rw.kind).then((x) => { rw.view = x; rw.loading = false; if (st.page === "result" && !busy) render(); })
        .catch((e) => { rw.loading = false; st.error = errText(e); render(); });
    };
    if (rw.for !== v && !rw.loading) load();
    const x = rw.view;
    if (!x || rw.for !== v) return h("p", { class: "dim" }, "書き換えレポートを読んでいます…");
    if (!x.has) return h("p", { class: "dim" }, "この案件には書き換えレポートがありません（旧コンフィグがあり、「新旧対照表（機器）」に新ホスト名を書いた更改の案件で出ます）。");
    const pick = (field) => (val) => { rw[field] = val; load(); };
    return h("div", null,
      h("p", { class: "dim small" }, "旧コンフィグのすべての行を、どれか 1 つの区分に入れ、旧の行 → 行き先（パラメータブックの表と列）→ 新の行を並べます。読み合わせと入れ忘れの確認に使います。「自動で書き換えた」「無視」以外の行を見てください。"),
      h("div", { class: "row" },
        x.kinds.map((k) => button(`${k.word}（${k.count}）`, () => pick("kind")(rw.kind === k.id ? "" : k.id), { class: rw.kind === k.id ? "on" : null })),
        select([["", "すべての旧機器"]].concat(x.devices.map((d) => [d.device, d.device + (d.mark ? "（見る行あり）" : "")])), rw.device, pick("device")),
        x.report && x.report.xlsx ? button("書き換えレポートを Excel で開く", () => show(x.report.xlsx)) : null),
      boxes("info", x.notes),
      x.cut ? h("div", { class: "box warn" }, `${x.matched} 行のうち、先頭の 5,000 行を出しています。区分か機器で絞り込んでください。`) : null,
      tableView(x.table, { id: `result/rewrite/${rw.device}/${rw.kind}` }),
      x.newOnly && x.newOnly.rows.length ? h("details", null, h("summary", null, `新にだけある行（${x.newOnly.rows.length}）`), tableView(x.newOnly, { id: "result/rewrite/new" })) : null);
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
    // Findings picked to approve together (spec 14.10.8): only those a
    // person can approve.
    const can = (i) => !v.draft && list[i].state === "open";
    for (const k of Object.keys(rs.many)) if (!v.findings.some((f) => f.key === k && f.state === "open")) delete rs.many[k];
    const bar = h("div", { class: "row", id: "many-bar" });
    const drawBar = () => {
      const n = Object.keys(rs.many).length;
      bar.textContent = "";
      if (!n) {
        add(bar, [h("span", { class: "dim small" }, v.draft ? "下書きでチェックした結果では、承認はできません。" : "左の箱で指摘を選ぶと、同じ理由でまとめて承認できます（エラーも。承認は 1 件ずつ残ります）。")]);
        return;
      }
      const ok = button(`選んだ ${n} 件を承認する`, approveMany, { class: "primary", id: "many-go" });
      const check = () => { ok.disabled = !rs.manyReason.trim() || !(rs.by || "").trim(); };
      add(bar, [
        h("b", null, `${n} 件を選んでいます`),
        h("input", { type: "text", id: "many-reason", placeholder: "承認の理由", value: rs.manyReason, oninput: (e) => { rs.manyReason = e.target.value; check(); } }),
        h("input", { type: "text", id: "many-by", placeholder: "承認者", value: rs.by || "", oninput: (e) => { rs.by = e.target.value; check(); } }),
        ok,
        button("選ぶのをやめる", () => { rs.many = {}; render(); }, { class: "link" })]);
      check();
    };
    drawBar();
    const checks = { can, on: (i) => !!rs.many[list[i].key], set: (i, on) => { if (on) rs.many[list[i].key] = true; else delete rs.many[list[i].key]; }, changed: drawBar };
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
      bar,
      h("div", { class: "lines", id: "findings" },
        h("div", { class: "findings-list" }, tableView(t, { id: "result/findings", onPick: pick, selected: sel, checks })),
        side));
  }

  // approveMany approves the findings picked, one approval each, with one
  // reason (spec 14.10.8).
  async function approveMany() {
    const rs = st.res;
    const keys = Object.keys(rs.many);
    if (!keys.length) return;
    const errors = rs.view.findings.filter((f) => rs.many[f.key] && f.severity === "error").length;
    if (!(await ask("まとめて承認しますか", `選んだ ${keys.length} 件（うちエラー ${errors} 件）を、同じ理由で承認します。\n理由: ${rs.manyReason}\n\n承認は 1 件ずつ残ります（後から増えた指摘には当たりません）。`, "承認する"))) return;
    run("承認して、チェックをやり直しています", async () => {
      showResult(await api().ApproveMany(keys, rs.manyReason, rs.by || ""));
      rs.many = {};
      rs.manyReason = "";
    }, true, "台");
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
      f.at ? [h("dt", null, "パラメータブックのセル"), h("dd", { class: "code" }, f.at, " ", (o.rs ? null : rs.view && rs.view.draft && rs.view.moved) ? null : button("このセルを表で開く", () => { st.edit.from = f.device; openEditAt(f.at); }, { class: "link open-cell" }))] : null,
      h("dt", null, "内容"), h("dd", null, f.message),
      f.fix ? [h("dt", null, "直し方"), h("dd", null, f.fix)] : null,
      f.evidence.length ? [h("dt", null, "根拠"), f.evidence.map((p) => h("dd", { class: "code" }, `${p.label}: ${p.value}`))] : null,
      f.line ? [h("dt", null, `コンフィグの ${f.line} 行目`), h("dd", { class: "code" }, f.text)] : null,
      f.state === "approved" ? [
        h("dt", null, "承認"), h("dd", null, `${f.approval}（${f.by}、${when(f.when)}）`),
        h("dt", null, "承認の理由"), h("dd", null, f.reason)] : null)];

    if (o.extra) out.push(o.extra(f));
    if (f.state === "approved") {
      out.push(h("div", { class: "row" }, button("この承認を取り消す", () => (o.unapprove || unapprove)(f.approval))));
    } else if (f.state === "fix") {
      out.push(h("div", { class: "box warn" }, "この指摘は承認できません。パラメータブックの値（またはテンプレート・追加コンフィグ）を、画面の表か Excel で直して、もう一度チェックしてください。"),
        h("div", { class: "row" },
          f.at && !(rs.view && rs.view.draft && rs.view.moved) ? button("このセルを表で開く", () => { st.edit.from = f.device; openEditAt(f.at); }) : button("パラメータブックを表で編集", () => openEdit(null)),
          button("パラメータブックを Excel で開く", () => show(st.cas.book))));
    } else if (f.state === "open" && !o.rs && rs.view && rs.view.draft) {
      out.push(h("div", { class: "box info" }, "下書きでチェックした結果です。承認は、表の編集で「保存してチェック」をした後の結果で行います。"),
        h("div", { class: "row" }, button("表の編集へ戻る", () => openEdit(null))));
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
        h("p", { class: "dim small" }, "指摘の内容を確かめて、そのままでよいと判断したときに、理由を書いて承認します。承認は案件に残ります。"),
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
    // The fingerprint ties a row to its finding; it means nothing to a
    // person, so the list does not show it (the row keeps its place).
    const shown = col < 0 ? t : Object.assign({}, t, {
      header: t.header.filter((_, i) => i !== col),
      rows: t.rows.map((r) => r.filter((_, i) => i !== col)),
      code: (t.code || []).filter((c) => c !== col).map((c) => (c > col ? c - 1 : c)),
    });
    if (col < 0 || !items || !items.length) return tableView(shown, { id });
    const byKey = new Map(items.map((f) => [f.key, f]));
    const side = h("div", { class: "lines-side" });
    const detail = (f) => {
      side.textContent = "";
      add(side, [f
        ? findingDetail(null, f, { rs: mg, migration: true, extra, approve: (key) => approveMig(key, reload), unapprove: (id) => unapproveMig(id, reload) })
        : h("p", { class: "dim" }, "指摘のある行（「状態」の欄に字がある行）を押すと、指摘の内容と根拠を出します。承認もここで行います。")]);
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
      h("div", { class: "findings-list" }, tableView(shown, { id, onPick: pick, selected: sel })),
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
      h("p", { class: "dim" }, "この案件の承認と、今回それぞれが当たった指摘の件数です。"),
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
    if (!c.has && c.report && c.report.xlsx) {
      return h("div", null,
        h("p", { class: "dim" }, "変更一覧は、生成したときにだけ画面に出ます。この出力の変更一覧のファイルは開けます。"),
        h("div", { class: "row" },
          button("変更一覧を Excel で開く", () => show(c.report.xlsx)),
          button("変更一覧をブラウザで開く", () => show(c.report.html))));
    }
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

  // ---- 外部一覧 ----

  const OPS = { empty: "が空", notempty: "が空でない", contains: "が次を含む", notcontains: "が次を含まない", equals: "が次と同じ", notequals: "が次と違う" };
  const STEPS = {
    trim: "前後の空白を除く", width: "全角を半角にする", lower: "小文字にする", upper: "大文字にする",
    split: "区切って片方を残す", replace: "文字を置き換える", regex: "正規表現で置き換える", map: "置き換え表",
    cidr: "マスクをプレフィックス長にする",
  };
  const SOURCES = [["column", "一覧の列"], ["template", "組み立て"], ["value", "決まった値"]];

  function loadMaps() {
    run("列の割り当てファイルを読んでいます", async () => {
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
      h("h1", null, "外部一覧"),
      h("p", { class: "dim" }, "他のシステムなどが出力した一覧（input/customer/）を、列の割り当てファイル（mappings/）のとおりに読んで、パラメータブックと比べます。"),
      h("div", { class: "row" },
        seg([["compare", "比べる"], ["maps", "列の割り当て"]], cu.tab, pickTab),
        button("input/customer を開く", () => show("input/customer")),
        button("mappings を開く", () => show("mappings"))),
      dropZone("customer"),
      cu.tab === "compare" ? customerCompare() : customerMaps(),
    ];
  }

  // ---- ひな形から作る (spec 9.7, 14.7) ----

  function newFil() {
    return { view: null, tab: "templates", sel: "", name: "", text: "", saved: "", faults: null, msg: "", draft: { file: "", device: "", view: null },
      redit: null, rnew: "", run: { template: "", diff: "", devices: "", view: null, sel: "" } };
  }

  function loadFill(then) {
    run("ひな形を読んでいます", async () => {
      const fl = st.fil;
      fl.view = await api().Fill();
      const v = fl.view;
      if (fl.sel && !v.templates.some((t) => t.path === fl.sel)) fl.sel = "";
      if (!fl.sel && !fl.text && v.templates.length) openTemplate(v.templates[0]);
      if (!fl.run.template && v.templates.length) fl.run.template = v.templates[0].name;
      if (then) await then();
    });
  }

  function openTemplate(t) {
    const fl = st.fil;
    fl.sel = t ? t.path : "";
    fl.name = t ? t.name : "";
    fl.text = t ? t.text : "";
    fl.saved = fl.text;
    fl.faults = t && t.err ? [t.err] : null;
    fl.draft.view = null;
  }

  function fillPage() {
    const fl = st.fil;
    if (!fl.view) return [h("h1", null, "ひな形から作る"), h("p", { class: "dim" }, "読んでいます。")];
    return [
      h("h1", null, "ひな形から作る"),
      h("p", { class: "dim" }, "変数入りのひな形（fill/）に、一覧ファイル（input/customer/）の値とパラメータブックの値を入れて、機器ごとのコンフィグや投入を作ります。一覧のどの列をどの変数にするかは変数ルール（vars/）に書きます。"),
      h("div", { class: "row" },
        seg([["templates", "ひな形"], ["rules", "変数ルール"], ["run", "作る"]], fl.tab, (x) => { fl.tab = x; render(); }),
        button("読み直す", () => loadFill()),
        button("fill を開く", () => show("fill")),
        button("vars を開く", () => show("vars")),
        button("input/customer を開く", () => show("input/customer"))),
      boxes("info", fl.msg ? [fl.msg] : []),
      fl.tab === "templates" ? fillTemplates() : fl.tab === "rules" ? fillRules() : fillRun(),
    ];
  }

  // insertAt puts text at the caret of the editor (or around the chosen
  // lines, for a part: before and after are whole lines).
  function insertAt(ta, text, before, after) {
    const fl = st.fil;
    const s = ta.selectionStart, e = ta.selectionEnd, v = ta.value;
    if (before) {
      const ls = v.lastIndexOf("\n", s - 1) + 1;
      let le = v.indexOf("\n", Math.max(e - 1, s));
      if (le < 0) le = v.length;
      const mid = v.slice(ls, le);
      ta.value = v.slice(0, ls) + before + "\n" + mid + (mid ? "\n" : "") + after + v.slice(le);
    } else {
      ta.value = v.slice(0, s) + text + v.slice(e);
      ta.selectionStart = ta.selectionEnd = s + text.length;
    }
    fl.text = ta.value;
    ta.focus();
  }

  function fillTemplates() {
    const fl = st.fil;
    const v = fl.view;
    const ta = h("textarea", { class: "fill-text", spellcheck: "false", rows: "26", value: fl.text, oninput: (e) => { fl.text = e.target.value; }, id: "fill-text" });
    const pick = async (path) => {
      if (fl.text !== fl.saved && !(await ask("ひな形を切り替えますか", "保存していない変更は消えます。", "切り替える"))) { render(); return; }
      openTemplate(v.templates.find((t) => t.path === path) || null);
      render();
    };
    const save = () => run("ひな形を保存しています", async () => {
      let name = fl.name;
      if (!fl.sel) {
        name = (fl.name || "").trim();
        if (!name) throw new Error("新しいひな形の名前を入れてください（例: branch）");
      }
      const [rel, fault] = await api().TemplateSave(fl.sel, name, fl.text);
      fl.view = await api().Fill();
      openTemplate(fl.view.templates.find((t) => t.path === rel) || null);
      fl.faults = fault ? [fault] : (await api().TemplateCheck(fl.text));
      fl.msg = `保存しました: ${rel}`;
    });
    const check = () => run("ひな形を確かめています", async () => { fl.faults = await api().TemplateCheck(fl.text); });
    const remove = async () => {
      if (!(await ask("このひな形を消しますか", `${fl.sel} を fill から外します（state/backup に移すので、戻せます）。`, "消す"))) return;
      run("ひな形を消しています", async () => {
        const kept = await api().TemplateDelete(fl.sel);
        fl.msg = `消しました: ${fl.sel}（控え: ${kept}）`;
        openTemplate(null);
        fl.view = await api().Fill();
      });
    };
    const draft = () => run("1 台から案を作っています", async () => {
      const d = fl.draft;
      d.view = await api().FillDraft(d.file, d.device);
      fl.sel = "";
      fl.text = d.view.text;
      fl.saved = "";
      fl.name = fl.name || "";
      fl.faults = null;
    });
    const parts = [
      ["{{#each}}", "{{/each}}", "選んだ行を、一覧の行の数だけ繰り返します（間の変数から一覧を決めます）"],
      ["{{#if 名前}}", "{{/if}}", "選んだ行を、条件に合うときだけ出します（名前を書き換えます）"],
      ["{{#sync}}", "{{/sync}}", "今との差で投入を作るとき、選んだブロックをひな形どおりにそろえます（ひな形に無い行を消します）"],
    ];
    const opts = [["", "（新しいひな形）"]].concat(v.templates.map((t) => [t.path, t.name + (t.err ? "（誤りあり）" : "")]));
    const dr = fl.draft;
    if (!dr.file && v.configs.length) dr.file = v.configs[0];
    return [
      h("div", { class: "row" },
        h("span", null, "ひな形"), select(opts, fl.sel, pick),
        fl.sel ? null : [h("span", null, "名前"), h("input", { type: "text", value: fl.name, placeholder: "例: branch", oninput: (e) => { fl.name = e.target.value; }, id: "fill-name" })],
        button("保存", save, { class: "primary", id: "fill-save" }),
        button("確かめる", check, { id: "fill-check" }),
        fl.sel ? button("このひな形を消す", remove, { class: "link" }) : null),
      fl.faults && fl.faults.length ? h("div", { class: "box error" }, "ひな形の誤り:\n" + fl.faults.join("\n")) : fl.faults ? h("div", { class: "box info" }, "誤りはありません。") : null,
      h("div", { class: "row small" },
        h("span", { class: "dim" }, "選んだ行を囲む:"),
        parts.map(([b, a, tip]) => button(b + " … " + a, () => insertAt($("fill-text"), "", b, a), { title: tip })),
        button("{{! メモ}}", () => insertAt($("fill-text"), "{{! }}"), { title: "人のためのメモ。出力しません" })),
      v.names.length
        ? h("details", { class: "fill-names" }, h("summary", null, "変数を入れる（押すとカーソルの位置に入ります）"),
          v.names.map((g) => h("div", { class: "row small" },
            h("span", { class: "dim" }, g.title),
            button(`{{#each ${g.each}}}`, () => insertAt($("fill-text"), "", `{{#each ${g.each}}}`, "{{/each}}"), { class: "link", title: "選んだ行を、この一覧・表の行の数だけ繰り返します" }),
            g.names.map((n) => button(`{{${n}}}`, () => insertAt($("fill-text"), `{{${n}}}`), { class: "link" })))))
        : h("p", { class: "dim small" }, "変数ルールがまだありません。「変数ルール」のタブで作ると、ここに変数のボタンが出ます。"),
      ta,
      h("p", { class: "dim small" }, "書き方: {{名前}} 値（空ならその機器は作らない）、{{名前?}} 空ならその行を出さない、{{表.列}} パラメータブックの値、{{#each}}…{{/each}} 一覧の行の数だけ繰り返す、{{#if 名前}}…{{#else}}…{{/if}} 条件（名前=値・!名前も可）、{{#sync}}…{{/sync}} 今との差でそろえる所、{{#sync ip route}} その語の行をそろえる、{{! メモ}}。"),
      h("h3", null, "1 台から案を作る"),
      h("p", { class: "dim small" }, "実機 1 台のコンフィグと、その機器の値（変数ルールの一覧の行・ブックの「機器」の表）から、値の所を {{変数}} に置き換えた案を上の欄に入れます（まだ保存しません）。"),
      v.configs.length
        ? h("div", { class: "row" },
          h("span", null, "コンフィグ"), select(v.configs.map((f) => [f, f]), dr.file, (x) => { dr.file = x; }),
          h("span", null, "機器"), h("input", { type: "text", value: dr.device, placeholder: "例: rt-a01", oninput: (e) => { dr.device = e.target.value; }, id: "fill-draft-device" }),
          button("案を作る", draft, { id: "fill-draft" }))
        : h("p", { class: "dim small" }, "input/old・new・logs にコンフィグがありません。"),
      dr.view ? [
        boxes("info", [`${dr.view.file} の ${dr.view.device} から作りました。` + (dr.view.replaced.length ? "置き換えた変数: " + dr.view.replaced.join("、") : "置き換えた値はありません。")]),
        boxes("warn", dr.view.notes)] : null,
    ];
  }

  function newVarRule(file) {
    return { version: 1, file, sheet: "", sheetIndex: 0, headerRow: 0, dataRow: 0, startCol: "", mergeOnlyFirst: false, exclude: [],
      match: { width: false, space: false, domain: false, dash: false }, device: newField(""), vars: [], also: [], name: "", path: "" };
  }

  // locOf is a variable rule as a column mapping that says where the table
  // is: what the sample of a list takes.
  const locOf = (r) => ({ version: 1, file: r.file, sheet: r.sheet, sheetIndex: Number(r.sheetIndex) || 0, headerRow: Number(r.headerRow) || 0,
    dataRow: Number(r.dataRow) || 0, startCol: r.startCol, mergeOnlyFirst: r.mergeOnlyFirst, names: "new", exclude: [], rows: [], path: "" });

  // emptyOf is what an empty value of a variable does, as the page offers
  // it (PalaTerm's three: not made, the line left out, a fixed value).
  const emptyOf = (x) => x._empty || (x.empty === "line" ? "line" : x.default ? "value" : "stop");

  // varName makes a variable's name of a header (PalaTerm fills the name in
  // from the column chosen first).
  const varName = (hd) => hd.replace(/[^\p{L}\p{N}_-]+/gu, "_").replace(/^[^\p{L}_]+/u, "").replace(/_+$/, "") || "v";

  function cleanField(f) {
    const c = JSON.parse(JSON.stringify(f));
    const src = srcOf(c);
    delete c._src; delete c._try; delete c._tryFrom;
    c.fallback = (c.fallback || []).filter((x) => x);
    if (src !== "column") { c.column = ""; c.fallback = []; }
    if (src !== "template") c.template = "";
    if (src !== "value") c.value = "";
    return c;
  }

  function cleanVarRule(r) {
    const c = JSON.parse(JSON.stringify(r));
    c.headerRow = Number(c.headerRow) || 0;
    c.dataRow = Number(c.dataRow) || 0;
    c.sheetIndex = Number(c.sheetIndex) || 0;
    c.device = cleanField(r.device);
    c.vars = r.vars.map((x) => {
      const e = emptyOf(x);
      const f = Object.assign(cleanField(x), { name: x.name, empty: e === "line" ? "line" : "" });
      if (e !== "value") f.default = "";
      return f;
    });
    c.also = r.also.map((a) => ({ label: a.label || "", device: cleanField(a.device), vars: a.vars.map((x) => Object.assign(cleanField(x), { name: x.name, empty: "" })) }));
    c.exclude = r.exclude.map((x) => ({ column: x._on === "var" ? "" : x.column || "", var: x._on === "var" ? x.var || "" : "", op: x.op, value: x.value || "" }));
    c.match = Object.assign({ width: false, space: false, domain: false, dash: false }, r.match || {});
    return c;
  }

  async function varSample(ed) {
    ed.tryv = null;
    try {
      ed.sample = await api().ListSample(locOf(ed.rule));
      ed.err = "";
      if (!Number(ed.rule.headerRow)) ed.rule.headerRow = ed.sample.headerRow;
      ed.rule.sheet = ed.sample.sheet;
      // The device column: the guess of the host name column.
      if (!ed.rule.device.column) {
        const host = ed.sample.suggest.find((f) => f.to === "hostname" && f.column);
        const names = ed.sample.headers.map((x) => x.name);
        const guess = host ? host.column : names.find((x) => /host|ホスト|機器|device|装置|ルータ|router|switch|スイッチ/i.test(x)) || names[0];
        if (guess) ed.rule.device.column = guess;
      }
    } catch (e) {
      ed.sample = null;
      ed.err = errText(e);
    }
  }

  function fillRules() {
    const fl = st.fil;
    const v = fl.view;
    if (fl.redit) return varEditor();
    const t = {
      name: "変数ルール", header: ["ファイル", "一覧", "機器名の列", "変数", "状態"], code: [],
      rows: v.rules.map((x) => {
        const r = x.rule;
        return [x.path, r ? r.file : "", r ? r.device.column || r.device.template || r.device.value : "", r ? r.vars.map((y) => y.name).join("、") : "",
          x.problems.length ? "誤りがあります:\n" + x.problems.join("\n") : "読めます"];
      }),
      marks: v.rules.map((x) => (x.problems.length ? "unknown" : "")),
    };
    const open = (i) => {
      const x = v.rules[i];
      if (!x.rule) { st.error = `${x.path} は変数ルールとして読めません。テキストエディタで直してください:\n${x.problems.join("\n")}`; render(); return; }
      run("一覧を読んでいます", async () => {
        const ed = { path: x.path, name: x.name, rule: JSON.parse(JSON.stringify(x.rule)), sample: null, tryv: null, err: "", open: "" };
        await varSample(ed);
        fl.redit = ed;
        fl.msg = "";
      });
    };
    const create = (file) => run("一覧を読んでいます", async () => {
      const ed = { path: "", name: fl.rnew || "", rule: newVarRule(file), sample: null, tryv: null, err: "", open: "" };
      await varSample(ed);
      fl.redit = ed;
      fl.msg = "";
    });
    if (!fl.rfile || !v.lists.includes(fl.rfile)) fl.rfile = v.lists[0] || "";
    return [
      h("h2", null, "変数ルール"),
      h("p", { class: "dim" }, "「一覧のどの列を、どの変数にするか」を書いたファイルです。一覧の読み方（見出しの行・除外・加工）は外部一覧の列の割り当てと同じです。行を押すと開きます。"),
      v.rules.length ? tableView(t, { id: "fill/rules", onPick: open }) : h("p", { class: "dim" }, "まだありません。"),
      h("h2", null, "新しく作る"),
      v.lists.length
        ? h("div", { class: "row" },
          h("span", null, "一覧のファイル"), select(v.lists.map((f) => [f, f]), fl.rfile, (x) => { fl.rfile = x; }),
          button("この一覧の変数ルールを作る", () => create(fl.rfile), { class: "primary", id: "var-new" }))
        : h("p", { class: "dim" }, "input/customer にファイルがありません。一覧（xlsx・xls・csv）を置いてから「読み直す」を押してください。"),
    ];
  }

  function varEditor() {
    const fl = st.fil;
    const ed = fl.redit;
    const r = ed.rule;
    const sm = ed.sample;
    const v = fl.view;
    const headers = sm ? sm.headers.map((x) => x.name) : [];
    const text = (obj, key, props) => h("input", Object.assign({ type: "text", value: obj[key] || "", oninput: (e) => { obj[key] = e.target.value; } }, props));
    const headerSelect = (value, onpick) => {
      const opts = [["", "（選ぶ）"]].concat(headers.map((x) => [x, x]));
      if (value && !headers.includes(value)) opts.push([value, value + "（一覧にありません）"]);
      return select(opts, value || "", onpick);
    };
    const resampleV = () => run("一覧を読んでいます", () => varSample(ed));
    const firstValue = (f) => {
      if (!sm || srcOf(f) !== "column") return "";
      const hd = sm.headers.find((x) => x.name === f.column);
      if (!hd) return "";
      for (let i = Number(r.headerRow) || 0; i < sm.grid.length; i++) {
        const val = (sm.grid[i] || [])[hd.col - 1];
        if (val) return val;
      }
      return "";
    };
    // fieldCells are the cells of a field: where its value comes from, the
    // steps and the default.
    const fieldCells = (f, id, noDefault) => {
      const src = srcOf(f);
      return [
        h("td", null, select(SOURCES, src, (x) => { f._src = x; render(); })),
        h("td", null, src === "column" ? headerSelect(f.column, (x) => { f.column = x; if ("name" in f && !f.name && x) f.name = varName(x); render(); })
          : src === "template" ? text(f, "template", { placeholder: "{見出し}/{見出し}" }) : text(f, "value")),
        h("td", null, h("span", { class: "small" }, f.steps.length ? f.steps.map((s) => STEPS[s.type] || s.type).join(" → ") : "（なし）"), " ",
          button(ed.open === id ? "たたむ" : "直す", () => { ed.open = ed.open === id ? "" : id; render(); }, { class: "link" })),
        noDefault ? null : h("td", null, text(f, "default")),
      ];
    };
    const stepRow = (f, id, n) => (ed.open === id ? h("tr", { class: "steps" }, h("td", { colspan: String(n) }, stepEditor(f, false, firstValue(f)))) : null);
    const close = async () => {
      if (!(await ask("変数ルールを閉じますか", "保存していない変更は消えます。", "閉じる"))) return;
      fl.redit = null;
      render();
    };
    const tryIt = () => run("試しに読んでいます", async () => { ed.tryv = await api().VarTry(cleanVarRule(r)); st.jump = "var-try"; });
    const save = () => run("変数ルールを保存しています", async () => {
      ed.path = await api().VarSave(ed.path, ed.name, cleanVarRule(r));
      fl.msg = `保存しました: ${ed.path}`;
      fl.view = await api().Fill();
    });
    const remove = async () => {
      if (!(await ask("この変数ルールを消しますか", `${ed.path} を vars から外します（state/backup に移すので、戻せます）。`, "消す"))) return;
      run("変数ルールを消しています", async () => {
        const kept = await api().VarDelete(ed.path);
        fl.msg = `消しました: ${ed.path}（控え: ${kept}）`;
        fl.redit = null;
        fl.view = await api().Fill();
      });
    };
    const EMPTY = [["stop", "その機器は作らない"], ["line", "その行を出さない"], ["value", "決まった値を使う"]];
    if (!r.match) r.match = { width: false, space: false, domain: false, dash: false };
    for (const c of r.exclude) if (!c._on) c._on = c.var ? "var" : "column";
    const out = [
      h("h2", null, ed.path ? `変数ルール ${ed.path}` : "新しい変数ルール（まだ保存していません）"),
      h("div", { class: "row" },
        ed.path ? null : [h("span", null, "名前"), text(ed, "name", { placeholder: "例: sites（{{#each sites}} で呼べます）", id: "var-name" })],
        button("試しに読む", tryIt, { id: "var-try-button" }),
        button("保存", save, { class: "primary", id: "var-save" }),
        button("閉じる", close),
        ed.path ? button("このファイルを消す", remove, { class: "link" }) : null),
      boxes("error", ed.err ? [ed.err] : []),
      h("h3", null, "一覧のどこを読むか"),
      h("div", { class: "form-grid" },
        h("label", null, "一覧のファイル", text(r, "file", { onchange: resampleV })),
        h("label", null, "シート", sm && sm.sheets.length
          ? select(sm.sheets.map((x) => [x, x]), sm.sheet, (x) => { r.sheet = x; r.sheetIndex = 0; r.headerRow = 0; resampleV(); })
          : text(r, "sheet", { onchange: resampleV })),
        h("label", null, "見出しの行", h("input", { type: "number", min: "1", value: r.headerRow || "", onchange: (e) => { r.headerRow = Number(e.target.value) || 0; resampleV(); } })),
        h("label", null, "値の最初の行", h("input", { type: "number", min: "0", value: r.dataRow || "", placeholder: "見出しの次の行", onchange: (e) => { r.dataRow = Number(e.target.value) || 0; resampleV(); } })),
        h("label", null, "表の左端の列", text(r, "startCol", { placeholder: "自動（A、D、AB…）", onchange: resampleV })),
        h("label", null, "結合セル", select([["", "範囲の全部のセルに値がある"], ["1", "左上のセルだけに値がある"]], r.mergeOnlyFirst ? "1" : "", (x) => { r.mergeOnlyFirst = x === "1"; resampleV(); }))),
    ];
    if (sm) {
      const width = Math.max(1, ...sm.grid.map((row) => row.length));
      const cols = [];
      for (let c = 0; c < width; c++) cols.push(c);
      const letter = (n) => { let s = ""; for (n += 1; n > 0; n = Math.floor((n - 1) / 26)) s = String.fromCharCode(65 + ((n - 1) % 26)) + s; return s; };
      out.push(boxes("warn", sm.err ? [sm.err] : []), boxes("warn", sm.warnings),
        h("p", { class: "dim small" }, `${sm.file} のシート「${sm.sheet}」の上の ${sm.grid.length} 行です（全部で ${sm.rows} 行）。行番号を押すと、その行を見出しの行にします。`),
        h("div", { class: "table-wrap sample" }, h("table", null,
          h("thead", null, h("tr", null, h("th", null, ""), cols.map((c) => h("th", null, letter(c))))),
          h("tbody", null, sm.grid.map((row, i) => h("tr", { class: cls(i + 1 === Number(r.headerRow) && "hdr", i + 1 < Number(r.headerRow) && "above") },
            h("th", { class: "pick", onclick: () => { r.headerRow = i + 1; r.dataRow = 0; resampleV(); } }, String(i + 1)),
            cols.map((c) => h("td", { title: row[c] || null }, row[c] || ""))))))));
    }
    out.push(
      h("h3", null, "読まない行（除外）"),
      h("p", { class: "dim small" }, "どれか 1 つに当てはまる行は使いません（機器名のある行なら、その機器の続きの行も使いません）。「一覧の列」は元の値、「変数」は加工した後の値で決めます。"),
      r.exclude.map((c, i) => h("div", { class: "row" },
        select([["column", "一覧の列"], ["var", "変数"]], c._on, (x) => { c._on = x; render(); }),
        c._on === "var"
          ? select([["", "（選ぶ）"]].concat(r.vars.map((y) => [y.name, y.name])), c.var || "", (x) => { c.var = x; })
          : headerSelect(c.column, (x) => { c.column = x; }),
        select(v.ops.map((o) => [o, OPS[o] || o]), c.op, (x) => { c.op = x; render(); }),
        c.op === "empty" || c.op === "notempty" ? null : text(c, "value"),
        button("消す", () => { r.exclude.splice(i, 1); render(); }, { class: "link" }))),
      h("div", { class: "row" }, button("条件を足す", () => { r.exclude.push({ column: "", var: "", op: "equals", value: "", _on: "column" }); render(); })));
    const head = (first) => h("thead", null, h("tr", null, [first, "値の元", "値", "加工", "空のときの値"].map((x) => h("th", null, x))));
    out.push(
      h("h3", null, "機器名の列"),
      h("p", { class: "dim small" }, "ブックの機器・今のコンフィグのホスト名と突き合わせる列です（前後の空白・全角半角・大文字小文字の違いは見ません）。"),
      h("div", { class: "table-wrap" }, h("table", { class: "fields" }, head(""), h("tbody", null,
        h("tr", null, h("td", null, "★ 機器名"), fieldCells(r.device, "dev")), stepRow(r.device, "dev", 5)))),
      h("div", { class: "row small" },
        h("span", { class: "dim" }, "機器名の照合: 一覧の機器名とブックの機器名を比べるとき、次の違いを無視します（大文字・小文字と前後の空白は常に無視します）"),
        [["width", "全角・半角"], ["space", "空白"], ["domain", "ドメイン部（最初の「.」から後ろ）"], ["dash", "「-」と「_」"]].map(([k, l]) =>
          h("label", { class: "check" }, h("input", { type: "checkbox", checked: !!r.match[k], onchange: (e) => { r.match[k] = e.target.checked; } }), " " + l))),
      h("h3", null, "変数"),
      h("p", { class: "dim small" }, "名前は文字か _ で始め、文字・数字・_・- で付けます（列を先に選ぶと、名前は見出しから入ります）。ひな形には {{名前}} と書きます。同じ機器名の行が何行もあると、その機器の 2 行目以降になります。機器名のセルが空の行は、上の機器の続きの行です。"));
    const vrows = [];
    r.vars.forEach((x, i) => {
      const id = "v" + i;
      vrows.push(h("tr", null,
        h("td", null, text(x, "name", { placeholder: "例: wan_ip" })),
        fieldCells(x, id, true),
        h("td", null, select(EMPTY, emptyOf(x), (y) => { x._empty = y; render(); }), emptyOf(x) === "value" ? text(x, "default", { placeholder: "決まった値" }) : null),
        h("td", null, button("消す", () => { r.vars.splice(i, 1); ed.open = ""; render(); }, { class: "link" }))));
      vrows.push(stepRow(x, id, 6));
    });
    out.push(
      h("div", { class: "table-wrap" }, h("table", { class: "fields" },
        h("thead", null, h("tr", null, ["変数の名前", "値の元", "値", "加工", "値が空のとき", ""].map((x) => h("th", null, x)))),
        h("tbody", null, vrows))),
      h("div", { class: "row" },
        button("変数を足す", () => { r.vars.push(Object.assign(newField(""), { name: "", empty: "" })); render(); }, { id: "var-add" }),
        headers.length ? button("見出しから全部足す", () => {
          const have = new Set(r.vars.map((x) => x.column));
          for (const hd of headers) {
            if (hd === r.device.column || have.has(hd)) continue;
            r.vars.push(Object.assign(newField(""), { name: varName(hd), column: hd, empty: "" }));
          }
          render();
        }, { title: "一覧の見出しを、そのまま変数にします（名前は後で直せます）" }) : null));
    // The second and later devices of a line.
    out.push(h("h3", null, "同じ行の 2 台目以降（冗長の予備機など）"),
      h("p", { class: "dim small" }, "1 行に 2 台分（冗長構成のメインとバックアップなど）が載っている一覧は、2 台目を足して、2 台目の機器名の列と、2 台目で列が違う変数だけを選びます。選ばない変数は 1 台目と同じ列の値です。2 台目の機器名が空の行では 2 台目を作りません。機器名がどちらも空の行は、上の 2 台それぞれの続きです。"));
    r.also.forEach((a, ai) => {
      const rows = [h("tr", null, h("td", null, `★ ${ai + 2} 台目の機器名`, h("br"), text(a, "label", { placeholder: "名前（例: バックアップ）" })), fieldCells(a.device, `a${ai}`), h("td", null, ""))];
      rows.push(stepRow(a.device, `a${ai}`, 6));
      a.vars.forEach((x, i) => {
        const id = `a${ai}v${i}`;
        rows.push(h("tr", null,
          h("td", null, select(r.vars.map((y) => [y.name, y.name]), x.name, (y) => { x.name = y; })),
          fieldCells(x, id),
          h("td", null, button("消す", () => { a.vars.splice(i, 1); render(); }, { class: "link" }))));
        rows.push(stepRow(x, id, 6));
      });
      out.push(h("div", { class: "target-card" },
        h("div", { class: "table-wrap" }, h("table", { class: "fields" }, head(""), h("tbody", null, rows))),
        h("div", { class: "row" },
          button("この台の変数を足す", () => { a.vars.push(Object.assign(newField(""), { name: (r.vars[0] || {}).name || "", empty: "" })); render(); }, { disabled: !r.vars.length }),
          button(`${ai + 2} 台目を消す`, () => { r.also.splice(ai, 1); render(); }, { class: "link" }))));
    });
    out.push(h("div", { class: "row" }, button("2 台目以降を足す", () => { r.also.push({ device: newField(""), vars: [] }); render(); })));
    if (ed.tryv) {
      const tv = ed.tryv;
      out.push(h("h3", { id: "var-try" }, "試しに読んだ結果（何も書いていません）"));
      if (tv.faults.length) out.push(h("div", { class: "box error" }, "この変数ルールには誤りがあります:\n" + tv.faults.join("\n")));
      else {
        const more = tv.devices - tv.known;
        out.push(h("p", null, `${tv.file}: ${tv.lines} 行を読み（除外 ${tv.excluded} 行）、機器 ${tv.devices} 台の値になりました。`),
          tv.noBook ? h("p", { class: "dim small" }, "パラメータブックがまだ無いので、機器名をブックと照らし合わせていません。")
            : boxes(more > 0 ? "warn" : "info", [more > 0 ? `ブックに無い機器が ${more} 台あります: ${tv.unknown.join("、")}${more > tv.unknown.length ? " ほか" : ""}（機器名の照合の設定も見てください）` : `機器 ${tv.known} 台とも、ブックにあります。`]),
          boxes("warn", tv.missing.length ? [`一覧に、見出し「${tv.missing.join("」「")}」がありません。空として読みました。`] : []),
          boxes("warn", tv.warnings),
          h("h4", null, "先頭の行の読み方"),
          h("p", { class: "dim small" }, "一覧の元の値 → 変数の値。続きの行は、上の機器の行になります。"),
          tableView(tv.sample, { id: "fill/try-sample" }),
          h("h4", null, "機器ごとの値"),
          tableView(tv.table, { id: "fill/try" }));
      }
    }
    return out;
  }

  function fillRun() {
    const fl = st.fil;
    const v = fl.view;
    const rn = fl.run;
    if (!v.templates.length) return h("p", { class: "dim" }, "ひな形がまだありません。「ひな形」のタブで作ってください。");
    if (!v.templates.some((t) => t.name === rn.template)) rn.template = v.templates[0].name;
    const go = () => run(rn.diff ? "今との差から投入を作っています" : "ひな形に値を入れています", async () => {
      const devs = rn.devices.split(/[,、\s]+/).map((x) => x.trim()).filter(Boolean);
      rn.view = await api().FillRun(rn.template, rn.diff, devs);
      rn.sel = "";
    });
    const out = [
      h("div", { class: "form-grid" },
        h("label", null, "ひな形", select(v.templates.map((t) => [t.name, t.name]), rn.template, (x) => { rn.template = x; })),
        h("label", null, "作るもの", select([["", "全文（新規コンフィグ・変更コマンド）"], ["old", "今との差から投入（今 = 旧コンフィグ input/old）"], ["new", "今との差から投入（今 = 新コンフィグ input/new）"], ["logs", "今との差から投入（今 = 投入後のログ input/logs）"]], rn.diff, (x) => { rn.diff = x; render(); })),
        h("label", null, "機器", h("input", { type: "text", value: rn.devices, placeholder: "空 = 全部。例: rt-a01, rt-a02", oninput: (e) => { rn.devices = e.target.value; }, id: "fill-devices" }))),
      fillDevicePicker(rn),
      rn.diff ? h("p", { class: "dim small" }, "ひな形を「あるべき姿」として、機器ごとの今のコンフィグとの差から投入を作ります。足す行は全部、消す行（no …）は {{#sync}} の印の所だけです。保存のコマンドは付けません。") : null,
      h("div", { class: "row" }, button("作る", go, { class: "primary", id: "fill-run" })),
    ];
    const rv = rn.view;
    if (!rv) return out;
    out.push(
      h("h2", null, `結果: ${rv.template}`),
      h("div", { class: "row" },
        h("span", null, `機器 ${rv.devices.length}、作った ${rv.made}、作らなかった ${rv.notMade}`),
        rv.dir ? button("出力のフォルダを開く", () => show(rv.dir)) : null,
        button("報告書を開く", () => show(rv.report))),
      boxes("warn", rv.notes));
    const RES = { made: "作った", unchanged: "変更なし", not: "作らなかった" };
    const t = { name: "機器", header: ["機器", "結果", "理由・行数"], code: [],
      rows: rv.devices.map((d) => [d.name, RES[d.result], d.result === "made" ? `${d.lines} 行` + (d.left.length ? `（値が空で出さなかった行 ${d.left.length}）` : "") : d.why]),
      marks: rv.devices.map((d) => (d.result === "not" ? "unknown" : "")) };
    out.push(tableView(t, { id: "fill/run", onPick: (i) => { rn.sel = rv.devices[i].name; st.jump = "fill-one"; render(); }, selected: rv.devices.findIndex((d) => d.name === rn.sel) }));
    const d = rv.devices.find((x) => x.name === rn.sel);
    if (d) {
      out.push(h("h3", { id: "fill-one" }, `${d.name}: ${RES[d.result]}`),
        d.why ? h("div", { class: "box error" }, d.why) : null,
        d.values.length ? h("p", { class: "dim small" }, d.values.join("　")) : null,
        d.text ? h("div", { class: "row" },
          button(rv.diff ? "この機器の投入を全部コピー" : "この機器の文面を全部コピー", () => api().ClipboardSet(d.text.split(/\r?\n/).join("\r\n"))
            .then(() => toast(`${d.name} の${rv.diff ? "投入" : "文面"}をコピーしました`)).catch((e) => toast(errText(e), "err")), { class: "primary", id: "fill-copy" })) : null,
        d.text ? h("pre", { class: "fill-out" }, d.text) : null);
      if (rv.diff) {
        const list = (title, items, mark) => (items.length ? [h("h4", null, `${title}（${items.length}）`), h("pre", { class: "fill-out" }, items.map((x) => mark + " " + x).join("\n"))] : null);
        out.push(list("足す行", d.added, "+"), list("消す行", d.removed, "-"), list("残る行（ひな形に無いが消していない）", d.kept, "="));
      }
    }
    return out;
  }

  // fillDevicePicker lets the devices be picked from a list: those of the
  // book and those of the last run (spec 14.12.5). The picks go into the
  // field of the devices.
  function fillDevicePicker(rn) {
    const nb = st.nb;
    if (!nb.devices && !nb.loading && st.home && st.home.hasBook) {
      nb.loading = true;
      api().BookDevices().then((list) => { nb.devices = list || []; nb.loading = false; if (st.page === "fill" && !busy) render(); }).catch(() => { nb.devices = []; nb.loading = false; });
    }
    const names = [...new Set([].concat(nb.devices || [], rn.view ? rn.view.devices.map((d) => d.name) : []))];
    if (!names.length) return null;
    const chosen = new Set(rn.devices.split(/[,、\s]+/).map((x) => x.trim()).filter(Boolean));
    const set = (name, on) => {
      if (on) chosen.add(name);
      else chosen.delete(name);
      rn.devices = names.filter((x) => chosen.has(x)).concat([...chosen].filter((x) => !names.includes(x))).join(", ");
      render();
    };
    return h("details", { class: "fill-pick", id: "fill-pick", open: rn.pickOpen || null, ontoggle: (e) => { rn.pickOpen = e.target.open; } },
      h("summary", null, `機器を一覧から選ぶ（${names.length} 台。何も選ばなければ全部）`),
      h("div", { class: "row" },
        names.map((x) => h("label", null, h("input", { type: "checkbox", checked: chosen.has(x), onchange: (e) => set(x, e.target.checked) }), " " + x)),
        button("選ぶのをやめる", () => { rn.devices = ""; render(); }, { class: "link", disabled: !chosen.size })));
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
      h("h1", null, "検証機のログ"),
      h("p", { class: "dim" }, "検証機や本番機で取った端末ログ（input/logs/。1 台で 1 ファイル）を読みます。「投入ログ」は機器に弾かれたコマンドを探し、「投入後の確認」は投入後の設定全体（show running-config）をパラメータブックと比べます。"),
      h("div", { class: "row" },
        seg([["load", "投入ログ"], ["verify", "投入後の確認"]], what, (x) => { lg.tab = x; mg.sel = ""; render(); }),
        button("input/logs を開く", () => show("input/logs"))),
      dropZone("logs"),
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
    const text = `外部一覧の値を、パラメータブックに書きます（${n} 件）。書く前に、今のブックの控えを state/backup に取ります。Excel でブックを開いているときは、先に閉じてください。`;
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
    const read = () => run("外部一覧を読んで、パラメータブックと比べています", async () => {
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

  // ---- 列の割り当て ----

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
        delete f._try;
        delete f._tryFrom;
        f.fallback = (f.fallback || []).filter((x) => x);
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
    if (!m) return h("p", { class: "dim" }, "列の割り当てファイルを読んでいます。");
    if (cu.edit) return mapEditor();
    const t = {
      name: "列の割り当てファイル", header: ["ファイル", "一覧", "シート", "機器名", "行き先", "状態"], code: [],
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
    const dl = cu.del || (cu.del = { path: "" });
    if (!m.mappings.some((x) => x.path === dl.path)) dl.path = m.mappings.length ? m.mappings[0].path : "";
    const removeMap = async () => {
      const path = dl.path;
      if (!(await ask("この列の割り当てファイルを消しますか", `${path} を mappings から外します（state/backup に移すので、戻せます）。`, "消す"))) return;
      run("列の割り当てファイルを消しています", async () => {
        const kept = await api().ListDelete(path);
        cu.msg = `消しました: ${path}（控え: ${kept}）`;
        cu.maps = await api().Mappings();
      });
    };
    return [
      boxes("info", cu.msg ? [cu.msg] : []),
      h("h2", null, "列の割り当てファイル"),
      h("p", { class: "dim" }, "「一覧のどの見出しが、設計データのどの表のどの列か」を書いたファイルです。一覧 1 つ（シート 1 枚の表 1 つ）につき 1 つ作ります。行を押すと開きます。誤りがあるファイルが 1 つでもあると「読んで比べる」は始まりません。"),
      m.mappings.length
        ? tableView(t, { id: "customer/maps", onPick: (i) => {
          if (m.mappings[i].rule) openMap(m.mappings[i]);
          else { st.error = `${m.mappings[i].path} は、列の割り当てファイルとして読めません。テキストエディタで直してください:\n${m.mappings[i].problems.join("\n")}`; render(); }
        } })
        : h("p", { class: "dim" }, "まだありません。"),
      h("div", { class: "row" }, button("読み直す", loadMaps),
        m.mappings.length ? [
          h("span", { class: "dim" }, "消すファイル"),
          select(m.mappings.map((x) => [x.path, x.path]), dl.path, (x) => { dl.path = x; }),
          button("消す", removeMap, { id: "map-delete-listed" })] : null),
      h("h2", null, "新しく作る"),
      m.files.length
        ? h("div", { class: "row" },
          h("span", null, "一覧のファイル"), select(m.files.map((f) => [f, f]), cs.file, (x) => { cs.file = x; }),
          button("この一覧の列の割り当てを作る", () => newMap(cs.file), { class: "primary", id: "map-new" }))
        : h("p", { class: "dim" }, "input/customer にファイルがありません。「input/customer を開く」で、外部一覧（xlsx・xls・csv）を置いてから「読み直す」を押してください。"),
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
    // firstValue is the first value the list has for a field: what the
    // steps are tried on. The values of a secret column are not shown.
    const firstValue = (f, secret) => {
      if (!sm || secret || srcOf(f) !== "column") return "";
      const hd = sm.headers.find((x) => x.name === f.column);
      if (!hd) return "";
      for (let i = Number(r.headerRow) || 0; i < sm.grid.length; i++) {
        const v = (sm.grid[i] || [])[hd.col - 1];
        if (v) return v;
      }
      return "";
    };
    // fallbacks are the columns read in turn while the column is empty.
    const fallbacks = (f) => {
      f.fallback = f.fallback || [];
      return [
        f.fallback.map((fb, i) => h("div", { class: "fallback" },
          h("span", { class: "dim small" }, "空のときは →"),
          headerSelect(fb, (x) => { f.fallback[i] = x; }),
          button("消す", () => { f.fallback.splice(i, 1); render(); }, { class: "link" }))),
        f.column && f.fallback.length < 10
          ? button("空のときの別の列を足す", () => { f.fallback.push(""); render(); }, { class: "link small" })
          : null,
      ];
    };
    const remove = async () => {
      if (!(await ask("この列の割り当てファイルを消しますか", `${ed.path} を mappings から外します（state/backup に移すので、戻せます）。保存していない変更は消えます。`, "消す"))) return;
      run("列の割り当てファイルを消しています", async () => {
        const kept = await api().ListDelete(ed.path);
        cu.msg = `消しました: ${ed.path}（控え: ${kept}）`;
        cu.edit = null;
        cu.maps = await api().Mappings();
      });
    };
    const close = async () => {
      if (!(await ask("列の割り当てを閉じますか", "保存していない変更は消えます。", "閉じる"))) return;
      cu.edit = null;
      render();
    };
    const tryIt = () => run("試しに読んでいます", async () => { ed.tryv = await api().ListTry(cleanRule(r)); st.jump = "map-try"; });
    const save = () => run("列の割り当てを保存しています", async () => {
      ed.path = await api().ListSave(ed.path, cleanRule(r));
      cu.msg = `保存しました: ${ed.path}`;
      cu.maps = await api().Mappings();
    });

    const out = [
      boxes("info", cu.msg ? [cu.msg] : []),
      h("h2", null, ed.path ? `列の割り当て ${ed.path}` : "新しい列の割り当て（まだ保存していません）"),
      h("div", { class: "row" },
        button("試しに読む", tryIt, { id: "map-try-button" }),
        button("保存", save, { class: "primary", id: "map-save" }),
        button("閉じる", close),
        ed.path ? button("このファイルを消す", remove, { class: "link", id: "map-delete" }) : null),
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
            ? [headerSelect(f.column, (x) => { f.column = x; if (c.secret) resample(); else render(); }), fallbacks(f)]
            : src === "template"
              ? text(f, "template", { placeholder: "{見出し}/{見出し}" })
              : text(f, "value", { placeholder: c.values && c.values.length ? c.values.join(" | ") : "" })),
          h("td", null,
            h("span", { class: "small" }, f.steps.length ? f.steps.map((s) => STEPS[s.type] || s.type).join(" → ") : "（なし）"), " ",
            button(ed.open === id ? "たたむ" : "直す", () => { ed.open = ed.open === id ? "" : id; render(); }, { class: "link" })),
          h("td", null, text(f, "default")),
          h("td", null, c.key ? null : button("消す", () => { tg.columns.splice(fi, 1); ed.open = ""; render(); }, { class: "link" }))));
        if (ed.open === id) rows.push(h("tr", { class: "steps" }, h("td", { colspan: "6" }, stepEditor(f, !!c.secret, firstValue(f, c.secret)), c.note ? h("p", { class: "dim small" }, `列「${c.title}」（${c.type}）: ${c.note}`) : null)));
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
  function stepEditor(f, secret, first) {
    const m = st.cus.maps || st.fil.view;
    // The value the steps are tried on: the first one of the list, until the
    // column changes or a person types another.
    if (f._try == null || f._tryFrom !== f.column) { f._try = first; f._tryFrom = f.column; }
    const shown = (v) => (v === "" ? "（空）" : secret ? "●".repeat(Math.min(v.length, 12)) : v);
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
    const box = h("div", { class: "steps-box" },
      h("div", { class: "row" },
        h("span", { class: "dim" }, "試す値"),
        h("input", { type: secret ? "password" : "text", class: "try-value", value: f._try || "", placeholder: "ここに値を打つと、下に 1 段ごとの結果が出ます", oninput: (e) => { f._try = e.target.value; } })),
      f.steps.map((s, i) => h("div", { class: "row" },
        h("span", { class: "dim" }, `${i + 1}.`),
        select(m.steps.map((x) => [x, STEPS[x] || x]), s.type, (x) => { f.steps[i] = { type: x, rows: [] }; render(); }),
        params(s),
        button("消す", () => { f.steps.splice(i, 1); render(); }, { class: "link" }),
        h("span", { class: "after", "data-after": String(i) }))),
      h("div", { class: "row" },
        select(m.steps.map((x) => [x, STEPS[x] || x]), add.type, (x) => { add.type = x; }),
        button("加工を足す", () => { f.steps.push({ type: add.type, rows: [] }); render(); })),
      h("div", { class: "row" }, h("span", { class: "dim" }, "結果"), h("span", { class: "after", "data-result": "1" })),
      h("p", { class: "dim small" }, "上から順に当てます。「→」の後ろが、その段の後の値です。結果が空なら「空のときの値」になります。一覧の全部の行は「試しに読む」で確かめます。"));
    // trace asks for the value after each step and writes it in place (a
    // render would take the caret out of the box being typed in).
    let seq = 0;
    const trace = async () => {
      const my = ++seq;
      const v = f._try || "";
      let vals = [];
      try { vals = (await api().ListTrace(v, f.steps)) || []; } catch (e) { vals = []; }
      if (my !== seq) return;
      box.querySelectorAll("[data-after]").forEach((el) => {
        const x = vals[Number(el.getAttribute("data-after"))];
        el.textContent = "→ " + shown(x === undefined ? "" : x);
      });
      box.querySelector("[data-result]").textContent = shown(vals.length ? vals[vals.length - 1] : v);
    };
    box.addEventListener("input", trace);
    box.addEventListener("change", trace);
    trace();
    return box;
  }

  function tryView(v) {
    const out = [h("h3", { id: "map-try" }, "試しに読んだ結果（何も書いていません）")];
    if (v.faults.length) {
      out.push(h("div", { class: "box error" }, "この列の割り当てには誤りがあります。直してから、もう一度押してください:\n" + v.faults.join("\n")));
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

  // diffJump moves to the next (1) or the previous (-1) difference of the
  // comparison side by side (F7 / F8 as in PalaTerm's diff window).
  function diffJump(dir) {
    const rows = Array.from(document.querySelectorAll("#diff-side tr.chg"));
    if (!rows.length) return;
    const df = st.diff;
    df.cur = dir > 0 ? Math.min(rows.length - 1, df.cur + 1) : Math.max(0, df.cur - 1);
    rows.forEach((r, i) => r.classList.toggle("cur", i === df.cur));
    rows[df.cur].scrollIntoView({ block: "center" });
  }

  // marked draws a text with the parts in spans ([from, to) in UTF-16
  // units) marked.
  function marked(text, spans) {
    if (!spans || !spans.length) return [text];
    const out = [];
    let at = 0;
    for (const [a, b] of spans) {
      if (a > at) out.push(text.slice(at, a));
      out.push(h("mark", null, text.slice(a, b)));
      at = b;
    }
    if (at < text.length) out.push(text.slice(at));
    return out;
  }

  function diffSide(df) {
    const d = df.view;
    const rows = d.side.slice(0, df.shown).map((r) => r.op === "gap"
      ? h("tr", { class: "gap" }, h("td", { colspan: "4" }, "… " + r.ta + " …"))
      : h("tr", { class: cls(r.op, r.op !== "same" && "chg") },
        h("td", { class: "no" }, r.a ? String(r.a) : ""),
        h("td", { class: "tx a" }, r.a || r.op === "same" ? marked(r.ta, r.sa) : ""),
        h("td", { class: "no" }, r.b ? String(r.b) : ""),
        h("td", { class: "tx b" }, r.b || r.op === "same" ? marked(r.tb, r.sb) : "")));
    return h("div", { class: "diff side" }, h("table", { id: "diff-side" },
      h("thead", null, h("tr", null, h("th", { class: "no" }), h("th", null, "A"), h("th", { class: "no" }), h("th", null, "B"))),
      h("tbody", null, rows)));
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
    const save = async () => {
      try {
        const p = await api().DiffSave(df.a, df.b);
        if (p) toast("保存しました: " + p);
      } catch (e) {
        toast(errText(e), "err");
      }
    };
    const total = df.layout === "side" ? d.side.length : d.lines.length;
    return h("div", null,
      h("div", { class: "row" },
        seg([["stack", "上下"], ["side", "左右"]], df.layout, (m) => { df.layout = m; df.cur = -1; render(); }),
        df.layout === "side" ? [button("前の違い ▲", () => diffJump(-1), { title: "F8" }), button("次の違い ▼", () => diffJump(1), { title: "F7" })] : null,
        button("HTML に保存", save, { title: "左右に並べた差分を、1 つの HTML ファイルに保存します" }),
        h("span", null, `A にだけある行 ${d.removed} 行、B にだけある行 ${d.added} 行`),
        total > df.shown ? [h("span", { class: "dim small" }, `${total} 行のうち ${df.shown} 行を出しています`), button("さらに表示", () => { df.shown += LINES; render(); })] : null),
      df.layout === "side" ? diffSide(df) : h("div", { class: "diff" }, lines));
  }

  // ---- 表の編集 ----
  //
  // A sheet of the parameter book as a table that is edited the way a sheet
  // of Excel is (spec 14.6; the model is the table of the device list of
  // PalaTerm): select, type, Ctrl+C / Ctrl+V to and from Excel, Ctrl+D, undo,
  // rows put in between, a sort and filters over what is seen, search and
  // replace, the fill handle. Nothing reaches the book until 保存: until then
  // the changes are a draft the tool keeps in the case folder, written after
  // every change. The page knows no table and no column: both come with the
  // sheet.
  //
  // Two numbers name a column: its place among all the columns of the sheet
  // (ci: G.cols, the cells of a row) and its place among the ones that are
  // shown (c: G.vc, the selection, the cells that are drawn). G.vc[c] = ci.

  const G_PAD = 20; // the empty rows under the table, to type or paste into
  const G_ROWS = 1000; // rows of the book drawn at once
  const G_UNDO = 200;
  const G_RH = 58; // the width of the row numbers
  const G_VALUES = 1000; // the values a list to filter by shows
  const G_ARROWS = { ArrowUp: [-1, 0], ArrowDown: [1, 0], ArrowLeft: [0, -1], ArrowRight: [0, 1] };
  let G = null; // the open table
  let gUid = 0;
  let gDrag = null; // what the mouse drags: { mode: "cell" | "row" | "col" | "fill" | "rz", … }
  const gSorter = new Intl.Collator("ja", { numeric: true });

  // newFind is the bar of search and replace with nothing in it. It outlives
  // the sheet: a search goes on in the next one.
  function newFind() {
    return { open: false, replace: false, q: "", r: "", cs: false, whole: false, scope: "sheet", onlySel: false, range: null, said: "", all: 0, focus: false };
  }

  function newEdit() {
    return { list: null, want: null, msg: "", prefs: {}, find: newFind(), back: "" }; // back: the screen a finding opened the table from
  }

  // dialog asks with any content and any buttons; it gives the id of the
  // button that was pressed.
  function dialog(title, body, buttons) {
    return new Promise((resolve) => {
      const ov = h("div", { class: "overlay g-dialog" },
        h("div", { class: "dialog wide" },
          h("div", { class: "dialog-title" }, title),
          h("div", { class: "dialog-body" }, body),
          h("div", { class: "dialog-buttons" }, buttons.map((b) => button(b.label, () => { ov.remove(); resolve(b.id); }, { class: b.cls || null })))));
      document.body.appendChild(ov);
    });
  }

  const gDialogOpen = () => !!document.querySelector(".g-dialog") || !$("ask").hidden || !$("busy").hidden;

  // askText asks for one line of text; null = given up.
  function askText(title, text, value, yes, secret) {
    const input = h("input", { type: secret ? "password" : "text", value: value || "", class: "g-ask" });
    const body = h("div", null, h("p", null, text), input);
    const answer = dialog(title, body, [{ id: "cancel", label: "キャンセル" }, { id: "ok", label: yes, cls: "primary" }]);
    input.addEventListener("keydown", (e) => {
      e.stopPropagation();
      if (e.isComposing || e.keyCode === 229) return;
      if (e.key === "Enter") {
        e.preventDefault();
        body.closest(".overlay").querySelector("button.primary").click();
      }
    });
    setTimeout(() => { input.focus(); input.select(); }, 0);
    return answer.then((c) => (c === "ok" ? input.value : null));
  }

  // ---- rows and columns ----

  function gRowOf(r) {
    return {
      uid: ++gUid, row: r.row, sum: r.sum, base: r.base.slice(), cells: r.cells.slice(), formula: new Set(r.formula),
      theirs: Object.assign({}, r.theirs), bad: Object.assign({}, r.bad), gone: r.gone, del: r.deleted, clash: r.clash,
    };
  }

  function gNewRow() {
    return { uid: ++gUid, row: 0, sum: "", base: G.cols.map(() => ""), cells: G.cols.map(() => ""), formula: new Set(), theirs: {}, bad: {}, gone: false, del: false, clash: false };
  }

  // A new row nobody typed into is not saved. (The cells of a column that
  // was taken out do not count: G.live is the columns that are there.)
  const gBlank = (r) => r.row === 0 && G.live.every((i) => r.cells[i] === "");
  const gDirty = (r) => G.live.some((i) => r.cells[i] !== r.base[i]);
  const gColAt = (c) => G.cols[G.vc[c]];
  const gSorted = () => G.sort.ci >= 0;

  function gPad() {
    let n = 0;
    for (let i = G.rows.length - 1; i >= 0 && gBlank(G.rows[i]); i--) n++;
    for (; n < G_PAD; n++) G.rows.push(gNewRow());
  }

  // gLayout works out which columns are there (not taken out) and which are
  // shown (not hidden; the ones that tell the rows apart always are), and
  // where the first of them stand still when the table is scrolled sideways.
  function gLayout() {
    G.live = [];
    G.vc = [];
    G.cols.forEach((c, i) => {
      if (c.dropped) return;
      G.live.push(i);
      if (c.isKey || !G.pref.hidden.has(c.key)) G.vc.push(i);
    });
    G.lefts = [G_RH];
    while (G.lefts.length <= 3 && G.lefts.length <= G.vc.length && gColAt(G.lefts.length - 1).isKey) G.lefts.push(G.lefts[G.lefts.length - 1] + gColWidth(gColAt(G.lefts.length - 1)));
  }

  function gCounts() {
    const c = { rows: 0, cells: 0, added: 0, deleted: 0, moved: 0, bad: 0, clash: 0, cols: 0, unseen: 0 };
    const seen = new Set(G.vc);
    for (const r of G.rows) {
      if (gBlank(r)) continue;
      c.rows++;
      if (r.row === 0) c.added++;
      else if (r.del) {
        c.deleted++;
        if (r.clash) c.clash++;
      } else {
        if (!r.gone && G.moved.has(r.uid)) c.moved++;
        let any = false;
        for (const i of G.live) {
          if (r.cells[i] === r.base[i]) continue;
          c.cells++;
          any = true;
          if (i in r.theirs) {
            c.clash++;
            if (!seen.has(i)) c.unseen++;
          }
        }
        if (r.gone && any) c.clash++;
      }
      if (!r.del) {
        for (const k in r.bad) {
          if (!G.cols[k] || G.cols[k].dropped) continue;
          c.bad++;
          if (!seen.has(+k)) c.unseen++;
        }
      }
    }
    for (const col of G.cols) {
      if (col.added ? !col.dropped : col.dropped || col.was) c.cols++;
    }
    c.all = c.cells + c.added + c.deleted + c.moved + c.cols;
    return c;
  }

  // gMovedRows tells which rows of the book stand at another place than in
  // the book: all but the longest run of rows whose numbers still go up (the
  // fewest rows that have to move to get the order of the table).
  function gMovedRows() {
    const out = new Set(), seq = [];
    let sorted = true;
    for (const r of G.rows) {
      if (r.row === 0 || r.gone) continue;
      if (seq.length && r.row < seq[seq.length - 1].row) sorted = false;
      seq.push(r);
    }
    if (sorted) return out;
    const tails = [], prev = [];
    for (let i = 0; i < seq.length; i++) {
      let lo = 0, hi = tails.length;
      while (lo < hi) {
        const mid = (lo + hi) >> 1;
        if (seq[tails[mid]].row < seq[i].row) lo = mid + 1;
        else hi = mid;
      }
      prev[i] = lo > 0 ? tails[lo - 1] : -1;
      tails[lo] = i;
    }
    const stay = new Set();
    for (let i = tails[tails.length - 1]; i >= 0; i = prev[i]) stay.add(i);
    seq.forEach((r, i) => { if (!stay.has(i)) out.add(r.uid); });
    return out;
  }

  // gPart is the draft of this sheet: what was changed, with the values the
  // cells had when the table read them. A new row above a row of the book
  // is to go in there (before); the ones under the last row go to the end.
  // A row of the book that was moved goes to its place the same way (from).
  function gPart() {
    const p = { sheet: G.id, changed: [], added: [], deleted: [], columns: [] };
    for (const col of G.cols) {
      if (col.added) {
        if (!col.dropped) p.columns.push({ op: "add", title: col.title, to: "" });
      } else if (col.dropped) p.columns.push({ op: "drop", title: col.was || col.title, to: "" });
      else if (col.was) p.columns.push({ op: "rename", title: col.was, to: col.title });
    }
    const keyOf = (r) => {
      const k = {};
      G.cols.forEach((c, i) => { if (c.isKey) k[c.key] = r.base[i]; });
      return k;
    };
    let above = [];
    for (const r of G.rows) {
      if (r.row === 0) {
        if (gBlank(r)) continue;
        const cells = {};
        for (const i of G.live) if (r.cells[i] !== "") cells[G.cols[i].key] = r.cells[i];
        const a = { cells, before: 0, beforeKey: null, from: 0, fromKey: null };
        p.added.push(a);
        above.push(a);
        continue;
      }
      if (G.moved.has(r.uid)) {
        // A row of the book that stands at another place than in the book
        // goes there, like a new row; one that is deleted is deleted.
        if (!r.gone && !r.del) {
          const a = { cells: null, before: 0, beforeKey: null, from: r.row, fromKey: keyOf(r) };
          p.added.push(a);
          above.push(a);
        }
      } else if (!r.gone) {
        for (const a of above) {
          a.before = r.row;
          a.beforeKey = keyOf(r);
        }
        above = [];
      }
      if (r.del) {
        p.deleted.push({ row: r.row, key: keyOf(r), sum: r.sum });
      } else if (gDirty(r)) {
        const o = {}, n = {};
        for (const i of G.live) {
          if (r.cells[i] === r.base[i]) continue;
          o[G.cols[i].key] = r.base[i];
          n[G.cols[i].key] = r.cells[i];
        }
        p.changed.push({ row: r.row, key: keyOf(r), old: o, new: n });
      }
    }
    return p;
  }

  const gPartCount = (p) => p.changed.reduce((m, r) => m + Object.keys(r.new).length, 0) + p.added.length + p.deleted.length + p.columns.length;

  // ---- open, draft, save ----

  async function gLoad(id) {
    const v = await api().EditSheet(id);
    const pref = st.edit.prefs[v.id] || {};
    G = {
      id: v.id, v, cols: v.columns, rows: v.rows.map(gRowOf), live: [], vc: [], lefts: [G_RH],
      pref: { hidden: new Set(pref.hidden || []), widths: Object.assign({}, pref.widths || {}) },
      sel: { ar: 0, ac: 0, fr: 0, fc: 0 }, edit: null, list: null, undo: [], redo: [], show: false,
      filter: "", dev: "", colFilter: {}, hidden: null, sort: { ci: -1, dir: 1 }, rank: null,
      shown: G_ROWS, full: [], view: [], total: 0, moved: new Set(), tds: [], painted: [], clip: null,
      hadDraft: v.draft > 0, saveT: 0, savedAt: null, trouble: "", say: "", busy: false, scroll: [0, 0], tabC: null,
    };
    gLayout();
    gPad();
    gView();
    st.edit.find.range = null;
    st.edit.find.onlySel = false;
    const c = gCounts();
    if (c.all) G.say = `下書きを読み込みました（${c.all} 件）`;
  }

  // gReveal makes a row (by its place in G.rows) one of the rows that are
  // drawn, drawing more rows if it lies beyond them, and gives its place in
  // the view (-1 = a filter hides it).
  function gReveal(ri) {
    const vr = G.view.indexOf(ri);
    if (vr >= 0) return vr;
    const pos = G.full.indexOf(ri);
    if (pos < 0) return -1;
    let n = 0;
    for (let p = 0; p <= pos; p++) if (G.rows[G.full[p]].row !== 0) n++;
    G.shown = Math.ceil((n + 1) / G_ROWS) * G_ROWS;
    gView();
    return G.view.indexOf(ri);
  }

  // gJump puts the cursor on a cell of the sheet by its place in the book.
  function gJump(row, col) {
    const ci = G.cols.findIndex((c) => c.col === col && !c.dropped);
    const ri = G.rows.findIndex((r) => r.row === row && !r.gone);
    if (ri < 0) {
      G.say = `${row} 行目は、このシートにありません（Excel 側で行が動いたか、消えています）`;
      return;
    }
    // A cell a finding names is shown, whatever was hidden.
    if (ci >= 0 && G.vc.indexOf(ci) < 0) {
      G.pref.hidden.delete(G.cols[ci].key);
      gSavePref();
      gLayout();
    }
    const vr = gReveal(ri), c = Math.max(G.vc.indexOf(ci), 0);
    G.sel = { ar: vr, ac: c, fr: vr, fc: c };
    G.jumped = true;
  }

  // openEdit opens the page of the table: the sheet asked for (by its name in
  // the file or its internal name), or the one that was open, or the first.
  function openEdit(want) {
    gLeave().then(() => openEditNow(want));
  }

  function openEditNow(want) {
    st.page = "edit";
    st.error = "";
    run("パラメータブックを読んでいます", async () => {
      const ed = st.edit;
      ed.list = await api().EditSheets();
      ed.prefs = await api().EditPrefs();
      if (st.home) st.home.draft = ed.list.draft;
      if (!ed.list.hasBook || !ed.list.sheets.length) {
        G = null;
        return;
      }
      const id = (want && want.sheet) || (G && G.id) || (ed.list.sheets.find((s) => !s.missing) || ed.list.sheets[0]).id;
      await gLoad(id);
      if (want && want.row) gJump(want.row, want.col);
    });
  }

  // openEditAt opens the table at a cell a finding names ("シート!C5").
  function openEditAt(at) {
    // The save goes back to the screen of the finding (spec 14.18.7).
    if (st.page !== "edit") st.edit.back = st.page;
    const i = at.lastIndexOf("!");
    const want = { sheet: i < 0 ? at : at.slice(0, i), row: 0, col: 0 };
    const m = i < 0 ? null : /^([A-Z]+)(\d+)$/.exec(at.slice(i + 1));
    if (m) {
      want.row = +m[2];
      want.col = [...m[1]].reduce((n, ch) => n * 26 + ch.charCodeAt(0) - 64, 0) - 1;
    }
    openEdit(want);
  }

  function gScheduleSave() {
    clearTimeout(G.saveT);
    const g = G;
    G.saveT = setTimeout(() => gSaveDraft(g), 200);
  }

  // gSaveDraft writes the draft of the sheet. A draft that cannot be written
  // is told in the status line until one is.
  async function gSaveDraft(g) {
    g = g || G;
    if (!g) return;
    clearTimeout(g.saveT);
    g.saveT = 0;
    const part = gPartOf(g);
    const n = part.changed.length + part.added.length + part.deleted.length + part.columns.length;
    if (!n && !g.hadDraft) return;
    try {
      const all = await api().EditDraft(part);
      g.hadDraft = n > 0;
      g.trouble = "";
      g.savedAt = new Date();
      if (st.home) st.home.draft = all;
      if (st.edit.list) {
        st.edit.list.draft = all;
        const e = st.edit.list.sheets.find((x) => x.id === g.id);
        if (e) e.draft = gPartCount(part);
      }
    } catch (e) {
      g.trouble = "下書きを保存できません: " + errText(e);
    }
    if (g === G) gStatus();
  }

  function gPartOf(g) {
    const keep = G;
    G = g;
    try {
      return gPart();
    } finally {
      G = keep;
    }
  }

  // gLeave keeps what is being typed and writes the draft, before the table
  // goes out of sight.
  function gLeave() {
    if (!G) return Promise.resolve();
    gCommitEdit();
    gCloseList();
    gCloseMenus();
    return gSaveDraft(G);
  }

  // gSwitch opens another sheet. then, if given, runs once the sheet is
  // read and before it is drawn.
  async function gSwitch(id, then) {
    await gLeave();
    await run("シートを読んでいます", async () => {
      await gLoad(id);
      if (then) then();
    });
  }

  async function gReload() {
    await gLeave();
    openEdit({ sheet: G ? G.id : "" });
  }

  async function gDiscard() {
    if (!G || G.busy) return;
    gCommitEdit();
    const c = gCounts();
    if (!c.all && !G.hadDraft) {
      gSay("捨てる変更はありません");
      return;
    }
    if (!(await ask("下書きを捨てる", `シート「${G.v.name}」の保存していない変更（${c.all} 件）をすべて取り消し、パラメータブックの今の値に戻します。`, "捨てる"))) return;
    clearTimeout(G.saveT);
    const id = G.id;
    run("下書きを捨てています", async () => {
      const all = await api().EditDiscard(id);
      if (st.home) st.home.draft = all;
      st.edit.list = await api().EditSheets();
      await gLoad(id);
      G.say = "下書きを捨てました";
    });
  }

  async function gClose() {
    if (!G) {
      go("home");
      return;
    }
    await gLeave();
    const n = st.edit.list ? st.edit.list.draft : gCounts().all;
    if (n) {
      const c = await dialog("表の編集を終える",
        h("p", null, `保存していない変更が ${n} 件あります。下書きは残しておき、あとで「表の編集」から続けられます。「生成とチェック」などは、保存済みのパラメータブックで動きます。`),
        [{ id: "cancel", label: "キャンセル" }, { id: "discard", label: "下書きを捨てて閉じる" }, { id: "keep", label: "下書きを残して閉じる", cls: "primary" }]);
      if (c !== "keep" && c !== "discard") return;
      if (c === "discard") {
        try {
          await api().EditDiscard("");
        } catch (e) {
          st.error = errText(e);
          render();
          return;
        }
      }
    }
    const back = st.edit.back;
    G = null;
    st.edit = newEdit();
    // Back to the screen the table was opened from (spec 14.18.7).
    go(back && back !== "edit" ? back : "home");
  }

  // gSave writes the draft of every sheet into the book. A change that meets
  // one made in Excel since is asked about first.
  async function gSave() {
    if (!G || G.busy) return;
    await gLeave();
    const n = st.edit.list ? st.edit.list.draft : gCounts().all;
    if (!n) {
      gSay("保存する変更はありません");
      return "none";
    }
    if (!(await ask("表の変更を保存", `表の編集の下書き（${n} 件。すべてのシートの分）を、パラメータブック ${st.cas.book} に書きます。書く前に、今のブックの控えを state/backup に取ります。Excel でブックを開いているときは、先に閉じてください。`, "保存する"))) return "";
    const choice = {};
    const id = G.id;
    st.edit.msg = "";
    for (;;) {
      let res = null;
      await run("表の変更を保存しています", async () => { res = await api().EditSave(choice); });
      if (!res) return ""; // what went wrong is shown; the draft stays
      const open = res.conflicts.filter((c) => !(c.id in choice));
      if (!res.saved && open.length) {
        const picked = await gResolve(open);
        if (!picked) return "";
        Object.assign(choice, picked);
        continue;
      }
      let done = "";
      await run("パラメータブックを読み直しています", async () => {
        st.edit.msg = res.message;
        st.edit.redo = res.redo && res.redo.length ? res.redo : null;
        st.edit.list = await api().EditSheets();
        if (st.home) st.home = await api().Home();
        await gLoad(id);
        done = "saved";
      });
      return done;
    }
  }

  // gSaveCheck saves the draft and runs "check only" at once, then shows
  // the findings left (spec 14.12.3).
  async function gSaveCheck() {
    if (!G || G.busy) return;
    const saved = await gSave();
    if (!saved) return;
    await gCheckNow();
  }

  // gCheckNow runs "check only" and shows the findings left, for the device
  // the table was opened for.
  async function gCheckNow() {
    const from = st.edit.from || "";
    st.edit.back = "";
    const rs = st.res;
    await run("チェックしています", async () => {
      showResult(await api().Check());
      rs.tab = "findings";
      rs.state = "todo";
      rs.sev = "";
      rs.device = from && rs.view.devices.some((d) => d.host === from) ? from : "";
      G = null;
      st.page = "result";
    }, true, "台");
  }

  // mapSheetNote tells, over a sheet of the mapping in a migration case,
  // which columns to write and where the save goes on to (spec 14.18.3).
  function mapSheetNote() {
    const migration = !st.cas || st.cas.kind !== "新規";
    if (!G || !migration || (G.id !== "@devices" && G.id !== "@ifs")) return null;
    const back = button("新旧対照表へ戻る", () => go("mapping"), { class: "link", id: "g-to-mapping" });
    return G.id === "@devices"
      ? h("div", { class: "box info", id: "g-map-note" }, "色の付いた見出しの 3 列（新ホスト名・新メーカー・新機種）を書いて「保存」を押してください。▼ の一覧は、列の中の複数のセルを選んでから選ぶと、まとめて入ります。新機種を選ぶと、空の新メーカーも埋まります。持っていかない機器は、新ホスト名に - を書きます。保存すると、新旧対照表の「2. 新旧対照表（ポート）」へ進みます。 ", back)
      : h("div", { class: "box info", id: "g-map-note" }, "色の付いた見出しの「新 IF 名」を確かめて、空の行を書いて「保存」を押してください。新 IF 名は ▼（スペースか F4）で、新機種のポートから選べます（ほかの旧 IF に割り当て済みのものには印が出ます。一覧に無い名前も打てます）。「旧 IF の設定」を見て、設定のある IF から割り当ててください。持っていかない IF は - と書きます。保存すると、新旧対照表の「3. 確かめて書く」へ進みます。 ", back,
        h("div", { class: "row" },
          button("設定のない旧 IF を「-」にする", gIfsDropBare, { id: "g-ifs-bare", title: "新 IF 名が空で、旧 IF の設定が「（設定なし）」か「無効（shutdown）」だけの行に - を入れます（下書き。保存するまで書きません）" }),
          button("空いているポートを上から順に入れる", gIfsFill, { id: "g-ifs-fill", title: "新 IF 名が空の物理ポートの行に、その機器の新機種で空いているポートを、表の上から順に入れます。複数の行を選んでいれば、その行だけです（下書き。保存するまで書きません）" })));
  }

  // The bulk steps of 新旧対照表（ポート） (spec 14.19.9). They change the draft
  // of the table only; the save writes them.
  function gIfsCols() {
    const k = (key) => G.cols.findIndex((x) => x.key === key);
    return { host: k("new_host"), old: k("old_if"), now: k("now"), neu: k("new_if") };
  }

  // gIfsRows are the rows the steps go over: the selected ones when more than
  // one row is selected, else all, in the order of the table.
  function gIfsRows() {
    const g = gRange(), out = [];
    if (g.r1 > g.r0) {
      for (let r = g.r0; r <= g.r1; r++) out.push(gRowAt(r));
    } else {
      for (let r = 0; r < G.view.length; r++) out.push(gRowAt(r));
    }
    return out.filter((r) => r && !r.del && !gBlank(r));
  }

  function gIfsDropBare() {
    const c = gIfsCols();
    if (c.neu < 0 || c.now < 0) return;
    let n = 0;
    gMutate((ops) => {
      for (const r of gIfsRows()) {
        const now = (r.cells[c.now] || "").trim();
        if ((r.cells[c.neu] || "").trim() || !(now === "（設定なし）" || now === "無効（shutdown）")) continue;
        if (ops.set(r, c.neu, "-")) n++;
      }
    });
    gSay(n ? `設定のない旧 IF ${n} 行に - を入れました（保存すると書きます）` : "設定のない旧 IF で、新 IF 名が空の行はありませんでした");
  }

  // A logical interface or a subinterface is not a port to be given one of
  // the model: the proposals name those after the new vendor.
  const LOGICAL_IF = /^(lo|loopback|tu|tunnel|vlan|vl|di|dialer|po|port-channel|bvi|bdi|virtual|null|nve|vt|vi|cellular|async|bri|mfr|multilink|ppp|bridge|vlan-interface)/i;

  function gIfsFill() {
    const c = gIfsCols(), col = G.cols[c.neu];
    if (c.neu < 0 || c.host < 0 || !col || !col.choiceSets) return;
    // What every device has taken: the names in any row of it, the rows that
    // are not looked at too.
    const used = {};
    for (const r of G.rows) {
      if (!r || r.del || gBlank(r)) continue;
      const host = (r.cells[c.host] || "").trim(), v = (r.cells[c.neu] || "").trim();
      if (host && v && v !== "-") (used[host] = used[host] || {})[v] = true;
    }
    let n = 0, none = 0;
    gMutate((ops) => {
      for (const r of gIfsRows()) {
        const host = (r.cells[c.host] || "").trim(), old = (r.cells[c.old] || "").trim();
        if (!host || (r.cells[c.neu] || "").trim() || LOGICAL_IF.test(old) || /\.0*[1-9]\d*$/.test(old)) continue;
        const ports = (col.choiceSets[host] || []).filter((v) => v !== "-" && !(used[host] && used[host][v]));
        if (!ports.length) { none++; continue; }
        if (ops.set(r, c.neu, ports[0])) {
          (used[host] = used[host] || {})[ports[0]] = true;
          n++;
        }
      }
    });
    gSay((n ? `空いているポートを ${n} 行に入れました（保存すると書きます）` : "入れる行はありませんでした（新 IF 名が空の物理ポートの行がありません）")
      + (none ? `。${none} 行は、その機器の新機種に空いているポートがありません（新旧対照表（機器）の新機種を確かめてください）` : ""));
  }

  // gSaveGo saves the draft. In a migration case, a save made on a sheet of
  // the mapping goes on to the next part of the mapping screen (spec
  // 14.17.3): 新旧対照表（ポート） after 新旧対照表（機器）, 確かめて書く after it.
  async function gSaveGo() {
    const id = G && G.id;
    const saved = await gSave();
    if (saved !== "saved") return;
    const migration = !st.cas || st.cas.kind !== "新規";
    if (!migration || (id !== "@devices" && id !== "@ifs")) {
      // Opened from a finding: back to it, checked again (spec 14.18.7).
      const back = st.edit.back;
      if (back === "result") await gCheckNow();
      else if (back && back !== "edit") { st.edit.back = ""; go(back); }
      return;
    }
    const msg = st.edit.msg;
    const redo = id === "@devices" && st.edit.redo ? st.edit.redo : null;
    G = null;
    st.edit = newEdit();
    const mp = st.map;
    mp.tab = id === "@devices" ? "ifs" : "open";
    mp.redo = null;
    if (redo) {
      const picked = {};
      for (const x of redo) picked[x] = true;
      mp.redo = { picked, view: null };
    }
    st.hint = "";
    st.page = "mapping";
    st.mig = newMig();
    await run("新旧対照表を読み直しています", async () => {
      mp.view = await api().Mapping();
      st.mig.msg = msg;
    }, true);
  }

  // gDraftCheck checks the book with the draft put in, without writing it
  // (spec 14.12.3).
  async function gDraftCheck() {
    if (!G || G.busy) return;
    await gLeave();
    const rs = st.res;
    await run("下書きでチェックしています", async () => {
      showResult(await api().DraftCheck());
      rs.tab = "findings";
      rs.state = "todo";
      rs.device = "";
      G = null;
      st.page = "result";
    }, true, "台");
  }

  // gResolve asks which side stays for each change that meets one made in
  // Excel. It gives id → "mine" | "theirs", or null if the user gave up.
  function gResolve(list) {
    const picks = {};
    const cards = list.map((c) => {
      const what = { cell: "Excel 側で、同じセルが変わっています", gone: "Excel 側で、この行が消えたか、行を見分ける列が変わっています", deleted: "この表で消した行が、Excel 側で変わっています", sheet: "このシートは、もうパラメータブックにありません" }[c.kind];
      const mine = { cell: "この画面の値で上書きする", gone: "この画面の値で行を足す", deleted: "消す" }[c.kind];
      const theirs = { cell: "Excel の値を残す（この編集をやめる）", gone: "編集をやめる", deleted: "消すのをやめる", sheet: "このシートの編集をやめる" }[c.kind];
      picks[c.id] = mine ? "mine" : "theirs";
      const radio = (v, label) => {
        const el = h("input", { type: "radio", name: "gc-" + c.id, checked: picks[c.id] === v, onchange: () => { picks[c.id] = v; } });
        el.dataset.pick = v;
        return h("label", { class: "g-pick" }, el, " " + label);
      };
      return h("div", { class: "card g-conflict" },
        h("div", null, h("b", null, [c.sheet, c.row ? `${c.row} 行目` : "", c.label].filter(Boolean).join(" / ")), " ", h("span", { class: "dim small" }, what)),
        c.kind === "cell" ? h("table", null,
          h("thead", null, h("tr", null, ["列", "編集したときの値", "Excel の今の値", "この画面の値"].map((x) => h("th", null, x)))),
          h("tbody", null, h("tr", null, [c.column, c.old, c.theirs, c.mine].map((x) => h("td", null, x || "（空）"))))) : null,
        c.kind === "gone" && c.mine ? h("p", { class: "small" }, "この画面で直した値: " + c.mine) : null,
        mine ? radio("mine", mine) : null,
        radio("theirs", theirs));
    });
    const all = (v) => () => {
      for (const c of list) if (v === "theirs" || c.kind !== "sheet") picks[c.id] = v;
      body.querySelectorAll("input[type=radio]").forEach((el) => { el.checked = picks[el.name.slice(3)] === el.dataset.pick; });
    };
    const body = h("div", null,
      h("p", { class: "dim" }, "表を開いてから、パラメータブックが Excel などで変わりました。どちらを残すかを選んでください（選ぶまで、ブックには何も書きません）。"),
      list.length > 1 ? h("div", { class: "row" }, button("すべて、この画面の値にする", all("mine"), { class: "link" }), button("すべて、Excel の値を残す", all("theirs"), { class: "link" })) : null,
      h("div", { class: "g-conflicts" }, cards));
    return dialog(`Excel 側の変更とぶつかっています（${list.length} 件）`, body, [{ id: "cancel", label: "キャンセル" }, { id: "ok", label: "決定して保存", cls: "primary" }])
      .then((c) => (c === "ok" ? picks : null));
  }

  // ---- what is shown ----

  // gLabel is what a value shows as: a secret is masked until it is asked for.
  const gLabel = (col, v) => (v && col.secret && !G.show ? "●●●●" : v || "");

  // gShow is how a value shows in the table.
  function gShow(col, v) {
    if (!v) return "";
    if (col.secret) return G.show ? v : "●●●●";
    return v.indexOf("\n") < 0 ? v : v.replace(/\r?\n/g, " ⏎ ");
  }

  // The filters (the words, the device, the values picked for a column) are
  // applied once, as Excel does: a row that is edited or added afterwards
  // stays in sight until the filter changes.
  function gRefilter() {
    const words = G.filter.toLowerCase().split(/\s+/).filter(Boolean);
    const byCol = Object.keys(G.colFilter).map((k) => [+k, G.colFilter[k]]).filter(([ci]) => G.cols[ci] && !G.cols[ci].dropped);
    if (!words.length && !G.dev && !byCol.length) {
      G.hidden = null;
      return;
    }
    G.hidden = new Set();
    for (const r of G.rows) {
      if (r.row === 0) continue;
      let ok = !G.dev || (G.v.device >= 0 && r.cells[G.v.device] === G.dev);
      if (ok && words.length) {
        const all = G.vc.map((i) => gShow(G.cols[i], r.cells[i])).join("\n").toLowerCase();
        ok = words.every((w) => all.includes(w));
      }
      if (ok && byCol.length) ok = byCol.every(([ci, set]) => set.has(gLabel(G.cols[ci], r.cells[ci])));
      if (!ok) G.hidden.add(r.uid);
    }
  }

  const gFiltered = () => !!(G.filter.trim() || G.dev || Object.keys(G.colFilter).length);

  // gResort fixes the order of the rows by the column that is sorted by. As
  // with the filter, a row that is edited afterwards stays where it is, and
  // the rows that are filled in later follow in the order of the table. Only
  // what is seen changes: the rows of the book stay in their order.
  function gResort() {
    G.rank = null;
    if (!gSorted() || !G.cols[G.sort.ci] || G.cols[G.sort.ci].dropped) {
      G.sort = { ci: -1, dir: 1 };
      return;
    }
    const ci = G.sort.ci, col = G.cols[ci], dir = G.sort.dir;
    const order = [];
    G.rows.forEach((r, i) => { if (!gBlank(r)) order.push(i); });
    order.sort((a, b) => dir * gSorter.compare(gLabel(col, G.rows[a].cells[ci]), gLabel(col, G.rows[b].cells[ci])) || a - b);
    G.rank = new Map(order.map((i, n) => [G.rows[i].uid, n]));
  }

  // gView lists the rows that are drawn (by their place in G.rows): the ones
  // the filter leaves, in the order of the sheet or of the sort, the rows of
  // the book up to the number drawn at once. G.full is the same without that
  // limit (a search looks there).
  function gView() {
    G.moved = gMovedRows();
    const idx = [];
    G.rows.forEach((r, i) => { if (r.row === 0 || !G.hidden || !G.hidden.has(r.uid)) idx.push(i); });
    if (G.rank) {
      const big = G.rank.size, at = new Map();
      for (const i of idx) {
        const r = G.rows[i], x = G.rank.get(r.uid);
        at.set(i, gBlank(r) ? Infinity : x !== undefined ? x : big + i);
      }
      idx.sort((a, b) => (at.get(a) === at.get(b) ? a - b : at.get(a) - at.get(b)));
    }
    G.full = idx;
    let last = -1, total = 0;
    idx.forEach((i, p) => {
      if (G.rows[i].row === 0) return;
      last = p;
      total++;
    });
    G.total = total;
    if (total <= G.shown) {
      G.view = idx;
      return;
    }
    // Beyond the limit only the rows under the last row of the book are
    // drawn: the new ones, and the empty ones to type into.
    const out = [];
    let n = 0;
    idx.forEach((i, p) => {
      if (G.rows[i].row !== 0) n++;
      if (n <= G.shown || p > last) out.push(i);
    });
    G.view = out;
  }

  const gRowAt = (vr) => G.rows[G.view[vr]];

  // gKeepSel runs fn, which may change which rows and columns are shown and
  // in what order, and brings the selection along: the cursor follows its
  // row and its column. A range stays one only while it covers the same rows
  // in the same order; otherwise it folds to one cell (after a sort the two
  // ends lie far apart, with rows between them nobody selected).
  function gKeepSel(fn) {
    const s = G.sel;
    const uidAt = (r) => (G.view[r] != null && G.rows[G.view[r]] ? G.rows[G.view[r]].uid : null);
    const ua = uidAt(s.ar), uf = uidAt(s.fr), ka = G.vc[s.ac], kf = G.vc[s.fc];
    const lo = Math.min(s.ar, s.fr), hi = Math.max(s.ar, s.fr);
    const between = G.view.slice(lo, hi + 1).map((i) => G.rows[i].uid);
    fn();
    const at = (uid) => (uid == null ? -1 : G.view.findIndex((i) => G.rows[i].uid === uid));
    const na = at(ua), nf = at(uf);
    const nlo = Math.min(na, nf), nhi = Math.max(na, nf);
    const same = na >= 0 && nf >= 0 && nhi - nlo === hi - lo && G.view.slice(nlo, nhi + 1).every((i, k) => G.rows[i].uid === between[k]);
    if (same) {
      s.ar = na;
      s.fr = nf;
    } else if (na >= 0) s.ar = s.fr = na;
    else if (nf >= 0) s.ar = s.fr = nf;
    else s.fr = s.ar;
    const nac = ka == null ? -1 : G.vc.indexOf(ka), nfc = kf == null ? -1 : G.vc.indexOf(kf);
    if (nac >= 0 && nfc >= 0) {
      s.ac = nac;
      s.fc = nfc;
    } else if (nac >= 0) s.ac = s.fc = nac;
    else if (nfc >= 0) s.ac = s.fc = nfc;
    gClamp();
  }

  // gRedrawAll draws the table anew after the columns, the sort or a filter
  // changed. The frame is made again, its scroll offset is kept.
  function gRedrawAll() {
    gCloseMenus();
    gKeepSel(() => {
      gLayout();
      gRefilter();
      gView();
    });
    render();
  }

  function gColWidth(c) {
    if (G.pref.widths[c.key]) return G.pref.widths[c.key];
    const byType = { ip4: 130, ip6: 220, cidr4: 150, cidr6: 240, net4: 150, net6: 240, bool: 80, int: 90, text: 220 };
    const title = [...c.title].reduce((n, ch) => n + (ch.charCodeAt(0) > 255 ? 14 : 8), 46);
    return Math.min(280, Math.max(byType[c.type] || (c.free ? 160 : 120), title));
  }

  function gColTip(c) {
    const parts = [c.title];
    if (c.added) parts.push("足した自由列です。保存すると、シートの右端に見出しを足します");
    else if (c.free) parts.push("定義に無い列（自由列）");
    else if (c.type) parts.push("型: " + (c.list ? `${c.type} の並び` : c.type));
    if (c.was) parts.push(`保存すると、見出しを「${c.was}」から変えます`);
    if (c.mark === "write") parts.push("書く列です");
    if (c.mark === "shown") parts.push("表示のための列です（書き換えても、ツールの動きは変わりません）");
    if (c.mark === "tool") parts.push("ツールが書く列です");
    if (c.note) parts.push(c.note);
    if (c.secret) parts.push("秘密の値（伏せて出します）");
    if (c.col < 0 && !c.added) parts.push("シートに無い列です。値を入れて保存すると、シートの右端に見出しを足します");
    return parts.join("\n");
  }

  // gSavePref keeps the hidden columns and the widths of this sheet with the
  // settings of the user (not in the case).
  function gSavePref() {
    const pref = { hidden: [...G.pref.hidden], widths: Object.assign({}, G.pref.widths) };
    st.edit.prefs[G.id] = pref;
    api().EditPref(G.id, pref).catch(() => {});
  }

  function gRowEl(vr) {
    const r = gRowAt(vr);
    const blank = gBlank(r);
    const dirty = !blank && gDirty(r);
    const moved = !blank && r.row !== 0 && !r.del && !r.gone && G.moved.has(r.uid);
    const mark = blank ? "" : r.row === 0 ? "新" : r.del ? "×" : r.gone ? "？" : moved ? "↕" : dirty ? "●" : "";
    const tip = blank ? "" : r.row === 0 ? "新しい行（まだ保存していません。保存すると、この位置に入ります。表の下の行は、シートの末尾に入ります）"
      : r.del ? (r.clash ? "消す行（Excel 側で変わっています。保存のときに選べます）" : "消す行（保存するまで、パラメータブックには残っています）")
        : r.gone ? "Excel 側で消えたか、行を見分ける列が変わった行です（保存のときに選べます）"
          : moved ? "動かした行（保存すると、シートのこの位置へ動きます。行番号は、今のブックの行番号です）" : dirty ? "直した行（まだ保存していません）" : "";
    const th = h("th", { class: "g-rh", title: tip || null, "data-r": String(vr) }, mark ? h("span", { class: cls("g-mark", r.row === 0 && "new") }, mark) : null, r.row ? String(r.row) : "");
    const tr = h("tr", { class: cls(blank && "g-blank", r.row === 0 && !blank && "g-new", r.del && "g-del", (r.gone || (r.del && r.clash)) && "g-conf") || null }, th);
    for (let c = 0; c < G.vc.length; c++) {
      const ci = G.vc[c], col = G.cols[ci];
      const v = r.cells[ci];
      const bad = !r.del && r.bad[ci];
      const conf = !r.del && v !== r.base[ci] && ci in r.theirs;
      const ro = r.formula.has(ci);
      const td = document.createElement("td");
      const k = cls(c < G.lefts.length - 1 && "g-stick", !blank && v !== r.base[ci] && "chg", bad && "err", conf && "conf", ro && "ro");
      if (k) td.className = k;
      if (c < G.lefts.length - 1) td.style.left = G.lefts[c] + "px";
      const title = bad || (conf ? "Excel 側で変わっています（今の値: " + (col.secret ? "***" : r.theirs[ci] || "空") + "）。保存のときに、どちらを残すかを選べます" : ro ? "式の入ったセルです（この表では直せません）" : "");
      if (title) td.title = title;
      td.dataset.r = vr;
      td.dataset.c = c;
      td.textContent = gShow(col, v);
      tr.appendChild(td);
    }
    return tr;
  }

  function gRenderBody() {
    const body = $("g-body");
    if (!body) return;
    body.textContent = "";
    const frag = document.createDocumentFragment();
    for (let vr = 0; vr < G.view.length; vr++) frag.appendChild(gRowEl(vr));
    body.appendChild(frag);
    G.tds = [...body.rows].map((tr) => [...tr.cells].slice(1));
    G.painted = [];
    gPaintSel();
    gStatus();
  }

  // gRedraw draws some rows again (by their place in the view).
  function gRedraw(list) {
    const body = $("g-body");
    if (!body) return;
    for (const vr of new Set(list)) {
      if (vr < 0 || vr >= G.view.length || !body.rows[vr]) continue;
      const tr = gRowEl(vr);
      body.replaceChild(tr, body.rows[vr]);
      G.tds[vr] = [...tr.cells].slice(1);
    }
    G.painted = [];
    gPaintSel();
    gStatus();
  }

  function gSay(text) {
    G.say = text;
    gStatus();
  }

  function gStatus() {
    const el = $("g-status");
    if (!el || !G) return;
    const c = gCounts();
    const parts = [`全 ${c.rows} 行`];
    if (c.cells) parts.push(`変更 ${c.cells} セル`);
    if (c.added) parts.push(`追加 ${c.added} 行`);
    if (c.deleted) parts.push(`削除 ${c.deleted} 行`);
    if (c.moved) parts.push(`移動 ${c.moved} 行`);
    if (c.cols) parts.push(`自由列の変更 ${c.cols} 列`);
    el.textContent = "";
    add(el, [parts.join("・")]);
    if (c.bad) add(el, ["・", h("span", { class: "g-bad" }, `型に合わない値 ${c.bad} 個（赤い枠。マウスを当てると理由が出ます）`)]);
    if (c.clash) add(el, ["・", h("span", { class: "g-warn" }, `Excel 側の変更とぶつかり ${c.clash} 件（保存のときに選べます）`)]);
    if (c.unseen) add(el, ["・", h("span", { class: "g-warn" }, `そのうち ${c.unseen} 個は、隠した列にあります `), button("列の表示", gColsDialog, { class: "link" })]);
    if (G.total > G.shown) add(el, ["・", `${G.total} 行のうち ${G.shown} 行を出しています `, button("さらに表示", () => { gCommitEdit(); G.shown += G_ROWS; gView(); gRenderBody(); }, { class: "link" })]);
    else if (G.hidden) add(el, ["・", `絞り込みで ${G.total} 行を出しています`]);
    if (gFiltered()) add(el, [" ", button("絞り込みを解く", gClearFilters, { class: "link", id: "g-nofilter" })]);
    if (gSorted()) add(el, ["・", `「${G.cols[G.sort.ci].title}」の${G.sort.dir > 0 ? "昇順" : "降順"}で並べています（見え方だけ。ブックの行の順は変わりません） `, button("並べ替えを解く", () => gSortBy(-1, 1), { class: "link", id: "g-nosort" })]);
    const off = G.live.length - G.vc.length;
    if (off) add(el, ["・", `${off} 列を隠しています `, button("列の表示", gColsDialog, { class: "link" })]);
    if (G.trouble) add(el, ["・", h("span", { class: "g-bad" }, G.trouble)]);
    else if (c.all) add(el, ["・", G.savedAt ? `下書きを自動保存 ${G.savedAt.toTimeString().slice(0, 8)}` : "下書きは自動で保存されます"]);
    else add(el, ["・保存していない変更はありません"]);
    if (G.say) add(el, ["・", h("b", null, G.say)]);
    const u = $("g-undo"), rd = $("g-redo");
    if (u) u.disabled = !G.undo.length;
    if (rd) rd.disabled = !G.redo.length;
    const del = $("g-del");
    if (del) del.textContent = gSelRows().length && gSelRows().every((r) => r.del) ? "削除を取り消す" : "行を削除";
  }

  // ---- selection ----

  function gRange() {
    const s = G.sel;
    return { r0: Math.min(s.ar, s.fr), r1: Math.max(s.ar, s.fr), c0: Math.min(s.ac, s.fc), c1: Math.max(s.ac, s.fc) };
  }

  function gClamp() {
    const R = Math.max(0, G.view.length - 1), C = Math.max(0, G.vc.length - 1);
    const s = G.sel;
    s.ar = Math.min(Math.max(0, s.ar), R);
    s.fr = Math.min(Math.max(0, s.fr), R);
    s.ac = Math.min(Math.max(0, s.ac), C);
    s.fc = Math.min(Math.max(0, s.fc), C);
  }

  // A cell's box in the pixels of the frame of the table (what left and top
  // of a thing laid over the table take).
  function gBox(el) {
    const w = G.wrap, wr = w.getBoundingClientRect(), r = el.getBoundingClientRect();
    const x = r.left - wr.left - w.clientLeft + w.scrollLeft;
    const y = r.top - wr.top - w.clientTop + w.scrollTop;
    return { left: x, top: y, right: x + r.width, bottom: y + r.height, width: r.width, height: r.height };
  }

  function gPaintSel(fillTo) {
    if (!G.wrap) return;
    for (const el of G.painted) el.classList.remove("in", "cur", "hl", "fillp");
    G.painted = [];
    const dd = $("g-dd"), fill = $("g-fill");
    if (!G.view.length || !G.tds.length || !G.vc.length) {
      if (dd) dd.hidden = true;
      if (fill) fill.hidden = true;
      return;
    }
    gClamp();
    const g = gRange();
    const mark = (el, k) => {
      if (!el) return;
      el.classList.add(k);
      G.painted.push(el);
    };
    for (let r = g.r0; r <= g.r1; r++) for (let c = g.c0; c <= g.c1; c++) mark(G.tds[r] && G.tds[r][c], "in");
    if (fillTo) {
      for (let r = fillTo.r0; r <= fillTo.r1; r++) for (let c = fillTo.c0; c <= fillTo.c1; c++) {
        const td = G.tds[r] && G.tds[r][c];
        if (td && !td.classList.contains("in")) mark(td, "fillp");
      }
    }
    const cur = G.tds[G.sel.fr] && G.tds[G.sel.fr][G.sel.fc];
    mark(cur, "cur");
    const head = G.wrap.querySelectorAll("thead th[data-c]");
    for (let c = g.c0; c <= g.c1; c++) mark(head[c], "hl");
    const body = $("g-body");
    for (let r = g.r0; r <= g.r1; r++) mark(body.rows[r] && body.rows[r].cells[0], "hl");
    const info = $("g-selinfo");
    if (info) {
      const nr = g.r1 - g.r0 + 1, nc = g.c1 - g.c0 + 1;
      info.textContent = nr * nc > 1 ? `選択: ${nr} 行 × ${nc} 列` : "";
    }
    // The fill handle sits inside the bottom right corner of the selection
    // (hanging over the edge, the one of the last row lay under the scroll bar).
    const corner = G.tds[g.r1] && G.tds[g.r1][g.c1];
    if (fill && corner && !G.edit) {
      const b = gBox(corner);
      fill.style.left = b.right - 9 + "px";
      fill.style.top = b.bottom - 9 + "px";
      fill.hidden = false;
    } else if (fill) fill.hidden = true;
    // The ▼ of a column with choices sits at the right end of the current cell.
    const col = gColAt(G.sel.fc);
    if (dd && cur && !G.edit && col.choices.length) {
      const b = gBox(cur);
      dd.style.left = b.right - 20 + "px";
      dd.style.top = b.top + 1 + "px";
      dd.style.height = Math.max(b.height - 2, 16) + "px";
      dd.hidden = false;
    } else if (dd) dd.hidden = true;
    const del = $("g-del");
    if (del) del.textContent = gSelRows().length && gSelRows().every((r) => r.del) ? "削除を取り消す" : "行を削除";
  }

  function gEnsureVisible(r, c) {
    const td = G.tds[r] && G.tds[r][c];
    if (!td) return;
    const w = G.wrap, b = gBox(td);
    const top = b.top - w.scrollTop, bottom = b.bottom - w.scrollTop, x0 = b.left - w.scrollLeft, x1 = b.right - w.scrollLeft;
    const head = w.querySelector("thead").getBoundingClientRect().height;
    const left = c < G.lefts.length - 1 ? G_RH : G.lefts[G.lefts.length - 1];
    if (top < head) w.scrollTop -= head - top;
    else if (bottom > w.clientHeight - 4) w.scrollTop += bottom - w.clientHeight + 4;
    if (x0 < left) w.scrollLeft -= left - x0;
    else if (x1 > w.clientWidth - 4) w.scrollLeft += x1 - w.clientWidth + 4;
  }

  function gSelect(ar, ac, fr, fc) {
    G.sel = { ar, ac, fr: fr == null ? ar : fr, fc: fc == null ? ac : fc };
    G.tabC = null;
    gClamp();
    gPaintSel();
    const g = gRange();
    // A whole row or column picked stays where it is.
    if (!(g.c0 === 0 && g.c1 === G.vc.length - 1 && G.vc.length > 1) && !(g.r0 === 0 && g.r1 === G.view.length - 1 && G.view.length > 1)) gEnsureVisible(G.sel.fr, G.sel.fc);
  }

  function gMove(dr, dc, extend) {
    const s = G.sel;
    if (extend) {
      s.fr += dr;
      s.fc += dc;
      gClamp();
      gPaintSel();
      gEnsureVisible(s.fr, s.fc);
    } else gSelect(s.fr + dr, s.fc + dc);
  }

  // Tab … Tab, then Enter: as in Excel, Enter goes down and back to the
  // column the Tabs started from.
  function gTab(back) {
    const c = G.tabC == null ? G.sel.fc : G.tabC;
    gMove(0, back ? -1 : 1);
    G.tabC = c;
  }

  function gEnterDown(up) {
    const c = G.tabC;
    if (c != null && !up) gSelect(G.sel.fr + 1, c);
    else gMove(up ? -1 : 1, 0);
  }

  function gSelRows() {
    if (!G.view.length) return [];
    const g = gRange(), out = [];
    for (let r = g.r0; r <= g.r1; r++) if (gRowAt(r)) out.push(gRowAt(r));
    return out;
  }

  // ---- edits (every change goes through gMutate: undo, redraw, draft) ----

  // gMutate runs fn, which changes the table with the calls of ops only, and
  // keeps what it did for undo. It tells whether anything changed.
  function gMutate(fn) {
    if (G.busy) return false;
    const tx = [];
    const ops = {
      // set puts a value into a cell (ci: its place among all the columns).
      // A cell with a formula and a row that is to be deleted are left alone.
      set(r, ci, v) {
        if (!r || ci < 0 || ci >= G.cols.length) return false;
        if (r.del || r.formula.has(ci)) {
          ops.skipped++;
          return false;
        }
        if (r.cells[ci] === v) return false;
        // A new row under a table that shows one device belongs to it.
        if (r.row === 0 && gBlank(r) && G.dev && G.v.device >= 0 && ci !== G.v.device && v !== "") ops.set(r, G.v.device, G.dev);
        tx.push({ t: "c", r, ci, a: r.cells[ci], b: v });
        r.cells[ci] = v;
        return true;
      },
      del(r, on) {
        if (r.del === on) return;
        tx.push({ t: "d", r, a: r.del, b: on });
        r.del = on;
      },
      insert(at, r) {
        tx.push({ t: "i", r, at });
        G.rows.splice(at, 0, r);
      },
      remove(r) {
        const at = G.rows.indexOf(r);
        if (at < 0) return;
        tx.push({ t: "x", r, at });
        G.rows.splice(at, 1);
      },
      // col changes what a free column is: its heading, whether it is there.
      col(col, next) {
        const a = {};
        for (const k in next) a[k] = col[k];
        tx.push({ t: "col", col, a, b: next });
        Object.assign(col, next);
      },
      // parts notes a change made to the draft of other sheets (a
      // replacement in every sheet): undo puts their parts back.
      parts(before, after) {
        tx.push({ t: "all", before, after });
      },
      skipped: 0,
    };
    fn(ops);
    if (ops.skipped) G.say = `${ops.skipped} 個のセルは変えていません（式の入ったセルと、消す行のセル）`;
    else G.say = "";
    if (!tx.length) {
      gStatus();
      return false;
    }
    G.undo.push(tx);
    while (G.undo.length > G_UNDO) G.undo.shift();
    G.redo = [];
    gAfter(tx);
    return true;
  }

  // gAfter draws what a change touched, checks the new values and keeps the
  // draft.
  function gAfter(tx) {
    const structural = tx.some((o) => o.t === "i" || o.t === "x");
    const before = G.rows.length;
    if (tx.some((o) => o.t === "col")) {
      gKeepSel(gLayout);
      gPad();
      gView();
      render();
    } else {
      gPad();
      if (structural || G.rows.length !== before) {
        gView();
        gRenderBody();
      } else {
        const at = new Map(G.view.map((ri, vr) => [G.rows[ri].uid, vr]));
        gRedraw(tx.map((o) => (o.r && at.has(o.r.uid) ? at.get(o.r.uid) : -1)));
      }
    }
    gCheck(tx.filter((o) => o.t === "c"));
    gScheduleSave();
  }

  function gUndo(redo) {
    if (G.busy || G.edit) return;
    const from = redo ? G.redo : G.undo, to = redo ? G.undo : G.redo;
    if (!from.length) return;
    const tx = from.pop();
    const list = redo ? tx : tx.slice().reverse();
    for (const o of list) {
      if (o.t === "c") o.r.cells[o.ci] = redo ? o.b : o.a;
      else if (o.t === "d") o.r.del = redo ? o.b : o.a;
      else if (o.t === "col") Object.assign(o.col, redo ? o.b : o.a);
      else if (o.t === "all") gPutParts(redo ? o.after : o.before);
      else if ((o.t === "i") === redo) G.rows.splice(Math.min(o.at, G.rows.length), 0, o.r);
      else G.rows.splice(G.rows.indexOf(o.r), 1);
    }
    to.push(tx);
    G.say = "";
    gAfter(tx);
  }

  // gPutParts puts parts of the draft of other sheets in place (the undo of
  // a replacement in every sheet).
  async function gPutParts(parts) {
    const g = G;
    try {
      for (const p of parts) await api().EditDraft(p);
      st.edit.list = await api().EditSheets();
      if (st.home) st.home.draft = st.edit.list.draft;
      if (g === G) gSheetLabels();
    } catch (e) {
      g.trouble = "ほかのシートの下書きを戻せません: " + errText(e);
      if (g === G) gStatus();
    }
  }

  // gCheck asks what is wrong with the values that were just put in (the
  // check of the type that reading the book does), by column.
  function gCheck(list) {
    const g = G;
    const byCol = new Map();
    for (const o of list) {
      const c = g.cols[o.ci];
      if (!c.type || c.free) continue;
      if (!byCol.has(o.ci)) byCol.set(o.ci, []);
      byCol.get(o.ci).push(o.r);
    }
    for (const [ci, rows] of byCol) {
      const values = rows.map((r) => r.cells[ci]);
      api().EditCheck(g.id, g.cols[ci].key, values).then((said) => {
        if (g !== G) return;
        const touched = [];
        const at = new Map(G.view.map((ri, vr) => [G.rows[ri].uid, vr]));
        rows.forEach((r, i) => {
          if (r.cells[ci] !== values[i]) return; // changed again since
          const had = r.bad[ci] || "";
          if (said[i]) r.bad[ci] = said[i];
          else delete r.bad[ci];
          if (had !== (said[i] || "") && at.has(r.uid)) touched.push(at.get(r.uid));
        });
        if (touched.length) gRedraw(touched);
      }).catch((e) => {
        g.trouble = "値を確かめられません: " + errText(e);
        if (g === G) gStatus();
      });
    }
  }

  // gNorm is what typing or pasting puts into a cell: the text as it is; a
  // line break stays only where the column takes several lines or values.
  function gNorm(col, text) {
    text = String(text == null ? "" : text).replace(/\r\n?/g, "\n");
    if (col.secret || col.type === "text" || col.list || col.free) return text;
    return text.replace(/\n/g, " ");
  }

  // gSetCells sets [viewRow, viewColumn, text] cells.
  function gSetCells(list) {
    return gMutate((ops) => {
      for (const [vr, c, text] of list) ops.set(gRowAt(vr), G.vc[c], gNorm(gColAt(c), text));
    });
  }

  // ---- the cell editor ----

  function gBeginEdit(initial) {
    if (G.edit || !G.view.length || G.busy) return;
    const vr = G.sel.fr, c = G.sel.fc, ci = G.vc[c];
    const col = G.cols[ci], r = gRowAt(vr), td = G.tds[vr] && G.tds[vr][c];
    if (!col || !r || !td) return;
    if (r.del) {
      gSay("消す行です。直すには、先に「削除を取り消す」を押してください");
      return;
    }
    if (r.formula.has(ci)) {
      gSay("式の入ったセルは、この表では直せません（値で上書きすると式が消えるため。Excel で直してください）");
      return;
    }
    gCloseList();
    gEnsureVisible(vr, c);
    const ta = h("textarea", { class: cls("g-editor", col.secret && !G.show && "g-masked"), spellcheck: "false", rows: "1", wrap: "off" });
    ta.value = initial != null ? initial : r.cells[ci];
    const b = gBox(td);
    const lines = ta.value.split("\n").length;
    ta.style.left = b.left + "px";
    ta.style.top = b.top + "px";
    ta.style.minWidth = Math.max(b.width, 140) + "px";
    ta.style.height = Math.max(b.height, lines > 1 ? Math.min(lines, 8) * 20 + 8 : 0) + "px";
    G.wrap.appendChild(ta);
    G.edit = { ta, uid: r.uid, c, mode: initial != null ? "enter" : "edit" };
    $("g-dd").hidden = true;
    $("g-fill").hidden = true;
    ta.focus();
    if (initial == null) ta.setSelectionRange(ta.value.length, ta.value.length);
    ta.addEventListener("keydown", gEditKey);
    // Focus leaving the editor commits it, as in Excel. The window losing
    // the focus is not that: the editor stays, and typing goes on later.
    ta.addEventListener("blur", (ev) => {
      if (!(G && G.edit && G.edit.ta === ta)) return;
      if (!ev.relatedTarget && !document.hasFocus()) return;
      gCommitEdit(null, false, ev.relatedTarget);
    });
  }

  function gCommitEdit(move, all, focusTo) {
    const e = G && G.edit;
    if (!e) return;
    G.edit = null;
    const text = e.ta.value;
    e.ta.remove();
    const vr = G.view.findIndex((ri) => G.rows[ri].uid === e.uid);
    if (all) {
      // Ctrl+Enter: every selected cell takes the value; the empty rows
      // under the table stay empty.
      const g = gRange(), list = [], many = g.r1 > g.r0;
      for (let r = g.r0; r <= g.r1; r++) {
        const row = gRowAt(r);
        if (!row || (many && row.uid !== e.uid && gBlank(row))) continue;
        for (let c = g.c0; c <= g.c1; c++) list.push([r, c, text]);
      }
      gSetCells(list);
    } else if (vr >= 0) gSetCells([[vr, e.c, text]]);
    if (move) gMove(move[0], move[1]);
    else gPaintSel();
    const typing = focusTo && /^(INPUT|TEXTAREA|SELECT)$/.test(focusTo.tagName || "");
    if (!typing && G.wrap) G.wrap.focus({ preventScroll: true });
  }

  function gCancelEdit() {
    const e = G.edit;
    if (!e) return;
    G.edit = null;
    e.ta.remove();
    gPaintSel();
    G.wrap.focus({ preventScroll: true });
  }

  function gEditKey(ev) {
    ev.stopPropagation();
    // Keys while the IME is converting are the IME's, not the table's.
    if (ev.isComposing || ev.keyCode === 229) return;
    const e = G.edit;
    // As in Excel: started by typing, an arrow key enters the value and
    // moves (F2 switches to moving within the text, and back).
    if (ev.key === "F2") {
      ev.preventDefault();
      e.mode = e.mode === "enter" ? "edit" : "enter";
      return;
    }
    if (e.mode === "enter" && G_ARROWS[ev.key] && !ev.shiftKey && !ev.ctrlKey && !ev.altKey) {
      ev.preventDefault();
      gCommitEdit(G_ARROWS[ev.key]);
      return;
    }
    if (ev.key === "Enter" && ev.altKey) {
      // A line break inside the cell, as in Excel.
      ev.preventDefault();
      const ta = e.ta, at = ta.selectionStart;
      ta.value = ta.value.slice(0, at) + "\n" + ta.value.slice(ta.selectionEnd);
      ta.setSelectionRange(at + 1, at + 1);
      ta.style.height = Math.min(ta.value.split("\n").length, 8) * 20 + 8 + "px";
      return;
    }
    if (ev.key === "Enter" && (ev.ctrlKey || ev.metaKey)) {
      ev.preventDefault();
      gCommitEdit(null, true);
      return;
    }
    if (ev.key === "Enter") {
      ev.preventDefault();
      gCommitEdit(null);
      gEnterDown(ev.shiftKey);
      return;
    }
    if (ev.key === "Tab") {
      ev.preventDefault();
      const c = G.tabC == null ? G.sel.fc : G.tabC;
      gCommitEdit(ev.shiftKey ? [0, -1] : [0, 1]);
      G.tabC = c;
      return;
    }
    if (ev.key === "Escape") {
      ev.preventDefault();
      gCancelEdit();
      return;
    }
    if ((ev.ctrlKey || ev.metaKey) && ev.key.toLowerCase() === "s") {
      ev.preventDefault();
      gSaveGo();
    }
  }

  // ---- the list of a column with choices ----

  function gOpenList() {
    const col = gColAt(G.sel.fc), cur = G.tds[G.sel.fr] && G.tds[G.sel.fr][G.sel.fc], el = $("g-list");
    if (!col || !col.choices.length || !cur || !el || G.edit) return;
    const r = gRowAt(G.sel.fr);
    const now = r ? r.cells[G.vc[G.sel.fc]] : "";
    el.textContent = "";
    // The choices of the value in the column that narrows them (the models
    // of the new vendor, spec 14.18.2); all when it is empty or has none.
    let choices = col.choices;
    const byCi = col.choiceBy ? G.cols.findIndex((x) => x.key === col.choiceBy) : -1;
    const by = byCi >= 0 && r ? (r.cells[byCi] || "").trim() : "";
    if (by && col.choiceFor) {
      const some = choices.filter((v) => col.choiceFor[v] === by);
      if (some.length) choices = some;
    }
    // The new interface names of the row's new device (spec 14.19.6), each
    // told whether another old interface of the device has it already.
    const used = {};
    if (col.choiceSets) {
      if (by && col.choiceSets[by]) choices = col.choiceSets[by];
      const oi = G.cols.findIndex((x) => x.key === "old_if");
      for (let i = 0; i < G.rows.length; i++) {
        const o = G.rows[i];
        if (!o || o === r || !o.cells || (o.cells[byCi] || "").trim() !== by) continue;
        const v = (o.cells[G.vc[G.sel.fc]] || "").trim();
        if (v && v !== "-") used[v] = (used[v] ? used[v] + "・" : "") + (oi >= 0 ? o.cells[oi] : "");
      }
    }
    const opts = [""].concat(choices);
    opts.forEach((v, i) => {
      const note = v && (used[v] ? "割り当て済み（旧 " + used[v] + "）" : col.choiceNotes && col.choiceNotes[v]);
      const o = h("div", { class: cls("g-opt", v === now && "on"), "data-i": String(i) }, v === "" ? "（空にする）" : v,
        note ? h("span", { class: "dim small" }, "　" + note) : null);
      el.appendChild(o);
    });
    const b = gBox(cur);
    el.style.left = b.left + "px";
    el.style.top = b.bottom + "px";
    el.style.minWidth = Math.max(b.width, 120) + "px";
    el.hidden = false;
    G.list = { opts, at: Math.max(0, opts.indexOf(now)) };
    gListHL();
    el.focus({ preventScroll: true });
  }

  function gListHL() {
    const el = $("g-list");
    [...el.children].forEach((o, i) => o.classList.toggle("hl", i === G.list.at));
    const o = el.children[G.list.at];
    if (o) o.scrollIntoView({ block: "nearest" });
  }

  function gCloseList() {
    if (!G || !G.list) return;
    G.list = null;
    const el = $("g-list");
    if (el) el.hidden = true;
  }

  // gPickList puts the choice into every selected cell (the empty rows under
  // the table stay empty when several rows are picked).
  function gPickList(i) {
    const v = G.list.opts[i];
    gCloseList();
    const g = gRange(), list = [], many = g.r1 > g.r0;
    for (let r = g.r0; r <= g.r1; r++) {
      if (many && gBlank(gRowAt(r))) continue;
      for (let c = g.c0; c <= g.c1; c++) list.push([r, c, v]);
    }
    // A model picked in a row without a vendor fills the vendor too (spec
    // 14.18.2).
    const more = [];
    for (const [r, c] of list) {
      const col = gColAt(c);
      const byCi = col && col.choiceBy ? G.cols.findIndex((x) => x.key === col.choiceBy) : -1;
      const row = gRowAt(r);
      const to = v && col && col.choiceFor ? col.choiceFor[v] : "";
      if (byCi >= 0 && to && row && !(row.cells[byCi] || "").trim()) more.push([row, byCi, to]);
    }
    gMutate((ops) => {
      for (const [vr, c, text] of list) ops.set(gRowAt(vr), G.vc[c], gNorm(gColAt(c), text));
      for (const [row, ci, text] of more) ops.set(row, ci, text);
    });
    G.wrap.focus({ preventScroll: true });
  }

  function gListKey(e) {
    if (!G.list) return;
    e.stopPropagation();
    const n = G.list.opts.length;
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      G.list.at = (G.list.at + (e.key === "ArrowDown" ? 1 : n - 1)) % n;
      gListHL();
    } else if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      gPickList(G.list.at);
    } else if (e.key === "Escape" || e.key === "Tab") {
      e.preventDefault();
      gCloseList();
      G.wrap.focus({ preventScroll: true });
    }
  }

  // ---- the clipboard (tab-separated, as Excel copies and pastes) ----

  // A value that starts like a formula would run as one when pasted into
  // Excel: it goes out with an apostrophe before it, and pasting back takes
  // exactly one away.
  const gCellOut = (v) => (v && "=+-@\t\r'".includes(v[0]) ? "'" + v : v);
  const gCellIn = (v) => (v && v.length > 1 && v[0] === "'" && "=+-@\t\r'".includes(v[1]) ? v.slice(1) : v);

  function gTSV(g) {
    const lines = [];
    for (let r = g.r0; r <= g.r1; r++) {
      const row = gRowAt(r), vals = [];
      for (let c = g.c0; c <= g.c1; c++) {
        const col = gColAt(c);
        // A secret value goes out only while the secrets are shown.
        const v = gCellOut(col.secret && !G.show ? "" : row.cells[G.vc[c]]);
        vals.push(/[\t\r\n"]/.test(v) ? '"' + v.replace(/"/g, '""') + '"' : v);
      }
      lines.push(vals.join("\t"));
    }
    return lines.join("\r\n");
  }

  // gQuoteEnd: the index of the quote that closes a cell starting at i, or
  // -1 when it is not a quoted cell.
  function gQuoteEnd(text, i) {
    for (let j = i + 1; j < text.length; j++) {
      if (text[j] !== '"') continue;
      if (text[j + 1] === '"') {
        j++;
        continue;
      }
      return j + 1 >= text.length || "\t\r\n".includes(text[j + 1]) ? j : -1;
    }
    return -1;
  }

  function gParseTSV(text) {
    const rows = [];
    let row = [], cur = "", q = false;
    for (let i = 0; i < text.length; i++) {
      const ch = text[i];
      if (q) {
        if (ch === '"') {
          if (text[i + 1] === '"') {
            cur += '"';
            i++;
          } else q = false;
        } else cur += ch;
        continue;
      }
      if (ch === '"' && cur === "" && gQuoteEnd(text, i) >= 0) {
        q = true;
        continue;
      }
      if (ch === "\t") {
        row.push(cur);
        cur = "";
        continue;
      }
      if (ch === "\r" || ch === "\n") {
        if (ch === "\r" && text[i + 1] === "\n") i++;
        row.push(cur);
        rows.push(row);
        row = [];
        cur = "";
        continue;
      }
      cur += ch;
    }
    if (cur !== "" || row.length) {
      row.push(cur);
      rows.push(row);
    }
    return rows;
  }

  const gClipText = (text) => (text.endsWith("\r\n") ? text.slice(0, -2) : text.endsWith("\n") ? text.slice(0, -1) : text);

  async function gCopy(cut) {
    if (!G.view.length) return;
    const g = gRange();
    const text = gTSV(g);
    try {
      await api().ClipboardSet(text);
    } catch (e) {
      gSay("コピーできませんでした: " + errText(e));
      return;
    }
    // raw: the cells as they are (the secrets too, whatever is shown), so
    // that a paste of this table's own copy writes the real values.
    const raw = [];
    for (let r = g.r0; r <= g.r1; r++) {
      const row = gRowAt(r), vals = [];
      for (let c = g.c0; c <= g.c1; c++) vals.push(row.cells[G.vc[c]]);
      raw.push(vals);
    }
    G.clip = { text, raw };
    const n = (g.r1 - g.r0 + 1) * (g.c1 - g.c0 + 1);
    const masked = !G.show && G.vc.slice(g.c0, g.c1 + 1).some((i) => G.cols[i].secret);
    if (cut) {
      const list = [];
      for (let r = g.r0; r <= g.r1; r++) for (let c = g.c0; c <= g.c1; c++) list.push([r, c, ""]);
      gSetCells(list);
    }
    gSay(`${n} セルを${cut ? "切り取りました" : "コピーしました"}${masked ? "（秘密の値は、見せている間だけ Excel へコピーされます）" : ""}`);
  }

  async function gPaste() {
    let text = "";
    try {
      text = await api().ClipboardGet();
    } catch (e) {
      text = "";
    }
    const own = G.clip && gClipText(G.clip.text) === gClipText(text || "");
    if (!text && !own) {
      gSay("クリップボードが空です");
      return;
    }
    gPasteText(text || "");
  }

  function gPasteText(text) {
    text = gClipText(text);
    const own = G.clip && gClipText(G.clip.text) === text ? G.clip : null;
    const data = own ? own.raw : gParseTSV(text).map((r) => r.map(gCellIn));
    const g = G.view.length ? gRange() : { r0: 0, r1: 0, c0: 0, c1: 0 };
    if (!data.length) {
      // One empty cell copied in Excel is a line end alone: it empties the selection.
      if (!own && text.trim() === "") gClear();
      return;
    }
    const C = G.vc.length;
    const hh = data.length, w = Math.max(...data.map((r) => r.length));
    const over = g.c0 + w - C;
    const sr = g.r1 - g.r0 + 1, sc = g.c1 - g.c0 + 1;
    let endR = g.r0 + hh - 1, endC = Math.min(g.c0 + w - 1, C - 1);
    let kept = 0;
    gMutate((ops) => {
      const list = [];
      const many = sr > 1;
      const skip = (r) => many && gBlank(gRowAt(r));
      if (hh === 1 && w === 1 && (sr > 1 || sc > 1)) {
        // One value into a selection: every selected cell gets it.
        for (let r = g.r0; r <= g.r1; r++) {
          if (skip(r)) continue;
          for (let c = g.c0; c <= g.c1; c++) list.push([r, c, data[0][0]]);
        }
        endR = g.r1;
        endC = g.c1;
      } else if ((sr > hh || sc > w) && sr % hh === 0 && sc % w === 0) {
        // A block into a selection it fits a whole number of times: tiled.
        for (let r = 0; r < sr; r++) {
          if (skip(g.r0 + r)) continue;
          for (let c = 0; c < sc; c++) list.push([g.r0 + r, g.c0 + c, data[r % hh][c % w] == null ? "" : data[r % hh][c % w]]);
        }
        endR = g.r1;
        endC = g.c1;
      } else {
        // Rows past the end become new rows, as in Excel.
        for (let i = 0; i < hh; i++) {
          const r = g.r0 + i;
          if (r >= G.view.length) {
            const nr = gNewRow();
            ops.insert(G.rows.length, nr);
            G.view.push(G.rows.length - 1);
          }
          for (let j = 0; j < w; j++) if (g.c0 + j < C) list.push([r, g.c0 + j, data[i][j] == null ? "" : data[i][j]]);
        }
      }
      for (const [r, c, v] of list) {
        const row = gRowAt(r), ci = G.vc[c], col = G.cols[ci];
        // A secret column reads empty while the secrets are masked: an
        // empty value pasted over a secret leaves it.
        if (col.secret && !G.show && !own && v === "" && row && row.cells[ci]) {
          kept++;
          continue;
        }
        ops.set(row, ci, gNorm(col, v));
      }
    });
    const said = [];
    if (over > 0) said.push(`${over} 列は右端を超えるため貼り付けませんでした`);
    if (kept) said.push(`秘密の値 ${kept} 個は伏せているため残しました（消すには Delete か「秘密の値を見せる」）`);
    if (said.length) G.say = [G.say].concat(said).filter(Boolean).join("。");
    G.sel = { ar: g.r0, ac: g.c0, fr: Math.min(endR, G.view.length - 1), fc: endC };
    gPaintSel();
    gEnsureVisible(G.sel.fr, G.sel.fc);
    gStatus();
  }

  function gClear() {
    if (!G.view.length) return;
    const g = gRange(), list = [];
    for (let r = g.r0; r <= g.r1; r++) for (let c = g.c0; c <= g.c1; c++) list.push([r, c, ""]);
    gSetCells(list);
  }

  // gXfer is a value copied from one column into another: a secret is never
  // written into a column that shows it in clear.
  const gXfer = (src, dst, v) => (src && src.secret && dst && !dst.secret ? "" : v);

  // Ctrl+D / Ctrl+R: the top row (left column) of the selection copied over
  // the rest of it; with one row (column) selected, the one above (left).
  function gFillDown(right) {
    if (!G.view.length) return;
    const g = gRange(), list = [];
    const many = g.r1 > g.r0;
    const skip = (r) => many && gBlank(gRowAt(r));
    if (!right) {
      const src = g.r0 === g.r1 ? g.r0 - 1 : g.r0;
      if (src < 0) return;
      for (let r = g.r0 === g.r1 ? g.r0 : g.r0 + 1; r <= g.r1; r++) {
        if (skip(r)) continue;
        for (let c = g.c0; c <= g.c1; c++) list.push([r, c, gRowAt(src).cells[G.vc[c]]]);
      }
    } else {
      const src = g.c0 === g.c1 ? g.c0 - 1 : g.c0;
      if (src < 0) return;
      for (let c = g.c0 === g.c1 ? g.c0 : g.c0 + 1; c <= g.c1; c++) {
        for (let r = g.r0; r <= g.r1; r++) {
          if (!skip(r)) list.push([r, c, gXfer(gColAt(src), gColAt(c), gRowAt(r).cells[G.vc[src]])]);
        }
      }
    }
    gSetCells(list);
  }

  // ---- the fill handle (■): a series, or a copy ----

  // gTail splits a value at the number a series counts on: the last number
  // of the text, or the last part of an address that has a length after it
  // (192.0.2.1/24 goes on with 192.0.2.2/24, not with /25).
  function gTail(v) {
    v = v || "";
    let m = /^((?:\d{1,3}\.){3})(\d{1,3})(\/\d{1,2})?$/.exec(v);
    if (m) return { head: m[1], num: m[2], tail: m[3] || "", max: 255, hex: false };
    m = /^(.*:)([0-9a-fA-F]{1,4})(\/\d{1,3})?$/.exec(v);
    if (m && (v.match(/:/g) || []).length >= 2) return { head: m[1], num: m[2], tail: m[3] || "", max: 0xffff, hex: true };
    m = /^(.*?)(\d{1,15})$/.exec(v);
    return m ? { head: m[1], num: m[2], tail: "", max: Number.MAX_SAFE_INTEGER, hex: false } : null;
  }

  // gSeries gives the n values that follow vals: text that ends in a number
  // counts on (rt-a01 → rt-a02, 192.0.2.10 → 192.0.2.11), a number alone is
  // copied as in Excel (Ctrl swaps the two), several values with one step
  // between them go on with that step; anything else is copied.
  function gSeries(vals, n, col, ctrl) {
    const copy = () => Array.from({ length: n }, (_, i) => vals[i % vals.length]);
    if (col.secret || col.choices.length || col.type === "bool") return copy();
    const ps = vals.map(gTail);
    if (ps.some((p) => !p) || ps.some((p) => p.head !== ps[0].head || p.tail !== ps[0].tail || p.hex !== ps[0].hex)) return copy();
    const radix = ps[0].hex ? 16 : 10;
    const nums = ps.map((p) => parseInt(p.num, radix));
    let step = 1;
    if (vals.length === 1) {
      const pure = ps[0].head === "" && ps[0].tail === "";
      if (pure !== !!ctrl) return copy();
    } else {
      step = nums[1] - nums[0];
      if (nums.some((x, i) => i && x - nums[i - 1] !== step) || ctrl) return copy();
    }
    const last = ps[ps.length - 1].num;
    const width = last.length > 1 && last.startsWith("0") ? last.length : 0;
    const upper = /[A-F]/.test(last);
    const out = [];
    for (let i = 1; i <= n; i++) {
      const x = nums[nums.length - 1] + step * i;
      if (x < 0 || x > ps[0].max) return copy();
      let s = x.toString(radix).padStart(width, "0");
      if (upper) s = s.toUpperCase();
      out.push(ps[0].head + s + ps[0].tail);
    }
    return out;
  }

  function gFillTarget(g, r, c) {
    if (r > g.r1) return { dir: "down", t: { r0: g.r1 + 1, r1: r, c0: g.c0, c1: g.c1 } };
    if (r < g.r0) return { dir: "up", t: { r0: r, r1: g.r0 - 1, c0: g.c0, c1: g.c1 } };
    if (c > g.c1) return { dir: "right", t: { r0: g.r0, r1: g.r1, c0: g.c1 + 1, c1: c } };
    if (c < g.c0) return { dir: "left", t: { r0: g.r0, r1: g.r1, c0: c, c1: g.c0 - 1 } };
    return null;
  }

  // gFill fills the cells of ft.t from the selection g.
  function gFill(g, ft, ctrl) {
    const more = ft.t.r1 - (G.view.length - 1);
    const list = [];
    if (ft.dir === "down" || ft.dir === "up") {
      const n = ft.t.r1 - ft.t.r0 + 1;
      for (let c = g.c0; c <= g.c1; c++) {
        const src = [];
        for (let r = g.r0; r <= g.r1; r++) src.push(gRowAt(r).cells[G.vc[c]]);
        if (ft.dir === "up") src.reverse();
        gSeries(src, n, gColAt(c), ctrl).forEach((v, i) => list.push([ft.dir === "down" ? ft.t.r0 + i : ft.t.r1 - i, c, v]));
      }
    } else {
      // Across columns each value lands in a column of another kind:
      // counting on makes no sense there, so it is a copy.
      const n = ft.t.c1 - ft.t.c0 + 1;
      for (let r = g.r0; r <= g.r1; r++) {
        const src = [];
        for (let c = g.c0; c <= g.c1; c++) src.push([gColAt(c), gRowAt(r).cells[G.vc[c]]]);
        if (ft.dir === "left") src.reverse();
        for (let i = 0; i < n; i++) {
          const c = ft.dir === "right" ? ft.t.c0 + i : ft.t.c1 - i;
          const [sc, v] = src[i % src.length];
          list.push([r, c, gXfer(sc, gColAt(c), v)]);
        }
      }
    }
    // One step to undo, the rows a fill past the end added included.
    gMutate((ops) => {
      for (let i = 0; i < more; i++) {
        ops.insert(G.rows.length, gNewRow());
        G.view.push(G.rows.length - 1);
      }
      for (const [r, c, text] of list) {
        const row = gRowAt(r);
        if (row) ops.set(row, G.vc[c], gNorm(gColAt(c), text));
      }
    });
    G.sel = { ar: Math.min(g.r0, ft.t.r0), ac: Math.min(g.c0, ft.t.c0), fr: Math.max(g.r1, ft.t.r1), fc: Math.max(g.c1, ft.t.c1) };
    gPaintSel();
  }

  // A double click on the fill handle: down to the last row of the table.
  function gFillToEnd() {
    const g = gRange();
    let last = -1;
    G.view.forEach((ri, r) => { if (!gBlank(G.rows[ri])) last = r; });
    if (last <= g.r1) return;
    gFill(g, { dir: "down", t: { r0: g.r1 + 1, r1: last, c0: g.c0, c1: g.c1 } }, false);
  }

  function gFillHint(text) {
    const el = $("g-selinfo");
    if (el && text) el.textContent = text;
  }

  // ---- rows ----

  // gDeleteRows marks the selected rows of the book as to be deleted (they
  // stay in sight, struck through, until the save) and takes the new rows
  // away. With only rows that are marked selected, it takes the mark off.
  function gDeleteRows() {
    const rows = gSelRows().filter((r) => !gBlank(r));
    if (!rows.length) return;
    const g = gRange();
    const undo = rows.every((r) => r.del);
    gMutate((ops) => {
      for (const r of rows) {
        if (r.row === 0) ops.remove(r);
        else ops.del(r, !undo);
      }
    });
    gSelect(Math.min(g.r0, G.view.length - 1), g.c0, Math.min(g.r1, G.view.length - 1), g.c1);
    const n = rows.filter((r) => r.row !== 0).length;
    if (n) gSay(undo ? `${n} 行の削除を取り消しました` : `${n} 行を消す行にしました（保存するまで、パラメータブックには残っています。取り消すには、もう一度押すか Ctrl+Z）`);
  }

  // A row put in between shows where it was asked for only while the rows
  // stand in the order of the sheet.
  function gNotSorted() {
    if (!gSorted()) return true;
    gSay("並べ替えている間は、行を入れたり動かしたりできません（「並べ替えを解く」を押してから）");
    return false;
  }

  // gSelectRows selects the rows given (the whole width when wide).
  function gSelectRows(rows, c0, c1, wide) {
    const idx = rows.map((r) => G.view.findIndex((i) => G.rows[i] === r)).filter((i) => i >= 0);
    if (!idx.length) return;
    gSelect(Math.min(...idx), wide ? 0 : c0, Math.max(...idx), wide ? G.vc.length - 1 : c1);
  }

  // gInsertAt puts empty rows in above (or below) the selected rows, as many
  // as are selected. The save puts them into the sheet at that place.
  function gInsertAt(below) {
    if (G.busy || !G.view.length || !gNotSorted()) return;
    gCommitEdit();
    const g = gRange();
    const at = below ? G.rows.indexOf(gRowAt(g.r1)) + 1 : G.rows.indexOf(gRowAt(g.r0));
    const fresh = Array.from({ length: g.r1 - g.r0 + 1 }, gNewRow);
    gMutate((ops) => fresh.forEach((r, i) => ops.insert(at + i, r)));
    gSelectRows(fresh, g.c0, g.c1, true);
    gSay(`${fresh.length} 行を入れました（打った行だけが、保存のときにこの位置に入ります）`);
  }

  // gDuplicateRows puts copies of the selected rows in right under them, as
  // new rows.
  function gDuplicateRows() {
    const rows = gSelRows().filter((r) => !gBlank(r));
    if (!rows.length || !gNotSorted()) return;
    const at = G.rows.indexOf(rows[rows.length - 1]) + 1;
    const copies = rows.map((r) => {
      const n = gNewRow();
      // The value a formula shows is not copied: the copy would hold it as text.
      n.cells = r.cells.map((v, i) => (r.formula.has(i) || G.cols[i].dropped ? "" : v));
      return n;
    });
    gMutate((ops) => copies.forEach((n, i) => ops.insert(at + i, n)));
    gSelectRows(copies, 0, 0, true);
    gCheck(copies.flatMap((r) => r.cells.map((v, ci) => ({ r, ci })).filter((o) => o.r.cells[o.ci] !== "")));
    gSay(`${copies.length} 行を複製しました（すぐ下の新しい行。行を見分ける列を直してください）`);
  }

  // gMoveRows moves the selected rows one row up or down (past the row that
  // is seen above or below them). The save moves the rows of the book.
  function gMoveRows(dir) {
    if (G.busy || !G.view.length || !gNotSorted()) return;
    gCommitEdit();
    const g = gRange(), sel = [];
    for (let r = g.r0; r <= g.r1; r++) sel.push(gRowAt(r));
    if (sel.some((r) => gBlank(r) || r.gone)) {
      gSay("空の行と、Excel 側で消えた行は動かせません");
      return;
    }
    const nb = dir < 0 ? gRowAt(g.r0 - 1) : gRowAt(g.r1 + 1);
    if (!nb || gBlank(nb)) return;
    gMutate((ops) => {
      for (const r of sel) ops.remove(r);
      const at = G.rows.indexOf(nb) + (dir > 0 ? 1 : 0);
      sel.forEach((r, i) => ops.insert(at + i, r));
    });
    gSelectRows(sel, g.c0, g.c1, false);
  }

  // gToEnd moves the cursor to the first empty row under the table.
  function gToEnd() {
    let vr = -1;
    for (let r = G.view.length - 1; r >= 0 && gBlank(gRowAt(r)); r--) vr = r;
    if (vr >= 0) gSelect(vr, 0);
    G.wrap.focus({ preventScroll: true });
  }

  // ---- sort and filters ----

  function gSortBy(ci, dir) {
    gCommitEdit();
    G.sort = { ci, dir };
    gResort();
    gRedrawAll();
  }

  function gClearFilters() {
    gCommitEdit();
    G.filter = "";
    G.dev = "";
    G.colFilter = {};
    G.shown = G_ROWS;
    gRedrawAll();
  }

  // gFilterByCell leaves the rows that have the value of the current cell in
  // its column.
  function gFilterByCell() {
    const r = gRowAt(G.sel.fr), ci = G.vc[G.sel.fc];
    if (!r || ci == null) return;
    G.colFilter[ci] = new Set([gLabel(G.cols[ci], r.cells[ci])]);
    G.shown = G_ROWS;
    gRedrawAll();
  }

  // ---- menus (the right click, the ▾ of a heading) ----

  function gCloseMenus() {
    document.querySelectorAll(".g-menu, .g-hpop").forEach((x) => x.remove());
  }

  function gPlace(el, x, y) {
    document.body.appendChild(el);
    el.style.left = Math.max(4, Math.min(x, window.innerWidth - el.offsetWidth - 8)) + "px";
    el.style.top = Math.max(4, Math.min(y, window.innerHeight - el.offsetHeight - 8)) + "px";
  }

  const gFocusBack = () => { if (G && G.wrap && G.wrap.isConnected && !G.edit && !gDialogOpen()) G.wrap.focus({ preventScroll: true }); };

  // gMenu shows a menu at a place of the window. items: [label, fn], or null
  // for a line between them.
  function gMenu(items, x, y) {
    gCloseMenus();
    const el = h("div", { class: "g-menu", tabindex: "-1" }, items.map((it) => (it
      ? button(it[0], () => { gCloseMenus(); it[1](); gFocusBack(); }, { class: "g-mi" })
      : h("div", { class: "g-msep" }))));
    el.addEventListener("keydown", (e) => {
      if (e.key !== "Escape") return;
      e.preventDefault();
      e.stopPropagation();
      gCloseMenus();
      gFocusBack();
    });
    el.addEventListener("contextmenu", (e) => e.preventDefault());
    gPlace(el, x, y);
    el.focus({ preventScroll: true });
  }

  function gContextMenu(e) {
    e.preventDefault();
    e.stopPropagation();
    if (!G || G.busy) return;
    if (G.edit) gCommitEdit();
    gCloseList();
    const at = e.target.closest(".g-corner") ? null : gCellAt(e.clientX, e.clientY);
    if (at && at.r == null && at.c != null) {
      gHeaderMenu(G.vc[at.c], e.clientX, e.clientY);
      return;
    }
    const g = gRange();
    if (at && at.r != null && at.c != null && !(at.r >= g.r0 && at.r <= g.r1 && at.c >= g.c0 && at.c <= g.c1)) gSelect(at.r, at.c);
    else if (at && at.r != null && at.c == null && !(at.r >= g.r0 && at.r <= g.r1)) gSelect(at.r, 0, at.r, G.vc.length - 1);
    gCellMenu(e.clientX, e.clientY);
  }

  function gCellMenu(x, y) {
    const undo = gSelRows().length && gSelRows().every((r) => r.del);
    const items = [
      ["切り取り（Ctrl+X）", () => gCopy(true)],
      ["コピー（Ctrl+C）", () => gCopy(false)],
      ["貼り付け（Ctrl+V）", () => gPaste()],
      null,
      ["上に行を入れる（Ctrl+Shift++）", () => gInsertAt(false)],
      ["下に行を入れる", () => gInsertAt(true)],
      ["行を複製", gDuplicateRows],
      ["行を上へ動かす（Alt+↑）", () => gMoveRows(-1)],
      ["行を下へ動かす（Alt+↓）", () => gMoveRows(1)],
      [undo ? "行の削除を取り消す" : "行を削除（Ctrl+－）", gDeleteRows],
      null,
      ["セルを空にする（Delete）", gClear],
      ["上の値を下へ（Ctrl+D）", () => gFillDown(false)],
      ["左の値を右へ（Ctrl+R）", () => gFillDown(true)],
      null,
      ["このセルの値で絞り込む", gFilterByCell],
    ];
    if (gFiltered()) items.push(["絞り込みを解く", gClearFilters]);
    gMenu(items, x, y);
  }

  // gHeaderMenu is the menu of a column: sort, filter by value, hide; for a
  // free column also its heading and taking it out.
  function gHeaderMenu(ci, x, y) {
    gCloseMenus();
    gCommitEdit();
    const col = G.cols[ci];
    if (!col) return;
    const counts = new Map();
    for (const r of G.rows) {
      if (gBlank(r) || r.row === 0) continue;
      const l = gLabel(col, r.cells[ci]);
      counts.set(l, (counts.get(l) || 0) + 1);
    }
    const all = [...counts.keys()].sort(gSorter.compare);
    const vals = all.slice(0, G_VALUES);
    const cur = G.colFilter[ci];
    const item = (label, fn, id) => button(label, () => { gCloseMenus(); fn(); gFocusBack(); }, { class: "g-mi", "data-a": id });
    const boxes = vals.map((v) => {
      const b = h("input", { type: "checkbox", checked: !cur || cur.has(v) });
      b.dataset.v = v;
      return h("label", { class: "g-hv" }, b, " " + (v === "" ? "（空白）" : v) + " ", h("span", { class: "dim" }, String(counts.get(v))));
    });
    const checks = () => boxes.map((l) => l.firstChild);
    const allBox = h("input", { type: "checkbox", checked: !cur, onchange: (e) => { for (const l of boxes) if (l.style.display !== "none") l.firstChild.checked = e.target.checked; } });
    const search = h("input", { type: "text", class: "g-hq", placeholder: "値を探す", oninput: (e) => {
      const q = e.target.value.toLowerCase();
      for (const l of boxes) l.style.display = l.firstChild.dataset.v.toLowerCase().includes(q) ? "" : "none";
    } });
    const pop = h("div", { class: "g-hpop" },
      item("昇順で並べ替え", () => gSortBy(ci, 1), "asc"),
      item("降順で並べ替え", () => gSortBy(ci, -1), "desc"),
      gSorted() ? item("並べ替えを解く", () => gSortBy(-1, 1), "nosort") : null,
      h("div", { class: "g-msep" }),
      h("div", { class: "g-hlabel" }, "値で絞り込み"),
      search,
      h("label", { class: "g-hv" }, allBox, " （すべて選択）"),
      h("div", { class: "g-hlist" }, boxes),
      all.length > vals.length ? h("div", { class: "dim small" }, `値が ${all.length} 種類あります。はじめの ${vals.length} 種類を出しています（上の絞り込みの欄も使えます）`) : null,
      h("div", { class: "row" },
        button("絞り込む", () => {
          const on = checks().filter((b) => b.checked).map((b) => b.dataset.v);
          if (on.length === vals.length) delete G.colFilter[ci];
          else G.colFilter[ci] = new Set(on);
          G.shown = G_ROWS;
          gRedrawAll();
        }, { class: "primary", "data-a": "ok" }),
        button("この列の絞り込みを解く", () => { delete G.colFilter[ci]; G.shown = G_ROWS; gRedrawAll(); }, { "data-a": "clear" })),
      h("div", { class: "g-msep" }),
      col.isKey ? null : item("この列を隠す", () => gHideCols([ci]), "hide"),
      col.free ? item("列の名前を変える…", () => gRenameCol(ci), "rename") : null,
      col.free ? item("この列を消す…", () => gDropCol(ci), "drop") : null);
    pop.addEventListener("keydown", (e) => {
      e.stopPropagation();
      if (e.isComposing || e.keyCode === 229 || e.key !== "Escape") return;
      e.preventDefault();
      gCloseMenus();
      gFocusBack();
    });
    pop.addEventListener("contextmenu", (e) => e.preventDefault());
    gPlace(pop, x, y);
    search.focus();
  }

  // ---- columns: shown or hidden, their widths, the free ones ----

  function gHideCols(list) {
    for (const ci of list) {
      const col = G.cols[ci];
      if (col.isKey) continue;
      G.pref.hidden.add(col.key);
      // Hiding the column the rows are sorted by would leave a sort nobody sees.
      if (G.sort.ci === ci) {
        G.sort = { ci: -1, dir: 1 };
        G.rank = null;
      }
      delete G.colFilter[ci];
    }
    gSavePref();
    gRedrawAll();
  }

  // gEmptyCols lists the columns no row has a value in.
  function gEmptyCols() {
    const rows = G.rows.filter((r) => !gBlank(r));
    return G.live.filter((i) => !G.cols[i].isKey && rows.every((r) => r.cells[i] === ""));
  }

  function gColsDialog() {
    gCommitEdit();
    gCloseMenus();
    const boxes = G.live.map((ci) => {
      const col = G.cols[ci];
      const b = h("input", { type: "checkbox", checked: col.isKey || !G.pref.hidden.has(col.key), disabled: col.isKey });
      b.dataset.ci = ci;
      return h("label", { class: "g-hv", title: gColTip(col) }, b, " " + col.title + (col.free ? "（自由列）" : ""));
    });
    const apply = () => {
      for (const l of boxes) {
        const b = l.firstChild, col = G.cols[+b.dataset.ci];
        if (b.disabled) continue;
        if (b.checked) G.pref.hidden.delete(col.key);
        else G.pref.hidden.add(col.key);
      }
      if (gSorted() && G.pref.hidden.has(G.cols[G.sort.ci].key)) {
        G.sort = { ci: -1, dir: 1 };
        G.rank = null;
      }
      gSavePref();
      gRedrawAll();
    };
    for (const l of boxes) l.firstChild.addEventListener("change", apply);
    const body = h("div", { id: "g-colsdlg" },
      h("p", { class: "dim" }, "表に出す列を選びます。隠した列の値はそのまま残り、貼り付けや置換は隠した列に触りません。行を見分ける列は隠せません。選んだ内容と列幅は、この PC の設定に表ごとに覚えます（案件のファイルには書きません）。"),
      h("div", { class: "row" },
        button("空の列を隠す", () => {
          const empty = new Set(gEmptyCols());
          for (const l of boxes) if (empty.has(+l.firstChild.dataset.ci)) l.firstChild.checked = false;
          apply();
        }, { id: "gv-empty", title: "どの行にも値の無い列を、まとめて隠します" }),
        button("すべて出す", () => { for (const l of boxes) l.firstChild.checked = true; apply(); }, { id: "gv-all" }),
        button("列幅を元に戻す", () => { G.pref.widths = {}; gSavePref(); gRedrawAll(); }, { id: "gv-widths" })),
      h("div", { class: "g-colgrid" }, boxes));
    dialog("列の表示", body, [{ id: "ok", label: "閉じる", cls: "primary" }]).then(gFocusBack);
  }

  // gAutoFit makes a column as wide as its widest value among the rows drawn.
  function gAutoFit(c) {
    const col = gColAt(c);
    if (!col) return;
    const span = h("span", { style: "position:absolute;visibility:hidden;white-space:nowrap;font-size:13px" });
    document.body.appendChild(span);
    let w = 0;
    for (const text of [col.title + "　　"].concat(G.view.map((ri) => gShow(col, G.rows[ri].cells[G.vc[c]])))) {
      span.textContent = text;
      w = Math.max(w, span.offsetWidth);
    }
    span.remove();
    gSetWidth(c, Math.max(50, Math.min(600, w + 20)));
  }

  function gSetWidth(c, w) {
    G.pref.widths[gColAt(c).key] = Math.round(w);
    gSavePref();
    // The columns that stand still lie at places that follow from the widths.
    if (c < G.lefts.length - 1) {
      gLayout();
      render();
      return;
    }
    const el = G.wrap.querySelectorAll("#g-table col")[c + 1];
    if (el) el.style.width = Math.round(w) + "px";
    gPaintSel();
  }

  const gHeadings = (but) => G.cols.filter((c) => !c.dropped && c !== but).map((c) => c.title).concat(G.cols.filter((c) => c.was && !c.dropped && c !== but).map((c) => c.was));
  const gFilled = (ci) => G.rows.filter((r) => !gBlank(r) && !r.del && r.cells[ci] !== "").length;

  // gAddCol adds a free column: a heading the definition does not have. The
  // save writes the heading at the right end of the sheet.
  async function gAddCol() {
    if (G.busy) return;
    gCommitEdit();
    const title = await askText("列を足す", "足す列の見出しを書いてください（定義に無い列 = 自由列。メモや管理用の値を置けます。生成には使われません）。保存すると、シートの右端にできます。", "", "足す");
    if (title == null) return gFocusBack();
    let key = "";
    try {
      key = await api().EditColumn(G.id, title, gHeadings(null));
    } catch (e) {
      gSay("列を足せません: " + errText(e));
      return gFocusBack();
    }
    const col = { key, title: title.trim(), col: -1, type: "", list: false, required: false, isKey: false, secret: false, free: true, choices: [], note: "", added: true, was: "", dropped: true };
    G.cols.push(col);
    for (const r of G.rows) {
      r.cells.push("");
      r.base.push("");
    }
    gMutate((ops) => ops.col(col, { dropped: false }));
    const c = G.vc.indexOf(G.cols.indexOf(col));
    if (c >= 0) {
      gSelect(G.sel.fr, c);
      gEnsureVisible(G.sel.fr, c);
    }
    gSay(`列「${col.title}」を足しました（保存すると、シートの右端にできます）`);
  }

  async function gRenameCol(ci) {
    const col = G.cols[ci];
    if (!col || !col.free || G.busy) return;
    const n = gFilled(ci);
    const title = await askText("列の名前を変える", `列「${col.title}」の新しい見出しを書いてください。${n ? `この列には値が ${n} 個入っています（値はそのまま残ります）。` : ""}保存すると、シートの見出しが変わります。`, col.title, "変える");
    if (title == null || title.trim() === col.title) return gFocusBack();
    let key = "";
    try {
      key = await api().EditColumn(G.id, title, gHeadings(col));
    } catch (e) {
      gSay("列の名前を変えられません: " + errText(e));
      return gFocusBack();
    }
    // A column of the book keeps its name until the save (the cells of the
    // draft go by it); one that was added here has no other name yet.
    const was = col.added ? "" : col.was || col.title;
    const next = { title: title.trim(), was: was === title.trim() ? "" : was };
    if (col.added) next.key = key;
    gMutate((ops) => ops.col(col, next));
    gSay(`列の見出しを「${next.title}」にしました`);
  }

  async function gDropCol(ci) {
    const col = G.cols[ci];
    if (!col || !col.free || G.busy) return;
    const n = gFilled(ci);
    if (n && !(await ask("列を消す", `列「${col.title}」には値が ${n} 個入っています。列を消すと、保存のときに値も消えます（保存するまでは Ctrl+Z で戻せます）。`, "消す"))) return gFocusBack();
    gMutate((ops) => ops.col(col, { dropped: true }));
    gSay(`列「${col.title}」を消しました（保存すると、シートから消えます。Ctrl+Z で戻せます）`);
  }

  // ---- search and replace ----

  function gFindOpen(replace) {
    if (!G || G.busy) return;
    gCommitEdit();
    gCloseList();
    const F = st.edit.find;
    const g = gRange();
    F.range = null;
    if (G.view.length && (g.r1 > g.r0 || g.c1 > g.c0)) {
      const uids = new Set(), cis = new Set();
      for (let r = g.r0; r <= g.r1; r++) uids.add(gRowAt(r).uid);
      for (let c = g.c0; c <= g.c1; c++) cis.add(G.vc[c]);
      F.range = { uids, cis };
    }
    F.onlySel = !!F.range;
    F.open = true;
    F.replace = replace;
    F.said = "";
    gFindDraw(true);
    gFit();
  }

  function gFindClose() {
    st.edit.find.open = false;
    gFindDraw(false);
    gFit();
    gFocusBack();
  }

  function gFindSay(text) {
    st.edit.find.said = text;
    const el = $("gf-info");
    if (el) el.textContent = text;
  }

  function gFindDraw(focus) {
    const bar = $("g-findbar"), F = st.edit.find;
    if (!bar) return;
    bar.textContent = "";
    bar.hidden = !F.open;
    if (!F.open) return;
    const q = h("input", { type: "text", id: "gf-q", value: F.q, placeholder: "検索する文字", oninput: (e) => { F.q = e.target.value; } });
    const rep = F.replace ? h("input", { type: "text", id: "gf-r", value: F.r, placeholder: "置換後の文字", oninput: (e) => { F.r = e.target.value; } }) : null;
    const box = (id, label, key, off) => h("label", { class: "g-fopt" }, h("input", { type: "checkbox", id, checked: F[key] && !off, disabled: off, onchange: (e) => { F[key] = e.target.checked; } }), " " + label);
    const scope = Object.assign(select([["sheet", "今のシート"], ["all", "全シート"]], F.scope, (v) => { F.scope = v; gFindDraw(true); }), { id: "gf-scope", title: "全シート: 「次へ」が今のシートの最後を過ぎると、次の当たったシートへ移ります" });
    add(bar, [
      q, rep, scope,
      F.range ? box("gf-sel", "選択範囲のみ", "onlySel", F.scope === "all") : null,
      box("gf-case", "大文字と小文字を区別", "cs"),
      box("gf-whole", "セル全体が一致", "whole"),
      button("前へ", () => gFindStep(-1), { id: "gf-prev" }),
      button("次へ", () => gFindStep(1), { id: "gf-next" }),
      F.replace ? [button("置換", gReplaceOne, { id: "gf-rep" }), button("すべて置換", gReplaceAll, { id: "gf-all" })] : button("置換…", () => gFindOpenKeep(true), { id: "gf-more", class: "link" }),
      h("span", { class: "dim small", id: "gf-info" }, F.said),
      button("×", gFindClose, { id: "gf-close", class: "link", title: "閉じる（Esc）" }),
    ]);
    bar.onkeydown = (e) => {
      e.stopPropagation();
      // Enter and Esc while the IME is converting are the IME's.
      if (e.isComposing || e.keyCode === 229) return;
      const lk = e.key.toLowerCase();
      if (e.key === "Escape") {
        e.preventDefault();
        gFindClose();
      } else if (e.key === "Enter" && e.target.tagName === "INPUT" && e.target.type === "text") {
        e.preventDefault();
        gFindStep(e.shiftKey ? -1 : 1);
      } else if ((e.ctrlKey || e.metaKey) && (lk === "f" || lk === "h")) {
        e.preventDefault();
        gFindOpenKeep(lk === "h");
      } else if ((e.ctrlKey || e.metaKey) && ["r", "d", "s", "p"].includes(lk)) e.preventDefault();
    };
    if (focus) {
      q.focus();
      q.select();
    }
  }

  // gFindOpenKeep switches the bar between search and replace without
  // touching the range it was opened for.
  function gFindOpenKeep(replace) {
    st.edit.find.replace = replace;
    gFindDraw(true);
    gFit();
  }

  function gFindOpts() {
    const F = st.edit.find, hidden = {};
    for (const k in st.edit.prefs) hidden[k] = st.edit.prefs[k].hidden || [];
    return { case: F.cs, whole: F.whole, secrets: !!G.show, hidden };
  }

  function gHit(text) {
    const F = st.edit.find;
    const a = F.cs ? text : text.toLowerCase(), b = F.cs ? F.q : F.q.toLowerCase();
    return F.whole ? a === b : a.includes(b);
  }

  // gMatches lists the cells of this sheet that hold the text: [place of the
  // row in G.full, place of the column among the ones shown]. The rows a
  // filter hides and the columns that are hidden are not searched; a secret
  // column is searched only while the secrets are shown.
  function gMatches() {
    const F = st.edit.find;
    if (!F.q) return [];
    const only = F.scope === "sheet" && F.onlySel && F.range ? F.range : null;
    const out = [];
    G.full.forEach((ri, pos) => {
      const r = G.rows[ri];
      if (only && !only.uids.has(r.uid)) return;
      for (let c = 0; c < G.vc.length; c++) {
        const ci = G.vc[c], col = G.cols[ci];
        if (only && !only.cis.has(ci)) continue;
        if (col.secret && !G.show) continue;
        if (gHit(r.cells[ci])) out.push([pos, c]);
      }
    });
    return out;
  }

  // gFindGo puts the cursor on a match, drawing more rows if it lies beyond
  // the ones that are drawn.
  function gFindGo(m, i) {
    const F = st.edit.find;
    const ri = G.full[m[i][0]];
    let vr = G.view.indexOf(ri);
    if (vr < 0) {
      vr = gReveal(ri);
      gRenderBody();
    }
    G.sel = { ar: vr, ac: m[i][1], fr: vr, fc: m[i][1] };
    gPaintSel();
    gEnsureVisible(vr, m[i][1]);
    gFindSay(`${m.length} 件中 ${i + 1} 件目` + (F.scope === "all" && F.all ? `（全シート ${F.all} 件）` : ""));
  }

  async function gFindStep(dir) {
    const F = st.edit.find;
    if (!G || G.busy) return;
    if (!F.q) {
      gFindSay("検索する文字を書いてください");
      return;
    }
    gCommitEdit();
    const m = gMatches();
    const here = G.full.indexOf(G.view[G.sel.fr]);
    const at = here * 1000 + G.sel.fc;
    let i = dir > 0 ? m.findIndex(([p, c]) => p * 1000 + c > at) : m.map(([p, c]) => p * 1000 + c < at).lastIndexOf(true);
    if (i < 0 && F.scope === "all" && (await gFindOther(dir))) return;
    if (!m.length) {
      gFindSay(F.scope === "all" ? "どのシートにも見つかりません" : "見つかりません");
      return;
    }
    if (i < 0) i = dir > 0 ? 0 : m.length - 1;
    gFindGo(m, i);
  }

  // gFindOther goes on with the search in the next sheet that holds the
  // text. It tells whether it did.
  async function gFindOther(dir) {
    const F = st.edit.find;
    await gSaveDraft(G);
    let counts = [];
    try {
      counts = await api().EditFind(F.q, gFindOpts());
    } catch (e) {
      gFindSay(errText(e));
      return true;
    }
    F.all = counts.reduce((n, c) => n + c.count, 0);
    const ids = st.edit.list.sheets.map((s) => s.id), has = new Set(counts.map((c) => c.sheet));
    const from = ids.indexOf(G.id), n = ids.length;
    for (let k = 1; k < n; k++) {
      const id = ids[(((from + dir * k) % n) + n) % n];
      if (!has.has(id)) continue;
      F.focus = true;
      await gSwitch(id, () => {
        const m = gMatches();
        if (!m.length) return;
        const i = dir > 0 ? 0 : m.length - 1;
        const vr = gReveal(G.full[m[i][0]]);
        G.sel = { ar: vr, ac: m[i][1], fr: vr, fc: m[i][1] };
        G.jumped = true;
        F.said = `シート「${G.v.name}」の ${m.length} 件中 ${i + 1} 件目（全シート ${F.all} 件）`;
      });
      return true;
    }
    return false;
  }

  // gReplaced is the value of a cell after the replacement, or null when the
  // column has choices and the outcome is none of them: the cell is left.
  function gReplaced(col, v) {
    const F = st.edit.find;
    let out;
    if (F.whole) out = gNorm(col, F.r);
    else {
      const re = new RegExp(F.q.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), F.cs ? "g" : "gi");
      out = gNorm(col, v.replace(re, () => F.r));
    }
    if (col.choices.length && out !== "" && !col.choices.includes(out)) return null;
    return out;
  }

  function gReplaceOne() {
    const F = st.edit.find;
    if (!G || G.busy || !F.q) return;
    gCommitEdit();
    const here = G.full.indexOf(G.view[G.sel.fr]);
    const hit = gMatches().find(([p, c]) => p === here && c === G.sel.fc);
    let left = "";
    if (hit) {
      const ci = G.vc[hit[1]], col = G.cols[ci], row = G.rows[G.full[hit[0]]];
      const nv = gReplaced(col, row.cells[ci]);
      if (nv === null) left = "置換後の値が選択肢に無いため飛ばしました。";
      else gMutate((ops) => ops.set(row, ci, nv));
    }
    gFindStep(1).then(() => { if (left) gFindSay(left + st.edit.find.said); });
  }

  async function gReplaceAll() {
    const F = st.edit.find;
    if (!G || G.busy) return;
    if (!F.q) {
      gFindSay("検索する文字を書いてください");
      return;
    }
    gCommitEdit();
    const m = gMatches();
    let other = null, rest = [];
    if (F.scope === "all") {
      await gSaveDraft(G);
      try {
        rest = (await api().EditFind(F.q, gFindOpts())).filter((c) => c.sheet !== G.id);
      } catch (e) {
        gFindSay(errText(e));
        return;
      }
      const n = rest.reduce((k, c) => k + c.count, 0);
      if (!m.length && !n) {
        gFindSay("どのシートにも見つかりません");
        return;
      }
      const where = rest.length ? `、ほかのシート ${rest.length} 枚の ${n} 件（${rest.map((c) => `${c.name} ${c.count}`).join("、")}）` : "";
      if (!(await ask("全シートで置換", `「${F.q}」を「${F.r}」に置換します。このシートの ${m.length} 件${where}。\n置換は下書きに入ります（パラメータブックは「保存」まで変わりません）。ほかのシートの分は、シートを切り替えるまで Ctrl+Z で戻せます。`, "置換する"))) return;
      if (n) {
        try {
          other = await api().EditReplace(F.q, F.r, gFindOpts(), G.id);
        } catch (e) {
          gFindSay(errText(e));
          return;
        }
      }
    } else if (!m.length) {
      gFindSay("見つかりません");
      return;
    }
    let done = 0, left = 0;
    gMutate((ops) => {
      for (const [pos, c] of m) {
        const ci = G.vc[c], col = G.cols[ci], row = G.rows[G.full[pos]];
        const nv = gReplaced(col, row.cells[ci]);
        if (nv === null || row.del || row.formula.has(ci)) left++;
        else if (ops.set(row, ci, nv)) done++;
      }
      if (other && other.after.length) ops.parts(other.before, other.after);
    });
    const said = [`${done} 件を置換しました`];
    if (other) {
      const n = other.counts.reduce((k, c) => k + c.count, 0);
      left += other.counts.reduce((k, c) => k + c.skipped, 0);
      said[0] = `このシートの ${done} 件と、ほかのシート ${other.after.length} 枚の ${n} 件を置換しました`;
      try {
        st.edit.list = await api().EditSheets();
        if (st.home) st.home.draft = st.edit.list.draft;
        gSheetLabels();
      } catch (e) {
        G.trouble = errText(e);
      }
    }
    if (left) said.push(`${left} 件は飛ばしました（消す行・式のセル・置換後の値が選択肢に無いセル）`);
    gFindSay(said.join("。"));
    gStatus();
  }

  // ---- keys and the mouse ----

  // gJumpTarget is where Ctrl+arrow lands, as in Excel.
  function gJumpTarget(r, c, dr, dc) {
    const R = G.view.length - 1, C = G.vc.length - 1;
    const inside = (y, x) => y >= 0 && y <= R && x >= 0 && x <= C;
    const filled = (y, x) => !!gRowAt(y).cells[G.vc[x]];
    if (!inside(r + dr, c + dc)) return [r, c];
    if (filled(r, c) && filled(r + dr, c + dc)) {
      while (inside(r + dr, c + dc) && filled(r + dr, c + dc)) {
        r += dr;
        c += dc;
      }
      return [r, c];
    }
    r += dr;
    c += dc;
    while (inside(r + dr, c + dc) && !filled(r, c)) {
      r += dr;
      c += dc;
    }
    return [r, c];
  }

  function gKey(e) {
    if (!G || G.edit || gDialogOpen() || e.target.closest(".g-editor")) return;
    const k = e.key, ctrl = e.ctrlKey || e.metaKey, lk = k.toLowerCase();
    const R = G.view.length - 1, C = G.vc.length - 1;
    const done = () => {
      e.preventDefault();
      e.stopPropagation();
    };
    if (ctrl) {
      if (lk === "c") { done(); gCopy(false); return; }
      if (lk === "x") { done(); gCopy(true); return; }
      if (lk === "v") { done(); gPaste(); return; }
      if (lk === "z" && e.shiftKey) { done(); gUndo(true); return; }
      if (lk === "z") { done(); gUndo(false); return; }
      if (lk === "y") { done(); gUndo(true); return; }
      if (lk === "d") { done(); gFillDown(false); return; }
      if (lk === "r") { done(); gFillDown(true); return; }
      if (lk === "a") { done(); gSelect(0, 0, R, C); return; }
      if (lk === "s") { done(); gSaveGo(); return; }
      if (lk === "f") { done(); gFindOpen(false); return; }
      if (lk === "h") { done(); gFindOpen(true); return; }
      if (k === "+" || (e.shiftKey && (k === "=" || k === ";"))) { done(); gInsertAt(false); return; }
      if (k === "-") { done(); gDeleteRows(); return; }
      if (k === "Home") { done(); if (e.shiftKey) gMove(-G.sel.fr, -G.sel.fc, true); else gSelect(0, 0); return; }
      if (k === "End") {
        done();
        let last = 0;
        G.view.forEach((ri, r) => { if (!gBlank(G.rows[ri])) last = r; });
        if (e.shiftKey) gMove(last - G.sel.fr, C - G.sel.fc, true);
        else gSelect(last, C);
        return;
      }
      if (G_ARROWS[k]) {
        done();
        const [tr, tc] = gJumpTarget(G.sel.fr, G.sel.fc, G_ARROWS[k][0], G_ARROWS[k][1]);
        gMove(tr - G.sel.fr, tc - G.sel.fc, e.shiftKey);
        return;
      }
      if (k === " ") { done(); gSelect(0, G.sel.ac, R, G.sel.fc); return; }
      return;
    }
    if (k === "ContextMenu" || (k === "F10" && e.shiftKey)) {
      done();
      const cur = G.tds[G.sel.fr] && G.tds[G.sel.fr][G.sel.fc];
      const b = cur ? cur.getBoundingClientRect() : G.wrap.getBoundingClientRect();
      gCellMenu(b.left + 12, b.bottom);
      return;
    }
    if (k === " " && e.shiftKey) { done(); gSelect(G.sel.ar, 0, G.sel.fr, C); return; }
    if (e.altKey && (k === "ArrowUp" || k === "ArrowDown")) { done(); gMoveRows(k === "ArrowUp" ? -1 : 1); return; }
    if (G_ARROWS[k]) { done(); gMove(G_ARROWS[k][0], G_ARROWS[k][1], e.shiftKey); return; }
    if (k === "Tab") { done(); gTab(e.shiftKey); return; }
    if (k === "Enter") { done(); gEnterDown(e.shiftKey); return; }
    if (k === "PageDown") { done(); gMove(20, 0, e.shiftKey); return; }
    if (k === "PageUp") { done(); gMove(-20, 0, e.shiftKey); return; }
    if (k === "Home") { done(); gMove(0, -G.sel.fc, e.shiftKey); return; }
    if (k === "End") { done(); gMove(0, C - G.sel.fc, e.shiftKey); return; }
    if (k === "F2") { done(); gBeginEdit(null); return; }
    if ((k === " " || k === "F4") && gColAt(G.sel.fc).choices.length) { done(); gOpenList(); return; }
    if (k === "Delete") { done(); gClear(); return; }
    if (k === "Backspace") { done(); gBeginEdit(""); return; }
    if (k === "Escape") { done(); gSelect(G.sel.fr, G.sel.fc); return; }
    // Typing starts editing: the editor takes the focus during this
    // keydown, so the key itself (or the composition of the IME) lands in it.
    if ((k.length === 1 && !e.altKey) || k === "Process") {
      e.stopPropagation();
      gBeginEdit("");
    }
  }

  function gCellAt(x, y) {
    const el = document.elementFromPoint(x, y);
    if (!el) return null;
    const td = el.closest("#g-body td[data-r]");
    if (td) return { r: +td.dataset.r, c: +td.dataset.c };
    const rh = el.closest("#g-body th.g-rh");
    if (rh) return { r: +rh.dataset.r, c: null };
    const ch = el.closest("#g-table thead th[data-c]");
    if (ch) return { r: null, c: +ch.dataset.c };
    return null;
  }

  function gMouseDown(e) {
    if (e.button !== 0 || e.target.closest(".g-editor")) return;
    const opt = e.target.closest(".g-opt");
    if (opt) {
      e.preventDefault();
      gPickList(+opt.dataset.i);
      return;
    }
    if (e.target.closest("#g-list")) {
      e.preventDefault();
      return;
    }
    if (e.target.id === "g-dd") {
      e.preventDefault();
      const open = !!G.list;
      gCloseList();
      if (!open) gOpenList();
      else G.wrap.focus({ preventScroll: true });
      return;
    }
    gCloseList();
    if (G.edit) gCommitEdit();
    const hm = e.target.closest(".g-hm");
    if (hm) {
      e.preventDefault();
      const b = hm.getBoundingClientRect();
      gHeaderMenu(G.vc[+hm.dataset.hm], b.left, b.bottom + 2);
      return;
    }
    const rz = e.target.closest(".g-rz");
    if (rz) {
      const c = +rz.dataset.rz;
      gDrag = { mode: "rz", c, x: e.clientX, w: G.wrap.querySelectorAll("#g-table col")[c + 1].getBoundingClientRect().width || gColWidth(gColAt(c)), nw: 0 };
      e.preventDefault();
      e.stopPropagation();
      return;
    }
    if (e.target.id === "g-fill") {
      gDrag = { mode: "fill", g: gRange(), ft: null };
      e.preventDefault();
      return;
    }
    const at = e.target.closest(".g-corner") ? { all: true } : gCellAt(e.clientX, e.clientY);
    const R = G.view.length - 1, C = G.vc.length - 1;
    if (!at) return;
    if (at.all) gSelect(0, 0, R, C);
    else if (at.r != null && at.c != null) {
      if (e.shiftKey) {
        G.sel.fr = at.r;
        G.sel.fc = at.c;
        gPaintSel();
      } else gSelect(at.r, at.c);
      gDrag = { mode: "cell" };
    } else if (at.r != null) {
      if (e.shiftKey) {
        G.sel.fr = at.r;
        G.sel.ac = 0;
        G.sel.fc = C;
        gPaintSel();
      } else gSelect(at.r, 0, at.r, C);
      gDrag = { mode: "row" };
    } else {
      if (e.shiftKey) {
        G.sel.fc = at.c;
        G.sel.ar = 0;
        G.sel.fr = R;
        gPaintSel();
      } else gSelect(0, at.c, R, at.c);
      gDrag = { mode: "col" };
    }
    G.wrap.focus({ preventScroll: true });
    e.preventDefault();
  }

  function gMouseMove(e) {
    if (!gDrag || !G || !G.wrap || !G.wrap.isConnected) return;
    if (gDrag.mode === "rz") {
      const w = Math.max(40, Math.min(600, gDrag.w + e.clientX - gDrag.x));
      const col = G.wrap.querySelectorAll("#g-table col")[gDrag.c + 1];
      if (col) col.style.width = w + "px";
      gDrag.nw = w;
      return;
    }
    const w = G.wrap.getBoundingClientRect();
    // Dragging past the edge scrolls, as in Excel.
    if (e.clientY > w.bottom - 16) G.wrap.scrollTop += 16;
    else if (e.clientY < w.top + 30) G.wrap.scrollTop -= 16;
    if (e.clientX > w.right - 16) G.wrap.scrollLeft += 24;
    else if (e.clientX < w.left + 60) G.wrap.scrollLeft -= 24;
    const at = gCellAt(Math.min(Math.max(e.clientX, w.left + 60), w.right - 4), Math.min(Math.max(e.clientY, w.top + 32), w.bottom - 4)) || {};
    if (gDrag.mode === "fill") {
      // Below the last row the fill goes on into new rows, as in Excel.
      const last = G.tds.length && G.tds[G.tds.length - 1][0];
      if (last && gDrag.g.r1 === G.view.length - 1 && !gSorted()) {
        const lr = last.getBoundingClientRect();
        if (e.clientY > lr.bottom) {
          const extra = Math.min(500, Math.ceil((e.clientY - lr.bottom) / Math.max(lr.height, 1)));
          gDrag.ft = { dir: "down", t: { r0: gDrag.g.r1 + 1, r1: gDrag.g.r1 + extra, c0: gDrag.g.c0, c1: gDrag.g.c1 } };
          gPaintSel(gDrag.ft.t);
          gFillHint(`下に ${extra} 行を足して埋めます`);
          return;
        }
      }
      if (at.r == null || at.c == null) return;
      gDrag.ft = gFillTarget(gDrag.g, at.r, at.c);
      gPaintSel(gDrag.ft ? gDrag.ft.t : null);
      if (gDrag.ft) gFillHint(`${gDrag.ft.t.r1 - gDrag.ft.t.r0 + 1} 行 × ${gDrag.ft.t.c1 - gDrag.ft.t.c0 + 1} 列を埋めます（数字で終わる値は連番。Ctrl を押しながら離すと逆）`);
      return;
    }
    if (gDrag.mode === "cell" && at.r != null && at.c != null) {
      G.sel.fr = at.r;
      G.sel.fc = at.c;
    } else if (gDrag.mode === "row" && at.r != null) G.sel.fr = at.r;
    else if (gDrag.mode === "col" && at.c != null) G.sel.fc = at.c;
    else return;
    gPaintSel();
  }

  function gMouseUp(e) {
    const d = gDrag;
    gDrag = null;
    if (!d || !G || !G.wrap || !G.wrap.isConnected) return;
    if (d.mode === "rz") {
      if (d.nw) gSetWidth(d.c, d.nw);
    } else if (d.mode === "fill") {
      if (d.ft) gFill(d.g, d.ft, e.ctrlKey || e.metaKey);
      else gPaintSel();
    }
  }

  // gDocKey: while the table is on the screen, the keys that would open the
  // search of the browser or load the page anew are the table's.
  function gDocKey(e) {
    if (st.page !== "edit" || !G || !G.wrap || !G.wrap.isConnected || !(e.target instanceof Element)) return;
    if (e.key === "F5") {
      e.preventDefault();
      return;
    }
    const lk = e.key.toLowerCase();
    if (!(e.ctrlKey || e.metaKey) || !["r", "f", "h", "p"].includes(lk)) return;
    // The table and the bar of the search take them by themselves; the
    // editor of a cell does not.
    if (e.target.closest("#g-wrap, #g-findbar") && !e.target.closest(".g-editor")) return;
    e.preventDefault();
    e.stopPropagation();
    if ((lk === "f" || lk === "h") && !gDialogOpen()) gFindOpen(lk === "h");
  }

  // gFit caps the table at the bottom edge of the window from where its top is.
  function gFit() {
    const w = G && G.wrap;
    if (!w || !w.isConnected) return;
    const hgt = Math.max(180, Math.floor(window.innerHeight - w.getBoundingClientRect().top - 20));
    if (w.style.maxHeight !== hgt + "px") {
      w.style.maxHeight = hgt + "px";
      gPaintSel();
    }
  }

  // ---- the page ----

  // draftNote tells, on the screens that work on the saved book, that the
  // table holds changes that are not in it yet.
  function draftNote() {
    const n = st.home ? st.home.draft : 0;
    if (!n) return null;
    return h("div", { class: "box warn", id: "draft-note" },
      `保存していない変更が ${n} 件あります（表の編集の下書き）。「生成とチェック」などは、保存済みのパラメータブックで動きます。 `,
      button("表の編集を開く", () => openEdit(null), { class: "link" }));
  }

  // gSheetLabels writes the names of the sheets to choose from anew (the
  // number of changes of their drafts is in them).
  function gSheetLabels() {
    const sel = $("g-sheet");
    if (!sel || !st.edit.list) return;
    const by = new Map(st.edit.list.sheets.map((s) => [s.id, s]));
    for (const o of sel.options) if (by.has(o.value)) o.textContent = gSheetLabel(by.get(o.value));
  }

  const gSheetLabel = (s) => (s.missing ? `${s.name}${s.draft ? `（下書き ${s.draft} 件）` : ""}` : `${s.name}（${s.rows} 行${s.draft ? `・下書き ${s.draft} 件` : ""}）`);

  function editPage() {
    const ed = st.edit;
    const out = [h("h1", null, "表の編集")];
    if (ed.msg) out.push(h("div", { class: "box info", id: "edit-msg" }, button("閉じる", () => { ed.msg = ""; ed.redo = null; render(); }, { class: "link close" }), ed.msg,
      ed.redo ? h("div", { class: "row" }, button("案を作り直す", () => { const hs = ed.redo; ed.msg = ""; ed.redo = null; openRedo(hs); }, { id: "edit-redo" })) : null));
    if (!ed.list) {
      out.push(h("p", { class: "dim" }, "パラメータブックのシートを、表として開いて直します。"), h("div", { class: "row" }, button("開く", () => openEdit(null), { class: "primary" })));
      return out;
    }
    if (!ed.list.hasBook || !G) {
      out.push(h("div", { class: "box info" }, ed.list.hasBook ? `パラメータブック ${ed.list.book} に、表で開けるシートがありません。` : `パラメータブック ${ed.list.book} はまだありません。旧コンフィグを取り込むか、ホームの「空のブックを作る」で作られます。`),
        h("div", { class: "row" }, button("ホームへ", () => go("home"))));
      return out;
    }
    if (ed.list.lost.length) out.push(h("div", { class: "box warn" }, `下書きに、パラメータブックにもう無いシートの変更があります（${ed.list.lost.join("、")}）。保存のときに、その分をやめるかを聞きます。`));
    const have = ed.list.sheets.filter((s) => !s.missing), lack = ed.list.sheets.filter((s) => s.missing);
    const opt = (s) => h("option", { value: s.id }, gSheetLabel(s));
    const sheets = h("select", { id: "g-sheet", onchange: (e) => gSwitch(e.target.value) },
      // The sheets of the values first, the sheets of the mapping and the
      // common values apart (spec 14.18.5).
      h("optgroup", { label: "パラメータブックの値" }, have.filter((x) => !x.id.startsWith("@")).map(opt)),
      have.some((x) => x.id.startsWith("@")) ? h("optgroup", { label: "新旧対照表・共通値" }, have.filter((x) => x.id.startsWith("@")).map(opt)) : null,
      lack.length ? h("optgroup", { label: `まだシートが無い表（${lack.length}）`, id: "g-lack" }, lack.map(opt)) : null);
    sheets.value = G.id;
    const devs = new Set();
    if (G.v.device >= 0) for (const r of G.rows) if (r.row !== 0 && r.cells[G.v.device]) devs.add(r.cells[G.v.device]);
    const devOpts = [["", G.cols[G.v.device] ? `すべて（${G.cols[G.v.device].title}）` : "すべて"]].concat([...devs].sort().map((d) => [d, d]));
    const refilter = () => {
      gCommitEdit();
      G.shown = G_ROWS;
      gRefilter();
      gView();
      G.sel = { ar: 0, ac: G.sel.fc, fr: 0, fc: G.sel.fc };
      gRenderBody();
    };
    const filter = h("input", { type: "text", id: "g-filter", value: G.filter, placeholder: "絞り込み（空白で区切ると、すべてを含む行）",
      oninput: (e) => { if (e.isComposing) return; G.filter = e.target.value; refilter(); },
      onkeydown: (e) => { if (!e.isComposing && (e.key === "Enter" || e.key === "Escape")) { e.preventDefault(); G.wrap.focus(); } } });
    filter.addEventListener("compositionend", (e) => { G.filter = e.target.value; refilter(); });
    const tool = (label, fn, props) => button(label, () => { fn(); gFocusBack(); }, props);
    out.push(
      h("div", { class: "row" },
        h("span", null, "シート: "), sheets,
        G.v.device >= 0 ? Object.assign(select(devOpts, G.dev, (d) => { G.dev = d; refilter(); }), { id: "g-dev" }) : null,
        filter,
        button("保存", gSaveGo, { class: "primary", id: "g-save", title: "Ctrl+S。下書きをパラメータブック（Excel）に書きます" }),
        button("保存してチェック", gSaveCheck, { id: "g-save-check", title: "保存して、続けて「チェックだけ」を行い、結果の指摘を出します" }),
        button("表の編集を終える", gClose, { id: "g-close" })),
      // The tools used most stay in sight; the rest are in three menus
      // (spec 14.12.5).
      h("div", { class: "row g-tools" },
        tool("元に戻す", () => gUndo(false), { id: "g-undo", title: "Ctrl+Z" }),
        tool("やり直す", () => gUndo(true), { id: "g-redo", title: "Ctrl+Y" }),
        button("検索と置換", () => gFindOpen(true), { id: "g-find", title: "Ctrl+F（検索）・Ctrl+H（置換）" }),
        menu("行", "g-menu-rows", [
          tool("上に行を入れる", () => gInsertAt(false), { id: "g-ins-above", title: "Ctrl+Shift++。選んだ行の数だけ、空の行を上に入れます。保存すると、シートのその位置に入ります" }),
          tool("下に行を入れる", () => gInsertAt(true), { id: "g-ins-below", title: "選んだ行の数だけ、空の行を下に入れます" }),
          tool("行を複製", gDuplicateRows, { id: "g-dup", title: "選んだ行の写しを、すぐ下の新しい行に作ります" }),
          tool("上へ", () => gMoveRows(-1), { id: "g-up", title: "Alt+↑。選んだ行を 1 行上へ動かします。保存すると、シートの行が動きます" }),
          tool("下へ", () => gMoveRows(1), { id: "g-down", title: "Alt+↓。選んだ行を 1 行下へ動かします" }),
          tool("行を削除", gDeleteRows, { id: "g-del", title: "Ctrl+－。パラメータブックの行は、保存するまで消えません" }),
          tool("新しい行へ", gToEnd, { id: "g-end", title: "表の下の空の行へ移ります。打つか貼ると、新しい行になります（保存すると、シートの末尾に入ります）" })]),
        menu("列", "g-menu-cols", [
          button("列の表示", gColsDialog, { id: "g-cols", title: "表に出す列を選びます（空の列をまとめて隠せます）" }),
          button("列を足す", gAddCol, { id: "g-addcol", title: "定義に無い列（自由列）を足します" })]),
        menu("下書き", "g-menu-draft", [
          button("下書きでチェック", gDraftCheck, { id: "g-draft-check", title: "パラメータブックを書かずに、下書きを当てた内容でチェックします" }),
          button("下書きを捨てる", gDiscard, { id: "g-discard" }),
          button("読み直す", gReload, { id: "g-reload", title: "Excel で直して保存した後に押すと、今のパラメータブックを読み直します（下書きは残ります）" }),
          button("Excel で開く", () => show(st.cas.book))]),
        tool(G.show ? "秘密の値を隠す" : "秘密の値を見せる", () => { gCommitEdit(); G.show = !G.show; gRedrawAll(); }, { id: "g-show", hidden: !G.cols.some((c) => c.secret) })),
      h("div", { class: "g-findbar", id: "g-findbar", hidden: true }),
      mapSheetNote(),
      G.v.missing ? h("div", { class: "box info", id: "g-missing" }, `シート「${G.v.name}」は、まだパラメータブックにありません。下の表に行を足して「保存」を押すと、ブックにこのシートを作ってから行を書きます（何も足さなければ、シートは作りません）。`) : null,
      h("div", { class: "row" },
        h("span", { class: "dim small", id: "g-status" }),
        h("span", { class: "dim small", id: "g-selinfo" })),
      h("p", { class: "dim small" }, "Excel と同じ操作です: セルを選んで打つ・Ctrl+C / Ctrl+V で Excel と相互にコピーと貼り付け・Ctrl+D で上の値を下へ・右下の ■ をドラッグして連番・Ctrl+F / Ctrl+H で検索と置換・右クリックでメニュー・見出しの ▾ で並べ替えと絞り込み・Ctrl+Z で元に戻す・Alt+Enter でセルの中の改行。行番号は、パラメータブックの行番号です。直した内容は下書きとして残り、「保存」を押すまでパラメータブックは変わりません。"));

    const head = h("tr", null, h("th", { class: "g-corner g-rh", title: "すべて選択" }));
    G.vc.forEach((ci, c) => {
      const col = G.cols[ci];
      const sorted = G.sort.ci === ci, filtered = !!G.colFilter[ci];
      head.appendChild(h("th", { "data-c": String(c), class: cls(col.isKey && "g-key", col.free && "g-free", col.col < 0 && "g-nocol", c < G.lefts.length - 1 && "g-stick", col.mark && "g-m-" + col.mark) || null,
        style: c < G.lefts.length - 1 ? `left:${G.lefts[c]}px` : null, title: gColTip(col) },
      col.title, col.required && !col.isKey ? h("span", { class: "g-req" }, " *") : null, col.secret ? h("span", { class: "g-req" }, " 🔒") : null,
      h("button", { type: "button", class: cls("g-hm", (sorted || filtered) && "on"), "data-hm": String(c), tabindex: "-1", title: "並べ替え・絞り込み・列を隠す" }, (sorted ? (G.sort.dir > 0 ? "▲" : "▼") : "") + (filtered ? "⏷" : "▾")),
      h("span", { class: "g-rz", "data-rz": String(c), title: "ドラッグで列幅を変えます（ダブルクリックで中身に合わせます）" })));
    });
    const wrap = h("div", { class: "grid-wrap", id: "g-wrap", tabindex: "0" },
      h("table", { class: "grid", id: "g-table" },
        h("colgroup", null, h("col", { style: `width:${G_RH}px` }), G.vc.map((ci) => h("col", { style: `width:${gColWidth(G.cols[ci])}px` }))),
        h("thead", null, head),
        h("tbody", { id: "g-body" })),
      h("div", { class: "g-fill", id: "g-fill", hidden: true, title: "ドラッグで埋めます（数字で終わる値は連番。ダブルクリックで表の最後の行まで）" }),
      h("button", { class: "g-dd", id: "g-dd", type: "button", tabindex: "-1", hidden: true, title: "一覧から選ぶ（Space・F4 でも開きます。複数のセルを選んでいれば、まとめて入ります）" }, "▼"),
      h("div", { class: "g-list", id: "g-list", tabindex: "-1", hidden: true }));
    out.push(wrap);

    wrap.addEventListener("keydown", gKey);
    wrap.addEventListener("mousedown", gMouseDown);
    wrap.addEventListener("contextmenu", gContextMenu);
    wrap.addEventListener("dblclick", (e) => {
      if (e.target.id === "g-fill") {
        gFillToEnd();
        return;
      }
      const rz = e.target.closest(".g-rz");
      if (rz) {
        gAutoFit(+rz.dataset.rz);
        return;
      }
      if (e.target.closest(".g-hm")) return;
      const at = gCellAt(e.clientX, e.clientY);
      if (!at || at.r == null || at.c == null) return;
      gSelect(at.r, at.c);
      gBeginEdit(null);
    });
    wrap.addEventListener("scroll", () => { G.scroll = [wrap.scrollLeft, wrap.scrollTop]; });
    afterDraw.push(() => {
      G.wrap = wrap;
      G.edit = null;
      G.list = null;
      const list = $("g-list");
      list.addEventListener("keydown", gListKey);
      list.addEventListener("blur", () => setTimeout(() => { if (G && G.list && document.activeElement !== list) gCloseList(); }, 0));
      const F = st.edit.find;
      gFindDraw(false);
      gRenderBody();
      gFit();
      wrap.scrollLeft = G.scroll[0];
      wrap.scrollTop = G.scroll[1];
      if (G.jumped) {
        G.jumped = false;
        gEnsureVisible(G.sel.fr, G.sel.fc);
      }
      gPaintSel();
      if (gDialogOpen()) return;
      if (F.open && F.focus && $("gf-q")) $("gf-q").focus();
      else wrap.focus({ preventScroll: true });
      F.focus = false;
    });
    return out;
  }

  // ---- 差分の投入 (spec 9.8, 14.7.1) ----

  function loadDelta() {
    run("差分の投入の準備をしています", async () => {
      const v = await api().Delta();
      const d = st.dlt;
      d.view = v;
      if (!v.sides.some((x) => x.side === d.side && x.files > 0)) {
        const first = v.sides.find((x) => x.files > 0);
        d.side = first ? first.side : "old";
      }
      st.page = "delta";
    });
  }

  function deltaPage() {
    const d = st.dlt;
    const v = d.view;
    const out = [
      h("h1", null, "差分の投入"),
      h("p", { class: "dim" }, "パラメータブックから生成したコンフィグと、機器の今のコンフィグ（旧コンフィグ・新コンフィグ・投入後のログ）を比べて、変えるコマンドだけの投入を作ります。設定変更の案件で「変更分だけ入れる」ときに使います（全文を入れるときは「生成とチェック」のコンフィグを使います）。 ",
        tip("消すのは、ブックで消した・変えた設定のうち、ツールが表に取り込める行だけです。取り込めない行（読めなかった行）や、パックが書けない値の行は消さずに「残る行」に出します。値を変えた行は、新しい行で上書きされる前提で古い行を消しません（「置き換わる行」）。今のコンフィグのメーカーが違う機器と、IF 名が変わる機器は作りません。設定モードへの出入りは付けますが、保存のコマンドは付けません。")),
    ];
    if (!v) {
      out.push(h("div", { class: "row" }, button("始める", loadDelta, { class: "primary" })));
      return out;
    }
    const sides = v.sides.map((x) => [x.side, `今 = ${x.title}（${x.files} ファイル）`]);
    const chosen = new Set(d.devices);
    const go = () => run("差分の投入を作っています", async () => {
      d.run = await api().DeltaRun(d.side, [...chosen]);
      d.sel = "";
    }, true, "台");
    out.push(
      h("div", { class: "row" },
        h("span", null, "今のコンフィグ"),
        Object.assign(select(sides, d.side, (x) => { d.side = x; }), { id: "delta-side" }),
        button("作る", go, { class: "primary", id: "delta-run", disabled: !v.sides.some((x) => x.files > 0) }),
        button("読み直す", loadDelta)),
      v.devices.length ? h("details", { class: "fill-pick", open: d.pickOpen || null, ontoggle: (e) => { d.pickOpen = e.target.open; } },
        h("summary", null, chosen.size ? `機器: ${chosen.size} 台を選んでいます` : `機器: 全部（${v.devices.length} 台。一覧から選べます）`),
        h("div", { class: "row" },
          v.devices.map((x) => h("label", null, h("input", { type: "checkbox", checked: chosen.has(x), onchange: (e) => {
            if (e.target.checked) chosen.add(x); else chosen.delete(x);
            d.devices = v.devices.filter((y) => chosen.has(y));
            render();
          } }), " " + x)),
          button("選ぶのをやめる", () => { d.devices = []; render(); }, { class: "link", disabled: !chosen.size }))) : null);
    const r = d.run;
    if (!r) return out;
    out.push(
      h("h2", null, `結果: 作った ${r.made}・変更なし ${r.unchanged}・作らなかった ${r.notMade}`),
      boxes("warn", r.notes),
      h("div", { class: "row" },
        r.dir ? button("出力のフォルダを開く", () => show(r.dir)) : null,
        r.reportXlsx ? button("報告書を Excel で開く", () => show(r.reportXlsx)) : null,
        r.report ? button("報告書をブラウザで開く", () => show(r.report)) : null),
      tableView(r.table, { id: "delta/run", onPick: (i) => { d.sel = r.devices[i].name; st.jump = "delta-one"; render(); }, selected: r.devices.findIndex((x) => x.name === d.sel) }));
    const one = r.devices.find((x) => x.name === d.sel);
    if (one) {
      const list = (title, items, mark) => (items.length ? [h("h4", null, `${title}（${items.length}）`), h("pre", { class: "fill-out" }, items.map((x) => mark + " " + x).join("\n"))] : null);
      out.push(h("h3", { id: "delta-one" }, `${one.name}: ${one.word}`),
        one.why ? h("div", { class: "box warn" }, one.why) : null,
        one.now ? h("p", { class: "dim small" }, `今のコンフィグ: ${one.now}` + (one.file ? `　投入: ${one.file}` : "")) : null,
        one.text ? h("div", { class: "row" },
          button("この機器の投入を全部コピー", () => api().ClipboardSet(one.text.split(/\r?\n/).join("\r\n"))
            .then(() => toast(`${one.name} の投入をコピーしました`)).catch((e) => toast(errText(e), "err")), { class: "primary", id: "delta-copy" })) : null,
        one.text ? h("pre", { class: "fill-out" }, one.text) : null,
        list("足す行", one.added, "+"), list("消す行", one.removed, "-"), list("置き換わる行（新しい行で上書きされるので消さない）", one.replaced, "~"), list("残る行（消していない）", one.kept, "="));
    }
    return out;
  }

  // ---- 出力 (spec 14.12.6) ----

  function loadOutputs() {
    run("出力の一覧を読んでいます", async () => {
      st.outs = { list: await api().Outputs(), picked: {} };
      st.page = "outputs";
    });
  }

  function outputsPage() {
    const o = st.outs;
    const out = [
      h("h1", null, "出力"),
      h("p", { class: "dim" }, "out フォルダの、実行 1 回ごとのフォルダです。押すとフォルダを開きます。出力は、同じ入力からいつでも作り直せます。要らないものは選んで消せます（消したものは戻せません）。"),
    ];
    if (!o) return out;
    const size = (n) => (n >= 1048576 ? `${(n / 1048576).toFixed(1)} MB` : `${Math.max(1, Math.round(n / 1024))} KB`);
    const picked = o.list.filter((x) => o.picked[x.dir]);
    const drop = async () => {
      const kept = picked.filter((x) => x.keep);
      const text = `選んだ ${picked.length} 個の出力のフォルダを消します（${size(picked.reduce((a, x) => a + x.size, 0))}）。消したものは戻せません。` +
        (kept.length ? `\nいちばん新しい生成か納品一式のフォルダが ${kept.length} 個入っています。` : "");
      if (!(await ask("出力を消しますか", text, "消す"))) return;
      run("出力を消しています", async () => {
        o.list = await api().DropOutputs(picked.map((x) => x.dir));
        o.picked = {};
        toast(`${picked.length} 個の出力を消しました`);
      });
    };
    const total = o.list.reduce((a, x) => a + x.size, 0);
    out.push(
      h("div", { class: "row" },
        h("span", null, `${o.list.length} 個（${size(total)}）`),
        button("古いものを選ぶ", () => { o.picked = {}; for (const x of o.list) if (!x.keep) o.picked[x.dir] = true; render(); }, { title: "いちばん新しい生成と納品一式のフォルダのほかを、すべて選びます", disabled: !o.list.some((x) => !x.keep) }),
        button("選ぶのをやめる", () => { o.picked = {}; render(); }, { class: "link", disabled: !picked.length }),
        button(picked.length ? `選んだ ${picked.length} 個を消す` : "選んだものを消す", drop, { disabled: !picked.length, id: "outputs-drop" }),
        button("読み直す", loadOutputs),
        button("out フォルダを開く", () => show("out"))),
      o.list.length ? h("div", { class: "table-wrap outputs" }, h("table", { class: "listed" },
        h("thead", null, h("tr", null, h("th", null, ""), h("th", null, "日時"), h("th", null, "何をした出力か"), h("th", null, "ファイル"), h("th", null, "大きさ"), h("th", null, ""))),
        h("tbody", null, o.list.map((x) => h("tr", { class: o.picked[x.dir] ? "sel" : null },
          h("td", { class: "pickbox" }, h("input", { type: "checkbox", checked: !!o.picked[x.dir], "data-dir": x.dir, onchange: (e) => { if (e.target.checked) o.picked[x.dir] = true; else delete o.picked[x.dir]; render(); } })),
          h("td", null, x.when),
          h("td", { class: "kinds" }, x.kinds.join("・"), x.keep ? h("span", { class: "dim small" }, "（いちばん新しい）") : null),
          h("td", null, String(x.files)),
          h("td", null, size(x.size)),
          h("td", null, button("開く", () => show(x.dir), { class: "link" }))))))) : h("p", { class: "dim" }, "出力はまだありません。"));
    return out;
  }

  // ---- 検索 (spec 14.8.8; PalaTerm's ログの grep) ----

  function loadSearch() {
    run("検索の準備をしています", async () => {
      st.srch.view = await api().Search({ query: "", regex: false, case: false, scopes: [], out: "", misses: false });
      st.page = "search";
    });
  }

  function searchPage() {
    const sr = st.srch;
    const v = sr.view;
    if (!v) return [h("h1", null, "検索")];
    if (!sr.scopes) {
      sr.scopes = {};
      for (const sc of v.scopes) sr.scopes[sc.id] = true;
    }
    const search = () => {
      if (!sr.q.trim()) return;
      run("ファイルを検索しています", async () => {
        const scopes = v.scopes.filter((sc) => sr.scopes[sc.id]).map((sc) => sc.id);
        if (!scopes.length) throw new Error("探す場所を 1 つ以上選んでください");
        sr.view = await api().Search({ query: sr.q, regex: sr.regex, case: sr.cs, scopes, out: sr.out, misses: sr.misses });
        sr.file = null;
      });
    };
    if (sr.file) return fileViewer(sr);
    const q = h("input", { type: "text", id: "srch-q", value: sr.q, placeholder: "探す文字（例: 192.0.2.1、ip route）", oninput: (e) => { sr.q = e.target.value; },
      onkeydown: (e) => { if (e.key === "Enter" && !e.isComposing) search(); } });
    afterDraw.push(() => { if (!sr.view.files.length && document.activeElement === document.body) q.focus(); });
    const out = [
      h("h1", null, "検索"),
      h("p", { class: "dim" }, "案件のファイルをまとめて検索します。当たった行を押すと、そのファイルをその行で開きます。 ", tip("旧コンフィグ・新コンフィグ・ログ（input の各フォルダの直下）、追加コンフィグ、ひな形、作ったコンフィグ（out の選んだフォルダの configs と fill）を探します。Excel のファイルは探しません。")),
      h("div", { class: "box warn" }, "ファイルの文字をそのまま出します（秘密の値も伏せません）。"),
      h("div", { class: "row" }, q, button("検索", search, { class: "primary", id: "srch-run" })),
      h("div", { class: "row" },
        h("label", null, h("input", { type: "checkbox", checked: sr.regex, onchange: (e) => { sr.regex = e.target.checked; } }), " 正規表現"),
        h("label", null, h("input", { type: "checkbox", checked: sr.cs, onchange: (e) => { sr.cs = e.target.checked; } }), " 大文字と小文字を区別"),
        h("label", null, h("input", { type: "checkbox", checked: sr.misses, onchange: (e) => { sr.misses = e.target.checked; } }), " 当たらなかったファイルも出す")),
      h("div", { class: "row" }, h("span", null, "探す場所"),
        v.scopes.map((sc) => h("label", null, h("input", { type: "checkbox", checked: !!sr.scopes[sc.id], onchange: (e) => { sr.scopes[sc.id] = e.target.checked; } }), " " + sc.label)),
        v.outs.length ? [h("span", { class: "dim" }, "作ったコンフィグのフォルダ"), select([["", "一番新しいもの"]].concat(v.outs.map((o) => [o, o])), sr.out, (x) => { sr.out = x; })] : null),
    ];
    if (v.read || v.files.length || v.errors.length) {
      const lines = [];
      for (const f of v.files) for (const hit of f.hits) lines.push(`${f.rel}:${hit.line}: ${hit.text}`);
      const text = lines.join("\n") + (v.missed.length ? "\n\n当たらなかったファイル:\n" + v.missed.join("\n") : "");
      out.push(h("div", { class: "row" },
        h("span", null, `${v.read} ファイルを読み、${v.files.length} ファイルの ${v.hits} 行が当たりました` + (v.out ? `（作ったコンフィグは ${v.out}）` : "")),
        button("結果をコピー", () => api().ClipboardSet(text).then(() => toast("コピーしました")).catch((e) => toast(errText(e), "err")), { disabled: !lines.length }),
        button("テキストに保存", () => api().SearchSave(text).then((p) => { if (p) toast("保存しました: " + p); }).catch((e) => toast(errText(e), "err")), { disabled: !lines.length })));
      if (v.capped) out.push(h("div", { class: "box warn" }, "当たった行が多いため、先頭の 5000 行だけを出しています。探す文字を絞ってください。"));
      out.push(boxes("error", v.errors));
      for (const f of v.files) {
        out.push(h("div", { class: "srch-file" },
          h("div", { class: "srch-head" }, h("b", null, f.rel), h("span", { class: "dim small" }, ` ${f.scope}・${f.hits.length} 行`)),
          f.hits.map((hit) => h("div", { class: "srch-hit", onclick: () => openFile(f.rel, hit.line) },
            h("span", { class: "no" }, String(hit.line)), h("span", { class: "tx" }, marked(hit.text, hit.spans))))));
      }
      if (v.missed.length) out.push(h("h2", null, `当たらなかったファイル（${v.missed.length}）`), h("div", { class: "box" }, v.missed.join("\n")));
    }
    return out;
  }

  // openFile shows a file of the case at a line (read only).
  function openFile(rel, line) {
    run("ファイルを開いています", async () => {
      const f = await api().ReadFile(rel);
      const sr = st.srch;
      sr.file = { rel: f.rel, lines: f.lines, cut: f.cut, at: line };
      sr.fq = sr.regex ? "" : sr.q;
      srchFind(sr);
      sr.fcur = sr.fhits.indexOf(line - 1);
    });
  }

  const VIEW_LINES = 20000; // lines of a file drawn around where it is read

  // srchFind marks the lines of the open file that hold what is looked for.
  function srchFind(sr) {
    sr.fhits = [];
    const w = sr.fq.toLowerCase();
    if (!w) return;
    sr.file.lines.forEach((l, i) => { if (l.toLowerCase().includes(w)) sr.fhits.push(i); });
  }

  function srchJump(dir) {
    const sr = st.srch;
    if (!sr.file || !sr.fhits.length) return;
    sr.fcur = sr.fcur < 0 ? (dir > 0 ? 0 : sr.fhits.length - 1) : (sr.fcur + dir + sr.fhits.length) % sr.fhits.length;
    sr.file.at = sr.fhits[sr.fcur] + 1;
    render();
  }

  function fileViewer(sr) {
    const f = sr.file;
    const w = sr.fq.toLowerCase();
    const from = Math.max(0, f.at - 1 - VIEW_LINES / 2);
    const to = Math.min(f.lines.length, from + VIEW_LINES);
    const spans = (l) => {
      if (!w) return [];
      const out = [];
      const low = l.toLowerCase();
      for (let i = low.indexOf(w); i >= 0; i = low.indexOf(w, i + w.length)) out.push([i, i + w.length]);
      return out;
    };
    const rows = [];
    for (let i = from; i < to; i++) {
      rows.push(h("div", { class: cls("fv-line", i === f.at - 1 && "at"), id: i === f.at - 1 ? "fv-at" : null },
        h("span", { class: "no" }, String(i + 1)), h("span", { class: "tx" }, marked(f.lines[i], spans(f.lines[i])))));
    }
    afterDraw.push(() => { const at = $("fv-at"); if (at) at.scrollIntoView({ block: "center" }); });
    const find = h("input", { type: "text", value: sr.fq, placeholder: "このファイルの中を検索", oninput: (e) => { sr.fq = e.target.value; },
      onkeydown: (e) => { if (e.key === "Enter" && !e.isComposing) { srchFind(sr); sr.fcur = -1; srchJump(e.shiftKey ? -1 : 1); render(); } } });
    return [
      h("h1", null, "検索"),
      h("div", { class: "row" }, button("← 結果に戻る", () => { sr.file = null; render(); }), h("b", { class: "path" }, f.rel), h("span", { class: "dim small" }, `${f.lines.length} 行`)),
      f.cut ? h("div", { class: "box warn" }, "長いファイルのため、先頭の 200000 行だけを読んでいます。") : null,
      h("div", { class: "row" }, find,
        button("前 ▲", () => srchJump(-1), { title: "Shift+F3" }), button("次 ▼", () => srchJump(1), { title: "F3" }),
        h("span", { class: "dim small" }, sr.fhits.length ? `${sr.fcur >= 0 ? sr.fcur + 1 : 0} / ${sr.fhits.length} か所` : (sr.fq ? "当たりはありません" : ""))),
      from > 0 || to < f.lines.length ? h("p", { class: "dim small" }, `${from + 1} 行目から ${to} 行目を出しています。`) : null,
      h("div", { class: "diff fv" }, rows),
    ];
  }

  // ---- バージョン情報 (spec 14.8.1; PalaTerm's バージョン情報) ----

  function aboutPage() {
    const u = st.upd;
    const i = u.found;
    const check = async () => {
      const was = u;
      st.upd = { found: null, busy: "check", error: "", path: "" };
      render();
      try {
        st.upd.found = await api().CheckLatest();
      } catch (e) {
        st.upd.error = errText(e);
      }
      // The exe downloaded before this check is still the latest: its steps
      // stay on the page.
      if (st.upd.found && was.found && was.path && st.upd.found.latest === was.found.latest) st.upd.path = was.path;
      st.upd.busy = "";
      render();
    };
    const download = async () => {
      const mb = (i.exeSize / 1048576).toFixed(1);
      const ok = await ask("最新版のダウンロード",
        `ConfigChief v${i.latest} の ConfigChief.exe（${mb} MB）を GitHub の公開リリースからダウンロードして、${i.saveDir} に保存します。よろしいですか？\n\n`
        + (i.hasDigest ? "保存したファイルは、リリースに登録された SHA-256 と照合します。" : "このリリースには SHA-256 の登録がないため、保存したファイルは大きさだけ照合します。")
        + "今の ConfigChief は置き換えず、起動もしません。", "ダウンロードする");
      if (!ok) return;
      u.busy = "dl";
      u.error = "";
      u.path = "";
      render();
      try {
        u.path = await api().DownloadLatest();
        u.swap = await api().CanSwap();
        toast("ダウンロードしました");
      } catch (e) {
        u.error = errText(e);
      }
      u.busy = "";
      render();
    };
    const showErr = (e) => toast(errText(e), "err");
    // swap puts the downloaded exe in the place of this one and starts it
    // (spec 14.11.1).
    const swap = async () => {
      const ok = await ask("入れ替えて起動し直す",
        `ConfigChief を閉じて、v${i.latest} で起動し直します。今の exe は ConfigChief-v${i.current}.exe の名前で同じフォルダに残します（戻すときは名前を戻します）。`
        + (st.cas ? "開いている案件は、新しい版で開き直します。表の編集の下書きは案件に残っています。" : ""), "入れ替える");
      if (!ok) return;
      u.busy = "swap";
      u.error = "";
      render();
      try {
        await api().SwapAndRestart();
        toast("新しい版を起動しました。この窓は閉じます");
      } catch (e) {
        u.error = errText(e);
        u.busy = "";
        try { setState((await api().Start()).state); } catch (x) { /* the state stays */ }
        render();
      }
    };
    const out = [
      h("h1", null, "バージョン情報"),
      h("p", { class: "dim" }, "GitHub の公開リリースと、この ConfigChief の版を比べます。", h("b", null, "ボタンを押したときだけ通信します"), "（送るのは GitHub への問い合わせだけで、案件の中身は送りません）。 ",
        tip("通信するのは「最新版を確認する」と「最新版をダウンロード」の 2 つだけです。プロキシは環境変数 HTTPS_PROXY か、Windows の設定に手動で入れたものを使います。自動設定のスクリプト（PAC）と認証の要るプロキシには対応していません。")),
      h("div", { class: "card about" },
        h("div", { class: "kv" },
          h("span", { class: "dim" }, "この ConfigChief"), h("b", null, "v" + (i ? i.current : st.version)),
          i ? [h("span", { class: "dim" }, "最新版"), h("span", null, h("b", null, "v" + i.latest), i.published ? h("span", { class: "dim small" }, `（${i.published} 公開）`) : null)] : null),
        i && !i.newer ? h("div", { class: "box info" }, "お使いの版は最新です。") : null,
        i && i.newer ? h("div", { class: "box upd-new" },
          h("div", { class: "upd-title" }, `新しい版があります（v${i.current} → v${i.latest}）`),
          !u.path ? (i.hasExe
            ? [h("p", { class: "dim small" }, `最新版の ConfigChief.exe を ${i.saveDir} に保存します（今の ConfigChief はそのまま動きます）。`),
              button(u.busy === "dl" ? "ダウンロード中…" : `⬇ 最新版をダウンロード（${(i.exeSize / 1048576).toFixed(1)} MB）`, download, { class: "primary", id: "upd-dl", disabled: !!u.busy })]
            : h("p", { class: "dim small" }, "最新版に ConfigChief.exe が添付されていません。リリースのページから入手してください。"))
            : [h("p", null, "✓ ダウンロードしました: " + u.path),
              u.swap ? h("div", { class: "row" },
                button("入れ替えて起動し直す", swap, { class: "primary", id: "upd-swap", disabled: !!u.busy }),
                tip(`今の exe は ConfigChief-v${i.current}.exe の名前で同じフォルダに残します（戻すときは名前を戻します）。開いている案件は、新しい版で開き直します`)) : null,
              h("div", { class: "section-label" }, u.swap ? "手で入れ替えるときの手順" : "あとの手順"),
              h("ol", null,
                h("li", null, "ConfigChief の窓を閉じます。"),
                h("li", null, i.inAppDir ? "今の ConfigChief.exe を消してから、ダウンロードしたファイルの名前を ConfigChief.exe に変えます（同じフォルダにあります）。"
                  : "ダウンロードしたファイルの名前を ConfigChief.exe に変えて、今の ConfigChief.exe に上書きします。"),
                h("li", null, "ConfigChief.exe を起動します。案件フォルダと設定はそのまま使えます。")),
              h("div", { class: "row" },
                button("ダウンロードしたフォルダを開く", () => api().ShowDownloaded().catch(showErr), { class: "primary" }),
                i.inAppDir ? null : button("今の ConfigChief のフォルダを開く", () => api().ShowAppFolder().catch(showErr)))]) : null,
        u.error ? h("div", { class: "box error" }, u.error) : null,
        h("div", { class: "row" },
          button(u.busy === "check" ? "確認中…" : i ? "もう一度確認する" : "最新版を確認する", check, { class: i ? null : "primary", id: "upd-check", disabled: !!u.busy }),
          i || u.error ? button("変更点をブラウザで見る", () => api().OpenWeb(i ? "notes:" + i.latest : "release"), { title: "最新版のリリースのページを、既定のブラウザで開きます" }) : null),
        i && i.notes.length ? [
          h("div", { class: "section-label" }, `この版から最新版までの変更点（${i.notes.length} 版分・新しい順）`),
          h("div", { class: "upd-notes" }, i.notes.map((n) => h("div", { class: "upd-note" },
            h("div", { class: "upd-note-h" }, h("b", null, "v" + n.version), h("span", { class: "dim small" }, n.published || "")),
            noteBody(n.body))))] : null),
      h("div", { class: "card about-card" },
        h("div", { class: "brand-big" }, "ConfigChief"),
        h("div", { class: "dim" }, "Network device configurations ⇄ a vendor-neutral Excel workbook"),
        h("p", null, "作者: ", h("span", { id: "ab-author", onclick: (e) => { if (e.detail === 3) authorHello(); } }, "KJO"), h("br"), "MIT License © 2026 KJO"),
        h("div", { class: "row" },
          button("GitHub", () => api().OpenWeb("github"), { class: "link" }),
          button("使い方", () => api().OpenWeb("manual"), { class: "link" }),
          button("申請用の説明", () => api().OpenWeb("approval"), { class: "link", title: "使う許可の申請に使う 1 枚の説明（通信・書き込む場所・インストールしないこと）" }))),
    ];
    return out;
  }

  // authorHello pops a hello up over the page, as PalaTerm does on three
  // clicks on the author's name (spec 14.19.2); a click anywhere or Esc puts
  // it away.
  function authorHello() {
    if (document.querySelector(".author-pop")) return;
    const ov = h("div", { class: "author-pop" }, h("div", { class: "author-hello" }, "It's-a me, Kojio! 🎉"));
    document.body.appendChild(ov);
    const done = () => { ov.remove(); document.removeEventListener("keydown", onKey, true); };
    const onKey = (e) => { if (e.key === "Escape") { e.preventDefault(); e.stopImmediatePropagation(); done(); } };
    document.addEventListener("keydown", onKey, true);
    ov.onclick = done;
  }

  // noteBody draws the Markdown of release notes plainly: **bold** and
  // `code` kept, "- " lines as items, "#" lines as headings.
  function noteBody(md) {
    return h("div", { class: "upd-note-b" }, String(md || "").split(/\r?\n/).map((l) => {
      const parts = [];
      const re = /\*\*(.+?)\*\*|`([^`]+)`/g;
      let at = 0;
      let m;
      const text = l.replace(/^\s*[-*] /, "").replace(/^#+ /, "");
      while ((m = re.exec(text))) {
        if (m.index > at) parts.push(text.slice(at, m.index));
        parts.push(m[1] != null ? h("b", null, m[1]) : h("code", null, m[2]));
        at = re.lastIndex;
      }
      if (at < text.length) parts.push(text.slice(at));
      if (/^\s*[-*] /.test(l)) return h("div", { class: "upd-li" }, parts);
      if (/^#+ /.test(l)) return h("div", { class: "upd-sub" }, parts);
      return l.trim() ? h("div", null, parts) : h("div", { class: "upd-gap" });
    }));
  }

  // ---- the start ----

  async function boot() {
    $("busy-stop").addEventListener("click", () => {
      if (bookWait) { // waiting for Excel: stop waiting, nothing is written
        bookWait.stop = true;
        $("busy-stop").disabled = true;
        return;
      }
      $("busy-stop").disabled = true;
      $("busy-text").textContent = "止めています…";
      api().Cancel();
    });
    // The table of the book: dragging a selection, and its size.
    document.addEventListener("mousemove", gMouseMove);
    document.addEventListener("mouseup", gMouseUp);
    document.addEventListener("mousedown", (e) => { if (!(e.target instanceof Element) || !e.target.closest(".g-menu, .g-hpop")) gCloseMenus(); }, true);
    document.addEventListener("keydown", gDocKey, true);
    window.addEventListener("resize", () => { gCloseMenus(); gFit(); });
    document.addEventListener("keydown", zoomKey, true);
    document.addEventListener("wheel", zoomWheel, { passive: false });
    document.addEventListener("keydown", (e) => {
      if (st.page === "compare" && st.diff.view && st.diff.layout === "side" && (e.key === "F7" || e.key === "F8")) {
        e.preventDefault();
        diffJump(e.key === "F7" ? 1 : -1);
      }
      if (st.page === "search" && st.srch.file && e.key === "F3") {
        e.preventDefault();
        srchJump(e.shiftKey ? -1 : 1);
      }
    });
    try {
      window.runtime.EventsOn("progress", onProgress);
      window.runtime.EventsOn("internal-error", (m) => toast(m, "err"));
      // Files dropped on the places that take them (spec 14.10.1).
      if (window.runtime.OnFileDrop) window.runtime.OnFileDrop(onFileDrop, true);
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
