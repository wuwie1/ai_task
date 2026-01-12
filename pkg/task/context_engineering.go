package task

import (
	"ai_task/pkg/clients/llm_model"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

// ContextEngineer 上下文工程师
// 实现 Manus 的3种上下文工程策略：
// 1. 上下文缩减（压缩、摘要化）
// 2. 上下文隔离（多代理）
// 3. 上下文卸载（工具设计）
type ContextEngineer struct {
	llmClient   *llm_model.ClientChatModel
	config      *ContextEngineerConfig
	compressor  *ContextCompressor
	summarizer  *ContextSummarizer
}

// ContextEngineerConfig 上下文工程配置
type ContextEngineerConfig struct {
	// 压缩配置
	MaxContextTokens        int  // 最大上下文令牌数
	CompressAfterToolCalls  int  // N次工具调用后压缩
	KeepRecentToolResults   int  // 保留最近N个工具结果完整

	// 摘要配置
	SummarizeThreshold      int  // 触发摘要的令牌阈值
	SummaryMaxTokens        int  // 摘要最大令牌数

	// 缓存优化
	EnableKVCacheOptimization bool // 启用KV缓存优化
	StablePromptPrefix        bool // 保持提示前缀稳定
}

// DefaultContextEngineerConfig 默认配置
func DefaultContextEngineerConfig() *ContextEngineerConfig {
	return &ContextEngineerConfig{
		MaxContextTokens:          4000,
		CompressAfterToolCalls:    5,
		KeepRecentToolResults:     3,
		SummarizeThreshold:        3000,
		SummaryMaxTokens:          500,
		EnableKVCacheOptimization: true,
		StablePromptPrefix:        true,
	}
}

// NewContextEngineer 创建上下文工程师
func NewContextEngineer(config *ContextEngineerConfig) *ContextEngineer {
	if config == nil {
		config = DefaultContextEngineerConfig()
	}

	return &ContextEngineer{
		llmClient:  llm_model.GetInstance(),
		config:     config,
		compressor: NewContextCompressor(config),
		summarizer: NewContextSummarizer(config),
	}
}

// ContextCompressor 上下文压缩器
type ContextCompressor struct {
	config *ContextEngineerConfig
}

// NewContextCompressor 创建压缩器
func NewContextCompressor(config *ContextEngineerConfig) *ContextCompressor {
	return &ContextCompressor{config: config}
}

// CompressToolResults 压缩工具调用结果
// 实现策略1：上下文缩减
func (cc *ContextCompressor) CompressToolResults(toolCalls []ToolCall) []ToolCall {
	if len(toolCalls) <= cc.config.KeepRecentToolResults {
		return toolCalls
	}

	compressed := make([]ToolCall, len(toolCalls))
	copy(compressed, toolCalls)

	// 压缩较旧的结果，只保留引用
	cutoff := len(compressed) - cc.config.KeepRecentToolResults
	for i := 0; i < cutoff; i++ {
		compressed[i] = cc.compressToolCall(compressed[i])
	}

	return compressed
}

// compressToolCall 压缩单个工具调用
func (cc *ContextCompressor) compressToolCall(tc ToolCall) ToolCall {
	// 保留关键信息，压缩结果
	compressed := tc
	compressed.Compressed = true

	// 提取并保留引用信息（URL、文件路径等）
	references := cc.extractReferences(tc.Result)
	if len(references) > 0 {
		compressed.Result = fmt.Sprintf("[压缩] 引用: %s", strings.Join(references, ", "))
	} else {
		compressed.Result = fmt.Sprintf("[压缩] 工具 %s 执行完成", tc.Name)
	}

	return compressed
}

// extractReferences 提取引用信息
func (cc *ContextCompressor) extractReferences(content string) []string {
	var refs []string

	// 提取文件路径
	// 简单的模式匹配，实际应用中可以更复杂
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "/") || strings.HasPrefix(line, "./") {
			refs = append(refs, line)
		}
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			refs = append(refs, line)
		}
	}

	// 限制引用数量
	if len(refs) > 5 {
		refs = refs[:5]
	}

	return refs
}

// ContextSummarizer 上下文摘要器
type ContextSummarizer struct {
	config    *ContextEngineerConfig
	llmClient *llm_model.ClientChatModel
}

// NewContextSummarizer 创建摘要器
func NewContextSummarizer(config *ContextEngineerConfig) *ContextSummarizer {
	return &ContextSummarizer{
		config:    config,
		llmClient: llm_model.GetInstance(),
	}
}

