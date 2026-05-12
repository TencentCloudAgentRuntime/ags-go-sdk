# Usage Examples

This page provides common examples from scratch: how to create sandboxes, execute code, read/write files, run commands and manage processes.

## Prerequisites

- Tencent Cloud account with Agent Sandbox access
- Go 1.22+
- Credentials configured (SDK Client initialization recommended)

## Import Path Conventions

- Code sandbox: `github.com/TencentCloudAgentRuntime/ags-go-sdk/sandbox/code`
- Browser sandbox: `github.com/TencentCloudAgentRuntime/ags-go-sdk/sandbox/browser`
- Core package: `github.com/TencentCloudAgentRuntime/ags-go-sdk/sandbox/core`
- Code execution: `github.com/TencentCloudAgentRuntime/ags-go-sdk/tool/code`
- Command/process: `github.com/TencentCloudAgentRuntime/ags-go-sdk/tool/command`
- Filesystem: `github.com/TencentCloudAgentRuntime/ags-go-sdk/tool/filesystem`

# Code Sandbox (sandbox/code)

Code sandbox provides high-level APIs suitable for most code execution, file operations, and command management scenarios.

## 1. Create Code Sandbox and Get Three Clients

Recommended to inject via Tencent Cloud SDK's AGS Client, then create sandbox; the returned Sandbox aggregates Files/Commands/Code three clients.

```go
package main

import (
	"context"
	"log"
	"os"

	sandboxcode "github.com/TencentCloudAgentRuntime/ags-go-sdk/sandbox/code"

	ags "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ags/v20250920"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

func main() {
	ctx := context.Background()

	// Initialize AGS Client (recommended)
	cred := &common.Credential{
		SecretId:  os.Getenv("TENCENTCLOUD_SECRET_ID"),
		SecretKey: os.Getenv("TENCENTCLOUD_SECRET_KEY"),
	}
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "ags.tencentcloudapi.com"
	client, err := ags.NewClient(cred, "ap-guangzhou", cpf)
	if err != nil {
		log.Fatal(err)
	}

	// Create sandbox (tool configured by server, e.g., "code-interpreter-v1")
	sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sb.Kill(ctx) }()

	log.Println("sandbox:", sb.SandboxId)
}
```

**Key Points**

- `sandbox/code.Create` wraps `core.Create`, automatically initializes Files/Commands/Code three clients, domain/authentication uniformly passed through.
- `sb.Kill(ctx)` is used to destroy and release; you can also use `sb.SetTimeoutSeconds(ctx, 600)` to set expiration time.

## 2. Run Code (Python, etc.)

`RunCode` supports directly passing code and optional language; if neither ContextId nor Language is provided, default Language=python.

```go
exec, err := sb.Code.RunCode(ctx, "print('hello')", nil, nil)
if err != nil {
	log.Fatal(err)
}
log.Printf("results=%d, stdout=%v, error=%v", len(exec.Results), exec.Logs.Stdout, exec.Error)
```

Minimal real-time output (onOutput) example:

```go
_, err := sb.Code.RunCode(ctx, "import time\nfor i in range(3):\n print(i); time.sleep(1)", &code.RunCodeConfig{Language: "python"}, &code.OnOutputConfig{
	OnStdout: func(s string) { log.Print("OUT:", s) },
	OnStderr: func(s string) { log.Print("ERR:", s) },
})
if err != nil {
	log.Fatal(err)
}
```

### Using Persistent Code Context (Reusable Execution Directory/Session)

```go
// Create context (can specify language and working directory)
ctxResp, err := sb.Code.CreateCodeContext(ctx, &code.CreateCodeContextConfig{
	Cwd:      "/home/user/project",
	Language: "python",
})
if err != nil {
	log.Fatal(err)
}

// Execute with bound context
exec, err := sb.Code.RunCode(ctx, "print('stateful run')", &code.RunCodeConfig{
	ContextId: ctxResp.Id,
}, nil)
if err != nil {
	log.Fatal(err)
}
```

## 3. Filesystem Operations (Read/Write/List/Check/Delete/Rename/MakeDir)

