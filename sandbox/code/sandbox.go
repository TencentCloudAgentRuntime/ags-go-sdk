package code

import (
	"context"

	"github.com/TencentCloudAgentRuntime/ags-go-sdk/connection"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/constant"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/sandbox/core"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/tool/code"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/tool/command"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/tool/filesystem"
	ags "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ags/v20250920"
)

// Sandbox represents a code execution sandbox instance that provides
// comprehensive capabilities for code execution, file management, and process control.
//
// The Sandbox embeds the core.Core functionality and extends it with specialized
// clients for different execution operations:
//
//   - Files: Provides filesystem operations (Read, Write, List, Remove, MakeDir, file watching)
//   - Commands: Enables process management and command execution within the sandbox
//   - Code: Offers code execution capabilities with support for multiple programming languages
//
// All clients are automatically configured with the appropriate connection settings
// including authentication tokens, domain endpoints, and proxy configurations.
//
// Example usage:
//
//	sandbox, err := Create(ctx, "python-executor",
//	    WithRegion("ap-guangzhou"),
//	    WithCredential(credential),
//	)
//	if err != nil {
//	    return err
//	}
//	defer sandbox.Kill(ctx)
//
//	// Use filesystem operations
//	reader, err := sandbox.Files.Read(ctx, "/path/to/file", nil)
//
//	// Execute commands
//	result, err := sandbox.Commands.RunForeground(ctx, &command.ProcessConfig{Cmd: "ls -la"}, nil)
//
//	// Execute code
//	execution, err := sandbox.Code.RunCode(ctx, "print('Hello World')", nil)
type Sandbox struct {
	// Core provides the fundamental sandbox functionality including connection management,
	// lifecycle operations (Kill, SetTimeoutSeconds), and host information retrieval.
	*core.Core

	// Files provides filesystem operations within the sandbox instance.
	// Supports Read, Write, List, Remove, Rename, MakeDir, file watching, and path operations.
	Files *filesystem.Client

	// Commands enables process management and command execution within the sandbox.
	// Supports Start, Connect, RunForeground, SendInput, SendSignal, and process monitoring.
	Commands *command.Client

	// Code provides code execution capabilities supporting multiple programming languages.
	// Enables running code snippets, creating execution contexts, and managing code sessions.
	Code *code.Client
}

// Create initializes and returns a new code execution sandbox instance with the specified tool and configuration options.
//
// This function calls the underlying core.Create operation and then orchestrates the setup of a complete code Sandbox by:
//  1. Creating the underlying sandbox instance using core.Create with the specified tool configuration
//  2. Initializing specialized clients for filesystem operations, process management, and code execution
//
// Parameters:
//   - ctx: Context for controlling the creation process, including timeouts and cancellation
//   - toolName: Identifier for the sandbox tool which contains the configuration of the code sandbox to be created.
//   - opts: Variable number of CreateOption configurations for customizing sandbox behavior
//
// The function automatically configures three specialized clients:
//   - Files: Connected to EnvdPort for filesystem operations
//   - Commands: Connected to EnvdPort for command execution
//   - Code: Connected to CodePort for code execution capabilities
//
// Returns:
//   - *Sandbox: Fully configured sandbox instance ready for code execution operations
//   - error: Any error encountered during sandbox creation or client initialization
//
// Example usage:
//
//	// Basic sandbox creation
//	sandbox, err := Create(ctx, "nodejs-executor")
//	if err != nil {
//	    return fmt.Errorf("failed to create sandbox: %w", err)
//	}
//	defer sandbox.Kill(ctx)
//
//	// Sandbox with custom configuration
//	sandbox, err := Create(ctx, "python-executor",
//	    WithRegion("ap-guangzhou"),
//	    WithCredential(credential),
//	)
//
// Error conditions:
//   - Sandbox creation failure (network, authentication, resource allocation)
//   - Client initialization failure (service unavailable, configuration errors)
//   - Invalid tool name or unsupported configuration options
func Create(ctx context.Context, toolName string, opts ...CreateOption) (*Sandbox, error) {
	// Evaluate and extract configuration options
	config := evaluateCreateOpts(opts)

	// Create the underlying sandbox instance
	createOptions := make([]core.CreateOption, len(config.clientOptions)+len(config.dataPlaneOptions)+len(config.coreCreateOptions))
	// Add client options (converted to CreateOptions)
	for i, opt := range config.clientOptions {
		createOptions[i] = opt
	}
	// Add core contact options
	offset := len(config.clientOptions)
	for i, opt := range config.dataPlaneOptions {
		createOptions[offset+i] = opt
	}
	// Add core create options
	offset += len(config.dataPlaneOptions)
	for i, opt := range config.coreCreateOptions {
		createOptions[offset+i] = opt
	}
	sandbox, err := core.Create(ctx, toolName, createOptions...)
	if err != nil {
		return nil, err
	}

	// Initialize the sandbox wrapper with core functionality
	ret := &Sandbox{
		Core: sandbox,
	}

	// Initialize all specialized clients
	if err := initializeClients(ret); err != nil {
		return nil, err
	}

	return ret, nil
}

