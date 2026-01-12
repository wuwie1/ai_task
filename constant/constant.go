package constant

const (
	DefaultPageLimit = 10
)

const (
	EmptyString = ""
)

// 对话拼接相关的提示词常量
const (
	// 摘要系统提示词，压缩的时候使用
	SummarySystemPrompt = "你是一个专业的对话摘要助手，擅长提取对话中的关键信息和重要事实。"

	// 摘要用户提示词模板，生成压缩摘要的时候使用
	SummaryUserPromptTemplate = `请对以下对话进行摘要，提取关键信息和重要事实。摘要应该简洁明了，保留重要的上下文信息。

对话内容：
%s

请生成摘要：`

	// 提取关键事实的系统提示词
	ExtractKeyFactsSystemPrompt = "你是一个信息提取助手，擅长从对话中提取关键事实和用户偏好。"

	// 提取关键事实的用户提示词模板，提取出关键信息到 user_profile 表
	ExtractKeyFactsUserPromptTemplate = `请从以下对话中提取关键事实和用户偏好，以键值对的形式返回。只提取明确提到的、重要的信息。

对话内容：
%s

请以 JSON 格式返回提取的关键事实，格式：{"key1": "value1", "key2": "value2"}
如果没有重要信息，返回空对象 {}。`

	// 长期记忆系统提示词前缀，这个是对话中使用
	LongTermMemoryPromptPrefix = "用户偏好和配置信息：\n"

	// 语义记忆上下文提示词模板，这个是对话中使用
	SemanticMemoryContextPromptTemplate = "相关历史对话片段：\n%s"
)
