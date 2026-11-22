// Package casefile is the file a case is written out to and read back from
// (spec 14.9; PalaTerm's 一式書き出し in one file): a zip of the files of the
// case with a manifest, sealed with a password when one is given.
//
// Layout:
//
//	"CCASE1\x00\x00" (8 bytes)  then the zip                     not encrypted
//	"CCASE1E\x00"    (8 bytes)  salt (16) nonce (12) sealed zip  encrypted
//
// The key is PBKDF2-SHA256 of the password (600 000 rounds, the salt), the
// seal AES-256-GCM; the standard library only. Nothing here touches a case
// folder: Pack takes the files, Unpack gives them back after checking them.
package casefile

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"
)

var (
	magicPlain  = []byte("CCASE1\x00\x00")
	magicSealed = []byte("CCASE1E\x00")
)

const (
	// ManifestName is the manifest inside the zip.
	ManifestName = "configchief-case.json"
	// Format is the version of the layout this tool writes and reads.
	Format = 1

	saltLen  = 16
	nonceLen = 12
	rounds   = 600000
	// MinPassword is the shortest password a case is sealed with.
	MinPassword = 8

	maxFile  = 1 << 30
	maxTotal = 4 << 30
	maxFiles = 100000
)

// ErrPassword is a wrong password or a broken file (GCM cannot tell them
// apart).
var ErrPassword = errors.New("パスワードが違うか、ファイルが壊れています")

// ErrNeedPassword is a sealed file read without a password.
var ErrNeedPassword = errors.New("このファイルはパスワードで暗号化されています。パスワードを入れてください")

// Manifest says what the file holds.
type Manifest struct {
	Format   int      `json:"format"`
	Name     string   `json:"name"`    // the name of the case
	Written  string   `json:"written"` // when, RFC 3339
	Version  string   `json:"version"` // the ConfigChief that wrote it
	NoSecret bool     `json:"noSecrets"`
	Files    []string `json:"files"` // the paths of the case folder, with "/"
}

// File is a file of a case: its path from the case folder ("/") and bytes.
type File struct {
	Path string
	Data []byte
}

// Sealed tells whether data is a sealed case file (false for a plain one or
// for something that is not a case file).
func Sealed(data []byte) bool { return bytes.HasPrefix(data, magicSealed) }

// Is tells whether data looks like a case file.
func Is(data []byte) bool { return bytes.HasPrefix(data, magicPlain) || Sealed(data) }

// Pack makes the file of a case. password "" leaves it unsealed.
func Pack(m Manifest, files []File, password string, now time.Time) ([]byte, error) {
	if password != "" && len([]rune(password)) < MinPassword {
		return nil, fmt.Errorf("パスワードは %d 文字以上にしてください", MinPassword)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	m.Format = Format
	m.Files = nil
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		if err := checkPath(f.Path); err != nil {
			return nil, err
		}
		h := &zip.FileHeader{Name: f.Path, Method: zip.Deflate, Modified: now}
		w, err := zw.CreateHeader(h)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(f.Data); err != nil {
			return nil, err
		}
		m.Files = append(m.Files, f.Path)
	}
	if m.Files == nil {
		m.Files = []string{}
	}
	mj, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	w, err := zw.CreateHeader(&zip.FileHeader{Name: ManifestName, Method: zip.Deflate, Modified: now})
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(mj); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	if password == "" {
		return append(append([]byte{}, magicPlain...), buf.Bytes()...), nil
	}
	salt := make([]byte, saltLen)
	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	gcm, err := sealer(password, salt)
	if err != nil {
		return nil, err
	}
	out := append(append(append([]byte{}, magicSealed...), salt...), nonce...)
	return gcm.Seal(out, nonce, buf.Bytes(), magicSealed), nil
}

func sealer(password string, salt []byte) (cipher.AEAD, error) {
	key, err := pbkdf2.Key(sha256.New, password, salt, rounds, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// Unpack opens the file of a case and gives its manifest and files. Only the
// files the manifest names are given; a name that leaves the case folder,
// too many or too large files stop it.
func Unpack(data []byte, password string) (*Manifest, []File, error) {
	var z []byte
	switch {
	case bytes.HasPrefix(data, magicPlain):
		z = data[len(magicPlain):]
	case Sealed(data):
		if password == "" {
			return nil, nil, ErrNeedPassword
		}
		rest := data[len(magicSealed):]
		if len(rest) < saltLen+nonceLen+16 {
			return nil, nil, ErrPassword
		}
		gcm, err := sealer(password, rest[:saltLen])
		if err != nil {
			return nil, nil, err
		}
		z, err = gcm.Open(nil, rest[saltLen:saltLen+nonceLen], rest[saltLen+nonceLen:], magicSealed)
		if err != nil {
			return nil, nil, ErrPassword
		}
	default:
		return nil, nil, errors.New("ConfigChief の案件のファイル（.ccase）ではありません")
	}
	zr, err := zip.NewReader(bytes.NewReader(z), int64(len(z)))
	if err != nil {
		return nil, nil, fmt.Errorf("ファイルが壊れています: %v", err)
	}
	if len(zr.File) > maxFiles+1 {
		return nil, nil, fmt.Errorf("ファイルの数が多すぎます（%d）", len(zr.File))
	}
	byName := map[string]*zip.File{}
	for _, f := range zr.File {
		byName[f.Name] = f
	}
	mf := byName[ManifestName]
	if mf == nil {
		return nil, nil, errors.New("ファイルに目録（" + ManifestName + "）がありません")
	}
	mj, err := readZip(mf, 1<<24)
	if err != nil {
		return nil, nil, err
	}
	var m Manifest
	if err := json.Unmarshal(mj, &m); err != nil {
		return nil, nil, fmt.Errorf("目録を読めません: %v", err)
	}
	if m.Format > Format {
		return nil, nil, fmt.Errorf("新しい ConfigChief（%s）で書き出したファイルです（形の版 %d）。この版では読めません。最新版にしてください", m.Version, m.Format)
	}
	var out []File
	var total int64
	for _, name := range m.Files {
		if err := checkPath(name); err != nil {
			return nil, nil, err
		}
		f := byName[name]
		if f == nil {
			return nil, nil, fmt.Errorf("目録にある %s が、ファイルにありません", name)
		}
		b, err := readZip(f, maxFile)
		if err != nil {
			return nil, nil, err
		}
		total += int64(len(b))
		if total > maxTotal {
			return nil, nil, errors.New("中身が大きすぎます（4 GB まで）")
		}
		out = append(out, File{Path: name, Data: b})
	}
	return &m, out, nil
}

func readZip(f *zip.File, limit int64) ([]byte, error) {
	if f.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("%s が大きすぎます", f.Name)
	}
	r, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("%s を読めません: %v", f.Name, err)
	}
	defer r.Close()
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, fmt.Errorf("%s を読めません: %v", f.Name, err)
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("%s が大きすぎます", f.Name)
	}
	return b, nil
}

// checkPath refuses a name that would leave the case folder.
func checkPath(p string) error {
	bad := p == "" || strings.ContainsAny(p, "\\:\x00") || strings.HasPrefix(p, "/") || p == ManifestName
	if !bad {
		for _, part := range strings.Split(p, "/") {
			if part == "" || part == "." || part == ".." {
				bad = true
			}
		}
		bad = bad || path.Clean(p) != p
	}
	if bad {
		return fmt.Errorf("ファイルの名前 %q は使えません（案件フォルダの外を指しています）", p)
	}
	return nil
}
