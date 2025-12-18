package model

// AntigravityRequest Antigravity API 请求结构
type AntigravityRequest struct {
	Model     string         `json:"model"`
	Project   string         `json:"project"`
	RequestID string         `json:"requestId"`
	UserAgent string         `json:"userAgent"`
	Request   map[string]any `json:"request"`
}

// AntigravityContent 内容结构
type AntigravityContent struct {
	Role  string            `json:"role"`
	Parts []AntigravityPart `json:"parts"`
}

// AntigravityPart 内容部分
type AntigravityPart struct {
	Text             string                   `json:"text,omitempty"`
	Thought          bool                     `json:"thought,omitempty"`
	ThoughtSignature string                   `json:"thoughtSignature,omitempty"`
	InlineData       *InlineData              `json:"inlineData,omitempty"`
	FunctionCall     *FunctionCall            `json:"functionCall,omitempty"`
	FunctionResponse *AntigravityFuncResponse `json:"functionResponse,omitempty"`
}

// InlineData 内联数据 (图片等)
type InlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// FunctionCall 函数调用
type FunctionCall struct {
	ID   string         `json:"id,omitempty"`
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

// AntigravityFuncResponse 函数响应
type AntigravityFuncResponse struct {
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

// ToolDeclaration 工具声明
type ToolDeclaration struct {
	FunctionDeclarations []any `json:"functionDeclarations"`
}
