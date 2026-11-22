package update

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// The downloaded exe takes the place of the running one (spec 14.11.1).
// Windows lets a running exe be renamed, not deleted: the running one goes
// by the name of its version, and the downloaded one by the name the
// running one had. Nothing here talks to the network.

// FileSHA256 is the SHA-256 of a file, in hexadecimal.
func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Swapped is what Swap did: Undo puts both files back.
type Swapped struct {
	Running string // the path the new exe has now (that of the running one)
	Kept    string // the path the running exe has now
	from    string // the path the downloaded exe had
}

// Swap renames the running exe to ConfigChief-v<current>.exe (with _2 and
// on when the name is taken) and the downloaded exe to the name the running
// one had. The downloaded file is checked against the SHA-256 it had when it
// was downloaded, and has to be in the folder of the running exe.
func Swap(running, downloaded, wantSHA, current string) (*Swapped, error) {
	if downloaded == "" {
		return nil, errors.New("この起動の間にダウンロードした最新版がありません。先に「最新版をダウンロード」を押してください")
	}
	if !sameDir(filepath.Dir(running), filepath.Dir(downloaded)) {
		return nil, fmt.Errorf("ダウンロードした exe（%s）が、今の exe と同じフォルダにありません。ConfigChief を閉じて、手で置き換えてください", downloaded)
	}
	if strings.EqualFold(filepath.Clean(running), filepath.Clean(downloaded)) {
		return nil, errors.New("ダウンロードした exe が、今動いている exe そのものです")
	}
	got, err := FileSHA256(downloaded)
	if err != nil {
		return nil, fmt.Errorf("ダウンロードした exe を読めません: %v", err)
	}
	if wantSHA == "" || !strings.EqualFold(got, wantSHA) {
		return nil, errors.New("ダウンロードした exe が、ダウンロードしたときと変わっています。もう一度「最新版をダウンロード」を押してください")
	}
	dir := filepath.Dir(running)
	name := FileName(current)
	base := strings.TrimSuffix(name, ".exe")
	kept := filepath.Join(dir, name)
	for n := 2; ; n++ {
		if _, err := os.Lstat(kept); err != nil {
			break
		}
		kept = filepath.Join(dir, fmt.Sprintf("%s_%d.exe", base, n))
	}
	if err := os.Rename(running, kept); err != nil {
		return nil, fmt.Errorf("今の exe の名前を変えられません: %v", err)
	}
	if err := os.Rename(downloaded, running); err != nil {
		if back := os.Rename(kept, running); back != nil {
			return nil, fmt.Errorf("新しい exe を置けず（%v）、今の exe の名前も戻せませんでした（%v）。%s の名前を %s に戻してください", err, back, kept, filepath.Base(running))
		}
		return nil, fmt.Errorf("新しい exe を置けません: %v", err)
	}
	return &Swapped{Running: running, Kept: kept, from: downloaded}, nil
}

// Undo puts the files back as they were before Swap.
func (s *Swapped) Undo() error {
	if err := os.Rename(s.Running, s.from); err != nil {
		return fmt.Errorf("新しい exe を元の名前に戻せません: %v", err)
	}
	if err := os.Rename(s.Kept, s.Running); err != nil {
		return fmt.Errorf("今の exe の名前を戻せません（%s を %s に戻してください）: %v", s.Kept, filepath.Base(s.Running), err)
	}
	return nil
}
