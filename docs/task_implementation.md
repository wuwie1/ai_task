# Task 模块实现文档

## 概述

Task 模块实现了类似 Manus 的任务规划和执行系统，基于 "Planning with Files" 的核心理念，通过持久化存储作为外部记忆，实现高效的上下文管理和任务执行。

**支持的存储方式：**
- **文件存储 (File)** - 纯文件系统存储，符合 Manus 原始理念
- **数据库存储 (DB)** - PostgreSQL 数据库存储，支持高级查询
- **混合存储 (Hybrid)** - 数据库 + 文件镜像，兼顾查询能力和可读性

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
- 每 N 次工具调用后触发重读（可配置，默认 10 次）
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
│                      API Layer (controller/task.go)         │
├─────────────────────────────────────────────────────────────┤
│                      Task Service (pkg/task/service.go)     │
│  ├── Manager         (任务管理)                              │
│  ├── Planner         (LLM 自动规划)                          │
│  ├── Executor        (任务执行)                              │
│  └── ContextEngineer (上下文工程)                            │
├─────────────────────────────────────────────────────────────┤
│                      Storage Interface                       │
│  ├── FileStorage     (文件存储实现)                          │
│  └── DBStorage       (数据库存储实现)                        │
├─────────────────────────────────────────────────────────────┤
│                      Repository Layer                        │
│  ├── TaskRepository          (任务仓库)                      │
│  ├── TaskFindingsRepository  (发现仓库)                      │
│  └── TaskProgressRepository  (进度仓库)                      │
├─────────────────────────────────────────────────────────────┤
│                      Persistence                             │
│  ├── File System                                             │
│  │   ├── task.json / task_plan.md                           │
│  │   ├── findings.json / findings.md                        │
│  │   └── progress.json / progress.md                        │
│  └── Database (PostgreSQL)                                   │
│      ├── tasks                                               │
│      ├── task_findings                                       │
│      └── task_progress                                       │
└─────────────────────────────────────────────────────────────┘
```

## 项目结构

```
ai_task/
├── constant/
│   └── task.go                 # 任务相关常量定义
├── entity/
│   └── task.go                 # 数据库实体定义
├── model/
│   └── task.go                 # 请求/响应模型
├── repository/
│   ├── task.go                 # 仓库接口定义
│   └── xormimplement/
│       └── task.go             # Xorm 实现
├── controller/
│   └── task.go                 # API 控制器
├── pkg/task/
│   ├── types.go                # 类型定义
│   ├── storage.go              # Storage 接口 + FileStorage
│   ├── storage_db.go           # DBStorage 实现
│   ├── manager.go              # 任务管理器
│   ├── planner.go              # LLM 规划器
│   ├── executor.go             # 任务执行器
│   ├── context_engineering.go  # 上下文工程
│   └── service.go              # 业务服务层
└── full.sql                    # 数据库建表语句
```

## 常量定义 (constant/task.go)

所有枚举值和默认配置都定义在 `constant` 包中：

```go
// 任务状态
const (
    TaskStatusPending    TaskStatus = "pending"     // 待处理
    TaskStatusInProgress TaskStatus = "in_progress" // 进行中
    TaskStatusCompleted  TaskStatus = "completed"   // 已完成
    TaskStatusFailed     TaskStatus = "failed"      // 失败
    TaskStatusCancelled  TaskStatus = "cancelled"   // 已取消
)

// 阶段状态
const (
    PhaseStatusPending    PhaseStatus = "pending"
    PhaseStatusInProgress PhaseStatus = "in_progress"
    PhaseStatusComplete   PhaseStatus = "complete"
    PhaseStatusFailed     PhaseStatus = "failed"
)

// 存储类型
const (
    StorageTypeFile   StorageType = "file"   // 仅文件存储
    StorageTypeDB     StorageType = "db"     // 仅数据库存储
    StorageTypeHybrid StorageType = "hybrid" // 混合模式
)

// 动作类型（用于2动作规则）
const (
    ActionTypeView    ActionType = "view"    // 查看类操作
    ActionTypeBrowser ActionType = "browser" // 浏览器操作
    ActionTypeSearch  ActionType = "search"  // 搜索操作
    ActionTypeWrite   ActionType = "write"   // 写入操作
    ActionTypeExecute ActionType = "execute" // 执行操作
)

// 发现类别
const (
    FindingCategoryResearch  FindingCategory = "research"  // 研究发现
    FindingCategoryTechnical FindingCategory = "technical" // 技术发现
    FindingCategoryVisual    FindingCategory = "visual"    // 视觉发现
    FindingCategoryResource  FindingCategory = "resource"  // 资源发现
)

