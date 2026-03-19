package example_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	sandboxcode "github.com/TencentCloudAgentRuntime/ags-go-sdk/sandbox/code"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/sandbox/core"

	"github.com/TencentCloudAgentRuntime/ags-go-sdk/tool/code"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/tool/command"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/tool/filesystem"

	ags "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ags/v20250920"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

// 1. 创建沙箱并获取三大客户端
func Example_createSandbox() {
	ctx := context.Background()

	// 初始化 AGS Client（推荐）
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

	// 创建沙箱（tool 由服务端配置，比如 "code-interpreter-v1"）
	sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sb.Kill(ctx) }()

	log.Println("sandbox:", sb.SandboxId)
	// Output:
}

// 2. 运行代码（Python 等）——基础运行
func Example_runCode_basic() {
	ctx := context.Background()

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

	sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sb.Kill(ctx) }()

	exec, err := sb.Code.RunCode(ctx, "print('hello')", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("results=%d, stdout=%v, error=%v", len(exec.Results), exec.Logs.Stdout, exec.Error)
	// Output:
}

// 2. 运行代码（Python 等）——使用持久化代码上下文
func Example_runCode_withContext() {
	ctx := context.Background()

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

	sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sb.Kill(ctx) }()

	// 创建上下文（可指定语言与工作目录）
	ctxResp, err := sb.Code.CreateCodeContext(ctx, &code.CreateCodeContextConfig{
		Cwd:      "/home/user/project",
		Language: "python",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 绑定上下文执行
	exec, err := sb.Code.RunCode(ctx, "print('stateful run')", &code.RunCodeConfig{
		ContextId: ctxResp.Id,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	_ = exec
	// Output:
}

// 2. 运行代码（Python 等）——onOutput 实时回调
func Example_runCode_onOutput() {
	ctx := context.Background()

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

	sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sb.Kill(ctx) }()

	codeStr := "import sys\nprint('hello-out')\nprint('hello-err', file=sys.stderr)"
	_, err = sb.Code.RunCode(ctx, codeStr, &code.RunCodeConfig{Language: "python"}, &code.OnOutputConfig{
		OnStdout: func(s string) { log.Print("OUT:", s) },
		OnStderr: func(s string) { log.Print("ERR:", s) },
	})
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

// 3. 文件系统操作（读/写/列/查/删/改名/建目录）
func Example_filesystem_ops() {
	ctx := context.Background()

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

	sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sb.Kill(ctx) }()

	// 写文件
	_, err = sb.Files.Write(ctx, "/home/user/demo.txt", bytes.NewBufferString("hello"), &filesystem.WriteConfig{User: "user"})
	if err != nil {
		log.Fatal(err)
	}

	// 读文件
	r, err := sb.Files.Read(ctx, "/home/user/demo.txt", &filesystem.ReadConfig{User: "user"})
	if err != nil {
		log.Fatal(err)
	}
	data, _ := io.ReadAll(r)
	log.Println("content:", string(data))

	// 列目录
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

	// 获取信息
	info, err := sb.Files.GetInfo(ctx, "/home/user/demo.txt", &filesystem.GetInfoConfig{User: "user"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("owner=%s perms=%s", info.Owner, info.Permissions)

	// 判断是否存在
	ok, err := sb.Files.Exists(ctx, "/home/user/demo.txt", &filesystem.ExistsConfig{User: "user"})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("exists:", ok)

	// 改名
	err = sb.Files.Rename(ctx, "/home/user/demo.txt", "/home/user/demo2.txt", &filesystem.RenameConfig{User: "user"})
	if err != nil {
		log.Fatal(err)
	}

	// 创建目录
	created, err := sb.Files.MakeDir(ctx, "/home/user/newdir", &filesystem.MakeDirConfig{User: "user"})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("dir created:", created)

	// 删除
	err = sb.Files.Remove(ctx, "/home/user/demo2.txt", &filesystem.RemoveConfig{User: "user"})
	if err != nil {
		log.Fatal(err)
	}
	// Output:
}

// 4. 命令/进程管理 —— 便捷前台运行
func Example_command_run() {
	ctx := context.Background()

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

	sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sb.Kill(ctx) }()

	res, err := sb.Commands.Run(ctx, "echo hello && uname -a", &command.ProcessConfig{
		User: "user", // 进程内用户，默认为 "user"
	}, &command.OnOutputConfig{
		OnStdout: func(b []byte) { log.Printf("STDOUT: %s", string(b)) },
		OnStderr: func(b []byte) { log.Printf("STDERR: %s", string(b)) },
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("exit=%d, stdout=%q, stderr=%q, err=%v", res.ExitCode, string(res.Stdout), string(res.Stderr), res.Error)
	// Output:
}

// 4. 命令/进程管理 —— 后台启动 + 等待
func Example_command_background() {
	ctx := context.Background()

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

	sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sb.Kill(ctx) }()

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
	// Output:
}

// 4. 命令/进程管理 —— 发送输入与信号
func Example_command_signals() {
	ctx := context.Background()

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

	sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sb.Kill(ctx) }()
	// 创建进程
	h, err := sb.Commands.Start(ctx, "read test\necho $test\nread test\necho $test", &command.ProcessConfig{
		User: "user",
	}, &command.OnOutputConfig{
		OnStdout: func(b []byte) { log.Print("OUT:", string(b)) },
		OnStderr: func(b []byte) { log.Print("ERR:", string(b)) },
	})
	if err != nil {
		log.Fatal(err)
	}
	// 连接已存在进程（通过 PID）
	h, err = sb.Commands.Connect(ctx, h.Pid, nil)
	if err != nil {
		log.Fatal(err)
	}

	// 一次性发送输入到 stdin
	_ = h.SendInput(ctx, h.Pid, []byte("hello\n"))

	// 发送 SIGTERM（也可用 h.Kill(ctx) 发送 SIGKILL）
	// 这里使用信号编号 2（示例与文档一致）
	_ = h.SendSignal(ctx, h.Pid, 2 /* process.Signal */)
	// Output:
}

// 4. 命令/进程管理 —— 列出运行中进程
func Example_command_list() {
	ctx := context.Background()

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

	sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sb.Kill(ctx) }()

	ps, err := sb.Commands.List(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range ps {
		log.Printf("pid=%d cmd=%s args=%v cwd=%v", p.Pid, p.Cmd, p.Args, p.Cwd)
	}
	// Output:
}

// 5. 直接使用 core 列表/连接/销毁（可选）
func Example_core_ops() {
	ctx := context.Background()

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

	// 列表
	instances, err := sandboxcode.List(ctx, sandboxcode.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	for _, ins := range instances {
		log.Println(*ins.InstanceId, *ins.Status)
	}
	sandbox, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	// 连接到指定沙箱（仅获取 token，不初始化三客户端）
	coreOnly, err := core.Connect(ctx, sandbox.SandboxId, core.WithClient(client))
	if err != nil {
		log.Fatal(err)
	}
	defer coreOnly.Kill(ctx)
	// Output:
}

// 6. 创建沙箱实例时，挂载沙箱工具内所描述的COS存储路径的subpath的示例
//
// MountOption使用说明:
//   - Name: 存储名称，创建沙箱工具时在 StorageMount.Name 中定义的名称
//   - MountPath: 挂载路径，在沙箱实例内的挂载点路径，如 "/home/user/workspace"。
//   - SubPath: 子路径，存储桶内的子路径，可用于不同租户隔离，如 "user-123"
//   - ReadOnly: 是否只读，true 为只读，false 为可读写（默认 false）
//
// 环境变量配置（优先级高于代码内默认值）:
//   - TENCENTCLOUD_SECRET_ID: 腾讯云 API 密钥 ID
//   - TENCENTCLOUD_SECRET_KEY: 腾讯云 API 密钥 Key
//   - AGS_REGION: 地域，默认 "ap-chongqing"
//   - AGS_TOOL_NAME: 工具名称，默认 "code-cos-test"
//   - AGS_STORAGE_NAME: 挂载名称（MountOption.Name），默认 "cos"
//   - AGS_MOUNT_PATH: 沙箱内挂载路径，默认 "/data/cos"
//   - AGS_SUB_PATH: 存储桶内子路径（可选）
//   - AGS_READ_ONLY: 是否只读，"true" 或 "false"，默认 "false"
func Example_cosMount() {
	ctx := context.Background()

	// 从环境变量读取腾讯云 API 密钥
	cred := &common.Credential{
		SecretId:  os.Getenv("TENCENTCLOUD_SECRET_ID"),
		SecretKey: os.Getenv("TENCENTCLOUD_SECRET_KEY"),
	}

	// 地域：优先使用环境变量，默认 ap-chongqing
	region := os.Getenv("AGS_REGION")
	if region == "" {
		region = "ap-chongqing"
	}

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "ags.tencentcloudapi.com"
	client, err := ags.NewClient(cred, region, cpf)
	if err != nil {
		log.Fatal(err)
	}

	// 工具名称：优先使用环境变量，默认 code-cos-test
	toolName := os.Getenv("AGS_TOOL_NAME")
	if toolName == "" {
		toolName = "code-cos-test"
	}

	// 挂载名称（对应工具定义中的 StorageMount.Name）：默认 cos
	storageName := os.Getenv("AGS_STORAGE_NAME")
	if storageName == "" {
		storageName = "cos"
	}

	// 沙箱内挂载路径：默认 /data/cos
	mountPath := os.Getenv("AGS_MOUNT_PATH")
	if mountPath == "" {
		mountPath = "/data/cos"
	}

	// 构造 MountOption
	mountOption := &sandboxcode.MountOption{
		// Name 对应创建沙箱工具时配置的 StorageMount.Name
		Name: &storageName,
		// MountPath 指定 COS 在沙箱实例内的挂载点，仅支持 /home、/workspace、/data、/mnt 前缀
		MountPath: &mountPath,
	}

	// SubPath：存储桶内子路径，用于多租户隔离（可选）
	if subPath := os.Getenv("AGS_SUB_PATH"); subPath != "" {
		mountOption.SubPath = &subPath
	}

	// ReadOnly：是否只读（可选，默认 false 即可读写）
	readOnly := os.Getenv("AGS_READ_ONLY") == "true"
	mountOption.ReadOnly = &readOnly

	log.Printf("MountOption: Name=%s, MountPath=%s, ReadOnly=%v", storageName, mountPath, readOnly)

	// 组装沙箱配置
	timeout := "10m"
	sandboxConfig := &sandboxcode.SandboxConfig{
		Timeout: &timeout,
		MountOptions: []*sandboxcode.MountOption{
			mountOption,
		},
	}

	// 创建携带 COS 挂载的沙箱实例
	sandbox, err := sandboxcode.Create(ctx, toolName,
		sandboxcode.WithClient(client),
		sandboxcode.WithSandboxConfig(sandboxConfig),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sandbox.Kill(ctx) }()

	log.Println("Successfully created sandbox with ID:", sandbox.SandboxId)

	// 向 COS 挂载路径写入文件（通过 Files 客户端）
	testFilePath := fmt.Sprintf("%s/test_file.txt", mountPath)
	fileContent := "hello from ags-go-sdk MountOption example"
	_, err = sandbox.Files.Write(ctx, testFilePath, bytes.NewBufferString(fileContent), nil)
	if err != nil {
		log.Fatal("Failed to write file to COS mount:", err)
	}
	log.Println("File written to COS mount:", testFilePath)

	// 从 COS 挂载路径读取文件，验证写入内容
	reader, err := sandbox.Files.Read(ctx, testFilePath, nil)
	if err != nil {
		log.Fatal("Failed to read file from COS mount:", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		log.Fatal("Failed to read file content:", err)
	}
	log.Printf("File content read back: %s", string(data))

	// 通过命令行列出挂载目录，验证 COS 挂载是否生效
	result, err := sandbox.Commands.Run(ctx, fmt.Sprintf("ls -la %s", mountPath), &command.ProcessConfig{
		User: "root",
	}, &command.OnOutputConfig{
		OnStdout: func(b []byte) { log.Printf("ls stdout: %s", string(b)) },
		OnStderr: func(b []byte) { log.Printf("ls stderr: %s", string(b)) },
	})
	if err != nil {
		log.Fatal("Failed to list mount directory:", err)
	}
	log.Printf("ls exit code: %d", result.ExitCode)

	log.Println("COS mount example completed successfully!")
	// Output:
}
