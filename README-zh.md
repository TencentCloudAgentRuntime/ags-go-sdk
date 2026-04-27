# Agent Sandbox Go SDK

> **验证分支 —— `verify/sh-shell`。** 本分支及 `v0.1.6-verify.1` tag 将命令包装从 `/bin/bash -l -c` 改为 `/bin/sh -c`，避免每次执行命令都加载 login shell 配置。仅用于验证该改动，未合入 `main`，请勿在生产环境依赖。

面向腾讯云 Agent Sandbox 的 Go 语言 SDK，提供：
- 沙箱生命周期管理（创建、连接、列出、销毁）
- 远程代码执行与上下文管理（tool/code）
- 远程命令/进程管理（tool/command）
- 远程文件系统操作（tool/filesystem）

## 文档

- 使用示例：[docs/examples.md](docs/examples.md)
- SDK 参考：[docs/sdk-reference.md](docs/sdk-reference.md)

## 目录

- **代码沙箱 (sandbox/code)**
  - [沙箱创建](docs/examples.md#1-创建代码沙箱并获取三大客户端)
  - [代码执行](docs/examples.md#2-运行代码python-等)
  - [文件操作](docs/examples.md#3-文件系统操作读写列查删改名建目录)
  - [终端命令执行](docs/examples.md#4-命令进程管理前台后台输入信号进程列表)
  - [沙箱管理](docs/examples.md#5-代码沙箱列表和管理)
- **浏览器沙箱 (sandbox/browser)**
  - 尚未实现
- **核心包 (sandbox/core)**
  - [直接创建](docs/examples.md#核心包-sandboxcore)
  - [连接现有沙箱](docs/examples.md#核心包-sandboxcore)
  - [列出沙箱](docs/examples.md#核心包-sandboxcore)
  - [销毁沙箱](docs/examples.md#核心包-sandboxcore)

## 安装

建议使用 go modules 引用：
```bash
go get github.com/TencentCloudAgentRuntime/ags-go-sdk@latest
```

## 快速开始

以下示例演示如何创建沙箱，并使用 Files/Commands/Code 三个工具客户端。

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
	// 1) 初始化 AGS Client（推荐）
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

	// 2) 创建沙箱并获取工具客户端
	sb, err := sandboxcode.Create(context.TODO(), "code-interpreter-v1", sandboxcode.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sb.Kill(context.TODO()) }()

	// 3) 使用远程代码执行
	exec, err := sb.Code.RunCode(context.TODO(), "print('hello')", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	// 实时输出（可选）
	_, _ = sb.Code.RunCode(context.TODO(), "print('hi')", &code.RunCodeConfig{Language: "python"}, &code.OnOutputConfig{
		OnStdout: func(s string) { log.Print("OUT:", s) },
		OnStderr: func(s string) { log.Print("ERR:", s) },
	})
	log.Printf("stdout=%v results=%d err=%v", exec.Logs.Stdout, len(exec.Results), exec.Error)

	// 4) 基础文件系统操作
	_, err = sb.Files.MakeDir(context.TODO(), "/home/user/demo", nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("sandbox:", sb.SandboxId)
}
```

更多示例与进阶用法，请参阅：
- [docs/examples.md](docs/examples.md)
- [docs/sdk-reference.md](docs/sdk-reference.md)

## 先决条件

- 腾讯云账号与 Agent Sandbox 访问权限
- 可用 Region（示例使用 ap-guangzhou）
- Go 1.20+（推荐）

## 环境变量配置

在使用 SDK 之前，需要设置以下环境变量：

| 变量名 | 说明 |
|--------|------|
| `TENCENTCLOUD_SECRET_ID` | 腾讯云 API 密钥 SecretId |
| `TENCENTCLOUD_SECRET_KEY` | 腾讯云 API 密钥 SecretKey |

您可以在 [腾讯云控制台 - API 密钥管理](https://console.cloud.tencent.com/cam/capi) 获取 SecretId 和 SecretKey。

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

## 目录结构

- `sandbox/core`：沙箱创建/连接/列表/销毁，Core 实例封装基础能力
- `sandbox/code`：便捷聚合，返回 Files、Commands、Code 三个工具客户端
- `tool/code`：代码执行与上下文管理
- `tool/command`：进程/命令管理
- `tool/filesystem`：文件系统读写、目录操作


