package llm_model

import (
	"ai_task/config"
	"ai_task/model"
	"ai_task/pkg/clients/httptool"
	"ai_task/pkg/tools"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/wuwie1/go-tools/env"
)

const (
	clientNameChatModel = "chat_model"
)

var (
	streamMessageStart = []byte("data: ")
	streamMessageEnd   = []byte("\n\n")
)

type ClientChatModel struct {
	config *Config
}

type ResponseMsg struct {
	Message string `json:"message"`
}

var (
	instance *ClientChatModel
	once     sync.Once
)

func GetInstance() *ClientChatModel {
	once.Do(func() {
		conf := &Config{
			Addr:        config.GetInstance().GetString(config.ClientChatModelAddr),
			V1Addr:      config.GetInstance().GetString(config.ClientChatModelAddr),
			Model:       config.GetInstance().GetString(config.ClientChatModelModel),
			Token:       env.GetModelApiKey(),
			Temperature: cast.ToFloat32(config.GetInstance().GetFloat64(config.ClientChatModelTemperature)),
			MaxTokens:   config.GetInstance().GetInt(config.ClientChatModelMaxTokens),
		}

		instance = &ClientChatModel{
			config: conf,
		}
	})
	return instance
}

// @Description 封装流式调用
// @Param c context.Context
// @Param message interface{}
// @Success string
// @Success error
func (zc *ClientChatModel) PostChatCompletions(c *context.Context, messages []openai.ChatCompletionMessage) error {
	ginCtx, ok := (*c).(*gin.Context)
	if !ok {
		return model.NewError(model.ErrorParams, nil)
	}

	defaultReq := openai.DefaultConfig(zc.config.Token)
	defaultReq.BaseURL = zc.config.V1Addr

	client := openai.NewClientWithConfig(defaultReq)

	stream, err := client.CreateChatCompletionStream(ginCtx, openai.ChatCompletionRequest{
		Model:       zc.config.Model,
		Messages:    messages,
		MaxTokens:   zc.config.MaxTokens,
		Temperature: zc.config.Temperature,
		Stream:      true,
	})

	if err != nil {
		log.Errorf("%s stream creation error: %v", clientNameChatModel, err)
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
			log.Errorf("%s stream.Recv error: %v", clientNameChatModel, err)
			return false
		}

		if len(response.Choices) > 0 {
			respMsg.Write(streamMessageStart)
			temp, err := json.Marshal(response.Choices)
			if err != nil {
				//c.Writer.WriteHeader(http.StatusInternalServerError)
				//_, _ = c.Writer.Write([]byte("Internal Server Error"))
				log.Errorf("%s: %+v json.Marshal error: %v", clientNameChatModel, response.Choices, err)
				return false
			}

			respMsg.Write(temp)
			respMsg.Write(streamMessageEnd)

			_, err = w.Write(respMsg.Bytes())
			if err != nil {
				log.Errorf("%s: %+v w.Write error: %v", clientNameChatModel, respMsg.String(), err)
				return false
			}
			ginCtx.Writer.Flush()
		}
		return true
	})

	return nil
}

// @Description 封装非流式调用，直接返回完整结果
// @Param c gin.Context
// @Param messages []openai.ChatCompletionMessage
// @Success *openai.ChatCompletionResponse
// @Success error
func (zc *ClientChatModel) PostChatCompletionsNonStream(c context.Context, messages []openai.ChatCompletionMessage) (*openai.ChatCompletionResponse, error) {
	defaultReq := openai.DefaultConfig(zc.config.Token)
	defaultReq.BaseURL = zc.config.V1Addr

	client := openai.NewClientWithConfig(defaultReq)

	// 创建请求结构
	request := openai.ChatCompletionRequest{
		Model:       zc.config.Model,
		Messages:    messages,
		MaxTokens:   zc.config.MaxTokens,
		Temperature: zc.config.Temperature,
		Stream:      false,
	}

	// debug 出完整的请求参数，json格式（仅在 debug 级别时序列化）
	if log.GetLevel() == log.DebugLevel {
		requestJson, err := json.MarshalIndent(request, "", "  ")
		if err != nil {
			log.Errorf("%s chat completion request json marshal error: %v", clientNameChatModel, err)
			return nil, err
		}
		// 直接输出格式化的 JSON 到标准输出，避免日志系统转义换行符
		if _, err := fmt.Fprintf(os.Stdout, "[DEBUG] %s chat completion request:\n%s\n", clientNameChatModel, string(requestJson)); err != nil {
			log.Warnf("%s failed to write debug output: %v", clientNameChatModel, err)
		}
	}

	response, err := client.CreateChatCompletion(c, request)

	if err != nil {
		log.Errorf("%s chat completion error: %v", clientNameChatModel, err)
		return nil, err
	}

	// debug 出完整的响应内容，json格式（仅在 debug 级别时序列化）
	if log.GetLevel() == log.DebugLevel {
		responseJson, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			log.Errorf("%s chat completion response json marshal error: %v", clientNameChatModel, err)
		} else {
			// 直接输出格式化的 JSON 到标准输出，避免日志系统转义换行符
			if _, err := fmt.Fprintf(os.Stdout, "[DEBUG] %s chat completion response:\n%s\n", clientNameChatModel, string(responseJson)); err != nil {
				log.Warnf("%s failed to write debug output: %v", clientNameChatModel, err)
			}
		}
	}

	return &response, nil
}

// @Description 封装非流式调用，只返回响应内容字符串
// @Param c context.Context
// @Param messages []openai.ChatCompletionMessage
// @Success string
// @Success error
func (zc *ClientChatModel) PostChatCompletionsNonStreamContent(c context.Context, messages []openai.ChatCompletionMessage) (string, error) {
	response, err := zc.PostChatCompletionsNonStream(c, messages)
	if err != nil {
		return "", err
	}

	if response == nil {
		log.Errorf("%s chat completion response is nil", clientNameChatModel)
		return "", fmt.Errorf("chat completion response is nil")
	}

	if len(response.Choices) == 0 {
		log.Errorf("%s chat completion response has no choices", clientNameChatModel)
		return "", fmt.Errorf("chat completion response has no choices")
	}

	content := response.Choices[0].Message.Content
	if content == "" {
		log.Warnf("%s chat completion response content is empty", clientNameChatModel)
	}

	return content, nil
}
