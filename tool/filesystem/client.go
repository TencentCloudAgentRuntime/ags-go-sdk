package filesystem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/TencentCloudAgentRuntime/ags-go-sdk/connection"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/pb/filesystem/filesystemconnect"

	"connectrpc.com/connect"
	fsproto "github.com/TencentCloudAgentRuntime/ags-go-sdk/pb/filesystem"
)

// Client provides filesystem operations in the sandbox
type Client struct {
	config     *connection.Config
	httpClient *http.Client
	rpcClient  filesystemconnect.FilesystemClient
}

// New creates a new filesystem client with the given configuration
func New(config *connection.Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	return &Client{
		config:     config,
		httpClient: newHttpClient(config),
		rpcClient:  newRpcClient(config),
	}, nil
}

// autoCloseReader automatically closes the reader when EOF is reached
type autoCloseReader struct {
	rc io.ReadCloser
}

func (r *autoCloseReader) Read(p []byte) (int, error) {
	n, err := r.rc.Read(p)
	if err == io.EOF {
		_ = r.rc.Close()
	}
	return n, err
}

// Read reads the contents of a file at the given path
func (client *Client) Read(ctx context.Context, path string, config *ReadConfig) (io.Reader, error) {
	if client == nil || client.config == nil || client.httpClient == nil {
		return nil, fmt.Errorf("filesystem client not initialized")
	}
	if client.config.Domain == "" {
		return nil, fmt.Errorf("connection domain is empty")
	}
	if path == "" {
		return nil, fmt.Errorf("path is empty")
	}

	config, err := config.valid()
	if err != nil {
		return nil, err
	}
	sandboxFileURL := url.URL{
		Scheme: client.config.GetScheme(),
		Host:   client.config.Domain,
		Path:   "/files",
	}
	query := sandboxFileURL.Query()
	query.Set("path", path)
	user := UserDefault
	if config != nil && config.User != "" {
		user = config.User
	}
	query.Set("username", user)
	sandboxFileURL.RawQuery = query.Encode()

	req, err := newHttpRequestWithHeaders(ctx, http.MethodGet, sandboxFileURL.String(), nil, client.config, config.User)
	if err != nil {
		return nil, err
	}
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("%d: failed to read response body: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("%d: %s", resp.StatusCode, string(b))
	}
	return &autoCloseReader{rc: resp.Body}, nil
}

// Write writes data to a file at the given path
func (client *Client) Write(ctx context.Context, path string, data io.Reader, config *WriteConfig) (*WriteInfo, error) {
	if client == nil || client.config == nil || client.httpClient == nil {
		return nil, fmt.Errorf("filesystem client not initialized")
	}
	if client.config.Domain == "" {
		return nil, fmt.Errorf("connection domain is empty")
	}
	if path == "" {
		return nil, fmt.Errorf("path is empty")
	}

	config, err := config.valid()
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	multipartWriter := multipart.NewWriter(&body)
	// 使用自定义方法创建 form file，支持非 ASCII 文件名编码
	part, err := createFormFileEncoded(multipartWriter, "file", path)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, data)
	if err != nil {
		return nil, err
	}
	err = multipartWriter.Close()
	if err != nil {
		return nil, err
	}

	sandboxFileURL := url.URL{
		Scheme: client.config.GetScheme(),
		Host:   client.config.Domain,
		Path:   "/files",
	}
	query := sandboxFileURL.Query()
	user := UserDefault
	if config.User != "" {
		user = config.User
	}
	query.Set("username", user)
	query.Set("path", path)
	sandboxFileURL.RawQuery = query.Encode()

	req, err := newHttpRequestWithHeaders(ctx, http.MethodPost, sandboxFileURL.String(), &body, client.config, config.User)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("%d: failed to read response body: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("%d: %s", resp.StatusCode, string(b))
	}

	var infos []WriteInfo
	if err := json.NewDecoder(resp.Body).Decode(&infos); err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, fmt.Errorf("expected write info in response")
	}
	info := infos[0]
	return &info, nil
}

// List lists entries in a directory
func (client *Client) List(ctx context.Context, path string, opts *ListConfig) ([]EntryInfo, error) {
	if client == nil || client.config == nil {
		return nil, fmt.Errorf("filesystem client not initialized")
	}
	if client.config.Domain == "" {
		return nil, fmt.Errorf("connection domain is empty")
	}
	if path == "" {
		return nil, fmt.Errorf("path is empty")
	}
	opts, err := opts.valid()
	if err != nil {
		return nil, err
	}
	rpcClient := client.rpcClient

	depth := 1
	if opts.Depth > 0 {
		depth = opts.Depth
	}
	if depth < 0 {
		depth = 0
	}
	req := newRPCRequestWithHeaders(&fsproto.ListDirRequest{
		Path:  path,
		Depth: uint32(depth), //nolint:gosec // depth is validated above
	}, client.config, opts.User)
	resp, err := rpcClient.ListDir(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("empty response from ListDir")
	}
	entries := resp.Msg.GetEntries()
	out := make([]EntryInfo, 0, len(entries))
	for _, e := range entries {
		if e == nil {
			continue
		}
		m := mapEntryInfo(e)
		if m != nil {
			out = append(out, *m)
		}
	}
	return out, nil
}

