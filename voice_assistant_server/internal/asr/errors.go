package asr

import "errors"

// ASR相关错误定义
var (
	ErrUnsupportedASRType   = errors.New("unsupported ASR type")
	ErrASRNotInitialized    = errors.New("ASR service not initialized")
	ErrInvalidAudioFormat   = errors.New("invalid audio format")
	ErrModelNotFound        = errors.New("ASR model not found")
	ErrModelLoadFailed      = errors.New("failed to load ASR model")
	ErrProcessingFailed     = errors.New("audio processing failed")
	ErrLanguageNotSupported = errors.New("language not supported")
	ErrInvalidConfig        = errors.New("invalid ASR configuration")
	ErrConnectionFailed     = errors.New("failed to connect to ASR service")
	ErrTimeout              = errors.New("ASR processing timeout")
	ErrInsufficientMemory   = errors.New("insufficient memory for ASR model")
	ErrGPUNotAvailable      = errors.New("GPU not available for ASR")
)
