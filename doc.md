# go-agent-skills - Documentation

> Back to [README](./README.md)

## Prerequisites

- Go 1.25 or higher
- At least one AI Agent credential (GitHub Copilot subscription or any of the following API keys):
  - `OPENAI_API_KEY`
  - `ANTHROPIC_API_KEY`
  - `GEMINI_API_KEY`
  - `NVIDIA_API_KEY`

## Installation

### From Source

```bash
git clone https://github.com/pardnchiu/go-agent-skills.git
cd go-agent-skills
go build -o agent-skills cmd/cli/main.go
```

### Using go install

```bash
go install github.com/pardnchiu/go-agent-skills/cmd/cli@latest
```

## Configuration

### Environment Variables

Copy `.env.example` and fill in the corresponding API keys:

```bash
cp .env.example .env
```

| Variable | Required | Description |
|----------|----------|-------------|
| `OPENAI_API_KEY` | No | OpenAI API key |
| `ANTHROPIC_API_KEY` | No | Anthropic Claude API key |
| `GEMINI_API_KEY` | No | Google Gemini API key |
| `NVIDIA_API_KEY` | No | Nvidia API key |

**Note:** GitHub Copilot uses Device Code authentication flow and does not require environment variables.

### Skill Scan Paths

The system automatically scans for `SKILL.md` files in the following paths:

```
{project}/.claude/skills/
{project}/.skills/
~/.claude/skills/
~/.opencode/skills/
~/.openai/skills/
~/.codex/skills/
/mnt/skills/public/
/mnt/skills/user/
/mnt/skills/examples/
```

Each Skill must follow this structure:

```
~/.claude/skills/
└── {skill_name}/
    ├── SKILL.md              # Skill definition file (required)
    ├── scripts/              # Executable scripts (optional)
    ├── templates/            # Template files (optional)
    └── assets/               # Static assets (optional)
```

## Usage

### List All Installed Skills

```bash
./agent-skills list
```

Example output:

```
Found 3 skill(s):

• commit-generate
  Generate semantic commit messages from git diff and project context
  Path: /Users/user/.claude/skills/commit-generate

• readme-generate
  Generate bilingual README from source code analysis
  Path: /Users/user/.claude/skills/readme-generate
```

### Execute Specific Skill

```bash
# Interactive mode (confirmation required before each tool call)
./agent-skills run commit-generate "generate commit message"

# Auto mode (skip all confirmations)
./agent-skills run readme-generate "generate readme" --allow
```

### Auto-Match Skill

When unsure which Skill to use, simply describe your need:

```bash
./agent-skills run "help me generate a README document"
```

The system uses LLM to automatically select the most relevant installed Skill.

### Direct Tool Usage (No Skill)

If the input doesn't match any installed Skill, the system falls back to direct tool execution mode:

```bash
./agent-skills run "read package.json and list all dependencies"
```

## CLI Reference

### Commands

| Command | Syntax | Description |
|---------|--------|-------------|
| `list` | `./agent-skills list` | List all installed Skills |
| `run` | `./agent-skills run <skill_name> <input> [--allow]` | Execute specified Skill |
| `run` | `./agent-skills run <input> [--allow]` | Auto-match Skill or use tool execution |

### Flags

| Flag | Description |
|------|-------------|
| `--allow` | Skip all interactive confirmation prompts and auto-execute all tool calls |

### Supported Agents

When executing `run` command, the system prompts for Agent selection:

| Agent | Authentication | Default Model | Environment Variable |
|-------|----------------|---------------|----------------------|
| GitHub Copilot | Device Code login | `gpt-4.1` | None (via OAuth) |
| OpenAI | API Key | `gpt-5-nano` | `OPENAI_API_KEY` |
| Claude | API Key | `claude-sonnet-4-5` | `ANTHROPIC_API_KEY` |
| Gemini | API Key | `gemini-2.0-flash-exp` | `GEMINI_API_KEY` |
| Nvidia | API Key | `meta/llama-3.3-70b-instruct` | `NVIDIA_API_KEY` |

