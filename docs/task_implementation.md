# Task 模块实现文档

## 概述

Task 模块实现了类似 Manus 的任务规划和执行系统，基于 "Planning with Files" 的核心理念，通过持久化的文件系统作为外部记忆，实现高效的上下文管理和任务执行。

## 核心原则

基于 Manus 的上下文工程原则：

### 1. 文件系统作为外部记忆
```
上下文窗口 = RAM（易失性，有限）
文件系统 = 磁盘（持久性，无限）
→ 任何重要的东西都要写入磁盘
```

### 2. 通过复述操纵注意力
- 在做重大决策之前重读计划
- 每 N 次工具调用后触发重读（可配置）
- 保持目标在注意力窗口中

### 3. 保留错误信息
- 记录所有错误用于学习
- 3次打击规则：同一错误3次后升级给用户
- 永不重复已知的失败操作

### 4. 2动作规则
- 每2次视图/浏览/搜索操作后保存发现
- 防止多模态信息丢失

### 5. KV 缓存优化
- 保持提示前缀稳定
- 掩码工具而非移除（保持缓存有效性）
- 压缩较旧的工具结果

## 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                      Task Service                            │
│  ├── Manager         (任务管理)                              │
│  ├── Planner         (LLM 自动规划)                         │
│  ├── Executor        (任务执行)                              │
│  └── ContextEngineer (上下文工程)                            │
├─────────────────────────────────────────────────────────────┤
│                      Storage Layer                           │
│  └── FileStorage     (基于文件的持久化)                      │
├─────────────────────────────────────────────────────────────┤
│                      File System                             │
│  ├── task.json / task_plan.md     (任务计划)                │
│  ├── findings.json / findings.md   (发现和决策)             │
│  └── progress.json / progress.md   (进度日志)               │
└─────────────────────────────────────────────────────────────┘
```

## 核心组件

### 1. Types (types.go)

定义所有数据结构：

```go
// 任务状态
type TaskStatus string
const (
    TaskStatusPending    TaskStatus = "pending"
    TaskStatusInProgress TaskStatus = "in_progress"
    TaskStatusCompleted  TaskStatus = "completed"
    TaskStatusFailed     TaskStatus = "failed"
    TaskStatusCancelled  TaskStatus = "cancelled"
)

// 核心任务结构
type Task struct {
    ID           string        // 唯一标识
    UserID       string        // 用户 ID
    SessionID    string        // 会话 ID
    Goal         string        // 任务目标
    CurrentPhase string        // 当前阶段
    Phases       []TaskPhase   // 阶段列表
    KeyQuestions []string      // 关键问题
    Decisions    []Decision    // 决策记录
    Errors       []ErrorRecord // 错误记录
    Status       TaskStatus    // 任务状态
    // ...
}
```

### 2. Storage (storage.go)

文件存储实现：

- 同时保存 JSON（程序读取）和 Markdown（人类可读）格式
- 支持任务、发现、进度的独立存储
- 线程安全的读写操作

### 3. Manager (manager.go)

任务管理核心：

```go
// 核心功能
- CreateTask()           // 创建任务
- GetTask()              // 获取任务
- UpdatePhaseStatus()    // 更新阶段状态
- CompleteStep()         // 完成步骤
- RecordError()          // 记录错误（3次打击规则）
- AddDecision()          // 添加决策
- AddFinding()           // 添加发现
- RecordViewAction()     // 2动作规则
- IncrementToolCallCount() // 工具调用计数（重读阈值）
- CheckCompletion()      // 完成检查
```

### 4. Planner (planner.go)

LLM 自动规划：

```go
// 使用 LLM 生成任务计划
- GeneratePlan()         // 根据目标生成阶段和步骤
- ConvertToTaskPhases()  // 转换为任务阶段
- RefinePhase()          // 细化阶段步骤
```

### 5. Executor (executor.go)

任务执行引擎：

```go
// 执行功能
- ExecuteStep()          // 执行单个步骤（含3次打击规则）
- ExecutePhase()         // 执行整个阶段
- ExecuteTask()          // 执行整个任务

// 钩子系统
- ActionTracker          // 2动作规则追踪
- ErrorTracker           // 3次打击规则追踪
- CompletionChecker      // 完成检查（5问题重启测试）

// 会话管理
- Session                // 会话封装
```

### 6. Context Engineering (context_engineering.go)

上下文工程策略：

```go
// 策略1：上下文缩减
- ContextCompressor      // 压缩工具调用结果
- ContextSummarizer      // 使用 LLM 生成摘要

// 策略2：上下文隔离
- MultiAgentCoordinator  // 多代理协调
- AgentRole              // 规划者/执行者/审查者/研究者

// 策略3：上下文卸载
- ToolLoader             // 渐进式工具加载
- MaskTools              // 工具掩码（保持 KV 缓存）

