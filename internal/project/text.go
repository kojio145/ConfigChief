package project

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
)

// ReadText reads a text file of the user: UTF-8 with or without a byte order
// mark, UTF-16 with one, or Shift_JIS (spec 4.2). The line ends become "\n".
func ReadText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("読めません（%v）", err)
	}
	return Decode(data), nil
}

// Decode turns the bytes of a text file into a string with "\n" line ends.
func Decode(data []byte) string {
	var s string
	switch {
	case bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}):
		s = strings.ToValidUTF8(string(data[3:]), string(utf8.RuneError))
	case bytes.HasPrefix(data, []byte{0xFF, 0xFE}):
		s = decodeUTF16(data[2:], false)
	case bytes.HasPrefix(data, []byte{0xFE, 0xFF}):
		s = decodeUTF16(data[2:], true)
	case utf8.Valid(data):
		s = string(data)
	default:
		out, err := japanese.ShiftJIS.NewDecoder().Bytes(data)
		if err != nil {
			s = strings.ToValidUTF8(string(data), string(utf8.RuneError))
		} else {
			s = string(out)
		}
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// encodingName gives the one name of an encoding a configuration can be
// written in: "utf-8", "utf-8-bom" or "shift_jis" ("" = not known).
func encodingName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "utf-8", "utf8":
		return "utf-8"
	case "utf-8-bom", "utf8-bom", "utf-8 bom":
		return "utf-8-bom"
	case "shift_jis", "shift-jis", "sjis", "cp932", "ms932", "windows-31j":
		return "shift_jis"
	}
	return ""
}

// KnownEncoding tells whether a configuration can be written in the encoding
// ("" = UTF-8).
func KnownEncoding(name string) bool { return encodingName(name) != "" }

// Encode turns a text with "\n" line ends into the bytes of a file (spec
// 9.6). newline is "lf", "crlf" or "" (= lf). A character the encoding does
// not have is an error: it is never written as something else.
func Encode(text, encoding, newline string) ([]byte, error) {
	if newline == "crlf" {
		text = strings.ReplaceAll(text, "\n", "\r\n")
	}
	switch encodingName(encoding) {
	case "utf-8":
		return []byte(text), nil
	case "utf-8-bom":
		return append([]byte{0xEF, 0xBB, 0xBF}, text...), nil
	case "shift_jis":
		enc := japanese.ShiftJIS.NewEncoder()
		for i, line := range strings.Split(text, "\n") {
			for _, r := range line {
				if _, err := enc.String(string(r)); err != nil {
					return nil, fmt.Errorf("%d 行目の「%c」は Shift_JIS で書けません", i+1, r)
				}
			}
		}
		out, err := japanese.ShiftJIS.NewEncoder().String(text)
		if err != nil {
			return nil, fmt.Errorf("Shift_JIS で書けません: %v", err)
		}
		return []byte(out), nil
	}
	return nil, fmt.Errorf("文字コード「%s」は使えません（utf-8、utf-8-bom、shift_jis）", encoding)
}

func decodeUTF16(b []byte, bigEndian bool) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		if bigEndian {
			u[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
		} else {
			u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
		}
	}
	return string(utf16.Decode(u))
}
