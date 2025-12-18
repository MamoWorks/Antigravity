package model

// OpenAIChatRequest OpenAI Chat Completions 请求
type OpenAIChatRequest struct {
	Model            string       `json:"model"`
	Messages         []OpenAIMsg  `json:"messages"`
	Stream           bool         `json:"stream"`
	MaxTokens        int          `json:"max_tokens,omitempty"`
	Temperature      float64      `json:"temperature,omitempty"`
	TopP             float64      `json:"top_p,omitempty"`
	N                int          `json:"n,omitempty"`
	Stop             any          `json:"stop,omitempty"`
	PresencePenalty  float64      `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64      `json:"frequency_penalty,omitempty"`
	Tools            []OpenAITool `json:"tools,omitempty"`
	ToolChoice       any          `json:"tool_choice,omitempty"`
}

// OpenAIMsg OpenAI 消息
type OpenAIMsg struct {
	Role       string `json:"role"`
	Content    any    `json:"content"`
	Name       string `json:"name,omitempty"`
	ToolCalls  []any  `json:"tool_calls,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// OpenAITool OpenAI 工具
type OpenAITool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Parameters  any    `json:"parameters,omitempty"`
	} `json:"function"`
}

// OpenAIChatResponse OpenAI Chat 响应
type OpenAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []any  `json:"tool_calls,omitempty"`
		} `json:"message,omitempty"`
		Delta        map[string]any `json:"delta,omitempty"`
		FinishReason string         `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}
