package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"antigravity/internal/model"
	"antigravity/internal/proxy"

	"github.com/gin-gonic/gin"
)

// HandleClaudeMessages 处理 Claude Messages API 请求
func HandleClaudeMessages(c *gin.Context) {
	accessToken := c.GetString("accessToken")
	projectID := c.GetString("projectID")

	var req model.ClaudeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"message": err.Error(), "type": "invalid_request_error"}})
		return
	}

	antigravityReq := proxy.ConvertClaudeToAntigravity(&req, projectID)
	resp, err := proxy.SendRequest(accessToken, antigravityReq, req.Stream)
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

	if req.Stream {
		HandleClaudeStream(c, resp, req.Model)
	} else {
		HandleClaudeNonStream(c, resp, req.Model)
	}
}

// HandleClaudeStream 处理 Claude 流式响应
func HandleClaudeStream(c *gin.Context, resp *http.Response, model string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	outputChan := make(chan string, 100)
	go proxy.ParseClaudeSSE(resp.Body, outputChan, model)

	c.Stream(func(w io.Writer) bool {
		if data, ok := <-outputChan; ok {
			c.Writer.WriteString(data)
			c.Writer.Flush()
			return true
		}
		return false
	})
}

// HandleClaudeNonStream 处理 Claude 非流式响应
func HandleClaudeNonStream(c *gin.Context, resp *http.Response, model string) {
	body, _ := io.ReadAll(resp.Body)
	var antigravityResp map[string]any
	json.Unmarshal(body, &antigravityResp)

	claudeResp := proxy.ConvertToClaudeResponse(antigravityResp, model)
	if claudeResp == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": map[string]any{"message": "Failed to convert response", "type": "api_error"}})
		return
	}
	c.JSON(http.StatusOK, claudeResp)
}

// HandleClaudeCountTokens 处理 Claude Token 计数请求
func HandleClaudeCountTokens(c *gin.Context) {
	var req model.ClaudeCountTokensRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"message": err.Error(), "type": "invalid_request_error"}})
		return
	}

	// 简单估算: 每 4 个字符约 1 个 token
	totalChars := 0
	for _, msg := range req.Messages {
		switch content := msg.Content.(type) {
		case string:
			totalChars += len(content)
		case []any:
			for _, item := range content {
				if m, ok := item.(map[string]any); ok {
					if text, ok := m["text"].(string); ok {
						totalChars += len(text)
					}
				}
			}
		}
	}

	if req.System != nil {
		switch sys := req.System.(type) {
		case string:
			totalChars += len(sys)
		case []any:
			for _, item := range sys {
				if m, ok := item.(map[string]any); ok {
					if text, ok := m["text"].(string); ok {
						totalChars += len(text)
					}
				}
			}
		}
	}

	estimatedTokens := totalChars / 4
	if estimatedTokens < 1 {
		estimatedTokens = 1
	}

	c.JSON(http.StatusOK, gin.H{"input_tokens": estimatedTokens})
}
