package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"antigravity/internal/model"
	"antigravity/internal/proxy"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HandleChatCompletions 处理 OpenAI Chat Completions 请求
func HandleChatCompletions(c *gin.Context) {
	accessToken := c.GetString("accessToken")
	projectID := c.GetString("projectID")

	var req model.OpenAIChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"message": err.Error(), "type": "invalid_request_error"}})
		return
	}

	antigravityReq := proxy.ConvertOpenAIChatToAntigravity(&req, projectID)
	mappedModel := proxy.Alias2ModelName(req.Model)
	isImageModel := proxy.IsImageModel(mappedModel)

	// Image 模型强制使用非流式请求
	useStream := req.Stream && !isImageModel
	resp, err := proxy.SendRequest(accessToken, antigravityReq, useStream)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": map[string]any{"message": err.Error(), "type": "api_error"}})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, gin.H{"error": map[string]any{"message": string(body), "type": "api_error"}})
		return
	}

	if isImageModel && req.Stream {
		// Image 模型：非流式获取，流式格式返回
		HandleImageModelStream(c, resp, req.Model)
	} else if req.Stream {
		HandleOpenAIStream(c, resp, req.Model)
	} else {
		HandleOpenAINonStream(c, resp, req.Model)
	}
}

// HandleOpenAIStream 处理 OpenAI 流式响应
func HandleOpenAIStream(c *gin.Context, resp *http.Response, model string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	outputChan := make(chan string, 100)
	go proxy.ParseOpenAISSE(resp.Body, outputChan, model)

	c.Stream(func(w io.Writer) bool {
		if data, ok := <-outputChan; ok {
			c.Writer.WriteString(data)
			c.Writer.Flush()
			return true
		}
		return false
	})
}

// HandleOpenAINonStream 处理 OpenAI 非流式响应
func HandleOpenAINonStream(c *gin.Context, resp *http.Response, model string) {
	body, _ := io.ReadAll(resp.Body)
	var antigravityResp map[string]any
	json.Unmarshal(body, &antigravityResp)

	openaiResp := proxy.ConvertToOpenAIChatResponse(antigravityResp, model)
	if openaiResp == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": map[string]any{"message": "Failed to convert response", "type": "api_error"}})
		return
	}
	c.JSON(http.StatusOK, openaiResp)
}

// HandleImageModelStream 处理 Image 模型流式响应（非流式获取，流式格式返回）
func HandleImageModelStream(c *gin.Context, resp *http.Response, model string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	body, _ := io.ReadAll(resp.Body)
	var antigravityResp map[string]any
	json.Unmarshal(body, &antigravityResp)

	openaiResp := proxy.ConvertToOpenAIChatResponse(antigravityResp, model)
	if openaiResp == nil {
		c.Writer.WriteString("data: {\"error\": \"Failed to convert response\"}\n\n")
		c.Writer.Flush()
		return
	}

	// 提取 content 作为流式 chunk 发送
	choices := openaiResp["choices"].([]map[string]any)
	if len(choices) > 0 {
		message := choices[0]["message"].(map[string]any)
		content := message["content"].(string)

		chunk := map[string]any{
			"id":      "chatcmpl-" + uuid.NewString()[:8],
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   model,
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{"content": content},
			}},
		}
		chunkJSON, _ := json.Marshal(chunk)
		c.Writer.WriteString("data: " + string(chunkJSON) + "\n\n")
		c.Writer.Flush()

		// 发送结束 chunk
		endChunk := map[string]any{
			"id":      "chatcmpl-" + uuid.NewString()[:8],
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   model,
			"choices": []map[string]any{{
				"index":         0,
				"delta":         map[string]any{},
				"finish_reason": "stop",
			}},
		}
		if usage, ok := openaiResp["usage"]; ok {
			endChunk["usage"] = usage
		}
		endJSON, _ := json.Marshal(endChunk)
		c.Writer.WriteString("data: " + string(endJSON) + "\n\n")
		c.Writer.Flush()
	}

	c.Writer.WriteString("data: [DONE]\n\n")
	c.Writer.Flush()
}
