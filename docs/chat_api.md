# 智能体聊天 API 文档

## 概述

本 API 提供了支持三层记忆结构的智能体聊天功能：
- **短期记忆（Session Memory）**：会话历史，最近 N 轮对话
- **长期记忆（Long-term Memory）**：用户偏好、配置等结构化信息
- **语义记忆（Semantic Memory）**：基于向量检索的相关历史对话片段

## API 端点

### POST /api/v1/chat

聊天接口，支持流式和非流式返回。

#### 请求体

```json
{
  "user_id": "user123",
  "session_id": "session456",
  "messages": [
    {
      "role": "user",
      "content": "你好，我是张三"
    }
  ],
  "stream": false,
  "session_memory_limit": 10,
  "semantic_memory_limit": 5,
  "semantic_threshold": 0.7
}
```

#### 请求参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | string | 是 | 用户 ID |
| session_id | string | 是 | 会话 ID |
| messages | array | 是 | 消息列表，格式同 OpenAI Chat API |
| stream | boolean | 否 | 是否流式返回，默认 false |
| session_memory_limit | int | 否 | 短期记忆条数，默认 10 |
| semantic_memory_limit | int | 否 | 语义记忆条数，默认 5 |
| semantic_threshold | float | 否 | 语义相似度阈值，默认 0.7 |
| compress_threshold | int | 否 | 压缩阈值（超过此数量时压缩旧记忆），默认 20 |
| enable_summary | boolean | 否 | 是否启用摘要功能，默认 false |
| enable_auto_extract | boolean | 否 | 是否自动提取关键事实到长期记忆，默认 false |
| chunk_max_size | int | 否 | 最大块大小（字符数），默认 1000。超过此大小的消息会被智能分块 |
| chunk_overlap | int | 否 | 重叠窗口大小（字符数），默认 100。相邻块之间的重叠字符数，用于保持上下文连续性 |
| chunk_min_size | int | 否 | 最小块大小（字符数），默认 200。小于此大小的块不会被创建 |
| chunk_strategy | string | 否 | 分块策略，可选值："paragraph"（按段落，默认）、"sentence"（按句子）、"fixed"（固定长度）。策略会按优先级降级使用 |

#### 响应（非流式）

```json
{
  "message": "你好，张三！很高兴认识你。",
  "session_id": "session456"
}
```

#### 响应（流式）

流式返回格式同 OpenAI Chat API 的流式响应。

## 记忆系统说明

### 短期记忆（Session Memory）

- **存储内容**：每轮对话的 user/assistant 消息
- **存储位置**：PostgreSQL `chat_memory_chunks` 表
- **检索方式**：按 `user_id` 和 `session_id` 获取最近 N 条记录
- **使用场景**：让模型记住"这次对话里刚发生的事"

### 长期记忆（Long-term Memory）

- **存储内容**：用户偏好、配置、重要事实等结构化信息
- **存储位置**：PostgreSQL `user_profile` 表
- **检索方式**：按 `user_id` 获取所有相关记录
- **使用场景**：跨会话记住用户信息、稳定偏好
- **详细说明**：参见 [user_profile 表说明文档](user_profile_explanation.md)

### 语义记忆（Semantic Memory）

- **存储内容**：对话片段的 embedding 向量
- **存储位置**：PostgreSQL `chat_memory_chunks` 表（带向量索引）
- **检索方式**：基于向量相似度检索（余弦相似度）
- **使用场景**：当用户说"上次我们聊过那个 XXX"，能找回相关片段

## 技术特性

### Embedding 客户端增强

- **批量切分**：每批最多 64 条，自动切分处理
- **重试机制**：指数退避，最多重试 3 次
- **LRU 缓存**：容量 5000，对相同文本的 embedding 进行缓存
- **指标统计**：记录 ingest 条数、query 次数、embedding 耗时

### 记忆系统优化（参考 LangChain 最新实现）

- **摘要生成**：使用 LLM 对长对话生成摘要，压缩记忆占用
- **记忆压缩**：当会话记忆超过阈值时，自动压缩旧记忆为摘要
- **关键事实提取**：自动从对话中提取关键事实和用户偏好到长期记忆
- **智能记忆整合**：参考 LangChain 的记忆操作模式，让 LLM 决定如何整合记忆状态
- **智能分块**：对超长消息进行智能分块，提升向量搜索质量

### 智能分块功能

当消息长度超过 `chunk_max_size` 时，系统会自动将消息分块处理：

1. **分块策略**（按优先级降级）：
   - **段落分块**（默认）：优先按段落（`\n\n`）分割，保持段落完整性
   - **句子分块**：如果段落分块失败，按句子（`。！？.!?`）分割
   - **固定长度分块**：最后备选，按固定长度分割，并在单词/字符边界处截断

