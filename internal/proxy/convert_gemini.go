package proxy

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"

	"antigravity/internal/model"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

// ConvertGeminiToAntigravity 将 Gemini 请求转换为 Antigravity 格式
func ConvertGeminiToAntigravity(req *model.GeminiRequest, modelName, projectID string) map[string]any {
	mappedModel := Alias2ModelName(modelName)
	result := map[string]any{
		"model":     mappedModel,
		"project":   projectID,
		"requestId": "agent-" + uuid.NewString(),
		"userAgent": "antigravity",
		"request": map[string]any{
			"contents":  req.Contents,
			"sessionId": GenerateSessionID(),
		},
	}

	// Image 模型特殊处理
	if IsImageModel(mappedModel) {
		result["requestType"] = "image_gen"
		result["request"].(map[string]any)["generationConfig"] = map[string]any{
			"candidateCount": 1,
		}
	} else {
		if req.SystemInstruction != nil {
			result["request"].(map[string]any)["systemInstruction"] = req.SystemInstruction
		}
		if req.GenerationConfig != nil {
			result["request"].(map[string]any)["generationConfig"] = req.GenerationConfig
		}
		if len(req.Tools) > 0 {
			result["request"].(map[string]any)["tools"] = req.Tools
		}
	}

	return result
}

// ConvertToGeminiResponse 将 Antigravity 响应转换为 Gemini 响应
func ConvertToGeminiResponse(resp map[string]any) map[string]any {
	if respData, ok := resp["response"].(map[string]any); ok {
		return respData
	}
	return resp
}

// ParseGeminiSSE 解析 SSE 响应并转换为 Gemini SSE 格式
func ParseGeminiSSE(reader io.Reader, outputChan chan string) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(nil, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		// 解析 Antigravity 响应，提取 response 字段
		parsed := gjson.Parse(data)
		if !parsed.Get("response").Exists() {
			continue
		}

		// 提取 response 内容作为 Gemini 格式输出
		geminiData := parsed.Get("response").Raw
		if geminiData != "" {
			outputChan <- "data: " + geminiData + "\n\n"
		}
	}

	close(outputChan)
}

// ConvertToGeminiChunk 将单条 Antigravity 响应转换为 Gemini chunk
func ConvertToGeminiChunk(resp map[string]any) string {
	rawJSON, _ := json.Marshal(resp)
	root := gjson.ParseBytes(rawJSON)

	if !root.Get("response").Exists() {
		return ""
	}

	return root.Get("response").Raw
}
