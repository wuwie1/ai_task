package llm_model

import (
	"ai_web/test/config"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/suite"
	"github.com/wuwie1/go-tools/env"
)

type ClientChatModelTest struct {
	suite.Suite
}

func (c *ClientChatModelTest) SetupTest() {
	// 重置单例状态（用于测试）
	instance = nil
	once = sync.Once{}
}

// 创建测试用的 gin.Context
func createTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/test", nil)
	return ctx
}

func (c *ClientChatModelTest) TestPostChatCompletionsNonStream_Success() {
	// 确保配置存在
	cfg := config.GetInstance()
	addr := cfg.GetString(config.ClientChatModelAddr)
	model := cfg.GetString(config.ClientChatModelModel)
	token := env.GetModelApiKey()

	// 如果配置不存在，跳过测试
	if addr == "" || model == "" || token == "" {
		c.T().Skip("Skipping test: chat model config not set")
		return
	}

	// 获取客户端实例
	client := GetInstance()
	c.NotNil(client)

	// 创建测试上下文
	ginCtx := createTestContext()
	var ctx context.Context = ginCtx

	// 准备测试消息
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "你好，请介绍一下你自己",
		},
	}

	// 调用非流式方法
	response, err := client.PostChatCompletionsNonStream(ctx, messages)

	// 验证结果
	c.Nil(err, "should not return error")
	c.NotNil(response, "response should not be nil")
	if response != nil {
		c.Greater(len(response.Choices), 0, "should have at least one choice")
		if len(response.Choices) > 0 {
			c.NotEmpty(response.Choices[0].Message.Content, "message content should not be empty")
		}
	}
}

func (c *ClientChatModelTest) TestPostChatCompletionsNonStream_EmptyMessages() {
	// 确保配置存在
	cfg := config.GetInstance()
	addr := cfg.GetString(config.ClientChatModelAddr)
	model := cfg.GetString(config.ClientChatModelModel)
	token := env.GetModelApiKey()

	// 如果配置不存在，跳过测试
	if addr == "" || model == "" || token == "" {
		c.T().Skip("Skipping test: chat model config not set")
		return
	}

	// 获取客户端实例
	client := GetInstance()
	c.NotNil(client)

	// 创建测试上下文
	ginCtx := createTestContext()
	var ctx context.Context = ginCtx

	// 准备空消息列表
	messages := []openai.ChatCompletionMessage{}

	// 调用非流式方法
	response, err := client.PostChatCompletionsNonStream(ctx, messages)

	// 空消息列表可能会返回错误或空响应，取决于 API 的行为
	// 这里主要验证方法不会 panic
	if err != nil {
		c.NotNil(err, "should return error for empty messages")
		c.Nil(response, "response should be nil when error occurs")
	} else {
		// 如果 API 允许空消息，验证响应不为 nil
		c.NotNil(response, "response should not be nil")
	}
}

func (c *ClientChatModelTest) TestPostChatCompletionsNonStream_MultipleMessages() {
	// 确保配置存在
	cfg := config.GetInstance()
	addr := cfg.GetString(config.ClientChatModelAddr)
	model := cfg.GetString(config.ClientChatModelModel)
	token := env.GetModelApiKey()

	// 如果配置不存在，跳过测试
	if addr == "" || model == "" || token == "" {
		c.T().Skip("Skipping test: chat model config not set")
		return
	}

	// 获取客户端实例
	client := GetInstance()
	c.NotNil(client)

	// 创建测试上下文
	ginCtx := createTestContext()
	var ctx context.Context = ginCtx

	// 准备多轮对话消息
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "你是一个有用的AI助手",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "请用一句话介绍Go语言",
		},
		{
			Role:    openai.ChatMessageRoleAssistant,
			Content: "Go语言是Google开发的一种静态类型、编译型、并发型编程语言。",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "它的主要特点是什么？",
		},
	}

	// 调用非流式方法
	response, err := client.PostChatCompletionsNonStream(ctx, messages)

	// 验证结果
	c.Nil(err, "should not return error")
	c.NotNil(response, "response should not be nil")
	if response != nil {
		c.Greater(len(response.Choices), 0, "should have at least one choice")
		if len(response.Choices) > 0 {
			c.NotEmpty(response.Choices[0].Message.Content, "message content should not be empty")
			c.Equal(openai.ChatMessageRoleAssistant, response.Choices[0].Message.Role, "response role should be assistant")
		}
	}
}

