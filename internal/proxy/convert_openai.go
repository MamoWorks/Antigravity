package proxy

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
	"time"

	"antigravity/internal/model"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

// ConvertOpenAIChatToAntigravity 将 OpenAI Chat 请求转换为 Antigravity 格式
func ConvertOpenAIChatToAntigravity(req *model.OpenAIChatRequest, projectID string) map[string]any {
	contents := make([]Content, 0)

	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		} else if role == "system" {
			role = "user"
		}

		var parts []Part
		switch content := msg.Content.(type) {
		case string:
			parts = []Part{{Text: content}}
		case []any:
			for _, item := range content {
				if m, ok := item.(map[string]any); ok {
					switch m["type"] {
					case "text":
						parts = append(parts, Part{Text: m["text"].(string)})
					case "image_url":
						if imgURL, ok := m["image_url"].(map[string]any); ok {
							if url, ok := imgURL["url"].(string); ok && strings.HasPrefix(url, "data:") {
								urlParts := strings.SplitN(url, ",", 2)
								if len(urlParts) == 2 {
									mimeType := strings.TrimPrefix(strings.Split(urlParts[0], ";")[0], "data:")
									parts = append(parts, Part{
										InlineData: &PartInlineData{
											MimeType: mimeType,
											Data:     urlParts[1],
										},
									})
								}
							}
						}
					}
				}
			}
		}

		// 处理 tool_calls
		if msg.ToolCalls != nil {
			for _, tc := range msg.ToolCalls {
				if tcMap, ok := tc.(map[string]any); ok {
					if fn, ok := tcMap["function"].(map[string]any); ok {
						var args map[string]any
						if argsStr, ok := fn["arguments"].(string); ok {
							json.Unmarshal([]byte(argsStr), &args)
						}
						parts = append(parts, Part{
							FunctionCall: &PartFunctionCall{
								Name: fn["name"].(string),
								Args: args,
							},
						})
					}
				}
			}
		}

		// 处理 tool 响应
		if msg.Role == "tool" && msg.ToolCallID != "" {
			if content, ok := msg.Content.(string); ok {
				parts = []Part{{
					FunctionResponse: &PartFuncResponse{
						Name:     msg.Name,
						Response: map[string]any{"result": content},
					},
				}}
				role = "function"
			}
		}

		if len(parts) > 0 {
			contents = append(contents, Content{
				Role:  role,
				Parts: parts,
			})
		}
	}

	genConfig := map[string]any{}
	if req.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		genConfig["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		genConfig["topP"] = req.TopP
	}

	// 工具转换
	var tools []map[string]any
	if len(req.Tools) > 0 {
		funcDecls := make([]any, 0)
		for _, tool := range req.Tools {
			if tool.Type == "function" {
				funcDecls = append(funcDecls, map[string]any{
					"name":                 tool.Function.Name,
					"description":          tool.Function.Description,
					"parametersJsonSchema": tool.Function.Parameters,
				})
			}
		}
		if len(funcDecls) > 0 {
			tools = []map[string]any{{"functionDeclarations": funcDecls}}
		}
	}

	mappedModel := Alias2ModelName(req.Model)
	result := map[string]any{
		"model":     mappedModel,
		"project":   projectID,
		"requestId": "agent-" + uuid.NewString(),
		"userAgent": "antigravity",
		"request": map[string]any{
			"contents":         contents,
			"generationConfig": genConfig,
			"sessionId":        GenerateSessionID(),
		},
	}

	// Image 模型特殊处理
	if IsImageModel(mappedModel) {
		result["requestType"] = "image_gen"
		result["request"].(map[string]any)["generationConfig"] = map[string]any{
			"candidateCount": 1,
		}
		// 删除不需要的字段
		delete(result["request"].(map[string]any), "tools")
	} else if len(tools) > 0 {
		result["request"].(map[string]any)["tools"] = tools
	}

	return result
}

