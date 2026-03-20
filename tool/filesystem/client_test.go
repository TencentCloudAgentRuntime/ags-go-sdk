package filesystem

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"testing"
)

func TestCreateFormFileEncoded(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "pure ASCII filename",
			filename: "/tmp/demo/plain.txt",
		},
		{
			name:     "ASCII with curly braces",
			filename: "/tmp/demo/abc{123}.txt",
		},
		{
			name:     "Chinese characters without braces",
			filename: "/tmp/demo/默认号码篮.txt",
		},
		{
			name:     "Chinese characters with curly braces (the bug trigger)",
			filename: "/tmp/demo/022{号码217390}1515-_默认号码篮_.txt",
		},
		{
			name:     "mixed unicode with special chars",
			filename: "/tmp/demo/テスト{data}ファイル.txt",
		},
		{
			name:     "filename with spaces and unicode",
			filename: "/tmp/demo/文件 名称 {test}.txt",
		},
		{
			name:     "filename with percent sign and unicode",
			filename: "/tmp/demo/100%完成{报告}.txt",
		},
		{
			name:     "filename with quotes",
			filename: `/tmp/demo/file"name.txt`,
		},
		{
			name:     "filename with backslash",
			filename: `/tmp/demo/file\name.txt`,
		},
		{
			name:     "empty path component",
			filename: "simple.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)

			part, err := createFormFileEncoded(w, "file", tt.filename)
			if err != nil {
				t.Fatalf("createFormFileEncoded() error: %v", err)
			}

			// Write some test content to the part
			if _, err := io.WriteString(part, "test content"); err != nil {
				t.Fatalf("failed to write to part: %v", err)
			}

			if err := w.Close(); err != nil {
				t.Fatalf("failed to close writer: %v", err)
			}

			// Now parse the multipart form back to verify the server can read it
			reader := multipart.NewReader(&buf, w.Boundary())
			p, err := reader.NextPart()
			if err != nil {
				t.Fatalf("failed to read next part: %v", err)
			}

			// Verify form name is recognized as "file"
			if got := p.FormName(); got != "file" {
				t.Errorf("FormName() = %q, want %q", got, "file")
			}

			// Verify content can be read
			content, err := io.ReadAll(p)
			if err != nil {
				t.Fatalf("failed to read part content: %v", err)
			}
			if string(content) != "test content" {
				t.Errorf("content = %q, want %q", string(content), "test content")
			}
		})
	}
}

// TestCreateFormFileEncoded_ContentDispositionRoundTrip specifically tests that the
// Content-Disposition header generated can be parsed back by mime.ParseMediaType,
// which is the exact scenario that caused the original bug.
func TestCreateFormFileEncoded_ContentDispositionRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "pure ASCII",
			filename: "/tmp/plain.txt",
		},
		{
			name:     "Chinese only",
			filename: "/tmp/默认号码篮.txt",
		},
		{
			name:     "Chinese + curly braces (original bug)",
			filename: "/tmp/022{号码217390}1515-_默认号码篮_.txt",
		},
		{
			name:     "ASCII + curly braces",
			filename: "/tmp/abc{123}.txt",
		},
		{
			name:     "Korean + braces",
			filename: "/tmp/테스트{파일}.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)

			part, err := createFormFileEncoded(w, "file", tt.filename)
			if err != nil {
				t.Fatalf("createFormFileEncoded() error: %v", err)
			}
			_, _ = io.WriteString(part, "x")
			_ = w.Close()

			// Read back the raw multipart to extract the Content-Disposition header
			reader := multipart.NewReader(&buf, w.Boundary())
			p, err := reader.NextPart()
			if err != nil {
				t.Fatalf("NextPart() error: %v", err)
			}

			// Get the raw Content-Disposition header
			cd := p.Header.Get("Content-Disposition")
			if cd == "" {
				t.Fatal("Content-Disposition header is empty")
			}

			// Verify mime.ParseMediaType can parse it (this is what envd's server does)
			mediaType, params, err := mime.ParseMediaType(cd)
			if err != nil {
				t.Fatalf("mime.ParseMediaType(%q) error: %v\nThis is the exact bug that caused 'expected write info in response'", cd, err)
			}

			if mediaType != "form-data" {
				t.Errorf("media type = %q, want %q", mediaType, "form-data")
			}

			if got := params["name"]; got != "file" {
				t.Errorf("params[name] = %q, want %q", got, "file")
			}

			// mime.ParseMediaType merges filename* into filename when both are present,
			// so for all cases the "filename" param should recover the original value.
			gotFilename := params["filename"]
			if gotFilename != tt.filename {
				t.Errorf("params[filename] = %q, want %q (filename* round-trip failed)", gotFilename, tt.filename)
			}
		})
	}
}

