# Agent Sandbox Go SDK

Go SDK for Tencent Cloud Agent Sandbox, providing:
- Sandbox lifecycle management (create, connect, list, destroy)
- Remote code execution and context management (tool/code)
- Remote command/process management (tool/command)
- Remote filesystem operations (tool/filesystem)

## Documentation

- Usage examples: [docs/examples.md](docs/examples.md)
- SDK reference: [docs/sdk-reference.md](docs/sdk-reference.md)

## Table of Contents

- **Code Sandbox (sandbox/code)**
  - [Sandbox creation](docs/examples.md#1-create-code-sandbox-and-get-three-clients)
  - [Code execution](docs/examples.md#2-run-code-python-etc)
  - [File operations](docs/examples.md#3-filesystem-operations-readwritelistcheckdeleterenanemakedir)
  - [Terminal command execution](docs/examples.md#4-commandprocess-management-foregroundbackgroundinputsignalprocess-list)
  - [Sandbox management](docs/examples.md#5-code-sandbox-list-and-management)
- **Browser Sandbox (sandbox/browser)**
  - Not yet implemented
- **Core Package (sandbox/core)**
  - [Direct creation](docs/examples.md#core-package-sandboxcore)
  - [Connect to existing sandbox](docs/examples.md#core-package-sandboxcore)
  - [List sandboxes](docs/examples.md#core-package-sandboxcore)
  - [Destroy sandbox](docs/examples.md#core-package-sandboxcore)

## Installation

Recommended to use go modules:
```bash
go get github.com/TencentCloudAgentRuntime/ags-go-sdk@latest
```

## Quick Start

The following example demonstrates how to create a sandbox and use the Files/Commands/Code tool clients.

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
	// 1) Initialize AGS Client (recommended)
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

	// 2) Create sandbox and get tool clients
	sb, err := sandboxcode.Create(context.TODO(), "code-interpreter-v1", sandboxcode.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sb.Kill(context.TODO()) }()

	// 3) Use remote code execution
	exec, err := sb.Code.RunCode(context.TODO(), "print('hello')", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Real-time output (optional)
	_, _ = sb.Code.RunCode(context.TODO(), "print('hi')", &code.RunCodeConfig{Language: "python"}, &code.OnOutputConfig{
		OnStdout: func(s string) { log.Print("OUT:", s) },
		OnStderr: func(s string) { log.Print("ERR:", s) },
	})
	log.Printf("stdout=%v results=%d err=%v", exec.Logs.Stdout, len(exec.Results), exec.Error)

	// 4) Basic filesystem operations
	_, err = sb.Files.MakeDir(context.TODO(), "/home/user/demo", nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("sandbox:", sb.SandboxId)
}
```

For more examples and advanced usage, please refer to:
- [docs/examples.md](docs/examples.md)
- [docs/sdk-reference.md](docs/sdk-reference.md)

## Troubleshooting

If you encounter errors, see the [Troubleshooting Guide](docs/troubleshooting.md) ([中文版](docs/troubleshooting-zh.md)) for solutions to common issues including:
- Authentication and credential failures
- Network and connection timeouts
- Sandbox lifecycle errors
- Code execution and filesystem errors

## Prerequisites

- Tencent Cloud account with Agent Sandbox access
- Available Region (example uses ap-guangzhou)
- Go 1.20+ (recommended)

## Environment Variables

Before using the SDK, you need to set the following environment variables:

| Variable | Description |
|----------|-------------|
| `TENCENTCLOUD_SECRET_ID` | Tencent Cloud API SecretId |
| `TENCENTCLOUD_SECRET_KEY` | Tencent Cloud API SecretKey |

You can obtain your SecretId and SecretKey from the [Tencent Cloud Console - API Keys](https://console.cloud.tencent.com/cam/capi).

**Linux/macOS:**
```bash
export TENCENTCLOUD_SECRET_ID="your-secret-id"
export TENCENTCLOUD_SECRET_KEY="your-secret-key"
```

**Windows (PowerShell):**
```powershell
$env:TENCENTCLOUD_SECRET_ID="your-secret-id"
$env:TENCENTCLOUD_SECRET_KEY="your-secret-key"
```

**Windows (CMD):**
```cmd
set TENCENTCLOUD_SECRET_ID=your-secret-id
set TENCENTCLOUD_SECRET_KEY=your-secret-key
```

## Directory Structure

- `sandbox/core`: Sandbox create/connect/list/destroy, Core instance encapsulates basic capabilities
- `sandbox/code`: Convenient aggregation, returns Files, Commands, Code three tool clients
- `tool/code`: Code execution and context management
- `tool/command`: Process/command management
- `tool/filesystem`: Filesystem read/write, directory operations