// SummarizeContext 摘要上下文
func (cs *ContextSummarizer) SummarizeContext(ctx context.Context, taskCtx *TaskContext) (string, error) {
	if taskCtx == nil || taskCtx.Task == nil {
		return "", nil
	}

	// 构建待摘要的内容
	content := cs.buildContextContent(taskCtx)

	// 如果内容不够长，不需要摘要
	if len(content) < cs.config.SummarizeThreshold {
		return content, nil
	}

	// 使用 LLM 生成摘要
	summaryPrompt := fmt.Sprintf(`请将以下任务上下文压缩为简洁的摘要，保留关键信息：

%s

要求：
1. 保留目标和当前状态
2. 保留关键决策和理由
3. 保留重要错误和解决方案
4. 移除冗余细节
5. 最多 %d 个字符

只输出摘要内容。`, content, cs.config.SummaryMaxTokens)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "你是一个上下文压缩专家，帮助将长文本压缩为简洁的摘要，同时保留关键信息。",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: summaryPrompt,
		},
	}

	summary, err := cs.llmClient.PostChatCompletionsNonStreamContent(ctx, messages)
	if err != nil {
		log.Warnf("Failed to summarize context: %v", err)
		// 返回原始内容的截断版本
		if len(content) > cs.config.SummaryMaxTokens {
			return content[:cs.config.SummaryMaxTokens] + "...", nil
		}
		return content, nil
	}

	return strings.TrimSpace(summary), nil
}

// buildContextContent 构建上下文内容
func (cs *ContextSummarizer) buildContextContent(taskCtx *TaskContext) string {
	var sb strings.Builder

	// 任务信息
	sb.WriteString(fmt.Sprintf("目标: %s\n", taskCtx.Task.Goal))
	sb.WriteString(fmt.Sprintf("状态: %s\n", taskCtx.Task.Status))
	sb.WriteString(fmt.Sprintf("当前阶段: %s\n\n", taskCtx.Task.CurrentPhase))

	// 阶段进度
	sb.WriteString("阶段进度:\n")
	for _, phase := range taskCtx.Task.Phases {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", phase.Name, phase.Status))
	}

	// 决策
	if len(taskCtx.Task.Decisions) > 0 {
		sb.WriteString("\n决策:\n")
		for _, d := range taskCtx.Task.Decisions {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", d.Decision, d.Rationale))
		}
	}

	// 错误
	if len(taskCtx.Task.Errors) > 0 {
		sb.WriteString("\n错误:\n")
		for _, e := range taskCtx.Task.Errors {
			sb.WriteString(fmt.Sprintf("- %s (尝试 %d): %s\n", e.Error, e.Attempt, e.Resolution))
		}
	}

	// 发现
	if taskCtx.Findings != nil && len(taskCtx.Findings.Findings) > 0 {
		sb.WriteString("\n发现:\n")
		for _, f := range taskCtx.Findings.Findings {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", f.Category, f.Content))
		}
	}

	return sb.String()
}

// BuildOptimizedContext 构建优化的上下文
// 综合使用压缩、摘要和缓存优化
func (ce *ContextEngineer) BuildOptimizedContext(ctx context.Context, taskCtx *TaskContext, toolCalls []ToolCall) (*OptimizedContext, error) {
	result := &OptimizedContext{
		Timestamp: time.Now(),
	}

	// 1. 压缩工具调用结果
	compressedCalls := ce.compressor.CompressToolResults(toolCalls)

	// 2. 构建系统提示（稳定，用于KV缓存）
	if ce.config.StablePromptPrefix {
		result.SystemPrompt = ce.buildStableSystemPrompt()
	} else {
		result.SystemPrompt = ce.buildDynamicSystemPrompt(taskCtx)
	}

	// 3. 构建任务上下文
	taskContext := ce.buildTaskContext(taskCtx, compressedCalls)

	// 4. 检查是否需要摘要
	if len(taskContext) > ce.config.SummarizeThreshold {
		summary, err := ce.summarizer.SummarizeContext(ctx, taskCtx)
		if err != nil {
			log.Warnf("Failed to summarize, using full context: %v", err)
			result.TaskContext = taskContext
		} else {
			result.TaskContext = summary
			result.IsSummarized = true
		}
	} else {
		result.TaskContext = taskContext
	}

	// 5. 提取引用信息（用于恢复完整数据）
	result.References = ce.extractAllReferences(taskCtx, compressedCalls)

	return result, nil
}

