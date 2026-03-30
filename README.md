# antigravity-decryptor

本地工具集，用于读取 Antigravity 保存在 `~/.gemini/antigravity/conversations/*.pb` 下的会话文件并导出为结构化 JSON 或可读文本。

核心原则：不走 AES 猜解方案，直接复用官方 `language_server_macos_arm` 本地 HTTP 接口。

## 安装

```bash
go build -o bin/antigravity-decryptor ./cmd/antigravity-decryptor/
```

## 使用

### 导出 raw JSON（默认格式）

```bash
# 通过 cascadeId（对话 UUID）
./bin/antigravity-decryptor 762506a2-5119-41e2-b4d9-98c944135b68

# 通过 .pb 文件路径（自动提取文件名 stem 作为 cascadeId）
./bin/antigravity-decryptor ~/.gemini/antigravity/conversations/762506a2-5119-41e2-b4d9-98c944135b68.pb
```

### 导出归一化 JSON

```bash
./bin/antigravity-decryptor --format normalized 762506a2-5119-41e2-b4d9-98c944135b68
```

### 导出 markdown transcript

```bash
./bin/antigravity-decryptor --format markdown 762506a2-5119-41e2-b4d9-98c944135b68
```

### 写到文件

```bash
./bin/antigravity-decryptor --format raw --output out.json 762506a2-5119-41e2-b4d9-98c944135b68
```

### 列出所有可见 conversation

```bash
./bin/antigravity-decryptor list
```

### 自定义 language server 路径

```bash
# 通过 CLI 参数
./bin/antigravity-decryptor --ls-binary /path/to/language_server_macos_arm <cascadeId>

# 通过环境变量
ANTIGRAVITY_LS_PATH=/path/to/language_server_macos_arm ./bin/antigravity-decryptor <cascadeId>
```

## 工作原理

1. 优先复用已有 language server（读取 `~/.gemini/antigravity/daemon/ls_*.json`）
2. 如果没有存活的服务，自动拉起新的 standalone language server
3. 调用 `GetCascadeTrajectory` 接口读取完整对话轨迹
4. 按指定格式导出

## 工程结构

```
cmd/
  antigravity-decryptor/   CLI 入口
internal/
  server/
    discovery.go        discovery 文件解析 + 日志端口提取
    launcher.go         language server 启动/复用
    client.go           本地 HTTP 接口调用
  model/
    raw.go              原始响应结构（宽松解析）
    normalized.go       归一化结构 + 字段提取
  export/
    json.go             raw/normalized JSON 输出
    markdown.go         markdown transcript 输出
  app/
    run.go              主流程编排
```

## 环境要求

- macOS arm64（language_server_macos_arm 是 arm 二进制）
- Antigravity 已安装：`/Applications/Antigravity.app`
- `~/.gemini/antigravity/conversations/<cascadeId>.pb` 文件存在
