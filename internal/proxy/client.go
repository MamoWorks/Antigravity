package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"antigravity/internal/auth"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// DefaultSafetySettings 返回默认安全设置
func DefaultSafetySettings() []map[string]string {
	return []map[string]string{
		{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "OFF"},
		{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "OFF"},
		{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "OFF"},
		{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "OFF"},
		{"category": "HARM_CATEGORY_CIVIC_INTEGRITY", "threshold": "BLOCK_NONE"},
	}
}

// AttachDefaultSafetySettings 附加默认安全设置
func AttachDefaultSafetySettings(rawJSON []byte, path string) []byte {
	if gjson.GetBytes(rawJSON, path).Exists() {
		return rawJSON
	}
	out, err := sjson.SetBytes(rawJSON, path, DefaultSafetySettings())
	if err != nil {
		return rawJSON
	}
	return out
}

// FetchModels 从 Antigravity 后端获取可用模型列表
func FetchModels(accessToken string) ([]string, error) {
	baseURLs := []string{auth.BaseURLDaily, auth.BaseURLProd}

	var lastErr error
	for _, baseURL := range baseURLs {
		url := baseURL + auth.ModelsPath
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte("{}")))
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("User-Agent", auth.DefaultUserAgent)
		req.Header.Set("Accept", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			continue
		}

		var buf bytes.Buffer
		if _, err := buf.ReadFrom(resp.Body); err != nil {
			lastErr = err
			continue
		}

		// 解析 models 字段的 keys
		models := gjson.GetBytes(buf.Bytes(), "models").Map()
		result := make([]string, 0, len(models))
		for key := range models {
			result = append(result, key)
		}

		return result, nil
	}

	return nil, fmt.Errorf("fetch models failed: %v", lastErr)
}

// SendRequest 发送请求到 Antigravity 后端
func SendRequest(accessToken string, reqBody map[string]any, stream bool) (*http.Response, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	baseURLs := []string{auth.BaseURLDaily, auth.BaseURLProd}
	path := auth.GeneratePath
	if stream {
		path = auth.StreamPath + "?alt=sse"
	}

	var lastErr error
	for _, baseURL := range baseURLs {
		url := baseURL + path
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("User-Agent", auth.DefaultUserAgent)
		if stream {
			req.Header.Set("Accept", "text/event-stream")
		} else {
			req.Header.Set("Accept", "application/json")
		}

		client := &http.Client{Timeout: 5 * time.Minute}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("all endpoints failed: %v", lastErr)
}
