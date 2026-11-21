# 提示词注入功能实施指南

## 功能概述

为sub2api添加API密钥级别的提示词注入功能，允许用户为每个API密钥配置自定义的提示词/上下文，在向上游发送请求时自动注入，用于改善OpenAI 6系模型的行为问题（幻觉、过度防御等）。

## 已完成的工作

### 1. 数据库Schema更新 ✅
文件：`backend/ent/schema/api_key.go`

添加了两个新字段：
- `prompt_injection_enabled` (bool): 是否启用提示词注入
- `prompt_injection_content` (text): 要注入的提示词内容

## 待完成的步骤

### 步骤1：生成Ent代码（需要Go环境）

```bash
cd backend
go generate ./ent
go generate ./cmd/server
```

这将基于更新的schema生成ORM代码。

### 步骤2：更新数据库迁移

运行应用时，Ent会自动检测schema变化并创建迁移。首次启动后端时：

```bash
cd backend
go run ./cmd/server
```

应用会自动添加新列到`api_keys`表。

### 步骤3：更新Backend服务层

文件：`backend/internal/service/api_key.go`

需要在以下结构体中添加字段：

1. **CreateAPIKeyRequest** (约第217行)
```go
type CreateAPIKeyRequest struct {
    // ... 现有字段 ...
    
    // Prompt injection fields
    PromptInjectionEnabled *bool   `json:"prompt_injection_enabled"`
    PromptInjectionContent *string `json:"prompt_injection_content"`
}
```

2. **UpdateAPIKeyRequest** (约第235行)
```go
type UpdateAPIKeyRequest struct {
    // ... 现有字段 ...
    
    // Prompt injection fields
    PromptInjectionEnabled *bool   `json:"prompt_injection_enabled"`
    PromptInjectionContent *string `json:"prompt_injection_content"`
}
```

3. **APIKeyUpdateFields** (约第67行)
```go
type APIKeyUpdateFields struct {
    // ... 现有字段 ...
    PromptInjection bool
}
```

4. 在Create方法中（约第572行）添加字段设置：
```go
apiKey := &APIKey{
    // ... 现有字段 ...
}

if req.PromptInjectionEnabled != nil {
    apiKey.PromptInjectionEnabled = *req.PromptInjectionEnabled
}
if req.PromptInjectionContent != nil {
    apiKey.PromptInjectionContent = *req.PromptInjectionContent
}
```

5. 在Update方法中（约第920行）添加字段更新逻辑：
```go
// Update prompt injection fields
if req.PromptInjectionEnabled != nil {
    apiKey.PromptInjectionEnabled = *req.PromptInjectionEnabled
    fields.PromptInjection = true
}
if req.PromptInjectionContent != nil {
    apiKey.PromptInjectionContent = *req.PromptInjectionContent
    fields.PromptInjection = true
}
```

### 步骤4：更新Handler层

文件：`backend/internal/handler/api_key_handler.go`

1. 更新CreateAPIKeyRequest结构体（约第34行）：
```go
type CreateAPIKeyRequest struct {
    // ... 现有字段 ...
    
    // Prompt injection fields
    PromptInjectionEnabled *bool   `json:"prompt_injection_enabled"`
    PromptInjectionContent *string `json:"prompt_injection_content"`
}
```

2. 更新UpdateAPIKeyRequest结构体（约第50行）：
```go
type UpdateAPIKeyRequest struct {
    // ... 现有字段 ...
    
    // Prompt injection fields
    PromptInjectionEnabled *bool   `json:"prompt_injection_enabled"`
    PromptInjectionContent *string `json:"prompt_injection_content"`
}
```

3. 在Create方法中（约第200行）传递字段：
```go
svcReq := service.CreateAPIKeyRequest{
    // ... 现有字段 ...
    PromptInjectionEnabled: req.PromptInjectionEnabled,
    PromptInjectionContent: req.PromptInjectionContent,
}
```

4. 在Update方法中（约第254行）传递字段：
```go
svcReq := service.UpdateAPIKeyRequest{
    // ... 现有字段 ...
    PromptInjectionEnabled: req.PromptInjectionEnabled,
    PromptInjectionContent: req.PromptInjectionContent,
}
```

### 步骤5：实现请求拦截和提示词注入

需要在网关层拦截上游请求并注入提示词。根据不同的上游API格式：

**OpenAI Chat Completions格式：**
在system role的messages中添加或追加提示词。

**Anthropic Messages格式：**
在system参数中添加或追加提示词。

