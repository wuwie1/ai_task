# 记忆功能测试示例

## 测试场景说明

以下示例展示了如何测试三层记忆结构的功能。

## 1. 基础聊天（自动保存短期记忆）

### 第一轮对话

```bash
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session456",
    "messages": [
      {
        "role": "user",
        "content": "你好，我是张三，我喜欢编程，特别是 Go 语言"
      }
    ],
    "stream": false
  }'
```

**预期结果**：
- 返回 AI 的回复
- 自动保存到短期记忆（Session Memory）
- 生成 embedding 用于语义检索

### 第二轮对话（测试短期记忆）

```bash
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session456",
    "messages": [
      {
        "role": "user",
        "content": "我刚才说我叫什么名字？"
      }
    ],
    "stream": false,
    "session_memory_limit": 10
  }'
```

**预期结果**：
- AI 应该能回答"张三"（从短期记忆中获取）
- 说明短期记忆正常工作

## 2. 测试语义记忆（跨会话检索）

### 会话 A - 第一次对话

```bash
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session_A",
    "messages": [
      {
        "role": "user",
        "content": "我想学习机器学习，特别是深度学习相关的知识"
      }
    ],
    "stream": false
  }'
```

### 会话 B - 测试语义检索

```bash
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session_B",
    "messages": [
      {
        "role": "user",
        "content": "我之前提到过想学习什么技术？"
      }
    ],
    "stream": false,
    "semantic_memory_limit": 5,
    "semantic_threshold": 0.7
  }'
```

**预期结果**：
- AI 应该能通过语义检索找到"机器学习、深度学习"相关的历史对话
- 说明语义记忆（向量检索）正常工作

## 3. 测试长期记忆（用户偏好）

### 保存长期记忆（需要额外的 API，这里展示概念）

首先需要保存用户偏好到长期记忆。假设有 API `/api/v1/memory/long-term`：

```bash
# 这个 API 需要实现，目前可以通过对话自动提取
```

### 测试自动提取关键事实

```bash
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session456",
    "messages": [
      {
        "role": "user",
        "content": "我的名字是李四，我今年30岁，我喜欢阅读和旅游，我住在北京"
      }
    ],
    "stream": false,
    "enable_auto_extract": true
  }'
```

**预期结果**：
- 自动提取关键事实（姓名、年龄、爱好、居住地）到长期记忆
- 后续对话中，AI 应该能记住这些信息

### 测试长期记忆的检索

```bash
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session_new",
    "messages": [
      {
        "role": "user",
        "content": "我住在哪里？"
      }
    ],
    "stream": false
  }'
```

**预期结果**：
- AI 应该能回答"北京"（从长期记忆中获取）
- 说明长期记忆正常工作

## 4. 测试记忆压缩功能

### 创建多轮对话（超过压缩阈值）

```bash
# 第一轮
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session_long",
    "messages": [{"role": "user", "content": "第一轮对话"}],
    "stream": false
  }'

# 第二轮
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session_long",
    "messages": [{"role": "user", "content": "第二轮对话"}],
    "stream": false
  }'

# ... 继续多轮对话，直到超过 compress_threshold（默认20条）

# 第21轮对话（触发压缩）
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session_long",
    "messages": [{"role": "user", "content": "第21轮对话"}],
    "stream": false,
    "compress_threshold": 20
  }'
```

**预期结果**：
- 当记忆超过20条时，自动压缩旧记忆为摘要
- 保留最新的20条记忆
- 日志中会显示压缩信息

## 5. 测试摘要功能

### 启用摘要的长对话

```bash
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session_summary",
    "messages": [
      {
        "role": "user",
        "content": "我今天学习了 Go 语言的并发编程，包括 goroutine、channel 和 select 语句的使用。还了解了 context 包的作用，以及如何优雅地关闭 goroutine。"
      }
    ],
    "stream": false,
    "enable_summary": true
  }'
```

**预期结果**：
- 对于较长的消息（>200字符），自动生成摘要
- 摘要保存在 `summary` 字段中

## 6. 完整功能测试（推荐）

### 多轮对话完整测试

```bash
# 第一轮：自我介绍
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user",
    "session_id": "test_session",
    "messages": [
      {
        "role": "user",
        "content": "你好，我是王五，我是一名软件工程师，主要使用 Go 语言开发后端服务"
      }
    ],
    "stream": false,
    "enable_auto_extract": true
  }'

# 第二轮：询问之前的信息（测试短期记忆）
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user",
    "session_id": "test_session",
    "messages": [
      {
        "role": "user",
        "content": "我刚才说我叫什么名字？"
      }
    ],
    "stream": false,
    "session_memory_limit": 10
  }'

# 第三轮：询问职业（测试长期记忆）
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user",
    "session_id": "test_session",
    "messages": [
      {
        "role": "user",
        "content": "我的职业是什么？"
      }
    ],
    "stream": false
  }'

# 第四轮：新会话测试语义记忆
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user",
    "session_id": "test_session_new",
    "messages": [
      {
        "role": "user",
        "content": "我之前提到过使用什么编程语言？"
      }
    ],
    "stream": false,
    "semantic_memory_limit": 5,
    "semantic_threshold": 0.6
  }'
```

## 7. 流式返回测试

```bash
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user",
    "session_id": "test_stream",
    "messages": [
      {
        "role": "user",
        "content": "请介绍一下 Go 语言的特性"
      }
    ],
    "stream": true,
    "session_memory_limit": 10
  }'
```

**注意**：流式返回时，记忆保存需要在客户端收集完整响应后手动调用（或通过回调）。

## 验证记忆是否生效

### 检查日志

查看服务日志，应该能看到：
- `Saved X session memory chunks for user=xxx, session=xxx`
- `Semantic memory search: user=xxx, query=xxx, found=X results`
- `Saved long-term memory: user=xxx, key=xxx, value=xxx`
- `Compressed X memories into summary for user=xxx, session=xxx`（如果启用压缩）

### 检查数据库

```sql
-- 查看短期记忆
SELECT id, user_id, session_id, text, summary, start_ts 
FROM chat_memory_chunks 
WHERE user_id = 'test_user' 
ORDER BY start_ts DESC 
LIMIT 10;

-- 查看长期记忆
SELECT id, user_id, key, value, confidence, updated_at 
FROM user_profile 
WHERE user_id = 'test_user';
```

## 常见问题

1. **记忆没有生效**：
   - 检查 `user_id` 和 `session_id` 是否一致
   - 检查数据库连接是否正常
   - 查看日志是否有错误

2. **语义检索没有结果**：
   - 降低 `semantic_threshold`（如 0.6）
   - 增加 `semantic_memory_limit`
   - 确保历史对话已保存并生成了 embedding

3. **摘要功能不工作**：
   - 确保 `enable_summary: true`
   - 检查消息长度是否超过 200 字符
   - 查看日志是否有摘要生成错误

