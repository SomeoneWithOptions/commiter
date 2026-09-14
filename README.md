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

1. Store your OpenRouter API key in the config file:

   ```bash
   mkdir -p ~/.config/commiter
   install -m 600 /dev/null ~/.config/commiter/config.json
   ${EDITOR:-vi} ~/.config/commiter/config.json
   ```

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

   Config takes precedence over `OPENROUTER_API_KEY`.

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
