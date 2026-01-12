package task

import (
	"ai_task/pkg/clients/llm_model"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

// Planner 任务规划器
// 使用 LLM 自动生成任务计划
type Planner struct {
	llmClient *llm_model.ClientChatModel
}

// NewPlanner 创建规划器
func NewPlanner() *Planner {
	return &Planner{
		llmClient: llm_model.GetInstance(),
	}
}

// 规划系统提示词
const plannerSystemPrompt = `你是一个任务规划专家。你的职责是将用户的目标分解为清晰、可执行的阶段和步骤。

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

// PlannerResult 规划结果
type PlannerResult struct {
	Phases       []PlanPhase `json:"phases"`
	KeyQuestions []string    `json:"key_questions"`
	Estimate     string      `json:"estimate"`
	Risks        []string    `json:"risks"`
}

// PlanPhase 规划阶段
type PlanPhase struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Steps       []PlanStep `json:"steps"`
}

// PlanStep 规划步骤
type PlanStep struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// GeneratePlan 生成任务计划
func (p *Planner) GeneratePlan(ctx context.Context, req *PlanRequest) (*PlannerResult, error) {
	// 构建用户提示
	userPrompt := fmt.Sprintf("请为以下目标创建详细的执行计划：\n\n目标: %s", req.Goal)

	if req.Context != "" {
		userPrompt += fmt.Sprintf("\n\n上下文信息: %s", req.Context)
	}

	if len(req.Constraints) > 0 {
		userPrompt += fmt.Sprintf("\n\n约束条件:\n- %s", strings.Join(req.Constraints, "\n- "))
	}

	if len(req.Preferences) > 0 {
		userPrompt += fmt.Sprintf("\n\n偏好设置:\n- %s", strings.Join(req.Preferences, "\n- "))
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: plannerSystemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userPrompt,
		},
	}

	result, err := p.llmClient.PostChatCompletionsNonStreamContent(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	// 解析结果
	planResult, err := p.parsePlanResult(result)
	if err != nil {
		log.Warnf("Failed to parse plan result, using default plan: %v", err)
		return p.createDefaultPlan(req.Goal), nil
	}

	return planResult, nil
}

// parsePlanResult 解析规划结果
func (p *Planner) parsePlanResult(result string) (*PlannerResult, error) {
	// 清理响应内容
	result = strings.TrimSpace(result)
	result = strings.TrimPrefix(result, "```json")
	result = strings.TrimPrefix(result, "```")
	result = strings.TrimSuffix(result, "```")
	result = strings.TrimSpace(result)

	var planResult PlannerResult
	if err := json.Unmarshal([]byte(result), &planResult); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

	return &planResult, nil
}

// createDefaultPlan 创建默认计划
func (p *Planner) createDefaultPlan(goal string) *PlannerResult {
	return &PlannerResult{
		Phases: []PlanPhase{
			{
				ID:          "phase_1",
				Name:        "需求与发现",
				Description: "理解需求并收集信息",
				Steps: []PlanStep{
					{ID: "step_1_1", Description: "理解用户意图"},
					{ID: "step_1_2", Description: "识别约束和需求"},
					{ID: "step_1_3", Description: "记录发现"},
				},
			},
			{
				ID:          "phase_2",
				Name:        "规划与设计",
				Description: "确定技术方案",
				Steps: []PlanStep{
					{ID: "step_2_1", Description: "定义技术方案"},
					{ID: "step_2_2", Description: "创建项目结构"},
					{ID: "step_2_3", Description: "记录决策"},
				},
			},
			{
				ID:          "phase_3",
				Name:        "实现",
				Description: "执行实现",
				Steps: []PlanStep{
					{ID: "step_3_1", Description: "按步骤执行"},
					{ID: "step_3_2", Description: "编写代码"},
					{ID: "step_3_3", Description: "增量测试"},
				},
			},
			{
				ID:          "phase_4",
				Name:        "测试与验证",
				Description: "测试和验证",
				Steps: []PlanStep{
					{ID: "step_4_1", Description: "验证需求"},
					{ID: "step_4_2", Description: "记录测试结果"},
					{ID: "step_4_3", Description: "修复问题"},
				},
			},
			{
				ID:          "phase_5",
				Name:        "交付",
				Description: "交付和总结",
				Steps: []PlanStep{
					{ID: "step_5_1", Description: "审查输出"},
					{ID: "step_5_2", Description: "确保完整"},
					{ID: "step_5_3", Description: "交付用户"},
				},
			},
		},
		KeyQuestions: []string{},
		Estimate:     "待评估",
		Risks:        []string{},
	}
}

// ConvertToTaskPhases 将规划结果转换为任务阶段
func (p *Planner) ConvertToTaskPhases(planResult *PlannerResult) []TaskPhase {
	phases := make([]TaskPhase, len(planResult.Phases))

	for i, pp := range planResult.Phases {
		steps := make([]TaskStep, len(pp.Steps))
		for j, ps := range pp.Steps {
			steps[j] = TaskStep{
				ID:          ps.ID,
				Description: ps.Description,
				Completed:   false,
			}
		}

		phases[i] = TaskPhase{
			ID:          pp.ID,
			Name:        pp.Name,
			Description: pp.Description,
			Status:      PhaseStatusPending,
			Steps:       steps,
			Order:       i + 1,
		}
	}

	return phases
}

// RefinePhase 细化阶段
// 当需要更详细的步骤时使用
func (p *Planner) RefinePhase(ctx context.Context, taskContext *TaskContext, phaseID string) ([]TaskStep, error) {
	if taskContext == nil || taskContext.Task == nil {
		return nil, fmt.Errorf("task context is nil")
	}

	// 找到目标阶段
	var targetPhase *TaskPhase
	for i := range taskContext.Task.Phases {
		if taskContext.Task.Phases[i].ID == phaseID {
			targetPhase = &taskContext.Task.Phases[i]
			break
		}
	}

	if targetPhase == nil {
		return nil, fmt.Errorf("phase not found: %s", phaseID)
	}

	refinePrompt := fmt.Sprintf(`请为以下阶段生成更详细的执行步骤：

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
只输出 JSON。`, taskContext.Task.Goal, targetPhase.Name, targetPhase.Description, formatSteps(targetPhase.Steps))

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "你是一个任务细化专家，帮助将粗略的步骤分解为更详细、可执行的小步骤。",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: refinePrompt,
		},
	}

	result, err := p.llmClient.PostChatCompletionsNonStreamContent(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to refine phase: %w", err)
	}

	// 解析结果
	result = cleanJSONResponse(result)

	var refineResult struct {
		Steps []PlanStep `json:"steps"`
	}

	if err := json.Unmarshal([]byte(result), &refineResult); err != nil {
		return nil, fmt.Errorf("failed to parse refine result: %w", err)
	}

	// 转换为 TaskStep
	steps := make([]TaskStep, len(refineResult.Steps))
	for i, s := range refineResult.Steps {
		steps[i] = TaskStep{
			ID:          s.ID,
			Description: s.Description,
			Completed:   false,
		}
	}

	return steps, nil
}

// formatSteps 格式化步骤列表
func formatSteps(steps []TaskStep) string {
	var sb strings.Builder
	for _, s := range steps {
		status := "[ ]"
		if s.Completed {
			status = "[x]"
		}
		sb.WriteString(fmt.Sprintf("- %s %s\n", status, s.Description))
	}
	return sb.String()
}

// cleanJSONResponse 清理 JSON 响应
func cleanJSONResponse(response string) string {
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
	}
	response = strings.TrimSuffix(response, "```")
	return strings.TrimSpace(response)
}
