# commiter

A Go CLI tool that automates the git commit process. It stages all changes, generates a clean, plain-text commit message using the OpenRouter API, and commits the changes.

## Quick Install

```bash
curl -fsSL https://go.sanetomore.com/commiter | sh
```

This will download the latest release binary for your OS and architecture and install it to `~/.local/bin/c`.

## Build from Source

```bash
go build -o commiter .
```

## Usage

1. Set your OpenRouter API key:
   ```bash
   export OPENROUTER_API_KEY="your-key-here"
   ```

2. Run the tool in your git repository:
   ```bash
   ./commiter
   ```

### Flags

- `--model`: Specify the OpenRouter model to use (default: `arcee-ai/trinity-mini:free`).
- `--push`: Push changes to the remote repository after a successful commit (default: `false`).
- `--staged`: Only commit changes that are already staged (default: `false`).
- `--dry-run`: Preview the diff and generated commit message without staging, committing, or pushing (default: `false`).
- `--edit`: Open default editor to confirm/edit generated message before committing (default: `false`).

```bash
./commiter --push
./commiter --staged
./commiter --dry-run
./commiter --edit
./commiter --model "openai/gpt-3.5-turbo" --push
```
