// Package update asks GitHub for the newest public release of ConfigChief
// and, on request, downloads its ConfigChief.exe (spec 14.8.1). It is the
// only code of the tool that talks to the network, and it does so only when
// the user presses a button: nothing is sent at the start, nothing about the
// case. The addresses are fixed (nonet_test.go checks that no other is
// written here). The running exe is never replaced.
//
// The model is PalaTerm's 最新版の確認 (update_app.go, v1.10.8).
package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// The only places the tool asks.
	releaseAPI  = "https://api.github.com/repos/kojio145/ConfigChief/releases/latest"
	releasesAPI = "https://api.github.com/repos/kojio145/ConfigChief/releases?per_page=50"
	// AssetPrefix: only an asset of this repository's releases is downloaded.
	AssetPrefix = "https://github.com/kojio145/ConfigChief/releases/download/"
	// The pages the browser is sent to (the tool itself fetches nothing there).
	ReleasePage = "https://github.com/kojio145/ConfigChief/releases/latest"
	tagPage     = "https://github.com/kojio145/ConfigChief/releases/tag/v%s"
	RepoPage    = "https://github.com/kojio145/ConfigChief"
	ManualPage  = "https://kojio145.github.io/ConfigChief/manual.html"
	// ApprovalPage is the sheet for a request to use the tool (spec 14.13).
	ApprovalPage = "https://kojio145.github.io/ConfigChief/approval.html"

	ExeName = "ConfigChief.exe"
	maxExe  = 64 << 20
)

// Info is the newest public release compared with this build.
type Info struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Newer     bool   `json:"newer"`
	Published string `json:"published"`
	HasExe    bool   `json:"hasExe"`
	ExeSize   int64  `json:"exeSize"`
	// HasDigest: the release gives the exe's SHA-256 (the download is checked
	// against it; without one, against the size only).
	HasDigest bool `json:"hasDigest"`
	// SaveDir is where a download goes; InAppDir: it is the folder the
	// running exe is in.
	SaveDir  string `json:"saveDir"`
	InAppDir bool   `json:"inAppDir"`
	// Notes are the changes of every release newer than this build up to the
	// latest, the newest first.
	Notes []Note `json:"notes"`
}

// Note is one release's changes.
type Note struct {
	Version   string `json:"version"`
	Published string `json:"published"`
	Body      string `json:"body"`
}

