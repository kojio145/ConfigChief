package tmpl

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	mj "github.com/mitsuhiko/minijinja/minijinja-go/v3"
	mjsyntax "github.com/mitsuhiko/minijinja/minijinja-go/v3/syntax"
	mjv "github.com/mitsuhiko/minijinja/minijinja-go/v3/value"

	"github.com/kojio145/configchief/internal/netx"
)

// Record is a value with named fields, like a row of a table. The engine asks
// for a field every time an expression reads it, so the owner can keep track
// of what was read.
type Record interface {
	Field(name string) (any, bool)
	Fields() []string
}

// Func is a function that expressions can call: lookup('table', name=x).
type Func func(args []any, kwargs map[string]any) (any, error)

// Options are the pack's parts of the filters. A nil function makes its
// filter an error.
type Options struct {
	IfName  func(string) (string, error)
	IfShort func(string) (string, error)
	Lex     func(kind, word string) (string, error)
	Unlex   func(kind, word string) (string, error)
}

// Engine runs Jinja2 templates and expressions. The engine behind it
// (minijinja-go) is not visible to the callers. Not safe for concurrent use.
type Engine struct {
	env      *mj.Environment
	mu       sync.Mutex
	captured mjv.Value
	cache    map[string]*mj.Template
	kind     string // what the filter that failed last was: "required", "lex"
	// fails counts the times a filter or a function stopped the run, and
	// skip is how many of them are passed over (see Rest).
	fails, skip int
}

// Error is a failed template or expression.
type Error struct {
	Msg  string
	Name string // template name, if known
	Line int    // 1-based line in the template; 0 if not known
	// Kind tells the failures a caller treats apart: "required" (the filter
	// found an empty value), "lex" (a word has no counterpart).
	Kind string
	// Stopped tells that a filter or a function ended the template (fail,
	// required, a value a filter does not take): Rest can look past it.
	Stopped bool
}

func (e *Error) Error() string { return e.Msg }

