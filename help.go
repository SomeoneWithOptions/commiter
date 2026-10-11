package main

import (
	"fmt"
	"io"
)

func printHelp(w io.Writer) error {
	_, err := fmt.Fprintf(w, `commiter — AI commit messages, with a reviewable workflow

USAGE
  c [options]
  Use ./commiter instead of c when running a source build.

DEFAULT BEHAVIOR
  Stage all changes across the repository (including untracked files), generate
  a Conventional Commits subject with OpenRouter, then commit. No push by default.
  Example: c

OPTIONS AND EXAMPLES
  --staged
      Use only changes already staged with git add. Leave unstaged changes alone.
      Example: git add src/main.go
               c --staged

  --suggest
      Generate a message and write ONLY that message to stdout, followed by a
      newline. No commit, push, editor, or spinner. Existing staging is restored
      after previewing all changes; --staged avoids temporary staging entirely.
      Errors go to stderr; generation still requires an OpenRouter API key.
      Example: c --suggest
               c --staged --suggest > /tmp/commit-message.txt

  --message-file <path|->
      Commit a message from a file, or read stdin when the path is -.
      Skip AI generation and config loading: no API key required. Multiline
      subjects/bodies and letter case are preserved, subject to Git's normal
      message cleanup. Empty messages are rejected before staging changes.
      Stage all changes unless --staged is supplied. No editor opens.
      Example: c --staged --message-file /tmp/commit-message.txt
               c --staged --message-file - < /tmp/commit-message.txt
               c --message-file /tmp/commit-message.txt --push

  --dry-run
      Print the diff and message without committing or pushing. Temporarily
      stage all changes for the preview, then restore existing staging.
      Use --staged to preview only staged changes. With --message-file, preview
      the supplied message without calling OpenRouter. --push/--edit are ignored.
      Example: c --dry-run
               c --staged --dry-run
               c --staged --dry-run --message-file /tmp/commit-message.txt

  --edit
      Open Git's configured editor to review/change the generated message before
      committing. Git selects the editor using GIT_EDITOR, core.editor, VISUAL,
      or EDITOR. If editing/committing fails, restore original staging when this
      tool staged changes. For agents, use --suggest then --message-file.
      Example: c --edit
               c --staged --edit --push

  --push
      Run git push only after a successful commit, using Git's configured remote
      and upstream. If push fails, the local commit remains; retry with git push.
      Example: c --push
               c --staged --message-file /tmp/commit-message.txt --push

  --model <id>
      Select an OpenRouter model for message generation.
      Default: %s
      Ignored when using --message-file.
      Example: c --model "%s" --staged --suggest

  --config <path>
      Read OpenRouter credentials from a JSON config file. Default: commiter's
      config.json inside the platform user config directory; on Linux this is
      $XDG_CONFIG_HOME/commiter/config.json or ~/.config/commiter/config.json.
      Config credentials take precedence over OPENROUTER_API_KEY. An empty or
      placeholder key falls back to the environment. An explicitly supplied
      missing/invalid config fails. Ignored when using --message-file.
      Example: c --config ~/.config/commiter/work.json --staged --suggest

  --help, -h
      Show this help, including examples. No repository or API key required.
      Example: c --help

AGENT WORKFLOW
  1. Stage the intended files: git add src/main.go
  2. Generate: c --staged --suggest > /tmp/commit-message.txt
  3. Read/edit that file; append details after a blank line, for example:

       fix(auth): refresh expired tokens before retrying

       Preserve request body when retrying authenticated uploads.
       Verified with auth integration tests.

  4. Commit: c --staged --message-file /tmp/commit-message.txt
     Keep staged changes unchanged between generation and commit. If they
     change, review the diff/message again. Keep message files outside the repo
     to avoid accidentally including them when staging all changes.

CONFIGURATION
  Config example: {"openrouter":{"api_key":"your-key-here"}}
  Replace the placeholder with your key, or export OPENROUTER_API_KEY.
  Unix config files must have private permissions: chmod 600 <config-path>.
  1Password references are supported: {"openrouter":{"api_key":"op://Private/OpenRouter/personal"}}
  References require the installed, authenticated 1Password CLI (op).

COMBINATIONS AND EXIT STATUS
  --suggest cannot combine with --dry-run, --edit, --push, or --message-file.
  --message-file cannot combine with --edit; edit the supplied file first.
  Flags may use --name=value. Boolean flags default to false.
  Exit 0: success/help. Exit 1: invalid options, no changes, or an operation failed.
`, defaultModel, defaultModel)
	return err
}
