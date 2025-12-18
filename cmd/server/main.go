package main

import (
	"fmt"
	"os"

	"antigravity/internal/auth"
	"antigravity/internal/router"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	r := router.SetupRouter()

	// 启动 token 刷新器
	auth.StartTokenRefresher()

	fmt.Printf("Antigravity Proxy started on port %s\n", port)
	fmt.Println("Token format: refresh_token")
	fmt.Println("\nSupported endpoints:")
	fmt.Println("  GET  /v1/models")
	fmt.Println("  POST /v1/chat/completions")
	fmt.Println("  POST /v1/messages")
	fmt.Println("  POST /v1/messages/count_tokens")
	fmt.Println("  GET  /v1beta/models")
	fmt.Println("  POST /v1beta/models/:action")

	if err := r.Run(":" + port); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