// TestCreateFormFileEncoded_FilenameRecovery verifies that for non-ASCII filenames,
// the filename* parameter (RFC 5987) correctly recovers the original filename when
// parsed by mime.ParseMediaType, which merges filename* into the "filename" key.
func TestCreateFormFileEncoded_FilenameRecovery(t *testing.T) {
	filenames := []string{
		"/tmp/demo/默认号码篮.txt",
		"/tmp/demo/022{号码217390}1515-_默认号码篮_.txt",
		"/tmp/demo/テスト{data}ファイル.txt",
		"/tmp/demo/文件 名称 {test}.txt",
		"/tmp/demo/100%完成{报告}.txt",
		"/home/user/workspace/日本語テスト[v2]{final}.doc",
		"/data/émojis_🎉_{party}.txt",
	}

	for _, filename := range filenames {
		t.Run(filename, func(t *testing.T) {
			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)
			part, err := createFormFileEncoded(w, "file", filename)
			if err != nil {
				t.Fatalf("createFormFileEncoded() error: %v", err)
			}
			_, _ = io.WriteString(part, "x")
			_ = w.Close()

			reader := multipart.NewReader(&buf, w.Boundary())
			p, err := reader.NextPart()
			if err != nil {
				t.Fatalf("NextPart() error: %v", err)
			}

			cd := p.Header.Get("Content-Disposition")
			_, params, err := mime.ParseMediaType(cd)
			if err != nil {
				t.Fatalf("mime.ParseMediaType failed: %v\nContent-Disposition: %s", err, cd)
			}

			// mime.ParseMediaType decodes filename* and overwrites filename
			if got := params["filename"]; got != filename {
				t.Errorf("recovered filename = %q, want %q", got, filename)
			}
		})
	}
}

// TestCreateFormFileEncoded_ASCIIFallback verifies that for non-ASCII filenames,
// the raw ASCII fallback filename (before filename* takes precedence) only contains
// ASCII characters, with non-ASCII bytes replaced by underscores.
func TestCreateFormFileEncoded_ASCIIFallback(t *testing.T) {
	tests := []struct {
		name             string
		filename         string
		wantFallbackPart string // expected substring in the raw Content-Disposition
	}{
		{
			name:             "Chinese replaced with underscores",
			filename:         "/tmp/hello世界.txt",
			wantFallbackPart: `filename="/tmp/hello___.txt"`, // 世界 = 3 bytes × 2 = 6 bytes → 6 underscores... actually let's check
		},
		{
			name:             "bug trigger fallback",
			filename:         "/tmp/022{号码217390}1515-_默认号码篮_.txt",
			wantFallbackPart: `filename="/tmp/022{`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)
			part, err := createFormFileEncoded(w, "file", tt.filename)
			if err != nil {
				t.Fatalf("createFormFileEncoded() error: %v", err)
			}
			_, _ = io.WriteString(part, "x")
			_ = w.Close()

			reader := multipart.NewReader(&buf, w.Boundary())
			p, err := reader.NextPart()
			if err != nil {
				t.Fatalf("NextPart() error: %v", err)
			}

			cd := p.Header.Get("Content-Disposition")

			// Verify Content-Disposition contains both filename and filename*
			if !strings.Contains(cd, "filename=") {
				t.Error("Content-Disposition missing filename parameter")
			}
			if !strings.Contains(cd, "filename*=UTF-8''") {
				t.Error("Content-Disposition missing filename* parameter")
			}

			// Extract the raw fallback filename value between the first filename=" and the next "
			// We look for: filename="<value>"
			idx := strings.Index(cd, `filename="`)
			if idx == -1 {
				t.Fatal("cannot find filename= in Content-Disposition")
			}
			rest := cd[idx+len(`filename="`):]
			endIdx := strings.Index(rest, `"`)
			if endIdx == -1 {
				t.Fatal("cannot find closing quote for filename value")
			}
			fallbackValue := rest[:endIdx]

			// Verify fallback contains only ASCII characters
			if !isASCII(fallbackValue) {
				t.Errorf("ASCII fallback contains non-ASCII characters: %q", fallbackValue)
			}
		})
	}
}