**Gemini格式：**
在systemInstruction中添加或追加提示词。

创建新文件：`backend/internal/service/prompt_injection.go`

```go
package service

import (
    "encoding/json"
    "github.com/tidwall/gjson"
    "github.com/tidwall/sjson"
)

// InjectPromptIfEnabled 如果API密钥启用了提示词注入，则注入自定义提示词
func InjectPromptIfEnabled(apiKey *APIKey, body []byte, platform string) ([]byte, error) {
    if !apiKey.PromptInjectionEnabled || apiKey.PromptInjectionContent == "" {
        return body, nil
    }

    switch platform {
    case PlatformOpenAI, PlatformComposite:
        return injectOpenAIPrompt(body, apiKey.PromptInjectionContent)
    case PlatformAnthropic:
        return injectAnthropicPrompt(body, apiKey.PromptInjectionContent)
    case PlatformGemini:
        return injectGeminiPrompt(body, apiKey.PromptInjectionContent)
    default:
        return body, nil
    }
}

func injectOpenAIPrompt(body []byte, prompt string) ([]byte, error) {
    messages := gjson.GetBytes(body, "messages")
    if !messages.Exists() || !messages.IsArray() {
        return body, nil
    }

    // 查找是否已有system消息
    hasSystem := false
    messages.ForEach(func(key, value gjson.Result) bool {
        if value.Get("role").String() == "system" {
            hasSystem = true
            return false
        }
        return true
    })

    if hasSystem {
        // 追加到现有system消息
        var messagesList []map[string]interface{}
        json.Unmarshal([]byte(messages.Raw), &messagesList)
        
        for i, msg := range messagesList {
            if msg["role"] == "system" {
                content := msg["content"].(string)
                msg["content"] = content + "\n\n" + prompt
                messagesList[i] = msg
                break
            }
        }
        
        newMessages, _ := json.Marshal(messagesList)
        return sjson.SetRawBytes(body, "messages", newMessages)
    } else {
        // 在开头插入新的system消息
        systemMsg := map[string]interface{}{
            "role":    "system",
            "content": prompt,
        }
        systemMsgJSON, _ := json.Marshal(systemMsg)
        
        var messagesList []json.RawMessage
        json.Unmarshal([]byte(messages.Raw), &messagesList)
        messagesList = append([]json.RawMessage{systemMsgJSON}, messagesList...)
        
        newMessages, _ := json.Marshal(messagesList)
        return sjson.SetRawBytes(body, "messages", newMessages)
    }
}

func injectAnthropicPrompt(body []byte, prompt string) ([]byte, error) {
    existingSystem := gjson.GetBytes(body, "system")
    
    if existingSystem.Exists() {
        // 追加到现有system
        newSystem := existingSystem.String() + "\n\n" + prompt
        return sjson.SetBytes(body, "system", newSystem)
    } else {
        // 添加新的system字段
        return sjson.SetBytes(body, "system", prompt)
    }
}

func injectGeminiPrompt(body []byte, prompt string) ([]byte, error) {
    existingInstruction := gjson.GetBytes(body, "systemInstruction.parts.0.text")
    
    if existingInstruction.Exists() {
        // 追加到现有systemInstruction
        newText := existingInstruction.String() + "\n\n" + prompt
        return sjson.SetBytes(body, "systemInstruction.parts.0.text", newText)
    } else {
        // 添加新的systemInstruction
        instruction := map[string]interface{}{
            "parts": []map[string]string{
                {"text": prompt},
            },
        }
        instructionJSON, _ := json.Marshal(instruction)
        return sjson.SetRawBytes(body, "systemInstruction", instructionJSON)
    }
}
```

### 步骤6：在网关处调用注入逻辑

在以下文件中的请求转发前调用InjectPromptIfEnabled：

1. `backend/internal/handler/gateway_handler_chat_completions.go`（约第299行）
2. `backend/internal/handler/gateway_handler_responses.go`（类似位置）

在ForwardAsChatCompletions调用前添加：

```go
// 应用提示词注入
forwardBody, err = service.InjectPromptIfEnabled(apiKey, forwardBody, account.Platform)
if err != nil {
    reqLog.Warn("gateway.cc.prompt_injection_failed", zap.Error(err))
    // 继续使用原始body，不因注入失败而中断请求
    forwardBody = body
}
```

### 步骤7：更新前端类型定义

文件：`frontend/src/types/index.ts`（约第749行）