```go
// Write file
_, err := sb.Files.Write(ctx, "/home/user/demo.txt", bytes.NewBufferString("hello"), &filesystem.WriteConfig{User: "user"})
if err != nil {
	log.Fatal(err)
}

// Read file
r, err := sb.Files.Read(ctx, "/home/user/demo.txt", &filesystem.ReadConfig{User: "user"})
if err != nil {
	log.Fatal(err)
}
data, _ := io.ReadAll(r)
log.Println("content:", string(data))

// List directory
entries, err := sb.Files.List(ctx, "/home/user", &filesystem.ListConfig{Depth: 1, User: "user"})
if err != nil {
	log.Fatal(err)
}
for _, e := range entries {
	log.Printf("[%s] %s size=%d", func() string {
		if e.Type != nil {
			return string(*e.Type)
		}
		return "unknown"
	}(), e.Path, e.Size)
}

// Get info
info, err := sb.Files.GetInfo(ctx, "/home/user/demo.txt", &filesystem.GetInfoConfig{User: "user"})
if err != nil {
	log.Fatal(err)
}
log.Printf("owner=%s perms=%s", info.Owner, info.Permissions)

// Check existence
ok, err := sb.Files.Exists(ctx, "/home/user/demo.txt", &filesystem.ExistsConfig{User: "user"})
if err != nil {
	log.Fatal(err)
}
log.Println("exists:", ok)

// Rename
err = sb.Files.Rename(ctx, "/home/user/demo.txt", "/home/user/demo2.txt", &filesystem.RenameConfig{User: "user"})
if err != nil {
	log.Fatal(err)
}

// Create directory
created, err := sb.Files.MakeDir(ctx, "/home/user/newdir", &filesystem.MakeDirConfig{User: "user"})
if err != nil {
	log.Fatal(err)
}
log.Println("dir created:", created)

// Delete
err = sb.Files.Remove(ctx, "/home/user/demo2.txt", &filesystem.RemoveConfig{User: "user"})
if err != nil {
	log.Fatal(err)
}
```

**Note**

- All `*Config` `User` fields only allow "user" or "root"; defaults to "user" when empty.
- Access paths must use sandbox internal filesystem paths.

## 4. Command/Process Management (Foreground/Background, Input/Signal, Process List)

### Convenient Foreground Run (Auto-aggregates stdout/stderr)

```go
res, err := sb.Commands.Run(ctx, "echo hello && uname -a", &command.ProcessConfig{
	User: "user", // User in process, defaults to "user"
}, &command.OnOutputConfig{
	OnStdout: func(b []byte) { log.Printf("STDOUT: %s", string(b)) },
	OnStderr: func(b []byte) { log.Printf("STDERR: %s", string(b)) },
})
if err != nil {
	log.Fatal(err)
}
log.Printf("exit=%d, stdout=%q, stderr=%q, err=%v", res.ExitCode, string(res.Stdout), string(res.Stderr), res.Error)
```

### Background Start + Wait

```go
h, err := sb.Commands.Start(ctx, "sleep 5; echo done", &command.ProcessConfig{
	User: "user",
}, &command.OnOutputConfig{
	OnStdout: func(b []byte) { log.Print("OUT:", string(b)) },
	OnStderr: func(b []byte) { log.Print("ERR:", string(b)) },
})
if err != nil {
	log.Fatal(err)
}

ret, err := h.Wait(ctx)
if err != nil {
	log.Fatal(err)
}
log.Printf("exit=%d, error=%v", ret.ExitCode, ret.Error)
```

### Send Input and Signal

```go
// Create process
h, err := sb.Commands.Start(ctx, "cat", &command.ProcessConfig{
	User: "user",
}, &command.OnOutputConfig{
	OnStdout: func(b []byte) { log.Print("OUT:", string(b)) },
	OnStderr: func(b []byte) { log.Print("ERR:", string(b)) },
})
if err != nil {
	log.Fatal(err)
}

// Send input to stdin at once
_ = h.SendInput(ctx, h.Pid, []byte("hello\n"))

// Send signal (SIGTERM=15 SIGKILL=9, currently only these two signals are accepted)
_ = h.SendSignal(ctx, h.Pid, 15)

// Or directly send SIGKILL
_ = h.Kill(ctx)
```

