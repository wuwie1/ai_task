package embedding

import (
	"ai_web/test/config"
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	log "github.com/sirupsen/logrus"
	"github.com/wuwie1/go-tools/env"
)

const (
	// MaxBatchSize 每批最多处理的数量
	MaxBatchSize = 64
	// MaxRetries 最大重试次数
	MaxRetries = 3
	// LRUCacheCapacity LRU 缓存容量
	LRUCacheCapacity = 5000
)

var (
	instance *Client
	once     sync.Once
	initErr  error
)

// Client Embedding 客户端
type Client struct {
	client    openai.Client
	modelName string
	cache     *LRUCache // embedding 缓存
	metrics   *Metrics  // 指标统计
}

// Metrics 指标统计
type Metrics struct {
	IngestCount      int64         // ingest 条数
	QueryCount       int64         // query 次数
	EmbeddingLatency time.Duration // embedding 总耗时
	mu               sync.Mutex
}

// LRUCache LRU 缓存实现
type LRUCache struct {
	capacity int
	cache    map[string]*CacheNode
	head     *CacheNode
	tail     *CacheNode
	mu       sync.RWMutex
}

// CacheNode 缓存节点
type CacheNode struct {
	key   string
	value []float64
	prev  *CacheNode
	next  *CacheNode
}

// NewLRUCache 创建新的 LRU 缓存
func NewLRUCache(capacity int) *LRUCache {
	if capacity <= 0 {
		capacity = LRUCacheCapacity
	}
	head := &CacheNode{}
	tail := &CacheNode{}
	head.next = tail
	tail.prev = head
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*CacheNode),
		head:     head,
		tail:     tail,
	}
}

// Get 从缓存获取
func (lru *LRUCache) Get(key string) ([]float64, bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	node, ok := lru.cache[key]
	if !ok {
		return nil, false
	}

	// 移动到头部
	lru.moveToHead(node)
	return node.value, true
}

// Put 放入缓存
func (lru *LRUCache) Put(key string, value []float64) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if node, ok := lru.cache[key]; ok {
		// 更新现有节点
		node.value = value
		lru.moveToHead(node)
		return
	}

	// 创建新节点
	node := &CacheNode{
		key:   key,
		value: value,
	}
	lru.cache[key] = node
	lru.addToHead(node)

	// 如果超过容量，删除尾部节点
	if len(lru.cache) > lru.capacity {
		lru.removeTail()
	}
}

func (lru *LRUCache) addToHead(node *CacheNode) {
	node.prev = lru.head
	node.next = lru.head.next
	lru.head.next.prev = node
	lru.head.next = node
}