```typescript
export interface ApiKey {
  // ... 现有字段 ...
  prompt_injection_enabled: boolean
  prompt_injection_content: string
}

export interface CreateApiKeyRequest {
  // ... 现有字段 ...
  prompt_injection_enabled?: boolean
  prompt_injection_content?: string
}

export interface UpdateApiKeyRequest {
  // ... 现有字段 ...
  prompt_injection_enabled?: boolean
  prompt_injection_content?: string
}
```

### 步骤8：更新前端API客户端

文件：`frontend/src/api/keys.ts`

在create和update函数的参数中添加新字段即可，无需修改函数签名（因为使用的是展开运算符）。

### 步骤9：添加前端UI控件

需要找到并更新API密钥的创建/编辑表单组件。搜索：

```bash
find frontend/src/components -name "*.vue" -o -name "*.tsx" | xargs grep -l "CreateApiKeyRequest\|api.*key.*form\|密钥"
```

在表单中添加：
1. 一个开关控件（Switch/Toggle）用于启用/禁用提示词注入
2. 一个多行文本框（Textarea）用于输入提示词内容
3. 只有当开关启用时才显示文本框

Vue 3示例：
```vue
<div class="form-group">
  <label class="flex items-center">
    <input type="checkbox" v-model="form.prompt_injection_enabled" />
    <span class="ml-2">启用提示词注入</span>
  </label>
  <p class="text-sm text-gray-500 mt-1">
    为该API密钥启用自定义提示词注入，改善模型行为
  </p>
</div>

<div v-if="form.prompt_injection_enabled" class="form-group">
  <label>注入的提示词内容</label>
  <textarea
    v-model="form.prompt_injection_content"
    rows="6"
    placeholder="输入要注入到每个请求中的自定义提示词..."
    class="w-full border rounded p-2"
  />
  <p class="text-sm text-gray-500 mt-1">
    此提示词会自动添加到发送给上游的每个请求中
  </p>
</div>
```

### 步骤10：添加国际化文本（可选）

如果项目使用i18n，在语言文件中添加：

```json
{
  "apiKey": {
    "promptInjection": {
      "enabled": "启用提示词注入",
      "content": "提示词内容",
      "placeholder": "输入要注入到每个请求中的自定义提示词...",
      "description": "为该API密钥启用自定义提示词注入，改善模型行为"
    }
  }
}
```

## 测试计划

### 1. 后端单元测试
- 测试CreateAPIKeyRequest和UpdateAPIKeyRequest的字段验证
- 测试InjectPromptIfEnabled函数的不同场景
- 测试不同平台的提示词注入逻辑

### 2. 集成测试
- 创建启用提示词注入的API密钥
- 发送请求验证提示词是否正确注入
- 测试OpenAI、Anthropic、Gemini三种格式

### 3. 前端测试
- 测试表单的显示/隐藏逻辑
- 测试创建和更新API密钥
- 测试字段验证

## 使用示例

### 创建带提示词注入的API密钥

```bash
curl -X POST http://localhost:8080/api/v1/api-keys \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Key with Injection",
    "prompt_injection_enabled": true,
    "prompt_injection_content": "You are a helpful assistant. Be concise and accurate. Avoid hallucinations."
  }'
```

### 更新API密钥的提示词

```bash
curl -X PUT http://localhost:8080/api/v1/api-keys/123 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt_injection_enabled": true,
    "prompt_injection_content": "Updated prompt: Focus on code quality and best practices."
  }'
```

## 注意事项

1. **性能影响**：每个请求都会进行JSON解析和修改，但影响可以忽略不计
2. **提示词长度**：考虑添加长度限制（建议2000字符以内）
3. **缓存失效**：修改提示词后需要清除认证缓存
4. **安全性**：提示词内容应该被视为敏感信息，日志中不要记录
5. **兼容性**：确保注入的提示词不会破坏原始请求的结构

## 故障排查

### 提示词未生效
1. 检查API密钥的`prompt_injection_enabled`是否为true
2. 检查`prompt_injection_content`是否为空
3. 查看网关日志中是否有注入失败的警告

### 请求失败
1. 检查注入后的JSON是否格式正确
2. 验证提示词内容是否包含特殊字符导致JSON无效
3. 测试不同平台的注入逻辑

## 后续优化

1. 添加提示词模板功能
2. 支持变量替换（如{model}、{user}等）
3. 添加提示词效果统计
4. 支持条件注入（根据模型、端点等）