// initializeClients initializes the specialized clients for the Sandbox.
// This function sets up the Files, Commands, and Code clients using the sandbox's
// connection configuration and host information.
//
// Parameters:
//   - sandbox: The Sandbox instance to initialize clients for
//
// Returns:
//   - error: Any error encountered during client initialization
//
// The function initializes three clients:
//   - Files: Connected to EnvdPort for filesystem operations
//   - Commands: Connected to EnvdPort for command execution
//   - Code: Connected to CodePort for code execution capabilities
func initializeClients(sandbox *Sandbox) error {
	var err error

	// Initialize filesystem client for file operations
	// Connected to EnvdPort for sandbox filesystem access
	sandbox.Files, err = filesystem.New(
		&connection.Config{
			Domain:      sandbox.GetHost(constant.EnvdPort),
			AccessToken: sandbox.ConnectionConfig.AccessToken,
			Headers:     sandbox.ConnectionConfig.Headers,
			Proxy:       sandbox.ConnectionConfig.Proxy,
			Scheme:      sandbox.ConnectionConfig.Scheme,
		})
	if err != nil {
		return err
	}

	// Initialize command execution client
	// Connected to EnvdPort for shell and script execution within the sandbox
	sandbox.Commands, err = command.New(&connection.Config{
		Domain:      sandbox.GetHost(constant.EnvdPort),
		AccessToken: sandbox.ConnectionConfig.AccessToken,
		Headers:     sandbox.ConnectionConfig.Headers,
		Proxy:       sandbox.ConnectionConfig.Proxy,
		Scheme:      sandbox.ConnectionConfig.Scheme,
	})
	if err != nil {
		return err
	}

	// Initialize code execution client
	// Connected to CodePort for running code in various programming languages
	sandbox.Code = code.New(&connection.Config{
		Domain:      sandbox.GetHost(constant.CodePort),
		AccessToken: sandbox.ConnectionConfig.AccessToken,
		Headers:     sandbox.ConnectionConfig.Headers,
		Proxy:       sandbox.ConnectionConfig.Proxy,
		Scheme:      sandbox.ConnectionConfig.Scheme,
	})

	return nil
}

