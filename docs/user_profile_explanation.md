# user_profile 表说明

## 表的作用

`user_profile` 表用于存储**长期记忆（Long-term Memory）**，是三层记忆结构中的第二层。

### 存储内容

存储用户的**结构化信息**，包括：

1. **用户偏好**：语言偏好、交互风格、详细程度等
   - 例如：`{"key": "language_preference", "value": "中文"}`
   - 例如：`{"key": "detail_level", "value": "详细"}`

2. **用户信息**：姓名、年龄、职业、居住地等
   - 例如：`{"key": "name", "value": "张三"}`
   - 例如：`{"key": "age", "value": "30"}`
   - 例如：`{"key": "occupation", "value": "软件工程师"}`
   - 例如：`{"key": "location", "value": "北京"}`

3. **业务配置**：默认参数、常用设置等
   - 例如：`{"key": "default_region", "value": "us-east-1"}`
   - 例如：`{"key": "preferred_timezone", "value": "Asia/Shanghai"}`

4. **重要事实**：用户明确提到的、需要跨会话记住的信息

### 表结构

```sql
CREATE TABLE IF NOT EXISTS user_profile (
  id            bigserial PRIMARY KEY,
  user_id       text NOT NULL,           -- 用户 ID
  key           text NOT NULL,            -- 键（如 "name", "occupation"）
  value         text NOT NULL,           -- 值（如 "张三", "软件工程师"）
  confidence    float4 NOT NULL DEFAULT 1.0,  -- 置信度（0.0-1.0）
  updated_at    timestamptz NOT NULL DEFAULT now(),  -- 更新时间
  source_msg_id bigint,                  -- 来源消息 ID（可选）
  meta          jsonb NOT NULL DEFAULT '{}'::jsonb,  -- 元数据
  UNIQUE(user_id, key)                   -- 同一用户的同一 key 唯一
);
```

**字段说明**：
- `user_id` + `key` 组成唯一键，同一用户的同一属性只会有一条记录
- `confidence`：置信度，表示信息的可信程度（0.0-1.0）
- `source_msg_id`：可选，记录这条信息来自哪条消息
- `meta`：JSONB 格式，存储额外的元数据

## 写入时机

### 1. 自动提取（推荐）

**触发条件**：当请求中 `enable_auto_extract: true` 时

**写入流程**：
1. 用户发送消息后，调用 `SaveSessionMemory` 保存短期记忆
2. 如果 `options.EnableAutoExtract == true`，调用 `ExtractKeyFacts` 方法
3. 使用 LLM 从对话中提取关键事实（姓名、职业、偏好等）
4. 自动调用 `SaveLongTermMemory` 保存到 `user_profile` 表

**代码位置**：`service/memory/memory.go:162-169`

```go
// 如果启用自动提取，提取关键事实到长期记忆
if options != nil && options.EnableAutoExtract {
    facts, err := s.summarizer.ExtractKeyFacts(ctx, openaiMessages)
    if err == nil && len(facts) > 0 {
        for key, value := range facts {
            _ = s.SaveLongTermMemory(ctx, userID, key, value, 0.8, nil)
        }
        log.Infof("Auto-extracted %d key facts for user=%s", len(facts), userID)
    }
}
```

**示例请求**：
```bash
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "session_id": "session456",
    "messages": [
      {"role": "user", "content": "我的名字是张三，我今年30岁，我是一名软件工程师，我住在北京"}
    ],
    "enable_auto_extract": true
  }'
```

**提取结果**（LLM 自动提取）：
- `{"key": "name", "value": "张三"}`
- `{"key": "age", "value": "30"}`
- `{"key": "occupation", "value": "软件工程师"}`
- `{"key": "location", "value": "北京"}`

### 2. 手动保存（API 调用）

**方法**：`SaveLongTermMemory(ctx, userID, key, value, confidence, sourceMsgID)`

**使用场景**：
- 需要显式保存用户偏好
- 从其他数据源同步用户信息
- 批量导入用户配置

**代码位置**：`service/memory/memory.go:229-250`

```go
func (s *Service) SaveLongTermMemory(ctx context.Context, userID, key, value string, confidence float32, sourceMsgID *int64) error {
    // ... 保存逻辑
    // 使用 Upsert，如果已存在则更新，不存在则插入
}
```

**注意**：目前没有对外暴露的 API，需要扩展实现。

## 读取时机

### 1. 构建对话上下文时自动读取