// New makes an engine with the common filters and functions (appendix B 5.3).
func New(opt Options) *Engine {
	env := mj.NewEnvironment()
	env.SetWhitespace(mjsyntax.WhitespaceConfig{TrimBlocks: true, LstripBlocks: true, KeepTrailingNewline: true})
	env.SetUndefinedBehavior(mj.UndefinedStrict)
	env.SetDebug(true) // without it, runtime errors carry no line
	// The output is a configuration, whatever the name of the template ends in.
	env.SetAutoEscapeFunc(func(string) mj.AutoEscape { return mj.AutoEscapeNone })
	e := &Engine{env: env, cache: map[string]*mj.Template{}}

	str1 := func(name string, f func(string) (string, error)) {
		env.AddFilter(name, func(_ mj.FilterState, v mjv.Value, args []mjv.Value, _ map[string]mjv.Value) (mjv.Value, error) {
			if len(args) != 0 {
				return e.stop("%s に引数は要りません", name)
			}
			if f == nil {
				return e.stop("このパックでは %s を使えません", name)
			}
			s, err := f(text(v))
			if err != nil {
				return e.stop("%s", err.Error())
			}
			return mjv.FromString(s), nil
		})
	}
	str2 := func(name string, f func(string, string) (string, error)) {
		env.AddFilter(name, func(_ mj.FilterState, v mjv.Value, args []mjv.Value, _ map[string]mjv.Value) (mjv.Value, error) {
			if len(args) != 1 {
				return e.stop("%s には引数が 1 つ要ります", name)
			}
			if f == nil {
				return e.stop("このパックでは %s を使えません", name)
			}
			s, err := f(text(v), text(args[0]))
			if err != nil {
				return e.stop("%s", err.Error())
			}
			return mjv.FromString(s), nil
		})
	}
	str1("ip", netx.IP)
	str1("mask", netx.Mask)
	str1("wildcard", netx.Wildcard)
	str1("network", netx.Network)
	str1("broadcast", netx.Broadcast)
	str1("ifname", opt.IfName)
	str1("ifshort", opt.IfShort)
	str2("with_mask", netx.WithMask)
	str2("with_wildcard", netx.WithWildcard)
	// The kind comes second in the template: {{ word | lex('encryption') }}.
	swap := func(f func(kind, word string) (string, error)) func(string, string) (string, error) {
		if f == nil {
			return nil
		}
		return func(word, kind string) (string, error) {
			out, err := f(kind, word)
			if err != nil {
				e.kind = "lex"
			}
			return out, err
		}
	}
	str2("lex", swap(opt.Lex))
	str2("unlex", swap(opt.Unlex))

	env.AddFilter("prefixlen", func(_ mj.FilterState, v mjv.Value, _ []mjv.Value, _ map[string]mjv.Value) (mjv.Value, error) {
		n, err := netx.PrefixLen(text(v))
		if err != nil {
			return e.stop("%s", err.Error())
		}
		return mjv.FromInt(int64(n)), nil
	})
	env.AddFilter("host", func(_ mj.FilterState, v mjv.Value, args []mjv.Value, _ map[string]mjv.Value) (mjv.Value, error) {
		if len(args) != 1 {
			return e.stop("host には番号が 1 つ要ります")
		}
		n, err := strconv.Atoi(text(args[0]))
		if err != nil {
			return e.stop("host の番号「%s」が数として読めません", text(args[0]))
		}
		s, err := netx.Host(text(v), n)
		if err != nil {
			return e.stop("%s", err.Error())
		}
		return mjv.FromString(s), nil
	})
	env.AddFilter("in_net", func(_ mj.FilterState, v mjv.Value, args []mjv.Value, _ map[string]mjv.Value) (mjv.Value, error) {
		if len(args) != 1 {
			return e.stop("in_net にはネットワークが 1 つ要ります")
		}
		ok, err := netx.InNet(text(v), text(args[0]))
		if err != nil {
			return e.stop("%s", err.Error())
		}
		return mjv.FromBool(ok), nil
	})
	env.AddFilter("required", func(_ mj.FilterState, v mjv.Value, args []mjv.Value, _ map[string]mjv.Value) (mjv.Value, error) {
		if v.IsUndefined() || v.IsNone() || text(v) == "" {
			what := "値"
			if len(args) > 0 {
				what = text(args[0])
			}
			e.kind = "required"
			return e.stop("%s が空です", what)
		}
		return v, nil
	})
	env.AddFilter("version_ge", func(_ mj.FilterState, v mjv.Value, args []mjv.Value, _ map[string]mjv.Value) (mjv.Value, error) {
		if len(args) != 1 {
			return e.stop("version_ge には版が 1 つ要ります")
		}
		return mjv.FromBool(compareVersions(text(v), text(args[0])) >= 0), nil
	})
	// With attribute='name', rows are put in the order of that column.
	env.AddFilter("natural", func(_ mj.FilterState, v mjv.Value, _ []mjv.Value, kw map[string]mjv.Value) (mjv.Value, error) {
		items := append([]mjv.Value{}, v.Iter()...)
		key := func(x mjv.Value) string { return text(x) }
		if attr, ok := kw["attribute"]; ok {
			name := text(attr)
			key = func(x mjv.Value) string { return text(x.GetAttr(name)) }
		}
		sort.SliceStable(items, func(i, j int) bool { return NaturalLess(key(items[i]), key(items[j])) })
		return mjv.FromSlice(items), nil
	})
	// {{ p.allowed_vlans | ranges }}: "10,20,30-32" (or a list of such items)
	// as the numbers it stands for, in order and once each. Words (all, none)
	// stand for no number.
	env.AddFilter("ranges", func(_ mj.FilterState, v mjv.Value, _ []mjv.Value, _ map[string]mjv.Value) (mjv.Value, error) {
		var items []string
		if l, ok := v.AsSlice(); ok {
			for _, x := range l {
				items = append(items, text(x))
			}
		} else {
			items = []string{text(v)}
		}
		nums, err := Ranges(items)
		if err != nil {
			return e.stop("%s", err.Error())
		}
		out := make([]mjv.Value, len(nums))
		for i, n := range nums {
			out[i] = mjv.FromInt(int64(n))
		}
		return mjv.FromSlice(out), nil
	})
	// {% if i.ipv4 is cidr %}: an address with a length, not a word like dhcp.
	env.AddTest("cidr", func(_ mj.TestState, v mjv.Value, _ []mjv.Value) (bool, error) {
		s, ok := v.AsString()
		if !ok || !strings.Contains(s, "/") {
			return false, nil
		}
		_, err := netx.PrefixLen(s)
		return err == nil, nil
	})
	// {% if v is digits %}: nothing but digits, and at least one.
	env.AddTest("digits", func(_ mj.TestState, v mjv.Value, _ []mjv.Value) (bool, error) {
		s := text(v)
		return s != "" && strings.IndexFunc(s, notDigit) < 0, nil
	})
	env.AddFunction("fail", func(_ *mj.State, args []mjv.Value, _ map[string]mjv.Value) (mjv.Value, error) {
		if len(args) == 0 {
			return e.stop("fail")
		}
		return e.stop("%s", text(args[0]))
	})
	env.AddFunction("capture", func(_ *mj.State, args []mjv.Value, _ map[string]mjv.Value) (mjv.Value, error) {
		if len(args) == 1 {
			e.captured = args[0]
		}
		return mjv.FromString(""), nil
	})
	return e
}

