package model

// GeminiRequest Gemini API 请求
type GeminiRequest struct {
	Contents          []GeminiContent `json:"contents"`
	SystemInstruction *GeminiContent  `json:"systemInstruction,omitempty"`
	GenerationConfig  *GeminiGenCfg   `json:"generationConfig,omitempty"`
	Tools             []any           `json:"tools,omitempty"`
	SafetySettings    []any           `json:"safetySettings,omitempty"`
}

// GeminiContent Gemini 内容
type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart Gemini 部分
type GeminiPart struct {
	Text             string      `json:"text,omitempty"`
	InlineData       *InlineData `json:"inlineData,omitempty"`
	FunctionCall     any         `json:"functionCall,omitempty"`
	FunctionResponse any         `json:"functionResponse,omitempty"`
}

// GeminiGenCfg Gemini 生成配置
type GeminiGenCfg struct {
	Temperature     float64         `json:"temperature,omitempty"`
	TopP            float64         `json:"topP,omitempty"`
	TopK            int             `json:"topK,omitempty"`
	MaxOutputTokens int             `json:"maxOutputTokens,omitempty"`
	CandidateCount  int             `json:"candidateCount,omitempty"`
	StopSequences   []string        `json:"stopSequences,omitempty"`
	ThinkingConfig  *ThinkingConfig `json:"thinkingConfig,omitempty"`
}

// ThinkingConfig 思考配置
type ThinkingConfig struct {
	ThinkingBudget  int  `json:"thinkingBudget,omitempty"`
	IncludeThoughts bool `json:"include_thoughts,omitempty"`
}

// GeminiResponse Gemini 响应
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text,omitempty"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason  string `json:"finishReason"`
		SafetyRatings []any  `json:"safetyRatings"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Created     int64  `json:"created"`
	OwnedBy     string `json:"owned_by"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
}
