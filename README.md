# Scout

**Scout** is an agentic AI coding assistant CLI tool that runs locally with your LLM of choice. It provides an interactive terminal interface where you can work with AI like a pair programmer, complete with file system access, shell execution, and persistent memory across sessions.

## Features

- 🤖 **Agentic Workflow**: Structured thought process (understand → plan → execute) for every task
- 💾 **Persistent Memory**: Remember important decisions, goals, and lessons learned across sessions
- 📂 **Session Persistence**: Load previous conversations with `/load` command
- 🔧 **File Tools**: Read/write files, insert lines, patch code
- ⚡ **Shell Execution**: Run commands with safety timeouts and output truncation
- 🎯 **Project-Scope**: Sessions scoped to the directory where Scout starts
- 🖥️ **Local-first**: Works with local LLMs (LM Studio, etc.) or cloud APIs

## Installation

1. Clone the repository:
```bash
git clone https://github.com/connormullett/scout
cd scout
```

2. Build:
```bash
go build -o scout .
```

3. Initialize your configuration:
```bash
./scout init
```

This creates `~/.scout/config.json` and `~/.scout/memory/` with starter files.

## Configuration

Edit `~/.scout/config.json`:

```json
{
  "api_key": "your-api-key",
  "model": "gemma4:e4b",
  "max_tokens": 10000,
  "base_url": "http://localhost:11434",
  "show_thinking": true
}
```

| Setting | Default | Description |
|---------|---------|-------------|
| `api_key` | required | Your Anthropic API key or local LLM auth |
| `model` | `gemma4:e4b` | Model identifier |
| `max_tokens` | 10000 | Max response tokens |
| `base_url` | `http://localhost:11434` | API endpoint (local or cloud) |
| `show_thinking` | `true` | Display model's thinking process |

## Usage

```bash
./scout
```

Scout runs in an interactive REPL. Type your request and press Enter to send it.

### Commands

- `exit` — Exit the REPL
- `/load` — Load a previous session from this project directory

### Environment Variables

- `SCOUT_SHELL_TIMEOUT_SECONDS` — Override default 2-minute shell command timeout

## Memory System

Scout maintains persistent memory in `~/.scout/memory/`:

| File | Purpose |
|------|---------|
| `memories.txt` | Important conversations and decisions |
| `goals.txt` | Project objectives and targets |
| `learning.txt` | Knowledge acquired over time |
| `README.md` | Memory documentation |

### Memory Usage

**Before starting a task**: Check relevant memory files for context from previous conversations.

**When completing work**: Write summaries to memory (e.g., "Completed X feature, Y lessons learned").

**When stuck**: Review learning.txt and memories.txt for similar past solutions.

## Tools Available

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents |
| `write_file` | Write content to a file |
| `insert_line_to_file` | Insert a line at a specific line number |
| `patch_file` | Replace all occurrences of a search string |
| `shell_command` | Execute shell commands (with timeout and output limits) |

## Security

- Shell commands run with configurable timeouts (default 2 minutes)
- Command output is truncated to 32KB with head/tail preservation
- Destructive commands (e.g., `rm -rf`) require explicit user confirmation
- Process groups are managed to prevent orphaned background processes

## Session Storage

Sessions are saved to `~/.scout/sessions/<timestamp>-<projectname>-<id>.json`. The `/load` command filters to only show sessions from the current working directory.

## License

MIT License
