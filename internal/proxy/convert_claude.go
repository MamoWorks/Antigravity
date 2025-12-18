package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"antigravity/internal/model"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ConvertClaudeToAntigravity 将 Claude 请求转换为 Antigravity 格式
func ConvertClaudeToAntigravity(req *model.ClaudeRequest, projectID string) map[string]any {
	rawJSON, _ := json.Marshal(req)
	rawJSON = []byte(strings.Replace(string(rawJSON), `"url":{"type":"string","format":"uri",`, `"url":{"type":"string",`, -1))

	// 检查是否是 Claude 模型
	mappedModel := Alias2ModelName(req.Model)
	isClaudeModel := strings.Contains(mappedModel, "claude")

	// system instruction
	var systemInstruction *Content
	systemResult := gjson.GetBytes(rawJSON, "system")
	if systemResult.IsArray() {
		systemResults := systemResult.Array()
		systemInstruction = &Content{Role: "user", Parts: []Part{}}
		for i := 0; i < len(systemResults); i++ {
			systemPromptResult := systemResults[i]
			systemTypePromptResult := systemPromptResult.Get("type")
			if systemTypePromptResult.Type == gjson.String && systemTypePromptResult.String() == "text" {
				systemPrompt := systemPromptResult.Get("text").String()
				systemPart := Part{Text: systemPrompt}
				systemInstruction.Parts = append(systemInstruction.Parts, systemPart)
			}
		}
		if len(systemInstruction.Parts) == 0 {
			systemInstruction = nil
		}
	}

	// contents
	contents := make([]Content, 0)
	messagesResult := gjson.GetBytes(rawJSON, "messages")
	if messagesResult.IsArray() {
		messageResults := messagesResult.Array()
		for i := 0; i < len(messageResults); i++ {
			messageResult := messageResults[i]
			roleResult := messageResult.Get("role")
			if roleResult.Type != gjson.String {
				continue
			}
			role := roleResult.String()
			if role == "assistant" {
				role = "model"
			}
			clientContent := Content{Role: role, Parts: []Part{}}
			contentsResult := messageResult.Get("content")
			if contentsResult.IsArray() {
				contentResults := contentsResult.Array()
				for j := 0; j < len(contentResults); j++ {
					contentResult := contentResults[j]
					contentTypeResult := contentResult.Get("type")
					if contentTypeResult.Type == gjson.String && contentTypeResult.String() == "thinking" {
						prompt := contentResult.Get("thinking").String()
						signatureResult := contentResult.Get("signature")
						signature := geminiCLIClaudeThoughtSignature
						if signatureResult.Exists() {
							signature = signatureResult.String()
						}
						clientContent.Parts = append(clientContent.Parts, Part{Text: prompt, Thought: true, ThoughtSignature: signature})
					} else if contentTypeResult.Type == gjson.String && contentTypeResult.String() == "text" {
						prompt := contentResult.Get("text").String()
						clientContent.Parts = append(clientContent.Parts, Part{Text: prompt})
					} else if contentTypeResult.Type == gjson.String && contentTypeResult.String() == "tool_use" {
						functionName := contentResult.Get("name").String()
						functionArgs := contentResult.Get("input").String()
						functionID := contentResult.Get("id").String()
						var args map[string]any
						if err := json.Unmarshal([]byte(functionArgs), &args); err == nil {
							if isClaudeModel {
								clientContent.Parts = append(clientContent.Parts, Part{
									FunctionCall: &PartFunctionCall{ID: functionID, Name: functionName, Args: args},
								})
							} else {
								clientContent.Parts = append(clientContent.Parts, Part{
									FunctionCall:     &PartFunctionCall{ID: functionID, Name: functionName, Args: args},
									ThoughtSignature: geminiCLIClaudeThoughtSignature,
								})
							}
						}
					} else if contentTypeResult.Type == gjson.String && contentTypeResult.String() == "tool_result" {
						toolCallID := contentResult.Get("tool_use_id").String()
						if toolCallID != "" {
							funcName := toolCallID
							toolCallIDs := strings.Split(toolCallID, "-")
							if len(toolCallIDs) > 1 {
								funcName = strings.Join(toolCallIDs[0:len(toolCallIDs)-1], "-")
							}
							responseData := contentResult.Get("content").Raw
							functionResponse := PartFuncResponse{ID: toolCallID, Name: funcName, Response: map[string]any{"result": responseData}}
							clientContent.Parts = append(clientContent.Parts, Part{FunctionResponse: &functionResponse})
						}
					} else if contentTypeResult.Type == gjson.String && contentTypeResult.String() == "image" {
						sourceResult := contentResult.Get("source")
						if sourceResult.Get("type").String() == "base64" {
							inlineData := &PartInlineData{
								MimeType: sourceResult.Get("media_type").String(),
								Data:     sourceResult.Get("data").String(),
							}
							clientContent.Parts = append(clientContent.Parts, Part{InlineData: inlineData})
						}
					}
				}
				contents = append(contents, clientContent)
			} else if contentsResult.Type == gjson.String {
				prompt := contentsResult.String()
				contents = append(contents, Content{Role: role, Parts: []Part{{Text: prompt}}})
			}
		}
	}

	// tools
	var tools []ToolDecl
	toolsResult := gjson.GetBytes(rawJSON, "tools")
	if toolsResult.IsArray() {
		tools = make([]ToolDecl, 1)
		tools[0].FunctionDeclarations = make([]any, 0)
		toolsResults := toolsResult.Array()
		for i := 0; i < len(toolsResults); i++ {
			toolResult := toolsResults[i]

			// 检查是否是 type: custom 格式
			toolType := toolResult.Get("type").String()
			if toolType == "custom" {
				// Custom 格式: { type: "custom", custom: { name, description, input_schema } }
				customResult := toolResult.Get("custom")
				if customResult.Exists() && customResult.IsObject() {
					inputSchemaResult := customResult.Get("input_schema")
					if inputSchemaResult.Exists() && inputSchemaResult.IsObject() {
						inputSchema := inputSchemaResult.Raw
						tool, _ := sjson.Delete(customResult.Raw, "input_schema")
						tool, _ = sjson.SetRaw(tool, "parametersJsonSchema", inputSchema)
						var toolDeclaration any
						if err := json.Unmarshal([]byte(tool), &toolDeclaration); err == nil {
							tools[0].FunctionDeclarations = append(tools[0].FunctionDeclarations, toolDeclaration)
						}
					}
				}
			} else {
				// 简单格式: { name, description, input_schema }
				inputSchemaResult := toolResult.Get("input_schema")
				if inputSchemaResult.Exists() && inputSchemaResult.IsObject() {
					inputSchema := inputSchemaResult.Raw
					tool, _ := sjson.Delete(toolResult.Raw, "input_schema")
					tool, _ = sjson.SetRaw(tool, "parametersJsonSchema", inputSchema)
					tool, _ = sjson.Delete(tool, "strict")
					tool, _ = sjson.Delete(tool, "input_examples")
					tool, _ = sjson.Delete(tool, "type")
					var toolDeclaration any
					if err := json.Unmarshal([]byte(tool), &toolDeclaration); err == nil {
						tools[0].FunctionDeclarations = append(tools[0].FunctionDeclarations, toolDeclaration)
					}
				}
			}
		}
	} else {
		tools = make([]ToolDecl, 0)
	}

	// Build output Gemini CLI request JSON
	out := `{"model":"","request":{"contents":[]}}`
	out, _ = sjson.Set(out, "model", Alias2ModelName(req.Model))
	if systemInstruction != nil {
		b, _ := json.Marshal(systemInstruction)
		out, _ = sjson.SetRaw(out, "request.systemInstruction", string(b))
	}
	if len(contents) > 0 {
		b, _ := json.Marshal(contents)
		out, _ = sjson.SetRaw(out, "request.contents", string(b))
	}
	if len(tools) > 0 && len(tools[0].FunctionDeclarations) > 0 {
		b, _ := json.Marshal(tools)
		out, _ = sjson.SetRaw(out, "request.tools", string(b))
	}

	// 对于 Claude 模型，将 parametersJsonSchema 转换为 parameters
	if isClaudeModel {
		gjson.Get(out, "request.tools").ForEach(func(key, tool gjson.Result) bool {
			tool.Get("functionDeclarations").ForEach(func(funKey, funcDecl gjson.Result) bool {
				if funcDecl.Get("parametersJsonSchema").Exists() {
					out, _ = sjson.SetRaw(out, fmt.Sprintf("request.tools.%d.functionDeclarations.%d.parameters", key.Int(), funKey.Int()), funcDecl.Get("parametersJsonSchema").Raw)
					out, _ = sjson.Delete(out, fmt.Sprintf("request.tools.%d.functionDeclarations.%d.parameters.$schema", key.Int(), funKey.Int()))
					out, _ = sjson.Delete(out, fmt.Sprintf("request.tools.%d.functionDeclarations.%d.parametersJsonSchema", key.Int(), funKey.Int()))
				}
				return true
			})
			return true
		})
	}

	// Map Anthropic thinking -> Gemini thinkingBudget/include_thoughts when type==enabled
	if req.Thinking != nil && req.Thinking.Type == "enabled" && req.Thinking.BudgetTokens > 0 {
		out, _ = sjson.Set(out, "request.generationConfig.thinkingConfig.thinkingBudget", req.Thinking.BudgetTokens)
		out, _ = sjson.Set(out, "request.generationConfig.thinkingConfig.include_thoughts", true)
	}
	if req.Temperature > 0 {
		out, _ = sjson.Set(out, "request.generationConfig.temperature", req.Temperature)
	}
	if req.TopP > 0 {
		out, _ = sjson.Set(out, "request.generationConfig.topP", req.TopP)
	}
	if req.TopK > 0 {
		out, _ = sjson.Set(out, "request.generationConfig.topK", req.TopK)
	}
	if req.MaxTokens > 0 {
		out, _ = sjson.Set(out, "request.generationConfig.maxOutputTokens", req.MaxTokens)
	}

	// 添加 Antigravity 必须的字段
	out, _ = sjson.Set(out, "userAgent", "antigravity")
	if projectID != "" {
		out, _ = sjson.Set(out, "project", projectID)
	} else {
		out, _ = sjson.Set(out, "project", GenerateProjectID())
	}
	out, _ = sjson.Set(out, "requestId", "agent-"+uuid.NewString())
	out, _ = sjson.Set(out, "request.sessionId", GenerateSessionID())
	out, _ = sjson.Set(out, "request.toolConfig.functionCallingConfig.mode", "VALIDATED")

	// 清理 Antigravity 不支持的 JSON Schema 字段
	out = CleanUnsupportedSchemaFields(out)

	var result map[string]any
	json.Unmarshal([]byte(out), &result)
	return result
}

