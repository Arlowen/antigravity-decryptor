# 技术方案

## 设计目标

将 Antigravity 存储在本地的 `.pb` 会话文件导出为可读格式（JSON / Markdown），用于备份、审计和二次分析。

### 核心原则

**不走 AES 猜解方案**——`.pb` 文件经过加密存储，直接逆向解密成本高且易随版本失效。本工具选择复用 Antigravity 官方自带的 `language_server_macos_arm` 本地 HTTP 接口，由官方代码负责解密和反序列化，我们只做"调接口 → 格式化输出"。

## 整体架构

```
┌──────────────┐
│   CLI 入口    │  cmd/antigravity-decryptor/main.go
│  参数解析      │  手写 flag 解析，零外部依赖
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   App 编排    │  internal/app/run.go
│  主流程控制    │  解析输入 → 获取 Server → 调接口 → 格式化输出
└──────┬───────┘
       │
       ▼
┌──────────────────────────────────────────┐
│            Server 层                      │  internal/server/
│                                          │
│  discovery.go  — 读取 daemon 目录下的      │
│                  ls_*.json 获取端口        │
│  launcher.go   — 复用/拉起 language server │
│  client.go     — HTTP 接口封装             │
└──────────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────┐
│            Model 层                       │  internal/model/
│                                          │
│  raw.go        — 原始响应结构（宽松解析）    │
│  normalized.go — 归一化结构 + 字段提取      │
└──────────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────┐
│            Export 层                      │  internal/export/
│                                          │
│  json.go       — raw / normalized JSON   │
│  markdown.go   — Markdown 对话记录         │
└──────────────────────────────────────────┘
```

## 关键实现

### 1. Language Server 复用与启动

流程位于 `internal/server/launcher.go`：

```
AcquireServer()
  ├─ 1. 读 ~/.gemini/antigravity/daemon/ls_*.json
  │     └─ 按修改时间选择最新 discovery
  │     └─ 解析 httpPort，探活（POST /GetAllCascadeTrajectories）
  │     └─ 如活 → 直接复用，返回 Server{port}
  │
  └─ 2. 如果没有存活的服务 → launchServer()
        ├─ 执行 language_server_macos_arm -standalone -persistent_mode
        ├─ setsid + 重定向 stdout/stderr，保证进程可跨命令存活
        ├─ 仅等待“当前 pid 写出的” discovery 文件
        └─ 超时 30s 未就绪 → kill 进程，返回错误
```

**关键设计决策**：

- **按 pid 绑定 discovery**：启动新进程后，只接受 `pid == cmd.Process.Pid` 的 discovery 记录，避免误读陈旧端口。
- **进程生命周期**：standalone server 以持久化模式运行，CLI 退出时不主动关闭，后续命令直接复用。
- **探活机制**：对 `GetAllCascadeTrajectories` 端点发 POST，要求返回 2xx 才认为存活。

### 2. API 接口调用

`internal/server/client.go` 封装两个 gRPC-Web 风格的 HTTP 接口：

| 接口 | 方法 | 用途 |
|------|------|------|
| `/exa.language_server_pb.LanguageServerService/GetCascadeTrajectory` | POST `{"cascadeId":"..."}` | 获取单个对话完整轨迹 |
| `/exa.language_server_pb.LanguageServerService/GetAllCascadeTrajectories` | POST `{}` | 列出所有对话摘要 |

### 3. 数据模型

#### 原始模型 (`model/raw.go`)

采用 `map[string]any` + `omitempty` 做**宽松反序列化**，避免 API 字段增删导致解析失败。

#### 归一化模型 (`model/normalized.go`)

从原始 JSON 中提取高价值字段：

```
NormalizedTrajectory
├── cascadeId / trajectoryId / trajectoryType
├── workspaceUris[]
└── steps[]
    ├── index / type / createdAt
    └── text  ← 通过 extractStepText() 从多种嵌套结构中提取
```

`extractStepText()` 按优先级遍历 `userInput.text` → `plannerResponse.text` → `notifyUser.message` 等路径，兼容不同 step 类型的结构差异。

### 4. 导出格式

| 格式 | 实现 | 说明 |
|------|------|------|
| `raw` | `export/json.go` | 原始 JSON 美化输出（`json.Indent`） |
| `normalized` | `export/json.go` | 归一化结构体序列化为 JSON |
| `markdown` | `export/markdown.go` | 按 step 生成可读对话记录，带角色图标 |

### 5. CLI 设计

- **零外部依赖**：手写参数解析，不引入 `cobra` / `pflag` 等框架，整个项目 `go.mod` 只有 module 声明。
- **输入自动识别**：传入 `.pb` 路径时自动提取文件名 stem 作为 `cascadeId`；传入 UUID 字符串时直接使用。
- **环境变量后备**：`--ls-binary` > `ANTIGRAVITY_LS_PATH` > 默认路径。

## 工程结构

```
antigravity-decryptor/
├── cmd/antigravity-decryptor/main.go   # CLI 入口 + 参数解析
├── internal/
│   ├── app/run.go                      # 主流程编排
│   ├── server/
│   │   ├── discovery.go                # daemon 目录发现
│   │   ├── launcher.go                 # server 复用/启动
│   │   └── client.go                   # HTTP 接口封装
│   ├── model/
│   │   ├── raw.go                      # 宽松反序列化结构
│   │   └── normalized.go               # 归一化结构 + 字段提取
│   └── export/
│       ├── json.go                     # JSON 导出
│       └── markdown.go                 # Markdown 导出
├── docs/                               # 技术文档
├── bin/                                # 编译产物（.gitignore）
├── go.mod
└── README.md
```