// 默认配置
const (
    DefaultTaskStoragePath         = ".tasks"
    DefaultRereadThreshold         = 10  // Manus 的 10 次规则
    DefaultMaxRetries              = 3   // 3 次打击规则
    DefaultMaxToolResultsInContext = 5
)
```

## 数据库实体 (entity/task.go)

```go
// Task 任务数据库实体
type Task struct {
    ID            string     `xorm:"pk varchar(64) 'id'"`
    UserID        string     `xorm:"varchar(64) index 'user_id'"`
    SessionID     string     `xorm:"varchar(64) index 'session_id'"`
    Goal          string     `xorm:"text 'goal'"`
    CurrentPhase  string     `xorm:"varchar(64) 'current_phase'"`
    PhasesJSON    string     `xorm:"text 'phases_json'"`
    QuestionsJSON string     `xorm:"text 'questions_json'"`
    DecisionsJSON string     `xorm:"text 'decisions_json'"`
    ErrorsJSON    string     `xorm:"text 'errors_json'"`
    Status        string     `xorm:"varchar(32) index 'status'"`
    ToolCallCount int        `xorm:"int 'tool_call_count'"`
    NeedsReread   bool       `xorm:"bool 'needs_reread'"`
    CreatedAt     time.Time  `xorm:"created 'created_at'"`
    UpdatedAt     time.Time  `xorm:"updated 'updated_at'"`
    CompletedAt   *time.Time `xorm:"'completed_at'"`
}

// TaskFindings 任务发现数据库实体
type TaskFindings struct {
    ID               int64     `xorm:"pk autoincr 'id'"`
    TaskID           string    `xorm:"varchar(64) index 'task_id'"`
    RequirementsJSON string    `xorm:"text 'requirements_json'"`
    FindingsJSON     string    `xorm:"text 'findings_json'"`
    ResourcesJSON    string    `xorm:"text 'resources_json'"`
    UpdatedAt        time.Time `xorm:"updated 'updated_at'"`
}

// TaskProgress 任务进度数据库实体
type TaskProgress struct {
    ID              int64     `xorm:"pk autoincr 'id'"`
    TaskID          string    `xorm:"varchar(64) index 'task_id'"`
    SessionDate     string    `xorm:"varchar(16) 'session_date'"`
    EntriesJSON     string    `xorm:"text 'entries_json'"`
    TestResultsJSON string    `xorm:"text 'test_results_json'"`
    ErrorLogJSON    string    `xorm:"text 'error_log_json'"`
    UpdatedAt       time.Time `xorm:"updated 'updated_at'"`
}
```

## Repository 接口 (repository/task.go)

```go
// TaskRepository 任务仓库接口
type TaskRepository interface {
    Upsert(req *model.UpsertTaskCondition) error
    Get(taskID string) (*entity.Task, error)
    List(condition *model.TaskListCondition) ([]*entity.Task, error)
    Delete(taskID string) error
    Query(condition *model.TaskQueryCondition) ([]*entity.Task, int64, error)
    GetStats(userID string) (*model.TaskStats, error)
}

// TaskFindingsRepository 任务发现仓库接口
type TaskFindingsRepository interface {
    Upsert(req *model.UpsertTaskFindingsCondition) error
    Get(taskID string) (*entity.TaskFindings, error)
    Delete(taskID string) error
}

// TaskProgressRepository 任务进度仓库接口
type TaskProgressRepository interface {
    Upsert(req *model.UpsertTaskProgressCondition) error
    Get(taskID string) (*entity.TaskProgress, error)
    Delete(taskID string) error
}
```

## 存储配置

### 文件存储（默认）

```go
import "ai_task/pkg/task"

// 使用默认配置（文件存储）
config := task.DefaultTaskManagerConfig()
manager, err := task.NewManager(config)
```

### 数据库存储

```go
import (
    "ai_task/pkg/task"
    "ai_task/constant"
    "ai_task/repository/xormimplement"
)

// 获取 Repository Factory
factory := xormimplement.GetRepositoryFactoryInstance()

// 配置数据库存储
config := &task.TaskManagerConfig{
    StorageType:     constant.StorageTypeDB,
    StoragePath:     ".tasks",  // 备用文件路径
    EnableFileSync:  false,     // 不同步到文件
    RereadThreshold: constant.DefaultRereadThreshold,
    MaxRetries:      constant.DefaultMaxRetries,
}