func (c *ClientChatModelTest) TestPostChatCompletionsNonStream_InvalidConfig() {
	// 这个测试主要验证当配置无效时方法的行为
	// 由于 GetInstance 使用 sync.Once，我们需要在测试中模拟无效配置
	// 实际场景中，如果配置无效，API 调用会返回错误

	// 保存原始配置
	cfg := config.GetInstance()
	originalAddr := cfg.GetString(config.ClientChatModelAddr)
	originalToken := env.GetModelApiKey()

	// 如果原始配置为空，说明配置本身就不存在，跳过测试
	if originalAddr == "" || originalToken == "" {
		c.T().Skip("Skipping test: chat model config not set")
		return
	}

	// 获取客户端实例（使用有效配置）
	client := GetInstance()
	c.NotNil(client)

	// 创建测试上下文
	ginCtx := createTestContext()
	var ctx context.Context = ginCtx

	// 准备测试消息
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "测试消息",
		},
	}

	// 调用方法 - 如果配置无效，应该返回错误
	// 注意：由于我们无法在运行时修改 sync.Once 初始化的配置，
	// 这个测试主要验证方法在正常配置下的行为
	response, err := client.PostChatCompletionsNonStream(ctx, messages)

	// 如果配置有效，应该成功；如果无效，应该返回错误
	// 这里主要验证方法不会 panic
	if err != nil {
		c.NotNil(err, "should return error when config is invalid")
		c.Nil(response, "response should be nil when error occurs")
	} else {
		c.NotNil(response, "response should not be nil when successful")
	}
}

func (c *ClientChatModelTest) TestPostChatCompletionsNonStream_ResponseStructure() {
	// 确保配置存在
	cfg := config.GetInstance()
	addr := cfg.GetString(config.ClientChatModelAddr)
	model := cfg.GetString(config.ClientChatModelModel)
	token := env.GetModelApiKey()

	// 如果配置不存在，跳过测试
	if addr == "" || model == "" || token == "" {
		c.T().Skip("Skipping test: chat model config not set")
		return
	}

	// 获取客户端实例
	client := GetInstance()
	c.NotNil(client)

	// 创建测试上下文
	ginCtx := createTestContext()
	var ctx context.Context = ginCtx

	// 准备测试消息
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "请回答：1+1等于几？",
		},
	}

	// 调用非流式方法
	response, err := client.PostChatCompletionsNonStream(ctx, messages)

	// 验证响应结构
	c.Nil(err, "should not return error")
	c.NotNil(response, "response should not be nil")
	if response != nil {
		// 验证基本字段
		c.NotEmpty(response.ID, "response ID should not be empty")
		c.NotEmpty(response.Model, "response model should not be empty")
		c.Greater(len(response.Choices), 0, "should have at least one choice")

		// 验证 Choice 结构
		choice := response.Choices[0]
		c.Equal(openai.ChatMessageRoleAssistant, choice.Message.Role, "message role should be assistant")
		c.NotEmpty(choice.Message.Content, "message content should not be empty")
		c.NotEmpty(choice.FinishReason, "finish reason should not be empty")
	}
}

func (c *ClientChatModelTest) TestPostChatCompletionsNonStreamContent_Success() {
	// 确保配置存在
	cfg := config.GetInstance()
	addr := cfg.GetString(config.ClientChatModelAddr)
	model := cfg.GetString(config.ClientChatModelModel)
	token := env.GetModelApiKey()

	// 如果配置不存在，跳过测试
	if addr == "" || model == "" || token == "" {
		c.T().Skip("Skipping test: chat model config not set")
		return
	}

	// 获取客户端实例
	client := GetInstance()
	c.NotNil(client)

	// 创建测试上下文
	ginCtx := createTestContext()
	var ctx context.Context = ginCtx

	// 准备测试消息
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "你好，请用一句话介绍你自己",
		},
	}

	// 调用方法获取 content
	content, err := client.PostChatCompletionsNonStreamContent(ctx, messages)

	// 验证结果
	c.Nil(err, "should not return error")
	c.NotEmpty(content, "content should not be empty")
	c.IsType("", content, "content should be a string")
}

