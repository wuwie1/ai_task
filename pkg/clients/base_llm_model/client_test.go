package base_llm_model

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/suite"
)

// TestClientParams 测试用的客户端参数，从环境变量获取
type TestClientParams struct {
	BaseURL   string
	APIKey    string
	ModelName string
}

// 从环境变量获取测试参数
func getTestParams() TestClientParams {
	return TestClientParams{
		BaseURL:   os.Getenv("MODEL_BASE_GLM_URL"),
		APIKey:    os.Getenv("LLM_GML_API_KEY"),
		ModelName: os.Getenv("LLM_GML_MODEL"),
	}
}

// isTestParamsValid 检查测试参数是否有效
func isTestParamsValid(params TestClientParams) bool {
	return params.BaseURL != "" && params.APIKey != "" && params.ModelName != ""
}

type BaseLLMClientTest struct {
	suite.Suite
	testParams TestClientParams
}

func (s *BaseLLMClientTest) SetupTest() {
	s.testParams = getTestParams()
}

// createTestContext 创建测试用的 gin.Context
func createTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/test", nil)
	return ctx
}

// TestNewClientWithParams 测试使用结构体参数创建客户端
func (s *BaseLLMClientTest) TestNewClientWithParams() {
	params := ClientParams{
		BaseURL:   "https://api.example.com/v1",
		APIKey:    "test-api-key",
		ModelName: "test-model",
	}

	client := NewClientWithParams(params)

	s.NotNil(client)
	s.NotNil(client.config)
	s.Equal(params.BaseURL, client.config.BaseURL)
	s.Equal(params.APIKey, client.config.APIKey)
	s.Equal(params.ModelName, client.config.ModelName)
	// 验证默认值
	s.Equal(float32(0.7), client.config.Temperature)
	s.Equal(4096, client.config.MaxTokens)
}

// TestNewClientWithParamsAndOptions 测试使用结构体参数和选项创建客户端
func (s *BaseLLMClientTest) TestNewClientWithParamsAndOptions() {
	params := ClientParams{
		BaseURL:   "https://api.example.com/v1",
		APIKey:    "test-api-key",
		ModelName: "test-model",
	}

	client := NewClientWithParams(params,
		WithTemperature(0.5),
		WithMaxTokens(8192),
	)

	s.NotNil(client)
	s.NotNil(client.config)
	s.Equal(params.BaseURL, client.config.BaseURL)
	s.Equal(params.APIKey, client.config.APIKey)
	s.Equal(params.ModelName, client.config.ModelName)
	s.Equal(float32(0.5), client.config.Temperature)
	s.Equal(8192, client.config.MaxTokens)
}

// TestNewClient 测试使用单独参数创建客户端
func (s *BaseLLMClientTest) TestNewClient() {
	client := NewClient(
		"https://api.example.com/v1",
		"test-api-key",
		"test-model",
	)

	s.NotNil(client)
	s.NotNil(client.config)
	s.Equal("https://api.example.com/v1", client.config.BaseURL)
	s.Equal("test-api-key", client.config.APIKey)
	s.Equal("test-model", client.config.ModelName)
}

// TestNewClientWithOptions 测试使用单独参数和选项创建客户端
func (s *BaseLLMClientTest) TestNewClientWithOptions() {
	client := NewClient(
		"https://api.example.com/v1",
		"test-api-key",
		"test-model",
		WithTemperature(0.3),
		WithMaxTokens(2048),
	)

	s.NotNil(client)
	s.Equal(float32(0.3), client.config.Temperature)
	s.Equal(2048, client.config.MaxTokens)
}

// TestNewClientWithConfig 测试使用完整配置创建客户端
func (s *BaseLLMClientTest) TestNewClientWithConfig() {
	config := &Config{
		BaseURL:     "https://api.example.com/v1",
		APIKey:      "test-api-key",
		ModelName:   "test-model",
		Temperature: 0.8,
		MaxTokens:   1024,
	}

	client := NewClientWithConfig(config)

	s.NotNil(client)
	s.Equal(config.BaseURL, client.config.BaseURL)
	s.Equal(config.APIKey, client.config.APIKey)
	s.Equal(config.ModelName, client.config.ModelName)
	s.Equal(config.Temperature, client.config.Temperature)
	s.Equal(config.MaxTokens, client.config.MaxTokens)
}

// TestGetConfig 测试获取配置
func (s *BaseLLMClientTest) TestGetConfig() {
	params := ClientParams{
		BaseURL:   "https://api.example.com/v1",
		APIKey:    "test-api-key",
		ModelName: "test-model",
	}

	client := NewClientWithParams(params)
	config := client.GetConfig()

	s.NotNil(config)
	s.Equal(params.BaseURL, config.BaseURL)
	s.Equal(params.APIKey, config.APIKey)
	s.Equal(params.ModelName, config.ModelName)
}

