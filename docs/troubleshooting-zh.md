# 故障排查

本文档涵盖使用 Agent Sandbox Go SDK 时可能遇到的常见错误及解决方法。

## 认证失败

### 凭证缺失或无效

**现象：** 沙箱创建立即失败，报认证错误。

```
Error: [TencentCloudSDKError] Code=AuthFailure.SecretIdNotFound, Message=The SecretId is not found...
```

**原因：** 腾讯云 SecretID 或 SecretKey 缺失、错误或已过期。

**解决方法：**

1. 从[腾讯云控制台 - API 密钥](https://console.cloud.tencent.com/cam/capi)获取凭证。
2. 设置环境变量：
   ```bash
   export TENCENTCLOUD_SECRET_ID="your-secret-id"
   export TENCENTCLOUD_SECRET_KEY="your-secret-key"
   ```
3. 在代码中验证凭证正确读取：
   ```go
   cred := &common.Credential{
       SecretId:  os.Getenv("TENCENTCLOUD_SECRET_ID"),
       SecretKey: os.Getenv("TENCENTCLOUD_SECRET_KEY"),
   }
   if cred.SecretId == "" || cred.SecretKey == "" {
       log.Fatal("TENCENTCLOUD_SECRET_ID and TENCENTCLOUD_SECRET_KEY must be set")
   }
   ```

### 授权失败（权限不足）

**现象：**

```
Error: [TencentCloudSDKError] Code=AuthFailure.UnauthorizedOperation...
```

**原因：** 凭证有效但账户缺少所需权限。

**解决方法：**

1. 确保腾讯云子账户已关联 `QcloudAGSFullAccess` 策略。
2. 如使用临时 STS Token，确认未过期且包含所需 AGS 权限。

---

## 网络和连接问题

### 连接超时

**现象：**

```
Error: Post "https://ags.tencentcloudapi.com": dial tcp: i/o timeout
```

**原因：** SDK 无法访问腾讯云 API 端点。

**解决方法：**

1. **检查连通性：**
   ```bash
   curl -I https://ags.tencentcloudapi.com
   ```
2. **企业代理：** 如在代理后面，设置代理：
   ```bash
   export HTTP_PROXY="http://your-proxy:port"
   export HTTPS_PROXY="http://your-proxy:port"
   ```
   或在 SDK 客户端配置中设置：
   ```go
   cpf := profile.NewClientProfile()
   cpf.HttpProfile.Proxy = "http://your-proxy:port"
   ```
3. **防火墙：** 确保 HTTPS（443 端口）到 `ags.tencentcloudapi.com` 和 `*.tencentags.com` 的出站连接被允许。

### 数据面连接失败

**现象：** 沙箱创建成功，但代码执行或文件操作报连接错误。

```
Error: Post "https://49999-<sandbox-id>.ap-guangzhou.tencentags.com/...": dial tcp: i/o timeout
```

**原因：** 数据面端点（用于代码执行、文件操作和命令）不可达。

**解决方法：**

1. 确认沙箱处于运行/可连接状态。
2. 确保 `*.tencentags.com` 未被网络阻止。
3. 检查连接中的地域是否与沙箱创建地域一致。

### 地域错误

**现象：** `sandbox not found` 或列表为空，但确实有沙箱存在。

**原因：** SDK 客户端指向了与沙箱创建时不同的地域。

**解决方法：** 确保地域一致：

```go
client, err := ags.NewClient(cred, "ap-guangzhou", cpf)  // 必须与沙箱地域匹配
```

---

## 沙箱生命周期错误

### 沙箱创建失败

**现象：**

```
Error: failed to create sandbox: ...
```

**可能原因和解决方法：**

1. **工具名称不存在：** 确认工具名称正确且存在于你的账户/地域中。
2. **配额超限：** 在[腾讯云控制台](https://console.cloud.tencent.com/ags)检查 AGS 资源配额。
3. **配置无效：** 确保传入的 `sandbox/core` 选项有效。

### 沙箱未处于可连接状态

**现象：**

```
Error: sandbox is not in connectable state
```

**原因：** 沙箱仍在启动中、已终止或初始化时出错。

**解决方法：**

1. 稍等片刻后重试——沙箱配置可能需要几秒钟。
2. 如果沙箱已被销毁，创建新的沙箱。
3. 仅对确认运行中的沙箱使用 `Connect()`。

### 无效的沙箱 ID

**现象：**

```
Error: invalid sandbox ID
```

**解决方法：** 沙箱 ID 由 `Create()` 或 `List()` 返回，确保传入正确、完整的沙箱 ID 字符串。

---

## 代码执行错误

### 返回 ExecutionError

**现象：** 代码运行但在 `Execution.Error` 字段中返回错误。

```go
exec, err := sb.Code.RunCode(ctx, code, nil, nil)
if exec.Error != nil {
    // exec.Error.Name  = "ValueError"
    // exec.Error.Value = "invalid literal for int()..."
    // exec.Error.Traceback = "..."
}
```

**原因：** 沙箱中执行的代码抛出了异常。

**解决方法：**

1. 这是**代码级别的错误**，不是 SDK 错误。检查 `Traceback` 字段获取详情。
2. 如果代码使用外部包，确保已在沙箱中安装依赖。
3. 验证代码对目标语言有效。

### 上下文未找到

**现象：** 在之前创建的上下文中运行代码时出错。

**原因：** 执行上下文可能已被回收或沙箱已重启。

**解决方法：** 创建新的上下文并使用新的上下文 ID。

---

## 文件系统错误

### 文件或目录不存在

**现象：**

```
Error: no such file or directory
```

**解决方法：**

1. 操作前使用 `Check()` 验证路径是否存在：
   ```go
   exists, err := sb.Files.Check(ctx, "/home/user/data.txt")
   ```
2. 先创建父目录：
   ```go
   _, err := sb.Files.MakeDir(ctx, "/home/user/output", nil)
   ```

---

## 常见错误速查表

| 错误 | 可能原因 | 快速修复 |
|------|---------|---------|
| `AuthFailure.SecretIdNotFound` | SecretID 无效 | 检查 `TENCENTCLOUD_SECRET_ID` |
| `AuthFailure.UnauthorizedOperation` | 权限不足 | 关联 `QcloudAGSFullAccess` 策略 |
| `dial tcp: i/o timeout` | 网络问题 | 检查代理、防火墙、DNS |
| `sandbox not found` | 地域错误或 ID 无效 | 确认地域和沙箱 ID |
| `sandbox is not in connectable state` | 沙箱未运行 | 等待或创建新沙箱 |
| `ExecutionError` | 代码抛出异常 | 检查 `Traceback` 字段 |
| `no such file or directory` | 路径不存在 | 先使用 `Check()` 或 `MakeDir()` |

---

## 仍然无法解决？

如果以上步骤无法解决你的问题：

1. 启用详细日志查看请求/响应详情。
2. 在 [GitHub](https://github.com/TencentCloudAgentRuntime/ags-go-sdk/issues) 上提交 issue，包含：
   - 完整的错误信息
   - Go 版本（`go version`）
   - SDK 版本（检查 `go.mod`）
   - 可复现问题的最小代码片段（隐藏凭证）
