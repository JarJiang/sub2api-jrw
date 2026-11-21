package service

import (
	"encoding/json"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// InjectPromptIfEnabled 如果API密钥启用了提示词注入，则注入自定义提示词到上游请求
func InjectPromptIfEnabled(apiKey *APIKey, body []byte, platform string) ([]byte, error) {
	if apiKey == nil || !apiKey.PromptInjectionEnabled || apiKey.PromptInjectionContent == "" {
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
		// 其他平台不支持或暂不处理
		return body, nil
	}
}

// injectOpenAIPrompt 为OpenAI格式请求注入system提示词
func injectOpenAIPrompt(body []byte, prompt string) ([]byte, error) {
	messages := gjson.GetBytes(body, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return body, nil
	}

	// 解析现有messages
	var messagesList []map[string]interface{}
	if err := json.Unmarshal([]byte(messages.Raw), &messagesList); err != nil {
		return body, err
	}

	// 查找是否已有system消息
	hasSystem := false
	for i, msg := range messagesList {
		if role, ok := msg["role"].(string); ok && role == "system" {
			// 追加到现有system消息
			if content, ok := msg["content"].(string); ok {
				msg["content"] = content + "\n\n" + prompt
				messagesList[i] = msg
			}
			hasSystem = true
			break
		}
	}

	if !hasSystem {
		// 在开头插入新的system消息
		systemMsg := map[string]interface{}{
			"role":    "system",
			"content": prompt,
		}
		messagesList = append([]map[string]interface{}{systemMsg}, messagesList...)
	}

	// 序列化回JSON
	newMessages, err := json.Marshal(messagesList)
	if err != nil {
		return body, err
	}

	return sjson.SetRawBytes(body, "messages", newMessages)
}

// injectAnthropicPrompt 为Anthropic格式请求注入system提示词
func injectAnthropicPrompt(body []byte, prompt string) ([]byte, error) {
	existingSystem := gjson.GetBytes(body, "system")

	if existingSystem.Exists() {
		// 已有system字段，追加内容
		var newSystem string
		if existingSystem.IsArray() {
			// system是数组格式（prompt blocks）
			var blocks []interface{}
			if err := json.Unmarshal([]byte(existingSystem.Raw), &blocks); err != nil {
				return body, err
			}
			// 在末尾追加新的text block
			blocks = append(blocks, map[string]interface{}{
				"type": "text",
				"text": prompt,
			})
			blocksJSON, err := json.Marshal(blocks)
			if err != nil {
				return body, err
			}
			return sjson.SetRawBytes(body, "system", blocksJSON)
		} else {
			// system是字符串格式
			newSystem = existingSystem.String() + "\n\n" + prompt
			return sjson.SetBytes(body, "system", newSystem)
		}
	} else {
		// 没有system字段，添加新的
		return sjson.SetBytes(body, "system", prompt)
	}
}

// injectGeminiPrompt 为Gemini格式请求注入systemInstruction
func injectGeminiPrompt(body []byte, prompt string) ([]byte, error) {
	existingInstruction := gjson.GetBytes(body, "systemInstruction")

	if existingInstruction.Exists() {
		// 已有systemInstruction，追加到parts
		existingText := gjson.GetBytes(body, "systemInstruction.parts.0.text")
		if existingText.Exists() {
			newText := existingText.String() + "\n\n" + prompt
			return sjson.SetBytes(body, "systemInstruction.parts.0.text", newText)
		}
		// 如果结构不同，尝试追加新的part
		parts := gjson.GetBytes(body, "systemInstruction.parts")
		if parts.IsArray() {
			var partsList []interface{}
			if err := json.Unmarshal([]byte(parts.Raw), &partsList); err != nil {
				return body, err
			}
			partsList = append(partsList, map[string]string{"text": prompt})
			partsJSON, err := json.Marshal(partsList)
			if err != nil {
				return body, err
			}
			return sjson.SetRawBytes(body, "systemInstruction.parts", partsJSON)
		}
	}

	// 没有systemInstruction，创建新的
	instruction := map[string]interface{}{
		"parts": []map[string]string{
			{"text": prompt},
		},
	}
	instructionJSON, err := json.Marshal(instruction)
	if err != nil {
		return body, err
	}
	return sjson.SetRawBytes(body, "systemInstruction", instructionJSON)
}
