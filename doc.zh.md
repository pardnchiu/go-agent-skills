# go-agent-skills - 技術文件

> 返回 [README](./README.zh.md)

## 前置需求

- Go 1.25.1 或更高版本
- 至少一組 AI Agent 憑證（GitHub Copilot 訂閱或以下任一 API Key）：
  - `OPENAI_API_KEY`
  - `ANTHROPIC_API_KEY`
  - `GEMINI_API_KEY`
  - `NVIDIA_API_KEY`

## 安裝

### 從原始碼建置

```bash
git clone https://github.com/pardnchiu/go-agent-skills.git
cd go-agent-skills
go build -o agent-skills cmd/cli/main.go
```

### 使用 go install

```bash
go install github.com/pardnchiu/go-agent-skills/cmd/cli@latest
```

## 設定

### 環境變數

複製 `.env.example` 並填入對應的 API Key：

```bash
cp .env.example .env
```

| 變數 | 必要 | 說明 |
|------|------|------|
| `OPENAI_API_KEY` | 否 | OpenAI API 金鑰 |
| `ANTHROPIC_API_KEY` | 否 | Anthropic Claude API 金鑰 |
| `GEMINI_API_KEY` | 否 | Google Gemini API 金鑰 |
| `NVIDIA_API_KEY` | 否 | Nvidia API 金鑰 |

**注意：** GitHub Copilot 使用 Device Code 登入流程，不需要環境變數。

### Agent 模型設定（config.json）

在 `~/.config/go-agent-skills/config.json` 的 `models` 陣列中定義可用的 Agent 與 selectorBot 路由優先順序：

```json
{
  "models": [
    { "name": "claude@claude-sonnet-4-5", "description": "適合高品質生成與 Skill 執行" },
    { "name": "nvidia@openai/gpt-oss-120b", "description": "適合快速摘要與條列輸出" }
  ]
}
```

每個條目的 `name` 欄位採用 `provider@model` 格式；selectorBot 依據 `description` 判斷任務適配度，自動選擇最合適的 Agent。

### Skill 掃描路徑

系統會自動掃描以下路徑中的 `SKILL.md` 檔案：

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

每個 Skill 需遵循以下結構：

```
~/.claude/skills/
└── {skill_name}/
    ├── SKILL.md              # Skill 定義檔（必要）
    ├── scripts/              # 可執行腳本（選填）
    ├── templates/            # 範本檔案（選填）
    └── assets/               # 靜態資源（選填）
```

## 使用方式

### 列出所有已安裝的 Skill

```bash
./agent-skills list
```

輸出範例：

```
Found 3 skill(s):

• commit-generate
  從 git diff 與專案脈絡生成語義化提交訊息
  Path: /Users/user/.claude/skills/commit-generate

• readme-generate
  從原始碼分析自動生成雙語 README
  Path: /Users/user/.claude/skills/readme-generate
```

### 執行指定的 Skill

```bash
# 互動模式（每次工具呼叫前會要求確認）
./agent-skills run commit-generate "generate commit message"

# 自動模式（跳過所有確認）
./agent-skills run readme-generate "generate readme" --allow
```

### 自動匹配 Skill

當不確定使用哪個 Skill 時，直接描述需求：

```bash
./agent-skills run "幫我生成一份 README 文件"
```

系統會透過 LLM 自動從已安裝的 Skill 中選擇最符合的項目執行。

### 直接使用工具（無 Skill）

若輸入不符合任何已安裝的 Skill，系統會退回至直接工具執行模式：

```bash
./agent-skills run "讀取 package.json 並列出所有相依套件"
```

## 命令列參考

### 指令

| 指令 | 語法 | 說明 |
|------|------|------|
| `list` | `./agent-skills list` | 列出所有已安裝的 Skill |
| `run` | `./agent-skills run <skill_name> <input> [--allow]` | 執行指定的 Skill |
| `run` | `./agent-skills run <input> [--allow]` | 自動匹配 Skill 或使用純工具模式 |

### 旗標

