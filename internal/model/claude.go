package model

// ClaudeRequest Claude Messages 请求
type ClaudeRequest struct {
	Model       string       `json:"model"`
	Messages    []ClaudeMsg  `json:"messages"`
	System      any          `json:"system,omitempty"`
	MaxTokens   int          `json:"max_tokens"`
	Stream      bool         `json:"stream"`
	Temperature float64      `json:"temperature,omitempty"`
	TopP        float64      `json:"top_p,omitempty"`
	TopK        int          `json:"top_k,omitempty"`
	Tools       []ClaudeTool `json:"tools,omitempty"`
	ToolChoice  any          `json:"tool_choice,omitempty"`
	Thinking    *ClaudeThink `json:"thinking,omitempty"`
}

// ClaudeMsg Claude 消息
type ClaudeMsg struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// ClaudeTool Claude 工具 (支持多种格式)
type ClaudeTool struct {
	Type        string          `json:"type,omitempty"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	InputSchema any             `json:"input_schema,omitempty"`
	Custom      *ClaudeToolSpec `json:"custom,omitempty"`
}

// ClaudeToolSpec Claude 自定义工具规格
type ClaudeToolSpec struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema"`
}

// ClaudeThink Claude 思考配置
type ClaudeThink struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

// ClaudeCountTokensRequest Claude 计数请求
type ClaudeCountTokensRequest struct {
	Model    string      `json:"model"`
	Messages []ClaudeMsg `json:"messages"`
	System   any         `json:"system,omitempty"`
	Tools    []any       `json:"tools,omitempty"`
}

// ClaudeResponse Claude 响应
type ClaudeResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	Model        string `json:"model"`
	Content      []any  `json:"content"`
	StopReason   string `json:"stop_reason"`
	StopSequence any    `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}