2. **重叠窗口**：
   - 相邻块之间会有 `chunk_overlap` 大小的重叠
   - 确保上下文连续性，避免在关键信息处断开

3. **分块元数据**：
   - 每个 chunk 的 `meta` 字段包含：
     - `chunk_index`：块索引（从0开始）
     - `chunk_total`：总块数
     - `chunk_start_idx`、`chunk_end_idx`：在原文本中的位置（如果被分块）

4. **优势**：
   - **提升向量搜索质量**：更细粒度的语义匹配，避免长文本的语义稀释
   - **提高召回率**：查询更容易匹配到相关片段
   - **保持上下文**：通过重叠窗口和智能边界，保持语义完整性

#### 使用示例

```json
{
  "user_id": "user123",
  "session_id": "session456",
  "messages": [
    {
      "role": "user",
      "content": "这是一段很长的消息..."
    }
  ],
  "chunk_max_size": 1000,
  "chunk_overlap": 100,
  "chunk_strategy": "paragraph"
}
```

### 记忆管理策略

1. **写入策略**：
   - 每次对话后自动保存到短期记忆
   - 长期记忆需要显式调用 API 保存（可扩展）
   - 所有记忆都会生成 embedding 用于语义检索

2. **检索策略**：
   - 每次对话前，自动检索相关记忆
   - 长期记忆放在 system prompt 中
   - 语义记忆作为上下文注入
   - 短期记忆按时间顺序拼接

3. **Token 预算管理**：
   - 通过 `session_memory_limit` 和 `semantic_memory_limit` 控制
   - 避免上下文爆炸

## 数据库初始化

执行以下 SQL 创建表结构：

```sql
-- 见 migrations/create_memory_tables.sql
```

注意：需要安装 `pgvector` 扩展：

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

## 使用示例

### 基本聊天（自动保存记忆）

```bash
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session456",
    "messages": [
      {"role": "user", "content": "你好，我是张三，我喜欢编程"}
    ]
  }'
```

### 测试短期记忆（多轮对话）

```bash
# 第一轮对话
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session456",
    "messages": [
      {"role": "user", "content": "你好，我是张三"}
    ]
  }'

# 第二轮对话（测试记忆）
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session456",
    "messages": [
      {"role": "user", "content": "我刚才说我叫什么名字？"}
    ],
    "session_memory_limit": 10
  }'
```

### 测试语义记忆（跨会话检索）

```bash
# 会话 A
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session_A",
    "messages": [
      {"role": "user", "content": "我想学习机器学习"}
    ]
  }'

# 会话 B（测试语义检索）
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session_B",
    "messages": [
      {"role": "user", "content": "我之前提到过想学习什么？"}
    ],
    "semantic_memory_limit": 5,
    "semantic_threshold": 0.7
  }'
```

### 启用自动提取关键事实

```bash
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session456",
    "messages": [
      {"role": "user", "content": "我的名字是李四，我今年30岁，我喜欢阅读，我住在北京"}
    ],
    "enable_auto_extract": true
  }'
```

### 启用摘要和压缩

```bash
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session456",
    "messages": [
      {"role": "user", "content": "一段很长的对话内容..."}
    ],
    "enable_summary": true,
    "compress_threshold": 20
  }'
```

### 流式聊天

```bash
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session456",
    "messages": [
      {"role": "user", "content": "你好"}
    ],
    "stream": true
  }'
```

### 完整功能测试示例

```bash
# 第一轮：自我介绍（启用自动提取）
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user",
    "session_id": "test_session",
    "messages": [
      {"role": "user", "content": "你好，我是王五，我是一名软件工程师，主要使用 Go 语言开发"}
    ],
    "enable_auto_extract": true
  }'

# 第二轮：测试短期记忆
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user",
    "session_id": "test_session",
    "messages": [
      {"role": "user", "content": "我刚才说我叫什么名字？"}
    ],
    "session_memory_limit": 10
  }'

# 第三轮：测试长期记忆
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user",
    "session_id": "test_session",
    "messages": [
      {"role": "user", "content": "我的职业是什么？"}
    ]
  }'

# 第四轮：新会话测试语义记忆
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user",
    "session_id": "test_session_new",
    "messages": [
      {"role": "user", "content": "我之前提到过使用什么编程语言？"}
    ],
    "semantic_memory_limit": 5,
    "semantic_threshold": 0.6
  }'
```

**详细测试示例请参考**：`docs/test_examples.md`

## 注意事项

1. 流式返回时，记忆保存需要在客户端收集完整响应后手动调用（或通过回调）
2. 向量检索需要确保 PostgreSQL 已安装 `pgvector` 扩展
3. 建议根据实际场景调整记忆条数和相似度阈值
4. 长期记忆的保存需要额外的 API（可扩展实现）