| 旗標 | 說明 |
|------|------|
| `--allow` | 跳過所有互動式確認提示，自動執行所有工具呼叫 |

### 支援的 Agent

執行 `run` 指令時，系統透過 selectorBot（`nvidia@openai/gpt-oss-20b`）自動從 Agent Registry 選擇最適合當前任務的 AI 後端，無需手動選擇。若要手動指定，可在輸入中使用 `use <agent>` 語法（例如：`use claude 幫我重構這段程式碼`）。

| Agent | 認證方式 | 預設模型 | 環境變數 |
|-------|----------|----------|----------|
| GitHub Copilot | Device Code 登入 | `gpt-4.1` | 無（透過 OAuth） |
| OpenAI | API Key | `gpt-5-mini` | `OPENAI_API_KEY` |
| Claude | API Key | `claude-sonnet-4-5` | `ANTHROPIC_API_KEY` |
| Gemini | API Key | `gemini-2.5-pro` | `GEMINI_API_KEY` |
| Nvidia | API Key | `openai/gpt-oss-120b` | `NVIDIA_API_KEY` |

> **注意：** Anthropic API Tier 1 的單次請求 input token 上限僅 30,000，不足以支撐大多數 Skill 執行。**建議使用 Tier 2 以上**以確保穩定運作。詳見 [Anthropic rate limits](https://docs.anthropic.com/en/api/rate-limits)。

**GitHub Copilot 認證流程：**

首次使用 Copilot Agent 時，系統會自動啟動 Device Code 登入流程：

1. 終端機顯示認證 URL 與 User Code
2. 在瀏覽器開啟 URL 並輸入 User Code
3. 完成 GitHub 授權
4. Token 自動儲存至 `~/.config/go-agent-skills/`

Token 會在過期前自動更新，無需手動管理。

### 內建工具

所有 Agent 共享以下工具集合：

| 工具 | 參數 | 說明 |
|------|------|------|
| `read_file` | `path` | 讀取指定路徑的檔案內容 |
| `list_files` | `path`, `recursive?` | 列出目錄內容（`recursive` 為選填布林值） |
| `glob_files` | `pattern` | 使用 glob 模式搜尋檔案（例如 `**/*.go`） |
| `write_file` | `path`, `content` | 寫入或建立檔案（會覆蓋現有內容） |
| `search_content` | `pattern`, `file_pattern?` | 使用正規表達式搜尋檔案內容；`file_pattern` 為選填 glob 篩選 |
| `patch_edit` | `path`, `old_string`, `new_string` | 將 `old_string` 的第一個精確匹配替換為 `new_string` |
| `run_command` | `command` | 執行白名單內的 shell 指令 |
| `fetch_yahoo_finance` | `symbol`, `interval?`, `range?` | 股票報價與 K 線資料；interval: `1m`–`1wk`，range: `1d`–`max` |
| `fetch_google_rss` | `keyword`, `time`, `lang?` | Google News RSS 搜尋；time: `1h`/`3h`/`6h`/`12h`/`24h`/`7d` |
| `send_http_request` | `url`, `method?`, `headers?`, `body?`, `content_type?`, `timeout?` | 通用 HTTP 請求；method 預設 `GET`，timeout 最大 300 秒 |
| `fetch_weather` | `city?`, `days?`, `hourly_interval?` | 透過 wttr.in 取得天氣預報；`days=-1` 僅回當前狀況，預設三天預報 |
| `fetch_page` | `url` | 以 Chrome 無頭瀏覽器開啟網址，等待 JS 完整渲染後以 Markdown 格式返回頁面內容 |
| `search_web` | `query`, `range?`, `limit?` | 透過 DuckDuckGo 搜尋網路；`range`: `1h`/`3h`/`6h`/`12h`/`1d`/`7d`/`1m`/`1y`；`limit` 最大 50 |
| `calculate` | `expression` | 計算數學表達式；支援 `+`、`-`、`*`、`/`、`%`、`^`（冪次）、`()` 及函式：`sqrt`、`abs`、`pow(base,exp)`、`ceil`、`floor`、`round`、`log`、`log2`、`log10`、`sin`、`cos`、`tan` |
| `search_history` | `session_id`, `keyword`, `limit?` | 在對話歷史中搜尋關鍵字，回傳相關歷史片段，排除最新 4 筆 |

#### run_command 安全機制

**白名單指令：**
```
git, go, node, npm, npx, yarn, pnpm, python3, pip3,
make, docker, kubectl, curl, jq, cat, grep, find, sed,
awk, ls, pwd, echo, date, wc, head, tail, sort, uniq
```

**rm 指令攔截：**

當 LLM 嘗試執行 `rm` 時，系統會自動攔截並改為移動至專案根目錄的 `.Trash/` 資料夾：

```bash
# LLM 執行：rm old_file.txt
# 實際行為：mv old_file.txt .Trash/old_file_20260207_143052.txt
```

若 `.Trash/` 中已存在同名檔案，會自動附加時間戳記避免覆蓋。

**危險指令阻擋：**

以下指令不在白名單內，執行時會直接拒絕：
- `sudo`、`su`
- `chmod`、`chown`
- `dd`、`mkfs`
- 任何非白名單的二進位檔案

#### 動態 API 工具擴展（api.json）

在專案目錄的 `.config/apis/` 或全域路徑 `~/.config/go-agent-skills/apis/` 下放置 JSON 檔案，每個檔案定義一個自訂 API 工具，啟動時自動以 `api_` 前綴載入。

**檔案格式**（參考 `internal/tools/apiAdapter/example.json`）：

```json
{
  "name": "tool_name",
  "description": "此工具的功能描述",
  "endpoint": {
    "url": "https://your.api/{id}/resource",
    "method": "GET",
    "content_type": "json",
    "headers": { "X-Custom-Header": "value" },
    "query": { "static_param": "value" },
    "timeout": 15
  },
  "auth": {
    "type": "bearer",
    "header": "Authorization",
    "env": "YOUR_API_KEY_ENV"
  },
  "parameters": {
    "id": {
      "type": "string",
      "description": "資源 ID，對應 URL 中的 {id}",
      "required": true,
      "default": ""
    }
  },
  "response": {
    "format": "json"
  }
}
```

> **Gemini 相容性注意：** Gemini 不支援工具定義中參數的 `"type": "integer"`。數字型 enum 欄位應改用 `"type": "string"` 搭配字串 enum（例如 `["1", "2", "6"]`），執行器內部會自動處理字串轉整數。

## API 參考

### Agent Interface

所有 Agent 實作必須實作以下介面：

```go
type Agent interface {
    Send(ctx context.Context, messages []Message, toolDefs []tools.Tool) (*OpenAIOutput, error)
    Execute(ctx context.Context, skill *skill.Skill, userInput string, events chan<- atypes.Event, allowAll bool) error
}
```

#### Send

```go
Send(ctx context.Context, messages []Message, toolDefs []tools.Tool) (*OpenAIOutput, error)
```

處理單次 LLM API 呼叫，傳入對話歷史與工具定義，回傳包含文字或工具呼叫的回應。

**參數：**
- `ctx`：執行上下文，用於取消或逾時控制
- `messages`：對話歷史（system、user、assistant、tool 角色）
- `toolDefs`：可用的工具定義陣列

**回傳：**
- `*OpenAIOutput`：LLM 回應，包含文字內容或工具呼叫請求
- `error`：API 錯誤或網路錯誤

#### Execute

```go
Execute(ctx context.Context, skill *skill.Skill, userInput string, events chan<- atypes.Event, allowAll bool) error
```

管理完整的 Skill 執行迴圈，處理最多 32 次工具呼叫迭代。當 `skill` 為 `nil` 時，使用基礎系統提示詞直接執行工具。

**參數：**
- `ctx`：執行上下文
- `skill`：要執行的 Skill（`nil` 代表直接工具模式）
- `userInput`：使用者輸入的任務描述
- `events`：事件 Channel，用於接收工具呼叫請求、確認提示與執行結果事件
- `allowAll`：是否跳過互動式確認

**回傳：**
- `error`：執行過程中的錯誤

### NewScanner

```go
func NewScanner() *Scanner
```

建立 Skill 掃描器並立即掃描所有設定路徑。掃描過程為並行執行，使用 goroutine 同時處理多個路徑。

**回傳：**
- `*Scanner`：包含已掃描 Skill 的掃描器實例

**使用範例：**

```go
scanner := skill.NewScanner()

// 列出所有 Skill 名稱
names := scanner.List()

// 根據名稱取得 Skill
if s, ok := scanner.Skills.ByName["commit-generate"]; ok {
    fmt.Println(s.Description)
}
```

### NewExecutor

```go
func NewExecutor(workPath string) (*Executor, error)
```

建立工具執行器，載入工具定義並設定工作目錄。

**參數：**
- `workPath`：工具執行的工作目錄路徑

**回傳：**
- `*Executor`：已初始化的執行器
- `error`：初始化錯誤（例如工具定義檔案讀取失敗）

### Execute (Executor)

```go
func (e *Executor) Execute(name string, args json.RawMessage) (string, error)
```

執行指定的工具並回傳結果。所有錯誤會轉換為字串回傳，確保 LLM 能理解錯誤訊息。

**參數：**
- `name`：工具名稱（例如 `read_file`）
- `args`：JSON 格式的工具參數

**回傳：**
- `string`：工具執行結果或錯誤訊息
- `error`：僅在工具不存在時回傳

**使用範例：**

```go
exec, _ := tools.NewExecutor("/path/to/project")

// 讀取檔案
result, _ := exec.Execute("read_file", json.RawMessage(`{"path": "README.md"}`))

// 執行指令
result, _ := exec.Execute("run_command", json.RawMessage(`{"command": "git status"}`))
```

## 進階用法

### 自訂 Skill 開發

建立自訂 Skill 的基本步驟：

1. 在任一掃描路徑中建立資料夾（例如 `~/.claude/skills/my-skill/`）
2. 建立 `SKILL.md` 檔案，包含以下 metadata：

```markdown
---
name: my-skill
description: 簡短描述此 Skill 的功能
---

# My Skill

[詳細的 Skill 指引與範例]
```

3. （選填）建立輔助資源：
   - `scripts/` — 可執行腳本（Python、Shell 等）
   - `templates/` — 範本檔案
   - `assets/` — 靜態資源（圖片、設定檔等）

4. 重新執行 `./agent-skills list` 驗證 Skill 已被掃描

**路徑解析規則：**

在 `SKILL.md` 中引用 `scripts/`、`templates/`、`assets/` 時，系統會自動解析為絕對路徑：

```markdown
執行以下指令：
python3 scripts/analyze.py
```

實際執行時會替換為：
```
python3 /Users/user/.claude/skills/my-skill/scripts/analyze.py
```

### 程式化使用

```go
package main

import (
    "context"

    atypes "github.com/pardnchiu/go-agent-skills/internal/agents/types"
    "github.com/pardnchiu/go-agent-skills/internal/agents/openai"
    "github.com/pardnchiu/go-agent-skills/internal/skill"
)

func main() {
    // 初始化 Agent
    agent, _ := openai.New()

    // 掃描 Skill
    scanner := skill.NewScanner()
    targetSkill := scanner.Skills.ByName["commit-generate"]

    // 建立事件 Channel 並消費事件
    events := make(chan atypes.Event, 64)
    go func() {
        for event := range events {
            // 處理工具呼叫請求、確認提示與執行結果
            _ = event
        }
    }()

    // 執行 Skill
    ctx := context.Background()
    agent.Execute(ctx, targetSkill, "generate commit message", events, false)
}
```

### Tool Call 迭代限制

系統預設最多執行 32 次工具呼叫迭代，避免 LLM 進入無限迴圈。若需調整此限制：

```go
import "github.com/pardnchiu/go-agent-skills/internal/agents"

func init() {
    agents.MaxToolIterations = 8  // 調整至 64 次
}
```

***

©️ 2026 [邱敬幃 Pardn Chiu](https://linkedin.com/in/pardnchiu)