// TestDefaultConfig 测试默认配置
func (s *BaseLLMClientTest) TestDefaultConfig() {
	config := DefaultConfig()

	s.NotNil(config)
	s.Equal(float32(0.7), config.Temperature)
	s.Equal(4096, config.MaxTokens)
	s.Empty(config.BaseURL)
	s.Empty(config.APIKey)
	s.Empty(config.ModelName)
}

// TestOptions 测试各个选项函数
func (s *BaseLLMClientTest) TestOptions() {
	config := DefaultConfig()

	// 测试 WithBaseURL
	WithBaseURL("https://test.com")(config)
	s.Equal("https://test.com", config.BaseURL)

	// 测试 WithAPIKey
	WithAPIKey("test-key")(config)
	s.Equal("test-key", config.APIKey)

	// 测试 WithModelName
	WithModelName("test-model")(config)
	s.Equal("test-model", config.ModelName)

	// 测试 WithTemperature
	WithTemperature(0.9)(config)
	s.Equal(float32(0.9), config.Temperature)

	// 测试 WithMaxTokens
	WithMaxTokens(2000)(config)
	s.Equal(2000, config.MaxTokens)
}

// TestPostChatCompletionsNonStream_Success 测试非流式调用成功
func (s *BaseLLMClientTest) TestPostChatCompletionsNonStream_Success() {
	if !isTestParamsValid(s.testParams) {
		s.T().Skip("Skipping test: TEST_LLM_BASE_URL, TEST_LLM_API_KEY, TEST_LLM_MODEL_NAME env vars not set")
		return
	}

	params := ClientParams{
		BaseURL:   s.testParams.BaseURL,
		APIKey:    s.testParams.APIKey,
		ModelName: s.testParams.ModelName,
	}

	client := NewClientWithParams(params)
	s.NotNil(client)

	ctx := context.Background()
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "你好，请用一句话介绍你自己",
		},
	}

	response, err := client.PostChatCompletionsNonStream(ctx, messages)

	s.Nil(err, "should not return error")
	s.NotNil(response, "response should not be nil")
	if response != nil {
		s.Greater(len(response.Choices), 0, "should have at least one choice")
		if len(response.Choices) > 0 {
			fmt.Println(response.Choices[0].Message.Content)
			s.NotEmpty(response.Choices[0].Message.Content, "message content should not be empty")
		}
	}
}

// TestPostChatCompletionsNonStreamContent_Success 测试非流式调用获取内容
func (s *BaseLLMClientTest) TestPostChatCompletionsNonStreamContent_Success() {
	if !isTestParamsValid(s.testParams) {
		s.T().Skip("Skipping test: TEST_LLM_BASE_URL, TEST_LLM_API_KEY, TEST_LLM_MODEL_NAME env vars not set")
		return
	}

	params := ClientParams{
		BaseURL:   s.testParams.BaseURL,
		APIKey:    s.testParams.APIKey,
		ModelName: s.testParams.ModelName,
	}

	client := NewClientWithParams(params)
	s.NotNil(client)

	ctx := context.Background()
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "请回答：1+1等于几？",
		},
	}

	content, err := client.PostChatCompletionsNonStreamContent(ctx, messages)

	s.Nil(err, "should not return error")
	s.NotEmpty(content, "content should not be empty")
}

// TestChat_Success 测试简单对话方法
func (s *BaseLLMClientTest) TestChat_Success() {
	if !isTestParamsValid(s.testParams) {
		s.T().Skip("Skipping test: TEST_LLM_BASE_URL, TEST_LLM_API_KEY, TEST_LLM_MODEL_NAME env vars not set")
		return
	}

	params := ClientParams{
		BaseURL:   s.testParams.BaseURL,
		APIKey:    s.testParams.APIKey,
		ModelName: s.testParams.ModelName,
	}

	client := NewClientWithParams(params)
	s.NotNil(client)

	ctx := context.Background()
	content, err := client.Chat(ctx, "你好")

	s.Nil(err, "should not return error")
	s.NotEmpty(content, "content should not be empty")
}

