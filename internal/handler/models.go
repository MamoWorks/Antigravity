package handler

import (
	"fmt"
	"net/http"
	"time"

	"antigravity/internal/proxy"

	"github.com/gin-gonic/gin"
)

// FallbackModelList 备用模型列表（API 请求失败时使用）
var FallbackModelList = []string{
	"gemini-2.5-flash",
	"gemini-2.5-flash-lite",
	"gemini-3-pro-preview",
	"gemini-3-pro-image-preview",
	"claude-sonnet-4-5",
	"claude-sonnet-4-5-thinking",
	"claude-opus-4-5-thinking",
}

// HandleModels 返回 OpenAI 格式的模型列表
func HandleModels(c *gin.Context) {
	accessToken, _ := c.Get("accessToken")
	models := getModelList(accessToken.(string))

	data := make([]map[string]any, 0, len(models))
	for _, id := range models {
		data = append(data, map[string]any{
			"id":       id,
			"object":   "model",
			"owned_by": "antigravity",
			"created":  time.Now().Unix(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

// HandleGeminiModels 返回 Gemini 格式的模型列表
func HandleGeminiModels(c *gin.Context) {
	accessToken, _ := c.Get("accessToken")
	models := getModelList(accessToken.(string))

	geminiModels := make([]map[string]any, 0, len(models))
	for _, id := range models {
		geminiModels = append(geminiModels, map[string]any{
			"name":                       fmt.Sprintf("models/%s", id),
			"displayName":                id,
			"supportedGenerationMethods": []string{"generateContent", "streamGenerateContent"},
		})
	}

	c.JSON(http.StatusOK, gin.H{"models": geminiModels})
}

// getModelList 获取模型列表，失败时返回备用列表
func getModelList(accessToken string) []string {
	models, err := proxy.FetchModels(accessToken)
	if err != nil {
		fmt.Printf("[Models] Fetch failed: %v, using fallback\n", err)
		return FallbackModelList
	}
	return models
}