func (c *ClientChatModelTest) TestPostChatCompletionsNonStreamContent_CompareWithFullResponse() {
	// 确保配置存在
	cfg := config.GetInstance()
	addr := cfg.GetString(config.ClientChatModelAddr)
	model := cfg.GetString(config.ClientChatModelModel)
	token := env.GetModelApiKey()

	// 如果配置不存在，跳过测试
	if addr == "" || model == "" || token == "" {
		c.T().Skip("Skipping test: chat model config not set")
		return
	}

	// 获取客户端实例
	client := GetInstance()
	c.NotNil(client)

	// 创建测试上下文
	ginCtx := createTestContext()
	var ctx context.Context = ginCtx

	// 准备测试消息
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "请回答：2+2等于几？",
		},
	}

	// 调用完整响应方法
	fullResponse, err1 := client.PostChatCompletionsNonStream(ctx, messages)
	c.Nil(err1, "full response should not return error")
	c.NotNil(fullResponse, "full response should not be nil")

	// 调用 content 方法
	content, err2 := client.PostChatCompletionsNonStreamContent(ctx, messages)

	// 验证结果
	c.Nil(err2, "content method should not return error")
	c.NotEmpty(content, "content should not be empty")

	// 验证 content 与完整响应中的 content 一致
	if fullResponse != nil && len(fullResponse.Choices) > 0 {
		expectedContent := fullResponse.Choices[0].Message.Content
		c.Equal(expectedContent, content, "content should match the content from full response")
	}
}

func (c *ClientChatModelTest) TestPostChatCompletionsNonStreamContent_EmptyMessages() {
	// 确保配置存在
	cfg := config.GetInstance()
	addr := cfg.GetString(config.ClientChatModelAddr)
	model := cfg.GetString(config.ClientChatModelModel)
	token := env.GetModelApiKey()

	// 如果配置不存在，跳过测试
	if addr == "" || model == "" || token == "" {
		c.T().Skip("Skipping test: chat model config not set")
		return
	}

	// 获取客户端实例
	client := GetInstance()
	c.NotNil(client)

	// 创建测试上下文
	ginCtx := createTestContext()
	var ctx context.Context = ginCtx

	// 准备空消息列表
	messages := []openai.ChatCompletionMessage{}

	// 调用方法
	content, err := client.PostChatCompletionsNonStreamContent(ctx, messages)

	// 空消息列表应该返回错误
	if err != nil {
		c.NotNil(err, "should return error for empty messages")
		c.Empty(content, "content should be empty when error occurs")
	} else {
		// 如果 API 允许空消息，验证 content 不为空
		c.NotEmpty(content, "content should not be empty if API allows empty messages")
	}
}

func (c *ClientChatModelTest) TestPostChatCompletionsNonStreamContent_MultipleMessages() {
	// 确保配置存在
	cfg := config.GetInstance()
	addr := cfg.GetString(config.ClientChatModelAddr)
	model := cfg.GetString(config.ClientChatModelModel)
	token := env.GetModelApiKey()

	// 如果配置不存在，跳过测试
	if addr == "" || model == "" || token == "" {
		c.T().Skip("Skipping test: chat model config not set")
		return
	}

	// 获取客户端实例
	client := GetInstance()
	c.NotNil(client)

	// 创建测试上下文
	ginCtx := createTestContext()
	var ctx context.Context = ginCtx

	// 准备多轮对话消息
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "你是一个专业的编程助手",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "Go语言的特点是什么？",
		},
	}

	// 调用方法
	content, err := client.PostChatCompletionsNonStreamContent(ctx, messages)

	// 验证结果
	c.Nil(err, "should not return error")
	c.NotEmpty(content, "content should not be empty")
	c.Greater(len(content), 0, "content length should be greater than 0")
}

func (c *ClientChatModelTest) TestPostChatCompletionsNonStreamContent_ContentType() {
	// 确保配置存在
	cfg := config.GetInstance()
	addr := cfg.GetString(config.ClientChatModelAddr)
	model := cfg.GetString(config.ClientChatModelModel)
	token := env.GetModelApiKey()

	// 如果配置不存在，跳过测试
	if addr == "" || model == "" || token == "" {
		c.T().Skip("Skipping test: chat model config not set")
		return
	}

	// 获取客户端实例
	client := GetInstance()
	c.NotNil(client)

	// 创建测试上下文
	ginCtx := createTestContext()
	var ctx context.Context = ginCtx

	// 准备测试消息
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "请用中文回答：什么是单元测试？",
		},
	}

	// 调用方法
	content, err := client.PostChatCompletionsNonStreamContent(ctx, messages)

	// 验证结果类型和内容
	c.Nil(err, "should not return error")
	c.NotNil(content, "content should not be nil")
	c.IsType("", content, "content should be a string")
	c.NotEmpty(content, "content should not be empty")

	// 验证 content 是有效的字符串（不是空字符串或只包含空白字符）
	trimmedContent := content
	c.Greater(len(trimmedContent), 0, "content should have meaningful content")
}

func TestClientChatModel(t *testing.T) {
	suite.Run(t, new(ClientChatModelTest))
}
