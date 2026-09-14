# commiter

A Go CLI tool that automates the git commit process. It stages all changes, generates a clean, plain-text commit message using the OpenRouter API, and commits the changes.

## Quick Install

```bash
curl -fsSL https://go.sanetomore.com/commiter | sh
```

This downloads the latest release binary to `~/.local/bin/c` and creates a private config containing a placeholder credential. Existing config is never overwritten.

## Build from Source

```bash
go build -o commiter .
```

## Usage

1. Configure your OpenRouter API key in the file created by the installer:

   ```bash
   ${EDITOR:-vi} ~/.config/commiter/config.json
   ```

   For a Linux source install, create it first with:

   ```bash
   mkdir -p ~/.config/commiter
   touch ~/.config/commiter/config.json
   chmod 600 ~/.config/commiter/config.json
   ```

   To read the key directly from 1Password, install and sign in to the
   [1Password CLI](https://developer.1password.com/docs/cli/), then use a secret reference:

   ```json
   {
     "openrouter": {
       "api_key": "op://Private/OpenRouter/personal"
     }
   }
   ```

   `commiter` invokes `op read` directly without a shell, so shell commands and
   substitutions in the config are never executed. The referenced field must
   contain only the OpenRouter API key.

   A literal API key is also supported:

   ```json
   {
     "openrouter": {
       "api_key": "your-key-here"
     }
   }
   ```

   On Linux, the default config directory follows `$XDG_CONFIG_HOME` when set. On macOS and Windows, the platform user config directory is used. Config files with group or other permissions are rejected on Unix.

   Alternatively, keep using the environment variable when no key is configured:

   ```bash
   export OPENROUTER_API_KEY="your-key-here"
   ```

   Config takes precedence over `OPENROUTER_API_KEY`. The installer placeholder `your-key-here` is treated as unset, allowing environment fallback.

2. Run the tool in your git repository:

   ```bash
   ./commiter
   ```

### Flags

- `--model`: Specify the OpenRouter model to use (default: `google/gemini-2.5-flash-lite`).
- `--config`: Use a different JSON config file.
- `--push`: Push changes to the remote repository after a successful commit (default: `false`).
- `--staged`: Only commit changes that are already staged (default: `false`).
- `--dry-run`: Preview the diff and generated commit message without staging, committing, or pushing (default: `false`).
- `--edit`: Open default editor to confirm/edit generated message before committing (default: `false`).

```bash
./commiter --push
./commiter --staged
./commiter --dry-run
./commiter --edit
./commiter --config ~/.config/commiter/work.json
./commiter --model "openai/gpt-3.5-turbo" --push
```
