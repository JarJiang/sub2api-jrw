# 动态提示词注入系统 - API 文档

## 概览

动态提示词注入系统提供一套完整的 API 端点和配置选项，允许用户启用或禁用提示词注入功能，并自动根据编程语言、框架和任务类型生成优化的提示词。

---

## 核心概念

### 提示词组合维度

系统支持 2,112 种提示词组合，由以下维度组成：

| 维度 | 数量 | 示例 |
|------|------|------|
| **编程语言** | 12 | Python, Go, Java, JavaScript, TypeScript, Rust, C++, C#, PHP, Ruby, Swift, Kotlin |
| **框架** | 11 | Django, FastAPI, Gin, Spring, Express, NestJS, Actix, Laravel, Rails, SwiftUI, Android |
| **任务类型** | 8 | code_generation, bug_fixing, optimization, documentation, testing, refactoring, architecture, debugging |
| **上下文模块** | 2 | best_practices, common_pitfalls |
| **总计** | **2,112** | 12 × 11 × 8 × 2 = 2,112 |

### 上下文模块说明

- **best_practices (最佳实践)** - 针对特定语言/框架的行业最佳实践和约定
- **common_pitfalls (常见陷阱)** - 该语言/框架中容易出现的错误和反模式

---

## API 端点

### 创建 API 密钥

**端点：** `POST /api/admin/keys`

**请求体：**
```json
{
  "name": "开发环境密钥",
  "quota": 100000,
  "prompt_injection_enabled": true,
  "rate_limit": 100
}
```

**响应：**
```json
{
  "id": "key_abc123def456",
  "api_key": "sk_live_abc123def456...",
  "name": "开发环境密钥",
  "quota": 100000,
  "used": 0,
  "prompt_injection_enabled": true,
  "rate_limit": 100,
  "created_at": "2026-11-21T10:00:00Z",
  "expires_at": null
}
```

**注意：** API 密钥创建成功后，只会显示一次完整的 API 密钥。请妥善保管。

---

### 更新 API 密钥

**端点：** `PUT /api/admin/keys/{key_id}`

**请求体 (可选字段)：**
```json
{
  "name": "生产环境密钥",
  "quota": 500000,
  "prompt_injection_enabled": true,
  "rate_limit": 200
}
```

**响应：**
```json
{
  "id": "key_abc123def456",
  "name": "生产环境密钥",
  "quota": 500000,
  "used": 1024,
  "prompt_injection_enabled": true,
  "rate_limit": 200,
  "updated_at": "2026-11-21T11:00:00Z"
}
```

---

### 聊天完成 (启用提示词注入)

**端点：** `POST /v1/chat/completions`

**请求头：**
```
Authorization: Bearer sk_live_abc123def456...
Content-Type: application/json
```

**请求体：**
```json
{
  "model": "gpt-3.5-turbo",
  "messages": [
    {
      "role": "user",
      "content": "如何在 Python 中实现二叉树搜索？"
    }
  ],
  "language": "python",
  "framework": "fastapi",
  "task_type": "code_generation",
  "temperature": 0.7,
  "max_tokens": 2000
}
```

**参数说明：**

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `model` | string | ✓ | 使用的模型名称 |
| `messages` | array | ✓ | 消息列表 |
| `language` | string | - | 编程语言 (启用提示词注入时使用) |
| `framework` | string | - | 框架 (启用提示词注入时使用) |
| `task_type` | string | - | 任务类型 (启用提示词注入时使用) |
| `temperature` | number | - | 生成温度 (0-2, 默认 0.7) |
| `max_tokens` | number | - | 最大输出令牌数 |

**响应 (流式)：**
```
data: {"choices":[{"delta":{"content":"def"},"index":0}]}
data: {"choices":[{"delta":{"content":" binary_search"},"index":0}]}
data: {"choices":[{"delta":{"content":"("},"index":0}]}
...
data: [DONE]
```

**响应 (非流式)：**
```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1700617200,
  "model": "gpt-3.5-turbo",
  "usage": {
    "prompt_tokens": 256,
    "completion_tokens": 512,
    "total_tokens": 768,
    "prompt_injection_tokens": 128
  },
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "def binary_search(arr, target):\n    left, right = 0, len(arr) - 1\n    while left <= right:\n        mid = (left + right) // 2\n        if arr[mid] == target:\n            return mid\n        elif arr[mid] < target:\n            left = mid + 1\n        else:\n            right = mid - 1\n    return -1"
      },
      "finish_reason": "stop"
    }
  ]
}
```

---

### 支持的语言和框架

**编程语言 (12)：**
- Python
- Go
- Java
- JavaScript
- TypeScript
- Rust
- C++
- C#
- PHP
- Ruby
- Swift
- Kotlin

**框架 (11)：**
- Django (Python)
- FastAPI (Python)
- Gin (Go)
- Spring (Java)
- Express (JavaScript)
- NestJS (TypeScript)
- Actix (Rust)
- Laravel (PHP)
- Rails (Ruby)
- SwiftUI (Swift)
- Android (Kotlin)

**任务类型 (8)：**
- `code_generation` - 代码生成
- `bug_fixing` - 错误修复
- `optimization` - 性能优化
- `documentation` - 文档编写
- `testing` - 测试编写
- `refactoring` - 代码重构
- `architecture` - 架构设计
- `debugging` - 调试帮助

---

## 使用示例

### 示例 1：启用 Python/FastAPI 代码生成

