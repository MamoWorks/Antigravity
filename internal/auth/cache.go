package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// TokenCache 存储用户的 Token 缓存信息
type TokenCache struct {
	AccessToken  string
	RefreshToken string
	ProjectID    string
	Email        string
	ExpiresAt    time.Time
	LastRefresh  time.Time
}

// TokenResponse Google OAuth token 响应
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// UserInfo Google 用户信息
type UserInfo struct {
	Email string `json:"email"`
}

var (
	tokenMap   = make(map[string]*TokenCache)
	tokenMutex sync.RWMutex
)

// Sha256Hash 计算 SHA256 哈希值
func Sha256Hash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// RefreshAccessToken 使用 refresh token 获取新的 access token
func RefreshAccessToken(refreshToken string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", ClientID)
	form.Set("client_secret", ClientSecret)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	req, err := http.NewRequest(http.MethodPost, GoogleTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", DefaultUserAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed (%d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// FetchUserInfo 获取用户信息
func FetchUserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &UserInfo{}, nil
	}

	var info UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// FetchProjectID 获取项目 ID
func FetchProjectID(accessToken string) (string, error) {
	reqBody := map[string]any{
		"metadata": map[string]string{
			"ideType":    "IDE_UNSPECIFIED",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	endpointURL := BaseURLProd + LoadCodeAssistPath
	req, err := http.NewRequest(http.MethodPost, endpointURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "google-api-nodejs-client/9.15.1")
	req.Header.Set("X-Goog-Api-Client", "google-cloud-sdk vscode_cloudshelleditor/0.1")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("loadCodeAssist failed (%d): %s", resp.StatusCode, string(respBytes))
	}

	var result map[string]any
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", err
	}

	if projectID, ok := result["cloudaicompanionProject"].(string); ok {
		return strings.TrimSpace(projectID), nil
	}

	return "", fmt.Errorf("no cloudaicompanionProject in response")
}

// GetOrRefreshToken 获取或刷新 token
func GetOrRefreshToken(refreshToken string) (*TokenCache, error) {
	tokenHash := Sha256Hash(refreshToken)

	// 检查缓存
	tokenMutex.RLock()
	cached, exists := tokenMap[tokenHash]
	tokenMutex.RUnlock()

	// 如果缓存存在且未过期（提前 5 分钟刷新）
	if exists && time.Now().Add(5*time.Minute).Before(cached.ExpiresAt) {
		return cached, nil
	}

	// 刷新 token
	tokenResp, err := RefreshAccessToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// 获取用户信息
	email := ""
	if info, err := FetchUserInfo(tokenResp.AccessToken); err == nil {
		email = info.Email
	}

	// 获取项目 ID
	projectID := ""
	if pid, err := FetchProjectID(tokenResp.AccessToken); err == nil {
		projectID = pid
	}

	cache := &TokenCache{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: refreshToken,
		ProjectID:    projectID,
		Email:        email,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		LastRefresh:  time.Now(),
	}

	// 更新缓存
	tokenMutex.Lock()
	tokenMap[tokenHash] = cache
	tokenMutex.Unlock()

	return cache, nil
}

// RefreshAllTokens 刷新所有缓存的 token
func RefreshAllTokens() {
	tokenMutex.RLock()
	count := len(tokenMap)
	tokenMutex.RUnlock()

	if count == 0 {
		return
	}

	fmt.Printf("[Token Refresher] Starting refresh cycle for %d tokens...\n", count)
	refreshCount := 0

	tokenMutex.RLock()
	tokens := make(map[string]*TokenCache)
	for k, v := range tokenMap {
		tokens[k] = v
	}
	tokenMutex.RUnlock()

	for hash, cache := range tokens {
		tokenResp, err := RefreshAccessToken(cache.RefreshToken)
		if err != nil {
			fmt.Printf("[Token Refresher] Failed to refresh token %s...: %v\n", hash[:8], err)
			tokenMutex.Lock()
			delete(tokenMap, hash)
			tokenMutex.Unlock()
			continue
		}

		tokenMutex.Lock()
		if tokenMap[hash] != nil {
			tokenMap[hash].AccessToken = tokenResp.AccessToken
			tokenMap[hash].ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
			tokenMap[hash].LastRefresh = time.Now()
		}
		tokenMutex.Unlock()

		refreshCount++
	}

	fmt.Printf("[Token Refresher] Refreshed %d/%d tokens\n", refreshCount, count)
}

// StartTokenRefresher 启动定时 token 刷新器
func StartTokenRefresher() {
	go func() {
		ticker := time.NewTicker(45 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			RefreshAllTokens()
		}
	}()
}
