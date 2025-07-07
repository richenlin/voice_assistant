package llm

import "errors"

// LLM相关错误定义
var (
	ErrUnsupportedLLMType    = errors.New("unsupported LLM type")
	ErrLLMNotInitialized     = errors.New("LLM service not initialized")
	ErrInvalidModel          = errors.New("invalid model name")
	ErrModelNotFound         = errors.New("LLM model not found")
	ErrModelLoadFailed       = errors.New("failed to load LLM model")
	ErrGenerationFailed      = errors.New("text generation failed")
	ErrInvalidPrompt         = errors.New("invalid prompt")
	ErrInvalidConfig         = errors.New("invalid LLM configuration")
	ErrConnectionFailed      = errors.New("failed to connect to LLM service")
	ErrTimeout               = errors.New("LLM processing timeout")
	ErrAPIKeyMissing         = errors.New("API key is missing")
	ErrAPIKeyInvalid         = errors.New("API key is invalid")
	ErrRateLimitExceeded     = errors.New("rate limit exceeded")
	ErrQuotaExceeded         = errors.New("quota exceeded")
	ErrTokenLimitExceeded    = errors.New("token limit exceeded")
	ErrContextTooLong        = errors.New("context too long")
	ErrInsufficientMemory    = errors.New("insufficient memory for LLM model")
	ErrGPUNotAvailable       = errors.New("GPU not available for LLM")
	ErrConversationNotFound  = errors.New("conversation not found")
	ErrInvalidConversationID = errors.New("invalid conversation ID")
	ErrStreamingNotSupported = errors.New("streaming not supported")
	ErrFunctionCallFailed    = errors.New("function call failed")
)