**触发时机**：每次调用 `BuildContextWithMemory` 时

**读取逻辑**：
1. 调用 `GetLongTermMemory(userID)` 获取用户的所有长期记忆
2. 将长期记忆转换为 system prompt 的一部分
3. 注入到对话上下文中

**代码位置**：`service/memory/memory.go:399-420`

```go
func (s *Service) buildSystemPromptWithLongTermMemory(memories []*entity.UserProfile) string {
    if len(memories) == 0 {
        return ""
    }
    
    var builder strings.Builder
    builder.WriteString("以下是用户的长期记忆信息：\n")
    for _, mem := range memories {
        builder.WriteString(fmt.Sprintf("- %s: %s\n", mem.Key, mem.Value))
    }
    return builder.String()
}
```

**示例输出**：
```
以下是用户的长期记忆信息：
- name: 张三
- age: 30
- occupation: 软件工程师
- location: 北京
```

## 使用示例

### 示例 1：自动提取用户信息

```bash
# 第一轮对话：用户自我介绍（启用自动提取）
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user",
    "session_id": "session1",
    "messages": [
      {"role": "user", "content": "你好，我是李四，我今年28岁，是一名产品经理，我住在上海"}
    ],
    "enable_auto_extract": true
  }'
```

**结果**：
- 短期记忆：保存对话到 `chat_memory_chunks`
- 长期记忆：自动提取并保存到 `user_profile`：
  - `name: 李四`
  - `age: 28`
  - `occupation: 产品经理`
  - `location: 上海`

### 示例 2：跨会话使用长期记忆

```bash
# 新会话：询问用户信息（自动从长期记忆中获取）
curl -X POST http://localhost:4096/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user",
    "session_id": "session2",
    "messages": [
      {"role": "user", "content": "我住在哪里？"}
    ]
  }'
```

**结果**：
- AI 会从 `user_profile` 表中读取 `location: 上海`
- 在 system prompt 中注入："以下是用户的长期记忆信息：- location: 上海"
- AI 回答："你住在上海"

### 示例 3：查看数据库中的长期记忆

```sql
-- 查看某个用户的所有长期记忆
SELECT key, value, confidence, updated_at 
FROM user_profile 
WHERE user_id = 'test_user'
ORDER BY updated_at DESC;

-- 结果示例：
-- key          | value      | confidence | updated_at
-- -------------+------------+------------+------------------
-- location     | 上海       | 0.8        | 2024-01-15 10:30:00
-- occupation   | 产品经理   | 0.8        | 2024-01-15 10:30:00
-- age          | 28         | 0.8        | 2024-01-15 10:30:00
-- name         | 李四       | 0.8        | 2024-01-15 10:30:00
```

## 与短期记忆的区别

| 特性 | 短期记忆（chat_memory_chunks） | 长期记忆（user_profile） |
|------|------------------------------|------------------------|
| **存储内容** | 对话历史（原始消息） | 结构化事实（键值对） |
| **作用范围** | 单次会话 | 跨所有会话 |
| **检索方式** | 按 session_id 查询最近 N 条 | 按 user_id 查询所有记录 |
| **更新频率** | 每次对话都保存 | 只在提取到新事实时更新 |
| **数据格式** | 文本消息 | 键值对（key-value） |
| **用途** | 记住"这次对话里刚发生的事" | 记住"用户是谁、喜欢什么" |

## 注意事项

1. **唯一性约束**：`user_id` + `key` 唯一，同一用户的同一属性只会有一条记录
   - 如果再次提取到相同的 key，会**更新**现有记录（Upsert）
   - `updated_at` 会更新为当前时间

2. **置信度**：
   - 自动提取时，默认置信度为 `0.8`
   - 可以根据提取的准确性调整置信度
   - 低置信度的信息可以在后续对话中更新

3. **隐私和合规**：
   - 谨慎存储敏感信息（如身份证号、银行卡号等）
   - 建议对敏感信息进行加密或脱敏处理
   - 提供用户删除长期记忆的 API（需要扩展实现）

4. **提取准确性**：
   - LLM 提取可能不完美，建议：
     - 在关键场景下人工审核
     - 提供用户手动修正的机制
     - 根据置信度过滤低质量信息

5. **性能考虑**：
   - `user_profile` 表有索引 `(user_id)`，查询性能良好
   - 每次对话都会读取用户的全部长期记忆，如果记录过多可能影响性能
   - 建议：定期清理过时或低置信度的记录

