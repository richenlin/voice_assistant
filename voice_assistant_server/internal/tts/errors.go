package tts

import "errors"

// TTS相关错误定义
var (
	ErrUnsupportedTTSType   = errors.New("unsupported TTS type")
	ErrTTSNotInitialized    = errors.New("TTS service not initialized")
	ErrInvalidVoice         = errors.New("invalid voice")
	ErrVoiceNotFound        = errors.New("voice not found")
	ErrModelNotFound        = errors.New("TTS model not found")
	ErrModelLoadFailed      = errors.New("failed to load TTS model")
	ErrSynthesisFailed      = errors.New("text synthesis failed")
	ErrInvalidText          = errors.New("invalid text for synthesis")
	ErrTextTooLong          = errors.New("text too long for synthesis")
	ErrInvalidConfig        = errors.New("invalid TTS configuration")
	ErrConnectionFailed     = errors.New("failed to connect to TTS service")
	ErrTimeout              = errors.New("TTS processing timeout")
	ErrAPIKeyMissing        = errors.New("API key is missing")
	ErrAPIKeyInvalid        = errors.New("API key is invalid")
	ErrRateLimitExceeded    = errors.New("rate limit exceeded")
	ErrQuotaExceeded        = errors.New("quota exceeded")
	ErrLanguageNotSupported = errors.New("language not supported")
	ErrFormatNotSupported   = errors.New("audio format not supported")
	ErrInvalidSampleRate    = errors.New("invalid sample rate")
	ErrInvalidChannels      = errors.New("invalid number of channels")
	ErrInsufficientMemory   = errors.New("insufficient memory for TTS model")
	ErrGPUNotAvailable      = errors.New("GPU not available for TTS")
	ErrFileWriteFailed      = errors.New("failed to write audio file")
	ErrStreamWriteFailed    = errors.New("failed to write to audio stream")
)
