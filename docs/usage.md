# 使用指南

## 编译

```bash
go build -o bin/antigravity-decryptor ./cmd/antigravity-decryptor/
```

## 基本用法

```
antigravity-decryptor [flags] <cascadeId | path/to/<cascadeId>.pb>
antigravity-decryptor list [flags]
```

## 导出对话

### 通过 cascadeId 导出

cascadeId 即对话 UUID，可通过 `list` 子命令获取，也可在 `~/.gemini/antigravity/conversations/` 目录下查看 `.pb` 文件名。

```bash
./bin/antigravity-decryptor 762506a2-5119-41e2-b4d9-98c944135b68
```

### 通过 .pb 文件路径导出

直接传入 `.pb` 文件的完整路径，工具会自动提取文件名作为 cascadeId。

```bash
./bin/antigravity-decryptor ~/.gemini/antigravity/conversations/762506a2-5119-41e2-b4d9-98c944135b68.pb
```

## 输出格式

通过 `--format`（或 `-f`）指定输出格式，支持三种：

### raw（默认）

原始 JSON，Language Server 返回什么就输出什么，仅做美化缩进。适合需要完整数据的场景。

```bash
./bin/antigravity-decryptor 762506a2-5119-41e2-b4d9-98c944135b68
# 等价于
./bin/antigravity-decryptor --format raw 762506a2-5119-41e2-b4d9-98c944135b68
```

### normalized

从原始响应中提取关键字段，输出结构化 JSON。包含 cascadeId、trajectoryId、workspaceUris 以及每个 step 的类型、时间、文本内容。

```bash
./bin/antigravity-decryptor --format normalized 762506a2-5119-41e2-b4d9-98c944135b68
```

输出示例：

```json
{
  "cascadeId": "762506a2-...",
  "trajectoryId": "abc123",
  "trajectoryType": "CORTEX",
  "steps": [
    {
      "index": 0,
      "type": "CORTEX_STEP_TYPE_USER_INPUT",
      "createdAt": "2026-03-24T01:49:12Z",
      "text": "帮我审查一下这个项目的代码"
    },
    {
      "index": 1,
      "type": "CORTEX_STEP_TYPE_PLANNER_RESPONSE",
      "text": "我来帮你审查代码..."
    }
  ]
}
```

### markdown

生成可读的 Markdown 对话记录，带角色图标。默认只保留用户可见步骤，适合存档和阅读。

```bash
./bin/antigravity-decryptor --format markdown 762506a2-5119-41e2-b4d9-98c944135b68
```

如果你确实需要把内部/system-only steps 一起导出，可以显式加上 `--include-internal`：

```bash
./bin/antigravity-decryptor --format markdown --include-internal 762506a2-5119-41e2-b4d9-98c944135b68
```

输出示例：

```markdown
# Conversation Transcript

- **cascadeId**: `762506a2-...`
- **trajectoryId**: `abc123`
- **totalSteps**: 12

---

### [0] 👤 User (2026-03-24T01:49:12Z)

帮我审查一下这个项目的代码

### [1] 🤖 Assistant

我来帮你审查代码...
```

## 写入文件

默认输出到 stdout，用 `--output`（或 `-o`）写入文件：

```bash
# 导出 raw JSON 到文件
./bin/antigravity-decryptor --output conversation.json 762506a2-5119-41e2-b4d9-98c944135b68

# 导出 Markdown 到文件
./bin/antigravity-decryptor --format markdown --output transcript.md 762506a2-5119-41e2-b4d9-98c944135b68

# 也可以用 shell 重定向
./bin/antigravity-decryptor 762506a2-5119-41e2-b4d9-98c944135b68 > out.json
```

## 列出所有对话

```bash
./bin/antigravity-decryptor list
```

返回所有可见对话的摘要信息（raw JSON 格式）。

## 自定义 Language Server

工具需要调用 Antigravity 自带的 Language Server 来读取 `.pb` 文件。默认路径为：

```
/Applications/Antigravity.app/Contents/Resources/app/extensions/antigravity/bin/language_server_macos_arm
```

如果你的安装路径不同，可以通过以下方式指定：

```bash
# CLI 参数（优先级最高）
./bin/antigravity-decryptor --ls-binary /path/to/language_server_macos_arm <cascadeId>

# 环境变量
export ANTIGRAVITY_LS_PATH=/path/to/language_server_macos_arm
./bin/antigravity-decryptor <cascadeId>
```

优先级：`--ls-binary` > `ANTIGRAVITY_LS_PATH` 环境变量 > 默认路径。

## 调试模式

加 `--verbose`（或 `-v`）可在 stderr 输出调试日志：

```bash
./bin/antigravity-decryptor --verbose 762506a2-5119-41e2-b4d9-98c944135b68
```

输出类似：

```
[info] cascadeId: 762506a2-5119-41e2-b4d9-98c944135b68
[info] acquiring language server...
[info] language server HTTP port: 52094
```

## 完整参数一览

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--format` | `-f` | 输出格式：`raw` / `normalized` / `markdown` | `raw` |
| `--include-internal` | — | Markdown 模式下包含内部/system-only steps | 关闭 |
| `--output` | `-o` | 输出文件路径 | stdout |
| `--ls-binary` | — | Language Server 二进制路径 | 系统默认 |
| `--verbose` | `-v` | 输出调试日志到 stderr | 关闭 |
| `--help` | `-h` | 显示帮助信息 | — |
