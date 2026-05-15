# Troubleshooting

This guide covers common errors you may encounter when using the Agent Sandbox Go SDK and how to resolve them.

## Authentication Failures

### Missing or Invalid Credentials

**Symptom:** Sandbox creation fails immediately with an authentication error.

```
Error: [TencentCloudSDKError] Code=AuthFailure.SecretIdNotFound, Message=The SecretId is not found...
```

**Cause:** The Tencent Cloud SecretID or SecretKey is missing, incorrect, or expired.

**Fix:**

1. Obtain credentials from the [Tencent Cloud Console - API Keys](https://console.cloud.tencent.com/cam/capi).
2. Set the required environment variables:
   ```bash
   export TENCENTCLOUD_SECRET_ID="your-secret-id"
   export TENCENTCLOUD_SECRET_KEY="your-secret-key"
   ```
3. Verify in your code that credentials are being read correctly:
   ```go
   cred := &common.Credential{
       SecretId:  os.Getenv("TENCENTCLOUD_SECRET_ID"),
       SecretKey: os.Getenv("TENCENTCLOUD_SECRET_KEY"),
   }
   // Check they are not empty
   if cred.SecretId == "" || cred.SecretKey == "" {
       log.Fatal("TENCENTCLOUD_SECRET_ID and TENCENTCLOUD_SECRET_KEY must be set")
   }
   ```

### Authorization Failed (Insufficient Permissions)

**Symptom:**

```
Error: [TencentCloudSDKError] Code=AuthFailure.UnauthorizedOperation...
```

**Cause:** Credentials are valid but the account lacks the required permissions.

**Fix:**

1. Ensure the Tencent Cloud sub-account has the `QcloudAGSFullAccess` policy attached.
2. If using a temporary STS token, verify it has not expired and includes the required AGS permissions.

---

## Network and Connection Issues

### Connection Timeout

**Symptom:**

```
Error: Post "https://ags.tencentcloudapi.com": dial tcp: i/o timeout
```

**Cause:** The SDK cannot reach the Tencent Cloud API endpoint.

**Fix:**

1. **Check network connectivity:**
   ```bash
   curl -I https://ags.tencentcloudapi.com
   ```
2. **Corporate proxy:** If behind a proxy, set the proxy in your environment:
   ```bash
   export HTTP_PROXY="http://your-proxy:port"
   export HTTPS_PROXY="http://your-proxy:port"
   ```
   Or configure it in the SDK client profile:
   ```go
   cpf := profile.NewClientProfile()
   cpf.HttpProfile.Proxy = "http://your-proxy:port"
   ```
3. **Firewall:** Ensure outbound HTTPS (port 443) to `ags.tencentcloudapi.com` and `*.tencentags.com` is allowed.

### Data Plane Connection Failure

**Symptom:** Sandbox is created successfully, but code execution or file operations fail with connection errors.

```
Error: Post "https://49999-<sandbox-id>.ap-guangzhou.tencentags.com/...": dial tcp: i/o timeout
```

**Cause:** The data plane endpoint (used for code execution, file operations, and commands) is not reachable.

**Fix:**

1. Verify the sandbox is in a running/connectable state:
   ```go
   // After Create(), the sandbox should be connectable
   sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
   ```
2. Ensure `*.tencentags.com` is not blocked by your network.
3. Check that the region in the connection matches where the sandbox was created.

### Wrong Region

**Symptom:** `sandbox not found` or empty list when you know sandboxes exist.

**Cause:** The SDK client is pointing to a different region than where the sandbox was created.

**Fix:** Make sure the region matches:

```go
client, err := ags.NewClient(cred, "ap-guangzhou", cpf)  // Must match the sandbox region
```

---

## Sandbox Lifecycle Errors

### Sandbox Creation Fails

**Symptom:**

```
Error: failed to create sandbox: ...
```

**Possible causes and fixes:**

1. **Tool name not found:** Verify the tool name is correct and exists in your account/region:
   ```go
   // Use the correct tool name
   sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
   ```
2. **Quota exceeded:** Check your AGS resource quota in the [Tencent Cloud Console](https://console.cloud.tencent.com/ags).
3. **Invalid configuration:** Ensure the `sandbox/core` options (if passed) are valid.

### Sandbox Not in Connectable State

**Symptom:**

```
Error: sandbox is not in connectable state
```

**Cause:** The sandbox is still starting up, has been terminated, or encountered an error during initialization.

**Fix:**

1. Wait briefly and retry — sandbox provisioning may take a few seconds.
2. If the sandbox was previously killed, create a new one.
3. Use `Connect()` only for sandboxes that are confirmed running:
   ```go
   sb, err := sandboxcode.Connect(ctx, "existing-sandbox-id",
       sandboxcode.WithClient(client),
   )
   ```

### Invalid Sandbox ID

**Symptom:**

```
Error: invalid sandbox ID
```

**Fix:** Sandbox IDs are returned from `Create()` or `List()`. Ensure you are passing the correct, full sandbox ID string.

---

## Code Execution Errors

### ExecutionError Returned

**Symptom:** Code runs but returns an error in the `Execution.Error` field.

```go
exec, err := sb.Code.RunCode(ctx, code, nil, nil)
if exec.Error != nil {
    // exec.Error.Name  = "ValueError"
    // exec.Error.Value = "invalid literal for int()..."
    // exec.Error.Traceback = "..."
}
```

**Cause:** The code executed in the sandbox raised an exception.

**Fix:**

1. This is a **code-level error**, not an SDK error. Check the `Traceback` field for details.
2. Ensure dependencies are installed in the sandbox if your code uses external packages.
3. Verify the code is valid for the target language.

### Context Not Found

**Symptom:** Error when running code in a previously created context.

**Cause:** The execution context may have been garbage-collected or the sandbox restarted.

**Fix:**

1. Create a new context:
   ```go
   ctxId, err := sb.Code.CreateContext(ctx, nil)
   ```
2. Use the new context ID for subsequent executions.

---

## Filesystem Errors

### File or Directory Not Found

**Symptom:**

```
Error: no such file or directory
```

**Fix:**

1. Use `Check()` to verify a path exists before operating on it:
   ```go
   exists, err := sb.Files.Check(ctx, "/home/user/data.txt")
   ```
2. Create parent directories first:
   ```go
   _, err := sb.Files.MakeDir(ctx, "/home/user/output", nil)
   ```

---

## Common Error Quick Reference

| Error | Likely Cause | Quick Fix |
|---|---|---|
| `AuthFailure.SecretIdNotFound` | Invalid SecretID | Check `TENCENTCLOUD_SECRET_ID` |
| `AuthFailure.UnauthorizedOperation` | Insufficient permissions | Attach `QcloudAGSFullAccess` policy |
| `dial tcp: i/o timeout` | Network issue | Check proxy, firewall, DNS |
| `sandbox not found` | Wrong region or invalid ID | Verify region and sandbox ID |
| `sandbox is not in connectable state` | Sandbox not running | Wait or create a new sandbox |
| `ExecutionError` in result | Code raised an exception | Check `Traceback` field |
| `no such file or directory` | Path does not exist | Use `Check()` or `MakeDir()` first |

---

## Still Stuck?

If the above steps do not resolve your issue:

1. Enable verbose logging to see request/response details.
2. Open an issue on [GitHub](https://github.com/TencentCloudAgentRuntime/ags-go-sdk/issues) with:
   - The full error message
   - Your Go version (`go version`)
   - SDK version (check `go.mod`)
   - A minimal code snippet that reproduces the issue (redact credentials)
