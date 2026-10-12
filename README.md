# commiter

A Go CLI tool that automates the git commit process. It stages all changes, generates a clean, plain-text commit message using the OpenRouter API, and commits the changes. Agents can also generate a suggestion, review or extend it, then commit the supplied message without opening an editor.

## Quick Install

```bash
curl -fsSL https://go.sanetomore.com/commiter | sh
```

This downloads the latest release binary to `~/.local/bin/c` and creates a private config containing a placeholder credential and the `openrouter/free` model. Existing config is never overwritten.

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

   Set the OpenRouter model used to generate messages. The installer writes `openrouter/free` into new configs:

   ```json
   {
     "openrouter": {
       "api_key": "your-key-here",
       "model": "openrouter/free"
     }
   }
   ```

   `--model` overrides the config. There is no built-in default: if neither is set, `commiter` fails with an error. Reinstalling or upgrading never touches an existing config, so configs created before this option must add `model` by hand.

2. Run the tool in your git repository:

   ```bash
   ./commiter
   ```

### Help

Run `c --help` or `c -h` for detailed descriptions and examples for every option, including configuration and the agent workflow. Help works outside a Git repository and does not require an API key. The examples below use the installed `c` command; replace it with `./commiter` for a source build.

### Default behavior

`c` stages all changes throughout the repository, including untracked files, generates a Conventional Commits subject with OpenRouter, then commits. It does not push unless `--push` is supplied. If staging, generation, or committing fails, it attempts to restore the original staged changes.

### Agent workflow

Stage the files you intend to commit, generate a suggestion into a file outside the repository, then let the agent read and edit that file:

```bash
git add src/main.go
c --staged --suggest > /tmp/commit-message.txt

# Read/edit /tmp/commit-message.txt, then commit it:
c --staged --message-file /tmp/commit-message.txt
```

The message can include a body after a blank line:

```text
fix(auth): refresh expired tokens before retrying

Preserve request body when retrying authenticated uploads.
Verified with auth integration tests.
```

Keep staged changes unchanged between generation and commit. If they change, review the diff and message again. Keep message files outside the repository so that staging all changes does not accidentally include them.

### Options and examples

All boolean options default to `false`. Flags also accept `--name=value`.

#### `--staged`

Use only changes already staged with `git add`. Unstaged changes and untracked files are left alone. Works with generation, previews, supplied messages, editing, and pushing.

```bash
git add src/main.go
c --staged
c --staged --suggest
```

#### `--suggest`

Generate a suggestion and print **only the message**, followed by a newline, to stdout. No commit, push, editor, or spinner. Errors go to stderr. Generation still requires an OpenRouter API key.

Without `--staged`, all changes are temporarily staged to generate the diff, then the original staging is restored before generation. With `--staged`, no temporary staging is needed.

```bash
c --suggest
c --staged --suggest > /tmp/commit-message.txt
```

Cannot combine with `--dry-run`, `--edit`, `--push`, or `--message-file`.

#### `--message-file <path|->`

Read the final message from a file, or use `-` to read stdin. Skip AI generation and config loading, so no API key is required. Multiline bodies and letter case are preserved, subject to Git's normal message cleanup. Empty, unreadable, or NUL-containing messages fail before staging changes. No editor or spinner opens.

By default, stage and commit all changes. Add `--staged` to commit only staged changes, `--dry-run` to preview the diff and supplied message, or `--push` to push after committing. Cannot combine with `--edit`; edit the file first.

```bash
c --staged --message-file /tmp/commit-message.txt
c --staged --message-file - < /tmp/commit-message.txt
c --message-file /tmp/commit-message.txt --push
```

#### `--dry-run`

Print the diff and message without committing or pushing. Without `--staged`, temporarily stage all changes for the preview, then restore the original staging. With `--message-file`, preview the supplied message without an API call. `--edit` and `--push` are ignored during a dry run.

This output includes headings and the diff; use `--suggest` when you need only the generated message.

```bash
c --dry-run
c --staged --dry-run
c --staged --dry-run --message-file /tmp/commit-message.txt
```

#### `--edit`

Open Git's configured editor to review or change the generated message before committing. Git selects the editor using `GIT_EDITOR`, `core.editor`, `VISUAL`, or `EDITOR`. If editing or committing fails, the tool attempts to restore original staging when it staged changes itself. Agents should use `--suggest` followed by `--message-file`.

```bash
c --edit
c --staged --edit --push
```

#### `--push`

Run `git push` after a successful commit, using Git's configured remote and upstream. If pushing fails, the local commit remains; retry with `git push` after addressing the failure.

```bash
c --push
c --staged --message-file /tmp/commit-message.txt --push
```

#### `--model <id>`

Choose the OpenRouter model for generation. Overrides `openrouter.model` from the config. Fails if neither is set. Ignored when using `--message-file`.

```bash
c --model "openrouter/free" --staged --suggest
```

#### `--config <path>`

Choose a different JSON config file for the API key and model. By default, use `commiter/config.json` inside the platform user config directory: on Linux, `$XDG_CONFIG_HOME/commiter/config.json` when set, otherwise `~/.config/commiter/config.json`.

Config credentials take precedence over `OPENROUTER_API_KEY`; empty or placeholder credentials fall back to the environment. An explicitly supplied missing or invalid config fails. Unix config files must have private permissions (`chmod 600`). Literal keys and `op://` 1Password references are supported, as described above. Ignored when using `--message-file`.

```bash
c --config ~/.config/commiter/work.json --staged --suggest
```

#### `--help`, `-h`

Print the full help page with examples and exit without accessing Git or OpenRouter.

```bash
c --help
c -h
```

### Exit status and output

Exit `0` means success or help; exit `1` means invalid options, no changes to commit, or an operation failed. Errors and progress go to stderr. `--suggest` writes only its message to stdout; regular commits print the message and a success notice.