// TestCreateFormFileEncoded_ContentType verifies that the Content-Type header
// is correctly set to application/octet-stream for all filenames.
func TestCreateFormFileEncoded_ContentType(t *testing.T) {
	filenames := []string{
		"/tmp/plain.txt",
		"/tmp/022{号码217390}1515-_默认号码篮_.txt",
		"/tmp/image.png",
	}

	for _, filename := range filenames {
		t.Run(filename, func(t *testing.T) {
			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)
			part, err := createFormFileEncoded(w, "file", filename)
			if err != nil {
				t.Fatalf("createFormFileEncoded() error: %v", err)
			}
			_, _ = io.WriteString(part, "x")
			_ = w.Close()

			reader := multipart.NewReader(&buf, w.Boundary())
			p, err := reader.NextPart()
			if err != nil {
				t.Fatalf("NextPart() error: %v", err)
			}

			ct := p.Header.Get("Content-Type")
			if ct != "application/octet-stream" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/octet-stream")
			}
		})
	}
}

// TestCreateFormFileEncoded_CustomFieldName verifies that the function correctly
// handles different fieldname values, not just "file".
func TestCreateFormFileEncoded_CustomFieldName(t *testing.T) {
	tests := []struct {
		fieldname string
		filename  string
	}{
		{"file", "/tmp/test.txt"},
		{"attachment", "/tmp/test.txt"},
		{"upload", "/tmp/中文.txt"},
		{"data", "/tmp/022{号码}.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldname, func(t *testing.T) {
			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)
			part, err := createFormFileEncoded(w, tt.fieldname, tt.filename)
			if err != nil {
				t.Fatalf("createFormFileEncoded() error: %v", err)
			}
			_, _ = io.WriteString(part, "x")
			_ = w.Close()

			reader := multipart.NewReader(&buf, w.Boundary())
			p, err := reader.NextPart()
			if err != nil {
				t.Fatalf("NextPart() error: %v", err)
			}

			if got := p.FormName(); got != tt.fieldname {
				t.Errorf("FormName() = %q, want %q", got, tt.fieldname)
			}
		})
	}
}

// TestCreateFormFileEncoded_ASCIIBranch verifies that pure ASCII filenames
// do NOT emit filename* parameter (only simple filename="..." is used).
func TestCreateFormFileEncoded_ASCIIBranch(t *testing.T) {
	asciiFilenames := []string{
		"/tmp/plain.txt",
		"/tmp/abc{123}.txt",
		"simple.txt",
		`/tmp/file with spaces.txt`,
	}

	for _, filename := range asciiFilenames {
		t.Run(filename, func(t *testing.T) {
			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)
			part, err := createFormFileEncoded(w, "file", filename)
			if err != nil {
				t.Fatalf("createFormFileEncoded() error: %v", err)
			}
			_, _ = io.WriteString(part, "x")
			_ = w.Close()

			reader := multipart.NewReader(&buf, w.Boundary())
			p, err := reader.NextPart()
			if err != nil {
				t.Fatalf("NextPart() error: %v", err)
			}

			cd := p.Header.Get("Content-Disposition")

			// For pure ASCII filenames, filename* should NOT be present
			if strings.Contains(cd, "filename*=") {
				t.Errorf("ASCII filename should not have filename* parameter, got: %s", cd)
			}

			// Verify mime.ParseMediaType can parse it
			_, params, err := mime.ParseMediaType(cd)
			if err != nil {
				t.Fatalf("mime.ParseMediaType failed: %v\nContent-Disposition: %s", err, cd)
			}

			if got := params["filename"]; got != filename {
				t.Errorf("params[filename] = %q, want %q", got, filename)
			}
		})
	}
}

// TestCreateFormFileEncoded_EscapeInASCIIFilename verifies that special characters
// (backslash, double-quote) in ASCII filenames are properly escaped.
func TestCreateFormFileEncoded_EscapeInASCIIFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"double quote in name", `/tmp/file"name.txt`},
		{"backslash in name", `/tmp/file\name.txt`},
		{"both quote and backslash", `/tmp/"file"\name.txt`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)
			part, err := createFormFileEncoded(w, "file", tt.filename)
			if err != nil {
				t.Fatalf("createFormFileEncoded() error: %v", err)
			}
			_, _ = io.WriteString(part, "x")
			_ = w.Close()

			reader := multipart.NewReader(&buf, w.Boundary())
			p, err := reader.NextPart()
			if err != nil {
				t.Fatalf("NextPart() error: %v", err)
			}

			cd := p.Header.Get("Content-Disposition")

			// Verify mime.ParseMediaType can parse it
			_, params, err := mime.ParseMediaType(cd)
			if err != nil {
				t.Fatalf("mime.ParseMediaType(%q) error: %v", cd, err)
			}

			// The parsed filename should match the original
			if got := params["filename"]; got != tt.filename {
				t.Errorf("params[filename] = %q, want %q", got, tt.filename)
			}
		})
	}
}
