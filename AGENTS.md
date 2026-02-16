# AGENTS.md

This file provides context and instructions for AI agents working on the `commiter` repository.

## Project Overview
`commiter` is a CLI tool written in Go that automates the git commit process. It stages all changes, generates a Conventional Commits message using the OpenRouter API, and commits the changes.

## Build and Run
- **Build:** `go build -o commiter`
- **Run:** `go run .` or `./commiter`
- **Dependencies:** Standard Go library.
- **Environment:** Requires `OPENROUTER_API_KEY` environment variable to be set.

## Testing
- **Command:** `go test ./...`
- **Single Test:** `go test -run TestName ./...`
- **Current State:** The project currently lacks test files. New features should include unit tests in `*_test.go` files using the standard `testing` package.

## Code Style & Conventions

### Formatting
- **Standard:** Strictly follow `gofmt` standards.
- **Imports:** Group standard library imports first, followed by third-party (if any), then internal imports.

### Naming
- **Files:** Lowercase, snake_case is acceptable but single-word preferred (e.g., `git.go`, `spinner.go`).
- **Functions/Types:** PascalCase for exported, camelCase for internal.
- **Variables:** Short, descriptive names (e.g., `err`, `ctx`, `diff`).

### Error Handling
- **Pattern:** Use `if err != nil` immediately after the function call.
- **Context:** Wrap errors with context when bubbling up (e.g., `fmt.Errorf("failed to ...: %w", err)`).
- **Custom Errors:** See `git.go` for the `errorf` helper pattern, though standard `fmt.Errorf` is preferred for new code unless specific formatting is needed.

### Architecture
- **Package:** Currently uses a flat `main` package structure. Refactor to separate packages (e.g., `pkg/git`, `pkg/llm`) if complexity grows.
- **Configuration:** Use flags for runtime options (see `main.go`).

## Development Workflow
1.  **Analyze:** specific files related to the task.
2.  **Plan:** Describe the changes.
3.  **Modify:** Use `edit` or `write` to update code.
4.  **Verify:** Run `go fmt ./...` and `go vet ./...` to ensure code quality.