// ParseClaudeSSE 解析 Antigravity SSE 响应并转换为 Claude SSE 格式
func ParseClaudeSSE(reader io.Reader, outputChan chan string, model string) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(nil, 1024*1024)
	params := &ClaudeSSEParams{}

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		events := ConvertToClaudeSSE([]byte(data), model, params)
		for _, event := range events {
			outputChan <- event
		}
	}

	// 发送结束事件
	if params.HasContent {
		output := ""
		AppendFinalEvents(params, &output, true)
		outputChan <- output + "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	}

	close(outputChan)
}

// ConvertToClaudeSSE 将单条 SSE 数据转换为 Claude SSE 事件
func ConvertToClaudeSSE(rawJSON []byte, model string, params *ClaudeSSEParams) []string {
	var events []string

	// 初始化消息
	if !params.HasFirstResponse {
		messageStartTemplate := `{"type": "message_start", "message": {"id": "msg_1nZdL29xx5MUA1yADyHTEsnR8uuvGzszyY", "type": "message", "role": "assistant", "content": [], "model": "claude-3-5-sonnet-20241022", "stop_reason": null, "stop_sequence": null, "usage": {"input_tokens": 0, "output_tokens": 0}}}`

		if modelVersionResult := gjson.GetBytes(rawJSON, "response.modelVersion"); modelVersionResult.Exists() {
			messageStartTemplate, _ = sjson.Set(messageStartTemplate, "message.model", modelVersionResult.String())
		}
		if responseIDResult := gjson.GetBytes(rawJSON, "response.responseId"); responseIDResult.Exists() {
			messageStartTemplate, _ = sjson.Set(messageStartTemplate, "message.id", responseIDResult.String())
		}
		events = append(events, fmt.Sprintf("event: message_start\ndata: %s\n\n", messageStartTemplate))
		params.HasFirstResponse = true
	}

	// 处理响应部分
	partsResult := gjson.GetBytes(rawJSON, "response.candidates.0.content.parts")
	if partsResult.IsArray() {
		partResults := partsResult.Array()
		for i := 0; i < len(partResults); i++ {
			partResult := partResults[i]
			partTextResult := partResult.Get("text")
			functionCallResult := partResult.Get("functionCall")

			if partTextResult.Exists() {
				// 处理 thinking 内容
				if partResult.Get("thought").Bool() {
					if thoughtSignature := partResult.Get("thoughtSignature"); thoughtSignature.Exists() && thoughtSignature.String() != "" {
						data, _ := sjson.Set(fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"signature_delta","signature":""}}`, params.ResponseIndex), "delta.signature", thoughtSignature.String())
						events = append(events, fmt.Sprintf("event: content_block_delta\ndata: %s\n\n", data))
						params.HasContent = true
					} else if params.ResponseType == 2 {
						data, _ := sjson.Set(fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"thinking_delta","thinking":""}}`, params.ResponseIndex), "delta.thinking", partTextResult.String())
						events = append(events, fmt.Sprintf("event: content_block_delta\ndata: %s\n\n", data))
						params.HasContent = true
					} else {
						if params.ResponseType != 0 {
							events = append(events, fmt.Sprintf("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", params.ResponseIndex))
							params.ResponseIndex++
						}
						events = append(events, fmt.Sprintf("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":%d,\"content_block\":{\"type\":\"thinking\",\"thinking\":\"\"}}\n\n", params.ResponseIndex))
						data, _ := sjson.Set(fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"thinking_delta","thinking":""}}`, params.ResponseIndex), "delta.thinking", partTextResult.String())
						events = append(events, fmt.Sprintf("event: content_block_delta\ndata: %s\n\n", data))
						params.ResponseType = 2
						params.HasContent = true
					}
				} else {
					finishReasonResult := gjson.GetBytes(rawJSON, "response.candidates.0.finishReason")
					if partTextResult.String() != "" || !finishReasonResult.Exists() {
						if params.ResponseType == 1 {
							data, _ := sjson.Set(fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":""}}`, params.ResponseIndex), "delta.text", partTextResult.String())
							events = append(events, fmt.Sprintf("event: content_block_delta\ndata: %s\n\n", data))
							params.HasContent = true
						} else {
							if params.ResponseType != 0 {
								events = append(events, fmt.Sprintf("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", params.ResponseIndex))
								params.ResponseIndex++
							}
							if partTextResult.String() != "" {
								events = append(events, fmt.Sprintf("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":%d,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n", params.ResponseIndex))
								data, _ := sjson.Set(fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":""}}`, params.ResponseIndex), "delta.text", partTextResult.String())
								events = append(events, fmt.Sprintf("event: content_block_delta\ndata: %s\n\n", data))
								params.ResponseType = 1
								params.HasContent = true
							}
						}
					}
				}
			} else if functionCallResult.Exists() {
				params.HasToolUse = true
				fcName := functionCallResult.Get("name").String()

				if params.ResponseType == 3 {
					events = append(events, fmt.Sprintf("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", params.ResponseIndex))
					params.ResponseIndex++
					params.ResponseType = 0
				}

				if params.ResponseType != 0 {
					events = append(events, fmt.Sprintf("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", params.ResponseIndex))
					params.ResponseIndex++
				}

				data := fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"tool_use","id":"","name":"","input":{}}}`, params.ResponseIndex)
				data, _ = sjson.Set(data, "content_block.id", fmt.Sprintf("%s-%d-%d", fcName, time.Now().UnixNano(), atomic.AddUint64(&toolUseIDCounter, 1)))
				data, _ = sjson.Set(data, "content_block.name", fcName)
				events = append(events, fmt.Sprintf("event: content_block_start\ndata: %s\n\n", data))

				if fcArgsResult := functionCallResult.Get("args"); fcArgsResult.Exists() {
					data, _ = sjson.Set(fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"input_json_delta","partial_json":""}}`, params.ResponseIndex), "delta.partial_json", fcArgsResult.Raw)
					events = append(events, fmt.Sprintf("event: content_block_delta\ndata: %s\n\n", data))
				}
				params.ResponseType = 3
				params.HasContent = true
			}
		}
	}

	if finishReasonResult := gjson.GetBytes(rawJSON, "response.candidates.0.finishReason"); finishReasonResult.Exists() {
		params.HasFinishReason = true
		params.FinishReason = finishReasonResult.String()
	}

	if usageResult := gjson.GetBytes(rawJSON, "response.usageMetadata"); usageResult.Exists() {
		params.HasUsageMetadata = true
		params.PromptTokenCount = usageResult.Get("promptTokenCount").Int()
		params.CandidatesTokenCount = usageResult.Get("candidatesTokenCount").Int()
		params.ThoughtsTokenCount = usageResult.Get("thoughtsTokenCount").Int()
		params.TotalTokenCount = usageResult.Get("totalTokenCount").Int()
		if params.CandidatesTokenCount == 0 && params.TotalTokenCount > 0 {
			params.CandidatesTokenCount = params.TotalTokenCount - params.PromptTokenCount - params.ThoughtsTokenCount
			if params.CandidatesTokenCount < 0 {
				params.CandidatesTokenCount = 0
			}
		}
	}

	if params.HasUsageMetadata && params.HasFinishReason {
		output := ""
		AppendFinalEvents(params, &output, false)
		if output != "" {
			events = append(events, output)
		}
	}

	return events
}

