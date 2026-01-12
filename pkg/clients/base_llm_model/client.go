package base_llm_model

import (
	"ai_task/model"
	"ai_task/pkg/clients/httptool"
	"ai_task/pkg/tools"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

const (
	clientNameBaseLLM = "base_llm_model"
)

var (
	streamMessageStart = []byte("data: ")
	streamMessageEnd   = []byte("\n\n")
)

// Client 基础LLM模型客户端
type Client struct {
	config *Config
	client *openai.Client
}

// NewClient 创建新的LLM客户端
// 必须传入 baseURL, apiKey, modelName 三个参数
func NewClient(baseURL, apiKey, modelName string, opts ...Option) *Client {
	params := ClientParams{
		BaseURL:   baseURL,
		APIKey:    apiKey,
		ModelName: modelName,
	}
	return NewClientWithParams(params, opts...)
}

// NewClientWithParams 使用参数结构体创建新的LLM客户端
// params 包含必填的 BaseURL, APIKey, ModelName
func NewClientWithParams(params ClientParams, opts ...Option) *Client {
	config := DefaultConfig()
	config.BaseURL = params.BaseURL
	config.APIKey = params.APIKey
	config.ModelName = params.ModelName

	// 应用可选配置
	for _, opt := range opts {
		opt(config)
	}

	// 创建OpenAI客户端配置
	clientConfig := openai.DefaultConfig(config.APIKey)
	clientConfig.BaseURL = config.BaseURL

	return &Client{
		config: config,
		client: openai.NewClientWithConfig(clientConfig),
	}
}

// NewClientWithConfig 使用完整配置创建客户端
func NewClientWithConfig(config *Config) *Client {
	clientConfig := openai.DefaultConfig(config.APIKey)
	clientConfig.BaseURL = config.BaseURL

	return &Client{
		config: config,
		client: openai.NewClientWithConfig(clientConfig),
	}
}

// GetConfig 获取当前配置
func (c *Client) GetConfig() *Config {
	return c.config
}

// PostChatCompletions 流式调用，将响应流式写入gin.Context
func (c *Client) PostChatCompletions(ctx *context.Context, messages []openai.ChatCompletionMessage) error {
	ginCtx, ok := (*ctx).(*gin.Context)
	if !ok {
		return model.NewError(model.ErrorParams, nil)
	}

	stream, err := c.client.CreateChatCompletionStream(ginCtx, openai.ChatCompletionRequest{
		Model:       c.config.ModelName,
		Messages:    messages,
		MaxTokens:   c.config.MaxTokens,
		Temperature: c.config.Temperature,
		Stream:      true,
	})

	if err != nil {
		log.Errorf("%s stream creation error: %v", clientNameBaseLLM, err)
		return err
	}

	ginCtx.Writer.Header().Set(httptool.HeaderContentType, httptool.HeaderContentTypeStream)
	ginCtx.Writer.Header().Set(httptool.HeaderContentCache, httptool.HeaderContentCacheNo)
	ginCtx.Writer.Header().Set(httptool.HeaderContentConnection, httptool.HeaderContentKeepAlive)
	ginCtx.Writer.Header().Set(httptool.HeaderContentTransfer, httptool.HeaderContentChunked)

	ginCtx.Writer.Flush()

	defer tools.ErrorWithPrintContext(stream.Close, "close stream")

	ginCtx.Stream(func(w io.Writer) bool {
		var respMsg bytes.Buffer

		response, err := stream.Recv()
		if err == io.EOF {
			return false
		}
		if err != nil {
			log.Errorf("%s stream.Recv error: %v", clientNameBaseLLM, err)
			return false
		}

		if len(response.Choices) > 0 {
			respMsg.Write(streamMessageStart)
			temp, err := json.Marshal(response.Choices)
			if err != nil {
				log.Errorf("%s: %+v json.Marshal error: %v", clientNameBaseLLM, response.Choices, err)
				return false
			}

			respMsg.Write(temp)
			respMsg.Write(streamMessageEnd)

			_, err = w.Write(respMsg.Bytes())
			if err != nil {
				log.Errorf("%s: %+v w.Write error: %v", clientNameBaseLLM, respMsg.String(), err)
				return false
			}
			ginCtx.Writer.Flush()
		}
		return true
	})

	return nil
}

// PostChatCompletionsNonStream 非流式调用，返回完整响应
func (c *Client) PostChatCompletionsNonStream(ctx context.Context, messages []openai.ChatCompletionMessage) (*openai.ChatCompletionResponse, error) {
	request := openai.ChatCompletionRequest{
		Model:       c.config.ModelName,
		Messages:    messages,
		MaxTokens:   c.config.MaxTokens,
		Temperature: c.config.Temperature,
		Stream:      false,
	}

	// debug 出完整的请求参数，json格式（仅在 debug 级别时序列化）
	if log.GetLevel() == log.DebugLevel {
		requestJson, err := json.MarshalIndent(request, "", "  ")
		if err != nil {
			log.Errorf("%s chat completion request json marshal error: %v", clientNameBaseLLM, err)
			return nil, err
		}
		if _, err := fmt.Fprintf(os.Stdout, "[DEBUG] %s chat completion request:\n%s\n", clientNameBaseLLM, string(requestJson)); err != nil {
			log.Warnf("%s failed to write debug output: %v", clientNameBaseLLM, err)
		}
	}

	response, err := c.client.CreateChatCompletion(ctx, request)

	if err != nil {
		log.Errorf("%s chat completion error: %v", clientNameBaseLLM, err)
		return nil, err
	}

	// debug 出完整的响应内容，json格式（仅在 debug 级别时序列化）
	if log.GetLevel() == log.DebugLevel {
		responseJson, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			log.Errorf("%s chat completion response json marshal error: %v", clientNameBaseLLM, err)
		} else {
			if _, err := fmt.Fprintf(os.Stdout, "[DEBUG] %s chat completion response:\n%s\n", clientNameBaseLLM, string(responseJson)); err != nil {
				log.Warnf("%s failed to write debug output: %v", clientNameBaseLLM, err)
			}
		}
	}

	return &response, nil
}

// PostChatCompletionsNonStreamContent 非流式调用，只返回响应内容字符串
func (c *Client) PostChatCompletionsNonStreamContent(ctx context.Context, messages []openai.ChatCompletionMessage) (string, error) {
	response, err := c.PostChatCompletionsNonStream(ctx, messages)
	if err != nil {
		return "", err
	}

	if response == nil {
		log.Errorf("%s chat completion response is nil", clientNameBaseLLM)
		return "", fmt.Errorf("chat completion response is nil")
	}

	if len(response.Choices) == 0 {
		log.Errorf("%s chat completion response has no choices", clientNameBaseLLM)
		return "", fmt.Errorf("chat completion response has no choices")
	}

	content := response.Choices[0].Message.Content
	if content == "" {
		log.Warnf("%s chat completion response content is empty", clientNameBaseLLM)
	}

	return content, nil
}

// ChatWithSystemPrompt 使用系统提示词进行对话的便捷方法
func (c *Client) ChatWithSystemPrompt(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userMessage,
		},
	}
	return c.PostChatCompletionsNonStreamContent(ctx, messages)
}

// Chat 简单对话的便捷方法
func (c *Client) Chat(ctx context.Context, userMessage string) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userMessage,
		},
	}
	return c.PostChatCompletionsNonStreamContent(ctx, messages)
}
