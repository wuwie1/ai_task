package constant

// =============================================
// 任务状态常量
// =============================================

// TaskStatus 任务状态类型
type TaskStatus string

const (
	// TaskStatusPending 待处理
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusInProgress 进行中
	TaskStatusInProgress TaskStatus = "in_progress"
	// TaskStatusCompleted 已完成
	TaskStatusCompleted TaskStatus = "completed"
	// TaskStatusFailed 失败
	TaskStatusFailed TaskStatus = "failed"
	// TaskStatusCancelled 已取消
	TaskStatusCancelled TaskStatus = "cancelled"
)

// String 返回状态的字符串值
func (s TaskStatus) String() string {
	return string(s)
}

// IsValid 检查状态是否有效
func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusPending, TaskStatusInProgress, TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled:
		return true
	}
	return false
}

// =============================================
// 阶段状态常量
// =============================================

// PhaseStatus 阶段状态类型
type PhaseStatus string

const (
	// PhaseStatusPending 待处理
	PhaseStatusPending PhaseStatus = "pending"
	// PhaseStatusInProgress 进行中
	PhaseStatusInProgress PhaseStatus = "in_progress"
	// PhaseStatusComplete 已完成
	PhaseStatusComplete PhaseStatus = "complete"
	// PhaseStatusFailed 失败
	PhaseStatusFailed PhaseStatus = "failed"
)

// String 返回状态的字符串值
func (s PhaseStatus) String() string {
	return string(s)
}

// IsValid 检查状态是否有效
func (s PhaseStatus) IsValid() bool {
	switch s {
	case PhaseStatusPending, PhaseStatusInProgress, PhaseStatusComplete, PhaseStatusFailed:
		return true
	}
	return false
}

// =============================================
// 存储类型常量
// =============================================

// StorageType 存储类型
type StorageType string

const (
	// StorageTypeFile 仅文件存储
	StorageTypeFile StorageType = "file"
	// StorageTypeDB 仅数据库存储
	StorageTypeDB StorageType = "db"
	// StorageTypeHybrid 混合模式（数据库 + 文件镜像）
	StorageTypeHybrid StorageType = "hybrid"
)

// String 返回存储类型的字符串值
func (s StorageType) String() string {
	return string(s)
}

// IsValid 检查存储类型是否有效
func (s StorageType) IsValid() bool {
	switch s {
	case StorageTypeFile, StorageTypeDB, StorageTypeHybrid:
		return true
	}
	return false
}

// =============================================
// 动作类型常量（用于2动作规则）
// =============================================

// ActionType 动作类型
type ActionType string

const (
	// ActionTypeView 查看类操作
	ActionTypeView ActionType = "view"
	// ActionTypeBrowser 浏览器操作
	ActionTypeBrowser ActionType = "browser"
	// ActionTypeSearch 搜索操作
	ActionTypeSearch ActionType = "search"
	// ActionTypeWrite 写入操作
	ActionTypeWrite ActionType = "write"
	// ActionTypeExecute 执行操作
	ActionTypeExecute ActionType = "execute"
)

// String 返回动作类型的字符串值
func (s ActionType) String() string {
	return string(s)
}

// IsViewAction 是否为查看类动作（用于2动作规则判断）
func (s ActionType) IsViewAction() bool {
	switch s {
	case ActionTypeView, ActionTypeBrowser, ActionTypeSearch:
		return true
	}
	return false
}

// =============================================
// 发现类别常量
// =============================================

// FindingCategory 发现类别
type FindingCategory string

const (
	// FindingCategoryResearch 研究发现
	FindingCategoryResearch FindingCategory = "research"
	// FindingCategoryTechnical 技术发现
	FindingCategoryTechnical FindingCategory = "technical"
	// FindingCategoryVisual 视觉发现
	FindingCategoryVisual FindingCategory = "visual"
	// FindingCategoryResource 资源发现
	FindingCategoryResource FindingCategory = "resource"
)

// String 返回类别的字符串值
func (c FindingCategory) String() string {
	return string(c)
}

// =============================================
// 测试结果状态常量
// =============================================

// TestResultStatus 测试结果状态
type TestResultStatus string

const (
	// TestResultStatusPass 通过
	TestResultStatusPass TestResultStatus = "✓"
	// TestResultStatusFail 失败
	TestResultStatusFail TestResultStatus = "✗"
	// TestResultStatusPending 待测试
	TestResultStatusPending TestResultStatus = "pending"
)

// String 返回状态的字符串值
func (s TestResultStatus) String() string {
	return string(s)
}

// =============================================
// 默认配置常量
// =============================================

const (
	// DefaultTaskStoragePath 默认任务存储路径
	DefaultTaskStoragePath = ".tasks"
	// DefaultRereadThreshold 默认重读计划的工具调用阈值（Manus的10次规则）
	DefaultRereadThreshold = 10
	// DefaultMaxRetries 默认最大重试次数（3次打击规则）
	DefaultMaxRetries = 3
	// DefaultMaxToolResultsInContext 上下文中保留的最大工具结果数
	DefaultMaxToolResultsInContext = 5
)
