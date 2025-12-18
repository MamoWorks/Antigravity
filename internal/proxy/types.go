package proxy

// Content 表示 Gemini CLI API 的内容结构
type Content struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

// Part 表示内容的一部分
type Part struct {
	Thought          bool              `json:"thought,omitempty"`
	Text             string            `json:"text,omitempty"`
	InlineData       *PartInlineData   `json:"inlineData,omitempty"`
	ThoughtSignature string            `json:"thoughtSignature,omitempty"`
	FunctionCall     *PartFunctionCall `json:"functionCall,omitempty"`
	FunctionResponse *PartFuncResponse `json:"functionResponse,omitempty"`
}

// PartInlineData 内联数据
type PartInlineData struct {
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"`
}

// PartFunctionCall 函数调用
type PartFunctionCall struct {
	ID   string         `json:"id,omitempty"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// PartFuncResponse 函数响应
type PartFuncResponse struct {
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

// ToolDecl 工具声明
type ToolDecl struct {
	FunctionDeclarations []any `json:"functionDeclarations"`
}

// ClaudeSSEParams 流式响应状态
type ClaudeSSEParams struct {
	HasFirstResponse     bool
	ResponseType         int // 0=none, 1=content, 2=thinking, 3=function
	ResponseIndex        int
	HasFinishReason      bool
	FinishReason         string
	HasUsageMetadata     bool
	PromptTokenCount     int64
	CandidatesTokenCount int64
	ThoughtsTokenCount   int64
	TotalTokenCount      int64
	HasSentFinalEvents   bool
	HasToolUse           bool
	HasContent           bool
}