// SetFunc adds a function that expressions can call.
func (e *Engine) SetFunc(name string, f Func) {
	e.env.AddFunction(name, func(_ *mj.State, args []mjv.Value, kwargs map[string]mjv.Value) (mjv.Value, error) {
		a := make([]any, len(args))
		for i, v := range args {
			a[i] = fromValue(v)
		}
		k := make(map[string]any, len(kwargs))
		for n, v := range kwargs {
			k[n] = fromValue(v)
		}
		out, err := f(a, k)
		if err != nil {
			return e.stop("%s", err.Error())
		}
		return toValue(out), nil
	})
}

// Expand runs a short template, like a value in an import rule:
// "{{ ip | with_mask(mask) }}". A text without tags comes back as is.
func (e *Engine) Expand(src string, vars map[string]any) (string, error) {
	if !strings.Contains(src, "{{") && !strings.Contains(src, "{%") && !strings.Contains(src, "{#") {
		return src, nil
	}
	if s, ok := copied(src, vars); ok {
		return s, nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	tpl, err := e.compile("", src)
	if err != nil {
		return "", err
	}
	out, err := tpl.Render(named(vars))
	if err != nil {
		return "", wrap(err)
	}
	return out, nil
}

// copied gives the value of a template that only names a text that is there:
// "{{ name }}", or "{{ name.field }}" of a map of texts. Most values of the
// import rules are of this kind, and a configuration of some hundred lines
// asks for thousands of them: the engine is not needed to copy a text
// (spec 16). Everything else (a value that is no text, a name that is not
// there, a filter) is left to the engine, which says what is wrong with it.
func copied(src string, vars map[string]any) (string, bool) {
	if len(src) < 5 || src[:2] != "{{" || src[len(src)-2:] != "}}" {
		return "", false
	}
	in := strings.Trim(src[2:len(src)-2], " ")
	dot := -1
	for i := 0; i < len(in); i++ {
		c := in[i]
		switch {
		case c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
			if i == 0 || i == dot+1 {
				return "", false
			}
		case c == '.' && dot < 0 && i > 0 && i < len(in)-1:
			dot = i
		default:
			return "", false
		}
	}
	name := in
	if dot >= 0 {
		name = in[:dot]
	}
	switch name {
	case "", "true", "false", "none", "True", "False", "None":
		return "", false
	}
	switch v := vars[name].(type) {
	case string:
		if dot < 0 {
			return v, true
		}
	case map[string]string:
		if dot >= 0 {
			s, ok := v[in[dot+1:]]
			return s, ok
		}
	}
	return "", false
}

// Eval gives the value of one expression: a `when`, or a `let` of a derive.
// The result is nil, bool, int64, float64, string, []any, map[string]any or
// the Record that went in.
func (e *Engine) Eval(expr string, vars map[string]any) (any, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	tpl, err := e.compile("", "{{ capture("+expr+") }}")
	if err != nil {
		return nil, err
	}
	e.captured = mjv.Undefined()
	if _, err := tpl.Render(named(vars)); err != nil {
		return nil, wrap(err)
	}
	return fromValue(e.captured), nil
}

// SetLoader tells where the templates that Render and {% include %} name come
// from. The names are used as written: there are no folders to join.
func (e *Engine) SetLoader(load func(name string) (string, bool)) {
	e.env.SetLoader(func(name string) (string, error) {
		if src, ok := load(name); ok {
			return src, nil
		}
		return "", fmt.Errorf("テンプレート %s がありません", name)
	})
	e.env.SetPathJoinCallback(func(name, _ string) string { return name })
}

// Piece is a part of the output of Render with the place in the templates
// that wrote it.
type Piece struct {
	Start, End int    // bytes of the output
	Name       string // the template
	Line       int    // the line in the template, from 1
	Expr       bool   // the value of an expression; false = text of the template
}

// writer hands every write of the engine to the caller of Render.
type writer struct {
	sb      strings.Builder
	onWrite func(start, end int)
}

func (w *writer) Write(p []byte) (int, error) {
	start := w.sb.Len()
	w.sb.Write(p)
	if w.onWrite != nil && len(p) > 0 {
		w.onWrite(start, w.sb.Len())
	}
	return len(p), nil
}

// Render runs a template of the loader. onWrite, if set, is called after each
// part of the output is written, in order: what the Records were asked
// between two calls is what that part was made from. When the template
// fails, the text written so far comes back with the error.
func (e *Engine) Render(name string, vars map[string]any, onWrite func(start, end int)) (string, []Piece, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.kind, e.fails, e.skip = "", 0, 0
	tpl, err := e.env.GetTemplate(name)
	if err != nil {
		return "", nil, e.wrapKind(err)
	}
	ctx := toValue(vars)
	w := &writer{onWrite: onWrite}
	if err := tpl.RenderToWrite(ctx, w); err != nil {
		return w.sb.String(), nil, e.wrapKind(err)
	}
	// The same run again, for the places in the templates: the engine gives
	// them only without a writer of the caller.
	out, spans, err := tpl.RenderWithSpans(ctx)
	if err != nil || out != w.sb.String() {
		return w.sb.String(), nil, nil
	}
	pieces := make([]Piece, len(spans))
	for i, s := range spans {
		pieces[i] = Piece{Start: s.OutputStart, End: s.OutputEnd, Name: s.Name, Line: s.Line(), Expr: s.Kind == mj.OutputSpanExpression}
	}
	return out, pieces, nil
}

func (e *Engine) wrapKind(err error) error {
	out := wrap(err)
	if te, ok := out.(*Error); ok {
		te.Kind = e.kind
		te.Stopped = e.fails > e.skip
	}
	return out
}

// Rest gives the failures of a template after its first one, at most max of
// them, and tells whether there are more. The template runs again and again,
// each time passing one more of the places where a filter or a function
// stopped it: there they give an empty value. Only such places are told; an
// error of another kind after a place that was passed may come from the empty
// value, and ends the search.
func (e *Engine) Rest(name string, vars map[string]any, max int) (rest []*Error, more bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	tpl, err := e.env.GetTemplate(name)
	if err != nil {
		return nil, false
	}
	defer func() { e.skip = 0 }()
	ctx := toValue(vars)
	for {
		e.kind, e.fails, e.skip = "", 0, len(rest)+1
		err := tpl.RenderToWrite(ctx, &writer{})
		if err == nil {
			return rest, false
		}
		te, ok := e.wrapKind(err).(*Error)
		if !ok || !te.Stopped {
			return rest, false
		}
		if len(rest) == max {
			return rest, true
		}
		rest = append(rest, te)
	}
}

// stop is how a filter or a function ends a template: with an error, or, at
// the places Rest passes, with an empty value.
func (e *Engine) stop(format string, a ...any) (mjv.Value, error) {
	e.fails++
	if e.fails <= e.skip {
		e.kind = ""
		return mjv.FromString(""), nil
	}
	return fail(format, a...)
}

// Check compiles a template without running it: nil, or what is wrong in it.
func (e *Engine) Check(name, src string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err := e.compile(name, src)
	return err
}

// Truth tells whether an Eval result counts as true in a condition.
func Truth(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case int64:
		return x != 0
	case float64:
		return x != 0
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	case Record:
		// Like the engine: a record without fields (a lookup that found
		// nothing) is false.
		return len(x.Fields()) > 0
	}
	return true
}