// AppendFinalEvents 追加最终事件
func AppendFinalEvents(params *ClaudeSSEParams, output *string, force bool) {
	if params.HasSentFinalEvents {
		return
	}
	if !params.HasUsageMetadata && !force {
		return
	}
	if !params.HasContent {
		return
	}

	if params.ResponseType != 0 {
		*output = *output + fmt.Sprintf("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}\n\n", params.ResponseIndex)
		params.ResponseType = 0
	}

	stopReason := ResolveStopReason(params)
	usageOutputTokens := params.CandidatesTokenCount + params.ThoughtsTokenCount
	if usageOutputTokens == 0 && params.TotalTokenCount > 0 {
		usageOutputTokens = params.TotalTokenCount - params.PromptTokenCount
		if usageOutputTokens < 0 {
			usageOutputTokens = 0
		}
	}

	delta := fmt.Sprintf(`{"type":"message_delta","delta":{"stop_reason":"%s","stop_sequence":null},"usage":{"input_tokens":%d,"output_tokens":%d}}`, stopReason, params.PromptTokenCount, usageOutputTokens)
	*output = *output + "event: message_delta\ndata: " + delta + "\n\n"

	params.HasSentFinalEvents = true
}

// ResolveStopReason 解析停止原因
func ResolveStopReason(params *ClaudeSSEParams) string {
	if params.HasToolUse {
		return "tool_use"
	}
	switch params.FinishReason {
	case "MAX_TOKENS":
		return "max_tokens"
	case "STOP", "FINISH_REASON_UNSPECIFIED", "UNKNOWN":
		return "end_turn"
	}
	return "end_turn"
}