func (lru *LRUCache) removeNode(node *CacheNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

func (lru *LRUCache) moveToHead(node *CacheNode) {
	lru.removeNode(node)
	lru.addToHead(node)
}

func (lru *LRUCache) removeTail() {
	node := lru.tail.prev
	lru.removeNode(node)
	delete(lru.cache, node.key)
}

// GetInstance 获取 Embedding 客户端单例
func GetInstance() (*Client, error) {
	once.Do(func() {
		cfg := config.GetInstance()

		apiKey := env.GetModelApiKey()
		if apiKey == "" {
			initErr = fmt.Errorf("%s is required", env.GetModelApiKey())
			return
		}

		modelName := cfg.GetString(config.EmbeddingConfigKeyModelName)
		if modelName == "" {
			initErr = fmt.Errorf("%s is required", config.EmbeddingConfigKeyModelName)
			return
		}

		baseURL := cfg.GetString(config.EmbeddingConfigKeyBaseURL)

		// 创建 OpenAI 客户端
		opts := []option.RequestOption{
			option.WithAPIKey(apiKey),
		}

		// 如果配置了 base_url，则使用自定义的 base_url（用于兼容其他兼容 OpenAI API 的服务）
		if baseURL != "" {
			opts = append(opts, option.WithBaseURL(baseURL))
		}

		client := openai.NewClient(opts...)

		instance = &Client{
			client:    client,
			modelName: modelName,
			cache:     NewLRUCache(LRUCacheCapacity),
			metrics:   &Metrics{},
		}
	})

	return instance, initErr
}

// GetTextEmbedding 获取单个文本的 Embedding 向量（带缓存）
func (c *Client) GetTextEmbedding(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := c.GetTextEmbeddingBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return embeddings[0], nil
}

// GetTextEmbeddingBatch 批量获取文本的 Embedding 向量（带批量切分、重试和缓存）
func (c *Client) GetTextEmbeddingBatch(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	// 更新查询计数
	c.metrics.mu.Lock()
	c.metrics.QueryCount++
	c.metrics.mu.Unlock()

	startTime := time.Now()
	defer func() {
		c.metrics.mu.Lock()
		c.metrics.EmbeddingLatency += time.Since(startTime)
		c.metrics.mu.Unlock()
	}()

	// 检查缓存并收集需要请求的文本
	type textWithIndex struct {
		text  string
		index int
	}
	needRequest := make([]textWithIndex, 0)
	result := make([][]float64, len(texts))
	cacheHits := 0

	for i, text := range texts {
		if cached, ok := c.cache.Get(text); ok {
			result[i] = cached
			cacheHits++
		} else {
			needRequest = append(needRequest, textWithIndex{text: text, index: i})
		}
	}

	if len(needRequest) == 0 {
		log.Debugf("All embeddings retrieved from cache (count: %d)", len(texts))
		return result, nil
	}

	// 批量切分处理
	allEmbeddings := make([][]float64, len(texts))
	for i := 0; i < len(needRequest); i += MaxBatchSize {
		end := i + MaxBatchSize
		if end > len(needRequest) {
			end = len(needRequest)
		}

		batch := needRequest[i:end]
		batchTexts := make([]string, len(batch))
		for j, item := range batch {
			batchTexts[j] = item.text
		}

		// 带重试的批量请求
		embeddings, err := c.getTextEmbeddingBatchWithRetry(ctx, batchTexts)
		if err != nil {
			return nil, fmt.Errorf("failed to get embeddings for batch %d-%d: %w", i, end, err)
		}

		// 填充结果并更新缓存
		for j, item := range batch {
			if j < len(embeddings) {
				allEmbeddings[item.index] = embeddings[j]
				c.cache.Put(item.text, embeddings[j])
			}
		}
	}

	// 合并缓存结果和新请求结果
	for i := range texts {
		if result[i] == nil {
			result[i] = allEmbeddings[i]
		}
	}

	log.Debugf("Embedding batch completed: total=%d, cache_hits=%d, requests=%d",
		len(texts), cacheHits, len(needRequest))

	// 更新 ingest 计数
	c.metrics.mu.Lock()
	c.metrics.IngestCount += int64(len(needRequest))
	c.metrics.mu.Unlock()

	return result, nil
}

// getTextEmbeddingBatchWithRetry 带重试机制的批量获取 Embedding
func (c *Client) getTextEmbeddingBatchWithRetry(ctx context.Context, texts []string) ([][]float64, error) {
	var lastErr error

	for attempt := 0; attempt < MaxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避：1s, 2s, 4s
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			log.Warnf("Retrying embedding request (attempt %d/%d) after %v", attempt+1, MaxRetries, backoff)
			time.Sleep(backoff)
		}

		embeddings, err := c.getTextEmbeddingBatchOnce(ctx, texts)
		if err == nil {
			return embeddings, nil
		}

		lastErr = err
		log.Errorf("Embedding request failed (attempt %d/%d): %v", attempt+1, MaxRetries, err)
	}

	return nil, fmt.Errorf("failed after %d retries: %w", MaxRetries, lastErr)
}

// getTextEmbeddingBatchOnce 单次批量获取 Embedding（不重试）
func (c *Client) getTextEmbeddingBatchOnce(ctx context.Context, texts []string) ([][]float64, error) {
	// 构建输入参数
	input := openai.EmbeddingNewParamsInputUnion{
		OfArrayOfStrings: texts,
	}

	// 调用 OpenAI Embeddings API
	resp, err := c.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(c.modelName),
		Input: input,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	// 提取 embedding 向量（注意：OpenAI API 返回的是 []float64）
	result := make([][]float64, 0, len(resp.Data))
	for _, item := range resp.Data {
		result = append(result, item.Embedding)
	}

	return result, nil
}

// GetMetrics 获取指标统计
func (c *Client) GetMetrics() Metrics {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()
	return Metrics{
		IngestCount:      c.metrics.IngestCount,
		QueryCount:       c.metrics.QueryCount,
		EmbeddingLatency: c.metrics.EmbeddingLatency,
	}
}
