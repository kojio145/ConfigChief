package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// The pages of the project site (GitHub Pages, without Jekyll: the documents
// hold template examples in {{ }} and {% %} that Jekyll would run) are made
// from the Markdown of docs/. Only what these documents use is read:
// headings, paragraphs, lists (one level of nesting), tables, fenced code,
// quotes, and in a line `code`, **bold**, [links](…), ![pictures](…) and
// <a id="…"></a>. A link to a .md file of docs/ becomes the .html page.

// page wraps the body of a document.
func page(title, body string) string {
	return `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<link rel="icon" type="image/png" href="favicon.png">
<title>` + html.EscapeString(title) + ` - ConfigChief</title>
<style>
  :root { --ground:#f6f8fb; --panel:#fff; --panel-2:#eef2f7; --ink:#1b2430; --ink-soft:#4c5a6b; --line:#dde4ec; --accent:#2563eb; }
  @media (prefers-color-scheme: dark) { :root { --ground:#0f141b; --panel:#171e28; --panel-2:#1f2835; --ink:#e6ebf2; --ink-soft:#aab6c5; --line:#2a3542; --accent:#6ea0ff; } }
  * { box-sizing:border-box; }
  body { margin:0; background:var(--ground); color:var(--ink); font-family:"Segoe UI","Hiragino Sans","Noto Sans JP",system-ui,sans-serif; line-height:1.7; }
  main { max-width:960px; margin:0 auto; padding:32px 20px 64px; }
  nav { font-size:.9rem; margin-bottom:24px; } nav a { margin-right:14px; }
  h1 { font-size:1.9rem; margin:0 0 16px; } h2 { font-size:1.3rem; margin:36px 0 12px; padding-bottom:6px; border-bottom:1px solid var(--line); } h3 { font-size:1.1rem; margin:24px 0 8px; }
  a { color:var(--accent); }
  code { background:var(--panel-2); padding:1px 5px; border-radius:4px; font-size:.92em; }
  pre { background:var(--panel-2); padding:12px 14px; border-radius:8px; overflow-x:auto; line-height:1.5; } pre code { background:none; padding:0; }
  table { border-collapse:collapse; margin:12px 0; display:block; overflow-x:auto; } th, td { border:1px solid var(--line); padding:5px 9px; vertical-align:top; } th { background:var(--panel-2); text-align:left; }
  blockquote { margin:12px 0; padding:8px 16px; border-left:4px solid #e0a800; background:var(--panel); }
  img { max-width:100%; height:auto; border:1px solid var(--line); border-radius:8px; }
  footer { margin-top:48px; color:var(--ink-soft); font-size:.85rem; border-top:1px solid var(--line); padding-top:16px; }
</style>
</head>
<body>
<main>
<nav><a href="index.html">ConfigChief</a><a href="manual.html">使い方</a><a href="tables.html">シートと列</a><a href="checks.html">チェック</a><a href="pack-guide.html">パックの書き方</a><a href="https://github.com/kojio145/ConfigChief">GitHub</a></nav>
` + body + `<footer>ConfigChief (MIT License) by KJO. このページは docs/ の Markdown から作っています。</footer>
</main>
</body>
</html>
`
}

var (
	reCode  = regexp.MustCompile("`([^`]+)`")
	reBold  = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reImg   = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)\)`)
	reLink  = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	reAnchr = regexp.MustCompile(`&lt;a id=&#34;([\w-]+)&#34;&gt;&lt;/a&gt;`)
	reOl    = regexp.MustCompile(`^(\d+)\. (.*)$`)
	reHead  = regexp.MustCompile(`^(#{1,4}) (.*)$`)
)

// href makes the links of docs/ point at the pages: manual.md → manual.html.
// Links that leave docs/ (../README.md, ../internal/…) go to the repository.
func href(u string) string {
	if strings.HasPrefix(u, "http") || strings.HasPrefix(u, "#") {
		return u
	}
	if strings.HasPrefix(u, "../") {
		return "https://github.com/kojio145/ConfigChief/blob/main/" + strings.TrimPrefix(u, "../")
	}
	if p, frag, ok := strings.Cut(u, "#"); ok && strings.HasSuffix(p, ".md") {
		return strings.TrimSuffix(p, ".md") + ".html#" + frag
	}
	if strings.HasSuffix(u, ".md") {
		return strings.TrimSuffix(u, ".md") + ".html"
	}
	return u
}