// ConvertToClaudeResponse 将 Antigravity 响应转换为 Claude 非流式响应
func ConvertToClaudeResponse(resp map[string]any, model string) map[string]any {
	rawJSON, _ := json.Marshal(resp)
	root := gjson.ParseBytes(rawJSON)

	promptTokens := root.Get("response.usageMetadata.promptTokenCount").Int()
	candidateTokens := root.Get("response.usageMetadata.candidatesTokenCount").Int()
	thoughtTokens := root.Get("response.usageMetadata.thoughtsTokenCount").Int()
	totalTokens := root.Get("response.usageMetadata.totalTokenCount").Int()
	outputTokens := candidateTokens + thoughtTokens
	if outputTokens == 0 && totalTokens > 0 {
		outputTokens = totalTokens - promptTokens
		if outputTokens < 0 {
			outputTokens = 0
		}
	}

	response := map[string]any{
		"id":            root.Get("response.responseId").String(),
		"type":          "message",
		"role":          "assistant",
		"model":         root.Get("response.modelVersion").String(),
		"content":       []any{},
		"stop_reason":   nil,
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":  promptTokens,
			"output_tokens": outputTokens,
		},
	}

	parts := root.Get("response.candidates.0.content.parts")
	var contentBlocks []any
	textBuilder := strings.Builder{}
	thinkingBuilder := strings.Builder{}
	toolIDCounter := 0
	hasToolCall := false

	flushText := func() {
		if textBuilder.Len() == 0 {
			return
		}
		contentBlocks = append(contentBlocks, map[string]any{
			"type": "text",
			"text": textBuilder.String(),
		})
		textBuilder.Reset()
	}

	flushThinking := func() {
		if thinkingBuilder.Len() == 0 {
			return
		}
		contentBlocks = append(contentBlocks, map[string]any{
			"type":     "thinking",
			"thinking": thinkingBuilder.String(),
		})
		thinkingBuilder.Reset()
	}

	if parts.IsArray() {
		for _, part := range parts.Array() {
			if text := part.Get("text"); text.Exists() && text.String() != "" {
				if part.Get("thought").Bool() {
					flushText()
					thinkingBuilder.WriteString(text.String())
					continue
				}
				flushThinking()
				textBuilder.WriteString(text.String())
				continue
			}

			if functionCall := part.Get("functionCall"); functionCall.Exists() {
				flushThinking()
				flushText()
				hasToolCall = true

				name := functionCall.Get("name").String()
				toolIDCounter++
				toolBlock := map[string]any{
					"type":  "tool_use",
					"id":    fmt.Sprintf("tool_%d", toolIDCounter),
					"name":  name,
					"input": map[string]any{},
				}

				if args := functionCall.Get("args"); args.Exists() {
					var parsed any
					if err := json.Unmarshal([]byte(args.Raw), &parsed); err == nil {
						toolBlock["input"] = parsed
					}
				}

				contentBlocks = append(contentBlocks, toolBlock)
				continue
			}
		}
	}

	flushThinking()
	flushText()

	response["content"] = contentBlocks

	stopReason := "end_turn"
	if hasToolCall {
		stopReason = "tool_use"
	} else {
		if finish := root.Get("response.candidates.0.finishReason"); finish.Exists() {
			switch finish.String() {
			case "MAX_TOKENS":
				stopReason = "max_tokens"
			case "STOP", "FINISH_REASON_UNSPECIFIED", "UNKNOWN":
				stopReason = "end_turn"
			default:
				stopReason = "end_turn"
			}
		}
	}
	response["stop_reason"] = stopReason

	return response
}