func (e *Engine) compile(name, src string) (*mj.Template, error) {
	if t, ok := e.cache[name+"\x00"+src]; ok {
		return t, nil
	}
	var t *mj.Template
	var err error
	if name == "" {
		t, err = e.env.TemplateFromString(src)
	} else {
		t, err = e.env.TemplateFromNamedString(name, src)
	}
	if err != nil {
		return nil, wrap(err)
	}
	e.cache[name+"\x00"+src] = t
	return t, nil
}

func fail(format string, a ...any) (mjv.Value, error) {
	// A plain error loses its line; the engine's own error type keeps it.
	return mjv.Undefined(), mj.NewError(mj.ErrInvalidOperation, fmt.Sprintf(format, a...))
}

func wrap(err error) error {
	var me *mj.Error
	if errors.As(err, &me) {
		// What fails inside an included template comes wrapped in an error
		// of the template that includes it; the one inside tells what is
		// wrong, and where.
		for me.Kind == mj.ErrBadInclude {
			var in *mj.Error
			if cause := me.Unwrap(); cause == nil || !errors.As(cause, &in) {
				break
			}
			me = in
		}
		out := &Error{Msg: me.Message, Name: me.Name}
		if me.Kind != mj.ErrInvalidOperation {
			out.Msg = fmt.Sprintf("%s: %s", me.Kind, me.Message)
		}
		if me.Span != nil {
			out.Line = int(me.Span.StartLine)
		}
		return out
	}
	return &Error{Msg: err.Error()}
}