// 创建 Manager
manager, err := task.NewManager(config, task.WithRepositoryFactory(factory))
```

### 混合存储（推荐生产环境）

```go
config := &task.TaskManagerConfig{
    StorageType:     constant.StorageTypeHybrid,
    StoragePath:     ".tasks",
    EnableFileSync:  true,  // 同时同步到文件（异步）
    RereadThreshold: 10,
    MaxRetries:      3,
}

manager, err := task.NewManager(config, task.WithRepositoryFactory(factory))
```

## 核心组件

### 1. Manager (manager.go)

任务管理核心：

```go
// 核心功能
- CreateTask()              // 创建任务
- GetTask()                 // 获取任务
- UpdatePhaseStatus()       // 更新阶段状态
- CompleteStep()            // 完成步骤
- RecordError()             // 记录错误（3次打击规则）
- AddDecision()             // 添加决策
- AddFinding()              // 添加发现
- RecordViewAction()        // 2动作规则
- IncrementToolCallCount()  // 工具调用计数（重读阈值）
- CheckCompletion()         // 完成检查
```

### 2. Planner (planner.go)

LLM 自动规划：

```go
- GeneratePlan()         // 根据目标生成阶段和步骤
- ConvertToTaskPhases()  // 转换为任务阶段
- RefinePhase()          // 细化阶段步骤
```

### 3. Executor (executor.go)

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
```

### 4. Context Engineering (context_engineering.go)

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

### 5. DBStorage 特有功能

数据库存储提供额外的查询能力：

```go
// 高级查询
tasks, total, err := dbStorage.QueryTasks(&task.TaskQueryOptions{
    UserID:    "user_123",
    Status:    "in_progress",
    Keyword:   "注册",
    StartDate: time.Now().AddDate(0, -1, 0),
    Limit:     10,
    Offset:    0,
    OrderBy:   "created_at",
    OrderDesc: true,
})

// 统计信息
stats, err := dbStorage.GetTaskStats("user_123")
// stats.Total, stats.Pending, stats.InProgress, stats.Completed, stats.Failed
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
    {"id": "phase_2", "name": "规划与设计", "status": "pending"}
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
    // 存储配置
    StorageType    StorageType // 存储类型：file, db, hybrid
    StoragePath    string      // 文件存储路径，默认 ".tasks"
    EnableFileSync bool        // 混合模式下是否同步到文件

    // 行为配置
    RereadThreshold      int    // 重读阈值，默认 10 次工具调用
    TwoActionRuleEnabled bool   // 启用2动作规则
    MaxRetries           int    // 最大重试次数（3次打击规则）
    EnableAutoPlanning   bool   // 启用 LLM 自动规划
    
    // 上下文压缩配置
    Compression ContextCompression
}

type ContextCompression struct {
    MaxToolResultsInContext int  // 上下文中保留的工具结果数
    CompressOlderResults    bool // 压缩较旧结果
    KeepReferencesOnly      bool // 只保留引用
}
```

## 数据库表结构

详见 `full.sql`，包含以下表：

| 表名 | 描述 |
|------|------|
| tasks | 任务主表，存储任务规划和执行状态 |
| task_findings | 任务发现表，存储研究发现和资源 |
| task_progress | 任务进度表，存储执行进度和测试结果 |

所有字段都有详细的中文注释。

## 文件结构（文件存储模式）

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
| 文件系统作为外部记忆 | Storage 接口 + FileStorage/DBStorage |
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
2. **Repository 模式**：数据访问层与业务逻辑分离
3. **配置化**：所有阈值和规则都可配置
4. **模块化**：各组件（Manager、Planner、Executor）可独立使用
5. **钩子系统**：支持 PreAction/PostAction 钩子扩展

## 存储方式选择建议

| 场景 | 推荐存储 | 理由 |
|------|---------|------|
| 开发/调试 | File | 便于查看和调试 |
| 单机部署 | File 或 Hybrid | 简单可靠 |
| 生产环境 | Hybrid | 兼顾查询能力和可读性 |
| 高并发场景 | DB | 更好的并发控制 |
| 需要复杂查询 | DB 或 Hybrid | 支持分页、过滤、统计 |

## 后续改进方向

1. 集成更多 LLM 模型支持
2. 添加任务模板系统
3. 支持任务依赖和编排
4. 添加 WebSocket 实时通知
5. 集成向量数据库用于语义搜索
6. 支持分布式任务执行
7. 添加任务导入/导出功能

---

*文档版本: 1.1.0*
*更新日期: 2026-01-12*
*更新内容: 添加数据库存储支持、Repository 层设计、常量定义说明*
