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
// - Name: 存储名称，创建沙箱工具时定义的StorageMount.Name
// - MountPath: 挂载路径，在沙箱实例内的挂载点路径，如 "/data/myname/cos"。仅支持以下路径/home, /workspace, /data, /mnt
// - SubPath: 子路径，存储桶内的子路径，可用于隔离不同租户，如 "my-user-1"
// - ReadOnly: 是否只读，true为只读，false为可读写
//
// 环境变量配置:
// - TENCENTCLOUD_SECRET_ID: 腾讯云API密钥ID
// - TENCENTCLOUD_SECRET_KEY: 腾讯云API密钥Key
// - AGS_REGION: 区域，如 "ap-guangzhou"
// - AGS_TOOL_NAME: 工具名称，如 "code-interpreter-with-cos"
// - AGS_STORAGE_NAME: COS存储桶名称（必选，只有设置了该变量才会创建mountOptions）
// - AGS_MOUNT_PATH: 挂载路径，如 "/data/myname/cos"（可选）
// - AGS_SUB_PATH: 子路径，如 "my-user-1"（可选）
// - AGS_READ_ONLY: 是否只读，true或false（可选）
func Example_cosMount() {
	ctx := context.Background()

	// Create credential from environment variables
	cred := &common.Credential{
		SecretId:  os.Getenv("TENCENTCLOUD_SECRET_ID"),
		SecretKey: os.Getenv("TENCENTCLOUD_SECRET_KEY"),
	}

	// Get region from environment variable, default to "ap-guangzhou"
	region := os.Getenv("AGS_REGION")
	if region == "" {
		region = "ap-guangzhou"
	}

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "ags.tencentcloudapi.com"
	client, err := ags.NewClient(cred, region, cpf)
	if err != nil {
		log.Fatal(err)
	}

	// Get tool name from environment variable, default to "code-interpreter-v1"
	toolName := os.Getenv("AGS_TOOL_NAME")
	if toolName == "" {
		log.Fatal("AGS_TOOL_NAME environment variable is required")
	}

	// Get storage configuration from environment variables
	var mountOptions []*sandboxcode.MountOption
	var mountPath string
	storageName := os.Getenv("AGS_STORAGE_NAME")

	// Only create mount options if storage name is set
	if storageName != "" {
		mountOption := &sandboxcode.MountOption{
			Name: &storageName,
		}

		// Add mount path if set
		mountPath = os.Getenv("AGS_MOUNT_PATH")
		if mountPath != "" {
			mountOption.MountPath = &mountPath
		}

		// Add sub path if set
		subPath := os.Getenv("AGS_SUB_PATH")
		if subPath != "" {
			mountOption.SubPath = &subPath
		}

		// Add read only option if set
		readOnlyStr := os.Getenv("AGS_READ_ONLY")
		if readOnlyStr != "" {
			readOnly := readOnlyStr == "true"
			mountOption.ReadOnly = &readOnly
		}

		mountOptions = append(mountOptions, mountOption)
		log.Println("COS mount options configured successfully")
	} else {
		log.Println("AGS_STORAGE_NAME environment variable is not set, skipping COS mount")
	}

	// Create sandbox configuration with mount options
	timeout := "10m"
	sandboxConfig := &sandboxcode.SandboxConfig{
		Timeout:      &timeout,
		MountOptions: mountOptions,
	}

	// Create sandbox instance with COS mount
	sandbox, err := sandboxcode.Create(ctx, toolName,
		sandboxcode.WithClient(client),
		sandboxcode.WithSandboxConfig(sandboxConfig),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer sandbox.Kill(ctx)

	log.Println("Successfully created sandbox instance with ID:", sandbox.SandboxId)

	// Only write to COS mount if mountPath is set
	if mountPath != "" {
		// Write file to COS mount path
		fileContent := "hello from hongxinchu"
		filePath := fmt.Sprintf("%s/test_file.txt", mountPath)

		cmd := fmt.Sprintf("echo '%s' > %s", fileContent, filePath)

		log.Println("Writing file command:", cmd)

		// Execute the command to write file to COS mounted path
		result, err := sandbox.Commands.Run(ctx, cmd, &command.ProcessConfig{
			User: "root",
		}, nil)
		if err != nil {
			log.Fatal("Failed to execute command:", err)
		}

		log.Printf("Command executed successfully. Exit code: %d", result.ExitCode)
		if len(result.Stdout) > 0 {
			log.Printf("Stdout: %s", string(result.Stdout))
		}
		if len(result.Stderr) > 0 {
			log.Printf("Stderr: %s", string(result.Stderr))
		}

		// Verify the file was created by reading it back
		reader, err := sandbox.Files.Read(ctx, filePath, nil)
		if err != nil {
			log.Fatal("Failed to read file:", err)
		}
		defer func() {
			if closer, ok := reader.(io.Closer); ok {
				closer.Close()
			}
		}()

		// Read the file content
		buf := make([]byte, 1024)
		n, err := reader.Read(buf)
		if err != nil {
			log.Fatal("Failed to read file content:", err)
		}

		log.Printf("File content verified: %s", string(buf[:n]))
	}
	log.Println("Sandbox operation completed successfully!")

	// Output:
}
