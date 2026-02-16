# Commiter - Implementation Plan

## Overview

A Go CLI tool that automatically generates and applies Conventional Commit messages by sending `git diff` output to OpenRouter's API (model: `arcee-ai/trinity-mini:free`).

**Workflow:** `commiter` → stages all changes → reads diff → calls OpenRouter → auto-commits with generated message.

---

## Project Structure

```
commiter/
├── main.go              # Entry point, orchestrates the pipeline
├── git.go               # Git operations (diff, stage, commit, validation)
├── openrouter.go        # OpenRouter API client
├── prompt.go            # System/user prompt construction
├── go.mod
└── go.sum
```

Single-package (`package main`), no external dependencies beyond the Go standard library. This keeps the binary small, the build fast, and the attack surface minimal.

---

## Step-by-Step Implementation

### 1. Project Init

- `go mod init commiter` in `~/code/commiter`
- Go 1.22+ (for standard library improvements)

### 2. `git.go` - Git Operations

All git interactions via `os/exec` calling the `git` binary:

| Function | Purpose |
|---|---|
| `isGitRepo()` | Run `git rev-parse --is-inside-work-tree` - fail fast if not a repo |
| `stageAll()` | Run `git add -A` to stage all changes |
| `getStagedDiff()` | Run `git diff --cached` to get the staged diff |
| `commit(msg)` | Run `git commit -m <msg>` |

**Security:** All `exec.Command` calls use argument arrays (no shell interpolation), preventing command injection. The commit message is passed as a direct argument, not through a shell.

**Error handling:** Each function returns `(result, error)`. Errors include the stderr output from git for debuggability.

### 3. `openrouter.go` - API Client

- Endpoint: `POST https://openrouter.ai/api/v1/chat/completions`
- Auth: `Authorization: Bearer $OPENROUTER_API_KEY`
- Model: `arcee-ai/trinity-mini:free` (hardcoded default, overridable via `--model` flag)
- Request body:

```json
{
  "model": "arcee-ai/trinity-mini:free",
  "messages": [
    {"role": "system", "content": "<system prompt>"},
    {"role": "user", "content": "<diff>"}
  ],
  "temperature": 0.3,
  "max_tokens": 100
}
```

- Low temperature (0.3) for deterministic, focused output
- `max_tokens: 100` to enforce short messages
- HTTP client with **10-second timeout** (`http.Client{Timeout: 10 * time.Second}`)
- Parse response, extract `choices[0].message.content`, trim whitespace

**Security:**
- API key read from `os.Getenv("OPENROUTER_API_KEY")` only - never logged, never embedded
- Fail immediately with clear error if env var is empty
- TLS enforced by using `https://` endpoint (Go's `net/http` validates certificates by default)

**Error handling:**
- Non-2xx responses: read body, return descriptive error with HTTP status
- JSON parse failures: return raw body in error for debugging
- Empty `choices` array: explicit error message

### 4. `prompt.go` - Prompt Engineering

System prompt (hardcoded):

```
You are a commit message generator. Given a git diff, produce a single-line
commit message following the Conventional Commits format.

Rules:
- Use one of: feat, fix, docs, style, refactor, perf, test, build, ci, chore
- Format: <type>(<optional scope>): <description>
- Description must be lowercase, imperative mood, no period at the end
- Maximum 72 characters total
- Output ONLY the commit message, nothing else
```

The user message is simply the raw diff content.

### 5. `main.go` - Pipeline Orchestration

```go
func main():
  1. Parse flags (--model to override default model)
  2. Read OPENROUTER_API_KEY from env → fail if missing
  3. isGitRepo() → fail if not
  4. stageAll() → fail if git add fails
  5. getStagedDiff() → fail if empty (nothing to commit)
  6. generateCommitMessage(diff) → call OpenRouter
  7. Print the generated message to stdout
  8. commit(message) → apply the commit
  9. Print success confirmation
```

**Exit codes:**
- `0` - success
- `1` - any error (git, API, env)

Uses `flag` package from stdlib for `--model` flag. Uses `fmt.Fprintf(os.Stderr, ...)` for errors, `fmt.Println` for normal output.

---

## Security Checklist

| Concern | Mitigation |
|---|---|
| API key exposure | Read from env var only, never logged or printed |
| Command injection | `exec.Command` with argument arrays, no shell |
| TLS/MITM | Go's `net/http` enforces TLS certificate validation |
| Commit message injection | Passed as single `-m` argument, no shell expansion |
| Dependency supply chain | Zero external dependencies (stdlib only) |

---

## Performance Considerations

| Area | Approach |
|---|---|
| Binary size | Single binary, no CGO, stdlib only (~5-7MB) |
| Startup time | Near-instant (no runtime, no dependency loading) |
| Network | Single HTTP request with 10s timeout |
| Git calls | 3-4 subprocess calls (sub-millisecond each) |

---

## CLI Flags

| Flag | Default | Description |
|---|---|---|
| `--model` | `arcee-ai/trinity-mini:free` | OpenRouter model to use |

---

## Build & Install

```bash
# Build
go build -o commiter .

# Install to PATH (local)
go build -o commiter . && mv commiter /usr/local/bin/
```

---

## Example Usage

```bash
$ export OPENROUTER_API_KEY="sk-or-..."

$ cd my-project
$ # make some changes...
$ commiter
Staging all changes...
Generating commit message...
feat(auth): add jwt token refresh logic
Committed successfully.
```

---

## Error Scenarios

```bash
$ commiter                    # no API key
Error: OPENROUTER_API_KEY environment variable is not set

$ commiter                    # not in a git repo
Error: not a git repository

$ commiter                    # no changes
Error: no changes to commit
```

---

## Design Decisions

1. **No interactive confirmation** - Auto-commit for speed (user preference)
2. **Stage all + commit** - Simple UX, single command does everything
3. **Conventional Commits** - Standard format, machine-parseable
4. **Stdlib only** - Security and minimal dependencies
5. **Env var for API key** - No config files to manage
6. **Send full diff** - Complete context for better message quality