// inline turns the marks of one line into HTML. Code is kept apart first, so
// that nothing inside it is read as a mark.
func inline(s string) string {
	var codes []string
	s = reCode.ReplaceAllStringFunc(s, func(m string) string {
		codes = append(codes, "<code>"+html.EscapeString(m[1:len(m)-1])+"</code>")
		return fmt.Sprintf("\x00%d\x00", len(codes)-1)
	})
	s = html.EscapeString(s)
	s = reAnchr.ReplaceAllString(s, `<a id="$1"></a>`)
	s = reImg.ReplaceAllStringFunc(s, func(m string) string {
		p := reImg.FindStringSubmatch(m)
		return `<img src="` + p[2] + `" alt="` + p[1] + `">`
	})
	s = reLink.ReplaceAllStringFunc(s, func(m string) string {
		p := reLink.FindStringSubmatch(m)
		return `<a href="` + href(html.UnescapeString(p[2])) + `">` + p[1] + `</a>`
	})
	s = reBold.ReplaceAllString(s, "<b>$1</b>")
	for i, c := range codes {
		s = strings.Replace(s, fmt.Sprintf("\x00%d\x00", i), c, 1)
	}
	return s
}

func cells(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(strings.TrimSuffix(line, "|"), "|")
	// A "|" written as "\|" stays in the cell.
	parts := strings.Split(strings.ReplaceAll(line, `\|`, "\x01"), "|")
	for i, p := range parts {
		parts[i] = strings.ReplaceAll(strings.TrimSpace(p), "\x01", "|")
	}
	return parts
}

// render gives the HTML of a Markdown document and its first heading.
func render(md string) (body, title string) {
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")
	var b strings.Builder
	var para []string
	flush := func() {
		if len(para) > 0 {
			b.WriteString("<p>" + inline(strings.Join(para, " ")) + "</p>\n")
			para = nil
		}
	}
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		t := strings.TrimSpace(l)
		switch {
		case t == "":
			flush()
		case strings.HasPrefix(t, "```"):
			flush()
			var code []string
			for i++; i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```"); i++ {
				code = append(code, lines[i])
			}
			b.WriteString("<pre><code>" + html.EscapeString(strings.Join(code, "\n")) + "</code></pre>\n")
		case reHead.MatchString(l):
			flush()
			m := reHead.FindStringSubmatch(l)
			n := len(m[1])
			if n == 1 && title == "" {
				title = m[2]
			}
			fmt.Fprintf(&b, "<h%d>%s</h%d>\n", n, inline(m[2]), n)
		case t == "---":
			flush()
			b.WriteString("<hr>\n")
		case strings.HasPrefix(t, "|"):
			flush()
			head := cells(l)
			b.WriteString("<table>\n<tr>")
			for _, c := range head {
				b.WriteString("<th>" + inline(c) + "</th>")
			}
			b.WriteString("</tr>\n")
			i++ // the line of dashes
			for i+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i+1]), "|") {
				i++
				b.WriteString("<tr>")
				for _, c := range cells(lines[i]) {
					b.WriteString("<td>" + inline(c) + "</td>")
				}
				b.WriteString("</tr>\n")
			}
			b.WriteString("</table>\n")
		case strings.HasPrefix(t, "> "):
			flush()
			var q []string
			for ; i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ">"); i++ {
				q = append(q, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), ">")))
			}
			i--
			b.WriteString("<blockquote><p>" + inline(strings.Join(q, " ")) + "</p></blockquote>\n")
		case strings.HasPrefix(l, "- ") || reOl.MatchString(l):
			flush()
			tag := "ul"
			if !strings.HasPrefix(l, "- ") {
				tag = "ol"
			}
			b.WriteString("<" + tag + ">\n")
			open := false
			for ; i < len(lines); i++ {
				x := lines[i]
				switch {
				case strings.HasPrefix(x, "- ") || reOl.MatchString(x):
					if open {
						b.WriteString("</li>\n")
					}
					text := strings.TrimPrefix(x, "- ")
					if m := reOl.FindStringSubmatch(x); m != nil && tag == "ol" {
						text = m[2]
					}
					b.WriteString("<li>" + inline(text))
					open = true
				case strings.HasPrefix(x, "  - ") || strings.HasPrefix(x, "   - "):
					b.WriteString("<ul>")
					for ; i < len(lines) && (strings.HasPrefix(lines[i], "  - ") || strings.HasPrefix(lines[i], "   - ")); i++ {
						b.WriteString("<li>" + inline(strings.TrimSpace(lines[i])[2:]) + "</li>")
					}
					i--
					b.WriteString("</ul>")
				case strings.HasPrefix(x, "   ") && strings.TrimSpace(x) != "":
					b.WriteString(" " + inline(strings.TrimSpace(x)))
				default:
					i--
					goto done
				}
			}
		done:
			if open {
				b.WriteString("</li>\n")
			}
			b.WriteString("</" + tag + ">\n")
		default:
			para = append(para, t)
		}
	}
	flush()
	return b.String(), title
}
