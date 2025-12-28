package tracing

import (
	"unicode/utf8"
)

// Span attribute key constants.
const (
	AttrSpanInputs    = "mlflow.spanInputs"
	AttrSpanOutputs   = "mlflow.spanOutputs"
	AttrSpanType      = "mlflow.spanType"
	AttrRequestID     = "mlflow.traceRequestId"
	AttrExperimentID  = "mlflow.experimentId"
	AttrTokenUsage    = "mlflow.chat.tokenUsage"
	AttrChatTools     = "mlflow.chat.tools"
	AttrMessageFormat = "mlflow.message.format"
	AttrLLMReasoning  = "mlflow.llm.reasoning"
	AttrFunctionName  = "mlflow.spanFunctionName"
	AttrLinkedPrompts = "mlflow.linkedPrompts"
)

// Truncation limits.
const (
	MaxAttributeBytes = 10240 // 10KB
	MaxPreviewChars   = 1000
)

// TruncateString truncates a string to the specified byte limit at UTF-8 boundaries.
func TruncateString(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}

	// Reserve 3 bytes for "..." suffix
	targetBytes := maxBytes - 3
	if targetBytes < 0 {
		targetBytes = 0
	}

	// Truncate at UTF-8 character boundary
	truncated := s
	for len(truncated) > targetBytes {
		_, size := utf8.DecodeLastRuneInString(truncated)
		truncated = truncated[:len(truncated)-size]
	}

	return truncated + "..."
}

// TruncatePreview truncates a string for preview display.
func TruncatePreview(s string) string {
	// Count runes (characters), not bytes
	runes := []rune(s)
	if len(runes) <= MaxPreviewChars {
		return s
	}

	return string(runes[:MaxPreviewChars]) + "..."
}
