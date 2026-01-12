# 记忆系统优化说明（参考 LangChain 最新实现）

## 优化概述

本次优化参考了 LangChain 最新版本的记忆实现，主要改进包括：

1. **摘要生成机制**：使用 LLM 对长对话生成摘要
2. **记忆压缩机制**：当会话记忆过多时自动压缩旧记忆
3. **关键事实提取**：自动从对话中提取关键事实到长期记忆
4. **智能记忆整合**：参考 LangChain 的记忆操作模式

## 新增功能

### 1. 摘要生成（Summarizer）

**文件**：`pkg/memory/summarizer.go`

- **SummarizeConversation**：对对话进行摘要，压缩记忆占用
- **ExtractKeyFacts**：从对话中提取关键事实和用户偏好（用于长期记忆）

**使用场景**：
- 长对话自动生成摘要
- 压缩旧记忆时生成摘要
- 提取用户偏好到长期记忆

### 2. 记忆压缩（Compressor）

**文件**：`pkg/memory/compressor.go`

- **CompressOldMemories**：当会话记忆超过阈值时，将旧记忆压缩为摘要
- **ShouldCompress**：判断是否需要压缩

**工作原理**：
1. 检查会话记忆数量是否超过阈值
2. 保留最新的 N 条记忆
3. 将旧记忆转换为对话格式
4. 使用 LLM 生成摘要
5. 将摘要保存为新的记忆条目

### 3. 增强的记忆选项

**新增配置项**：
- `compress_threshold`：压缩阈值（默认 20）
- `enable_summary`：是否启用摘要功能（默认 false）
- `enable_auto_extract`：是否自动提取关键事实（默认 false）

## 参考 LangChain 的设计模式

### 记忆操作模式

参考 LangChain 的记忆操作模式：
1. **接收对话和当前记忆状态**
2. **提示 LLM 决定如何扩展或整合记忆状态**
3. **返回更新后的记忆状态**

在我们的实现中：
- `ExtractKeyFacts` 使用 LLM 决定提取哪些关键事实
- `SummarizeConversation` 使用 LLM 决定如何压缩记忆
- `BuildContextWithMemory` 整合三层记忆状态

### 记忆类型

参考 LangChain 的三种记忆类型：

1. **语义记忆（Semantic Memory）**：基于向量检索
   - 已实现：`SearchSemanticMemory`
   - 使用 pgvector 进行相似度搜索

2. **情景记忆（Episodic Memory）**：对话历史
   - 已实现：`SaveSessionMemory` / `GetSessionMemory`
   - 支持摘要和压缩

3. **程序性记忆（Procedural Memory）**：用户偏好和配置
   - 已实现：`SaveLongTermMemory` / `GetLongTermMemory`
   - 支持自动提取关键事实

## 使用示例

### 启用摘要功能

```json
{
  "user_id": "user123",
  "session_id": "session456",
  "messages": [...],
  "enable_summary": true,
  "compress_threshold": 20
}
```

### 启用自动提取关键事实

```json
{
  "user_id": "user123",
  "session_id": "session456",
  "messages": [...],
  "enable_auto_extract": true
}
```

## 性能考虑

1. **摘要生成**：会增加 LLM 调用，建议按需启用
2. **记忆压缩**：在记忆数量超过阈值时触发，避免频繁压缩
3. **自动提取**：会增加 LLM 调用，建议在重要对话中启用

## 未来优化方向

1. **记忆重要性评分**：评估记忆的重要性，优先保留重要记忆
2. **增量摘要**：对新增对话进行增量摘要，而不是全量重新摘要
3. **记忆版本化**：支持记忆的版本管理和回滚
4. **多模态记忆**：支持图片、文件等非文本记忆

