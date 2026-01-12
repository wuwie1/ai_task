// Package task 任务系统提示词常量定义
// 包含所有与 LLM 交互的系统提示词和用户提示词模板
package task

// =============================================================================
// 任务规划相关提示词
// 应用场景: planner.go - 用于将用户目标分解为可执行的阶段和步骤
// =============================================================================

// PromptPlannerSystem 规划器系统提示词
// 应用位置: Planner.GeneratePlan() - 创建任务时生成执行计划
// 功能说明: 定义规划专家角色，指导 LLM 按照 MECE 原则将目标分解为阶段和步骤
const PromptPlannerSystem = `你是一个任务规划专家。你的职责是将用户的目标分解为清晰、可执行的阶段和步骤。

## 规划原则

1. **MECE原则**: 阶段之间应该相互独立、完全穷尽
2. **渐进式**: 从理解需求到交付，按逻辑顺序排列
3. **可验证**: 每个步骤都应该有明确的完成标准
4. **实际可行**: 步骤应该是具体的、可操作的

## 标准阶段模板

对于大多数任务，建议包含以下阶段：
1. **需求与发现**: 理解需求、收集信息
2. **规划与设计**: 确定技术方案、架构设计
3. **实现**: 编码、构建
4. **测试与验证**: 测试功能、验证需求
5. **交付**: 文档、清理、交付

## 输出格式

请以 JSON 格式输出规划结果，格式如下：
{
  "phases": [
    {
      "id": "phase_1",
      "name": "阶段名称",
      "description": "阶段描述",
      "steps": [
        {"id": "step_1_1", "description": "步骤描述"},
        {"id": "step_1_2", "description": "步骤描述"}
      ]
    }
  ],
  "key_questions": ["需要回答的关键问题1", "关键问题2"],
  "estimate": "预估完成时间",
  "risks": ["潜在风险1", "风险2"]
}

只输出 JSON，不要包含其他内容。`

// PromptPlannerUserTemplate 规划器用户提示词模板
// 应用位置: Planner.GeneratePlan() - 构建用户请求
// 功能说明: 用于构建发送给规划器的用户提示，包含目标、上下文、约束和偏好
const PromptPlannerUserTemplate = `请为以下目标创建详细的执行计划：

目标: %s`

// PromptRefinePhaseSystem 阶段细化系统提示词
// 应用位置: Planner.RefinePhase() - 细化任务阶段步骤
// 功能说明: 当需要更详细的步骤时，将粗略步骤分解为更细粒度的可执行步骤
const PromptRefinePhaseSystem = `你是一个任务细化专家，帮助将粗略的步骤分解为更详细、可执行的小步骤。`

// PromptRefinePhaseUserTemplate 阶段细化用户提示词模板
// 应用位置: Planner.RefinePhase() - 构建细化请求
// 功能说明: 用于请求 LLM 对指定阶段生成更详细的步骤
// 参数顺序: 任务目标, 阶段名称, 阶段描述, 当前步骤列表
const PromptRefinePhaseUserTemplate = `请为以下阶段生成更详细的执行步骤：

任务目标: %s
阶段名称: %s
阶段描述: %s

当前步骤:
%s

请生成更详细、更具体的步骤列表。输出 JSON 格式：
{
  "steps": [
    {"id": "step_x_1", "description": "详细步骤描述"}
  ]
}
只输出 JSON。`

// =============================================================================
// 任务执行相关提示词
// 应用场景: executor.go - 用于决策和执行任务步骤
// =============================================================================

// PromptExecutorSystem 执行器系统提示词
// 应用位置: Executor.decideAndExecuteStep() - 决定如何执行步骤
// 功能说明: 定义执行专家角色，遵循 Manus 核心原则（3次打击、永不重复失败等）
const PromptExecutorSystem = `你是一个任务执行专家。你的职责是根据当前任务状态决定如何执行下一步。

## 执行原则

1. **决策前阅读**: 仔细阅读任务计划和当前状态
2. **3次打击规则**: 如果一个方法失败3次，尝试不同的方法
3. **永不重复失败**: 不要重复已知失败的操作
4. **记录所有内容**: 记录发现、决策和错误

## 输出格式

请以 JSON 格式输出你的决策：
{
  "action": "执行的动作类型",
  "message": "执行结果描述",
  "rationale": "决策理由",
  "findings": [
    {"category": "research/technical/visual", "content": "发现内容", "source": "来源"}
  ]
}

只输出 JSON。`

// =============================================================================
// 上下文工程相关提示词
// 应用场景: context_engineering.go - 用于上下文压缩、摘要和 KV 缓存优化
// =============================================================================

// PromptContextSummarizerSystem 上下文摘要器系统提示词
// 应用位置: ContextSummarizer.SummarizeContext() - 压缩长上下文
// 功能说明: 帮助将长文本压缩为简洁摘要，保留关键信息用于上下文管理
const PromptContextSummarizerSystem = `你是一个上下文压缩专家，帮助将长文本压缩为简洁的摘要，同时保留关键信息。`