// OptimizedContext 优化后的上下文
type OptimizedContext struct {
	SystemPrompt string            `json:"system_prompt"`
	TaskContext  string            `json:"task_context"`
	References   map[string]string `json:"references"` // 引用到完整数据的映射
	IsSummarized bool              `json:"is_summarized"`
	Timestamp    time.Time         `json:"timestamp"`
}

// buildStableSystemPrompt 构建稳定的系统提示
// 用于 KV 缓存优化
func (ce *ContextEngineer) buildStableSystemPrompt() string {
	return `你是一个智能任务执行助手，遵循以下原则：

1. **计划优先**: 始终根据任务计划行动
2. **记录一切**: 记录所有发现、决策和错误
3. **永不重复失败**: 避免重复已知的失败操作
4. **2动作规则**: 每2次查看/搜索操作后保存发现
5. **3次打击规则**: 同一错误3次后升级给用户

你将接收任务上下文，请根据当前状态决定下一步行动。`
}

// buildDynamicSystemPrompt 构建动态系统提示
func (ce *ContextEngineer) buildDynamicSystemPrompt(taskCtx *TaskContext) string {
	if taskCtx == nil || taskCtx.Task == nil {
		return ce.buildStableSystemPrompt()
	}

	return fmt.Sprintf(`你是一个智能任务执行助手。

当前任务: %s
目标: %s
状态: %s

请根据任务计划执行下一步操作。`, taskCtx.Task.ID, taskCtx.Task.Goal, taskCtx.Task.Status)
}

// buildTaskContext 构建任务上下文
func (ce *ContextEngineer) buildTaskContext(taskCtx *TaskContext, toolCalls []ToolCall) string {
	var sb strings.Builder

	if taskCtx == nil || taskCtx.Task == nil {
		return ""
	}

	// 任务摘要
	sb.WriteString("## 任务状态\n")
	sb.WriteString(fmt.Sprintf("目标: %s\n", taskCtx.Task.Goal))
	sb.WriteString(fmt.Sprintf("当前阶段: %s\n", taskCtx.Task.CurrentPhase))
	sb.WriteString(fmt.Sprintf("状态: %s\n\n", taskCtx.Task.Status))

	// 阶段进度
	sb.WriteString("## 进度\n")
	for _, phase := range taskCtx.Task.Phases {
		icon := "⬜"
		switch phase.Status {
		case PhaseStatusComplete:
			icon = "✅"
		case PhaseStatusInProgress:
			icon = "🔄"
		case PhaseStatusFailed:
			icon = "❌"
		}
		sb.WriteString(fmt.Sprintf("%s %s\n", icon, phase.Name))
	}

	// 关键决策
	if len(taskCtx.Task.Decisions) > 0 {
		sb.WriteString("\n## 关键决策\n")
		// 只显示最近3个
		start := 0
		if len(taskCtx.Task.Decisions) > 3 {
			start = len(taskCtx.Task.Decisions) - 3
		}
		for i := start; i < len(taskCtx.Task.Decisions); i++ {
			d := taskCtx.Task.Decisions[i]
			sb.WriteString(fmt.Sprintf("- %s\n", d.Decision))
		}
	}

	// 错误记录
	if len(taskCtx.Task.Errors) > 0 {
		sb.WriteString("\n## 避免的错误\n")
		// 只显示最近3个
		start := 0
		if len(taskCtx.Task.Errors) > 3 {
			start = len(taskCtx.Task.Errors) - 3
		}
		for i := start; i < len(taskCtx.Task.Errors); i++ {
			e := taskCtx.Task.Errors[i]
			sb.WriteString(fmt.Sprintf("- %s\n", e.Error))
		}
	}

	// 工具调用结果
	if len(toolCalls) > 0 {
		sb.WriteString("\n## 最近操作\n")
		for _, tc := range toolCalls {
			if tc.Compressed {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", tc.Name, tc.Result))
			} else {
				// 截断长结果
				result := tc.Result
				if len(result) > 200 {
					result = result[:200] + "..."
				}
				sb.WriteString(fmt.Sprintf("- %s: %s\n", tc.Name, result))
			}
		}
	}

	return sb.String()
}

