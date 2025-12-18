package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"antigravity/internal/model"
	"antigravity/internal/proxy"

	"github.com/gin-gonic/gin"
)

// HandleGeminiAction 处理 Gemini generateContent/streamGenerateContent 请求
func HandleGeminiAction(c *gin.Context) {
	accessToken := c.GetString("accessToken")
	projectID := c.GetString("projectID")

	action := c.Param("action")
	action = strings.TrimPrefix(action, "/")

	// 解析 action: model:action 或 model:action?alt=sse
	parts := strings.Split(action, ":")
	if len(parts) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"message": "Invalid action format", "type": "invalid_request_error"}})
		return
	}

	modelName := parts[0]
	actionType := strings.Split(parts[1], "?")[0]
	stream := strings.Contains(c.Request.URL.RawQuery, "alt=sse")

	var req model.GeminiRequest
	if c.Request.Method == http.MethodPost {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"message": err.Error(), "type": "invalid_request_error"}})
			return
		}
	}

	antigravityReq := proxy.ConvertGeminiToAntigravity(&req, modelName, projectID)
	mappedModel := proxy.Alias2ModelName(modelName)
	isImageModel := proxy.IsImageModel(mappedModel)

	// 根据 action 类型决定是否流式
	if actionType == "streamGenerateContent" {
		stream = true
	}

	// Image 模型强制非流式
	if isImageModel {
		stream = false
	}

	resp, err := proxy.SendRequest(accessToken, antigravityReq, stream)
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

	if stream {
		HandleGeminiStream(c, resp)
	} else {
		HandleGeminiNonStream(c, resp)
	}
}

// HandleGeminiStream 处理 Gemini 流式响应
func HandleGeminiStream(c *gin.Context, resp *http.Response) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	outputChan := make(chan string, 100)
	go proxy.ParseGeminiSSE(resp.Body, outputChan)

	c.Stream(func(w io.Writer) bool {
		if data, ok := <-outputChan; ok {
			c.Writer.WriteString(data)
			c.Writer.Flush()
			return true
		}
		return false
	})
}

// HandleGeminiNonStream 处理 Gemini 非流式响应
func HandleGeminiNonStream(c *gin.Context, resp *http.Response) {
	body, _ := io.ReadAll(resp.Body)
	var antigravityResp map[string]any
	json.Unmarshal(body, &antigravityResp)

	geminiResp := proxy.ConvertToGeminiResponse(antigravityResp)
	c.JSON(http.StatusOK, geminiResp)
}
