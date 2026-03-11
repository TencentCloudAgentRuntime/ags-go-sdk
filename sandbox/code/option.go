package code

import (
	"time"

	"github.com/TencentCloudAgentRuntime/ags-go-sdk/sandbox/core"
	ags "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ags/v20250920"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

// SandboxConfig contains configuration parameters for sandbox instance creation.
// This is a type alias for core.SandboxConfig.
type SandboxConfig = core.SandboxConfig

// MountOption contains configuration parameters for mounting storage in sandbox instance creation.
// This is a type alias for core.MountOption.
type MountOption = core.MountOption

// #===================================================================================================================#
// #                                                 CreateOption                                                      #
// #===================================================================================================================#

// configs for Create operations specific to code sandbox.
type createConfig struct {
	*clientConfig
	*dataPlaneConfig
	// Core create options that will be passed to core.Create
	coreCreateOptions []core.CreateOption
}

// evaluateCreateOpts evaluates the provided options and returns the configuration
func evaluateCreateOpts(options []CreateOption) *createConfig {
	config := &createConfig{
		clientConfig: &clientConfig{
			clientOptions: make([]core.ClientOption, 0),
		},
		dataPlaneConfig: &dataPlaneConfig{
			dataPlaneOptions: make([]core.DataPlaneOption, 0),
		},
		coreCreateOptions: make([]core.CreateOption, 0),
	}

	for _, option := range options {
		option.applyCreate(config)
	}

	return config
}

// CreateOption defines options specific to code sandbox Create operations
type CreateOption interface {
	applyCreate(*createConfig)
}

// createOptionFunc is a function adapter for CreateOption
type createOptionFunc func(*createConfig)

func (f createOptionFunc) applyCreate(config *createConfig) {
	f(config)
}

// WithSandboxTimeout sets the timeout for the sandbox instance lifecycle.
// The timeout parameter should be a time.Duration (e.g., 300*time.Second, 5*time.Minute, 1*time.Hour).
// This determines how long the sandbox instance will remain active before automatic termination.
// Default timeout is 300s if not specified.
func WithSandboxTimeout(timeout time.Duration) CreateOption {
	return createOptionFunc(func(config *createConfig) {
		config.coreCreateOptions = append(config.coreCreateOptions, core.WithSandboxTimeout(timeout))
	})
}

// WithSandboxConfig sets the sandbox configuration including storage parameters and optional timeout.
// This option allows specifying detailed configuration for the sandbox instance.
// Note: This option is ignored if WithSandboxConfig is used and contains a timeout value.
func WithSandboxConfig(sandboxConfig *SandboxConfig) CreateOption {
	return createOptionFunc(func(config *createConfig) {
		config.coreCreateOptions = append(config.coreCreateOptions, core.WithSandboxConfig(sandboxConfig))
	})
}

// #===================================================================================================================#
// #                                                ConnectOption                                                      #
// #===================================================================================================================#

// configs for Connect operations specific to code sandbox.
type connectConfig struct {
	*clientConfig
	*dataPlaneConfig
	// Core connect options that will be passed to core.Connect
	coreConnectOptions []core.ConnectOption
}

// evaluateConnectOpts evaluates the provided options and returns the configuration
func evaluateConnectOpts(options []ConnectOption) *connectConfig {
	config := &connectConfig{
		clientConfig: &clientConfig{
			clientOptions: make([]core.ClientOption, 0),
		},
		dataPlaneConfig: &dataPlaneConfig{
			dataPlaneOptions: make([]core.DataPlaneOption, 0),
		},
		coreConnectOptions: make([]core.ConnectOption, 0),
	}

	for _, option := range options {
		option.applyConnect(config)
	}

	return config
}

// ConnectOption defines options specific to code sandbox Connect operations
type ConnectOption interface {
	applyConnect(*connectConfig)
}

// connectOptionFunc is a function adapter for ConnectOption
//
//nolint:unused // reserved for future use
type connectOptionFunc func(*connectConfig)

//nolint:unused // reserved for future use
func (f connectOptionFunc) applyConnect(config *connectConfig) {
	f(config)
}

// #===================================================================================================================#
// #                                                 ClientOption                                                      #
// #===================================================================================================================#

// clientConfig contains the client configuration options for sandbox operations.
type clientConfig struct {
	// Client options that will be passed to core operations
	clientOptions []core.ClientOption
}

// ClientOption defines a option for configuring Tencent Cloud SDK Agent Sandbox client.
type ClientOption interface {
	CreateOption
	ConnectOption
	ListOption
	KillOption
}

// clientOptionFunc is a function adapter for ClientOption
type clientOptionFunc func(*clientConfig)

func (f clientOptionFunc) applyCreate(config *createConfig) {
	f(config.clientConfig)
}

func (f clientOptionFunc) applyConnect(config *connectConfig) {
	f(config.clientConfig)
}

func (f clientOptionFunc) applyList(config *listConfig) {
	f(config.clientConfig)
}

func (f clientOptionFunc) applyKill(config *killConfig) {
	f(config.clientConfig)
}

// WithClient sets the AGS client instance. This option has higher priority than WithCredential and WithRegion.
// When this option is set, operations are performed using the given AGS client.
func WithClient(client *ags.Client) ClientOption {
	return clientOptionFunc(func(config *clientConfig) {
		config.clientOptions = append(config.clientOptions, core.WithClient(client))
	})
}

// WithCredential sets the credentials of the AGS client that will be created in order to call
// Tencent Cloud AgentSandbox APIs to perform operations.
// When WithClient option is not set, this option is used together with WithRegion to create a new AGS client which
// will be used to perform operations.
// However, if WithRegion option is not set, the default region will be used.
func WithCredential(credential common.CredentialIface) ClientOption {
	return clientOptionFunc(func(config *clientConfig) {
		config.clientOptions = append(config.clientOptions, core.WithCredential(credential))
	})
}

// WithRegion sets the Tencent Cloud region of the AGS client that will be created in order to call Tencent Cloud
// AgentSandbox APIs to perform operations.
// When WithClient option is not set, this option is used together with WithCredential to create a new AGS client which
// will be used to perform operations.
// However, if WithCredential option is not set, an error will be returned by the operation function.
func WithRegion(region string) ClientOption {
	return clientOptionFunc(func(config *clientConfig) {
		config.clientOptions = append(config.clientOptions, core.WithRegion(region))
	})
}

// #===================================================================================================================#
// #                                            DataPlaneOption                                                   #
// #===================================================================================================================#

// DataPlaneOption defines options for configuring domain, applicable to Create and Connect operations.
type DataPlaneOption interface {
	CreateOption
	ConnectOption
}

// dataPlaneOptionFunc is a function adapter for DataPlaneOption
type dataPlaneOptionFunc func(*dataPlaneConfig)

func (f dataPlaneOptionFunc) applyCreate(config *createConfig) {
	f(config.dataPlaneConfig)
}

func (f dataPlaneOptionFunc) applyConnect(config *connectConfig) {
	f(config.dataPlaneConfig)
}

// dataPlaneConfig contains the sandbox contact configuration options.
type dataPlaneConfig struct {
	// Core sandbox contact options that will be passed to core operations
	dataPlaneOptions []core.DataPlaneOption
}

// WithDataPlaneDomain sets a custom domain for contacting sandboxes. This sets the data plane domain.
// Default is core.DefaultDataPlaneDomain.
func WithDataPlaneDomain(domain string) DataPlaneOption {
	return dataPlaneOptionFunc(func(config *dataPlaneConfig) {
		config.dataPlaneOptions = append(config.dataPlaneOptions, core.WithDataPlaneDomain(domain))
	})
}

// WithScheme sets the URL scheme for data plane connections.
// Supported values are "http" and "https". Default is "https".
func WithScheme(scheme string) DataPlaneOption {
	return dataPlaneOptionFunc(func(config *dataPlaneConfig) {
		config.dataPlaneOptions = append(config.dataPlaneOptions, core.WithScheme(scheme))
	})
}

// #===================================================================================================================#
// #                                                 ListOption                                                        #
// #===================================================================================================================#

// configs for List operations specific to code sandbox.
type listConfig struct {
	*clientConfig
}

// evaluateListOpts evaluates the provided options and returns the configuration
func evaluateListOpts(options []ListOption) *listConfig {
	config := &listConfig{
		clientConfig: &clientConfig{
			clientOptions: make([]core.ClientOption, 0),
		},
	}

	for _, option := range options {
		option.applyList(config)
	}

	return config
}

// ListOption defines options specific to code sandbox List operations
type ListOption interface {
	applyList(*listConfig)
}

// listOptionFunc is a function adapter for ListOption
//
//nolint:unused // reserved for future use
type listOptionFunc func(*listConfig)

//nolint:unused // reserved for future use
func (f listOptionFunc) applyList(config *listConfig) {
	f(config)
}

// #===================================================================================================================#
// #                                                 KillOption                                                        #
// #===================================================================================================================#

// configs for Kill operations specific to code sandbox.
type killConfig struct {
	*clientConfig
}

// evaluateKillOpts evaluates the provided options and returns the configuration
func evaluateKillOpts(options []KillOption) *killConfig {
	config := &killConfig{
		clientConfig: &clientConfig{
			clientOptions: make([]core.ClientOption, 0),
		},
	}

	for _, option := range options {
		option.applyKill(config)
	}

	return config
}

// KillOption defines options specific to code sandbox Kill operations
type KillOption interface {
	applyKill(*killConfig)
}

// killOptionFunc is a function adapter for KillOption
//
//nolint:unused // reserved for future use
type killOptionFunc func(*killConfig)

//nolint:unused // reserved for future use
func (f killOptionFunc) applyKill(config *killConfig) {
	f(config)
}