// ParseOpenAISSE 解析 SSE 响应并转换为 OpenAI SSE 格式
func ParseOpenAISSE(reader io.Reader, outputChan chan string, model string) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(nil, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			outputChan <- "data: [DONE]\n\n"
			break
		}

		var resp map[string]any
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			continue
		}

		chunk := ConvertToOpenAIChunk(resp, model)
		if chunk != "" {
			outputChan <- "data: " + chunk + "\n\n"
		}
	}

	close(outputChan)
}

// ConvertToOpenAIChunk 将单条数据转换为 OpenAI chunk 格式
func ConvertToOpenAIChunk(resp map[string]any, model string) string {
	rawJSON, _ := json.Marshal(resp)
	root := gjson.ParseBytes(rawJSON)

	parts := root.Get("response.candidates.0.content.parts")
	if !parts.IsArray() || len(parts.Array()) == 0 {
		return ""
	}

	var delta map[string]any
	for _, p := range parts.Array() {
		if text := p.Get("text"); text.Exists() {
			if !p.Get("thought").Bool() {
				delta = map[string]any{"content": text.String()}
				break
			}
		}
	}

	if delta == nil {
		return ""
	}

	chunk := map[string]any{
		"id":      "chatcmpl-" + uuid.NewString()[:8],
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{{
			"index": 0,
			"delta": delta,
		}},
	}

	jsonBytes, _ := json.Marshal(chunk)
	return string(jsonBytes)
}

// ConvertToOpenAIChatResponse 将 Antigravity 响应转换为 OpenAI Chat 响应
func ConvertToOpenAIChatResponse(resp map[string]any, model string) map[string]any {
	rawJSON, _ := json.Marshal(resp)
	root := gjson.ParseBytes(rawJSON)

	parts := root.Get("response.candidates.0.content.parts")
	if !parts.IsArray() || len(parts.Array()) == 0 {
		return nil
	}

	var fullText strings.Builder
	var toolCalls []map[string]any
	var imageMarkdowns []string

	for _, p := range parts.Array() {
		if text := p.Get("text"); text.Exists() {
			if !p.Get("thought").Bool() {
				fullText.WriteString(text.String())
			}
		}
		if fc := p.Get("functionCall"); fc.Exists() {
			args, _ := json.Marshal(fc.Get("args").Value())
			toolCalls = append(toolCalls, map[string]any{
				"id":   "call_" + uuid.NewString()[:8],
				"type": "function",
				"function": map[string]any{
					"name":      fc.Get("name").String(),
					"arguments": string(args),
				},
			})
		}
		// 处理图片数据
		if inlineData := p.Get("inlineData"); inlineData.Exists() {
			mimeType := inlineData.Get("mimeType").String()
			data := inlineData.Get("data").String()
			if mimeType != "" && data != "" {
				imageURL := "data:" + mimeType + ";base64," + data
				imageMarkdowns = append(imageMarkdowns, "![image]("+imageURL+")")
			}
		}
	}

	// 拼接图片 markdown
	content := fullText.String()
	if len(imageMarkdowns) > 0 {
		if content != "" {
			content += "\n\n"
		}
		content += strings.Join(imageMarkdowns, "\n\n")
	}

	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}
	if fr := root.Get("response.candidates.0.finishReason"); fr.Exists() && fr.String() == "MAX_TOKENS" {
		finishReason = "length"
	}

	message := map[string]any{
		"role":    "assistant",
		"content": content,
	}
	if len(toolCalls) > 0 {
		message["tool_calls"] = toolCalls
	}

	usage := map[string]any{
		"prompt_tokens":     root.Get("response.usageMetadata.promptTokenCount").Int(),
		"completion_tokens": root.Get("response.usageMetadata.candidatesTokenCount").Int(),
		"total_tokens":      root.Get("response.usageMetadata.totalTokenCount").Int(),
	}

	return map[string]any{
		"id":      "chatcmpl-" + uuid.NewString()[:8],
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{{
			"index":         0,
			"message":       message,
			"finish_reason": finishReason,
		}},
		"usage": usage,
	}
}
