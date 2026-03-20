package code

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/TencentCloudAgentRuntime/ags-go-sdk/connection"
)

// Client provides code execution capabilities
type Client struct {
	config     *connection.Config
	httpClient *http.Client
}

// New creates a new code client with the given configuration
func New(config *connection.Config) *Client {
	return &Client{
		config:     config,
		httpClient: newHttpClient(config),
	}
}

// RunCode executes code and returns the execution result, with optional real-time onOutput callbacks
func (client *Client) RunCode(ctx context.Context, code string, config *RunCodeConfig, onOutput *OnOutputConfig) (*Execution, error) {
	if client == nil || client.config == nil || client.httpClient == nil {
		return nil, fmt.Errorf("code client not initialized")
	}
	if client.config.Domain == "" {
		return nil, fmt.Errorf("connection domain is empty")
	}
	if config != nil && config.Language != "" && config.ContextId != "" {
		return nil, fmt.Errorf("cannot use RunCode with both contextId and language")
	}
	config = config.loadDefault()
	onOutput = onOutput.loadDefault()

	// Build URL: {scheme}://{domain}/execute
	sandboxExecuteURL := url.URL{
		Scheme: client.config.GetScheme(),
		Host:   client.config.Domain,
		Path:   "/execute",
	}

	// Prepare request body
	var langPtr *string
	if config.Language != "" {
		langPtr = &config.Language
	}
	var ctxIdPtr *string
	if config.ContextId != "" {
		ctxIdPtr = &config.ContextId
	}
	body := executeRequest{
		Code:      code,
		ContextId: ctxIdPtr,
		Language:  langPtr,
		EnvVars:   config.Envs,
	}
	bin, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	// Build HTTP request
	req, err := newHttpRequestWithHeaders(ctx, http.MethodPost, sandboxExecuteURL.String(), bytes.NewReader(bin), client.config)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request and parse line by line
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

	exec := &Execution{
		Results: []Result{},
		Logs:    Logs{Stdout: []string{}, Stderr: []string{}},
	}

	// Create a pipe to make the scanner cancellable
	pipeReader, pipeWriter := io.Pipe()
	done := make(chan struct{})
	defer close(done)

	// Copy data from response body to pipe in a separate goroutine
	go func() {
		defer pipeWriter.Close()
		_, err := io.Copy(pipeWriter, resp.Body)
		if err != nil {
			pipeWriter.CloseWithError(err)
		}
	}()

	// Monitor context cancellation in another goroutine
	go func() {
		select {
		case <-ctx.Done():
			pipeWriter.CloseWithError(ctx.Err())
		case <-done:
			// RunCode finished normally, exit goroutine
			return
		}
	}()

	scanner := bufio.NewScanner(pipeReader)
	// Use configurable max buffer size for large results
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, config.MaxBufferSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		// Each line is a JSON object
		var raw map[string]any
		if err := json.Unmarshal(line, &raw); err != nil {
			// Skip non-JSON lines
			continue
		}
		typ, ok := raw["type"].(string)
		if !ok {
			continue
		}
		switch typ {
		case "result":
			// Map the entire line to Result, preserving all fields
			var r Result
			b2, err := json.Marshal(raw)
			if err != nil {
				continue
			}
			if err := json.Unmarshal(b2, &r); err != nil {
				continue
			}
			exec.Results = append(exec.Results, r)
		case "stdout":
			if txt, ok := raw["text"].(string); ok {
				exec.Logs.Stdout = append(exec.Logs.Stdout, txt)
				onOutput.OnStdout(txt)
			}
		case "stderr":
			if txt, ok := raw["text"].(string); ok {
				exec.Logs.Stderr = append(exec.Logs.Stderr, txt)
				onOutput.OnStderr(txt)
			}
		case "error":
			var e ExecutionError
			b2, err := json.Marshal(raw)
			if err != nil {
				continue
			}
			if err := json.Unmarshal(b2, &e); err != nil {
				continue
			}
			exec.Error = &e
		case "number_of_executions":
			if cnt, ok := raw["execution_count"].(float64); ok {
				c := int(cnt)
				exec.ExecutionCount = &c
			}
		default:
			// Ignore other types (e.g., keepalive)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return exec, nil
}

// CreateCodeContext creates a code execution context
func (client *Client) CreateCodeContext(ctx context.Context, config *CreateCodeContextConfig) (*CodeContext, error) {
	if client == nil || client.config == nil || client.httpClient == nil {
		return nil, fmt.Errorf("code client not initialized")
	}
	if client.config.Domain == "" {
		return nil, fmt.Errorf("connection domain is empty")
	}
	config = config.loadDefault()

	sandboxContextsURL := url.URL{
		Scheme: client.config.GetScheme(),
		Host:   client.config.Domain,
		Path:   "/contexts",
	}
	var langPtr *string
	if config.Language != "" {
		langPtr = &config.Language
	}
	var cwdPtr *string
	if config.Cwd != "" {
		cwdPtr = &config.Cwd
	}
	reqBody := createContextRequest{
		Language: langPtr,
		Cwd:      cwdPtr,
	}
	bin, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := newHttpRequestWithHeaders(ctx, http.MethodPost, sandboxContextsURL.String(), bytes.NewReader(bin), client.config)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

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

	var out CodeContext
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
