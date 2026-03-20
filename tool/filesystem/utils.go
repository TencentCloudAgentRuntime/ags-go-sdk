package filesystem

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/connection"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/pb/filesystem"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/pb/filesystem/filesystemconnect"
)

// newHttpClient creates an HTTP client with proxy support
func newHttpClient(config *connection.Config) *http.Client {
	httpClient := &http.Client{}
	if config.Proxy != nil {
		httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(config.Proxy),
		}
	}
	return httpClient
}

// newHttpRequestWithHeaders creates an HTTP request with configured headers
func newHttpRequestWithHeaders(ctx context.Context, method, url string, body io.Reader, cfg *connection.Config, user string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	for k, vv := range cfg.Headers {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}
	if cfg.AccessToken != "" {
		req.Header.Set("X-Access-Token", cfg.AccessToken)
	}
	req.Header.Set(
		"Authorization",
		fmt.Sprintf(
			"Basic %v",
			base64.StdEncoding.EncodeToString(
				[]byte(user),
			),
		),
	)
	return req, err
}

// newRpcClient creates a Filesystem RPC client
func newRpcClient(config *connection.Config) filesystemconnect.FilesystemClient {
	httpClient := newHttpClient(config)
	host := fmt.Sprintf("%v://%v", config.GetScheme(), config.Domain)
	cli := filesystemconnect.NewFilesystemClient(
		httpClient, host, connect.WithProtoJSON(),
	)
	return cli
}

// newRPCRequestWithHeaders creates an RPC request with configured headers
func newRPCRequestWithHeaders[T any](message *T, config *connection.Config, user string) *connect.Request[T] {
	req := connect.NewRequest(message)
	if config.Headers != nil {
		for k, vv := range config.Headers {
			for _, v := range vv {
				req.Header().Add(k, v)
			}
		}
	}
	req.Header().Set("X-Access-Token", config.AccessToken)
	// Temporary Header To Pass Gateway
	req.Header().Set(
		"Authorization",
		fmt.Sprintf(
			"Basic %v",
			base64.StdEncoding.EncodeToString(
				[]byte(user+":"),
			),
		),
	)
	return req
}

// mapEntryInfo maps proto EntryInfo to client EntryInfo
func mapEntryInfo(p *filesystem.EntryInfo) *EntryInfo {
	var tptr *FileType
	switch p.GetType() {
	case filesystem.FileType_FILE_TYPE_FILE:
		t := File
		tptr = &t
	case filesystem.FileType_FILE_TYPE_DIRECTORY:
		t := Dir
		tptr = &t
	default:
	}

	out := &EntryInfo{
		WriteInfo: WriteInfo{
			Name: p.GetName(),
			Type: tptr,
			Path: p.GetPath(),
		},
		Size:        p.GetSize(),
		Mode:        int(p.GetMode()),
		Permissions: p.GetPermissions(),
		Owner:       p.GetOwner(),
		Group:       p.GetGroup(),
	}

	if ts := p.GetModifiedTime(); ts != nil {
		out.ModifiedTime = ts.AsTime()
	}

	if st := p.GetSymlinkTarget(); st != "" {
		out.SymlinkTarget = &st
	}
	return out
}

// isASCII reports whether s contains only ASCII characters.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

// percentEncodeRFC5987 encodes a string according to RFC 5987 (attr-char).
// Only unreserved characters (ALPHA / DIGIT / "!" / "#" / "$" / "&" / "+" /
// "-" / "." / "^" / "_" / "`" / "|" / "~") are left unencoded;
// everything else (including "{", "}", "%", etc.) is percent-encoded.
func percentEncodeRFC5987(s string) string {
	var b strings.Builder
	b.Grow(len(s) * 3) // worst-case: every byte is encoded
	for i := 0; i < len(s); i++ {
		c := s[i]
		// attr-char unreserved set from RFC 5987 §3.2.1
		if (c >= 'A' && c <= 'Z') ||
			(c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') ||
			c == '!' || c == '#' || c == '$' || c == '&' ||
			c == '+' || c == '-' || c == '.' || c == '^' ||
			c == '_' || c == '`' || c == '|' || c == '~' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}