// extractAllReferences 提取所有引用
func (ce *ContextEngineer) extractAllReferences(taskCtx *TaskContext, toolCalls []ToolCall) map[string]string {
	refs := make(map[string]string)

	// 从发现中提取资源
	if taskCtx.Findings != nil {
		for i, r := range taskCtx.Findings.Resources {
			refs[fmt.Sprintf("resource_%d", i)] = r
		}
	}

	// 从工具调用中提取引用
	for _, tc := range toolCalls {
		if tc.Compressed {
			continue
		}
		extracted := ce.compressor.extractReferences(tc.Result)
		for i, ref := range extracted {
			refs[fmt.Sprintf("%s_ref_%d", tc.Name, i)] = ref
		}
	}

	return refs
}

// MultiAgentCoordinator 多代理协调器
// 实现策略2：上下文隔离
type MultiAgentCoordinator struct {
	manager    *Manager
	llmClient  *llm_model.ClientChatModel
}

// NewMultiAgentCoordinator 创建多代理协调器
func NewMultiAgentCoordinator(manager *Manager) *MultiAgentCoordinator {
	return &MultiAgentCoordinator{
		manager:   manager,
		llmClient: llm_model.GetInstance(),
	}
}

// AgentRole 代理角色
type AgentRole string

const (
	AgentRolePlanner   AgentRole = "planner"   // 规划者
	AgentRoleExecutor  AgentRole = "executor"  // 执行者
	AgentRoleReviewer  AgentRole = "reviewer"  // 审查者
	AgentRoleResearcher AgentRole = "researcher" // 研究者
)

// AgentTask 代理任务
type AgentTask struct {
	ID          string                 `json:"id"`
	Role        AgentRole              `json:"role"`
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Status      string                 `json:"status"`
}

// DelegateTask 委派任务给子代理
func (mac *MultiAgentCoordinator) DelegateTask(ctx context.Context, parentTaskID string, agentTask *AgentTask) (*AgentTask, error) {
	// 获取父任务上下文
	parentCtx, err := mac.manager.GetTaskContext(ctx, parentTaskID)
	if err != nil {
		return nil, err
	}

	// 构建子代理提示
	prompt := mac.buildAgentPrompt(agentTask, parentCtx)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: mac.getAgentSystemPrompt(agentTask.Role),
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	result, err := mac.llmClient.PostChatCompletionsNonStreamContent(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	// 解析结果
	agentTask.Output = make(map[string]interface{})
	if err := json.Unmarshal([]byte(cleanJSONResponse(result)), &agentTask.Output); err != nil {
		// 如果不是 JSON，作为文本结果
		agentTask.Output["result"] = result
	}
	agentTask.Status = "completed"

	return agentTask, nil
}

// getAgentSystemPrompt 获取代理系统提示
func (mac *MultiAgentCoordinator) getAgentSystemPrompt(role AgentRole) string {
	prompts := map[AgentRole]string{
		AgentRolePlanner: `你是任务规划专家。你的职责是：
1. 分析任务需求
2. 制定详细的执行计划
3. 识别潜在风险和依赖
输出 JSON 格式的计划。`,

		AgentRoleExecutor: `你是任务执行专家。你的职责是：
1. 按照计划执行任务
2. 记录执行结果
3. 报告任何问题
输出 JSON 格式的执行结果。`,

		AgentRoleReviewer: `你是质量审查专家。你的职责是：
1. 检查任务完成质量
2. 验证是否满足需求
3. 提供改进建议
输出 JSON 格式的审查结果。`,

		AgentRoleResearcher: `你是研究专家。你的职责是：
1. 收集相关信息
2. 分析和总结发现
3. 提供研究报告
输出 JSON 格式的研究结果。`,
	}

	if prompt, ok := prompts[role]; ok {
		return prompt
	}
	return "你是一个任务助手。"
}

// buildAgentPrompt 构建代理提示
func (mac *MultiAgentCoordinator) buildAgentPrompt(task *AgentTask, parentCtx *TaskContext) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 任务\n%s\n\n", task.Description))

	if parentCtx != nil && parentCtx.Task != nil {
		sb.WriteString("## 上下文\n")
		sb.WriteString(fmt.Sprintf("父任务目标: %s\n", parentCtx.Task.Goal))
		sb.WriteString(fmt.Sprintf("当前阶段: %s\n", parentCtx.Task.CurrentPhase))
	}

	if len(task.Input) > 0 {
		sb.WriteString("\n## 输入\n")
		for k, v := range task.Input {
			sb.WriteString(fmt.Sprintf("- %s: %v\n", k, v))
		}
	}

	return sb.String()
}