### List Running Processes

```go
ps, err := sb.Commands.List(ctx)
if err != nil {
	log.Fatal(err)
}
for _, p := range ps {
	log.Printf("pid=%d cmd=%s args=%v cwd=%v", p.Pid, p.Cmd, p.Args, p.Cwd)
}
```

## 5. Code Sandbox List and Management

```go
// List all sandboxes
instances, err := sandboxcode.List(ctx, sandboxcode.WithClient(client))
if err != nil {
	log.Fatal(err)
}
for _, ins := range instances {
	log.Println(*ins.InstanceId, *ins.Status)
}

// Connect to existing code sandbox
sb, err := sandboxcode.Connect(ctx, "SBOX-XXXX", sandboxcode.WithClient(client))
if err != nil {
	log.Fatal(err)
}
defer sb.Kill(ctx)
```

**Tips**

- `sandbox/code.Create` has already constructed the connection domain and AccessToken for you and assigned them to various tool clients, no manual management required.
- If you need custom proxy or extra request headers, you can modify `sb.Core.ConnectionConfig` after connection, then recreate clients as needed.

# Browser Sandbox (sandbox/browser)

**Note: Browser sandbox package is not yet implemented.**

Browser sandbox will provide Web automation, browser control, and Web-based testing environment APIs.

# Core Package (sandbox/core)

Core package provides low-level APIs suitable for custom sandbox implementations or scenarios requiring maximum control.

## 1. Create Sandbox Directly Using Core API

```go
import "github.com/TencentCloudAgentRuntime/ags-go-sdk/sandbox/core"

// Create sandbox
coreInstance, err := core.Create(ctx, "code-interpreter-v1",
	core.WithCredential(cred),
	core.WithRegion("ap-guangzhou"),
)
if err != nil {
	log.Fatal(err)
}
defer coreInstance.Kill(ctx)

log.Println("Sandbox ID:", coreInstance.SandboxId)
log.Println("Connection Config:", coreInstance.ConnectionConfig)
```

## 2. Connect to Existing Sandbox

```go
// Connect to specified sandbox (only gets token, does not initialize tool clients)
coreInstance, err := core.Connect(ctx, "SBOX-XXXX", 
	core.WithCredential(cred),
	core.WithRegion("ap-guangzhou"),
)
if err != nil {
	log.Fatal(err)
}
defer coreInstance.Kill(ctx)
```

## 3. List All Sandboxes

```go
// List all sandbox instances
instances, err := core.List(ctx,
	core.WithCredential(cred),
	core.WithRegion("ap-guangzhou"),
)
if err != nil {
	log.Fatal(err)
}

for _, ins := range instances {
	log.Printf("Sandbox: %s, Status: %s", *ins.InstanceId, *ins.Status)
}
```

## 4. Destroy Sandbox

```go
// Directly destroy specified sandbox
err := core.Kill(ctx, "SBOX-XXXX",
	core.WithCredential(cred),
	core.WithRegion("ap-guangzhou"),
)
if err != nil {
	log.Fatal(err)
}
```

## 5. Core Instance Methods

```go
// Get sandbox host address
host := coreInstance.GetHost(8080)
log.Println("Sandbox host:", host)

// Set timeout
err := coreInstance.SetTimeoutSeconds(ctx, 600)
if err != nil {
	log.Fatal(err)
}

// Get sandbox detailed info
info, err := coreInstance.GetInfo(ctx)
if err != nil {
	log.Fatal(err)
}
log.Printf("Sandbox info: %+v", info)
```

## Use Cases

**Use Code Sandbox (sandbox/code) when you need:**

- Code execution and file operations
- Command and process management
- Out-of-the-box high-level APIs

**Use Browser Sandbox (sandbox/browser) when you need:**

- Web automation and browser control (not yet implemented)
- Web-based testing environment

**Use Core Package (sandbox/core) when you need:**

- Implement custom sandbox types
- Direct control over sandbox management
- Maximum flexibility and control
- Don't care about specific sandbox type, only need basic sandbox operations
