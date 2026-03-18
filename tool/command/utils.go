package command

import (
	"fmt"
	"io"
	"net/http"

	"github.com/TencentCloudAgentRuntime/ags-go-sdk/pb/process"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/pb/process/processconnect"

	"connectrpc.com/connect"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/connection"
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

// newRpcClient creates a Process RPC client
func newRpcClient(config *connection.Config) processconnect.ProcessClient {
	httpClient := newHttpClient(config)
	host := fmt.Sprintf("%v://%v", config.GetScheme(), config.Domain)
	cli := processconnect.NewProcessClient(
		httpClient, host, connect.WithProtoJSON(),
	)
	return cli
}

// newRequestWithHeaders creates a request with configured headers
func newRequestWithHeaders[T any](message *T, config *connection.Config) *connect.Request[T] {
	req := connect.NewRequest(message)
	if config.Headers != nil {
		for k, vv := range config.Headers {
			for _, v := range vv {
				req.Header().Add(k, v)
			}
		}
	}
	req.Header().Set("X-Access-Token", config.AccessToken)
	return req
}

// buildProcessConfig builds a ProcessConfig for the command
func buildProcessConfig(cmd string, config *ProcessConfig) *process.ProcessConfig {
	pc := &process.ProcessConfig{
		Cmd:  "/bin/bash",
		Args: []string{"-l", "-c", cmd},
	}
	if config != nil {
		if config.Envs != nil {
			pc.Envs = config.Envs
		}
		if config.Cwd != nil {
			pc.Cwd = config.Cwd
		}
		if len(config.Args) > 0 {
			// When explicit args are provided, use cmd as the executable directly
			// instead of wrapping it through /bin/bash -c, to avoid argument
			// misinterpretation (e.g. "bash" + ["-lc", "ls /"] would hang).
			pc.Cmd = cmd
			pc.Args = config.Args
		}
	}
	return pc
}

// mapProcessInfo maps proto ProcessInfo to client ProcessInfo
func mapProcessInfo(p *process.ProcessInfo) *ProcessInfo {
	if p == nil || p.Config == nil {
		return nil
	}
	var tagPtr *string
	if t := p.GetTag(); t != "" {
		tagPtr = &t
	}
	var cwdPtr *string
	if c := p.Config.GetCwd(); c != "" {
		cwdPtr = &c
	}
	return &ProcessInfo{
		Pid:  p.GetPid(),
		Tag:  tagPtr,
		Cmd:  p.Config.GetCmd(),
		Args: append([]string{}, p.Config.GetArgs()...),
		Envs: func(m map[string]string) map[string]string {
			if m == nil {
				return nil
			}
			out := make(map[string]string, len(m))
			for k, v := range m {
				out[k] = v
			}
			return out
		}(p.Config.GetEnvs()),
		Cwd: cwdPtr,
	}
}

// eventStream provides a unified interface for consuming process events
type eventStream interface {
	next() (*process.ProcessEvent, error)
	close() error
}

// startStream wraps the Start stream to implement eventStream interface
type startStream struct {
	stream *connect.ServerStreamForClient[process.StartResponse]
}

// next reads the next process event from the server stream.
// It returns the event when available, io.EOF when the stream is exhausted,
// or an error if the underlying stream reports one.
func (stream *startStream) next() (*process.ProcessEvent, error) {
	if stream == nil || stream.stream == nil {
		return nil, fmt.Errorf("no server stream")
	}
	if stream.stream.Receive() {
		msg := stream.stream.Msg()
		if msg != nil {
			return msg.Event, nil
		}
	}
	if err := stream.stream.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

// Close the receive side of the stream.
// Close is non-blocking. To gracefully close the stream and allow for connection
// resuse ensure all messages have been received before calling Close.
// All messages are received when Receive returns false.
func (stream *startStream) close() error {
	return stream.stream.Close()
}

// connectStream wraps the Connect stream to implement eventStream interface
type connectStream struct {
	stream *connect.ServerStreamForClient[process.ConnectResponse]
}

// next reads the next process event from the server stream.
// It returns the event when available, io.EOF when the stream is exhausted,
// or an error if the underlying stream reports one.
func (stream *connectStream) next() (*process.ProcessEvent, error) {
	if stream == nil || stream.stream == nil {
		return nil, fmt.Errorf("no server stream")
	}
	if stream.stream.Receive() {
		msg := stream.stream.Msg()
		if msg != nil {
			return msg.Event, nil
		}
	}
	if err := stream.stream.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

// Close the receive side of the stream.
// Close is non-blocking. To gracefully close the stream and allow for connection
// resuse ensure all messages have been received before calling Close.
// All messages are received when Receive returns false.
func (stream *connectStream) close() error {
	return stream.stream.Close()
}