// ToolLoader 工具加载器
// 实现策略3：上下文卸载
type ToolLoader struct {
	manager *Manager
}

// NewToolLoader 创建工具加载器
func NewToolLoader(manager *Manager) *ToolLoader {
	return &ToolLoader{manager: manager}
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters"`
	Category    string            `json:"category"` // file_, browser_, shell_, etc.
}

// GetAvailableTools 获取可用工具
// 实现渐进式披露：只在需要时加载工具
func (tl *ToolLoader) GetAvailableTools(phase PhaseStatus) []ToolDefinition {
	// 基础工具（始终可用）
	baseTools := []ToolDefinition{
		{Name: "read_file", Description: "读取文件内容", Category: "file_"},
		{Name: "write_file", Description: "写入文件内容", Category: "file_"},
		{Name: "list_dir", Description: "列出目录内容", Category: "file_"},
	}

	// 根据阶段添加工具
	switch phase {
	case PhaseStatusPending:
		// 发现阶段：添加搜索和浏览工具
		return append(baseTools, []ToolDefinition{
			{Name: "web_search", Description: "搜索网络", Category: "browser_"},
			{Name: "web_fetch", Description: "获取网页内容", Category: "browser_"},
		}...)
	case PhaseStatusInProgress:
		// 执行阶段：添加执行工具
		return append(baseTools, []ToolDefinition{
			{Name: "run_command", Description: "执行命令", Category: "shell_"},
			{Name: "edit_file", Description: "编辑文件", Category: "file_"},
		}...)
	case PhaseStatusComplete:
		// 验证阶段：添加测试工具
		return append(baseTools, []ToolDefinition{
			{Name: "run_test", Description: "运行测试", Category: "shell_"},
			{Name: "verify", Description: "验证结果", Category: "shell_"},
		}...)
	}

	return baseTools
}

// MaskTools 掩码工具（用于KV缓存优化）
// 实现原则2：掩码而非移除
func (tl *ToolLoader) MaskTools(allTools []ToolDefinition, allowedCategories []string) []ToolDefinition {
	if len(allowedCategories) == 0 {
		return allTools
	}

	// 创建允许类别的集合
	allowed := make(map[string]bool)
	for _, cat := range allowedCategories {
		allowed[cat] = true
	}

	// 掩码不允许的工具（保留但标记为不可用）
	masked := make([]ToolDefinition, len(allTools))
	for i, tool := range allTools {
		masked[i] = tool
		if !allowed[tool.Category] {
			masked[i].Description = "[不可用] " + tool.Description
		}
	}

	return masked
}

// KVCacheOptimizer KV缓存优化器
// 实现原则1：围绕KV缓存设计
type KVCacheOptimizer struct {
	stablePrefix string
}

// NewKVCacheOptimizer 创建KV缓存优化器
func NewKVCacheOptimizer() *KVCacheOptimizer {
	return &KVCacheOptimizer{
		stablePrefix: buildStablePrefix(),
	}
}

// buildStablePrefix 构建稳定前缀
func buildStablePrefix() string {
	// 这个前缀应该保持稳定，不包含时间戳等变化内容
	return `你是一个智能任务助手，遵循以下核心原则：

## 核心原则
1. 计划优先：始终根据任务计划行动
2. 记录一切：记录所有发现、决策和错误  
3. 永不重复失败：避免重复已知的失败操作
4. 2动作规则：每2次查看/搜索操作后保存发现
5. 3次打击规则：同一错误3次后升级给用户

## 工作模式
- 文件系统作为外部记忆（持久化）
- 上下文窗口作为工作记忆（临时）
- 重要信息必须写入文件

`
}

// BuildOptimizedMessages 构建优化的消息
// 确保前缀稳定以提高缓存命中率
func (kvo *KVCacheOptimizer) BuildOptimizedMessages(dynamicContent string) []openai.ChatCompletionMessage {
	return []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: kvo.stablePrefix,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: dynamicContent,
		},
	}
}

// GetCacheKey 获取缓存键
func (kvo *KVCacheOptimizer) GetCacheKey() string {
	// 返回稳定前缀的哈希作为缓存键
	// 实际实现中应该使用更复杂的哈希算法
	return fmt.Sprintf("prefix_%d", len(kvo.stablePrefix))
}