// TestChatWithSystemPrompt_Success 测试带系统提示词的对话方法
func (s *BaseLLMClientTest) TestChatWithSystemPrompt_Success() {
	if !isTestParamsValid(s.testParams) {
		s.T().Skip("Skipping test: TEST_LLM_BASE_URL, TEST_LLM_API_KEY, TEST_LLM_MODEL_NAME env vars not set")
		return
	}

	params := ClientParams{
		BaseURL:   s.testParams.BaseURL,
		APIKey:    s.testParams.APIKey,
		ModelName: s.testParams.ModelName,
	}

	client := NewClientWithParams(params)
	s.NotNil(client)

	ctx := context.Background()
	content, err := client.ChatWithSystemPrompt(ctx, "你是一个数学助手", "1+1等于多少？")

	s.Nil(err, "should not return error")
	s.NotEmpty(content, "content should not be empty")
}

// TestMultipleMessages_Success 测试多轮对话
func (s *BaseLLMClientTest) TestMultipleMessages_Success() {
	if !isTestParamsValid(s.testParams) {
		s.T().Skip("Skipping test: TEST_LLM_BASE_URL, TEST_LLM_API_KEY, TEST_LLM_MODEL_NAME env vars not set")
		return
	}

	params := ClientParams{
		BaseURL:   s.testParams.BaseURL,
		APIKey:    s.testParams.APIKey,
		ModelName: s.testParams.ModelName,
	}

	client := NewClientWithParams(params)
	s.NotNil(client)

	ctx := context.Background()
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "你是一个专业的编程助手",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "Go语言的特点是什么？",
		},
		{
			Role:    openai.ChatMessageRoleAssistant,
			Content: "Go语言的主要特点包括简洁、高效、并发支持好等。",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "请举一个并发的例子",
		},
	}

	response, err := client.PostChatCompletionsNonStream(ctx, messages)

	s.Nil(err, "should not return error")
	s.NotNil(response, "response should not be nil")
	if response != nil {
		s.Greater(len(response.Choices), 0, "should have at least one choice")
	}
}

// TestDifferentModels 测试使用不同模型参数创建多个客户端
func (s *BaseLLMClientTest) TestDifferentModels() {
	// 创建使用不同参数的客户端
	client1 := NewClientWithParams(ClientParams{
		BaseURL:   "https://api.openai.com/v1",
		APIKey:    "key1",
		ModelName: "gpt-4",
	})

	client2 := NewClientWithParams(ClientParams{
		BaseURL:   "https://api.deepseek.com/v1",
		APIKey:    "key2",
		ModelName: "deepseek-chat",
	})

	client3 := NewClientWithParams(ClientParams{
		BaseURL:   "https://api.anthropic.com/v1",
		APIKey:    "key3",
		ModelName: "claude-3",
	})

	// 验证每个客户端配置独立
	s.Equal("https://api.openai.com/v1", client1.config.BaseURL)
	s.Equal("gpt-4", client1.config.ModelName)

	s.Equal("https://api.deepseek.com/v1", client2.config.BaseURL)
	s.Equal("deepseek-chat", client2.config.ModelName)

	s.Equal("https://api.anthropic.com/v1", client3.config.BaseURL)
	s.Equal("claude-3", client3.config.ModelName)
}

// TestEmptyMessages 测试空消息列表
func (s *BaseLLMClientTest) TestEmptyMessages() {
	if !isTestParamsValid(s.testParams) {
		s.T().Skip("Skipping test: TEST_LLM_BASE_URL, TEST_LLM_API_KEY, TEST_LLM_MODEL_NAME env vars not set")
		return
	}

	params := ClientParams{
		BaseURL:   s.testParams.BaseURL,
		APIKey:    s.testParams.APIKey,
		ModelName: s.testParams.ModelName,
	}

	client := NewClientWithParams(params)
	ctx := context.Background()
	messages := []openai.ChatCompletionMessage{}

	_, err := client.PostChatCompletionsNonStream(ctx, messages)

	// 空消息列表应该返回错误
	s.NotNil(err, "should return error for empty messages")
}

// TestInvalidAPIKey 测试无效的API密钥
func (s *BaseLLMClientTest) TestInvalidAPIKey() {
	if s.testParams.BaseURL == "" {
		s.T().Skip("Skipping test: TEST_LLM_BASE_URL env var not set")
		return
	}

	params := ClientParams{
		BaseURL:   s.testParams.BaseURL,
		APIKey:    "invalid-api-key",
		ModelName: "test-model",
	}

	client := NewClientWithParams(params)
	ctx := context.Background()
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "测试",
		},
	}

	_, err := client.PostChatCompletionsNonStream(ctx, messages)

	// 无效的API密钥应该返回错误
	s.NotNil(err, "should return error for invalid API key")
}

func TestBaseLLMClient(t *testing.T) {
	suite.Run(t, new(BaseLLMClientTest))
}