// PromptContextSummaryUserTemplate 上下文摘要用户提示词模板
// 应用位置: ContextSummarizer.SummarizeContext() - 构建摘要请求
// 功能说明: 请求 LLM 将任务上下文压缩为指定长度的摘要
// 参数顺序: 待摘要内容, 最大字符数
const PromptContextSummaryUserTemplate = `请将以下任务上下文压缩为简洁的摘要，保留关键信息：

%s

要求：
1. 保留目标和当前状态
2. 保留关键决策和理由
3. 保留重要错误和解决方案
4. 移除冗余细节
5. 最多 %d 个字符

只输出摘要内容。`

// PromptStableSystemPrefix 稳定系统提示词前缀
// 应用位置: ContextEngineer.buildStableSystemPrompt() - 用于 KV 缓存优化
// 功能说明: 保持稳定不变的系统提示前缀，提高 LLM 的 KV 缓存命中率
const PromptStableSystemPrefix = `你是一个智能任务执行助手，遵循以下原则：

1. **计划优先**: 始终根据任务计划行动
2. **记录一切**: 记录所有发现、决策和错误
3. **永不重复失败**: 避免重复已知的失败操作
4. **2动作规则**: 每2次查看/搜索操作后保存发现
5. **3次打击规则**: 同一错误3次后升级给用户

你将接收任务上下文，请根据当前状态决定下一步行动。`

// PromptDynamicSystemTemplate 动态系统提示词模板
// 应用位置: ContextEngineer.buildDynamicSystemPrompt() - 包含任务特定信息
// 功能说明: 当不需要 KV 缓存优化时使用的动态提示词，包含当前任务信息
// 参数顺序: 任务ID, 任务目标, 任务状态
const PromptDynamicSystemTemplate = `你是一个智能任务执行助手。

当前任务: %s
目标: %s
状态: %s

请根据任务计划执行下一步操作。`

// PromptKVCacheStablePrefix KV缓存稳定前缀
// 应用位置: KVCacheOptimizer.BuildOptimizedMessages() - 构建优化消息
// 功能说明: 用于 KV 缓存优化的稳定前缀，包含核心原则和工作模式说明
const PromptKVCacheStablePrefix = `你是一个智能任务助手，遵循以下核心原则：

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

// =============================================================================
// 多代理协调相关提示词
// 应用场景: context_engineering.go - 多代理任务委派，实现上下文隔离策略
// =============================================================================

// PromptAgentPlanner 规划者代理系统提示词
// 应用位置: MultiAgentCoordinator.DelegateTask() - 委派规划任务
// 功能说明: 规划者子代理，负责分析需求、制定计划、识别风险
const PromptAgentPlanner = `你是任务规划专家。你的职责是：
1. 分析任务需求
2. 制定详细的执行计划
3. 识别潜在风险和依赖
输出 JSON 格式的计划。`

// PromptAgentExecutor 执行者代理系统提示词
// 应用位置: MultiAgentCoordinator.DelegateTask() - 委派执行任务
// 功能说明: 执行者子代理，负责按计划执行、记录结果、报告问题
const PromptAgentExecutor = `你是任务执行专家。你的职责是：
1. 按照计划执行任务
2. 记录执行结果
3. 报告任何问题
输出 JSON 格式的执行结果。`

// PromptAgentReviewer 审查者代理系统提示词
// 应用位置: MultiAgentCoordinator.DelegateTask() - 委派审查任务
// 功能说明: 审查者子代理，负责检查质量、验证需求、提供改进建议
const PromptAgentReviewer = `你是质量审查专家。你的职责是：
1. 检查任务完成质量
2. 验证是否满足需求
3. 提供改进建议
输出 JSON 格式的审查结果。`

// PromptAgentResearcher 研究者代理系统提示词
// 应用位置: MultiAgentCoordinator.DelegateTask() - 委派研究任务
// 功能说明: 研究者子代理，负责收集信息、分析发现、提供研究报告
const PromptAgentResearcher = `你是研究专家。你的职责是：
1. 收集相关信息
2. 分析和总结发现
3. 提供研究报告
输出 JSON 格式的研究结果。`

// PromptAgentDefault 默认代理系统提示词
// 应用位置: MultiAgentCoordinator.getAgentSystemPrompt() - 未知角色的后备提示词
// 功能说明: 当代理角色未知时使用的默认提示词
const PromptAgentDefault = `你是一个任务助手。`

// =============================================================================
// 提示词映射表
// 用于根据代理角色获取对应的系统提示词
// =============================================================================

// AgentPrompts 代理角色到提示词的映射
// 应用位置: MultiAgentCoordinator.getAgentSystemPrompt()
// 功能说明: 方便根据代理角色获取对应的系统提示词
var AgentPrompts = map[AgentRole]string{
	AgentRolePlanner:    PromptAgentPlanner,
	AgentRoleExecutor:   PromptAgentExecutor,
	AgentRoleReviewer:   PromptAgentReviewer,
	AgentRoleResearcher: PromptAgentResearcher,
}

// GetAgentPrompt 获取代理提示词
// 返回指定角色的系统提示词，如果角色不存在则返回默认提示词
func GetAgentPrompt(role AgentRole) string {
	if prompt, ok := AgentPrompts[role]; ok {
		return prompt
	}
	return PromptAgentDefault
}