type ghRelease struct {
	TagName     string    `json:"tag_name"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	Assets      []struct {
		Name   string `json:"name"`
		URL    string `json:"browser_download_url"`
		Size   int64  `json:"size"`
		Digest string `json:"digest"`
	} `json:"assets"`
}

// proxyRefused is a proxy's refusal of the CONNECT of an https request (407,
// 403). Go hands it back as an error with the status text only, so the
// transport keeps the code.
type proxyRefused struct{ code int }

const proxyAuthMsg = "プロキシが認証を求めました（HTTP 407）。ConfigChief はプロキシの認証に対応していません。ブラウザーでリリースのページから入手してください"

func (e proxyRefused) Error() string {
	if e.code == http.StatusProxyAuthRequired {
		return proxyAuthMsg
	}
	return fmt.Sprintf("プロキシに断られました（HTTP %d）。プロキシやセキュリティ製品が GitHub への通信を止めている可能性があります", e.code)
}

var transport = func() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = proxyFor
	t.OnProxyConnectResponse = func(_ context.Context, _ *url.URL, _ *http.Request, res *http.Response) error {
		if res.StatusCode == http.StatusProxyAuthRequired || res.StatusCode == http.StatusForbidden {
			return proxyRefused{res.StatusCode}
		}
		return nil
	}
	return t
}()

var (
	client = &http.Client{Timeout: 60 * time.Second, Transport: transport}
	// downloadClient has no Timeout of its own (it would also cover reading
	// the body, and the exe does not come in a minute on a slow line or
	// through a proxy that inspects it): the request's context bounds it.
	downloadClient = &http.Client{Transport: transport}
)

// connectErr is a request that did not reach GitHub, with what to try.
func connectErr(what string, err error) error {
	var pr proxyRefused
	if errors.As(err, &pr) {
		return pr
	}
	if proxyScriptOnly() {
		return fmt.Errorf("%s（この PC のプロキシは自動設定のスクリプトです。ConfigChief はスクリプトを読めないので、Windows の設定でプロキシを手動で入れるか、ブラウザーでリリースのページから入手してください）: %v", what, err)
	}
	return fmt.Errorf("%s（インターネットに出られない PC では使えません。プロキシは Windows の手動設定か、環境変数 HTTPS_PROXY のものを使います）: %v", what, err)
}

// statusErr is an answer that is not 200 said so the user knows what to do,
// for the usual ones; nil for any other.
func statusErr(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusForbidden, http.StatusTooManyRequests:
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			when := "しばらくしてから"
			if sec, err := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil && sec > 0 {
				when = time.Unix(sec, 0).Local().Format("15:04") + " 以降に"
			}
			return fmt.Errorf("GitHub の回数制限にかかりました（同じ回線からの問い合わせは 1 時間に 60 回まで）。%sもう一度お試しください", when)
		}
		return fmt.Errorf("GitHub に断られました（HTTP %d）。プロキシやセキュリティ製品が GitHub への通信を止めている可能性があります", resp.StatusCode)
	case http.StatusProxyAuthRequired:
		return errors.New(proxyAuthMsg)
	}
	return nil
}

func get(ctx context.Context, url, version string, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ConfigChief/"+version)
	resp, err := client.Do(req)
	if err != nil {
		return nil, connectErr("GitHub に接続できませんでした", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if err := statusErr(resp); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("GitHub から最新版を取得できませんでした（HTTP %d）", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, limit))
}

func latest(ctx context.Context, version string) (*ghRelease, error) {
	data, err := get(ctx, releaseAPI, version, 4<<20)
	if err != nil {
		return nil, err
	}
	var rel ghRelease
	if err := json.Unmarshal(data, &rel); err != nil {
		return nil, fmt.Errorf("GitHub の応答を読めませんでした: %v", err)
	}
	if rel.TagName == "" {
		return nil, errors.New("GitHub の応答に版数がありません")
	}
	return &rel, nil
}

// NoteBody is what a release body says changed, in Japanese when it has a
// "## 日本語" section, without its download section.
func NoteBody(b string) string {
	b = strings.ReplaceAll(b, "\r\n", "\n")
	if i := strings.Index(b, "\n## Download"); i >= 0 {
		b = b[:i]
	}
	if i := strings.Index(b, "## 日本語\n"); i >= 0 {
		b = b[i+len("## 日本語\n"):]
	}
	return strings.TrimSpace(b)
}

// notesOf are the notes of the releases newer than current up to latestTag,
// the newest first. A draft, a pre-release and a tag past the latest are not
// what a download gives.
func notesOf(rels []ghRelease, latestTag, current string) []Note {
	var notes []Note
	for _, r := range rels {
		if r.Draft || r.Prerelease || !Newer(r.TagName, current) || Newer(r.TagName, latestTag) {
			continue
		}
		n := Note{Version: strings.TrimPrefix(r.TagName, "v"), Body: NoteBody(r.Body)}
		if !r.PublishedAt.IsZero() {
			n.Published = r.PublishedAt.Local().Format("2006-01-02")
		}
		notes = append(notes, n)
	}
	sort.SliceStable(notes, func(i, j int) bool { return Newer(notes[i].Version, notes[j].Version) })
	return notes
}

// parts reads "v1.9.41" / "1.9.41" as numbers (missing parts 0). The
// development build "0.0.0-dev" is 0.0.0: every public version is newer.
func parts(v string) [3]int {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, " -+"); i >= 0 {
		v = v[:i]
	}
	for i, p := range strings.SplitN(v, ".", 3) {
		n, _ := strconv.Atoi(p)
		out[i] = n
	}
	return out
}

// Newer reports whether a is a later version than b.
func Newer(a, b string) bool {
	x, y := parts(a), parts(b)
	for i := range x {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}

// Check asks GitHub for the newest public release and compares it with
// current. exeDir is the folder of the running exe.
func Check(ctx context.Context, current, exeDir string) (*Info, error) {
	rel, err := latest(ctx, current)
	if err != nil {
		return nil, err
	}
	info := &Info{Current: current, Latest: strings.TrimPrefix(rel.TagName, "v"), Notes: []Note{}, SaveDir: SaveDir(exeDir)}
	info.InAppDir = sameDir(info.SaveDir, exeDir)
	info.Newer = Newer(rel.TagName, current)
	if info.Newer {
		if data, err := get(ctx, releasesAPI, current, 8<<20); err == nil {
			var rels []ghRelease
			if json.Unmarshal(data, &rels) == nil {
				info.Notes = notesOf(rels, rel.TagName, current)
			}
		}
		if len(info.Notes) == 0 {
			info.Notes = []Note{{Version: info.Latest, Body: NoteBody(rel.Body)}}
		}
	}
	if !rel.PublishedAt.IsZero() {
		info.Published = rel.PublishedAt.Local().Format("2006-01-02 15:04")
	}
	for _, as := range rel.Assets {
		if as.Name == ExeName && strings.HasPrefix(as.URL, AssetPrefix) {
			info.HasExe, info.ExeSize = true, as.Size
			info.HasDigest = strings.HasPrefix(as.Digest, "sha256:")
		}
	}
	return info, nil
}

// SaveDir is where a downloaded exe goes: the folder of the running exe,
// the user's Downloads folder when that one cannot be written to (Program
// Files, a read-only share), else the temporary folder.
func SaveDir(exeDir string) string {
	if exeDir != "" && writable(exeDir) {
		return exeDir
	}
	if h := os.Getenv("USERPROFILE"); h != "" {
		d := filepath.Join(h, "Downloads")
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			return d
		}
	}
	return os.TempDir()
}

func writable(dir string) bool {
	f, err := os.CreateTemp(dir, "ConfigChief-download-*.part")
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(f.Name())
	return true
}

func sameDir(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if strings.EqualFold(filepath.Clean(a), filepath.Clean(b)) {
		return true
	}
	sa, err1 := os.Stat(a)
	sb, err2 := os.Stat(b)
	return err1 == nil && err2 == nil && os.SameFile(sa, sb)
}

// Download fetches the newest release's ConfigChief.exe into SaveDir(exeDir)
// as ConfigChief-v<版>.exe and gives its path. The file is checked against
// the size and, when GitHub gives one, the SHA-256 of the release asset.
func Download(ctx context.Context, current, exeDir string) (string, error) {
	rel, err := latest(ctx, current)
	if err != nil {
		return "", err
	}
	var url, digest string
	var size int64
	for _, as := range rel.Assets {
		if as.Name == ExeName && strings.HasPrefix(as.URL, AssetPrefix) {
			url, size, digest = as.URL, as.Size, as.Digest
		}
	}
	if url == "" {
		return "", fmt.Errorf("最新版（%s）に %s が添付されていません。リリースのページから入手してください", rel.TagName, ExeName)
	}
	if size <= 0 || size > maxExe {
		return "", fmt.Errorf("最新版の %s の大きさが想定外です（%d バイト）", ExeName, size)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ConfigChief/"+current)
	resp, err := downloadClient.Do(req)
	if err != nil {
		return "", connectErr("ダウンロードできませんでした", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if err := statusErr(resp); err != nil {
			return "", err
		}
		return "", fmt.Errorf("ダウンロードできませんでした（HTTP %d）", resp.StatusCode)
	}
	dir := SaveDir(exeDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	final := filepath.Join(dir, FileName(rel.TagName))
	tmp, err := os.CreateTemp(dir, "ConfigChief-download-*.part")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(resp.Body, maxExe+1))
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", fmt.Errorf("ダウンロードが途中で止まりました: %v", err)
	}
	if n != size {
		return "", fmt.Errorf("ダウンロードした大きさが合いません（%d / %d バイト）", n, size)
	}
	if want, ok := strings.CutPrefix(digest, "sha256:"); ok && !strings.EqualFold(want, hex.EncodeToString(h.Sum(nil))) {
		return "", errors.New("ダウンロードしたファイルの SHA-256 が、リリースに登録された値と合いません")
	}
	_ = os.Remove(final)
	if err := os.Rename(tmp.Name(), final); err != nil {
		return "", err
	}
	return final, nil
}

// FileName is the name a downloaded release is saved under:
// ConfigChief-v1.3.0.exe, kept to letters, digits and . - _.
func FileName(tag string) string {
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return strings.Map(func(r rune) rune {
		if r == '.' || r == '-' || r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			return r
		}
		return '_'
	}, "ConfigChief-"+tag+".exe")
}

// NotesPage is the release page of version (the latest page when the
// version cannot be read): what the browser opens for 変更点.
func NotesPage(version string) string {
	p := parts(version)
	if p == [3]int{} {
		return ReleasePage
	}
	return fmt.Sprintf(tagPage, fmt.Sprintf("%d.%d.%d", p[0], p[1], p[2]))
}
