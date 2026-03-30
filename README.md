# antigravity-decryptor

读取 Antigravity 本地会话文件（`~/.gemini/antigravity/conversations/*.pb`），通过复用官方 Language Server 本地 HTTP 接口，将对话轨迹导出为 JSON 或 Markdown。

## 快速开始

```bash
# 编译
go build -o bin/antigravity-decryptor ./cmd/antigravity-decryptor/

# 导出对话（raw JSON）
./bin/antigravity-decryptor <cascadeId>

# 导出为 Markdown
./bin/antigravity-decryptor --format markdown <cascadeId>

# 列出所有对话
./bin/antigravity-decryptor list
```

## 主要功能

| 功能 | 说明 |
|------|------|
| `<cascadeId>` | 按对话 ID 导出轨迹 |
| `<path>.pb` | 按 `.pb` 文件路径导出 |
| `list` | 列出所有可见对话摘要 |
| `--format` | 输出格式：`raw`（默认）/ `normalized` / `markdown` |
| `--output` | 写入文件（默认 stdout） |
| `--ls-binary` | 自定义 Language Server 路径（或设置 `ANTIGRAVITY_LS_PATH`） |

## 环境要求

- macOS arm64
- Antigravity 已安装（`/Applications/Antigravity.app`）
- 目标 `.pb` 对话文件存在

## 文档

技术方案与实现细节详见 [docs/](docs/) 目录。