// KV 缓存优化
- KVCacheOptimizer       // 稳定前缀优化
```

### 7. Service (service.go)

业务逻辑层：

```go
// 封装所有组件
- CreateTask()           // 创建任务（含 LLM 规划）
- ExecuteTask()          // 执行任务
- StartSession()         // 开始会话
- GetOptimizedContext()  // 获取优化上下文
// ...
```

## API 接口

### 任务管理

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/v1/task | 创建任务 |
| GET | /api/v1/task/:task_id | 获取任务 |
| DELETE | /api/v1/task/:task_id | 删除任务 |
| GET | /api/v1/tasks | 列出任务 |
| POST | /api/v1/task/execute | 执行任务 |

### 上下文管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/task/:task_id/context | 获取完整上下文 |
| GET | /api/v1/task/:task_id/summary | 获取任务摘要 |
| POST | /api/v1/task/:task_id/optimized-context | 获取优化上下文 |

### 阶段和步骤

| 方法 | 路径 | 描述 |
|------|------|------|
| PUT | /api/v1/task/:task_id/phase | 更新阶段状态 |
| PUT | /api/v1/task/:task_id/step | 完成步骤 |

### 发现、决策和错误

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/v1/task/:task_id/finding | 添加发现 |
| POST | /api/v1/task/:task_id/decision | 添加决策 |
| POST | /api/v1/task/:task_id/error | 记录错误 |

### 完成检查

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/task/:task_id/completion | 检查完成状态 |
| POST | /api/v1/task/:task_id/view-action | 记录视图动作 |

### 会话管理

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/v1/session | 开始会话 |
| POST | /api/v1/session/:session_id/execute | 执行会话 |
| GET | /api/v1/session/:session_id/stop | 检查是否可停止 |

## 使用示例

### 1. 创建任务

```bash
curl -X POST http://localhost:8080/api/v1/task \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_123",
    "session_id": "session_456",
    "goal": "实现一个用户注册功能"
  }'
```

响应：
```json
{
  "task_id": "abc123",
  "goal": "实现一个用户注册功能",
  "phases": [
    {"id": "phase_1", "name": "需求与发现", "status": "pending"},
    {"id": "phase_2", "name": "规划与设计", "status": "pending"},
    ...
  ]
}
```

### 2. 执行任务

```bash
curl -X POST http://localhost:8080/api/v1/task/execute \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "abc123"
  }'
```

### 3. 添加发现

```bash
curl -X POST http://localhost:8080/api/v1/task/abc123/finding \
  -H "Content-Type: application/json" \
  -d '{
    "category": "research",
    "content": "发现 bcrypt 适合密码加密",
    "source": "https://docs.example.com"
  }'
```

### 4. 记录视图动作（2动作规则）

```bash
curl -X POST http://localhost:8080/api/v1/task/abc123/view-action \
  -H "Content-Type: application/json" \
  -d '{
    "action_type": "search"
  }'
```

响应：
```json
{
  "message": "action recorded",
  "needs_save": true,
  "action_rule": "2-action rule"
}
```

### 5. 检查完成状态

```bash
curl http://localhost:8080/api/v1/task/abc123/completion
```

响应：
```json
{
  "complete": false,
  "incomplete_phases": ["Testing & Verification", "Delivery"],
  "reboot_check": {
    "where_am_i": "Implementation",
    "where_going": "Testing & Verification → Delivery",
    "what_is_goal": "实现一个用户注册功能",
    "what_learned": "5 个发现",
    "what_done": "12 个进度条目",
    "all_answered": true
  },
  "can_stop": false
}
```

## 配置选项

```go
type TaskManagerConfig struct {
    StoragePath          string // 存储路径，默认 ".tasks"
    RereadThreshold      int    // 重读阈值，默认 10 次工具调用
    TwoActionRuleEnabled bool   // 启用2动作规则
    MaxRetries           int    // 最大重试次数（3次打击规则）
    EnableAutoPlanning   bool   // 启用 LLM 自动规划
    
    Compression struct {
        MaxToolResultsInContext int  // 上下文中保留的工具结果数
        CompressOlderResults    bool // 压缩较旧结果
        KeepReferencesOnly      bool // 只保留引用
    }
}
```

## 文件结构

任务创建后的文件结构：

```
.tasks/
└── {task_id}/
    ├── task.json           # 任务数据（JSON）
    ├── task_plan.md        # 任务计划（Markdown）
    ├── findings.json       # 发现数据（JSON）
    ├── findings.md         # 发现文档（Markdown）
    ├── progress.json       # 进度数据（JSON）
    └── progress.md         # 进度日志（Markdown）
```

## 与 Manus 的对应关系

| Manus 原则 | 实现 |
|-----------|------|
| 文件系统作为外部记忆 | FileStorage 同时保存 JSON 和 Markdown |
| 通过复述操纵注意力 | IncrementToolCallCount + RereadThreshold |
| 保留错误信息 | RecordError + ErrorTracker |
| 2动作规则 | RecordViewAction + ActionTracker |
| 3次打击规则 | ErrorTracker + MaxRetries |
| 5问题重启测试 | CompletionChecker.performRebootCheck |
| KV 缓存优化 | KVCacheOptimizer + 稳定前缀 |
| 上下文压缩 | ContextCompressor + ContextSummarizer |
| 多代理协调 | MultiAgentCoordinator |
| 渐进式工具加载 | ToolLoader + MaskTools |

## Token 节省策略

1. **上下文压缩**：压缩较旧的工具调用结果，只保留引用
2. **摘要生成**：对长上下文使用 LLM 生成摘要
3. **渐进式加载**：只在需要时加载工具和信息
4. **KV 缓存优化**：保持稳定前缀提高缓存命中率
5. **文件卸载**：将完整数据存储在文件系统，上下文中只保留摘要

## 扩展性

该模块设计遵循以下扩展原则：

1. **接口抽象**：Storage 接口允许替换不同的存储实现
2. **配置化**：所有阈值和规则都可配置
3. **模块化**：各组件（Manager、Planner、Executor）可独立使用
4. **钩子系统**：支持 PreAction/PostAction 钩子扩展

## 后续改进方向

1. 集成更多 LLM 模型支持
2. 添加任务模板系统
3. 支持任务依赖和编排
4. 添加 WebSocket 实时通知
5. 集成向量数据库用于语义搜索
6. 支持分布式任务执行

---

*文档版本: 1.0.0*
*创建日期: 2026-01-12*