```bash
curl -X POST http://localhost:8081/v1/chat/completions \
  -H "Authorization: Bearer sk_live_abc123def456..." \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "创建一个 FastAPI 端点来处理用户认证"
      }
    ],
    "language": "python",
    "framework": "fastapi",
    "task_type": "code_generation"
  }'
```

### 示例 2：启用 Go/Gin 性能优化建议

```bash
curl -X POST http://localhost:8081/v1/chat/completions \
  -H "Authorization: Bearer sk_live_abc123def456..." \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "如何优化这个 Gin 中间件的性能？"
      }
    ],
    "language": "go",
    "framework": "gin",
    "task_type": "optimization"
  }'
```

### 示例 3：启用 Java/Spring 测试编写

```bash
curl -X POST http://localhost:8081/v1/chat/completions \
  -H "Authorization: Bearer sk_live_abc123def456..." \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "为这个 Spring Service 编写单元测试"
      }
    ],
    "language": "java",
    "framework": "spring",
    "task_type": "testing"
  }'
```

---

## 错误处理

### HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 认证失败 (API 密钥无效) |
| 403 | 禁止访问 (配额已用尽或提示词注入未启用) |
| 404 | 资源不存在 |
| 429 | 请求过于频繁 (超过速率限制) |
| 500 | 服务器错误 |

### 错误响应示例

```json
{
  "error": {
    "message": "Invalid language parameter: javascript123",
    "type": "invalid_request_error",
    "param": "language",
    "code": "invalid_value"
  }
}
```

---

## 最佳实践

### 1. API 密钥管理

```bash
# ✅ 推荐：使用环境变量
export SUB2API_KEY="sk_live_abc123def456..."
curl -H "Authorization: Bearer $SUB2API_KEY" ...

# ❌ 不推荐：在请求中硬编码密钥
curl -H "Authorization: Bearer sk_live_abc123def456..." ...
```

### 2. 提示词注入参数

```bash
# ✅ 推荐：总是提供 language 和 framework，仅当需要时提供 task_type
"language": "python",
"framework": "fastapi",
"task_type": "code_generation"

# ❌ 不推荐：提供无效或不匹配的参数
"language": "javascript",
"framework": "django"  # Django 不是 JavaScript 框架
```

### 3. 使用示例模式

```python
# Python 示例
import requests

api_key = "sk_live_abc123def456..."
headers = {
    "Authorization": f"Bearer {api_key}",
    "Content-Type": "application/json"
}

payload = {
    "model": "gpt-3.5-turbo",
    "messages": [
        {"role": "user", "content": "在 FastAPI 中实现分页"}
    ],
    "language": "python",
    "framework": "fastapi",
    "task_type": "code_generation"
}

response = requests.post(
    "http://localhost:8081/v1/chat/completions",
    headers=headers,
    json=payload
)

print(response.json())
```

### 4. 配额监控

```bash
# 定期检查配额使用情况
curl -H "Authorization: Bearer $SUB2API_KEY" \
  http://localhost:8081/api/admin/keys/current | jq '.used, .quota'
```

---

## 令牌计数

启用提示词注入时，系统会注入额外的上下文提示词。令牌计数包括：

- **prompt_tokens** - 用户消息 + 注入的提示词
- **completion_tokens** - 模型生成的响应
- **prompt_injection_tokens** - 注入提示词的令牌数 (包含在 prompt_tokens 中)
- **total_tokens** - 总计

**示例计算：**
```
用户消息: 100 tokens
注入提示词: 128 tokens
-------
prompt_tokens: 228 tokens

生成响应: 512 tokens
-------
completion_tokens: 512 tokens

总计: 740 tokens
```

---

## 速率限制

API 密钥可配置速率限制，防止滥用：

```json
{
  "rate_limit": 100,  // 每分钟最多 100 个请求
  "quota": 500000     // 总计配额 500,000 令牌
}
```

超过速率限制时，响应：
```json
{
  "error": {
    "message": "Rate limit exceeded",
    "type": "rate_limit_error",
    "retry_after": 60
  }
}
```

---

## 监控和日志

### 启用请求日志

在环境配置中设置：
```bash
LOG_LEVEL=debug
LOG_FORMAT=json
```

### 日志示例

```json
{
  "timestamp": "2026-11-21T10:30:45Z",
  "level": "info",
  "service": "gateway_handler",
  "action": "chat_completion",
  "api_key_id": "key_abc123",
  "language": "python",
  "framework": "fastapi",
  "task_type": "code_generation",
  "prompt_injection_enabled": true,
  "tokens_used": 740,
  "response_time_ms": 2500,
  "status": 200
}
```

---

## 常见问题

**Q: 如何禁用提示词注入？**
A: 更新 API 密钥，将 `prompt_injection_enabled` 设置为 `false`，或在聊天完成请求中不提供 `language`/`framework` 参数。

**Q: 提示词注入会增加成本吗？**
A: 是的，注入的提示词会计入令牌配额。平均每个请求增加 100-150 个额外令牌。

**Q: 支持自定义提示词吗？**
A: 当前版本不支持自定义提示词。系统使用预定义的 2,112 种组合。

**Q: 如何处理不支持的语言/框架组合？**
A: 系统会忽略无效组合，按正常请求处理（不注入提示词）。

---

## 支持和反馈

- 文档：https://github.com/JarJiang/sub2api-jrw/wiki
- 问题报告：https://github.com/JarJiang/sub2api-jrw/issues
- 联系方式：support@example.com

---

*最后更新：2026-11-21*
*API 版本：v1*
*系统版本：0.2.25*
