package filesystem

import (
	"testing"
)

func TestIsASCII(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"pure ASCII letters", "hello.txt", true},
		{"ASCII with curly braces", "abc{123}.txt", true},
		{"empty string", "", true},
		{"all printable ASCII", " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~", true},
		{"ASCII with null byte", "a\x00b", true},
		{"ASCII DEL char (0x7F)", "a\x7Fb", true},
		{"first non-ASCII byte (0x80)", "\x80", false},
		{"Chinese characters", "默认.txt", false},
		{"mixed ASCII and Chinese", "abc{号码}.txt", false},
		{"Chinese at start", "中文file.txt", false},
		{"Chinese at end", "file中文.txt", false},
		{"multi-byte UTF-8 emoji", "file🎉.txt", false},
		{"Japanese characters", "テスト.txt", false},
		{"single non-ASCII byte in long string", "abcdefghijklmnop\x80qrstuvwxyz", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isASCII(tt.input); got != tt.want {
				t.Errorf("isASCII(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPercentEncodeRFC5987(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Empty string
		{"empty string", "", ""},

		// Unreserved characters that should pass through (RFC 5987 attr-char)
		{"lowercase letters", "abcdefghijklmnopqrstuvwxyz", "abcdefghijklmnopqrstuvwxyz"},
		{"uppercase letters", "ABCDEFGHIJKLMNOPQRSTUVWXYZ", "ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{"digits", "0123456789", "0123456789"},
		{"all attr-char special chars", "!#$&+-.^_`|~", "!#$&+-.^_`|~"},
		{"mixed unreserved", "hello.txt", "hello.txt"},
		{"unreserved with digits", "abc123", "abc123"},

		// Characters that MUST be encoded
		{"curly braces", "{}", "%7B%7D"},
		{"curly braces in context", "a{b}c", "a%7Bb%7Dc"},
		{"space", "a b", "a%20b"},
		{"percent sign", "100%", "100%25"},
		{"forward slash", "/tmp/file", "%2Ftmp%2Ffile"},
		{"backslash", `a\b`, "a%5Cb"},
		{"double quote", `a"b`, "a%22b"},
		{"single quote", "a'b", "a%27b"},
		{"parentheses", "a(b)c", "a%28b%29c"},
		{"square brackets", "a[b]c", "a%5Bb%5Dc"},
		{"at sign", "user@host", "user%40host"},
		{"equals sign", "a=b", "a%3Db"},
		{"question mark", "file?.txt", "file%3F.txt"},
		{"colon", "a:b", "a%3Ab"},
		{"semicolon", "a;b", "a%3Bb"},
		{"comma", "a,b", "a%2Cb"},
		{"asterisk", "a*b", "a%2Ab"},
		{"tab character", "a\tb", "a%09b"},
		{"null byte", "a\x00b", "a%00b"},

		// Multi-byte UTF-8 encoding
		{"Chinese characters", "中文", "%E4%B8%AD%E6%96%87"},
		{"Japanese hiragana", "あ", "%E3%81%82"},
		{"emoji (4-byte UTF-8)", "🎉", "%F0%9F%8E%89"},
		{"mixed ASCII and Chinese", "file中文.txt", "file%E4%B8%AD%E6%96%87.txt"},

		// Real-world bug trigger scenario
		{"bug trigger: Chinese + curly braces",
			"/tmp/022{号码217390}1515-_默认号码篮_.txt",
			"%2Ftmp%2F022%7B%E5%8F%B7%E7%A0%81217390%7D1515-_%E9%BB%98%E8%AE%A4%E5%8F%B7%E7%A0%81%E7%AF%AE_.txt"},

		// Multiple special chars combined
		{"all special chars combined", "{} [] () %", "%7B%7D%20%5B%5D%20%28%29%20%25"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := percentEncodeRFC5987(tt.input); got != tt.want {
				t.Errorf("percentEncodeRFC5987(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestPercentEncodeRFC5987_AllAttrCharUnreserved verifies that every character in the
// RFC 5987 attr-char unreserved set passes through without encoding.
func TestPercentEncodeRFC5987_AllAttrCharUnreserved(t *testing.T) {
	unreserved := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!#$&+-.^_`|~"
	for i := 0; i < len(unreserved); i++ {
		c := string(unreserved[i])
		if got := percentEncodeRFC5987(c); got != c {
			t.Errorf("percentEncodeRFC5987(%q) = %q, want %q (should be unreserved)", c, got, c)
		}
	}
}

// TestPercentEncodeRFC5987_CommonReservedCharsEncoded verifies that common reserved
// characters that are NOT in the attr-char set are properly percent-encoded.
func TestPercentEncodeRFC5987_CommonReservedCharsEncoded(t *testing.T) {
	// Characters that people might expect to be safe but must be encoded
	reserved := map[byte]string{
		' ':  "%20",
		'"':  "%22",
		'%':  "%25",
		'\'': "%27",
		'(':  "%28",
		')':  "%29",
		'*':  "%2A",
		',':  "%2C",
		'/':  "%2F",
		':':  "%3A",
		';':  "%3B",
		'<':  "%3C",
		'=':  "%3D",
		'>':  "%3E",
		'?':  "%3F",
		'@':  "%40",
		'[':  "%5B",
		'\\': "%5C",
		']':  "%5D",
		'{':  "%7B",
		'}':  "%7D",
	}
	for c, want := range reserved {
		got := percentEncodeRFC5987(string(c))
		if got != want {
			t.Errorf("percentEncodeRFC5987(%q) = %q, want %q", string(c), got, want)
		}
	}
}