> **Note:** Anthropic API Tier 1 limits input tokens to 30,000 per request, which is insufficient for most Skill executions. **Tier 2 or above is recommended** for reliable operation. See [Anthropic rate limits](https://docs.anthropic.com/en/api/rate-limits) for tier details.

**GitHub Copilot Authentication Flow:**

When using Copilot Agent for the first time, the system automatically initiates Device Code authentication:

1. Terminal displays authentication URL and User Code
2. Open URL in browser and enter User Code
3. Complete GitHub authorization
4. Token automatically saved to `~/.config/github-copilot/`

Tokens are automatically refreshed before expiration without manual management.

### Built-in Tools

All Agents share the following tool collection:

| Tool | Parameters | Description |
|------|------------|-------------|
| `read_file` | `path` | Read file content at specified path |
| `write_file` | `path`, `content` | Write or create file (overwrites existing content) |
| `list_files` | `path`, `recursive` | List directory contents (`recursive` is optional boolean) |
| `glob_files` | `pattern` | Search files using glob pattern (e.g., `**/*.go`) |
| `search_content` | `pattern`, `file_pattern` | Search file content using regex |
| `run_command` | `command` | Execute whitelisted shell commands |

#### run_command Safety Mechanisms

**Whitelisted Commands:**
```
git, go, node, npm, npx, yarn, pnpm, python3, pip3,
make, docker, kubectl, curl, jq, cat, grep, find, sed,
awk, ls, pwd, echo, date, wc, head, tail, sort, uniq
```

**rm Command Interception:**

When LLM attempts to execute `rm`, the system intercepts and moves files to `.Trash/` folder in project root:

```bash
# LLM executes: rm old_file.txt
# Actual behavior: mv old_file.txt .Trash/old_file_20260207_143052.txt
```

If a file with the same name exists in `.Trash/`, a timestamp is automatically appended to avoid overwriting.

**Dangerous Command Blocking:**

The following commands are not whitelisted and will be rejected:
- `sudo`, `su`
- `chmod`, `chown`
- `dd`, `mkfs`
- Any non-whitelisted binaries

## API Reference

### Agent Interface

All Agent implementations must implement the following interface:

```go
type Agent interface {
    Send(ctx context.Context, messages []Message, toolDefs []tools.Tool) (*OpenAIOutput, error)
    Execute(ctx context.Context, skill *skill.Skill, userInput string, output io.Writer, allowAll bool) error
}
```

#### Send

```go
Send(ctx context.Context, messages []Message, toolDefs []tools.Tool) (*OpenAIOutput, error)
```

Handles a single LLM API call, passing conversation history and tool definitions, returning responses containing text or tool calls.

**Parameters:**
- `ctx`: Execution context for cancellation or timeout control
- `messages`: Conversation history (system, user, assistant, tool roles)
- `toolDefs`: Array of available tool definitions

**Returns:**
- `*OpenAIOutput`: LLM response containing text content or tool call requests
- `error`: API errors or network errors

#### Execute

```go
Execute(ctx context.Context, skill *skill.Skill, userInput string, output io.Writer, allowAll bool) error
```

Manages complete Skill execution loop, handling up to 32 tool call iterations. When `skill` is `nil`, uses base system prompt for direct tool execution.

**Parameters:**
- `ctx`: Execution context
- `skill`: Skill to execute (`nil` for direct tool mode)
- `userInput`: User's task description
- `output`: Output stream (typically `os.Stdout`)
- `allowAll`: Whether to skip interactive confirmations

**Returns:**
- `error`: Errors during execution

### NewScanner

```go
func NewScanner() *Scanner
```

Creates a Skill scanner and immediately scans all configured paths. Scanning is concurrent, using goroutines to process multiple paths simultaneously.

**Returns:**
- `*Scanner`: Scanner instance containing scanned Skills

**Example Usage:**

```go
scanner := skill.NewScanner()

// List all Skill names
names := scanner.List()

// Get Skill by name
if s, ok := scanner.Skills.ByName["commit-generate"]; ok {
    fmt.Println(s.Description)
}
```

### NewExecutor

```go
func NewExecutor(workPath string) (*Executor, error)
```

Creates tool executor, loads tool definitions and sets working directory.

**Parameters:**
- `workPath`: Working directory path for tool execution

**Returns:**
- `*Executor`: Initialized executor
- `error`: Initialization errors (e.g., tool definition file read failure)

### Execute (Executor)

```go
func (e *Executor) Execute(name string, args json.RawMessage) (string, error)
```

Executes specified tool and returns result. All errors are converted to strings to ensure LLM can understand error messages.

**Parameters:**
- `name`: Tool name (e.g., `read_file`)
- `args`: Tool parameters in JSON format

**Returns:**
- `string`: Tool execution result or error message
- `error`: Only returned when tool doesn't exist

**Example Usage:**

```go
exec, _ := tools.NewExecutor("/path/to/project")

// Read file
result, _ := exec.Execute("read_file", json.RawMessage(`{"path": "README.md"}`))

// Execute command
result, _ := exec.Execute("run_command", json.RawMessage(`{"command": "git status"}`))
```

## Advanced Usage

### Custom Skill Development

Basic steps to create custom Skills:

1. Create folder in any scan path (e.g., `~/.claude/skills/my-skill/`)
2. Create `SKILL.md` file with metadata:

```markdown
---
name: my-skill
description: Brief description of this Skill's functionality
---

# My Skill

[Detailed Skill guidelines and examples]
```

3. (Optional) Create auxiliary resources:
   - `scripts/` — Executable scripts (Python, Shell, etc.)
   - `templates/` — Template files
   - `assets/` — Static assets (images, config files, etc.)

4. Re-run `./agent-skills list` to verify Skill is scanned

**Path Resolution Rules:**

When referencing `scripts/`, `templates/`, `assets/` in `SKILL.md`, the system automatically resolves to absolute paths:

```markdown
Execute the following command:
python3 scripts/analyze.py
```

At runtime, this is replaced with:
```
python3 /Users/user/.claude/skills/my-skill/scripts/analyze.py
```

### Programmatic Usage

```go
package main

import (
    "context"
    "os"

    "github.com/pardnchiu/go-agent-skills/internal/agents/openai"
    "github.com/pardnchiu/go-agent-skills/internal/skill"
)

func main() {
    // Initialize Agent
    agent, _ := openai.New()

    // Scan Skills
    scanner := skill.NewScanner()
    targetSkill := scanner.Skills.ByName["commit-generate"]

    // Execute Skill
    ctx := context.Background()
    agent.Execute(ctx, targetSkill, "generate commit message", os.Stdout, false)
}
```

### Tool Call Iteration Limit

The system defaults to a maximum of 32 tool call iterations to prevent LLM infinite loops. To adjust this limit:

```go
import "github.com/pardnchiu/go-agent-skills/internal/agents"

func init() {
    agents.MaxToolIterations = 64  // Adjust to 64 iterations
}
```

***

©️ 2026 [邱敬幃 Pardn Chiu](https://linkedin.com/in/pardnchiu)
