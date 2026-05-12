# Error Handling Guide

This guide covers common errors you may encounter when using the Agent Sandbox Go SDK, their causes, and how to resolve them.

## Table of Contents

- [Error Categories](#error-categories)
- [SDK Errors vs Execution Errors](#sdk-errors-vs-execution-errors)
- [1. Client Initialization Errors](#1-client-initialization-errors)
- [2. Sandbox Lifecycle Errors](#2-sandbox-lifecycle-errors)
- [3. Code Execution Errors](#3-code-execution-errors)
- [4. Filesystem Errors](#4-filesystem-errors)
- [5. Command/Process Errors](#5-commandprocess-errors)
- [6. Network and Connectivity Errors](#6-network-and-connectivity-errors)
- [7. Context and Timeout Errors](#7-context-and-timeout-errors)
- [Error Handling Patterns](#error-handling-patterns)
- [Retry Strategy](#retry-strategy)
- [FAQ](#faq)

---

## Error Categories

| Category | Retryable | Description |
|----------|-----------|-------------|
| Client initialization | No | Missing credentials or invalid configuration |
| Authentication/authorization | No | Invalid SecretId/SecretKey or insufficient permissions |
| Sandbox lifecycle | Sometimes | Instance creation, connection, or termination failures |
| Code execution | No | Errors in user code (returned in `ExecutionError`) |
| Filesystem validation | No | Invalid path, user, or depth parameters |
| Command/process | No | Invalid handle or signal parameters |
| Network/connectivity | Yes | Transient connection issues or service unavailability |
| Context/timeout | Sometimes | Operation timed out or was cancelled |

---

## SDK Errors vs Execution Errors

The SDK distinguishes between two kinds of errors:

### SDK Errors (returned as `error`)

These are Go errors returned by SDK methods. They indicate that the SDK operation itself failed — for example, the network request could not be completed, or input validation failed.

```go
exec, err := sb.Code.RunCode(ctx, code, nil, nil)
if err != nil {
    // SDK error: the RunCode request failed (network, auth, validation, etc.)
    log.Fatal("SDK error:", err)
}
```

### Execution Errors (returned in `ExecutionError`)

These indicate that your code ran successfully on the sandbox, but the code itself produced an error (e.g., a Python exception). The SDK call succeeds (returns `nil` error), but the `Execution.Error` field is populated.

```go
exec, err := sb.Code.RunCode(ctx, "1/0", nil, nil)
if err != nil {
    log.Fatal("SDK error:", err)
}
if exec.Error != nil {
    // Execution error: your code raised an exception
    log.Printf("Code error: %s: %s\n%s",
        exec.Error.Name,      // e.g., "ZeroDivisionError"
        exec.Error.Value,     // e.g., "division by zero"
        exec.Error.Traceback, // Full traceback
    )
}
```

The `ExecutionError` struct:

```go
type ExecutionError struct {
    Name      string // Error type name (e.g., "ZeroDivisionError")
    Value     string // Error message (e.g., "division by zero")
    Traceback string // Full stack traceback
}
```

---

## 1. Client Initialization Errors

### "client cannot be initialized. Make sure you have provided a valid client, or a credential and region pair"

**Cause:** Neither a pre-built AGS client (`WithClient`) nor a credential+region pair (`WithCredential` + `WithRegion`) was provided.

**Solution:** Provide one of the following when calling `core.Create`, `core.Connect`, `core.List`, or `core.Kill`:

```go
// Option A: Pass a pre-built client
client, _ := ags.NewClient(cred, "ap-guangzhou", cpf)
sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))

// Option B: Pass credential and region separately
sb, err := sandboxcode.Create(ctx, "code-interpreter-v1",
    sandboxcode.WithCredential(cred),
    sandboxcode.WithRegion("ap-guangzhou"),
)
```

### Tencent Cloud SDK authentication errors

**Cause:** Invalid `TENCENTCLOUD_SECRET_ID` or `TENCENTCLOUD_SECRET_KEY`.

**Solution:**
1. Verify your credentials at the [Tencent Cloud Console - API Keys](https://console.cloud.tencent.com/cam/capi).
2. Ensure environment variables are set:
   ```bash
   export TENCENTCLOUD_SECRET_ID="your-secret-id"
   export TENCENTCLOUD_SECRET_KEY="your-secret-key"
   ```
3. Check that the credential object is not nil and contains valid values.

---

## 2. Sandbox Lifecycle Errors

### "not found sandbox instance"

**Cause:** `core.GetInfo()` was called with a sandbox ID that does not exist or is no longer active.

**Solution:**
- Verify the sandbox ID is correct.
- Check if the sandbox has been terminated or timed out using `core.List()`.
- Re-create the sandbox if needed.

### "StartSandboxInstance response.Response is nil"

**Cause:** The API returned an unexpected empty response when creating a sandbox.

**Solution:**
- This is typically a transient server-side issue. Retry after a brief delay.
- If persistent, check your network connectivity and service status.

### "StartSandboxInstance response.Response.Instance is nil"

**Cause:** The sandbox creation request succeeded but returned no instance data.

**Solution:**
- Verify the tool name is correct (e.g., `"code-interpreter-v1"`).
- Check if your account has available quota for sandbox instances.

### "StartSandboxInstance response.Response.Instance.InstanceId is nil"

**Cause:** The sandbox instance was created but did not receive an ID assignment.

**Solution:** Retry the creation request. If persistent, contact support.

### "AcquireSandboxInstanceToken response.Response is nil" / "...Token is nil"

**Cause:** Token acquisition failed for the sandbox instance.

**Solution:**
- Verify the sandbox instance is running (not stopped or expired).
- Check that your credentials have permission to access the instance.
- Retry the operation.

### "DescribeSandboxInstanceList response.Response is nil"

**Cause:** The list API returned an unexpected empty response.

**Solution:** Retry the request. Verify network connectivity if persistent.

---

## 3. Code Execution Errors

### "code client not initialized"

**Cause:** The `code.Client` was not properly initialized before calling `RunCode` or `CreateCodeContext`.

**Solution:** Ensure you initialize the client via `code.New(connectionConfig)` or use `sandboxcode.Create/Connect` which initializes all three clients automatically.

```go
// Recommended: use sandbox/code which initializes all clients
sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
// sb.Code is ready to use
```

### "connection domain is empty"

**Cause:** The connection config passed to the client has an empty `Domain` field.

**Solution:** Use `sandboxcode.Create` or `sandboxcode.Connect` which sets up the domain automatically. If constructing a client manually, ensure the domain follows the format `"{port}-{sandboxId}.{region}.tencentags.com"`.

### "cannot use RunCode with both contextId and language"

**Cause:** `RunCodeConfig` has both `Language` and `ContextId` set. These are mutually exclusive — a context already has a language associated with it.

**Solution:** Use one or the other:

```go
// Option A: Specify language (creates a temporary context)
config := &code.RunCodeConfig{Language: "python"}

// Option B: Use an existing context
config := &code.RunCodeConfig{ContextId: "ctx-abc123"}
```

### HTTP status code errors (e.g., "401: Unauthorized", "500: Internal Server Error")

**Cause:** The sandbox data plane returned an HTTP error.

**Solution:**
- **401/403:** Check that the access token is valid and not expired. Reconnect to the sandbox.
- **404:** The sandbox instance may have been terminated.
- **500:** Server-side issue. Retry after a brief delay.

---

## 4. Filesystem Errors

### "filesystem client not initialized"

**Cause:** The `filesystem.Client` was not initialized before use.

**Solution:** Use `sandboxcode.Create/Connect` or manually initialize with `filesystem.New(connectionConfig)`.

### "config cannot be nil"

**Cause:** A nil config was passed to `filesystem.New()`.

**Solution:** Pass a valid `connection.Config`:

```go
client, err := filesystem.New(sb.ConnectionConfig)
```

### "path is empty"

**Cause:** An empty string was passed as the file/directory path.

**Solution:** Provide a valid path:

```go
content, err := sb.Files.Read(ctx, "/home/user/myfile.txt", nil)
```

### "source or destination path is empty"

**Cause:** The `Rename` operation received an empty source or destination path.

**Solution:** Provide both paths:

```go
err := sb.Files.Rename(ctx, "/home/user/old.txt", "/home/user/new.txt", nil)
```

### "invalid user: X, must be user or root"

**Cause:** The `User` field in a filesystem config was set to a value other than `"user"` or `"root"`.

**Solution:** Use `"user"` (default) or `"root"`:

```go
config := &filesystem.ReadConfig{User: "root"} // or "user"
```

### "invalid depth: X, must be >= 0"

**Cause:** A negative depth was passed to `ListConfig`.

**Solution:** Use a non-negative depth value (default is 1):

```go
config := &filesystem.ListConfig{Depth: 2}
```

### "empty response from ListDir" / "empty response from Stat" / "expected write info in response" / "empty response from MakeDir"

**Cause:** The sandbox returned an empty response for a filesystem operation.

**Solution:** Verify the sandbox is still running and the path exists. Retry if transient.

---

## 5. Command/Process Errors

### "command client not initialized"

**Cause:** The `command.Client` was not initialized before use.

**Solution:** Use `sandboxcode.Create/Connect` or manually initialize with `command.New(connectionConfig)`.

### "handle is nil" / "handle not initialized"

**Cause:** Attempted to call a method on a nil or uninitialized `Handle`.

**Solution:** Ensure `Start` or `Connect` returned a valid handle:

```go
handle, err := sb.Commands.Start(ctx, "python3 server.py", nil, nil)
if err != nil {
    log.Fatal(err)
}
// handle is now valid
result, err := handle.Wait(ctx)
```

### "client is nil"

**Cause:** The handle's internal client reference is nil, typically because the handle was not created through proper SDK methods.

**Solution:** Only use handles returned by `Client.Start` or `Client.Connect`.

### "invalid signal: X, only SIGTERM(15) and SIGKILL(9) supported"

**Cause:** An unsupported signal value was passed to `SendSignal`.

**Solution:** Use only SIGTERM (15) or SIGKILL (9):

```go
import "github.com/TencentCloudAgentRuntime/ags-go-sdk/pb/process"

err := handle.SendSignal(ctx, handle.Pid, process.Signal_SIGNAL_SIGTERM) // 15
err := handle.SendSignal(ctx, handle.Pid, process.Signal_SIGNAL_SIGKILL) // 9
```

### "empty response from List"

**Cause:** The process list API returned an empty response.

**Solution:** Verify the sandbox is running. Retry if transient.

### "no server stream"

**Cause:** The gRPC stream for command output could not be established.

**Solution:** Verify the sandbox is running and the connection is healthy. Re-connect if needed.

---

## 6. Network and Connectivity Errors

### Connection refused / dial errors

**Cause:** The sandbox instance is not reachable.

**Solution:**
1. Verify the sandbox is still running with `core.List()` or `core.GetInfo()`.
2. Check that your network can reach `*.tencentags.com`.
3. If using a proxy, verify the proxy configuration in `connection.Config.Proxy`.

### TLS / certificate errors

**Cause:** TLS handshake failure when connecting to the sandbox.

**Solution:**
- Ensure your system's CA certificates are up to date.
- Verify the connection scheme is `"https"` (default).

---

## 7. Context and Timeout Errors

### "context deadline exceeded"

**Cause:** The operation did not complete within the context's deadline.

**Solution:**
- Increase the timeout for long-running operations:
  ```go
  ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
  defer cancel()
  ```
- For sandbox creation, the API call may take several seconds; use a timeout of at least 30 seconds.
- For code execution, consider the complexity of the code being run.

### "context canceled"

**Cause:** The context was explicitly cancelled (e.g., via `cancel()`), or the parent context was cancelled.

**Solution:** Ensure the context is not cancelled prematurely. Check your context hierarchy.

---

## Error Handling Patterns

### Basic error handling

```go
sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
if err != nil {
    log.Fatalf("Failed to create sandbox: %v", err)
}
defer func() { _ = sb.Kill(context.Background()) }()
```

### Distinguishing SDK errors from execution errors

```go
exec, err := sb.Code.RunCode(ctx, userCode, nil, nil)
if err != nil {
    // SDK-level failure: network, auth, timeout, etc.
    return fmt.Errorf("RunCode failed: %w", err)
}
if exec.Error != nil {
    // User code raised an exception
    return fmt.Errorf("code error [%s]: %s", exec.Error.Name, exec.Error.Value)
}
// Success: use exec.Results and exec.Logs
```

### Wrapping errors for context

```go
content, err := sb.Files.Read(ctx, path, nil)
if err != nil {
    return fmt.Errorf("failed to read %s from sandbox %s: %w", path, sb.SandboxId, err)
}
```

### Handling process lifecycle

```go
handle, err := sb.Commands.Start(ctx, "python3 app.py", nil, &command.OnOutputConfig{
    OnStdout: func(data []byte) { fmt.Print(string(data)) },
    OnStderr: func(data []byte) { fmt.Fprint(os.Stderr, string(data)) },
})
if err != nil {
    log.Fatalf("Failed to start process: %v", err)
}

// Wait for completion with timeout
waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
defer cancel()

result, err := handle.Wait(waitCtx)
if err != nil {
    // Could be context deadline exceeded, context canceled, or stream error
    log.Printf("Process wait error: %v", err)
    _ = handle.Kill(context.Background()) // Clean up
    return
}
if result.Error != nil {
    log.Printf("Process error: %s", *result.Error)
}
log.Printf("Exit code: %d", result.ExitCode)
```

---

## Retry Strategy

| Error Type | Retryable | Suggested Action |
|------------|-----------|------------------|
| Client initialization errors | No | Fix configuration |
| Authentication errors | No | Fix credentials |
| `"not found sandbox instance"` | No | Re-create sandbox |
| `"context deadline exceeded"` | Yes | Retry with longer timeout |
| HTTP 5xx errors | Yes | Retry with backoff |
| HTTP 401/403 errors | No | Reconnect to sandbox (token may have expired) |
| Connection refused/reset | Yes | Retry with backoff; verify sandbox is running |
| Nil response errors | Yes | Retry once; report if persistent |
| Validation errors (empty path, invalid user) | No | Fix input parameters |
| Code `ExecutionError` | No | Fix user code |

**Example retry with exponential backoff:**

```go
func withRetry(ctx context.Context, maxRetries int, fn func() error) error {
    var err error
    for i := 0; i < maxRetries; i++ {
        err = fn()
        if err == nil {
            return nil
        }
        // Don't retry validation or auth errors
        errMsg := err.Error()
        if strings.Contains(errMsg, "not initialized") ||
            strings.Contains(errMsg, "is empty") ||
            strings.Contains(errMsg, "invalid") ||
            strings.Contains(errMsg, "cannot be nil") {
            return err
        }
        backoff := time.Duration(1<<uint(i)) * time.Second
        select {
        case <-time.After(backoff):
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    return fmt.Errorf("failed after %d retries: %w", maxRetries, err)
}
```

---

## FAQ

### Q: My sandbox creation succeeds but tool operations fail immediately.

**A:** After creating a sandbox, it takes a moment for the internal services to be ready. If you get connection errors immediately after creation, add a brief delay or retry the first operation:

```go
sb, err := sandboxcode.Create(ctx, "code-interpreter-v1", sandboxcode.WithClient(client))
if err != nil {
    log.Fatal(err)
}
// The sandbox services may need a moment to initialize
```

### Q: How do I know if a code execution error is from my code or the SDK?

**A:** Check both return values:
- `err != nil` -> SDK error (request failed)
- `exec.Error != nil` -> Your code raised an exception (request succeeded)

### Q: My sandbox stops responding after a while.

**A:** Sandboxes have a configurable timeout. Use `SetTimeoutSeconds` to extend it:

```go
err := sb.SetTimeoutSeconds(ctx, 3600) // 1 hour
```

### Q: Can I reconnect to a sandbox after a network interruption?

**A:** Yes. Use `sandboxcode.Connect` with the sandbox ID:

```go
sb, err := sandboxcode.Connect(ctx, savedSandboxId, sandboxcode.WithClient(client))
```

This acquires a fresh access token for the existing sandbox instance.

### Q: What happens if I call Kill on an already-terminated sandbox?

**A:** The API will return an error. Check the sandbox status first with `List` or `GetInfo` if you need to handle this gracefully.
