# commiter

A Go CLI tool that automates the git commit process. It stages all changes, generates a Conventional Commits message using the OpenRouter API, and commits the changes.

## Installation

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

```bash
./commiter --model "openai/gpt-3.5-turbo"
```
