package auth

const (
	// Antigravity OAuth 配置
	ClientID     = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	ClientSecret = "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"
	GoogleTokenURL = "https://oauth2.googleapis.com/token"
	UserInfoURL    = "https://www.googleapis.com/oauth2/v1/userinfo?alt=json"

	// Antigravity API 端点
	BaseURLDaily  = "https://daily-cloudcode-pa.sandbox.googleapis.com"
	BaseURLProd   = "https://cloudcode-pa.googleapis.com"
	StreamPath    = "/v1internal:streamGenerateContent"
	GeneratePath  = "/v1internal:generateContent"
	ModelsPath    = "/v1internal:fetchAvailableModels"
	LoadCodeAssistPath = "/v1internal:loadCodeAssist"

	DefaultUserAgent = "antigravity/1.11.5 windows/amd64"
)
