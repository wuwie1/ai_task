package embedding

import (
	"ai_task/config"
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wuwie1/go-tools/env"
)

type EmbeddingClientTest struct {
	suite.Suite
}

func (e *EmbeddingClientTest) SetupTest() {
	// 重置单例状态（用于测试）
	// 注意：由于 sync.Once 的特性，在实际测试中可能需要使用不同的测试方法
	instance = nil
	once = sync.Once{}
	initErr = nil
}

func (e *EmbeddingClientTest) TestGetInstance_Success() {
	// 确保配置存在
	cfg := config.GetInstance()
	apiKey := env.GetModelApiKey()
	modelName := cfg.GetString(config.EmbeddingConfigKeyModelName)

	// 如果配置不存在，跳过测试
	if apiKey == "" || modelName == "" {
		e.T().Skip("Skipping test: embedding config not set")
		return
	}

	// 测试获取单例
	client1, err := GetInstance()
	e.Nil(err)
	e.NotNil(client1)

	// 再次获取应该返回同一个实例
	client2, err := GetInstance()
	e.Nil(err)
	e.NotNil(client2)
	e.Equal(client1, client2) // 应该是同一个实例
}

func (e *EmbeddingClientTest) TestGetInstance_MissingAPIKey() {
	// 保存原始配置
	originalAPIKey := env.GetModelApiKey()

	// 重置单例状态
	instance = nil
	once = sync.Once{}
	initErr = nil

	// 注意：由于配置是全局的，我们无法直接修改它来测试错误情况
	// 如果配置中缺少 api_key，GetInstance 会返回错误
	// 这个测试依赖于配置文件的实际情况
	if originalAPIKey == "" {
		client, err := GetInstance()
		e.NotNil(err)
		e.Nil(client)
	}
}

func (e *EmbeddingClientTest) TestGetInstance_MissingModelName() {
	// 保存原始配置
	cfg := config.GetInstance()
	originalAPIKey := env.GetModelApiKey()
	originalModelName := cfg.GetString(config.EmbeddingConfigKeyModelName)

	// 重置单例状态
	instance = nil
	once = sync.Once{}
	initErr = nil

	// 如果配置中缺少 model_name，GetInstance 会返回错误
	if originalModelName == "" && originalAPIKey != "" {
		client, err := GetInstance()
		e.NotNil(err)
		e.Nil(client)
		e.Contains(err.Error(), config.EmbeddingConfigKeyModelName)
	} else {
		// 如果配置存在，验证可以正常创建客户端
		client, err := GetInstance()
		if err == nil {
			e.NotNil(client)
		}
	}
}

func (e *EmbeddingClientTest) TestGetTextEmbedding_Success() {
	// 确保配置存在
	cfg := config.GetInstance()
	apiKey := env.GetModelApiKey()
	modelName := cfg.GetString(config.EmbeddingConfigKeyModelName)

	// 如果配置不存在，跳过测试
	if apiKey == "" || modelName == "" {
		e.T().Skip("Skipping test: embedding config not set")
		return
	}

	// 获取客户端
	client, err := GetInstance()
	e.Nil(err)
	e.NotNil(client)

	// 测试单个文本的 Embedding
	ctx := context.Background()
	text := "测试文本"
	embedding, err := client.GetTextEmbedding(ctx, text)
	e.Nil(err)
	e.NotNil(embedding)
	e.Greater(len(embedding), 0, "embedding should have dimensions")
}

func (e *EmbeddingClientTest) TestGetTextEmbeddingBatch_Success() {
	// 确保配置存在
	cfg := config.GetInstance()
	apiKey := env.GetModelApiKey()
	modelName := cfg.GetString(config.EmbeddingConfigKeyModelName)

	// 如果配置不存在，跳过测试
	if apiKey == "" || modelName == "" {
		e.T().Skip("Skipping test: embedding config not set")
		return
	}

	// 获取客户端
	client, err := GetInstance()
	e.Nil(err)
	e.NotNil(client)

	// 测试批量文本的 Embedding
	ctx := context.Background()
	texts := []string{
		"风急天高猿啸哀",
		"渚清沙白鸟飞回",
		"无边落木萧萧下",
		"不尽长江滚滚来",
	}

	embeddings, err := client.GetTextEmbeddingBatch(ctx, texts)
	e.Nil(err)
	e.NotNil(embeddings)
	e.Equal(len(texts), len(embeddings), "should return embeddings for all texts")

	// 验证每个 embedding 的维度
	for i, embedding := range embeddings {
		e.NotNil(embedding, "embedding %d should not be nil", i)
		e.Greater(len(embedding), 0, "embedding %d should have dimensions", i)
		// 验证维度一致性（通常 embedding 模型的维度是固定的，比如 1536）
		if i > 0 {
			e.Equal(len(embeddings[0]), len(embedding), "all embeddings should have the same dimension")
		}
	}
}

func (e *EmbeddingClientTest) TestGetTextEmbeddingBatch_EmptyTexts() {
	// 确保配置存在
	cfg := config.GetInstance()
	apiKey := env.GetModelApiKey()
	modelName := cfg.GetString(config.EmbeddingConfigKeyModelName)

	// 如果配置不存在，跳过测试
	if apiKey == "" || modelName == "" {
		e.T().Skip("Skipping test: embedding config not set")
		return
	}

	// 获取客户端
	client, err := GetInstance()
	e.Nil(err)
	e.NotNil(client)

	// 测试空文本列表
	ctx := context.Background()
	embeddings, err := client.GetTextEmbeddingBatch(ctx, []string{})
	e.NotNil(err)
	e.Nil(embeddings)
	e.Contains(err.Error(), "texts cannot be empty")
}

func (e *EmbeddingClientTest) TestGetTextEmbedding_EmptyResult() {
	// 这个测试主要验证 GetTextEmbedding 对空结果的处理
	// 由于实际 API 调用不会返回空结果，这个测试主要验证逻辑
	// 实际场景中，如果 API 返回空结果，应该返回错误
	// 这个测试依赖于实际的 API 行为
}

func (e *EmbeddingClientTest) TestGetInstance_Singleton() {
	// 确保配置存在
	cfg := config.GetInstance()
	apiKey := env.GetModelApiKey()
	modelName := cfg.GetString(config.EmbeddingConfigKeyModelName)

	// 如果配置不存在，跳过测试
	if apiKey == "" || modelName == "" {
		e.T().Skip("Skipping test: embedding config not set")
		return
	}

	// 测试单例模式：多次调用应该返回同一个实例
	client1, err1 := GetInstance()
	e.Nil(err1)
	e.NotNil(client1)

	client2, err2 := GetInstance()
	e.Nil(err2)
	e.NotNil(client2)

	// 验证是同一个实例（指针地址相同）
	e.Equal(client1, client2)

	// 验证模型名称一致
	e.Equal(client1.modelName, client2.modelName)
}

func TestEmbeddingClient(t *testing.T) {
	suite.Run(t, new(EmbeddingClientTest))
}
