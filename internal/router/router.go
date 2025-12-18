package router

import (
	"net/http"

	"antigravity/internal/auth"
	"antigravity/internal/handler"

	"github.com/gin-gonic/gin"
)

// SetupRouter 配置路由
func SetupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 根端点 - 重定向到 Bilibili
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "https://www.bilibili.com/video/BV1ezmRBwEeX")
	})

	// OpenAI 兼容端点
	v1 := r.Group("/v1")
	v1.Use(auth.Middleware())
	{
		v1.GET("/models", handler.HandleModels)
		v1.POST("/chat/completions", handler.HandleChatCompletions)
		v1.POST("/messages", handler.HandleClaudeMessages)
		v1.POST("/messages/count_tokens", handler.HandleClaudeCountTokens)
	}

	// Gemini 兼容端点
	v1beta := r.Group("/v1beta")
	v1beta.Use(auth.Middleware())
	{
		v1beta.GET("/models", handler.HandleGeminiModels)
		v1beta.POST("/models/*action", handler.HandleGeminiAction)
		v1beta.GET("/models/*action", handler.HandleGeminiAction)
	}

	return r
}
