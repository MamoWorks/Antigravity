//go:build ignore

package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	antigravityClientID     = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	antigravityClientSecret = "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"
	antigravityCallbackPort = 51121
)

var antigravityScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/cclog",
	"https://www.googleapis.com/auth/experimentsandconfigs",
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type userInfo struct {
	Email string `json:"email"`
}

type callbackResult struct {
	Code  string
	Error string
	State string
}

func main() {
	fmt.Println("=== Antigravity Refresh Token 获取工具 ===")
	fmt.Println()

	// 生成随机 state
	state, err := generateRandomState()
	if err != nil {
		fmt.Printf("生成 state 失败: %v\n", err)
		return
	}

	// 启动回调服务器
	resultCh := make(chan callbackResult, 1)
	srv, port, err := startCallbackServer(resultCh)
	if err != nil {
		fmt.Printf("启动回调服务器失败: %v\n", err)
		return
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	// 构建授权 URL
	redirectURI := fmt.Sprintf("http://localhost:%d/oauth-callback", port)
	authURL := buildAuthURL(redirectURI, state)

	fmt.Println("请在浏览器中打开以下链接进行授权：")
	fmt.Println()
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("等待授权回调...")

	// 等待回调
	var cbRes callbackResult
	select {
	case res := <-resultCh:
		cbRes = res
	case <-time.After(5 * time.Minute):
		fmt.Println("授权超时")
		return
	}

	if cbRes.Error != "" {
		fmt.Printf("授权失败: %s\n", cbRes.Error)
		return
	}
	if cbRes.State != state {
		fmt.Println("State 验证失败")
		return
	}
	if cbRes.Code == "" {
		fmt.Println("未获取到授权码")
		return
	}

	fmt.Println("授权成功，正在交换 Token...")

	// 交换 Token
	tokens, err := exchangeCode(cbRes.Code, redirectURI)
	if err != nil {
		fmt.Printf("Token 交换失败: %v\n", err)
		return
	}

	// 获取用户信息
	email := ""
	if tokens.AccessToken != "" {
		if info, err := fetchUserInfo(tokens.AccessToken); err == nil && info.Email != "" {
			email = info.Email
		}
	}

	// 获取 Project ID
	projectID := ""
	if tokens.AccessToken != "" {
		if pid, err := fetchProjectID(tokens.AccessToken); err == nil {
			projectID = pid
		}
	}

	fmt.Println()
	fmt.Println("=== 获取成功 ===")
	fmt.Println()
	if email != "" {
		fmt.Printf("Email:         %s\n", email)
	}
	if projectID != "" {
		fmt.Printf("Project ID:    %s\n", projectID)
	}
	fmt.Printf("Refresh Token: %s\n", tokens.RefreshToken)
	fmt.Println()
	fmt.Println("请将 Refresh Token 作为 Antigravity Proxy 的认证密钥使用")
}

func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func startCallbackServer(resultCh chan<- callbackResult) (*http.Server, int, error) {
	addr := fmt.Sprintf(":%d", antigravityCallbackPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth-callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		res := callbackResult{
			Code:  strings.TrimSpace(q.Get("code")),
			Error: strings.TrimSpace(q.Get("error")),
			State: strings.TrimSpace(q.Get("state")),
		}
		resultCh <- res

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if res.Code != "" && res.Error == "" {
			_, _ = w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>授权成功</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #fff; }
        .card { background: #f9fafb; padding: 40px; border-radius: 16px; box-shadow: 0 4px 20px rgba(0,0,0,0.08); text-align: center; }
        h1 { color: #22c55e; margin-bottom: 16px; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="card">
        <h1>✓ 授权成功</h1>
        <p>您可以关闭此窗口并返回命令行查看结果</p>
    </div>
</body>
</html>`))
		} else {
			_, _ = w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>授权失败</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: linear-gradient(135deg, #f87171 0%, #dc2626 100%); }
        .card { background: white; padding: 40px; border-radius: 16px; box-shadow: 0 10px 40px rgba(0,0,0,0.2); text-align: center; }
        h1 { color: #ef4444; margin-bottom: 16px; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="card">
        <h1>✗ 授权失败</h1>
        <p>请查看命令行输出了解详情</p>
    </div>
</body>
</html>`))
		}
	})

	srv := &http.Server{Handler: mux}
	go func() {
		_ = srv.Serve(listener)
	}()

	return srv, port, nil
}

func buildAuthURL(redirectURI, state string) string {
	params := url.Values{}
	params.Set("access_type", "offline")
	params.Set("client_id", antigravityClientID)
	params.Set("prompt", "consent")
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(antigravityScopes, " "))
	params.Set("state", state)
	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

func exchangeCode(code, redirectURI string) (*tokenResponse, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", antigravityClientID)
	data.Set("client_secret", antigravityClientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequest(http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var token tokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func fetchUserInfo(accessToken string) (*userInfo, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.googleapis.com/oauth2/v1/userinfo?alt=json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &userInfo{}, nil
	}

	var info userInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func fetchProjectID(accessToken string) (string, error) {
	reqBody := map[string]any{
		"metadata": map[string]string{
			"ideType":    "IDE_UNSPECIFIED",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest(http.MethodPost, "https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "google-api-nodejs-client/9.15.1")
	req.Header.Set("X-Goog-Api-Client", "google-cloud-sdk vscode_cloudshelleditor/0.1")
	req.Header.Set("Client-Metadata", `{"ideType":"IDE_UNSPECIFIED","platform":"PLATFORM_UNSPECIFIED","pluginType":"GEMINI"}`)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("request failed: %s", string(body))
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// 尝试获取 projectID
	if id, ok := result["cloudaicompanionProject"].(string); ok {
		return strings.TrimSpace(id), nil
	}
	if projectMap, ok := result["cloudaicompanionProject"].(map[string]any); ok {
		if id, ok := projectMap["id"].(string); ok {
			return strings.TrimSpace(id), nil
		}
	}

	return "", fmt.Errorf("no project ID found")
}