// Exists checks if a file or directory exists at the given path
func (client *Client) Exists(ctx context.Context, path string, opts *ExistsConfig) (bool, error) {
	if client == nil || client.config == nil {
		return false, fmt.Errorf("filesystem client not initialized")
	}
	if client.config.Domain == "" {
		return false, fmt.Errorf("connection domain is empty")
	}
	if path == "" {
		return false, fmt.Errorf("path is empty")
	}
	opts, err := opts.valid()
	if err != nil {
		return false, err
	}
	rpcClient := client.rpcClient
	req := newRPCRequestWithHeaders(&fsproto.StatRequest{Path: path}, client.config, opts.User)
	_, err = rpcClient.Stat(ctx, req)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetInfo returns detailed information about a file or directory
func (client *Client) GetInfo(ctx context.Context, path string, opts *GetInfoConfig) (*EntryInfo, error) {
	if client == nil || client.config == nil {
		return nil, fmt.Errorf("filesystem client not initialized")
	}
	if client.config.Domain == "" {
		return nil, fmt.Errorf("connection domain is empty")
	}
	if path == "" {
		return nil, fmt.Errorf("path is empty")
	}
	opts, err := opts.valid()
	if err != nil {
		return nil, err
	}
	rpcClient := client.rpcClient
	req := newRPCRequestWithHeaders(&fsproto.StatRequest{Path: path}, client.config, opts.User)
	resp, err := rpcClient.Stat(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Entry == nil {
		return nil, fmt.Errorf("empty response from Stat")
	}
	fileEntry := resp.Msg.Entry
	out := mapEntryInfo(fileEntry)
	return out, nil
}

// Remove deletes a file or directory at the given path
func (client *Client) Remove(ctx context.Context, path string, opts *RemoveConfig) error {
	if client == nil || client.config == nil {
		return fmt.Errorf("filesystem client not initialized")
	}
	if client.config.Domain == "" {
		return fmt.Errorf("connection domain is empty")
	}
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	opts, err := opts.valid()
	if err != nil {
		return err
	}
	rpcClient := client.rpcClient
	req := newRPCRequestWithHeaders(&fsproto.RemoveRequest{Path: path}, client.config, opts.User)
	_, err = rpcClient.Remove(ctx, req)
	return err
}

// Rename moves a file or directory from oldPath to newPath
func (client *Client) Rename(ctx context.Context, oldPath, newPath string, opts *RenameConfig) error {
	if client == nil || client.config == nil {
		return fmt.Errorf("filesystem client not initialized")
	}
	if client.config.Domain == "" {
		return fmt.Errorf("connection domain is empty")
	}
	if oldPath == "" || newPath == "" {
		return fmt.Errorf("source or destination path is empty")
	}
	opts, err := opts.valid()
	if err != nil {
		return err
	}
	rpcClient := client.rpcClient
	req := newRPCRequestWithHeaders(&fsproto.MoveRequest{
		Source:      oldPath,
		Destination: newPath,
	}, client.config, opts.User)
	_, err = rpcClient.Move(ctx, req)
	return err
}

// MakeDir creates a directory at the given path
func (client *Client) MakeDir(ctx context.Context, path string, opts *MakeDirConfig) (bool, error) {
	if client == nil || client.config == nil {
		return false, fmt.Errorf("filesystem client not initialized")
	}
	if client.config.Domain == "" {
		return false, fmt.Errorf("connection domain is empty")
	}
	if path == "" {
		return false, fmt.Errorf("path is empty")
	}
	opts, err := opts.valid()
	if err != nil {
		return false, err
	}
	rpcClient := client.rpcClient
	req := newRPCRequestWithHeaders(&fsproto.MakeDirRequest{Path: path}, client.config, opts.User)
	if opts.User != "" {
		req.Header().Set("X-User", opts.User)
	}
	resp, err := rpcClient.MakeDir(ctx, req)
	if err != nil {
		return false, err
	}
	if resp == nil || resp.Msg == nil {
		return false, fmt.Errorf("empty response from MakeDir")
	}
	return resp.Msg.Entry != nil, nil
}

// createFormFileEncoded creates a form file part with proper encoding for non-ASCII filenames.
//
// When the filename is pure ASCII, it uses a simple quoted `filename="..."` parameter.
// When the filename contains non-ASCII characters, it emits both `filename` (with ASCII
// fallback) and `filename*` (RFC 5987 UTF-8 percent-encoded) to maximize server compatibility.
//
// This replaces the previous implementation that relied on mime.FormatMediaType, which had a
// known round-trip issue: characters like '{' and '}' were not percent-encoded in the
// filename* value, causing mime.ParseMediaType on the server side to fail with
// "mime: invalid media parameter" when the filename contained both non-ASCII characters
// and curly braces.
func createFormFileEncoded(w *multipart.Writer, fieldname, filename string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)

	var disposition string
	if isASCII(filename) {
		// Pure ASCII: use simple quoted filename parameter.
		// Escape any backslash or double-quote in the value.
		escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(filename)
		disposition = fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldname, escaped)
	} else {
		// Non-ASCII: emit both filename (ASCII fallback) and filename* (RFC 5987).
		// The ASCII fallback replaces non-ASCII bytes with underscores so that servers
		// which do not support RFC 5987 still receive a usable (if degraded) name.
		var asciiFallback strings.Builder
		for i := 0; i < len(filename); i++ {
			if filename[i] > 127 {
				asciiFallback.WriteByte('_')
			} else {
				asciiFallback.WriteByte(filename[i])
			}
		}
		escapedFallback := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(asciiFallback.String())
		encoded := percentEncodeRFC5987(filename)
		disposition = fmt.Sprintf(`form-data; name="%s"; filename="%s"; filename*=UTF-8''%s`,
			fieldname, escapedFallback, encoded)
	}

	h.Set("Content-Disposition", disposition)
	h.Set("Content-Type", "application/octet-stream")
	return w.CreatePart(h)
}
