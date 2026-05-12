# Contributing Guide

Thank you for your interest in Agent Sandbox Go SDK! We welcome all forms of contributions, including but not limited to:

- Bug reports
- Feature suggestions
- Code fixes or new features
- Documentation improvements

## Development Environment

### Prerequisites

- Go 1.22+
- [buf](https://buf.build/) (for proto code generation)
- Tencent Cloud account with Agent Sandbox access

### Install Development Tools

```bash
# Install buf
make tools
```

### Clone Repository

```bash
git clone https://github.com/TencentCloudAgentRuntime/ags-go-sdk.git
cd ags-go-sdk
```

## Development Workflow

### 1. Create Branch

Please create your feature branch based on `main`:

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
```

Branch naming conventions:
- `feature/xxx` - New features
- `fix/xxx` - Bug fixes
- `docs/xxx` - Documentation updates
- `refactor/xxx` - Code refactoring
- `chore/xxx` - Build/tooling related

### 2. Write Code

Please follow these guidelines:

- Follow Go official code style, use `gofmt` to format code
- Add documentation comments for exported functions and types
- Ensure code passes `go vet` and `golint` checks
- Write unit tests to cover new code

### 3. Proto File Modifications

If you modify proto files in the `proto/` directory, please run:

```bash
make gen
```

to regenerate Go code.

### 4. Run Tests

```bash
cd test
go test -v ./...
```

### 5. Commit Code

Commit messages should follow this format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation update
- `style`: Code formatting (no logic changes)
- `refactor`: Code refactoring
- `test`: Test related
- `chore`: Build/tooling related

Example:
```
feat(sandbox): add browser sandbox support

- Implement BrowserSandbox type
- Add browser operation APIs
- Add unit tests

Closes #123
```

### 6. Submit Merge Request

- Ensure all tests pass
- Ensure code is formatted
- Fill in clear MR description
- Link related Issues or TAPD tickets

## Code Standards

### Go Code Style

- Use `gofmt` to format code
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### Naming Conventions

- Package names: lowercase, short, no underscores
- Exported identifiers: CamelCase
- Non-exported identifiers: camelCase
- Constants: CamelCase

### Error Handling

- Prefer using `errors.New()` or `fmt.Errorf()` to create errors
- Error messages start with lowercase, no trailing punctuation
- Use `%w` to wrap errors to preserve error chain

```go
if err != nil {
    return fmt.Errorf("failed to create sandbox: %w", err)
}
```

### Comment Guidelines

- All exported types, functions, constants must have documentation comments
- Comments start with the name of the described object
- Use complete sentences

```go
// Create creates a new code sandbox with the given template.
// It returns a Sandbox instance and any error encountered.
func Create(ctx context.Context, template string, opts ...Option) (*Sandbox, error) {
    // ...
}
```

## Directory Structure

```
ags-go-sdk/
├── connection/     # Connection management
├── constant/       # Constant definitions
├── docs/           # Documentation
├── example/        # Usage examples
├── pb/             # Generated protobuf code
├── proto/          # Proto definition files
├── sandbox/        # Sandbox core implementation
│   ├── code/       # Code sandbox
│   ├── core/       # Core functionality
│   └── browser/    # Browser sandbox (to be implemented)
├── test/           # Test code
└── tool/           # Tool clients
    ├── code/       # Code execution
    ├── command/    # Command execution
    └── filesystem/ # Filesystem operations
```

## Reporting Bugs

If you find a bug, please report it through:

1. Create an Issue in the project
2. Include the following information:
   - SDK version
   - Go version
   - Operating system
   - Steps to reproduce
   - Expected behavior
   - Actual behavior
   - Related logs or error messages

## Feature Suggestions

If you have feature suggestions, please:

1. Create an Issue in the project
2. Describe your use case
3. Explain the expected feature behavior

## Contact

For any questions, please contact the project maintainers.

## License

By submitting code, you agree that your contributions will be licensed under the project license.