// Connect returns a Sandbox struct for accessing an existing code execution sandbox instance with the specified sandbox ID and configuration options.
//
// This function calls the underlying core.Connect operation and then sets up access to a previously created sandbox instance by:
//  1. Calling core.Connect to obtain access credentials for the existing sandbox instance
//  2. Initializing specialized clients for filesystem operations, process management, and code execution
//
// Note: This function does not directly establish a connection to the sandbox. No I/O is performed.
// Instead, it returns a Sandbox struct whose Core, Files, Commands, and Code fields can be used to
// establish connections and access the sandbox services.
//
// Parameters:
//   - ctx: Context for controlling the access preparation process, including timeouts and cancellation
//   - sandboxId: Unique identifier of the existing sandbox instance to prepare access for
//   - opts: Variable number of ConnectOption configurations for customizing access preparation behavior
//
// The function automatically configures three specialized clients:
//   - Files: Connected to EnvdPort for filesystem operations
//   - Commands: Connected to EnvdPort for command execution
//   - Code: Connected to CodePort for code execution capabilities
//
// Returns:
//   - *Sandbox: Fully configured sandbox instance ready for code execution operations
//   - error: Any error encountered during sandbox connection or client initialization
//
// Example usage:
//
//	// Basic sandbox connection using sandbox ID
//	sandbox, err := Connect(ctx, "9ec2e89cfea7408bb8ffcf27a27e26af")
//	if err != nil {
//	    return fmt.Errorf("failed to connect to sandbox: %w", err)
//	}
//	defer sandbox.Kill(ctx)
//
//	// Sandbox connection with custom configuration
//	sandbox, err := Connect(ctx, "9ec2e89cfea7408bb8ffcf27a27e26af",
//	    WithRegion("ap-guangzhou"),
//	    WithCredential(credential),
//	)
//
// Error conditions:
//   - Sandbox connection failure (network, authentication, sandbox not found)
//   - Client initialization failure (service unavailable, configuration errors)
//   - Invalid sandbox ID or sandbox instance does not exist
func Connect(ctx context.Context, sandboxId string, opts ...ConnectOption) (*Sandbox, error) {
	// Evaluate and extract configuration options
	config := evaluateConnectOpts(opts)

	// Connect to the existing sandbox instance
	connectOptions := make([]core.ConnectOption, len(config.clientOptions)+len(config.dataPlaneOptions)+len(config.coreConnectOptions))
	for i, opt := range config.clientOptions {
		connectOptions[i] = opt
	}
	// Add core contact options
	offset := len(config.clientOptions)
	for i, opt := range config.dataPlaneOptions {
		connectOptions[offset+i] = opt
	}
	// Add core connect options
	offset += len(config.dataPlaneOptions)
	for i, opt := range config.coreConnectOptions {
		connectOptions[offset+i] = opt
	}
	sandbox, err := core.Connect(ctx, sandboxId, connectOptions...)
	if err != nil {
		return nil, err
	}

	// Initialize the sandbox wrapper with core functionality
	ret := &Sandbox{
		Core: sandbox,
	}

	// Initialize all specialized clients
	if err := initializeClients(ret); err != nil {
		return nil, err
	}

	return ret, nil
}

// List returns a list of sandbox instances by calling the underlying core.List operation.
//
// This function delegates to core.List, allowing users to retrieve information about
// existing sandbox instances. Only sandbox instances visible to the credential given by
// WithCredential or the credential set to the client set via WithClient are returned.
//
// Parameters:
//   - ctx: Context for controlling the list operation, including timeouts and cancellation
//   - opts: Variable number of ListOption configurations for customizing the list behavior
//
// The function calls core.List to retrieve sandbox instance information from the AGS service.
//
// Returns:
//   - []*ags.SandboxInstance: Array of sandbox instances matching the specified criteria
//   - error: Any error encountered during the list operation
//
// Example usage:
//
//	// List all sandboxes with default configuration
//	instances, err := List(ctx)
//	if err != nil {
//	    return fmt.Errorf("failed to list sandboxes: %w", err)
//	}
//
//	// List sandboxes with custom client configuration
//	instances, err := List(ctx,
//	    WithRegion("ap-guangzhou"),
//	    WithCredential(credential),
//	)
//
// Error conditions:
//   - Network connectivity issues
//   - Authentication or authorization failures
//   - Invalid configuration options
//   - AGS service unavailability
func List(ctx context.Context, opts ...ListOption) ([]*ags.SandboxInstance, error) {
	// Evaluate and extract configuration options
	config := evaluateListOpts(opts)

	// Delegate to core.List with the extracted core options
	listOptions := make([]core.ListOption, len(config.clientOptions))
	for i, opt := range config.clientOptions {
		listOptions[i] = opt
	}
	return core.List(ctx, listOptions...)
}

// Kill terminates the sandbox by calling the underlying core.Kill operation.
//
// This function delegates to core.Kill to terminate and delete the specified sandbox instance.
// All associated resources including compute, storage, and network resources will be released.
//
// Parameters:
//   - ctx: Context for controlling the kill operation, including timeouts and cancellation
//   - sandboxId: Unique identifier of the sandbox instance to terminate
//   - opts: Variable number of KillOption configurations for customizing kill behavior
//
// Returns:
//   - error: Any error encountered during the termination process
//
// The function calls core.Kill to terminate and remove the specified sandbox instance.
//
// Warning: This operation is irreversible. All data and state within the sandbox
// will be permanently lost.
func Kill(ctx context.Context, sandboxId string, opts ...KillOption) error {
	// Evaluate and extract configuration options
	config := evaluateKillOpts(opts)

	// Delegate to core.Kill with the extracted core options
	killOptions := make([]core.KillOption, len(config.clientOptions))
	for i, opt := range config.clientOptions {
		killOptions[i] = opt
	}
	return core.Kill(ctx, sandboxId, killOptions...)
}