// text gives a value as the string a rule or a filter works on.
func text(v mjv.Value) string {
	if v.IsUndefined() || v.IsNone() {
		return ""
	}
	return v.String()
}

// record is how a Record looks to the engine: a map whose reads go through
// Field.
// names hands the names of a short expression to the engine as they are
// asked for. An import rule has a dozen names and an expression reads one or
// two of them; made ready all at once, and copied by the engine for every
// run, they take more time than the expression itself (spec 16).
type names struct {
	vars map[string]any
	seen map[string]mjv.Value // what was made ready, so that a name read twice is the same value
}

// named gives the names for one run of a short expression.
func named(vars map[string]any) mjv.Value {
	return mjv.FromObject(&names{vars: vars})
}

func (o *names) GetAttr(name string) mjv.Value {
	if v, ok := o.seen[name]; ok {
		return v
	}
	x, ok := o.vars[name]
	if !ok {
		return mjv.Undefined()
	}
	v := toValue(x)
	if o.seen == nil {
		o.seen = map[string]mjv.Value{}
	}
	o.seen[name] = v
	return v
}

func (o *names) Keys() []string {
	out := make([]string, 0, len(o.vars))
	for k := range o.vars {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (o *names) ObjectRepr() mjv.ObjectRepr { return mjv.ObjectReprMap }

type record struct{ r Record }

func (o record) GetAttr(name string) mjv.Value {
	v, ok := o.r.Field(name)
	if !ok {
		return mjv.Undefined()
	}
	return toValue(v)
}

func (o record) Keys() []string { return o.r.Fields() }

// Prepare makes a value ready for the templates once, for a value that many
// runs take: handing it to Render, Expand or Eval then costs nothing more.
// What comes back is good only for handing on.
func Prepare(v any) any { return toValue(v) }

func toValue(v any) mjv.Value {
	switch x := v.(type) {
	case nil:
		return mjv.None()
	case mjv.Value:
		return x
	case Record:
		return mjv.FromObject(record{x})
	case string:
		return mjv.FromString(x)
	case bool:
		return mjv.FromBool(x)
	case int:
		return mjv.FromInt(int64(x))
	case int64:
		return mjv.FromInt(x)
	case float64:
		return mjv.FromFloat(x)
	case []string:
		out := make([]mjv.Value, len(x))
		for i, s := range x {
			out[i] = mjv.FromString(s)
		}
		return mjv.FromSlice(out)
	case []any:
		out := make([]mjv.Value, len(x))
		for i, s := range x {
			out[i] = toValue(s)
		}
		return mjv.FromSlice(out)
	case map[string]string:
		out := make(map[string]mjv.Value, len(x))
		for k, s := range x {
			out[k] = mjv.FromString(s)
		}
		return mjv.FromMap(out)
	case map[string]any:
		out := make(map[string]mjv.Value, len(x))
		for k, s := range x {
			out[k] = toValue(s)
		}
		return mjv.FromMap(out)
	}
	return mjv.FromAny(v)
}

func fromValue(v mjv.Value) any {
	if v.IsUndefined() || v.IsNone() {
		return nil
	}
	if o, ok := v.AsObject(); ok {
		if r, ok := o.(record); ok {
			return r.r
		}
	}
	if b, ok := v.AsBool(); ok {
		return b
	}
	if s, ok := v.AsString(); ok {
		return s
	}
	if v.IsActualInt() {
		n, _ := v.AsInt()
		return n
	}
	if v.IsActualFloat() {
		f, _ := v.AsFloat()
		return f
	}
	if m, ok := v.AsMap(); ok {
		out := make(map[string]any, len(m))
		for k, x := range m {
			out[k] = fromValue(x)
		}
		return out
	}
	if l, ok := v.AsSlice(); ok {
		out := make([]any, len(l))
		for i, x := range l {
			out[i] = fromValue(x)
		}
		return out
	}
	return v.String()
}

// compareVersions compares dotted versions number by number: 16.10 > 16.9.
func compareVersions(a, b string) int {
	as, bs := strings.FieldsFunc(a, notDigit), strings.FieldsFunc(b, notDigit)
	for i := 0; i < len(as) || i < len(bs); i++ {
		var x, y int
		if i < len(as) {
			x, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			y, _ = strconv.Atoi(bs[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

func notDigit(r rune) bool { return r < '0' || r > '9' }

// NaturalLess orders "Gi0/0/2" before "Gi0/0/10".
func NaturalLess(a, b string) bool {
	for a != "" && b != "" {
		ad, bd := digits(a), digits(b)
		if ad > 0 && bd > 0 {
			x, _ := strconv.ParseUint(a[:ad], 10, 64)
			y, _ := strconv.ParseUint(b[:bd], 10, 64)
			if x != y {
				return x < y
			}
			a, b = a[ad:], b[bd:]
			continue
		}
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		a, b = a[1:], b[1:]
	}
	return len(a) < len(b)
}

func digits(s string) int {
	n := 0
	for n < len(s) && s[n] >= '0' && s[n] <= '9' {
		n++
	}
	return n
}

// Ranges gives the numbers that a list such as "10,20,30-32" stands for, in
// order and once each. The items may be split by commas or spaces. A word
// (all, none) stands for no number; a range may hold 4094 numbers at most.
func Ranges(items []string) ([]int, error) {
	seen := map[int]bool{}
	for _, item := range items {
		for _, part := range strings.FieldsFunc(item, func(r rune) bool { return r == ',' || r == ' ' }) {
			ends := strings.SplitN(part, "-", 2)
			lo, err := strconv.Atoi(ends[0])
			if err != nil {
				if len(ends) == 1 && strings.IndexFunc(part, notDigit) >= 0 && strings.IndexFunc(part, func(r rune) bool { return r >= '0' && r <= '9' }) < 0 {
					continue
				}
				return nil, fmt.Errorf("「%s」は、番号でも番号の範囲でもありません", part)
			}
			hi := lo
			if len(ends) == 2 {
				if hi, err = strconv.Atoi(ends[1]); err != nil || hi < lo || hi-lo > 4094 {
					return nil, fmt.Errorf("「%s」は、番号の範囲として読めません", part)
				}
			}
			for n := lo; n <= hi; n++ {
				seen[n] = true
			}
		}
	}
	out := make([]int, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Ints(out)
	return out, nil
}
