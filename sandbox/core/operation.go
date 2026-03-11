package core

import (
	"context"
	"fmt"

	"github.com/TencentCloudAgentRuntime/ags-go-sdk/connection"
	"github.com/TencentCloudAgentRuntime/ags-go-sdk/constant"
	ags "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ags/v20250920"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

// Create creates a new sandbox instance with the specified tool configuration and options.
//
// This function orchestrates the creation of a sandbox by:
//  1. Starting a new sandbox instance using the specified tool configuration
//  2. Acquiring an access token for the created sandbox instance
//  3. Returning a Core struct configured with the instance ID and access token
//
// Parameters:
//   - ctx: Context for controlling the creation process, including timeouts and cancellation
//   - toolName: Name of the tool configuration that defines the sandbox instance to create
//   - opts: Variable number of CreateOption configurations for customizing creation behavior
//
// Returns:
//   - *Core: Core sandbox instance with connection configuration
//   - error: Any error encountered during sandbox creation or token acquisition
//
// Error conditions:
//   - Invalid tool name or tool configuration not found
//   - Network connectivity issues or AGS service unavailability
//   - Authentication or authorization failures
//   - Resource allocation failures (quota exceeded, insufficient resources)
func Create(ctx context.Context, toolName string, opts ...CreateOption) (*Core, error) {
	config := evaluateCreateOpts(opts)
	client, err := initializeClient(config.clientConfig)
	if err != nil {
		return nil, err
	}

	timeoutString := config.sandboxTimeout.String()

	// Build StartSandboxInstanceRequest with full SandboxConfig parameters
	request := &ags.StartSandboxInstanceRequest{
		ToolName: &toolName,
		Timeout:  &timeoutString,
	}

	if config.sandboxConfig != nil {
		// Add MountOptions if configured
		if config.sandboxConfig.MountOptions != nil {
			request.MountOptions = convertMountOptions(config.sandboxConfig.MountOptions)
		}
	}

	startResponse, err := client.StartSandboxInstanceWithContext(ctx, request)
	if err != nil {
		return nil, err
	}
	if startResponse.Response == nil {
		return nil, fmt.Errorf("StartSandboxInstance response.Response is nil")
	}
	if startResponse.Response.Instance == nil {
		return nil, fmt.Errorf("StartSandboxInstance response.Response.Instance is nil. Response: %+v", startResponse.Response)
	}
	if startResponse.Response.Instance.InstanceId == nil {
		return nil, fmt.Errorf("StartSandboxInstance response.Response.Instance.InstanceId is nil")
	}

	tokenResponse, err := client.AcquireSandboxInstanceTokenWithContext(ctx, &ags.AcquireSandboxInstanceTokenRequest{
		InstanceId: startResponse.Response.Instance.InstanceId,
	})
	if err != nil {
		return nil, err
	}
	if tokenResponse.Response == nil {
		return nil, fmt.Errorf("AcquireSandboxInstanceToken response.Response is nil")
	}
	if tokenResponse.Response.Token == nil {
		return nil, fmt.Errorf("AcquireSandboxInstanceToken response.Response.Token is nil")
	}

	return NewCore(client, *startResponse.Response.Instance.InstanceId, &connection.Config{
		Domain:      fmt.Sprintf("%v.%v", client.GetRegion(), config.domain),
		AccessToken: *tokenResponse.Response.Token,
		Scheme:      config.scheme,
	}), nil
}

// Connect prepares access to an existing sandbox instance using its ID.
//
// This function prepares access to a previously created sandbox by:
//  1. Acquiring an access token for the specified sandbox instance
//  2. Returning a Core struct configured with the instance ID and access token
//
// Note: This function does not directly establish a connection to the sandbox. No I/O is performed.
// It only stores the instance ID and access token to a Core for later use.
//
// Parameters:
//   - ctx: Context for controlling the access preparation process, including timeouts and cancellation
//   - sandboxId: Unique identifier of the existing sandbox instance to prepare access for
//   - opts: Variable number of ConnectOption configurations for customizing access preparation behavior
//
// Returns:
//   - *Core: Core sandbox instance with connection configuration
//   - error: Any error encountered during token acquisition
//
// Error conditions:
//   - Invalid sandbox ID or sandbox instance does not exist
//   - Network connectivity issues or AGS service unavailability
//   - Authentication or authorization failures
//   - Sandbox instance is not in a connectable state
func Connect(ctx context.Context, sandboxId string, opts ...ConnectOption) (*Core, error) {
	cfg := evaluateConnectOpts(opts)
	client, err := initializeClient(cfg.clientConfig)
	if err != nil {
		return nil, err
	}
	token, err := client.AcquireSandboxInstanceTokenWithContext(
		ctx, &ags.AcquireSandboxInstanceTokenRequest{InstanceId: &sandboxId},
	)
	if err != nil {
		return nil, err
	}
	if token.Response == nil {
		return nil, fmt.Errorf("AcquireSandboxInstanceToken response.Response is nil")
	}
	if token.Response.Token == nil {
		return nil, fmt.Errorf("AcquireSandboxInstanceToken response.Response.Token is nil")
	}

	return NewCore(client, sandboxId, &connection.Config{
		Domain:      fmt.Sprintf("%v.%v", client.GetRegion(), cfg.domain),
		AccessToken: *token.Response.Token,
		Scheme:      cfg.scheme,
	}), nil
}

// List retrieves information about existing sandbox instances.
//
// This function queries the AGS service to obtain a list of sandbox instances
// that are visible to the authenticated user. Only instances accessible with
// the provided credentials are returned.
//
// Parameters:
//   - ctx: Context for controlling the list operation, including timeouts and cancellation
//   - opts: Variable number of ListOption configurations for customizing list behavior
//
// Returns:
//   - []*ags.SandboxInstance: Array of sandbox instances accessible to the authenticated user
//   - error: Any error encountered during the list operation
//
// Note: Only sandbox instances visible to the credential provided via WithCredential
// or the credential associated with the client set via WithClient are returned.
//
// Error conditions:
//   - Network connectivity issues or AGS service unavailability
//   - Authentication or authorization failures
//   - Invalid configuration options
func List(ctx context.Context, opts ...ListOption) ([]*ags.SandboxInstance, error) {
	cfg := evaluateListOpts(opts)
	client, err := initializeClient(cfg.clientConfig)
	if err != nil {
		return nil, err
	}
	res, err := client.DescribeSandboxInstanceListWithContext(ctx, &ags.DescribeSandboxInstanceListRequest{})
	if err != nil {
		return nil, err
	}
	if res.Response == nil {
		return nil, fmt.Errorf("DescribeSandboxInstanceList response.Response is nil")
	}
	return res.Response.InstanceSet, nil
}

// Kill terminates and deletes a sandbox instance by its ID.
//
// This function stops and removes the specified sandbox instance, releasing
// all associated resources including compute, storage, and network resources.
//
// Parameters:
//   - ctx: Context for controlling the kill operation, including timeouts and cancellation
//   - sandboxId: Unique identifier of the sandbox instance to terminate
//   - opts: Variable number of KillOption configurations for customizing kill behavior
//
// Returns:
//   - error: Any error encountered during the termination process
//
// Warning: This operation is irreversible. All data and state within the sandbox
// will be permanently lost. Ensure any important data is saved before calling Kill.
//
// Error conditions:
//   - Invalid sandbox ID or sandbox instance does not exist
//   - Network connectivity issues or AGS service unavailability
//   - Authentication or authorization failures
//   - Sandbox instance is already terminated or in a non-killable state
func Kill(ctx context.Context, sandboxId string, opts ...KillOption) error {
	cfg := evaluateKillOpts(opts)
	client, err := initializeClient(cfg.clientConfig)
	if err != nil {
		return err
	}
	_, err = client.StopSandboxInstanceWithContext(ctx, &ags.StopSandboxInstanceRequest{
		InstanceId: &sandboxId,
	})
	if err != nil {
		return err
	}
	return nil
}

// initializeClient creates and returns an AGS Tencent Cloud SDK client based on the provided configuration.
func initializeClient(cfg *clientConfig) (*ags.Client, error) {
	if cfg.client != nil {
		return cfg.client, nil
	}

	if cfg.credential != nil && cfg.region != "" {
		cpf := profile.NewClientProfile()
		cpf.HttpProfile.Endpoint = constant.AgentSandboxInternalEndpoint

		client, err := ags.NewClient(cfg.credential, cfg.region, cpf)
		if err != nil {
			return nil, err
		}

		return client, nil
	}

	return nil, fmt.Errorf("client cannot be initialized. Make sure you have provided a valid client, or a credential and region pair")
}

// convertMountOptions converts our MountOption to AGS SDK MountOption
func convertMountOptions(options []*MountOption) []*ags.MountOption {
	if len(options) == 0 {
		return nil
	}

	agsOptions := make([]*ags.MountOption, 0, len(options))
	for _, opt := range options {
		if opt == nil {
			continue
		}
		agsOpt := &ags.MountOption{
			Name:      opt.Name,
			MountPath: opt.MountPath,
			SubPath:   opt.SubPath,
			ReadOnly:  opt.ReadOnly,
		}
		agsOptions = append(agsOptions, agsOpt)
	}

	return agsOptions
}
